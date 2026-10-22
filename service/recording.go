package service

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	recordingEventLimit = 500000
	recordingPPQ        = 480
	recordingTempoUS    = 500000
)

type RecordingState struct {
	Status     string  `json:"status"`
	DraftID    string  `json:"draftId"`
	DurationMs float64 `json:"durationMs"`
	NoteCount  int     `json:"noteCount"`
	Error      string  `json:"error"`
}

type recordingTrackKey struct {
	DeviceID int   `json:"deviceId"`
	Channel  uint8 `json:"channel"`
}

type recordingNoteKey struct {
	Track recordingTrackKey `json:"track"`
	Note  uint8             `json:"note"`
}

type recordingPedalKey struct {
	Track      recordingTrackKey `json:"track"`
	Controller uint8             `json:"controller"`
}

type recordingInputEvent struct {
	AtUS       int64             `json:"atUs"`
	Track      recordingTrackKey `json:"track"`
	Kind       uint8             `json:"kind"`
	Data1      uint8             `json:"data1"`
	Data2      uint8             `json:"data2"`
}

// Persist only a stopped draft; no frontend canvas history is involved.
type recordingSnapshot struct {
	ID         string                `json:"id"`
	DurationUS int64                 `json:"durationUs"`
	NoteCount  int                   `json:"noteCount"`
	Error      string                `json:"error"`
	Events     []recordingInputEvent `json:"events"`
}

var recording = struct {
	loaded     bool
	status     string
	id         string
	started    time.Time
	durationUS int64
	noteCount  int
	lastError  string
	events     []recordingInputEvent
	held       map[recordingNoteKey]bool
	pedals     map[recordingPedalKey]uint8
	tracks     map[recordingTrackKey]bool
	physical   map[recordingTrackKey]uint8
	software   bool
	saving     bool
}{
	status: "idle",
	held:   make(map[recordingNoteKey]bool),
	pedals: make(map[recordingPedalKey]uint8),
	tracks: make(map[recordingTrackKey]bool),
	physical: make(map[recordingTrackKey]uint8),
}

func recordingDraftPath() string {
	return filepath.Join(filepath.Dir(configFilePath), "recording-draft.json")
}

// Caller holds inputEventsMu, the same lock as the real MIDI input pipeline.
func loadRecordingDraftLocked() {
	if recording.loaded {
		return
	}
	recording.loaded = true
	data, err := os.ReadFile(recordingDraftPath())
	if err != nil {
		return
	}
	var snap recordingSnapshot
	if json.Unmarshal(data, &snap) != nil || snap.ID == "" || snap.NoteCount <= 0 || len(snap.Events) == 0 || len(snap.Events) > recordingEventLimit {
		return
	}
	recording.status, recording.id = "draft", snap.ID
	recording.durationUS, recording.noteCount = snap.DurationUS, snap.NoteCount
	recording.lastError = snap.Error
	recording.events = snap.Events
}

func recordingStateLocked() RecordingState {
	loadRecordingDraftLocked()
	duration := recording.durationUS
	if recording.status == "recording" {
		duration = time.Since(recording.started).Microseconds()
	}
	return RecordingState{
		Status: recording.status, DraftID: recording.id,
		DurationMs: float64(duration) / 1000,
		NoteCount: recording.noteCount, Error: recording.lastError,
	}
}

func (k *Keyboard) GetRecordingState() RecordingState {
	inputEventsMu.Lock()
	defer inputEventsMu.Unlock()
	return recordingStateLocked()
}

func (k *Keyboard) StartRecording() (RecordingState, error) {
	inputEventsMu.Lock()
	defer inputEventsMu.Unlock()
	loadRecordingDraftLocked()
	if recording.status == "recording" {
		return recordingStateLocked(), nil
	}
	if recording.status == "draft" || recording.saving {
		return recordingStateLocked(), errors.New("请先保存或放弃上一段录制")
	}
	recording.status = "recording"
	recording.id = fmt.Sprintf("%d", time.Now().UnixNano())
	recording.started = time.Now()
	recording.durationUS, recording.noteCount = 0, 0
	recording.lastError = ""
	recording.events = make([]recordingInputEvent, 0, 1024)
	recording.held = make(map[recordingNoteKey]bool)
	recording.pedals = make(map[recordingPedalKey]uint8)
	recording.tracks = make(map[recordingTrackKey]bool)
	recording.physical = make(map[recordingTrackKey]uint8)
	recording.software = false
	recordingSoftwareSustainLocked(softwareSustain)
	// If a real damper is already pressed when recording begins, retain its
	// initial controller state instead of waiting for the next pedal movement.
	midiMu.RLock()
	for deviceID, pedal := range Midis.PedalStatus {
		if pedal == nil {
			continue
		}
		for channel, down := range pedal.Channels {
			if down {
				recordingControlLocked(deviceID, channel, 64, 127)
			}
		}
	}
	midiMu.RUnlock()
	return recordingStateLocked(), nil
}

