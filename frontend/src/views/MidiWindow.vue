<template>
    <main class="midi-placeholder">
        <section class="panel">
            <div class="eyebrow">MIDI 中心</div>
            <h1>MIDI 文件播放 / 练习功能已移除</h1>
            <p>当前版本保留 MIDI 设备输入、输出和通道设置；这个窗口只作为后续功能入口占位。</p>
        </section>
    </main>
</template>

<script setup>
</script>

<style lang="scss" scoped>
.midi-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100vw;
    height: 100vh;
    box-sizing: border-box;
    padding: 24px;
    background: #f5f7fb;
    color: #172033;
    --wails-draggable: drag;
}

.panel {
    max-width: 560px;
    text-align: center;
}

.eyebrow {
    margin-bottom: 10px;
    font-size: 13px;
    color: #64748b;
}

h1 {
    margin: 0;
    font-size: 24px;
    font-weight: 750;
}

p {
    margin: 14px 0 0;
    font-size: 14px;
    line-height: 1.7;
    color: #475569;
}
</style>
