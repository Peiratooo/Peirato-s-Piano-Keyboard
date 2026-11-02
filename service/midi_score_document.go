package service

import (
    "errors"
    "fmt"
)

// MidiScoreDocument describes the notation source. For MIDI, the frontend
// derives a best-effort readable score from the immutable playback timeline.
// For MusicXML/MXL, the original source is returned without converting it
// back from timed note events, preserving engraving and voice information.
type MidiScoreDocument struct {
    XML string `json:"xml"`
    Native bool `json:"native"`
}

func (k *Keyboard) GetMidiScoreDocument(id string) (MidiScoreDocument, error) {
    song, ok := FindUserMidiByID(id)
    if !ok {
        return MidiScoreDocument{}, errors.New("曲目不存在")
    }
    // Validate the original source against the exact same parser used by
    // playback before making any notation source available to the UI.
    if _, err := GetOrParse(song); err != nil {
        return MidiScoreDocument{}, fmt.Errorf("无法读取曲目: %w", err)
    }
    if !isMusicXMLPath(song.Path) {
        return MidiScoreDocument{}, nil
    }
    raw, err := readMusicXML(song.Path)
    if err != nil {
        return MidiScoreDocument{}, fmt.Errorf("无法读取 MusicXML 乐谱: %w", err)
    }
    return MidiScoreDocument{XML: string(raw), Native: true}, nil
}
