package service

import (
	"errors"
	"sort"
	"sync"
	"time"
)

const (
	defaultMidiWindowMs = 100.0
	playerTickInterval  = 3 * time.Millisecond
	stateEmitInterval   = 50 * time.Millisecond
)

type MidiPlayStatus string

const (
	MidiPlayIdle    MidiPlayStatus = "idle"
	MidiPlayPlaying MidiPlayStatus = "playing"
	MidiPlayPaused  MidiPlayStatus = "paused"
	MidiPlayError   MidiPlayStatus = "error"
)

type MidiPlaybackMode string

const (
	MidiPlaybackModePlay   MidiPlaybackMode = "play"
	MidiPlaybackModeFollow MidiPlaybackMode = "follow"
)

type MidiHandMode string

const (
	MidiHandModeLeft  MidiHandMode = "left"
	MidiHandModeRight MidiHandMode = "right"
	MidiHandModeBoth  MidiHandMode = "both"
)

type MidiPlaybackOptions struct {
	MidiID string `json:"midiId"`

	Mode     MidiPlaybackMode `json:"mode"`
	HandMode MidiHandMode     `json:"handMode"`

	LeftMs  float64 `json:"leftMs"`
	RightMs float64 `json:"rightMs"`
	Speed   float64 `json:"speed"`
	Loop    bool    `json:"loop"`

	LeadWindowMs  float64 `json:"leadWindowMs"`
	GroupWindowMs float64 `json:"groupWindowMs"`
}

type MidiPlayerState struct {
	MidiID        string            `json:"midiId"`
	Status        MidiPlayStatus    `json:"status"`
	Mode          MidiPlaybackMode  `json:"mode"`
	Hand          MidiHandMode      `json:"handMode"`
	DurationMs    float64           `json:"durationMs"`
	CurrentMs     float64           `json:"currentMs"`
	LeftMs        float64           `json:"leftMs"`
	RightMs       float64           `json:"rightMs"`
	Speed         float64           `json:"speed"`
	Loop          bool              `json:"loop"`
	LeadWindowMs  float64           `json:"leadWindowMs"`
	GroupWindowMs float64           `json:"groupWindowMs"`
	Waiting       bool              `json:"waiting"`
	CurrentStep   *MidiPracticeStep `json:"currentStep,omitempty"`
	MutedTracks   map[int]bool      `json:"mutedTracks"`
	MutedHands    map[MidiHand]bool `json:"mutedHands"`
	Error         string            `json:"error,omitempty"`
}

type MidiPracticeNote struct {
	Note       int      `json:"note"`
	Velocity   int      `json:"velocity"`
	Hand       MidiHand `json:"hand"`
	TrackIndex int      `json:"trackIndex"`
	Ms         float64  `json:"ms"`
}

type MidiPracticeStep struct {
	Index int                `json:"index"`
	Ms    float64            `json:"ms"`
	Notes []MidiPracticeNote `json:"notes"`
}

type MidiVisualEvent struct {
	Note     int      `json:"note"`
	Velocity int      `json:"velocity"`
	Channel  uint8    `json:"channel"`
	Hand     MidiHand `json:"hand"`
	Source   string   `json:"source"`
	Active   bool     `json:"active"`
}

type MidiPlayerRuntime struct {
	mu sync.Mutex

	events []MidiEvent
	state  MidiPlayerState

	eventIndex int

	baseMs   float64
	baseTime time.Time

	followSteps   []MidiPracticeStep
	followIndex   int
	hintActive    bool
	acceptedNotes map[int]bool

	lastStateEmit time.Time

	ticker *time.Ticker
	stopCh chan struct{}
}

var MidiPlayer = NewMidiPlayerRuntime()

func NewMidiPlayerRuntime() *MidiPlayerRuntime {
	return &MidiPlayerRuntime{
		events:        make([]MidiEvent, 0),
		followSteps:   make([]MidiPracticeStep, 0),
		acceptedNotes: make(map[int]bool),
		state:         newIdleMidiPlayerState(),
		stopCh:        make(chan struct{}),
	}
}

