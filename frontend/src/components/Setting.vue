<template>
    <div class="settings-shell">
        <aside class="setting-menu">
            <button
                v-for="item in menus"
                :key="item.key"
                class="menu-item"
                :class="{active: store.controlMenu === item.key}"
                @click="store.controlMenu = item.key"
            >
                <span class="menu-icon">{{ item.icon }}</span>
                <span>{{ item.label }}</span>
            </button>
        </aside>

        <main class="setting-content">
            <BasicSettings v-if="store.controlMenu === 'basic'" />
            <MidiDevicesSettings v-else-if="store.controlMenu === 'devices'" />
            <KeyboardDisplaySettings v-else-if="store.controlMenu === 'keyboard'" />
            <AppearanceSettings v-else-if="store.controlMenu === 'appearance'" />
            <SoundFont v-else-if="store.controlMenu === 'soundfont'" />
            <AboutSettings v-else-if="store.controlMenu === 'about'" />
        </main>
    </div>
</template>

<script setup>
import {inject, onBeforeUnmount} from 'vue'
import BasicSettings from './settings/BasicSettings.vue'
import MidiDevicesSettings from './settings/MidiDevicesSettings.vue'
import KeyboardDisplaySettings from './settings/KeyboardDisplaySettings.vue'
import AppearanceSettings from './settings/AppearanceSettings.vue'
import SoundFont from './settings/SoundFont.vue'
import AboutSettings from './settings/AboutSettings.vue'

const store = inject('store')
const changeConfig = inject('changeConfig')

const menus = [
    {key: 'basic', label: '基础设置', icon: '⌘'},
    {key: 'devices', label: 'MIDI 设备', icon: '🎛'},
    {key: 'keyboard', label: '键盘显示', icon: '🎹'},
    {key: 'appearance', label: '外观主题', icon: '🎨'},
    {key: 'soundfont', label: '音源管理', icon: '🎧'},
    {key: 'about', label: '关于软件', icon: 'i'},
]

onBeforeUnmount(() => {
    changeConfig?.()
})
</script>

<style lang="scss">
.settings-shell {
    --setting-panel-border: 1px solid rgba(148, 163, 184, 0.22);
    --setting-panel-bg: rgba(255, 255, 255, 0.76);
    --setting-subtle-bg: rgba(248, 250, 252, 0.9);
    --setting-text-main: #1e293b;
    --setting-text-muted: #64748b;
    --setting-text-soft: #475569;
    --setting-brand: #2563eb;
    --setting-shadow-soft: 0 18px 50px rgba(15, 23, 42, 0.08);
    --setting-shadow-hover: 0 12px 30px rgba(15, 23, 42, 0.08);
    --setting-blur-panel: blur(18px);

    display: grid;
    grid-template-columns: 210px minmax(0, 1fr);
    gap: 18px;
    padding: 0 28px 28px;
    box-sizing: border-box;
}

.settings-shell .setting-menu {
    display: flex;
    flex-direction: column;
    gap: 8px;
    background: rgba(255, 255, 255, 0.66);
    border: var(--setting-panel-border);
    border-radius: 22px;
    padding: 12px;
    box-shadow: var(--setting-shadow-soft);
    backdrop-filter: var(--setting-blur-panel);
}

.settings-shell .menu-item {
    display: flex;
    align-items: center;
    gap: 10px;
    border: 0;
    border-radius: 14px;
    padding: 11px 12px;
    text-align: left;
    cursor: pointer;
    background: transparent;
    color: var(--setting-text-soft);
    transition: 180ms;

    &:hover,
    &.active {
        color: #0f172a;
        background: rgba(37, 99, 235, 0.09);
    }
}

.settings-shell .menu-icon {
    width: 20px;
    text-align: center;
}

.settings-shell .setting-content {
    min-width: 0;
    min-height: 520px;
}

.settings-shell .content-card {
    min-height: 520px;
    background: var(--setting-panel-bg);
    border: var(--setting-panel-border);
    border-radius: 24px;
    padding: 24px;
    box-shadow: var(--setting-shadow-soft);
    backdrop-filter: var(--setting-blur-panel);
    box-sizing: border-box;
}

.settings-shell .section-title {
    margin-bottom: 22px;
    font-size: 18px;
    font-weight: 700;
    color: var(--setting-text-main);
}

.settings-shell .setting-grid {
    display: flex;
    flex-direction: column;
    gap: 18px;
}

.settings-shell .setting-row {
    display: grid;
    grid-template-columns: 230px minmax(0, 1fr);
    align-items: center;
    gap: 22px;

    &.vertical {
        grid-template-columns: minmax(0, 1fr);
        align-items: stretch;
        gap: 10px;
    }
}

.settings-shell .label {
    font-size: 14px;
    font-weight: 650;
    color: var(--setting-text-main);
}

.settings-shell .desc {
    margin-top: 4px;
    font-size: 12px;
    line-height: 1.6;
    color: var(--setting-text-muted);
}

.settings-shell .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.settings-shell .path-text {
    word-break: break-all;
    color: var(--setting-text-soft);
}

.settings-shell .error-text {
    color: #b91c1c;
    font-size: 12px;
    line-height: 1.6;
}

.settings-shell .setting-actions {
    margin-top: 8px;
}

.settings-shell .tip-box,
.settings-shell .placeholder-card {
    border-radius: 16px;
    padding: 16px;
    background: rgba(37, 99, 235, 0.07);
    color: var(--setting-text-soft);
    font-size: 13px;
    line-height: 1.7;
}

.settings-shell .placeholder-title {
    margin-bottom: 6px;
    font-size: 15px;
    font-weight: 700;
    color: var(--setting-text-main);
}

@media (max-width: 860px) {
    .settings-shell {
        grid-template-columns: minmax(0, 1fr);
        padding: 0 18px 18px;
    }

    .settings-shell .setting-menu {
        flex-direction: row;
        overflow-x: auto;
        border-radius: 18px;
    }

    .settings-shell .menu-item {
        flex: 0 0 auto;
        white-space: nowrap;
    }

    .settings-shell .content-card {
        min-height: auto;
        padding: 18px;
        border-radius: 20px;
    }

    .settings-shell .setting-row {
        grid-template-columns: minmax(0, 1fr);
        align-items: stretch;
        gap: 10px;
    }
}
</style>
