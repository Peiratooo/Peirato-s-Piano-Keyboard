package service

import (
	"testing"
	"time"
)

func TestNormalizeMidiPlaybackOptions(t *testing.T) {
	options := normalizeMidiPlaybackOptions(MidiPlaybackOptions{
		Mode:     "invalid",
		HandMode: "invalid",
		LeftMs:   -100,
		RightMs:  0,
		Speed:    9,
	}, 1000)

	if options.Mode != MidiPlaybackModePlay {
		t.Fatalf("expected invalid mode to normalize to play, got %q", options.Mode)
	}
	if options.HandMode != MidiHandModeBoth {
		t.Fatalf("expected invalid hand mode to normalize to both, got %q", options.HandMode)
	}
	if options.LeftMs != 0 {
		t.Fatalf("expected left anchor to clamp to 0, got %v", options.LeftMs)
	}
	if options.RightMs != 1000 {
		t.Fatalf("expected empty right anchor to default to duration, got %v", options.RightMs)
	}
	if options.Speed != 3 {
		t.Fatalf("expected speed to clamp to 3, got %v", options.Speed)
	}
	if options.LeadWindowMs != defaultMidiWindowMs {
		t.Fatalf("expected default lead window, got %v", options.LeadWindowMs)
	}
	if options.GroupWindowMs != defaultMidiWindowMs {
		t.Fatalf("expected default group window, got %v", options.GroupWindowMs)
	}

	options = normalizeMidiPlaybackOptions(MidiPlaybackOptions{
		LeftMs:  950,
		RightMs: 900,
		Speed:   0.1,
	}, 1000)

	if options.RightMs <= options.LeftMs {
		t.Fatalf("expected right anchor after left anchor, got left=%v right=%v", options.LeftMs, options.RightMs)
	}
	if options.Speed != 0.25 {
		t.Fatalf("expected speed to clamp to 0.25, got %v", options.Speed)
	}
}

func TestTickLoopsOrStopsAtRightAnchor(t *testing.T) {
	t.Run("loop enabled", func(t *testing.T) {
		player := NewMidiPlayerRuntime()
		defer player.Stop()

		if err := player.Start(testMidiFile(), MidiPlaybackOptions{
			MidiID:  "test",
			LeftMs:  0,
			RightMs: 100,
			Speed:   1,
			Loop:    true,
		}); err != nil {
			t.Fatal(err)
		}

		player.mu.Lock()
		player.baseTime = time.Now().Add(-200 * time.Millisecond)
		player.mu.Unlock()

		player.Tick()
		state := player.State()
		if state.Status != MidiPlayPlaying {
			t.Fatalf("expected player to keep playing when loop is enabled, got %q", state.Status)
		}
		if state.CurrentMs != state.LeftMs {
			t.Fatalf("expected loop to reset current time to left anchor, got current=%v left=%v", state.CurrentMs, state.LeftMs)
		}
	})

	t.Run("loop disabled", func(t *testing.T) {
		player := NewMidiPlayerRuntime()
		defer player.Stop()

		if err := player.Start(testMidiFile(), MidiPlaybackOptions{
			MidiID:  "test",
			LeftMs:  0,
			RightMs: 100,
			Speed:   1,
			Loop:    false,
		}); err != nil {
			t.Fatal(err)
		}

		player.mu.Lock()
		player.baseTime = time.Now().Add(-200 * time.Millisecond)
		player.mu.Unlock()

		player.Tick()
		state := player.State()
		if state.Status != MidiPlayIdle {
			t.Fatalf("expected player to stop when loop is disabled, got %q", state.Status)
		}
		if state.CurrentMs != state.LeftMs {
			t.Fatalf("expected stop to reset current time to left anchor, got current=%v left=%v", state.CurrentMs, state.LeftMs)
		}
	})
}

func TestSetOptionsRebuildsFollowStepsAndPreservesStateOnInvalidFollowMode(t *testing.T) {
	player := NewMidiPlayerRuntime()
	defer player.Stop()

	if err := player.Start(testMidiFile(), MidiPlaybackOptions{
		MidiID:   "test",
		Mode:     MidiPlaybackModePlay,
		HandMode: MidiHandModeBoth,
		LeftMs:   0,
		RightMs:  1000,
		Speed:    1,
	}); err != nil {
		t.Fatal(err)
	}

	if err := player.SetOptions(MidiPlaybackOptions{
		MidiID:   "test",
		Mode:     MidiPlaybackModeFollow,
		HandMode: MidiHandModeRight,
		LeftMs:   0,
		RightMs:  1000,
		Speed:    1.5,
		Loop:     true,
	}); err != nil {
		t.Fatal(err)
	}

	state := player.State()
	if state.Mode != MidiPlaybackModeFollow {
		t.Fatalf("expected follow mode, got %q", state.Mode)
	}
	if state.Hand != MidiHandModeRight {
		t.Fatalf("expected right hand mode, got %q", state.Hand)
	}
	if state.Speed != 1.5 {
		t.Fatalf("expected speed update, got %v", state.Speed)
	}
	if !state.Loop {
		t.Fatal("expected loop update")
	}
	if state.CurrentStep == nil || len(state.CurrentStep.Notes) == 0 {
		t.Fatal("expected follow steps to be rebuilt")
	}

	if err := player.SetOptions(MidiPlaybackOptions{
		MidiID:   "test",
		Mode:     MidiPlaybackModeFollow,
		HandMode: MidiHandModeLeft,
		LeftMs:   0,
		RightMs:  100,
		Speed:    1,
	}); err == nil {
		t.Fatal("expected invalid follow range to return an error")
	}

	state = player.State()
	if state.Mode != MidiPlaybackModeFollow || state.Hand != MidiHandModeRight {
		t.Fatalf("expected failed update to preserve previous mode and hand, got mode=%q hand=%q", state.Mode, state.Hand)
	}
	if state.RightMs != 1000 {
		t.Fatalf("expected failed update to preserve previous range, got right=%v", state.RightMs)
	}
}

func testMidiFile() *Midi {
	return &Midi{
		UserMidi: UserMidi{
			ID:         "test",
			Name:       "Test MIDI",
			DurationMs: 1000,
			TrackCount: 2,
		},
		Events: []MidiEvent{
			{Type: MidiEventNoteOn, Ms: 200, Note: 60, Velocity: 80, Hand: MidiHandLeft, TrackIndex: 0},
			{Type: MidiEventNoteOff, Ms: 300, Note: 60, Hand: MidiHandLeft, TrackIndex: 0},
			{Type: MidiEventNoteOn, Ms: 500, Note: 72, Velocity: 80, Hand: MidiHandRight, TrackIndex: 1},
			{Type: MidiEventNoteOff, Ms: 600, Note: 72, Hand: MidiHandRight, TrackIndex: 1},
		},
	}
}
