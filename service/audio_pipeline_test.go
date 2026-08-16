package service

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/sinshu/go-meltysynth/meltysynth"
)

func TestMasterVolumeGain(t *testing.T) {
	tests := []struct {
		volume int32
		want   float32
	}{
		{volume: 0, want: 0},
		{volume: 100, want: 0.5},
	}

	for _, test := range tests {
		got := masterVolumeGain(test.volume)
		if math.Abs(float64(got-test.want)) > 0.0001 {
			t.Errorf("masterVolumeGain(%d) = %f, want %f", test.volume, got, test.want)
		}
	}

	if !(masterVolumeGain(50) > masterVolumeGain(0) && masterVolumeGain(50) < masterVolumeGain(80) && masterVolumeGain(80) < masterVolumeGain(100)) {
		t.Fatal("master volume gain should increase smoothly between mute and the unity reference")
	}
}

func TestClampMidiByte(t *testing.T) {
	for input, want := range map[int]int{-10: 0, 0: 0, 64: 64, 127: 127, 200: 127} {
		if got := clampMidiByte(input); got != want {
			t.Errorf("clampMidiByte(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestMeltySynthSustainControlChange(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "soundfont", "Yamaha-Grand-Lite-v2.0.sf2"))
	if err != nil {
		t.Fatal(err)
	}
	font, err := meltysynth.NewSoundFont(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	withSustain := newTestSynth(t, font)
	withoutSustain := newTestSynth(t, font)

	withSustain.ProcessMidiMessage(0, midiCommandNoteOn, 60, 110)
	withoutSustain.ProcessMidiMessage(0, midiCommandNoteOn, 60, 110)
	withSustain.ProcessMidiMessage(0, midiCommandControlChange, midiCCSustain, 127)
	withSustain.ProcessMidiMessage(0, midiCommandNoteOff, 60, 0)
	withoutSustain.ProcessMidiMessage(0, midiCommandNoteOff, 60, 0)

	sustainedEnergy := renderEnergy(withSustain, 4096)
	releaseEnergy := renderEnergy(withoutSustain, 4096)
	if sustainedEnergy <= releaseEnergy {
		t.Fatalf("CC64 sustain energy = %f, normal release energy = %f", sustainedEnergy, releaseEnergy)
	}

	withSustain.ProcessMidiMessage(0, midiCommandControlChange, midiCCSustain, 0)
}

func TestMeltySynthInteractiveChannelRecoversFromMidiMute(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "soundfont", "Yamaha-Grand-Lite-v2.0.sf2"))
	if err != nil {
		t.Fatal(err)
	}
	font, err := meltysynth.NewSoundFont(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	synth := newTestSynth(t, font)

	// This is the state a MIDI file can leave behind after CC7/CC11 events.
	synth.ProcessMidiMessage(0, midiCommandControlChange, midiCCChannelVolume, 0)
	synth.ProcessMidiMessage(0, midiCommandControlChange, midiCCExpression, 0)
	synth.ProcessMidiMessage(0, midiCommandNoteOn, 60, 100)
	mutedEnergy := renderEnergy(synth, 2048)

	synth.ProcessMidiMessage(0, midiCommandControlChange, midiCCChannelVolume, 127)
	synth.ProcessMidiMessage(0, midiCommandControlChange, midiCCPan, 64)
	synth.ProcessMidiMessage(0, midiCommandControlChange, midiCCExpression, 127)
	synth.ProcessMidiMessage(0, midiCommandNoteOn, 60, 100)
	recoveredEnergy := renderEnergy(synth, 2048)
	if mutedEnergy != 0 || recoveredEnergy <= 0 {
		t.Fatalf("interactive channel recovery failed: muted=%f recovered=%f", mutedEnergy, recoveredEnergy)
	}
}

func newTestSynth(t *testing.T, font *meltysynth.SoundFont) *meltysynth.Synthesizer {
	t.Helper()
	synth, err := meltysynth.NewSynthesizer(font, meltysynth.NewSynthesizerSettings(44100))
	if err != nil {
		t.Fatal(err)
	}
	return synth
}

func renderEnergy(synth *meltysynth.Synthesizer, samples int) float64 {
	left := make([]float32, samples)
	right := make([]float32, samples)
	synth.Render(left, right)
	var energy float64
	for i := range left {
		energy += math.Abs(float64(left[i])) + math.Abs(float64(right[i]))
	}
	return energy
}
