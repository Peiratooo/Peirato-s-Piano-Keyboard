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
                <span v-if="item.key === 'about' && updater.available" class="update-dot" role="img" aria-label="有新版本可更新" title="有新版本可更新"></span>
            </button>
        </aside>

        <main class="setting-content">
            <BasicSettings v-if="store.controlMenu === 'basic'" />
            <MidiDevicesSettings v-else-if="store.controlMenu === 'devices'" />
            <KeyboardDisplaySettings v-else-if="store.controlMenu === 'keyboard'" />
            <KeyboardMappingSettings v-else-if="store.controlMenu === 'keymap'" />
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
import KeyboardMappingSettings from './settings/KeyboardMappingSettings.vue'
import AppearanceSettings from './settings/AppearanceSettings.vue'
import SoundFont from './settings/SoundFont.vue'
import AboutSettings from './settings/AboutSettings.vue'
import {useUpdater} from '../store/updater'

const store = inject('store')
const updater = useUpdater()
const changeConfig = inject('changeConfig')

const menus = [
    {key: 'basic', label: '基础设置', icon: '⌘'},
    {key: 'devices', label: 'MIDI 设备', icon: '🎛'},
    {key: 'keyboard', label: '键盘显示', icon: '🎹'},
    {key: 'keymap', label: '按键方案', icon: '⌨'},
    {key: 'appearance', label: '外观主题', icon: '🎨'},
    {key: 'soundfont', label: '音源管理', icon: '🎧'},
    {key: 'about', label: '关于软件', icon: 'i'},
]

onBeforeUnmount(() => {
    changeConfig?.()
})
</script>

<style lang="scss" src="../../../shared/piano/settings.scss"></style>
