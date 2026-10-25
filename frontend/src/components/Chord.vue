<template>
    <div v-if="displayChord.type || pressedMidiNotes.length" class="chord-display" :class="{compact}" aria-live="off">
        <div class="chord-heading">
            <div v-if="displayChord.type === 'chord'" class="chord-symbol">{{ displayChord.chord }}</div>
            <div v-else-if="displayChord.type === 'note'" class="chord-symbol">{{ displayChord.note }}</div>
            <div v-else-if="displayChord.type === 'unknown'" class="chord-symbol unknown">{{ displayChord.notes.join(' · ') }}</div>
            <div v-if="displayChord.type === 'chord'" class="chord-description">
                <div class="description-top">
                    <span v-if="displayChord.chinese" class="cn">{{ displayChord.chinese }}</span>
                    <span v-if="displayChord.alternativeSymbols?.length" class="symbols">{{ displayChord.alternativeSymbols.slice(0, 3).join(' · ') }}</span>
                </div>
                <div class="en">{{ displayChord.name }}</div>
            </div>
        </div>
        <div class="current-keys" aria-label="当前按键" :title="pressedMidiNotes.map(noteName).join(' · ')">
            <span class="current-keys-label">当前按键</span>
            <span v-for="note in pressedMidiNotes" :key="note" class="pressed-note">{{ noteName(note) }}</span>
            <span v-if="!pressedMidiNotes.length" class="released">—</span>
        </div>
    </div>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, shallowRef, watch} from 'vue'
import {detectChord, getPressedMidiNotes, midiToPitchClass, NOTE_NAME} from '../services/chordEngine'

defineProps({compact: {type: Boolean, default: false}})
const store = inject('store')
const pressedMidiNotes = computed(() => getPressedMidiNotes(store.pressedKey))
const noteName = note => `${NOTE_NAME[midiToPitchClass(note)]}${Math.floor(note / 12) - 1}`

// Recognition follows physical key presses, not sustain-held visual notes.
// A small semantic hold preserves a settled chord through staggered finger
// releases/retriggers without feeding stale notes back into the piano.
const rawChord = computed(() => detectChord(store.pressedKey, store.chordsName))
const displayChord = shallowRef({})
const signature = chord => JSON.stringify(chord ?? {})
let committedSignature = signature({})
let pendingSignature = ''
let pendingChord = null
let pendingAttack = false
let lastNoteCount = 0
let pendingTimer = null

function cancelPending() {
    if (pendingTimer !== null) clearTimeout(pendingTimer)
    pendingTimer = null
    pendingSignature = ''
    pendingChord = null
    pendingAttack = false
}
function settleDelay(next) {
    if (!next.type) return 240 // Keep the frame through short release/retrigger gaps.
    if (displayChord.value.type === 'chord' && next.type !== 'chord') return 200
    return 100 // Let staggered chord attacks settle before replacing the title.
}

watch(rawChord, next => {
    const noteCount = pressedMidiNotes.value.length
    const releasing = noteCount < lastNoteCount
    lastNoteCount = noteCount
    // A short chord may be released before the settling timer runs. Keep the
    // complete attack, including a seventh lost by the first finger release.
    if (pendingChord && pendingAttack && releasing) {
        committedSignature = pendingSignature
        displayChord.value = pendingChord
    }
    const nextSignature = signature(next)
    if (nextSignature === committedSignature) {
        cancelPending()
        return
    }
    // A second backend/input event yielding the same musical identity must
    // not restart the settling interval indefinitely.
    if (nextSignature === pendingSignature) return
    cancelPending()
    // Show the first note together with its live key row, never an empty title.
    if (!displayChord.value.type && next.type) {
        committedSignature = nextSignature
        displayChord.value = next
        return
    }
    pendingSignature = nextSignature
    pendingChord = next
    pendingAttack = !releasing
    pendingTimer = setTimeout(() => {
        pendingTimer = null
        pendingSignature = ''
        pendingChord = null
        pendingAttack = false
        // Ignore callbacks superseded by more recent input.
        if (signature(rawChord.value) !== nextSignature) return
        committedSignature = nextSignature
        displayChord.value = next
    }, settleDelay(next))
}, {immediate: true, flush: 'sync'})

onBeforeUnmount(cancelPending)
</script>

<style lang="scss" scoped>
* { user-select: none; }

.chord-display { display: flex; flex-direction: column; gap: 7px; color: inherit; }
.chord-heading { display: flex; align-items: center; gap: 16px; height: 36px; min-width: 0; }
.chord-symbol { flex: 0 1 auto; overflow: hidden; text-overflow: ellipsis; font-size: 30px; line-height: 36px; font-weight: 500; white-space: nowrap; }
.chord-description { flex: 1; min-width: 0; }
.description-top { display: flex; align-items: baseline; gap: 8px; }
.cn { flex-shrink: 0; font-size: 13px; line-height: 18px; }
.symbols { font-size: 10px; opacity: .6; }
.description-top, .symbols, .en { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.en { font-size: 11px; line-height: 16px; opacity: .65; }
.unknown { font-size: 20px; }
.current-keys { display: flex; align-items: center; gap: 5px; height: 20px; overflow: hidden; white-space: nowrap; font-size: 11px; }
.current-keys-label { flex-shrink: 0; opacity: .6; margin-right: 3px; }
.pressed-note { flex-shrink: 0; padding: 1px 5px; line-height: 14px; border: 1px solid color-mix(in srgb, currentColor 20%, transparent); border-radius: 4px; background: color-mix(in srgb, currentColor 6%, transparent); font-variant-numeric: tabular-nums; }
.released { opacity: .45; }
.compact { gap: 4px; }
.compact .chord-heading { height: 30px; gap: 12px; }
.compact .chord-symbol { font-size: 25px; line-height: 30px; }
.compact .unknown { font-size: 18px; }
.compact .cn { font-size: 12px; line-height: 16px; }
.compact .en { font-size: 10px; line-height: 14px; }
.compact .current-keys { font-size: 10px; height: 18px; }
</style>
