<template>
    <section class="content-card about-page">
        <div class="about-heading"><h2>关于软件</h2></div>
        <section class="app-card" aria-label="应用信息">
            <div class="app-summary">
                <div class="app-identity">
                    <img class="app-logo" src="/appicon.png" :alt="APP_NAME + ' 标志'" />
                    <div><h3>{{ APP_NAME }}</h3><span class="version-tag">当前版本 {{ store.config.version || '—' }}</span><p class="full-name">{{ APP_FULL_NAME }} · 开源离线版</p></div>
                </div>
            </div>
        </section>

        <section class="author-section" aria-label="作者信息">
            <Author />
        </section>
        <section class="reset-section">
            <div><h3>恢复默认设置</h3><p>重新开始配置，让软件回到初始状态。</p></div>
            <n-button secondary round :disabled="resetting" @click="showReset = true">恢复默认</n-button>
        </section>

        <n-modal v-model:show="showReset" preset="dialog" :style="{borderRadius:'18px'}" type="warning" title="恢复默认设置？" positive-text="确定恢复" negative-text="保留设置" :loading="resetting" :mask-closable="!resetting" :closable="!resetting" :close-on-esc="!resetting" :negative-button-props="{disabled:resetting}" @positive-click="confirmReset">
            <p class="dialog-copy">外观、键盘映射和音频设置将恢复默认，已添加的音源与 MIDI 列表记录也会清空。</p>
            <p class="dialog-note">不会删除电脑上的音源和 MIDI 原始文件。采样率与缓冲大小在重启后生效。</p>
        </n-modal>
    </section>
</template>

<script setup>
import {inject, ref} from 'vue'
import {NButton, NModal} from 'naive-ui'
import Author from '../Author.vue'
import {APP_FULL_NAME, APP_NAME} from '../../services/appIdentity'
const store = inject('store')
const resetConfig = inject('resetConfig')
const showReset = ref(false)
const resetting = ref(false)
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
.app-card { background:var(--setting-subtle-bg); border:1px solid #e1e1e1; border-radius:18px; overflow:hidden; }
.app-summary { display:flex; justify-content:space-between; align-items:center; gap:20px; padding:22px 24px; }
.app-identity { display:flex; align-items:center; gap:16px; min-width:0; }
.app-logo { width:68px; height:68px; border-radius:0; box-shadow:0 5px 12px #00000016; flex-shrink:0; }
h3 { margin:0; font-size:19px; font-weight:700; color:#303030; letter-spacing:-.4px; }
.app-identity p { font-size:12px; color:#666666; margin:5px 0 9px; }
.app-identity .full-name { margin:9px 0 0; font-size:11px; }
.version-tag { display:inline-flex; padding:3px 8px; background:#fff; border:1px solid #e5e5e5; border-radius:6px; color:#626262; font-size:11px; }
.author-section { padding:0 4px; }
.reset-section { display:flex; justify-content:space-between; align-items:center; gap:20px; border-top:1px solid #e6e6e6; padding-top:20px; margin-top:auto; }
.reset-section h3 { font-size:13px; font-weight:600; letter-spacing:0; }
.reset-section p { font-size:12px; color:#666666; margin:5px 0 0; }
.dialog-copy { color:#444444; line-height:1.8; }
.dialog-note { color:#666666; font-size:12px; line-height:1.8; }
</style>
