package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const appName = "Peirato's Piano Keyboard"

var (
	configFilePath = "config.json"
	configMu       sync.RWMutex
)

type Color struct {
	Label string `json:"label"`
	Color string `json:"color"`
}

type Window struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	X      int `json:"x"`
	Y      int `json:"y"`
}

type Config struct {
	Colors            map[string]Color `json:"colors"`
	KeyLabel          string           `json:"keyLabel"`
	KeyboardType      int              `json:"keyboardType"`
	Velocity          uint8            `json:"velocity"`
	Opacity           int              `json:"opacity"`
	Version           string           `json:"version"`
	ShowPedal         bool             `json:"showPedal"`
	Volume            int32            `json:"volume"`
	SampleRate        int32            `json:"sampleRate"`
	BufferSize        int32            `json:"bufferSize"`
	MidiChannel       uint8            `json:"midiChannel"`
	ActiveSoundFontID string           `json:"activeSoundFontId"`
	SoundFonts        []UserSoundFont  `json:"soundFonts"`
	MidiStore         []UserMidi       `json:"midiStore"`
}

var DefaultConfig = Config{
	Colors: map[string]Color{
		"whiteKey": {
			Label: "白键按下",
			Color: "#9AF7B3",
		},
		"blackKey": {
			Label: "黑键按下",
			Color: "#5FFF5F",
		},
		"whiteKeyLeft": {
			Label: "白键按下(左)",
			Color: "#f7e89a",
		},
		"blackKeyLeft": {
			Label: "黑键按下(左)",
			Color: "#ffd25f",
		},
		"damperPedal": {
			Label: "延音踏板踩下",
			Color: "#e7b510",
		},
		"softPedal": {
			Label: "柔音踏板踩下",
			Color: "#10e786",
		},
		"sostenutoPedal": {
			Label: "消音踏板踩下",
			Color: "#1054e7",
		},
	},
	KeyLabel:          "octave_key",
	KeyboardType:      0,
	Velocity:          80,
	Volume:            80,
	SampleRate:        44100,
	BufferSize:        2048,
	Opacity:           100,
	ShowPedal:         true,
	MidiChannel:       0,
	SoundFonts:        []UserSoundFont{},
	ActiveSoundFontID: "",
}

var UserConfig = cloneDefaultConfig()

func cloneDefaultConfig() Config {
	config := DefaultConfig
	config.Colors = make(map[string]Color, len(DefaultConfig.Colors))
	for key, value := range DefaultConfig.Colors {
		config.Colors[key] = value
	}
	return config
}

// LoadConfig 负责读取用户配置，并自动补齐旧配置里缺失的新字段。
// 后续新增配置字段时，优先在 mergeConfigWithDefaults 中补默认值，避免旧用户升级后出现空字段。
func LoadConfig(version string) error {
	ucd, err := os.UserConfigDir()
	if err != nil {
		ucd = "./assets"
	}

	configFilePath = filepath.Join(ucd, appName, "config.json")
	config := cloneDefaultConfig()

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		config.Version = version
		if saveErr := SaveConfig(config); saveErr != nil {
			return fmt.Errorf("保存默认配置失败: %w", saveErr)
		}
		return nil
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		config.Version = version
		UserConfig = config
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		config = cloneDefaultConfig()
		config.Version = version
		UserConfig = config
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	config = mergeConfigWithDefaults(config)
	config.Version = version

	config = normalizeConfigRanges(config)

	return SaveConfig(config)
}

// mergeConfigWithDefaults 用于兼容旧版本 config.json。
// 例如用户旧配置没有 midiChannel / soundFontPath 时，这里会补上安全默认值。
func mergeConfigWithDefaults(config Config) Config {
	merged := cloneDefaultConfig()

	merged.KeyLabel = config.KeyLabel
	merged.KeyboardType = config.KeyboardType
	merged.Velocity = config.Velocity
	merged.Opacity = config.Opacity
	merged.Version = config.Version
	merged.ShowPedal = config.ShowPedal
	merged.Volume = config.Volume
	merged.SampleRate = config.SampleRate
	merged.BufferSize = config.BufferSize
	merged.MidiChannel = config.MidiChannel
	merged.SoundFonts = config.SoundFonts
	merged.ActiveSoundFontID = config.ActiveSoundFontID
	merged.MidiStore = config.MidiStore

	for key, value := range config.Colors {
		merged.Colors[key] = value
	}

	return normalizeConfigRanges(merged)
}

func normalizeConfigRanges(config Config) Config {
	if config.Opacity < 20 || config.Opacity > 100 {
		config.Opacity = DefaultConfig.Opacity
	}
	if config.Volume < 0 || config.Volume > 100 {
		config.Volume = DefaultConfig.Volume
	}
	if config.SampleRate != 22050 && config.SampleRate != 44100 && config.SampleRate != 48000 {
		config.SampleRate = DefaultConfig.SampleRate
	}
	if config.BufferSize != 512 && config.BufferSize != 1024 && config.BufferSize != 2048 && config.BufferSize != 4096 {
		config.BufferSize = DefaultConfig.BufferSize
	}
	if config.Velocity == 0 || config.Velocity > 127 {
		config.Velocity = DefaultConfig.Velocity
	}
	if config.MidiChannel > 15 {
		config.MidiChannel = DefaultConfig.MidiChannel
	}
	return config
}

func SaveConfig(config Config) error {
	configMu.Lock()
	nextConfig := mergeConfigWithDefaults(config)

	data, err := json.MarshalIndent(nextConfig, "", "  ")
	if err != nil {
		configMu.Unlock()
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	dir := filepath.Dir(configFilePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			configMu.Unlock()
			return fmt.Errorf("创建配置目录失败: %w", err)
		}
	}

	if err := os.WriteFile(configFilePath, data, 0644); err != nil {
		configMu.Unlock()
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	UserConfig = nextConfig

	configMu.Unlock()
	EmitConfigChanged()

	return nil
}

func GetUserConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return UserConfig
}

func (k *Keyboard) SendConfig() Config {
	return GetUserConfig()
}

func (k *Keyboard) GetDefaultConfig() Config {
	return cloneDefaultConfig()
}

func (k *Keyboard) ReceiveConfig(config Config) (bool, string) {
	if err := SaveConfig(config); err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (k *Keyboard) ResetConfig() Config {
	resetConfig := cloneDefaultConfig()
	resetConfig.Version = GetUserConfig().Version
	_ = SaveConfig(resetConfig)
	return resetConfig
}

func EmitConfigChanged() {
	if App != nil {
		App.Event.Emit("configChanged", GetUserConfig())
	}
}
