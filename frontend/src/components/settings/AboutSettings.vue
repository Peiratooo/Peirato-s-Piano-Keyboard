<template>
    <section class="content-card about-page">
        <div class="about-heading"><h2>关于软件</h2></div>
        <section class="update-card" aria-label="软件更新" :aria-busy="updater.busy">
            <div class="app-summary">
                <div class="app-identity">
                    <img class="app-logo" src="/app-icon.png" alt="Peirato Piano 标志" />
                    <div><h3>Peirato's Piano</h3><p>Keyboard · 桌面钢琴键盘</p><span class="version-tag">当前版本 {{ store.config.version || '—' }}</span></div>
                </div>
                <div class="update-action">
                    <span v-if="updater.phase === 'latest'" class="latest-badge"><svg viewBox="0 0 20 20" aria-hidden="true"><path d="m5 10 3 3 7-7" /></svg>已是最新版本</span>
                    <span v-else-if="updater.phase === 'files-current'" class="latest-badge">文件已是最新</span>
                    <span v-else-if="updater.phase === 'ready'" class="ready-label">新版本已准备好</span>
                    <span v-else-if="updater.available" class="new-label">发现新版本 {{ updater.version }}</span>
                    <span v-else-if="updater.phase === 'checking'" class="muted">正在检查更新…</span>
                    <n-button v-if="updater.phase === 'ready'" type="primary" color="#376be6" round @click="showInstall = true">立即更新</n-button>
                    <n-button v-else-if="updater.available && !updater.downloading && updater.phase !== 'installing'" type="primary" color="#376be6" round :loading="updater.requesting" :disabled="!updater.supported || updater.busy" @click="updater.download()">下载更新</n-button>
                    <n-button v-else-if="!updater.downloading && updater.phase !== 'installing'" quaternary size="small" :loading="updater.busy" @click="updater.check()">{{ updater.phase === 'error' ? '重新检测' : '检查更新' }}</n-button>
                </div>
            </div>
            <div class="update-detail" aria-live="polite" role="status">
                <template v-if="showProgress">
                    <div class="progress-caption"><strong>{{ progressTitle }}</strong><span>{{ progressDetail }}</span></div>
                    <n-progress v-if="updater.total > 0 && updater.phase === 'downloading'" type="line" :percentage="updater.percent" :show-indicator="false" :height="5" color="#376be6" rail-color="#e4eaf5" />
                    <div v-else-if="updater.phase === 'ready'" class="complete-track"><span /></div>
                    <div v-else class="indeterminate-track" role="progressbar" :aria-label="progressTitle"><span /></div>
                    <p>{{ updater.phase === 'ready' ? '下载与校验已完成。你可以准备好后再安装。' : updater.phase === 'installing' ? '正在准备安装，即将关闭软件；更新完成后会自动重新打开。' : '你可以继续使用软件，下载完成后再选择安装。' }}</p>
                </template>
                <div v-else-if="updater.phase === 'error'" class="friendly-error"><span class="error-icon">!</span><div><strong>暂时无法完成更新</strong><p>{{ updateErrorMessage(updater.error) }}</p></div><button v-if="updater.available" class="text-button" @click="updater.check()">重新检测</button></div>
                <p v-else-if="updater.available && !updater.supported">发现新版本，但更新组件不可用，请使用完整安装包修复软件。</p>
                <p v-else-if="updater.available">新版本 {{ updater.version }} 已可用。仅下载需要更新的文件，安装前会再次确认。</p>
                <p v-else-if="updater.phase === 'files-current'">本地文件已与 {{ updater.version }} 一致，无需下载安装。请退出并重新打开软件，确认当前运行版本。</p>
                <p v-else-if="updater.phase === 'latest'">当前版本已是最新，安心享受弹奏。</p>
                <p v-else>进入此页面会自动检查更新，无需手动操作。</p>
            </div>
        </section>

        <section class="author-section" aria-label="作者信息">
            <Author />
        </section>
        <section class="reset-section">
            <div><h3>恢复默认设置</h3><p>重新开始配置，让软件回到初始状态。</p></div>
            <n-button secondary round :disabled="updater.busy || resetting" @click="showReset = true">恢复默认</n-button>
        </section>

        <n-modal v-model:show="showInstall" preset="dialog" :style="{borderRadius:'18px'}" title="更新并重新启动？" positive-text="确定更新并重启" negative-text="稍后再说" :positive-button-props="{type:'primary',color:'#376be6'}" @positive-click="confirmInstall">
            <p class="dialog-copy">将安装 {{ updater.version }}。软件会关闭，完成更新后自动重新打开。</p>
            <p class="dialog-note">请先结束当前演奏或 MIDI 播放，确认准备好后再继续。</p>
        </n-modal>
        <n-modal v-model:show="showReset" preset="dialog" :style="{borderRadius:'18px'}" type="warning" title="恢复默认设置？" positive-text="确定恢复" negative-text="保留设置" :loading="resetting" :mask-closable="!resetting" :closable="!resetting" :close-on-esc="!resetting" :negative-button-props="{disabled:resetting}" @positive-click="confirmReset">
            <p class="dialog-copy">外观、键盘映射和音频设置将恢复默认，已添加的音源与 MIDI 列表记录也会清空。</p>
            <p class="dialog-note">不会删除电脑上的音源和 MIDI 原始文件。采样率与缓冲大小在重启后生效。</p>
        </n-modal>
    </section>
