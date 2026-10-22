<template>
    <section class="midi-transport" data-piano-control aria-label="MIDI 播放控制">
        <div class="transport-main">
            <NButton circle type="primary" size="large" :loading="session.loading" :disabled="!session.selectedId || session.busy || session.mode === 'free'" :aria-label="playLabel" @click="session.togglePlayback">
                <template #icon><PhPause v-if="session.state.status === 'playing'" :size="20" weight="fill" /><PhPlay v-else :size="20" weight="fill" /></template>
            </NButton>
            <NButton quaternary circle :disabled="session.state.status === 'idle' || session.busy" aria-label="停止并回到区间起点" @click="session.stop"><template #icon><PhStop :size="18" weight="fill" /></template></NButton>
            <div class="track-details"><strong>{{ session.selectedSong?.name || '请先选择曲目' }}</strong><span>{{ statusLabel }}</span></div>
            <span class="time-readout">{{ formatMidiTime(currentValue) }} <i>/</i> {{ formatMidiTime(session.durationMs) }}</span>
            <NButton quaternary circle size="small" :disabled="!session.selectedId || session.busy" aria-label="重置练习区间为整首曲目" title="重置为整首曲目" @click="resetRange"><template #icon><PhArrowCounterClockwise :size="15" /></template></NButton>
        </div>

        <div ref="timelineRef" class="timeline-control" :class="{disabled: !session.selectedId || session.loading || session.busy}"
            :title="gridTitle" aria-label="播放进度与小节练习区间"
            @pointerdown="onTrackPointerDown" @pointermove="onPointerMove" @pointerup="onPointerUp" @pointercancel="onPointerUp">
            <div class="timeline-rail">
                <div class="played-progress" :style="{width: currentPercent + '%'}"></div>
                <div class="range-highlight" :style="{left: rangeStartPercent + '%', width: Math.max(0, rangeEndPercent - rangeStartPercent) + '%'}"></div>
            </div>

            <div v-for="marker in visibleMeasureMarkers" :key="marker.index" class="measure-marker"
                :class="{major: marker.major}" :style="{left: marker.percent + '%'}" aria-hidden="true">
                <span v-if="marker.major" class="measure-label">{{ marker.label }}</span>
            </div>

            <div class="playhead" :style="{left: currentPercent + '%'}" aria-hidden="true"><i></i></div>

            <button class="range-handle handle-a" :style="{left: rangeStartPercent + '%'}" type="button" role="slider"
                aria-label="练习起点 A" :aria-valuemin="0" :aria-valuemax="session.durationMs" :aria-valuenow="rangeValue[0]"
                :aria-valuetext="'第 ' + leftMeasureNumber + ' 小节起点，' + formatMidiTime(rangeValue[0])"
                :disabled="!session.selectedId || session.loading || session.busy"
                @pointerdown.stop.prevent="beginRangeDrag('left', $event)" @keydown.stop="onHandleKeydown('left', $event)">A</button>
            <button class="range-handle handle-b" :style="{left: rangeEndPercent + '%'}" type="button" role="slider"
                aria-label="练习终点 B" :aria-valuemin="0" :aria-valuemax="session.durationMs" :aria-valuenow="rangeValue[1]"
                :aria-valuetext="'第 ' + rightMeasureNumber + ' 小节边界，' + formatMidiTime(rangeValue[1])"
                :disabled="!session.selectedId || session.loading || session.busy"
                @pointerdown.stop.prevent="beginRangeDrag('right', $event)" @keydown.stop="onHandleKeydown('right', $event)">B</button>
        </div>
        <p v-if="approximateGrid && session.selectedId" class="grid-hint">当前 MIDI 未提供可用的小节信息，网格按 4/4 拍近似划分。</p>
    </section>
</template>

<script setup>
import {computed, onBeforeUnmount, ref, watch} from 'vue'
import {NButton} from 'naive-ui'
import {PhArrowCounterClockwise, PhPause, PhPlay, PhStop} from '@phosphor-icons/vue'
import {formatMidiTime} from '../services/midiSession'

const props = defineProps({session: {type: Object, required: true}})
const timelineRef = ref(null)
const rangeDraft = ref(null)
const dragEdge = ref('')
const currentValue = computed(() => Math.max(0, Number(props.session.state.currentMs) || 0))
const duration = computed(() => Math.max(0, Number(props.session.durationMs) || 0))
const rangeValue = computed(() => rangeDraft.value || [
    Math.max(0, Number(props.session.options.leftMs) || 0),
    Number(props.session.options.rightMs) || duration.value,
])
const playLabel = computed(() => props.session.state.status === 'playing' ? '暂停' : props.session.state.status === 'paused' ? '继续' : props.session.mode === 'follow' ? '开始练习' : '播放')
const baseStatusLabel = computed(() => props.session.loading ? '正在加载曲目…' : props.session.state.waiting ? '等待按对琴键' : props.session.state.status === 'playing' ? (props.session.mode === 'follow' ? '跟弹练习中' : '正在播放') : props.session.state.status === 'paused' ? '已暂停' : props.session.selectedId ? '已就绪' : '支持 .mid / .midi')

