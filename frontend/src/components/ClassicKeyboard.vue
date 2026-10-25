<template>
    <div
        class="classic-keyboard"
        :class="{'has-note-feedback': hasNoteFeedback}"
        ref="keybed"
        @mouseup="releaseMouseKey"
        @mouseleave="releaseMouseKey"
    >
        <div
            class="key"
            v-for="item in visibleKeys"
            :class="[item.color, item.note, getKeyClass(item)]"
            :style="keyStyle(item.index)"
            :key="item.index"
            :data-midi-note="item.index"
            @mouseenter="enterKey($event, item.index)"
            @mousedown.prevent="pressKey($event, item.index)"
            @mouseup="releaseKey(item.index)"
        >
            <span v-if="hintHand(item.index)" class="practice-dot" :class="{'is-left': hintHand(item.index) === 'left'}" aria-hidden="true"></span>
            <div class="label" v-if="store.config.keyLabel !== '' && item[store.config.keyLabel]" :class="{'pitch-label': store.config.keyLabel === 'pitch'}">
                <template v-if="store.config.keyLabel === 'pitch'">
                    <span v-if="pitchMarkCount(item) > 0 && pitchMarksAbove(item)" class="pitch-marks above" aria-hidden="true">
                        <span v-for="mark in pitchMarkCount(item)" :key="`above-${item.index}-${mark}`" class="pitch-mark"></span>
                    </span>
                    <span class="pitch-number">{{ item[store.config.keyLabel] }}</span>
                    <span v-if="pitchMarkCount(item) > 0 && !pitchMarksAbove(item)" class="pitch-marks below" aria-hidden="true">
                        <span v-for="mark in pitchMarkCount(item)" :key="`below-${item.index}-${mark}`" class="pitch-mark"></span>
                    </span>
                </template>
                <template v-else>{{ item[store.config.keyLabel] }}</template>
            </div>
        </div>
    </div>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, onMounted, reactive, ref, watch} from 'vue'
import {Events} from '@wailsio/runtime'
import {numberedPitch, keyVisualClass} from '../services/keyDisplay'
import {createKeyboardLayout} from '../services/keyboardLayout'
import {subscribeLivePerformance} from '../services/livePerformance'
import {getKeyLabelColor} from '../services/uiTheme'

const props = defineProps({layout: {type: Array, default: null}, showHints: {type: Boolean, default: true}})

const store = inject('store')
const setInteractiveKey = inject('setInteractiveKey')
const keybed = ref(null)
const keybedWidth = ref(0)
let resizeObserver = null
const attackAnimations = new Map()
const feedbackUnsubscribers = []
const hasNoteFeedback = computed(() => store.mainMode !== 'free'
    || ['playing', 'paused'].includes(store.midiPlayerState?.status))

// Note-on is an event, rather than a change of held state. Replaying this
// animation also works when NoteOff/NoteOn arrive in one Vue render frame.
function flashAttack(note, hand = 'right') {
    if (!Number.isInteger(note) || note < 0 || note > 127) return
    const element = keybed.value?.querySelector(`[data-midi-note="${note}"]`)
    if (!element?.animate) return
    const black = element.classList.contains('black')
    const left = hand === 'left' && !store.pressedKey[note] && !store.midiPlaybackKey[note]
    const token = `${black ? 'blackKey' : 'whiteKey'}${left ? 'Left' : ''}`
    const palette = store.config.colors?.[token]
    const color = (typeof palette === 'string' ? palette : palette?.color)
        || (left ? '#f7e89a' : '#9af7b3')
    const base = black ? '#333333' : '#f6f6f6'
    const baseText = black ? '#eeeeee' : '#616161'
    const activeText = getKeyLabelColor(color)
    attackAnimations.get(element)?.cancel()
    const animation = element.animate([
        {backgroundColor: base, color: baseText, offset: 0},
        {backgroundColor: base, color: baseText, offset: .3},
        {backgroundColor: color, color: activeText, offset: .7},
        {backgroundColor: color, color: activeText, offset: 1},
    ], {duration: 100, easing: 'linear'})
    attackAnimations.set(element, animation)
    const forget = () => {
        if (attackAnimations.get(element) === animation) attackAnimations.delete(element)
    }
    animation.onfinish = forget
    animation.oncancel = forget
}
function clearAttackFeedback() {
    for (const animation of attackAnimations.values()) animation.cancel()
    attackAnimations.clear()
}