</template>

<script setup>
import {computed, inject, onMounted, ref} from 'vue'
import {NButton, NModal, NProgress} from 'naive-ui'
import Author from '../Author.vue'
import {useUpdater, updateErrorMessage} from '../../store/updater'
const store = inject('store')
const resetConfig = inject('resetConfig')
const updater = useUpdater()
const showInstall = ref(false)
const showReset = ref(false)
const resetting = ref(false)
const showProgress = computed(() => updater.downloading || ['ready','installing'].includes(updater.phase))
const progressTitle = computed(() => ({preparing:'正在准备更新', downloading:'正在下载更新', verifying:'正在校验更新文件', ready:'更新已就绪', installing:'正在准备安装'})[updater.phase] || '')
const size = value => `${(value / 1024 / 1024).toFixed(1)} MB`
const progressDetail = computed(() => updater.phase === 'ready' ? '下载完成' : updater.phase === 'downloading' ? (updater.total > 0 ? `${size(updater.received)} / ${size(updater.total)} · ${updater.percent}%` : `已下载 ${size(updater.received)}`) : '请稍候')
onMounted(() => { updater.check() })
function confirmInstall() { showInstall.value = false; updater.install() }
async function confirmReset() {
    if (resetting.value) return false
    resetting.value = true
    try {
        const ok = await resetConfig()
        if (!ok) return false
        window.$message?.success('已恢复默认设置')
        return true
    } finally { resetting.value = false }
}
</script>

<style scoped lang="scss">
.about-page { display:flex; flex-direction:column; gap:20px; }
.about-heading { display:flex; justify-content:space-between; align-items:baseline; gap:12px; }
.about-heading h2 { margin:0; font-size:18px; font-weight:700; }
.update-card { background:linear-gradient(120deg,#f1f5ff 0%,#f8faff 65%,#f5f9ff 100%); border:1px solid #dfe7f7; border-radius:18px; overflow:hidden; }
.app-summary { display:flex; justify-content:space-between; align-items:center; gap:20px; padding:22px 24px; }
.app-identity { display:flex; align-items:center; gap:16px; min-width:0; }
.app-logo { width:68px; height:68px; border-radius:0; box-shadow:0 5px 12px #1e293b16; flex-shrink:0; }
h3 { margin:0; font-size:19px; font-weight:700; color:#1e293b; letter-spacing:-.4px; }
.app-identity p { font-size:12px; color:#64748b; margin:5px 0 9px; }
.version-tag { display:inline-flex; padding:3px 8px; background:#fff; border:1px solid #e4eaf5; border-radius:6px; color:#52617a; font-size:11px; }
.update-action { display:flex; flex-direction:column; align-items:flex-end; gap:10px; flex-shrink:0; font-size:12px; }
.latest-badge { display:flex; align-items:center; gap:5px; color:#247653; font-weight:600; }
.latest-badge svg { width:18px; height:18px; stroke:currentColor; fill:none; stroke-width:1.8; stroke-linecap:round; stroke-linejoin:round; }
.ready-label,.new-label { color:#315ebc; font-weight:600; }
.muted { color:#64748b; }
.update-detail { border-top:1px solid #e0e7f3; padding:14px 24px; background:#ffffff6b; }
.update-detail p { margin:0; font-size:12px; line-height:1.7; color:#64748b; }
.progress-caption { display:flex; justify-content:space-between; gap:12px; font-size:12px; margin-bottom:12px; color:#64748b; font-variant-numeric:tabular-nums; }
.progress-caption strong { color:#334155; font-weight:600; }
.progress-caption ~ p { margin-top:9px; }
.indeterminate-track,.complete-track { height:5px; background:#e4eaf5; border-radius:5px; overflow:hidden; }
.indeterminate-track span { display:block; height:100%; width:35%; background:#376be6; border-radius:5px; animation:progress 1.6s ease-in-out infinite; }
.complete-track span { display:block; height:100%; width:100%; background:#3b9b76; }
@keyframes progress { from { transform:translateX(-100%); } to { transform:translateX(390%); } }
.friendly-error { display:flex; align-items:center; gap:10px; }
.friendly-error strong { font-size:12px; font-weight:600; color:#725b37; }
.friendly-error p { margin-top:3px; }
.error-icon { width:22px; height:22px; background:#fff1d6; color:#936623; border-radius:50%; display:grid; place-items:center; font-weight:700; flex-shrink:0; }
.text-button { margin-left:auto; background:none; border:0; color:#315ebc; white-space:nowrap; cursor:pointer; }
.author-section { padding:0 4px; }
.reset-section { display:flex; justify-content:space-between; align-items:center; gap:20px; border-top:1px solid #e8edf4; padding-top:20px; margin-top:auto; }
.reset-section h3 { font-size:13px; font-weight:600; letter-spacing:0; }
.reset-section p { font-size:12px; color:#64748b; margin:5px 0 0; }
.dialog-copy { color:#334155; line-height:1.8; }
.dialog-note { color:#64748b; font-size:12px; line-height:1.8; }
@media(max-width:650px) { .app-summary { align-items:flex-start; flex-direction:column; padding:20px; } .update-action { align-items:flex-start; } .update-detail { padding:14px 20px; } .reset-section { flex-wrap:wrap; } }
@media(prefers-reduced-motion:reduce) { .indeterminate-track span { animation:none; width:100%; opacity:.5; } }
</style>
