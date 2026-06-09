package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultKeymapProfileID = "default"
	keymapExportFormat     = "peirato-keymap-profile"
	keymapExportVersion    = 1
)

type KeymapProfile struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Mapping map[string]uint8 `json:"mapping"`
}

type keymapProfileExport struct {
	Format  string           `json:"format"`
	Version int              `json:"version"`
	Name    string           `json:"name"`
	Mapping map[string]uint8 `json:"mapping"`
}

func defaultKeymapProfile() KeymapProfile {
	return KeymapProfile{
		ID:      defaultKeymapProfileID,
		Name:    "默认方案",
		Mapping: defaultKeymapMapping(),
	}
}

func defaultKeymapMapping() map[string]uint8 {
	return map[string]uint8{
		"1":  36,
		"!":  37,
		"2":  38,
		"@":  39,
		"3":  40,
		"4":  41,
		"$":  42,
		"5":  43,
		"%":  44,
		"6":  45,
		"^":  46,
		"7":  47,
		"8":  48,
		"*":  49,
		"9":  50,
		"(":  51,
		"0":  52,
		"-":  53,
		"_":  54,
		"=":  55,
		"+":  56,
		"q":  57,
		"Q":  58,
		"w":  59,
		"e":  60,
		"E":  61,
		"r":  62,
		"R":  63,
		"t":  64,
		"y":  65,
		"Y":  66,
		"u":  67,
		"U":  68,
		"i":  69,
		"I":  70,
		"o":  71,
		"p":  72,
		"P":  73,
		"[":  74,
		"{":  75,
		"]":  76,
		"\\": 77,
		"|":  78,
		"a":  72,
		"A":  73,
		"s":  74,
		"S":  75,
		"d":  76,
		"f":  77,
		"F":  78,
		"g":  79,
		"G":  80,
		"h":  81,
		"H":  82,
		"j":  83,
		"k":  84,
		"K":  85,
		"l":  86,
		"L":  87,
		";":  88,
		"'":  89,
		"z":  84,
		"Z":  85,
		"x":  86,
		"X":  87,
		"c":  88,
		"v":  89,
		"V":  90,
		"b":  91,
		"B":  92,
		"n":  93,
		"N":  94,
		"m":  95,
		",":  96,
		"<":  97,
		".":  98,
		">":  99,
		"/":  100,
		"?":  101,
	}
}

func cloneKeymapProfiles(profiles []KeymapProfile) []KeymapProfile {
	if profiles == nil {
		return nil
	}
	clone := make([]KeymapProfile, 0, len(profiles))
	for _, profile := range profiles {
		profile.Mapping = cloneKeymapMapping(profile.Mapping)
		clone = append(clone, profile)
	}
	return clone
}

func cloneKeymapMapping(mapping map[string]uint8) map[string]uint8 {
	clone := make(map[string]uint8, len(mapping))
	for key, value := range mapping {
		clone[key] = value
	}
	return clone
}

func normalizeKeymapConfig(config Config) Config {
	profiles := normalizeKeymapProfiles(config.KeymapProfiles)
	config.KeymapProfiles = profiles

	if !keymapProfileExists(profiles, config.ActiveKeymapProfileID) {
		config.ActiveKeymapProfileID = defaultKeymapProfileID
	}
	return config
}

func normalizeKeymapProfiles(profiles []KeymapProfile) []KeymapProfile {
	normalized := make([]KeymapProfile, 0, len(profiles)+1)
	seenIDs := map[string]bool{}
	hasDefault := false

	for _, profile := range profiles {
		profile.ID = strings.TrimSpace(profile.ID)
		if profile.ID == "" {
			profile.ID = newKeymapProfileID(normalized)
		}
		if seenIDs[profile.ID] {
			profile.ID = newKeymapProfileID(normalized)
		}
		if profile.ID == defaultKeymapProfileID {
			hasDefault = true
		}

		profile.Name = strings.TrimSpace(profile.Name)
		if profile.Name == "" {
			if profile.ID == defaultKeymapProfileID {
				profile.Name = "默认方案"
			} else {
				profile.Name = "未命名方案"
			}
		}
		profile.Mapping = sanitizeKeymapMapping(profile.Mapping)

		seenIDs[profile.ID] = true
		normalized = append(normalized, profile)
	}

	if !hasDefault {
		normalized = append([]KeymapProfile{defaultKeymapProfile()}, normalized...)
	}

	return normalized
}

