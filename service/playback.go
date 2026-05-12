package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	PlayerStatusIdle     = "idle"
	PlayerStatusReady    = "ready"
	PlayerStatusPlaying  = "playing"
	PlayerStatusPaused   = "paused"
	PlayerStatusStopped  = "stopped"
	PlayerStatusFinished = "finished"
)

// MidiPlaybackState 是后端播放器暴露给前端的轻量状态。
// 前端只需要用它渲染进度、按钮状态和错误信息；真正调度由 Go 负责。
type MidiPlaybackState struct {
	FileName     string  `json:"fileName"`
	Duration     float64 `json:"duration"`
	CurrentTime  float64 `json:"currentTime"`
	PlaybackRate float64 `json:"playbackRate"`
	Status       string  `json:"status"`
	TotalNotes   int     `json:"totalNotes"`
	Error        string  `json:"error"`
}

type midiPlaybackRuntime struct {
	mu sync.Mutex

	file        MidiFileInfo
	notes       []MidiNote
	nextIndex   int
	status      string
	currentTime float64
	rate        float64
	errorText   string

	cancel          context.CancelFunc
	activeNotes     map[string]MidiNote
	activeMidiCount map[uint8]int
}

var playback = &midiPlaybackRuntime{
	status:          PlayerStatusIdle,
	rate:            1,
	activeNotes:     make(map[string]MidiNote),
	activeMidiCount: make(map[uint8]int),
}

// LoadMidiFileBase64 是正式的 Wails 绑定入口：前端把 MIDI 文件读成 base64，交给 Go 解析并缓存。
// 后续播放、暂停、进度、跟弹分组都可以基于这份缓存完成，避免 WebView 主线程承担大文件计算。
func (k *Keyboard) LoadMidiFileBase64(fileName string, encoded string) (MidiFileInfo, error) {
	parsed, err := k.ParseMidiFileBase64(fileName, encoded)
	if err != nil {
		playback.setError(err)
		return MidiFileInfo{}, err
	}
	playback.load(parsed)
	emitMidiPlaybackLoaded(parsed)
	emitMidiPlaybackState(playback.state())
	return parsed, nil
}

func (k *Keyboard) StartMidiPlayback() MidiPlaybackState {
	playback.start(k)
	state := playback.state()
	emitMidiPlaybackState(state)
	return state
}

func (k *Keyboard) PauseMidiPlayback() MidiPlaybackState {
	playback.pause(k)
	state := playback.state()
	emitMidiPlaybackState(state)
	return state
}

func (k *Keyboard) StopMidiPlayback() MidiPlaybackState {
	playback.stop(k, true, true)
	state := playback.state()
	emitMidiPlaybackState(state)
	return state
}

func (k *Keyboard) SeekMidiPlayback(seconds float64) MidiPlaybackState {
	playback.seek(k, seconds)
	state := playback.state()
	emitMidiPlaybackState(state)
	return state
}

func (k *Keyboard) SetMidiPlaybackRate(rate float64) MidiPlaybackState {
	playback.setRate(k, rate)
	state := playback.state()
	emitMidiPlaybackState(state)
	return state
}

func (k *Keyboard) GetMidiPlaybackState() MidiPlaybackState {
	return playback.state()
}

func (p *midiPlaybackRuntime) load(file MidiFileInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cancelLocked()
	p.releaseActiveNotesLocked(&Keyboard{})
	p.clearPlaybackKeysLocked()

	p.file = file
	p.notes = append([]MidiNote(nil), file.Notes...)
	sort.SliceStable(p.notes, func(i, j int) bool {
		if p.notes[i].Start == p.notes[j].Start {
			return p.notes[i].Midi < p.notes[j].Midi
		}
		return p.notes[i].Start < p.notes[j].Start
	})
	p.nextIndex = 0
	p.currentTime = 0
	p.rate = normalizePlaybackRate(p.rate)
	p.status = PlayerStatusReady
	p.errorText = ""
}

func (p *midiPlaybackRuntime) start(k *Keyboard) {
	p.mu.Lock()
	if len(p.notes) == 0 || p.status == PlayerStatusPlaying {
		p.mu.Unlock()
		return
	}
	if p.currentTime >= p.file.Duration {
		p.currentTime = 0
	}
	p.cancelLocked()
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.status = PlayerStatusPlaying
	p.nextIndex = p.findNextNoteIndexLocked(p.currentTime)
	startPosition := p.currentTime
	rate := normalizePlaybackRate(p.rate)
	p.rate = rate
	p.mu.Unlock()

	go p.run(ctx, k, startPosition, rate)
}

func (p *midiPlaybackRuntime) pause(k *Keyboard) {
	p.mu.Lock()
	if p.status != PlayerStatusPlaying {
		p.mu.Unlock()
		return
	}
	p.cancelLocked()
	p.releaseActiveNotesLocked(k)
	p.status = PlayerStatusPaused
	p.mu.Unlock()
}

func (p *midiPlaybackRuntime) stop(k *Keyboard, resetTime bool, markStopped bool) {
	p.mu.Lock()
	p.cancelLocked()
	p.releaseActiveNotesLocked(k)
	p.clearPlaybackKeysLocked()
	p.nextIndex = 0
	if resetTime {
		p.currentTime = 0
	}
	if markStopped {
		if len(p.notes) > 0 {
			p.status = PlayerStatusStopped
		} else {
			p.status = PlayerStatusIdle
		}
	}
	p.mu.Unlock()

	k.AllNotesOff()
}

