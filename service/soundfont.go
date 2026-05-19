package service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/sinshu/go-meltysynth/meltysynth"
)

type Synthesizer struct {
	Synth      *meltysynth.Synthesizer
	SampleRate int32
	BufferSize int32
	Streamer   beep.Streamer
	Path       string
}

type UserSoundFont struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"createdAt"`
	LastUsedAt time.Time `json:"lastUsedAt"`
	Missing    bool      `json:"missing"`
	Error      string    `json:"error"`
}

var (
	PianoPlayer    *Synthesizer
	synthMu        sync.Mutex
	speakerStarted bool
)

func ValidateSoundFont(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("音源路径为空")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %w", err)
	}
	absPath = filepath.Clean(absPath)

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("音源文件不存在")
	}

	if info.IsDir() {
		return fmt.Errorf("请选择 .sf2 文件，而不是文件夹")
	}

	if strings.ToLower(filepath.Ext(absPath)) != ".sf2" {
		return fmt.Errorf("请选择 .sf2 音源文件")
	}

	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("打开音源失败: %w", err)
	}
	defer file.Close()

	if _, err := meltysynth.NewSoundFont(file); err != nil {
		return fmt.Errorf("解析音源失败: %w", err)
	}

	return nil
}

func EnsureSpeakerStarted() error {
	if speakerStarted {
		return nil
	}
	return InitSpeaker()
}

func SwitchSoundFont(sf UserSoundFont) error {
	if strings.TrimSpace(sf.Path) == "" {
		return fmt.Errorf("音源路径为空")
	}

	config := normalizeConfigRanges(GetUserConfig())

	AllSynthNotesOff()

	if err := LoadSoundFont(sf, config.SampleRate, config.BufferSize); err != nil {
		return err
	}

	return EnsureSpeakerStarted()
}

func BuildDefaultSoundFont() (UserSoundFont, error) {
	sf, err := BuildSoundFontFromPath(defaultSoundFontPath)
	if err != nil {
		return UserSoundFont{}, fmt.Errorf("默认音源不可用: %w", err)
	}
	return sf, nil
}

func SwitchToDefaultSoundFont() error {
	sf, err := BuildDefaultSoundFont()
	if err != nil {
		return err
	}

	if err := SwitchSoundFont(sf); err != nil {
		return fmt.Errorf("加载默认音源失败: %w", err)
	}

	return nil
}

func FindSoundFontByID(items []UserSoundFont, id string) (int, UserSoundFont, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return -1, UserSoundFont{}, false
	}

	for i, item := range items {
		if item.ID == id {
			return i, item, true
		}
	}

	return -1, UserSoundFont{}, false
}

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

func FileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("计算文件 MD5 失败: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func IsSamePath(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)

	absA, errA := filepath.Abs(a)
	if errA == nil {
		a = absA
	}

	absB, errB := filepath.Abs(b)
	if errB == nil {
		b = absB
	}

	a = filepath.Clean(a)
	b = filepath.Clean(b)

	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}

	return a == b
}

func LoadSoundFont(sf UserSoundFont, sampleRate, bufferSize int32) error {
	path := strings.TrimSpace(sf.Path)
	if path == "" {
		return fmt.Errorf("音源路径为空")
	}

	sf2, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开音源失败: %w", err)
	}
	defer sf2.Close()

	soundfont, err := meltysynth.NewSoundFont(sf2)
	if err != nil {
		return fmt.Errorf("解析音源失败: %w", err)
	}

	settings := meltysynth.NewSynthesizerSettings(sampleRate)

	synthesizer, err := meltysynth.NewSynthesizer(soundfont, settings)
	if err != nil {
		return fmt.Errorf("创建合成器失败: %w", err)
	}

	synthMu.Lock()
	PianoPlayer = &Synthesizer{
		Synth:      synthesizer,
		SampleRate: sampleRate,
		BufferSize: bufferSize,
		Path:       path,
	}
	synthMu.Unlock()

	return nil
}

func InitSoundFont(sf UserSoundFont) error {
	return SwitchSoundFont(sf)
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

func BuildSoundFontFromPath(path string) (UserSoundFont, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return UserSoundFont{}, fmt.Errorf("音源路径为空")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return UserSoundFont{}, fmt.Errorf("获取音源绝对路径失败: %w", err)
	}
	absPath = filepath.Clean(absPath)

	info, err := os.Stat(absPath)
	if err != nil {
		return UserSoundFont{}, fmt.Errorf("音源文件不存在: %w", err)
	}

	if err := ValidateSoundFont(absPath); err != nil {
		return UserSoundFont{}, err
	}

	id, err := FileMD5(absPath)
	if err != nil {
		return UserSoundFont{}, fmt.Errorf("获取音源文件 MD5 失败: %w", err)
	}

	return UserSoundFont{
		ID:        id,
		Name:      strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath)),
		Path:      absPath,
		Size:      info.Size(),
		CreatedAt: time.Now(),
		Missing:   false,
		Error:     "",
	}, nil
}

func AddSoundFontByPath(path string) error {
	sf, err := BuildSoundFontFromPath(path)
	if err != nil {
		return err
	}

	config := GetUserConfig()

	for _, item := range config.SoundFonts {
		if item.ID == sf.ID {
			return fmt.Errorf("音源已存在")
		}

		if IsSamePath(item.Path, sf.Path) {
			return fmt.Errorf("音源已存在")
		}
	}

	config.SoundFonts = append(config.SoundFonts, sf)

	return SaveConfig(config)
}

func (k *Keyboard) RemoveSoundFontByID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("音源 ID 不能为空")
	}

	config := GetUserConfig()

	index, removed, ok := FindSoundFontByID(config.SoundFonts, id)
	if !ok {
		return fmt.Errorf("音源不存在")
	}

	config.SoundFonts = append(config.SoundFonts[:index], config.SoundFonts[index+1:]...)

	if config.ActiveSoundFontID == id {
		config.ActiveSoundFontID = ""

		if err := SwitchToDefaultSoundFont(); err != nil {
			if saveErr := SaveConfig(config); saveErr != nil {
				return saveErr
			}

			return fmt.Errorf("已删除音源 %s，但恢复默认音源失败: %w", removed.Name, err)
		}
	}

	return SaveConfig(config)
}

func (k *Keyboard) SelectSoundFontByID(id string) error {
	id = strings.TrimSpace(id)

	config := GetUserConfig()

	if id == "" {
		if err := SwitchToDefaultSoundFont(); err != nil {
			return err
		}

		config.ActiveSoundFontID = ""

		return SaveConfig(config)
	}

	index, sf, ok := FindSoundFontByID(config.SoundFonts, id)
	if !ok {
		return fmt.Errorf("音源不存在")
	}

	if _, err := os.Stat(sf.Path); err != nil {
		config.SoundFonts[index].Missing = true
		config.SoundFonts[index].Error = "文件不存在"

		if saveErr := SaveConfig(config); saveErr != nil {
			return saveErr
		}

		return fmt.Errorf("音源文件不存在，请删除该记录或重新添加")
	}

	if err := ValidateSoundFont(sf.Path); err != nil {
		config.SoundFonts[index].Missing = false
		config.SoundFonts[index].Error = err.Error()

		if saveErr := SaveConfig(config); saveErr != nil {
			return saveErr
		}

		return err
	}

	if err := SwitchSoundFont(sf); err != nil {
		config.SoundFonts[index].Missing = false
		config.SoundFonts[index].Error = err.Error()

		if saveErr := SaveConfig(config); saveErr != nil {
			return saveErr
		}

		return err
	}

	config.ActiveSoundFontID = id
	config.SoundFonts[index].LastUsedAt = time.Now()
	config.SoundFonts[index].Missing = false
	config.SoundFonts[index].Error = ""

	return SaveConfig(config)
}

func (k *Keyboard) RefreshSoundFonts() error {
	return RefreshSoundFonts()
}

func RefreshSoundFonts() error {
	config := GetUserConfig()

	changed := false

	for i := range config.SoundFonts {
		sf := &config.SoundFonts[i]

		path := strings.TrimSpace(sf.Path)
		if path == "" {
			if !sf.Missing || sf.Error != "路径为空" {
				sf.Missing = true
				sf.Error = "路径为空"
				changed = true
			}
			continue
		}

		if _, err := os.Stat(path); err != nil {
			if !sf.Missing || sf.Error != "文件不存在" {
				sf.Missing = true
				sf.Error = "文件不存在"
				changed = true
			}
			continue
		}

		if sf.Missing || sf.Error == "文件不存在" || sf.Error == "路径为空" {
			sf.Missing = false
			sf.Error = ""
			changed = true
		}
	}

	if changed {
		return SaveConfig(config)
	}

	return nil
}

func InitSoundFontFromConfig() error {
	if err := RefreshSoundFonts(); err != nil {
		return err
	}

	config := GetUserConfig()

	if strings.TrimSpace(config.ActiveSoundFontID) != "" {
		_, sf, ok := FindSoundFontByID(config.SoundFonts, config.ActiveSoundFontID)
		if ok && !sf.Missing {
			if _, err := os.Stat(sf.Path); err == nil {
				if err := SwitchSoundFont(sf); err == nil {
					return nil
				}
			}
		}
	}

	return SwitchToDefaultSoundFont()
}
