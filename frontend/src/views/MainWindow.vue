<template>
    <main ref="containerRef" class="piano-window" :class="{ 'is-mac': isMac, 'is-fullscreen': isFullscreen, 'is-simple': isSimple, 'is-midi': isMidi, 'resizing-keyboard': keyboardResizing, 'show-tools': toolbarVisible || panel }"
        :style="windowStyle" @pointerdown="dismissOutside" @mouseenter="showToolbar" @mouseleave="hideToolbar">
        <div class="waterfall-area" :aria-hidden="isSimple || !!workspaceView" :inert="!!workspaceView">
            <WaterfallCanvas v-show="!isMidi" :layout="layout" :width="size.width" :height="waterfallHeight"
                :active="!isSimple && !isMidi && !workspaceView" :colors="store.config.colors" />
            <div v-show="isMidi" class="midi-waterfall-layer" :style="{top: midiCanvasTop + 'px'}">
                <MidiWaterfallCanvas :layout="layout" :width="size.width" :height="midiCanvasHeight"
                    :active="!isSimple && isMidi && !workspaceView" :colors="store.config.colors" :timeline="session.timeline" :player-state="session.state" />
            </div>
        </div>

        <header ref="toolbarRef" class="toolbar" :inert="!!workspaceView" :aria-hidden="!!workspaceView">
            <div class="toolbar-left" data-piano-control>
                <button class="tool-button" type="button" :aria-expanded="panel === 'menu'" aria-label="打开主菜单"
                    @pointerdown.prevent @click="openPanel('menu')"><PhList :size="20" /><span>菜单</span></button>
                <NRadioGroup class="performance-mode" :value="store.mainMode" size="small"
                    :disabled="modeSwitching || session.busy" data-preserve-piano-holds
                    aria-label="演奏模式" @update:value="changeMode">
                    <NRadioButton value="free">自由</NRadioButton>
                    <NRadioButton value="play">播放</NRadioButton>
                    <NRadioButton value="follow">练习</NRadioButton>
                </NRadioGroup>
            </div>
            <div v-show="isMidi && !isSimple" class="midi-header-controls" data-piano-control>
                <NButton class="library-toggle" :type="panel === 'library' ? 'primary' : 'default'" size="small" :aria-expanded="panel === 'library'" @click="openPanel('library')">
                    <template #icon><PhMusicNotes :size="18" /></template>曲库
                </NButton>
                <MidiPracticeControls :session="session" :panel="panel" @open-speed="closePanel" @speed-visibility="speedPanelOpen = $event" />
            </div>
            <div class="drag-region" @dblclick="toggleMaximize"><span v-if="!isSimple">{{ isMidi ? session.selectedSong?.name || '选择一首 MIDI' : APP_NAME }}</span></div>
            <div class="toolbar-right" data-piano-control>
                <NRadioGroup class="display-switch" data-preserve-piano-holds :value="store.mainDisplayMode" :disabled="switching" size="small" aria-label="显示方式" @update:value="setDisplayMode">
                    <NRadioButton value="simple" :disabled="isMidi || modeSwitching" :title="isMidi ? '简单模式仅支持自由演奏' : '简单键盘'">简单</NRadioButton><NRadioButton value="waterfall">瀑布</NRadioButton>
                </NRadioGroup>
                <button class="tool-button settings-toggle" type="button" :aria-expanded="panel === 'settings'" title="设置"
                    @pointerdown.prevent @click="openPanel('settings')"><PhGearSix :size="20" /><span>设置</span></button>
                <button class="tool-button" type="button" title="MIDI 工坊" :aria-expanded="!!workspaceView" @pointerdown.prevent @click="openWorkspace()"><PhUsersThree :size="20"/><span>工坊</span></button>
                <template v-if="!isMac">
                    <span class="toolbar-separator"></span>
                    <button class="window-button" type="button" aria-label="最小化" title="最小化" @pointerdown.prevent @click="minimize"><PhMinus :size="16" /></button>
                    <button v-if="!isSimple" class="window-button" type="button" aria-label="最大化或还原" title="最大化 / 还原"
                        @pointerdown.prevent @click="toggleMaximize"><PhArrowsOutSimple :size="15" /></button>
                    <button class="window-button close-window" type="button" aria-label="退出软件" title="退出" @pointerdown.prevent @click="requestQuit"><PhX :size="17" /></button>
                </template>
            </div>
        </header>

        <RecordingControls v-if="!isMidi && !isSimple && !workspaceView" :recording="recording" />

        <div v-show="isMidi && !isSimple" ref="midiControlsRef" :inert="!!workspaceView" :aria-hidden="!!workspaceView" class="midi-workspace-controls" :style="{top: toolbarHeight + 'px'}" data-piano-control>
            <MidiTransport :session="session" />
        </div>

        <Transition name="panel">
            <section v-if="panel === 'menu' && !isSimple" class="main-menu floating-panel" data-piano-control aria-label="主菜单">
                <button class="menu-close" type="button" aria-label="收起菜单" @click="closePanel"><PhX :size="17" /></button>
                <img class="menu-mark" src="/appicon.png" :alt="APP_NAME + ' Logo'" width="48" height="48" />
                <h1>{{ APP_NAME }}</h1>
                <button class="start-playing" type="button" @click="startFreePlay"><PhPlay :size="17" weight="fill" />{{ isMidi ? '自由演奏' : '开始弹奏' }}<PhArrowRight :size="18" /></button>
                <div class="menu-links">
                    <button type="button" @click="openPanel('library')"><PhMusicNotes :size="20" /><span>曲库</span><PhArrowRight :size="16" /></button>
                    <button type="button" @click="changeMode('play', true)"><PhPlay :size="20" /><span>MIDI 播放</span><PhArrowRight :size="16" /></button>
                    <button type="button" @click="changeMode('follow', true)"><PhPianoKeys :size="20" /><span>跟弹练习</span><PhArrowRight :size="16" /></button>
                    <button type="button" @click="openWorkspace()"><PhUsersThree :size="20"/><span>MIDI 工坊</span><PhArrowRight :size="16"/></button>
                    <button type="button" @click="openWorkspace('practice')"><PhChartBar :size="20"/><span>我的练习</span><PhArrowRight :size="16"/></button>
                    <button type="button" @click="closePanel();accountOpen=true"><PhUserCircle :size="20"/><span>{{community.user?'我的账号':'账号登录'}}</span><PhArrowRight :size="16"/></button>
                    <button type="button" @click="openFullSettings"><PhGearSix :size="20" /><span>完整设置</span><PhArrowUpRight :size="16" /></button>
                </div>
                <footer class="menu-footer">
                    <span class="input-status"><i :class="{ connected: hasMidiInput }"></i>{{ hasMidiInput ? 'MIDI 键盘已连接' : '电脑键盘 / 鼠标演奏' }}</span>
                    <button type="button" @click="requestQuit"><PhPower :size="15" />退出</button>
                </footer>
            </section>
        </Transition>

        <NModal :show="panel === 'library' && !isSimple" :auto-focus="true" :close-on-esc="true"
            @update:show="show => { if (!show) closePanel() }">
            <MidiLibraryPanel :session="session" :style="uiTheme.cssVariables" @selected="selectLibrarySong" @close="closePanel" />
        </NModal>

        <KeepAlive><CommunityWorkspace v-if="workspaceView" :client="community" :session="session" :theme-style="uiTheme.cssVariables" :initial-view="workspaceView" logo-url="/appicon.png" :native-titlebar="isMac" @piano="workspaceView=''" @account="accountOpen=true" @settings="openFullSettings" @toggle-playback="toggleWorkshopPlayback">
            <template v-if="!isMac" #window-controls><button class="window-button" aria-label="最小化" @click="minimize"><PhMinus :size="16"/></button><button class="window-button" aria-label="最大化或还原" @click="toggleMaximize"><PhArrowsOutSimple :size="15"/></button><button class="window-button close-window" aria-label="退出软件" @click="requestQuit"><PhX :size="17"/></button></template>
        </CommunityWorkspace></KeepAlive>
        <NModal v-model:show="accountOpen" :auto-focus="true" :close-on-esc="true"><AccountDialog :client="community" :theme-style="uiTheme.cssVariables" @close="accountOpen=false"/></NModal>
        <Transition name="panel">
            <QuickPianoSettings v-if="panel === 'settings' && !isSimple" class="floating-panel settings-panel"
                @close="closePanel" @open-full="openFullSettings" />
        </Transition>

        <div class="keyboard-zone" :inert="!!workspaceView" :aria-hidden="!!workspaceView" :style="{ height: keyboardHeight + 'px' }">
            <div v-if="!isSimple" class="strike-line"></div>
            <div v-if="!isSimple" class="keyboard-resize-handle" data-piano-control data-preserve-piano-holds
                role="separator" tabindex="0" aria-label="调整琴键高度" aria-orientation="horizontal"
                :aria-valuemin="120" :aria-valuemax="Math.round(maxKeyboardHeight)" :aria-valuenow="keyboardHeight"
                :aria-valuetext="keyboardHeight + ' 像素'" title="拖动调整琴键高度 · 双击恢复自动高度"
                @pointerdown.stop.prevent="beginKeyboardResize" @pointermove="resizeKeyboard"
                @pointerup="finishKeyboardResize" @pointercancel="cancelKeyboardResize"
                @lostpointercapture="cancelKeyboardResize" @dblclick.stop="resetKeyboardHeight" @keydown="resizeKeyboardByKey">
                <span></span>
            </div>
            <ClassicKeyboard v-if="store.keyboardLoaded" :layout="layout" :show-hints="!session.state.scrubbing" />
        </div>

        <Chord v-if="store.loaded" class="chord-status" :compact="isSimple" />
        <div v-if="store.loaded && store.config.showPedal" class="pedal-status" :class="{ compact: isSimple }">
            <span v-for="pedal in pedalIndicators" :key="pedal.key" :class="{ active: pedal.down }" :style="{ '--pedal-color': pedal.color }"><i></i>{{ pedal.label }}</span>
        </div>
    </main>
