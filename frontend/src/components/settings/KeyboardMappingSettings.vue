<template>
    <section class="content-card keymap-page" @click.self="clearSelection">
        <div class="keymap-top" @click.self="clearSelection">
            <div>
                <div class="profile-switcher" :class="{editing: editingName}">
                    <button
                        v-if="!editingName"
                        type="button"
                        class="profile-name-button"
                        :disabled="isBusy || !activeProfile"
                        title="点击修改方案名"
                        @click="startEditName"
                    >
                        <span class="profile-name">{{ profileName || '未命名方案' }}</span>
                        <span v-if="isDefaultProfile" class="default-badge">默认</span>
                    </button>
                    <n-input
                        v-else
                        ref="nameInputRef"
                        v-model:value="profileName"
                        size="small"
                        placeholder="方案名称"
                        :disabled="isBusy"
                        @click.stop
                        @blur="finishEditName"
                        @keydown="handleNameInputKeydown"
                    />

                    <n-popover
                        v-model:show="profilePickerOpen"
                        trigger="click"
                        placement="bottom-end"
                        :disabled="isBusy"
                    >
                        <template #trigger>
                            <button type="button" class="profile-arrow-button" :disabled="isBusy" title="选择方案">
                                <span class="chevron"></span>
                            </button>
                        </template>
                        <div class="profile-popover-panel" @click.stop>
                            <button
                                v-for="profile in profiles"
                                :key="profile.id"
                                type="button"
                                class="profile-option"
                                :class="{active: profile.id === store.config.activeKeymapProfileId}"
                                @click="selectProfile(profile.id)"
                            >
                                <span class="profile-option-name">{{ profile.name || '未命名方案' }}</span>
                                <span v-if="profile.id === defaultKeymapProfileId" class="profile-option-badge">默认</span>
                            </button>
                        </div>
                    </n-popover>
                </div>

            </div>

            <div class="profile-actions">
                <n-button size="small" secondary :loading="loadingAction === 'add'" :disabled="isBusy" @click="addProfile">新增</n-button>
                <n-popconfirm
                    :disabled="isDefaultProfile || isBusy"
                    @positive-click="deleteProfile"
                >
                    <template #trigger>
                        <n-button size="small" secondary type="error" :disabled="isDefaultProfile || isBusy">删除</n-button>
                    </template>
                    删除当前按键方案？
                </n-popconfirm>
                <n-button size="small" secondary :loading="loadingAction === 'import'" :disabled="isBusy" @click="importProfile">导入</n-button>
                <n-button size="small" secondary :loading="loadingAction === 'export'" :disabled="isBusy || !activeProfile" @click="exportProfile">导出</n-button>
            </div>
        </div>

        <div v-if="keymapError" class="keymap-alert">
            {{ keymapError }}
        </div>

        <div class="keymap-middle" @click.self="clearSelection">
            <template v-if="selectedNote">
                <div class="binding-summary">
                    <div>
                        <div class="panel-title">{{ formatNote(selectedNote) }}</div>
                    </div>
                    <div class="status-cluster">
                        <span v-if="listening" class="capture-pill">等待按键</span>
                        <span class="note-index">MIDI {{ selectedNote.index }}</span>
                    </div>
                </div>

                <div class="binding-tags">
                    <n-tag
                        v-for="binding in selectedBindings"
                        :key="binding.key"
                        size="small"
                        closable
                        @close="removeBinding(binding.key)"
                    >
                        {{ binding.label }}
                    </n-tag>
                    <span v-if="!selectedBindings.length" class="empty-bindings">暂无绑定</span>
                </div>
            </template>
            <template v-else>
                <div class="binding-summary">
                    <div>
                        <div class="panel-title">选择一个琴键开始绑定</div>
                    </div>
                </div>
            </template>
        </div>

        <div class="keymap-bottom" @click.self="clearSelection">
            <div class="piano-head">
                <div>
                    <div class="panel-title">钢琴映射</div>
                </div>
            </div>
            <div class="mini-keyboard-scroll" @click.self="clearSelection">
                <div class="mini-keyboard" @click.self="clearSelection">
                    <button
                        v-for="item in keyboardKeys"
                        :key="item.index"
                        type="button"
                        class="mini-key"
                        :class="[item.color, item.note, {selected: selectedMidiKey === item.index, mapped: bindingsByNote[item.index]?.length}]"
                        @click.stop="selectNote(item)"
                    >
                        <span class="key-note">{{ formatNote(item) }}</span>
                        <span v-if="bindingsByNote[item.index]?.length" class="key-binding">{{ bindingsByNote[item.index][0] }}</span>
                    </button>
                </div>
            </div>
        </div>
    </section>
