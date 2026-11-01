const keyboardLabelSkipKeys = new Set(['k', 'K', 'l', 'L', ';', "'", '\\', '|', '`', '~', '[', '{', ']', '}', 'p', 'P'])

export function setKeyboardMappingLabels(keyboard, mapping) {
    const reverse = {}
    for (const [key, note] of Object.entries(mapping)) {
        if (!keyboardLabelSkipKeys.has(key) && !(note in reverse)) reverse[note] = key === ' ' ? 'Space' : key
    }
    for (const item of keyboard) item.keyboard = reverse[item.index] || ''
}

export function isPianoEditableTarget(target) {
    return !!(target?.isContentEditable || target?.closest?.(
        'input, textarea, select, [contenteditable=""], [contenteditable="true"], '
        + '[role="textbox"], [role="combobox"], [role="slider"], [role="spinbutton"], [role="switch"]',
    ))
}

export function isPianoControlTarget(target) {
    return isPianoEditableTarget(target)
        || !!target?.closest?.('button, [role="button"], [data-piano-control]')
}

// Keep the note chosen on keydown until keyup, even if focus, Shift or the mapping changes.
export function createComputerKeyboard({mapping, enabled, press, release}) {
    const physical = new Map()
    const notes = new Map()
    function keydown(event) {
        const id = event.code || event.key
        if (!enabled() || event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey
            || isPianoControlTarget(event.target) || event.repeat || physical.has(id)) return
        const note = Number(mapping()[event.key])
        if (!Number.isInteger(note) || note < 0 || note > 127) return
        event.preventDefault()
        physical.set(id, note)
        const count = notes.get(note) || 0
        notes.set(note, count + 1)
        if (!count) press(note)
    }
    function keyup(event) {
        const id = event.code || event.key
        if (!physical.has(id)) return
        const note = physical.get(id)
        physical.delete(id)
        const count = notes.get(note) - 1
        if (count) notes.set(note, count)
        else { notes.delete(note); release(note) }
    }
    function clear() {
        for (const note of notes.keys()) release(note)
        physical.clear()
        notes.clear()
    }
    function focusin(event) {
        if (event.target?.closest?.('[data-preserve-piano-holds]')) return
        if (isPianoEditableTarget(event.target)) clear()
    }
    return {keydown, keyup, clear, focusin}
}
