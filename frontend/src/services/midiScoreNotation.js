// Generate a rhythmically readable display without collapsing fast attacks. Playback retains the source
// timing; distinct 32nd/64th attacks retain separate notated columns.
import {Key, Note} from 'tonal'

const DIVISIONS = 16
const RELEASE_TOLERANCE = DIVISIONS / 4
const DURATIONS = [
    {value: 64, type: 'whole'}, {value: 48, type: 'half', dot: true},
    {value: 32, type: 'half'}, {value: 24, type: 'quarter', dot: true},
    {value: 16, type: 'quarter'}, {value: 12, type: 'eighth', dot: true},
    {value: 8, type: 'eighth'}, {value: 6, type: '16th', dot: true},
    {value: 4, type: '16th'}, {value: 2, type: '32nd'}, {value: 1, type: '64th'},
]

function spellingFor(fifths, mode = 'major') {
    const major = Key.majorKey(Key.majorTonicFromKeySignature(fifths))
    const spelling = new Map(major.scale.map(name => {
        const note = Note.get(name)
        return [note.chroma, note]
    }))
    if (mode === 'minor') for (const name of Key.minorKey(major.minorRelative).harmonic.scale) {
        const note = Note.get(name)
        if (!spelling.has(note.chroma)) spelling.set(note.chroma, note)
    }
    return spelling
}

function pitchXML(midi, key) {
    const note = key.spelling.get(midi % 12)
        || Note.get(key.fifths < 0 ? Note.fromMidi(midi) : Note.fromMidiSharps(midi))
    const natural = Note.get(note.letter).chroma
    const octave = Math.round((midi - natural - note.alt) / 12) - 1
    return '<pitch><step>' + note.letter + '</step>'
        + (note.alt ? '<alter>' + note.alt + '</alter>' : '') + '<octave>' + octave + '</octave></pitch>'
}

function keyXML(key) {
    return '<key><fifths>' + key.fifths + '</fifths><mode>' + key.mode + '</mode></key>'
}

// Split at readable rhythmic boundaries instead of spelling arbitrary release
// times as chains of 32nd/64th notes. Dotted notes stay single symbols.
function durationParts(start, end, keys, beatUnits) {
    const parts = []
    for (let cursor = start; cursor < end;) {
        const boundary = keys.find(key => key.at > cursor)?.at ?? end
        const part = DURATIONS.find(item => item.value <= Math.min(end, boundary) - cursor
            && cursor % Math.min(beatUnits, item.dot ? item.value / 1.5 : item.value) === 0)
        parts.push({...part, at: cursor})
        cursor += part.value
    }
    return parts
}

function restXML(start, end, voice, staff, keys, beatUnits) {
    return durationParts(start, end, keys, beatUnits).map(part => {
        const change = voice === 1 && part.at > 0 && keys.find(key => key.at === part.at)
        return (change ? '<attributes>' + keyXML(change) + '</attributes>' : '')
            + '<note><rest/><duration>' + part.value + '</duration><voice>' + voice
            + '</voice><type>' + part.type + '</type>' + (part.dot ? '<dot/>' : '')
            + '<staff>' + staff + '</staff></note>'
    }).join('')
}

