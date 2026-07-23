package PushToTalk

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Client is a single WebSocket connection with its outbound queue.
// A Client always corresponds to exactly one PttUser.
type Client struct {
	ID       string
	Name     string
	Conn     *websocket.Conn
	Send     chan []byte
	Channels map[string]struct{}

	// lastPong is the last time we received a Pong (or any read) from the peer,
	// stored as UnixNano so it can be updated without a mutex.
	lastPong int64

	// closeOnce guards the shutdown path so it can be invoked from either the
	// reader or writer goroutine safely.
	closeOnce sync.Once
	closed    chan struct{}
}

// NewClient wires up the outbound queue and closed channel with the configured
// capacity. The caller is expected to Register / Unregister the client with the
// hub after HELLO/on disconnect.
func NewClient(id, name string, conn *websocket.Conn, sendQueueSize int) *Client {
	if sendQueueSize <= 0 {
		sendQueueSize = 256
	}
	return &Client{
		ID:       id,
		Name:     name,
		Conn:     conn,
		Send:     make(chan []byte, sendQueueSize),
		Channels: make(map[string]struct{}),
		lastPong: time.Now().UnixNano(),
		closed:   make(chan struct{}),
	}
}

// TouchPong records that the peer is alive right now.
func (c *Client) TouchPong() { atomic.StoreInt64(&c.lastPong, time.Now().UnixNano()) }

// LastPongAt returns the time of the last pong.
func (c *Client) LastPongAt() time.Time { return time.Unix(0, atomic.LoadInt64(&c.lastPong)) }

// Closed returns a channel that is closed once Close has been called.
func (c *Client) Closed() <-chan struct{} { return c.closed }

// Close tears down the client exactly once. Idempotent.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.Conn.Close()
	})
}

// Enqueue tries to deliver raw bytes to this client without blocking the
// caller. Returning false means the client is too slow and this frame is
// dropped for this client only (fan-out never stalls on one slow peer).
func (c *Client) Enqueue(raw []byte) bool {
	select {
	case <-c.closed:
		return false
	case c.Send <- raw:
		return true
	default:
		return false
	}
}

// ---- Hub state ---------------------------------------------------------
//
// ChannelClients maps a channel name to the set of currently connected WS
// clients subscribed to it. Multiple clients per channel is allowed.
// OnlineClients maps userID → *Client for O(1) presence lookup.
// ClientSession maps *Client → *SessionBucket for the most recent START.

var (
	ChannelClients   = make(map[string]map[*Client]struct{})
	ChannelClientsMu sync.RWMutex

	OnlineClients sync.Map // userID -> *Client
	ClientSession sync.Map // *Client -> *SessionBucket (filled in later phases)
)

// RegisterClient adds client to every channel in its subscription set and
// records it as online. If another connection existed for the same user, the
// previous one is closed (last-writer-wins semantics — a common pattern for
// PTT where a device only holds one active session).
func RegisterClient(c *Client, channels []string) {
	if prev, ok := OnlineClients.Swap(c.ID, c); ok {
		if pc, ok := prev.(*Client); ok && pc != c {
			log.Printf("[ptt-ws] user=%s replaced by newer conn", c.ID)
			pc.Close()
		}
	}

	ChannelClientsMu.Lock()
	for _, ch := range channels {
		set, ok := ChannelClients[ch]
		if !ok {
			set = make(map[*Client]struct{})
			ChannelClients[ch] = set
		}
		set[c] = struct{}{}
		c.Channels[ch] = struct{}{}
	}
	ChannelClientsMu.Unlock()
}

// UnregisterClient removes the client from all its channels and drops any
// online-presence entry that still points at this exact client.
func UnregisterClient(c *Client) {
	ChannelClientsMu.Lock()
	for ch := range c.Channels {
		if set, ok := ChannelClients[ch]; ok {
			delete(set, c)
			if len(set) == 0 {
				delete(ChannelClients, ch)
			}
		}
	}
	ChannelClientsMu.Unlock()

	// Only clear the online slot if it still points at *this* client — a
	// newer connection may have replaced us via RegisterClient above.
	if cur, ok := OnlineClients.Load(c.ID); ok {
		if cc, ok := cur.(*Client); ok && cc == c {
			OnlineClients.Delete(c.ID)
		}
	}

	ClientSession.Delete(c)
}

// ChannelMembers returns a snapshot of clients subscribed to the channel.
// The returned slice is safe to iterate outside of any lock.
func ChannelMembers(channel string) []*Client {
	ChannelClientsMu.RLock()
	defer ChannelClientsMu.RUnlock()
	set, ok := ChannelClients[channel]
	if !ok {
		return nil
	}
	out := make([]*Client, 0, len(set))
	for cli := range set {
		out = append(out, cli)
	}
	return out
}

// FanoutRaw forwards a pre-encoded frame to every client in `channel` except
// the sender. Slow clients are dropped for this frame only.
func FanoutRaw(channel, excludeUserID string, raw []byte) {
	for _, cli := range ChannelMembers(channel) {
		if cli.ID == excludeUserID {
			continue
		}
		if !cli.Enqueue(raw) {
			log.Printf("[ptt-ws] drop frame to slow client user=%s ch=%s bytes=%d",
				cli.ID, channel, len(raw))
		}
	}
}
