package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const maxTurn = 5 * time.Minute
const gracePeriod = 15 * time.Second
const maxWSBytes = 4096

type Client struct {
	ws             *websocket.Conn
	hub            *Hub
	send           chan []byte
	roomID         string
	explicitLeave  bool
	user           *User
	lastMessage    time.Time
	eventsInWindow int
	windowStart    time.Time
}
type Stage struct {
	room           Room
	clients        map[*Client]bool
	queue          []uint
	performer      uint
	performerName  string
	turnEnds       time.Time
	disconnectedAt time.Time
	queueLostAt    map[uint]time.Time
	sequence       uint64
	notes          map[int]int // note -> velocity
	pedal          bool
}
type Hub struct {
	srv   *Server
	mu    sync.Mutex
	rooms map[string]*Stage
}

func NewHub(s *Server) *Hub {
	return &Hub{srv: s, rooms: map[string]*Stage{"lobby": {room: Room{ID: "lobby", Title: "大众琴房", Visibility: "public"}, clients: map[*Client]bool{}, notes: map[int]int{}, queueLostAt: map[uint]time.Time{}}}}
}
func (h *Hub) runClock() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		now := time.Now()
		for _, room := range h.rooms {
			if room.room.ID != "lobby" {
				continue
			}
			if room.performer != 0 && ((!room.turnEnds.IsZero() && now.After(room.turnEnds)) || (!room.disconnectedAt.IsZero() && now.Sub(room.disconnectedAt) > gracePeriod)) {
				h.vacateLocked(room)
			}
			// Queued players get the same short reconnect grace as the performer.
			changed := false
			filtered := room.queue[:0]
			for _, id := range room.queue {
				lost := room.queueLostAt[id]
				if !lost.IsZero() && now.Sub(lost) > gracePeriod {
					delete(room.queueLostAt, id)
					changed = true
					continue
				}
				filtered = append(filtered, id)
			}
			room.queue = filtered
			h.assignNextLocked(room)
			if changed {
				h.emitSnapshotLocked(room)
			}
		}
		h.mu.Unlock()
	}
}
func (h *Hub) presentLocked(room *Stage, id uint) bool {
	for c := range room.clients {
		if c.user != nil && c.user.ID == id {
			return true
		}
	}
	return false
}
func (h *Hub) assignNextLocked(room *Stage) {
	if room.performer != 0 || len(room.queue) == 0 {
		return
	}
	for len(room.queue) > 0 {
		id := room.queue[0]
		room.queue = room.queue[1:]
		if !h.presentLocked(room, id) {
			if room.queueLostAt[id].IsZero() {
				// No active connection and no grace window: discard an orphan.
				continue
			}
			// Preserve FIFO while the next player is briefly reconnecting.
			room.queue = append([]uint{id}, room.queue...)
			return
		}
		delete(room.queueLostAt, id)
		room.performer = id
		room.disconnectedAt = time.Time{}
		room.turnEnds = time.Now().Add(maxTurn)
		for c := range room.clients {
			if c.user != nil && c.user.ID == id {
				room.performerName = c.user.Username
				break
			}
		}
		h.emitSnapshotLocked(room)
		return
	}
	h.emitSnapshotLocked(room)
}
func (h *Hub) vacateLocked(room *Stage) {
	if room.performer == 0 {
		return
	}
	h.broadcastLocked(room, map[string]any{"type": "clear", "reason": "performer-left"})
	room.notes = make(map[int]int)
	room.pedal = false
	room.performer = 0
	room.performerName = ""
	room.turnEnds = time.Time{}
	room.disconnectedAt = time.Time{}
	h.emitSnapshotLocked(room)
}
func (h *Hub) snapshotLocked(room *Stage) map[string]any {
	queue := make([]map[string]any, 0, len(room.queue))
	for i, id := range room.queue {
		queue = append(queue, map[string]any{"position": i + 1, "userId": id})
	}
	active := make([]map[string]int, 0, len(room.notes))
	for note, vel := range room.notes {
		active = append(active, map[string]int{"note": note, "velocity": vel})
	}
	return map[string]any{"type": "snapshot", "room": room.room, "audience": len(room.clients), "queue": queue, "performerId": room.performer, "performerName": room.performerName, "turnEndsAt": room.turnEnds, "serverTime": time.Now().UTC(), "notes": active, "sustain": room.pedal, "seq": room.sequence}
}
func (h *Hub) emitSnapshotLocked(room *Stage) {
	h.broadcastLocked(room, h.snapshotLocked(room))
}

