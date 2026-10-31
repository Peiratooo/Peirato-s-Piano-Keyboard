<template>
    <canvas ref="canvas" class="waterfall-canvas" aria-hidden="true" />
</template>

<script setup>
import {computed, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {getLivePerformanceFrame, subscribeLivePerformance} from '../services/livePerformance'

const props = defineProps({
    layout: {type: Array, default: () => []},
    width: {type: Number, default: 0},
    height: {type: Number, default: 0},
    active: {type: Boolean, default: true},
    colors: {type: Object, default: () => ({})},
})

const canvas = ref(null)
let frameRequest = 0
let context = null
let mounted = false
let unsubscribe = null
const keyLookup = computed(() => new Map(props.layout.map(key => [key.note, key])))

function noteColor(isBlack) {
    const value = props.colors[isBlack ? 'blackKey' : 'whiteKey']
    return (typeof value === 'string' ? value : value?.color) || (isBlack ? '#70da98' : '#9af7b3')
}

function resizeCanvas() {
    if (!canvas.value) return
    const width = Math.max(0, props.width)
    const height = Math.max(0, props.height)
    const ratio = Math.min(window.devicePixelRatio || 1, 2)
    canvas.value.width = Math.round(width * ratio)
    canvas.value.height = Math.round(height * ratio)
    canvas.value.style.width = `${width}px`
    canvas.value.style.height = `${height}px`
    context = canvas.value.getContext('2d')
    context?.setTransform(ratio, 0, 0, ratio, 0, 0)
}

function roundedRect(x, y, width, height, radius) {
    const r = Math.min(radius, width / 2, height / 2)
    context.beginPath()
    context.moveTo(x + r, y)
    context.arcTo(x + width, y, x + width, y + height, r)
    context.arcTo(x + width, y + height, x, y + height, r)
    context.arcTo(x, y + height, x, y, r)
    context.arcTo(x, y, x + width, y, r)
    context.closePath()
}

function drawSegment(key, from, to, now, historyMs, color, alpha, glow = false) {
    const top = props.height - (now - from) / historyMs * props.height
    const bottom = props.height - (now - to) / historyMs * props.height
    if (bottom < 0 || top > props.height) return
    const y = Math.max(-6, top)
    const height = Math.min(props.height, Math.max(bottom, top + 3)) - y
    if (height <= 0) return
    const inset = 1.5
    const width = Math.max(2, key.width - inset * 2)
    const x = key.x + inset
    context.globalAlpha = alpha
    context.fillStyle = color
    context.shadowColor = color
    context.shadowBlur = glow ? 5 : 0
    roundedRect(x, y, width, height, Math.min(3, width / 4))
    context.fill()
    context.shadowBlur = 0
    if (glow) {
        context.strokeStyle = 'rgba(255,255,255,.58)'
        context.lineWidth = 1
        context.stroke()
    }
    context.globalAlpha = 1
}

function render(now) {
    if (!context || props.width <= 0 || props.height <= 0) return
    const width = props.width
    const height = props.height
    context.clearRect(0, 0, width, height)
    const background = context.createLinearGradient(0, 0, 0, height)
    background.addColorStop(0, '#242424')
    background.addColorStop(1, '#202020')
    context.fillStyle = background
    context.fillRect(0, 0, width, height)

    const keys = keyLookup.value
    for (const key of props.layout) {
        if (key.isBlack) continue
        context.strokeStyle = key.note % 12 === 0 ? 'rgba(235,235,235,.13)' : 'rgba(235,235,235,.045)'
        context.beginPath()
        context.moveTo(Math.round(key.x) + .5, 0)
        context.lineTo(Math.round(key.x) + .5, height)
        context.stroke()
    }

    const frame = getLivePerformanceFrame(now)
    for (const voice of frame.notes) {
        const key = keys.get(voice.note)
        if (!key) continue
        const color = noteColor(key.isBlack)
        // A faint tail keeps pedal resonance distinct from physical key length.
        if (voice.releasedAt !== null && (voice.endedAt === null || voice.endedAt > voice.releasedAt)) {
            drawSegment(key, voice.releasedAt, voice.endedAt ?? frame.nowMs, frame.nowMs, frame.historyMs, color, .18)
        }
        drawSegment(key, voice.startedAt, voice.releasedAt ?? frame.nowMs,
            frame.nowMs, frame.historyMs, color, .64 + voice.velocity / 127 * .3, true)
    }

    const edge = context.createLinearGradient(0, height - 22, 0, height)
    edge.addColorStop(0, 'rgba(210,210,210,0)')
    edge.addColorStop(1, 'rgba(210,210,210,.04)')
    context.fillStyle = edge
    context.fillRect(0, Math.max(0, height - 22), width, 22)
    context.strokeStyle = 'rgba(210,210,210,.45)'
    context.beginPath()
    context.moveTo(0, height - .5)
    context.lineTo(width, height - .5)
    context.stroke()
    // Nothing visible: suspend rAF until an input event arrives.
    return frame.notes.length > 0
}

function tick(now) {
    frameRequest = 0
    if (!props.active || !mounted) return
    if (render(now)) frameRequest = requestAnimationFrame(tick)
}

function syncAnimation() {
    if (frameRequest) cancelAnimationFrame(frameRequest)
    frameRequest = 0
    if (mounted && props.active && render(performance.now())) {
        frameRequest = requestAnimationFrame(tick)
    }
}

watch(() => [props.width, props.height], () => {
    resizeCanvas()
    if (props.active) syncAnimation()
})
watch(() => props.active, syncAnimation)
watch(() => [props.layout, props.colors], () => {
    if (props.active) syncAnimation()
}, {deep: true})

onMounted(() => {
    mounted = true
    resizeCanvas()
    unsubscribe = subscribeLivePerformance(() => {
        if (props.active && !frameRequest) syncAnimation()
    })
    window.addEventListener('resize', resizeCanvas)
    syncAnimation()
})

onBeforeUnmount(() => {
    mounted = false
    if (frameRequest) cancelAnimationFrame(frameRequest)
    unsubscribe?.()
    window.removeEventListener('resize', resizeCanvas)
})
</script>

<style scoped>
.waterfall-canvas {
    display: block;
    pointer-events: none;
}
</style>
