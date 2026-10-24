<template>
    <main ref="containerRef" class="piano-window" :class="{ 'is-simple': isSimple, 'show-tools': toolbarVisible || panel }"
        :style="windowStyle" @pointerdown="dismissOutside" @mouseenter="showToolbar" @mouseleave="hideToolbar">
        <div class="waterfall-area" :aria-hidden="isSimple">
            <WaterfallCanvas v-show="!isMidi" :layout="layout" :width="size.width" :height="waterfallHeight"
                :active="!isSimple && !isMidi" :colors="store.config.colors" />
            <div v-show="isMidi" class="midi-waterfall-layer" :style="{top: midiCanvasTop + 'px'}">
                <MidiWaterfallCanvas :layout="layout" :width="size.width" :height="midiCanvasHeight"
                    :active="!isSimple && isMidi" :colors="store.config.colors" :timeline="session.timeline" :player-state="session.state" />
            </div>
        </div>

        <header class="toolbar">
            <div class="toolbar-left" data-piano-control>
                <button class="tool-button" type="button" :aria-expanded="panel === 'menu'" aria-label="打开主菜单"
                    @pointerdown.prevent @click="openPanel('menu')"><PhList :size="20" /><span>菜单</span></button>
                <NSelect class="performance-mode" :value="store.mainMode" :options="modeOptions" size="small"
                    :disabled="modeSwitching || session.busy" :menu-props="{'data-piano-control': ''}"
                    aria-label="演奏模式" @update:value="changeMode" />
            </div>
            <div class="drag-region" @dblclick="toggleMaximize"><span v-if="!isSimple">{{ isMidi ? session.selectedSong?.name || '选择一首 MIDI' : 'Peirato’s Piano' }}</span></div>
            <div class="toolbar-right" data-piano-control>
                <NRadioGroup class="display-switch" data-preserve-piano-holds :value="store.mainDisplayMode" :disabled="switching" size="small" aria-label="显示方式" @update:value="setDisplayMode">
                    <NRadioButton value="simple">简单</NRadioButton><NRadioButton value="waterfall">瀑布</NRadioButton>
                </NRadioGroup>
                <button class="tool-button" type="button" :aria-expanded="panel === 'settings'" title="钢琴设置"
                    @pointerdown.prevent @click="openPanel('settings')"><PhSlidersHorizontal :size="20" /><span>钢琴</span></button>
                <span class="toolbar-separator"></span>
                <button class="window-button" type="button" aria-label="最小化" title="最小化" @pointerdown.prevent @click="minimize"><PhMinus :size="16" /></button>
                <button v-if="!isSimple" class="window-button" type="button" aria-label="最大化或还原" title="最大化 / 还原"
                    @pointerdown.prevent @click="toggleMaximize"><PhArrowsOutSimple :size="15" /></button>
                <button class="window-button close-window" type="button" aria-label="退出软件" title="退出" @pointerdown.prevent @click="requestQuit"><PhX :size="17" /></button>
            </div>
        </header>

        <RecordingControls v-if="!isMidi" :recording="recording" :simple="isSimple"
            @show-draft="setDisplayMode('waterfall')" />

        <div v-show="isMidi && !isSimple" ref="midiControlsRef" class="midi-workspace-controls" data-piano-control>
            <div class="midi-options-row">
                <NButton class="library-toggle" :type="panel === 'library' ? 'primary' : 'default'" size="small" :aria-expanded="panel === 'library'" @click="openPanel('library')">
                    <template #icon><PhMusicNotes :size="18" /></template>曲库
                </NButton>
                <MidiPracticeControls :session="session" />
            </div>
            <MidiTransport :session="session" />
        </div>

        <Transition name="panel">
            <section v-if="panel === 'menu' && !isSimple" class="main-menu floating-panel" data-piano-control aria-label="主菜单">
                <button class="menu-close" type="button" aria-label="收起菜单" @click="closePanel"><PhX :size="17" /></button>
                <PhPianoKeys class="menu-mark" :size="36" weight="light" />
                <h1>Peirato’s Piano</h1>
                <button class="start-playing" type="button" @click="startFreePlay"><PhPlay :size="17" weight="fill" />{{ isMidi ? '自由演奏' : '开始弹奏' }}<PhArrowRight :size="18" /></button>
                <div class="menu-links">
                    <button type="button" @click="changeMode('play')"><PhMusicNotes :size="20" /><span>MIDI 播放</span><PhArrowRight :size="16" /></button>
                    <button type="button" @click="changeMode('follow')"><PhPianoKeys :size="20" /><span>跟弹练习</span><PhArrowRight :size="16" /></button>
                    <button type="button" @click="openFullSettings"><PhGearSix :size="20" /><span>完整设置</span><PhArrowUpRight :size="16" /></button>
                </div>
                <footer class="menu-footer">
                    <span class="input-status"><i :class="{ connected: hasMidiInput }"></i>{{ hasMidiInput ? 'MIDI 键盘已连接' : '电脑键盘 / 鼠标演奏' }}</span>
                    <button type="button" @click="requestQuit"><PhPower :size="15" />退出</button>
                </footer>
                <p class="menu-hint">{{ isMidi ? '选择曲目后开始播放或跟弹' : '弹下第一个音，菜单自动收起' }}</p>
            </section>
        </Transition>

        <Transition name="panel">
            <MidiLibraryPanel v-if="panel === 'library' && !isSimple && isMidi" class="floating-panel library-panel"
                :session="session" @selected="closePanel" />
        </Transition>

        <Transition name="panel">
            <QuickPianoSettings v-if="panel === 'settings' && !isSimple" class="floating-panel settings-panel"
                @close="closePanel" @open-full="openFullSettings" />
        </Transition>

        <div class="keyboard-zone" :style="{ height: keyboardHeight + 'px' }">
            <div v-if="!isSimple" class="strike-line"></div>
            <ClassicKeyboard v-if="store.keyboardLoaded" :layout="layout" />
        </div>

        <div v-if="store.loaded" class="chord-status" :class="{ compact: isSimple }"><Chord /></div>
        <div v-if="store.loaded && store.config.showPedal" class="pedal-status" :class="{ compact: isSimple }">
            <span v-for="pedal in pedalIndicators" :key="pedal.key" :class="{ active: pedal.down }" :style="{ '--pedal-color': pedal.color }"><i></i>{{ pedal.label }}</span>
        </div>
    </main>
