<template>
    <section class="content-card practice-card">
        <div class="section-header">
            <div>
                <div class="section-title">MIDI 练习中心</div>
                <div class="section-desc">先导入 MIDI 文件并预览整首，再进入跟弹练习。播放只做完整预览，区间只用于跟弹。</div>
            </div>
            <n-tag :bordered="false" :type="hasFile ? 'success' : 'warning'">
                {{ hasFile ? '已导入' : '未导入' }}
            </n-tag>
        </div>

        <div class="practice-layout">
            <div class="flow-card">
                <div class="flow-step" :class="{active: !hasFile}">
                    <span>1</span>
                    <div>
                        <strong>导入 MIDI</strong>
                        <p>前端选择文件，上传给 Go 解析。</p>
                    </div>
                </div>
                <div class="flow-line"></div>
                <div class="flow-step" :class="{active: hasFile && !followRunning}">
                    <span>2</span>
                    <div>
                        <strong>预览整首</strong>
                        <p>播放相当于试听一次 MIDI。</p>
                    </div>
                </div>
                <div class="flow-line"></div>
                <div class="flow-step" :class="{active: followRunning}">
                    <span>3</span>
                    <div>
                        <strong>进入跟弹</strong>
                        <p>可设置区间和左右手练习。</p>
                    </div>
                </div>
            </div>

            <div class="import-card">
                <input ref="fileInput" class="hidden-input" type="file" accept=".mid,.midi,audio/midi" @change="handleFileChange" />
                <div class="import-icon">♪</div>
                <div class="import-body">
                    <div class="label">导入 MIDI 文件</div>
                    <div class="desc">支持 .mid / .midi。导入后会自动生成预览数据和跟弹练习计划。</div>
                </div>
                <n-button type="primary" size="small" @click="chooseFile">选择文件</n-button>
            </div>

            <div v-if="store.player.error" class="error-box">{{ store.player.error }}</div>

            <div v-if="hasFile" class="file-info-card">
                <div>
                    <div class="file-name">{{ store.player.fileName }}</div>
                    <div class="desc">
                        时长 {{ formatTime(store.player.duration) }} · {{ store.player.totalNotes }} 个音符 · {{ store.player.tracks.length }} 个轨道
                    </div>
                </div>
                <n-tag :bordered="false" type="info">{{ previewStatusText }}</n-tag>
            </div>

            <div v-if="hasFile" class="section-card">
                <div class="subsection-title">预览播放</div>
                <div class="desc block-desc">预览始终播放整首 MIDI，用来先听一遍内容；不要在这里做区间播放，避免播放器和练习器职责混在一起。</div>

                <div class="progress-block">
                    <div class="time-row">
                        <span>{{ formatTime(store.player.currentTime) }}</span>
                        <span>{{ previewProgressPercent }}%</span>
                        <span>{{ formatTime(store.player.duration) }}</span>
                    </div>
                    <n-slider
                        v-model:value="previewProgressValue"
                        :min="0"
                        :max="store.player.duration"
                        :step="0.01"
                        :tooltip="false"
                        @dragend="seekPreview"
                    />
                </div>

                <div class="control-row">
                    <n-button v-if="!previewPlaying" type="primary" size="small" @click="playPreview">预览播放</n-button>
                    <n-button v-else type="warning" size="small" @click="pausePreview">暂停预览</n-button>
                    <n-button size="small" @click="stopPreview">停止预览</n-button>
                    <n-select
                        v-model:value="store.player.playbackRate"
                        :options="rateOptions"
                        size="small"
                        class="rate-select"
                        @update:value="changePreviewRate"
                    />
                </div>
            </div>

            <div v-if="hasFile" class="section-card">
                <div class="subsection-title">跟弹练习</div>
                <div class="desc block-desc">跟弹会复用当前 MIDI 文件。练右手时左手声部自动播放，练左手时右手声部自动播放；如果只有一个音轨，则自动降级为单手练习。</div>

                <div class="practice-config-grid">
                    <div class="setting-row">
                        <div>
                            <div class="label">练习声部</div>
                            <div class="desc">多轨 MIDI 会按平均音高推断左右手。</div>
                        </div>
                        <n-select
                            v-model:value="practiceHand"
                            :options="handOptions"
                            size="small"
                            class="select"
                            :disabled="practicePlan.singleTrackOnly"
                            @update:value="rebuildPracticePlan"
                        />
                    </div>

                    <div class="setting-row">
                        <div>
                            <div class="label">区间练习</div>
                            <div class="desc">只影响跟弹步骤，不影响上面的整首预览。</div>
                        </div>
                        <div class="range-row">
                            <n-input-number v-model:value="intervalStart" :min="0" :max="maxDuration" :step="1" size="small" @update:value="rebuildPracticePlan" />
                            <span class="range-separator">到</span>
                            <n-input-number v-model:value="intervalEnd" :min="0" :max="maxDuration" :step="1" size="small" @update:value="rebuildPracticePlan" />
                        </div>
                    </div>

                    <div class="setting-row">
                        <div>
                            <div class="label">自动播放非练习声部</div>
                            <div class="desc">建议开启，这样单手练习时仍然能听到另一只手的声部。</div>
                        </div>
                        <n-switch v-model:value="autoPlayOtherHand" @update:value="rebuildPracticePlan" />
                    </div>
                </div>

                <div class="practice-summary-row">
                    <n-tag :bordered="false" :type="practicePlan.singleTrackOnly ? 'warning' : 'info'">
                        {{ practicePlan.singleTrackOnly ? '单轨 MIDI：单手练习' : '多轨 MIDI：支持左右手' }}
                    </n-tag>
                    <span>{{ practicePlan.steps.length }} 个步骤 · {{ practicePlan.totalPracticeNotes || 0 }} 个练习音 · {{ practicePlan.totalAutoPlayNotes || 0 }} 个自动播放音</span>
                </div>

                <div class="assignment-card" v-if="practicePlan.assignments.length">
                    <div class="track-title">左右手推断</div>
                    <div class="assignment-list">
                        <div class="assignment-item" v-for="item in practicePlan.assignments" :key="item.trackIndex">
                            <span>{{ item.trackName || `Track ${item.trackIndex + 1}` }}</span>
                            <span>{{ handText(item.hand) }} · 平均音高 {{ Math.round(item.averageMidi || 0) }}</span>
                        </div>
                    </div>
                </div>

                <div class="follow-actions">
                    <n-button type="primary" size="small" @click="startFollow" :disabled="!practicePlan.steps.length">开始跟弹</n-button>
                    <n-button size="small" @click="restartFollow" :disabled="!practicePlan.steps.length">重新开始当前练习</n-button>
                    <n-button size="small" @click="prevFollowStep" :disabled="!followRunning">上一步</n-button>
                    <n-button size="small" @click="nextFollowStep" :disabled="!followRunning">下一步</n-button>
                    <n-button size="small" @click="stopFollow">停止跟弹</n-button>
                </div>

                <div v-if="followRunning" class="current-step-card" :class="{success: stepSuccessVisible}">
                    <div class="group-header">
                        <span>第 {{ currentStepIndex + 1 }} / {{ practicePlan.steps.length }} 步</span>
                        <span>{{ formatTime(currentStep?.time) }}</span>
                    </div>

                    <transition name="success-pop">
                        <div v-if="stepSuccessVisible" class="step-success-badge">✓ 当前步骤完成</div>
                    </transition>

                    <div class="note-block">
                        <div class="note-block-title">需要你按下</div>
                        <div class="note-pill-row" v-if="currentStep?.practiceNotes?.length">
                            <span
                                v-for="note in currentStep.practiceNotes"
                                :key="note.id"
                                class="note-pill"
                                :class="{done: store.pressedKey[note.midi]}"
                            >
                                {{ note.name || note.midi }}
                            </span>
                        </div>
                        <div v-else class="desc">当前步骤没有练习音，会自动播放非练习声部后继续。</div>
                    </div>

                    <div class="note-block" v-if="currentStep?.autoPlayNotes?.length">
                        <div class="note-block-title">自动播放声部</div>
                        <div class="note-pill-row">
                            <span v-for="note in currentStep.autoPlayNotes" :key="note.id" class="note-pill auto">
                                {{ note.name || note.midi }}
                            </span>
                        </div>
                    </div>
                </div>
            </div>

            <div class="track-list" v-if="hasFile">
                <div class="track-title">轨道概览</div>
                <div class="track-item" v-for="track in visibleTracks" :key="track.index">
                    <span>{{ track.name || `Track ${track.index + 1}` }}</span>
                    <span>{{ track.noteCount }} notes</span>
                </div>
            </div>

            <div class="tip-box">
                当前页面已经把“播放”和“跟弹”合并：播放是预览 MIDI，跟弹是练习 MIDI。文件仍然由前端选择并上传给 Go，Go 负责解析、播放调度和练习计划生成。
            </div>
        </div>
    </section>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, ref, watch} from 'vue'
