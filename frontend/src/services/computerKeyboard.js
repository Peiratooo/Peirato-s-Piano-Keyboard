// Keep the note chosen on keydown until keyup, even if Shift or the mapping changes.
export function createComputerKeyboard({mapping, enabled, press, release}) {
    const physical = new Map()
    const notes = new Map()
    function keydown(event) {
        const id = event.code || event.key
        if (!enabled() || event.repeat || physical.has(id)) return
        const note = Number(mapping()[event.key])
        if (!Number.isInteger(note) || note < 0 || note > 127) return
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
    return {keydown, keyup, clear}
}