func sanitizeKeymapMapping(mapping map[string]uint8) map[string]uint8 {
	sanitized := make(map[string]uint8, len(mapping))
	for key, value := range mapping {
		if key == "" || value > 127 {
			continue
		}
		sanitized[key] = value
	}
	return sanitized
}

func keymapProfileExists(profiles []KeymapProfile, id string) bool {
	_, ok := findKeymapProfileIndex(profiles, id)
	return ok
}

func findKeymapProfileIndex(profiles []KeymapProfile, id string) (int, bool) {
	for index, profile := range profiles {
		if profile.ID == id {
			return index, true
		}
	}
	return -1, false
}

func newKeymapProfileID(profiles []KeymapProfile) string {
	existing := make(map[string]bool, len(profiles))
	for _, profile := range profiles {
		existing[profile.ID] = true
	}

	base := fmt.Sprintf("keymap-%d", time.Now().UnixNano())
	if !existing[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !existing[candidate] {
			return candidate
		}
	}
}

func nextKeymapProfileName(profiles []KeymapProfile, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "新方案"
	}

	existing := make(map[string]bool, len(profiles))
	for _, profile := range profiles {
		existing[profile.Name] = true
	}
	if !existing[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s %d", base, i)
		if !existing[candidate] {
			return candidate
		}
	}
}

func saveKeymapConfig(config Config) (Config, error) {
	if err := SaveConfig(config); err != nil {
		return Config{}, err
	}
	return GetUserConfig(), nil
}

func (k *Keyboard) AddKeymapProfile() (Config, error) {
	config := normalizeKeymapConfig(GetUserConfig())
	profile := KeymapProfile{
		ID:      newKeymapProfileID(config.KeymapProfiles),
		Name:    nextKeymapProfileName(config.KeymapProfiles, "新方案"),
		Mapping: map[string]uint8{},
	}
	config.KeymapProfiles = append(config.KeymapProfiles, profile)
	config.ActiveKeymapProfileID = profile.ID
	return saveKeymapConfig(config)
}

func (k *Keyboard) DeleteKeymapProfile(id string) (Config, error) {
	if id == defaultKeymapProfileID {
		return Config{}, fmt.Errorf("默认方案不能删除")
	}

	config := normalizeKeymapConfig(GetUserConfig())
	index, ok := findKeymapProfileIndex(config.KeymapProfiles, id)
	if !ok {
		return Config{}, fmt.Errorf("按键方案不存在")
	}

	config.KeymapProfiles = append(config.KeymapProfiles[:index], config.KeymapProfiles[index+1:]...)
	if config.ActiveKeymapProfileID == id {
		config.ActiveKeymapProfileID = defaultKeymapProfileID
	}
	return saveKeymapConfig(config)
}

func (k *Keyboard) SelectKeymapProfile(id string) (Config, error) {
	config := normalizeKeymapConfig(GetUserConfig())
	if !keymapProfileExists(config.KeymapProfiles, id) {
		return Config{}, fmt.Errorf("按键方案不存在")
	}
	config.ActiveKeymapProfileID = id
	return saveKeymapConfig(config)
}

func (k *Keyboard) RenameKeymapProfile(id string, name string) (Config, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Config{}, fmt.Errorf("方案名称不能为空")
	}

	config := normalizeKeymapConfig(GetUserConfig())
	index, ok := findKeymapProfileIndex(config.KeymapProfiles, id)
	if !ok {
		return Config{}, fmt.Errorf("按键方案不存在")
	}

	config.KeymapProfiles[index].Name = name
	return saveKeymapConfig(config)
}

