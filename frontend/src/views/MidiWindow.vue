<template>
    <main class="midi-window">
        <aside class="library">
            <div class="titlebar">
                <div>
                    <div class="title">MIDI 中心</div>
                    <div class="subtitle">{{ midiItems.length }} 个文件</div>
                </div>
                <button class="primary" @click="importMidi">导入</button>
            </div>

            <div class="midi-list">
                <button
                    v-for="item in midiItems"
                    :key="item.id"
                    class="midi-item"
                    :class="{selected: item.id === selectedId}"
                    @click="selectMidi(item)"
                >
                    <span>{{ item.name }}</span>
                    <small>{{ formatDuration(item.durationMs) }}</small>
                </button>
            </div>
        </aside>

        <section class="workspace">
            <div class="controls">
                <div class="segmented">
                    <button :class="{active: options.mode === 'play'}" @click="options.mode = 'play'">播放</button>
                    <button :class="{active: options.mode === 'follow'}" @click="options.mode = 'follow'">练习</button>
                </div>

                <div class="segmented">
                    <button :class="{active: options.handMode === 'left'}" @click="options.handMode = 'left'">左手</button>
                    <button :class="{active: options.handMode === 'right'}" @click="options.handMode = 'right'">右手</button>
                    <button :class="{active: options.handMode === 'both'}" @click="options.handMode = 'both'">双手</button>
                </div>

                <label class="field">
                    <span>速度</span>
                    <input type="range" min="0.25" max="3" step="0.05" v-model.number="options.speed" @input="setSpeed">
                    <b>{{ options.speed.toFixed(2) }}x</b>
                </label>
            </div>

            <div class="transport">
                <button class="transport-button" :disabled="!selectedId" @click="startPlayback">开始</button>
                <button class="transport-button" :disabled="playerState.status !== 'playing'" @click="pausePlayback">暂停</button>
                <button class="transport-button" :disabled="playerState.status !== 'paused'" @click="resumePlayback">继续</button>
                <button class="transport-button" :disabled="playerState.status === 'idle'" @click="stopPlayback">停止</button>
                <button class="danger" :disabled="!selectedId" @click="removeSelected">移除</button>
            </div>

            <div class="progress">
                <input
                    type="range"
                    min="0"
                    :max="durationMs"
                    step="10"
                    :value="playerState.currentMs || 0"
                    :disabled="!selectedId"
                    @change="seekPlayback"
                >
                <div class="time-row">
                    <span>{{ formatDuration(playerState.currentMs || 0) }}</span>
                    <span>{{ statusText }}</span>
                    <span>{{ formatDuration(durationMs) }}</span>
                </div>
            </div>

            <div class="range-grid">
                <label class="field">
                    <span>左锚点</span>
                    <input type="number" min="0" :max="durationMs" step="100" v-model.number="options.leftMs" @change="setRange">
                </label>
                <label class="field">
                    <span>右锚点</span>
                    <input type="number" min="0" :max="durationMs" step="100" v-model.number="options.rightMs" @change="setRange">
                </label>
                <label class="field">
                    <span>提前判定</span>
                    <input type="number" min="20" max="500" step="10" v-model.number="options.leadWindowMs">
                </label>
                <label class="field">
                    <span>同显窗口</span>
                    <input type="number" min="20" max="500" step="10" v-model.number="options.groupWindowMs">
                </label>
            </div>
        </section>
    </main>
</template>

<script setup>
import {computed, inject, onMounted, reactive, ref, watch} from 'vue'

const store = inject('store')
const Keyboard = inject('Keyboard')

const midiItems = ref([])
const selectedId = ref('')

const options = reactive({
    midiId: '',
    mode: 'play',
    handMode: 'both',
    speed: 1,
    leftMs: 0,
    rightMs: 0,
    leadWindowMs: 80,
    groupWindowMs: 80,
})

const playerState = computed(() => store.midiPlayerState || {})
const selectedMidi = computed(() => midiItems.value.find((item) => item.id === selectedId.value))
const durationMs = computed(() => selectedMidi.value?.durationMs || playerState.value.durationMs || 0)
const statusText = computed(() => {
    if (playerState.value.waiting) return '等待按键'
    if (playerState.value.status === 'playing') return options.mode === 'follow' ? '练习中' : '播放中'
    if (playerState.value.status === 'paused') return '已暂停'
    return '空闲'
})

async function refreshMidiStore() {
    midiItems.value = await Keyboard.GetMidiStore()
    if (!selectedId.value && midiItems.value.length > 0) {
        await selectMidi(midiItems.value[0])
    }
}

async function importMidi() {
    const item = await Keyboard.OpenMidiFileDialog()
    await refreshMidiStore()
    if (item?.id) {
        await selectMidi(item)
    }
}

async function selectMidi(item) {
    selectedId.value = item.id
    const loaded = await Keyboard.LoadMidiByID(item.id)
    options.midiId = loaded.id
    options.leftMs = 0
    options.rightMs = loaded.durationMs || 0
}

