<template>
    <section ref="container" class="palette-performance" aria-label="当前配色的左右手瀑布与琴键预览">
        <header class="preview-header">
            <div class="hand-legend">
                <span><i :style="{background: colors.whiteKeyLeft.color}"></i>左手</span>
                <span><i :style="{background: colors.whiteKey.color}"></i>右手</span>
            </div>
            <div class="pedal-legend">
                <span v-for="pedal in pedals" :key="pedal.key"><i :style="{background: colors[pedal.key].color}"></i>{{ pedal.label }}</span>
            </div>
        </header>
        <MidiWaterfallCanvas :layout="layout" :width="width" :height="fallHeight" :colors="colors"
            :timeline="timeline" :player-state="previewState" />
        <div class="preview-keybed"><ExampleKeyboard :layout="layout" /></div>
    </section>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, onMounted, ref} from 'vue'
import MidiWaterfallCanvas from './MidiWaterfallCanvas.vue'
import ExampleKeyboard from './ExampleKeyboard.vue'
import {createKeyboardLayout} from '../services/keyboardLayout'

defineProps({colors: {type: Object, required: true}})
const store = inject('store')
const container = ref(null)
const width = ref(0)
const fallHeight = ref(88)
const layout = computed(() => createKeyboardLayout(store.keyboardConfig.filter(key => key.index >= 48 && key.index <= 71), width.value))
const pedals = [{key: 'damperPedal', label: '延音'}, {key: 'softPedal', label: '柔音'}, {key: 'sostenutoPedal', label: '持音'}]
const previewState = {status: 'paused', mode: 'play', handMode: 'both', currentMs: 0, leftMs: 0, rightMs: 5000, speed: 1}
const timeline = {durationMs: 5000, measures: [{index: 1, ms: 0}, {index: 2, ms: 2500}], notes: [
    {id: 'left-c', note: 48, startMs: 0, endMs: 1800, hand: 'left', velocity: 90, trackIndex: 0},
    {id: 'left-eb', note: 51, startMs: 0, endMs: 900, hand: 'left', velocity: 90, trackIndex: 0},
    {id: 'left-g', note: 55, startMs: 0, endMs: 1400, hand: 'left', velocity: 90, trackIndex: 0},
    {id: 'right-eb', note: 63, startMs: 0, endMs: 1000, hand: 'right', velocity: 90, trackIndex: 1},
    {id: 'right-g', note: 67, startMs: 0, endMs: 2000, hand: 'right', velocity: 90, trackIndex: 1},
    {id: 'right-bb', note: 70, startMs: 0, endMs: 1500, hand: 'right', velocity: 90, trackIndex: 1},
    {id: 'left-next', note: 48, startMs: 3000, endMs: 4000, hand: 'left', velocity: 90, trackIndex: 0},
    {id: 'right-next', note: 67, startMs: 3300, endMs: 4500, hand: 'right', velocity: 90, trackIndex: 1},
]}
let observer
onMounted(() => {
    observer = new ResizeObserver(entries => {
        width.value = entries[0].contentRect.width
        fallHeight.value = window.innerHeight < 800 ? 76 : 88
    })
    observer.observe(container.value)
})
onBeforeUnmount(() => observer.disconnect())
</script>

<style scoped>
.palette-performance { overflow: hidden; border-radius: 12px; background: #242424; }
.preview-header { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px 24px; padding: 12px 16px; }
.hand-legend, .pedal-legend { display: flex; align-items: center; gap: 18px; color: #ddd; font-size: 12px; }
.hand-legend span, .pedal-legend span { display: flex; align-items: center; gap: 6px; }
i { width: 8px; height: 8px; border-radius: 50%; }
.preview-keybed { height: 112px; }
.pedal-legend { justify-content: flex-end; font-size: 11px; color: #b5b5b5; }
@media (max-height: 800px) { .preview-keybed { height: 100px; } }
</style>
