package service

import (
	"os"
	"path/filepath"
)

// CheckMidiPathExists 供 MIDI 独立窗口在切换目录记录时做轻量校验。
// MIDI 目录只保存绝对路径，不复制文件；真实文件被移动或删除时，前端会提示用户移除这条记录。
func (k *Keyboard) CheckMidiPathExists(path string) bool {
	if path == "" || !filepath.IsAbs(path) {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// LoadMidiFileFromPath 从系统绝对路径读取并解析 MIDI。
// 这和前端上传 base64 的入口并存：独立 MIDI 窗口优先用绝对路径加载，旧逻辑和兼容场景仍可用 base64。
func (k *Keyboard) LoadMidiFileFromPath(path string) (MidiFileInfo, error) {
	if path == "" || !filepath.IsAbs(path) {
		return MidiFileInfo{}, os.ErrInvalid
	}
	data, err := os.ReadFile(path)
	if err != nil {
		playback.setError(err)
		return MidiFileInfo{}, err
	}
	parsed, err := ParseMidiFileBytes(filepath.Base(path), data)
	if err != nil {
		playback.setError(err)
		return MidiFileInfo{}, err
	}
	playback.load(parsed)
	emitMidiPlaybackLoaded(parsed)
	emitMidiPlaybackState(playback.state())
	return parsed, nil
}
