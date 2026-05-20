package service

import "testing"

func TestMidiHandFiltering(t *testing.T) {
	player := NewMidiPlayerRuntime()
	player.state = newIdleMidiPlayerState()
	player.state.Mode = MidiPlaybackModePlay
	player.state.Hand = MidiHandModeLeft

	left := MidiEvent{Type: MidiEventNoteOn, Hand: MidiHandLeft}
	right := MidiEvent{Type: MidiEventNoteOn, Hand: MidiHandRight}

	if player.shouldMuteEventLocked(left) {
		t.Fatal("left-hand play mode muted a left-hand event")
	}
	if !player.shouldMuteEventLocked(right) {
		t.Fatal("left-hand play mode did not mute a right-hand event")
	}

	player.state.Mode = MidiPlaybackModeFollow
	if !player.shouldMuteEventLocked(left) {
		t.Fatal("left-hand follow mode did not mute the practice hand")
	}
	if player.shouldMuteEventLocked(right) {
		t.Fatal("left-hand follow mode muted the accompaniment hand")
	}
}

func TestBuildMidiPracticeStepsGroupsNearbyNotes(t *testing.T) {
	events := []MidiEvent{
		{Type: MidiEventNoteOn, Hand: MidiHandRight, Ms: 100, Note: 60, Velocity: 80},
		{Type: MidiEventNoteOn, Hand: MidiHandRight, Ms: 170, Note: 64, Velocity: 80},
		{Type: MidiEventNoteOn, Hand: MidiHandRight, Ms: 240, Note: 67, Velocity: 80},
	}

	steps := buildMidiPracticeSteps(events, MidiPlaybackOptions{
		HandMode:      MidiHandModeRight,
		RightMs:       1000,
		GroupWindowMs: 80,
	})

	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if len(steps[0].Notes) != 2 {
		t.Fatalf("expected first step to group 2 notes, got %d", len(steps[0].Notes))
	}
	if steps[0].Notes[0].Note != 60 || steps[0].Notes[1].Note != 64 {
		t.Fatalf("unexpected grouped notes: %+v", steps[0].Notes)
	}
}

func TestFollowAdvanceAcceptsEarlyInputAtDueTime(t *testing.T) {
	player := newTestFollowPlayer()
	defer player.mu.Unlock()

	hint, clear := player.advanceFollowLocked(30)
	if hint == nil || clear {
		t.Fatalf("expected hint before due time, hint=%v clear=%v", hint, clear)
	}

	player.acceptedNotes[60] = true
	hint, clear = player.advanceFollowLocked(100)
	if hint != nil || !clear {
		t.Fatalf("expected step clear at due time, hint=%v clear=%v", hint, clear)
	}
	if player.state.Waiting {
		t.Fatal("player entered waiting state after early accepted input")
	}
	if player.followIndex != 1 {
		t.Fatalf("expected follow index 1, got %d", player.followIndex)
	}
}

func TestFollowWaitsUntilUserPresses(t *testing.T) {
	player := newTestFollowPlayer()
	defer player.mu.Unlock()

	player.advanceFollowLocked(30)
	_, _ = player.advanceFollowLocked(100)
	if !player.state.Waiting {
		t.Fatal("player did not wait when the practice note was missing")
	}

	player.mu.Unlock()
	player.HandleUserNoteOn(60)
	player.mu.Lock()

	if player.state.Waiting {
		t.Fatal("player remained waiting after the correct note")
	}
	if player.followIndex != 1 {
		t.Fatalf("expected follow index 1, got %d", player.followIndex)
	}
}

func TestResetDoesNotClearRuntimeMidiCache(t *testing.T) {
	RuntimeMidiCache.mu.Lock()
	RuntimeMidiCache.items = map[string]*Midi{
		"cached": {UserMidi: UserMidi{ID: "cached"}},
	}
	RuntimeMidiCache.mu.Unlock()

	if err := NewMidiPlayerRuntime().Reset(); err != nil {
		t.Fatal(err)
	}

	RuntimeMidiCache.mu.RLock()
	defer RuntimeMidiCache.mu.RUnlock()
	if _, ok := RuntimeMidiCache.items["cached"]; !ok {
		t.Fatal("player reset cleared MIDI runtime cache")
	}
}

func newTestFollowPlayer() *MidiPlayerRuntime {
	player := NewMidiPlayerRuntime()
	player.mu.Lock()
	player.state = MidiPlayerState{
		Status:       MidiPlayPlaying,
		Mode:         MidiPlaybackModeFollow,
		Hand:         MidiHandModeRight,
		LeftMs:       0,
		RightMs:      1000,
		CurrentMs:    0,
		Speed:        1,
		LeadWindowMs: 80,
		MutedTracks:  map[int]bool{},
		MutedHands:   map[MidiHand]bool{},
	}
	player.followSteps = []MidiPracticeStep{
		{
			Index: 0,
			Ms:    100,
			Notes: []MidiPracticeNote{{Note: 60, Velocity: 80, Hand: MidiHandRight, Ms: 100}},
		},
	}
	return player
}
