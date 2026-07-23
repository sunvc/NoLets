package PushToTalk

import (
	"encoding/binary"
	"encoding/json"
	"hash/fnv"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sunvc/NoLets/common"
)

// SessionMeta captures the negotiated codec parameters for one talk session.
// It is embedded in SessionBucket for future ring buffer lookups and is also
// echoed inside START_BROADCAST / REPLAY_BEGIN payloads.
type SessionMeta struct {
	Codec      string `json:"codec"`
	SampleRate int    `json:"sample_rate"`
	FrameMs    int    `json:"frame_ms"`
	Bitrate    int    `json:"bitrate"`
}

// SessionBucket represents one active talk session (from START to END).
// It is stored in the ring buffer and in ClientSession while the sender is
// still connected. See ActiveSessions / ChannelSessions for the cross-client
// indices used by SUBSCRIBE and the GC loop.
type SessionBucket struct {
	ID        string
	Channel   string
	From      string
	FromName  string
	StartedAt int64 // unix ms
	Meta      SessionMeta
	Tag       uint32 // 4-byte tag injected into every AUDIO payload so receivers can demux concurrent talkers

	// pushed guards the one-shot APNs wake-up for offline channel members.
	// It flips true the moment the first AUDIO frame is observed.
	pushed atomic.Bool

	mu         sync.Mutex
	packets    []PacketRecord
	ended      bool
	lastActive int64
}

// PacketRecord is a single AUDIO frame kept for late subscribers to replay.
type PacketRecord struct {
	Seq     uint32
	TsMs    uint32
	Payload []byte // must be a *copy*, not aliased from the WS read buffer
}

// ringCapacity returns the maximum number of AUDIO frames a bucket keeps for
// replay. Derived from WSRingBufferTTL and the session's own frame_ms so a
// 5 s window at 20 ms/frame ≈ 250 packets. If the metadata is missing we fall
// back to 20 ms and clamp so no misconfigured session ever grows unbounded.
func (s *SessionBucket) ringCapacity() int {
	ttl := common.LocalConfig.System.WSRingBufferTTL
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	frameMs := s.Meta.FrameMs
	if frameMs <= 0 {
		frameMs = 20
	}
	capacity := int(ttl / time.Millisecond) / frameMs
	if capacity < 32 {
		capacity = 32 // safety floor for tiny TTLs
	}
	return capacity
}

// Append records a packet in the bucket. Ordering is by arrival which mirrors
// the sender's seq (senders are single-threaded per session, so this holds).
// The buffer is bounded by ringCapacity — older packets are dropped from the
// head so Snapshot returns "the most recent N ≈ WSRingBufferTTL of audio".
func (s *SessionBucket) Append(rec PacketRecord) {
	s.mu.Lock()
	s.packets = append(s.packets, rec)
	capacity := s.ringCapacity()
	if drop := len(s.packets) - capacity; drop > 0 {
		s.packets = s.packets[drop:]
	}
	s.lastActive = time.Now().UnixMilli()
	s.mu.Unlock()
}

// MarkEnded flips the terminal flag and refreshes lastActive so the ring
// buffer TTL is measured from END, not from the last audio frame.
func (s *SessionBucket) MarkEnded() {
	s.mu.Lock()
	s.ended = true
	s.lastActive = time.Now().UnixMilli()
	s.mu.Unlock()
}

// Snapshot returns a copy of the packet list for safe iteration outside the
// bucket lock. Callers on the replay path use this to avoid holding the lock
// while writing on a slow socket.
func (s *SessionBucket) Snapshot() []PacketRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PacketRecord, len(s.packets))
	copy(out, s.packets)
	return out
}

// Ended reports whether END has been observed for this session.
func (s *SessionBucket) Ended() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ended
}

// LastActive returns the last-activity timestamp (unix ms) for TTL bookkeeping.
func (s *SessionBucket) LastActive() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastActive
}

// ----- Wire payloads --------------------------------------------------------