func recordingAppendLocked(track recordingTrackKey, kind, data1, data2 uint8) {
	if recording.status != "recording" {
		return
	}
	if len(recording.events) >= recordingEventLimit {
		stopRecordingLocked("录制事件已达到上限，已自动停止，请保存当前草稿")
		return
	}
	recording.tracks[track] = true
	recording.events = append(recording.events, recordingInputEvent{
		AtUS: time.Since(recording.started).Microseconds(),
		Track: track, Kind: kind, Data1: data1, Data2: data2,
	})
}

func recordingNoteOnLocked(deviceID int, channel, note, velocity uint8) {
	if recording.status != "recording" {
		return
	}
	track := recordingTrackKey{deviceID, channel}
	if recording.software && !recording.tracks[track] {
		recordingEmitControlLocked(track, 64, 127)
	}
	key := recordingNoteKey{track, note}
	// A retrigger without a release must not create an endless preceding note.
	if recording.held[key] {
		recordingAppendLocked(track, 0x80, note, 0)
	}
	recording.held[key] = true
	recordingAppendLocked(track, 0x90, note, velocity)
	if recording.status == "recording" {
		recording.noteCount++
	}
}

func recordingNoteOffLocked(deviceID int, channel, note uint8) {
	if recording.status != "recording" {
		return
	}
	track := recordingTrackKey{deviceID, channel}
	key := recordingNoteKey{track, note}
	if !recording.held[key] {
		return
	}
	delete(recording.held, key)
	recordingAppendLocked(track, 0x80, note, 0)
}

func recordingControlLocked(deviceID int, channel, controller, value uint8) {
	if recording.status != "recording" {
		return
	}
	if controller != 64 && controller != 66 && controller != 67 {
		return
	}
	track := recordingTrackKey{deviceID, channel}
	if controller == 64 {
		recording.physical[track] = value
		if recording.software && value < 64 {
			value = 127
		}
	}
	recordingEmitControlLocked(track, controller, value)
}

func recordingEmitControlLocked(track recordingTrackKey, controller, value uint8) {
	key := recordingPedalKey{track, controller}
	if recording.pedals[key] == value {
		return
	}
	recording.pedals[key] = value
	recordingAppendLocked(track, 0xB0, controller, value)
}

// Include software sustain in the exported event stream, on the interactive
// channel and on channels already used by external MIDI sources.
func recordingSoftwareSustainLocked(enabled bool) {
	if recording.status != "recording" {
		return
	}
	recording.software = enabled
	channel := GetUserConfig().MidiChannel
	interactive := recordingTrackKey{interactiveDeviceID, channel}
	recording.tracks[interactive] = true
	for track := range recording.tracks {
		value := recording.physical[track]
		if enabled && value < 64 {
			value = 127
		}
		recordingEmitControlLocked(track, 64, value)
	}
}

func stopRecordingLocked(reason string) {
	if recording.status != "recording" {
		return
	}
	recording.durationUS = time.Since(recording.started).Microseconds()
	// Release all physical notes before resetting controller state.
	for key := range recording.held {
		recording.events = append(recording.events, recordingInputEvent{
			AtUS: recording.durationUS, Track: key.Track, Kind: 0x80, Data1: key.Note,
		})
	}
	for key, value := range recording.pedals {
		if value >= 64 {
			recording.events = append(recording.events, recordingInputEvent{
				AtUS: recording.durationUS, Track: key.Track, Kind: 0xB0, Data1: key.Controller,
			})
		}
	}
	clear(recording.held)
	clear(recording.pedals)
	clear(recording.physical)
	recording.software = false
	recording.lastError = reason
	if recording.noteCount == 0 {
		recording.status, recording.id = "idle", ""
		recording.lastError = "录制为空，没有可保存的音符"
		recording.events = nil
		_ = os.Remove(recordingDraftPath())
		return
	}
	recording.status = "draft"
	snap := recordingSnapshot{
		ID: recording.id, DurationUS: recording.durationUS,
		NoteCount: recording.noteCount, Error: recording.lastError,
		Events: recording.events,
	}
	data, err := json.Marshal(snap)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(recordingDraftPath()), 0700)
	}
	if err == nil {
		err = writeAtomicFile(recordingDraftPath(), data, 0600)
	}
	if err != nil {
		recording.lastError = "草稿未能写入本地备份，请在退出前保存：" + err.Error()
	}
}

