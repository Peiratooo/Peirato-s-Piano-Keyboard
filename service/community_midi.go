package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type communityMIDI struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Public       bool   `json:"public"`
	OwnerID      uint   `json:"ownerId"`
	ReviewStatus string `json:"reviewStatus"`
	SHA256       string `json:"sha256"`
}

func (k *Keyboard) LoadCommunityMIDI(id string) (UserMidi, error) {
	raw, _, err := communityFetch("/api/midis/"+url.PathEscape(id), "GET", nil, "", communityToken())
	if err != nil {
		return UserMidi{}, err
	}
	var song communityMIDI
	if err = json.Unmarshal(raw, &song); err != nil {
		return UserMidi{}, err
	}
	account := uint(0)
	if !song.Public || song.ReviewStatus != "approved" {
		account = communityCurrentUserID()
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return UserMidi{}, err
	}
	path := filepath.Join(dir, configDirectoryName, "community-midi", fmt.Sprint(account), song.ID+".mid")
	var result UserMidi
	cached := false
	for _, item := range GetMidiStore() {
		if item.CommunityMIDIID == id && item.CommunitySHA256 == song.SHA256 && item.CommunityAccountID == account {
			if _, err := os.Stat(item.Path); err == nil {
				result = item
				cached = true
				break
			}
		}
	}
	if !cached {
		data, _, err := communityFetch("/api/midis/"+url.PathEscape(id)+"/file", "GET", nil, "", communityToken())
		if err != nil {
			return result, err
		}
		if err = writeAtomicFile(path, data, 0600); err != nil {
			return result, err
		}
		result, err = k.AddMidiByPath(path)
		if err != nil {
			return result, err
		}
	}
	config := GetUserConfig()
	for i := range config.MidiStore {
		if config.MidiStore[i].ID == result.ID {
			config.MidiStore[i].Name = song.Title
			config.MidiStore[i].CommunityMIDIID = id
			config.MidiStore[i].CommunitySHA256 = song.SHA256
			config.MidiStore[i].CommunityAccountID = account
			result = config.MidiStore[i]
			break
		}
	}
	if err = SaveConfig(config); err != nil {
		return result, err
	}
	return result, nil
}
func (k *Keyboard) DownloadCommunityMIDI(id string) (bool, error) {
	raw, _, err := communityFetch("/api/midis/"+url.PathEscape(id), "GET", nil, "", communityToken())
	if err != nil {
		return false, err
	}
	var song communityMIDI
	if err = json.Unmarshal(raw, &song); err != nil {
		return false, err
	}
	name := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(song.Title) + ".mid"
	path, err := App.Dialog.SaveFile().SetMessage("保存 MIDI").SetFilename(name).AddFilter("MIDI", "*.mid").PromptForSingleSelection()
	if err != nil {
		return false, err
	}
	if path == "" {
		return false, nil
	}
	data, _, err := communityFetch("/api/midis/"+url.PathEscape(id)+"/file", "GET", nil, "", communityToken())
	if err != nil {
		return false, err
	}
	if len(data) == 0 {
		return false, errors.New("MIDI 文件为空")
	}
	if err = writeAtomicFile(path, data, 0600); err != nil {
		return false, err
	}
	_, _, err = communityFetch("/api/midis/"+url.PathEscape(id)+"/events", "POST", strings.NewReader(`{"type":"download"}`), "application/json", communityToken())
	return true, err
}
