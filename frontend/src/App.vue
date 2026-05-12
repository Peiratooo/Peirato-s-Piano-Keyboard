<template>
    <n-config-provider
        :date-locale="dateZhCN"
        :locale="zhCN"
        :style="{opacity: isMainWindow ? store.config.opacity / 100 : 1}"
    >
        <n-modal-provider>
            <n-loading-bar-provider>
                <n-message-provider>
                    <router-view />
                </n-message-provider>
            </n-loading-bar-provider>
        </n-modal-provider>
    </n-config-provider>
</template>

<script setup>
import {computed, onBeforeUnmount, onMounted, provide} from 'vue'
import {useRoute} from 'vue-router'
import {dateZhCN, NConfigProvider, NLoadingBarProvider, NMessageProvider, NModalProvider, zhCN} from 'naive-ui'
import {Events, WML} from '@wailsio/runtime'
import axios from 'axios'
import {Keyboard} from '../bindings/main/service'
import {data} from './store'
import {subscribeWindowBus} from './services/windowBus'
import {applyBackendPlayerState, applyParsedMidiToStore} from './services/backendMidiService'

const store = data()
const route = useRoute()
const isMainWindow = computed(() => route.path !== '/control')

const request = axios.create({
    baseURL: '/',
    headers: {
        Authorization: localStorage.getItem('token'),
    },
})

const pressedComputerKeys = new Set()
let unsubscribeWindowBus = null

// ========================
// 基础配置与设备同步
// ========================

async function getConfig() {
    const config = await Keyboard.SendConfig()
    store.config = {...store.config, ...config}
    setKeyColor()
}

function changeConfig() {
    Keyboard.ReceiveConfig(store.config)
}

function resetConfig() {
    Keyboard.ResetConfig().then((config) => {
        store.config = {...store.config, ...config}
        setKeyColor()
    })
}

async function getMidiDevices() {
    const devices = await Keyboard.GetMidiDevices()
    store.devices = {
        ...store.devices,
        ...devices,
        inMidiPool: devices.inMidiPool || {},
        outMidiPool: devices.outMidiPool || {},
        pedalStatus: devices.pedalStatus || {},
    }
    store.loaded = true
}

async function changeDevice(deviceType, deviceID) {
    // 只有切换输入设备时才需要重启监听；输出设备切换只需要清音，避免不必要地打断输入监听。
    const isInputDevice = deviceType === 'in'
    if (isInputDevice) {
        await Keyboard.MidiListenerStop()
    }

    const changed = await Keyboard.ChangeDevice(deviceType, Number(deviceID))
    if (!changed) {
        window.$message?.error?.('设备切换失败')
        if (isInputDevice) await Keyboard.MidiListenerStart()
        return
    }

    if (isInputDevice) {
        await Keyboard.MidiListenerStart()
    } else {
        await Keyboard.AllNotesOff()
    }

    await getMidiDevices()
    store.soundFontInfo = await Keyboard.GetSoundFontInfo()
}

// ========================
// 键盘配置与视觉样式
// ========================

async function initKeyboardConfig() {
    const skipKey = ['k', 'K', 'l', 'L', ';', "'", '\\', '|', '`', '~', '[', '{', ']', '}', 'p', 'P']
    const keyboardData = await (await fetch('/keyboard_config.json')).json()
    const keyMappingData = await (await fetch('/case-1.json')).json()
    const chordNames = await (await fetch('/ChordNames.json')).json()

    for (const key in keyMappingData) {
        keyMappingData[key] -= 12
    }
    store.keyMapping['case-1'] = keyMappingData
    store.chordsname = chordNames

    const reverseMapping = {}
    for (const key in keyMappingData) {
        if (skipKey.includes(key)) continue
        if (!(keyMappingData[key] in reverseMapping)) {
            reverseMapping[keyMappingData[key]] = key
        }
    }

    for (const item of keyboardData) {
        item.octave_key = item.note === 'C' ? item.note + item.octave : ''
        item.keyboard = item.index in reverseMapping ? reverseMapping[item.index] : ''
        if (item.pitch === 0) item.pitch = null
    }

    store.keyboardConfig = keyboardData
    store.keyboardLoaded = true
}

function darkenHexColor(hex, factor = 0.7) {
    hex = hex.replace('#', '')
    let r = parseInt(hex.slice(0, 2), 16)
    let g = parseInt(hex.slice(2, 4), 16)
    let b = parseInt(hex.slice(4, 6), 16)
    r = Math.round(r * factor)
    g = Math.round(g * factor)
    b = Math.round(b * factor)
    return `#${(1 << 24 | r << 16 | g << 8 | b).toString(16).slice(1)}`
}