func newIdleMidiPlayerState() MidiPlayerState {
	return MidiPlayerState{
		Status:        MidiPlayIdle,
		Mode:          MidiPlaybackModePlay,
		Hand:          MidiHandModeBoth,
		Speed:         1.0,
		LeadWindowMs:  defaultMidiWindowMs,
		GroupWindowMs: defaultMidiWindowMs,
		MutedTracks:   make(map[int]bool),
		MutedHands:    make(map[MidiHand]bool),
	}
}

func (p *MidiPlayerRuntime) PlayMidi(midiFile *Midi) error {
	return p.Start(midiFile, MidiPlaybackOptions{
		MidiID:        midiFile.ID,
		Mode:          MidiPlaybackModePlay,
		HandMode:      MidiHandModeBoth,
		Speed:         1,
		LeadWindowMs:  defaultMidiWindowMs,
		GroupWindowMs: defaultMidiWindowMs,
	})
}

func (p *MidiPlayerRuntime) Start(midiFile *Midi, options MidiPlaybackOptions) error {
	if midiFile == nil {
		return errors.New("MIDI 为空")
	}
	if len(midiFile.Events) == 0 {
		return errors.New("MIDI 事件为空")
	}

	events := append([]MidiEvent(nil), midiFile.Events...)
	sortMidiEvents(events)
	options = normalizeMidiPlaybackOptions(options, midiFile.DurationMs)
	if options.MidiID == "" {
		options.MidiID = midiFile.ID
	}

	followSteps := make([]MidiPracticeStep, 0)
	if options.Mode == MidiPlaybackModeFollow {
		followSteps = buildMidiPracticeSteps(events, options)
		if len(followSteps) == 0 {
			return errors.New("当前范围和练习手没有可练习音符")
		}
	}

	p.mu.Lock()
	p.stopLocked()
	p.events = events
	p.eventIndex = findMidiEventIndexByMs(p.events, options.LeftMs)
	p.followSteps = followSteps
	p.followIndex = 0
	p.hintActive = false
	p.acceptedNotes = make(map[int]bool)
	p.baseMs = options.LeftMs
	p.baseTime = time.Now()
	p.lastStateEmit = time.Time{}

	p.state = MidiPlayerState{
		MidiID:        options.MidiID,
		Status:        MidiPlayPlaying,
		Mode:          options.Mode,
		Hand:          options.HandMode,
		DurationMs:    midiFile.DurationMs,
		CurrentMs:     options.LeftMs,
		LeftMs:        options.LeftMs,
		RightMs:       options.RightMs,
		Speed:         options.Speed,
		Loop:          options.Loop,
		LeadWindowMs:  options.LeadWindowMs,
		GroupWindowMs: options.GroupWindowMs,
		MutedTracks:   make(map[int]bool),
		MutedHands:    make(map[MidiHand]bool),
	}

	if options.Mode == MidiPlaybackModeFollow && len(followSteps) > 0 {
		p.state.CurrentStep = cloneMidiPracticeStep(&followSteps[0])
	}

	p.startLocked()
	state := p.state
	p.mu.Unlock()

	AllSynthNotesOff()
	emitMidiVisualClear()
	emitMidiPlayerState(state)
	return nil
}

func normalizeMidiPlaybackOptions(options MidiPlaybackOptions, durationMs float64) MidiPlaybackOptions {
	if options.Mode != MidiPlaybackModeFollow {
		options.Mode = MidiPlaybackModePlay
	}
	if options.HandMode != MidiHandModeLeft && options.HandMode != MidiHandModeRight {
		options.HandMode = MidiHandModeBoth
	}
	if options.Speed <= 0 {
		options.Speed = 1
	}
	options.Speed = clampFloat(options.Speed, 0.25, 3.0)
	if options.LeadWindowMs <= 0 {
		options.LeadWindowMs = defaultMidiWindowMs
	}
	if options.GroupWindowMs <= 0 {
		options.GroupWindowMs = defaultMidiWindowMs
	}
	options.LeftMs = clampFloat(options.LeftMs, 0, durationMs)
	if options.RightMs <= 0 {
		options.RightMs = durationMs
	}
	options.RightMs = clampFloat(options.RightMs, 0, durationMs)
	if options.RightMs <= options.LeftMs {
		options.RightMs = options.LeftMs + 100
		if options.RightMs > durationMs {
			options.RightMs = durationMs
		}
	}
	return options
}