</template>

<script setup>
import {computed, inject, nextTick, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {NButton, NInput, NPopover, NPopconfirm, NTag} from 'naive-ui'

const defaultKeymapProfileId = 'default'
const ignoredComputerKeys = new Set([
    'Shift',
    'Control',
    'Alt',
    'Meta',
    'CapsLock',
    'Tab',
    'Escape',
    'Enter',
    'Backspace',
    'Delete',
    'ArrowUp',
    'ArrowDown',
    'ArrowLeft',
    'ArrowRight',
    'Home',
    'End',
    'PageUp',
    'PageDown',
    'Insert',
    'ContextMenu',
    'NumLock',
    'ScrollLock',
    'Pause',
    'PrintScreen',
])

const store = inject('store')
const Keyboard = inject('Keyboard')
const applyConfig = inject('applyConfig')

const profileName = ref('')
const selectedMidiKey = ref(null)
const listening = ref(false)
const loadingAction = ref('')
const keymapError = ref('')
const editingName = ref(false)
const committingName = ref(false)
const profilePickerOpen = ref(false)
const nameInputRef = ref(null)

const activeProfile = computed(() => store.activeKeymapProfile)
const activeMapping = computed(() => activeProfile.value?.mapping || {})
const profiles = computed(() => store.config.keymapProfiles || [])
const isDefaultProfile = computed(() => activeProfile.value?.id === defaultKeymapProfileId)
const isBusy = computed(() => Boolean(loadingAction.value))
const keyboardKeys = computed(() => store.keyboardConfig || [])
const selectedNote = computed(() => keyboardKeys.value.find((item) => item.index === selectedMidiKey.value) || null)
const selectedBindings = computed(() => {
    if (selectedMidiKey.value === null) return []
    return Object.entries(activeMapping.value)
        .filter(([, midiKey]) => Number(midiKey) === selectedMidiKey.value)
        .map(([key]) => ({key, label: formatComputerKeyLabel(key)}))
        .sort((a, b) => a.label.localeCompare(b.label))
})
const bindingsByNote = computed(() => {
    const result = {}
    for (const [key, midiKey] of Object.entries(activeMapping.value)) {
        const note = Number(midiKey)
        if (!Number.isFinite(note)) continue
        if (!result[note]) result[note] = []
        result[note].push(formatComputerKeyLabel(key))
    }
    for (const note in result) {
        result[note].sort((a, b) => a.localeCompare(b))
    }
    return result
})
watch(
    activeProfile,
    (profile) => {
        profileName.value = profile?.name || ''
    },
    {immediate: true}
)

async function addProfile() {
    await runConfigAction('add', () => Keyboard.AddKeymapProfile(), '方案已新增')
}

async function deleteProfile() {
    if (!activeProfile.value || isDefaultProfile.value) return
    await runConfigAction('delete', () => Keyboard.DeleteKeymapProfile(activeProfile.value.id), '方案已删除')
    selectedMidiKey.value = null
    listening.value = false
}

async function selectProfile(id) {
    profilePickerOpen.value = false
    if (!id || id === store.config.activeKeymapProfileId) return
    clearSelection()
    await runConfigAction('select', () => Keyboard.SelectKeymapProfile(id))
    editingName.value = false
}

async function startEditName() {
    if (!activeProfile.value || isBusy.value) return
    profilePickerOpen.value = false
    editingName.value = true
    listening.value = false
    await nextTick()
    nameInputRef.value?.focus?.()
}

function handleNameInputKeydown(event) {
    if (event.key === 'Enter') {
        event.preventDefault()
        event.stopPropagation()
        finishEditName()
        return
    }
    if (event.key === 'Escape') {
        event.preventDefault()
        event.stopPropagation()
        cancelEditName()
    }
}

async function finishEditName() {
    if (!editingName.value || committingName.value) return
    committingName.value = true
    editingName.value = false
    try {
        await saveProfileName()
    } finally {
        committingName.value = false
    }
}

function cancelEditName() {
    editingName.value = false
    profileName.value = activeProfile.value?.name || ''
}

async function saveProfileName() {
    if (!activeProfile.value || isBusy.value) return
    const nextName = profileName.value.trim()
    if (nextName === activeProfile.value.name) return
    if (!nextName) {
        profileName.value = activeProfile.value.name || ''
        keymapError.value = '方案名称不能为空'
        return
    }
    await runConfigAction('rename', () => Keyboard.RenameKeymapProfile(activeProfile.value.id, nextName), '方案已重命名')
}

async function importProfile() {
    await runConfigAction('import', () => Keyboard.ImportKeymapProfile(), '方案已导入')
}

async function exportProfile() {
    if (!activeProfile.value) return
    try {
        loadingAction.value = 'export'
        keymapError.value = ''
        await Keyboard.ExportActiveKeymapProfile()
        window.$notify?.success?.('方案已导出', '当前按键方案已写入所选文件。')
    } catch (error) {
        if (!isUserCancelled(error)) {
            showError('导出按键方案失败', error)
        }
    } finally {
        loadingAction.value = ''
    }
}

function selectNote(item) {
    selectedMidiKey.value = item.index
    listening.value = true
}

function clearSelection() {
    selectedMidiKey.value = null
    listening.value = false
}

async function bindComputerKey(key) {
    if (!activeProfile.value || selectedMidiKey.value === null || isBusy.value) return
    await runConfigAction(
        'bind',
        () => Keyboard.BindKeymapKey(activeProfile.value.id, key, selectedMidiKey.value),
        '按键已绑定'
    )
}

async function removeBinding(key) {
    if (!activeProfile.value || isBusy.value) return
    await runConfigAction(
        `remove:${key}`,
        () => Keyboard.RemoveKeymapBinding(activeProfile.value.id, key),
        '绑定已移除'
    )
}

async function runConfigAction(action, runner, successTitle = '') {
    if (isBusy.value) return
    try {
        loadingAction.value = action
        keymapError.value = ''
        const config = await runner()
        if (config) applyConfig?.(config)
        if (successTitle) window.$notify?.success?.(successTitle, '按键方案设置已保存。')
    } catch (error) {
        if (!isUserCancelled(error)) {
            showError('按键方案操作失败', error)
        }
    } finally {
        loadingAction.value = ''
    }
}

function handleKeydown(event) {
    if (!listening.value || selectedMidiKey.value === null) return
    if (isTextInput(event.target) || event.repeat) return
    if (!isBindableComputerKey(event.key)) return

    event.preventDefault()
    event.stopPropagation()
    bindComputerKey(event.key)
}

function isBindableComputerKey(key) {
    if (key === ' ') return true
    if (ignoredComputerKeys.has(key)) return false
    return key.length === 1
}

function isTextInput(target) {
    const tagName = target?.tagName?.toLowerCase()
    return tagName === 'input' || tagName === 'textarea' || target?.isContentEditable
}

function formatNote(item) {
    if (!item) return ''
    return `${item.note}${item.octave}`
}

function formatComputerKeyLabel(key) {
    if (key === ' ') return 'Space'
    return key
}

function showError(title, error) {
    keymapError.value = formatError(error)
    window.$notify?.error?.(title, keymapError.value)
}

function formatError(error) {
    return String(error?.message || error || '未知错误')
}

function isUserCancelled(error) {
    const message = formatError(error).toLowerCase()
    return message.includes('cancel') || message.includes('取消') || message.includes('未选择')
}

onMounted(() => {
    window.addEventListener('keydown', handleKeydown, true)
})

onBeforeUnmount(() => {
    window.removeEventListener('keydown', handleKeydown, true)
})
</script>

<style lang="scss" scoped>
.keymap-page {
    display: flex;
    flex-direction: column;
    gap: 12px;
    overflow: hidden;
    background:
        linear-gradient(180deg, rgba(255, 255, 255, 0.84), rgba(248, 250, 252, 0.76)),
        var(--setting-panel-bg);
}

.keymap-top,
.keymap-middle,
.keymap-bottom {
    position: relative;
    border: 1px solid rgba(148, 163, 184, 0.16);
    background: rgba(255, 255, 255, 0.68);
    box-shadow: 0 12px 30px rgba(15, 23, 42, 0.045);
}

.keymap-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
    min-height: 82px;
    padding: 16px 18px;
    border-radius: 16px;
    background: rgba(255, 255, 255, 0.74);
}