function setKeyColor() {
    for (const key in store.config.colors) {
        document.documentElement.style.setProperty('--' + key, store.config.colors[key].color)
        document.documentElement.style.setProperty('--' + key + '-o', darkenHexColor(store.config.colors[key].color, 0.1) + '66')
    }
}

function resize() {
    if (!store.keyboardConfig.length) return

    let whiteKeyCount = 0
    const range = store.keyboardRange[store.config.keyboardType]
    for (const key of store.keyboardConfig.slice(range[0], range[1])) {
        if (key.color === 'white') whiteKeyCount++
    }
    if (whiteKeyCount === 0) return

    const ratio = window.innerWidth / window.innerHeight < 8 ? 1.7 : 1.6
    const whiteWidth = window.innerWidth / whiteKeyCount
    const blackWidth = whiteWidth / ratio
    const gap = blackWidth / ratio

    document.documentElement.style.setProperty('--black-key-width', blackWidth + 'px')
    document.documentElement.style.setProperty('--white-key-width', whiteWidth + 'px')
    document.documentElement.style.setProperty('--white-key-offset', -blackWidth + 'px')
    document.documentElement.style.setProperty('--black-key-offset', -gap * 0.7 + 'px')
}

function changeKeyboardType() {
    resize()
    changeConfig()
}

// ========================
// 实时按键状态
// ========================

function keyboardListener() {
    window.addEventListener('keydown', (event) => {
        // 设置中心窗口会有输入控件，避免用户打字时触发钢琴。
        if (window.location.hash.includes('/control')) return
        const mapping = store.keyMapping['case-1'] || {}
        if (!pressedComputerKeys.has(event.key) && event.key in mapping) {
            pressedComputerKeys.add(event.key)
            const midiKey = mapping[event.key]
            Keyboard.KeyboardPlay(midiKey)
            store.setKeyState(midiKey, true)
        }
    })

    window.addEventListener('keyup', (event) => {
        if (window.location.hash.includes('/control')) return
        const mapping = store.keyMapping['case-1'] || {}
        if (event.key in mapping) {
            pressedComputerKeys.delete(event.key)
            const midiKey = mapping[event.key]
            Keyboard.KeyboardStop(midiKey)
            store.setKeyState(midiKey, false)
        }
    })
}

function updateScaleByWindowHeight() {
    if (window.innerHeight < 220) {
        store.scale = 0.8
    } else if (window.innerHeight < 280) {
        store.scale = 0.9
    } else {
        store.scale = 1
    }
}

function registerBackendEvents() {
    Events.On('down', (event) => {
        const signal = event.data[0]
        store.activeKey[signal.value] = true
    })
    Events.On('up', (event) => {
        const signal = event.data[0]
        store.activeKey[signal.value] = false
    })
    Events.On('pressedDown', (event) => {
        const signal = event.data[0]
        store.pressedKey[signal.value] = true
    })
    Events.On('pressedUp', (event) => {
        const signal = event.data[0]
        store.pressedKey[signal.value] = false
    })
    Events.On('pedal', (event) => {
        if (store.devices.selectedInDevice === -1) return
        const pedal = event.data[0]
        store.devices.pedalStatus[store.devices.selectedInDevice] = {
            ...store.devices.pedalStatus[store.devices.selectedInDevice],
            ...pedal,
        }
    })
    Events.On('devices', (event) => {
        const devices = event.data[0]
        store.devices = {
            ...store.devices,
            ...devices,
            inMidiPool: devices.inMidiPool || {},
            outMidiPool: devices.outMidiPool || {},
            pedalStatus: devices.pedalStatus || {},
        }
    })
    Events.On('configChanged', (event) => {
        store.config = {...store.config, ...event.data[0]}
        setKeyColor()
    })
    Events.On('allNotesOff', () => {
        store.clearAllKeys()
    })
    Events.On('midiPlayerLoaded', (event) => {
        // Go 侧 MIDI 解析完成后会推送完整文件信息；主窗口和设置窗口都从 store.player 读取同一份数据。
        applyParsedMidiToStore(store, event.data[0])
    })
    Events.On('midiPlayerState', (event) => {
        // Go 播放器每隔一小段时间推送进度，前端只负责渲染，不再自己调度大量音符。
        applyBackendPlayerState(store, event.data[0])
    })
    Events.On('playbackKey', (event) => {
        const payload = event.data[0] || {}
        if (payload.midi === undefined) return
        store.playbackKey[payload.midi] = !!payload.pressed
    })
    Events.On('playbackClear', () => {
        store.playbackKey = {}
    })
    Events.On('soundFontChanged', (event) => {
        store.soundFontInfo = {...store.soundFontInfo, ...event.data[0]}
    })
}