func (p *MidiPlayerRuntime) Tick() {
	var dueEvents []MidiEvent
	var state MidiPlayerState
	var hintStep *MidiPracticeStep
	var clearHint bool

	var emitState bool

	p.mu.Lock()
	if p.state.Status != MidiPlayPlaying {
		p.mu.Unlock()
		return
	}

	currentMs := p.calcCurrentMsLocked()
	if currentMs >= p.state.RightMs {
		if p.state.Loop {
			p.resetToRangeStartLocked()
			state = p.state
			p.mu.Unlock()
			AllSynthNotesOff()
			emitMidiVisualClear()
			emitMidiPlayerState(state)
			return
		} else {
			p.stopLocked()
			p.state.Status = MidiPlayIdle
			p.resetToRangeStartLocked()
			state = p.state
			p.mu.Unlock()
			AllSynthNotesOff()
			emitMidiVisualClear()
			emitMidiPlayerState(state)
			return
		}
	}

	if p.state.Mode == MidiPlaybackModeFollow {
		hintStep, clearHint = p.advanceFollowLocked(currentMs)
		if p.state.Waiting {
			currentMs = p.state.CurrentMs
		}
	}

	if !p.state.Waiting {
		dueEvents = p.collectDueEventsLocked(currentMs)
		p.state.CurrentMs = currentMs
	}

	if time.Since(p.lastStateEmit) >= stateEmitInterval || hintStep != nil || clearHint {
		state = p.state
		p.lastStateEmit = time.Now()
		emitState = true
	}
	p.mu.Unlock()

	if clearHint {
		emitMidiFollowHint(nil)
	}
	if hintStep != nil {
		emitMidiFollowHint(hintStep)
	}
	for _, event := range dueEvents {
		dispatchMidiEvent(event, true)
	}
	if emitState {
		emitMidiPlayerState(state)
	}
}

func (p *MidiPlayerRuntime) advanceFollowLocked(currentMs float64) (*MidiPracticeStep, bool) {
	if p.followIndex >= len(p.followSteps) {
		return nil, false
	}

	step := &p.followSteps[p.followIndex]
	p.state.CurrentStep = cloneMidiPracticeStep(step)

	if !p.hintActive && currentMs >= step.Ms-p.state.LeadWindowMs {
		p.hintActive = true
		p.acceptedNotes = make(map[int]bool)
		return cloneMidiPracticeStep(step), false
	}

	if currentMs < step.Ms {
		return nil, false
	}

	if p.stepAcceptedLocked(step) {
		p.completeFollowStepLocked(currentMs)
		return nil, true
	}

	p.state.Waiting = true
	p.state.CurrentMs = step.Ms
	p.baseMs = step.Ms
	p.baseTime = time.Now()
	return nil, false
}

func (p *MidiPlayerRuntime) HandleUserNoteOn(note int) {
	var state MidiPlayerState
	var clearHint bool
	var shouldEmit bool

	p.mu.Lock()
	if p.state.Status != MidiPlayPlaying || p.state.Mode != MidiPlaybackModeFollow {
		p.mu.Unlock()
		return
	}
	if p.followIndex >= len(p.followSteps) || !p.hintActive {
		p.mu.Unlock()
		return
	}

	step := &p.followSteps[p.followIndex]
	if !stepContainsNote(step, note) {
		p.mu.Unlock()
		return
	}

	p.acceptedNotes[note] = true
	if p.state.Waiting && p.stepAcceptedLocked(step) {
		p.completeFollowStepLocked(step.Ms)
		state = p.state
		clearHint = true
		shouldEmit = true
	}
	p.mu.Unlock()

	if clearHint {
		emitMidiFollowHint(nil)
	}
	if shouldEmit {
		emitMidiPlayerState(state)
	}
}

