<template>
    <div class="midi-window">
        <input ref="fileInput" class="hidden-input" type="file" accept=".mid,.midi,audio/midi" @change="handleFileChange" />

        <MidiDirectory
            :items="midiItems"
            :selected-id="selectedMidiId"
            @import="chooseMidiFile"
            @select="handleSelectMidi"
            @remove="handleRemoveMidi"
        />

        <main class="midi-main">
            <header class="top-area">
                <div class="file-meta">
                    <div class="window-title">MIDI 播放 / 练习</div>
                    <div class="file-title">{{ currentTitle }}</div>
                    <div class="file-path">{{ selectedMidi?.path || '请先从左侧导入 MIDI 文件' }}</div>
                </div>
                <MidiModeSelector v-model:mode="mode" />
            </header>

            <section v-if="hasFile" class="control-card" :class="{success: stepSuccess}">
                <div class="mode-panel" v-if="mode === 'practice'">
                    <div class="panel-title">练习声部</div>
                    <HandSelector
                        v-model:left="practiceLeft"
                        v-model:right="practiceRight"
                        :single-track-only="singleTrackOnly"
                    />
                    <div class="hint-text">
                        {{ singleTrackOnly ? '当前 MIDI 只有一个有效音轨，已自动降级为单手练习。' : '练一只手时，另一只手声部会按当前速度自动播放。' }}
                    </div>
                </div>

                <div v-else class="mode-panel preview-panel">
                    <div class="panel-title">播放模式</div>
                    <div class="hint-text">播放模式会直接播放当前 MIDI，并遵守底部左右锚点限制。</div>
                </div>

                <div class="stats-row">
                    <span>时长 {{ formatTime(store.player.duration) }}</span>
                    <span>{{ store.player.totalNotes }} notes</span>
                    <span>{{ store.player.tracks.length }} tracks</span>
                    <span>{{ statusText }}</span>
                </div>

                <div v-if="store.player.error || errorText" class="error-box">{{ store.player.error || errorText }}</div>
            </section>

            <section v-else class="empty-main">
                <div class="empty-mark">♪</div>
                <div>选择或导入一首 MIDI 后开始播放 / 练习</div>
                <small>目录只保存系统绝对路径，移除记录不会删除真实文件。</small>
            </section>

            <footer class="bottom-bar" :class="{disabled: !hasFile}">
                <AnchorProgress
                    :duration="store.player.duration"
                    :current="store.player.currentTime"
                    v-model:left-anchor="leftAnchor"
                    v-model:right-anchor="rightAnchor"
                    @seek="seekTo"
                />
                <PlaybackControls
                    :playing="isPlaying"
                    :paused="isPaused"
                    :rate="store.player.playbackRate"
                    @restart="restartFromLeftAnchor"
                    @toggle="togglePlayOrPractice"
                    @rate="setPlaybackRate"
                />
            </footer>
        </main>
    </div>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import MidiDirectory from '../components/midi/MidiDirectory.vue'
import MidiModeSelector from '../components/midi/MidiModeSelector.vue'
import HandSelector from '../components/midi/HandSelector.vue'
import AnchorProgress from '../components/midi/AnchorProgress.vue'
import PlaybackControls from '../components/midi/PlaybackControls.vue'
import {useMidiLibrary} from '../composables/useMidiLibrary'
import {PLAYER_STATUS} from '../services/midiPlaybackController'
import {applyBackendPlayerState, applyParsedMidiToStore} from '../services/backendMidiService'
import {emitWindowBus} from '../services/windowBus'

const store = inject('store')
const Keyboard = inject('Keyboard')
const fileInput = ref(null)

const {
    midiItems,
    selectedMidiId,
    selectedMidi,
    importMidiFile,
    selectMidi,
    removeMidi,
} = useMidiLibrary(Keyboard, store)

const mode = ref('play')
const practiceLeft = ref(true)
const practiceRight = ref(true)
const leftAnchor = ref(0)
const rightAnchor = ref(0)
const errorText = ref('')
const practicePlan = ref(createEmptyPracticePlan())
const practiceRunning = ref(false)
const practicePaused = ref(false)
const stepSuccess = ref(false)
const currentPracticeStep = ref(0)

