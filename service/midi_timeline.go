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
	MidiID                 string             `json:"midiId"`
	DurationMs             float64            `json:"durationMs"`
	Notes                  []MidiTimelineNote `json:"notes"`
	MeasureMarkers         []float64          `json:"measureMarkers"`
	MeasureGridApproximate bool               `json:"measureGridApproximate"`
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

type midiMeterPoint struct {
	tick        float64
	numerator   uint8
	denominator uint8
}

type midiTempoPoint struct {
	tick   float64
	micros float64
	bpm    float64
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
	timeline.MeasureMarkers, timeline.MeasureGridApproximate = buildMidiMeasureMarkers(midiFile, timeline.DurationMs)

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

// buildMidiMeasureMarkers follows MIDI tempo and time-signature meta events.
// Bar positions are expressed in the same absolute milliseconds as the player.
func buildMidiMeasureMarkers(midiFile *Midi, durationMs float64) ([]float64, bool) {
	if durationMs <= 0 {
		return []float64{0}, false
	}
	if midiFile == nil || midiFile.PPQ <= 0 || midiFile.DurationTick == 0 || midiFile.Path == "" {
		return approximateMeasureMarkers(durationMs), true
	}

	reader := smf.ReadTracks(midiFile.Path)
	meters := []midiMeterPoint{{tick: 0, numerator: 4, denominator: 4}}
	tempos := []midiTempoPoint{{tick: 0, micros: 0, bpm: 120}}
	reader.Do(func(event smf.TrackEvent) {
		tick := float64(max(int64(0), event.AbsTicks))
		numerator, denominator, clocks, thirtySeconds := uint8(0), uint8(0), uint8(0), uint8(0)
		if event.Message.GetMetaTimeSig(&numerator, &denominator, &clocks, &thirtySeconds) {
			if numerator > 0 && denominator > 0 {
				meters = append(meters, midiMeterPoint{tick: tick, numerator: numerator, denominator: denominator})
			}
		}
		bpm := float64(0)
		if event.Message.GetMetaTempo(&bpm) && bpm > 0 && !math.IsNaN(bpm) && !math.IsInf(bpm, 0) {
			tempos = append(tempos, midiTempoPoint{
				tick: tick, micros: float64(max(int64(0), event.AbsMicroSeconds)), bpm: bpm,
			})
		}
	})
	if reader.Error() != nil {
		return approximateMeasureMarkers(durationMs), true
	}

	sort.SliceStable(tempos, func(i, j int) bool { return tempos[i].tick < tempos[j].tick })
	tempos = collapseTempoPoints(tempos)
	sort.SliceStable(meters, func(i, j int) bool { return meters[i].tick < meters[j].tick })
	meters = collapseMeterPoints(meters)
	if len(tempos) == 0 {
		tempos = []midiTempoPoint{{tick: 0, micros: 0, bpm: 120}}
	}
	if len(meters) == 0 {
		meters = []midiMeterPoint{{tick: 0, numerator: 4, denominator: 4}}
	}

	durationTick := float64(midiFile.DurationTick)
	markers := make([]float64, 0, int(durationMs/2000)+2)
	appendMarker := func(tick float64) {
		ms := midiTickToMilliseconds(tick, float64(midiFile.PPQ), tempos)
		ms = math.Max(0, math.Min(durationMs, ms))
		if len(markers) == 0 || ms-markers[len(markers)-1] > .5 {
			markers = append(markers, ms)
		}
	}
	for i, meter := range meters {
		start := math.Max(0, meter.tick)
		if start > durationTick {
			break
		}
		end := durationTick
		if i+1 < len(meters) && meters[i+1].tick < end {
			end = meters[i+1].tick
		}
		barTicks := float64(midiFile.PPQ) * 4 * float64(meter.numerator) / float64(meter.denominator)
		if barTicks <= 0 || math.IsNaN(barTicks) || math.IsInf(barTicks, 0) {
			continue
		}
		for barTick := start; barTick <= end+.001 && barTick <= durationTick+.001; barTick += barTicks {
			appendMarker(barTick)
			if len(markers) > 10000 {
				return approximateMeasureMarkers(durationMs), true
			}
		}
	}
	if len(markers) == 0 || markers[0] != 0 {
		markers = append([]float64{0}, markers...)
	}
	return markers, false
}

func collapseTempoPoints(points []midiTempoPoint) []midiTempoPoint {
	out := make([]midiTempoPoint, 0, len(points))
	for _, point := range points {
		if len(out) > 0 && math.Abs(out[len(out)-1].tick-point.tick) < .001 {
			out[len(out)-1] = point
		} else {
			out = append(out, point)
		}
	}
	return out
}

func collapseMeterPoints(points []midiMeterPoint) []midiMeterPoint {
	out := make([]midiMeterPoint, 0, len(points))
	for _, point := range points {
		if len(out) > 0 && math.Abs(out[len(out)-1].tick-point.tick) < .001 {
			out[len(out)-1] = point
		} else {
			out = append(out, point)
		}
	}
	return out
}

func midiTickToMilliseconds(tick, ppq float64, tempos []midiTempoPoint) float64 {
	if ppq <= 0 || len(tempos) == 0 {
		return 0
	}
	tempo := tempos[0]
	for _, candidate := range tempos {
		if candidate.tick > tick {
			break
		}
		tempo = candidate
	}
	if tempo.bpm <= 0 {
		return 0
	}
	micros := tempo.micros + (tick-tempo.tick)*60000000/(tempo.bpm*ppq)
	return math.Max(0, micros/1000)
}

// SMPTE-timed files have no quarter-note tick grid. Display a clearly marked
// approximate 4/4 grid rather than pretending those markers are authoritative.
func approximateMeasureMarkers(durationMs float64) []float64 {
	const approximateFourFourBarMs = 2000
	markers := make([]float64, 0, int(durationMs/approximateFourFourBarMs)+1)
	for ms := float64(0); ms < durationMs; ms += approximateFourFourBarMs {
		markers = append(markers, ms)
		if len(markers) > 10000 {
			break
		}
	}
	if len(markers) == 0 {
		markers = append(markers, 0)
	}
	return markers
}
