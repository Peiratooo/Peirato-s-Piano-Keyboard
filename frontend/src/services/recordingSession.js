import {reactive, ref} from 'vue'

const emptyState = {status: 'idle', draftId: '', durationMs: 0, noteCount: 0, error: ''}

export function useRecordingSession({Keyboard, onSaved}) {
    const state = ref({...emptyState})
    const name = ref('')
    const durationMs = ref(0)
    const busy = ref(false)
    const previewing = ref(false)
    let clockStart = 0
    let ticker = null
    let pollCount = 0
    let disposed = false

    function applyState(next) {
        if (disposed || !next) return
        state.value = {...emptyState, ...next}
        durationMs.value = Number(next.durationMs) || 0
        if (next.status === 'recording') clockStart = Date.now() - durationMs.value
        if (next.status !== 'draft') previewing.value = false
    }

    function report(message, error) {
        window.$notify?.error?.(message, String(error))
        if (!window.$notify) window.$message?.error(message + '：' + String(error))
    }

    async function refresh() {
        const current = await Keyboard.GetRecordingState()
        applyState(current)
        return current
    }

    async function initialize() {
        try { await refresh() } catch (error) { report('无法读取录制状态', error) }
        if (ticker) return
        ticker = setInterval(() => {
            if (disposed) return
            if (state.value.status === 'recording') {
                durationMs.value = Math.max(durationMs.value, Date.now() - clockStart)
            }
            pollCount++
            if (pollCount % 4 === 0 && state.value.status === 'recording') {
                refresh().catch(error => report('录制状态同步失败', error))
            }
        }, 500)
    }

    async function operation(title, fn) {
        if (busy.value) return false
        busy.value = true
        try {
            await fn()
            await refresh()
            return true
        } catch (error) {
            report(title, error)
            try { await refresh() } catch {}
            return false
        } finally {
            busy.value = false
        }
    }

    async function start() {
        return operation('无法开始录制', async () => {
            await Keyboard.StartRecording()
            previewing.value = false
        })
    }
    async function stop() {
        if (state.value.status !== 'recording') return true
        return operation('无法停止录制', async () => {
            await Keyboard.StopRecording()
        })
    }
    async function preview() {
        return operation('无法试听录制', async () => {
            await Keyboard.PreviewRecording()
            previewing.value = true
        })
    }
    async function stopPreview() {
        return operation('无法停止试听', async () => {
            await Keyboard.StopRecordingPreview()
            previewing.value = false
        })
    }
    async function save() {
        if (busy.value || state.value.status !== 'draft') return false
        busy.value = true
        try {
            const song = await Keyboard.SaveRecording(name.value)
            await refresh()
            previewing.value = false
            name.value = ''
            try { await onSaved?.(song) } catch (error) { report('已保存，但曲库刷新失败', error) }
            window.$notify?.success?.('录制已保存', '文件已加入 MIDI 曲库，可用于播放和跟弹。')
            return true
        } catch (error) {
            const message = String(error)
            if (!/取消|cancel/i.test(message)) report('无法保存录制', error)
            try { await refresh() } catch {}
            return false
        } finally {
            busy.value = false
        }
    }
    async function discard() {
        return operation('无法放弃录制', async () => {
            await Keyboard.DiscardRecording()
            previewing.value = false
            name.value = ''
        })
    }
    function dispose() {
        disposed = true
        if (ticker) clearInterval(ticker)
        ticker = null
    }
    return reactive({state, name, durationMs, busy, previewing, initialize, refresh,
        start, stop, preview, stopPreview, save, discard, dispose})
}
