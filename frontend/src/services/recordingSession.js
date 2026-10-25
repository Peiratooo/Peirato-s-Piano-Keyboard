import {computed, reactive, ref} from 'vue'
import {Events} from '@wailsio/runtime'

const emptyState = {status: 'idle', draftId: '', durationMs: 0, noteCount: 0, error: '', backupSaved: false}

function recordingErrorMessage(error) {
    let message = error
    for (let depth = 0; depth < 4; depth++) {
        if (message && typeof message === 'object') {
            message = message.message || message.error || '操作未完成，请重试'
            continue
        }
        message = String(message || '操作未完成，请重试').replace(/^Error:\s*/, '')
        try {
            const details = JSON.parse(message)
            if (details && typeof details === 'object' && (details.message || details.error)) {
                message = details.message || details.error
                continue
            }
        } catch { /* Plain text is already suitable for display. */ }
        break
    }
    return String(message).replace(/Access is denied\.?|permission denied\.?/gi,
        '无法写入目标位置，请检查路径或权限后重试。')
}

export function useRecordingSession({Keyboard, onSaved, getPlayerState, canStart = () => true}) {
    const state = ref({...emptyState})
    const name = ref('')
    const durationMs = ref(0)
    const busy = ref(false)
    const previewing = ref(false)
    const available = computed(() => canStart())
    let clockStart = 0
    let ticker = null
    let pollCount = 0
    let refreshVersion = 0
    let disposed = false
    let previewStartTime = 0
    let unsubscribeExit = null

    function applyState(next) {
        if (disposed || !next) return
        state.value = {...emptyState, ...next, error: next.error ? recordingErrorMessage(next.error) : ''}
        durationMs.value = Number(next.durationMs) || 0
        if (next.status === 'recording') clockStart = Date.now() - durationMs.value
        if (next.status !== 'draft') previewing.value = false
    }

    function report(message, error) {
        const detail = recordingErrorMessage(error)
        window.$notify?.error?.(message, detail)
        if (!window.$notify) window.$message?.error(message + '：' + detail)
    }

    async function refresh() {
        const version = ++refreshVersion
        const current = await Keyboard.GetRecordingState()
        if (version === refreshVersion) applyState(current)
        return current
    }

    async function initialize() {
        if (!unsubscribeExit) unsubscribeExit = Events.On('recordingExitBlocked', event => {
            applyState(event.data)
            report('草稿备份失败，已保留窗口', event.data?.error || '请先保存 MIDI 后退出')
        })
        try { await refresh() } catch (error) { report('无法读取录制状态', error) }
        if (ticker) return
        ticker = setInterval(() => {
            if (disposed) return
            if (state.value.status === 'recording') {
                durationMs.value = Math.max(durationMs.value, Date.now() - clockStart)
            }
            if (previewing.value && Date.now() - previewStartTime > 700) {
                const player = getPlayerState?.()
                if (player?.midiId?.startsWith('recording:') && player.status === 'idle') {
                    previewing.value = false
                }
            }
            pollCount++
            if (pollCount % 4 === 0 && state.value.status === 'recording' && !busy.value) {
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
            try { await refresh() } catch {}
            report(typeof title === 'function' ? title() : title, error)
            return false
        } finally {
            busy.value = false
        }
    }

    async function start() {
        if (!available.value) return false
        return operation('无法开始录制', async () => {
            if (!available.value) return
            await Keyboard.StartRecording()
            previewing.value = false
        })
    }
    async function stop() {
        if (state.value.status !== 'recording'
            && !(state.value.status === 'draft' && !state.value.backupSaved)) return true
        return operation(() => state.value.status === 'draft' ? '录制已停止，草稿尚未备份' : '无法停止录制', async () => {
            await Keyboard.StopRecording()
        })
    }
    async function preview() {
        return operation('无法试听录制', async () => {
            await Keyboard.PreviewRecording()
            previewStartTime = Date.now()
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
        ++refreshVersion
        if (ticker) clearInterval(ticker)
        ticker = null
        unsubscribeExit?.()
        unsubscribeExit = null
    }
    return reactive({state, name, durationMs, busy, previewing, available, initialize, refresh,
        start, stop, preview, stopPreview, save, discard, dispose})
}
