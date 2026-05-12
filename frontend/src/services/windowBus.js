// windowBus 是主窗口和设置中心窗口之间的轻量通信层。
// 两个窗口是两个 WebView 上下文，Pinia 状态不会天然共享，所以播放高亮、跟弹提示这类实时 UI 状态，
// 统一通过 BroadcastChannel 同步。BroadcastChannel 不可用时，会退化为 localStorage 事件。

const CHANNEL_NAME = 'peirato-piano-window-bus'
const STORAGE_KEY = '__peirato_piano_window_bus__'

const channel = typeof BroadcastChannel !== 'undefined'
    ? new BroadcastChannel(CHANNEL_NAME)
    : null

export function emitWindowBus(type, payload = {}) {
    const message = {
        type,
        payload,
        createdAt: Date.now(),
        random: Math.random(),
    }

    if (channel) {
        channel.postMessage(message)
        return
    }

    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(message))
    } catch (error) {
        console.warn('窗口消息发送失败:', error)
    }
}

export function subscribeWindowBus(handler) {
    if (channel) {
        const listener = (event) => handler(event.data)
        channel.addEventListener('message', listener)
        return () => channel.removeEventListener('message', listener)
    }

    const listener = (event) => {
        if (event.key !== STORAGE_KEY || !event.newValue) return
        try {
            handler(JSON.parse(event.newValue))
        } catch (error) {
            console.warn('窗口消息解析失败:', error)
        }
    }

    window.addEventListener('storage', listener)
    return () => window.removeEventListener('storage', listener)
}