func (k *Keyboard) StopRecording() RecordingState {
	inputEventsMu.Lock()
	defer inputEventsMu.Unlock()
	stopRecordingLocked("")
	return recordingStateLocked()
}

func stopRecordingForShutdown() {
	inputEventsMu.Lock()
	defer inputEventsMu.Unlock()
	stopRecordingLocked("录制因退出或设备状态变化而停止")
}

func recordingBytes(snap recordingSnapshot) ([]byte, error) {
	if snap.NoteCount == 0 || len(snap.Events) == 0 {
		return nil, errors.New("空录制不能导出")
	}
	type trackData struct {
		key     recordingTrackKey
		channel uint8
		last    uint64
		data    bytes.Buffer
	}
	var tracks []*trackData
	byKey := make(map[recordingTrackKey]*trackData)
	usedChannel := make(map[uint8]bool)
	// Keep source/channel voices separate so a release on another input cannot
	// cut the same-pitch voice short in an external MIDI player.
	for _, event := range snap.Events {
		if byKey[event.Track] != nil {
			continue
		}
		channel := event.Track.Channel
		if channel == 9 || usedChannel[channel] {
			channel = 16
			for n := uint8(0); n < 16; n++ {
				if n != 9 && !usedChannel[n] {
					channel = n
					break
				}
			}
		}
		if channel >= 16 {
			return nil, errors.New("录制涉及超过 15 个独立输入通道，无法无冲突导出 MIDI")
		}
		usedChannel[channel] = true
		t := &trackData{key: event.Track, channel: channel}
		byKey[event.Track] = t
		tracks = append(tracks, t)
	}
	if len(tracks) > 0xffff-1 {
		return nil, errors.New("MIDI 轨道数量超出标准限制")
	}
	for _, event := range snap.Events {
		t := byKey[event.Track]
		tick := uint64((event.AtUS*recordingPPQ + recordingTempoUS/2) / recordingTempoUS)
		if tick < t.last {
			tick = t.last
		}
		writeRecordingVLQ(&t.data, tick-t.last)
		t.data.WriteByte(event.Kind | t.channel)
		t.data.WriteByte(event.Data1)
		t.data.WriteByte(event.Data2)
		t.last = tick
	}
	out := new(bytes.Buffer)
	out.WriteString("MThd")
	_ = binary.Write(out, binary.BigEndian, uint32(6))
	_ = binary.Write(out, binary.BigEndian, uint16(1))
	_ = binary.Write(out, binary.BigEndian, uint16(len(tracks)+1))
	_ = binary.Write(out, binary.BigEndian, uint16(recordingPPQ))
	// Tempo track: 120 BPM and explicit end-of-track.
	writeRecordingChunk(out, []byte{0, 0xff, 0x51, 3, 7, 0xa1, 0x20, 0, 0xff, 0x2f, 0})
	for _, t := range tracks {
		endTick := uint64((snap.DurationUS*recordingPPQ + recordingTempoUS/2) / recordingTempoUS)
		if endTick < t.last {
			endTick = t.last
		}
		writeRecordingVLQ(&t.data, endTick-t.last)
		t.data.Write([]byte{0xff, 0x2f, 0})
		writeRecordingChunk(out, t.data.Bytes())
	}
	return out.Bytes(), nil
}

func writeRecordingVLQ(out *bytes.Buffer, value uint64) {
	// SMF delta-times are 28-bit VLQs. One event gap longer than this is
	// represented by a sequence of harmless zero-text meta events.
	for value > 0x0fffffff {
		out.Write([]byte{0xff, 0xff, 0xff, 0x7f, 0xff, 0x01, 0})
		value -= 0x0fffffff
	}
	var buffer [4]byte
	count := 1
	buffer[3] = byte(value & 0x7f)
	for value >>= 7; value != 0; value >>= 7 {
		count++
		buffer[4-count] = byte(value&0x7f) | 0x80
	}
	out.Write(buffer[4-count:])
}

func writeRecordingChunk(out *bytes.Buffer, data []byte) {
	out.WriteString("MTrk")
	_ = binary.Write(out, binary.BigEndian, uint32(len(data)))
	out.Write(data)
}

func recordingDraftSnapshot() (recordingSnapshot, error) {
	inputEventsMu.Lock()
	defer inputEventsMu.Unlock()
	loadRecordingDraftLocked()
	if recording.status != "draft" || recording.noteCount == 0 {
		return recordingSnapshot{}, errors.New("没有可用的录制草稿")
	}
	return recordingSnapshot{
		ID: recording.id, DurationUS: recording.durationUS,
		NoteCount: recording.noteCount, Events: append([]recordingInputEvent(nil), recording.events...),
	}, nil
}

