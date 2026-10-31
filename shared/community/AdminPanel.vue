<template><section class="community"><div class="row spread"><h2>内容审核</h2><select v-model="status" @change="load"><option value="pending">待审核</option><option value="blocked">已下架</option><option value="rejected">已拒绝</option></select><button :disabled="busy" @click="load">刷新</button></div><p v-if="error" class="error" role="alert">{{error}}</p><article v-for="song in songs" :key="song.id" class="card"><h3>{{song.title}}</h3><p class="muted">{{song.ownerName}} · {{song.composer}} · {{song.noteCount}} 音</p><p class="description">{{song.description}}</p><p class="muted">{{song.tags}}</p><div class="row"><button :disabled="busy" @click="play(song)">试听</button><button class="primary" :disabled="busy" @click="review(song,'approved')">{{status==='blocked'?'解除下架并公开':'通过并公开'}}</button><button v-if="status!=='blocked'" :disabled="busy" @click="review(song,'rejected')">拒绝</button></div></article><p v-if="!songs.length" class="muted">此队列暂无作品。</p><h3>待处理举报</h3><article v-for="report in reports" :key="report.id" class="card"><strong>{{report.kind==='midi'?'MIDI':'琴房'}} · {{report.resourceId}}</strong><p class="description">{{report.reason}}</p><div class="row"><button :disabled="busy" @click="resolve(report,'resolved',report.kind==='midi'?'hide_midi':'close_room')">{{report.kind==='midi'?'下架 MIDI':'关闭琴房'}}</button><button :disabled="busy" @click="resolve(report,'resolved')">已处理</button><button :disabled="busy" @click="resolve(report,'dismissed')">忽略</button></div></article></section></template>
<script setup>
import {ref,onMounted} from 'vue'
import './community.css'
const props=defineProps({client:{type:Object,required:true}}),emit=defineEmits(['play'])
const status=ref('pending'),songs=ref([]),reports=ref([]),busy=ref(false),error=ref('')
async function run(action){busy.value=true;error.value='';try{await action()}catch(e){error.value=e.message||String(e)}finally{busy.value=false}}
async function refresh(){const [a,b]=await Promise.all([props.client.request('/api/admin/midis?status='+status.value),props.client.request('/api/admin/reports')]);songs.value=a.items||[];reports.value=b.items||[]}
const load=()=>run(refresh)
async function play(song){await run(async()=>{await props.client.loadSong(song,'play');emit('play')})}
async function review(song,value){if(status.value==='blocked'&&!confirm('解除此曲目的下架状态并恢复公开？'))return;await run(async()=>{await props.client.json(`/api/admin/midis/${song.id}/review`,'POST',{status:value});await refresh()})}
async function resolve(report,value,action=''){if(action&&!confirm('确认处理此举报并'+(action==='hide_midi'?'下架作品':'关闭琴房')+'？'))return;await run(async()=>{await props.client.json(`/api/admin/reports/${report.id}/resolve`,'POST',{status:value,action});await refresh()})}
onMounted(load)
</script>