// StartPayload is the JSON body of a client → server START frame.
type StartPayload struct {
	SessionID  string `json:"session_id"`
	Channel    string `json:"channel"`
	Codec      string `json:"codec"`
	SampleRate int    `json:"sample_rate"`
	FrameMs    int    `json:"frame_ms"`
	Bitrate    int    `json:"bitrate"`
}

// StartBroadcastPayload is fanned out to every other member of the channel
// the moment a session begins.
type StartBroadcastPayload struct {
	SessionID  string `json:"session_id"`
	SessionTag uint32 `json:"session_tag"`
	Channel    string `json:"channel"`
	From       string `json:"from"`
	FromName   string `json:"from_name"`
	StartedAt  int64  `json:"started_at"`
	Codec      string `json:"codec"`
	SampleRate int    `json:"sample_rate"`
	FrameMs    int    `json:"frame_ms"`
}

// EndPayload is the client → server END frame.
type EndPayload struct {
	SessionID     string `json:"session_id"`
	DurationMs    int    `json:"duration_ms"`
	TotalPackets  int    `json:"total_packets"`
}

// EndBroadcastPayload matches EndPayload but adds channel / sender identity
// so peers can update their UI without extra lookups.
type EndBroadcastPayload struct {
	SessionID    string `json:"session_id"`
	Channel      string `json:"channel"`
	From         string `json:"from"`
	DurationMs   int    `json:"duration_ms"`
	TotalPackets int    `json:"total_packets"`
}

// SubscribePayload is the client → server SUBSCRIBE frame requesting a
// replay of buffered audio for a specific session.
type SubscribePayload struct {
	SessionID string `json:"session_id"`
}

// PresencePayload is bidirectional. Server → client uses kind "join" / "leave" /
// "update" for deltas and "snapshot" for the initial full member list a fresh
// subscriber receives right after HELLO_ACK. Client → server only ever sends
// "update" and only sets User (id/name/lat/lng/timestamp) — the server ignores
// everything else and rebroadcasts a normalised copy to the rest of the channel.
type PresencePayload struct {
	Kind      string        `json:"kind"`
	Channel   string        `json:"channel"`
	User      *PttUserResp  `json:"user,omitempty"`
	Users     []PttUserResp `json:"users,omitempty"`
	Timestamp int64         `json:"timestamp"`
}

// ----- Session helpers ------------------------------------------------------

// ActiveSessions maps session_id → *SessionBucket. Used by SUBSCRIBE to look
// up a replay target and by the GC loop to sweep expired buckets. A bucket
// stays in this map from the moment START is handled until the GC decides
// its ring-buffer window has passed.
var ActiveSessions sync.Map

// activeSessionSeq is a source of monotonic ids for logging. The wire
// protocol identifies sessions by the client-supplied session_id, but this
// atomic counter is handy for tracing.
var activeSessionSeq uint64

func nextSessionSeq() uint64 { return atomic.AddUint64(&activeSessionSeq, 1) }

// registerActiveSession stores the bucket under its client-supplied ID. If a
// bucket with the same ID somehow already existed (duplicate START) we keep
// the newer one — the sender has re-issued START by definition.
func registerActiveSession(bucket *SessionBucket) {
	ActiveSessions.Store(bucket.ID, bucket)
}

// dropActiveSession is used by the GC loop after WSRingBufferTTL has elapsed.
// Not called from handleEnd; ended buckets linger so late SUBSCRIBE callers
// can still catch the tail.
func dropActiveSession(bucket *SessionBucket) {
	ActiveSessions.CompareAndDelete(bucket.ID, bucket)
}

// setClientSession stores the currently active bucket for a user. Keyed by
// userID instead of *Client so that reconnection under the same user identity
// still finds and can replace the previous bucket.
func setClientSession(clientID string, bucket *SessionBucket) {
	ClientSession.Store(clientID, bucket)
}

// clearClientSession is a no-op if there was no active session for the client.
func clearClientSession(clientID string) {
	ClientSession.Delete(clientID)
}

