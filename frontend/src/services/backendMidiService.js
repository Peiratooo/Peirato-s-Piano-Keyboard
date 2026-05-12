// Go / Wails 服务数据适配层。
// 注意：当前项目已进入“正式绑定模式”，前端不再保留前端解析或前端播放器回退实现。
// 修改 Go 暴露方法后，请先执行 `wails3 generate bindings`，再运行前端构建。

export function readFileAsBase64(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = () => resolve(String(reader.result || ''))
        reader.onerror = () => reject(reader.error || new Error('读取文件失败'))
        reader.readAsDataURL(file)
    })
}

export function applyParsedMidiToStore(store, parsedFile) {
    const notes = parsedFile?.notes || []
    const tracks = parsedFile?.tracks || []
    store.player = {
        ...store.player,
        fileName: parsedFile?.name || '',
        duration: Number(parsedFile?.duration || 0),
        notes,
        tracks,
        totalNotes: Number(parsedFile?.totalNotes ?? notes.length),
        currentTime: 0,
        status: 'ready',
        error: '',
    }
}

export function applyBackendPlayerState(store, state = {}) {
    store.player = {
        ...store.player,
        fileName: state.fileName ?? store.player.fileName,
        duration: Number(state.duration ?? store.player.duration ?? 0),
        currentTime: Number(state.currentTime ?? store.player.currentTime ?? 0),
        playbackRate: Number(state.playbackRate ?? store.player.playbackRate ?? 1),
        status: state.status || store.player.status,
        totalNotes: Number(state.totalNotes ?? store.player.totalNotes ?? 0),
        error: state.error || '',
    }
}
