// MIDI 播放状态常量。
// 真实解析和播放调度都由 Go / Wails 服务完成，前端只使用这些状态值做 UI 渲染。

export const PLAYER_STATUS = {
    IDLE: 'idle',
    READY: 'ready',
    PLAYING: 'playing',
    PAUSED: 'paused',
    STOPPED: 'stopped',
    FINISHED: 'finished',
}

export const PLAYER_MODE = {
    PLAYBACK: 'playback',
    FOLLOW: 'follow',
}

export function createInitialPlayerState() {
    return {
        fileName: '',
        duration: 0,
        notes: [],
        tracks: [],
        status: PLAYER_STATUS.IDLE,
        mode: PLAYER_MODE.PLAYBACK,
        currentTime: 0,
        playbackRate: 1,
        totalNotes: 0,
        error: '',
    }
}
