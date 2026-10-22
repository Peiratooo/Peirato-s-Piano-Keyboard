<template>
    <section class="midi-practice" data-piano-control aria-label="播放与跟弹设置">
        <div class="option"><span>{{ session.mode === 'follow' ? '练习手' : '演奏手' }}</span><NSelect :value="session.options.handMode" :options="hands" size="small" :disabled="session.busy" aria-label="演奏手" @update:value="session.setHand" /></div>
        <div class="option speed"><span>速度</span><NInputNumber :value="session.options.speed" :min=".25" :max="3" :step=".05" :update-value-on-input="false" size="small" :disabled="session.busy" aria-label="播放速度" @update:value="updateSpeed"><template #suffix>×</template></NInputNumber></div>
        <div class="loop-option"><PhRepeat :size="17" /><span>区间循环</span><NSwitch :value="session.options.loop" size="small" :disabled="session.busy" aria-label="区间循环" @update:value="session.setLoop" /></div>
        <div v-if="session.mode === 'follow'" class="follow-hint"><PhPianoKeys :size="17" /><span>等待按对后继续</span></div>
    </section>
</template>

<script setup>
import {NInputNumber, NSelect, NSwitch} from 'naive-ui'
import {PhPianoKeys, PhRepeat} from '@phosphor-icons/vue'
const props = defineProps({session: {type: Object, required: true}})
const hands = [{label: '双手', value: 'both'}, {label: '左手', value: 'left'}, {label: '右手', value: 'right'}]
function updateSpeed(value) { if (value !== null) props.session.setSpeed(value) }
</script>

<style scoped>
.midi-practice { display: flex; align-items: center; flex-wrap: wrap; gap: 18px; padding: 11px 16px; background: #242424ed; border: 1px solid #ffffff15; border-radius: 9px; color: #c9c9c9; backdrop-filter: blur(16px); }
.option, .loop-option, .follow-hint { display: flex; align-items: center; gap: 9px; font-size: 12px; white-space: nowrap; }
.option :deep(.n-select) { width: 92px; }
.speed :deep(.n-input-number) { width: 110px; }
.follow-hint { margin-left: auto; font-size: 11px; color: #909090; }
@media (max-width: 1000px) { .midi-practice { gap: 12px; } .follow-hint { display: none; } }
</style>
