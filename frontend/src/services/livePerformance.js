// A shared, non-reactive event model keeps real input independent of playback
// highlights. It continues collecting while the keyboard-only view is visible.
export const LIVE_HISTORY_MS = 6000
const MAX_VOICES = 4096
const voices = new Map()
const listeners = new Set()
let clockOffset = null
let lastTimestamp = 0
let lastSequence = 0

function localNow() {
    return globalThis.performance?.now() ?? Date.now()
}

function backendNow(now = localNow()) {
    return clockOffset === null ? 0 : Math.max(lastTimestamp, now - clockOffset)
}

function prune(now) {
    const oldest = now - LIVE_HISTORY_MS
    for (const [id, voice] of voices) {
        if (voice.endedAt !== null && voice.endedAt < oldest) voices.delete(id)
    }
    if (voices.size <= MAX_VOICES) return
    // Prefer keeping notes still physically held when a very dense input stream
    // reaches the hard cap; Map insertion order is the note-on order.
    for (const [id, voice] of voices) {
        if (voices.size <= MAX_VOICES) break
        if (voice.releasedAt !== null) voices.delete(id)
    }
    while (voices.size > MAX_VOICES) voices.delete(voices.keys().next().value)
}

export function acceptLivePerformance(signal) {
    if (!signal || !['noteOn', 'noteOff', 'noteEnd', 'clear'].includes(signal.type)) return
    const timestamp = Number(signal.timestampMs)
    const sequence = Number(signal.sequence)
    if (!Number.isFinite(timestamp) || timestamp < 0 || !Number.isSafeInteger(sequence) || sequence <= lastSequence) return
    if (signal.type !== 'clear' && (!signal.id || !Number.isInteger(signal.note) || signal.note < 0 || signal.note > 127)) return
    if (clockOffset === null) clockOffset = localNow() - timestamp
    lastTimestamp = Math.max(lastTimestamp, timestamp)
    lastSequence = sequence

    if (signal.type === 'clear') {
        voices.clear()
    } else if (signal.type === 'noteOn') {
        voices.set(signal.id, {
            id: signal.id,
            source: signal.source,
            channel: signal.channel,
            note: signal.note,
            velocity: Math.max(1, Math.min(127, Number(signal.velocity) || 80)),
            startedAt: timestamp,
            releasedAt: null,
            endedAt: null,
        })
    } else {
        const voice = voices.get(signal.id)
        if (voice) {
            if (signal.type === 'noteOff') {
                voice.releasedAt = Math.max(voice.startedAt, timestamp)
                if (!signal.sustained) voice.endedAt = voice.releasedAt
            } else {
                voice.releasedAt ??= Math.max(voice.startedAt, timestamp)
                voice.endedAt = Math.max(voice.releasedAt, timestamp)
            }
        }
    }
    // Prune here as well as on render: simple mode has no animation frames.
    prune(backendNow())
    for (const listener of listeners) listener(signal)
}

export function subscribeLivePerformance(listener) {
    listeners.add(listener)
    return () => listeners.delete(listener)
}

export function getLivePerformanceFrame(now = localNow()) {
    const nowMs = backendNow(now)
    prune(nowMs)
    return {nowMs, notes: Array.from(voices.values()), historyMs: LIVE_HISTORY_MS}
}
