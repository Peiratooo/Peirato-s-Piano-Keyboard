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

var Midis = MidiDevices{
	InMidiPool: map[int]InMidiDevice{
		-1: {
			Name:  "无",
			Value: -1,
		},
	},
	OutMidiPool: map[int]OutMidiDevice{
		-1: {
			Name:  "无",
			Value: -1,
		},
	},
	SelectedInDevice:  -1,
	SelectedOutDevice: -1,
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

	lastID := -1
	alive := map[int]bool{-1: true}

	for _, port := range outports {
		deviceID := port.Number()
		alive[deviceID] = true
		lastID = deviceID

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

	if Midis.SelectedOutDevice == -1 && lastID != -1 {
		Midis.SelectedOutDevice = lastID
	}

	for id, device := range Midis.OutMidiPool {
		if alive[id] {
			continue
		}
		if id == Midis.SelectedOutDevice {
			Midis.SelectedOutDevice = -1
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
		Keydown(int32(ch), int32(key), int32(vel))
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
		shouldReleaseNow := true

		midiMu.Lock()
		pedal := Midis.PedalStatus[deviceID]
		if pedal == nil {
			pedal = newPedalSignal(deviceID)
			Midis.PedalStatus[deviceID] = pedal
		}
		delete(pedal.DownKeys, midiKey)
		if pedal.DamperPedal {
			shouldReleaseNow = false
			if !containsUint8(pedal.DamperPedalKeys, midiKey) {
				pedal.DamperPedalKeys = append(pedal.DamperPedalKeys, midiKey)
			}
		}
		midiMu.Unlock()

		emitKeyboardEvent("pressedUp", midiKey, 0, ch)

		if shouldReleaseNow {
			emitKeyboardEvent("up", midiKey, 0, ch)
			Keyup(int32(ch), int32(key))
		}

	case msg.GetControlChange(&ch, &con, &vel):
		handlePedalMessage(deviceID, ch, con, vel)
	}
}

func handlePedalMessage(deviceID int, channel, controller, velocity uint8) {
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
		pedal.DamperPedal = velocity > 0
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

	for _, key := range releaseKeys {
		emitKeyboardEvent("up", key, 0, channel)
		Keyup(int32(channel), int32(key))
	}

	if App != nil {
		App.Event.Emit("pedal", pedalSnapshot)
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
	Keydown(int32(config.MidiChannel), int32(key), config.Volume)
	MidiPlayer.HandleUserNoteOn(int(key))

	midiMu.RLock()
	selectedOut := Midis.SelectedOutDevice
	outDevice, ok := Midis.OutMidiPool[selectedOut]
	midiMu.RUnlock()
	if !ok || selectedOut == -1 || outDevice.Device == nil {
		return
	}
	noteOn := midi.NoteOn(config.MidiChannel, key, config.Velocity)
	if err := outDevice.Device.Send(noteOn); err != nil {
		fmt.Println("发送 MIDI NoteOn 失败:", err)
	}
}

func (k *Keyboard) KeyboardStop(key uint8) {
	config := GetUserConfig()
	Keyup(int32(config.MidiChannel), int32(key))

	midiMu.RLock()
	selectedOut := Midis.SelectedOutDevice
	outDevice, ok := Midis.OutMidiPool[selectedOut]
	midiMu.RUnlock()
	if !ok || selectedOut == -1 || outDevice.Device == nil {
		return
	}

	noteOff := midi.NoteOff(config.MidiChannel, key)
	if err := outDevice.Device.Send(noteOff); err != nil {
		fmt.Println("发送 MIDI NoteOff 失败:", err)
	}
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

	config := GetUserConfig()
	midiMu.RLock()
	selectedOut := Midis.SelectedOutDevice
	outDevice, ok := Midis.OutMidiPool[selectedOut]
	midiMu.RUnlock()

	if ok && selectedOut != -1 && outDevice.Device != nil {
		for channel := uint8(0); channel < 16; channel++ {
			// CC 123 = All Notes Off，CC 120 = All Sound Off。
			_ = outDevice.Device.Send(midi.ControlChange(channel, 123, 0))
			_ = outDevice.Device.Send(midi.ControlChange(channel, 120, 0))
		}
		for key := uint8(0); key < 128; key++ {
			_ = outDevice.Device.Send(midi.NoteOff(config.MidiChannel, key))
		}
	}

	if App != nil {
		App.Event.Emit("allNotesOff")
	}
	emitMidiVisualClear()
}

func ListenDevices() {
	for {
		ListenMidiDevices()
		time.Sleep(3 * time.Second)
	}
}
