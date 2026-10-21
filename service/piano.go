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
	DeviceID        int                  `json:"deviceID"`
	DamperPedal     bool                 `json:"damperPedal"`    // 延音踏板 64
	SostenutoPedal  bool                 `json:"sostenutoPedal"` // 持音踏板 66
	SoftPedal       bool                 `json:"softPedal"`      // 柔音踏板 67
	DamperPedalKeys map[channelNote]bool `json:"-"`
	DownKeys        map[channelNote]bool `json:"-"`
	Channels        map[uint8]bool       `json:"-"`
}

type channelNote struct{ Channel, Note uint8 }

// Called with midiMu held. Frontend highlights aggregate every input/channel.
func noteVisualState(note uint8) (pressed, active bool) {
	for _, pedal := range Midis.PedalStatus {
		for key := range pedal.DownKeys {
			if key.Note == note {
				pressed = true
				active = true
			}
		}
		for key := range pedal.DamperPedalKeys {
			if key.Note == note {
				active = true
			}
		}
	}
	return
}

type MidiDevices struct {
	InMidiPool        map[int]InMidiDevice  `json:"inMidiPool"`
	OutMidiPool       map[int]OutMidiDevice `json:"outMidiPool"`
	SelectedInDevice  int                   `json:"selectedInDevice"`
	SelectedOutDevice int                   `json:"selectedOutDevice"`
	PedalStatus       map[int]*PedalSingal  `json:"pedalStatus"`
	SoftwareSustain   bool                  `json:"softwareSustain"`
	Listener          Listener              `json:"-"`
	Initialized       bool                  `json:"initialized"`
}

var (
	midiMu           sync.RWMutex
	midiListenerStop func()
	inputEventsMu    sync.Mutex
	deviceChangeMu   sync.Mutex
	// Listener requests and port lifecycle changes are serialized by deviceChangeMu.
	midiListeningEnabled    bool
	midiDevicesClosed       bool
	inputSelectionDisabled  bool
	softwareSustain         bool
	softwareSustainChannels = make(map[uint8]bool)
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
		DamperPedalKeys: make(map[channelNote]bool),
		DownKeys:        make(map[channelNote]bool),
		Channels:        make(map[uint8]bool),
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
		SoftwareSustain:   softwareSustain,
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
	deviceChangeMu.Lock()
	defer deviceChangeMu.Unlock()
	midiDevicesClosed = true
	midiListeningEnabled = false
	(&Keyboard{}).midiListenerStop()
	(&Keyboard{}).AllNotesOff()

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
	deviceChangeMu.Lock()
	defer deviceChangeMu.Unlock()
	if midiDevicesClosed {
		return
	}
	compareInDevices(inports)
	(&Keyboard{}).midiListenerStart()
}

func compareInDevices(inports midi.InPorts) {
	lastID := -1
	available := make(map[int]drivers.In, len(inports))
	for _, port := range inports {
		available[port.Number()] = port
		lastID = port.Number()
	}

	midiMu.RLock()
	selectedID := Midis.SelectedInDevice
	selectedName := Midis.InMidiPool[selectedID].Name
	var removed []InMidiDevice
	for id, device := range Midis.InMidiPool {
		if id == -1 {
			continue
		}
		port, exists := available[id]
		// Port numbers can be reused or shift when another device is unplugged.
		if !exists || port.String() != device.Name {
			removed = append(removed, device)
		}
	}
	midiMu.RUnlock()
	selectedRemoved := selectedID != -1 && (available[selectedID] == nil || available[selectedID].String() != selectedName)
	if selectedRemoved {
		(&Keyboard{}).midiListenerStop()
		(&Keyboard{}).AllNotesOff()
	}
	// Cancel callbacks before closing their port, without holding midiMu.
	for _, device := range removed {
		if device.Device != nil {
			_ = device.Device.Close()
		}
	}
	midiMu.Lock()
	for _, device := range removed {
		delete(Midis.PedalStatus, device.Value)
		delete(Midis.InMidiPool, device.Value)
	}
	for _, port := range inports {
		id := port.Number()
		if _, exists := Midis.InMidiPool[id]; !exists {
			Midis.InMidiPool[id] = InMidiDevice{Device: port, Name: port.String(), Value: id}
			Midis.PedalStatus[id] = newPedalSignal(id)
		}
	}
	if selectedRemoved {
		Midis.SelectedInDevice = -1
		for _, port := range inports {
			if port.String() == selectedName {
				Midis.SelectedInDevice = port.Number()
				break
			}
		}
	}
	if Midis.SelectedInDevice == -1 && !inputSelectionDisabled {
		Midis.SelectedInDevice = lastID
	}
	midiMu.Unlock()
}

