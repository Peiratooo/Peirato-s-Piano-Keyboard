<template>
    <div class="control-window">
        <div class="titlebar">
            <div>
                <div class="title">Peirato's Piano Keyboard 设置</div>
                <div class="subtitle">自定义你的键盘显示、声音与 MIDI 设备。</div>
            </div>
            <n-tag size="small" :bordered="false" type="info">v{{ store.config.version || '-' }}</n-tag>
        </div>
        <Setting />
    </div>
</template>

<script setup>
import {inject} from 'vue'
import {NTag} from 'naive-ui'
import Setting from '../components/Setting.vue'

const store = inject('store')
</script>

<style lang="scss" scoped>
.control-window {
    min-height: 100vh;
    background: #f3f3f3;
    color: #303030;
    user-select: none;
}

.titlebar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 22px 28px 16px;
}

.title {
    font-size: 22px;
    font-weight: 700;
    letter-spacing: -0.02em;
}

.subtitle {
    margin-top: 6px;
    font-size: 13px;
    color: #666666;
}
</style>
