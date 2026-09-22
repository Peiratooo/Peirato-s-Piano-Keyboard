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
	Revision              uint64           `json:"revision"`
	Colors                map[string]Color `json:"colors"`
	KeyLabel              string           `json:"keyLabel"`
	KeyTonic              int              `json:"keyTonic"`
	KeyboardType          int              `json:"keyboardType"`
	Velocity              uint8            `json:"velocity"`
	Opacity               int              `json:"opacity"`
	Version               string           `json:"version"`
	ShowPedal             bool             `json:"showPedal"`
	Volume                int32            `json:"volume"`
	SampleRate            int32            `json:"sampleRate"`
	BufferSize            int32            `json:"bufferSize"`
	MidiChannel           uint8            `json:"midiChannel"`
	ActiveSoundFontID     string           `json:"activeSoundFontId"`
	SoundFonts            []UserSoundFont  `json:"soundFonts"`
	MidiStore             []UserMidi       `json:"midiStore"`
	ActiveKeymapProfileID string           `json:"activeKeymapProfileId"`
	KeymapProfiles        []KeymapProfile  `json:"keymapProfiles"`
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
	KeyLabel:              "octave_key",
	KeyboardType:          0,
	Velocity:              80,
	Volume:                80,
	SampleRate:            44100,
	BufferSize:            2048,
	Opacity:               100,
	ShowPedal:             true,
	MidiChannel:           0,
	SoundFonts:            []UserSoundFont{},
	ActiveSoundFontID:     "",
	ActiveKeymapProfileID: defaultKeymapProfileID,
	KeymapProfiles:        []KeymapProfile{defaultKeymapProfile()},
}

var UserConfig = cloneDefaultConfig()

func cloneDefaultConfig() Config {
	return cloneConfig(DefaultConfig)
}

func cloneConfig(source Config) Config {
	result := source
	result.Colors = make(map[string]Color, len(source.Colors))
	for key, value := range source.Colors {
		result.Colors[key] = value
	}
	result.SoundFonts = append([]UserSoundFont{}, source.SoundFonts...)
	result.MidiStore = append([]UserMidi{}, source.MidiStore...)
	result.KeymapProfiles = cloneKeymapProfiles(source.KeymapProfiles)
	return result
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
		backup, backupErr := os.ReadFile(configFilePath + ".bak")
		config = cloneDefaultConfig()
		if backupErr != nil || json.Unmarshal(backup, &config) != nil {
			config.Version = version
			UserConfig = config
			return fmt.Errorf("配置与备份均无法读取: %w", err)
		}
	}

	config = mergeConfigWithDefaults(config)
	config.Version = version

	config = normalizeConfigRanges(config)
	configMu.Lock()
	UserConfig = cloneConfig(config)
	configMu.Unlock()

	return SaveConfig(config)
}

// mergeConfigWithDefaults 用于兼容旧版本 config.json。
// 例如用户旧配置没有 midiChannel / soundFontPath 时，这里会补上安全默认值。
func mergeConfigWithDefaults(config Config) Config {
	merged := cloneDefaultConfig()
	merged.Revision = config.Revision

	merged.KeyLabel = config.KeyLabel
	merged.KeyTonic = config.KeyTonic
	merged.KeyboardType = config.KeyboardType
	merged.Velocity = config.Velocity
	merged.Opacity = config.Opacity
	merged.Version = config.Version
	merged.ShowPedal = config.ShowPedal
	merged.Volume = config.Volume
	merged.SampleRate = config.SampleRate
	merged.BufferSize = config.BufferSize
	merged.MidiChannel = config.MidiChannel
	merged.SoundFonts = append([]UserSoundFont{}, config.SoundFonts...)
	merged.ActiveSoundFontID = config.ActiveSoundFontID
	merged.MidiStore = append([]UserMidi{}, config.MidiStore...)
	merged.ActiveKeymapProfileID = config.ActiveKeymapProfileID
	merged.KeymapProfiles = cloneKeymapProfiles(config.KeymapProfiles)

	for key, value := range config.Colors {
		merged.Colors[key] = value
	}

	return normalizeConfigRanges(merged)
}

func normalizeConfigRanges(config Config) Config {
	if config.KeyTonic < 0 || config.KeyTonic > 11 {
		config.KeyTonic = 0
	}
	if config.KeyboardType < 0 || config.KeyboardType > 5 {
		config.KeyboardType = DefaultConfig.KeyboardType
	}
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
	return normalizeKeymapConfig(config)
}

func SaveConfig(config Config) error {
	configMu.Lock()
	if config.Revision != UserConfig.Revision {
		configMu.Unlock()
		return fmt.Errorf("设置已被其他窗口修改，请刷新后重试")
	}
	nextConfig := mergeConfigWithDefaults(config)
	nextConfig.Revision++

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

	if previous, err := os.ReadFile(configFilePath); err == nil && json.Valid(previous) {
		if err = writeAtomicFile(configFilePath+".bak", previous, 0600); err != nil {
			configMu.Unlock()
			return fmt.Errorf("备份配置失败: %w", err)
		}
	}
	if err := writeAtomicFile(configFilePath, data, 0600); err != nil {
		configMu.Unlock()
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	UserConfig = nextConfig

	configMu.Unlock()
	SetMasterVolume(nextConfig.Volume)
	EmitConfigChanged()

	return nil
}

func GetUserConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return cloneConfig(UserConfig)
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

func (k *Keyboard) ResetConfig() (Config, error) {
	err := withSoundFontChange(func() error {
		current := GetUserConfig()
		resetConfig := cloneDefaultConfig()
		resetConfig.Version, resetConfig.Revision = current.Version, current.Revision
		if err := SwitchDefaultSoundFont(); err != nil {
			return err
		}
		return SaveConfig(resetConfig)
	})
	return GetUserConfig(), err
}

func EmitConfigChanged() {
	if App != nil {
		App.Event.Emit("configChanged", GetUserConfig())
	}
}
