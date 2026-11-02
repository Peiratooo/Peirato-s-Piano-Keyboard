package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type LocalPractice struct {
	LocalID       string           `json:"localId"`
	Title         string           `json:"title"`
	Sessions      int              `json:"sessions"`
	Seconds       float64          `json:"seconds"`
	LastPracticed time.Time        `json:"lastPracticed"`
	LastMode      MidiPlaybackMode `json:"lastMode"`
}

var localPractice = struct {
	sync.Mutex
	Loaded   bool
	Items    []LocalPractice
	Previous MidiPlayerState
	Last     time.Time
	Saved    time.Time
}{Items: []LocalPractice{}}

func localPracticePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configDirectoryName, "practice-history.json"), nil
}
func loadLocalPracticeLocked() error {
	if localPractice.Loaded {
		return nil
	}
	path, err := localPracticePath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil {
		if err = json.Unmarshal(raw, &localPractice.Items); err != nil {
			return err
		}
	}
	localPractice.Loaded = true
	return nil
}
func recordLocalPractice(state MidiPlayerState) {
	localPractice.Lock()
	defer localPractice.Unlock()
	if loadLocalPracticeLocked() != nil {
		return
	}
	now := time.Now()
	previous := localPractice.Previous
	if previous.Status == MidiPlayPlaying && !localPractice.Last.IsZero() {
		for i := range localPractice.Items {
			if localPractice.Items[i].LocalID == previous.MidiID {
				localPractice.Items[i].Seconds += now.Sub(localPractice.Last).Seconds()
				localPractice.Items[i].LastPracticed = now
				break
			}
		}
	}
	changed := state.Status != previous.Status || state.MidiID != previous.MidiID || state.Mode != previous.Mode
	if state.Status == MidiPlayPlaying && (state.MidiID != previous.MidiID || state.Mode != previous.Mode || previous.Status != MidiPlayPlaying && previous.Status != MidiPlayPaused) {
		index := -1
		for i := range localPractice.Items {
			if localPractice.Items[i].LocalID == state.MidiID {
				index = i
				break
			}
		}
		if index < 0 {
			for _, song := range GetMidiStore() {
				if song.ID == state.MidiID {
					localPractice.Items = append(localPractice.Items, LocalPractice{LocalID: song.ID, Title: song.Name})
					index = len(localPractice.Items) - 1
					break
				}
			}
		}
		if index >= 0 {
			item := &localPractice.Items[index]
			item.Sessions++
			item.LastPracticed = now
			item.LastMode = state.Mode
		}
	}
	if state.Status == MidiPlayPlaying {
		for i := range localPractice.Items {
			if localPractice.Items[i].LocalID == state.MidiID {
				localPractice.Items[i].LastMode = state.Mode
				localPractice.Items[i].LastPracticed = now
				break
			}
		}
	}
	localPractice.Previous = state
	localPractice.Last = now
	if changed || now.Sub(localPractice.Saved) >= 15*time.Second {
		path, err := localPracticePath()
		if err != nil {
			return
		}
		raw, err := json.Marshal(localPractice.Items)
		if err != nil {
			return
		}
		if err = writeAtomicFile(path, raw, 0600); err == nil {
			localPractice.Saved = now
		}
	}
}
func (k *Keyboard) GetLocalPractice() ([]LocalPractice, error) {
	localPractice.Lock()
	defer localPractice.Unlock()
	if err := loadLocalPracticeLocked(); err != nil {
		return nil, err
	}
	items := make([]LocalPractice, 0, len(localPractice.Items))
	items = append(items, localPractice.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].LastPracticed.After(items[j].LastPracticed) })
	return items, nil
}