.profile-switcher {
    display: grid;
    grid-template-columns: minmax(220px, 340px) 38px;
    align-items: center;
    width: min(100%, 390px);
    overflow: hidden;
    border-radius: 14px;
    background: rgba(255, 255, 255, 0.82);
    border: 1px solid rgba(148, 163, 184, 0.22);
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.05);
    transition:
        border-color 0.18s ease,
        background 0.18s ease;

    &:hover,
    &.editing {
        border-color: rgba(100, 116, 139, 0.34);
        background: rgba(255, 255, 255, 0.95);
    }
}

.profile-name-button,
.profile-arrow-button {
    height: 38px;
    border: 0;
    background: transparent;
    cursor: pointer;

    &:disabled {
        cursor: not-allowed;
        opacity: 0.58;
    }

    &:focus-visible {
        outline: 2px solid rgba(15, 23, 42, 0.2);
        outline-offset: -2px;
    }
}

.profile-switcher :deep(.n-input) {
    --n-border: 0 !important;
    --n-border-hover: 0 !important;
    --n-border-focus: 0 !important;
    --n-box-shadow-focus: none !important;
    --n-color: transparent !important;
    --n-color-focus: transparent !important;
}

.profile-name-button {
    display: flex;
    align-items: center;
    min-width: 0;
    gap: 9px;
    padding: 0 12px;
    color: var(--setting-text-main);
    text-align: left;
}

