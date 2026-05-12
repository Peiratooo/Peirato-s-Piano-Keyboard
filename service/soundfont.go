package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/sinshu/go-meltysynth/meltysynth"
)

type Sythesizer struct {
	Synth      *meltysynth.Synthesizer
	SampleRate int32
	BufferSize int32
	Streamer   beep.Streamer
	Path       string
}

type SoundFontInfo struct {
	Loaded bool   `json:"loaded"`
	Path   string `json:"path"`
	Name   string `json:"name"`
	Error  string `json:"error"`
}

var (
	PianoPlayer    *Sythesizer
	synthMu        sync.Mutex
	speakerStarted bool
	soundFontInfo  = SoundFontInfo{}
)

func InitSpeaker() error {
	if PianoPlayer == nil || PianoPlayer.Synth == nil {
		return fmt.Errorf("音源尚未加载，无法初始化扬声器")
	}

	if !speakerStarted {
		if err := speaker.Init(beep.SampleRate(PianoPlayer.SampleRate), int(PianoPlayer.BufferSize)); err != nil {
			return fmt.Errorf("初始化扬声器失败: %w", err)
		}
		speakerStarted = true
	}

	PianoPlayer.Streamer = beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		left := make([]float32, len(samples))
		right := make([]float32, len(samples))

		// Render 与 NoteOn / NoteOff 会在不同 goroutine 中调用，统一加锁可以避免偶发并发问题。
		synthMu.Lock()
		if PianoPlayer != nil && PianoPlayer.Synth != nil {
			PianoPlayer.Synth.Render(left, right)
		}
		synthMu.Unlock()

		for i := range samples {
			samples[i][0] = float64(left[i])
			samples[i][1] = float64(right[i])
		}
		return len(samples), true
	})

	volumeStreamer := effects.Volume{
		Streamer: PianoPlayer.Streamer,
		Base:     7,
		Volume:   1,
		Silent:   false,
	}
	speaker.Play(&volumeStreamer)
	return nil
}

func LoadSoundFont(path string, sampleRate, bufferSize int32) error {
	sf2, err := os.Open(path)
	if err != nil {
		soundFontInfo = SoundFontInfo{Loaded: false, Path: path, Name: filepath.Base(path), Error: err.Error()}
		return fmt.Errorf("打开音源失败: %w", err)
	}
	defer sf2.Close()

	soundfont, err := meltysynth.NewSoundFont(sf2)
	if err != nil {
		soundFontInfo = SoundFontInfo{Loaded: false, Path: path, Name: filepath.Base(path), Error: err.Error()}
		return fmt.Errorf("解析音源失败: %w", err)
	}

	settings := meltysynth.NewSynthesizerSettings(sampleRate)
	synthesizer, err := meltysynth.NewSynthesizer(soundfont, settings)
	if err != nil {
		soundFontInfo = SoundFontInfo{Loaded: false, Path: path, Name: filepath.Base(path), Error: err.Error()}
		return fmt.Errorf("创建合成器失败: %w", err)
	}

	synthMu.Lock()
	PianoPlayer = &Sythesizer{
		Synth:      synthesizer,
		SampleRate: sampleRate,
		BufferSize: bufferSize,
		Path:       path,
	}
	synthMu.Unlock()

	soundFontInfo = SoundFontInfo{Loaded: true, Path: path, Name: filepath.Base(path)}
	return nil
}

func InitSoundFont(path string) error {
	if err := LoadSoundFont(path, 44100, 1024); err != nil {
		return err
	}
	return InitSpeaker()
}

func Keydown(channel, key, velocity int32) {
	synthMu.Lock()
	defer synthMu.Unlock()
	if PianoPlayer == nil || PianoPlayer.Synth == nil {
		return
	}
	PianoPlayer.Synth.NoteOn(channel, key, velocity)
}

func Keyup(channel, key int32) {
	synthMu.Lock()
	defer synthMu.Unlock()
	if PianoPlayer == nil || PianoPlayer.Synth == nil {
		return
	}
	PianoPlayer.Synth.NoteOff(channel, key)
}

func AllSynthNotesOff() {
	synthMu.Lock()
	defer synthMu.Unlock()
	if PianoPlayer == nil || PianoPlayer.Synth == nil {
		return
	}
	for channel := int32(0); channel < 16; channel++ {
		for key := int32(0); key < 128; key++ {
			PianoPlayer.Synth.NoteOff(channel, key)
		}
	}
}

func (k *Keyboard) GetSoundFontInfo() SoundFontInfo {
	return soundFontInfo
}