let autoAdvanceTimer = null
let anchorStopTimer = null

const hasFile = computed(() => store.player.notes.length > 0)
const isPlaying = computed(() => store.player.status === PLAYER_STATUS.PLAYING || (practiceRunning.value && !practicePaused.value))
const isPaused = computed(() => store.player.status === PLAYER_STATUS.PAUSED || practicePaused.value)
const currentTitle = computed(() => selectedMidi.value?.name || store.player.fileName || '未选择 MIDI')
const singleTrackOnly = computed(() => practicePlan.value.singleTrackOnly || effectiveTrackCount.value <= 1)
const effectiveTrackCount = computed(() => store.player.tracks.filter((track) => track.noteCount > 0).length)
const statusText = computed(() => {
    const map = {
        [PLAYER_STATUS.IDLE]: '未加载',
        [PLAYER_STATUS.READY]: '已就绪',
        [PLAYER_STATUS.PLAYING]: '播放中',
        [PLAYER_STATUS.PAUSED]: '已暂停',
        [PLAYER_STATUS.STOPPED]: '已停止',
        [PLAYER_STATUS.FINISHED]: '已完成',
    }
    if (practicePaused.value) return '练习暂停'
    if (practiceRunning.value) return '练习中'
    return map[store.player.status] || store.player.status
})

watch(() => store.player.duration, (duration) => {
    // 新 MIDI 加载后，锚点默认覆盖整首；用户之后可手动收窄区间。
    leftAnchor.value = 0
    rightAnchor.value = Number(duration || 0)
}, {immediate: true})

watch([practiceLeft, practiceRight, () => store.player.notes], rebuildPracticePlan, {deep: true})

watch([leftAnchor, rightAnchor], async () => {
    // 区间变化后需要重新生成练习计划；如果正在练习，先停掉旧步骤，避免提示音和锚点区间不一致。
    if (!hasFile.value) return
    if (practiceRunning.value || practicePaused.value) await stopPractice()
    await rebuildPracticePlan()
    if (store.player.status === PLAYER_STATUS.PLAYING) armAnchorStopGuard()
})

watch(mode, async () => {
    // 切换播放 / 练习模式时，立即停止当前旧状态，避免两个模式互相残留声音或高亮。
    await stopAll(false)
})

watch(() => store.player.currentTime, (value) => {
    // 后端播放器没有区间概念，前端用右锚点做统一边界控制：到达右锚点就停止。
    if (!hasFile.value || !isPlaying.value) return
    if (rightAnchor.value > 0 && Number(value || 0) >= rightAnchor.value) {
        stopAll()
    }
})

function chooseMidiFile() {
    fileInput.value?.click()
}

async function handleFileChange(event) {
    const file = event.target.files?.[0]
    if (!file) return

    try {
        errorText.value = ''
        await stopAll()
        const result = await importMidiFile(file)
        if (result.status === 'exists') {
            window.$message?.info?.('该 MIDI 路径已存在，已直接选中。')
        }
        applyParsedMidiToStore(store, result.parsed)
        await rebuildPracticePlan()
    } catch (error) {
        errorText.value = error?.message || String(error)
        window.$message?.warning?.(errorText.value)
    } finally {
        event.target.value = ''
    }
}

async function handleSelectMidi(id) {
    await stopAll()
    selectMidi(id)
    store.resetPlayer()
    await loadSelectedMidiRecord()
}

async function loadSelectedMidiRecord() {
    const path = selectedMidi.value?.path || ''
    if (!path) return
    try {
        // 新版后端提供绝对路径读取；如果绑定还没重新生成，会自动降级为提示用户重新导入。
        if (typeof Keyboard.CheckMidiPathExists === 'function') {
            const exists = await Keyboard.CheckMidiPathExists(path)
            if (!exists) {
                errorText.value = '该 MIDI 文件路径已不存在，请从左侧移除这条 MIDI 记录。'
                return
            }
        }
        if (typeof Keyboard.LoadMidiFileFromPath === 'function') {
            const parsed = await Keyboard.LoadMidiFileFromPath(path)
            applyParsedMidiToStore(store, parsed)
            await rebuildPracticePlan()
            errorText.value = ''
            return
        }
        errorText.value = '已切换 MIDI。请重新运行 wails3 generate 生成绑定，或点击“导入”重新选择该文件完成加载。'
    } catch (error) {
        errorText.value = `加载 MIDI 失败：${error?.message || error}。如果文件已不存在，请移除该记录。`
    }
}