async function removeSelected() {
    if (!selectedId.value) return
    await stopPlayback()
    await Keyboard.RemoveMidiByID(selectedId.value)
    selectedId.value = ''
    await refreshMidiStore()
}

async function startPlayback() {
    if (!selectedId.value) return
    options.midiId = selectedId.value
    if (!options.rightMs) {
        options.rightMs = durationMs.value
    }
    await Keyboard.StartMidiPlayback({...options})
}

async function pausePlayback() {
    await Keyboard.PauseMidiPlayback()
}

async function resumePlayback() {
    await Keyboard.ResumeMidiPlayback()
}

async function stopPlayback() {
    await Keyboard.StopMidiPlayback()
}

async function seekPlayback(event) {
    await Keyboard.SeekMidiPlayback(Number(event.target.value))
}

async function setSpeed() {
    if (playerState.value.status !== 'idle') {
        await Keyboard.SetMidiPlaybackSpeed(options.speed)
    }
}

async function setRange() {
    if (options.rightMs <= options.leftMs) {
        options.rightMs = Math.min(durationMs.value, options.leftMs + 100)
    }
    if (playerState.value.status !== 'idle') {
        await Keyboard.SetMidiPlaybackRange(options.leftMs, options.rightMs)
    }
}

function formatDuration(ms = 0) {
    const totalSeconds = Math.max(0, Math.floor(ms / 1000))
    const minutes = Math.floor(totalSeconds / 60)
    const seconds = String(totalSeconds % 60).padStart(2, '0')
    return `${minutes}:${seconds}`
}

watch(durationMs, (value) => {
    if (!options.rightMs && value > 0) {
        options.rightMs = value
    }
})

onMounted(async () => {
    store.midiWindowOpen = true
    await refreshMidiStore()
})
</script>

<style lang="scss" scoped>
.midi-window {
    display: grid;
    grid-template-columns: 320px 1fr;
    width: 100vw;
    height: 100vh;
    background: #f5f7fb;
    color: #172033;
    overflow: hidden;
}

.library {
    border-right: 1px solid #d7dde8;
    background: #ffffff;
    min-width: 0;
}

.titlebar,
.controls,
.transport,
.time-row,
.range-grid {
    display: flex;
    align-items: center;
}

.titlebar {
    justify-content: space-between;
    gap: 16px;
    height: 72px;
    padding: 0 18px;
    border-bottom: 1px solid #e3e8f0;
    --wails-draggable: drag;
}

.title {
    font-size: 18px;
    font-weight: 760;
}

.subtitle {
    margin-top: 4px;
    font-size: 12px;
    color: #64748b;
}

.midi-list {
    height: calc(100vh - 72px);
    overflow: auto;
    padding: 10px;
}

.midi-item {
    width: 100%;
    border: 1px solid transparent;
    border-radius: 8px;
    background: transparent;
    padding: 11px 12px;
    text-align: left;
    cursor: pointer;
    display: flex;
    justify-content: space-between;
    gap: 12px;
    color: #1f2937;
}

.midi-item:hover {
    background: #eef3fa;
}

.midi-item.selected {
    border-color: #93b4df;
    background: #e7f0fb;
}

.midi-item span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.midi-item small {
    color: #64748b;
}

.workspace {
    padding: 22px 24px;
    min-width: 0;
}

.controls {
    justify-content: space-between;
    gap: 18px;
}

.segmented {
    display: inline-flex;
    border: 1px solid #ccd6e4;
    border-radius: 8px;
    overflow: hidden;
    background: #ffffff;
}

.segmented button {
    border: 0;
    border-right: 1px solid #ccd6e4;
    background: transparent;
    padding: 8px 14px;
    cursor: pointer;
    color: #334155;
}

.segmented button:last-child {
    border-right: 0;
}

.segmented button.active {
    background: #245a8f;
    color: #ffffff;
}

.field {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: #475569;
}

.field input[type="number"] {
    width: 110px;
    height: 32px;
    box-sizing: border-box;
    border: 1px solid #cbd5e1;
    border-radius: 6px;
    padding: 0 8px;
}

.transport {
    margin-top: 28px;
    gap: 10px;
}

button {
    font: inherit;
}

.primary,
.transport-button,
.danger {
    height: 34px;
    border: 1px solid #c4cfdd;
    border-radius: 8px;
    padding: 0 14px;
    background: #ffffff;
    color: #172033;
    cursor: pointer;
}

.primary,
.transport-button:first-child {
    border-color: #245a8f;
    background: #245a8f;
    color: #ffffff;
}

.danger {
    color: #a83232;
}

button:disabled {
    opacity: 0.45;
    cursor: not-allowed;
}

.progress {
    margin-top: 28px;
}

.progress input[type="range"] {
    width: 100%;
}

.time-row {
    justify-content: space-between;
    margin-top: 8px;
    font-size: 13px;
    color: #64748b;
}

.range-grid {
    margin-top: 24px;
    display: grid;
    grid-template-columns: repeat(4, minmax(120px, 1fr));
    gap: 14px;
}
</style>
