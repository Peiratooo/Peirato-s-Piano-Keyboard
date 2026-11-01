<template>
    <n-config-provider
        :date-locale="dateZhCN"
        :locale="zhCN"
        :theme="isMainWindow ? darkTheme : undefined"
        :theme-overrides="themeOverrides"
        :style="{...pianoUiTheme.cssVariables, opacity: isMainWindow && store.mainDisplayMode === 'simple' ? store.config.opacity / 100 : 1}"
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
import {computed, onBeforeUnmount, onMounted, provide} from 'vue'
import {useRoute} from 'vue-router'
import {
    dateZhCN,
    darkTheme,
    NConfigProvider,
    NLoadingBarProvider,
    NMessageProvider,
    NModalProvider,
    NNotificationProvider,
    zhCN
} from 'naive-ui'
import {Events, WML} from '@wailsio/runtime'
import {Keyboard} from '../bindings/main/service'
import {data} from './store'
import FeedbackBridge from './components/FeedbackBridge.vue'
import {useUpdater} from './store/updater'
import {createComputerKeyboard, setKeyboardMappingLabels} from './services/computerKeyboard'
import {acceptLivePerformance, subscribeLivePerformance} from './services/livePerformance'
import {getKeyLabelColor, getPianoUiTheme} from './services/uiTheme'
import {changeColor, composite, rgba} from 'seemly'

const store = data()
provide('subscribeKeyboardFeedback', listener => {
    const subscriptions = [
        Events.On('midiPlaybackKey', event => {
            const signal = event.data
            if (signal?.active && signal.attack && store.midiPlayerState.status === 'playing') listener({type: 'attack', note: signal.note, hand: signal.hand})
        }),
        Events.On('midiVisualClear', () => listener({type: 'clear'})),
        Events.On('allNotesOff', () => listener({type: 'clear'})),
        subscribeLivePerformance(signal => {
            if (signal.type === 'clear') listener({type: 'clear'})
            else if (signal.type === 'noteOn') listener({type: 'attack', note: signal.note})
        }),
    ]
    return () => subscriptions.forEach(unsubscribe => unsubscribe())
})
const updater = useUpdater()
const route = useRoute()
const isMainWindow = computed(() => route.path === '/')
const pianoUiTheme = computed(() => getPianoUiTheme(store.config.colors, isMainWindow.value ? 'dark' : 'light'))
const themeOverrides = computed(() => {
    const theme = pianoUiTheme.value
    if (isMainWindow.value) return theme.naiveOverrides
    return {
        common: {
            primaryColor: theme.accent,
            primaryColorHover: theme.accentHover,
            primaryColorPressed: theme.accentPressed,
            primaryColorSuppl: theme.accent,
        },
        Button: theme.naiveOverrides.Button,
    }
})

const unsubscribeBackendEvents = []
let unsubscribeKeyboardListener = null
let unsubscribeResizeListener = null
let lastWindowSaveError = 0


// ========================
// 基础配置与设备同步
// ========================

async function getConfig() {
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
        if (key === 'revision' || key === 'version' || key === 'mainWindowLayouts') continue
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
            // Window-size saves can advance the configuration revision between
            // reading it and submitting a slider change. Rebase only that conflict.
            for (let attempt = 0; attempt < 3; attempt++) {
                const latest = await Keyboard.SendConfig()
                const [ok, error] = await Keyboard.ReceiveConfig({...latest, ...changes})
                if (ok) break
                if (attempt < 2 && String(error).includes('设置已被其他窗口修改')) continue
                throw new Error(error)
            }
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
    if (configSaveTask && !await configSaveTask) return false
    try {
        const config = await Keyboard.ResetConfig()
        pendingConfig = {}
        applyConfig(config)
        setKeyColor()
        return true
    } catch (error) { console.error(error); window.$message?.error("恢复默认设置失败，原有设置已保留，请重试。"); return false }
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
        softwareSustain: !!devices.softwareSustain,
    }
    store.loaded = true
}

async function changeDevice(deviceType, deviceID) {
    try {
        if (!await Keyboard.ChangeDevice(deviceType, Number(deviceID))) {
            window.$message?.error('设备切换失败')
        }
        await getMidiDevices()
    } catch (error) { window.$message?.error(String(error)) }
}

async function setSoftwareSustain(enabled) {
    try {
        store.devices.softwareSustain = await Keyboard.SetSoftwareSustain(enabled)
    } catch (error) {
        window.$message?.error(`切换延音踏板失败：${String(error)}`)
        await getMidiDevices()
    }
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
    setKeyboardMappingLabels(store.keyboardConfig, store.activeKeyMapping)
}