async function handleRemoveMidi(id) {
    const wasSelected = selectedMidiId.value === id
    await stopAll()
    removeMidi(id)
    store.resetPlayer()
    if (wasSelected && selectedMidi.value) {
        await loadSelectedMidiRecord()
    }
}

async function togglePlayOrPractice() {
    if (!hasFile.value) return
    if (mode.value === 'play') {
        if (store.player.status === PLAYER_STATUS.PLAYING) {
            applyBackendPlayerState(store, await Keyboard.PauseMidiPlayback())
        } else {
            await startPlaybackFromCurrentOrLeft()
        }
        return
    }

    if (practiceRunning.value && !practicePaused.value) {
        await pausePractice()
    } else if (practicePaused.value) {
        await resumePractice()
    } else {
        await startPractice()
    }
}

async function startPlaybackFromCurrentOrLeft() {
    await stopPractice()
    const start = clampToAnchors(store.player.currentTime || leftAnchor.value)
    applyBackendPlayerState(store, await Keyboard.SeekMidiPlayback(start))
    applyBackendPlayerState(store, await Keyboard.StartMidiPlayback())
    armAnchorStopGuard()
}

async function restartFromLeftAnchor() {
    if (!hasFile.value) return
    await stopAll(false)
    applyBackendPlayerState(store, await Keyboard.SeekMidiPlayback(leftAnchor.value))
    if (mode.value === 'practice') {
        await startPractice()
    } else {
        applyBackendPlayerState(store, await Keyboard.StartMidiPlayback())
        armAnchorStopGuard()
    }
}

async function seekTo(value) {
    const target = clampToAnchors(value)
    applyBackendPlayerState(store, await Keyboard.SeekMidiPlayback(target))
    if (mode.value === 'practice' && practicePlan.value.steps.length) {
        currentPracticeStep.value = findPracticeStepIndex(target)
        if (practiceRunning.value && !practicePaused.value) showPracticeStep()
        if (practicePaused.value) showPracticeHintOnly()
    }
}

async function setPlaybackRate(rate) {
    applyBackendPlayerState(store, await Keyboard.SetMidiPlaybackRate(rate))
    if (store.player.status === PLAYER_STATUS.PLAYING) armAnchorStopGuard()
}

async function startPractice() {
    await stopPreviewOnly()
    await rebuildPracticePlan()
    if (!practicePlan.value.steps.length) {
        errorText.value = '当前区间没有可练习的音符，请调整左右锚点。'
        return
    }
    errorText.value = ''
    practiceRunning.value = true
    practicePaused.value = false
    currentPracticeStep.value = 0
    showPracticeStep()
}

async function pausePractice() {
    // 练习暂停只停自动伴奏和自动推进，不清掉当前步骤，继续时仍从当前提示继续。
    practicePaused.value = true
    clearTimeout(autoAdvanceTimer)
    await Keyboard.StopFollowAutoNotes()
}

async function resumePractice() {
    if (!practicePaused.value) return
    practicePaused.value = false
    practiceRunning.value = true
    showPracticeStep()
}

async function stopPractice() {
    practiceRunning.value = false
    practicePaused.value = false
    currentPracticeStep.value = 0
    clearTimeout(autoAdvanceTimer)
    await Keyboard.StopFollowAutoNotes()
    clearPracticeVisuals()
}


async function stopPreviewOnly(resetTime = false) {
    if (store.player.status === PLAYER_STATUS.PLAYING || store.player.status === PLAYER_STATUS.PAUSED) {
        applyBackendPlayerState(store, await Keyboard.StopMidiPlayback())
        if (!resetTime) applyBackendPlayerState(store, await Keyboard.SeekMidiPlayback(leftAnchor.value))
    }
}