// Broadcast ordering and monotonically increasing sequence numbers are shared
// by all clients in the room; snapshot resync can discard stale events safely.
func (h *Hub) broadcastLocked(room *Stage, value any) {
	if msg, ok := value.(map[string]any); ok {
		room.sequence++
		msg["seq"] = room.sequence
		if _, exists := msg["serverTime"]; !exists {
			msg["serverTime"] = time.Now().UTC()
		}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	for c := range room.clients {
		select {
		case c.send <- data:
		default: // disconnect slow readers, never block realtime events
			go c.ws.Close()
		}
	}
}
func (h *Hub) sendSnapshot(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room := h.rooms[c.roomID]; room != nil {
		c.sendReply(h.snapshotLocked(room))
	}
}
func (h *Hub) lobbySnapshot() map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	room := h.rooms["lobby"]
	return map[string]any{"audience": len(room.clients), "performerId": room.performer, "performerName": room.performerName, "queueLength": len(room.queue), "turnEndsAt": room.turnEnds}
}
func (h *Hub) liveRoomIDs() map[string]bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := map[string]bool{}
	for id, room := range h.rooms {
		if id != "lobby" && room.room.OwnerID != 0 && h.presentLocked(room, room.room.OwnerID) {
			m[id] = true
		}
	}
	return m
}
func (h *Hub) publicLiveRooms() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := []string{}
	for id, r := range h.rooms {
		if id != "lobby" && r.room.Visibility == "public" && !r.room.Closed && h.presentLocked(r, r.room.OwnerID) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
func (h *Hub) closeRoom(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r := h.rooms[id]; r != nil {
		h.broadcastLocked(r, map[string]any{"type": "room.closed"})
		for c := range r.clients {
			go c.ws.Close()
		}
		delete(h.rooms, id)
	}
}
func (h *Hub) closeOwnerRooms(owner uint) {
	h.mu.Lock()
	ids := []string{}
	for id, r := range h.rooms {
		if id != "lobby" && r.room.OwnerID == owner {
			ids = append(ids, id)
		}
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.closeRoom(id)
	}
}
func (h *Hub) refreshRoom(updated Room) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.rooms[updated.ID]
	if r == nil {
		return
	}
	changed := r.room.Visibility != updated.Visibility || r.room.InviteHash != updated.InviteHash
	r.room = updated
	if changed {
		for c := range r.clients {
			if c.user == nil || c.user.ID != updated.OwnerID {
				go c.ws.Close()
			}
		}
	}
	h.emitSnapshotLocked(r)
}

