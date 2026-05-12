// 和弦识别保持轻量：MIDI note -> 12 音级 -> 查 ChordNames.json。
// 这里不要引入太重的音乐理论库，后续要支持转位 / slash chord 时再逐步增强。
export const NOTE_NAME = {
    0: 'A',
    1: 'Bb',
    2: 'B',
    3: 'C',
    4: 'Db',
    5: 'D',
    6: 'Eb',
    7: 'E',
    8: 'F',
    9: 'Gb',
    10: 'G',
    11: 'Ab',
}

export function midiToPitchClass(midi) {
    return ((Number(midi) - 9) % 12 + 12) % 12
}

export function getPressedMidiNotes(pressedKey = {}) {
    return Object.keys(pressedKey)
        .filter((key) => pressedKey[key])
        .map(Number)
        .sort((a, b) => a - b)
}

export function getPitchClasses(midiNotes = []) {
    const result = []
    for (const midi of midiNotes) {
        const pitchClass = midiToPitchClass(midi)
        if (!result.includes(pitchClass)) {
            result.push(pitchClass)
        }
    }
    return result.sort((a, b) => a - b)
}

function normalizeRoot(root = '') {
    return String(root).replace('♭', 'b').replace('♯', '#')
}

function sameRoot(root, noteName) {
    return normalizeRoot(root) === normalizeRoot(noteName)
}

function selectBestChord(candidates = {}, bassNoteName = '') {
    const roots = Object.keys(candidates)
    if (roots.length === 0) return null
    if (roots.length === 1) return candidates[roots[0]]

    // 优先选择低音同名的候选，避免转位时显示完全不相关的根音。
    const bassRoot = roots.find((root) => sameRoot(root, bassNoteName))
    if (bassRoot) return candidates[bassRoot]

    // 再按自然音名兜底，保证显示稳定。
    for (const root of ['C', 'D', 'E', 'F', 'G', 'A', 'B']) {
        if (candidates[root]) return candidates[root]
    }
    return candidates[roots[0]]
}

export function detectChord(pressedKey = {}, chordNames = {}) {
    const midiNotes = getPressedMidiNotes(pressedKey)
    if (midiNotes.length === 0) return {}

    const pitchClasses = getPitchClasses(midiNotes)
    if (pitchClasses.length === 1) {
        return {type: 'note', note: NOTE_NAME[pitchClasses[0]]}
    }

    const chordKey = pitchClasses.join(' ')
    const candidates = chordNames[chordKey]
    if (!candidates) {
        return {
            type: 'unknown',
            notes: pitchClasses.map((pitchClass) => NOTE_NAME[pitchClass]),
        }
    }

    const bassPitchClass = midiToPitchClass(midiNotes[0])
    const chord = selectBestChord(candidates, NOTE_NAME[bassPitchClass])
    return chord ? {type: 'chord', bassNote: NOTE_NAME[bassPitchClass], ...chord} : {}
}