// currentClientSession returns the user's active session or nil.
func currentClientSession(clientID string) *SessionBucket {
	v, ok := ClientSession.Load(clientID)
	if !ok {
		return nil
	}
	if b, ok := v.(*SessionBucket); ok {
		return b
	}
	return nil
}

// ----- Frame handlers -------------------------------------------------------

// handleStart validates the request, creates a SessionBucket, and broadcasts
// START_BROADCAST to the rest of the channel. Phase 3 will also register the
// session with the ring buffer and fire APNs to offline members.
func handleStart(client *Client, frame *Frame) {
	var p StartPayload
	if err := json.Unmarshal(frame.Payload, &p); err != nil {
		sendError(client, ErrCodeBadFrame, "bad START payload")
		return
	}
	if p.SessionID == "" || p.Channel == "" {
		sendError(client, ErrCodeBadFrame, "missing session_id / channel")
		return
	}
	if _, ok := client.Channels[p.Channel]; !ok {
		sendError(client, ErrCodeChannelDenied, "not subscribed to channel")
		return
	}

	now := time.Now().UnixMilli()

	// Idempotent: the same client may send duplicate START frames
	// (PushToTalk framework transitions, WS reconnect). If the previous
	// session for this (client, sessionID) tuple hasn't been ENDed yet,
	// return without creating a second bucket — otherwise each duplicate
	// would spawn an independent firePushForSession call.
	if existing := currentClientSession(client.ID); existing != nil {
		if existing.ID == p.SessionID {
			log.Printf("[ptt-ws] duplicate START id=%s user=%s ignored", p.SessionID, client.ID)
			return
		}
		// Different sessionID for the same client: end the old one first
		// so the bucket and push guard are consistent.
		existing.MarkEnded()
		clearClientSession(client.ID)
	}

	// Generate a 4-byte session tag from the session ID + client ID. It
	// is injected into every AUDIO frame so receiving clients can demux
	// concurrent talkers without relying on server-side preemption alone.
	// fnv.New32a is fast, deterministic, and produces well-distributed
	// 32-bit tags for the lifetime of a session.
	tagHasher := fnv.New32a()
	tagHasher.Write([]byte(p.SessionID))
	tagHasher.Write([]byte(client.ID))
	sessionTag := tagHasher.Sum32()
	if sessionTag == 0 {
		sessionTag = 1 // reserve 0 for "no tag / backward compat"
	}

	bucket := &SessionBucket{
		ID:        p.SessionID,
		Channel:   p.Channel,
		From:      client.ID,
		FromName:  client.Name,
		StartedAt: now,
		Meta: SessionMeta{
			Codec:      p.Codec,
			SampleRate: p.SampleRate,
			FrameMs:    p.FrameMs,
			Bitrate:    p.Bitrate,
		},
		Tag:        sessionTag,
		lastActive: now,
	}
	setClientSession(client.ID, bucket)
	registerActiveSession(bucket)

	// Broadcast the START notice to same-channel peers.
	notice, _ := json.Marshal(StartBroadcastPayload{
		SessionID:  bucket.ID,
		SessionTag: bucket.Tag,
		Channel:    bucket.Channel,
		From:       bucket.From,
		FromName:   bucket.FromName,
		StartedAt:  bucket.StartedAt,
		Codec:      bucket.Meta.Codec,
		SampleRate: bucket.Meta.SampleRate,
		FrameMs:    bucket.Meta.FrameMs,
	})
	FanoutRaw(bucket.Channel, client.ID, EncodeFrame(TypeStartBroadcast, 0, 0, notice))
	log.Printf("[ptt-ws] session start id=%s tag=%d user=%s ch=%s sr=%d fr=%dms",
		bucket.ID, bucket.Tag, client.ID, bucket.Channel, bucket.Meta.SampleRate, bucket.Meta.FrameMs)
}