func (p *midiPlaybackRuntime) seek(k *Keyboard, seconds float64) {
	p.mu.Lock()
	wasPlaying := p.status == PlayerStatusPlaying
	p.cancelLocked()
	p.releaseActiveNotesLocked(k)
	p.clearPlaybackKeysLocked()
	p.currentTime = clampFloat(seconds, 0, p.file.Duration)
	p.nextIndex = p.findNextNoteIndexLocked(p.currentTime)
	if wasPlaying {
		p.status = PlayerStatusPaused
	}
	p.mu.Unlock()

	if wasPlaying {
		p.start(k)
	}
}

func (p *midiPlaybackRuntime) setRate(k *Keyboard, rate float64) {
	p.mu.Lock()
	wasPlaying := p.status == PlayerStatusPlaying
	p.cancelLocked()
	p.releaseActiveNotesLocked(k)
	p.rate = normalizePlaybackRate(rate)
	if wasPlaying {
		p.status = PlayerStatusPaused
	}
	p.mu.Unlock()

	if wasPlaying {
		p.start(k)
	}
}

func (p *midiPlaybackRuntime) run(ctx context.Context, k *Keyboard, startPosition float64, rate float64) {
	startedAt := time.Now()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentTime := startPosition + time.Since(startedAt).Seconds()*rate
			dueNotes, releaseNotes, finished := p.collectPlaybackWork(currentTime)

			for _, note := range dueNotes {
				k.KeyboardPlayWithVelocity(note.Midi, note.Velocity)
			}
			for _, note := range releaseNotes {
				k.KeyboardStop(note.Midi)
			}

			if len(dueNotes) > 0 || len(releaseNotes) > 0 || int(currentTime*20)%5 == 0 {
				emitMidiPlaybackState(p.state())
			}

			if finished {
				p.finish(k)
				return
			}
		}
	}
}

func (p *midiPlaybackRuntime) collectPlaybackWork(currentTime float64) ([]MidiNote, []MidiNote, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != PlayerStatusPlaying {
		return nil, nil, false
	}

	p.currentTime = clampFloat(currentTime, 0, p.file.Duration)
	dueNotes := make([]MidiNote, 0)
	releaseNotes := make([]MidiNote, 0)

	for p.nextIndex < len(p.notes) && p.notes[p.nextIndex].Start <= p.currentTime {
		note := p.notes[p.nextIndex]
		p.activeNotes[note.ID] = note
		count := p.activeMidiCount[note.Midi]
		p.activeMidiCount[note.Midi] = count + 1
		if count == 0 {
			emitPlaybackKey(note.Midi, true)
		}
		dueNotes = append(dueNotes, note)
		p.nextIndex++
	}

	for id, note := range p.activeNotes {
		if note.End <= p.currentTime {
			delete(p.activeNotes, id)
			if p.decreaseMidiCountLocked(note.Midi) {
				emitPlaybackKey(note.Midi, false)
			}
			releaseNotes = append(releaseNotes, note)
		}
	}

	finished := p.file.Duration > 0 && p.currentTime >= p.file.Duration
	return dueNotes, releaseNotes, finished
}

func (p *midiPlaybackRuntime) finish(k *Keyboard) {
	p.mu.Lock()
	p.cancelLocked()
	p.releaseActiveNotesLocked(k)
	p.clearPlaybackKeysLocked()
	p.currentTime = p.file.Duration
	p.status = PlayerStatusFinished
	p.mu.Unlock()

	k.AllNotesOff()
	emitMidiPlaybackState(p.state())
}

func (p *midiPlaybackRuntime) state() MidiPlaybackState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return MidiPlaybackState{
		FileName:     p.file.Name,
		Duration:     p.file.Duration,
		CurrentTime:  p.currentTime,
		PlaybackRate: p.rate,
		Status:       p.status,
		TotalNotes:   len(p.notes),
		Error:        p.errorText,
	}
}

func (p *midiPlaybackRuntime) setError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorText = fmt.Sprint(err)
	p.status = PlayerStatusIdle
}

func (p *midiPlaybackRuntime) cancelLocked() {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
}

func (p *midiPlaybackRuntime) releaseActiveNotesLocked(k *Keyboard) {
	for _, note := range p.activeNotes {
		k.KeyboardStop(note.Midi)
	}
	p.activeNotes = make(map[string]MidiNote)
	p.activeMidiCount = make(map[uint8]int)
}

func (p *midiPlaybackRuntime) clearPlaybackKeysLocked() {
	emitPlaybackClear()
}

func (p *midiPlaybackRuntime) decreaseMidiCountLocked(midi uint8) bool {
	count := p.activeMidiCount[midi]
	if count <= 1 {
		delete(p.activeMidiCount, midi)
		return true
	}
	p.activeMidiCount[midi] = count - 1
	return false
}

func (p *midiPlaybackRuntime) findNextNoteIndexLocked(seconds float64) int {
	index := sort.Search(len(p.notes), func(i int) bool {
		return p.notes[i].Start >= seconds
	})
	return index
}

func emitMidiPlaybackLoaded(file MidiFileInfo) {
	if App != nil {
		App.Event.Emit("midiPlayerLoaded", file)
	}
}

func emitMidiPlaybackState(state MidiPlaybackState) {
	if App != nil {
		App.Event.Emit("midiPlayerState", state)
	}
}

func emitPlaybackKey(midi uint8, pressed bool) {
	if App != nil {
		App.Event.Emit("playbackKey", map[string]any{
			"midi":    midi,
			"pressed": pressed,
		})
	}
}

func emitPlaybackClear() {
	if App != nil {
		App.Event.Emit("playbackClear")
	}
}

func normalizePlaybackRate(rate float64) float64 {
	if rate <= 0 {
		return 1
	}
	return clampFloat(rate, 0.25, 3)
}

func clampFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