</template>

<script setup>
import {computed, inject, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {Events, System, Window} from '@wailsio/runtime'
import {NButton, NModal, NRadioButton, NRadioGroup} from 'naive-ui'
import {PhList, PhMinus, PhArrowsOutSimple, PhX, PhPianoKeys, PhPlay,
    PhArrowRight, PhArrowUpRight, PhMusicNotes, PhGearSix, PhPower, PhChartBar, PhUserCircle, PhUsersThree} from '@phosphor-icons/vue'
import ClassicKeyboard from '../components/ClassicKeyboard.vue'
import Chord from '../components/Chord.vue'
import WaterfallCanvas from '../components/WaterfallCanvas.vue'
import QuickPianoSettings from '../components/QuickPianoSettings.vue'
import MidiLibraryPanel from '../components/MidiLibraryPanel.vue'
import MidiPracticeControls from '../components/MidiPracticeControls.vue'
import MidiTransport from '../components/MidiTransport.vue'
import MidiWaterfallCanvas from '../components/MidiWaterfallCanvas.vue'
import RecordingControls from '../components/RecordingControls.vue'
import CommunityWorkspace from '../../../shared/community/CommunityWorkspace.vue'
import AccountDialog from '../../../shared/community/AccountDialog.vue'
import {useDesktopCommunity} from '../services/community'
import {useCommunityPractice} from '../../../shared/community/usePractice'
import {createKeyboardLayout} from '../services/keyboardLayout'
import {subscribeLivePerformance} from '../services/livePerformance'
import {useMidiSession} from '../services/midiSession'
import {useRecordingSession} from '../services/recordingSession'
import {getPianoUiTheme} from '../services/uiTheme'
import {isPianoControlTarget} from '../services/computerKeyboard'
import {APP_NAME} from '../services/appIdentity'

const store = inject('store')
const Keyboard = inject('Keyboard')
const saveConfig = inject('changeConfig')
const route = useRoute()
const router = useRouter()
const isMac = System.IsMac()
const isFullscreen = ref(false)
const session = useMidiSession({store, Keyboard})
const community=useDesktopCommunity({Keyboard,session,changeMode})
const pageHidden=ref(document.hidden)
const communityVisibility=()=>pageHidden.value=document.hidden
document.addEventListener('visibilitychange',communityVisibility)
onBeforeUnmount(()=>document.removeEventListener('visibilitychange',communityVisibility))
useCommunityPractice(community,()=>({midiId:session.selectedSong?.communityMidiId,mode:session.mode,status:session.state.status,hidden:pageHidden.value}),message=>window.$message?.warning(message))
onMounted(()=>community.restoreSession().catch(error=>window.$message?.error(error.message||String(error))))
const recording = useRecordingSession({Keyboard,
    canStart: () => store.mainDisplayMode !== 'simple' && store.mainMode === 'free' && !switching.value && !modeSwitching.value,
    getPlayerState: () => store.midiPlayerState, onSaved: async song => {
    await session.refreshLibrary()
    if (song?.id) await session.selectSong(song.id)
}})
const uiTheme = computed(() => getPianoUiTheme(store.config.colors))
const containerRef = ref(null)
const toolbarRef = ref(null)
const toolbarHeight = ref(54)
const midiControlsRef = ref(null)
const midiControlsHeight = ref(70)
const panel = ref('menu'), workspaceView = ref(''), accountOpen = ref(false)
watch(() => !!workspaceView.value || accountOpen.value, value => {store.onlineWorkspaceOpen = value})
const speedPanelOpen = ref(false)
const switching = ref(false)
const modeSwitching = ref(false)
const toolbarVisible = ref(true)
const size = reactive({width: window.innerWidth, height: window.innerHeight})
const isSimple = computed(() => store.mainDisplayMode === 'simple')
const isMidi = computed(() => store.mainMode !== 'free')
const keyboardHeightDraft = ref(null)
const keyboardResizing = ref(false)
let keyboardResizePointer = null
const maxKeyboardHeight = computed(() => Math.max(120, Math.min(size.height * .5,
    size.height - (isMidi.value ? midiCanvasTop.value : toolbarHeight.value) - 160)))
const keyboardHeight = computed(() => {
    if (isSimple.value) return size.height
    if (keyboardHeightDraft.value !== null) return Math.round(keyboardHeightDraft.value)
    if (store.config.keyboardHeightRatio > 0) return Math.round(Math.max(120,
        Math.min(maxKeyboardHeight.value, size.height * store.config.keyboardHeightRatio)))
    const whiteCount = visibleKeys.value.filter(key => key.color !== 'black').length || 52
    const naturalHeight = Math.max(size.height * .19, size.width / whiteCount * 4, 120)
    return Math.round(Math.min(size.height * .26, naturalHeight))
})
const waterfallHeight = computed(() => Math.max(0, size.height - keyboardHeight.value))
const midiCanvasTop = computed(() => toolbarHeight.value + midiControlsHeight.value)
const midiCanvasHeight = computed(() => Math.max(0, waterfallHeight.value - midiCanvasTop.value))
const windowStyle = computed(() => ({
    ...uiTheme.value.cssVariables,
    '--keyboard-height': keyboardHeight.value + 'px',
    '--waterfall-height': waterfallHeight.value + 'px',
    '--toolbar-height': toolbarHeight.value + 'px',
    '--menu-top': Math.max(68, (waterfallHeight.value - 490) / 2) + 'px',
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
const unsubscribeWindowEvents = []

function beginKeyboardResize(event) {
    if (event.button !== 0 || isSimple.value) return
    keyboardResizePointer = {id: event.pointerId, y: event.clientY, height: keyboardHeight.value}
    keyboardHeightDraft.value = keyboardHeight.value
    keyboardResizing.value = true
    event.currentTarget.setPointerCapture(event.pointerId)
}
function resizeKeyboard(event) {
    if (keyboardResizePointer?.id !== event.pointerId) return
    keyboardHeightDraft.value = Math.max(120, Math.min(maxKeyboardHeight.value,
        keyboardResizePointer.height + keyboardResizePointer.y - event.clientY))
}
function finishKeyboardResize(event) {
    if (keyboardResizePointer?.id !== event.pointerId) return
    resizeKeyboard(event)
    store.config.keyboardHeightRatio = keyboardHeightDraft.value / size.height
    keyboardResizePointer = null
    keyboardResizing.value = false
    keyboardHeightDraft.value = null
    saveConfig()
}
function cancelKeyboardResize() {
    keyboardResizePointer = null
    keyboardResizing.value = false
    keyboardHeightDraft.value = null
}
function resetKeyboardHeight() {
    cancelKeyboardResize()
    store.config.keyboardHeightRatio = 0
    saveConfig()
}
function resizeKeyboardByKey(event) {
    if (!['ArrowUp', 'ArrowDown', 'Home', 'End', 'Enter'].includes(event.key)) return
    event.preventDefault()
    event.stopPropagation()
    if (event.key === 'Home' || event.key === 'Enter') { resetKeyboardHeight(); return }
    const next = event.key === 'End' ? maxKeyboardHeight.value
        : keyboardHeight.value + (event.key === 'ArrowUp' ? 1 : -1) * (event.shiftKey ? 24 : 8)
    store.config.keyboardHeightRatio = Math.max(120, Math.min(maxKeyboardHeight.value, next)) / size.height
    saveConfig()
}

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
async function openWorkspace(view = 'workshop') {
    if (isSimple.value && !await setDisplayMode('waterfall')) return
    if (recording.state.status === 'recording' && !await recording.stop()) return
    closePanel(); workspaceView.value = view
}
async function toggleWorkshopPlayback() {
    if (session.mode === 'free' && !await changeMode('play')) return
    await session.togglePlayback()
}
async function openPanel(name) {
    if (isSimple.value && !await setDisplayMode('waterfall')) return
    panel.value = panel.value === name ? null : name
    releaseControlFocus()
}
async function setDisplayMode(mode) {
    if (switching.value) return false
    if (mode === 'simple' && (isMidi.value || modeSwitching.value)) return false
    if (store.mainDisplayMode === mode) { releaseControlFocus(); return true }
    switching.value = true
    try {
        if (mode === 'simple') {
            const wasRecording = recording.state.status === 'recording'
            if (recording.previewing && !await recording.stopPreview()) return false
            if ((wasRecording || (recording.state.status === 'draft' && !recording.state.backupSaved))
                && !await recording.stop()) return false
            if (wasRecording) window.$message?.info('录制已停止，草稿已保留')
        }
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
async function changeMode(mode, chooseSong = false) {
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
        panel.value = mode !== 'free' && (chooseSong || !session.selectedId) ? 'library' : null
        releaseControlFocus()
        return true
    } finally { modeSwitching.value = false }
}
async function selectLibrarySong() {
    if (!isMidi.value && !await changeMode('play')) return
    closePanel()
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
    if (workspaceView.value || accountOpen.value) return
    if ((event.code === 'Space' || event.key === ' ') && isMidi.value && !isSimple.value) {
        if (event.defaultPrevented || event.ctrlKey || event.altKey || event.metaKey
            || panel.value || speedPanelOpen.value || isPianoControlTarget(event.target)) return
        event.preventDefault()
        if (!event.repeat && !session.busy && !session.loading && session.selectedId) void session.togglePlayback()
        return
    }
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
    if (isMac) {
        let fullscreenChanged = false
        const setFullscreen = fullscreen => {
            fullscreenChanged = true
            isFullscreen.value = fullscreen
        }
        unsubscribeWindowEvents.push(
            Events.On(Events.Types.Common.WindowFullscreen, () => setFullscreen(true)),
            // Clear the traffic-light area before the native exit animation starts.
            Events.On(Events.Types.Mac.WindowWillExitFullScreen, () => setFullscreen(false)),
            Events.On(Events.Types.Common.WindowUnFullscreen, () => setFullscreen(false)),
        )
        // Initialise restored/reloaded windows without overwriting a newer native event.
        void Window.IsFullscreen().then(fullscreen => {
            if (!fullscreenChanged) isFullscreen.value = fullscreen
        }).catch(error => console.error('无法读取全屏状态', error))
    }
    observer = new ResizeObserver(entries => {
        for (const entry of entries) {
            if (entry.target === containerRef.value) {
                size.width = entry.contentRect.width
                size.height = entry.contentRect.height
            } else if (entry.target === toolbarRef.value) {
                toolbarHeight.value = entry.target.getBoundingClientRect().height
            } else if (entry.contentRect.height > 0) {
                midiControlsHeight.value = entry.borderBoxSize?.[0]?.blockSize || entry.target.getBoundingClientRect().height
            }
        }
    })
    observer.observe(containerRef.value)
    observer.observe(toolbarRef.value)
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
watch(isSimple, cancelKeyboardResize)
watch(() => [isMidi.value, isSimple.value, switching.value], ([midi, simple, changing]) => {
    if (midi && simple && !changing) void setDisplayMode('waterfall')
})
onBeforeUnmount(() => {
    store.onlineWorkspaceOpen = false
    observer?.disconnect()
    unsubscribeLive?.()
    unsubscribeMidiRequest?.()
    unsubscribeWindowEvents.forEach(unsubscribe => unsubscribe())
    session.dispose()
    recording.dispose()
    clearTimeout(hideTimer)
    window.removeEventListener('keydown', keydown, true)
})
</script>

<style scoped src="../../../shared/piano/window.css"></style>