import {NButton, NInputNumber, NSelect, NSlider, NSwitch, NTag} from 'naive-ui'
import {PLAYER_STATUS} from '../../services/midiPlaybackController'
import {emitWindowBus} from '../../services/windowBus'
import {applyBackendPlayerState, applyParsedMidiToStore, readFileAsBase64} from '../../services/backendMidiService'

const store = inject('store')
const Keyboard = inject('Keyboard')
const fileInput = ref(null)

const previewProgressValue = ref(0)
const followRunning = ref(false)
const currentStepIndex = ref(0)
const practiceHand = ref('right')
const autoPlayOtherHand = ref(true)
const intervalStart = ref(0)
const intervalEnd = ref(0)
const practicePlan = ref(createEmptyPracticePlan())
const stepCompleted = ref(false)
const stepSuccessVisible = ref(false)

let autoAdvanceTimer = null
let stepSuccessTimer = null
let wrongTimers = new Map()

const rateOptions = [
    {label: '0.5x', value: 0.5},
    {label: '0.75x', value: 0.75},
    {label: '1.0x', value: 1},
    {label: '1.25x', value: 1.25},
    {label: '1.5x', value: 1.5},
    {label: '2.0x', value: 2},
]

const hasFile = computed(() => store.player.notes.length > 0)
const previewPlaying = computed(() => store.player.status === PLAYER_STATUS.PLAYING)
const visibleTracks = computed(() => store.player.tracks.slice(0, 8))
const maxDuration = computed(() => Math.max(0, Math.ceil(store.player.duration || 0)))
const currentStep = computed(() => practicePlan.value.steps[currentStepIndex.value])
const previewProgressPercent = computed(() => {
    if (!store.player.duration) return 0
    return Math.round((store.player.currentTime / store.player.duration) * 100)
})
const previewStatusText = computed(() => {
    const map = {
        [PLAYER_STATUS.IDLE]: '未导入',
        [PLAYER_STATUS.READY]: '可预览',
        [PLAYER_STATUS.PLAYING]: '预览中',
        [PLAYER_STATUS.PAUSED]: '已暂停',
        [PLAYER_STATUS.STOPPED]: '已停止',
        [PLAYER_STATUS.FINISHED]: '预览完成',
    }
    return map[store.player.status] || store.player.status
})
const handOptions = computed(() => {
    if (practicePlan.value.singleTrackOnly) {
        return [{label: '单手练习', value: 'single'}]
    }
    return [
        {label: '双手都练', value: 'both'},
        {label: '只练右手，左手自动播放', value: 'right'},
        {label: '只练左手，右手自动播放', value: 'left'},
    ]
})

