<template>
    <section class="content-card soundfont-page">
        <div class="soundfont-hero">
            <div>
                <div class="eyebrow">SoundFont Library</div>
                <div class="section-title hero-title">音源选择</div>
                <div class="hero-desc">管理本地 .sf2 音色库，切换后会同步应用到钢琴播放与练习。</div>
            </div>
            <div class="hero-actions">
                <n-button size="small" secondary :loading="loadingAction === 'refresh'" :disabled="isBusy" @click="refreshSoundFonts">刷新</n-button>
                <n-button type="primary" size="small" :loading="loadingAction === 'add'" :disabled="isBusy" @click="addSoundFont">导入音源</n-button>
            </div>
        </div>

        <div v-if="soundFontError" class="soundfont-alert">
            <span class="alert-dot"></span>
            <span>{{ soundFontError }}</span>
        </div>

        <div class="soundfont-toolbar">
            <div>
                <div class="label">音源库</div>
                <div class="desc">{{ activeSoundFontText }}</div>
            </div>
        </div>

        <div class="soundfont-list">
            <div class="soundfont-item" :class="{active: !store.config.activeSoundFontId}">
                <div class="soundfont-icon default-icon">
                    <span>SF2</span>
                </div>
                <div class="soundfont-item-main">
                    <div class="item-title-line">
                        <span class="label">Upright Piano KW</span>
                        <n-tag v-if="!store.config.activeSoundFontId" size="small" type="success" :bordered="false">正在使用</n-tag>
                        <n-tag size="small" type="info" :bordered="false">内置</n-tag>
                    </div>
                    <div class="desc">默认音源</div>
                </div>
                <div class="soundfont-actions">
                    <n-button
                        size="small"
                        secondary
                        :loading="loadingAction === 'select:'"
                        :disabled="!store.config.activeSoundFontId || isBusy"
                        @click="selectSoundFont('')"
                    >
                        使用
                    </n-button>
                </div>
            </div>

            <div v-if="!soundFonts.length" class="empty-library">
                <div class="empty-orb">+</div>
                <div>
                    <div class="label">还没有导入用户音源</div>
                    <div class="desc">当前使用内置 Upright Piano KW；你也可以导入自己的 .sf2 音源。</div>
                </div>
                <n-button size="small" type="primary" secondary :loading="loadingAction === 'add'" :disabled="isBusy" @click="addSoundFont">选择 .sf2 文件</n-button>
            </div>

            <div
                v-for="sf in soundFonts"
                :key="sf.id"
                class="soundfont-item"
                :class="{active: store.config.activeSoundFontId === sf.id, missing: sf.missing || sf.error}"
            >
                <div class="soundfont-icon user-icon">
                    <span>SF2</span>
                </div>
                <div class="soundfont-item-main">
                    <div class="item-title-line">
                        <span class="label">{{ sf.name }}</span>
                        <n-tag v-if="store.config.activeSoundFontId === sf.id" size="small" type="success" :bordered="false">正在使用</n-tag>
                        <n-tag v-if="sf.missing" size="small" type="error" :bordered="false">文件缺失</n-tag>
                        <n-tag v-else-if="sf.error" size="small" type="warning" :bordered="false">加载异常</n-tag>
                    </div>
                    <div class="desc mono path-text">{{ sf.path }}</div>
                    <div class="item-meta">{{ formatSize(sf.size) || '未知大小' }}</div>
                    <div v-if="sf.error" class="error-text">{{ sf.error }}</div>
                </div>
                <div class="soundfont-actions">
                    <n-button
                        size="small"
                        secondary
                        :loading="loadingAction === `select:${sf.id}`"
                        :disabled="store.config.activeSoundFontId === sf.id || isBusy"
                        @click="selectSoundFont(sf.id)"
                    >
                        使用
                    </n-button>
                    <n-button
                        size="small"
                        quaternary
                        type="error"
                        :loading="loadingAction === `remove:${sf.id}`"
                        :disabled="isBusy"
                        @click="removeSoundFont(sf.id)"
                    >
                        移除
                    </n-button>
                </div>
            </div>
        </div>
    </section>
</template>

<script setup>
import {computed, inject, ref} from 'vue'
import {NButton, NTag} from 'naive-ui'

const store = inject('store')
const Keyboard = inject('Keyboard')

const soundFontError = ref('')
const loadingAction = ref('')
const soundFonts = computed(() => store.config.soundFonts || [])
const isBusy = computed(() => Boolean(loadingAction.value))
const activeSoundFontText = computed(() => {
    if (!store.config.activeSoundFontId) return `当前使用内置 Upright Piano KW，可直接弹奏。已导入 ${soundFonts.value.length} 个用户音源。`
    return `已导入 ${soundFonts.value.length} 个用户音源。当前使用状态会直接显示在下方列表中。`
})

async function syncConfigFromBackend() {
    store.config = {...store.config, ...await Keyboard.SendConfig()}
}

async function addSoundFont() {
    if (isBusy.value) return
    try {
        loadingAction.value = 'add'
        soundFontError.value = ''
        await Keyboard.OpenSoundFontDialog()
        await syncConfigFromBackend()
        window.$notify?.success?.('音源已导入', '已加入音源库，可在列表中选择使用。')
    } catch (error) {
        if (isUserCancelled(error)) return
        showSoundFontError('音源导入失败', error)
    } finally {
        loadingAction.value = ''
    }
}

async function selectSoundFont(id) {
    if (isBusy.value) return
    try {
        loadingAction.value = `select:${id}`
        soundFontError.value = ''
        await Keyboard.SelectSoundFontByID(id)
        await syncConfigFromBackend()
        window.$notify?.success?.('音源已切换', '新的 SoundFont 已应用到播放和练习。')
    } catch (error) {
        showSoundFontError('音源切换失败', error)
        await syncConfigFromBackend()
    } finally {
        loadingAction.value = ''
    }
}

async function removeSoundFont(id) {
    if (isBusy.value) return
    try {
        loadingAction.value = `remove:${id}`
        soundFontError.value = ''
        await Keyboard.RemoveSoundFontByID(id)
        await syncConfigFromBackend()
        window.$notify?.success?.('音源已移除', '该 SoundFont 已从音源库移除。')
    } catch (error) {
        showSoundFontError('移除音源失败', error)
        await syncConfigFromBackend()
    } finally {
        loadingAction.value = ''
    }
}

async function refreshSoundFonts() {
    if (isBusy.value) return
    try {
        loadingAction.value = 'refresh'
        soundFontError.value = ''
        await Keyboard.RefreshSoundFonts()
        await syncConfigFromBackend()
    } catch (error) {
        showSoundFontError('刷新音源失败', error)
    } finally {
        loadingAction.value = ''
    }
}

function showSoundFontError(title, error) {
    soundFontError.value = formatError(error)
    window.$notify?.error?.(title, soundFontError.value)
}

function formatError(error) {
    return String(error?.message || error || '未知错误')
}

function isUserCancelled(error) {
    const message = formatError(error).toLowerCase()
    return message.includes('cancel') || message.includes('取消') || message.includes('未选择')
}

function formatSize(size) {
    const bytes = Number(size || 0)
    if (bytes <= 0) return ''
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
</script>

<style lang="scss" scoped src="../../../../shared/piano/soundfont.scss"></style>