const mouse = reactive({
    down: false,
    currentKey: -1,
})

const visibleKeys = computed(() => {
    const range = store.keyboardRange[store.config.keyboardType] || store.keyboardRange[0]
    return store.keyboardConfig.slice(range[0], range[1]).map(item => store.config.keyLabel === 'pitch'
        ? {...item, ...numberedPitch(item.index, store.config.keyTonic ?? 0)} : item)
})

const keyPositions = computed(() => new Map(
    (props.layout || createKeyboardLayout(visibleKeys.value, keybedWidth.value))
        .map(key => [key.note, key]),
))

function keyStyle(note) {
    const position = keyPositions.value.get(note)
    return position ? {left: `${position.x}px`, width: `${position.width}px`, '--key-width': `${position.width}px`} : {visibility: 'hidden'}
}

function getKeyClass(item) {
    const visual = keyVisualClass(store, item)
    return !props.showHints && visual.endsWith('-hint') ? '' : visual
}

function hintHand(note) {
    if (!props.showHints) return ''
    if (store.midiHintKey[note]) return 'right'
    return store.midiHintLeftKey[note] ? 'left' : ''
}

function pitchMarkCount(item) {
    return Math.abs(Number(item.octave) - 4)
}

function pitchMarksAbove(item) {
    return Number(item.octave) > 4
}

function pressKey(event, key) {
    if (event.button !== 0) return
    mouse.down = true
    if (mouse.currentKey === key) return

    releaseKey(mouse.currentKey)
    mouse.currentKey = key
    setInteractiveKey(key, true)
}

function releaseKey(key) {
    if (key === -1 || key === undefined || mouse.currentKey !== key) return
    mouse.currentKey = -1
    setInteractiveKey(key, false)
}

function releaseMouseKey() {
    mouse.down = false
    releaseKey(mouse.currentKey)
    mouse.currentKey = -1
}

function enterKey(event, key) {
    if (!(event.buttons & 1)) {
        releaseMouseKey()
        return
    }
    if (!mouse.down || mouse.currentKey === key) return
    pressKey({button: 0}, key)
}

onMounted(() => {
    resizeObserver = new ResizeObserver(entries => {
        keybedWidth.value = entries[0]?.contentRect.width || 0
    })
    resizeObserver.observe(keybed.value)
    keybedWidth.value = keybed.value.clientWidth
    window.addEventListener('mouseup', releaseMouseKey)
    window.addEventListener('blur', releaseMouseKey)
    feedbackUnsubscribers.push(Events.On('midiPlaybackKey', event => {
        const signal = event.data
        if (signal?.active && signal.attack && store.midiPlayerState?.status === 'playing') flashAttack(signal.note, signal.hand)
    }))
    feedbackUnsubscribers.push(Events.On('midiVisualClear', clearAttackFeedback))
    feedbackUnsubscribers.push(Events.On('allNotesOff', clearAttackFeedback))
    feedbackUnsubscribers.push(subscribeLivePerformance(signal => {
        if (signal.type === 'clear') clearAttackFeedback()
        else if (store.mainMode !== 'free' && signal.type === 'noteOn') flashAttack(signal.note)
    }))
})

watch(() => [store.mainMode, store.config.keyboardType], clearAttackFeedback)
watch(() => store.midiPlayerState?.status, status => {
    if (status !== 'playing') clearAttackFeedback()
})

onBeforeUnmount(() => {
    releaseMouseKey()
    resizeObserver?.disconnect()
    window.removeEventListener('mouseup', releaseMouseKey)
    window.removeEventListener('blur', releaseMouseKey)
    for (const unsubscribe of feedbackUnsubscribers) unsubscribe()
    clearAttackFeedback()
})
</script>

<style lang="scss" scoped>
.classic-keyboard {
    position: relative;
    height: 100%;
    width: 100%;
    overflow: hidden;
    touch-action: none;
}

.key {
    position: absolute;
    top: 0;
    box-sizing: border-box;
    transition: background-color 100ms, box-shadow 100ms;
    height: 100%;
    user-select: none;
}

.has-note-feedback .key { transition: none; }

