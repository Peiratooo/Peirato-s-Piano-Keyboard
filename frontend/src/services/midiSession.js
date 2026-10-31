import {computed, reactive, ref, shallowRef, watch} from 'vue'

const clamp = (value, min, max) => Math.min(Math.max(Number(value) || 0, min), Math.max(min, max))
export function formatMidiTime(ms = 0) {
    const seconds = Math.max(0, Math.floor(ms / 1000))
    return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`
}

function midiError(error) {
    const message = String(error?.message || error || '未知错误')
    if (/不存在|无法访问|no such file|cannot find|access.*denied/i.test(message)) return '文件不存在或无法访问，请重新导入。'
    return message.replace(/[A-Za-z]:[\\/][^\r\n"'<>]*/g, '所选文件')
        .replace(/\/(?:[^\s/]+\/)+[^\s:]*/g, '所选文件')
}

const summarizeSong = song => ({id: song.id, name: song.name || '未命名曲目',
    durationMs: Math.max(0, Number(song.durationMs) || 0), trackCount: song.trackCount || 0, missing: !!song.missing})

// One session belongs to the main window, never to a transient library/settings
// panel. Commands are serialized; selection tokens discard stale file loads.
export function useMidiSession({store, Keyboard}) {
    const library = ref([])
    const selectedId = ref('')
    const timeline = shallowRef(null)
    const loading = ref(false)
    const importing = ref(false)
    const ready = ref(false)
    const busyCount = ref(0)
    const clockRevision = ref(0)
    const receivedAt = ref(performance.now())
    const localCurrent = ref(0)
    const pendingPosition = ref(null)
    const previewPosition = ref(null)
    const options = reactive({mode: store.mainMode === 'follow' ? 'follow' : 'play', handMode: 'both', speed: 1, leftMs: 0, rightMs: 0, loop: true})
    let selectionVersion = 0
    let disposed = false
    let queue = Promise.resolve()
    let initialization = null
    const mode = computed(() => store.mainMode || 'free')
    const selectedSong = computed(() => library.value.find(song => song.id === selectedId.value) || null)
    const durationMs = computed(() => timeline.value?.durationMs || selectedSong.value?.durationMs || 0)
    const matchingState = computed(() => store.midiPlayerState?.midiId === selectedId.value && selectedId.value
        ? store.midiPlayerState : null)
    const state = computed(() => ({
        midiId: selectedId.value,
        status: 'idle', mode: options.mode, handMode: options.handMode,
        durationMs: durationMs.value, currentMs: localCurrent.value,
        leftMs: options.leftMs, rightMs: options.rightMs || durationMs.value,
        speed: options.speed, loop: options.loop, waiting: false,
        ...(matchingState.value?.status === 'idle' ? {} : matchingState.value),
        ...(pendingPosition.value === null ? {} : {currentMs: pendingPosition.value, waiting: false}),
        ...(previewPosition.value === null ? {} : {currentMs: previewPosition.value, waiting: false,
            currentStep: null, leadInMs: 0, scrubbing: true}),
        clockRevision: clockRevision.value,
        receivedAt: receivedAt.value,
    }))
    const busy = computed(() => busyCount.value > 0)

    function notify(title, error) {
        window.$notify?.error?.(title, midiError(error))
        if (!window.$notify) window.$message?.error(`${title}：${midiError(error)}`)
    }
    function enqueue(action, title = 'MIDI 操作失败') {
        busyCount.value++
        const task = queue.then(async () => {
            if (disposed) return false
            return await action()
        }).catch(error => { if (!disposed) notify(title, error); return false })
            .finally(() => { busyCount.value-- })
        queue = task.then(() => undefined)
        return task
    }
    async function syncState() {
        const next = await Keyboard.GetMidiPlayerState()
        if (!disposed) store.midiPlayerState = {...next, currentStep: next.currentStep ?? null, error: next.error || ''}
        return next
    }
    function buildOptions(overrides = {}) {
        return {midiId: selectedId.value, ...options, rightMs: options.rightMs || durationMs.value,
            leadWindowMs: 80, groupWindowMs: 80, ...overrides}
    }
    async function loadLibrary() {
        const items = await Keyboard.GetMidiStore()
        if (!disposed) library.value = (items || []).map(summarizeSong)
    }
    async function loadSelection(id, version, preservePlayback = false) {
        if (version !== selectionVersion || disposed) return false
        if (selectedId.value === id && timeline.value) return true
        if (!preservePlayback && store.midiPlayerState?.midiId) {
            await Keyboard.StopMidiPlayback()
            await syncState()
        }
        if (version !== selectionVersion || disposed) return false
        const song = await Keyboard.LoadMidiByID(id)
        if (version !== selectionVersion || disposed) return false
        const notes = await Keyboard.GetMidiTimeline(id)
        if (version !== selectionVersion || disposed) return false
        const summary = summarizeSong(song)
        library.value = library.value.map(item => item.id === id ? summary : item)
        selectedId.value = id
        timeline.value = {...notes, midiId: notes.midiId || notes.id || id, notes: notes.notes || []}
        options.leftMs = 0
        options.rightMs = Number(notes.durationMs || song.durationMs) || 0
        localCurrent.value = 0
        if (preservePlayback && store.midiPlayerState?.midiId === id
            && ['playing', 'paused'].includes(store.midiPlayerState.status)) {
            const current = store.midiPlayerState
            for (const key of ['mode', 'handMode', 'speed', 'leftMs', 'rightMs', 'loop']) {
                if (current[key] !== undefined) options[key] = current[key]
            }
            localCurrent.value = current.currentMs || 0
        }
        pendingPosition.value = null
        previewPosition.value = null
        clockRevision.value++
        return true
    }
    function initialize() {
        if (initialization) return initialization
        const version = selectionVersion
        loading.value = true
        initialization = enqueue(async () => {
            const [, current] = await Promise.all([loadLibrary(), syncState()])
            if (version !== selectionVersion || disposed) return true
            const id = library.value.find(song => song.id === selectedId.value)?.id
                || library.value.find(song => song.id === current?.midiId)?.id || library.value[0]?.id
            if (id) await loadSelection(id, ++selectionVersion, true)
            if (current?.status === 'playing' || current?.status === 'paused') {
                store.mainMode = current.mode === 'follow' ? 'follow' : 'play'
            }
            return true
        }, '曲库加载失败').then(success => {
            if (!disposed) {
                ready.value = !!success
                if (!success) initialization = null
            }
            return success
        }).finally(() => { if (!disposed) loading.value = false })
        return initialization
    }
    function refreshLibrary() { return enqueue(async () => { await loadLibrary(); return true }, '曲库刷新失败') }
    function selectSong(id) {
        if (!id) return Promise.resolve(false)
        if (id === selectedId.value && timeline.value && !loading.value) return Promise.resolve(true)
        const version = ++selectionVersion
        loading.value = true
        return enqueue(() => {
            if (version === selectionVersion) loading.value = true
            return loadSelection(id, version)
        }, '曲目加载失败')
            .finally(() => { if (version === selectionVersion && !disposed) loading.value = false })
    }
    function importSong() {
        if (importing.value) return Promise.resolve(false)
        importing.value = true
        return enqueue(async () => {
            let song
            try { song = await Keyboard.OpenMidiFileDialog() } catch (error) {
                if (/cancel|取消|未选择/i.test(String(error))) return false
                throw error
            }
            if (!song?.id || disposed) return false
            await loadLibrary()
            const version = ++selectionVersion
            loading.value = true
            try { return await loadSelection(song.id, version) } finally { loading.value = false }
        }, 'MIDI 导入失败').finally(() => { importing.value = false })
    }
    function removeSong(id) {
        if (!id) return Promise.resolve(false)
        const version = id === selectedId.value ? ++selectionVersion : selectionVersion
        return enqueue(async () => {
            const removingSelected = selectedId.value === id
            if (removingSelected) {
                await Keyboard.StopMidiPlayback()
                await syncState()
            }
            await Keyboard.RemoveMidiByID(id)
            await loadLibrary()
            if (removingSelected) {
                selectedId.value = ''
                timeline.value = null
                localCurrent.value = 0
                options.leftMs = options.rightMs = 0
                clockRevision.value++
                const next = library.value[0]?.id
                if (next && version === selectionVersion) await loadSelection(next, ++selectionVersion)
            }
            return true
        }, '移除曲目失败')
    }
    function setMode(next) {
        if (!['free', 'play', 'follow'].includes(next)) return Promise.resolve(false)
        return enqueue(async () => {
            if (next === mode.value) return true
            if (next === 'free') {
                await Keyboard.StopMidiPlayback()
                await syncState()
                store.clearMidiVisualKeys?.()
                localCurrent.value = options.leftMs
            } else if (matchingState.value && ['playing', 'paused'].includes(matchingState.value.status)) {
                await Keyboard.SetMidiPlaybackOptions(buildOptions({mode: next}))
                await syncState()
            }
            if (next !== 'free') options.mode = next
            store.mainMode = next
            clockRevision.value++
            return true
        }, '切换演奏方式失败')
    }
    async function startBackend() {
        if (!selectedId.value || !timeline.value || loading.value || mode.value === 'free') return false
        options.mode = mode.value === 'follow' ? 'follow' : 'play'
        const rightMs = options.rightMs || durationMs.value
        const startMs = clamp(localCurrent.value, options.leftMs, rightMs)
        // A–B always starts immediately, even when A is at the song's start.
        // A seek into the full song also skips the initial countdown.
        const wholeSong = options.leftMs <= 0.5 && rightMs >= durationMs.value - 0.5
        const leadIn = wholeSong && startMs <= 0.5
        await Keyboard.StartMidiPlayback(buildOptions({startMs, leadIn}))
        clockRevision.value++
        await syncState()
        return true
    }
    function start() { return enqueue(startBackend, '开始演奏失败') }
    function togglePlayback() {
        return enqueue(async () => {
            if (loading.value || !selectedId.value || mode.value === 'free') return false
            if (matchingState.value?.status === 'playing') await Keyboard.PauseMidiPlayback()
            else if (matchingState.value?.status === 'paused') await Keyboard.ResumeMidiPlayback()
            else return startBackend()
            clockRevision.value++
            await syncState()
            return true
        }, '播放操作失败')
    }
    function stop() {
        return enqueue(async () => {
            await Keyboard.StopMidiPlayback()
            localCurrent.value = options.leftMs
            pendingPosition.value = null
            previewPosition.value = null
            clockRevision.value++
            await syncState()
            return true
        }, '停止演奏失败')
    }
    function seek(ms) {
        const id = selectedId.value
        const version = selectionVersion
        return enqueue(async () => {
            if (!id || id !== selectedId.value || version !== selectionVersion || mode.value === 'free') return false
            const target = clamp(ms, options.leftMs, options.rightMs || durationMs.value)
            pendingPosition.value = target
            clockRevision.value++
            try {
                if (matchingState.value && matchingState.value.status !== 'idle') {
                    await Keyboard.SeekMidiPlayback(target)
                    await syncState()
                }
                localCurrent.value = target
                return true
            } finally { pendingPosition.value = null }
        }, '跳转播放位置失败')
    }
    // Dragging previews locally. Only its final position seeks the MIDI output.
    function previewSeek(ms) {
        previewPosition.value = clamp(ms, options.leftMs, options.rightMs || durationMs.value)
        clockRevision.value++
        return previewPosition.value
    }
    function endSeekPreview() { previewPosition.value = null; clockRevision.value++ }
    function updateOptions(changes, title) {
        const id = selectedId.value
        const version = selectionVersion
        return enqueue(async () => {
            if (id !== selectedId.value || version !== selectionVersion) return false
            const next = {...options, ...changes}
            if (matchingState.value && ['playing', 'paused'].includes(matchingState.value.status)) {
                if (Object.keys(changes).length === 1 && changes.speed !== undefined) {
                    await Keyboard.SetMidiPlaybackSpeed(next.speed)
                } else {
                    await Keyboard.SetMidiPlaybackOptions(buildOptions(next))
                }
                await syncState()
            }
            Object.assign(options, next)
            localCurrent.value = clamp(localCurrent.value, options.leftMs, options.rightMs || durationMs.value)
            clockRevision.value++
            return true
        }, title)
    }
    function setSpeed(speed) { return updateOptions({speed: clamp(speed, .25, 5)}, '调整速度失败') }
    function setHand(handMode) {
        if (!['left', 'right', 'both'].includes(handMode)) return Promise.resolve(false)
        return updateOptions({handMode}, '调整演奏手失败')
    }
    function setLoop(loop) { return updateOptions({loop: !!loop}, '调整循环失败') }
    function setRange(left, right) {
        const duration = durationMs.value
        if (!duration) return Promise.resolve(false)
        // Keep short MIDI measures selectable without imposing a 100ms minimum.
        const gap = Math.min(1, duration)
        const leftMs = clamp(left, 0, duration - gap)
        const rightMs = clamp(right, leftMs + gap, duration)
        return updateOptions({leftMs, rightMs}, '调整练习区间失败')
    }
    const stopWatching = watch(() => store.midiPlayerState, (current, previous) => {
        receivedAt.value = performance.now()
        if (!selectedId.value || current?.midiId !== selectedId.value || loading.value) return
        if (current.status === 'idle' && previous?.status === 'idle' && current.midiId === previous.midiId) return
        localCurrent.value = Number(current.currentMs) || 0
        for (const key of ['mode', 'handMode', 'speed', 'leftMs', 'rightMs', 'loop']) {
            if (current[key] !== undefined) options[key] = current[key]
        }
    }, {flush: 'sync'})
    function dispose() {
        disposed = true
        previewPosition.value = null
        ++selectionVersion
        stopWatching()
    }
    return reactive({library, selectedId, selectedSong, timeline, loading, importing, ready, busy, mode,
        options, state, durationMs, clockRevision, initialize, refreshLibrary, selectSong, importSong,
        removeSong, setMode, start, togglePlayback, stop, seek, previewSeek, endSeekPreview,
        setSpeed, setHand, setLoop, setRange, dispose})
}