// handleAudio forwards a single Opus packet to all channel peers. The frame
// buffer is duplicated for the ring buffer (a copy is required because the
// underlying gorilla WS read slice is reused on the next ReadMessage call).
func handleAudio(client *Client, frame *Frame, raw []byte) {
	bucket := currentClientSession(client.ID)
	if bucket == nil {
		sendError(client, ErrCodeNoActiveStart, "no active session")
		return
	}
	if bucket.Ended() {
		// Ignore late audio for a session we've already ended.
		return
	}

	// The first AUDIO frame is the trigger to wake offline channel members
	// via APNs. atomic.Bool.Swap gives us one-shot semantics without a mutex.
	if !bucket.pushed.Swap(true) {
		go firePushForSession(bucket)
	}

	// Copy the payload before stashing it — the ring buffer outlives the read
	// loop's shared buffer, so we cannot alias.
	payloadCopy := make([]byte, len(frame.Payload))
	copy(payloadCopy, frame.Payload)
	bucket.Append(PacketRecord{
		Seq:     frame.Seq,
		TsMs:    frame.Ts,
		Payload: payloadCopy,
	})

	// Build a tagged payload for receivers: [session_tag: uint32 BE | opus_data].
	// This lets the iOS client demux AUDIO frames to the correct ReceiveSession
	// even when multiple talkers are active on the same channel.
	taggedPayload := make([]byte, 4+len(frame.Payload))
	binary.BigEndian.PutUint32(taggedPayload[0:4], bucket.Tag)
	copy(taggedPayload[4:], frame.Payload)

	taggedRaw := EncodeFrame(TypeAudio, frame.Seq, frame.Ts, taggedPayload)
	FanoutRaw(bucket.Channel, client.ID, taggedRaw)
}

// handleEnd marks the session ended, broadcasts END_BROADCAST, and clears the
// per-client active-session pointer. The bucket itself is kept alive for the
// ring buffer TTL (implemented in phase 3).
func handleEnd(client *Client, frame *Frame) {
	var p EndPayload
	if err := json.Unmarshal(frame.Payload, &p); err != nil {
		sendError(client, ErrCodeBadFrame, "bad END payload")
		return
	}
	bucket := currentClientSession(client.ID)
	if bucket == nil || bucket.ID != p.SessionID {
		// End of an unknown session — likely a duplicate or a stale sender.
		return
	}
	bucket.MarkEnded()
	clearClientSession(client.ID)

	notice, _ := json.Marshal(EndBroadcastPayload{
		SessionID:    bucket.ID,
		Channel:      bucket.Channel,
		From:         bucket.From,
		DurationMs:   p.DurationMs,
		TotalPackets: p.TotalPackets,
	})
	FanoutRaw(bucket.Channel, client.ID, EncodeFrame(TypeEndBroadcast, 0, 0, notice))
	log.Printf("[ptt-ws] session end id=%s user=%s ch=%s dur=%dms packets=%d",
		bucket.ID, client.ID, bucket.Channel, p.DurationMs, p.TotalPackets)
}

// ReplayEndPayload marks the tail of a REPLAY_END frame. Kept minimal so the
// client only needs to correlate on session_id.
type ReplayEndPayload struct {
	SessionID string `json:"session_id"`
}