async function stopAll(resetTime = true) {
    clearTimeout(anchorStopTimer)
    await stopPractice()
    applyBackendPlayerState(store, await Keyboard.StopMidiPlayback())
    if (!resetTime) return
    applyBackendPlayerState(store, await Keyboard.SeekMidiPlayback(leftAnchor.value))
    await Keyboard.AllNotesOff()
}

async function rebuildPracticePlan() {
    if (!store.player.notes.length) {
        practicePlan.value = createEmptyPracticePlan()
        return
    }

    const hand = effectiveTrackCount.value <= 1 ? 'single' : resolvePracticeHand()
    const options = {
        threshold: 0.06,
        start: leftAnchor.value,
        end: rightAnchor.value || store.player.duration,
        practiceHand: hand,
        autoPlayOtherHand: hand !== 'both',
    }
    practicePlan.value = await Keyboard.BuildFollowPracticePlan(store.player.notes, options)
    if (practicePlan.value.singleTrackOnly) {
        practiceLeft.value = true
        practiceRight.value = false
    }
}

function resolvePracticeHand() {
    if (practiceLeft.value && practiceRight.value) return 'both'
    if (practiceLeft.value) return 'left'
    if (practiceRight.value) return 'right'
    // 防御：组件层已保证至少一个，这里再兜底一次。
    practiceRight.value = true
    return 'right'
}

function showPracticeStep() {
    clearTimeout(autoAdvanceTimer)
    clearPracticeVisuals()

    const step = practicePlan.value.steps[currentPracticeStep.value]
    if (!step) {
        stopPractice()
        return
    }

    // 练习模式没有后端连续进度，这里把当前步骤时间写回播放器状态，让进度条能跟随步骤前进。
    store.player = {...store.player, currentTime: step.time, status: PLAYER_STATUS.READY}
    showPracticeHintOnly()

    if (step.autoPlayNotes?.length && !practicePaused.value) {
        Keyboard.PlayFollowAutoNotes(step.autoPlayNotes, Number(store.player.playbackRate || 1))
    }

    // 当前步骤没有需要用户按的音时，按音符持续时间自动推进。
    if (!step.practiceNotes.length && !practicePaused.value) {
        const delay = Math.max(100, Math.min(1200, (step.duration || 0.12) * 1000 / Math.max(store.player.playbackRate || 1, 0.1)))
        autoAdvanceTimer = setTimeout(nextPracticeStep, delay)
    }
}

function showPracticeHintOnly() {
    clearPracticeVisuals()
    const step = practicePlan.value.steps[currentPracticeStep.value]
    const keys = step?.practiceNotes?.map((note) => note.midi) || []
    store.hintKey = Object.fromEntries(keys.map((key) => [key, true]))
    emitWindowBus('hint:set', {keys})
}

watch(() => ({...store.pressedKey}), () => {
    if (!practiceRunning.value) return
    const step = practicePlan.value.steps[currentPracticeStep.value]
    const required = step?.practiceNotes?.map((note) => note.midi) || []
    if (required.length && required.every((key) => store.pressedKey[key])) {
        nextPracticeStep()
    }
}, {deep: true})

function nextPracticeStep() {
    if (practicePaused.value) return
    stepSuccess.value = true
    setTimeout(() => { stepSuccess.value = false }, 140)
    currentPracticeStep.value++
    if (currentPracticeStep.value >= practicePlan.value.steps.length) {
        stopPractice()
        return
    }
    showPracticeStep()
}

function clearPracticeVisuals() {
    store.hintKey = {}
    store.playbackKey = {}
    emitWindowBus('hint:clear')
    emitWindowBus('playback:clear')
}

function armAnchorStopGuard() {
    clearTimeout(anchorStopTimer)
    if (!rightAnchor.value || rightAnchor.value <= leftAnchor.value) return
    const remain = Math.max(0, rightAnchor.value - (store.player.currentTime || leftAnchor.value))
    anchorStopTimer = setTimeout(() => stopAll(), remain * 1000 / Math.max(store.player.playbackRate || 1, 0.1) + 80)
}

