package service

import (
    "archive/zip"
    "bytes"
    "encoding/xml"
    "errors"
    "fmt"
    "io"
    "math"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "time"
)

// MusicXML files provide notation, while the existing playback engine consumes
// MIDI-like note events. This importer preserves exact written pitches and
// relative note durations without trying to infer a score from MIDI.
// First pass: score-partwise, divisions, voices, chords, backups, rests,
// ties and tempo directions. Repeat expansion and ornaments are not yet supported.
type musicXMLScore struct {
    XMLName xml.Name `xml:"score-partwise"`
    Parts []musicXMLPart `xml:"part"`
}
type musicXMLPart struct {
    ID string `xml:"id,attr"`
    Measures []musicXMLMeasure `xml:"measure"`
}
type musicXMLMeasure struct {
    Number string `xml:"number,attr"`
    Inner []byte `xml:",innerxml"`
}
type musicXMLNote struct {
    Pitch struct {
        Step string `xml:"step"`
        Alter int `xml:"alter"`
        Octave int `xml:"octave"`
    } `xml:"pitch"`
    Duration int `xml:"duration"`
    Voice string `xml:"voice"`
    Staff int `xml:"staff"`
    Chord *struct{} `xml:"chord"`
    Rest *struct{} `xml:"rest"`
    Grace *struct{} `xml:"grace"`
    Ties []struct { Type string `xml:"type,attr"` } `xml:"tie"`
}
type musicXMLAttributes struct {
    Divisions int `xml:"divisions"`
    Time struct {
        Beats int `xml:"beats"`
        BeatType int `xml:"beat-type"`
    } `xml:"time"`
}
type musicXMLMovement struct {
    Duration int `xml:"duration"`
}
type musicXMLDirection struct {
    Sound struct { Tempo float64 `xml:"tempo,attr"` } `xml:"sound"`
}
type scoreBeatNote struct {
    start, end float64
    pitch, track int
    hand MidiHand
    tieStart, tieStop bool
    tieKey string
}
type scoreTempo struct { at, bpm float64 }

const maxMusicXMLBytes = 24 << 20

func readMusicXML(path string) ([]byte, error) {
    if strings.EqualFold(filepath.Ext(path), ".mxl") {
        z, err := zip.OpenReader(path)
        if err != nil { return nil, fmt.Errorf("无效的 MXL 文件: %w", err) }
        defer z.Close()
        target := ""
        for _, f := range z.File {
            if f.Name != "META-INF/container.xml" { continue }
            r, err := f.Open()
            if err != nil { return nil, err }
            raw, readErr := io.ReadAll(io.LimitReader(r, 65536))
            r.Close()
            if readErr != nil { return nil, readErr }
            var container struct {
                Rootfiles []struct { FullPath string `xml:"full-path,attr"` } `xml:"rootfiles>rootfile"`
            }
            if xml.Unmarshal(raw, &container) == nil && len(container.Rootfiles) > 0 {
                target = container.Rootfiles[0].FullPath
            }
            break
        }
        for _, f := range z.File {
            if target != "" && f.Name != target { continue }
            if target == "" && (strings.HasPrefix(f.Name, "META-INF/") ||
                !(strings.HasSuffix(strings.ToLower(f.Name), ".xml") ||
                  strings.HasSuffix(strings.ToLower(f.Name), ".musicxml"))) { continue }
            if f.UncompressedSize64 > maxMusicXMLBytes { return nil, errors.New("MusicXML 文件过大") }
            r, err := f.Open()
            if err != nil { return nil, err }
            raw, readErr := io.ReadAll(io.LimitReader(r, maxMusicXMLBytes+1))
            r.Close()
            if readErr != nil { return nil, readErr }
            if len(raw) > maxMusicXMLBytes { return nil, errors.New("MusicXML 文件过大") }
            return raw, nil
        }
        return nil, errors.New("MXL 中找不到乐谱 XML")
    }
    f, err := os.Open(path)
    if err != nil { return nil, err }
    defer f.Close()
    raw, err := io.ReadAll(io.LimitReader(f, maxMusicXMLBytes+1))
    if err != nil { return nil, err }
    if len(raw) > maxMusicXMLBytes { return nil, errors.New("MusicXML 文件过大") }
    return raw, nil
}