// ImportSoundFontBase64 供前端 <input type="file"> 选择 .sf2 后调用。
// 设计上不使用后端原生文件选择器：用户仍然在前端选择文件，后端只负责保存、加载和记录配置。
func (k *Keyboard) ImportSoundFontBase64(fileName string, encoded string) (SoundFontInfo, error) {
	if strings.ToLower(filepath.Ext(fileName)) != ".sf2" {
		return GetSoundFontInfoSnapshot(), fmt.Errorf("请选择 .sf2 音源文件")
	}

	data, err := decodeBase64Payload(encoded)
	if err != nil {
		return GetSoundFontInfoSnapshot(), err
	}
	if len(data) == 0 {
		return GetSoundFontInfoSnapshot(), fmt.Errorf("音源文件为空")
	}

	soundFontDir := filepath.Join(filepath.Dir(configFilePath), "soundfonts")
	if err := os.MkdirAll(soundFontDir, 0755); err != nil {
		return GetSoundFontInfoSnapshot(), fmt.Errorf("创建音源目录失败: %w", err)
	}

	// 文件名加时间戳，避免用户多次导入同名音源时互相覆盖。
	safeName := sanitizeSoundFontFileName(fileName)
	storedPath := filepath.Join(soundFontDir, fmt.Sprintf("%d_%s", time.Now().Unix(), safeName))
	if err := os.WriteFile(storedPath, data, 0644); err != nil {
		return GetSoundFontInfoSnapshot(), fmt.Errorf("保存音源文件失败: %w", err)
	}

	if err := LoadSoundFont(storedPath, 44100, 1024); err != nil {
		return GetSoundFontInfoSnapshot(), err
	}
	if !speakerStarted {
		if err := InitSpeaker(); err != nil {
			return GetSoundFontInfoSnapshot(), err
		}
	}

	config := GetUserConfig()
	config.SoundFontPath = storedPath
	if err := SaveConfig(config); err != nil {
		return GetSoundFontInfoSnapshot(), err
	}
	EmitConfigChanged()
	EmitSoundFontChanged()
	return GetSoundFontInfoSnapshot(), nil
}

func sanitizeSoundFontFileName(fileName string) string {
	name := filepath.Base(strings.TrimSpace(fileName))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "user-soundfont.sf2"
	}
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}

// SelectSoundFont 供设置中心通过 Wails 绑定调用。
// 只有音源成功加载后才写入配置，避免用户误选坏文件导致下次启动持续失败。
func (k *Keyboard) SelectSoundFont(path string) (SoundFontInfo, error) {
	if err := LoadSoundFont(path, 44100, 1024); err != nil {
		return GetSoundFontInfoSnapshot(), err
	}
	if !speakerStarted {
		if err := InitSpeaker(); err != nil {
			return GetSoundFontInfoSnapshot(), err
		}
	}

	config := GetUserConfig()
	config.SoundFontPath = path
	if err := SaveConfig(config); err != nil {
		return GetSoundFontInfoSnapshot(), err
	}
	EmitConfigChanged()
	EmitSoundFontChanged()
	return GetSoundFontInfoSnapshot(), nil
}

// ReloadSoundFont 按当前配置重新加载音源。后续用户替换同路径文件时可以直接调用它。
func (k *Keyboard) ReloadSoundFont() (SoundFontInfo, error) {
	path := ResolveSoundFontPath()
	if err := LoadSoundFont(path, 44100, 1024); err != nil {
		return GetSoundFontInfoSnapshot(), err
	}
	if !speakerStarted {
		if err := InitSpeaker(); err != nil {
			return GetSoundFontInfoSnapshot(), err
		}
	}
	EmitSoundFontChanged()
	return GetSoundFontInfoSnapshot(), nil
}

// RestoreDefaultSoundFont 清空用户音源路径，并回退到默认音源。
func (k *Keyboard) RestoreDefaultSoundFont() (SoundFontInfo, error) {
	config := GetUserConfig()
	config.SoundFontPath = ""
	if err := SaveConfig(config); err != nil {
		return GetSoundFontInfoSnapshot(), err
	}
	if err := LoadSoundFont(defaultSoundFontPath, 44100, 1024); err != nil {
		return GetSoundFontInfoSnapshot(), err
	}
	if !speakerStarted {
		if err := InitSpeaker(); err != nil {
			return GetSoundFontInfoSnapshot(), err
		}
	}
	EmitConfigChanged()
	EmitSoundFontChanged()
	return GetSoundFontInfoSnapshot(), nil
}

func GetSoundFontInfoSnapshot() SoundFontInfo {
	return soundFontInfo
}

func EmitSoundFontChanged() {
	if App != nil {
		App.Event.Emit("soundFontChanged", GetSoundFontInfoSnapshot())
	}
}