function renderVoice(groups, staff, voice, grid, keys, beatUnits) {
    if (!groups.length && keys.length === 1) return '<note><rest measure="yes"/><duration>' + grid
        + '</duration><voice>' + voice + '</voice><staff>' + staff + '</staff></note>'
    let xml = '', cursor = 0
    for (const group of groups) {
        if (group.start > cursor) xml += restXML(cursor, group.start, voice, staff, keys, beatUnits)
        const parts = durationParts(group.start, group.end, keys, beatUnits)
        for (let index = 0; index < parts.length; index++) {
            const part = parts[index]
            const key = keys.findLast(key => key.at <= part.at)
            if (voice === 1 && part.at > 0 && key.at === part.at) xml += '<attributes>' + keyXML(key) + '</attributes>'
            for (let chord = 0; chord < group.notes.length; chord++) {
                const note = group.notes[chord]
                const stop = note.fromBefore || index > 0
                const start = note.toAfter || index + 1 < parts.length
                xml += '<note>' + (chord ? '<chord/>' : '') + pitchXML(note.note, key)
                    + '<duration>' + part.value + '</duration>'
                    + (stop ? '<tie type="stop"/>' : '') + (start ? '<tie type="start"/>' : '')
                    + '<voice>' + voice + '</voice><type>' + part.type + '</type>'
                    + (part.dot ? '<dot/>' : '') + '<staff>' + staff + '</staff>'
                    + ((stop || start) ? '<notations>' + (stop ? '<tied type="stop"/>' : '')
                        + (start ? '<tied type="start"/>' : '') + '</notations>' : '') + '</note>'
            }
        }
        cursor = group.end
    }
    if (cursor < grid) xml += restXML(cursor, grid, voice, staff, keys, beatUnits)
    return xml
}

