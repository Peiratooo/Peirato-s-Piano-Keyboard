<template>
<NConfigProvider :theme="darkTheme" :theme-overrides="uiTheme.naiveOverrides" :locale="zhCN" :date-locale="dateZhCN" :style="{...uiTheme.cssVariables,...safeStyles}" class="web-piano"><NMessageProvider><NNotificationProvider :max="3" placement="top-right"><FeedbackBridge/>

    <main ref="containerRef" class="piano-window is-web" :class="{ 'is-fullscreen': isFullscreen, 'is-simple': isSimple, 'is-midi': isMidi, 'resizing-keyboard': keyboardResizing, 'show-tools': toolbarVisible || panel }"
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

            </div>
        </header>

        <RecordingControls v-if="!isMidi && !isSimple && !workspaceView" :recording="recording" />

        <div v-show="isMidi && !isSimple" ref="midiControlsRef" :inert="!!workspaceView" :aria-hidden="!!workspaceView" class="midi-workspace-controls" :style="{top: toolbarHeight + 'px'}" data-piano-control>
            <MidiTransport :session="session" />
        </div>

        <Transition name="panel">
            <section v-if="panel === 'menu' && !isSimple" class="main-menu floating-panel" data-piano-control aria-label="主菜单">
                <button class="menu-close" type="button" aria-label="收起菜单" @click="closePanel"><PhX :size="17" /></button>
                <img class="menu-mark" :src="baseUrl + 'appicon.png'" :alt="APP_NAME + ' Logo'" width="48" height="48" />
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

        <KeepAlive><CommunityWorkspace v-if="workspaceView" :client="community" :session="session" :theme-style="uiTheme.cssVariables" :initial-view="workspaceView" :logo-url="baseUrl+'appicon.png'" @piano="workspaceView=''" @account="accountOpen=true" @settings="openFullSettings" @toggle-playback="toggleWorkshopPlayback">
            <template #rooms><RoomsPanel :community="community" :piano="piano" @toast="toast" @close="workspaceView=''"/></template>
            <template #admin><AdminPanel :client="community"/></template>
        </CommunityWorkspace></KeepAlive>
        <NModal v-model:show="accountOpen" :auto-focus="true" :close-on-esc="true"><AccountDialog :client="community" :theme-style="uiTheme.cssVariables" :desktop-code="desktopCode" @close="accountOpen=false"/></NModal>
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
        <NModal v-model:show="fullSettings" :auto-focus="true" :close-on-esc="true">
            <BrowserSettings :store="store" :piano="piano" :midi-status="presentation.midiDeviceStatus.value" :sound-font-name="presentation.soundFontName.value"
                @close="fullSettings=false" @connect-midi="connectMidi" @import-sf2="presentation.importSF2" @share="share" @immerse="enterImmersive"/>
        </NModal>
        <div v-if="community.inRoom" class="live-room-banner" :style="{top:(isMidi?midiCanvasTop:toolbarHeight)+12+'px'}" data-piano-control>
            <strong>{{community.room?.title}}</strong><span>{{community.isPerformer?'正在演奏':`正在围观 · ${community.audience} 人`}}</span>
            <span v-if="community.room?.id==='lobby'&&community.isPerformer">{{Math.floor(community.remainingSeconds/60)}}:{{String(community.remainingSeconds%60).padStart(2,'0')}}</span>
            <button v-if="community.room?.id==='lobby'&&community.isPerformer" @click="community.yieldSeat()">主动下场</button>
            <button @click="community.leave()">退出琴房</button>
        </div>
    </main>

</NNotificationProvider></NMessageProvider></NConfigProvider>
</template>

