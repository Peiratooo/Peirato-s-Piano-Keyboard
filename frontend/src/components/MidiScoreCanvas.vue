<template>
    <canvas ref="canvas" class="midi-score" :aria-label="description" role="img" />
</template>

<script setup>
import {computed, onBeforeUnmount, onMounted, ref, watch} from 'vue'

const props = defineProps({
    timeline: {type: Object, default: null},
    playerState: {type: Object, default: () => ({})},
    width: {type: Number, default: 0},
    height: {type: Number, default: 0},
    active: {type: Boolean, default: true},
    handMode: {type: String, default: 'both'},
})
// A lightweight performance-notation view for the first implementation.
// MIDI lacks pitch spelling, rests, beams and voices. Do not treat this as
// a faithful engraved score; MusicXML engraving will use original notation.
const canvas = ref(null)
const description = computed(() => '随播放从右向左滚动的五线谱演奏提示')
const notes = computed(() => (props.timeline?.notes || [])
    .filter(n => Number.isFinite(n.startMs) && Number.isFinite(n.endMs) && n.endMs > n.startMs)
    .slice().sort((a, b) => a.startMs - b.startMs))
const measures = computed(() => (props.timeline?.measures || [])
    .filter(m => Number.isFinite(m.ms)).slice().sort((a, b) => a.ms - b.ms))