watch(() => store.player.currentTime, (value) => {
    previewProgressValue.value = Number(value || 0)
})

watch(() => store.player.notes, async () => {
    intervalStart.value = 0
    intervalEnd.value = Math.ceil(store.player.duration || 0)
    await rebuildPracticePlan()
}, {immediate: true, deep: true})

watch(() => ({...store.pressedKey}), () => {
    if (!followRunning.value || !currentStep.value || stepCompleted.value) return
    markWrongPressedKeys()
    if (!currentStep.value.practiceNotes.length) return

    const requiredKeys = currentStep.value.practiceNotes.map((note) => note.midi)
    const completed = requiredKeys.every((key) => store.pressedKey[key])
    if (completed) {
        completeCurrentStep()
    }
}, {deep: true})

function chooseFile() {
    fileInput.value?.click()
}

async function handleFileChange(event) {
    const file = event.target.files?.[0]
    if (!file) return

    try {
        // 重新导入 MIDI 时必须强制停止旧预览和旧跟弹，避免旧的自动伴奏、提示键或播放高亮残留。
        await forceStopCurrentPractice()
        store.player.error = ''
        practicePlan.value = createEmptyPracticePlan()

        const encoded = await readFileAsBase64(file)
        const parsed = await Keyboard.LoadMidiFileBase64(file.name, encoded)
        applyParsedMidiToStore(store, parsed)
        previewProgressValue.value = 0
        await rebuildPracticePlan()
    } catch (error) {
        console.error(error)
        store.player.error = `MIDI 文件解析失败：${error?.message || error}`
    } finally {
        event.target.value = ''
    }
}