func CompareOutDevices(outports midi.OutPorts) {
	deviceChangeMu.Lock()
	defer deviceChangeMu.Unlock()
	if midiDevicesClosed {
		return
	}
	compareOutDevices(outports)
	(&Keyboard{}).midiListenerStart()
}

func compareOutDevices(outports midi.OutPorts) {
	available := make(map[int]drivers.Out, len(outports))
	for _, port := range outports {
		available[port.Number()] = port
	}
	midiMu.RLock()
	selectedID := Midis.SelectedOutDevice
	selectedName := Midis.OutMidiPool[selectedID].Name
	var removed []OutMidiDevice
	for id, device := range Midis.OutMidiPool {
		if id < 0 {
			continue
		}
		port, exists := available[id]
		if !exists || port.String() != device.Name {
			removed = append(removed, device)
		}
	}
	midiMu.RUnlock()
	selectedRemoved := selectedID >= 0 && (available[selectedID] == nil || available[selectedID].String() != selectedName)
	if selectedRemoved {
		(&Keyboard{}).midiListenerStop()
		(&Keyboard{}).AllNotesOff()
	}
	for _, device := range removed {
		if device.Device != nil {
			_ = device.Device.Close()
		}
	}
	midiMu.Lock()
	for _, device := range removed {
		delete(Midis.OutMidiPool, device.Value)
	}
	if selectedRemoved {
		Midis.SelectedOutDevice = midiOutputSoftwareSynth
	}
	midiMu.Unlock()
	for _, port := range outports {
		id := port.Number()
		midiMu.RLock()
		_, exists := Midis.OutMidiPool[id]
		midiMu.RUnlock()
		if exists {
			continue
		}
		if err := port.Open(); err != nil {
			fmt.Println("打开 MIDI 输出设备失败:", err)
			continue
		}
		midiMu.Lock()
		Midis.OutMidiPool[id] = OutMidiDevice{Device: port, Name: port.String(), Value: id}
		if selectedRemoved && port.String() == selectedName {
			Midis.SelectedOutDevice = id
		}
		midiMu.Unlock()
	}
}

func ListenMidiDevices() {
	deviceChangeMu.Lock()
	defer deviceChangeMu.Unlock()
	if midiDevicesClosed {
		return
	}
	compareInDevices(midi.GetInPorts())
	compareOutDevices(midi.GetOutPorts())
	(&Keyboard{}).midiListenerStart()

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
	deviceChangeMu.Lock()
	defer deviceChangeMu.Unlock()
	midiListeningEnabled = true
	k.midiListenerStart()
}

func (k *Keyboard) midiListenerStart() {
	if !midiListeningEnabled || midiDevicesClosed {
		return
	}
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
	inputEventsMu.Lock()
	defer inputEventsMu.Unlock()
	var bt []byte
	var ch, key, vel, con uint8

	switch {
	case msg.GetSysEx(&bt):
		return

	case msg.GetNoteStart(&ch, &key, &vel):
		midiKey := midi.Note(key).Value()
		ensureSoftwareSustainChannel(ch)
		playSelectedOutputNoteOn(ch, key, vel)
		MidiPlayer.HandleUserNoteOn(int(midiKey))

		midiMu.Lock()
		pedal := Midis.PedalStatus[deviceID]
		if pedal == nil {
			pedal = newPedalSignal(deviceID)
			Midis.PedalStatus[deviceID] = pedal
		}
		pedal.DownKeys[channelNote{ch, midiKey}] = true
		delete(pedal.DamperPedalKeys, channelNote{ch, midiKey})
		midiMu.Unlock()

		emitKeyboardEvent("down", midiKey, vel, ch)
		emitKeyboardEvent("pressedDown", midiKey, vel, ch)

	case msg.GetNoteEnd(&ch, &key):
		midiKey := midi.Note(key).Value()

		midiMu.Lock()
		pedal := Midis.PedalStatus[deviceID]
		if pedal == nil {
			pedal = newPedalSignal(deviceID)
			Midis.PedalStatus[deviceID] = pedal
		}
		delete(pedal.DownKeys, channelNote{ch, midiKey})
		damper := pedal.Channels[ch]
		if deviceID == interactiveDeviceID {
			if input := Midis.PedalStatus[Midis.SelectedInDevice]; input != nil {
				damper = damper || input.Channels[ch]
			}
		}
		if damper || softwareSustain {
			pedal.DamperPedalKeys[channelNote{ch, midiKey}] = true
		}
		pressed, active := noteVisualState(midiKey)
		outputHeld := false
		for _, source := range Midis.PedalStatus {
			outputHeld = outputHeld || source.DownKeys[channelNote{ch, midiKey}]
		}
		midiMu.Unlock()

		if !pressed {
			emitKeyboardEvent("pressedUp", midiKey, 0, ch)
		}
		if !outputHeld {
			playSelectedOutputNoteOff(ch, key)
		}
		if !active {
			emitKeyboardEvent("up", midiKey, 0, ch)
		}

	case msg.GetControlChange(&ch, &con, &vel):
		handlePedalMessage(deviceID, ch, con, vel)

	case msg.GetProgramChange(&ch, &con):
		playSelectedOutputProgramChange(ch, con)
	}
}