function registerWindowBusEvents() {
    unsubscribeWindowBus = subscribeWindowBus((message) => {
        if (!message?.type) return

        switch (message.type) {
            case 'playback:key': {
                const {midi, pressed} = message.payload || {}
                if (midi === undefined) return
                store.playbackKey[midi] = !!pressed
                break
            }
            case 'playback:clear': {
                store.playbackKey = {}
                break
            }
            case 'hint:set': {
                const keys = message.payload?.keys || []
                store.hintKey = {}
                for (const key of keys) {
                    store.hintKey[key] = true
                }
                break
            }
            case 'hint:clear': {
                store.hintKey = {}
                break
            }
            case 'wrong:key': {
                const {midi, active} = message.payload || {}
                if (midi === undefined) return
                store.wrongKey[midi] = !!active
                break
            }
            case 'all:clear': {
                store.clearAllKeys()
                break
            }
            default:
                break
        }
    })
}

onMounted(async () => {
    WML.Reload()
    await initKeyboardConfig()
    await getConfig()
    await getMidiDevices()
    store.soundFontInfo = await Keyboard.GetSoundFontInfo()
    await Keyboard.MidiListenerStart()
    registerBackendEvents()
    registerWindowBusEvents()
    keyboardListener()
    resize()
    updateScaleByWindowHeight()

    window.addEventListener('resize', () => {
        resize()
        updateScaleByWindowHeight()
    })

    store.menuBar = false
    store.keyboardMenu = false
    store.showSetting = false
})

onBeforeUnmount(() => {
    unsubscribeWindowBus?.()
})

provide('request', request)
provide('store', store)
provide('changeDevice', changeDevice)
provide('setKeyColor', setKeyColor)
provide('changeConfig', changeConfig)
provide('changeKeyboardType', changeKeyboardType)
provide('resetConfig', resetConfig)
provide('Keyboard', Keyboard)
provide('resize', resize)
</script>

<style lang="scss">
.n-slider {
    --n-dot-height: 2px !important;
    --n-dot-width: 2px !important;
    --n-fill-color: #3058f8 !important;
    --n-fill-color-hover: #3063e7 !important;
    --n-font-size: 12px !important;
    --n-handle-size: 12px !important;
    --n-rail-height: 3px !important;
    --n-mark-font-size: 12px !important;
}

.n-select-menu {
    --n-height: 50vh !important;
    --n-option-font-size: 12px !important;
    --n-option-check-color: #1c38de !important;
    --n-option-text-color-active: #1c38de !important;
    --n-option-text-color-disabled: rgba(194, 194, 194, 1);
    --n-option-text-color-pressed: #0c387a !important;
    --n-loading-color: #1833a0 !important;
    --n-option-height: 30px !important;
}

.n-base-selection {
    --n-border-active: 1px solid #2951b0 !important;
    --n-border-focus: 1px solid #1e57e8 !important;
    --n-border-hover: 1px solid #2d68ff !important;
    --n-box-shadow-active: 0 0 0 2px rgba(28, 75, 224, 0.2) !important;
    --n-box-shadow-focus: 0 0 0 2px rgba(24, 67, 160, 0.2) !important;
    --n-caret-color: #1d60f1 !important;
    --n-loading-color: #2458dc !important;
}

.n-modal {
    box-shadow: none;
    filter: drop-shadow(0 0 8px rgba(0, 0, 0, 0.2));
}

.vc-chrome-toggle-btn, #input__label__hex__828 {
    display: none;
}

.n-radio-group {
    --n-font-size: 12px !important;
    --n-button-border-color: rgba(218, 218, 218, 0.25) !important;
    --n-button-border-color-active: #1d60f1 !important;
    --n-button-border-radius: 3px;
    --n-button-box-shadow-focus: inset 0 0 0 1px #1449bb, 0 0 0 2px rgba(24, 92, 160, 0.3) !important;
    --n-button-color: rgba(197, 197, 197, 0.4) !important;
    --n-button-color-active: rgba(21, 88, 213, 0.71) !important;
    --n-button-text-color-hover: #5275ff !important;
    --n-button-text-color-active: #ffffff !important;
    --n-height: 28px !important;
}

.n-radio__label {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100%;
}

.n-radio-button {
    padding: 0 8px !important;
}

@keyframes blurFadeIN {
    0% { opacity: 0; }
    100% { opacity: 1; }
}

.blurFadeIN {
    animation: blurFadeIN 0.3s ease;
    position: absolute;
}

@keyframes blurFadeOUT {
    0% { opacity: 1; }
    100% { opacity: 0; }
}

.blurFadeOUT {
    animation: blurFadeOUT 0.3s ease;
    position: absolute;
}
</style>