func (p *MidiPlayerRuntime) stepAcceptedLocked(step *MidiPracticeStep) bool {
	if step == nil {
		return false
	}
	for _, note := range step.Notes {
		if !p.acceptedNotes[note.Note] {
			return false
		}
	}
	return true
}

func (p *MidiPlayerRuntime) completeFollowStepLocked(currentMs float64) {
	p.followIndex++
	p.hintActive = false
	p.acceptedNotes = make(map[int]bool)
	p.state.Waiting = false
	p.state.CurrentMs = currentMs
	p.baseMs = currentMs
	p.baseTime = time.Now()
	if p.followIndex < len(p.followSteps) {
		p.state.CurrentStep = cloneMidiPracticeStep(&p.followSteps[p.followIndex])
	} else {
		p.state.CurrentStep = nil
	}
}

func (p *MidiPlayerRuntime) collectDueEventsLocked(currentMs float64) []MidiEvent {
	result := make([]MidiEvent, 0)
	for p.eventIndex < len(p.events) {
		event := p.events[p.eventIndex]
		if event.Ms > currentMs {
			break
		}
		p.eventIndex++
		if event.Ms < p.state.LeftMs || event.Ms > p.state.RightMs {
			continue
		}
		if p.shouldMuteEventLocked(event) {
			continue
		}
		result = append(result, event)
	}
	return result
}

func (p *MidiPlayerRuntime) shouldMuteEventLocked(event MidiEvent) bool {
	if p.state.MutedTracks[event.TrackIndex] {
		return true
	}
	if p.state.MutedHands[event.Hand] {
		return true
	}
	if event.Type != MidiEventNoteOn && event.Type != MidiEventNoteOff {
		return false
	}
	if p.state.Mode == MidiPlaybackModePlay {
		return !handModeAllowsEvent(p.state.Hand, event.Hand)
	}
	if p.state.Mode == MidiPlaybackModeFollow {
		return handModeAllowsEvent(p.state.Hand, event.Hand)
	}
	return false
}

func handModeAllowsEvent(mode MidiHandMode, hand MidiHand) bool {
	switch mode {
	case MidiHandModeLeft:
		return hand == MidiHandLeft
	case MidiHandModeRight:
		return hand == MidiHandRight
	default:
		return hand == MidiHandLeft || hand == MidiHandRight || hand == MidiHandUnknown
	}
}

func buildMidiPracticeSteps(events []MidiEvent, options MidiPlaybackOptions) []MidiPracticeStep {
	steps := make([]MidiPracticeStep, 0)
	for i := 0; i < len(events); i++ {
		event := events[i]
		if event.Type != MidiEventNoteOn || event.Velocity <= 0 {
			continue
		}
		if event.Ms < options.LeftMs || event.Ms > options.RightMs {
			continue
		}
		if !handModeAllowsEvent(options.HandMode, event.Hand) {
			continue
		}

		step := MidiPracticeStep{
			Index: len(steps),
			Ms:    event.Ms,
			Notes: []MidiPracticeNote{practiceNoteFromEvent(event)},
		}

		lastGroupedIndex := i
		for j := i + 1; j < len(events); j++ {
			next := events[j]
			if next.Ms-event.Ms > options.GroupWindowMs {
				break
			}
			if next.Type != MidiEventNoteOn || next.Velocity <= 0 {
				continue
			}
			if next.Ms < options.LeftMs || next.Ms > options.RightMs {
				continue
			}
			if !handModeAllowsEvent(options.HandMode, next.Hand) {
				continue
			}
			if !practiceStepHasNote(step, next.Note) {
				step.Notes = append(step.Notes, practiceNoteFromEvent(next))
			}
			lastGroupedIndex = j
		}
		steps = append(steps, step)
		i = lastGroupedIndex
	}
	return steps
}

