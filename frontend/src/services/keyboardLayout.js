// Both the keybed and the note canvas use these pixel coordinates. MIDI pitch
// lives in `index` in keyboard_config.json; its `note` field is a display label.
export function createKeyboardLayout(keys, width) {
    const availableWidth = Math.max(0, Number(width) || 0)
    const whiteCount = keys.filter(key => key.color !== 'black').length
    if (!keys.length || !availableWidth || !whiteCount) return []

    const whiteWidth = availableWidth / whiteCount
    const blackWidth = Math.round(whiteWidth * 0.62)
    // Share pixel-aligned edges instead of rounding each key's left and width
    // independently in the browser, which can expose dark subpixel gaps.
    const whiteEdge = index => index === whiteCount ? availableWidth : Math.round(index * whiteWidth)
    let whiteIndex = 0

    return keys.map(key => {
        const isBlack = key.color === 'black'
        const x = isBlack
            ? Math.round(Math.max(0, Math.min(availableWidth - blackWidth, whiteEdge(whiteIndex) - blackWidth / 2)))
            : whiteEdge(whiteIndex)
        const keyWidth = isBlack ? blackWidth : whiteEdge(++whiteIndex) - x
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
