package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gitlab.com/gomidi/midi/v2/smf"
)

var RuntimeMidiCache = NewMidiRuntimeCache()

type MidiRuntimeCache struct {
	mu    sync.RWMutex
	items map[string]*Midi
}

func NewMidiRuntimeCache() *MidiRuntimeCache {
	return &MidiRuntimeCache{
		items: make(map[string]*Midi),
	}
}

// =========================
// MIDI 基础类型
// =========================

type MidiHand string

const (
	MidiHandUnknown MidiHand = "unknown"
	MidiHandLeft    MidiHand = "left"
	MidiHandRight   MidiHand = "right"
)

type MidiEventType string

const (
	MidiEventNoteOn        MidiEventType = "noteOn"
	MidiEventNoteOff       MidiEventType = "noteOff"
	MidiEventControlChange MidiEventType = "controlChange"
	MidiEventProgramChange MidiEventType = "programChange"
)

type MidiFrontEvent string

const (
	MidiFrontRightDown        MidiFrontEvent = "down"
	MidiFrontRightPressedDown MidiFrontEvent = "pressedDown"

	MidiFrontLeftDown        MidiFrontEvent = "leftDown"
	MidiFrontLeftPressedDown MidiFrontEvent = "leftPressedDown"
)

// =========================
// config 中保存的轻量 MIDI 对象
// =========================