</template>

<script setup>
import {computed, inject, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {Events, Window} from '@wailsio/runtime'
import {NButton, NRadioButton, NRadioGroup, NSelect} from 'naive-ui'
import {PhList, PhSlidersHorizontal, PhMinus, PhArrowsOutSimple, PhX, PhPianoKeys, PhPlay,
    PhArrowRight, PhArrowUpRight, PhMusicNotes, PhGearSix, PhPower} from '@phosphor-icons/vue'
import ClassicKeyboard from '../components/ClassicKeyboard.vue'
import Chord from '../components/Chord.vue'
import WaterfallCanvas from '../components/WaterfallCanvas.vue'
import QuickPianoSettings from '../components/QuickPianoSettings.vue'
import MidiLibraryPanel from '../components/MidiLibraryPanel.vue'
import MidiPracticeControls from '../components/MidiPracticeControls.vue'
import MidiTransport from '../components/MidiTransport.vue'
import MidiWaterfallCanvas from '../components/MidiWaterfallCanvas.vue'
import RecordingControls from '../components/RecordingControls.vue'
import {createKeyboardLayout} from '../services/keyboardLayout'
import {subscribeLivePerformance} from '../services/livePerformance'
import {useMidiSession} from '../services/midiSession'
import {useRecordingSession} from '../services/recordingSession'
import {getPianoUiTheme} from '../services/uiTheme'

const store = inject('store')
const Keyboard = inject('Keyboard')
const route = useRoute()
const router = useRouter()
const session = useMidiSession({store, Keyboard})
const recording = useRecordingSession({Keyboard, getPlayerState: () => store.midiPlayerState, onSaved: async song => {
    await session.refreshLibrary()
    if (song?.id) await session.selectSong(song.id)
}})
const uiTheme = computed(() => getPianoUiTheme(store.config.colors))
const containerRef = ref(null)
const midiControlsRef = ref(null)
const midiControlsHeight = ref(160)
const panel = ref('menu')
const switching = ref(false)
const modeSwitching = ref(false)
const toolbarVisible = ref(true)
const size = reactive({width: window.innerWidth, height: window.innerHeight})
const isSimple = computed(() => store.mainDisplayMode === 'simple')
const isMidi = computed(() => store.mainMode !== 'free')
const modeOptions = [{label: '自由演奏', value: 'free'}, {label: 'MIDI 播放', value: 'play'}, {label: '跟弹练习', value: 'follow'}]
const keyboardHeight = computed(() => isSimple.value ? size.height : Math.min(220, Math.max(140, Math.round(size.height * 0.22))))
const waterfallHeight = computed(() => Math.max(0, size.height - keyboardHeight.value))
const midiCanvasTop = computed(() => 62 + midiControlsHeight.value)
const midiCanvasHeight = computed(() => Math.max(0, waterfallHeight.value - midiCanvasTop.value))
const windowStyle = computed(() => ({
    ...uiTheme.value.cssVariables,
    '--keyboard-height': keyboardHeight.value + 'px',
    '--waterfall-height': waterfallHeight.value + 'px',
    '--menu-top': Math.max(76, (waterfallHeight.value - 400) / 2) + 'px',
}))
const visibleKeys = computed(() => {
    const [start, end] = store.keyboardRange[store.config.keyboardType] || store.keyboardRange[0]
    return store.keyboardConfig.slice(start, end)
})
const layout = computed(() => createKeyboardLayout(visibleKeys.value, size.width))
const hasMidiInput = computed(() => store.devices.selectedInDevice !== -1)
const pedalIndicators = computed(() => [
    {key: 'damperPedal', label: '延音'}, {key: 'softPedal', label: '柔音'}, {key: 'sostenutoPedal', label: '持音'},
].map(item => ({...item,
    color: store.config.colors[item.key]?.color || '#acd3a7',
    down: (item.key === 'damperPedal' && store.devices.softwareSustain)
        || Object.values(store.devices.pedalStatus || {}).some(state => state?.[item.key]),
})))
let observer
let hideTimer
let unsubscribeLive
let unsubscribeMidiRequest

function releaseControlFocus() {
    if (document.activeElement?.closest?.('[data-piano-control]')) document.activeElement.blur()
}
function closePanel() {
    panel.value = null
    releaseControlFocus()
}
function dismissOutside(event) {
    if (event.target.closest?.('[data-piano-control]')) return
    closePanel()
}
async function openPanel(name) {
    if (isSimple.value && !await setDisplayMode('waterfall')) return
    panel.value = panel.value === name ? null : name
    releaseControlFocus()
}
async function setDisplayMode(mode) {
    if (switching.value) return false
    if (store.mainDisplayMode === mode) { releaseControlFocus(); return true }
    switching.value = true
    try {
        const presentation = await Keyboard.SetMainDisplayMode(mode)
        store.mainDisplayMode = presentation.mode
        closePanel()
        showToolbar()
        await nextTick()
        return true
    } catch (error) {
        window.$message?.error('切换显示方式失败：' + String(error))
        return false
    } finally {
        switching.value = false
        releaseControlFocus()
    }
}
async function changeMode(mode) {
    if (modeSwitching.value) return false
    modeSwitching.value = true
    try {
        if (mode !== 'free' && recording.state.status === 'recording') {
            if (!await recording.stop()) return false
        }
        if (mode !== 'free' && recording.state.status === 'draft'
            && !recording.state.backupSaved && !await recording.stop()) return false
        if (mode !== 'free' && recording.previewing && !await recording.stopPreview()) return false
        if (mode !== 'free' && isSimple.value && !await setDisplayMode('waterfall')) return false
        await session.initialize()
        if (!await session.setMode(mode)) return false
        panel.value = mode === 'free' ? null : 'library'
        releaseControlFocus()
        return true
    } finally { modeSwitching.value = false }
}
async function startFreePlay() {
    if (isMidi.value) await changeMode('free')
    else closePanel()
}
async function consumeRequestedMode() {
    const mode = route.query.midiMode
    if (!mode) return
    if (await changeMode(mode === 'follow' ? 'follow' : 'play')) {
        const query = {...route.query}
        delete query.midiMode
        await router.replace({path: '/', query})
    }
}
async function requestQuit() {
    if (recording.state.status === 'recording'
        || (recording.state.status === 'draft' && !recording.state.backupSaved)) {
        if (!await recording.stop()) {
            window.$message?.error('草稿未备份，请先保存 MIDI')
            return
        }
    }
    if (recording.state.status === 'draft') {
        if (!recording.state.backupSaved) {
            window.$message?.error('草稿尚未备份，请先保存 MIDI')
            return
        }
        if (!window.confirm('录制草稿将保留，下次可继续保存。确定退出？')) return
    }
    try { await Keyboard.Quit() } catch (error) { window.$message?.error('退出失败：' + String(error)) }
}
async function openFullSettings() {
    closePanel()
    try { await Keyboard.OpenControlCenter() }
    catch (error) { window.$message?.error('无法打开完整设置：' + String(error)) }
}
function showToolbar() { clearTimeout(hideTimer); toolbarVisible.value = true }
function hideToolbar() { clearTimeout(hideTimer); hideTimer = setTimeout(() => { toolbarVisible.value = false }, 650) }
function keydown(event) {
    if (event.key !== 'Escape' || !panel.value) return
    event.preventDefault()
    event.stopPropagation()
    closePanel()
}
async function minimize() {
    try { await Window.Minimise() } catch (error) { window.$message?.error(String(error)) }
    releaseControlFocus()
}
async function toggleMaximize() {
    if (isSimple.value) return
    try { await Window.ToggleMaximise() } catch (error) { window.$message?.error(String(error)) }
    releaseControlFocus()
}

onMounted(async () => {
    observer = new ResizeObserver(entries => {
        for (const entry of entries) {
            if (entry.target === containerRef.value) {
                size.width = entry.contentRect.width
                size.height = entry.contentRect.height
            } else if (entry.contentRect.height > 0) {
                midiControlsHeight.value = entry.borderBoxSize?.[0]?.blockSize || entry.target.getBoundingClientRect().height
            }
        }
    })
    observer.observe(containerRef.value)
    observer.observe(midiControlsRef.value)
    unsubscribeLive = subscribeLivePerformance(signal => {
        if (signal.type === 'noteOn' && panel.value === 'menu') closePanel()
    })
    window.addEventListener('keydown', keydown, true)
    unsubscribeMidiRequest = Events.On('mainMidiModeRequested', event => changeMode(event.data?.mode === 'follow' ? 'follow' : 'play'))
    try {
        const state = await Keyboard.GetMainWindowPresentation()
        store.mainDisplayMode = state.mode
    } catch (error) { console.error('无法读取主窗口状态', error) }
    await session.initialize()
    await recording.initialize()
    await consumeRequestedMode()
})
watch(() => route.query.midiMode, consumeRequestedMode)
onBeforeUnmount(() => {
    observer?.disconnect()
    unsubscribeLive?.()
    unsubscribeMidiRequest?.()
    session.dispose()
    recording.dispose()
    clearTimeout(hideTimer)
    window.removeEventListener('keydown', keydown, true)
})
</script>

<style scoped>
.piano-window { width: 100vw; height: 100vh; position: relative; overflow: hidden; background: #242424; color: #eeeeee; border-radius: 7px; font-family: 'Segoe UI', 'Microsoft YaHei UI', 'Microsoft YaHei', sans-serif; user-select: none; }
.waterfall-area { position: absolute; inset: 0 0 var(--keyboard-height); overflow: hidden; }
.midi-waterfall-layer { position: absolute; left: 0; right: 0; bottom: 0; }
.toolbar { position: absolute; top: 0; left: 0; right: 0; height: 62px; display: flex; align-items: center; padding: 0 16px; z-index: 20; background: #242424b8; border-bottom: 1px solid #ffffff0c; }
.toolbar-left, .toolbar-right { display: flex; gap: 10px; align-items: center; flex-shrink: 0; }
.toolbar-left { gap: 12px; }
.performance-mode { width: 136px; --wails-draggable: no-drag; }
:where(button:not(.n-button)) { font: inherit; cursor: pointer; color: inherit; border: 0; background: transparent; --wails-draggable: no-drag; }
button:disabled { cursor: wait; opacity: .55; }
.tool-button { display: flex; align-items: center; justify-content: center; gap: 9px; height: 35px; padding: 0 12px; font-size: 13px; border: 1px solid #ffffff1d; border-radius: 7px; background: #ffffff03; transition: background 160ms, border-color 160ms; }
.tool-button:hover { background: #ffffff0c; border-color: #ffffff35; }
.app-name { font-size: 13px; font-weight: 500; color: #d8d8d8; letter-spacing: .02em; }
.drag-region { height: 100%; flex: 1; display: flex; align-items: center; justify-content: center; min-width: 36px; --wails-draggable: drag; }
.drag-region span { pointer-events: none; color: #a0a0a0; font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 100%; padding: 0 18px; }
.display-switch { --wails-draggable: no-drag; }
.midi-workspace-controls { position: absolute; top: 62px; left: 0; right: 0; z-index: 19; background: #242424; border-bottom: 1px solid #ffffff16; padding: 0 16px 8px; }
.midi-options-row { display: flex; align-items: center; gap: 18px; }
.library-toggle { flex-shrink: 0; }
.midi-options-row :deep(.midi-practice) { flex: 1; min-width: 0; border: 0; border-radius: 0; padding: 10px 0; background: transparent; }
.midi-workspace-controls :deep(.midi-transport) { width: 100%; padding: 5px 0 0; border: 0; border-radius: 0; box-shadow: none; background: transparent; }
.midi-workspace-controls :deep(.transport-main) { margin-bottom: 8px; }
.midi-workspace-controls :deep(.range-line) { margin-top: 6px; }
.library-panel { left: 16px; top: 76px; max-height: calc(var(--waterfall-height) - 92px); }
.toolbar-separator { width: 1px; height: 18px; margin: 0 2px; background: #ffffff12; }
.window-button { display: grid; place-items: center; width: 27px; height: 30px; border-radius: 5px; color: #b5b5b5; }
.window-button:hover { background: #ffffff10; color: white; }
.close-window:hover { background: #a64c43; }
.keyboard-zone { position: absolute; bottom: 0; left: 0; right: 0; }
.strike-line { position: absolute; z-index: 5; left: 0; right: 0; top: -2px; height: 2px; background: #b0b0b080; box-shadow: 0 0 8px #dddddd1a; pointer-events: none; }
.floating-panel { position: absolute; z-index: 30; --wails-draggable: no-drag; }
.main-menu { width: 350px; max-width: calc(100vw - 48px); top: var(--menu-top); max-height: calc(var(--waterfall-height) - var(--menu-top) - 16px); overflow: auto; left: 50%; transform: translateX(-50%); padding: 31px 28px 18px; background: #2b2b2bf5; border: 1px solid #ffffff1b; border-radius: 16px; box-shadow: 0 24px 90px #0004; text-align: center; backdrop-filter: blur(24px); }
.menu-close { position: absolute; top: 12px; right: 12px; display: grid; place-items: center; height: 27px; width: 27px; border-radius: 5px; color: #999999; }
.menu-close:hover { color: white; background: #ffffff0b; }
.menu-mark { color: var(--ui-accent); margin-bottom: 16px; }
h1 { font-size: 24px; font-weight: 500; line-height: 1.3; letter-spacing: -.025em; }
.menu-description { font-size: 12px; color: #a8a8a8; margin-top: 9px; line-height: 1.6; }
.start-playing { margin: 25px 0 14px; padding: 14px 16px; display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; color: var(--ui-accent-text); background: var(--ui-accent); border: 1px solid var(--ui-accent-ring); border-radius: 7px; font-size: 14px; font-weight: 600; transition: background 160ms; }
.start-playing > :last-child { margin-left: auto; }
.start-playing > :first-child { margin-right: auto; }
.start-playing:hover { background: var(--ui-accent-hover); }
.menu-links { display: flex; flex-direction: column; }
.menu-links button { display: flex; align-items: center; gap: 12px; padding: 13px 11px; border-radius: 6px; color: #c7c7c7; font-size: 13px; text-align: left; }
.menu-links button span { flex: 1; }
.menu-links button > :last-child { color: #888888; }
.menu-links button:hover { background: #ffffff07; color: #eeeeee; }
.menu-footer { border-top: 1px solid #ffffff10; margin-top: 14px; padding-top: 17px; display: flex; align-items: center; justify-content: space-between; gap: 8px; font-size: 10px; color: #a3a3a3; }
.input-status { display: flex; align-items: center; gap: 6px; }
.input-status i { width: 5px; height: 5px; border-radius: 50%; background: #858585; }
.input-status i.connected { background: var(--ui-accent); box-shadow: 0 0 5px var(--ui-accent-ring); }
.menu-footer button { display: flex; gap: 5px; align-items: center; font-size: 11px; color: #a9a9a9; }
.menu-footer button:hover { color: #f1c0b8; }
.menu-hint { font-size: 10px; color: #888888; margin-top: 20px; }
.settings-panel { top: 76px; right: 20px; max-height: calc(var(--waterfall-height) - 92px); }
.chord-status { position: absolute; left: 22px; bottom: calc(var(--keyboard-height) + 18px); z-index: 8; pointer-events: none; }
.chord-status :deep(.chord), .chord-status :deep(.note) { position: static; padding: 0; background: transparent; border: 0; box-shadow: none; backdrop-filter: none; color: #dddddd; }
.chord-status :deep(.left) { font-size: 24px !important; font-weight: 400; }
.chord-status :deep(.right) { display: none !important; }
.chord-status :deep(.bottom) { color: #969696; font-size: 10px !important; margin-top: 3px; }
.pedal-status { position: absolute; right: 24px; bottom: calc(var(--keyboard-height) + 22px); display: flex; gap: 16px; z-index: 8; pointer-events: none; }
.pedal-status span { display: flex; align-items: center; gap: 5px; font-size: 10px; color: #858585; }
.pedal-status i { height: 4px; width: 4px; border-radius: 50%; background: #606060; }
.pedal-status .active { color: #dddddd; }
.pedal-status .active i { background: var(--pedal-color); box-shadow: 0 0 7px var(--pedal-color); }
.is-simple { background: transparent; }
.is-simple .waterfall-area { visibility: hidden; }
.is-simple .toolbar { height: 45px; padding: 0 10px; background: #242424e8; opacity: 0; pointer-events: none; transform: translateY(-5px); transition: opacity 160ms, transform 160ms; }
.is-simple.show-tools .toolbar, .is-simple .toolbar:focus-within { opacity: 1; pointer-events: auto; transform: none; }
.is-simple .app-name { display: none; }
.is-simple .chord-status.compact { bottom: 18px; left: 18px; background: #f6f6f6df; border: 1px solid #0001; padding: 6px 10px; border-radius: 5px; }
.is-simple .chord-status :deep(.chord), .is-simple .chord-status :deep(.note) { color: #333333; }
.is-simple .chord-status :deep(.bottom) { color: #666666; }
.is-simple .pedal-status.compact { right: 18px; bottom: 18px; padding: 9px 12px; background: #f6f6f6df; border: 1px solid #0001; border-radius: 5px; }
.is-simple .pedal-status .active { color: #333333; }
.panel-enter-active, .panel-leave-active { transition: opacity 160ms; }
.panel-enter-from, .panel-leave-to { opacity: 0; }
button:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
@media (max-height: 650px) { .main-menu { padding-top: 21px; padding-bottom: 14px; } .menu-mark { margin-bottom: 10px; } .menu-description { margin-top: 5px; } .start-playing { margin-top: 18px; } .menu-hint { margin-top: 12px; } }
@media (max-width: 1000px) { .app-name { display: none; } .toolbar { padding: 0 12px; } .toolbar-right { gap: 7px; } }
@media (prefers-reduced-motion: reduce) { *, .is-simple .toolbar { transition: none !important; } }
</style>