var wsUpgrader = websocket.Upgrader{ReadBufferSize: 2048, WriteBufferSize: 2048, CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") != "" }} // exact origin allowlist enforced before upgrade
func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" || !s.origins[origin] {
		fail(w, 403, "WebSocket origin not allowed")
		return
	}
	// One connection attempt / 2 seconds on average, with a modest reconnect burst.
	if !s.limiter.allow("ws:"+clientIP(r), 30, time.Minute) {
		fail(w, 429, "连接过于频繁")
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{ws: conn, hub: s.hub, send: make(chan []byte, 128)}
	go client.writeLoop()
	client.readLoop(s)
}
func (c *Client) writeLoop() {
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	defer c.ws.Close()
	for {
		select {
		case data, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ping.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type WSMessage struct {
	Type       string `json:"type"`
	RoomID     string `json:"roomId"`
	Token      string `json:"token"`
	InviteCode string `json:"inviteCode"`
	Note       int    `json:"note"`
	Velocity   int    `json:"velocity"`
	On         bool   `json:"on"`
	ClientTime int64  `json:"clientTime,omitempty"`
}

func (c *Client) readLoop(s *Server) {
	defer func() { s.hub.detach(c); c.ws.Close() }()
	c.ws.SetReadLimit(maxWSBytes)
	_ = c.ws.SetReadDeadline(time.Now().Add(10 * time.Second)) // first join/auth must arrive promptly
	c.ws.SetPongHandler(func(string) error { return c.ws.SetReadDeadline(time.Now().Add(65 * time.Second)) })
	for {
		var msg WSMessage
		if err := c.ws.ReadJSON(&msg); err != nil {
			return
		}
		// No anonymous room events; first message must be join (including auth).
		if c.roomID == "" {
			if msg.Type != "join" {
				return
			}
			if err := s.hub.attach(c, msg); err != nil {
				// The connection closes immediately after this branch. A buffered
				// sendReply could be lost when readLoop's defer closes the socket.
				_ = c.ws.SetWriteDeadline(time.Now().Add(2 * time.Second))
				_ = c.ws.WriteJSON(map[string]any{"type": "error", "error": err.Error()})
				return
			}
			_ = c.ws.SetReadDeadline(time.Now().Add(65 * time.Second))
			continue
		}
		if msg.Type == "heartbeat" {
			c.sendReply(map[string]any{"type": "pong", "serverTime": time.Now().UTC(), "clientTime": msg.ClientTime})
			continue
		}
		if msg.Type == "room.sync" {
			s.hub.sendSnapshot(c)
			continue
		}
		if msg.Type == "room.leave" {
			c.explicitLeave = true
			return
		}
		s.hub.handle(c, msg)
	}
}
func (c *Client) sendReply(v any) {
	data, _ := json.Marshal(v)
	select {
	case c.send <- data:
	default:
	}
}
func (h *Hub) attach(c *Client, msg WSMessage) error {
	if msg.Token != "" {
		u, err := h.srv.userByToken(msg.Token)
		if err == nil {
			c.user = u
		}
	}
	id := msg.RoomID
	if id == "" {
		id = "lobby"
	}
	var dbRoom Room
	if id != "lobby" {
		if err := h.srv.db.Where("id = ? AND closed = ?", id, false).First(&dbRoom).Error; err != nil {
			return &configError{"琴房不存在"}
		}
		allowed := dbRoom.Visibility == "public" || (c.user != nil && c.user.ID == dbRoom.OwnerID) || hashToken(msg.InviteCode) == dbRoom.InviteHash
		if !allowed {
			return &configError{"私密琴房需要邀请凭证"}
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	room := h.rooms[id]
	if room == nil {
		room = &Stage{room: dbRoom, clients: map[*Client]bool{}, notes: map[int]int{}, queueLostAt: map[uint]time.Time{}}
		h.rooms[id] = room
	}
	if len(room.clients) >= 300 {
		return &configError{"琴房人数已满"}
	}
	room.clients[c] = true
	c.roomID = id
	if id == "lobby" && c.user != nil {
		delete(room.queueLostAt, c.user.ID)
		if c.user.ID == room.performer {
			room.disconnectedAt = time.Time{}
		}
		h.assignNextLocked(room)
	}
	h.emitSnapshotLocked(room)
	return nil
}
func (h *Hub) detach(c *Client) {
	if c.roomID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	room := h.rooms[c.roomID]
	if room == nil {
		return
	}
	delete(room.clients, c)
	if room.room.ID == "lobby" {
		if c.user != nil {
			id := c.user.ID
			if !h.presentLocked(room, id) {
				if c.explicitLeave {
					filtered := room.queue[:0]
					for _, queued := range room.queue {
						if queued != id {
							filtered = append(filtered, queued)
						}
					}
					room.queue = filtered
					delete(room.queueLostAt, id)
				} else {
					for _, queued := range room.queue {
						if queued == id {
							room.queueLostAt[id] = time.Now()
							break
						}
					}
				}
				if room.performer == id {
					// Clear held notes immediately, but preserve the seat for a short reconnect.
					h.broadcastLocked(room, map[string]any{"type": "clear", "reason": "disconnected"})
					room.notes = make(map[int]int)
					room.pedal = false
					if c.explicitLeave {
						h.vacateLocked(room)
						h.assignNextLocked(room)
					} else {
						room.disconnectedAt = time.Now()
					}
				}
			}
		}
	} else if room.room.OwnerID != 0 && !h.presentLocked(room, room.room.OwnerID) {
		h.broadcastLocked(room, map[string]any{"type": "clear", "reason": "owner-disconnected"})
		room.notes = make(map[int]int)
		room.pedal = false
	}
	h.emitSnapshotLocked(room)
	if len(room.clients) == 0 && room.room.ID != "lobby" {
		delete(h.rooms, c.roomID)
	}
}
func (h *Hub) handle(c *Client, msg WSMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room := h.rooms[c.roomID]
	if room == nil {
		return
	}
	id := uint(0)
	if c.user != nil {
		id = c.user.ID
	}
	switch msg.Type {
	case "queue.join":
		if room.room.ID != "lobby" || id == 0 {
			return
		}
		if room.performer == id {
			return
		}
		for _, q := range room.queue {
			if q == id {
				return
			}
		}
		if len(room.queue) >= 300 {
			return
		}
		room.queue = append(room.queue, id)
		h.assignNextLocked(room)
		h.emitSnapshotLocked(room)
	case "queue.leave":
		if room.room.ID != "lobby" || id == 0 {
			return
		}
		next := room.queue[:0]
		for _, q := range room.queue {
			if q != id {
				next = append(next, q)
			}
		}
		room.queue = next
		delete(room.queueLostAt, id)
		h.emitSnapshotLocked(room)
	case "seat.leave":
		if room.room.ID == "lobby" && room.performer == id && id != 0 {
			h.vacateLocked(room)
			h.assignNextLocked(room)
		}
	case "note.on", "note.off", "pedal":
		isPlayer := id != 0 && ((room.room.ID == "lobby" && room.performer == id && room.disconnectedAt.IsZero()) || (room.room.ID != "lobby" && room.room.OwnerID == id))
		if !isPlayer {
			return
		}
		// An expired turn must never send extra notes while waiting for the clock tick.
		if room.room.ID == "lobby" && !room.turnEnds.IsZero() && !time.Now().Before(room.turnEnds) {
			h.vacateLocked(room)
			h.assignNextLocked(room)
			return
		}
		now := time.Now()
		if now.Sub(c.windowStart) > time.Second {
			c.windowStart = now
			c.eventsInWindow = 0
		}
		c.eventsInWindow++
		if c.eventsInWindow > 120 {
			return
		}
		if msg.Type == "pedal" {
			room.pedal = msg.On
			h.broadcastLocked(room, map[string]any{"type": "pedal", "on": msg.On, "serverTime": now.UTC()})
			return
		}
		if msg.Note < 0 || msg.Note > 127 {
			return
		}
		if msg.Type == "note.on" {
			if msg.Velocity < 1 || msg.Velocity > 127 {
				return
			}
			room.notes[msg.Note] = msg.Velocity
		} else {
			delete(room.notes, msg.Note)
		}
		h.broadcastLocked(room, map[string]any{"type": msg.Type, "note": msg.Note, "velocity": msg.Velocity, "serverTime": now.UTC()})
	}
}
