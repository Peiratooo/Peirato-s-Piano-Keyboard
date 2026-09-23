import {defineStore} from 'pinia'
import {Keyboard} from '../../bindings/main/service'

export function updateErrorMessage(error) {
    let message = error?.message || String(error || '')
    for (let i = 0; i < 2; i++) {
        try { const parsed = JSON.parse(message.replace(/^Error:\s*/, '')); message = parsed.message || message } catch { break }
    }
    if (/404|410|没有可用的更新信息/.test(message)) return '暂时无法获取更新信息，请稍后再试。'
    if (/updater|执行权限/.test(message)) return '更新组件不可用，请使用完整安装包修复软件。'
    if (/timeout|deadline|网络|dial tcp|connection|lookup|EOF/i.test(message)) return '无法连接更新服务，请检查网络后重试。'
    if (/校验/.test(message)) return '更新文件校验失败，请重新下载。'
    if (/HTTP 429/.test(message)) return '更新服务繁忙，请稍后重试。'
    if (/HTTP 5\d\d/.test(message)) return '更新服务暂时不可用，请稍后重试。'
    return message.replace(/^Error:\s*/, '') || '操作未完成，请重试。'
}

export const useUpdater = defineStore('updater', {
    state: () => ({revision: 0, phase: 'idle', version: '', releaseId: 0, available: false,
        supported: true, received: 0, total: 0, error: '', requesting: false}),
    getters: {
        busy: s => ['checking', 'preparing', 'downloading', 'verifying', 'installing'].includes(s.phase) || s.requesting,
        downloading: s => ['preparing', 'downloading', 'verifying'].includes(s.phase),
        percent: s => s.total > 0 ? Math.min(100, Math.floor(s.received / s.total * 100)) : 0,
    },
    actions: {
        apply(state) {
            if (!state || state.revision < this.revision) return
            const {revision, phase, version, releaseId, available, supported, received, total, error} = state
            Object.assign(this, {revision, phase, version, releaseId, available, supported, received, total, error})
        },
        async sync() { this.apply(await Keyboard.GetUpdateState()) },
        async check() {
            if (this.busy || this.phase === 'ready') return
            this.requesting = true
            try { await Keyboard.CheckUpdate(); await this.sync() }
            catch (error) { this.error = updateErrorMessage(error); this.phase = 'error' }
            finally { this.requesting = false }
        },
        async download() {
            if (this.busy) return
            this.requesting = true
            try { await Keyboard.DownloadUpdate(this.releaseId); await this.sync() }
            catch (error) { this.error = updateErrorMessage(error); this.phase = 'error' }
            finally { this.requesting = false }
        },
        async install() {
            if (this.busy || this.phase !== 'ready') return
            this.requesting = true
            try { await Keyboard.InstallUpdate() }
            catch (error) { this.error = updateErrorMessage(error); this.phase = 'error' }
            finally { this.requesting = false }
        },
    },
})