func scorePitch(n musicXMLNote) (int, error) {
    semitone := map[string]int{"C":0,"D":2,"E":4,"F":5,"G":7,"A":9,"B":11}
    base, ok := semitone[strings.ToUpper(n.Pitch.Step)]
    if !ok || n.Pitch.Octave < -1 || n.Pitch.Octave > 9 || n.Pitch.Alter < -2 || n.Pitch.Alter > 2 {
        return 0, fmt.Errorf("MusicXML 包含无法识别的音高: %s%d", n.Pitch.Step, n.Pitch.Octave)
    }
    midi := (n.Pitch.Octave+1)*12 + base + n.Pitch.Alter
    if midi < 0 || midi > 127 { return 0, fmt.Errorf("音高超出 MIDI 范围: %d", midi) }
    return midi, nil
}

func scoreTimeMs(beat float64, changes []scoreTempo) float64 {
    if beat <= 0 { return 0 }
    ms, lastBeat, bpm := 0.0, 0.0, 120.0
    for _, t := range changes {
        if t.at > beat { break }
        if t.at > lastBeat {
            ms += (t.at-lastBeat)*60000/bpm
            lastBeat = t.at
        }
        bpm = t.bpm
    }
    return ms+(beat-lastBeat)*60000/bpm
}

func isMusicXMLPath(path string) bool {
    switch strings.ToLower(filepath.Ext(path)) {
    case ".xml", ".musicxml", ".mxl": return true
    }
    return false
}