<script setup>
import {computed, onBeforeUnmount, onMounted, reactive, ref} from 'vue'
import {darkTheme, dateZhCN, NButton, NConfigProvider, NMessageProvider, NModal, NNotificationProvider, NRadioButton, NRadioGroup, zhCN} from 'naive-ui'
import {PhList, PhX, PhPlay, PhArrowRight, PhArrowUpRight, PhMusicNotes, PhGearSix, PhPower, PhChartBar, PhUserCircle, PhUsersThree, PhPianoKeys} from '@phosphor-icons/vue'
import FeedbackBridge from '../../frontend/src/components/FeedbackBridge.vue'
import ClassicKeyboard from '../../frontend/src/components/ClassicKeyboard.vue'
import Chord from '../../frontend/src/components/Chord.vue'
import WaterfallCanvas from '../../frontend/src/components/WaterfallCanvas.vue'
import MidiWaterfallCanvas from '../../frontend/src/components/MidiWaterfallCanvas.vue'
import MidiPracticeControls from '../../frontend/src/components/MidiPracticeControls.vue'
import MidiTransport from '../../frontend/src/components/MidiTransport.vue'
import MidiLibraryPanel from '../../frontend/src/components/MidiLibraryPanel.vue'
import RecordingControls from '../../frontend/src/components/RecordingControls.vue'
import QuickPianoSettings from '../../frontend/src/components/QuickPianoSettings.vue'
import CommunityWorkspace from '../../shared/community/CommunityWorkspace.vue'
import AccountDialog from '../../shared/community/AccountDialog.vue'
import {createKeyboardLayout} from '../../frontend/src/services/keyboardLayout'
import {createComputerKeyboard, isPianoControlTarget} from '../../frontend/src/services/computerKeyboard'
import {APP_NAME} from '../../frontend/src/services/appIdentity'
import RoomsPanel from './components/RoomsPanel.vue'
import BrowserSettings from './components/BrowserSettings.vue'
import AdminPanel from '../../shared/community/AdminPanel.vue'
import {usePiano} from './composables/usePiano'
import {usePianoPresentation} from './composables/usePianoPresentation'
import {useCommunity} from './services/community'
import {useCommunityPractice} from '../../shared/community/usePractice'
import {immerseToy, shareToy, watchToyContainer} from './services/toy'

const toast = message => window.$message?.info(String(message), {duration: 3500})
const piano = reactive(usePiano(toast))
const community = reactive(useCommunity(piano, toast))
const presentation = usePianoPresentation(piano, community, toast)
const {store, session, recording, uiTheme, saveConfig} = presentation
const baseUrl = import.meta.env.BASE_URL
const desktopCode = new URLSearchParams(location.search).get('desktopCode') || ''
const containerRef = ref(null), toolbarRef = ref(null), midiControlsRef = ref(null)
const toolbarHeight = ref(54), midiControlsHeight = ref(64)
const workspaceView = ref(''), accountOpen = ref(!!desktopCode)
const panel = ref(desktopCode ? null : 'menu'), fullSettings = ref(false), speedPanelOpen = ref(false)
const modeSwitching = ref(false), switching = ref(false), toolbarVisible = ref(true), keyboardResizing = ref(false)
const isFullscreen = ref(!!document.fullscreenElement)
const size = reactive({width: window.innerWidth, height: window.innerHeight})
const safe = reactive({top: 0, right: 0, bottom: 0, left: 0})
const safeStyles = computed(() => ({paddingTop: safe.top + 'px', paddingRight: safe.right + 'px', paddingBottom: safe.bottom + 'px', paddingLeft: safe.left + 'px'}))
const isSimple = computed(() => piano.displayMode === 'simple'), isMidi = computed(() => piano.mode !== 'free')
const keyboardHeightDraft = ref(null)
const visibleKeys = computed(() => {
    const [start, end] = store.keyboardRange[store.config.keyboardType]
    return store.keyboardConfig.slice(start, end)
})
const maxKeyboardHeight = computed(() => Math.max(120, Math.min(size.height * .5,
    size.height - (isMidi.value ? midiCanvasTop.value : toolbarHeight.value) - 160)))
