<template>
    <section class="midi-transport" data-piano-control aria-label="MIDI 播放控制">
        <div class="transport-main">
            <NButton circle type="primary" size="large" :loading="session.loading"
                :disabled="!session.selectedId || session.busy || session.mode === 'free'"
                :aria-label="playLabel" @click="session.togglePlayback">
                <template #icon><PhPause v-if="session.state.status === 'playing'" :size="20" weight="fill" />
                    <PhPlay v-else :size="20" weight="fill" /></template>
            </NButton>
            <NButton quaternary circle :disabled="session.state.status === 'idle' || session.busy"
                aria-label="停止并回到区间起点" @click="session.stop">
                <template #icon><PhStop :size="18" weight="fill" /></template>
            </NButton>
            <div class="track-details"><strong>{{ session.selectedSong?.name || '请先选择曲目' }}</strong><span>{{ statusLabel }}</span></div>
            <div class="transport-end">
                <span class="time-readout">{{ formatMidiTime(currentValue) }} <i>/</i> {{ formatMidiTime(duration) }}</span>
                <NButton v-if="hasCustomRange" quaternary circle size="tiny" title="重置为整首曲目"
                    aria-label="重置练习区间" :disabled="!enabled" @click="resetRange">
                    <template #icon><PhArrowCounterClockwise :size="15" /></template>
                </NButton>
            </div>
        </div>

        <!-- One timeline only: click/drag to seek; drag A/B handles to set the
             practice loop. Every grid line is an actual MIDI bar boundary. -->
        <div ref="timelineRef" class="unified-timeline" role="group" aria-label="播放进度与 A–B 练习范围"
            :class="{'is-disabled': !enabled}">
            <div class="seek-surface" role="slider" :tabindex="enabled ? 0 : -1"
                aria-label="播放进度，点击或拖动定位"
                :aria-valuemin="0" :aria-valuemax="duration" :aria-valuenow="Math.round(currentValue)"
                :aria-valuetext="formatMidiTime(currentValue)"
                :aria-disabled="!enabled" @pointerdown="beginSeek"
                @pointermove="moveSeek" @pointerup="finishSeek" @pointercancel="cancelSeek"
                @keydown="seekByKeyboard">
                <div class="timeline-rail" aria-hidden="true"></div>
                <div class="selected-range" aria-hidden="true" :style="selectionStyle"></div>
                <div class="progress-fill" aria-hidden="true" :style="{width: percent(currentValue) + '%'}"></div>
                <div v-for="bar in visibleMeasures" :key="bar.index" class="measure-mark"
                    :class="{major: bar.major}" :style="{left: bar.percent + '%'}" aria-hidden="true">
                    <span v-if="bar.label" class="measure-label">{{ bar.index - 1 }}</span>
                </div>
                <div class="play-head" :style="{left: percent(currentValue) + '%'}" aria-hidden="true"></div>
            </div>
            <button v-for="side in ['A', 'B']" :key="side" type="button"
                class="range-handle" :class="side === 'A' ? 'handle-a' : 'handle-b'"
                role="slider" :tabindex="enabled ? 0 : -1" :disabled="!enabled"
                :aria-label="side === 'A' ? '练习起点 A，小节吸附' : '练习终点 B，小节吸附'"
                :aria-valuemin="Math.round(side === 'A' ? 0 : rangeValue[0])"
                :aria-valuemax="Math.round(side === 'A' ? rangeValue[1] : duration)"
                :aria-valuenow="Math.round(rangeValue[side === 'A' ? 0 : 1])"
                :aria-valuetext="formatMidiTime(rangeValue[side === 'A' ? 0 : 1])"
                :style="{left: percent(rangeValue[side === 'A' ? 0 : 1]) + '%'}"
                :title="side + ' · ' + formatMidiTime(rangeValue[side === 'A' ? 0 : 1])"
                @pointerdown="beginRange(side, $event)" @pointermove="moveRange(side, $event)"
                @pointerup="finishRange(side, $event)" @pointercancel="cancelRange"
                @keydown="rangeByKeyboard(side, $event)">
                <span>{{ side }}</span>
            </button>
        </div>
        <p class="timeline-hint" :aria-live="dragging ? 'off' : 'polite'">
            <span>{{ measures.marks.length > 1 ? "按小节划分 · 每 5 小节加粗" : "无可用小节信息" }}</span>
            <span>A {{ formatMidiTime(rangeValue[0]) }} — B {{ formatMidiTime(rangeValue[1]) }}</span>
        </p>
    </section>
