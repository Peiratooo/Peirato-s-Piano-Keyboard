<template>
    <div class="playback-controls">
        <button @click="$emit('restart')">重来</button>
        <button class="primary" @click="$emit('toggle')">{{ playing ? '暂停' : paused ? '继续' : '播放' }}</button>
        <label>
            <span>速度</span>
            <select :value="rate" @change="$emit('rate', Number($event.target.value))">
                <option v-for="item in rates" :key="item" :value="item">{{ item }}x</option>
            </select>
        </label>
    </div>
</template>

<script setup>
defineProps({
    playing: {type: Boolean, default: false},
    paused: {type: Boolean, default: false},
    rate: {type: Number, default: 1},
})
defineEmits(['restart', 'toggle', 'rate'])
const rates = [0.5, 0.75, 1, 1.25, 1.5, 2]
</script>

<style lang="scss" scoped>
.playback-controls {
    display: flex;
    align-items: center;
    gap: 10px;
}
button,
select {
    border: 1px solid rgba(148, 163, 184, 0.28);
    border-radius: 999px;
    min-height: 34px;
    padding: 0 14px;
    background: rgba(255, 255, 255, 0.78);
    color: #0f172a;
    cursor: pointer;
}
button.primary {
    border-color: #0f172a;
    background: #0f172a;
    color: #fff;
}
label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: #64748b;
}
select { min-height: 34px; padding: 0 12px; }
</style>