func practiceNoteFromEvent(event MidiEvent) MidiPracticeNote {
	return MidiPracticeNote{
		Note:       event.Note,
		Velocity:   event.Velocity,
		Hand:       event.Hand,
		TrackIndex: event.TrackIndex,
		Ms:         event.Ms,
	}
}

func practiceStepHasNote(step MidiPracticeStep, note int) bool {
	for _, item := range step.Notes {
		if item.Note == note {
			return true
		}
	}
	return false
}

func stepContainsNote(step *MidiPracticeStep, note int) bool {
	if step == nil {
		return false
	}
	for _, item := range step.Notes {
		if item.Note == note {
			return true
		}
	}
	return false
}

func cloneMidiPracticeStep(step *MidiPracticeStep) *MidiPracticeStep {
	if step == nil {
		return nil
	}
	clone := *step
	clone.Notes = append([]MidiPracticeNote(nil), step.Notes...)
	return &clone
}

func (p *MidiPlayerRuntime) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.Status != MidiPlayPlaying {
		return nil
	}
	p.state.CurrentMs = clampFloat(p.calcCurrentMsLocked(), p.state.LeftMs, p.state.RightMs)
	p.state.Status = MidiPlayPaused
	p.state.Waiting = false
	p.baseMs = p.state.CurrentMs
	p.baseTime = time.Now()
	p.stopLocked()
	AllSynthNotesOff()
	emitMidiVisualClear()
	emitMidiPlayerState(p.state)
	return nil
}

func (p *MidiPlayerRuntime) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.Status != MidiPlayPaused {
		return nil
	}
	p.baseMs = p.state.CurrentMs
	p.baseTime = time.Now()
	p.state.Status = MidiPlayPlaying
	p.startLocked()
	emitMidiPlayerState(p.state)
	return nil
}

func (p *MidiPlayerRuntime) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
	AllSynthNotesOff()
	emitMidiVisualClear()
	p.state.Status = MidiPlayIdle
	p.resetToRangeStartLocked()
	emitMidiPlayerState(p.state)
	return nil
}

func (p *MidiPlayerRuntime) Reset() error {
	p.mu.Lock()
	p.stopLocked()
	p.events = nil
	p.eventIndex = 0
	p.followSteps = nil
	p.followIndex = 0
	p.hintActive = false
	p.acceptedNotes = make(map[int]bool)
	p.baseMs = 0
	p.baseTime = time.Now()
	p.state = newIdleMidiPlayerState()
	state := p.state
	p.mu.Unlock()

	AllSynthNotesOff()
	emitMidiVisualClear()
	emitMidiPlayerState(state)
	return nil
}

func (p *MidiPlayerRuntime) Seek(ms float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	ms = clampFloat(ms, p.state.LeftMs, p.state.RightMs)
	AllSynthNotesOff()
	emitMidiVisualClear()
	p.state.CurrentMs = ms
	p.state.Waiting = false
	p.eventIndex = findMidiEventIndexByMs(p.events, ms)
	p.followIndex = findPracticeStepIndexByMs(p.followSteps, ms)
	p.hintActive = false
	p.acceptedNotes = make(map[int]bool)
	if p.followIndex < len(p.followSteps) {
		p.state.CurrentStep = cloneMidiPracticeStep(&p.followSteps[p.followIndex])
	} else {
		p.state.CurrentStep = nil
	}
	p.baseMs = ms
	p.baseTime = time.Now()
	emitMidiPlayerState(p.state)
	return nil
}

func (p *MidiPlayerRuntime) SetSpeed(speed float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	speed = clampFloat(speed, 0.25, 3.0)
	if p.state.Status == MidiPlayPlaying {
		p.state.CurrentMs = clampFloat(p.calcCurrentMsLocked(), p.state.LeftMs, p.state.RightMs)
	}
	p.state.Speed = speed
	p.baseMs = p.state.CurrentMs
	p.baseTime = time.Now()
	emitMidiPlayerState(p.state)
	return nil
}

