<template>
    <div class="mode-selector">
        <button :class="{active: mode === 'play'}" @click="$emit('update:mode', 'play')">播放模式</button>
        <button :class="{active: mode === 'practice'}" @click="$emit('update:mode', 'practice')">练习模式</button>
    </div>
</template>

<script setup>
defineProps({mode: {type: String, default: 'play'}})
defineEmits(['update:mode'])
</script>

<style lang="scss" scoped>
.mode-selector {
    display: inline-flex;
    padding: 4px;
    border-radius: 999px;
    background: rgba(15, 23, 42, 0.06);
}
button {
    border: 0;
    border-radius: 999px;
    padding: 9px 18px;
    background: transparent;
    color: #64748b;
    font-size: 13px;
    cursor: pointer;
    transition: 180ms;
}
button.active {
    background: #0f172a;
    color: #fff;
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.16);
}
</style>
