package service

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"gitlab.com/gomidi/midi/v2/smf"
)

// MidiTimeline is read-only score data. Its timing uses the same song
// milliseconds as MidiPlayerState rather than the frontend animation clock.
type MidiTimeline struct {
	MidiID     string             `json:"midiId"`
	DurationMs float64            `json:"durationMs"`
	Notes      []MidiTimelineNote `json:"notes"`
	Measures   []MidiMeasureMark `json:"measures"`
}

// Measure 1 starts at 0. Later markers are the true bar starts derived from
// the MIDI time signature, PPQ and tempo map (never equal-width approximations).
type MidiMeasureMark struct {
	Index int `json:"index"`
	Ms float64 `json:"ms"`
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
		Measures:   midiMeasureMarkers(midiFile),
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


type midiTempoMark struct {
	tick float64
	bpm float64
	startMs float64
	track int
}

type midiMeterMark struct {
	tick float64
	numerator float64
	denominator float64
	track int
}

// Build markers from musical time, not evenly divided duration. The original
// note timeline uses the same file tempo map through smf.ReadTracks.
func midiMeasureMarkers(file *Midi) []MidiMeasureMark {
	if file == nil || file.PPQ <= 0 || file.DurationTick == 0 {
		// SMPTE-timed files have no PPQ: do not present invented measure lines.
		return nil
	}
	tempoChanges := []midiTempoMark{{tick: 0, bpm: 120, track: -1}}
	meterChanges := []midiMeterMark{{tick: 0, numerator: 4, denominator: 4, track: -1}}
	reader := smf.ReadTracks(file.Path)
	reader.Do(func(ev smf.TrackEvent) {
		if ev.AbsTicks < 0 {
			return
		}
		var bpm float64
		if ev.Message.GetMetaTempo(&bpm) && bpm > 0 && !math.IsNaN(bpm) && !math.IsInf(bpm, 0) {
			tempoChanges = append(tempoChanges, midiTempoMark{
				tick: float64(ev.AbsTicks), bpm: bpm, track: ev.TrackNo,
			})
		}
		var numerator, denominator uint8
		if ev.Message.GetMetaMeter(&numerator, &denominator) &&
			numerator > 0 && denominator > 0 && denominator <= 64 {
			meterChanges = append(meterChanges, midiMeterMark{
				tick: float64(ev.AbsTicks), numerator: float64(numerator),
				denominator: float64(denominator), track: ev.TrackNo,
			})
		}
	})
	if reader.Error() != nil {
		return nil
	}
	// On conflicting simultaneous tempo/meter changes, favor the lowest
	// (conductor) track number, and retain the default only as fallback.
	sort.SliceStable(tempoChanges, func(i, j int) bool {
		if tempoChanges[i].tick == tempoChanges[j].tick {
			return tempoChanges[i].track > tempoChanges[j].track
		}
		return tempoChanges[i].tick < tempoChanges[j].tick
	})
	tempos := make([]midiTempoMark, 0, len(tempoChanges))
	for _, change := range tempoChanges {
		if len(tempos) > 0 && tempos[len(tempos)-1].tick == change.tick {
			tempos[len(tempos)-1] = change
		} else {
			tempos = append(tempos, change)
		}
	}
	ppq := float64(file.PPQ)
	for i := 1; i < len(tempos); i++ {
		last := tempos[i-1]
		tempos[i].startMs = last.startMs +
			(tempos[i].tick-last.tick)*60000/(ppq*last.bpm)
	}
	toMs := func(tick float64) float64 {
		index := sort.Search(len(tempos), func(i int) bool {return tempos[i].tick > tick}) - 1
		if index < 0 {
			index = 0
		}
		t := tempos[index]
		return t.startMs + (tick-t.tick)*60000/(ppq*t.bpm)
	}
	sort.SliceStable(meterChanges, func(i, j int) bool {
		if meterChanges[i].tick == meterChanges[j].tick {
			return meterChanges[i].track > meterChanges[j].track
		}
		return meterChanges[i].tick < meterChanges[j].tick
	})
	meters := make([]midiMeterMark, 0, len(meterChanges))
	for _, change := range meterChanges {
		if len(meters) > 0 && meters[len(meters)-1].tick == change.tick {
			meters[len(meters)-1] = change
		} else {
			meters = append(meters, change)
		}
	}
	// A time-signature change starts a new measure even if it occurs halfway
	// through the previous meter. Clamp to the actual playback duration.
	const maxMeasures = 20000
	const epsilon = 0.000001
	endTick := float64(file.DurationTick)
	markers := make([]MidiMeasureMark, 0, min(int(endTick/(4*ppq))+2, 4096))
	cursor := 0.0
	meterIndex := 0
	for index := 1; index <= maxMeasures && cursor < endTick-epsilon; index++ {
		for meterIndex+1 < len(meters) && meters[meterIndex+1].tick <= cursor+epsilon {
			meterIndex++
		}
		ms := toMs(cursor)
		if ms >= file.DurationMs-epsilon {
			break
		}
		if len(markers) == 0 || ms > markers[len(markers)-1].Ms+epsilon {
			markers = append(markers, MidiMeasureMark{Index: index, Ms: math.Max(0, ms)})
		}
		current := meters[meterIndex]
		measureTicks := ppq * 4 * current.numerator / current.denominator
		if measureTicks <= epsilon {
			break
		}
		next := cursor + measureTicks
		if meterIndex+1 < len(meters) {
			changeTick := meters[meterIndex+1].tick
			if changeTick > cursor+epsilon && changeTick < next {
				next = changeTick
			}
		}
		if next <= cursor+epsilon {
			break
		}
		cursor = next
	}
	return markers
}