async function playPreview() {
    await stopFollow()
    applyBackendPlayerState(store, await Keyboard.StartMidiPlayback())
}

async function pausePreview() {
    applyBackendPlayerState(store, await Keyboard.PauseMidiPlayback())
}

async function stopPreview() {
    applyBackendPlayerState(store, await Keyboard.StopMidiPlayback())
}

async function seekPreview() {
    applyBackendPlayerState(store, await Keyboard.SeekMidiPlayback(previewProgressValue.value))
}

async function changePreviewRate(rate) {
    applyBackendPlayerState(store, await Keyboard.SetMidiPlaybackRate(rate))
}

async function rebuildPracticePlan() {
    const notes = store.player.notes || []
    if (!notes.length) {
        practicePlan.value = createEmptyPracticePlan()
        return
    }

    const options = {
        threshold: 0.06,
        start: Number(intervalStart.value || 0),
        end: Number(intervalEnd.value || 0),
        practiceHand: practiceHand.value,
        autoPlayOtherHand: autoPlayOtherHand.value,
    }

    practicePlan.value = await Keyboard.BuildFollowPracticePlan(notes, options)
    if (practicePlan.value.singleTrackOnly) {
        practiceHand.value = 'single'
    }
}

async function startFollow() {
    if (!practicePlan.value.steps.length) return
    await stopPreview()
    clearAllPracticeVisualState()
    followRunning.value = true
    currentStepIndex.value = 0
    showCurrentFollowStep()
}

async function restartFollow() {
    if (!practicePlan.value.steps.length) return
    await stopPreview()
    followRunning.value = true
    currentStepIndex.value = 0
    await stopFollowAutoNotes()
    clearAllPracticeVisualState()
    showCurrentFollowStep()
}

async function stopFollow() {
    followRunning.value = false
    currentStepIndex.value = 0
    resetStepCompletionState()
    clearTimeout(autoAdvanceTimer)
    await stopFollowAutoNotes()
    clearAllPracticeVisualState()
}

function nextFollowStep() {
    if (!followRunning.value) return
    currentStepIndex.value++
    if (currentStepIndex.value >= practicePlan.value.steps.length) {
        stopFollow()
        return
    }
    showCurrentFollowStep()
}

function prevFollowStep() {
    if (!followRunning.value) return
    currentStepIndex.value = Math.max(0, currentStepIndex.value - 1)
    showCurrentFollowStep()
}

function showCurrentFollowStep() {
    clearTimeout(autoAdvanceTimer)
    stopFollowAutoNotes()
    clearPracticeHintsAndWrongKeys()
    resetStepCompletionState()

    const step = currentStep.value
    if (!step) return

    const practiceKeys = step.practiceNotes.map((note) => note.midi)
    for (const key of practiceKeys) {
        store.hintKey[key] = true
    }
    emitWindowBus('hint:set', {keys: practiceKeys})

    playFollowAutoNotes(step.autoPlayNotes || [])

    // 当前步骤如果只有自动伴奏，没有用户需要按的音，就按该步骤时长自动进入下一步。
    if (!step.practiceNotes.length) {
        const delay = Math.max(120, Math.min(900, (step.duration || 0.12) * 1000 / Math.max(store.player.playbackRate || 1, 0.1)))
        autoAdvanceTimer = setTimeout(() => nextFollowStep(), delay)
    }
}

function completeCurrentStep() {
    if (stepCompleted.value) return
    stepCompleted.value = true
    stepSuccessVisible.value = true

    // 完成后立刻清掉提示键，让用户感受到“命中成功”；稍等一小段时间再进入下一步。
    store.hintKey = {}
    emitWindowBus('hint:clear')

    clearTimeout(stepSuccessTimer)
    clearTimeout(autoAdvanceTimer)
    stepSuccessTimer = setTimeout(() => {
        stepSuccessVisible.value = false
        nextFollowStep()
    }, 220)
}

