<template>
    <div
        ref="containerRef"
        class="container"
        @mouseenter="store.showSetting = true"
        @mouseleave="store.showSetting = false"
    >
        <div class="drag-area">
            <transition enter-active-class="blurFadeIN" leave-active-class="blurFadeOUT">
                <div
                    class="panel main-panel"
                    v-if="store.showSetting || true"
                    :style="{left: store.menuBar ? '0' : '-128px'}"
                >
                    <button class="icon-button quit" title="退出" @click="Keyboard.Quit">
                        <n-icon>
                            <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24"><path d="M12 3c-.55 0-1 .45-1 1v8c0 .55.45 1 1 1s1-.45 1-1V4c0-.55-.45-1-1-1zm5.14 2.86a.99.99 0 0 0-.01 1.39c1.13 1.2 1.83 2.8 1.87 4.57c.09 3.83-3.08 7.13-6.91 7.17A6.981 6.981 0 0 1 5 12c0-1.84.71-3.51 1.87-4.76c.37-.39.37-1-.01-1.38a.993.993 0 0 0-1.43.02A8.92 8.92 0 0 0 3 11.74c-.14 4.88 3.83 9.1 8.71 9.25c5.1.16 9.29-3.93 9.29-9c0-2.37-.92-4.51-2.42-6.11c-.38-.41-1.04-.42-1.44-.02z" fill="currentColor"></path></svg>
                        </n-icon>
                    </button>
                    <button class="icon-button" title="打开设置中心" @click="Keyboard.OpenControlCenter">
                        <n-icon>
                            <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 32 32"><path d="M27 16.76V16v-.77l1.92-1.68A2 2 0 0 0 29.3 11l-2.36-4a2 2 0 0 0-1.73-1a2 2 0 0 0-.64.1l-2.43.82a11.35 11.35 0 0 0-1.31-.75l-.51-2.52a2 2 0 0 0-2-1.61h-4.68a2 2 0 0 0-2 1.61l-.51 2.52a11.48 11.48 0 0 0-1.32.75l-2.38-.86A2 2 0 0 0 6.79 6a2 2 0 0 0-1.73 1L2.7 11a2 2 0 0 0 .41 2.51L5 15.24v1.53l-1.89 1.68A2 2 0 0 0 2.7 21l2.36 4a2 2 0 0 0 1.73 1a2 2 0 0 0 .64-.1l2.43-.82a11.35 11.35 0 0 0 1.31.75l.51 2.52a2 2 0 0 0 2 1.61h4.72a2 2 0 0 0 2-1.61l.51-2.52a11.48 11.48 0 0 0 1.32-.75l2.42.82a2 2 0 0 0 .64.1a2 2 0 0 0 1.73-1l2.28-4a2 2 0 0 0-.41-2.51zM25.21 24l-3.43-1.16a8.86 8.86 0 0 1-2.71 1.57L18.36 28h-4.72l-.71-3.55a9.36 9.36 0 0 1-2.7-1.57L6.79 24l-2.36-4l2.72-2.4a8.9 8.9 0 0 1 0-3.13L4.43 12l2.36-4l3.43 1.16a8.86 8.86 0 0 1 2.71-1.57L13.64 4h4.72l.71 3.55a9.36 9.36 0 0 1 2.7 1.57L25.21 8l2.36 4l-2.72 2.4a8.9 8.9 0 0 1 0 3.13L27.57 20z" fill="currentColor"></path><path d="M16 22a6 6 0 1 1 6-6a5.94 5.94 0 0 1-6 6zm0-10a3.91 3.91 0 0 0-4 4a3.91 3.91 0 0 0 4 4a3.91 3.91 0 0 0 4-4a3.91 3.91 0 0 0-4-4z" fill="currentColor"></path></svg>
                        </n-icon>
                    </button>
                    <button class="icon-button" title="打开 MIDI 中心" @click="Keyboard.OpenMidiCenter">
                        <n-icon>
                            <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 16 16"><g fill="none"><path d="M12 5.21v5.058A2 2 0 1 0 13 12V1.926a.8.8 0 0 0-1.07-.754l-6.4 2.286a.8.8 0 0 0-.53.753v7.056A2 2 0 1 0 6 13V7.353l6-2.142zm0-1.062L6 6.29V4.351l6-2.143v1.939zM11 11a1 1 0 1 1 0 2a1 1 0 0 1 0-2zm-7 1a1 1 0 1 1 0 2a1 1 0 0 1 0-2z" fill="currentColor"></path></g></svg>
                        </n-icon>
                    </button>
                    <button
                        class="icon-button switcher"
                        @click="store.menuBar = !store.menuBar"
                        :style="{transform: store.menuBar ? 'rotate(0deg)' : 'rotate(180deg)'}"
                        title="收起/展开"
                    >
                       <n-icon>
                           <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24"><path d="M15.41 16.59L10.83 12l4.58-4.59L14 6l-6 6l6 6l1.41-1.41z" fill="currentColor"></path></svg>
                       </n-icon>
                    </button>
                </div>
            </transition>
        </div>

        <ClassicKeyboard v-if="store.keyboardLoaded"/>

        <div
            v-if="store.loaded && store.config.showPedal"
            ref="pedalRef"
            class="floating-card"
            :style="{ left: pedalDrag.x + 'px', top: pedalDrag.y + 'px' }"
            @mousedown="startDragPedal"
        >
            <Pedal />
        </div>

        <div
            v-if="store.loaded && !store.midiWindowOpen"
            ref="chordRef"
            class="floating-card"
            :style="{ left: chordPos.x + 'px', top: chordPos.y + 'px' }"
            @mousedown="startDragChord"
        >
            <Chord />
        </div>
    </div>