.profile-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 16px;
    font-weight: 760;
}

.default-badge {
    flex: 0 0 auto;
    padding: 3px 7px;
    border-radius: 999px;
    color: #334155;
    font-size: 11px;
    font-weight: 700;
    background: rgba(15, 23, 42, 0.06);
}

.profile-arrow-button {
    display: flex;
    align-items: center;
    justify-content: center;
    border-left: 1px solid rgba(148, 163, 184, 0.18);
    transition: background 0.18s ease;

    &:hover {
        background: rgba(15, 23, 42, 0.05);
    }
}

.chevron {
    width: 8px;
    height: 8px;
    border-right: 2px solid #64748b;
    border-bottom: 2px solid #64748b;
    transform: translateY(-2px) rotate(45deg);
}

.profile-actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 8px;
    flex-shrink: 0;
}

.keymap-alert {
    padding: 10px 12px;
    border-radius: 12px;
    color: #b91c1c;
    background: rgba(254, 242, 242, 0.82);
    border: 1px solid rgba(239, 68, 68, 0.16);
    font-size: 13px;
    line-height: 1.5;
}

.keymap-middle,
.keymap-bottom {
    min-width: 0;
    border-radius: 18px;
}

.keymap-middle {
    min-height: 86px;
    padding: 16px 18px;
}

.keymap-bottom {
    padding: 12px 18px 4px;
}

.piano-head,
.binding-summary {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
}

.panel-title {
    color: var(--setting-text-main);
    font-size: 15px;
    font-weight: 760;
}

.mini-keyboard-scroll {
    overflow-x: auto;
    overflow-y: hidden;
    padding: 6px 0 10px;
}

.mini-keyboard {
    display: flex;
    width: max-content;
    height: 156px;
    padding-right: 24px;
}

.mini-key {
    position: relative;
    display: block;
    height: 100%;
    padding: 0;
    border: 0;
    cursor: pointer;
    user-select: none;
    transition:
        background 0.14s ease,
        box-shadow 0.14s ease,
        transform 0.14s ease;

    &:focus-visible {
        outline: 2px solid rgba(37, 99, 235, 0.5);
        outline-offset: -2px;
    }
}

