<template>
    <div
        class="classic-keyboard"
        @pointerup="previewNote = -1"
        @pointerleave="previewNote = -1"
    >
        <div
            class="key"
            v-for="item in layout"
            :key="item.note"
            :class="[item.isBlack ? 'black' : 'white', activeClass(item)]"
            :style="{left: item.x + 'px', width: item.width + 'px'}"
            @pointerdown.prevent="previewNote = item.note"
            @pointerenter="($event.buttons & 1) && (previewNote = item.note)"
        >
            <div v-if="item.pitchName === 'C'" class="label">
                {{ item.pitchName }}{{ item.octave }}
            </div>
        </div>
    </div>
</template>

<script setup>
import {ref} from 'vue'

defineProps({layout: {type: Array, required: true}})
const previewNote = ref(-1)
const activeNotes = new Set([48, 51, 55, 63, 67, 70])
function activeClass(key) {
    if (!activeNotes.has(key.note) && previewNote.value !== key.note) return ''
    return `${key.isBlack ? 'b' : 'w'}-${key.note < 60 ? 'l' : 'r'}-active`
}
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
    transition: background-color 100ms, color 100ms;
    height: 100%;
    user-select: none;
}

.label {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 8px;
    display: flex;
    justify-content: center;
}

.white {
    height: 100%;
    z-index: 1;
    border-left: 1px solid #bbb;
    border-bottom: 1px solid #bbb;
    border-right: 0 solid #bbb;
    border-radius: 0;
    box-shadow:
        -1px 0 0 rgba(255, 255, 255, 0.8) inset,
        0 0 5px #ccc inset,
        0 0 3px rgba(0, 0, 0, 0.2);
    background-color: #f6f6f6;
    color: #616161;

    .label {
        font-size: 11px;
        color: inherit;
    }
}

.black {
    height: 63%;
    z-index: 3;
    border-bottom: 1px solid #000;
    border-right: 0 solid #000;
    border-left: 1px solid #000;
    border-radius: 0 0 3px 3px;
    box-shadow:
        -1px -1px 2px rgba(255, 255, 255, 0.2) inset,
        0 -5px 2px 3px rgba(0, 0, 0, 0.6) inset,
        0 2px 4px rgba(0, 0, 0, 0.5);
    background-color: #333;
    color: #eeeeee;

    .label {
        font-size: 10px;
        color: inherit;
    }
}

.w-active,
.w-r-active {
    border-left: 1px solid var(--whiteKey-o);
    border-bottom: 1px solid var(--whiteKey-o);
    border-right: 1px solid var(--whiteKey-o);
    box-shadow:
        3px 0 3px rgba(0, 0, 0, 0.1) inset,
        -3px 0 8px rgba(0, 0, 0, 0.1) inset,
        0 0 3px rgba(0, 0, 0, 0.2);
    background-color: var(--whiteKey);
    color: var(--whiteKey-label);
}

.b-active,
.b-r-active {
    box-shadow:
        -1px -5px 2px rgba(131, 131, 131, 0.2) inset,
        0 -10px 10px 5px rgba(0, 0, 0, 0.6) inset,
        0 1px 2px rgba(0, 0, 0, 0.5);
    background-color: var(--blackKey);
    color: var(--blackKey-label);
    border-bottom: 1px solid var(--blackKey-o);
    border-right: 0 solid var(--blackKey-o);
    border-left: 1px solid var(--blackKey-o);
}

.b-l-active {
    box-shadow:
        -1px -5px 2px rgba(131, 131, 131, 0.2) inset,
        0 -10px 10px 5px rgba(0, 0, 0, 0.6) inset,
        0 1px 2px rgba(0, 0, 0, 0.5);
    background-color: var(--blackKeyLeft);
    color: var(--blackKeyLeft-label);
    border-bottom: 1px solid var(--blackKeyLeft-o);
    border-left: 1px solid var(--blackKeyLeft-o);
}

.w-l-active {
    border-left: 1px solid var(--whiteKeyLeft-o);
    border-bottom: 1px solid var(--whiteKeyLeft-o);
    border-right: 1px solid var(--whiteKeyLeft-o);
    box-shadow:
        3px 0 3px rgba(0, 0, 0, 0.1) inset,
        -3px 0 8px rgba(0, 0, 0, 0.1) inset,
        0 0 3px rgba(0, 0, 0, 0.2);
    background-color: var(--whiteKeyLeft);
    color: var(--whiteKeyLeft-label);
}

</style>
