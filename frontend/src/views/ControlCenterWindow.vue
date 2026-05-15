<template>
    <div class="control-window">
        <div class="titlebar">
            <div>
                <div class="title">Peirato's Piano Keyboard 设置</div>
                <div class="subtitle">自定义你的键盘显示、声音与练习体验。</div>
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
    background: linear-gradient(135deg, #f8f9fd, #eef2f8);
    color: #1f2937;
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
    color: #64748b;
}
</style>
