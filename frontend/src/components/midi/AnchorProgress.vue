<template>
    <div class="anchor-progress">
        <div class="time-row">
            <span>{{ formatTime(current) }}</span>
            <span>{{ formatTime(leftAnchor) }} - {{ formatTime(rightAnchor) }}</span>
            <span>{{ formatTime(duration) }}</span>
        </div>
        <div ref="barRef" class="bar" @pointerdown="seekByPointer">
            <div class="range" :style="rangeStyle"></div>
            <div class="fill" :style="fillStyle"></div>
            <button class="anchor left" :style="{left: percent(leftAnchor) + '%'}" @pointerdown.stop="startDrag('left', $event)"></button>
            <button class="anchor right" :style="{left: percent(rightAnchor) + '%'}" @pointerdown.stop="startDrag('right', $event)"></button>
        </div>
    </div>
</template>

<script setup>
import {computed, ref} from 'vue'

const props = defineProps({
    duration: {type: Number, default: 0},
    current: {type: Number, default: 0},
    leftAnchor: {type: Number, default: 0},
    rightAnchor: {type: Number, default: 0},
})
const emit = defineEmits(['seek', 'update:leftAnchor', 'update:rightAnchor'])
const barRef = ref(null)
let dragging = ''

const rangeStyle = computed(() => ({
    left: percent(props.leftAnchor) + '%',
    width: Math.max(0, percent(props.rightAnchor) - percent(props.leftAnchor)) + '%',
}))
const fillStyle = computed(() => ({
    width: percent(props.current) + '%',
}))

function percent(value) {
    if (!props.duration) return 0
    return Math.max(0, Math.min(100, Number(value || 0) / props.duration * 100))
}

function valueFromEvent(event) {
    const rect = barRef.value?.getBoundingClientRect()
    if (!rect || !props.duration) return 0
    const ratio = Math.max(0, Math.min(1, (event.clientX - rect.left) / rect.width))
    return Number((ratio * props.duration).toFixed(2))
}

function seekByPointer(event) {
    emit('seek', clampToAnchors(valueFromEvent(event)))
}

function startDrag(type, event) {
    dragging = type
    event.currentTarget.setPointerCapture?.(event.pointerId)
    updateAnchor(event)
    window.addEventListener('pointermove', updateAnchor)
    window.addEventListener('pointerup', stopDrag, {once: true})
}

function updateAnchor(event) {
    if (!dragging) return
    const value = valueFromEvent(event)
    if (dragging === 'left') {
        // 左锚点永远不能超过右锚点，两个锚点不会互换。
        emit('update:leftAnchor', Math.min(value, props.rightAnchor))
    } else {
        // 右锚点永远不能小于左锚点，两个锚点不会互换。
        emit('update:rightAnchor', Math.max(value, props.leftAnchor))
    }
}

function stopDrag() {
    dragging = ''
    window.removeEventListener('pointermove', updateAnchor)
}

function clampToAnchors(value) {
    return Math.max(props.leftAnchor, Math.min(props.rightAnchor, value))
}

function formatTime(seconds) {
    const safe = Math.max(0, Number(seconds || 0))
    const m = Math.floor(safe / 60)
    const s = Math.floor(safe % 60)
    return `${m}:${String(s).padStart(2, '0')}`
}
</script>

<style lang="scss" scoped>
.anchor-progress { width: 100%; }
.time-row {
    display: flex;
    justify-content: space-between;
    margin-bottom: 8px;
    font-size: 11px;
    color: #64748b;
}
.bar {
    position: relative;
    height: 10px;
    border-radius: 999px;
    background: rgba(15, 23, 42, 0.08);
    cursor: pointer;
}
.range,
.fill {
    position: absolute;
    top: 0;
    bottom: 0;
    border-radius: 999px;
    pointer-events: none;
}
.range { background: rgba(37, 99, 235, 0.18); }
.fill { left: 0; background: rgba(15, 23, 42, 0.8); }
.anchor {
    position: absolute;
    top: 50%;
    width: 18px;
    height: 18px;
    border: 2px solid #fff;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    background: #2563eb;
    box-shadow: 0 8px 18px rgba(37, 99, 235, 0.28);
    cursor: ew-resize;
}
.anchor.right { background: #0f172a; }
</style>