</template>

<script setup>
import { inject, ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import ClassicKeyboard from '../components/ClassicKeyboard.vue'
import Pedal from '../components/Pedal.vue'
import Chord from '../components/Chord.vue'
import { NIcon } from 'naive-ui'

const store = inject('store')
const Keyboard = inject('Keyboard')

const containerRef = ref(null)
const pedalRef = ref(null)
const chordRef = ref(null)

const EDGE_GAP = 12

const pedalDrag = ref({
    x: 0,
    y: 0,
    xr: 0.92,
    yr: 0.7,
})

const chordPos = ref({
    x: 0,
    y: 0,
    xr: 0.02,
    yr: 0.65,
})

let draggingType = null

function getContainerRect() {
    const el = containerRef.value
    if (!el) {
        return {
            width: window.innerWidth,
            height: window.innerHeight,
        }
    }

    return el.getBoundingClientRect()
}

function getElementSize(targetRef) {
    const el = targetRef.value

    if (!el || typeof el.getBoundingClientRect !== 'function') {
        return {
            width: 0,
            height: 0,
        }
    }

    const rect = el.getBoundingClientRect()

    return {
        width: rect.width,
        height: rect.height,
    }
}

function clamp(value, min, max) {
    if (max < min) return min
    return Math.min(Math.max(value, min), max)
}

function clampFloatingPosition(pos, targetRef) {
    const container = getContainerRect()
    const size = getElementSize(targetRef)

    const maxX = container.width - size.width - EDGE_GAP
    const maxY = container.height - size.height - EDGE_GAP

    pos.x = clamp(pos.x, EDGE_GAP, maxX)
    pos.y = clamp(pos.y, EDGE_GAP, maxY)

    pos.xr = container.width > 0 ? pos.x / container.width : pos.xr
    pos.yr = container.height > 0 ? pos.y / container.height : pos.yr
}

function syncPositionByRatio(pos, targetRef) {
    const container = getContainerRect()

    pos.x = container.width * pos.xr
    pos.y = container.height * pos.yr

    clampFloatingPosition(pos, targetRef)
}

function syncAllFloatingPositions() {
    syncPositionByRatio(pedalDrag.value, pedalRef)
    syncPositionByRatio(chordPos.value, chordRef)
}

function startDragPedal(event) {
    if (event.button !== 0) return

    draggingType = 'pedal'

    window.addEventListener('mousemove', handleDragging)
    window.addEventListener('mouseup', stopDragging)
}

function startDragChord(event) {
    if (event.button !== 0) return

    draggingType = 'chord'

    window.addEventListener('mousemove', handleDragging)
    window.addEventListener('mouseup', stopDragging)
}

function handleDragging(event) {
    if (!draggingType) return

    if (draggingType === 'pedal') {
        pedalDrag.value.x += event.movementX
        pedalDrag.value.y += event.movementY
        clampFloatingPosition(pedalDrag.value, pedalRef)
        return
    }

    if (draggingType === 'chord') {
        chordPos.value.x += event.movementX
        chordPos.value.y += event.movementY
        clampFloatingPosition(chordPos.value, chordRef)
    }
}

function stopDragging() {
    draggingType = null

    window.removeEventListener('mousemove', handleDragging)
    window.removeEventListener('mouseup', stopDragging)
}

async function handleResize() {
    await nextTick()
    syncAllFloatingPositions()
}

onMounted(async () => {
    await nextTick()
    syncAllFloatingPositions()

    window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
    window.removeEventListener('resize', handleResize)
    window.removeEventListener('mousemove', handleDragging)
    window.removeEventListener('mouseup', stopDragging)
})
</script>

<style lang="scss" scoped>
.container {
    position: relative;
    width: 100vw;
    height: 100vh;
    border-radius: 6px;
    overflow: hidden;
}
.floating-card {
    position: absolute;
    z-index: 999;
    cursor: grab;
    user-select: none;
    touch-action: none;
}

.floating-card:active {
    cursor: grabbing;
}

.drag-area {
    position: absolute;
    left: 0;
    right: 0;
    height: 25vh;
    background-color: #1f1f1f00;
    z-index: 999;
    --wails-draggable: drag;
    cursor: move;
    transition: 500ms;
    display: flex;
}

.panel {
    --wails-draggable: no-drag;
    height: 44px;
    align-items: center;
    font-size: 24px;
    gap: 12px;
    display: flex;
    position: relative;
    padding-left: 14px;
    padding-right: 8px;
    background-color: rgba(180, 180, 180, 0.29);
    border: 1px solid #99999933;
    box-shadow: 0 0 8px #00000033;
    backdrop-filter: blur(8px);
    cursor: auto;
    transition: 200ms;
}

.main-panel {
    border-radius: 0 0 16px 0;
    button {
        display: flex;
        align-items: center;
        justify-content: center;
    }
}

.icon-button,
.switcher {
    border: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: 200ms;
    --wails-draggable: no-drag;
}

.icon-button {
    width: 28px;
    height: 28px;
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.34);
    color: #333;
    font-size: 18px;
}

.icon-button:hover {
    background: rgba(255, 255, 255, 0.7);
}

.quit {
    background-color: rgba(194, 50, 50, 0.6);
    color: rgba(255, 255, 255, 0.8);
}

.quit:hover {
    background-color: rgba(155, 58, 58, 0.85);
    color: #fff;
}

.switcher {
    padding: 6px;
    font-size: 18px;
    color: #333;
}

</style>
