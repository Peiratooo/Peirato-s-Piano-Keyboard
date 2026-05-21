<template>
    <main class="midi-window">
        <aside class="library">
            <div class="library-head">
                <div class="library-title">
                    <UiIcon name="piano" :size="18" />
                    <div>
                        <strong>MIDI</strong>
                        <span>{{ midiItems.length }} 个文件</span>
                    </div>
                </div>

                <button class="head-action" type="button" title="导入 MIDI" @click="importMidi">
                    <UiIcon name="upload" :size="16" />
                </button>
            </div>

            <div class="midi-list">
                <div
                    v-for="item in midiItems"
                    :key="item.id"
                    class="midi-item"
                    :class="{
                        selected: item.id === selectedId,
                        playing: item.id === playerState.midiId && playerState.status === 'playing',
                    }"
                    @click="selectMidi(item)"
                >
                    <span class="midi-icon"><UiIcon name="music" :size="16" /></span>
                    <span class="midi-name">{{ item.name }}</span>
                    <span class="midi-time">{{ formatDuration(item.durationMs) }}</span>

                    <div class="midi-actions" @click.stop>
                        <button type="button" title="播放" @click="quickPlay(item)">
                            <UiIcon :name="item.id === playerState.midiId && playerState.status === 'playing' ? 'pause' : 'play'" :size="14" />
                        </button>
                        <button class="danger" type="button" title="删除" @click="removeMidiItem(item)">
                            <UiIcon name="trash" :size="14" />
                        </button>
                    </div>
                </div>

                <div v-if="midiItems.length === 0" class="empty-state">
                    <UiIcon name="music" :size="22" />
                    <span>暂无 MIDI 文件</span>
                    <button type="button" @click="importMidi">导入 MIDI</button>
                </div>
            </div>
        </aside>

        <section class="workspace">
            <header class="compact-topbar">
                <div class="track-info">
                    <span class="label">Now Editing</span>
                    <strong>{{ selectedMidi?.name || '选择一首 MIDI' }}</strong>
                </div>

                <div class="option-strip">
                    <div class="segmented">
                        <button :class="{ active: options.mode === 'play' }" type="button" @click="setPlaybackOption('mode', 'play')">
                            <UiIcon name="play" :size="14" />
                            播放
                        </button>
                        <button :class="{ active: options.mode === 'follow' }" type="button" @click="setPlaybackOption('mode', 'follow')">
                            <UiIcon name="keyboard" :size="14" />
                            练习
                        </button>
                    </div>

                    <div class="segmented hand-mode">
                        <button :class="{ active: options.handMode === 'left' }" type="button" @click="setPlaybackOption('handMode', 'left')">左</button>
                        <button :class="{ active: options.handMode === 'right' }" type="button" @click="setPlaybackOption('handMode', 'right')">右</button>
                        <button :class="{ active: options.handMode === 'both' }" type="button" @click="setPlaybackOption('handMode', 'both')">双</button>
                    </div>

                    <label class="speed-control">
                        <UiIcon name="speed" :size="15" />
                        <input
                            type="range"
                            min="0.25"
                            max="3"
                            step="0.05"
                            v-model.number="options.speed"
                            @input="syncPlaybackOptions"
                        >
                        <b>{{ options.speed.toFixed(2) }}x</b>
                    </label>
                </div>
            </header>

            <section class="player-panel">
                <div class="transport-bar">
                    <button
                        class="main-play"
                        :class="{ active: matchingPlayerState.status === 'playing' }"
                        :disabled="!selectedId"
                        type="button"
                        @click="toggleTransport"
                    >
                        <UiIcon :name="matchingPlayerState.status === 'playing' ? 'pause' : 'play'" :size="22" />
                    </button>

                    <div class="transport-meta">
                        <strong>{{ transportLabel }}</strong>
                        <span>{{ formatDuration(displayCurrentMs) }} / {{ formatDuration(durationMs) }}</span>
                    </div>

                    <button class="mini-tool" :disabled="playerState.status === 'idle'" type="button" @click="stopPlayback">
                        <UiIcon name="stop" :size="15" />
                        停止
                    </button>

                    <div class="range-readout">
                        <span>IN {{ formatDuration(displayLeftMs) }}</span>
                        <span>OUT {{ formatDuration(displayRightMs) }}</span>
                    </div>
                </div>

                <div
                    ref="timelineRef"
                    class="timeline"
                    :class="{
                        disabled: !selectedId || durationMs <= 0,
                        dragging: drag.type,
                        'dragging-seek': drag.type === 'seek',
                        'dragging-anchor': drag.type === 'left' || drag.type === 'right',
                    }"
                    @pointerdown="startSeekDrag"
                >
                    <div class="ruler" aria-hidden="true">
                        <span v-for="tick in 17" :key="tick"></span>
                    </div>

                    <div class="rail">
                        <div class="range-before" :style="{ width: leftPercent + '%' }"></div>
                        <div
                            class="range-active"
                            :style="{ left: leftPercent + '%', width: activePercent + '%' }"
                        ></div>
                        <div class="range-after" :style="{ left: rightPercent + '%' }"></div>

                        <button
                            class="anchor left-anchor"
                            :style="{ left: leftPercent + '%' }"
                            :disabled="!selectedId"
                            type="button"
                            aria-label="左锚点"
                            @pointerdown.stop="startAnchorDrag('left', $event)"
                        >
                            <span>IN</span>
                        </button>

                        <button
                            class="anchor right-anchor"
                            :style="{ left: rightPercent + '%' }"
                            :disabled="!selectedId"
                            type="button"
                            aria-label="右锚点"
                            @pointerdown.stop="startAnchorDrag('right', $event)"
                        >
                            <span>OUT</span>
                        </button>

                        <button
                            class="playhead"
                            :style="{ left: currentPercent + '%' }"
                            :disabled="!selectedId"
                            type="button"
                            aria-label="播放头"
                            @pointerdown.stop="startSeekDrag"
                        >
                            <i></i>
                        </button>
                    </div>
                </div>
            </section>
        </section>
    </main>