func (p *MidiPlayerRuntime) SetRange(leftMs float64, rightMs float64) error {
	p.mu.Lock()
	options := MidiPlaybackOptions{
		MidiID:        p.state.MidiID,
		Mode:          p.state.Mode,
		HandMode:      p.state.Hand,
		LeftMs:        leftMs,
		RightMs:       rightMs,
		Speed:         p.state.Speed,
		Loop:          p.state.Loop,
		LeadWindowMs:  p.state.LeadWindowMs,
		GroupWindowMs: p.state.GroupWindowMs,
	}
	p.mu.Unlock()

	return p.SetOptions(options)
}

func (p *MidiPlayerRuntime) SetOptions(options MidiPlaybackOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.events) == 0 || p.state.DurationMs <= 0 {
		return errors.New("MIDI 尚未加载")
	}

	currentMs := p.state.CurrentMs
	if p.state.Status == MidiPlayPlaying && !p.state.Waiting {
		currentMs = p.calcCurrentMsLocked()
	}

	if options.MidiID == "" {
		options.MidiID = p.state.MidiID
	}
	options = normalizeMidiPlaybackOptions(options, p.state.DurationMs)

	followSteps := make([]MidiPracticeStep, 0)
	if options.Mode == MidiPlaybackModeFollow {
		followSteps = buildMidiPracticeSteps(p.events, options)
		if len(followSteps) == 0 {
			return errors.New("当前范围和练习手没有可练习音符")
		}
	}

	currentMs = clampFloat(currentMs, options.LeftMs, options.RightMs)

	AllSynthNotesOff()
	emitMidiVisualClear()

	p.state.MidiID = options.MidiID
	p.state.Mode = options.Mode
	p.state.Hand = options.HandMode
	p.state.LeftMs = options.LeftMs
	p.state.RightMs = options.RightMs
	p.state.Speed = options.Speed
	p.state.Loop = options.Loop
	p.state.LeadWindowMs = options.LeadWindowMs
	p.state.GroupWindowMs = options.GroupWindowMs
	p.state.CurrentMs = currentMs
	p.state.Waiting = false
	p.state.Error = ""

	p.followSteps = followSteps
	p.followIndex = findPracticeStepIndexByMs(p.followSteps, currentMs)
	p.hintActive = false
	p.acceptedNotes = make(map[int]bool)
	if p.state.Mode == MidiPlaybackModeFollow && p.followIndex < len(p.followSteps) {
		p.state.CurrentStep = cloneMidiPracticeStep(&p.followSteps[p.followIndex])
	} else {
		p.state.CurrentStep = nil
	}

	p.eventIndex = findMidiEventIndexByMs(p.events, currentMs)
	p.baseMs = currentMs
	p.baseTime = time.Now()

	emitMidiPlayerState(p.state)
	return nil
}

func (p *MidiPlayerRuntime) resetToRangeStartLocked() {
	p.state.CurrentMs = p.state.LeftMs
	p.state.Waiting = false
	p.eventIndex = findMidiEventIndexByMs(p.events, p.state.LeftMs)
	p.followIndex = findPracticeStepIndexByMs(p.followSteps, p.state.LeftMs)
	p.hintActive = false
	p.acceptedNotes = make(map[int]bool)
	if p.state.Mode == MidiPlaybackModeFollow && p.followIndex < len(p.followSteps) {
		p.state.CurrentStep = cloneMidiPracticeStep(&p.followSteps[p.followIndex])
	} else {
		p.state.CurrentStep = nil
	}
	p.baseMs = p.state.LeftMs
	p.baseTime = time.Now()
}

