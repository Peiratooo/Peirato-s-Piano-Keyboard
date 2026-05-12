package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// MidiFileInfo 是前端和后端共享的 MIDI 文件模型。
// 注意：这里不要直接暴露第三方 MIDI 库的结构，后续无论替换解析实现还是优化播放器，前端都不用跟着改。
type MidiFileInfo struct {
	Name       string          `json:"name"`
	Duration   float64         `json:"duration"`
	PPQ        int             `json:"ppq"`
	BPM        float64         `json:"bpm"`
	Tracks     []MidiTrackInfo `json:"tracks"`
	Notes      []MidiNote      `json:"notes"`
	TotalNotes int             `json:"totalNotes"`
}

// MidiTrackInfo 只保存设置页需要展示的轨道概要。
// 轨道内的详细音符统一放在 MidiFileInfo.Notes，避免重复传输太多数据。
type MidiTrackInfo struct {
	Index     int        `json:"index"`
	Name      string     `json:"name"`
	NoteCount int        `json:"noteCount"`
	Notes     []MidiNote `json:"notes"`
}

// MidiNote 是播放器、跟弹模式、琴键高亮共同消费的音符结构。
// 时间单位统一使用秒；velocity 已归一到 1-127。
type MidiNote struct {
	ID         string  `json:"id"`
	Midi       uint8   `json:"midi"`
	Name       string  `json:"name"`
	Velocity   uint8   `json:"velocity"`
	Start      float64 `json:"start"`
	Duration   float64 `json:"duration"`
	End        float64 `json:"end"`
	TrackIndex int     `json:"trackIndex"`
	TrackName  string  `json:"trackName"`
	Channel    uint8   `json:"channel"`
}

// FollowGroup 是跟弹模式的后端分组结果。
// 开始时间相近的音符会合并为一组，用户需要一次性按下这一组音。
type FollowGroup struct {
	Index    int        `json:"index"`
	Time     float64    `json:"time"`
	Duration float64    `json:"duration"`
	Notes    []MidiNote `json:"notes"`
}

// FollowPracticeOptions 是跟弹练习计划的输入配置。
// 播放器本身始终按完整 MIDI 播放；区间只作用于跟弹练习，避免播放器职责被拉得过复杂。
type FollowPracticeOptions struct {
	Threshold         float64 `json:"threshold"`
	Start             float64 `json:"start"`
	End               float64 `json:"end"`
	PracticeHand      string  `json:"practiceHand"`      // single | left | right | both
	AutoPlayOtherHand bool    `json:"autoPlayOtherHand"` // 练右手时是否自动播放左手，反之亦然
}

// TrackHandAssignment 是 Go 侧根据轨道平均音高推断出的左右手归属。
// 多数钢琴 MIDI 会把左右手分在不同轨道：平均音高更低的轨道归左手，更高的轨道归右手。
// 如果 MIDI 只有一个轨道，则强制为 single，前端也只允许单手练习。
type TrackHandAssignment struct {
	TrackIndex  int     `json:"trackIndex"`
	TrackName   string  `json:"trackName"`
	NoteCount   int     `json:"noteCount"`
	AverageMidi float64 `json:"averageMidi"`
	Hand        string  `json:"hand"`
}

// FollowPracticeStep 是跟弹模式真正消费的一步。
// PracticeNotes 是用户需要按下的音；AutoPlayNotes 是非练习声部，会由程序自动播放声音。
type FollowPracticeStep struct {
	Index         int        `json:"index"`
	Time          float64    `json:"time"`
	Duration      float64    `json:"duration"`
	PracticeNotes []MidiNote `json:"practiceNotes"`
	AutoPlayNotes []MidiNote `json:"autoPlayNotes"`
	AllNotes      []MidiNote `json:"allNotes"`
}