type UserMidi struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`

	Size       int64     `json:"size"`
	ImportedAt time.Time `json:"importedAt"`
	Missing    bool      `json:"missing"`

	DurationMs float64 `json:"durationMs,omitempty"`
	TrackCount int     `json:"trackCount,omitempty"`
}

// =========================
// 运行时 MIDI 对象
// =========================

type Midi struct {
	UserMidi

	Format uint16 `json:"format"`
	PPQ    int    `json:"ppq"`

	DurationTick uint64 `json:"durationTick"`

	// 仅运行时使用，不写入 config。
	Events []MidiEvent `json:"-"`
}

type MidiEvent struct {
	Type MidiEventType `json:"type"`

	TrackIndex int      `json:"trackIndex"`
	Channel    int      `json:"channel"`
	Hand       MidiHand `json:"hand"`

	// 给前端区分右手/左手显示
	FrontEvent MidiFrontEvent `json:"frontEvent"`

	Tick uint64  `json:"tick"`
	Ms   float64 `json:"ms"`

	Note     int `json:"note,omitempty"`
	Velocity int `json:"velocity,omitempty"`

	Controller int `json:"controller,omitempty"`
	Value      int `json:"value,omitempty"`

	Program int `json:"program,omitempty"`
}

// =========================
// 对外函数：函数式风格
// =========================

func AddMidiByPath(path string) error {
	userMidi, err := ImportMidiByPath(path)
	if err != nil {
		return err
	}

	config := GetUserConfig()

	for i := range config.MidiStore {
		if config.MidiStore[i].ID == userMidi.ID {
			return fmt.Errorf("MIDI 文件已存在")
		}
	}

	config.MidiStore = append(config.MidiStore, userMidi)
	return SaveConfig(config)
}

func ImportMidiByPath(path string) (UserMidi, error) {
	midiFile, err := ParseMidiFile(path)
	if err != nil {
		return UserMidi{}, err
	}

	return MidiToUserMidi(midiFile), nil
}

func RemoveMidi(id string) error {
	config := GetUserConfig()

	for i := range config.MidiStore {
		if config.MidiStore[i].ID == id {
			config.MidiStore = append(config.MidiStore[:i], config.MidiStore[i+1:]...)
			RemoveMidiCacheByID(id)
			return SaveConfig(config)
		}
	}

	return SaveConfig(config)
}

func GetMidiStore() []UserMidi {
	config := GetUserConfig()
	return append([]UserMidi(nil), config.MidiStore...)
}

func FindUserMidiByID(id string) (UserMidi, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return UserMidi{}, false
	}

	for _, item := range GetUserConfig().MidiStore {
		if item.ID == id {
			return item, true
		}
	}

	return UserMidi{}, false
}

func (k *Keyboard) GetMidiStore() []UserMidi {
	return GetMidiStore()
}

func (k *Keyboard) OpenMidiFileDialog() (UserMidi, error) {
	if App == nil {
		return UserMidi{}, errors.New("应用尚未初始化")
	}

	path, err := App.Dialog.OpenFile().
		SetTitle("选择 MIDI 文件").
		AddFilter("MIDI", "*.mid;*.midi").
		AddFilter("All Files", "*.*").
		PromptForSingleSelection()
	if err != nil {
		return UserMidi{}, err
	}
	if strings.TrimSpace(path) == "" {
		return UserMidi{}, errors.New("未选择 MIDI 文件")
	}

	return k.AddMidiByPath(path)
}

func (k *Keyboard) AddMidiByPath(path string) (UserMidi, error) {
	userMidi, err := ImportMidiByPath(path)
	if err != nil {
		return UserMidi{}, err
	}

	config := GetUserConfig()
	for _, item := range config.MidiStore {
		if item.ID == userMidi.ID || IsSamePath(item.Path, userMidi.Path) {
			return item, nil
		}
	}

	config.MidiStore = append(config.MidiStore, userMidi)
	if err := SaveConfig(config); err != nil {
		return UserMidi{}, err
	}

	return userMidi, nil
}

func (k *Keyboard) RemoveMidiByID(id string) error {
	return RemoveMidi(id)
}

func (k *Keyboard) LoadMidiByID(id string) (UserMidi, error) {
	userMidi, ok := FindUserMidiByID(id)
	if !ok {
		return UserMidi{}, errors.New("MIDI 不存在")
	}

	midiFile, err := GetOrParse(userMidi)
	if err != nil {
		return UserMidi{}, err
	}

	return MidiToUserMidi(midiFile), nil
}

func (k *Keyboard) StartMidiPlayback(options MidiPlaybackOptions) error {
	userMidi, ok := FindUserMidiByID(options.MidiID)
	if !ok {
		return errors.New("MIDI 不存在")
	}

	midiFile, err := GetOrParse(userMidi)
	if err != nil {
		return err
	}

	return MidiPlayer.Start(midiFile, options)
}

func (k *Keyboard) SwitchMidiPlayback(options MidiPlaybackOptions) error {
	return k.StartMidiPlayback(options)
}

func (k *Keyboard) PauseMidiPlayback() error {
	return MidiPlayer.Pause()
}

func (k *Keyboard) ResumeMidiPlayback() error {
	return MidiPlayer.Resume()
}

func (k *Keyboard) StopMidiPlayback() error {
	return MidiPlayer.Stop()
}

func (k *Keyboard) ResetMidiPlayback() error {
	return MidiPlayer.Reset()
}

func (k *Keyboard) SeekMidiPlayback(ms float64) error {
	return MidiPlayer.Seek(ms)
}

func (k *Keyboard) SetMidiPlaybackSpeed(speed float64) error {
	return MidiPlayer.SetSpeed(speed)
}

func (k *Keyboard) SetMidiPlaybackRange(leftMs float64, rightMs float64) error {
	return MidiPlayer.SetRange(leftMs, rightMs)
}

func (k *Keyboard) SetMidiPlaybackOptions(options MidiPlaybackOptions) error {
	return MidiPlayer.SetOptions(options)
}

func (k *Keyboard) SetMidiTrackMuted(trackIndex int, muted bool) error {
	return MidiPlayer.SetTrackMuted(trackIndex, muted)
}

func (k *Keyboard) SetMidiHandMuted(hand MidiHand, muted bool) error {
	return MidiPlayer.SetHandMuted(hand, muted)
}

func (k *Keyboard) GetMidiPlayerState() MidiPlayerState {
	return MidiPlayer.State()
}

func MidiToUserMidi(midiFile *Midi) UserMidi {
	if midiFile == nil {
		return UserMidi{}
	}

	return UserMidi{
		ID:         midiFile.ID,
		Name:       midiFile.Name,
		Path:       midiFile.Path,
		Size:       midiFile.Size,
		ImportedAt: midiFile.ImportedAt,
		Missing:    midiFile.Missing,
		DurationMs: midiFile.DurationMs,
		TrackCount: midiFile.TrackCount,
	}
}

func UserMidiToMidi(userMidi UserMidi) (*Midi, error) {
	return ParseUserMidi(userMidi)
}

func ParseUserMidi(userMidi UserMidi) (*Midi, error) {
	if strings.TrimSpace(userMidi.Path) == "" {
		return nil, errors.New("MIDI 路径为空")
	}

	midiFile, err := ParseMidiFile(userMidi.Path)
	if err != nil {
		return nil, err
	}

	// 保留用户列表中的信息
	if userMidi.ID != "" {
		midiFile.ID = userMidi.ID
	}
	if userMidi.Name != "" {
		midiFile.Name = userMidi.Name
	}
	if !userMidi.ImportedAt.IsZero() {
		midiFile.ImportedAt = userMidi.ImportedAt
	}

	midiFile.Missing = userMidi.Missing

	return midiFile, nil
}

func GetOrParse(userMidi UserMidi) (*Midi, error) {
	if strings.TrimSpace(userMidi.Path) == "" {
		return nil, errors.New("MIDI 路径为空")
	}

	info, err := os.Stat(userMidi.Path)
	if err != nil {
		return nil, fmt.Errorf("MIDI 文件不存在或无法访问: %w", err)
	}

	cacheKey := buildMidiCacheKey(userMidi, info)

	RuntimeMidiCache.mu.RLock()
	if midiFile, ok := RuntimeMidiCache.items[cacheKey]; ok {
		RuntimeMidiCache.mu.RUnlock()
		return midiFile, nil
	}
	RuntimeMidiCache.mu.RUnlock()

	midiFile, err := ParseUserMidi(userMidi)
	if err != nil {
		return nil, err
	}

	RuntimeMidiCache.mu.Lock()
	RuntimeMidiCache.items[cacheKey] = midiFile
	RuntimeMidiCache.mu.Unlock()

	return midiFile, nil
}

func ClearMidiCache() {
	RuntimeMidiCache.mu.Lock()
	defer RuntimeMidiCache.mu.Unlock()

	RuntimeMidiCache.items = make(map[string]*Midi)
}

func RemoveMidiCacheByID(id string) {
	RuntimeMidiCache.mu.Lock()
	defer RuntimeMidiCache.mu.Unlock()

	for key, midiFile := range RuntimeMidiCache.items {
		if midiFile.ID == id {
			delete(RuntimeMidiCache.items, key)
		}
	}
}

// =========================
// MIDI 解析：仅 Events 模式
// =========================

func ParseMidiFile(path string) (*Midi, error) {
	absPath, err := validateMidiPath(path)
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("读取 MIDI 文件信息失败: %w", err)
	}

	raw, err := smf.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("不是有效的 MIDI 文件或文件已损坏: %w", err)
	}

	ppq := 0
	if ticks, ok := raw.TimeFormat.(smf.MetricTicks); ok {
		ppq = int(ticks.Resolution())
	}

	id, err := FileMD5(absPath)
	if err != nil {
		return nil, fmt.Errorf("生成 MIDI 文件 ID 失败: %w", err)
	}

	events, durationTick, durationMs, err := parseMidiEvents(absPath)
	if err != nil {
		return nil, err
	}

	if !hasPlayableNoteEvent(events) {
		return nil, errors.New("MIDI 文件中没有可播放的音符事件")
	}

	handByTrack := guessHandsByEvents(events)
	applyHandsToEvents(events, handByTrack)
	sortMidiEvents(events)

	return &Midi{
		UserMidi: UserMidi{
			ID:         id,
			Name:       strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath)),
			Path:       absPath,
			Size:       fileInfo.Size(),
			ImportedAt: time.Now(),
			Missing:    false,
			DurationMs: durationMs,
			TrackCount: int(raw.NumTracks()),
		},
		Format:       raw.Format(),
		PPQ:          ppq,
		DurationTick: durationTick,
		Events:       events,
	}, nil
}

func parseMidiEvents(path string) ([]MidiEvent, uint64, float64, error) {
	events := make([]MidiEvent, 0)

	var durationTick uint64
	var durationMs float64

	reader := smf.ReadTracks(path)

	reader.Do(func(ev smf.TrackEvent) {
		trackIndex := ev.TrackNo
		tick := safeUint64(ev.AbsTicks)
		ms := float64(ev.AbsMicroSeconds) / 1000.0

		if tick > durationTick {
			durationTick = tick
		}
		if ms > durationMs {
			durationMs = ms
		}

		msg := ev.Message

		var ch, key, vel uint8

		if msg.GetNoteStart(&ch, &key, &vel) {
			events = append(events, MidiEvent{
				Type:       MidiEventNoteOn,
				TrackIndex: trackIndex,
				Channel:    int(ch),
				Tick:       tick,
				Ms:         ms,
				Note:       int(key),
				Velocity:   int(vel),
			})
			return
		}

		if msg.GetNoteEnd(&ch, &key) {
			events = append(events, MidiEvent{
				Type:       MidiEventNoteOff,
				TrackIndex: trackIndex,
				Channel:    int(ch),
				Tick:       tick,
				Ms:         ms,
				Note:       int(key),
			})
			return
		}

		var controller, value uint8
		if msg.GetControlChange(&ch, &controller, &value) {
			events = append(events, MidiEvent{
				Type:       MidiEventControlChange,
				TrackIndex: trackIndex,
				Channel:    int(ch),
				Tick:       tick,
				Ms:         ms,
				Controller: int(controller),
				Value:      int(value),
			})
			return
		}

		var program uint8
		if msg.GetProgramChange(&ch, &program) {
			events = append(events, MidiEvent{
				Type:       MidiEventProgramChange,
				TrackIndex: trackIndex,
				Channel:    int(ch),
				Tick:       tick,
				Ms:         ms,
				Program:    int(program),
			})
			return
		}
	})

	if err := reader.Error(); err != nil {
		return nil, 0, 0, fmt.Errorf("解析 MIDI 事件失败: %w", err)
	}

	return events, durationTick, durationMs, nil
}

// =========================
// 左右手推断：直接基于 Events
// =========================

func guessHandsByEvents(events []MidiEvent) map[int]MidiHand {
	noteCountByTrack := make(map[int]int)
	noteSumByTrack := make(map[int]int)
	minNoteByTrack := make(map[int]int)
	isDrumByTrack := make(map[int]bool)

	for _, event := range events {
		if event.Channel == 9 {
			isDrumByTrack[event.TrackIndex] = true
		}

		if event.Type != MidiEventNoteOn {
			continue
		}

		if event.Velocity <= 0 {
			continue
		}

		trackIndex := event.TrackIndex

		noteCountByTrack[trackIndex]++
		noteSumByTrack[trackIndex] += event.Note

		if _, ok := minNoteByTrack[trackIndex]; !ok {
			minNoteByTrack[trackIndex] = event.Note
		} else if event.Note < minNoteByTrack[trackIndex] {
			minNoteByTrack[trackIndex] = event.Note
		}
	}

	validTracks := make([]int, 0)

	for trackIndex, count := range noteCountByTrack {
		if count == 0 {
			continue
		}

		if isDrumByTrack[trackIndex] {
			continue
		}

		validTracks = append(validTracks, trackIndex)
	}

	result := make(map[int]MidiHand)

	if len(validTracks) == 0 {
		return result
	}

	// 单轨统一右手
	if len(validTracks) == 1 {
		result[validTracks[0]] = MidiHandRight
		return result
	}

	sort.Slice(validTracks, func(i, j int) bool {
		a := validTracks[i]
		b := validTracks[j]

		avgA := float64(noteSumByTrack[a]) / float64(noteCountByTrack[a])
		avgB := float64(noteSumByTrack[b]) / float64(noteCountByTrack[b])

		if avgA == avgB {
			return minNoteByTrack[a] < minNoteByTrack[b]
		}

		return avgA < avgB
	})

	split := len(validTracks) / 2

	for i, trackIndex := range validTracks {
		if i < split {
			result[trackIndex] = MidiHandLeft
		} else {
			result[trackIndex] = MidiHandRight
		}
	}

	return result
}

func applyHandsToEvents(events []MidiEvent, handByTrack map[int]MidiHand) {
	for i := range events {
		hand := handByTrack[events[i].TrackIndex]
		if hand == "" {
			hand = MidiHandUnknown
		}

		events[i].Hand = hand
		events[i].FrontEvent = buildFrontEvent(events[i].Type, hand)
	}
}

func hasPlayableNoteEvent(events []MidiEvent) bool {
	for _, event := range events {
		if event.Type == MidiEventNoteOn && event.Velocity > 0 {
			return true
		}
	}

	return false
}

// =========================
// 工具函数
// =========================

func validateMidiPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("MIDI 路径不能为空")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("获取 MIDI 绝对路径失败: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	if ext != ".mid" && ext != ".midi" {
		return "", errors.New("文件格式不正确，只支持 .mid 或 .midi")
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("MIDI 文件不存在: %w", err)
	}

	if info.IsDir() {
		return "", errors.New("路径是文件夹，不是 MIDI 文件")
	}

	if info.Size() <= 0 {
		return "", errors.New("MIDI 文件为空")
	}

	return absPath, nil
}

func buildMidiCacheKey(userMidi UserMidi, info os.FileInfo) string {
	return fmt.Sprintf(
		"%s|%s|%d|%d",
		userMidi.ID,
		filepath.Clean(userMidi.Path),
		info.Size(),
		info.ModTime().UnixNano(),
	)
}

func safeUint64(value int64) uint64 {
	if value < 0 {
		return 0
	}

	return uint64(value)
}

func sortMidiEvents(events []MidiEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Ms == events[j].Ms {
			return midiEventOrder(events[i].Type) < midiEventOrder(events[j].Type)
		}

		return events[i].Ms < events[j].Ms
	})
}

func midiEventOrder(t MidiEventType) int {
	switch t {
	case MidiEventProgramChange:
		return 1
	case MidiEventControlChange:
		return 2
	case MidiEventNoteOff:
		return 3
	case MidiEventNoteOn:
		return 4
	default:
		return 9
	}
}

func buildFrontEvent(eventType MidiEventType, hand MidiHand) MidiFrontEvent {
	switch eventType {
	case MidiEventNoteOn:
		if hand == MidiHandLeft {
			return MidiFrontLeftDown
		}
		return MidiFrontRightDown

	case MidiEventNoteOff:
		if hand == MidiHandLeft {
			return MidiFrontLeftPressedDown
		}
		return MidiFrontRightPressedDown

	default:
		return ""
	}
}
