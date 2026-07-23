package PushToTalk

import (
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestSessionBucketLifecycle checks the Append / MarkEnded / Snapshot triad
// works under interleaved reads and writes.
func TestSessionBucketLifecycle(t *testing.T) {
	bucket := &SessionBucket{
		ID:        uuid.NewString(),
		Channel:   "test",
		From:      "u1",
		StartedAt: time.Now().UnixMilli(),
		Meta:      SessionMeta{Codec: "opus", SampleRate: 48000, FrameMs: 20, Bitrate: 24000},
	}

	for i := uint32(0); i < 5; i++ {
		bucket.Append(PacketRecord{Seq: i, TsMs: i * 20, Payload: []byte{byte(i)}})
	}
	if got := len(bucket.Snapshot()); got != 5 {
		t.Fatalf("snapshot len=%d want 5", got)
	}
	if bucket.Ended() {
		t.Fatal("bucket should not be ended yet")
	}

	bucket.MarkEnded()
	if !bucket.Ended() {
		t.Fatal("bucket should be ended after MarkEnded")
	}
	// Snapshot returns an independent slice header; the underlying payload
	// bytes are read-only in production so we intentionally do not
	// deep-copy them here (verified by the shared aliasing below).
	snap := bucket.Snapshot()
	snap = append(snap, PacketRecord{Seq: 99})
	if len(bucket.Snapshot()) != 5 {
		t.Fatal("appending to snapshot must not affect the bucket")
	}
}

// TestHandleStartHappyPath drives handleStart with a fake registered client
// and verifies the peer sees a START_BROADCAST plus that ClientSession is set.
func TestHandleStartHappyPath(t *testing.T) {
	sender := &Client{
		ID:       "sender",
		Name:     "Sender",
		Send:     make(chan []byte, 8),
		Channels: map[string]struct{}{"chA": {}},
		closed:   make(chan struct{}),
	}
	peer := &Client{
		ID:       "peer",
		Name:     "Peer",
		Send:     make(chan []byte, 8),
		Channels: map[string]struct{}{"chA": {}},
		closed:   make(chan struct{}),
	}

	RegisterClient(sender, []string{"chA"})
	RegisterClient(peer, []string{"chA"})
	t.Cleanup(func() {
		UnregisterClient(sender)
		UnregisterClient(peer)
	})

	body, _ := json.Marshal(StartPayload{
		SessionID:  "sess-1",
		Channel:    "chA",
		Codec:      "opus",
		SampleRate: 48000,
		FrameMs:    20,
		Bitrate:    24000,
	})
	handleStart(sender, &Frame{Type: TypeStart, Payload: body})

	bucket := currentClientSession(sender.ID)
	if bucket == nil || bucket.ID != "sess-1" {
		t.Fatalf("session not registered on sender")
	}

	select {
	case raw := <-peer.Send:
		f, err := DecodeFrame(raw)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if f.Type != TypeStartBroadcast {
			t.Fatalf("peer got type=%x want START_BROADCAST", f.Type)
		}
		var got StartBroadcastPayload
		if err := json.Unmarshal(f.Payload, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.SessionID != "sess-1" || got.From != "sender" || got.Channel != "chA" {
			t.Fatalf("bad payload: %+v", got)
		}
	default:
		t.Fatal("peer received no START_BROADCAST")
	}

	// Sender must not receive its own broadcast.
	select {
	case raw := <-sender.Send:
		t.Fatalf("sender should not receive fan-out, got %d bytes", len(raw))
	default:
	}
}

// TestHandleAudioForwardsAndBuffers exercises the AUDIO fast path.
func TestHandleAudioForwardsAndBuffers(t *testing.T) {
	sender := &Client{
		ID: "s", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	peer := &Client{
		ID: "p", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	RegisterClient(sender, []string{"ch"})
	RegisterClient(peer, []string{"ch"})
	t.Cleanup(func() {
		UnregisterClient(sender)
		UnregisterClient(peer)
	})

	body, _ := json.Marshal(StartPayload{SessionID: "s1", Channel: "ch", Codec: "opus"})
	handleStart(sender, &Frame{Type: TypeStart, Payload: body})
	<-peer.Send // drain START_BROADCAST

	payload := []byte{1, 2, 3, 4}
	raw := EncodeFrame(TypeAudio, 42, 800, payload)
	frame, _ := DecodeFrame(raw)
	handleAudio(sender, frame, raw)

	select {
	case fwd := <-peer.Send:
		out, err := DecodeFrame(fwd)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.Type != TypeAudio || out.Seq != 42 || out.Ts != 800 {
			t.Fatalf("bad header: %+v", out)
		}
		// Payload is now [session_tag: 4 bytes BE | original payload].
		if len(out.Payload) != 4+len(payload) {
			t.Fatalf("payload length mismatch: got %d, want %d", len(out.Payload), 4+len(payload))
		}
		tag := binary.BigEndian.Uint32(out.Payload[0:4])
		if tag == 0 {
			t.Fatalf("session_tag is zero")
		}
		if string(out.Payload[4:]) != string(payload) {
			t.Fatalf("payload mismatch after tag")
		}
	default:
		t.Fatal("peer got no forwarded audio")
	}

	bucket := currentClientSession(sender.ID)
	snap := bucket.Snapshot()
	if len(snap) != 1 || snap[0].Seq != 42 {
		t.Fatalf("ring not populated: %+v", snap)
	}
}

// TestHandleSubscribeInFlight replays buffered audio for an ongoing session.
func TestHandleSubscribeInFlight(t *testing.T) {
	client := &Client{
		ID: "sub", Send: make(chan []byte, 256),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	RegisterClient(client, []string{"ch"})
	t.Cleanup(func() { UnregisterClient(client) })

	// Simulate an active session with 3 packets.
	sender := &Client{
		ID: "s", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	RegisterClient(sender, []string{"ch"})
	t.Cleanup(func() { UnregisterClient(sender) })

	body, _ := json.Marshal(StartPayload{SessionID: "sub-s1", Channel: "ch", Codec: "opus", SampleRate: 48000, FrameMs: 20, Bitrate: 24000})
	handleStart(sender, &Frame{Type: TypeStart, Payload: body})

	for i := uint32(0); i < 3; i++ {
		raw := EncodeFrame(TypeAudio, i, i*20, []byte{byte(100 + i)})
		frame, _ := DecodeFrame(raw)
		handleAudio(sender, frame, raw)
	}

	// Drain the live fan-out from peer's Send queue.
	sub := client
	drain(sub)

	// Now subscribe — receiver should get REPLAY_BEGIN + 3 AUDIO + REPLAY_END.
	subBody, _ := json.Marshal(SubscribePayload{SessionID: "sub-s1"})
	handleSubscribe(sub, &Frame{Type: TypeSubscribe, Payload: subBody})

	// Collect frames.
	var frames []*Frame
	for len(sub.Send) > 0 {
		raw := <-sub.Send
		f, _ := DecodeFrame(raw)
		frames = append(frames, f)
	}
	if len(frames) < 5 {
		t.Fatalf("got %d frames, want >= 5 (REPLAY_BEGIN + 3 AUDIO + REPLAY_END)", len(frames))
	}
	if frames[0].Type != TypeReplayBegin {
		t.Fatalf("frame[0] type=%x want REPLAY_BEGIN", frames[0].Type)
	}
	for i := 1; i <= 3; i++ {
		if frames[i].Type != TypeAudio {
			t.Fatalf("frame[%d] type=%x want AUDIO", i, frames[i].Type)
		}
		if frames[i].Seq != uint32(i-1) {
			t.Fatalf("frame[%d] seq=%d want %d", i, frames[i].Seq, i-1)
		}
	}
	if frames[4].Type != TypeReplayEnd {
		t.Fatalf("frame[4] type=%x want REPLAY_END", frames[4].Type)
	}
}

// TestHandleSubscribeEndedSession replays audio for a session that has already
// ended (but whose ring-buffer hasn't been GC'd yet).
func TestHandleSubscribeEndedSession(t *testing.T) {
	client := &Client{
		ID: "sub", Send: make(chan []byte, 256),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	RegisterClient(client, []string{"ch"})
	t.Cleanup(func() { UnregisterClient(client) })

	sender := &Client{
		ID: "s", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	RegisterClient(sender, []string{"ch"})
	t.Cleanup(func() { UnregisterClient(sender) })

	body, _ := json.Marshal(StartPayload{SessionID: "sub-s2", Channel: "ch", Codec: "opus"})
	handleStart(sender, &Frame{Type: TypeStart, Payload: body})
	for i := uint32(0); i < 2; i++ {
		raw := EncodeFrame(TypeAudio, i, i*20, []byte{byte(i)})
		frame, _ := DecodeFrame(raw)
		handleAudio(sender, frame, raw)
	}
	endBody, _ := json.Marshal(EndPayload{SessionID: "sub-s2", DurationMs: 40, TotalPackets: 2})
	handleEnd(sender, &Frame{Type: TypeEnd, Payload: endBody})

	drain(client)

	subBody, _ := json.Marshal(SubscribePayload{SessionID: "sub-s2"})
	handleSubscribe(client, &Frame{Type: TypeSubscribe, Payload: subBody})

	var frames []*Frame
	for len(client.Send) > 0 {
		raw := <-client.Send
		f, _ := DecodeFrame(raw)
		frames = append(frames, f)
	}
	if len(frames) < 4 {
		t.Fatalf("got %d frames, want >= 4", len(frames))
	}
	if frames[0].Type != TypeReplayBegin {
		t.Fatalf("frame[0] type=%x want REPLAY_BEGIN", frames[0].Type)
	}
	if frames[len(frames)-1].Type != TypeReplayEnd {
		t.Fatalf("last frame type=%x want REPLAY_END", frames[len(frames)-1].Type)
	}
}

// TestHandleSubscribeNotFound returns an error for a session that was never
// registered or was already GC'd.
func TestHandleSubscribeNotFound(t *testing.T) {
	client := &Client{
		ID: "sub", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	RegisterClient(client, []string{"ch"})
	t.Cleanup(func() { UnregisterClient(client) })

	body, _ := json.Marshal(SubscribePayload{SessionID: "ghost-session"})
	handleSubscribe(client, &Frame{Type: TypeSubscribe, Payload: body})

	select {
	case raw := <-client.Send:
		f, _ := DecodeFrame(raw)
		if f.Type != TypeError {
			t.Fatalf("expected TypeError, got %x", f.Type)
		}
	default:
		t.Fatal("no error frame emitted")
	}
}

// drain empties a client's Send channel into a black hole. Helper for tests
// that need to clear the live fan-out before checking the replay output.
func drain(c *Client) {
	for len(c.Send) > 0 {
		<-c.Send
	}
}

// TestBroadcastJoinReachesPeers ensures broadcastJoin delivers a PRESENCE
// frame with kind="join" to every other channel member and NOT to the actor.
func TestBroadcastJoinReachesPeers(t *testing.T) {
	joiner := &Client{
		ID: "joiner", Name: "Joiner", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	peer := &Client{
		ID: "peer", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	// Register both so ChannelMembers returns them for the broadcast.
	RegisterClient(joiner, []string{"ch"})
	RegisterClient(peer, []string{"ch"})
	t.Cleanup(func() {
		UnregisterClient(joiner)
		UnregisterClient(peer)
	})
	// broadcastJoin reads lat/lng from GlobalUsers.
	GlobalUsers.Store("joiner", PttUser{ID: "joiner", Name: "Joiner", Latitude: 1.5, Longitude: 2.5})
	t.Cleanup(func() { GlobalUsers.Delete("joiner") })

	broadcastJoin(joiner, "ch")

	select {
	case raw := <-peer.Send:
		f, err := DecodeFrame(raw)
		if err != nil || f.Type != TypePresence {
			t.Fatalf("peer got wrong frame type=%x err=%v", f.Type, err)
		}
		var p PresencePayload
		if err := json.Unmarshal(f.Payload, &p); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if p.Kind != "join" || p.Channel != "ch" || p.User == nil || p.User.ID != "joiner" {
			t.Fatalf("bad presence payload: %+v", p)
		}
		if p.User.Latitude != 1.5 || p.User.Longitude != 2.5 {
			t.Fatalf("location mismatch: %+v", p.User)
		}
	default:
		t.Fatal("peer received no PRESENCE frame")
	}
	select {
	case raw := <-joiner.Send:
		t.Fatalf("joiner must not receive its own join, got %d bytes", len(raw))
	default:
	}
}

// TestHandlePresenceUpdateRoundtrip walks the client → server → other-clients
// update path: sending a PRESENCE update frame refreshes GlobalUsers and fans
// out a normalised copy to same-channel peers.
func TestHandlePresenceUpdateRoundtrip(t *testing.T) {
	sender := &Client{
		ID: "sender", Name: "Sender", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"chP": {}}, closed: make(chan struct{}),
	}
	peer := &Client{
		ID: "peer", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"chP": {}}, closed: make(chan struct{}),
	}
	RegisterClient(sender, []string{"chP"})
	RegisterClient(peer, []string{"chP"})
	t.Cleanup(func() {
		UnregisterClient(sender)
		UnregisterClient(peer)
	})
	GlobalUsers.Store("sender", PttUser{ID: "sender", Name: "Sender", Latitude: 0, Longitude: 0})
	t.Cleanup(func() { GlobalUsers.Delete("sender") })

	body, _ := json.Marshal(PresencePayload{
		Kind: "update",
		User: &PttUserResp{ID: "sender", Name: "Sender", Latitude: 10.25, Longitude: 20.5, Timestamp: 42},
	})
	handlePresence(sender, &Frame{Type: TypePresence, Payload: body})

	// GlobalUsers now reflects the new location.
	if v, ok := GlobalUsers.Load("sender"); ok {
		if u, ok := v.(PttUser); ok {
			if u.Latitude != 10.25 || u.Longitude != 20.5 {
				t.Fatalf("GlobalUsers not updated: %+v", u)
			}
		}
	} else {
		t.Fatal("GlobalUsers entry missing after update")
	}

	// Peer received a broadcast update.
	select {
	case raw := <-peer.Send:
		f, _ := DecodeFrame(raw)
		if f.Type != TypePresence {
			t.Fatalf("peer got type=%x want PRESENCE", f.Type)
		}
		var p PresencePayload
		_ = json.Unmarshal(f.Payload, &p)
		if p.Kind != "update" || p.User == nil || p.User.ID != "sender" || p.User.Latitude != 10.25 {
			t.Fatalf("bad rebroadcast: %+v", p)
		}
	default:
		t.Fatal("peer received no PRESENCE update")
	}

	// Sender does NOT receive its own broadcast.
	select {
	case raw := <-sender.Send:
		t.Fatalf("sender must not receive its own update, got %d bytes", len(raw))
	default:
	}
}

// TestHandleAudioWithoutStart makes sure a naive client that skips START
// receives an ERROR and no fan-out happens.
func TestHandleAudioWithoutStart(t *testing.T) {
	c := &Client{
		ID: "s", Send: make(chan []byte, 2),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	RegisterClient(c, []string{"ch"})
	t.Cleanup(func() { UnregisterClient(c) })

	raw := EncodeFrame(TypeAudio, 1, 20, []byte{0})
	frame, _ := DecodeFrame(raw)
	handleAudio(c, frame, raw)

	select {
	case fwd := <-c.Send:
		out, _ := DecodeFrame(fwd)
		if out.Type != TypeError {
			t.Fatalf("expected TypeError, got %x", out.Type)
		}
	default:
		t.Fatal("no error frame emitted")
	}
}

// TestHandleEndClearsSession verifies END unregisters the active session and
// broadcasts END_BROADCAST.
func TestHandleEndClearsSession(t *testing.T) {
	sender := &Client{
		ID: "s", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	peer := &Client{
		ID: "p", Send: make(chan []byte, 4),
		Channels: map[string]struct{}{"ch": {}}, closed: make(chan struct{}),
	}
	RegisterClient(sender, []string{"ch"})
	RegisterClient(peer, []string{"ch"})
	t.Cleanup(func() {
		UnregisterClient(sender)
		UnregisterClient(peer)
	})

	body, _ := json.Marshal(StartPayload{SessionID: "s1", Channel: "ch"})
	handleStart(sender, &Frame{Type: TypeStart, Payload: body})
	<-peer.Send

	endBody, _ := json.Marshal(EndPayload{SessionID: "s1", DurationMs: 1500, TotalPackets: 75})
	handleEnd(sender, &Frame{Type: TypeEnd, Payload: endBody})

	if currentClientSession(sender.ID) != nil {
		t.Fatal("client session should be cleared after END")
	}
	select {
	case raw := <-peer.Send:
		out, _ := DecodeFrame(raw)
		if out.Type != TypeEndBroadcast {
			t.Fatalf("expected END_BROADCAST, got %x", out.Type)
		}
	default:
		t.Fatal("peer received no END_BROADCAST")
	}
}