// FollowPracticePlan 是跟弹模式的完整计划。
// 前端只负责根据这个结构展示、判断用户按键、切换步骤，不需要再理解轨道和左右手推断细节。
type FollowPracticePlan struct {
	Steps              []FollowPracticeStep  `json:"steps"`
	Assignments        []TrackHandAssignment `json:"assignments"`
	AvailableHands     []string              `json:"availableHands"`
	PracticeHand       string                `json:"practiceHand"`
	SingleTrackOnly    bool                  `json:"singleTrackOnly"`
	Start              float64               `json:"start"`
	End                float64               `json:"end"`
	AutoPlayOtherHand  bool                  `json:"autoPlayOtherHand"`
	TotalPracticeNotes int                   `json:"totalPracticeNotes"`
	TotalAutoPlayNotes int                   `json:"totalAutoPlayNotes"`
}

type tempoEvent struct {
	tick                int64
	microsecondsPerBeat int64
}

type rawNote struct {
	midi       uint8
	velocity   uint8
	startTick  int64
	endTick    int64
	trackIndex int
	trackName  string
	channel    uint8
}

type startedNote struct {
	midi       uint8
	velocity   uint8
	startTick  int64
	trackIndex int
	trackName  string
	channel    uint8
}

type trackParseResult struct {
	name   string
	notes  []rawNote
	tempos []tempoEvent
}

