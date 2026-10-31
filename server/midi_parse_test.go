package main

import (
	"encoding/hex"
	"testing"
)

func TestParseBasicMIDI(t *testing.T) {
	// Type 0; tempo default, 480 PPQ; one C4 quarter note.
	b, _ := hex.DecodeString("4d546864000000060000000101e04d54726b0000000d00903c648360803c0000ff2f00")
	info, err := parseMIDIInfo(b)
	if err != nil {
		t.Fatal(err)
	}
	if info.NoteCount != 1 || info.TrackCount != 1 || info.DurationMs != 500 || info.Measures != 1 {
		t.Fatalf("unexpected metadata: %+v", info)
	}
}
func TestRejectInvalidMIDI(t *testing.T) {
	if _, err := parseMIDIInfo([]byte("bogus")); err == nil {
		t.Fatal("expected error")
	}
}
