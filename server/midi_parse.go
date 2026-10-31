package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

type MIDIInfo struct {
	NoteCount     int
	TrackCount    int
	Measures      int
	DurationMs    int64
	TempoBPM      float64
	TimeSignature string
	KeySignature  string
	TrackNames    string
}
type tempoPoint struct {
	tick   uint64
	micros uint32
}
type meterPoint struct {
	tick        uint64
	numerator   uint8
	denominator uint16
}

func parseVLQ(data []byte, p *int, end int) (uint64, error) {
	var v uint64
	for i := 0; i < 4; i++ {
		if *p >= end {
			return 0, errors.New("truncated VLQ")
		}
		b := data[*p]
		*p = *p + 1
		v = (v << 7) | uint64(b&127)
		if b&128 == 0 {
			return v, nil
		}
	}
	return 0, errors.New("invalid VLQ")
}
func parseMIDIInfo(data []byte) (MIDIInfo, error) {
	var out MIDIInfo
	if len(data) < 14 || string(data[:4]) != "MThd" {
		return out, errors.New("invalid MIDI header")
	}
	hlen := int(binary.BigEndian.Uint32(data[4:8]))
	if hlen < 6 || 8+hlen > len(data) {
		return out, errors.New("invalid header length")
	}
	format := binary.BigEndian.Uint16(data[8:10])
	tracks := int(binary.BigEndian.Uint16(data[10:12]))
	ppq := binary.BigEndian.Uint16(data[12:14])
	if format > 1 || tracks == 0 || tracks > 128 || ppq == 0 || ppq&0x8000 != 0 {
		return out, errors.New("only SMF type 0/1 with PPQ timing supported")
	}
	out.TrackCount = tracks
	pos := 8 + hlen
	tempos := []tempoPoint{{0, 500000}}
	meters := []meterPoint{{0, 4, 4}}
	keySF := 99
	keyMinor := false
	totalEvents := 0
	var maxTick uint64
	names := make([]string, tracks)
	for t := 0; t < tracks; t++ {
		if pos+8 > len(data) || string(data[pos:pos+4]) != "MTrk" {
			return out, fmt.Errorf("missing MIDI track %d", t)
		}
		length := int(binary.BigEndian.Uint32(data[pos+4 : pos+8]))
		end := pos + 8 + length
		if length < 0 || end < pos+8 || end > len(data) {
			return out, errors.New("invalid MIDI track size")
		}
		p := pos + 8
		tick := uint64(0)
		running := byte(0)
		names[t] = fmt.Sprintf("轨道 %d", t+1)
		for p < end {
			delta, err := parseVLQ(data, &p, end)
			if err != nil {
				return out, err
			}
			tick += delta
			if tick > maxTick {
				maxTick = tick
			}
			if p >= end {
				return out, errors.New("truncated MIDI event")
			}
			status := data[p]
			if status&0x80 != 0 {
				p++
				if status < 0xf0 {
					running = status
				} else {
					running = 0
				}
			} else {
				if running == 0 {
					return out, errors.New("invalid running status")
				}
				status = running
			}
			totalEvents++
			if totalEvents > 500000 {
				return out, errors.New("too many MIDI events")
			}
			switch {
			case status == 0xff:
				if p >= end {
					return out, errors.New("truncated meta event")
				}
				kind := data[p]
				p++
				n, err := parseVLQ(data, &p, end)
				if err != nil {
					return out, err
				}
				if n > uint64(end-p) {
					return out, errors.New("truncated meta payload")
				}
				payload := data[p : p+int(n)]
				p += int(n)
				switch kind {
				case 0x03:
					if len(payload) > 0 {
						names[t] = cleanText(string(payload), 64)
					}
				case 0x51:
					if len(payload) == 3 {
						tempo := uint32(payload[0])<<16 | uint32(payload[1])<<8 | uint32(payload[2])
						if tempo > 0 {
							tempos = append(tempos, tempoPoint{tick, tempo})
						}
					}
				case 0x58:
					if len(payload) >= 2 && payload[0] > 0 && payload[1] <= 7 {
						meters = append(meters, meterPoint{tick, payload[0], 1 << payload[1]})
					}
				case 0x59:
					if len(payload) >= 2 {
						sf := int(int8(payload[0]))
						if sf >= -7 && sf <= 7 {
							keySF = sf
							keyMinor = payload[1] == 1
						}
					}
				}
			case status == 0xf0 || status == 0xf7:
				n, err := parseVLQ(data, &p, end)
				if err != nil {
					return out, err
				}
				if n > uint64(end-p) {
					return out, errors.New("invalid sysex length")
				}
				p += int(n)
			case status >= 0x80 && status <= 0xef:
				kind := status & 0xf0
				need := 2
				if kind == 0xc0 || kind == 0xd0 {
					need = 1
				}
				if p+need > end {
					return out, errors.New("truncated channel event")
				}
				a := data[p]
				b := byte(0)
				if need == 2 {
					b = data[p+1]
				}
				p += need
				if kind == 0x90 && b > 0 && a <= 127 {
					out.NoteCount++
				}
			default:
				return out, errors.New("unsupported MIDI system event")
			}
		}
		pos = end
	}
	if out.NoteCount == 0 {
		return out, errors.New("MIDI has no note-on events")
	}
	// Meta may appear on multiple tracks. Sort and use the final event per tick.
	sortTempo(tempos)
	sortMeter(meters)
	tempos = mergeTempo(tempos)
	meters = mergeMeter(meters)
	elapsed := 0.0
	lastTick := uint64(0)
	currentTempo := uint32(500000)
	for _, event := range tempos {
		if event.tick > maxTick {
			break
		}
		if event.tick > lastTick {
			elapsed += float64(event.tick-lastTick) * float64(currentTempo) / (float64(ppq) * 1000)
			lastTick = event.tick
		}
		currentTempo = event.micros
	}
	if maxTick > lastTick {
		elapsed += float64(maxTick-lastTick) * float64(currentTempo) / (float64(ppq) * 1000)
	}
	out.DurationMs = int64(math.Round(elapsed))
	out.TempoBPM = math.Round(60000000/float64(tempos[0].micros)*10) / 10
	out.TimeSignature = fmt.Sprintf("%d/%d", meters[0].numerator, meters[0].denominator)
	out.KeySignature = "未标注"
	if keySF != 99 {
		major := []string{"Cb", "Gb", "Db", "Ab", "Eb", "Bb", "F", "C", "G", "D", "A", "E", "B", "F#", "C#"}
		minor := []string{"Abm", "Ebm", "Bbm", "Fm", "Cm", "Gm", "Dm", "Am", "Em", "Bm", "F#m", "C#m", "G#m", "D#m", "A#m"}
		if keyMinor {
			out.KeySignature = minor[keySF+7]
		} else {
			out.KeySignature = major[keySF+7]
		}
	}
	// Count measures using time-signature segments. Mid-bar signature changes
	// start a new measure; the result is a conservative metadata estimate.
	measures := 0
	for i, m := range meters {
		if m.tick > maxTick {
			break
		}
		end := maxTick
		if i+1 < len(meters) && meters[i+1].tick <= maxTick {
			end = meters[i+1].tick
		}
		if end < m.tick {
			continue
		}
		ticksPerBar := float64(ppq) * 4 * float64(m.numerator) / float64(m.denominator)
		if ticksPerBar < 1 {
			continue
		}
		bars := int(math.Ceil(float64(end-m.tick) / ticksPerBar))
		if bars < 1 {
			bars = 1
		}
		measures += bars
	}
	if measures > 100000 {
		measures = 100000
	}
	out.Measures = measures
	packed, _ := json.Marshal(names)
	out.TrackNames = string(packed)
	return out, nil
}
func sortTempo(a []tempoPoint) {
	sort.SliceStable(a, func(i, j int) bool { return a[i].tick < a[j].tick })
}
func sortMeter(a []meterPoint) {
	sort.SliceStable(a, func(i, j int) bool { return a[i].tick < a[j].tick })
}
func mergeTempo(list []tempoPoint) []tempoPoint {
	out := make([]tempoPoint, 0, len(list))
	for _, item := range list {
		if len(out) > 0 && out[len(out)-1].tick == item.tick {
			out[len(out)-1] = item
		} else {
			out = append(out, item)
		}
	}
	return out
}
func mergeMeter(list []meterPoint) []meterPoint {
	out := make([]meterPoint, 0, len(list))
	for _, item := range list {
		if len(out) > 0 && out[len(out)-1].tick == item.tick {
			out[len(out)-1] = item
		} else {
			out = append(out, item)
		}
	}
	return out
}
func sanitizeTags(tags string) string {
	parts := strings.Split(tags, ",")
	out := make([]string, 0, 8)
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.ToLower(cleanText(p, 20))
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
		if len(out) == 8 {
			break
		}
	}
	return strings.Join(out, ",")
}