// ParseMidiFileBytes 解析标准 MIDI 文件。这里实现一个轻量 SMF 解析器，而不是继续依赖前端 @tonejs/midi。
// 迁到 Go 的好处：大文件解析和后续播放调度不再压在 WebView 主线程上，UI 会更稳。
func ParseMidiFileBytes(fileName string, data []byte) (MidiFileInfo, error) {
	reader := bytes.NewReader(data)

	headerID, err := readString(reader, 4)
	if err != nil || headerID != "MThd" {
		return MidiFileInfo{}, fmt.Errorf("不是有效的 MIDI 文件")
	}

	headerLength, err := readUint32(reader)
	if err != nil {
		return MidiFileInfo{}, fmt.Errorf("读取 MIDI 头失败: %w", err)
	}
	if headerLength < 6 {
		return MidiFileInfo{}, fmt.Errorf("MIDI 头长度异常")
	}

	format, err := readUint16(reader)
	if err != nil {
		return MidiFileInfo{}, err
	}
	trackCount, err := readUint16(reader)
	if err != nil {
		return MidiFileInfo{}, err
	}
	division, err := readUint16(reader)
	if err != nil {
		return MidiFileInfo{}, err
	}

	// 最高位为 1 表示 SMPTE time division。当前项目优先支持常见 PPQ MIDI。
	if division&0x8000 != 0 {
		return MidiFileInfo{}, fmt.Errorf("暂不支持 SMPTE 时间格式的 MIDI 文件")
	}
	ppq := int(division)
	if ppq <= 0 {
		return MidiFileInfo{}, fmt.Errorf("MIDI PPQ 异常")
	}

	if headerLength > 6 {
		if _, err := reader.Seek(int64(headerLength-6), 1); err != nil {
			return MidiFileInfo{}, err
		}
	}

	parsedTracks := make([]trackParseResult, 0, trackCount)
	allTempos := []tempoEvent{{tick: 0, microsecondsPerBeat: 500000}}

	for trackIndex := 0; trackIndex < int(trackCount); trackIndex++ {
		trackID, err := readString(reader, 4)
		if err != nil {
			return MidiFileInfo{}, fmt.Errorf("读取轨道 %d 失败: %w", trackIndex+1, err)
		}
		if trackID != "MTrk" {
			return MidiFileInfo{}, fmt.Errorf("轨道 %d 缺少 MTrk 标记", trackIndex+1)
		}

		trackLength, err := readUint32(reader)
		if err != nil {
			return MidiFileInfo{}, err
		}
		trackData := make([]byte, trackLength)
		if _, err := reader.Read(trackData); err != nil {
			return MidiFileInfo{}, fmt.Errorf("读取轨道 %d 数据失败: %w", trackIndex+1, err)
		}

		result, err := parseMidiTrack(trackData, trackIndex)
		if err != nil {
			return MidiFileInfo{}, fmt.Errorf("解析轨道 %d 失败: %w", trackIndex+1, err)
		}
		parsedTracks = append(parsedTracks, result)
		allTempos = append(allTempos, result.tempos...)
	}

	if format > 2 {
		return MidiFileInfo{}, fmt.Errorf("不支持的 MIDI format: %d", format)
	}

	tempoMap := normalizeTempoMap(allTempos)
	tracks := make([]MidiTrackInfo, 0, len(parsedTracks))
	allNotes := make([]MidiNote, 0)

	for trackIndex, track := range parsedTracks {
		trackName := strings.TrimSpace(track.name)
		if trackName == "" {
			trackName = fmt.Sprintf("Track %d", trackIndex+1)
		}

		trackNotes := make([]MidiNote, 0, len(track.notes))
		for noteIndex, note := range track.notes {
			start := ticksToSeconds(note.startTick, ppq, tempoMap)
			end := ticksToSeconds(note.endTick, ppq, tempoMap)
			if end <= start {
				// 防御性处理：极短音符也给一个最小时长，避免前端高亮一闪而过或 stop 逻辑错乱。
				end = start + 0.01
			}

			normalized := MidiNote{
				ID:         fmt.Sprintf("%d-%d", trackIndex, noteIndex),
				Midi:       note.midi,
				Name:       midiNoteName(note.midi),
				Velocity:   note.velocity,
				Start:      roundTime(start),
				Duration:   roundTime(end - start),
				End:        roundTime(end),
				TrackIndex: trackIndex,
				TrackName:  trackName,
				Channel:    note.channel,
			}
			trackNotes = append(trackNotes, normalized)
			allNotes = append(allNotes, normalized)
		}

		tracks = append(tracks, MidiTrackInfo{
			Index:     trackIndex,
			Name:      trackName,
			NoteCount: len(trackNotes),
			Notes:     trackNotes,
		})
	}

	sort.SliceStable(allNotes, func(i, j int) bool {
		if allNotes[i].Start == allNotes[j].Start {
			return allNotes[i].Midi < allNotes[j].Midi
		}
		return allNotes[i].Start < allNotes[j].Start
	})

	duration := 0.0
	for _, note := range allNotes {
		if note.End > duration {
			duration = note.End
		}
	}

	return MidiFileInfo{
		Name:       fileName,
		Duration:   roundTime(duration),
		PPQ:        ppq,
		BPM:        roundTime(60000000 / float64(tempoMap[0].microsecondsPerBeat)),
		Tracks:     tracks,
		Notes:      allNotes,
		TotalNotes: len(allNotes),
	}, nil
}

// ParseMidiFileBase64 供 Wails 绑定调用。前端读取文件为 base64 后传给 Go 解析。
func (k *Keyboard) ParseMidiFileBase64(fileName string, encoded string) (MidiFileInfo, error) {
	data, err := decodeBase64Payload(encoded)
	if err != nil {
		return MidiFileInfo{}, err
	}
	return ParseMidiFileBytes(fileName, data)
}

// BuildFollowGroups 供前端跟弹模式调用。threshold 单位是秒，建议默认 0.06。
func (k *Keyboard) BuildFollowGroups(notes []MidiNote, threshold float64) []FollowGroup {
	return BuildFollowGroups(notes, threshold)
}

func BuildFollowGroups(notes []MidiNote, threshold float64) []FollowGroup {
	if threshold <= 0 {
		threshold = 0.06
	}
	ordered := append([]MidiNote(nil), notes...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Start == ordered[j].Start {
			return ordered[i].Midi < ordered[j].Midi
		}
		return ordered[i].Start < ordered[j].Start
	})

	groups := make([]FollowGroup, 0)
	for _, note := range ordered {
		if len(groups) == 0 || math.Abs(note.Start-groups[len(groups)-1].Time) > threshold {
			groups = append(groups, FollowGroup{
				Index:    len(groups),
				Time:     note.Start,
				Duration: note.Duration,
				Notes:    []MidiNote{note},
			})
			continue
		}
		last := &groups[len(groups)-1]
		last.Notes = append(last.Notes, note)
		if note.Duration > last.Duration {
			last.Duration = note.Duration
		}
	}
	return groups
}