// handleSubscribe replays the currently-buffered audio for the session named
// in the payload. It emits REPLAY_BEGIN, then every buffered AUDIO frame in
// seq order, then REPLAY_END. Live fan-out continues in parallel — the client's
// jitter buffer dedupes on seq.
//
// If the session isn't (or is no longer) in ActiveSessions the client is told
// the replay window has expired; the caller may still get live audio if the
// session is still running elsewhere on the same channel.
func handleSubscribe(client *Client, frame *Frame) {
	var p SubscribePayload
	if err := json.Unmarshal(frame.Payload, &p); err != nil {
		sendError(client, ErrCodeBadFrame, "bad SUBSCRIBE payload")
		return
	}
	if p.SessionID == "" {
		sendError(client, ErrCodeBadFrame, "missing session_id")
		return
	}

	v, ok := ActiveSessions.Load(p.SessionID)
	if !ok {
		sendError(client, ErrCodeSessionExpired, "session unknown or replay window expired")
		return
	}
	bucket, ok := v.(*SessionBucket)
	if !ok {
		sendError(client, ErrCodeInternal, "bad session record")
		return
	}

	// REPLAY_BEGIN carries the same shape as START_BROADCAST so the receiver
	// can reuse its handleStartBroadcast code path to build state.
	begin, _ := json.Marshal(StartBroadcastPayload{
		SessionID:  bucket.ID,
		SessionTag: bucket.Tag,
		Channel:    bucket.Channel,
		From:       bucket.From,
		FromName:   bucket.FromName,
		StartedAt:  bucket.StartedAt,
		Codec:      bucket.Meta.Codec,
		SampleRate: bucket.Meta.SampleRate,
		FrameMs:    bucket.Meta.FrameMs,
	})
	client.Send <- EncodeFrame(TypeReplayBegin, 0, 0, begin)

	// Snapshot outside the fan-out lock — writing on a slow client should not
	// stall other writers.
	// Replay AUDIO frames must carry the session_tag prefix so the receiver
	// can route them before START_BROADCAST arrives during APNs wake-up.
	snap := bucket.Snapshot()
	for _, rec := range snap {
		taggedPayload := make([]byte, 4+len(rec.Payload))
		binary.BigEndian.PutUint32(taggedPayload[0:4], bucket.Tag)
		copy(taggedPayload[4:], rec.Payload)
		client.Send <- EncodeFrame(TypeAudio, rec.Seq, rec.TsMs, taggedPayload)
	}

	end, _ := json.Marshal(ReplayEndPayload{SessionID: bucket.ID})
	client.Send <- EncodeFrame(TypeReplayEnd, 0, 0, end)
	log.Printf("[ptt-ws] replay id=%s -> user=%s packets=%d", bucket.ID, client.ID, len(snap))
}

// ----- Leave ----------------------------------------------------------------

// handleLeave processes an explicit client → server LEAVE frame. Unlike a
// WS disconnect (which is transient — the user may be in background), this
// is a deliberate "I'm going off the air" signal. The user is removed from
// Channels and GlobalUsers so future talk sessions won't target them. The
// teardown path (after clientReader returns) will still call broadcastLeave
// so WS-connected peers see the departure immediately.
func handleLeave(client *Client, _ *Frame) {
	log.Printf("[ptt-ws] user=%s explicit leave", client.ID)

	ChannelLock.Lock()
	for ch := range client.Channels {
		if c, ok := Channels[ch]; ok {
			delete(c.UserIDs, client.ID)
			if len(c.UserIDs) == 0 {
				delete(Channels, ch)
			}
		}
	}
	delete(UserChannels, client.ID)
	ChannelLock.Unlock()

	GlobalUsers.Delete(client.ID)
}

// ----- Presence -------------------------------------------------------------

// broadcastPresence marshals a PresencePayload and fan-outs it to every WS
// client on `channel`, excluding `excludeUserID` (usually the actor). Kept
// small and no-op-safe so callers on the hot path don't need to check errors.
func broadcastPresence(channel, excludeUserID string, payload PresencePayload) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	FanoutRaw(channel, excludeUserID, EncodeFrame(TypePresence, 0, 0, body))
}

