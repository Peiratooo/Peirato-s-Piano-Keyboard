<template>
 <footer class="workspace-player" aria-label="工坊播放控制">
  <div class="preview-title"><strong :title="session.selectedSong?.name">{{session.selectedSong?.name||'选择一首想弹的曲子'}}</strong><span>{{composer||(session.selectedId?'MIDI 试听':'点击曲目右上角试听，或打开详情开始跟弹')}}</span></div>
  <div class="preview-actions"><button class="preview-loop" :class="{selected:session.options.loop}" :disabled="!session.selectedId" :aria-pressed="session.options.loop" aria-label="循环播放" @click="session.setLoop(!session.options.loop)"><PhRepeat :size="22"/></button>
   <button class="preview-toggle" :disabled="!session.selectedId||session.busy" :aria-label="session.state.status==='playing'?'暂停试听':'播放试听'" @click="$emit('toggle')"><PhPause v-if="session.state.status==='playing'" :size="25" weight="fill"/><PhPlay v-else :size="25" weight="fill"/></button></div>
  <div class="preview-progress"><NSlider :value="seekDraft??session.state.currentMs" :min="0" :max="Math.max(1,session.durationMs)" :tooltip="false" :disabled="!session.selectedId" aria-label="试听进度" @update:value="previewSeek" @dragend="commitSeek" @keyup="seekKey"/><span>{{time(seekDraft??session.state.currentMs)}} <i>/</i> {{time(session.durationMs)}}</span></div>
  <button class="return-piano" aria-label="返回演奏" @click="$emit('piano')"><PhPianoKeys :size="24"/><span>返回演奏</span></button>
 </footer>
</template>
<script setup>
import {ref,watch} from 'vue'
import {NSlider} from 'naive-ui'
import {PhPause,PhPlay,PhRepeat,PhPianoKeys} from '@phosphor-icons/vue'
const props=defineProps({session:{type:Object,required:true},composer:{type:String,default:''}})
defineEmits(['piano','toggle'])
const seekDraft=ref(null)
const time=ms=>{const seconds=Math.max(0,Math.floor((ms||0)/1000));return `${Math.floor(seconds/60)}:${String(seconds%60).padStart(2,'0')}`}
function previewSeek(value){seekDraft.value=value;props.session.previewSeek(value)}
async function commitSeek(){if(seekDraft.value===null)return;const value=seekDraft.value;await props.session.seek(value);if(seekDraft.value===value){seekDraft.value=null;props.session.endSeekPreview()}}
function seekKey(event){if(['ArrowLeft','ArrowRight','ArrowUp','ArrowDown','Home','End'].includes(event.key))void commitSeek()}
watch(()=>props.session.selectedId,()=>{seekDraft.value=null;props.session.endSeekPreview()})
</script>
<style scoped>
.workspace-player{display:flex;align-items:center;gap:28px;padding:16px 30px;border-top:1px solid #ffffff12;background:#232527;color:#eceef0;min-width:0}
button{font:inherit;cursor:pointer;display:inline-flex;align-items:center;justify-content:center;border:0;color:inherit;background:transparent;flex-shrink:0}button:disabled{opacity:.35;cursor:default}button:focus-visible{outline:2px solid var(--ui-accent);outline-offset:4px}
.preview-title{flex:0 1 370px;min-width:0;display:flex;flex-direction:column;gap:7px}.preview-title strong{font-size:16px;font-weight:600;white-space:nowrap;text-overflow:ellipsis;overflow:hidden;line-height:1.3}.preview-title span{font-size:13px;color:#a5a8ad;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;line-height:1.3}
.preview-actions{display:flex;align-items:center;gap:25px}.preview-loop{width:30px;height:35px;color:#7c8087}.preview-loop.selected{color:var(--ui-accent)}.preview-toggle{width:51px;height:51px;border-radius:50%;background:var(--ui-accent);color:var(--ui-accent-text)}.preview-toggle:hover{background:var(--ui-accent-hover)}
.preview-progress{flex:1;min-width:80px;display:flex;align-items:center;gap:24px}.preview-progress>:first-child{flex:1;min-width:0}.preview-progress>span{color:#b2b6bc;font-size:12px;white-space:nowrap;font-variant-numeric:tabular-nums}.preview-progress i{color:#747981;font-style:normal;margin:0 5px}.return-piano{min-height:48px;gap:12px;padding:0 20px;border:1px solid #ffffff24;border-radius:10px;font-size:14px}.return-piano:hover{border-color:var(--ui-accent);color:var(--ui-accent)}
@media(max-width:1000px){.workspace-player{gap:18px;padding:12px 20px}.preview-title{flex-basis:230px}.preview-actions{gap:12px}.preview-progress{gap:14px}.preview-progress>span{font-size:11px}.preview-toggle{height:43px;width:43px}.return-piano{padding:0 12px}}
@media(max-width:650px){.workspace-player{gap:12px;padding:10px 16px}.preview-title{flex:1}.preview-title strong{font-size:14px}.preview-title span{font-size:11px}.preview-loop,.preview-progress{display:none}.return-piano{min-height:40px;padding:0 10px}.return-piano span{display:none}.preview-toggle{height:40px;width:40px}.preview-actions{gap:0}}
</style>