// BuildFollowPracticePlan 供 Wails 绑定调用，负责把 MIDI notes 变成“可练习”的步骤。
// 这里放在 Go 侧是为了后续支持更大的 MIDI、轨道筛选、左右手算法升级时，不阻塞 WebView UI。
func (k *Keyboard) BuildFollowPracticePlan(notes []MidiNote, options FollowPracticeOptions) FollowPracticePlan {
	return BuildFollowPracticePlan(notes, options)
}

func BuildFollowPracticePlan(notes []MidiNote, options FollowPracticeOptions) FollowPracticePlan {
	threshold := options.Threshold
	if threshold <= 0 {
		threshold = 0.06
	}

	ordered := append([]MidiNote(nil), notes...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Start == ordered[j].Start {
			return ordered[i].Midi < ordered[j].Midi
		}
		return ordered[i].Start < ordered[j].Start
	})

	start := math.Max(0, options.Start)
	end := options.End
	if end <= start {
		end = 0
	}

	filtered := make([]MidiNote, 0, len(ordered))
	for _, note := range ordered {
		if note.Start < start {
			continue
		}
		if end > 0 && note.Start > end {
			continue
		}
		filtered = append(filtered, note)
	}

	assignments := inferTrackHands(filtered)
	assignmentMap := make(map[int]string, len(assignments))
	for _, assignment := range assignments {
		assignmentMap[assignment.TrackIndex] = assignment.Hand
	}

	singleTrackOnly := len(assignments) <= 1
	practiceHand := normalizePracticeHand(options.PracticeHand, singleTrackOnly)
	availableHands := []string{"single"}
	if !singleTrackOnly {
		availableHands = []string{"both", "right", "left"}
	}

	steps := make([]FollowPracticeStep, 0)
	for _, note := range filtered {
		if len(steps) == 0 || math.Abs(note.Start-steps[len(steps)-1].Time) > threshold {
			steps = append(steps, FollowPracticeStep{
				Index:    len(steps),
				Time:     note.Start,
				Duration: note.Duration,
				AllNotes: []MidiNote{},
			})
		}

		last := &steps[len(steps)-1]
		last.AllNotes = append(last.AllNotes, note)
		if note.Duration > last.Duration {
			last.Duration = note.Duration
		}

		noteHand := assignmentMap[note.TrackIndex]
		if noteHand == "" {
			noteHand = "single"
		}
		if shouldPracticeNote(noteHand, practiceHand, singleTrackOnly) {
			last.PracticeNotes = append(last.PracticeNotes, note)
		} else if options.AutoPlayOtherHand {
			last.AutoPlayNotes = append(last.AutoPlayNotes, note)
		}
	}

	totalPractice := 0
	totalAuto := 0
	for i := range steps {
		steps[i].Index = i
		totalPractice += len(steps[i].PracticeNotes)
		totalAuto += len(steps[i].AutoPlayNotes)
	}

	return FollowPracticePlan{
		Steps:              steps,
		Assignments:        assignments,
		AvailableHands:     availableHands,
		PracticeHand:       practiceHand,
		SingleTrackOnly:    singleTrackOnly,
		Start:              start,
		End:                end,
		AutoPlayOtherHand:  options.AutoPlayOtherHand,
		TotalPracticeNotes: totalPractice,
		TotalAutoPlayNotes: totalAuto,
	}
}

var followAutoPlay = &followAutoPlayRuntime{}