.practice-dot {
    --hint-color: var(--whiteKey);
    position: absolute;
    left: 50%;
    top: 78%;
    width: clamp(8px, calc(var(--key-width) * .37), 18px);
    aspect-ratio: 1;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    background: var(--hint-color);
    border: 2px solid #ffffffd9;
    box-shadow: 0 0 0 2px #242424ab, 0 0 13px var(--hint-color);
    pointer-events: none;
}
.practice-dot.is-left { --hint-color: var(--whiteKeyLeft); }
.black .practice-dot { --hint-color: var(--blackKey); }
.black .practice-dot.is-left { --hint-color: var(--blackKeyLeft); }

.label {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 16px;
    display: flex;
    justify-content: center;
    opacity: 0.5;
}

.pitch-label {
    flex-direction: column;
    align-items: center;
    gap: 2px;
    line-height: 1;
}

.pitch-marks {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
}

.pitch-marks.below {
    margin-top: 1px;
}

.pitch-mark {
    width: 3px;
    height: 3px;
    border-radius: 50%;
    background: currentColor;
}

.black .label {
    opacity: 0.8;
}

.white {
    height: 100%;
    z-index: 1;
    border-left: 1px solid #bbb;
    border-bottom: 1px solid #bbb;
    border-radius: 0;
    box-shadow: -1px 0 0 rgba(255, 255, 255, 0.5) inset, 0 0 5px #ccc inset, 0 0 3px rgba(0, 0, 0, 0.1);
    background-color: #f6f6f6;
    color: #616161;

    .label {
        font-size: 14px;
        color: inherit;
        opacity: 1;
    }
}

.w-active {
    border-left: 1px solid var(--whiteKey-o);
    border-bottom: 1px solid var(--whiteKey-o);
    box-shadow: 3px 0 3px rgba(0, 0, 0, 0.1) inset, -3px 0 8px rgba(0, 0, 0, 0.1) inset, 0 0 3px rgba(0, 0, 0, 0.2);
    background-color: var(--whiteKey);
    color: var(--whiteKey-label);
}

.w-l-active {
    border-left: 1px solid var(--whiteKeyLeft-o);
    border-bottom: 1px solid var(--whiteKeyLeft-o);
    box-shadow: 3px 0 3px rgba(0, 0, 0, 0.1) inset, -3px 0 8px rgba(0, 0, 0, 0.1) inset, 0 0 3px rgba(0, 0, 0, 0.2);
    background-color: var(--whiteKeyLeft);
    color: var(--whiteKeyLeft-label);
}


.black {
    height: 63%;
    z-index: 3;
    border-bottom: 1px solid #000;
    border-left: 1px solid #000;
    border-radius: 0 0 3px 3px;
    box-shadow: -1px -1px 2px rgba(255, 255, 255, 0.2) inset, 0 -5px 2px 3px rgba(0, 0, 0, 0.6) inset, 0 2px 4px rgba(0, 0, 0, 0.5);
    background-color: #333;
    color: #eeeeee;

    .label {
        font-size: 13px;
        color: inherit;
    }
}

.b-active {
    box-shadow: -1px -5px 2px rgba(131, 131, 131, 0.2) inset, 0 -10px 10px 5px rgba(0, 0, 0, 0.6) inset, 0 1px 2px rgba(0, 0, 0, 0.5);
    background-color: var(--blackKey);
    color: var(--blackKey-label);
    border-bottom: 1px solid var(--blackKey-o);
    border-left: 1px solid var(--blackKey-o);
}
.b-l-active {
    box-shadow: -1px -5px 2px rgba(131, 131, 131, 0.2) inset, 0 -10px 10px 5px rgba(0, 0, 0, 0.6) inset, 0 1px 2px rgba(0, 0, 0, 0.5);
    background-color: var(--blackKeyLeft);
    color: var(--blackKeyLeft-label);
    border-bottom: 1px solid var(--blackKeyLeft-o);
    border-left: 1px solid var(--blackKeyLeft-o);
}



.w-sustained {
    background-color: color-mix(in srgb, var(--whiteKey) 38%, #f6f6f6);
    color: var(--whiteKey-sustained-label);
}

.w-hint, .b-hint {
    box-shadow: inset 0 0 0 2px var(--whiteKey), 0 0 7px var(--whiteKey-o);
}
.w-l-hint, .b-l-hint {
    box-shadow: inset 0 0 0 2px var(--whiteKeyLeft), 0 0 7px var(--whiteKeyLeft-o);
}

.b-sustained {
    background-color: color-mix(in srgb, var(--blackKey) 38%, #333);
    color: var(--blackKey-sustained-label);
}

</style>
