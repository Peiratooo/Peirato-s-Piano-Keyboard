<template>
    <section class="midi-transport" data-piano-control aria-label="MIDI 播放控制">
        <div class="transport-main">
            <NButton circle type="primary" size="large" :loading="session.loading" :disabled="!session.selectedId || session.busy || session.mode === 'free'" :aria-label="playLabel" @click="session.togglePlayback">
                <template #icon><PhPause v-if="session.state.status === 'playing'" :size="20" weight="fill" /><PhPlay v-else :size="20" weight="fill" /></template>
            </NButton>
            <NButton quaternary circle :disabled="session.state.status === 'idle' || session.busy" aria-label="停止并回到区间起点" @click="session.stop"><template #icon><PhStop :size="18" weight="fill" /></template></NButton>
            <div class="track-details"><strong>{{ session.selectedSong?.name || '请先选择曲目' }}</strong><span>{{ statusLabel }}</span></div>
            <span class="time-readout">{{ formatMidiTime(currentValue) }} <i>/</i> {{ formatMidiTime(session.durationMs) }}</span>
        </div>
        <NSlider :value="currentValue" :min="0" :max="Math.max(1, session.durationMs)" :step="10" :disabled="!session.selectedId || session.loading || session.busy" :format-tooltip="formatMidiTime" aria-label="播放进度" @update:value="updateSeek" @dragstart="seeking = true" @dragend="finishSeek" />
        <div class="range-line">
            <NButton quaternary size="tiny" :disabled="!session.selectedId || session.busy" @click="session.setRange(session.state.currentMs, session.options.rightMs)">A {{ formatMidiTime(rangeValue[0]) }}</NButton>
            <NSlider :value="rangeValue" range :min="0" :max="Math.max(1, session.durationMs)" :step="10" :disabled="!session.selectedId || session.loading || session.busy" :format-tooltip="formatMidiTime" aria-label="A B 练习区间" @update:value="updateRange" @dragstart="ranging = true" @dragend="finishRange" />
            <NButton quaternary size="tiny" :disabled="!session.selectedId || session.busy" @click="session.setRange(session.options.leftMs, session.state.currentMs)">B {{ formatMidiTime(rangeValue[1]) }}</NButton>
            <NButton quaternary circle size="tiny" :disabled="!session.selectedId || session.busy" aria-label="重置为整首曲目" title="重置区间" @click="session.setRange(0, session.durationMs)"><template #icon><PhArrowCounterClockwise :size="14" /></template></NButton>
        </div>
    </section>
</template>

<script setup>
import {computed, onBeforeUnmount, ref, watch} from 'vue'
import {NButton, NSlider} from 'naive-ui'
import {PhArrowCounterClockwise, PhPause, PhPlay, PhStop} from '@phosphor-icons/vue'
import {formatMidiTime} from '../services/midiSession'
const props = defineProps({session: {type: Object, required: true}})
const seeking = ref(false)
const ranging = ref(false)
const seekDraft = ref(null)
const rangeDraft = ref(null)
let seekTimer, rangeTimer, draftEpoch = 0
const currentValue = computed(() => seekDraft.value ?? props.session.state.currentMs ?? 0)
const rangeValue = computed(() => rangeDraft.value ?? [props.session.options.leftMs, props.session.options.rightMs || props.session.durationMs])
const playLabel = computed(() => props.session.state.status === 'playing' ? '暂停' : props.session.state.status === 'paused' ? '继续' : props.session.mode === 'follow' ? '开始练习' : '播放')
const statusLabel = computed(() => props.session.loading ? '正在加载曲目…' : props.session.state.waiting ? '等待按对琴键' : props.session.state.status === 'playing' ? (props.session.mode === 'follow' ? '跟弹练习中' : '正在播放') : props.session.state.status === 'paused' ? '已暂停' : props.session.selectedId ? '已就绪' : '支持 .mid / .midi')
function updateSeek(value) { seekDraft.value = value; if (!seeking.value) { clearTimeout(seekTimer); seekTimer = setTimeout(finishSeek, 100) } }
async function finishSeek() {
    clearTimeout(seekTimer)
    seeking.value = false
    const target = seekDraft.value
    const epoch = draftEpoch
    if (target === null) return
    await props.session.seek(target)
    if (epoch === draftEpoch && seekDraft.value === target) seekDraft.value = null
}
function updateRange(value) { rangeDraft.value = value; if (!ranging.value) { clearTimeout(rangeTimer); rangeTimer = setTimeout(finishRange, 100) } }
async function finishRange() {
    clearTimeout(rangeTimer)
    ranging.value = false
    const target = rangeDraft.value
    const epoch = draftEpoch
    if (!target) return
    await props.session.setRange(target[0], target[1])
    if (epoch === draftEpoch && rangeDraft.value === target) rangeDraft.value = null
}
function clearDrafts() {
    ++draftEpoch
    clearTimeout(seekTimer)
    clearTimeout(rangeTimer)
    seeking.value = ranging.value = false
    seekDraft.value = rangeDraft.value = null
}
watch(() => props.session.selectedId, clearDrafts, {flush: 'sync'})
watch(() => props.session.loading, loading => { if (loading) clearDrafts() }, {flush: 'sync'})
onBeforeUnmount(clearDrafts)
</script>

<style scoped>
.midi-transport { width: min(680px, calc(100vw - 40px)); padding: 14px 19px 9px; border: 1px solid #ffffff1a; border-radius: 11px; background: #242424ed; backdrop-filter: blur(18px); box-shadow: 0 12px 45px #0003; color: #e8e8e8; }
.transport-main { display: flex; align-items: center; gap: 12px; margin-bottom: 13px; }
.track-details { display: flex; flex: 1; flex-direction: column; min-width: 0; gap: 7px; }
.track-details strong { font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.track-details span { font-size: 10px; color: #a2a2a2; }
.time-readout { font-size: 11px; font-variant-numeric: tabular-nums; white-space: nowrap; color: #c0c0c0; }
.time-readout i { color: #656565; font-style: normal; margin: 0 4px; }
.range-line { display: flex; gap: 12px; align-items: center; margin-top: 9px; }
.range-line > :deep(.n-slider) { flex: 1; min-width: 50px; }
.range-line :deep(.n-button) { color: #aaa; font-size: 10px; font-variant-numeric: tabular-nums; }
</style>