// followAutoPlayRuntime 管理跟弹模式里的“非练习声部自动播放”。
// 跟弹页面切换步骤、停止练习或重新开始时，都需要取消上一组自动播放，避免旧声部继续发声或误改琴键高亮。
type followAutoPlayRuntime struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

// PlayFollowAutoNotes 播放跟弹计划中的非练习声部。
// 前端只需把当前 step.AutoPlayNotes 传进来；Go 负责真实发声、UI playback 高亮和按时停止。
func (k *Keyboard) PlayFollowAutoNotes(notes []MidiNote, playbackRate float64) {
	followAutoPlay.play(k, notes, playbackRate)
}

// StopFollowAutoNotes 只停止跟弹自动伴奏，不会清空用户手指正在按的 pressedKey。
// 主要用于“上一步 / 下一步 / 停止跟弹”，避免旧步骤的伴奏残留。
func (k *Keyboard) StopFollowAutoNotes() {
	followAutoPlay.stop(k)
}

func (r *followAutoPlayRuntime) play(k *Keyboard, notes []MidiNote, playbackRate float64) {
	r.stop(k)
	if len(notes) == 0 {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()

	rate := normalizePlaybackRate(playbackRate)
	for _, note := range notes {
		n := note
		go func() {
			duration := n.Duration / rate
			if duration <= 0 {
				duration = 0.08
			}

			k.KeyboardPlayWithVelocity(n.Midi, n.Velocity)
			emitPlaybackKey(n.Midi, true)

			timer := time.NewTimer(time.Duration(duration * float64(time.Second)))
			defer timer.Stop()

			select {
			case <-ctx.Done():
				k.KeyboardStop(n.Midi)
				emitPlaybackKey(n.Midi, false)
				return
			case <-timer.C:
				k.KeyboardStop(n.Midi)
				emitPlaybackKey(n.Midi, false)
			}
		}()
	}
}

func (r *followAutoPlayRuntime) stop(k *Keyboard) {
	r.mu.Lock()
	cancel := r.cancel
	r.cancel = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	// 这里只清理“播放高亮”，不调用 AllNotesOff。
	// 原因：跟弹切换步骤时会先停止上一组自动伴奏，再设置下一组提示键；
	// 如果这里触发 AllNotesOff，会把前端 hintKey / pressedKey 一起清掉，导致提示闪烁或丢失。
	emitPlaybackClear()
}

func inferTrackHands(notes []MidiNote) []TrackHandAssignment {
	type trackStat struct {
		index int
		name  string
		count int
		sum   int
	}

	stats := map[int]*trackStat{}
	for _, note := range notes {
		stat := stats[note.TrackIndex]
		if stat == nil {
			stat = &trackStat{index: note.TrackIndex, name: note.TrackName}
			stats[note.TrackIndex] = stat
		}
		stat.count++
		stat.sum += int(note.Midi)
		if strings.TrimSpace(stat.name) == "" {
			stat.name = fmt.Sprintf("Track %d", note.TrackIndex+1)
		}
	}

	assignments := make([]TrackHandAssignment, 0, len(stats))
	for _, stat := range stats {
		avg := 0.0
		if stat.count > 0 {
			avg = float64(stat.sum) / float64(stat.count)
		}
		assignments = append(assignments, TrackHandAssignment{
			TrackIndex:  stat.index,
			TrackName:   stat.name,
			NoteCount:   stat.count,
			AverageMidi: roundTime(avg),
		})
	}

	sort.SliceStable(assignments, func(i, j int) bool {
		if assignments[i].AverageMidi == assignments[j].AverageMidi {
			return assignments[i].TrackIndex < assignments[j].TrackIndex
		}
		return assignments[i].AverageMidi < assignments[j].AverageMidi
	})

	if len(assignments) <= 1 {
		for i := range assignments {
			assignments[i].Hand = "single"
		}
		return assignments
	}

	// 平均音高低的一半视为左手，高的一半视为右手。
	// 这是一个实用启发式：后续如果你加入轨道手动标记，可以直接覆盖这里的结果。
	leftCount := len(assignments) / 2
	for i := range assignments {
		if i < leftCount {
			assignments[i].Hand = "left"
		} else {
			assignments[i].Hand = "right"
		}
	}

	sort.SliceStable(assignments, func(i, j int) bool {
		return assignments[i].TrackIndex < assignments[j].TrackIndex
	})
	return assignments
}

func normalizePracticeHand(hand string, singleTrackOnly bool) string {
	if singleTrackOnly {
		return "single"
	}
	switch strings.ToLower(strings.TrimSpace(hand)) {
	case "left", "right", "both":
		return strings.ToLower(strings.TrimSpace(hand))
	default:
		return "right"
	}
}

func shouldPracticeNote(noteHand string, practiceHand string, singleTrackOnly bool) bool {
	if singleTrackOnly || practiceHand == "single" || practiceHand == "both" {
		return true
	}
	return noteHand == practiceHand
}

func parseMidiTrack(data []byte, trackIndex int) (trackParseResult, error) {
	reader := bytes.NewReader(data)
	absoluteTick := int64(0)
	var runningStatus byte
	trackName := ""
	openNotes := make(map[string][]startedNote)
	result := trackParseResult{}

	for reader.Len() > 0 {
		delta, err := readVarInt(reader)
		if err != nil {
			return result, err
		}
		absoluteTick += int64(delta)

		statusByte, err := reader.ReadByte()
		if err != nil {
			return result, err
		}

		if statusByte < 0x80 {
			if runningStatus == 0 {
				return result, fmt.Errorf("running status 缺失")
			}
			_ = reader.UnreadByte()
			statusByte = runningStatus
		} else if statusByte < 0xF0 {
			runningStatus = statusByte
		}

		switch {
		case statusByte == 0xFF:
			metaType, err := reader.ReadByte()
			if err != nil {
				return result, err
			}
			length, err := readVarInt(reader)
			if err != nil {
				return result, err
			}
			payload := make([]byte, length)
			if _, err := reader.Read(payload); err != nil {
				return result, err
			}

			switch metaType {
			case 0x03:
				trackName = string(payload)
			case 0x51:
				if len(payload) == 3 {
					us := int64(payload[0])<<16 | int64(payload[1])<<8 | int64(payload[2])
					if us > 0 {
						result.tempos = append(result.tempos, tempoEvent{tick: absoluteTick, microsecondsPerBeat: us})
					}
				}
			case 0x2F:
				result.name = trackName
				return result, nil
			}

		case statusByte == 0xF0 || statusByte == 0xF7:
			length, err := readVarInt(reader)
			if err != nil {
				return result, err
			}
			if _, err := reader.Seek(int64(length), 1); err != nil {
				return result, err
			}

		default:
			messageType := statusByte & 0xF0
			channel := statusByte & 0x0F

			switch messageType {
			case 0x80, 0x90:
				key, err := reader.ReadByte()
				if err != nil {
					return result, err
				}
				velocity, err := reader.ReadByte()
				if err != nil {
					return result, err
				}

				mapKey := fmt.Sprintf("%d:%d", channel, key)
				if messageType == 0x90 && velocity > 0 {
					openNotes[mapKey] = append(openNotes[mapKey], startedNote{
						midi:       key,
						velocity:   normalizeVelocity(velocity),
						startTick:  absoluteTick,
						trackIndex: trackIndex,
						trackName:  trackName,
						channel:    channel,
					})
					continue
				}

				queue := openNotes[mapKey]
				if len(queue) == 0 {
					continue
				}
				started := queue[0]
				if len(queue) == 1 {
					delete(openNotes, mapKey)
				} else {
					openNotes[mapKey] = queue[1:]
				}
				result.notes = append(result.notes, rawNote{
					midi:       started.midi,
					velocity:   started.velocity,
					startTick:  started.startTick,
					endTick:    absoluteTick,
					trackIndex: started.trackIndex,
					trackName:  started.trackName,
					channel:    started.channel,
				})

			case 0xA0, 0xB0, 0xE0:
				if _, err := reader.ReadByte(); err != nil {
					return result, err
				}
				if _, err := reader.ReadByte(); err != nil {
					return result, err
				}
			case 0xC0, 0xD0:
				if _, err := reader.ReadByte(); err != nil {
					return result, err
				}
			default:
				return result, fmt.Errorf("不支持的 MIDI 事件: 0x%X", statusByte)
			}
		}
	}

	result.name = trackName
	return result, nil
}

func normalizeTempoMap(events []tempoEvent) []tempoEvent {
	if len(events) == 0 {
		return []tempoEvent{{tick: 0, microsecondsPerBeat: 500000}}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].tick == events[j].tick {
			return events[i].microsecondsPerBeat < events[j].microsecondsPerBeat
		}
		return events[i].tick < events[j].tick
	})

	result := []tempoEvent{{tick: 0, microsecondsPerBeat: 500000}}
	for _, event := range events {
		if event.microsecondsPerBeat <= 0 {
			continue
		}
		if len(result) > 0 && result[len(result)-1].tick == event.tick {
			result[len(result)-1] = event
			continue
		}
		result = append(result, event)
	}
	return result
}

