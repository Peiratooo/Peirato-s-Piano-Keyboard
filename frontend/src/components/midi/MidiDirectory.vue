<template>
    <aside class="midi-directory">
        <div class="directory-header">
            <div>
                <div class="title">MIDI 目录</div>
                <div class="subtitle">按系统绝对路径保存</div>
            </div>
            <button class="import-button" @click="$emit('import')">导入</button>
        </div>

        <div v-if="items.length" class="midi-list">
            <button
                v-for="item in items"
                :key="item.id"
                class="midi-item"
                :class="{active: item.id === selectedId}"
                @click="$emit('select', item.id)"
            >
                <span class="music-dot">♪</span>
                <span class="item-body">
                    <strong>{{ item.name }}</strong>
                    <small>{{ item.path }}</small>
                </span>
                <span class="remove" title="只移除记录，不删除真实文件" @click.stop="$emit('remove', item.id)">×</span>
            </button>
        </div>

        <div v-else class="empty-box">
            <div class="empty-icon">♫</div>
            <div>还没有导入 MIDI</div>
            <small>导入后会显示在这里，不会复制或删除真实文件。</small>
        </div>
    </aside>
</template>

<script setup>
defineProps({
    items: {type: Array, default: () => []},
    selectedId: {type: String, default: ''},
})

defineEmits(['import', 'select', 'remove'])
</script>

<style lang="scss" scoped>
.midi-directory {
    width: 285px;
    height: 100%;
    padding: 14px;
    box-sizing: border-box;
    border-right: 1px solid rgba(148, 163, 184, 0.2);
    background: rgba(255, 255, 255, 0.58);
}
.directory-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
}
.title { font-size: 15px; font-weight: 760; color: #0f172a; }
.subtitle { margin-top: 2px; font-size: 11px; color: #64748b; }
.import-button {
    border: 0;
    border-radius: 999px;
    padding: 7px 12px;
    background: #0f172a;
    color: #fff;
    cursor: pointer;
}
.midi-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: calc(100vh - 92px);
    overflow: auto;
}
.midi-item {
    position: relative;
    display: grid;
    grid-template-columns: 28px 1fr 20px;
    align-items: center;
    gap: 8px;
    width: 100%;
    border: 1px solid rgba(148, 163, 184, 0.18);
    border-radius: 16px;
    padding: 10px;
    text-align: left;
    background: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    transition: 180ms;
}
.midi-item:hover,
.midi-item.active {
    background: rgba(219, 234, 254, 0.88);
    border-color: rgba(37, 99, 235, 0.28);
}
.music-dot {
    width: 28px;
    height: 28px;
    border-radius: 10px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: rgba(15, 23, 42, 0.08);
    color: #2563eb;
}
.item-body { min-width: 0; }
.item-body strong,
.item-body small {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
.item-body strong { font-size: 12px; color: #0f172a; }
.item-body small { margin-top: 2px; font-size: 10px; color: #64748b; }
.remove {
    width: 20px;
    height: 20px;
    border-radius: 999px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: #94a3b8;
}
.remove:hover { color: #ef4444; background: rgba(239, 68, 68, 0.1); }
.empty-box {
    display: flex;
    height: calc(100% - 50px);
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    text-align: center;
    color: #64748b;
}
.empty-icon { font-size: 28px; color: #2563eb; }
.empty-box small { max-width: 190px; line-height: 1.6; }
</style>
