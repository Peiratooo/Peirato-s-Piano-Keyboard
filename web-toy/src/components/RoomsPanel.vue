<template><div class="community rooms-panel"><p v-if="error" class="error">{{error}}</p>
      <div class="card">
        <div class="row spread"><strong>大众琴房</strong><span class="muted">5 分钟 / 人</span></div>
        <p>所有人都可以观看。每次只有一位演奏者，排队上台或主动让位。</p>
        <p v-if="community.room?.id==='lobby'" class="message">{{community.connectionState==='reconnecting'?'正在恢复连接…':community.connectionState==='connected'?'已连接':'连接中'}} · 在线 {{community.audience}} · 当前演奏：{{community.performerName||'等待上台'}}<span v-if="community.latencyMs!==null"> · 单程约 {{community.latencyMs}}ms</span></p>
        <div class="row room-actions">
          <button class="primary" :disabled="community.room?.id==='lobby'" @click="run(()=>community.join('lobby'))">{{community.room?.id==='lobby'?(community.connected?'已在大众琴房':'正在重新连接…'):'进入大众琴房'}}</button>
          <button v-if="community.room?.id==='lobby'" class="secondary" @click="community.leave()">离开</button>
        </div>
        <div v-if="community.room?.id==='lobby'" class="room-seat-actions">
          <p v-if="!community.connected" class="muted">连接中断，正在自动重连。恢复前不能演奏或排队；短暂掉线保留席位和队列最多 15 秒。</p>
          <p v-if="community.isPerformer">轮到你了 · 剩余 <strong>{{formatCountdown(community.remainingSeconds)}}</strong></p>
          <p v-else-if="community.myPosition">当前排队第 <strong>{{community.myPosition}}</strong> 位</p>
          <p v-else>正在观看 · 可以排队获得演奏席位</p>
          <button v-if="community.isPerformer" class="secondary" :disabled="!community.connected" @click="community.yieldSeat()">主动下场 · 交给下一位</button>
          <button v-else-if="community.myPosition" class="secondary" :disabled="!community.connected" @click="community.leaveQueue()">取消排队</button>
          <button v-else class="secondary" :disabled="!community.user||!community.connected" @click="community.queueUp()">排队上台</button>
        </div>
      </div>
      <div class="section-label">自建琴房</div>
      <div class="card">
        <input v-model="roomTitle" maxlength="80" placeholder="我的练琴房"/>
        <select v-model="roomVisibility"><option value="public">公开 · 可以随机发现</option><option value="private">私密 · 仅邀请进入</option></select>
        <button class="primary" :disabled="!community.user||busy" @click="createRoom">创建我的琴房</button>
        <p class="muted">房主独奏，其他人只围观；每个账号同一时间只能开一间琴房。</p>
        <template v-if="community.latestInvite">
          <div class="invite-code">邀请凭证：<code>{{community.latestInvite}}</code></div>
          <p class="muted">邀请凭证仅在创建或重置时显示，请自己保管。只有房间 ID 加凭证才可进入私密房。</p>
        </template>
      </div>
      <div class="row room-actions">
        <button class="secondary" @click="run(()=>community.randomRoom())">随机围观一个公开琴房</button>
        <button class="secondary" @click="loadRooms">刷新列表</button>
      </div>
      <div v-for="room in publicRooms" :key="room.id" class="card room-row">
        <div><strong>{{room.title}}</strong><small>{{room.ownerName}} · 公开</small></div>
        <div class="row"><button @click="run(()=>community.join(room.id))">围观</button><button v-if="community.user" title="举报公开房间" @click="report('room',room.id)">举报</button></div>
      </div>
      <div class="section-label">凭邀请进入私密琴房</div>
      <input v-model.trim="joinId" placeholder="房间 ID"/>
      <input v-model.trim="joinCode" placeholder="邀请凭证"/>
      <button class="secondary" @click="run(()=>community.join(joinId,joinCode))">进入私密琴房</button></div></template>
<script setup>
import {ref,onMounted} from 'vue'
const props=defineProps({community:{type:Object,required:true},piano:{type:Object,required:true}})
const emit=defineEmits(['toast','close'])
const busy=ref(false),error=ref('')
const roomTitle=ref(''),roomVisibility=ref('public'),publicRooms=ref([]),joinId=ref(''),joinCode=ref('')
function formatCountdown(v){return `${Math.floor(v/60)}:${String(v%60).padStart(2,'0')}`}
async function run(fn,closeAfter=false){
 if(busy.value)return
 busy.value=true;error.value=''
 try{const result=await fn();if(closeAfter)emit('close');return result}
 catch(e){error.value=e.message||String(e);emit('toast',error.value)}
 finally{busy.value=false}
}
async function loadRooms(){await run(async()=>{const data=await props.community.request('/api/rooms');publicRooms.value=data.rooms||[]})}
async function createRoom(){await run(async()=>{
 await props.piano.unlock()
 const r=await props.community.createRoom(roomTitle.value,roomVisibility.value);emit('toast',r.room.visibility==='private'?'私密琴房已创建，请保存邀请凭证':'公开琴房已创建');emit('close')})}
async function report(kind,resourceId){
 const reason=prompt('请输入举报原因（至少 5 字）')
 if(!reason)return
 await run(async()=>{await props.community.json('/api/reports','POST',{kind,resourceId,reason});emit('toast','举报已提交')})
}
onMounted(loadRooms)
</script>
<style scoped>
.section-label{font-weight:600;padding-top:8px}.room-actions>button{flex:1}.room-seat-actions{display:flex;flex-direction:column;gap:10px;border-top:1px solid #ffffff18;padding-top:12px}.room-row{flex-direction:row;align-items:center;justify-content:space-between}.room-row>div:first-child{display:flex;flex-direction:column;gap:5px}.room-row small{font-size:11px;color:#999}.invite-code{overflow-wrap:anywhere;border-radius:7px;background:#202023;padding:10px}
</style>