func ticksToSeconds(tick int64, ppq int, tempos []tempoEvent) float64 {
	if tick <= 0 || ppq <= 0 {
		return 0
	}
	seconds := 0.0
	lastTick := int64(0)
	currentTempo := int64(500000)

	for _, tempo := range tempos {
		if tempo.tick <= 0 {
			currentTempo = tempo.microsecondsPerBeat
			continue
		}
		if tempo.tick >= tick {
			break
		}
		if tempo.tick > lastTick {
			seconds += ticksDeltaToSeconds(tempo.tick-lastTick, ppq, currentTempo)
			lastTick = tempo.tick
		}
		currentTempo = tempo.microsecondsPerBeat
	}

	if tick > lastTick {
		seconds += ticksDeltaToSeconds(tick-lastTick, ppq, currentTempo)
	}
	return seconds
}

func ticksDeltaToSeconds(delta int64, ppq int, microsecondsPerBeat int64) float64 {
	return float64(delta) * float64(microsecondsPerBeat) / float64(ppq) / 1000000
}

func readString(reader *bytes.Reader, length int) (string, error) {
	buf := make([]byte, length)
	_, err := reader.Read(buf)
	return string(buf), err
}

func readUint16(reader *bytes.Reader) (uint16, error) {
	var value uint16
	err := binary.Read(reader, binary.BigEndian, &value)
	return value, err
}

