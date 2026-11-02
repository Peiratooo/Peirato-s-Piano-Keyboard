// A MIDI file records performance timing rather than notation. This adapter
// produces an explicitly approximate, readable score without changing the
// original notes, playback timing or MIDI file on disk.
const STEP = ['C','C','D','D','E','F','F','G','G','A','A','B']
const ALTER = [0,1,0,1,0,0,1,0,1,0,1,0]
const DURATIONS = [
    [64,'whole'], [32,'half'], [16,'quarter'], [8,'eighth'],
    [4,'16th'], [2,'32nd'], [1,'64th'],
]
const GRID = 64 // four quarter notes, quantized to 64th notes

function xmlEscape(value) {
    return String(value).replace(/&/g, '&amp;').replace(/</g, '&lt;')
        .replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}
function clamp(n, min, max) { return Math.min(max, Math.max(min, n)) }
function pitchXML(midi) {
    const index = ((midi % 12) + 12) % 12
    return '<pitch><step>' + STEP[index] + '</step>'
        + (ALTER[index] ? '<alter>1</alter>' : '')
        + '<octave>' + (Math.floor(midi / 12) - 1) + '</octave></pitch>'
}
function restParts(duration, voice, staff) {
    let remaining = duration, output = ''
    for (const [value, type] of DURATIONS) {
        while (remaining >= value) {
            output += '<note><rest/><duration>' + value + '</duration>'
                + '<voice>' + voice + '</voice><type>' + type + '</type><staff>' + staff + '</staff></note>'
            remaining -= value
        }
    }
    return output
}
function noteParts(note, duration, voice, staff, chord) {
    let output = '', remaining = duration, partIndex = 0
    const pieces = []
    for (const [value, type] of DURATIONS) {
        while (remaining >= value) {
            pieces.push([value, type])
            remaining -= value
        }
    }
    for (const [value, type] of pieces) {
        const stop = note.fromBefore || partIndex > 0
        const start = note.toAfter || partIndex + 1 < pieces.length
        output += '<note>' + (chord && partIndex === 0 ? '<chord/>' : '')
            + pitchXML(note.note)
            + '<duration>' + value + '</duration>'
            + (stop ? '<tie type="stop"/>' : '')
            + (start ? '<tie type="start"/>' : '')
            + '<voice>' + voice + '</voice><type>' + type + '</type>'
            + '<staff>' + staff + '</staff>'
            + ((stop || start) ? '<notations>'
                + (stop ? '<tied type="stop"/>' : '')
                + (start ? '<tied type="start"/>' : '') + '</notations>' : '')
            + '</note>'
        partIndex++
    }
    return output
}
function getMeasures(timeline) {
    const duration = Number(timeline?.durationMs) || 0
    const markers = (timeline?.measures || []).filter(m => Number.isFinite(m.ms) && m.ms >= 0 && m.ms < duration)
        .slice().sort((a,b) => a.ms-b.ms)
    if (!markers.length || markers[0].ms > .01) markers.unshift({ms:0,index:1})
    const clean = []
    for (const mark of markers) if (!clean.length || mark.ms - clean[clean.length-1].start > 1)
        clean.push({start:mark.ms,index:mark.index})
    return clean.map((measure,i) => ({
        start:measure.start,
        end:i + 1 < clean.length ? clean[i+1].start : duration,
        number:measure.index || i+1,
    })).filter(m => m.end > m.start)
}
function collectSegments(notes, measure, staff) {
    const result = []
    for (const note of notes) {
        if (note.startMs >= measure.end || note.endMs <= measure.start) continue
        const hand = note.hand === 'left' ? 2 : 1
        if (hand !== staff) continue
        const start = clamp(Math.round((Math.max(note.startMs,measure.start)-measure.start)
            / (measure.end-measure.start)*GRID),0,GRID-1)
        const end = clamp(Math.round((Math.min(note.endMs,measure.end)-measure.start)
            / (measure.end-measure.start)*GRID),start+1,GRID)
        result.push({
            note:note.note, start, end,
            fromBefore:note.startMs < measure.start - .5,
            toAfter:note.endMs > measure.end + .5,
        })
    }
    return result.sort((a,b) => a.start-b.start || a.end-b.end || a.note-b.note)
}
function groupVoices(segments) {
    const voices = []
    for (const segment of segments) {
        let target = voices.find(voice => {
            const last = voice[voice.length-1]
            return last && last.start === segment.start && last.end === segment.end
        })
        if (!target) target = voices.find(voice => voice[voice.length-1].end <= segment.start)
        if (!target) {target = []; voices.push(target)}
        target.push(segment)
    }
    return voices
}
function renderVoice(voice, staff, number) {
    let xml = '', cursor = 0
    for (let i=0; i<voice.length; i++) {
        const note = voice[i]
        const chord = i>0 && voice[i-1].start===note.start && voice[i-1].end===note.end
        if (!chord && note.start>cursor) xml += restParts(note.start-cursor,number,staff)
        xml += noteParts(note,note.end-note.start,number,staff,chord)
        cursor=Math.max(cursor,note.end)
    }
    if (cursor<GRID) xml += restParts(GRID-cursor,number,staff)
    return xml
}
export function midiTimelineToMusicXML(timeline) {
    if (!timeline?.notes?.length) return ''
    const notes = timeline.notes.filter(n => Number.isInteger(n.note) && n.note>=0 && n.note<=127
        && Number.isFinite(n.startMs) && Number.isFinite(n.endMs) && n.endMs>n.startMs)
    const measures = getMeasures(timeline)
    if (!measures.length || !notes.length) return ''
    let xml = '<?xml version="1.0" encoding="utf-8"?>'
        + '<!DOCTYPE score-partwise  PUBLIC "-//Recordare//DTD MusicXML 4.0 Partwise//EN" "http://www.musicxml.org/dtds/partwise.dtd">'
        + '<score-partwise version="4.0"><work><work-title>MIDI 自动排版</work-title></work>'
        + '<movement-title>MIDI 自动排版</movement-title>'
        + '<identification><creator type="software">PP Piano Keyboard</creator></identification>'
        + '<defaults><scaling><millimeters>7</millimeters><tenths>40</tenths></scaling></defaults>'
        + '<part-list><score-part id="P1"><part-name>Piano</part-name>'
        + '<part-abbreviation>Pno</part-abbreviation><score-instrument id="I1">'
        + '<instrument-name>Piano</instrument-name></score-instrument></score-part></part-list><part id="P1">'
    for (let i=0; i<measures.length; i++) {
        const measure=measures[i]
        xml += '<measure number="' + xmlEscape(i+1) + '">'
        if (i===0) xml += '<attributes><divisions>16</divisions>'
            + '<key><fifths>0</fifths></key><time><beats>4</beats><beat-type>4</beat-type></time>'
            + '<staves>2</staves><clef number="1"><sign>G</sign><line>2</line></clef>'
            + '<clef number="2"><sign>F</sign><line>4</line></clef></attributes>'
        let first=true, voiceNumber=1
        for (const staff of [1,2]) {
            let voices=groupVoices(collectSegments(notes,measure,staff))
            if (!voices.length) voices=[[]]
            for (const voice of voices) {
                if (!first) xml+='<backup><duration>' + GRID + '</duration></backup>'
                xml+=renderVoice(voice,staff,voiceNumber++)
                first=false
            }
        }
        xml+='</measure>'
    }
    return xml+'</part></score-partwise>'
}
