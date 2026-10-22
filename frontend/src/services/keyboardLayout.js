// Both the keybed and the note canvas use these pixel coordinates. MIDI pitch
// lives in `index` in keyboard_config.json; its `note` field is a display label.
export function createKeyboardLayout(keys, width) {
    const availableWidth = Math.max(0, Number(width) || 0)
    const whiteCount = keys.filter(key => key.color !== 'black').length
    if (!keys.length || !availableWidth || !whiteCount) return []

    const whiteWidth = availableWidth / whiteCount
    const blackWidth = whiteWidth * 0.62
    let whiteIndex = 0

    return keys.map(key => {
        const isBlack = key.color === 'black'
        const keyWidth = isBlack ? blackWidth : whiteWidth
        const x = isBlack
            ? Math.max(0, Math.min(availableWidth - keyWidth, whiteIndex * whiteWidth - keyWidth / 2))
            : whiteIndex++ * whiteWidth
        return {
            ...key,
            pitchName: key.note,
            note: Number(key.index),
            x,
            width: keyWidth,
            isBlack,
        }
    })
}