const keyboardHeight = computed(() => {
    if (isSimple.value) return size.height
    if (keyboardHeightDraft.value !== null) return Math.round(keyboardHeightDraft.value)
    if (store.config.keyboardHeightRatio > 0) return Math.round(Math.max(120,
        Math.min(maxKeyboardHeight.value, size.height * store.config.keyboardHeightRatio)))
    const whiteCount = visibleKeys.value.filter(key => key.color !== 'black').length || 52
    return Math.round(Math.min(size.height * .26, Math.max(size.height * .19, size.width / whiteCount * 4, 120)))
})
const waterfallHeight = computed(() => Math.max(0, size.height - keyboardHeight.value))
const midiCanvasTop = computed(() => toolbarHeight.value + midiControlsHeight.value)
const midiCanvasHeight = computed(() => Math.max(0, waterfallHeight.value - midiCanvasTop.value))
const windowStyle = computed(() => ({...uiTheme.value.cssVariables, '--keyboard-height': keyboardHeight.value + 'px',
    '--waterfall-height': waterfallHeight.value + 'px', '--toolbar-height': toolbarHeight.value + 'px',
    '--menu-top': Math.max(68, (waterfallHeight.value - 490) / 2) + 'px'}))
const layout = computed(() => createKeyboardLayout(visibleKeys.value, size.width))
const hasMidiInput = computed(() => store.devices.selectedInDevice !== -1)
const pedalIndicators = computed(() => ['damperPedal', 'softPedal', 'sostenutoPedal'].map((key, i) => ({key,
    label: ['延音', '柔音', '持音'][i], color: store.config.colors[key].color,
    down: key === 'damperPedal' && presentation.sustainActive.value})))
const pageHidden = ref(document.hidden)
useCommunityPractice(community, () => ({midiId: piano.selected?.serverMidiId, mode: piano.mode, status: piano.state.status, hidden: pageHidden.value || community.inRoom}), toast)

let keyboardResizePointer, resizeObserver, hideTimer, stopContainer, midiAccess
const connectedMidiInputs = new Set()
function closePanel() { panel.value = null }
function openPanel(next) { panel.value = panel.value === next ? null : next; showToolbar() }
async function openWorkspace(view = 'workshop') {
    if (piano.recording.status === 'recording') await recording.stop()
    piano.displayMode = 'waterfall'; closePanel(); workspaceView.value = view
}
async function toggleWorkshopPlayback() {
    if (piano.mode === 'free') piano.changeMode('play')
    await session.togglePlayback()
}
function openFullSettings() { closePanel(); fullSettings.value = true }
function dismissOutside(event) { if (!isPianoControlTarget(event.target)) closePanel() }
function showToolbar() { clearTimeout(hideTimer); toolbarVisible.value = true }
function hideToolbar() { clearTimeout(hideTimer); hideTimer = setTimeout(() => { toolbarVisible.value = false }, 600) }
function startFreePlay() { changeMode('free'); closePanel() }
function setDisplayMode(mode) { piano.displayMode = mode; closePanel(); showToolbar() }
async function changeMode(mode, chooseSong = false) {
    if (community.inRoom) { toast('请先退出琴房再切换模式'); return }
    if (piano.recording.status === 'recording') await recording.stop()
    piano.stopRecordingPreview()
    piano.changeMode(mode)
    panel.value = mode !== 'free' && (chooseSong || !session.selectedId) ? 'library' : null
}
function selectLibrarySong() { if (!isMidi.value) piano.changeMode('play'); closePanel() }
function requestQuit() { piano.stop(); piano.releaseAllUsers(); if (community.inRoom) community.leave(); closePanel() }
function toggleMaximize() { if (document.fullscreenElement) void document.exitFullscreen(); else void document.documentElement.requestFullscreen().catch(() => toast('当前容器不支持全屏')) }
function beginKeyboardResize(event) {
    if (event.button !== 0 || isSimple.value) return
    keyboardResizePointer = {id: event.pointerId, y: event.clientY, height: keyboardHeight.value}
    keyboardHeightDraft.value = keyboardHeight.value; keyboardResizing.value = true
    event.currentTarget.setPointerCapture(event.pointerId)
}
function resizeKeyboard(event) {
    if (keyboardResizePointer?.id !== event.pointerId) return
    keyboardHeightDraft.value = Math.max(120, Math.min(maxKeyboardHeight.value, keyboardResizePointer.height + keyboardResizePointer.y - event.clientY))
}
function cancelKeyboardResize() { keyboardResizePointer = null; keyboardHeightDraft.value = null; keyboardResizing.value = false }
function finishKeyboardResize(event) {
    if (keyboardResizePointer?.id !== event.pointerId) return
    resizeKeyboard(event); store.config.keyboardHeightRatio = keyboardHeightDraft.value / size.height
    cancelKeyboardResize(); saveConfig()
}
function resetKeyboardHeight() { cancelKeyboardResize(); store.config.keyboardHeightRatio = 0; saveConfig() }
function resizeKeyboardByKey(event) {
    if (!['ArrowUp', 'ArrowDown', 'Home'].includes(event.key)) return
    event.preventDefault()
    if (event.key === 'Home') resetKeyboardHeight()
    else { store.config.keyboardHeightRatio = Math.max(120, Math.min(maxKeyboardHeight.value, keyboardHeight.value + (event.key === 'ArrowUp' ? 12 : -12))) / size.height; saveConfig() }
}
const input = createComputerKeyboard({mapping: () => store.activeKeyMapping, enabled: () => !workspaceView.value && !accountOpen.value && !fullSettings.value && (!community.inRoom || community.isPerformer),
    press: note => piano.userNoteOn(note, store.config.velocity, 'keyboard'), release: note => piano.userNoteOff(note, 'keyboard')})