// sendPresenceSnapshot delivers the current WS-connected member list of
// `channel` to a single client as a `snapshot` frame. Called immediately
// after HELLO_ACK so the receiver has a full picture before any incremental
// frames arrive. Uses ChannelMembers (WS hub) rather than Channel.UserListResp
// (registered users) — offline users are deliberately excluded from the
// live snapshot because they'll get an APNs wake-up push if they need to
// hear something.
func sendPresenceSnapshot(client *Client, channel string) {
	members := ChannelMembers(channel)
	users := make([]PttUserResp, 0, len(members))
	for _, m := range members {
		if u := userRespFor(m.ID); u != nil {
			users = append(users, *u)
		}
	}

	body, err := json.Marshal(PresencePayload{
		Kind:      "snapshot",
		Channel:   channel,
		Users:     users,
		Timestamp: time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	select {
	case client.Send <- EncodeFrame(TypePresence, 0, 0, body):
	default:
		// Client send queue full — the snapshot is best-effort.
	}
}

// broadcastJoin announces a fresh subscriber to every other member of the
// channel. Called from ServeWS right after RegisterClient.
func broadcastJoin(client *Client, channel string) {
	user := userRespFor(client.ID)
	if user == nil {
		return
	}
	broadcastPresence(channel, client.ID, PresencePayload{
		Kind:      "join",
		Channel:   channel,
		User:      user,
		Timestamp: time.Now().UnixMilli(),
	})
}

// broadcastLeave announces a departure. Called from ServeWS right before
// UnregisterClient so the shared PTT registry still holds the outgoing user.
func broadcastLeave(client *Client, channel string) {
	user := userRespFor(client.ID)
	if user == nil {
		// Fall back to bare id — peers can still remove by matching on ID.
		user = &PttUserResp{ID: client.ID, Name: client.Name}
	}
	broadcastPresence(channel, client.ID, PresencePayload{
		Kind:      "leave",
		Channel:   channel,
		User:      user,
		Timestamp: time.Now().UnixMilli(),
	})
}

// userRespFor pulls a PttUserResp view of userID out of GlobalUsers. Returns
// nil when the user record is missing (which shouldn't happen for a client
// that just passed HELLO — SyncChannels stores the record — but we handle it
// defensively).
func userRespFor(userID string) *PttUserResp {
	v, ok := GlobalUsers.Load(userID)
	if !ok {
		return nil
	}
	u, ok := v.(PttUser)
	if !ok {
		return nil
	}
	return &PttUserResp{
		ID:        u.ID,
		Name:      u.Name,
		Latitude:  u.Latitude,
		Longitude: u.Longitude,
		Timestamp: u.Timestamp,
	}
}

// handlePresence processes a client → server PRESENCE frame. In this direction
// only "update" is meaningful — the client is refreshing its location. We
// rewrite the GlobalUsers record and rebroadcast a normalised copy to the
// rest of the channel(s) the client is subscribed to.
//
// Any other kind is ignored (clients are not authoritative for join/leave/
// snapshot — those come from the server itself).
func handlePresence(client *Client, frame *Frame) {
	var p PresencePayload
	if err := json.Unmarshal(frame.Payload, &p); err != nil {
		sendError(client, ErrCodeBadFrame, "bad PRESENCE payload")
		return
	}
	if p.Kind != "update" || p.User == nil {
		return
	}

	// Refresh the shared user record. Only fields the client is authoritative
	// for are copied over; name / token / host are re-read from GlobalUsers if
	// we can, otherwise we leave them empty for downstream consumers to fall
	// back on the frame's own values.
	if existing, ok := GlobalUsers.Load(client.ID); ok {
		if u, ok := existing.(PttUser); ok {
			u.Latitude = p.User.Latitude
			u.Longitude = p.User.Longitude
			u.Timestamp = p.User.Timestamp
			if p.User.Name != "" {
				u.Name = p.User.Name
			}
			GlobalUsers.Store(client.ID, u)
		}
	}

	// Rebroadcast (server-authored) to every channel this client is in.
	// The exclusion set is the sender itself.
	normalised := &PttUserResp{
		ID:        client.ID,
		Name:      p.User.Name,
		Latitude:  p.User.Latitude,
		Longitude: p.User.Longitude,
		Timestamp: p.User.Timestamp,
	}
	if normalised.Name == "" {
		normalised.Name = client.Name
	}
	for channel := range client.Channels {
		broadcastPresence(channel, client.ID, PresencePayload{
			Kind:      "update",
			Channel:   channel,
			User:      normalised,
			Timestamp: time.Now().UnixMilli(),
		})
	}
}