func readUint32(reader *bytes.Reader) (uint32, error) {
	var value uint32
	err := binary.Read(reader, binary.BigEndian, &value)
	return value, err
}

func readVarInt(reader *bytes.Reader) (uint32, error) {
	var value uint32
	for i := 0; i < 4; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		value = (value << 7) | uint32(b&0x7F)
		if b&0x80 == 0 {
			return value, nil
		}
	}
	return 0, fmt.Errorf("可变长度整数过长")
}

func decodeBase64Payload(encoded string) ([]byte, error) {
	payload := strings.TrimSpace(encoded)
	if comma := strings.Index(payload, ","); comma >= 0 {
		payload = payload[comma+1:]
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil, fmt.Errorf("文件内容为空")
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("base64 解码失败: %w", err)
	}
	return data, nil
}

func normalizeVelocity(value byte) uint8 {
	if value == 0 {
		return 1
	}
	if value > 127 {
		return 127
	}
	return uint8(value)
}

func midiNoteName(midi uint8) string {
	names := []string{"C", "C#", "D", "Eb", "E", "F", "F#", "G", "Ab", "A", "Bb", "B"}
	octave := int(midi)/12 - 1
	return fmt.Sprintf("%s%d", names[int(midi)%12], octave)
}

func roundTime(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}
