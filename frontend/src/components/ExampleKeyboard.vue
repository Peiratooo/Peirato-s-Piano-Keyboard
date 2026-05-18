<template>
    <div
        class="classic-keyboard"
        @mousedown="mouse.down = true"
        @mouseup="resetMouse"
        @mouseleave="resetMouse"
    >
        <div
            class="key"
            v-for="(item, index) in keyboardKeys"
            :key="item.midi || item.note || index"
            :class="[
                item.color,
                item.note,
                getHandKeyClass(item, index, 'left'),
                getHandKeyClass(item, index, 'right'),
                getMouseActiveClass(item, index),
            ]"
            @mouseenter="mouse.keyIndex = index"
        >
            <div class="label" v-if="activeLabel !== ''">
                {{ item[activeLabel] }}
            </div>
        </div>
    </div>
</template>

<script setup>
import { computed, inject, onMounted, reactive, ref } from "vue"

const store = inject("store")

const keyColorMap = {
    left: {
        black: "b-l-active",
        white: "w-l-active",
    },
    right: {
        black: "b-r-active",
        white: "w-r-active",
    },
}

const KEY_START_INDEX = 39

const keyboardKeys = computed(() => {
    return store.keyboardConfig.slice(KEY_START_INDEX, 63)
})


const activeKeys = reactive({
    left: [2, 5, 8],
    right: [12, 15, 18],
})

const mouse = reactive({
    down: false,
    keyIndex: -1,
})

const activeLabel = ref("")

const size = reactive({
    wWidth: 0,
    bWidth: 0,
    gap: 0,
    ratio: 1.6,
})

function getHandKeyClass(item, index, hand) {
    if (!activeKeys[hand]?.includes(index)) {
        return ""
    }

    return keyColorMap[hand][item.color] || ""
}

function getMouseActiveClass(item, index) {
    if (!mouse.down || mouse.keyIndex !== index) {
        return ""
    }

    return keyColorMap.right[item.color] || ""
}

function resetMouse() {
    mouse.down = false
    mouse.keyIndex = -1
}

function resize() {
    size.wWidth = 38
    size.bWidth = size.wWidth / size.ratio
    size.gap = size.bWidth / size.ratio

    document.documentElement.style.setProperty(
        "--example-black-key-width",
        `${size.bWidth}px`
    )

    document.documentElement.style.setProperty(
        "--example-white-key-width",
        `${size.wWidth}px`
    )

    document.documentElement.style.setProperty(
        "--example-white-key-offset",
        `${-size.bWidth}px`
    )

    document.documentElement.style.setProperty(
        "--example-black-key-offset",
        `${-size.gap * 0.7}px`
    )
}

onMounted(() => {
    resize()
})
</script>

<style lang="scss" scoped>
.classic-keyboard {
    display: flex;
    flex-direction: row;
    height: 100%;
}

.key {
    position: relative;
    transition: 150ms;
    height: 100%;
    user-select: none;
}

.label {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 10%;
    display: flex;
    justify-content: center;
}

.white {
    height: 100%;
    width: var(--example-white-key-width);
    z-index: 1;
    border-left: 1px solid #bbb;
    border-bottom: 1px solid #bbb;
    border-right: 0 solid #bbb;
    border-radius: 0 0 5px 5px;
    box-shadow:
        -1px 0 0 rgba(255, 255, 255, 0.8) inset,
        0 0 5px #ccc inset,
        0 0 3px rgba(0, 0, 0, 0.2);
    background-color: #eee;

    .label {
        font-size: 18px;
    }
}

.black {
    height: 63%;
    width: var(--example-black-key-width);
    transform: translateX(var(--example-black-key-offset));
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

    .label {
        font-size: 13px;
        color: #eee;
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
}

.b-active,
.b-r-active {
    box-shadow:
        -1px -5px 2px rgba(131, 131, 131, 0.2) inset,
        0 -10px 10px 5px rgba(0, 0, 0, 0.6) inset,
        0 1px 2px rgba(0, 0, 0, 0.5);
    background-color: var(--blackKey);
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
}

.A:first-child {
    margin: 0;
}

.B,
.D,
.E,
.A,
.G {
    margin-left: var(--example-white-key-offset);
}
</style>