<template>
    <div
        class="classic-keyboard"
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
            @mouseenter="enterKey($event, item.index)"
            @mousedown.prevent="pressKey($event, item.index)"
            @mouseup="releaseKey(item.index)"
        >
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
import {computed, inject, onBeforeUnmount, onMounted, reactive, ref} from 'vue'
import {numberedPitch, keyVisualClass} from '../services/keyDisplay'
import {createKeyboardLayout} from '../services/keyboardLayout'

const props = defineProps({layout: {type: Array, default: null}})

const store = inject('store')
const Keyboard = inject('Keyboard')
const keybed = ref(null)
const keybedWidth = ref(0)
let resizeObserver = null

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
    return position ? {left: `${position.x}px`, width: `${position.width}px`} : {visibility: 'hidden'}
}

function getKeyClass(item) {
    return keyVisualClass(store, item)
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
    store.setKeyState(key, true)
    Keyboard.KeyboardPlay(key)
}

function releaseKey(key) {
    if (key === -1 || key === undefined || mouse.currentKey !== key) return
    mouse.currentKey = -1
    store.setKeyState(key, false)
    Keyboard.KeyboardStop(key)
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
})

onBeforeUnmount(() => {
    releaseMouseKey()
    resizeObserver?.disconnect()
    window.removeEventListener('mouseup', releaseMouseKey)
    window.removeEventListener('blur', releaseMouseKey)
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
    border-radius: 0 0 5px 5px;
    box-shadow: -1px 0 0 rgba(255, 255, 255, 0.5) inset, 0 0 5px #ccc inset, 0 0 3px rgba(0, 0, 0, 0.1);
    background-color: #f6f6f6;

    .label {
        font-size: 14px;
        color: #616161;
        opacity: 1;
    }
}

.w-active {
    border-left: 1px solid var(--whiteKey-o);
    border-bottom: 1px solid var(--whiteKey-o);
    box-shadow: 3px 0 3px rgba(0, 0, 0, 0.1) inset, -3px 0 8px rgba(0, 0, 0, 0.1) inset, 0 0 3px rgba(0, 0, 0, 0.2);
    background-color: var(--whiteKey);
}

.w-l-active {
    border-left: 1px solid var(--whiteKeyLeft-o);
    border-bottom: 1px solid var(--whiteKeyLeft-o);
    box-shadow: 3px 0 3px rgba(0, 0, 0, 0.1) inset, -3px 0 8px rgba(0, 0, 0, 0.1) inset, 0 0 3px rgba(0, 0, 0, 0.2);
    background-color: var(--whiteKeyLeft);
}


.black {
    height: 63%;
    z-index: 3;
    border-bottom: 1px solid #000;
    border-left: 1px solid #000;
    border-radius: 0 0 3px 3px;
    box-shadow: -1px -1px 2px rgba(255, 255, 255, 0.2) inset, 0 -5px 2px 3px rgba(0, 0, 0, 0.6) inset, 0 2px 4px rgba(0, 0, 0, 0.5);
    background-color: #333;

    .label {
        font-size: 13px;
        color: #eee;
    }
}

.b-active {
    box-shadow: -1px -5px 2px rgba(131, 131, 131, 0.2) inset, 0 -10px 10px 5px rgba(0, 0, 0, 0.6) inset, 0 1px 2px rgba(0, 0, 0, 0.5);
    background-color: var(--blackKey);
    border-bottom: 1px solid var(--blackKey-o);
    border-left: 1px solid var(--blackKey-o);
}
.b-l-active {
    box-shadow: -1px -5px 2px rgba(131, 131, 131, 0.2) inset, 0 -10px 10px 5px rgba(0, 0, 0, 0.6) inset, 0 1px 2px rgba(0, 0, 0, 0.5);
    background-color: var(--blackKeyLeft);
    border-bottom: 1px solid var(--blackKeyLeft-o);
    border-left: 1px solid var(--blackKeyLeft-o);
}



.w-sustained {
    background-color: color-mix(in srgb, var(--whiteKey) 38%, #f6f6f6);
}

.w-hint, .b-hint {
    box-shadow: inset 0 0 0 2px var(--whiteKey), 0 0 7px var(--whiteKey-o);
}
.w-l-hint, .b-l-hint {
    box-shadow: inset 0 0 0 2px var(--whiteKeyLeft), 0 0 7px var(--whiteKeyLeft-o);
}

.b-sustained {
    background-color: color-mix(in srgb, var(--blackKey) 38%, #333);
}

</style>