function resetStepCompletionState() {
    stepCompleted.value = false
    stepSuccessVisible.value = false
    clearTimeout(stepSuccessTimer)
}

function playFollowAutoNotes(notes) {
    if (!notes.length) return
    Keyboard.PlayFollowAutoNotes(notes, Number(store.player.playbackRate || 1))
}

async function stopFollowAutoNotes() {
    await Keyboard.StopFollowAutoNotes()
    store.playbackKey = {}
    emitWindowBus('playback:clear')
}

async function forceStopCurrentPractice() {
    await stopPreview()
    await stopFollow()
    await Keyboard.AllNotesOff()
    store.clearAllKeys()
    emitWindowBus('all:clear')
}

function markWrongPressedKeys() {
    const required = new Set((currentStep.value?.practiceNotes || []).map((note) => note.midi))
    for (const [key, pressed] of Object.entries(store.pressedKey)) {
        const midi = Number(key)
        if (!pressed || required.has(midi)) continue
        markWrongKey(midi)
    }
}

function markWrongKey(midi) {
    store.wrongKey[midi] = true
    emitWindowBus('wrong:key', {midi, active: true})
    clearTimeout(wrongTimers.get(midi))
    wrongTimers.set(midi, setTimeout(() => {
        store.wrongKey[midi] = false
        emitWindowBus('wrong:key', {midi, active: false})
        wrongTimers.delete(midi)
    }, 260))
}

function clearPracticeHintsAndWrongKeys() {
    store.hintKey = {}
    store.wrongKey = {}
    emitWindowBus('hint:clear')
    for (const timer of wrongTimers.values()) {
        clearTimeout(timer)
    }
    wrongTimers.clear()
}

function clearAllPracticeVisualState() {
    clearPracticeHintsAndWrongKeys()
    store.playbackKey = {}
    emitWindowBus('playback:clear')
}

function createEmptyPracticePlan() {
    return {
        steps: [],
        assignments: [],
        availableHands: ['single'],
        practiceHand: 'single',
        singleTrackOnly: true,
        start: 0,
        end: 0,
        autoPlayOtherHand: true,
        totalPracticeNotes: 0,
        totalAutoPlayNotes: 0,
    }
}

function handText(hand) {
    const map = {single: '单手', left: '左手', right: '右手', both: '双手'}
    return map[hand] || hand
}

function formatTime(seconds) {
    const safeSeconds = Math.max(0, Number(seconds || 0))
    const minutes = Math.floor(safeSeconds / 60)
    const secs = Math.floor(safeSeconds % 60)
    return `${minutes}:${String(secs).padStart(2, '0')}`
}

onBeforeUnmount(() => {
    stopPreview()
    stopFollow()
})
</script>

<style lang="scss" scoped>
.practice-card {
    min-height: 520px;
}

.content-card {
    min-height: 520px;
    background: rgba(255, 255, 255, 0.76);
    border: 1px solid rgba(148, 163, 184, 0.22);
    border-radius: 24px;
    padding: 24px;
    box-shadow: 0 18px 50px rgba(15, 23, 42, 0.08);
    backdrop-filter: blur(18px);
    box-sizing: border-box;
}

.section-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 20px;
}

.section-title {
    font-size: 18px;
    font-weight: 750;
    color: #0f172a;
}

.section-desc,
.desc {
    margin-top: 4px;
    font-size: 12px;
    line-height: 1.6;
    color: #64748b;
}

.practice-layout {
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.flow-card,
.import-card,
.file-info-card,
.section-card,
.track-list,
.tip-box {
    border-radius: 18px;
    border: 1px solid rgba(15, 23, 42, 0.08);
    background: rgba(255, 255, 255, 0.82);
}

.flow-card {
    display: grid;
    grid-template-columns: 1fr 34px 1fr 34px 1fr;
    align-items: center;
    padding: 14px;
}

.flow-step {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px;
    border-radius: 14px;
    color: #64748b;
}

.flow-step.active {
    color: #1d4ed8;
    background: rgba(37, 99, 235, 0.08);
}

.flow-step span {
    width: 28px;
    height: 28px;
    border-radius: 999px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 13px;
    font-weight: 800;
    background: rgba(148, 163, 184, 0.18);
}

.flow-step strong {
    display: block;
    font-size: 13px;
}

.flow-step p {
    margin: 2px 0 0;
    font-size: 11px;
}

.flow-line {
    height: 1px;
    background: rgba(148, 163, 184, 0.32);
}

.import-card,
.file-info-card,
.control-row,
.practice-summary-row,
.group-header,
.assignment-item,
.track-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
}