const approximateGrid = computed(() => !!props.session.timeline?.measureGridApproximate
    || !Array.isArray(props.session.timeline?.measureMarkers) || !props.session.timeline.measureMarkers.length)
const measureMarkers = computed(() => {
    const end = duration.value
    const source = props.session.timeline?.measureMarkers
    if (Array.isArray(source) && source.length) {
        return [...new Set(source.map(Number).filter(ms => Number.isFinite(ms) && ms >= 0 && ms < end + .5))]
            .sort((a, b) => a - b)
    }
    const fallback = []
    for (let ms = 0; ms < end && fallback.length < 10000; ms += 2000) fallback.push(ms)
    return fallback.length ? fallback : [0]
})
const barBoundaries = computed(() => {
    const values = [...measureMarkers.value, duration.value].filter(ms => Number.isFinite(ms) && ms >= 0 && ms <= duration.value)
    values.sort((a, b) => a - b)
    return values.filter((value, index) => index === 0 || value - values[index - 1] > .5)
})
const visibleMeasureMarkers = computed(() => {
    const end = duration.value
    if (end <= 0) return []
    return measureMarkers.value.map((ms, index) => ({
        index, percent: clamp(ms / end * 100, 0, 100),
        major: index > 0 && index % 5 === 0,
        label: String(index),
    })).filter(marker => marker.percent < 99.95)
})
const currentPercent = computed(() => duration.value > 0 ? clamp(currentValue.value / duration.value * 100, 0, 100) : 0)
const rangeStartPercent = computed(() => duration.value > 0 ? clamp(rangeValue.value[0] / duration.value * 100, 0, 100) : 0)
const rangeEndPercent = computed(() => duration.value > 0 ? clamp(rangeValue.value[1] / duration.value * 100, 0, 100) : 100)
const leftBoundaryIndex = computed(() => nearestBoundaryIndex(rangeValue.value[0]))
const rightBoundaryIndex = computed(() => nearestBoundaryIndex(rangeValue.value[1]))
const leftMeasureNumber = computed(() => Math.max(1, leftBoundaryIndex.value + 1))
const rightMeasureNumber = computed(() => Math.max(leftMeasureNumber.value, rightBoundaryIndex.value))
const rangeDescription = computed(() => leftMeasureNumber.value + '–' + rightMeasureNumber.value + ' 小节')
const statusLabel = computed(() => props.session.selectedId ? baseStatusLabel.value + ' · 练习 ' + rangeDescription.value : baseStatusLabel.value)
const gridTitle = computed(() => approximateGrid.value
    ? '近似小节网格：按 4/4 拍、120 BPM 划分；拖动 A/B 可选择练习范围'
    : '小节网格根据 MIDI 的拍号与速度变化生成；每五小节使用明显分隔线。拖动 A/B 可选择练习范围')

function clamp(value, min, max) { return Math.min(Math.max(Number(value) || 0, min), Math.max(min, max)) }
function nearestBoundaryIndex(ms) {
    const values = barBoundaries.value
    if (!values.length) return 0
    let best = 0, distance = Infinity
    values.forEach((value, index) => {
        const nextDistance = Math.abs(value - ms)
        if (nextDistance < distance) { best = index; distance = nextDistance }
    })
    return best
}
function nearestBoundary(ms, edge, otherMs) {
    const values = barBoundaries.value
    const candidates = values.filter(value => edge === 'left' ? value < otherMs - .5 : value > otherMs + .5)
    if (!candidates.length) return edge === 'left' ? values[0] || 0 : values[values.length - 1] || duration.value
    return candidates.reduce((best, value) => Math.abs(value - ms) < Math.abs(best - ms) ? value : best, candidates[0])
}
function pointerTime(event) {
    const rect = timelineRef.value?.getBoundingClientRect()
    if (!rect || rect.width <= 0) return 0
    return clamp((event.clientX - rect.left) / rect.width, 0, 1) * duration.value
}
function onTrackPointerDown(event) {
    if (event.button !== 0 || !props.session.selectedId || props.session.loading || props.session.busy) return
    event.preventDefault()
    props.session.seek(pointerTime(event))
}
function beginRangeDrag(edge, event) {
    if (!props.session.selectedId || props.session.loading || props.session.busy || duration.value <= 0) return
    dragEdge.value = edge
    rangeDraft.value = [...rangeValue.value]
    event.currentTarget.setPointerCapture?.(event.pointerId)
}
function onPointerMove(event) {
    if (!dragEdge.value || !rangeDraft.value) return
    const next = [...rangeDraft.value]
    const index = dragEdge.value === 'left' ? 0 : 1
    next[index] = nearestBoundary(pointerTime(event), dragEdge.value, next[1 - index])
    rangeDraft.value = next
}
async function onPointerUp() {
    if (!dragEdge.value) return
    dragEdge.value = ''
    const target = rangeDraft.value
    if (!target) return
    const succeeded = await props.session.setRange(target[0], target[1])
    if (succeeded || !props.session.busy) rangeDraft.value = null
}
async function resetRange() {
    rangeDraft.value = null
    await props.session.setRange(0, duration.value)
}
async function onHandleKeydown(edge, event) {
    const key = event.key
    if (!['ArrowLeft', 'ArrowDown', 'ArrowRight', 'ArrowUp', 'Home', 'End'].includes(key)) return
    event.preventDefault()
    const values = barBoundaries.value
    if (values.length < 2) return
    const next = [...rangeValue.value]
    const index = edge === 'left' ? 0 : 1
    let position = nearestBoundaryIndex(next[index])
    if (key === 'Home') position = edge === 'left' ? 0 : nearestBoundaryIndex(next[0]) + 1
    else if (key === 'End') position = edge === 'right' ? values.length - 1 : nearestBoundaryIndex(next[1]) - 1
    else position += key === 'ArrowLeft' || key === 'ArrowDown' ? -1 : 1
    const minIndex = edge === 'left' ? 0 : nearestBoundaryIndex(next[0]) + 1
    const maxIndex = edge === 'left' ? nearestBoundaryIndex(next[1]) - 1 : values.length - 1
    position = clamp(position, minIndex, maxIndex)
    next[index] = values[position]
    rangeDraft.value = next
    const succeeded = await props.session.setRange(next[0], next[1])
    if (succeeded || !props.session.busy) rangeDraft.value = null
}
function clearDrag() { dragEdge.value = ''; rangeDraft.value = null }
watch(() => props.session.selectedId, clearDrag, {flush: 'sync'})
watch(() => props.session.loading, loading => { if (loading) clearDrag() }, {flush: 'sync'})
onBeforeUnmount(clearDrag)
</script>

