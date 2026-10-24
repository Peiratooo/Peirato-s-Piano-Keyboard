<template><canvas ref="canvas" class="midi-waterfall" aria-hidden="true" /></template>

<script setup>
import {computed, onBeforeUnmount, onMounted, ref, watch} from 'vue'

const props = defineProps({
    layout: {type: Array, default: () => []},
    width: {type: Number, default: 0},
    height: {type: Number, default: 0},
    active: {type: Boolean, default: true},
    colors: {type: Object, default: () => ({})},
    timeline: {type: Object, default: null},
    playerState: {type: Object, default: () => ({})},
})
const PREVIEW_MS = 6000
const canvas = ref(null)
let context, frameRequest = 0, mounted = false
let clock = {ms: 0, at: 0, state: {}}
let revision = null
const keys = computed(() => new Map(props.layout.map(key => [key.note, key])))

// An interval tree keeps long songs cheap to draw, including notes held across
// many seconds. Rendering visits only intersecting notes and their ancestors.
const noteTree = computed(() => {
    const notes = (props.timeline?.notes || []).filter(note => Number.isFinite(note.startMs)
        && Number.isFinite(note.endMs) && note.endMs > note.startMs).slice().sort((a, b) => a.startMs - b.startMs)
    function build(left, right) {
        if (left >= right) return null
        const middle = (left + right) >> 1
        const first = build(left, middle), last = build(middle + 1, right), note = notes[middle]
        return {note, left: first, right: last, maxEnd: Math.max(note.endMs, first?.maxEnd || 0, last?.maxEnd || 0)}
    }
    return build(0, notes.length)
})
function visibleNotes(node, from, to, output) {
    if (!node || node.maxEnd < from) return
    visibleNotes(node.left, from, to, output)
    if (node.note.startMs > to) return
    if (node.note.endMs >= from) output.push(node.note)
    visibleNotes(node.right, from, to, output)
}
function captureClock() {
    const state = props.playerState || {}
    const now = performance.now()
    const discontinuity = revision !== state.clockRevision
    revision = state.clockRevision
    const received = Number(state.receivedAt)
    clock = {ms: Math.max(0, Number(state.currentMs) || 0),
        at: !discontinuity && Number.isFinite(received) ? Math.min(now, received) : now, state: {...state}}
    if (state.waiting && Number.isFinite(state.currentStep?.ms)) clock.ms = state.currentStep.ms
}
function currentMs(now) {
    const state = clock.state
    let current = clock.ms
    if (state.status === 'playing' && !state.waiting) {
        current += Math.min(200, Math.max(0, now - clock.at)) * (Number(state.speed) || 1)
        // The next follow step is a hard visual boundary until the backend
        // confirms it was accepted, even if its waiting event is still in IPC.
        if (state.mode === 'follow' && Number.isFinite(state.currentStep?.ms)
            && state.currentStep.ms >= clock.ms) current = Math.min(current, state.currentStep.ms)
    }
    return Math.max(0, Math.min(current, Number(state.rightMs) || props.timeline?.durationMs || current))
}
function resize() {
    if (!canvas.value) return
    const ratio = Math.min(window.devicePixelRatio || 1, 2)
    const width = Math.max(0, props.width), height = Math.max(0, props.height)
    canvas.value.width = Math.round(width * ratio)
    canvas.value.height = Math.round(height * ratio)
    canvas.value.style.width = `${width}px`
    canvas.value.style.height = `${height}px`
    context = canvas.value.getContext('2d')
    context?.setTransform(ratio, 0, 0, ratio, 0, 0)
}
function colorFor(note, key) {
    const token = `${key.isBlack ? 'blackKey' : 'whiteKey'}${note.hand === 'left' ? 'Left' : ''}`
    const value = props.colors[token]
    return (typeof value === 'string' ? value : value?.color) || (note.hand === 'left' ? '#efbd82' : '#9af7b3')
}
function roundedRect(x, y, width, height) {
    const radius = Math.min(5, width / 4, height / 2)
    context.beginPath()
    context.moveTo(x + radius, y)
    context.arcTo(x + width, y, x + width, y + height, radius)
    context.arcTo(x + width, y + height, x, y + height, radius)
    context.arcTo(x, y + height, x, y, radius)
    context.arcTo(x, y, x + width, y, radius)
    context.closePath()
}
function drawNote(note, time) {
    const key = keys.value.get(note.note)
    if (!key) return
    const state = clock.state
    const left = Number(state.leftMs) || 0, right = Number(state.rightMs) || props.timeline?.durationMs || Infinity
    if (note.startMs < left || note.startMs >= right) return
    const expected = state.mode === 'follow' && state.currentStep?.notes?.some(target => target.note === note.note
        && target.trackIndex === note.trackIndex && Math.abs(target.ms - note.startMs) < 1)
    const start = state.waiting && expected ? state.currentStep.ms : note.startMs
    const end = Math.min(right, start + note.endMs - note.startMs)
    const top = props.height - (end - time) / PREVIEW_MS * props.height
    const bottom = props.height - (start - time) / PREVIEW_MS * props.height
    if (bottom < 0 || top > props.height) return
    const y = Math.max(-6, top), height = Math.min(props.height, bottom) - y
    if (height <= 0) return
    const inset = Math.min(3, key.width * .12), width = Math.max(2, key.width - inset * 2), x = key.x + inset
    const muted = state.mutedTracks?.[note.trackIndex] || state.mutedHands?.[note.hand]
        || (state.mode === 'play' && state.handMode !== 'both' && note.hand !== state.handMode)
    const color = colorFor(note, key)
    context.globalAlpha = muted ? .18 : .65 + Math.min(127, Number(note.velocity) || 80) / 127 * .3
    context.fillStyle = color
    context.shadowColor = color
    context.shadowBlur = expected && state.waiting ? 18 : 7
    roundedRect(x, y, width, Math.max(3, height))
    context.fill()
    context.shadowBlur = 0
    context.strokeStyle = expected && state.waiting ? 'rgba(255,255,255,.92)' : 'rgba(255,255,255,.4)'
    context.lineWidth = expected && state.waiting ? 1.5 : 1
    context.stroke()
    context.globalAlpha = 1
}
function render(now) {
    if (!context || props.width <= 0 || props.height <= 0) return
    context.clearRect(0, 0, props.width, props.height)
    const background = context.createLinearGradient(0, 0, 0, props.height)
    background.addColorStop(0, '#202020')
    background.addColorStop(1, '#171717')
    context.fillStyle = background
    context.fillRect(0, 0, props.width, props.height)
    for (const key of props.layout) {
        if (key.isBlack) continue
        context.strokeStyle = key.note % 12 === 0 ? 'rgba(255,255,255,.13)' : 'rgba(255,255,255,.045)'
        context.beginPath()
        context.moveTo(Math.round(key.x) + .5, 0)
        context.lineTo(Math.round(key.x) + .5, props.height)
        context.stroke()
    }
    const time = currentMs(now), visible = []
    visibleNotes(noteTree.value, time, time + PREVIEW_MS, visible)
    for (const note of visible) drawNote(note, time)
    context.strokeStyle = 'rgba(255,255,255,.32)'
    context.beginPath()
    context.moveTo(0, props.height - .5)
    context.lineTo(props.width, props.height - .5)
    context.stroke()
}
function isAnimating() {
    return clock.state.status === 'playing' && !clock.state.waiting
}
function tick(now) {
    frameRequest = 0
    if (!mounted || !props.active) return
    render(now)
    if (isAnimating()) frameRequest = requestAnimationFrame(tick)
}
function syncAnimation() {
    if (frameRequest) cancelAnimationFrame(frameRequest)
    frameRequest = 0
    if (mounted && props.active) {
        captureClock()
        render(performance.now())
        if (isAnimating()) frameRequest = requestAnimationFrame(tick)
    }
}
watch(() => props.playerState, () => {
    captureClock()
    if (!mounted || !props.active) return
    if (!isAnimating() || !frameRequest) syncAnimation()
}, {deep: true})
watch(() => props.active, syncAnimation)
watch(() => [props.width, props.height], () => { resize(); if (props.active) render(performance.now()) })
watch(() => [props.timeline, props.layout, props.colors], () => { if (props.active) render(performance.now()) })
onMounted(() => { mounted = true; resize(); window.addEventListener('resize', resize); syncAnimation() })
onBeforeUnmount(() => { mounted = false; if (frameRequest) cancelAnimationFrame(frameRequest); window.removeEventListener('resize', resize) })
</script>

<style scoped>
.midi-waterfall { display: block; pointer-events: none; }
</style>