function keydown(event) {
    if (workspaceView.value || accountOpen.value) return
    if (event.defaultPrevented) return
    if (event.key === 'Escape') { closePanel(); fullSettings.value = false; return }
    if (event.code === 'Space' && !event.repeat && !isPianoControlTarget(event.target) && !fullSettings.value) {
        event.preventDefault()
        if (isMidi.value) void piano.togglePlayback()
        else presentation.toggleSustain(true)
        return
    }
    input.keydown(event)
}
function keyup(event) { if (event.code === 'Space' && !isMidi.value) presentation.toggleSustain(false); input.keyup(event) }
const offPiano = piano.subscribe(event => { if (event.source === 'user' && event.type === 'on' && panel.value === 'menu') closePanel() })
function visibilityChange() {
    pageHidden.value = document.hidden; piano.setHidden(document.hidden)
    if (document.hidden) { input.clear(); if (piano.recording.status === 'recording') void presentation.persistDraft() }
}
function pageHide() { input.clear(); void presentation.persistDraft() }
function fullscreenChange() { isFullscreen.value = !!document.fullscreenElement }
async function connectMidi() {
    if (!navigator.requestMIDIAccess) { toast('当前浏览器或 Toy 容器不支持 Web MIDI'); return }
    midiAccess = await navigator.requestMIDIAccess({sysex: false}).catch(error => { toast('MIDI 连接失败：' + error.message); return null })
    if (!midiAccess) return
    const attach = () => {
        const connected = new Set()
        for (const device of midiAccess.inputs.values()) {
            if (device.state === 'disconnected') continue
            connected.add(device.id)
            device.onmidimessage = ({data: [status, note, velocity]}) => {
                if (community.inRoom && !community.isPerformer) return
                const kind = status & 0xf0, source = 'device-' + device.id
                if (kind === 0x90 && velocity) piano.userNoteOn(note, velocity, source)
                else if (kind === 0x80 || kind === 0x90) piano.userNoteOff(note, source)
                else if (kind === 0xb0 && note === 64) presentation.toggleSustain(velocity >= 64)
            }
        }
        for (const id of connectedMidiInputs) if (!connected.has(id)) piano.releaseSource('device-' + id)
        connectedMidiInputs.clear(); connected.forEach(id => connectedMidiInputs.add(id))
        presentation.midiDeviceStatus.value = connected.size ? `已连接 ${connected.size} 台` : '未检测到设备'
        store.devices.selectedInDevice = connected.size ? 0 : -1
    }
    midiAccess.onstatechange = attach; attach()
}
async function enterImmersive() { if (!await immerseToy()) toggleMaximize() }
async function share() {
    const result = await shareToy()
    if (result === 'shared') return
    if (navigator.share) await navigator.share({title: APP_NAME, url: location.href}).catch(() => {})
    else await navigator.clipboard.writeText(location.href).then(() => toast('链接已复制'), () => toast('请使用浏览器分享菜单'))
}
onMounted(async () => {
    resizeObserver = new ResizeObserver(entries => {
        for (const entry of entries) {
            if (entry.target === containerRef.value) { size.width = entry.contentRect.width; size.height = entry.contentRect.height }
            else if (entry.target === toolbarRef.value) toolbarHeight.value = entry.target.getBoundingClientRect().height
            else if (entry.contentRect.height > 0) midiControlsHeight.value = entry.target.getBoundingClientRect().height
        }
    })
    for (const element of [containerRef.value, toolbarRef.value, midiControlsRef.value]) resizeObserver.observe(element)
    window.addEventListener('keydown', keydown)
    window.addEventListener('keyup', keyup)
    window.addEventListener('blur', input.clear)
    window.addEventListener('focusin', input.focusin)
    window.addEventListener('pagehide', pageHide)
    document.addEventListener('visibilitychange', visibilityChange)
    document.addEventListener('fullscreenchange', fullscreenChange)
    stopContainer = watchToyContainer(state => Object.assign(safe, state.safeArea || {}))
    await presentation.initialize().catch(error => toast('本地数据加载失败：' + error.message))
})
onBeforeUnmount(() => {
    resizeObserver?.disconnect(); stopContainer?.(); clearTimeout(hideTimer); offPiano(); input.clear()
    window.removeEventListener('keydown', keydown); window.removeEventListener('keyup', keyup); window.removeEventListener('blur', input.clear)
    window.removeEventListener('focusin', input.focusin); window.removeEventListener('pagehide', pageHide)
    document.removeEventListener('visibilitychange', visibilityChange); document.removeEventListener('fullscreenchange', fullscreenChange)
    if (midiAccess) { midiAccess.onstatechange = null; for (const device of midiAccess.inputs.values()) device.onmidimessage = null }
})
</script>

<style scoped src="../../shared/piano/window.css"></style>
<style scoped>
.piano-window.is-web { width: 100%; height: 100%; border-radius: 0; }
.live-room-banner { position: absolute; z-index: 8; left: 16px; display: flex; align-items: center; gap: 10px; flex-wrap: wrap; max-width: calc(100% - 32px); padding: 10px 12px; background: #292929ed; border: 1px solid #ffffff25; border-radius: 8px; font-size: 12px; }
.live-room-banner button { border: 1px solid #ffffff25; border-radius: 5px; padding: 5px 8px; }
@media (max-width: 600px) {
  .is-midi:not(.is-simple) .toolbar { grid-template-columns: auto minmax(0, 1fr) auto; }
  .toolbar { gap: 6px; padding: 0 8px; }
  .toolbar-left, .toolbar-right { gap: 6px; }
  .tool-button { padding: 0 8px; }
  .tool-button span { display: none; }
  .drag-region { min-width: 0; }
  .drag-region span { display: none; }
  .display-switch :deep(.n-radio-button) { padding: 0 5px !important; }
  .settings-panel { right: 8px; max-width: calc(100% - 16px); }
  .main-menu { max-width: calc(100% - 32px); }
}
</style>
