package service

import (
	"fmt"
	"time"
)

// LivePerformanceSignal is a separate stream of actual player input. Playback
// output and the sustain-delayed legacy keyboard highlights never enter it.
type LivePerformanceSignal struct {
	Type        string  `json:"type"`
	Source      string  `json:"source"`
	Channel     uint8   `json:"channel"`
	Note        uint8   `json:"note"`
	Velocity    uint8   `json:"velocity"`
	TimestampMs float64 `json:"timestampMs"`
	Sequence    uint64  `json:"sequence"`
	ID          string  `json:"id"`
	Sustained   bool    `json:"sustained"`
}

type livePerformanceKey struct {
	deviceID int
	channel  uint8
	note     uint8
}

type livePerformanceNote struct {
	key      livePerformanceKey
	id       string
	velocity uint8
	pressed  bool
}

type livePerformancePedalKey struct {
	deviceID int
	channel  uint8
}

type livePerformancePedal struct {
	damper    bool
	sostenuto bool
	captured  map[string]bool
}

// All accesses are serialized by inputEventsMu, just like handleMidiMessage.
// Retain only the latest voice per source/channel/pitch; retriggering a note
// ends its previous visual voice, so a held pedal cannot grow this map forever.
var livePerformance = struct {
	epoch    time.Time
	sequence uint64
	software bool
	notes    map[livePerformanceKey]*livePerformanceNote
	pedals   map[livePerformancePedalKey]*livePerformancePedal
}{
	epoch:  time.Now(),
	notes:  make(map[livePerformanceKey]*livePerformanceNote),
	pedals: make(map[livePerformancePedalKey]*livePerformancePedal),
}

func livePerformanceSource(deviceID int) string {
	if deviceID == interactiveDeviceID {
		return "interactive"
	}
	return fmt.Sprintf("midi:%d", deviceID)
}

func emitLivePerformance(kind string, note *livePerformanceNote, sustained bool) {
	livePerformance.sequence++
	signal := LivePerformanceSignal{
		Type:        kind,
		TimestampMs: float64(time.Since(livePerformance.epoch).Microseconds()) / 1000,
		Sequence:    livePerformance.sequence,
		Sustained:   sustained,
	}
	if note != nil {
		signal.Source = livePerformanceSource(note.key.deviceID)
		signal.Channel = note.key.channel
		signal.Note = note.key.note
		signal.Velocity = note.velocity
		signal.ID = note.id
	}
	if App != nil {
		App.Event.Emit("livePerformance", signal)
	}
}

func livePerformanceNoteOn(deviceID int, channel, note, velocity uint8) {
	key := livePerformanceKey{deviceID, channel, note}
	if previous := livePerformance.notes[key]; previous != nil {
		if previous.pressed {
			emitLivePerformance("noteOff", previous, false)
		}
		emitLivePerformance("noteEnd", previous, false)
		forgetLivePerformanceCapture(previous.id)
	}
	voice := &livePerformanceNote{
		key:      key,
		id:       fmt.Sprintf("%s:%d:%d:%d", livePerformanceSource(deviceID), channel, note, livePerformance.sequence+1),
		velocity: velocity,
		pressed:  true,
	}
	livePerformance.notes[key] = voice
	emitLivePerformance("noteOn", voice, false)
}

func livePerformanceNoteOff(deviceID int, channel, note uint8) {
	key := livePerformanceKey{deviceID, channel, note}
	voice := livePerformance.notes[key]
	if voice == nil || !voice.pressed {
		return
	}
	voice.pressed = false
	sustained := livePerformanceIsSustained(voice)
	emitLivePerformance("noteOff", voice, sustained)
	if !sustained {
		emitLivePerformance("noteEnd", voice, false)
		delete(livePerformance.notes, key)
		forgetLivePerformanceCapture(voice.id)
	}
}

func livePerformancePedalApplies(pedal livePerformancePedalKey, voice *livePerformanceNote) bool {
	return pedal.channel == voice.key.channel && (pedal.deviceID == voice.key.deviceID ||
		(voice.key.deviceID == interactiveDeviceID && pedal.deviceID >= 0))
}

func livePerformanceIsSustained(voice *livePerformanceNote) bool {
	if livePerformance.software {
		return true
	}
	for key, pedal := range livePerformance.pedals {
		if livePerformancePedalApplies(key, voice) &&
			(pedal.damper || (pedal.sostenuto && pedal.captured[voice.id])) {
			return true
		}
	}
	return false
}

func forgetLivePerformanceCapture(id string) {
	for _, pedal := range livePerformance.pedals {
		delete(pedal.captured, id)
	}
}

func releaseLivePerformanceTails() {
	for key, voice := range livePerformance.notes {
		if !voice.pressed && !livePerformanceIsSustained(voice) {
			emitLivePerformance("noteEnd", voice, false)
			delete(livePerformance.notes, key)
			forgetLivePerformanceCapture(voice.id)
		}
	}
}

func livePerformanceControlChange(deviceID int, channel, controller, value uint8) {
	if controller != 64 && controller != 66 {
		return
	}
	key := livePerformancePedalKey{deviceID, channel}
	pedal := livePerformance.pedals[key]
	if pedal == nil {
		pedal = &livePerformancePedal{captured: make(map[string]bool)}
		livePerformance.pedals[key] = pedal
	}
	down := value >= 64
	if controller == 64 {
		pedal.damper = down
	} else {
		if down && !pedal.sostenuto {
			for _, voice := range livePerformance.notes {
				if voice.pressed && livePerformancePedalApplies(key, voice) {
					pedal.captured[voice.id] = true
				}
			}
		}
		pedal.sostenuto = down
		if !down {
			clear(pedal.captured)
		}
	}
	releaseLivePerformanceTails()
}

func livePerformanceSoftwareSustain(enabled bool) {
	livePerformance.software = enabled
	releaseLivePerformanceTails()
}

func clearLivePerformance() {
	clear(livePerformance.notes)
	clear(livePerformance.pedals)
	livePerformance.software = false
	emitLivePerformance("clear", nil, false)
}
