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
            <div v-if="!soundFonts.length" class="empty-library">
                <div class="empty-orb">+</div>
                <div>
                    <div class="label">还没有导入用户音源</div>
                    <div class="desc">未导入音源时软件会正常运行，但不会通过内置音源发声。</div>
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
    if (!soundFonts.value.length) return '当前未启用内置音源。导入并选择 .sf2 后才会发声。'
    if (!store.config.activeSoundFontId) return `已导入 ${soundFonts.value.length} 个用户音源，当前未启用内置音源。`
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

<style lang="scss" scoped>
.soundfont-page {
    position: relative;
    overflow: hidden;

    &::before {
        content: '';
        position: absolute;
        top: -150px;
        right: -140px;
        width: 320px;
        height: 320px;
        pointer-events: none;
        background:
            radial-gradient(circle, rgba(37, 99, 235, 0.12), transparent 62%),
            radial-gradient(circle at 30% 30%, rgba(99, 102, 241, 0.08), transparent 54%);
    }
}

.soundfont-hero {
    position: relative;
    z-index: 1;
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 20px;
    padding-bottom: 18px;
    margin-bottom: 4px;
    border-bottom: 1px solid rgba(148, 163, 184, 0.14);
}

.eyebrow {
    margin-bottom: 7px;
    font-size: 11px;
    font-weight: 720;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: rgba(37, 99, 235, 0.82);
}

.hero-title {
    margin-bottom: 0;
}

.hero-desc {
    max-width: 560px;
    margin-top: 7px;
    color: var(--setting-text-muted);
    font-size: 13px;
    line-height: 1.7;
}

.hero-actions {
    display: flex;
    flex-shrink: 0;
    gap: 10px;
}

.soundfont-alert {
    position: relative;
    z-index: 1;
    display: flex;
    align-items: flex-start;
    gap: 9px;
    margin-top: 14px;
    padding: 11px 12px;
    border-radius: 16px;
    color: #b42318;
    font-size: 13px;
    line-height: 1.55;
    background: rgba(254, 242, 242, 0.82);
    border: 1px solid rgba(239, 68, 68, 0.16);
}

.alert-dot {
    flex: 0 0 auto;
    width: 7px;
    height: 7px;
    margin-top: 7px;
    border-radius: 50%;
    background: #ef4444;
}

.soundfont-toolbar {
    position: relative;
    z-index: 1;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
    padding: 18px 2px 12px;
}

.soundfont-list {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.soundfont-item {
    position: relative;
    display: grid;
    grid-template-columns: 46px minmax(0, 1fr) auto;
    align-items: center;
    gap: 14px;
    padding: 14px;
    border-radius: 20px;
    background: rgba(255, 255, 255, 0.66);
    border: 1px solid rgba(148, 163, 184, 0.16);
    box-shadow: 0 10px 26px rgba(15, 23, 42, 0.04);
    transition:
        transform 0.18s ease,
        border-color 0.18s ease,
        box-shadow 0.18s ease,
        background 0.18s ease;

    &::before {
        content: '';
        position: absolute;
        left: 0;
        top: 16px;
        bottom: 16px;
        width: 3px;
        border-radius: 99px;
        background: transparent;
        transition: background 0.18s ease;
    }

    &:hover {
        transform: translateY(-1px);
        border-color: rgba(37, 99, 235, 0.2);
        background: rgba(255, 255, 255, 0.86);
        box-shadow: 0 14px 32px rgba(15, 23, 42, 0.065);
    }

    &.active {
        border-color: rgba(37, 99, 235, 0.32);
        background:
            linear-gradient(135deg, rgba(37, 99, 235, 0.075), rgba(255, 255, 255, 0.82)),
            rgba(255, 255, 255, 0.88);
        box-shadow: 0 14px 34px rgba(37, 99, 235, 0.08);


        .soundfont-icon {
            color: #fff;
            background:
                linear-gradient(160deg, rgba(37, 99, 235, 0.96), rgba(99, 102, 241, 0.82)),
                rgba(37, 99, 235, 0.9);
            box-shadow: 0 10px 22px rgba(37, 99, 235, 0.18);
            border-color: rgba(255, 255, 255, 0.52);
        }
    }

    &.missing {
        border-color: rgba(239, 68, 68, 0.24);
        background: rgba(254, 242, 242, 0.62);

        &::before {
            background: #ef4444;
        }
    }
}

.soundfont-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 46px;
    height: 46px;
    border-radius: 16px;
    font-size: 12px;
    font-weight: 780;
    color: var(--setting-brand);
    background: rgba(37, 99, 235, 0.085);
    border: 1px solid rgba(37, 99, 235, 0.1);
    transition:
        color 0.18s ease,
        background 0.18s ease,
        box-shadow 0.18s ease,
        border-color 0.18s ease;

    span {
        transform: translateY(0.5px);
    }
}

.user-icon {
    color: #475569;
    background: rgba(15, 23, 42, 0.052);
    border-color: rgba(15, 23, 42, 0.06);
}

.soundfont-item-main {
    min-width: 0;
}

.item-title-line {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
}

.path-text {
    margin-top: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.item-meta {
    margin-top: 6px;
    color: var(--setting-text-muted);
    font-size: 12px;
}

.soundfont-actions {
    display: flex;
    flex-shrink: 0;
    gap: 8px;
}

.empty-library {
    display: grid;
    grid-template-columns: 42px minmax(0, 1fr) auto;
    align-items: center;
    gap: 14px;
    padding: 16px;
    border-radius: 20px;
    border: 1px dashed rgba(37, 99, 235, 0.24);
    background:
        linear-gradient(135deg, rgba(37, 99, 235, 0.045), rgba(255, 255, 255, 0.72)),
        rgba(255, 255, 255, 0.62);
}

.empty-orb {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    border-radius: 50%;
    color: var(--setting-brand);
    font-size: 22px;
    line-height: 1;
    background: rgba(255, 255, 255, 0.76);
    border: 1px solid rgba(37, 99, 235, 0.12);
}

@media (max-width: 720px) {
    .soundfont-hero,
    .soundfont-toolbar {
        align-items: stretch;
        flex-direction: column;
    }

    .hero-actions {
        justify-content: flex-end;
    }

    .soundfont-item,
    .empty-library {
        grid-template-columns: minmax(0, 1fr);
    }

    .soundfont-icon,
    .empty-orb {
        display: none;
    }

    .soundfont-actions {
        justify-content: flex-end;
    }
}
</style>