</template>

<script setup>
import {computed, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {NButton} from 'naive-ui'
import {PhArrowCounterClockwise, PhPause, PhPlay, PhStop} from '@phosphor-icons/vue'
import {formatMidiTime} from '../services/midiSession'
const props = defineProps({session: {type: Object, required: true}})
const timelineRef = ref(null)
const timelineWidth = ref(600)
const seekDraft = ref(null)
const rangeDraft = ref(null)
const dragging = ref(false)
let resizeObserver
let activePointer = null
let editEpoch = 0

const duration = computed(() => Math.max(0, Number(props.session.durationMs) || 0))
const enabled = computed(() => !!props.session.selectedId && !props.session.loading
    && !props.session.busy && duration.value > 0)
const currentValue = computed(() => seekDraft.value ?? props.session.state.currentMs ?? 0)
const rangeValue = computed(() => rangeDraft.value
    ?? [props.session.options.leftMs, props.session.options.rightMs || duration.value])
const hasCustomRange = computed(() => rangeValue.value[0] > 0.5 || rangeValue.value[1] < duration.value - 0.5)
const playLabel = computed(() => props.session.state.status === 'playing' ? '暂停'
    : props.session.state.status === 'paused' ? '继续'
    : props.session.mode === 'follow' ? '开始练习' : '播放')
const statusLabel = computed(() => props.session.loading ? '正在加载曲目…'
    : props.session.state.waiting ? '等待按对琴键'
    : props.session.state.status === 'playing' ? (props.session.mode === 'follow' ? '跟弹练习中' : '正在播放')
    : props.session.state.status === 'paused' ? '已暂停'
    : props.session.selectedId ? '已就绪' : '支持 .mid / .midi')

const clamp = (value, minimum, maximum) => Math.max(minimum, Math.min(maximum, value))
function percent(value) { return duration.value > 0 ? clamp(Number(value) / duration.value * 100, 0, 100) : 0 }
const selectionStyle = computed(() => ({
    left: percent(rangeValue.value[0]) + '%',
    width: Math.max(0, percent(rangeValue.value[1]) - percent(rangeValue.value[0])) + '%',
}))
const measures = computed(() => {
    const marks = (props.session.timeline?.measures || [])
        .filter(mark => Number.isFinite(mark.ms) && mark.ms >= 0 && mark.ms < duration.value)
        .map(mark => ({index: Number(mark.index), ms: mark.ms}))
        .filter(mark => Number.isInteger(mark.index) && mark.index >= 1)
    const times = [0, ...marks.map(mark => mark.ms), duration.value].sort((a, b) => a - b)
    return {marks, times: times.filter((value, index) => index === 0 || value > times[index - 1] + .001)}
})

// Render at the available pixel density, preserving strong 5-bar lines.
// Snapping still uses *all* actual measure boundaries, including hidden lines.
const visibleMeasures = computed(() => {
    const width = Math.max(1, timelineWidth.value)
    const marks = measures.value.marks.filter(mark => mark.index > 1 && mark.ms < duration.value)
        .map(mark => ({index: mark.index, percent: percent(mark.ms),
            x: percent(mark.ms) / 100 * width, major: (mark.index - 1) % 5 === 0}))
    const majors = []
    let lastMajor = -100
    for (const mark of marks) {
        if (!mark.major || mark.x - lastMajor < 9) continue
        majors.push(mark)
        lastMajor = mark.x
    }
    const picked = [...majors]
    let lastMinor = -100, nextMajor = 0
    for (const mark of marks) {
        while (nextMajor < majors.length && majors[nextMajor].x < mark.x) nextMajor++
        const nearMajor = (nextMajor > 0 && Math.abs(majors[nextMajor - 1].x - mark.x) < 3)
            || (nextMajor < majors.length && Math.abs(majors[nextMajor].x - mark.x) < 3)
        if (mark.major || mark.x - lastMinor < 3 || nearMajor) continue
        picked.push(mark)
        lastMinor = mark.x
    }
    picked.sort((a, b) => a.index - b.index)
    let labelX = -100
    for (const mark of picked) {
        mark.label = mark.major && mark.x - labelX >= 45
        if (mark.label) labelX = mark.x
    }
    return picked
})

function fromPointer(event) {
    const rect = timelineRef.value?.getBoundingClientRect()
    if (!rect || rect.width <= 0) return 0
    return clamp((event.clientX - rect.left) / rect.width * duration.value, 0, duration.value)
}
function nearestMeasure(value, lower, upper) {
    const times = measures.value.times
    let lo = 0, hi = times.length
    while (lo < hi) {
        const middle = (lo + hi) >> 1
        if (times[middle] < value) lo = middle + 1
        else hi = middle
    }
    let best = null, distance = Infinity
    for (const index of [lo - 1, lo, 0, times.length - 1]) {
        const candidate = times[index]
        if (candidate === undefined || candidate < lower || candidate > upper) continue
        if (Math.abs(candidate - value) < distance) {
            best = candidate
            distance = Math.abs(candidate - value)
        }
    }
    // If the closest neighbor lies outside the other handle, search for the
    // actual nearest legal bar rather than inventing an off-grid position.
    if (best === null) {
        let left = 0, right = times.length
        while (left < right) {
            const mid = (left + right) >> 1
            if (times[mid] < lower) left = mid + 1
            else right = mid
        }
        best = times[left] <= upper ? times[left] : clamp(value, lower, upper)
    }
    return best
}
function beginSeek(event) {
    if (!enabled.value || event.button !== 0) return
    event.preventDefault()
    activePointer = {type: 'seek', id: event.pointerId}
    event.currentTarget.setPointerCapture(event.pointerId)
    dragging.value = true
    seekDraft.value = fromPointer(event)
}
function moveSeek(event) {
    if (activePointer?.type === 'seek' && activePointer.id === event.pointerId)
        seekDraft.value = fromPointer(event)
}
async function finishSeek(event) {
    if (activePointer?.type !== 'seek' || activePointer.id !== event.pointerId) return
    const target = seekDraft.value
    activePointer = null
    dragging.value = false
    const epoch = editEpoch
    if (target !== null) await props.session.seek(target)
    if (epoch === editEpoch && seekDraft.value === target) seekDraft.value = null
}
function cancelSeek() {
    if (activePointer?.type !== 'seek') return
    activePointer = null
    dragging.value = false
    seekDraft.value = null
}
function beginRange(side, event) {
    if (!enabled.value || event.button !== 0) return
    event.preventDefault()
    event.stopPropagation()
    event.currentTarget.setPointerCapture(event.pointerId)
    activePointer = {type: side, id: event.pointerId}
    dragging.value = true
    rangeDraft.value = rangeValue.value.slice()
}
function moveRange(side, event) {
    if (activePointer?.type !== side || activePointer.id !== event.pointerId) return
    const next = rangeDraft.value?.slice()
    if (!next) return
    const minimumGap = Math.min(1, duration.value)
    if (side === 'A') next[0] = nearestMeasure(fromPointer(event), 0, next[1] - minimumGap)
    else next[1] = nearestMeasure(fromPointer(event), next[0] + minimumGap, duration.value)
    rangeDraft.value = next
}
async function finishRange(side, event) {
    if (activePointer?.type !== side || activePointer.id !== event.pointerId) return
    const target = rangeDraft.value
    activePointer = null
    dragging.value = false
    const epoch = editEpoch
    if (target) await props.session.setRange(target[0], target[1])
    if (epoch === editEpoch && rangeDraft.value === target) rangeDraft.value = null
}
function cancelRange() {
    if (activePointer?.type === 'seek') return
    activePointer = null
    dragging.value = false
    rangeDraft.value = null
}
function seekByKeyboard(event) {
    if (!enabled.value || !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
    event.preventDefault()
    event.stopPropagation()
    const times = measures.value.times
    const min = rangeValue.value[0], max = rangeValue.value[1]
    let target = event.key === 'Home' ? min : event.key === 'End' ? max : currentValue.value
    if (event.key === 'ArrowRight') target = times.find(ms => ms > currentValue.value + .5 && ms <= max) ?? max
    if (event.key === 'ArrowLeft') target = times.slice().reverse().find(ms => ms < currentValue.value - .5 && ms >= min) ?? min
    void props.session.seek(target)
}
function rangeByKeyboard(side, event) {
    if (!enabled.value || !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
    event.preventDefault()
    event.stopPropagation()
    const index = side === 'A' ? 0 : 1
    const range = rangeValue.value.slice()
    const minGap = Math.min(1, duration.value)
    const lower = side === 'A' ? 0 : range[0] + minGap
    const upper = side === 'A' ? range[1] - minGap : duration.value
    let target = range[index]
    if (event.key === 'Home') target = nearestMeasure(lower, lower, upper)
    if (event.key === 'End') target = nearestMeasure(upper, lower, upper)
    if (event.key === 'ArrowRight') target = nearestMeasure(target + Math.max(1, duration.value / 1000), Math.min(target + .001, upper), upper)
    if (event.key === 'ArrowLeft') target = nearestMeasure(target - Math.max(1, duration.value / 1000), lower, Math.max(target - .001, lower))
    if (target === range[index]) return
    range[index] = target
    void props.session.setRange(range[0], range[1])
}
function resetRange() {
    if (enabled.value) void props.session.setRange(0, duration.value)
}
function clearDrafts() {
    ++editEpoch
    activePointer = null
    dragging.value = false
    seekDraft.value = rangeDraft.value = null
}
watch(() => props.session.selectedId, clearDrafts, {flush: 'sync'})
watch(() => props.session.loading, loading => { if (loading) clearDrafts() }, {flush: 'sync'})
onMounted(() => {
    resizeObserver = new ResizeObserver(entries => {
        for (const entry of entries) timelineWidth.value = entry.contentRect.width
    })
    if (timelineRef.value) resizeObserver.observe(timelineRef.value)
})
onBeforeUnmount(() => { resizeObserver?.disconnect(); clearDrafts() })
</script>

<style scoped>
.midi-transport { width: min(680px, calc(100vw - 40px)); padding: 14px 19px 9px; border: 1px solid #ffffff1a; border-radius: 11px; background: #242424ed; backdrop-filter: blur(18px); box-shadow: 0 12px 45px #0003; color: #e8e8e8; }
.transport-main { display: flex; align-items: center; gap: 12px; margin-bottom: 14px; }
.track-details { display: flex; flex: 1; flex-direction: column; min-width: 0; gap: 7px; }
.track-details strong { font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.track-details span { font-size: 10px; color: #a2a2a2; }
.transport-end { display: flex; align-items: center; gap: 7px; }
.time-readout { font-size: 11px; font-variant-numeric: tabular-nums; white-space: nowrap; color: #c0c0c0; }
.time-readout i { color: #656565; font-style: normal; margin: 0 4px; }
.unified-timeline { position: relative; height: 35px; margin: 6px 4px 1px; touch-action: none; user-select: none; }
.unified-timeline.is-disabled { opacity: .48; }
.seek-surface { position: absolute; inset: 0; cursor: crosshair; border-radius: 8px; touch-action: none; }
.seek-surface:focus-visible { outline: 2px solid var(--ui-accent, #c4c4c4); outline-offset: 3px; }
.timeline-rail, .selected-range { position: absolute; top: 13px; height: 9px; border-radius: 3px; }
.timeline-rail { left: 0; right: 0; background: #555555; box-shadow: inset 0 1px 3px #0008; }
.selected-range { background: var(--ui-accent-soft, #a0a0a040); box-shadow: inset 0 0 0 1px var(--ui-accent, #b5b5b5); opacity: .8; }
.progress-fill { position: absolute; top: 19px; left: 0; height: 3px; background: var(--ui-accent, #cecece); opacity: .9; }
.measure-mark { position: absolute; top: 13px; width: 1px; height: 9px; background: #ffffff47; pointer-events: none; }
.measure-mark.major { top: 10px; width: 2px; height: 15px; background: #eaeaeaab; }
.measure-label { position: absolute; bottom: 16px; left: 0; transform: translateX(-50%); font-size: 9px; color: #bababa; font-variant-numeric: tabular-nums; }
.play-head { position: absolute; top: 5px; width: 2px; height: 25px; background: #eeeeee; transform: translateX(-50%); box-shadow: 0 0 7px #ffffff73; pointer-events: none; }
.range-handle { position: absolute; top: 4px; width: 18px; height: 27px; padding: 0; transform: translateX(-50%); border: 1px solid var(--ui-accent, #c5c5c5); border-radius: 5px; color: #202020; background: #d0d0d0; box-shadow: 0 1px 7px #0008; cursor: ew-resize; touch-action: none; z-index: 2; font-size: 10px; font-weight: 700; }
.range-handle span { pointer-events: none; }
.range-handle:focus-visible { outline: 2px solid #fff; outline-offset: 3px; }
.range-handle:disabled { cursor: default; }
.timeline-hint { display: flex; justify-content: space-between; gap: 12px; margin: 1px 0 0; font-size: 10px; color: #999; font-variant-numeric: tabular-nums; }
@media (max-width: 600px) {
    .midi-transport { padding: 10px 12px 7px; }
    .transport-main { gap: 7px; }
    .timeline-hint span:first-child { display: none; }
}
@media (prefers-reduced-motion: reduce) { .range-handle, .play-head { transition: none; } }
</style>