.mini-key.white {
    width: 31px;
    z-index: 1;
    border-left: 1px solid #cbd5e1;
    border-bottom: 1px solid #cbd5e1;
    border-radius: 0 0 5px 5px;
    background:
        linear-gradient(180deg, #ffffff, #eef2f7);
    box-shadow:
        inset 0 1px 0 rgba(255, 255, 255, 0.9),
        inset 0 -8px 14px rgba(15, 23, 42, 0.045);
}

.mini-key.black {
    width: 20px;
    height: 63%;
    z-index: 3;
    transform: translateX(-13px);
    border-left: 1px solid #020617;
    border-bottom: 1px solid #020617;
    border-radius: 0 0 3px 3px;
    background: #111827;
    box-shadow: inset 0 -8px 8px rgba(0, 0, 0, 0.7), 0 2px 4px rgba(15, 23, 42, 0.35);
}

.mini-key.B,
.mini-key.D,
.mini-key.E,
.mini-key.A,
.mini-key.G {
    margin-left: -20px;
}

.mini-key:hover {
    box-shadow:
        inset 0 0 0 2px rgba(37, 99, 235, 0.22),
        0 8px 18px rgba(15, 23, 42, 0.08);
}

.mini-key.mapped.white {
    background:
        linear-gradient(180deg, #f0fdf4, #dcfce7);
    border-left-color: rgba(34, 197, 94, 0.38);
    border-bottom-color: rgba(34, 197, 94, 0.42);
}

.mini-key.mapped.black {
    background:
        linear-gradient(180deg, #14532d, #052e16);
}

.mini-key.selected {
    box-shadow:
        inset 0 0 0 2px rgba(37, 99, 235, 0.78),
        0 0 0 1px rgba(37, 99, 235, 0.18);
}

.mini-key.selected::after {
    content: '';
    position: absolute;
    left: 50%;
    bottom: 6px;
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: #2563eb;
    transform: translateX(-50%);
}

.key-note,
.key-binding {
    position: absolute;
    left: 0;
    right: 0;
    text-align: center;
    pointer-events: none;
}

.key-note {
    bottom: 12px;
    color: rgba(15, 23, 42, 0.42);
    font-size: 10px;
}

.key-binding {
    bottom: 24px;
    color: #15803d;
    font-size: 10px;
    font-weight: 700;
}

.mini-key.black .key-note,
.mini-key.black .key-binding {
    color: rgba(255, 255, 255, 0.82);
}

.capture-pill {
    flex: 0 0 auto;
    padding: 4px 7px;
    border-radius: 999px;
    color: #1d4ed8;
    background: rgba(37, 99, 235, 0.1);
    font-size: 11px;
    font-weight: 650;
}

.status-cluster {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 8px;
}

.note-index {
    flex: 0 0 auto;
    padding: 6px 10px;
    border-radius: 999px;
    color: #475569;
    font-size: 12px;
    font-weight: 700;
    background: rgba(15, 23, 42, 0.05);
}

.binding-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 16px;
}

.empty-bindings {
    color: var(--setting-text-muted);
    font-size: 13px;
}

@media (max-width: 900px) {
    .keymap-top,
    .piano-head,
    .binding-summary {
        flex-direction: column;
        align-items: stretch;
    }

    .profile-switcher {
        width: 100%;
        grid-template-columns: minmax(0, 1fr) 38px;
    }

    .profile-actions {
        justify-content: flex-start;
    }
}

@media (prefers-reduced-motion: reduce) {
    .profile-switcher,
    .profile-arrow-button,
    .mini-key {
        transition: none;
    }
}

.profile-popover-panel {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 250px;
    max-width: 320px;


    background: rgba(255, 255, 255, 0.98);

}

.profile-option {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-height: 34px;
    padding: 0 11px;
    border: 0;
    border-radius: 9px;
    background: transparent;
    color: #334155;
    font-size: 13px;
    font-weight: 650;
    text-align: left;
    cursor: pointer;
    transition:
        background 0.14s ease,
        color 0.14s ease;

    &:hover {
        color: #0f172a;
        background: rgba(15, 23, 42, 0.05);
    }

    &:focus-visible {
        outline: 2px solid rgba(15, 23, 42, 0.18);
        outline-offset: -2px;
    }

    &.active {
        color: #0f172a;
        background: rgba(15, 23, 42, 0.08);
    }

    &.active::after {
        content: '';
        width: 6px;
        height: 6px;
        margin-left: auto;
        border-radius: 50%;
        background: #0f172a;
    }
}

.profile-option-name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.profile-option-badge {
    flex: 0 0 auto;
    padding: 2px 6px;
    border-radius: 999px;
    color: #475569;
    font-size: 11px;
    font-weight: 700;
    background: rgba(15, 23, 42, 0.06);
}
</style>
