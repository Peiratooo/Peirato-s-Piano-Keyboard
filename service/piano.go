package service

import (
	"fmt"
	"sync"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

type KeyboardSignal struct {
	Value    uint8 `json:"value"`
	Velocity uint8 `json:"velocity"`
	Channel  uint8 `json:"channel"`
}

type Listener struct {
	Down    chan bool
	Started bool
}

type InMidiDevice struct {
	// Device 是 Go 后端内部使用的驱动对象，不能直接暴露给前端。
	Device drivers.In `json:"-"`
	Name   string     `json:"name"`
	Value  int        `json:"value"`
}

type OutMidiDevice struct {
	Device drivers.Out `json:"-"`
	Name   string      `json:"name"`
	Value  int         `json:"value"`
}

type PedalSingal struct {
	DeviceID        int             `json:"deviceID"`
	DamperPedal     bool            `json:"damperPedal"`    // 延音踏板 64
	SostenutoPedal  bool            `json:"sostenutoPedal"` // 持音踏板 66
	SoftPedal       bool            `json:"softPedal"`      // 柔音踏板 67
	DamperPedalKeys []uint8         `json:"-"`
	DownKeys        map[uint8]uint8 `json:"-"` // key -> channel
}

type MidiDevices struct {
	InMidiPool        map[int]InMidiDevice  `json:"inMidiPool"`
	OutMidiPool       map[int]OutMidiDevice `json:"outMidiPool"`
	SelectedInDevice  int                   `json:"selectedInDevice"`
	SelectedOutDevice int                   `json:"selectedOutDevice"`
	PedalStatus       map[int]*PedalSingal  `json:"pedalStatus"`
	Listener          Listener              `json:"-"`
	Initialized       bool                  `json:"initialized"`
}

var (
	midiMu           sync.RWMutex
	midiListenerStop func()
)

const (
	midiOutputNone          = -1
	midiOutputSoftwareSynth = -2
)

var Midis = MidiDevices{
	InMidiPool: map[int]InMidiDevice{
		-1: {
			Name:  "无",
			Value: -1,
		},
	},
	OutMidiPool: map[int]OutMidiDevice{
		midiOutputNone: {
			Name:  "无",
			Value: midiOutputNone,
		},
		midiOutputSoftwareSynth: {
			Name:  "软件音源",
			Value: midiOutputSoftwareSynth,
		},
	},
	SelectedInDevice:  -1,
	SelectedOutDevice: midiOutputSoftwareSynth,
	PedalStatus:       make(map[int]*PedalSingal),
	Listener: Listener{
		Down:    make(chan bool),
		Started: false,
	},
	Initialized: false,
}

type Keyboard struct{}

func newPedalSignal(deviceID int) *PedalSingal {
	return &PedalSingal{
		DeviceID:        deviceID,
		DamperPedal:     false,
		SostenutoPedal:  false,
		SoftPedal:       false,
		DamperPedalKeys: make([]uint8, 0),
		DownKeys:        make(map[uint8]uint8),
	}
}

// snapshotMidiDevices 生成一份只包含前端需要字段的快照。
// 注意不要把 drivers.In / drivers.Out 直接返回给前端，避免序列化不稳定。
func snapshotMidiDevices() MidiDevices {
	midiMu.RLock()
	defer midiMu.RUnlock()

	snapshot := MidiDevices{
		InMidiPool:        make(map[int]InMidiDevice, len(Midis.InMidiPool)),
		OutMidiPool:       make(map[int]OutMidiDevice, len(Midis.OutMidiPool)),
		SelectedInDevice:  Midis.SelectedInDevice,
		SelectedOutDevice: Midis.SelectedOutDevice,
		PedalStatus:       make(map[int]*PedalSingal, len(Midis.PedalStatus)),
		Initialized:       Midis.Initialized,
	}

	for id, device := range Midis.InMidiPool {
		snapshot.InMidiPool[id] = InMidiDevice{Name: device.Name, Value: device.Value}
	}
	for id, device := range Midis.OutMidiPool {
		snapshot.OutMidiPool[id] = OutMidiDevice{Name: device.Name, Value: device.Value}
	}
	for id, pedal := range Midis.PedalStatus {
		if pedal == nil {
			continue
		}
		snapshot.PedalStatus[id] = &PedalSingal{
			DeviceID:       pedal.DeviceID,
			DamperPedal:    pedal.DamperPedal,
			SostenutoPedal: pedal.SostenutoPedal,
			SoftPedal:      pedal.SoftPedal,
		}
	}

	return snapshot
}

func CloseMidiDevice() {
	(&Keyboard{}).MidiListenerStop()
	(&Keyboard{}).AllNotesOff()

	midiMu.Lock()
	defer midiMu.Unlock()
	for id, device := range Midis.InMidiPool {
		if id != -1 && device.Device != nil {
			_ = device.Device.Close()
		}
	}
	for id, device := range Midis.OutMidiPool {
		if id != -1 && device.Device != nil {
			_ = device.Device.Close()
		}
	}

	midi.CloseDriver()
	drivers.Close()
	fmt.Println("midi devices closed")
}

func CompareInDevices(inports midi.InPorts) {
	midiMu.Lock()
	defer midiMu.Unlock()

	lastID := -1
	alive := map[int]bool{-1: true}

	for _, port := range inports {
		deviceID := port.Number()
		alive[deviceID] = true
		lastID = deviceID

		if _, ok := Midis.InMidiPool[deviceID]; ok {
			continue
		}

		Midis.InMidiPool[deviceID] = InMidiDevice{
			Device: port,
			Name:   port.String(),
			Value:  deviceID,
		}
		Midis.PedalStatus[deviceID] = newPedalSignal(deviceID)
	}

	if Midis.SelectedInDevice == -1 && lastID != -1 {
		Midis.SelectedInDevice = lastID
	}

	for id, device := range Midis.InMidiPool {
		if alive[id] {
			continue
		}
		if id == Midis.SelectedInDevice {
			Midis.SelectedInDevice = -1
			// 当前输入设备被拔掉时，停止监听并清理本地音符，避免残留卡音。
			go (&Keyboard{}).MidiListenerStop()
			go (&Keyboard{}).AllNotesOff()
		}
		if id != -1 && device.Device != nil {
			_ = device.Device.Close()
		}
		delete(Midis.PedalStatus, id)
		delete(Midis.InMidiPool, id)
	}
}

func CompareOutDevices(outports midi.OutPorts) {
	midiMu.Lock()
	defer midiMu.Unlock()

	alive := map[int]bool{
		midiOutputNone:          true,
		midiOutputSoftwareSynth: true,
	}

	for _, port := range outports {
		deviceID := port.Number()
		alive[deviceID] = true

		if _, ok := Midis.OutMidiPool[deviceID]; ok {
			continue
		}

		if err := port.Open(); err != nil {
			fmt.Println("打开 MIDI 输出设备失败:", err)
			continue
		}

		Midis.OutMidiPool[deviceID] = OutMidiDevice{
			Device: port,
			Name:   port.String(),
			Value:  deviceID,
		}
	}

	for id, device := range Midis.OutMidiPool {
		if alive[id] {
			continue
		}
		if id == Midis.SelectedOutDevice {
			Midis.SelectedOutDevice = midiOutputSoftwareSynth
			go (&Keyboard{}).AllNotesOff()
		}
		if id != -1 && device.Device != nil {
			_ = device.Device.Close()
		}
		delete(Midis.OutMidiPool, id)
	}
}

func ListenMidiDevices() {
	CompareInDevices(midi.GetInPorts())
	CompareOutDevices(midi.GetOutPorts())

	midiMu.Lock()
	Midis.Initialized = true
	midiMu.Unlock()

	if App != nil {
		App.Event.Emit("devices", snapshotMidiDevices())
	}
}

func (k *Keyboard) GetMidiDevices() MidiDevices {
	return snapshotMidiDevices()
}

func (k *Keyboard) MidiListenerStart() {
	midiMu.Lock()
	if Midis.Listener.Started || Midis.SelectedInDevice == -1 {
		midiMu.Unlock()
		return
	}
	deviceID := Midis.SelectedInDevice
	device, ok := Midis.InMidiPool[deviceID]
	if !ok || device.Device == nil {
		midiMu.Unlock()
		return
	}
	midiMu.Unlock()

	fmt.Println("midi listener start")
	stop, err := midi.ListenTo(device.Device, func(msg midi.Message, timestampms int32) {
		handleMidiMessage(deviceID, msg)
	}, midi.UseSysEx())
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	midiMu.Lock()
	midiListenerStop = stop
	Midis.Listener.Started = true
	midiMu.Unlock()
}

func handleMidiMessage(deviceID int, msg midi.Message) {
	var bt []byte
	var ch, key, vel, con uint8

	switch {
	case msg.GetSysEx(&bt):
		return

	case msg.GetNoteStart(&ch, &key, &vel):
		midiKey := midi.Note(key).Value()
		playSelectedOutputNoteOn(ch, key, vel)
		MidiPlayer.HandleUserNoteOn(int(midiKey))

		midiMu.Lock()
		pedal := Midis.PedalStatus[deviceID]
		if pedal == nil {
			pedal = newPedalSignal(deviceID)
			Midis.PedalStatus[deviceID] = pedal
		}
		pedal.DownKeys[midiKey] = ch
		midiMu.Unlock()

		emitKeyboardEvent("down", midiKey, vel, ch)
		emitKeyboardEvent("pressedDown", midiKey, vel, ch)

	case msg.GetNoteEnd(&ch, &key):
		midiKey := midi.Note(key).Value()
		shouldReleaseVisualNow := true

		midiMu.Lock()
		pedal := Midis.PedalStatus[deviceID]
		if pedal == nil {
			pedal = newPedalSignal(deviceID)
			Midis.PedalStatus[deviceID] = pedal
		}
		delete(pedal.DownKeys, midiKey)
		if pedal.DamperPedal {
			shouldReleaseVisualNow = false
			if !containsUint8(pedal.DamperPedalKeys, midiKey) {
				pedal.DamperPedalKeys = append(pedal.DamperPedalKeys, midiKey)
			}
		}
		midiMu.Unlock()

		emitKeyboardEvent("pressedUp", midiKey, 0, ch)
		playSelectedOutputNoteOff(ch, key)
		if shouldReleaseVisualNow {
			emitKeyboardEvent("up", midiKey, 0, ch)
		}

	case msg.GetControlChange(&ch, &con, &vel):
		handlePedalMessage(deviceID, ch, con, vel)

	case msg.GetProgramChange(&ch, &con):
		playSelectedOutputProgramChange(ch, con)
	}
}

func handlePedalMessage(deviceID int, channel, controller, velocity uint8) {
	playSelectedOutputControlChange(channel, controller, velocity)

	if controller != 64 && controller != 66 && controller != 67 {
		return
	}

	var releaseKeys []uint8
	midiMu.Lock()
	pedal := Midis.PedalStatus[deviceID]
	if pedal == nil {
		pedal = newPedalSignal(deviceID)
		Midis.PedalStatus[deviceID] = pedal
	}

	switch controller {
	case 64:
		pedal.DamperPedal = velocity >= 64
		if !pedal.DamperPedal {
			for _, sustainedKey := range pedal.DamperPedalKeys {
				if _, stillDown := pedal.DownKeys[sustainedKey]; !stillDown {
					releaseKeys = append(releaseKeys, sustainedKey)
				}
			}
			pedal.DamperPedalKeys = make([]uint8, 0)
		}
	case 66:
		pedal.SostenutoPedal = velocity > 0
	case 67:
		pedal.SoftPedal = velocity > 0
	}
	pedalSnapshot := &PedalSingal{
		DeviceID:       pedal.DeviceID,
		DamperPedal:    pedal.DamperPedal,
		SostenutoPedal: pedal.SostenutoPedal,
		SoftPedal:      pedal.SoftPedal,
	}
	midiMu.Unlock()

	if App != nil {
		App.Event.Emit("pedal", pedalSnapshot)
	}
	for _, key := range releaseKeys {
		emitKeyboardEvent("up", key, 0, channel)
	}
}

func emitKeyboardEvent(event string, key uint8, velocity uint8, channel uint8) {
	if App == nil {
		return
	}
	App.Event.Emit(event, &KeyboardSignal{Value: key, Velocity: velocity, Channel: channel})
}

func containsUint8(list []uint8, target uint8) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func (k *Keyboard) KeyboardPlay(key uint8) {
	config := GetUserConfig()
	MidiPlayer.HandleUserNoteOn(int(key))
	if selectedOut, _, _ := currentOutputDevice(); selectedOut == midiOutputSoftwareSynth {
		// A MIDI file can leave CC7/CC11 at zero on the configured channel.
		// Restore the interactive piano channel before a computer/mouse key press.
		ProcessSynthMidiMessage(int32(config.MidiChannel), midiCommandControlChange, midiCCChannelVolume, 127)
		ProcessSynthMidiMessage(int32(config.MidiChannel), midiCommandControlChange, midiCCPan, 64)
		ProcessSynthMidiMessage(int32(config.MidiChannel), midiCommandControlChange, midiCCExpression, 127)
	}
	playSelectedOutputNoteOn(config.MidiChannel, key, config.Velocity)
}

func (k *Keyboard) KeyboardStop(key uint8) {
	config := GetUserConfig()
	playSelectedOutputNoteOff(config.MidiChannel, key)
}

func (k *Keyboard) MidiListenerStop() {
	midiMu.Lock()
	if !Midis.Listener.Started {
		midiMu.Unlock()
		return
	}
	stop := midiListenerStop
	midiListenerStop = nil
	Midis.Listener.Started = false
	midiMu.Unlock()

	if stop != nil {
		stop()
	}
	fmt.Println("midi listener stop")
}

func (k *Keyboard) ChangeDevice(deviceType string, deviceID int) bool {
	midiMu.Lock()
	defer midiMu.Unlock()

	switch deviceType {
	case "in":
		if _, ok := Midis.InMidiPool[deviceID]; !ok {
			return false
		}
		Midis.SelectedInDevice = deviceID
	case "out":
		if _, ok := Midis.OutMidiPool[deviceID]; !ok {
			return false
		}
		Midis.SelectedOutDevice = deviceID
	default:
		return false
	}
	return true
}

func (k *Keyboard) AllNotesOff() {
	AllSynthNotesOff()

	midiMu.RLock()
	selectedOut := Midis.SelectedOutDevice
	outDevice, ok := Midis.OutMidiPool[selectedOut]
	midiMu.RUnlock()

	if ok && selectedOut != midiOutputNone && selectedOut != midiOutputSoftwareSynth && outDevice.Device != nil {
		for channel := uint8(0); channel < 16; channel++ {
			// Reset sustain, controllers, and sounding notes so seek/pause/loop
			// transitions cannot leave an external synth holding a note.
			_ = outDevice.Device.Send(midi.ControlChange(channel, midiCCSustain, 0))
			_ = outDevice.Device.Send(midi.ControlChange(channel, midiCCResetControllers, 0))
			_ = outDevice.Device.Send(midi.ControlChange(channel, midiCCAllNotesOff, 0))
			_ = outDevice.Device.Send(midi.ControlChange(channel, midiCCAllSoundOff, 0))
		}
		for channel := uint8(0); channel < 16; channel++ {
			for key := uint8(0); key < 128; key++ {
				_ = outDevice.Device.Send(midi.NoteOff(channel, key))
			}
		}
	}

	if App != nil {
		App.Event.Emit("allNotesOff")
	}
	emitMidiVisualClear()
}

const (
	midiCommandNoteOff       = 0x80
	midiCommandNoteOn        = 0x90
	midiCommandControlChange = 0xB0
	midiCommandProgramChange = 0xC0

	midiCCChannelVolume    = 7
	midiCCPan              = 10
	midiCCExpression       = 11
	midiCCSustain          = 64
	midiCCReverbSend       = 91
	midiCCChorusSend       = 93
	midiCCAllSoundOff      = 120
	midiCCResetControllers = 121
	midiCCAllNotesOff      = 123

	midiPercussionChannel = 9
)

func playSelectedOutputNoteOn(channel, note, velocity uint8) {
	selectedOut, outDevice, ok := currentOutputDevice()
	switch selectedOut {
	case midiOutputNone:
		return
	case midiOutputSoftwareSynth:
		if channel == midiPercussionChannel {
			return
		}
		ProcessSynthMidiMessage(int32(channel), midiCommandNoteOn, int32(note), int32(velocity))
	default:
		if !ok || outDevice.Device == nil {
			return
		}
		if err := outDevice.Device.Send(midi.NoteOn(channel, note, velocity)); err != nil {
			fmt.Println("发送 MIDI NoteOn 失败:", err)
		}
	}
}

func playSelectedOutputNoteOff(channel uint8, note uint8) {
	selectedOut, outDevice, ok := currentOutputDevice()
	switch selectedOut {
	case midiOutputNone:
		return
	case midiOutputSoftwareSynth:
		ProcessSynthMidiMessage(int32(channel), midiCommandNoteOff, int32(note), 0)
	default:
		if !ok || outDevice.Device == nil {
			return
		}
		if err := outDevice.Device.Send(midi.NoteOff(channel, note)); err != nil {
			fmt.Println("发送 MIDI NoteOff 失败:", err)
		}
	}
}

func playSelectedOutputControlChange(channel, controller, value uint8) {
	selectedOut, outDevice, ok := currentOutputDevice()
	switch selectedOut {
	case midiOutputNone:
		return
	case midiOutputSoftwareSynth:
		ProcessSynthMidiMessage(int32(channel), midiCommandControlChange, int32(controller), int32(value))
	default:
		if !ok || outDevice.Device == nil {
			return
		}
		if err := outDevice.Device.Send(midi.ControlChange(channel, controller, value)); err != nil {
			fmt.Println("发送 MIDI ControlChange 失败:", err)
		}
	}
}

func playSelectedOutputProgramChange(channel, program uint8) {
	selectedOut, outDevice, ok := currentOutputDevice()
	if selectedOut == midiOutputNone || selectedOut == midiOutputSoftwareSynth || !ok || outDevice.Device == nil {
		return
	}
	if err := outDevice.Device.Send(midi.ProgramChange(channel, program)); err != nil {
		fmt.Println("发送 MIDI ProgramChange 失败:", err)
	}
}

func currentOutputDevice() (int, OutMidiDevice, bool) {
	midiMu.RLock()
	defer midiMu.RUnlock()
	selectedOut := Midis.SelectedOutDevice
	outDevice, ok := Midis.OutMidiPool[selectedOut]
	return selectedOut, outDevice, ok
}

func ListenDevices() {
	for {
		ListenMidiDevices()
		time.Sleep(3 * time.Second)
	}
}