func handlePedalMessage(deviceID int, channel, controller, velocity uint8) {
	if controller != 64 && controller != 66 && controller != 67 {
		playSelectedOutputControlChange(channel, controller, velocity)
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
		pedal.Channels[channel] = velocity >= 64
		if softwareSustain {
			softwareSustainChannels[channel] = true
			velocity = 127
		}
		pedal.DamperPedal = false
		for _, down := range pedal.Channels {
			pedal.DamperPedal = pedal.DamperPedal || down
		}
		if !pedal.Channels[channel] && !softwareSustain {
			for sustainedKey := range pedal.DamperPedalKeys {
				if sustainedKey.Channel != channel {
					continue
				}
				delete(pedal.DamperPedalKeys, sustainedKey)
				if _, active := noteVisualState(sustainedKey.Note); !active {
					releaseKeys = append(releaseKeys, sustainedKey.Note)
				}
			}
			if deviceID == Midis.SelectedInDevice {
				if interactive := Midis.PedalStatus[interactiveDeviceID]; interactive != nil {
					for note := range interactive.DamperPedalKeys {
						if note.Channel != channel {
							continue
						}
						delete(interactive.DamperPedalKeys, note)
						if _, active := noteVisualState(note.Note); !active {
							releaseKeys = append(releaseKeys, note.Note)
						}
					}
				}
			}
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
	playSelectedOutputControlChange(channel, controller, velocity)

	if App != nil {
		App.Event.Emit("pedal", pedalSnapshot)
	}
	for _, key := range releaseKeys {
		emitKeyboardEvent("up", key, 0, channel)
	}
}

// SetSoftwareSustain controls the live keyboard pedal. The sound font's own
// envelope supplies the decay; released notes remain held until both pedals lift.
func (k *Keyboard) SetSoftwareSustain(enabled bool) bool {
	inputEventsMu.Lock()
	defer inputEventsMu.Unlock()

	config := GetUserConfig()
	channels := make(map[uint8]bool)
	releaseCandidates := make(map[uint8]bool)
	midiMu.Lock()
	if softwareSustain == enabled {
		midiMu.Unlock()
		return enabled
	}
	softwareSustain = enabled
	if enabled {
		channels[config.MidiChannel] = true
		for id, pedal := range Midis.PedalStatus {
			if id != Midis.SelectedInDevice && id != interactiveDeviceID {
				continue
			}
			for note := range pedal.DownKeys {
				channels[note.Channel] = true
			}
			for note := range pedal.DamperPedalKeys {
				channels[note.Channel] = true
			}
		}
		for channel := range channels {
			softwareSustainChannels[channel] = true
		}
	} else {
		for channel := range softwareSustainChannels {
			channels[channel] = true
		}
		softwareSustainChannels = make(map[uint8]bool)
		for id, pedal := range Midis.PedalStatus {
			for note := range pedal.DamperPedalKeys {
				physicalDown := pedal.Channels[note.Channel]
				if id == interactiveDeviceID {
					if input := Midis.PedalStatus[Midis.SelectedInDevice]; input != nil {
						physicalDown = physicalDown || input.Channels[note.Channel]
					}
				}
				if !physicalDown {
					delete(pedal.DamperPedalKeys, note)
					releaseCandidates[note.Note] = true
				}
			}
		}
		for note := range releaseCandidates {
			if _, active := noteVisualState(note); active {
				delete(releaseCandidates, note)
			}
		}
	}
	physicalChannels := make(map[uint8]bool, len(channels))
	if input := Midis.PedalStatus[Midis.SelectedInDevice]; input != nil {
		for channel := range channels {
			physicalChannels[channel] = input.Channels[channel]
		}
	}
	midiMu.Unlock()

	for channel := range channels {
		value := uint8(0)
		if enabled || physicalChannels[channel] {
			value = 127
		}
		playSelectedOutputControlChange(channel, midiCCSustain, value)
	}
	if App != nil {
		App.Event.Emit("softwareSustain", enabled)
	}
	for note := range releaseCandidates {
		emitKeyboardEvent("up", note, 0, 0)
	}
	return enabled
}

func ensureSoftwareSustainChannel(channel uint8) {
	midiMu.Lock()
	needSustain := softwareSustain && !softwareSustainChannels[channel]
	if needSustain {
		softwareSustainChannels[channel] = true
	}
	midiMu.Unlock()
	if needSustain {
		playSelectedOutputControlChange(channel, midiCCSustain, 127)
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

var interactiveMu sync.Mutex
var interactiveNotes = make(map[uint8]struct {
	Channel uint8
	Count   int
})

const interactiveDeviceID = -3

func (k *Keyboard) KeyboardPlay(key uint8) {
	interactiveMu.Lock()
	defer interactiveMu.Unlock()
	state := interactiveNotes[key]
	state.Count++
	if state.Count > 1 {
		interactiveNotes[key] = state
		return
	}
	configMu.RLock()
	config := UserConfig
	configMu.RUnlock()
	state.Channel = config.MidiChannel
	interactiveNotes[key] = state
	if selectedOut, _, _ := currentOutputDevice(); selectedOut == midiOutputSoftwareSynth {
		// A MIDI file can leave CC7/CC11 at zero on the configured channel.
		// Restore the interactive piano channel before a computer/mouse key press.
		ProcessSynthMidiMessage(int32(config.MidiChannel), midiCommandControlChange, midiCCChannelVolume, 127)
		ProcessSynthMidiMessage(int32(config.MidiChannel), midiCommandControlChange, midiCCPan, 64)
		ProcessSynthMidiMessage(int32(config.MidiChannel), midiCommandControlChange, midiCCExpression, 127)
	}
	handleMidiMessage(interactiveDeviceID, midi.NoteOn(config.MidiChannel, key, config.Velocity))
}

func (k *Keyboard) KeyboardStop(key uint8) {
	interactiveMu.Lock()
	defer interactiveMu.Unlock()
	state, ok := interactiveNotes[key]
	if !ok {
		return
	}
	state.Count--
	if state.Count > 0 {
		interactiveNotes[key] = state
		return
	}
	delete(interactiveNotes, key)
	handleMidiMessage(interactiveDeviceID, midi.NoteOff(state.Channel, key))
}

func (k *Keyboard) MidiListenerStop() {
	deviceChangeMu.Lock()
	defer deviceChangeMu.Unlock()
	midiListeningEnabled = false
	k.midiListenerStop()
}

func (k *Keyboard) midiListenerStop() {
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
	deviceChangeMu.Lock()
	defer deviceChangeMu.Unlock()
	if midiDevicesClosed {
		return false
	}
	midiMu.RLock()
	_, validIn := Midis.InMidiPool[deviceID]
	_, validOut := Midis.OutMidiPool[deviceID]
	midiMu.RUnlock()
	if (deviceType != "in" && deviceType != "out") || (deviceType == "in" && !validIn) || (deviceType == "out" && !validOut) {
		return false
	}
	k.midiListenerStop()
	// Block interactive and input events through clearing and routing changes.
	interactiveMu.Lock()
	inputEventsMu.Lock()
	k.allNotesOffLocked()
	midiMu.Lock()
	if deviceType == "in" {
		Midis.SelectedInDevice = deviceID
		inputSelectionDisabled = deviceID == -1
	} else {
		Midis.SelectedOutDevice = deviceID
	}
	midiMu.Unlock()
	inputEventsMu.Unlock()
	interactiveMu.Unlock()
	k.midiListenerStart()
	return true
}

func (k *Keyboard) AllNotesOff() {
	interactiveMu.Lock()
	defer interactiveMu.Unlock()
	inputEventsMu.Lock()
	defer inputEventsMu.Unlock()
	k.allNotesOffLocked()
}

func (k *Keyboard) allNotesOffLocked() {
	interactiveNotes = make(map[uint8]struct {
		Channel uint8
		Count   int
	})
	midiMu.Lock()
	softwareSustain = false
	softwareSustainChannels = make(map[uint8]bool)
	for id := range Midis.PedalStatus {
		Midis.PedalStatus[id] = newPedalSignal(id)
	}
	midiMu.Unlock()
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
		App.Event.Emit("softwareSustain", false)
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
	if controller == midiCCSustain {
		midiMu.Lock()
		if softwareSustain {
			// Playback and hardware CC64 must not lift the software pedal.
			softwareSustainChannels[channel] = true
			value = 127
		}
		midiMu.Unlock()
	}
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