function setKeyColor() {
    for (const [key, {color}] of Object.entries(store.config.colors)) {
        const style = document.documentElement.style
        const [r, g, b] = rgba(color)
        style.setProperty('--' + key, color)
        style.setProperty('--' + key + '-o', `rgba(${Math.round(r * .1)}, ${Math.round(g * .1)}, ${Math.round(b * .1)}, .4)`)
        if (key.includes('Key')) {
            style.setProperty('--' + key + '-label', getKeyLabelColor(color))
            const base = key.startsWith('black') ? '#333333' : '#f6f6f6'
            style.setProperty('--' + key + '-sustained-label', getKeyLabelColor(composite(base, changeColor(color, {alpha: .38}))))
        }
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

const interactiveRequests = new Map()
function setInteractiveKey(note, pressed) {
    store.setKeyState(note, pressed)
    const send = () => pressed ? Keyboard.KeyboardPlay(note) : Keyboard.KeyboardStop(note)
    const previous = interactiveRequests.get(note)
    const request = previous ? previous.then(send, send) : send()
    interactiveRequests.set(note, request)
    request.then(() => {
        if (interactiveRequests.get(note) === request) interactiveRequests.delete(note)
    }, error => {
        if (interactiveRequests.get(note) === request) interactiveRequests.delete(note)
        window.$message?.error(`演奏输入失败：${String(error)}`)
    })
}

function keyboardListener() {
    unsubscribeKeyboardListener?.()
    const input = createComputerKeyboard({
        mapping: () => store.activeKeyMapping || {},
        enabled: () => isMainWindow.value && !store.onlineWorkspaceOpen,
        press: note => setInteractiveKey(note, true),
        release: note => setInteractiveKey(note, false),
    })
    window.addEventListener('keydown', input.keydown)
    window.addEventListener('keyup', input.keyup)
    window.addEventListener('blur', input.clear)
    window.addEventListener('focusin', input.focusin)
    unsubscribeKeyboardListener = () => {
        input.clear()
        window.removeEventListener('keydown', input.keydown)
        window.removeEventListener('keyup', input.keyup)
        window.removeEventListener('blur', input.clear)
        window.removeEventListener('focusin', input.focusin)
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
        unsubscribeBackendEvents.push(Events.On(eventName, event => callback(event.data)))
    }

    if (isMainWindow.value) on('livePerformance', acceptLivePerformance)
    on('mainWindowPresentation', presentation => {
        if (presentation?.mode) store.mainDisplayMode = presentation.mode
    })
    on('mainWindowPresentationSaveError', payload => {
        if (!isMainWindow.value || Date.now() - lastWindowSaveError < 10000) return
        lastWindowSaveError = Date.now()
        window.$message?.error('窗口尺寸未保存：' + (payload?.error || '请检查配置目录是否可写'))
    })

    on('down', (signal) => {
        if (!signal) return
        store.activeKey[signal.value] = true
    })
    on('up', (signal) => {
        if (!signal) return
        store.activeKey[signal.value] = false
    })
    on('pressedDown', (signal) => {
        if (!signal) return
        if (signal.hand !== 'unknown' || !store.pressedKey[signal.value]) store.setPressedHand(signal.value, signal.hand)
        store.backendPressedKey[signal.value] = true
    })
    on('pressedUp', (signal) => {
        if (!signal) return
        store.backendPressedKey[signal.value] = false
        if (!store.pressedKey[signal.value]) delete store.pressedKeyHands[signal.value]
    })
    on('pedal', (pedal) => {
        if (!pedal) return
        const source = pedal.deviceID ?? store.devices.selectedInDevice
        store.devices.pedalStatus[source] = {
            ...store.devices.pedalStatus[source],
            ...pedal,
        }
    })
    on('softwareSustain', enabled => {
        store.devices.softwareSustain = !!enabled
    })
    on('devices', (devices) => {
        store.devices = {
            ...store.devices,
            ...devices,
            inMidiPool: devices.inMidiPool || {},
            outMidiPool: devices.outMidiPool || {},
            pedalStatus: devices.pedalStatus || {},
            softwareSustain: !!devices.softwareSustain,
        }
    })
    on('updateState', state => updater.apply(state))
    on('configChanged', (config) => {
        applyConfig(config)
        setKeyColor()
        resize()
    })
    on('midiPlayerState', (state) => {
        if (!state) return
        store.midiPlayerState = {
            ...state,
            currentStep: state.currentStep ?? null,
            error: state.error || '',
        }
    })
    on('midiPlaybackKey', (signal) => {
        if (!signal) return
        store.setMidiVisualKey(signal.note, signal.hand, !!signal.active, 'playback')
    })
    on('midiFollowHint', (step) => {
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
    registerBackendEvents()
    await initKeyboardConfig()
    await getConfig()
    await getMidiDevices()
    await Keyboard.MidiListenerStart()
    keyboardListener()
    resize()
    updateScaleByWindowHeight()
    registerResizeListener()

    store.menuBar = false
    store.keyboardMenu = false
    store.showSetting = false
    try {
        await updater.sync()
        if (route.path === '/control' || (isMainWindow.value && import.meta.env.PROD)) await updater.check()
    } catch { /* A transient update failure must not interrupt playing. */ }

})

onBeforeUnmount(() => {
    cleanupBackendEvents()
    unsubscribeKeyboardListener?.()
    unsubscribeResizeListener?.()
})

provide('store', store)
provide('changeDevice', changeDevice)
provide('setSoftwareSustain', setSoftwareSustain)
provide('setKeyColor', setKeyColor)
provide('changeConfig', changeConfig)
provide('changeKeyboardType', changeKeyboardType)
provide('resetConfig', resetConfig)
provide('applyConfig', applyConfig)
provide('updateKeyboardMappingLabels', updateKeyboardMappingLabels)
provide('Keyboard', Keyboard)
provide('setInteractiveKey', setInteractiveKey)
provide('resize', resize)
</script>

<style lang="scss">

</style>