func (p *MidiPlayerRuntime) SetTrackMuted(trackIndex int, muted bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.MutedTracks == nil {
		p.state.MutedTracks = make(map[int]bool)
	}
	p.state.MutedTracks[trackIndex] = muted
	if muted {
		AllSynthNotesOff()
		emitMidiVisualClear()
	}
	emitMidiPlayerState(p.state)
	return nil
}

func (p *MidiPlayerRuntime) SetHandMuted(hand MidiHand, muted bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state.MutedHands == nil {
		p.state.MutedHands = make(map[MidiHand]bool)
	}
	p.state.MutedHands[hand] = muted
	if muted {
		AllSynthNotesOff()
		emitMidiVisualClear()
	}
	emitMidiPlayerState(p.state)
	return nil
}

func (p *MidiPlayerRuntime) State() MidiPlayerState {
	p.mu.Lock()
	defer p.mu.Unlock()
	state := p.state
	if state.Status == MidiPlayPlaying && !state.Waiting {
		state.CurrentMs = clampFloat(p.calcCurrentMsLocked(), state.LeftMs, state.RightMs)
	}
	return state
}

func (p *MidiPlayerRuntime) calcCurrentMsLocked() float64 {
	if p.state.Status != MidiPlayPlaying || p.state.Waiting {
		return p.state.CurrentMs
	}
	elapsedMs := float64(time.Since(p.baseTime).Microseconds()) / 1000.0
	return p.baseMs + elapsedMs*p.state.Speed
}

func (p *MidiPlayerRuntime) startLocked() {
	if p.ticker != nil {
		return
	}

	ticker := time.NewTicker(playerTickInterval)
	stopCh := make(chan struct{})

	p.ticker = ticker
	p.stopCh = stopCh

	go p.loop(ticker, stopCh)
}

func (p *MidiPlayerRuntime) stopLocked() {
	if p.ticker == nil {
		return
	}

	ticker := p.ticker
	stopCh := p.stopCh

	p.ticker = nil
	p.stopCh = nil

	ticker.Stop()

	if stopCh != nil {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
	}
}

func (p *MidiPlayerRuntime) loop(ticker *time.Ticker, stopCh <-chan struct{}) {
	for {
		select {
		case <-ticker.C:
			p.Tick()
		case <-stopCh:
			return
		}
	}
}
func findMidiEventIndexByMs(events []MidiEvent, ms float64) int {
	return sort.Search(len(events), func(i int) bool {
		return events[i].Ms >= ms
	})
}

func findPracticeStepIndexByMs(steps []MidiPracticeStep, ms float64) int {
	return sort.Search(len(steps), func(i int) bool {
		return steps[i].Ms >= ms
	})
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if maxValue < minValue {
		return minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func dispatchMidiEvent(event MidiEvent, emitVisual bool) {
	config := GetUserConfig()
	channel := config.MidiChannel

	switch event.Type {
	case MidiEventNoteOn:
		playSelectedOutputNoteOn(channel, uint8(event.Note), int32(event.Velocity), uint8(event.Velocity))
		if emitVisual {
			emitMidiPlaybackKey(event, channel, true)
		}
	case MidiEventNoteOff:
		playSelectedOutputNoteOff(channel, uint8(event.Note))
		if emitVisual {
			emitMidiPlaybackKey(event, channel, false)
		}
	}
}

func emitMidiPlayerState(state MidiPlayerState) {
	if App != nil {
		App.Event.Emit("midiPlayerState", state)
	}
}

func emitMidiPlaybackKey(event MidiEvent, channel uint8, active bool) {
	if App == nil {
		return
	}
	App.Event.Emit("midiPlaybackKey", MidiVisualEvent{
		Note:     event.Note,
		Velocity: event.Velocity,
		Channel:  channel,
		Hand:     event.Hand,
		Source:   "playback",
		Active:   active,
	})
}

func emitMidiFollowHint(step *MidiPracticeStep) {
	if App == nil {
		return
	}
	App.Event.Emit("midiFollowHint", step)
}

func emitMidiVisualClear() {
	if App != nil {
		App.Event.Emit("midiVisualClear")
	}
}