func (k *Keyboard) PreviewRecording() error {
	snap, err := recordingDraftSnapshot()
	if err != nil {
		return err
	}
	data, err := recordingBytes(snap)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "peirato-recording-*.mid")
	if err != nil {
		return err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	midiFile, err := ParseMidiFile(path)
	if err != nil {
		return err
	}
	midiFile.ID = "recording:" + snap.ID
	return MidiPlayer.Start(midiFile, MidiPlaybackOptions{
		MidiID: midiFile.ID, Mode: MidiPlaybackModePlay,
		HandMode: MidiHandModeBoth, Speed: 1,
	})
}

func (k *Keyboard) StopRecordingPreview() error {
	state := MidiPlayer.State()
	if strings.HasPrefix(state.MidiID, "recording:") {
		return MidiPlayer.Stop()
	}
	return nil
}

func (k *Keyboard) DiscardRecording() error {
	inputEventsMu.Lock()
	loadRecordingDraftLocked()
	if recording.status == "recording" || recording.saving {
		inputEventsMu.Unlock()
		return errors.New("请先停止录制，且等待保存操作完成")
	}
	recording.status, recording.id, recording.lastError = "idle", "", ""
	recording.events = nil
	recording.durationUS, recording.noteCount = 0, 0
	inputEventsMu.Unlock()
	_ = k.StopRecordingPreview()
	err := os.Remove(recordingDraftPath())
	if os.IsNotExist(err) { return nil }
	return err
}

func (k *Keyboard) SaveRecording(name string) (UserMidi, error) {
	inputEventsMu.Lock()
	loadRecordingDraftLocked()
	if recording.saving {
		inputEventsMu.Unlock()
		return UserMidi{}, errors.New("录制正在保存")
	}
	if recording.status != "draft" {
		inputEventsMu.Unlock()
		return UserMidi{}, errors.New("请先停止录制")
	}
	recording.saving = true
	inputEventsMu.Unlock()
	defer func() {
		inputEventsMu.Lock()
		recording.saving = false
		inputEventsMu.Unlock()
	}()
	snap, err := recordingDraftSnapshot()
	if err != nil {
		return UserMidi{}, err
	}
	data, err := recordingBytes(snap)
	if err != nil {
		return UserMidi{}, err
	}
	if App == nil {
		return UserMidi{}, errors.New("应用尚未初始化")
	}
	name = strings.TrimSpace(name)
	name = strings.NewReplacer("/", "-", "\", "-", ":", "-", "*", "-", "?", "-", """, "-", "<", "-", ">", "-", "|", "-").Replace(name)
	if name == "" {
		name = "自由演奏-" + time.Now().Format("20060102-150405")
	}
	if strings.EqualFold(filepath.Ext(name), ".mid") { name = name[:len(name)-4] }
	if len([]rune(name)) > 120 {
		name = string([]rune(name)[:120])
	}
	path, err := App.Dialog.SaveFile().
		SetTitle("保存录制 MIDI").
		SetFilename(name + ".mid").
		AddFilter("MIDI 文件", "*.mid").
		PromptForSingleSelection()
	if err != nil {
		return UserMidi{}, err
	}
	if strings.TrimSpace(path) == "" {
		return UserMidi{}, errors.New("已取消保存，录制草稿仍然保留")
	}
	if filepath.Ext(path) == "" {
		path += ".mid"
	}
	if strings.ToLower(filepath.Ext(path)) != ".mid" {
		return UserMidi{}, errors.New("录制只能保存为 .mid 文件")
	}
	// Refuse silent replacement of existing compositions.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return UserMidi{}, fmt.Errorf("无法创建 MIDI 文件（可能已存在）：%w", err)
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return UserMidi{}, err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		os.Remove(path)
		return UserMidi{}, err
	}
	if err = f.Close(); err != nil {
		os.Remove(path)
		return UserMidi{}, err
	}
	song, err := k.AddMidiByPath(path)
	if err != nil {
		// Preserve the draft for retry; the exported file stays on disk.
		return UserMidi{}, fmt.Errorf("MIDI 已导出，但未能加入曲库：%w", err)
	}
	_ = k.StopRecordingPreview()
	inputEventsMu.Lock()
	if recording.id == snap.ID && recording.status == "draft" {
		recording.status, recording.id, recording.lastError = "idle", "", ""
		recording.events = nil
		recording.durationUS, recording.noteCount = 0, 0
		_ = os.Remove(recordingDraftPath())
	}
	inputEventsMu.Unlock()
	return song, nil
}
