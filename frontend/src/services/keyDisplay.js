const degrees = [1, 0, 2, 0, 3, 4, 0, 5, 0, 6, 0, 7]

export const tonicOptions = ['C', 'D♭ / C♯', 'D', 'E♭ / D♯', 'E', 'F', 'G♭ / F♯', 'G', 'A♭ / G♯', 'A', 'B♭ / A♯', 'B']
    .map((label, value) => ({label: `${label} 调`, value}))

// Middle-register 1 is C4, D4, ... B4 according to the selected tonic.
export function numberedPitch(midi, tonic = 0) {
    const offset = Number(midi) - 60 - Number(tonic)
    return {pitch: degrees[((offset % 12) + 12) % 12], octave: 4 + Math.floor(offset / 12)}
}

export function keyVisualClass(store, item) {
    const key = item.index
    const prefix = item.color === 'black' ? 'b' : 'w'
    if (store.pressedKey[key] || store.midiPlaybackKey[key] || store.midiHintKey[key]) return `${prefix}-active`
    if (store.midiPlaybackLeftKey[key] || store.midiHintLeftKey[key]) return `${prefix}-l-active`
    return store.activeKey[key] ? `${prefix}-sustained` : ''
}