<style scoped>
.midi-transport { width: min(680px, calc(100vw - 40px)); padding: 14px 19px 9px; border: 1px solid #ffffff1a; border-radius: 11px; background: #242424ed; backdrop-filter: blur(18px); box-shadow: 0 12px 45px #0003; color: #e8e8e8; }
.transport-main { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; }
.track-details { display: flex; flex: 1; flex-direction: column; min-width: 0; gap: 7px; }
.track-details strong { font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.track-details span { font-size: 10px; color: #a2a2a2; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.time-readout { font-size: 11px; font-variant-numeric: tabular-nums; white-space: nowrap; color: #c0c0c0; }
.time-readout i { color: #656565; font-style: normal; margin: 0 4px; }
.timeline-control { position: relative; height: 31px; width: 100%; touch-action: none; user-select: none; cursor: pointer; }
.timeline-control.disabled { opacity: .55; cursor: default; }
.timeline-rail { position: absolute; left: 0; right: 0; top: 12px; height: 6px; border-radius: 999px; background: #ffffff20; overflow: hidden; }
.played-progress { position: absolute; inset: 0 auto 0 0; background: #ffffff23; border-radius: inherit; }
.range-highlight { position: absolute; top: 0; bottom: 0; background: var(--ui-accent, #b8a3e5); opacity: .48; border-radius: 2px; }
.measure-marker { position: absolute; z-index: 2; top: 7px; bottom: 5px; width: 1px; background: #ffffff4a; pointer-events: none; }
.measure-marker.major { top: 3px; bottom: 2px; width: 2px; background: #f0eafa; box-shadow: 0 0 5px #bca8e955; }
.measure-label { position: absolute; left: 2px; top: -4px; padding: 0 2px; border-radius: 2px; background: #242424; color: #e6dafa; font-size: 8px; font-weight: 700; line-height: 10px; white-space: nowrap; }
.playhead { position: absolute; z-index: 4; top: 7px; bottom: 4px; width: 2px; background: #f8f5ff; transform: translateX(-50%); pointer-events: none; box-shadow: 0 0 5px #fff8; }
.playhead i { position: absolute; top: 3px; left: 50%; width: 7px; height: 7px; transform: translate(-50%, -50%); border: 1px solid #26212e; border-radius: 50%; background: #fff; }
.range-handle { position: absolute; z-index: 6; top: 3px; display: grid; place-items: center; width: 22px; height: 22px; padding: 0; transform: translateX(-50%); border: 1px solid #2a2435; border-radius: 50%; background: var(--ui-accent, #c3afea); color: #211d29; font-size: 9px; font-weight: 800; line-height: 1; cursor: ew-resize; touch-action: none; box-shadow: 0 1px 5px #0007; }
.range-handle:hover, .range-handle:focus-visible { outline: 2px solid #f3edff; outline-offset: 2px; }
.range-handle:disabled { cursor: default; opacity: .6; }
.grid-hint { margin: 1px 0 0; color: #999; font-size: 9px; line-height: 1.3; }
@media (max-width: 520px) {
    .midi-transport { width: calc(100vw - 24px); padding: 10px 10px 6px; }
    .transport-main { gap: 7px; }
    .track-details { gap: 4px; }
    .time-readout { font-size: 10px; }
}
</style>
