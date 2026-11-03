<template>
    <div
        class="classic-keyboard"
        ref="keybed"
        @pointermove="movePointer"
        @pointerup="releasePointer"
        @pointercancel="releasePointer"
        @lostpointercapture="releasePointer"
    >
        <div
            class="key"
            v-for="item in visibleKeys"
            :class="[item.color, item.note, getKeyClass(item)]"
            :style="keyStyle(item.index)"
            :key="item.index"
            :data-midi-note="item.index"
            @pointerdown.prevent="pressKey($event, item.index)"
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
import {computed, inject, onBeforeUnmount, onMounted, ref} from 'vue'
import {numberedPitch, keyVisualClass} from '../services/keyDisplay'
import {createKeyboardLayout} from '../services/keyboardLayout'

const props = defineProps({layout: {type: Array, default: null}, showHints: {type: Boolean, default: true}})

const store = inject('store')
const setInteractiveKey = inject('setInteractiveKey')
const keybed = ref(null)
const keybedWidth = ref(0)
let resizeObserver = null

const pointers = new Map()

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
    if (event.pointerType === 'mouse' && event.button !== 0) return
    keybed.value.setPointerCapture(event.pointerId)
    pointers.set(event.pointerId, key)
    setInteractiveKey(key, true)
}
function releasePointer(event) {
    const key = pointers.get(event.pointerId)
    if (key !== undefined && key !== null) setInteractiveKey(key, false)
    pointers.delete(event.pointerId)
}
function releasePointers() {
    for (const key of pointers.values()) if (key !== null) setInteractiveKey(key, false)
    pointers.clear()
}
function movePointer(event) {
    if (!pointers.has(event.pointerId)) return
    const rect = keybed.value.getBoundingClientRect()
    const x = event.clientX - rect.left, y = event.clientY - rect.top
    const key = y < 0 || y >= rect.height ? null : [...keyPositions.value.values()]
        .sort((a, b) => Number(b.isBlack) - Number(a.isBlack))
        .find(item => x >= item.x && x < item.x + item.width && (!item.isBlack || y < rect.height * .63))?.note ?? null
    const previous = pointers.get(event.pointerId)
    if (previous === key) return
    if (previous !== null) setInteractiveKey(previous, false)
    pointers.set(event.pointerId, key)
    if (key !== null) setInteractiveKey(key, true)
}

onMounted(() => {
    resizeObserver = new ResizeObserver(entries => {
        keybedWidth.value = entries[0]?.contentRect.width || 0
    })
    resizeObserver.observe(keybed.value)
    keybedWidth.value = keybed.value.clientWidth
    window.addEventListener('blur', releasePointers)
})

onBeforeUnmount(() => {
    releasePointers()
    resizeObserver?.disconnect()
    window.removeEventListener('blur', releasePointers)
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
    height: 100%;
    user-select: none;
}


.practice-dot {
    --hint-color: var(--whiteKey);
    position: absolute;
    left: 50%;
    bottom: 32px;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    transform: translateX(-50%);
    background: var(--hint-color);
    border: 1px solid color-mix(in srgb, var(--hint-color), white 40%);
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--hint-color) 22%, transparent);
    pointer-events: none;
    z-index: 1;
}
.practice-dot.is-left { --hint-color: var(--whiteKeyLeft); }
.black .practice-dot {
    --hint-color: var(--blackKey);
    bottom: 24px;
    width: 6px;
    height: 6px;
    box-shadow: 0 0 0 1px color-mix(in srgb, var(--hint-color) 25%, transparent);
}
.black .practice-dot.is-left { --hint-color: var(--blackKeyLeft); }

.label {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 12px;
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
    border: 0;
    border-radius: 0 0 2px 2px;
    background-color: #f5f4f0;
    background-image: linear-gradient(180deg, #ffffff08 0%, #ffffff85 12%, #ffffff18 84%, #0000000a 100%);
    box-shadow: inset 1px 0 0 #bdbdb7, inset 0 -3px 0 #d8d7d1, inset 0 1px 0 #fffefa;
    color: #6b6c68;

    .label { font-size: 12px; color: inherit; opacity: 1; }
}

.w-active { background-color: var(--whiteKey); color: var(--whiteKey-label); }
.w-l-active { background-color: var(--whiteKeyLeft); color: var(--whiteKeyLeft-label); }

.black {
    height: 63%;
    z-index: 3;
    border: 0;
    border-radius: 0 0 3px 3px;
    background-color: #242628;
    background-image: linear-gradient(90deg, #00000072, #ffffff0c 14%, #ffffff02 55%, #00000022 85%, #00000080),
        linear-gradient(180deg, #ffffff09, transparent 12%, transparent 84%, #00000035);
    box-shadow: inset 1px 0 0 #ffffff12, inset -1px 0 0 #00000065, inset 0 -4px 0 #08090a, 0 2px 3px #00000038;
    color: #d4d6d5;

    .label { font-size: 10px; bottom: 9px; color: inherit; }
}
.b-active { background-color: var(--blackKey); color: var(--blackKey-label); }
.b-l-active { background-color: var(--blackKeyLeft); color: var(--blackKeyLeft-label); }

.key::after {
    content: '';
    position: absolute;
    inset: 0;
    box-sizing: border-box;
    border: 1px solid transparent;
    border-radius: inherit;
    pointer-events: none;
}
.w-hint::after { border-color: var(--whiteKey); }
.w-l-hint::after { border-color: var(--whiteKeyLeft); }
.b-hint::after { border-color: var(--blackKey); }
.b-l-hint::after { border-color: var(--blackKeyLeft); }

.w-sustained { background-color: color-mix(in srgb, var(--whiteKey) 38%, #f5f4f0); color: var(--whiteKey-sustained-label); }
.b-sustained { background-color: color-mix(in srgb, var(--blackKey) 38%, #242628); color: var(--blackKey-sustained-label); }
/* Keep the entire key face lit, including the bottom edge. */
.w-active,.w-l-active,.w-sustained,.w-l-sustained{background-image:none;box-shadow:inset 1px 0 0 #00000020,inset 0 -1px 0 #00000025}
.b-active,.b-l-active,.b-sustained,.b-l-sustained{background-image:none;box-shadow:inset 1px 0 0 #ffffff20,inset -1px 0 0 #00000025,0 1px 2px #00000030}
.w-l-sustained{background-color:color-mix(in srgb,var(--whiteKeyLeft) 38%,#f5f4f0);color:var(--whiteKeyLeft-sustained-label)}
.b-l-sustained{background-color:color-mix(in srgb,var(--blackKeyLeft) 38%,#242628);color:var(--blackKeyLeft-sustained-label)}
</style>
