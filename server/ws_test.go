package main

import (
	"testing"
	"time"
)

func TestLobbySeatFIFOAndYield(t *testing.T) {
	h := NewHub(&Server{})
	a := &Client{roomID: "lobby", user: &User{ID: 1, Username: "A"}, send: make(chan []byte, 30)}
	b := &Client{roomID: "lobby", user: &User{ID: 2, Username: "B"}, send: make(chan []byte, 30)}
	room := h.rooms["lobby"]
	room.clients[a] = true
	room.clients[b] = true
	h.handle(a, WSMessage{Type: "queue.join"})
	if room.performer != 1 || time.Until(room.turnEnds) > maxTurn {
		t.Fatalf("first player not assigned: %+v", room)
	}
	h.handle(b, WSMessage{Type: "queue.join"})
	h.handle(a, WSMessage{Type: "seat.leave"})
	if room.performer != 2 {
		t.Fatalf("expected second player, got %d", room.performer)
	}
	h.handle(a, WSMessage{Type: "note.on", Note: 60, Velocity: 80})
	if len(room.notes) != 0 {
		t.Fatal("spectator injected a note")
	}
	h.handle(b, WSMessage{Type: "note.on", Note: 60, Velocity: 80})
	if room.notes[60] != 80 {
		t.Fatal("performer did not trigger note")
	}
	h.handle(b, WSMessage{Type: "seat.leave"})
	if len(room.notes) != 0 {
		t.Fatal("held notes not cleared on yield")
	}
}

func TestQueueKeepsPositionWithinReconnectGrace(t *testing.T) {
	h := NewHub(&Server{})
	r := h.rooms["lobby"]
	a := &Client{roomID: "lobby", user: &User{ID: 1, Username: "A"}, send: make(chan []byte, 50)}
	b := &Client{roomID: "lobby", user: &User{ID: 2, Username: "B"}, send: make(chan []byte, 50)}
	r.clients[a], r.clients[b] = true, true
	h.handle(a, WSMessage{Type: "queue.join"})
	h.handle(b, WSMessage{Type: "queue.join"})
	h.detach(b) // network drop (not room.leave)
	if len(r.queue) != 1 || r.queue[0] != 2 || r.queueLostAt[2].IsZero() {
		t.Fatalf("queued user lost place: %+v", r.queue)
	}
	h.handle(a, WSMessage{Type: "seat.leave"})
	if r.performer != 0 || len(r.queue) != 1 {
		t.Fatal("grace must preserve FIFO without assigning offline user")
	}
	b2 := &Client{roomID: "lobby", user: &User{ID: 2, Username: "B"}, send: make(chan []byte, 50)}
	r.clients[b2] = true
	delete(r.queueLostAt, 2)
	h.mu.Lock()
	h.assignNextLocked(r)
	h.mu.Unlock()
	if r.performer != 2 {
		t.Fatal("reconnected queued user did not get the seat")
	}
}

func TestExplicitQueueLeaveRemovesPlace(t *testing.T) {
	h := NewHub(&Server{})
	r := h.rooms["lobby"]
	a := &Client{roomID: "lobby", user: &User{ID: 1}, send: make(chan []byte, 50)}
	b := &Client{roomID: "lobby", user: &User{ID: 2}, send: make(chan []byte, 50), explicitLeave: true}
	r.clients[a], r.clients[b] = true, true
	h.handle(a, WSMessage{Type: "queue.join"})
	h.handle(b, WSMessage{Type: "queue.join"})
	h.detach(b)
	if len(r.queue) != 0 {
		t.Fatal("explicit leave should remove the queued user immediately")
	}
}

func TestExpiredSeatCannotEmitNoteBeforeClockTick(t *testing.T) {
	h := NewHub(&Server{})
	r := h.rooms["lobby"]
	a := &Client{roomID: "lobby", user: &User{ID: 7}, send: make(chan []byte, 50)}
	r.clients[a] = true
	h.handle(a, WSMessage{Type: "queue.join"})
	r.turnEnds = time.Now().Add(-time.Second)
	h.handle(a, WSMessage{Type: "note.on", Note: 60, Velocity: 100})
	if r.performer != 0 || len(r.notes) != 0 {
		t.Fatal("expired performer must be vacated before processing event")
	}
}

func TestRoomSequenceMonotonic(t *testing.T) {
	h := NewHub(&Server{})
	r := h.rooms["lobby"]
	a := &Client{roomID: "lobby", user: &User{ID: 1}, send: make(chan []byte, 50)}
	r.clients[a] = true
	h.handle(a, WSMessage{Type: "queue.join"})
	before := r.sequence
	h.handle(a, WSMessage{Type: "note.on", Note: 64, Velocity: 76})
	if r.sequence <= before {
		t.Fatal("room events must advance sequence")
	}
}
