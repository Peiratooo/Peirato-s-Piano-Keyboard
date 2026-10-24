<template>
    <!-- Keep one stable display node. Rapid note changes no longer create
         overlapping enter/leave elements or repeatedly restart animations. -->
    <div v-if="displayChord.type" class="chord-display" aria-live="off">
        <div class="chord" v-if="displayChord.type === 'chord'">
            <div class="top">
                <div class="left">{{ displayChord.chord }}</div>
                <div class="right">
                    <div class="symbols" v-if="displayChord.alternativeSymbols && displayChord.alternativeSymbols.length > 0">
                        <div class="symbol" v-for="item in displayChord.alternativeSymbols.slice(0, 3)" :key="item">
                            {{ item }}
                        </div>
                    </div>
                    <div class="synonym-notes" v-else>
                        <div class="notes">{{ displayChord.notes?.join(' ') }}</div>
                    </div>
                    <div class="cn">{{ displayChord.chinese }}</div>
                </div>
            </div>
            <div class="bottom">
                <div class="en">{{ displayChord.name }}</div>
            </div>
        </div>

        <div class="note" v-else-if="displayChord.type === 'note'">
            {{ displayChord.note }}
        </div>

        <div class="note unknown" v-else-if="displayChord.type === 'unknown'">
            <span v-for="(item,index) in displayChord.notes" :key="index">
                {{ item }}
            </span>
        </div>
    </div>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, shallowRef, watch} from 'vue'
import {detectChord} from '../services/chordEngine'

const store = inject('store')

// Recognition follows physical key presses, not sustain-held visual notes.
// A small semantic hold preserves a settled chord through staggered finger
// releases/retriggers without feeding stale notes back into the piano.
const rawChord = computed(() => detectChord(store.pressedKey, store.chordsName))
const displayChord = shallowRef({})
const signature = chord => JSON.stringify(chord ?? {})
let committedSignature = signature({})
let pendingSignature = ''
let pendingTimer = null

function cancelPending() {
    if (pendingTimer !== null) clearTimeout(pendingTimer)
    pendingTimer = null
    pendingSignature = ''
}
function settleDelay(next) {
    if (!next?.type) return 170 // Short gaps and final release
    if (displayChord.value.type === 'chord' && next.type !== 'chord') return 155
    if (next.type === 'unknown') return 125
    if (next.type === 'chord') return 85
    return 75 // Single note
}

watch(rawChord, next => {
    const nextSignature = signature(next)
    if (nextSignature === committedSignature) {
        cancelPending()
        return
    }
    // A second backend/input event yielding the same musical identity must
    // not restart the settling interval indefinitely.
    if (nextSignature === pendingSignature) return
    cancelPending()
    pendingSignature = nextSignature
    pendingTimer = setTimeout(() => {
        pendingTimer = null
        pendingSignature = ''
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

.chord-display {
    display: flex;
    align-items: flex-end;
    min-width: 76px;
    min-height: 50px;
}

.chord, .note {
    background-color: #eeeeee55;
    border-radius: 4px;
    display: flex;
    flex-direction: column;
    padding: 8px;
    backdrop-filter: blur(4px);
    box-shadow: 0 0 6px 0 rgba(0, 0, 0, 0.25);
    gap: 6px;

    .top {
        display: flex;
        height: 80%;
        gap: 8px;

        .left {
            width: 100%;
            font-size: 36px;
            white-space: nowrap;
        }

        .right {
            padding: 3px 0;
            justify-content: space-between;
            display: flex;
            align-items: flex-end;
            flex-direction: column;
        }
    }
}

.note {
    font-size: 32px;
    padding: 12px 16px;
    color: #333333;
}

.unknown {
    font-size: 18px;
    display: flex;
    flex-direction: row;
    white-space: nowrap;
    gap: 4px;
}

.synonym-notes {
    font-size: 12px;
    display: flex;
    justify-content: space-between;
}

.symbols {
    display: flex;
    font-size: 12px;
    gap: 8px;
    color: #333333;
    white-space: nowrap;
}

.en, .cn {
    font-size: 12px;
    white-space: nowrap;
}
</style>
