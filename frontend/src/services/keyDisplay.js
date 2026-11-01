const degrees = [1, 0, 2, 0, 3, 4, 0, 5, 0, 6, 0, 7]

export const tonicOptions = ['C', 'D♭ / C♯', 'D', 'E♭ / D♯', 'E', 'F', 'G♭ / F♯', 'G', 'A♭ / G♯', 'A', 'B♭ / A♯', 'B']
    .map((label, value) => ({label: `${label} 调`, value}))

// Middle-register 1 is C4, D4, ... B4 according to the selected tonic.
export function numberedPitch(midi, tonic = 0) {
    const offset = Number(midi) - 60 - Number(tonic)
    return {pitch: degrees[((offset % 12) + 12) % 12], octave: 4 + Math.floor(offset / 12)}
}

export function keyHand(store, key) {
    if (store.midiPlaybackKey[key]) return 'right'
    if (store.midiPlaybackLeftKey[key]) return 'left'
    if (store.pressedKey[key]) return store.pressedKeyHands[key] || 'right'
    return store.midiHintLeftKey[key] && !store.midiHintKey[key] ? 'left' : 'right'
}

export function keyVisualClass(store, item) {
    const key = item.index
    const prefix = item.color === 'black' ? 'b' : 'w'
    if (store.pressedKey[key] || store.midiPlaybackKey[key] || store.midiPlaybackLeftKey[key]) {
        return `${prefix}${keyHand(store, key) === 'left' ? '-l' : ''}-active`
    }
    if (store.midiHintKey[key]) return `${prefix}-hint`
    if (store.midiHintLeftKey[key]) return `${prefix}-l-hint`
    return store.activeKey[key] ? `${prefix}-sustained` : ''
}
