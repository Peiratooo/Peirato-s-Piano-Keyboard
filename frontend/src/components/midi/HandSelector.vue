<template>
    <div class="hand-selector">
        <button
            class="hand-card"
            :class="{active: left}"
            :disabled="singleTrackOnly"
            @click="toggle('left')"
        >
            <span>左手</span>
            <small>{{ singleTrackOnly ? '单轨兼容' : '低音声部' }}</small>
        </button>
        <button
            class="hand-card"
            :class="{active: right}"
            :disabled="singleTrackOnly"
            @click="toggle('right')"
        >
            <span>右手</span>
            <small>{{ singleTrackOnly ? '单轨练习' : '高音声部' }}</small>
        </button>
    </div>
</template>

<script setup>
const props = defineProps({
    left: {type: Boolean, default: true},
    right: {type: Boolean, default: true},
    singleTrackOnly: {type: Boolean, default: false},
})
const emit = defineEmits(['update:left', 'update:right'])

function toggle(hand) {
    if (props.singleTrackOnly) return
    // 至少保留一个练习手，避免生成空练习计划。
    if (hand === 'left') {
        if (props.left && !props.right) return
        emit('update:left', !props.left)
    } else {
        if (props.right && !props.left) return
        emit('update:right', !props.right)
    }
}
</script>

<style lang="scss" scoped>
.hand-selector {
    display: grid;
    grid-template-columns: repeat(2, minmax(120px, 1fr));
    gap: 10px;
}
.hand-card {
    border: 1px solid rgba(148, 163, 184, 0.22);
    border-radius: 18px;
    padding: 13px 14px;
    background: rgba(255, 255, 255, 0.68);
    color: #64748b;
    text-align: left;
    cursor: pointer;
    transition: 180ms;
}
.hand-card.active {
    background: rgba(37, 99, 235, 0.1);
    border-color: rgba(37, 99, 235, 0.34);
    color: #1d4ed8;
}
.hand-card:disabled { cursor: not-allowed; opacity: 0.78; }
span { display: block; font-size: 15px; font-weight: 760; }
small { display: block; margin-top: 4px; font-size: 11px; }
</style>