function clampToAnchors(value) {
    return Math.max(leftAnchor.value, Math.min(rightAnchor.value || store.player.duration, Number(value || 0)))
}

function findPracticeStepIndex(time) {
    const steps = practicePlan.value.steps || []
    const target = Number(time || 0)
    const index = steps.findIndex((step) => step.time >= target)
    return index === -1 ? Math.max(0, steps.length - 1) : index
}

function createEmptyPracticePlan() {
    return {steps: [], assignments: [], singleTrackOnly: true, totalPracticeNotes: 0, totalAutoPlayNotes: 0}
}

function formatTime(seconds) {
    const safe = Math.max(0, Number(seconds || 0))
    const minutes = Math.floor(safe / 60)
    const secs = Math.floor(safe % 60)
    return `${minutes}:${String(secs).padStart(2, '0')}`
}

onMounted(() => {
    // 如果用户上次保存过 MIDI 目录，打开独立窗口后自动加载第一条记录，避免出现左侧有选中但右侧空白的黑箱状态。
    if (selectedMidi.value) loadSelectedMidiRecord()
})

onBeforeUnmount(() => stopAll())
</script>

<style lang="scss" scoped>
.midi-window {
    width: 100vw;
    height: 100vh;
    display: grid;
    grid-template-columns: 285px 1fr;
    overflow: hidden;
    background: linear-gradient(135deg, rgba(248, 250, 252, 0.96), rgba(226, 232, 240, 0.92));
    color: #0f172a;
    user-select: none;
}
.hidden-input { display: none; }
.midi-main {
    display: grid;
    grid-template-rows: auto 1fr auto;
    min-width: 0;
    padding: 14px 16px;
    box-sizing: border-box;
}
.top-area {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 18px;
    --wails-draggable: drag;
}
.file-meta { min-width: 0; }
.window-title { font-size: 12px; color: #64748b; }
.file-title {
    margin-top: 3px;
    font-size: 20px;
    font-weight: 800;
    letter-spacing: -0.03em;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}
.file-path {
    margin-top: 3px;
    max-width: 620px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 11px;
    color: #64748b;
}
.control-card,
.empty-main {
    align-self: center;
    border: 1px solid rgba(148, 163, 184, 0.22);
    border-radius: 24px;
    padding: 18px;
    background: rgba(255, 255, 255, 0.68);
    box-shadow: 0 18px 50px rgba(15, 23, 42, 0.08);
    backdrop-filter: blur(18px);
}
.control-card.success {
    border-color: rgba(34, 197, 94, 0.45);
    box-shadow: 0 18px 50px rgba(34, 197, 94, 0.12);
}
.mode-panel { margin-bottom: 14px; }
.preview-panel { min-height: 78px; }
.panel-title { margin-bottom: 8px; font-size: 15px; font-weight: 760; }
.hint-text { margin-top: 9px; font-size: 12px; line-height: 1.6; color: #64748b; }
.stats-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 12px;
}
.stats-row span {
    border-radius: 999px;
    padding: 6px 10px;
    background: rgba(15, 23, 42, 0.06);
    color: #475569;
    font-size: 11px;
}
.error-box {
    margin-top: 12px;
    padding: 10px 12px;
    border-radius: 14px;
    background: rgba(239, 68, 68, 0.1);
    color: #b91c1c;
    font-size: 12px;
    line-height: 1.6;
}
.empty-main {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    min-height: 155px;
    color: #64748b;
}
.empty-mark { font-size: 34px; color: #2563eb; }
.empty-main small { font-size: 12px; }
.bottom-bar {
    display: grid;
    grid-template-columns: 1fr auto;
    align-items: center;
    gap: 18px;
    border-radius: 22px;
    padding: 14px;
    background: rgba(255, 255, 255, 0.72);
    border: 1px solid rgba(148, 163, 184, 0.22);
    box-shadow: 0 14px 36px rgba(15, 23, 42, 0.08);
}
.bottom-bar.disabled { opacity: 0.45; pointer-events: none; }
</style>
