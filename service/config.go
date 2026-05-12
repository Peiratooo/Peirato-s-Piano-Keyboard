package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	appName              = "Peirato's Piano"
	defaultSoundFontPath = "./assets/Yamaha-Grand-Lite-v2.0.sf2"
)

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
	Colors        map[string]Color `json:"colors"`
	KeyLabel      string           `json:"keyLabel"`
	KeyboardType  int              `json:"keyboardType"`
	Velocity      uint8            `json:"velocity"`
	Opacity       int              `json:"opacity"`
	Version       string           `json:"version"`
	ShowPedal     bool             `json:"showPedal"`
	Volume        int32            `json:"volume"`
	MidiChannel   uint8            `json:"midiChannel"`
	SoundFontPath string           `json:"soundFontPath"`
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
	KeyLabel:      "octave_key",
	KeyboardType:  0,
	Velocity:      80,
	Volume:        80,
	Opacity:       100,
	ShowPedal:     true,
	MidiChannel:   0,
	SoundFontPath: "",
}

var UserConfig = DefaultConfig

// LoadConfig 负责读取用户配置，并自动补齐旧配置里缺失的新字段。
// 后续新增配置字段时，优先在 mergeConfigWithDefaults 中补默认值，避免旧用户升级后出现空字段。
func LoadConfig(version string) error {
	ucd, err := os.UserConfigDir()
	if err != nil {
		ucd = "./assets"
	}

	configFilePath = filepath.Join(ucd, appName, "config.json")
	config := DefaultConfig

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
		config = DefaultConfig
		config.Version = version
		UserConfig = config
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	config = mergeConfigWithDefaults(config)
	config.Version = version

	if config.Opacity < 20 || config.Opacity > 100 {
		config.Opacity = DefaultConfig.Opacity
	}
	if config.Volume < 0 || config.Volume > 127 {
		config.Volume = DefaultConfig.Volume
	}
	if config.MidiChannel > 15 {
		config.MidiChannel = DefaultConfig.MidiChannel
	}

	return SaveConfig(config)
}

// mergeConfigWithDefaults 用于兼容旧版本 config.json。
// 例如用户旧配置没有 midiChannel / soundFontPath 时，这里会补上安全默认值。
func mergeConfigWithDefaults(config Config) Config {
	merged := DefaultConfig
	merged.KeyLabel = config.KeyLabel
	merged.KeyboardType = config.KeyboardType
	merged.Velocity = config.Velocity
	merged.Opacity = config.Opacity
	merged.Version = config.Version
	merged.ShowPedal = config.ShowPedal
	merged.Volume = config.Volume
	merged.MidiChannel = config.MidiChannel
	merged.SoundFontPath = config.SoundFontPath

	merged.Colors = make(map[string]Color, len(DefaultConfig.Colors))
	for key, value := range DefaultConfig.Colors {
		merged.Colors[key] = value
	}
	for key, value := range config.Colors {
		merged.Colors[key] = value
	}

	return merged
}

func SaveConfig(config Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	UserConfig = mergeConfigWithDefaults(config)

	data, err := json.MarshalIndent(UserConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	dir := filepath.Dir(configFilePath)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}
	}

	if err := os.WriteFile(configFilePath, data, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

func GetUserConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return UserConfig
}

func ResolveSoundFontPath() string {
	config := GetUserConfig()
	if config.SoundFontPath != "" {
		if _, err := os.Stat(config.SoundFontPath); err == nil {
			return config.SoundFontPath
		}
	}
	return defaultSoundFontPath
}

func (k *Keyboard) SendConfig() Config {
	return GetUserConfig()
}

func (k *Keyboard) ReceiveConfig(config Config) (bool, string) {
	if err := SaveConfig(config); err != nil {
		return false, err.Error()
	}
	EmitConfigChanged()
	return true, ""
}

func (k *Keyboard) ResetConfig() Config {
	resetConfig := DefaultConfig
	resetConfig.Version = GetUserConfig().Version
	_ = SaveConfig(resetConfig)
	EmitConfigChanged()
	return resetConfig
}

func EmitConfigChanged() {
	if App != nil {
		App.Event.Emit("configChanged", GetUserConfig())
	}
}
