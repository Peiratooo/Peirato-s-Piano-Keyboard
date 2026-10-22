<template>
    <section class="recording-dock" :class="{'is-simple': simple, 'has-draft': recording.state.status === 'draft'}"
        data-piano-control aria-label="自由演奏录制">
        <div v-if="recording.state.status === 'recording'" class="recording-active">
            <span class="recording-indicator" aria-label="正在录制"></span>
            <span class="recording-label">录制中</span>
            <span class="recording-clock" role="timer">{{ formatTime(recording.durationMs) }}</span>
            <NButton size="small" type="error" :disabled="recording.busy" @click="recording.stop">停止</NButton>
        </div>
        <template v-else-if="recording.state.status === 'draft'">
            <template v-if="simple">
                <span class="draft-label">录制草稿待保存</span>
                <NButton size="small" @click="$emit('show-draft')">返回瀑布视图</NButton>
            </template>
            <template v-else>
                <div class="draft-heading">
                    <span>录制完成 · {{ formatTime(recording.state.durationMs) }} · {{ recording.state.noteCount }} 个音符</span>
                </div>
                <div class="draft-actions">
                    <NInput v-model:value="recording.name" size="small" maxlength="120" placeholder="录制名称（可留空）"
                        aria-label="录制名称" :disabled="recording.busy"/>
                    <NButton v-if="!recording.previewing" size="small" :disabled="recording.busy" @click="recording.preview">试听</NButton>
                    <NButton v-else size="small" :disabled="recording.busy" @click="recording.stopPreview">停止试听</NButton>
                    <NButton size="small" type="primary" :loading="recording.busy" @click="recording.save">保存 MIDI</NButton>
                    <NPopconfirm positive-text="放弃录制" negative-text="保留草稿"
                        :positive-button-props="{type: 'error'}" @positive-click="recording.discard">
                        <template #trigger>
                            <NButton size="small" quaternary :disabled="recording.busy">放弃</NButton>
                        </template>
                        确认放弃这段未保存的录制吗？此操作无法撤销。
                    </NPopconfirm>
                </div>
            </template>
        </template>
        <NButton v-else size="small" secondary :disabled="recording.busy" @click="recording.start">
            <span class="recording-empty-dot"></span>开始录制
        </NButton>
        <div v-if="recording.state.error && !simple" class="recording-warning" role="status">
            {{ recording.state.error }}
        </div>
    </section>
</template>

<script setup>
import {NButton, NInput, NPopconfirm} from 'naive-ui'
defineProps({
    recording: {type: Object, required: true},
    simple: {type: Boolean, default: false},
})
defineEmits(['show-draft'])
function formatTime(milliseconds) {
    const total = Math.floor(Math.max(0, Number(milliseconds) || 0) / 1000)
    const hours = Math.floor(total / 3600)
    const minutes = String(Math.floor(total / 60) % 60).padStart(2, '0')
    const seconds = String(total % 60).padStart(2, '0')
    return hours ? [hours, minutes, seconds].join(':') : minutes + ':' + seconds
}
</script>

<style scoped>
.recording-dock {
    position: absolute; z-index: 4; top: 66px; right: 17px;
    display: flex; flex-direction: column; align-items: stretch; gap: 8px;
    min-width: 124px; max-width: min(600px, calc(100vw - 34px));
    padding: 9px; border: 1px solid #ffffff1c;
    background: #242424f5; box-shadow: 0 7px 25px #0004;
    border-radius: 10px; color: #ededed; font-size: 12px;
}
.recording-active { display: flex; align-items: center; gap: 9px; }
.recording-indicator, .recording-empty-dot { display: inline-block; flex-shrink: 0; background: #d55d66; width: 9px; height: 9px; border-radius: 50%; }
.recording-indicator { animation: recording-pulse 1.1s ease-in-out infinite; }
.recording-clock { font-variant-numeric: tabular-nums; font-size: 13px; font-weight: 600; }
.recording-label { white-space: nowrap; }
.draft-heading { color: #c5c5c5; padding: 1px 3px; }
.draft-actions { display: flex; align-items: center; gap: 6px; }
.draft-actions :deep(.n-input) { width: 200px; min-width: 120px; }
.recording-warning { color: #efb1b5; line-height: 1.5; max-width: 410px; }
.is-simple { top: 50px; right: 8px; flex-direction: row; align-items: center; padding: 5px 7px; }
.draft-label { white-space: nowrap; }
@keyframes recording-pulse { 50% { opacity: .42; } }
@media (max-width: 590px) {
    .recording-dock.has-draft:not(.is-simple) { left: 8px; right: 8px; max-width: none; }
    .draft-actions { flex-wrap: wrap; }
    .draft-actions :deep(.n-input) { flex: 1 1 100%; width: 100%; }
}
@media (prefers-reduced-motion: reduce) { .recording-indicator { animation: none; } }
</style>
