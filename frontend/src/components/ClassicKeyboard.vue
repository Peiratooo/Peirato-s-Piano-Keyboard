<template>
    <div
        class="classic-keyboard"
        @mousedown="mouse.down = true"
        @mouseup="releaseMouseKey"
        @mouseleave="releaseMouseKey"
    >
        <div
            class="key"
            v-for="item in visibleKeys"
            :class="[item.color, item.note, getKeyClass(item)]"
            :key="item.index"
            @mouseenter="enterKey(item.index)"
            @mousedown.prevent="pressKey(item.index)"
            @mouseup="releaseKey(item.index)"
        >
            <div class="label" v-if="store.config.keyLabel !== '' && item[store.config.keyLabel]">
                {{ item[store.config.keyLabel] }}
            </div>
        </div>
    </div>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, onMounted, reactive} from 'vue'

const store = inject('store')
const Keyboard = inject('Keyboard')
const resize = inject('resize')

const keyColorMap = {
    black: 'b-active',
    white: 'w-active',
}

const mouse = reactive({
    down: false,
    currentKey: -1,
})

const visibleKeys = computed(() => {
    const range = store.keyboardRange[store.config.keyboardType]
    return store.keyboardConfig.slice(range[0], range[1])
})

function getKeyClass(item) {
    if (store.wrongKey[item.index]) return item.color === 'black' ? 'b-wrong' : 'w-wrong'
    if (store.activeKey[item.index] || store.playbackKey[item.index]) return keyColorMap[item.color]
    if (store.hintKey[item.index]) return item.color === 'black' ? 'b-hint' : 'w-hint'
    return ''
}

function pressKey(key) {
    mouse.down = true
    if (mouse.currentKey === key && store.pressedKey[key]) return

    releaseKey(mouse.currentKey)
    mouse.currentKey = key
    store.setKeyState(key, true)
    Keyboard.KeyboardPlay(key)
}

function releaseKey(key) {
    if (key === -1 || key === undefined || !store.pressedKey[key]) return
    store.setKeyState(key, false)
    Keyboard.KeyboardStop(key)
}

function releaseMouseKey() {
    mouse.down = false
    releaseKey(mouse.currentKey)
    mouse.currentKey = -1
}

function enterKey(key) {
    if (!mouse.down || mouse.currentKey === key) return
    pressKey(key)
}

onMounted(() => {
    resize()
    window.addEventListener('resize', resize)
    window.addEventListener('mouseup', releaseMouseKey)
})

onBeforeUnmount(() => {
    releaseMouseKey()
    window.removeEventListener('resize', resize)
    window.removeEventListener('mouseup', releaseMouseKey)
})
</script>

<style lang="scss" scoped>
.classic-keyboard {
    display: flex;
    flex-direction: row;
    height: 100vh;
    width: 100vw;
}

.key {
    position: relative;
    transition: 100ms;
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

.black .label {
    opacity: 0.8;
}

.white {
    height:100%;
    width:var(--white-key-width);
    z-index:1;
    border-left:1px solid #bbb;
    border-bottom:1px solid #bbb;
    border-radius:0 0 5px 5px;
    box-shadow:-1px 0 0 rgba(255,255,255,0.5) inset,0 0 5px #ccc inset,0 0 3px rgba(0,0,0,0.1);
    background-color: #f6f6f6;

    .label {
        font-size: 18px;
    }
}

.w-active {
    border-left:1px solid var(--whiteKey-o);
    border-bottom:1px solid var(--whiteKey-o);
    box-shadow:3px 0 3px rgba(0,0,0,0.1) inset,-3px 0 8px rgba(0,0,0,0.1) inset,0 0 3px rgba(0,0,0,0.2);
    background-color: var(--whiteKey);
}

.w-hint {
    background: linear-gradient(180deg, #fff, rgba(99, 102, 241, 0.28));
}

.w-wrong {
    background: linear-gradient(180deg, #fff, rgba(239, 68, 68, 0.35));
}

.black {
    height:63%;
    width: var(--black-key-width);
    transform: translateX(var(--black-key-offset));
    z-index:3;
    border-bottom:1px solid #000;
    border-left:1px solid #000;
    border-radius:0 0 3px 3px;
    box-shadow:-1px -1px 2px rgba(255,255,255,0.2) inset,0 -5px 2px 3px rgba(0,0,0,0.6) inset,0 2px 4px rgba(0,0,0,0.5);
    background-color:#333;

    .label {
        font-size: 13px;
        color: #eee;
    }
}

.b-active {
    box-shadow:-1px -5px 2px rgba(131, 131, 131, 0.2) inset,0 -10px 10px 5px rgba(0,0,0,0.6) inset,0 1px 2px rgba(0,0,0,0.5);
    background-color: var(--blackKey);
    border-bottom:1px solid var(--blackKey-o);
    border-left:1px solid var(--blackKey-o);
}

.b-hint {
    background: #4f46e5;
}

.b-wrong {
    background: #dc2626;
}

.A:first-child {
    margin: 0;
}

.B,.D,.E,.A,.G {
    margin-left: var(--white-key-offset);
}
</style>
