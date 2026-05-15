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
import {Keyboard} from '../bindings/main/service'
import {data} from './store'
import {subscribeWindowBus} from './services/windowBus'
import {applyBackendPlayerState, applyParsedMidiToStore} from './services/backendMidiService'

const store = data()
const route = useRoute()
const isMainWindow = computed(() => route.path === '/')

const pressedComputerKeys = new Set()
const unsubscribeBackendEvents = []
let unsubscribeWindowBus = null
let unsubscribeKeyboardListener = null
let unsubscribeResizeListener = null

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
    unsubscribeKeyboardListener?.()

    const handleKeydown = (event) => {
        // 设置中心窗口会有输入控件，避免用户打字时触发钢琴。
        if (window.location.hash.includes('/control') || window.location.hash.includes('/midi')) return
        const mapping = store.keyMapping['case-1'] || {}
        if (!pressedComputerKeys.has(event.key) && event.key in mapping) {
            pressedComputerKeys.add(event.key)
            const midiKey = mapping[event.key]
            Keyboard.KeyboardPlay(midiKey)
            store.setKeyState(midiKey, true)
        }
    }

    const handleKeyup = (event) => {
        if (window.location.hash.includes('/control') || window.location.hash.includes('/midi')) return
        const mapping = store.keyMapping['case-1'] || {}
        if (event.key in mapping) {
            pressedComputerKeys.delete(event.key)
            const midiKey = mapping[event.key]
            Keyboard.KeyboardStop(midiKey)
            store.setKeyState(midiKey, false)
        }
    }

    window.addEventListener('keydown', handleKeydown)
    window.addEventListener('keyup', handleKeyup)

    unsubscribeKeyboardListener = () => {
        window.removeEventListener('keydown', handleKeydown)
        window.removeEventListener('keyup', handleKeyup)
        pressedComputerKeys.clear()
    }
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
    cleanupBackendEvents()

    const on = (eventName, callback) => {
        const unsubscribe = Events.On(eventName, callback)
        if (typeof unsubscribe === 'function') {
            unsubscribeBackendEvents.push(unsubscribe)
        }
    }

    on('down', (event) => {
        const signal = getEventPayload(event)
        if (!signal) return
        store.activeKey[signal.value] = true
    })
    on('up', (event) => {
        const signal = getEventPayload(event)
        if (!signal) return
        store.activeKey[signal.value] = false
    })
    on('pressedDown', (event) => {
        const signal = getEventPayload(event)
        if (!signal) return
        store.pressedKey[signal.value] = true
    })
    on('pressedUp', (event) => {
        const signal = getEventPayload(event)
        if (!signal) return
        store.pressedKey[signal.value] = false
    })
    on('pedal', (event) => {
        if (store.devices.selectedInDevice === -1) return
        const pedal = getEventPayload(event)
        if (!pedal) return
        store.devices.pedalStatus[store.devices.selectedInDevice] = {
            ...store.devices.pedalStatus[store.devices.selectedInDevice],
            ...pedal,
        }
    })
    on('devices', (event) => {
        const devices = getEventPayload(event) || {}
        store.devices = {
            ...store.devices,
            ...devices,
            inMidiPool: devices.inMidiPool || {},
            outMidiPool: devices.outMidiPool || {},
            pedalStatus: devices.pedalStatus || {},
        }
    })
    on('configChanged', (event) => {
        store.config = {...store.config, ...getEventPayload(event)}
        setKeyColor()
        resize()
    })
    on('allNotesOff', () => {
        store.clearAllKeys()
    })
    on('midiPlayerLoaded', (event) => {
        // Go 侧 MIDI 解析完成后会推送完整文件信息；主窗口和设置窗口都从 store.player 读取同一份数据。
        applyParsedMidiToStore(store, getEventPayload(event))
    })
    on('midiPlayerState', (event) => {
        // Go 播放器每隔一小段时间推送进度，前端只负责渲染，不再自己调度大量音符。
        applyBackendPlayerState(store, getEventPayload(event))
    })
    on('playbackKey', (event) => {
        const payload = getEventPayload(event) || {}
        if (payload.midi === undefined) return
        store.playbackKey[payload.midi] = !!payload.pressed
    })
    on('playbackClear', () => {
        store.playbackKey = {}
    })
    on('soundFontChanged', (event) => {
        store.soundFontInfo = {...store.soundFontInfo, ...getEventPayload(event)}
    })
}

function getEventPayload(event) {
    const data = event?.data
    return Array.isArray(data) ? data[0] : data
}

function cleanupBackendEvents() {
    while (unsubscribeBackendEvents.length) {
        unsubscribeBackendEvents.pop()?.()
    }
}


function registerWindowBusEvents() {
    unsubscribeWindowBus?.()
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

function registerResizeListener() {
    unsubscribeResizeListener?.()

    const handleResize = () => {
        resize()
        updateScaleByWindowHeight()
    }

    window.addEventListener('resize', handleResize)
    unsubscribeResizeListener = () => window.removeEventListener('resize', handleResize)
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
    registerResizeListener()

    store.menuBar = false
    store.keyboardMenu = false
    store.showSetting = false
})

onBeforeUnmount(() => {
    cleanupBackendEvents()
    unsubscribeWindowBus()
    unsubscribeKeyboardListener()
    unsubscribeResizeListener()
})

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

</style>
