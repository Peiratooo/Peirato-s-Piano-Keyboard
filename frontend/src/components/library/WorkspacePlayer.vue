<template>
 <footer class="workspace-player" aria-label="曲库播放控制">
  <div class="preview-song">
   <span class="preview-art" aria-hidden="true"><PhMusicNotes :size="27" weight="duotone"/></span>
   <div class="preview-title"><strong :title="session.selectedSong?.name">{{session.selectedSong?.name||'选择一首想弹的曲子'}}</strong><span>{{session.state.leadInMs>0?'准备播放 · '+Math.ceil(session.state.leadInMs/1000):(session.selectedId?'MIDI 试听':'选择曲目开始试听')}}</span></div>
  </div>
  <div class="preview-actions">
   <button class="preview-skip" :disabled="session.busy||songIndex<=0" aria-label="上一首" @click="skip(-1)"><PhSkipBack :size="23" weight="fill"/></button>
   <button class="preview-toggle" :class="{pending:session.busy&&session.selectedId}" :aria-busy="session.busy" :disabled="!session.selectedId||session.busy" :aria-label="session.state.status==='playing'?'暂停试听':'播放试听'" @click="$emit('toggle')"><PhPause v-if="session.state.status==='playing'" :size="23" weight="fill"/><PhPlay v-else :size="23" weight="fill"/></button>
   <button class="preview-skip" :disabled="session.busy||songIndex<0||songIndex>=session.library.length-1" aria-label="下一首" @click="skip(1)"><PhSkipForward :size="23" weight="fill"/></button>
  </div>
  <div class="preview-tools">
   <span class="preview-time">{{time(seekDraft??session.state.currentMs)}} <i>/</i> {{time(session.durationMs)}}</span>
   <button class="preview-loop" :class="{selected:session.options.loop}" :disabled="!session.selectedId||session.busy" :aria-pressed="session.options.loop" aria-label="循环播放" @click="session.setLoop(!session.options.loop)"><PhRepeat :size="22"/></button>
   <button class="return-piano" aria-label="返回钢琴" @click="$emit('piano')"><PhPianoKeys :size="23"/><span>返回钢琴</span></button>
  </div>
  <div class="preview-progress" @pointerup.capture="commitSeek"><NSlider :value="seekDraft??session.state.currentMs" :min="0" :max="Math.max(1,session.durationMs)" :tooltip="false" :disabled="!session.selectedId||session.loading" aria-label="试听进度" @update:value="previewSeek" @dragend="commitSeek" @keyup="seekKey"/></div>
 </footer>
</template>
<script setup>
import {computed,ref,watch} from 'vue'
import {NSlider} from 'naive-ui'
import {PhPause,PhPlay,PhRepeat,PhPianoKeys,PhMusicNotes,PhSkipBack,PhSkipForward} from '@phosphor-icons/vue'
const props=defineProps({session:{type:Object,required:true}})
defineEmits(['piano','toggle'])
const seekDraft=ref(null)
const songIndex=computed(()=>props.session.library.findIndex(song=>song.id===props.session.selectedId))
const time=ms=>{const seconds=Math.max(0,Math.floor((ms||0)/1000));return `${Math.floor(seconds/60)}:${String(seconds%60).padStart(2,'0')}`}
async function skip(direction){const song=props.session.library[songIndex.value+direction];if(!await props.session.selectSong(song.id)||!await props.session.setMode('play'))return;await props.session.start()}
function previewSeek(value){seekDraft.value=value;props.session.previewSeek(value)}
async function commitSeek(){if(seekDraft.value===null)return;const value=seekDraft.value;seekDraft.value=null;props.session.endSeekPreview();await props.session.seek(value)}
function seekKey(event){if(['ArrowLeft','ArrowRight','ArrowUp','ArrowDown','Home','End'].includes(event.key))void commitSeek()}
watch(()=>props.session.selectedId,()=>{seekDraft.value=null;props.session.endSeekPreview()})
</script>
<style scoped>
.workspace-player{position:relative;display:grid;grid-template-columns:minmax(0,1fr) auto minmax(0,1fr);align-items:center;gap:24px;padding:10px 28px 14px;border-top:1px solid #ffffff12;background:#222325;color:#eceef0;min-width:0}
button{font:inherit;cursor:pointer;display:inline-flex;align-items:center;justify-content:center;border:0;color:inherit;background:transparent;flex-shrink:0}button:disabled{opacity:.3;cursor:default}button:focus-visible{outline:2px solid var(--ui-accent);outline-offset:3px}
.preview-song{display:flex;align-items:center;gap:12px;min-width:0}.preview-art{display:grid;place-items:center;width:46px;height:46px;flex-shrink:0;border-radius:8px;background:var(--ui-accent-soft);color:var(--ui-accent)}
.preview-title{min-width:0;display:flex;flex-direction:column;gap:5px}.preview-title strong{font-size:14px;font-weight:550;white-space:nowrap;text-overflow:ellipsis;overflow:hidden;line-height:1.4}.preview-title span{font-size:12px;color:#92969d;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;line-height:1.4}
.preview-actions{display:flex;align-items:center;gap:22px}.preview-skip{width:28px;height:36px;color:#c1c4c9}.preview-skip:hover:enabled{color:var(--ui-accent)}.preview-toggle{width:44px;height:44px;border-radius:50%;background:var(--ui-accent);color:var(--ui-accent-text)}.preview-toggle:hover:enabled{background:var(--ui-accent-hover)}.preview-toggle.pending:disabled{opacity:1}
.preview-tools{display:flex;align-items:center;justify-content:flex-end;gap:18px;min-width:0}.preview-loop{width:28px;height:34px;color:#7c8087}.preview-loop.selected{color:var(--ui-accent)}.preview-time{color:#969ba2;font-size:11px;white-space:nowrap;font-variant-numeric:tabular-nums}.preview-time i{color:#656b74;font-style:normal;margin:0 4px}.return-piano{min-height:36px;gap:8px;font-size:12px;color:#bfc3c9}.return-piano:hover{color:var(--ui-accent)}
.preview-progress{position:absolute;left:0;right:0;bottom:-6px;height:18px;display:flex;align-items:center}.preview-progress :deep(.n-slider){--n-rail-height:3px!important;--n-handle-size:9px!important;width:100%;padding:0}.preview-progress :deep(.n-slider-handle){opacity:0;transition:opacity .15s}.preview-progress:hover :deep(.n-slider-handle),.preview-progress:focus-within :deep(.n-slider-handle){opacity:1}
@media(max-width:1000px){.workspace-player{padding:10px 20px 14px;gap:18px}.preview-tools{gap:12px}.preview-actions{gap:16px}.return-piano span{display:none}}
@media(max-width:650px){.workspace-player{grid-template-columns:minmax(0,1fr) auto auto;padding:8px 16px 12px;gap:14px}.preview-art{width:38px;height:38px}.preview-title strong{font-size:13px}.preview-title span{font-size:11px}.preview-actions{gap:9px}.preview-toggle{height:38px;width:38px}.preview-time,.preview-loop{display:none}.preview-tools{gap:0}.preview-skip{width:24px}.return-piano{width:26px}}
</style>
