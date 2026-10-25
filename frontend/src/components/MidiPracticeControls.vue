<template>
    <section class="midi-practice" data-piano-control aria-label="播放与跟弹设置">
        <div class="hand-switch" role="group" aria-label="演奏手，两个点亮为双手">
            <NButton v-for="hand in hands" :key="hand.value" quaternary size="small"
                class="hand-button" :class="{'is-selected': selected(hand.value), 'is-left': hand.value === 'left'}"
                :disabled="session.busy" :aria-pressed="selected(hand.value)" :aria-label="hand.label"
                :title="hand.label + (selected(hand.value) ? ' · 已启用' : ' · 点击启用')"
                @click="toggleHand(hand.value)">
                <PhHandPalm :size="22" :weight="selected(hand.value) ? 'fill' : 'regular'" />
                <span>{{ hand.short }}</span>
            </NButton>
        </div>
        <NPopover trigger="click" placement="bottom" :show="showSpeed" :to="false" :width="280" @update:show="toggleSpeed">
            <template #trigger>
                <NButton quaternary size="small" class="speed-button" :disabled="session.loading"
                    :aria-expanded="showSpeed" aria-label="调整播放速度" title="调整播放速度">
                    <PhGauge :size="19" /><span>{{ speedPercent }}%</span>
                </NButton>
            </template>
            <section class="speed-panel" data-piano-control aria-label="播放速度设置">
                <header>
                    <div class="speed-heading"><span>播放速度</span>
                        <NButton size="tiny" quaternary circle title="恢复 100%" aria-label="恢复播放速度为 100%" @click="resetSpeed">
                            <template #icon><PhArrowCounterClockwise :size="15" /></template>
                        </NButton>
                    </div>
                    <strong>{{ speedPercent }}%</strong>
                </header>
                <NSlider :value="speedPercent" :min="25" :max="500" :step="5" :marks="speedMarks"
                    :format-tooltip="value => value + '%'" aria-label="播放速度百分比"
                    @update:value="updateSpeed" @dragend="commitSpeed" @keyup="commitSpeedKey" />
            </section>
        </NPopover>
        <NButton quaternary size="small" class="loop-toggle" :class="{'is-selected': session.options.loop}"
            :disabled="session.busy" :aria-pressed="session.options.loop" aria-label="区间循环"
            :title="session.options.loop ? '关闭区间循环' : '开启区间循环'" @click="session.setLoop(!session.options.loop)">
            <PhRepeat :size="18" /><span>循环</span>
        </NButton>
    </section>
</template>

<script setup>
import {computed, onBeforeUnmount, ref, watch} from 'vue'
import {NButton, NPopover, NSlider} from 'naive-ui'
import {PhArrowCounterClockwise, PhGauge, PhHandPalm, PhRepeat} from '@phosphor-icons/vue'
const props = defineProps({session: {type: Object, required: true}, panel: {type: String, default: null}})
const emit = defineEmits(['open-speed', 'speed-visibility'])
const showSpeed = ref(false)
const speedDraft = ref(null)
const speedPercent = computed(() => speedDraft.value ?? Math.round(props.session.options.speed * 100))
const speedMarks = {25: '25%', 100: '100%', 200: '200%', 300: '300%', 500: '500%'}
let speedTimer
const hands = [{label: '左手', short: '左', value: 'left'}, {label: '右手', short: '右', value: 'right'}]
function selected(hand) { return props.session.options.handMode === 'both' || props.session.options.handMode === hand }
function toggleHand(hand) {
    const current = props.session.options.handMode
    const next = current === 'both' ? (hand === 'left' ? 'right' : 'left')
        : current === hand ? current : 'both'
    if (next !== current) props.session.setHand(next)
}
function toggleSpeed(show) {
    showSpeed.value = show
    emit('speed-visibility', show)
    if (show) emit('open-speed')
    else if (speedDraft.value !== null) commitSpeed()
}
function updateSpeed(value) {
    speedDraft.value = value
    clearTimeout(speedTimer)
    speedTimer = setTimeout(commitSpeed, 120)
}
async function commitSpeed() {
    clearTimeout(speedTimer)
    if (speedDraft.value === null) return
    const target = speedDraft.value
    await props.session.setSpeed(target / 100)
    if (speedDraft.value === target) speedDraft.value = null
}
function commitSpeedKey(event) {
    if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) commitSpeed()
}
function resetSpeed() { speedDraft.value = 100; commitSpeed() }
watch(() => props.panel, panel => { if (panel) toggleSpeed(false) })
watch(() => props.session.mode, () => toggleSpeed(false))
onBeforeUnmount(() => clearTimeout(speedTimer))
</script>

<style scoped>
.midi-practice { display: flex; align-items: center; gap: 16px; min-width: 0; color: #c9c9c9; }
.hand-switch { display: flex; align-items: center; gap: 3px; padding-right: 13px; border-right: 1px solid #ffffff12; }
.hand-button { --n-padding: 0 6px !important; color: #777; border-radius: 5px; }
.hand-button :deep(.n-button__content) { display: flex; gap: 3px; align-items: center; }
.hand-button span { font-size: 10px; }
.hand-button:not(.is-left) svg { transform: scaleX(-1); }
.hand-button.is-selected { color: var(--ui-accent); background: var(--ui-accent-soft); }
.hand-button.is-left.is-selected { color: var(--whiteKeyLeft); background: color-mix(in srgb, var(--whiteKeyLeft) 12%, transparent); }
.loop-toggle { color: #777; border-radius: 5px; }
.loop-toggle :deep(.n-button__content) { display: flex; align-items: center; gap: 7px; font-size: 11px; white-space: nowrap; }
.loop-toggle.is-selected { color: var(--ui-accent); background: var(--ui-accent-soft); }
.speed-button :deep(.n-button__content) { display: flex; align-items: center; gap: 7px; }
.speed-button span { min-width: 36px; font-size: 12px; font-variant-numeric: tabular-nums; }
.speed-panel { padding: 4px 7px 22px; color: #ddd; }
.speed-panel header { display: flex; align-items: center; justify-content: space-between; font-size: 12px; }
.speed-heading { display: flex; align-items: center; gap: 6px; }
.speed-panel header { margin-bottom: 20px; }
.speed-panel header strong { color: var(--ui-accent); font-variant-numeric: tabular-nums; }
@media (max-width: 1150px) { .midi-practice { gap: 10px; } .hand-switch { padding-right: 8px; } .hand-button span, .loop-toggle span { display: none; } }
</style>