func ParseMusicXMLFile(path string) (*Midi, error) {
    raw, err := readMusicXML(path)
    if err != nil { return nil, err }
    var score musicXMLScore
    if err := xml.Unmarshal(raw, &score); err != nil { return nil, fmt.Errorf("MusicXML 解析失败: %w", err) }
    if score.XMLName.Local != "score-partwise" || len(score.Parts) == 0 {
        return nil, errors.New("目前只支持 score-partwise 格式的 MusicXML")
    }
    var beatNotes []scoreBeatNote
    tempos := []scoreTempo{{at:0,bpm:120}}
    measureBeats := []float64{}
    songEnd := 0.0
    for partIndex, part := range score.Parts {
        divisions := 1
        meterLength := 4.0
        beat := 0.0
        for _, measure := range part.Measures {
            if partIndex == 0 { measureBeats = append(measureBeats, beat) }
            cursor, maxCursor, lastNoteOnset := 0.0, 0.0, 0.0
            decoder := xml.NewDecoder(bytes.NewReader(measure.Inner))
            for {
                token, err := decoder.Token()
                if errors.Is(err, io.EOF) { break }
                if err != nil { return nil, fmt.Errorf("小节 XML 无效: %w", err) }
                start, ok := token.(xml.StartElement)
                if !ok { continue }
                switch start.Name.Local {
                case "attributes":
                    var a musicXMLAttributes
                    if err := decoder.DecodeElement(&a, &start); err != nil { return nil, err }
                    if a.Divisions > 0 { divisions = a.Divisions }
                    if a.Time.Beats > 0 && a.Time.BeatType > 0 {
                        meterLength = float64(a.Time.Beats)*4/float64(a.Time.BeatType)
                    }
                case "direction":
                    var d musicXMLDirection
                    if err := decoder.DecodeElement(&d, &start); err != nil { return nil, err }
                    if d.Sound.Tempo > 0 && d.Sound.Tempo < 1000 {
                        tempos = append(tempos, scoreTempo{at:beat+cursor,bpm:d.Sound.Tempo})
                    }
                case "sound":
                    var sound struct { Tempo float64 `xml:"tempo,attr"` }
                    if err := decoder.DecodeElement(&sound, &start); err != nil { return nil, err }
                    if sound.Tempo > 0 && sound.Tempo < 1000 {
                        tempos = append(tempos, scoreTempo{at:beat+cursor,bpm:sound.Tempo})
                    }
                case "backup", "forward":
                    var movement musicXMLMovement
                    if err := decoder.DecodeElement(&movement, &start); err != nil { return nil, err }
                    delta := float64(movement.Duration)/float64(divisions)
                    if start.Name.Local == "backup" { cursor = math.Max(0,cursor-delta) } else { cursor += delta }
                    maxCursor = math.Max(maxCursor,cursor)
                case "note":
                    var note musicXMLNote
                    if err := decoder.DecodeElement(&note, &start); err != nil { return nil, err }
                    if note.Grace != nil { continue } // grace performance needs ornament scheduling
                    duration := float64(note.Duration)/float64(divisions)
                    if duration < 0 { return nil, errors.New("MusicXML 包含负音符时值") }
                    onset := cursor
                    if note.Chord != nil { onset = lastNoteOnset } else { lastNoteOnset = onset }
                    if note.Rest == nil && duration > 0 {
                        pitch, err := scorePitch(note)
                        if err != nil { return nil, err }
                        hand := MidiHandRight
                        if note.Staff >= 2 || (len(score.Parts)>1 && partIndex>0 && note.Staff<=1) { hand = MidiHandLeft }
                        tieStart, tieStop := false, false
                        for _, tie := range note.Ties {
                            if tie.Type == "start" { tieStart = true }
                            if tie.Type == "stop" { tieStop = true }
                        }
                        voice := note.Voice
                        if voice == "" { voice = "1" }
                        beatNotes = append(beatNotes, scoreBeatNote{
                            start:beat+onset, end:beat+onset+duration,
                            pitch:pitch, track:partIndex, hand:hand,
                            tieStart:tieStart, tieStop:tieStop,
                            tieKey:strconv.Itoa(partIndex)+":"+strconv.Itoa(note.Staff)+":"+voice+":"+strconv.Itoa(pitch),
                        })
                    }
                    if note.Chord == nil {
                        cursor += duration
                        maxCursor = math.Max(maxCursor,cursor)
                    }
                }
            }
            length := math.Max(maxCursor, cursor)
            if length <= 0 { length = meterLength }
            beat += length
        }
        songEnd = math.Max(songEnd,beat)
    }
    if len(beatNotes) == 0 { return nil, errors.New("MusicXML 中没有可播放音符") }
    sort.SliceStable(tempos, func(i,j int) bool { return tempos[i].at < tempos[j].at })
    compact := make([]scoreTempo,0,len(tempos))
    for _, t := range tempos {
        if len(compact)>0 && math.Abs(compact[len(compact)-1].at-t.at)<0.000001 {
            compact[len(compact)-1] = t
        } else { compact = append(compact,t) }
    }
    sort.SliceStable(beatNotes, func(i,j int) bool {
        if beatNotes[i].start==beatNotes[j].start { return beatNotes[i].track<beatNotes[j].track }
        return beatNotes[i].start<beatNotes[j].start
    })
    // Merge notated ties to prevent an audible reattack at barlines.
    tied := map[string]int{}
    merged := make([]scoreBeatNote,0,len(beatNotes))
    for _, n := range beatNotes {
        if n.tieStop {
            if index, ok := tied[n.tieKey]; ok && math.Abs(merged[index].end-n.start)<0.0001 {
                merged[index].end = n.end
                if !n.tieStart { delete(tied,n.tieKey) }
                continue
            }
        }
        merged = append(merged,n)
        if n.tieStart { tied[n.tieKey] = len(merged)-1 }
    }
    events := make([]MidiEvent,0,len(merged)*2)
    for _, n := range merged {
        start, end := scoreTimeMs(n.start,compact), scoreTimeMs(n.end,compact)
        if end <= start { continue }
        tick := uint64(math.Round(n.start*480))
        endTick := uint64(math.Round(n.end*480))
        events = append(events,
            MidiEvent{Type:MidiEventNoteOn,TrackIndex:n.track,Channel:0,Hand:n.hand,Tick:tick,Ms:start,Note:n.pitch,Velocity:88},
            MidiEvent{Type:MidiEventNoteOff,TrackIndex:n.track,Channel:0,Hand:n.hand,Tick:endTick,Ms:end,Note:n.pitch})
    }
    for i := range events { events[i].FrontEvent = buildFrontEvent(events[i].Type, events[i].Hand) }
    sortMidiEvents(events)
    info, err := os.Stat(path)
    if err != nil { return nil, err }
    id, err := FileMD5(path)
    if err != nil { return nil, err }
    markers := make([]MidiMeasureMark,0,len(measureBeats))
    for i,b := range measureBeats {
        markers = append(markers,MidiMeasureMark{Index:i+1,Ms:scoreTimeMs(b,compact),StartBeat:b})
    }
    return &Midi{
        UserMidi:UserMidi{ID:id,Name:strings.TrimSuffix(filepath.Base(path),filepath.Ext(path)),
            Path:path,Size:info.Size(),ImportedAt:time.Now(),
            DurationMs:scoreTimeMs(songEnd,compact),TrackCount:len(score.Parts)},
        Format:1,PPQ:480,DurationTick:uint64(math.Round(songEnd*480)),
        Events:events,ScoreMeasures:markers,
    },nil
}