.import-card,
.file-info-card,
.section-card,
.track-list,
.tip-box {
    padding: 18px;
}

.import-icon {
    width: 44px;
    height: 44px;
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
    color: #1d4ed8;
    background: rgba(219, 234, 254, 0.9);
}

.import-body {
    flex: 1;
}

.label,
.subsection-title,
.track-title,
.note-block-title {
    font-size: 14px;
    font-weight: 700;
    color: #1e293b;
}

.subsection-title {
    margin-bottom: 6px;
    font-size: 15px;
}

.block-desc {
    margin-bottom: 14px;
}

.hidden-input {
    display: none;
}

.file-name {
    font-size: 15px;
    font-weight: 750;
    color: #0f172a;
}

.error-box {
    padding: 12px 14px;
    border-radius: 14px;
    color: #b91c1c;
    background: rgba(254, 226, 226, 0.82);
}

.progress-block {
    margin-bottom: 12px;
}

.time-row {
    display: flex;
    justify-content: space-between;
    margin-bottom: 8px;
    font-size: 12px;
    color: #64748b;
}

.rate-select,
.select {
    max-width: 280px;
}

.practice-config-grid {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-top: 14px;
}

.setting-row {
    display: grid;
    grid-template-columns: 250px 1fr;
    align-items: center;
    gap: 18px;
}

.range-row {
    display: flex;
    align-items: center;
    gap: 10px;
    max-width: 360px;
}

.range-separator {
    font-size: 12px;
    color: #64748b;
}

.practice-summary-row {
    justify-content: flex-start;
    margin-top: 14px;
    font-size: 12px;
    color: #475569;
}

.assignment-card,
.current-step-card {
    margin-top: 14px;
    padding: 16px;
    border-radius: 16px;
    background: rgba(248, 250, 252, 0.86);
}

.assignment-list,
.note-pill-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.assignment-list {
    flex-direction: column;
}

.assignment-item {
    padding: 10px 12px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.86);
    font-size: 12px;
    color: #475569;
}

.follow-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 14px;
}

.group-header {
    margin-bottom: 14px;
    color: #334155;
    font-weight: 700;
}

.note-block + .note-block {
    margin-top: 14px;
}

.note-pill {
    min-width: 42px;
    padding: 8px 12px;
    border-radius: 999px;
    text-align: center;
    font-weight: 700;
    color: #4338ca;
    background: rgba(224, 231, 255, 0.9);
}

.note-pill.done {
    color: #047857;
    background: rgba(209, 250, 229, 0.95);
}

.note-pill.auto {
    color: #0369a1;
    background: rgba(186, 230, 253, 0.85);
}

.track-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.track-item {
    padding: 10px 12px;
    border-radius: 12px;
    background: rgba(248, 250, 252, 0.86);
    color: #475569;
    font-size: 12px;
}

.tip-box {
    color: #475569;
    font-size: 13px;
    line-height: 1.7;
    background: rgba(37, 99, 235, 0.07);
}

.current-step-card.success {
    border-color: rgba(34, 197, 94, 0.38);
    box-shadow: 0 14px 32px rgba(34, 197, 94, 0.14);
}

.step-success-badge {
    display: inline-flex;
    align-items: center;
    align-self: flex-start;
    margin-bottom: 12px;
    padding: 7px 11px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 800;
    color: #15803d;
    background: rgba(220, 252, 231, 0.95);
    border: 1px solid rgba(34, 197, 94, 0.22);
}

.success-pop-enter-active,
.success-pop-leave-active {
    transition: 180ms ease;
}

.success-pop-enter-from,
.success-pop-leave-to {
    opacity: 0;
    transform: translateY(-4px) scale(0.98);
}

</style>
