<template>
    <n-config-provider
        :date-locale="dateZhCN"
        :locale="zhCN"
        :style="{opacity: isMainWindow ? store.config.opacity / 100 : 1}"
    >
        <n-modal-provider>
            <n-loading-bar-provider>
                <n-message-provider>
                    <n-notification-provider :max="3" placement="top-right">
                        <FeedbackBridge />
                        <router-view />
                    </n-notification-provider>
                </n-message-provider>
            </n-loading-bar-provider>
        </n-modal-provider>
    </n-config-provider>
</template>

<script setup>
import {computed, defineComponent, h, onBeforeUnmount, onMounted, provide} from 'vue'
import {useRoute} from 'vue-router'
import {
    dateZhCN,
    NConfigProvider,
    NLoadingBarProvider,
    NMessageProvider,
    NModalProvider,
    NNotificationProvider,
    useMessage,
    useNotification,
    zhCN
} from 'naive-ui'
import {Events, WML} from '@wailsio/runtime'
import {Keyboard} from '../bindings/main/service'
import {data} from './store'
import {createComputerKeyboard} from './services/computerKeyboard'

const store = data()
const route = useRoute()
const isMainWindow = computed(() => route.path === '/')

const unsubscribeBackendEvents = []
const keyboardLabelSkipKeys = ['k', 'K', 'l', 'L', ';', "'", '\\', '|', '`', '~', '[', '{', ']', '}', 'p', 'P']
let unsubscribeKeyboardListener = null
let unsubscribeResizeListener = null

const FeedbackBridge = defineComponent({
    name: 'FeedbackBridge',
    setup() {
        const message = useMessage()
        const notification = useNotification()

        window.$message = message
        window.$notify = {
            error(title, content) {
                notification.error({
                    title,
                    content,
                    duration: 5200,
                    keepAliveOnHover: true,
                })
            },
            success(title, content) {
                notification.success({
                    title,
                    content,
                    duration: 2800,
                    keepAliveOnHover: true,
                })
            },
            warning(title, content) {
                notification.warning({
                    title,
                    content,
                    duration: 4200,
                    keepAliveOnHover: true,
                })
            },
            info(title, content) {
                notification.info({
                    title,
                    content,
                    duration: 3200,
                    keepAliveOnHover: true,
                })
            },
        }

        return () => h('span', {style: 'display: none'})
    },
})

// ========================
// 基础配置与设备同步
// ========================

async function getConfig() {
    const defaultConfig = await Keyboard.GetDefaultConfig()
    applyConfig(defaultConfig)

    const config = await Keyboard.SendConfig()
    applyConfig(config)
    setKeyColor()
}

let confirmedConfig = null
let pendingConfig = {}
let savingConfig = {}
let configSaveTask = null
const copyConfig = value => JSON.parse(JSON.stringify(value))

function changeConfig() {
    for (const [key, value] of Object.entries(store.config)) {
        if (key === 'revision' || key === 'version') continue
        if (JSON.stringify(value) !== JSON.stringify(({...confirmedConfig, ...savingConfig, ...pendingConfig})[key])) {
            pendingConfig[key] = copyConfig(value)
        }
    }
    if (!configSaveTask) configSaveTask = savePendingConfig().finally(() => { configSaveTask = null })
    return configSaveTask
}

async function savePendingConfig() {
    while (Object.keys(pendingConfig).length) {
        const changes = pendingConfig
        pendingConfig = {}
        savingConfig = changes
        try {
            const latest = await Keyboard.SendConfig()
            const [ok, error] = await Keyboard.ReceiveConfig({...latest, ...changes})
            if (!ok) throw new Error(error)
            const saved = await Keyboard.SendConfig()
            savingConfig = {}
            applyConfig(saved)
        } catch (error) {
            savingConfig = {}
            pendingConfig = {...changes, ...pendingConfig}
            store.config = {...store.config, ...pendingConfig}
            window.$message?.error(`设置未保存：${String(error)}。修改已保留，请重试。`)
            return false
        }
    }
    return true
}

async function resetConfig() {
    if (configSaveTask && !await configSaveTask) return
    try {
        const config = await Keyboard.ResetConfig()
        pendingConfig = {}
        applyConfig(config)
        setKeyColor()
    } catch (error) { window.$message?.error(`恢复默认设置失败：${String(error)}`) }
}

