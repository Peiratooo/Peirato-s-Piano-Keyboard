<template>
    <div role="region" class="midi-library" data-piano-control aria-label="MIDI 曲库">
        <header>
            <div><PhMusicNotes :size="20" /><h2>曲库</h2><span>{{ session.library.length }} 首</span></div>
            <div class="library-actions"><NButton size="small" type="primary" :loading="session.importing" :disabled="session.importing" @click="importSong">
                <template #icon><PhDownloadSimple /></template>导入 MIDI
            </NButton><NButton quaternary circle size="small" aria-label="关闭曲库" @click="$emit('close')"><template #icon><PhX :size="19" /></template></NButton></div>
        </header>
        <NInput v-model:value="search" placeholder="搜索曲目" clearable size="small" aria-label="搜索曲目">
            <template #prefix><PhMagnifyingGlass :size="16" /></template>
        </NInput>
        <div class="library-scroll">
            <NScrollbar style="height: auto; max-height: var(--library-list-height)" trigger="hover" :aria-busy="session.loading">
                <div v-for="song in filteredSongs" :key="song.id" class="song-row" :class="{selected: song.id === session.selectedId}">
                    <NButton quaternary class="song-select" :aria-pressed="song.id === session.selectedId" @click="choose(song.id)">
                        <template #icon><PhMusicNote :size="19" /></template>
                        <span class="song-info"><strong>{{ song.name }}</strong><small>{{ song.missing ? '文件不可用，请重新导入' : formatMidiTime(song.durationMs) }}</small></span>
                        <PhCheck v-if="song.id === session.selectedId" class="selection-mark" :size="16" />
                    </NButton>
                    <NPopconfirm :positive-button-props="{type: 'error'}" positive-text="移除" negative-text="保留" @positive-click="session.removeSong(song.id)">
                        <template #trigger><NButton quaternary circle size="small" :disabled="session.busy" :aria-label="`移除 ${song.name}`"><template #icon><PhTrash :size="16" /></template></NButton></template>
                        从曲库移除“{{ song.name }}”？原 MIDI 文件会保留。
                    </NPopconfirm>
                </div>
                <div v-if="!filteredSongs.length" class="empty-library">
                    <PhMusicNotes :size="30" weight="light" />
                    <span>{{ search ? '没有匹配的曲目' : '导入 MIDI，开始播放或跟弹' }}</span>
                    <NButton v-if="!search" size="small" secondary :loading="session.importing" @click="importSong">选择 MIDI 文件</NButton>
                </div>
            </NScrollbar>
        </div>
        <footer>选择曲目后，点击播放或开始练习。</footer>
    </div>
</template>

<script setup>
import {computed, ref} from 'vue'
import {NButton, NInput, NPopconfirm, NScrollbar} from 'naive-ui'
import {PhCheck, PhDownloadSimple, PhMagnifyingGlass, PhMusicNote, PhMusicNotes, PhTrash, PhX} from '@phosphor-icons/vue'
import {formatMidiTime} from '../services/midiSession'
const props = defineProps({session: {type: Object, required: true}})
const emit = defineEmits(['selected', 'close'])
const search = ref('')
const filteredSongs = computed(() => props.session.library.filter(song => song.name.toLowerCase().includes(search.value.trim().toLowerCase())))
async function choose(id) {
    if (await props.session.selectSong(id)) emit('selected', id)
}
async function importSong() {
    if (await props.session.importSong()) emit('selected', props.session.selectedId)
}
</script>

<style scoped>
.midi-library { display: flex; flex-direction: column; width: 720px; max-width: calc(100vw - 48px); max-height: calc(100vh - 80px); padding: 24px; border: 1px solid #ffffff19; border-radius: 14px; background: #282828; box-shadow: 0 24px 100px #0008; color: #ededed; }
.library-actions { display: flex; align-items: center; gap: 10px; }
header, .midi-library > :deep(.n-input), footer { flex-shrink: 0; }
header, header > div { display: flex; align-items: center; gap: 9px; }
header { justify-content: space-between; margin-bottom: 16px; }
h2 { font-size: 15px; font-weight: 550; }
header span { font-size: 11px; color: #939393; }
.library-scroll { --library-list-height: min(450px, calc(100vh - 260px)); flex: 0 1 auto; min-height: 0; margin: 16px -5px 0; }
.library-scroll :deep(.n-scrollbar-container) { height: auto; overscroll-behavior: contain; }
.library-scroll :deep(.n-scrollbar-rail.n-scrollbar-rail--vertical) { width: 6px; right: 1px; }
.library-scroll :deep(.n-scrollbar-rail .n-scrollbar-rail__scrollbar) { border-radius: 6px; background: #a6a6a65e; }
.library-scroll :deep(.n-scrollbar-rail:hover .n-scrollbar-rail__scrollbar) { background: #bcbcbc99; }
.song-row { display: flex; align-items: center; gap: 3px; padding: 2px 4px; border-radius: 7px; margin: 2px 0; }
.song-row.selected { background: var(--ui-accent-soft, #ffffff0c); }
.song-select { flex: 1; min-width: 0; height: 58px; justify-content: flex-start; padding: 8px !important; }
.song-select :deep(.n-button__content) { width: 100%; min-width: 0; justify-content: flex-start; gap: 10px; }
.song-info { display: flex; flex: 1; flex-direction: column; min-width: 0; text-align: left; gap: 7px; }
.song-info strong { font-size: 13px; font-weight: 450; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.song-info small { color: #969696; font-size: 11px; }
.selection-mark { color: var(--ui-accent, #b8d1a6); flex-shrink: 0; }
.empty-library { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16px; padding: 35px 12px; color: #a5a5a5; font-size: 12px; }
footer { border-top: 1px solid #ffffff0d; padding-top: 13px; margin-top: 10px; font-size: 11px; line-height: 1.5; color: #939393; }
</style>