export function midiTimelineToScore(timeline) {
    if (!timeline?.notes?.length) return {xml: '', positions: []}
    const keySignatures = timeline.keySignatures.map(key => ({...key, at: Math.round(key.startBeat * DIVISIONS), spelling: spellingFor(key.fifths, key.mode)}))
    const neutral = {at: 0, fifths: 0, mode: 'major', spelling: spellingFor(0)}
    // Layered MIDI tracks often duplicate the same piano pitch at the exact
    // same onset. Notate that pitch once per hand, retaining its longest hold.
    const attacks = new Map()
    for (const source of timeline.notes) {
        const id = `${source.hand}:${source.startBeat}:${source.note}`
        const previous = attacks.get(id)
        if (previous) {
            previous.endBeat = Math.max(previous.endBeat, source.endBeat)
            previous.endMs = Math.max(previous.endMs, source.endMs)
        } else attacks.set(id, {...source})
    }
    const notes = [...attacks.values()].map(note => ({...note,
        start: Math.round(note.startBeat * DIVISIONS),
        end: Math.max(Math.round(note.startBeat * DIVISIONS) + 1, Math.round(note.endBeat * DIVISIONS)),
    })).sort((a, b) => a.start - b.start || a.note - b.note)
    const measures = timeline.measures.map((mark, index, marks) => {
        const start = Math.round(mark.startBeat * DIVISIONS)
        const numerator = mark.numerator || 4, denominator = mark.denominator || 4
        const fullGrid = Math.max(1, Math.round(numerator * 4 / denominator * DIVISIONS))
        const grid = index + 1 < marks.length
            ? Math.min(fullGrid, Math.round(marks[index + 1].startBeat * DIVISIONS) - start) : fullGrid
        const currentKey = keySignatures.findLast(key => key.at <= start) || neutral
        const keys = [{...currentKey, at: 0}, ...keySignatures.filter(key => key.at > start && key.at < start + grid).map(key => ({...key, at: key.at - start}))]
        const beatUnits = DIVISIONS * 4 / denominator * (denominator === 8 && numerator >= 6 && numerator % 3 === 0 ? 3 : 1)
        return {...mark, start, numerator, denominator, grid, keys, beatUnits, staves: [[], []]}
    })
    if (!measures.length) return {xml: '', positions: []}
    const positions = measures.map(mark => ({beat: mark.startBeat, ms: mark.ms, attack: false}))
    for (const note of notes) {
        positions.push({beat: note.start / DIVISIONS, ms: note.startMs, attack: true},
            {beat: note.end / DIVISIONS, ms: note.endMs, attack: false})
        let low = 0, high = measures.length - 1
        while (low < high) {
            const middle = (low + high + 1) >> 1
            if (measures[middle].start <= note.start) low = middle
            else high = middle - 1
        }
        for (let index = low; index < measures.length && measures[index].start < note.end; index++) {
            const measure = measures[index]
            const end = index + 1 < measures.length ? measures[index + 1].start : measure.start + measure.grid
            if (end <= note.start) continue
            measure.staves[note.hand === 'left' ? 1 : 0].push({note: note.note, track: note.trackIndex,
                start: Math.max(0, note.start - measure.start), end: Math.min(end, note.end) - measure.start,
                fromBefore: note.start < measure.start, toAfter: note.end > end})
        }
    }
    let xml = '<?xml version="1.0" encoding="utf-8"?><score-partwise version="4.0">'
        + '<part-list><score-part id="P1"><part-name>Piano</part-name></score-part></part-list><part id="P1">'
    for (let index = 0; index < measures.length; index++) {
        const measure = measures[index]
        xml += '<measure number="' + (index + 1) + '">'
        if (index === 0) xml += '<attributes><divisions>' + DIVISIONS + '</divisions>'
            + (keySignatures.length ? keyXML(measure.keys[0]) : '') + '<time><beats>' + measure.numerator + '</beats>'
            + '<beat-type>' + measure.denominator + '</beat-type></time><staves>2</staves>'
            + '<clef number="1"><sign>G</sign><line>2</line></clef>'
            + '<clef number="2"><sign>F</sign><line>4</line></clef></attributes>'
        else {
            const previous = measures[index - 1]
            const changedKey = previous.keys.at(-1).fifths !== measure.keys[0].fifths
                || previous.keys.at(-1).mode !== measure.keys[0].mode
            const changedTime = previous.numerator !== measure.numerator || previous.denominator !== measure.denominator
            if (changedKey || changedTime) xml += '<attributes>'
                + (changedKey ? keyXML(measure.keys[0]) : '')
                + (changedTime ? '<time><beats>' + measure.numerator + '</beats><beat-type>'
                    + measure.denominator + '</beat-type></time>' : '') + '</attributes>'
        }
        let voiceNumber = 1
        for (let staff = 1; staff <= 2; staff++) {
            const segments = measure.staves[staff - 1].sort((a, b) => a.start - b.start || b.end - a.end || a.note - b.note)
            const voices = []
            for (const segment of segments) {
                const chord = voices.map(voice => voice.at(-1)).find(last => last.start === segment.start
                    && Math.abs(last.end - segment.end) <= Math.min(RELEASE_TOLERANCE,
                        (last.end - last.start) / 4, (segment.end - segment.start) / 4))
                if (chord) {
                    if (!chord.notes.some(note => note.note === segment.note)) chord.notes.push(segment)
                    continue
                }
                let voice = null, bestCost = Infinity
                for (const candidate of voices) {
                    const last = candidate.at(-1), overlap = last.end - segment.start
                    const tolerance = Math.min(RELEASE_TOLERANCE, (last.end - last.start) / 4)
                    if (last.start >= segment.start || overlap > tolerance
                        || overlap > 0 && last.notes.some(note => note.toAfter)) continue
                    // Prefer the original track and nearby pitches, so a melody
                    // doesn't bounce between whichever voice happens to be free.
                    const cost = (last.track === segment.track ? 0 : 24)
                        + Math.min(...last.notes.map(note => Math.abs(note.note - segment.note)))
                    if (cost < bestCost) {voice = candidate; bestCost = cost}
                }
                if (!voice) {voice = []; voices.push(voice)}
                const previous = voice.at(-1)
                if (previous && Math.abs(segment.start - previous.end)
                    <= Math.min(RELEASE_TOLERANCE, (previous.end - previous.start) / 4)) previous.end = segment.start
                voice.push({start: segment.start, end: segment.end, track: segment.track, notes: [segment]})
            }
            for (const voice of voices) {
                const last = voice.at(-1)
                if (last && measure.grid - last.end <= Math.min(RELEASE_TOLERANCE, (last.end - last.start) / 4)) last.end = measure.grid
            }
            if (!voices.length) voices.push([])
            for (const voice of voices) {
                if (voiceNumber > 1) xml += '<backup><duration>' + measure.grid + '</duration></backup>'
                xml += renderVoice(voice, staff, voiceNumber++, measure.grid, measure.keys, measure.beatUnits)
            }
        }
        xml += '</measure>'
    }
    return {xml: xml + '</part></score-partwise>', positions}
}
