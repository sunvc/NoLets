package PushToTalk

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sunvc/NoLets/common"
)

// upgrader is process-wide; buffer sizes are tuned for small control frames
// and single 20 ms Opus packets (typically < 200 bytes @ 24 kbps).
//
// CheckOrigin policy: this is a mobile-first backend that authenticates via
// its own signature / token headers (see the Verification middleware), so
// browser same-origin protection buys us nothing. Starscream (iOS) always
// emits an Origin header derived from the target host, which would fail
// gorilla's default same-origin check and produce spurious 403s. Accept
// every upgrade — upstream middleware still enforces auth.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// HelloPayload is the first frame every client must send.
type HelloPayload struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Token     string   `json:"token"`
	Host      string   `json:"host"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Timestamp int64    `json:"timestamp"`
	Channels  []string `json:"channels"`
	Proto     int      `json:"proto"`
}

// HelloAckPayload is the server's reply to HELLO — mirrors the shape of
// JoinResponse so the client can reuse existing view models.
type HelloAckPayload struct {
	Host       string             `json:"host"`
	ServerTime int64              `json:"server_time"`
	Channels   []JoinResponseItem `json:"channels"`
}

// JoinResponseItem mirrors the existing REST JoinResponse per-channel entry.
type JoinResponseItem struct {
	Channel string        `json:"channel"`
	Users   []PttUserResp `json:"users"`
}

// ErrorPayload is the body of TypeError frames.
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ServeWS is the gin handler mounted at GET /ptt/ws. It upgrades the request
// to a WebSocket connection, performs a one-shot HELLO handshake, and then
// runs the read/write/ping loops until the peer disconnects.
func ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrader has already written a 4xx if applicable.
		return
	}

	// Bound message size at the WS layer as well; anything larger is buggy.
	conn.SetReadLimit(int64(FrameHeaderSize + MaxPayloadBytes))

	client, hello, ok := readHello(conn)
	if !ok {
		_ = conn.Close()
		return
	}

	// Sync the user's channel membership through the shared PTT tables so the
	// existing REST-based presence stays consistent.
	SyncChannels(PttUser{
		ID:        hello.ID,
		Name:      hello.Name,
		Latitude:  hello.Latitude,
		Longitude: hello.Longitude,
		Token:     hello.Token,
		Host:      hello.Host,
		Timestamp: hello.Timestamp,
	}, hello.Channels)
	RegisterClient(client, hello.Channels)

	sendHelloAck(client, hello)

	// Presence bootstrap: for every subscribed channel, hand this client the
	// current member snapshot and announce its arrival to the rest.
	for _, ch := range hello.Channels {
		sendPresenceSnapshot(client, ch)
		broadcastJoin(client, ch)
	}

	// Cancel the handshake deadline; the pinger + pong handler take over.
	_ = conn.SetReadDeadline(time.Time{})
	conn.SetPongHandler(func(string) error {
		client.TouchPong()
		return refreshReadDeadline(conn)
	})

	// Start the three goroutines: reader (this one), writer, pinger.
	go clientWriter(client)
	go clientPinger(client)

	log.Printf("[ptt-ws] user=%s connected channels=%v", client.ID, hello.Channels)

	clientReader(client) // blocks until the peer or the writer closes the conn

	// Teardown: announce leave, then drop the WS hub entry.
	//
	// IMPORTANT: GlobalUsers AND Channels are INTENTIONALLY NOT CLEARED.
	// GlobalUsers keeps the push token so firePushForSession can still find
	// offline members and wake them via APNs. Channels keeps the user's
	// channel membership so firePushForSession's channel-scoped enumeration
	// still covers them. Only the WS hub (ChannelClients) is cleaned up by
	// UnregisterClient — that's what prevents zombies from receiving live
	// fan-out frames.
	//
	// When the user comes back online (reconnect), ServeWS re-enters through
	// HELLO → RegisterClient → SyncChannels, which is idempotent.
	for ch := range client.Channels {
		broadcastLeave(client, ch)
	}
	UnregisterClient(client)

	log.Printf("[ptt-ws] user=%s disconnected", client.ID)
}

// readHello enforces the 5-second HELLO deadline and produces the Client.
func readHello(conn *websocket.Conn) (*Client, HelloPayload, bool) {
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	mt, raw, err := conn.ReadMessage()
	if err != nil || mt != websocket.BinaryMessage {
		return nil, HelloPayload{}, false
	}

	frame, err := DecodeFrame(raw)
	if err != nil || frame.Type != TypeHello {
		return nil, HelloPayload{}, false
	}

	var hp HelloPayload
	if err := json.Unmarshal(frame.Payload, &hp); err != nil || hp.ID == "" {
		return nil, HelloPayload{}, false
	}

	client := NewClient(hp.ID, hp.Name, conn, common.LocalConfig.System.WSSendQueueSize)
	return client, hp, true
}

// sendHelloAck constructs the current channel snapshot and enqueues it.
func sendHelloAck(client *Client, hello HelloPayload) {
	ack := HelloAckPayload{
		Host:       hello.Host,
		ServerTime: time.Now().UnixMilli(),
		Channels:   make([]JoinResponseItem, 0, len(hello.Channels)),
	}

	ChannelLock.RLock()
	for _, chName := range hello.Channels {
		ch, ok := Channels[chName]
		if !ok {
			continue
		}
		ack.Channels = append(ack.Channels, JoinResponseItem{
			Channel: chName,
			Users:   ch.UserListResp(),
		})
	}
	ChannelLock.RUnlock()

	body, _ := json.Marshal(ack)
	client.Send <- EncodeFrame(TypeHelloAck, 0, 0, body)
}

// clientReader consumes frames until the connection dies. It intentionally
// stays lean: control-plane frames get dispatched here, data-plane frames
// (START/AUDIO/END) will be added in phase 2.
func clientReader(client *Client) {
	defer client.Close()

	for {
		mt, raw, err := client.Conn.ReadMessage()
		if err != nil {
			// Any read error is terminal.
			return
		}
		if mt != websocket.BinaryMessage {
			// Text frames are not part of the protocol.
			continue
		}

		frame, err := DecodeFrame(raw)
		if err != nil {
			sendError(client, ErrCodeBadFrame, err.Error())
			continue
		}

		switch frame.Type {
		case TypePing:
			// Application-level heartbeat mirror. WS-level ping is already
			// handled by the pinger goroutine + gorilla's pong handler.
			client.Send <- EncodeFrame(TypePong, frame.Seq, frame.Ts, nil)
		case TypePong:
			client.TouchPong()
		case TypeStart:
			handleStart(client, frame)
		case TypeAudio:
			handleAudio(client, frame, raw)
		case TypeEnd:
			handleEnd(client, frame)
		case TypeSubscribe:
			handleSubscribe(client, frame)
		case TypePresence:
			handlePresence(client, frame)
		case TypeLeave:
			handleLeave(client, frame)
			// Client is going off the air deliberately. Tear the connection
			// down so the deferred teardown path (broadcastLeave etc.) runs.
			return
		default:
			// Unknown / not-yet-implemented types are ignored so the handshake
			// path remains testable even against older clients.
			log.Printf("[ptt-ws] user=%s unhandled frame type=0x%02x seq=%d",
				client.ID, frame.Type, frame.Seq)
		}
	}
}

// clientWriter serializes all outbound writes so only one goroutine ever
// touches the underlying net.Conn write side.
func clientWriter(client *Client) {
	defer client.Close()

	for {
		select {
		case <-client.Closed():
			return
		case buf, ok := <-client.Send:
			if !ok {
				return
			}
			_ = client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := client.Conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
				return
			}
		}
	}
}

// clientPinger drives the WS-level ping/pong keepalive.
func clientPinger(client *Client) {
	interval := common.LocalConfig.System.WSHeartbeatInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	readTimeout := common.LocalConfig.System.WSReadTimeout
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}
	// Bootstrap the initial read deadline so a dead peer is cleaned up even
	// before the first pong arrives.
	_ = client.Conn.SetReadDeadline(time.Now().Add(readTimeout))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-client.Closed():
			return
		case <-ticker.C:
			_ = client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				client.Close()
				return
			}
			// Health check: bail if we haven't heard from the peer in > readTimeout.
			if time.Since(client.LastPongAt()) > readTimeout {
				client.Close()
				return
			}
		}
	}
}

// refreshReadDeadline pushes the read deadline forward by WSReadTimeout.
func refreshReadDeadline(conn *websocket.Conn) error {
	readTimeout := common.LocalConfig.System.WSReadTimeout
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}
	return conn.SetReadDeadline(time.Now().Add(readTimeout))
}

// sendError emits a TypeError frame; failure is non-fatal.
func sendError(client *Client, code int, message string) {
	body, _ := json.Marshal(ErrorPayload{Code: code, Message: message})
	select {
	case client.Send <- EncodeFrame(TypeError, 0, 0, body):
	default:
		// Send queue full; the client is already in trouble.
	}
}