const pitchNames = ['C', 'C♯', 'D', 'D♯', 'E', 'F', 'F♯', 'G', 'G♯', 'A', 'A♯', 'B']
const diatonic = [0, 0, 1, 1, 2, 3, 3, 4, 4, 5, 5, 6]
let ctx, raf = 0, mounted = false, clock = {}, revision = null
function captureClock() {
    const state = props.playerState || {}
    const now = performance.now()
    const received = Number(state.receivedAt)
    const discontinuity = revision !== state.clockRevision
    revision = state.clockRevision
    clock = {state: {...state}, ms: Math.max(0, Number(state.currentMs) || 0),
        at: !discontinuity && Number.isFinite(received) ? Math.min(now, received) : now}
    if (state.waiting && Number.isFinite(state.currentStep?.ms)) clock.ms = state.currentStep.ms
}
function currentMs(now) {
    const state = clock.state || {}
    if (state.scrubbing || state.waiting) return clock.ms
    if (state.leadInMs > 0) {
        const elapsed = state.status === 'playing' ? Math.min(200, Math.max(0, now - clock.at)) : 0
        return clock.ms - Math.max(0, state.leadInMs - elapsed * (Number(state.speed) || 1))
    }
    let ms = clock.ms
    if (state.status === 'playing') ms += Math.min(200, Math.max(0, now - clock.at)) * (Number(state.speed) || 1)
    if (state.mode === 'follow' && Number.isFinite(state.currentStep?.ms)
        && state.currentStep.ms >= clock.ms) ms = Math.min(ms, state.currentStep.ms)
    return Math.min(Math.max(ms, 0), Number(state.rightMs) || props.timeline?.durationMs || ms)
}
function resize() {
    if (!canvas.value) return
    const ratio = Math.min(window.devicePixelRatio || 1, 2)
    canvas.value.width = Math.round(props.width * ratio)
    canvas.value.height = Math.round(props.height * ratio)
    canvas.value.style.width = props.width + 'px'
    canvas.value.style.height = props.height + 'px'
    ctx = canvas.value.getContext('2d')
    ctx?.setTransform(ratio, 0, 0, ratio, 0, 0)
}
function pitchStep(midi) {
    const octave = Math.floor(midi / 12) - 1
    return octave * 7 + diatonic[((midi % 12) + 12) % 12]
}
function drawStaff(top, width, clef) {
    ctx.strokeStyle = '#747474'
    ctx.lineWidth = 1
    for (let i = 0; i < 5; i++) {
        const y = top + i * 11 + .5
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(width, y); ctx.stroke()
    }
    ctx.fillStyle = '#d8d8d8'
    ctx.font = '39px "Segoe UI Symbol", "Noto Music", serif'
    ctx.fillText(clef === 'treble' ? '𝄞' : '𝄢', 8, top + (clef === 'treble' ? 43 : 36))
}
function drawNote(n, x, top, clef, time) {
    const bottom = top + 44
    const base = clef === 'treble' ? 30 : 18 // E4 or G2, diatonic steps
    const y = bottom - (pitchStep(n.note) - base) * 5.5
    const duration = n.endMs - n.startMs
    const state = clock.state || {}
    const expected = state.mode === 'follow' && state.currentStep?.notes?.some(t =>
        t.note === n.note && t.trackIndex === n.trackIndex && Math.abs(t.ms - n.startMs) < 1)
    const active = n.startMs <= time && n.endMs > time
    const color = expected && state.waiting ? '#ffd5a7' : active ? '#f3ad83' : clef === 'bass' ? '#a8cbdc' : '#e7e7e7'
    ctx.strokeStyle = color
    ctx.fillStyle = color
    ctx.lineWidth = 1.3
    // Ledger lines for notes outside the staff.
    for (let ledger = bottom + 11; ledger <= y + 1; ledger += 11) {
        ctx.beginPath(); ctx.moveTo(x - 10, ledger); ctx.lineTo(x + 10, ledger); ctx.stroke()
    }
    for (let ledger = top - 11; ledger >= y - 1; ledger -= 11) {
        ctx.beginPath(); ctx.moveTo(x - 10, ledger); ctx.lineTo(x + 10, ledger); ctx.stroke()
    }
    const filled = duration < 1450
    ctx.beginPath()
    ctx.ellipse(x, y, 6.7, 4.3, -.25, 0, Math.PI * 2)
    if (filled) ctx.fill(); else ctx.stroke()
    if (duration < 2500) {
        const down = y < top + 22
        ctx.beginPath()
        ctx.moveTo(x + (down ? -6 : 6), y)
        ctx.lineTo(x + (down ? -6 : 6), y + (down ? 29 : -29))
        ctx.stroke()
    }
    if (pitchNames[n.note % 12]?.includes('♯')) {
        ctx.font = '14px Georgia, serif'
        ctx.fillText('♯', x - 18, y + 4)
    }
}
function lowerBound(list, value, key) {
    let a = 0, b = list.length
    while (a < b) { const m = (a + b) >> 1; if (list[m][key] < value) a = m + 1; else b = m }
    return a
}
function render(now) {
    if (!ctx || props.width <= 0 || props.height <= 0) return
    const w = props.width, h = props.height
    ctx.fillStyle = '#1d1d1d'
    ctx.fillRect(0, 0, w, h)
    const time = currentMs(now), speed = Math.max(.25, Number(clock.state?.speed) || 1)
    const pxPerMs = Math.max(0.025, w * .72 / (6000 * speed))
    const playX = Math.max(88, Math.min(w * .27, 250))
    const hands = props.handMode === 'both' ? ['right', 'left'] : [props.handMode]
    const staffGap = h / (hands.length + .25)
    const staffTops = hands.map((_, i) => (i + .5) * staffGap - 23)
    for (let i = 0; i < hands.length; i++) drawStaff(staffTops[i], w, hands[i] === 'left' ? 'bass' : 'treble')
    const from = time - playX / pxPerMs - 60
    const to = time + (w - playX) / pxPerMs + 60
    const marks = measures.value
    for (let i = lowerBound(marks, from, 'ms'); i < marks.length && marks[i].ms <= to; i++) {
        const x = playX + (marks[i].ms - time) * pxPerMs
        ctx.strokeStyle = '#595959'
        ctx.lineWidth = 1
        for (const top of staffTops) {
            ctx.beginPath(); ctx.moveTo(x, top); ctx.lineTo(x, top + 44); ctx.stroke()
        }
        ctx.font = '11px "Segoe UI", sans-serif'
        ctx.fillStyle = '#8f8f8f'
        ctx.fillText(String(marks[i].index), x + 4, 12)
    }
    const list = notes.value
    for (let i = Math.max(0, lowerBound(list, from, 'startMs') - 1); i < list.length && list[i].startMs <= to; i++) {
        const n = list[i], hand = n.hand === 'left' ? 'left' : 'right'
        const idx = hands.indexOf(hand)
        if (idx < 0) continue
        const x = playX + (n.startMs - time) * pxPerMs
        if (x < -24 || x > w + 24) continue
        drawNote(n, x, staffTops[idx], hand === 'left' ? 'bass' : 'treble', time)
    }
    ctx.strokeStyle = '#e9a57d'
    ctx.lineWidth = 2
    ctx.beginPath(); ctx.moveTo(playX, 0); ctx.lineTo(playX, h); ctx.stroke()
}
function animating() {
    return clock.state?.status === 'playing' && !clock.state?.waiting && !clock.state?.scrubbing
}
function frame(now) {
    raf = 0
    if (!mounted || !props.active) return
    render(now)
    if (animating()) raf = requestAnimationFrame(frame)
}
function sync() {
    if (raf) cancelAnimationFrame(raf)
    raf = 0
    if (!mounted || !props.active) return
    captureClock()
    render(performance.now())
    if (animating()) raf = requestAnimationFrame(frame)
}
watch(() => props.playerState, sync, {deep: true})
watch(() => props.active, sync)
watch(() => [props.width, props.height], () => {resize(); sync()})
watch(() => [props.timeline, props.handMode], sync)
onMounted(() => {mounted = true; resize(); sync()})
onBeforeUnmount(() => {mounted = false; if (raf) cancelAnimationFrame(raf)})
</script>

<style scoped>
.midi-score { display: block; pointer-events: none; }
</style>