</template>

<script setup>
import { computed, defineComponent, h, inject, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'

const DEFAULT_WINDOW_MS = 80
const MIN_RANGE_MS = 100
const SEEK_DEBOUNCE_MS = 80
const SEEK_VISUAL_LOCK_MS = 220

const store = inject('store')
const Keyboard = inject('Keyboard')

const ICONS = {
    piano: [
        ['path', { d: 'M4 5h16v14H4z' }],
        ['path', { d: 'M7 5v14M11 5v14M15 5v14' }],
        ['path', { d: 'M8.5 5v7M12.5 5v7M16.5 5v7' }],
    ],
    upload: [
        ['path', { d: 'M12 16V4' }],
        ['path', { d: 'm7 9 5-5 5 5' }],
        ['path', { d: 'M20 16v3a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1v-3' }],
    ],
    music: [
        ['path', { d: 'M9 18V5l11-2v13' }],
        ['circle', { cx: '6', cy: '18', r: '3' }],
        ['circle', { cx: '17', cy: '16', r: '3' }],
    ],
    play: [['polygon', { points: '8 5 19 12 8 19 8 5' }]],
    pause: [['path', { d: 'M8 5v14M16 5v14' }]],
    stop: [['rect', { x: '6', y: '6', width: '12', height: '12', rx: '2' }]],
    trash: [
        ['path', { d: 'M3 6h18' }],
        ['path', { d: 'M8 6V4h8v2' }],
        ['path', { d: 'M19 6l-1 14H6L5 6' }],
        ['path', { d: 'M10 11v5M14 11v5' }],
    ],
    keyboard: [
        ['rect', { x: '3', y: '6', width: '18', height: '12', rx: '3' }],
        ['path', { d: 'M7 10v4M11 10v4M15 10v4M19 10v4' }],
    ],
    speed: [
        ['path', { d: 'M4 14a8 8 0 1 1 16 0' }],
        ['path', { d: 'm12 14 4-4' }],
        ['path', { d: 'M12 20h.01' }],
    ],
}

const UiIcon = defineComponent({
    name: 'UiIcon',
    props: {
        name: { type: String, required: true },
        size: { type: Number, default: 18 },
    },
    setup(props) {
        return () => h(
            'svg',
            {
                class: 'ui-icon',
                width: props.size,
                height: props.size,
                viewBox: '0 0 24 24',
                fill: 'none',
                stroke: 'currentColor',
                'stroke-width': '2',
                'stroke-linecap': 'round',
                'stroke-linejoin': 'round',
                'aria-hidden': 'true',
            },
            (ICONS[props.name] || ICONS.music).map(([tag, attrs]) => h(tag, attrs)),
        )
    },
})

const midiItems = ref([])
const selectedId = ref('')
const timelineRef = ref(null)
const localCurrentMs = ref(0)
const seekVisualLocked = ref(false)
let seekDebounceTimer = 0
let seekVisualLockTimer = 0

const options = reactive({
    midiId: '',
    mode: 'play',
    handMode: 'both',
    speed: 1,
    leftMs: 0,
    rightMs: 0,
    loop: true,
})

const drag = reactive({
    type: '',
    leftMs: 0,
    rightMs: 0,
    currentMs: 0,
    lockedCurrentMs: 0,
    pointerOffsetMs: 0,
})

const playerState = computed(() => store.midiPlayerState || {})
const selectedMidi = computed(() => midiItems.value.find((item) => item.id === selectedId.value))
const matchingPlayerState = computed(() => {
    if (playerState.value.midiId && playerState.value.midiId === selectedId.value) {
        return playerState.value
    }
    return {}
})
const durationMs = computed(() => selectedMidi.value?.durationMs || matchingPlayerState.value.durationMs || 0)
const isPlaybackActive = computed(() => {
    return playerState.value.status !== 'idle' && playerState.value.midiId === selectedId.value
})
const transportLabel = computed(() => {
    if (playerState.value.status === 'playing' && playerState.value.midiId === selectedId.value) return '暂停'
    if (playerState.value.status === 'paused' && playerState.value.midiId === selectedId.value) return '继续'
    return options.mode === 'follow' ? '开始练习' : '开始播放'
})

const displayLeftMs = computed(() => drag.type === 'left' || drag.type === 'right' ? drag.leftMs : options.leftMs)
const displayRightMs = computed(() => drag.type === 'left' || drag.type === 'right' ? drag.rightMs : normalizedRightMs.value)
const displayCurrentMs = computed(() => {
    if (drag.type === 'seek') return drag.currentMs
    if (drag.type === 'left' || drag.type === 'right') {
        return clamp(drag.lockedCurrentMs, 0, durationMs.value)
    }
    if (seekVisualLocked.value) {
        return clamp(localCurrentMs.value || options.leftMs, options.leftMs, normalizedRightMs.value)
    }
    if (isPlaybackActive.value) {
        return clamp(matchingPlayerState.value.currentMs || options.leftMs, options.leftMs, normalizedRightMs.value)
    }
    return clamp(localCurrentMs.value || options.leftMs, options.leftMs, normalizedRightMs.value)
})
const normalizedRightMs = computed(() => {
    return options.rightMs > options.leftMs ? options.rightMs : durationMs.value
})
const leftPercent = computed(() => msToPercent(displayLeftMs.value))
const rightPercent = computed(() => msToPercent(displayRightMs.value))
const currentPercent = computed(() => msToPercent(displayCurrentMs.value))
const activePercent = computed(() => Math.max(0, rightPercent.value - leftPercent.value))

async function refreshMidiStore() {
    midiItems.value = await Keyboard.GetMidiStore()
    if (!selectedId.value && midiItems.value.length > 0) {
        await selectMidi(midiItems.value[0])
    }
}

async function importMidi() {
    try {
        const item = await Keyboard.OpenMidiFileDialog()
        await refreshMidiStore()
        if (item?.id) {
            await selectMidi(item)
            window.$notify?.success?.('MIDI 已导入', item.name || '文件已加入 MIDI 列表')
        }
    } catch (error) {
        if (isUserCancelled(error)) return
        showErrorNotice('MIDI 导入失败', error)
    }
}

async function selectMidi(item) {
    if (!item?.id) return
    if (item.id === selectedId.value) return

    try {
        const shouldSwitchNow = playerState.value.status === 'playing'
        const loaded = await Keyboard.LoadMidiByID(item.id)
        selectedId.value = loaded.id
        options.midiId = loaded.id
        options.leftMs = 0
        options.rightMs = loaded.durationMs || 0
        localCurrentMs.value = 0

        if (shouldSwitchNow) {
            await runPlaybackCommand(() => Keyboard.SwitchMidiPlayback(buildPlaybackOptions({
                midiId: loaded.id,
                leftMs: 0,
                rightMs: loaded.durationMs || 0,
            })), '切换 MIDI 失败')
        }
    } catch (error) {
        showErrorNotice('MIDI 加载失败', error)
    }
}

async function quickPlay(item) {
    if (!item?.id) return
    if (item.id !== selectedId.value) {
        await selectMidi(item)
    }
    await toggleTransport()
}

async function removeMidiItem(item) {
    if (!item?.id) return
    const removingCurrent = item.id === selectedId.value
    if (removingCurrent) {
        await stopPlayback()
        selectedId.value = ''
        resetOptions()
    }
    await Keyboard.RemoveMidiByID(item.id)
    await refreshMidiStore()
}

async function removeSelected() {
    if (!selectedId.value) return
    const item = midiItems.value.find((midi) => midi.id === selectedId.value)
    await removeMidiItem(item)
}

async function toggleTransport() {
    if (!selectedId.value) return
    if (matchingPlayerState.value.status === 'playing') {
        await Keyboard.PauseMidiPlayback()
        return
    }
    if (matchingPlayerState.value.status === 'paused') {
        await Keyboard.ResumeMidiPlayback()
        return
    }
    await runPlaybackCommand(() => Keyboard.StartMidiPlayback(buildPlaybackOptions()), 'MIDI 播放失败')
}

async function stopPlayback() {
    if (playerState.value.status === 'idle') return
    await Keyboard.StopMidiPlayback()
    localCurrentMs.value = options.leftMs
}

async function setPlaybackOption(key, value) {
    if (options[key] === value) return
    const previous = options[key]
    options[key] = value
    const ok = await syncPlaybackOptions()
    if (!ok) {
        options[key] = previous
    }
}

async function syncPlaybackOptions() {
    if (!isPlaybackActive.value) return true
    return runPlaybackCommand(() => Keyboard.SetMidiPlaybackOptions(buildPlaybackOptions()), '播放设置未生效')
}

function buildPlaybackOptions(overrides = {}) {
    return {
        midiId: selectedId.value,
        mode: options.mode,
        handMode: options.handMode,
        speed: options.speed,
        leftMs: options.leftMs,
        rightMs: normalizedRightMs.value,
        loop: true,
        leadWindowMs: DEFAULT_WINDOW_MS,
        groupWindowMs: DEFAULT_WINDOW_MS,
        ...overrides,
    }
}

async function runPlaybackCommand(command, title = '操作失败') {
    try {
        await command()
        return true
    } catch (error) {
        showErrorNotice(title, error)
        return false
    }
}

function showErrorNotice(title, error) {
    window.$notify?.error?.(title, formatError(error))
}

function formatError(error) {
    return String(error?.message || error || '未知错误')
}

function isUserCancelled(error) {
    const message = formatError(error).toLowerCase()
    return message.includes('cancel') || message.includes('取消') || message.includes('未选择')
}

function startSeekDrag(event) {
    if (drag.type || !selectedId.value || durationMs.value <= 0) return
    if (event?.target?.closest?.('.anchor')) return
    event?.preventDefault?.()
    event?.stopPropagation?.()
    drag.type = 'seek'
    drag.pointerOffsetMs = 0
    updateSeekDraft(event)
    bindDragListeners()
}

function startAnchorDrag(type, event) {
    if (drag.type || !selectedId.value || durationMs.value <= 0) return
    event?.preventDefault?.()
    event?.stopPropagation?.()

    drag.type = type
    drag.leftMs = options.leftMs
    drag.rightMs = normalizedRightMs.value
    drag.currentMs = displayCurrentMs.value
    drag.lockedCurrentMs = displayCurrentMs.value

    const anchorMs = type === 'left' ? drag.leftMs : drag.rightMs
    drag.pointerOffsetMs = pointerToMs(event) - anchorMs

    bindDragListeners()
}

function bindDragListeners() {
    window.addEventListener('pointermove', handlePointerMove, { passive: false })
    window.addEventListener('pointerup', finishDrag)
    window.addEventListener('pointercancel', finishDrag)
}

function unbindDragListeners() {
    window.removeEventListener('pointermove', handlePointerMove)
    window.removeEventListener('pointerup', finishDrag)
    window.removeEventListener('pointercancel', finishDrag)
}

function handlePointerMove(event) {
    if (!drag.type) return
    event?.preventDefault?.()

    if (drag.type === 'seek') {
        updateSeekDraft(event)
        return
    }
    if (drag.type === 'left' || drag.type === 'right') {
        updateAnchorDraft(event)
    }
}

async function finishDrag() {
    const type = drag.type
    unbindDragListeners()

    if (type === 'seek') {
        const targetMs = drag.currentMs
        localCurrentMs.value = targetMs
        drag.type = ''
        lockSeekVisual()
        if (isPlaybackActive.value) {
            debounceSeek(targetMs)
        }
        return
    }

    if (type === 'left' || type === 'right') {
        const previousLeftMs = options.leftMs
        const previousRightMs = options.rightMs
        const previousCurrentMs = localCurrentMs.value
        options.leftMs = drag.leftMs
        options.rightMs = drag.rightMs
        localCurrentMs.value = clamp(drag.lockedCurrentMs, options.leftMs, options.rightMs)
        drag.type = ''
        drag.pointerOffsetMs = 0
        const ok = await syncPlaybackOptions()
        if (!ok) {
            options.leftMs = previousLeftMs
            options.rightMs = previousRightMs
            localCurrentMs.value = previousCurrentMs
        }
        return
    }

    drag.type = ''
}

function debounceSeek(ms) {
    window.clearTimeout(seekDebounceTimer)
    seekDebounceTimer = window.setTimeout(() => {
        runPlaybackCommand(() => Keyboard.SeekMidiPlayback(ms), '跳转播放位置失败')
    }, SEEK_DEBOUNCE_MS)
}

function lockSeekVisual() {
    seekVisualLocked.value = true
    window.clearTimeout(seekVisualLockTimer)
    seekVisualLockTimer = window.setTimeout(() => {
        seekVisualLocked.value = false
    }, SEEK_VISUAL_LOCK_MS)
}

function updateSeekDraft(event) {
    const ms = pointerToMs(event)
    drag.currentMs = clamp(ms, options.leftMs, normalizedRightMs.value)
}

function updateAnchorDraft(event) {
    const ms = pointerToMs(event) - drag.pointerOffsetMs
    const minGap = Math.min(MIN_RANGE_MS, durationMs.value)

    if (drag.type === 'left') {
        drag.leftMs = clamp(ms, 0, Math.max(0, drag.rightMs - minGap))
    } else if (drag.type === 'right') {
        drag.rightMs = clamp(ms, Math.min(durationMs.value, drag.leftMs + minGap), durationMs.value)
    }

    drag.currentMs = clamp(drag.currentMs, drag.leftMs, drag.rightMs)
}

function pointerToMs(event) {
    const rect = timelineRef.value?.getBoundingClientRect()
    if (!rect?.width) return 0
    const ratio = clamp((event.clientX - rect.left) / rect.width, 0, 1)
    return ratio * durationMs.value
}

function msToPercent(ms) {
    if (durationMs.value <= 0) return 0
    return clamp(ms / durationMs.value, 0, 1) * 100
}

function clamp(value, min, max) {
    if (max < min) return min
    return Math.min(Math.max(Number(value) || 0, min), max)
}

function resetOptions() {
    options.midiId = ''
    options.mode = 'play'
    options.handMode = 'both'
    options.speed = 1
    options.leftMs = 0
    options.rightMs = 0
    options.loop = true
    localCurrentMs.value = 0
}

function formatDuration(ms = 0) {
    const totalSeconds = Math.max(0, Math.floor(ms / 1000))
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = String(totalSeconds % 60).padStart(2, '0')
    return `${minutes}:${seconds}`
}

watch(playerState, (state) => {
    if (!state?.midiId || state.midiId !== selectedId.value || drag.type) return
    options.midiId = state.midiId
    options.mode = state.mode || options.mode
    options.handMode = state.handMode || options.handMode
    options.speed = state.speed || options.speed
    options.leftMs = state.leftMs || 0
    options.rightMs = state.rightMs || durationMs.value
    options.loop = state.loop !== false

    if (!seekVisualLocked.value) {
        localCurrentMs.value = state.currentMs || options.leftMs
    }
}, { deep: true })

watch(durationMs, (value) => {
    if (!value || options.rightMs) return
    options.rightMs = value
})

onMounted(async () => {
    store.midiWindowOpen = true
    await refreshMidiStore()
})

onBeforeUnmount(() => {
    unbindDragListeners()
    window.clearTimeout(seekDebounceTimer)
    window.clearTimeout(seekVisualLockTimer)
})
</script>

<style lang="scss" scoped>
:global(*) {
    box-sizing: border-box;
}

.midi-window {
    --blue: #1478ff;
    --blue-dark: #0759d6;
    --blue-soft: #eaf4ff;
    --ink: #101828;
    --muted: #667085;
    --line: rgba(136, 153, 176, 0.22);
    --surface: rgba(255, 255, 255, 0.78);

    display: grid;
    grid-template-columns: 270px minmax(0, 1fr);
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    color: var(--ink);
    background:
        radial-gradient(circle at 80% 0%, rgba(20, 120, 255, 0.12), transparent 34%),
        linear-gradient(135deg, #f7fbff 0%, #eef5ff 100%);
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "SF Pro Display", "Segoe UI", sans-serif;
}

button,
input {
    font: inherit;
}

button {
    -webkit-tap-highlight-color: transparent;
}

.ui-icon {
    display: block;
    flex: 0 0 auto;
}

.library {
    min-width: 0;
    padding: 12px;
    border-right: 1px solid var(--line);
    background: rgba(255, 255, 255, 0.58);
    backdrop-filter: blur(20px);
}

.library-head {
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    margin-bottom: 10px;
    --wails-draggable: drag;
}

.library-title {
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 9px;
    color: var(--blue-dark);
}

.library-title > div {
    min-width: 0;
}

.library-title strong {
    display: block;
    color: var(--ink);
    font-size: 15px;
    font-weight: 820;
    letter-spacing: -0.03em;
}

.library-title span {
    display: block;
    margin-top: 1px;
    color: var(--muted);
    font-size: 11px;
}

.head-action {
    width: 34px;
    height: 34px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid rgba(20, 120, 255, 0.18);
    border-radius: 12px;
    color: var(--blue-dark);
    background: rgba(255, 255, 255, 0.82);
    cursor: pointer;
    transition: 160ms ease;
    --wails-draggable: no-drag;
}

.head-action:hover {
    color: #fff;
    background: var(--blue);
    box-shadow: 0 10px 24px rgba(20, 120, 255, 0.22);
}

.midi-list {
    height: calc(100vh - 66px);
    overflow: auto;
    padding-right: 2px;
    scrollbar-width: thin;
    scrollbar-color: rgba(91, 107, 130, 0.28) transparent;
}

.midi-list::-webkit-scrollbar {
    width: 6px;
}

.midi-list::-webkit-scrollbar-thumb {
    border-radius: 999px;
    background: rgba(91, 107, 130, 0.28);
}

.midi-item {
    position: relative;
    min-height: 44px;
    display: grid;
    grid-template-columns: 30px minmax(0, 1fr) auto;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;
    padding: 8px;
    overflow: hidden;
    border: 1px solid transparent;
    border-radius: 14px;
    background: transparent;
    cursor: pointer;
    transition: 150ms ease;
}

.midi-item:hover {
    background: rgba(255, 255, 255, 0.74);
}

.midi-item.selected {
    border-color: rgba(20, 120, 255, 0.25);
    background: linear-gradient(135deg, rgba(20, 120, 255, 0.13), rgba(255, 255, 255, 0.78));
}

.midi-item.playing .midi-icon {
    color: #fff;
    background: var(--blue);
}

.midi-icon {
    width: 30px;
    height: 30px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 10px;
    color: var(--blue-dark);
    background: var(--blue-soft);
}

.midi-name {
    min-width: 0;
    overflow: hidden;
    color: #172033;
    font-size: 13px;
    font-weight: 720;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.midi-time {
    color: var(--muted);
    font-size: 11px;
    font-weight: 680;
}

.midi-actions {
    position: absolute;
    right: 6px;
    top: 50%;
    display: flex;
    align-items: center;
    gap: 5px;
    padding-left: 28px;
    opacity: 0;
    pointer-events: none;
    background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.96) 34%);
    transform: translate(8px, -50%);
    transition: 150ms ease;
}

.midi-item:hover .midi-actions {
    opacity: 1;
    pointer-events: auto;
    transform: translate(0, -50%);
}

.midi-actions button {
    width: 28px;
    height: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid rgba(148, 164, 184, 0.22);
    border-radius: 10px;
    color: var(--blue-dark);
    background: #fff;
    cursor: pointer;
    transition: 150ms ease;
}

.midi-actions button:hover {
    color: #fff;
    border-color: transparent;
    background: var(--blue);
}

.midi-actions button.danger {
    color: #b42318;
}

.midi-actions button.danger:hover {
    color: #fff;
    background: #e5484d;
}

.empty-state {
    min-height: 180px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    color: var(--muted);
    border: 1px dashed rgba(116, 132, 154, 0.28);
    border-radius: 16px;
    background: rgba(255, 255, 255, 0.44);
}

.empty-state button {
    height: 30px;
    padding: 0 12px;
    border: 0;
    border-radius: 999px;
    color: #fff;
    background: var(--blue);
    cursor: pointer;
}

.workspace {
    min-width: 0;
    display: grid;
    grid-template-rows: auto minmax(0, 1fr);

    overflow: hidden;
}

.compact-topbar,
.player-panel {

    background: var(--surface);
    box-shadow: 0 14px 34px rgba(16, 36, 68, 0.07);
    backdrop-filter: blur(20px);
}

.compact-topbar {
    min-height: 70px;
    display: grid;
    border-bottom: 1px solid rgba(148, 164, 184, 0.2);
    grid-template-columns: minmax(180px, 1fr) auto;
    align-items: center;
    gap: 14px;
    padding: 12px;
    --wails-draggable: drag;
}

.track-info {
    min-width: 0;
}

.label {
    display: block;
    color: var(--muted);
    font-size: 10px;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
}

.track-info strong {
    display: block;
    margin-top: 3px;
    overflow: hidden;
    color: #0f1b2f;
    font-size: clamp(16px, 2vw, 22px);
    font-weight: 850;
    letter-spacing: -0.05em;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.option-strip {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    flex-wrap: wrap;
    gap: 8px;
    --wails-draggable: no-drag;
}

.segmented {
    height: 34px;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 3px;
    border: 1px solid rgba(148, 164, 184, 0.2);
    border-radius: 13px;
    background: rgba(255, 255, 255, 0.72);
}

.segmented button {
    height: 26px;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 0 10px;
    border: 0;
    border-radius: 10px;
    color: #475467;
    background: transparent;
    cursor: pointer;
    font-size: 12px;
    font-weight: 760;
    transition: 150ms ease;
}

.segmented button:hover {
    color: var(--blue-dark);
    background: rgba(20, 120, 255, 0.08);
}

.segmented button.active {
    color: #fff;
    background: linear-gradient(135deg, #1687ff, #0759d6);
    box-shadow: 0 8px 18px rgba(20, 120, 255, 0.2);
}

.hand-mode button {
    min-width: 30px;
    justify-content: center;
    padding: 0 8px;
}

.speed-control {
    height: 34px;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-width: 210px;
    padding: 0 10px;
    border: 1px solid rgba(148, 164, 184, 0.2);
    border-radius: 13px;
    color: var(--blue-dark);
    background: rgba(255, 255, 255, 0.72);
}

.speed-control input {
    width: 110px;
    cursor: pointer;
    accent-color: var(--blue);
}

.speed-control b {
    width: 42px;
    color: #182230;
    font-size: 12px;
    font-weight: 820;
    text-align: right;
}

.player-panel {
    min-height: 190px;
    display: grid;
    grid-template-rows: auto 1fr;
    padding: 12px;
    overflow: hidden;
}

.transport-bar {
    min-height: 58px;
    display: grid;
    grid-template-columns: auto minmax(120px, 1fr) auto auto;
    align-items: center;
    gap: 10px;
    padding: 8px;
    border: 1px solid rgba(148, 164, 184, 0.14);
    border-radius: 18px;
    background: rgba(255, 255, 255, 0.58);
}

.main-play {
    width: 46px;
    height: 46px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 0;
    border-radius: 16px;
    color: #fff;
    background: linear-gradient(135deg, #1687ff, #0759d6);
    box-shadow: 0 12px 28px rgba(20, 120, 255, 0.24);
    cursor: pointer;
    transition: 150ms ease;
}

.main-play:hover:not(:disabled) {
    transform: translateY(-1px) scale(1.02);
}

.main-play.active {
    background: linear-gradient(135deg, #111827, #344054);
    box-shadow: 0 12px 28px rgba(15, 23, 42, 0.2);
}

.transport-meta {
    min-width: 0;
}

.transport-meta strong {
    display: block;
    color: #111827;
    font-size: 14px;
    font-weight: 820;
}

.transport-meta span {
    display: block;
    margin-top: 3px;
    color: var(--muted);
    font-size: 11px;
    font-weight: 700;
}

.mini-tool {
    height: 34px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 0 11px;
    border: 1px solid rgba(148, 164, 184, 0.2);
    border-radius: 12px;
    color: #475467;
    background: rgba(255, 255, 255, 0.8);
    cursor: pointer;
    transition: 150ms ease;
}

.mini-tool:hover:not(:disabled) {
    color: var(--blue-dark);
    background: var(--blue-soft);
}

.range-readout {
    display: inline-flex;
    align-items: center;
    gap: 6px;
}

.range-readout span {
    height: 26px;
    display: inline-flex;
    align-items: center;
    padding: 0 8px;
    border-radius: 999px;
    color: var(--muted);
    background: rgba(15, 23, 42, 0.05);
    font-size: 10px;
    font-weight: 840;
    letter-spacing: 0.04em;
}

.timeline {
    position: relative;
    min-height: 108px;
    padding: 14px 4px 16px;
    cursor: pointer;
    touch-action: none;
    user-select: none;
}

.timeline.disabled {
    cursor: not-allowed;
    opacity: 0.5;
}

.ruler {
    display: grid;
    grid-template-columns: repeat(16, 1fr) 1px;
    height: 24px;
    padding: 0 4px;
}

.ruler span {
    position: relative;
    border-left: 1px solid rgba(76, 92, 114, 0.14);
}

.ruler span:nth-child(odd)::after {
    content: "";
    position: absolute;
    left: -1px;
    top: 0;
    width: 1px;
    height: 14px;
    background: rgba(76, 92, 114, 0.14);
}

.ruler span:nth-child(even)::after {
    content: "";
    position: absolute;
    left: -1px;
    top: 0;
    width: 1px;
    height: 8px;
    background: rgba(76, 92, 114, 0.1);
}

.rail {
    position: relative;
    height: 32px;
    margin: 2px 4px 0;
    overflow: visible;
    border: 1px solid rgba(15, 23, 42, 0.08);
    border-radius: 12px;
    background:
        repeating-linear-gradient(90deg, rgba(102, 112, 133, 0.12) 0 1px, transparent 1px 18px),
        #dce7f4;
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.78), 0 10px 22px rgba(16, 36, 68, 0.08);
}

.range-before,
.range-after,
.range-active {
    position: absolute;
    top: 0;
    bottom: 0;
    overflow: hidden;
}

.range-before,
.range-after {
    background: rgba(172, 187, 207, 0.72);
}

.range-before {
    left: 0;
    border-radius: 12px 0 0 12px;
}

.range-after {
    right: 0;
    border-radius: 0 12px 12px 0;
}

.range-active {
    border-radius: 11px;
    background:
        repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.22) 0 1px, transparent 1px 20px),
        linear-gradient(90deg, #0969e8, #49a5ff);
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.32), 0 10px 22px rgba(20, 120, 255, 0.24);
}

.anchor,
.playhead {
    position: absolute;
    top: 50%;
    border: 0;
    padding: 0;
    cursor: grab;
    transform: translate(-50%, -50%);
    --wails-draggable: no-drag;
}

.anchor:active,
.playhead:active {
    cursor: grabbing;
}

.anchor {
    z-index: 9;
    width: 20px;
    height: 54px;
    border-radius: 10px;
    color: #fff;
    background: linear-gradient(180deg, #1687ff, #0759d6);
    box-shadow: 0 12px 22px rgba(20, 120, 255, 0.26);
    touch-action: none;
}

.anchor::before {
    content: \"\";
    position: absolute;
    inset: -10px -8px;
}

.anchor span {
    position: absolute;
    left: 50%;
    top: 6px;
    font-size: 7px;
    font-weight: 900;
    letter-spacing: 0.08em;
    transform: translateX(-50%);
}

.anchor::after {
    content: "";
    position: absolute;
    left: 50%;
    top: 22px;
    bottom: 8px;
    width: 2px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.7);
    transform: translateX(-50%);
}

.right-anchor {
    background: linear-gradient(180deg, #0a64d8, #082c75);
}

.playhead {
    z-index: 6;
    width: 24px;
    height: 70px;
    background: transparent;
}

.playhead::before {
    content: "";
    position: absolute;
    left: 50%;
    top: -11px;
    width: 22px;
    height: 18px;
    border-radius: 8px 8px 10px 10px;
    background: #101828;
    box-shadow: 0 10px 22px rgba(16, 24, 40, 0.22);
    transform: translateX(-50%);
}

.playhead::after {
    content: "";
    position: absolute;
    left: 50%;
    top: 3px;
    bottom: -12px;
    width: 2px;
    border-radius: 999px;
    background: #101828;
    box-shadow: 0 0 0 3px rgba(16, 24, 40, 0.08);
    transform: translateX(-50%);
}

.playhead i {
    position: absolute;
    left: 50%;
    top: 1px;
    z-index: 1;
    width: 9px;
    height: 9px;
    background: #101828;
    transform: translateX(-50%) rotate(45deg);
}

.timeline.dragging .anchor,
.timeline.dragging .playhead::before,
.timeline.dragging .playhead::after {
    filter: drop-shadow(0 10px 18px rgba(20, 120, 255, 0.2));
}

.timeline.dragging-anchor .playhead {
    pointer-events: none;
}

.timeline.dragging-anchor .playhead::before,
.timeline.dragging-anchor .playhead::after {
    filter: none;
}

button:disabled {
    cursor: not-allowed;
    opacity: 0.42;
    transform: none !important;
    box-shadow: none !important;
}

@media (max-width: 1060px) {
    .midi-window {
        grid-template-columns: 240px minmax(0, 1fr);
    }

    .compact-topbar {
        grid-template-columns: 1fr;
    }

    .option-strip {
        justify-content: flex-start;
    }
}

@media (max-width: 760px) {
    .midi-window {
        grid-template-columns: 1fr;
        grid-template-rows: auto minmax(0, 1fr);
    }

    .library {
        border-right: 0;
        border-bottom: 1px solid var(--line);
    }

    .midi-list {
        height: auto;
        display: grid;
        grid-auto-flow: column;
        grid-auto-columns: minmax(220px, 72vw);
        gap: 8px;
        overflow-x: auto;
        overflow-y: hidden;
        padding-bottom: 2px;
    }

    .midi-item {
        margin-bottom: 0;
    }

    .workspace {
        overflow: auto;
    }

    .transport-bar {
        grid-template-columns: auto minmax(0, 1fr) auto;
    }

    .range-readout {
        grid-column: 1 / -1;
    }
}

@media (max-width: 520px) {
    .library,
    .workspace {
        padding: 10px;
    }

    .option-strip,
    .speed-control {
        width: 100%;
    }

    .segmented,
    .speed-control {
        flex: 1;
    }

    .speed-control input {
        flex: 1;
        width: auto;
    }

    .transport-bar {
        grid-template-columns: auto minmax(0, 1fr);
    }

    .mini-tool {
        grid-column: 1 / -1;
        justify-content: center;
    }
}
</style>