func (k *Keyboard) BindKeymapKey(profileID string, computerKey string, midiKey uint8) (Config, error) {
	if computerKey == "" {
		return Config{}, fmt.Errorf("电脑按键不能为空")
	}
	if midiKey > 127 {
		return Config{}, fmt.Errorf("MIDI 音符超出范围")
	}

	config := normalizeKeymapConfig(GetUserConfig())
	index, ok := findKeymapProfileIndex(config.KeymapProfiles, profileID)
	if !ok {
		return Config{}, fmt.Errorf("按键方案不存在")
	}

	mapping := cloneKeymapMapping(config.KeymapProfiles[index].Mapping)
	delete(mapping, computerKey)
	mapping[computerKey] = midiKey
	config.KeymapProfiles[index].Mapping = mapping
	return saveKeymapConfig(config)
}

func (k *Keyboard) RemoveKeymapBinding(profileID string, computerKey string) (Config, error) {
	config := normalizeKeymapConfig(GetUserConfig())
	index, ok := findKeymapProfileIndex(config.KeymapProfiles, profileID)
	if !ok {
		return Config{}, fmt.Errorf("按键方案不存在")
	}

	mapping := cloneKeymapMapping(config.KeymapProfiles[index].Mapping)
	delete(mapping, computerKey)
	config.KeymapProfiles[index].Mapping = mapping
	return saveKeymapConfig(config)
}

func (k *Keyboard) ExportActiveKeymapProfile() error {
	if App == nil {
		return fmt.Errorf("应用尚未初始化")
	}

	config := normalizeKeymapConfig(GetUserConfig())
	index, ok := findKeymapProfileIndex(config.KeymapProfiles, config.ActiveKeymapProfileID)
	if !ok {
		return fmt.Errorf("当前按键方案不存在")
	}
	profile := config.KeymapProfiles[index]

	path, err := App.Dialog.SaveFile().
		SetFilename(safeKeymapFilename(profile.Name)+".pp-keymap.json").
		AddFilter("按键方案", "*.pp-keymap.json;*.json").
		CanCreateDirectories(true).
		PromptForSingleSelection()
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("未选择保存路径")
	}

	if filepath.Ext(path) == "" {
		path += ".pp-keymap.json"
	}

	payload := keymapProfileExport{
		Format:  keymapExportFormat,
		Version: keymapExportVersion,
		Name:    profile.Name,
		Mapping: cloneKeymapMapping(profile.Mapping),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化按键方案失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入按键方案失败: %w", err)
	}
	return nil
}

func (k *Keyboard) ImportKeymapProfile() (Config, error) {
	if App == nil {
		return Config{}, fmt.Errorf("应用尚未初始化")
	}

	path, err := App.Dialog.OpenFile().
		SetTitle("导入按键方案").
		AddFilter("按键方案", "*.pp-keymap.json;*.json").
		PromptForSingleSelection()
	if err != nil {
		return Config{}, err
	}
	if path == "" {
		return Config{}, fmt.Errorf("未选择按键方案文件")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取按键方案失败: %w", err)
	}

	profile, err := parseKeymapProfileExport(data)
	if err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(profile.Name) == "" {
		profile.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	config := normalizeKeymapConfig(GetUserConfig())
	profile.ID = newKeymapProfileID(config.KeymapProfiles)
	profile.Name = nextKeymapProfileName(config.KeymapProfiles, profile.Name)
	profile.Mapping = sanitizeKeymapMapping(profile.Mapping)
	config.KeymapProfiles = append(config.KeymapProfiles, profile)
	config.ActiveKeymapProfileID = profile.ID
	return saveKeymapConfig(config)
}

func parseKeymapProfileExport(data []byte) (KeymapProfile, error) {
	var payload keymapProfileExport
	if err := json.Unmarshal(data, &payload); err != nil {
		return KeymapProfile{}, fmt.Errorf("解析按键方案失败: %w", err)
	}
	if payload.Mapping == nil {
		return KeymapProfile{}, fmt.Errorf("按键方案文件缺少 mapping 字段")
	}

	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = "导入方案"
	}

	return KeymapProfile{
		Name:    name,
		Mapping: sanitizeKeymapMapping(payload.Mapping),
	}, nil
}

func safeKeymapFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "keymap"
	}

	invalid := strings.NewReplacer(
		"<", "_",
		">", "_",
		":", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	name = invalid.Replace(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 {
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(name, ". ")
	if name == "" {
		return "keymap"
	}
	return name
}