function applyConfig(config) {
    if (!config || (confirmedConfig && config.revision < confirmedConfig.revision)) return
    confirmedConfig = copyConfig(config)
    store.config = {...store.config, ...config, ...savingConfig, ...pendingConfig}
    updateKeyboardMappingLabels()
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

async function getMidiWindowState() {
    if (typeof Keyboard.GetMidiWindowOpen !== 'function') return
    store.midiWindowOpen = await Keyboard.GetMidiWindowOpen()
}

async function changeDevice(deviceType, deviceID) {
    try {
        if (!await Keyboard.ChangeDevice(deviceType, Number(deviceID))) {
            window.$message?.error('设备切换失败')
        }
        await getMidiDevices()
    } catch (error) { window.$message?.error(String(error)) }
}

// ========================
// 键盘配置与视觉样式
// ========================

async function initKeyboardConfig() {
    const keyboardData = await (await fetch('/keyboard_config.json')).json()
    const chordNames = await (await fetch('/ChordNames.json')).json()

    store.chordsName = chordNames

    for (const item of keyboardData) {
        item.octave_key = item.note === 'C' ? item.note + item.octave : ''
        item.keyboard = ''
        if (item.pitch === 0) item.pitch = null
    }

    store.keyboardConfig = keyboardData
    updateKeyboardMappingLabels()
    store.keyboardLoaded = true
}

function updateKeyboardMappingLabels() {
    if (!store.keyboardConfig.length) return

    const reverseMapping = {}
    const mapping = store.activeKeyMapping || {}
    for (const key in mapping) {
        if (keyboardLabelSkipKeys.includes(key)) continue
        const midiKey = Number(mapping[key])
        if (!Number.isFinite(midiKey)) continue
        if (!(midiKey in reverseMapping)) {
            reverseMapping[midiKey] = formatComputerKeyLabel(key)
        }
    }

    for (const item of store.keyboardConfig) {
        item.keyboard = item.index in reverseMapping ? reverseMapping[item.index] : ''
    }
}

function formatComputerKeyLabel(key) {
    if (key === ' ') return 'Space'
    return key
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
    if (!store.config.colors) return
    for (const key in store.config.colors) {
        document.documentElement.style.setProperty('--' + key, store.config.colors[key].color)
        document.documentElement.style.setProperty('--' + key + '-o', darkenHexColor(store.config.colors[key].color, 0.1) + '66')
    }
}

function resize() {
    if (!store.keyboardConfig.length) return
    let whiteKeyCount = 0
    const range = store.keyboardRange[store.config.keyboardType]
    if (!range) return
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
    const input = createComputerKeyboard({
        mapping: () => store.activeKeyMapping || {},
        enabled: () => isMainWindow.value,
        press: note => { store.setKeyState(note, true); Keyboard.KeyboardPlay(note) },
        release: note => { store.setKeyState(note, false); Keyboard.KeyboardStop(note) },
    })
    window.addEventListener('keydown', input.keydown)
    window.addEventListener('keyup', input.keyup)
    window.addEventListener('blur', input.clear)
    unsubscribeKeyboardListener = () => {
        input.clear()
        window.removeEventListener('keydown', input.keydown)
        window.removeEventListener('keyup', input.keyup)
        window.removeEventListener('blur', input.clear)
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
        store.backendPressedKey[signal.value] = true
    })
    on('pressedUp', (event) => {
        const signal = getEventPayload(event)
        if (!signal) return
        store.backendPressedKey[signal.value] = false
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
        applyConfig(getEventPayload(event))
        setKeyColor()
        resize()
    })
    on('midiWindowState', (event) => {
        const payload = getEventPayload(event)
        store.midiWindowOpen = !!payload?.open
        if (!store.midiWindowOpen) {
            store.clearMidiVisualKeys()
        }
    })
    on('midiPlayerState', (event) => {
        store.midiPlayerState = {
            ...store.midiPlayerState,
            ...getEventPayload(event),
        }
    })
    on('midiPlaybackKey', (event) => {
        const signal = getEventPayload(event)
        if (!signal) return
        store.setMidiVisualKey(signal.note, signal.hand, !!signal.active, 'playback')
    })
    on('midiFollowHint', (event) => {
        const step = getEventPayload(event)
        store.clearMidiHints()
        if (!step?.notes?.length) return
        for (const note of step.notes) {
            store.setMidiVisualKey(note.note, note.hand, true, 'followHint')
        }
    })
    on('midiVisualClear', () => {
        store.clearMidiVisualKeys()
    })
    on('allNotesOff', () => {
        store.clearAllKeys()
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
    await getMidiWindowState()
    await Keyboard.MidiListenerStart()
    registerBackendEvents()
    keyboardListener()
    resize()
    updateScaleByWindowHeight()
    registerResizeListener()

    store.menuBar = false
    store.keyboardMenu = false
    store.showSetting = false
    if (isMainWindow.value && import.meta.env.PROD) {
        Keyboard.CheckUpdate().then(info => {
            if (info.available) window.$notify?.info('发现新版本', `${info.version} 已可用，请在设置中心「关于」中安装并重启。`)
        }).catch(() => { /* Offline startup must not interrupt playing. */ })
    }
})

onBeforeUnmount(() => {
    cleanupBackendEvents()
    unsubscribeKeyboardListener?.()
    unsubscribeResizeListener?.()
})

provide('store', store)
provide('changeDevice', changeDevice)
provide('setKeyColor', setKeyColor)
provide('changeConfig', changeConfig)
provide('changeKeyboardType', changeKeyboardType)
provide('resetConfig', resetConfig)
provide('applyConfig', applyConfig)
provide('updateKeyboardMappingLabels', updateKeyboardMappingLabels)
provide('Keyboard', Keyboard)
provide('resize', resize)
</script>

<style lang="scss">

</style>
