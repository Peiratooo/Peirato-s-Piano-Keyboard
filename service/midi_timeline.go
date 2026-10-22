package service

import (
	"fmt"
	"strconv"
)

// MidiTimeline is read-only score data. Its timing uses the same song
// milliseconds as MidiPlayerState rather than the frontend animation clock.
type MidiTimeline struct {
	MidiID     string             `json:"midiId"`
	DurationMs float64            `json:"durationMs"`
	Notes      []MidiTimelineNote `json:"notes"`
}

type MidiTimelineNote struct {
	ID         string   `json:"id"`
	Note       int      `json:"note"`
	StartMs    float64  `json:"startMs"`
	EndMs      float64  `json:"endMs"`
	Velocity   int      `json:"velocity"`
	Hand       MidiHand `json:"hand"`
	TrackIndex int      `json:"trackIndex"`
	Channel    int      `json:"channel"`
}

func (k *Keyboard) GetMidiTimeline(id string) (MidiTimeline, error) {
	userMidi, ok := FindUserMidiByID(id)
	if !ok {
		return MidiTimeline{}, fmt.Errorf("MIDI 不存在")
	}
	midiFile, err := GetOrParse(userMidi)
	if err != nil {
		return MidiTimeline{}, err
	}
	return buildMidiTimeline(midiFile), nil
}

func buildMidiTimeline(midiFile *Midi) MidiTimeline {
	timeline := MidiTimeline{
		MidiID:     midiFile.ID,
		DurationMs: max(0, midiFile.DurationMs),
		Notes:      make([]MidiTimelineNote, 0, len(midiFile.Events)/2),
	}
	// Cached events are shared with the player: normalize a copy, including
	// velocity-zero NoteOn as NoteOff, before ordering simultaneous retriggers.
	events := append([]MidiEvent(nil), midiFile.Events...)
	for i := range events {
		if events[i].Type == MidiEventNoteOn && events[i].Velocity == 0 {
			events[i].Type = MidiEventNoteOff
		}
	}
	sortMidiEvents(events)
	type noteKey struct{ track, channel, note int }
	pending := make(map[noteKey][]int)
	for i, event := range events {
		if event.Type != MidiEventNoteOn && event.Type != MidiEventNoteOff {
			continue
		}
		key := noteKey{event.TrackIndex, event.Channel, event.Note}
		ms := max(0, min(event.Ms, timeline.DurationMs))
		if event.Type == MidiEventNoteOn {
			index := len(timeline.Notes)
			timeline.Notes = append(timeline.Notes, MidiTimelineNote{
				ID:         strconv.Itoa(i),
				Note:       event.Note,
				StartMs:    ms,
				EndMs:      timeline.DurationMs,
				Velocity:   event.Velocity,
				Hand:       event.Hand,
				TrackIndex: event.TrackIndex,
				Channel:    event.Channel,
			})
			pending[key] = append(pending[key], index)
			continue
		}
		queue := pending[key]
		if len(queue) == 0 {
			continue // Unmatched NoteOff does not create a visual note.
		}
		timeline.Notes[queue[0]].EndMs = ms
		if len(queue) == 1 {
			delete(pending, key)
		} else {
			pending[key] = queue[1:]
		}
	}
	// Unclosed notes already end at the song boundary. Omit zero-length notes
	// rather than inventing durations beyond the file/player's time range.
	valid := timeline.Notes[:0]
	for _, note := range timeline.Notes {
		if note.EndMs > note.StartMs {
			valid = append(valid, note)
		}
	}
	timeline.Notes = valid
	return timeline
}
