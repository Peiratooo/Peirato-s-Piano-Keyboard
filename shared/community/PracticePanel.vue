<template>
 <section class="community practice-workspace" aria-label="练习">
  <header class="catalog-heading"><div><h1>练习</h1><p>把每一次练习，留在这里</p></div><button class="primary" @click="$emit('practice')"><PhPianoKeys :size="19"/>选择曲目</button></header>
  <div class="practice-sources"><button :class="{selected:source==='local'}" @click="source='local';load()">本机记录</button><button v-if="client.user" :class="{selected:source==='cloud'}" @click="source='cloud';load()">同步记录</button><button class="refresh-history" :disabled="busy" @click="load"><PhArrowClockwise :size="18"/>刷新</button></div>
  <p v-if="error" class="error" role="alert">{{error}}</p>
  <div class="practice-summary"><div><strong>{{items.length}}</strong><span>练习曲目</span></div><div><strong>{{Math.floor(seconds/60)}}</strong><span>练习分钟</span></div><div><strong>{{items.reduce((sum,item)=>sum+item.sessions,0)}}</strong><span>练习次数</span></div></div>
  <div class="practice-grid"><article v-for="item in items" :key="item.localId||item.midiId" class="card"><h3>{{item.title}}</h3><p class="muted">{{item.sessions}} 次练习 · {{Math.floor(item.seconds/60)}} 分 {{Math.floor(item.seconds%60)}} 秒</p><p class="muted">上次练习 {{new Date(item.lastPracticed).toLocaleString('zh-CN')}}</p><button :disabled="busy" @click="resume(item)"><PhPlay/>继续练习</button></article></div>
  <div v-if="!items.length" class="catalog-empty"><PhPianoKeys :size="40" weight="duotone"/><h2>选一首喜欢的曲子开始吧</h2><p>不登录也会保存本机的练习记录。</p><button @click="$emit('practice')">选择曲目</button></div>
 </section>
</template>
<script setup>
import {ref,computed,onMounted,watch} from 'vue'
import {PhArrowClockwise,PhPlay,PhPianoKeys} from '@phosphor-icons/vue'
import './community.css'
import './workshop.css'
const props=defineProps({client:{type:Object,required:true}}),emit=defineEmits(['play','account','practice'])
const source=ref('local'),items=ref([]),busy=ref(false),error=ref(''),seconds=computed(()=>items.value.reduce((sum,x)=>sum+x.seconds,0))
async function run(action){busy.value=true;error.value='';try{await action()}catch(e){error.value=e.message||String(e)}finally{busy.value=false}}
async function load(){await run(async()=>{if(source.value==='cloud'&&props.client.user){const data=await props.client.request('/api/practice/stats');items.value=data.items||[]}else{source.value='local';items.value=await props.client.getLocalPractice()}})}
async function resume(item){await run(async()=>{await props.client.resumePractice(item);emit('play')})}
onMounted(load);watch(()=>props.client.user?.id,load)
</script>
<style scoped>
.practice-sources{display:flex;gap:8px}.refresh-history{margin-left:auto}.practice-summary{display:flex;gap:60px;margin:24px 0;padding:20px 0;border-top:1px solid #ffffff0e;border-bottom:1px solid #ffffff0e}.practice-summary>div{display:flex;flex-direction:column;gap:4px}.practice-summary strong{font-size:28px;font-weight:600}.practice-summary span{color:#aaa;font-size:13px}.practice-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(250px,1fr));gap:16px}.practice-grid .card{padding:20px;background:#303030;border-color:#ffffff12;gap:13px}.practice-grid .card h3{font-size:17px}.practice-grid .card button{align-self:flex-start;margin-top:4px}@media(max-width:600px){.practice-summary{gap:26px}.practice-summary strong{font-size:25px}}
@media(max-height:520px){.practice-summary{display:none}.practice-workspace .catalog-heading{margin-bottom:0}.practice-grid .card{display:grid;grid-template-columns:minmax(0,1fr) auto;align-items:center;gap:6px 16px;padding:16px}.practice-grid .card>h3,.practice-grid .card>p{grid-column:1}.practice-grid .card>button{grid-column:2;grid-row:1/4;align-self:center}.practice-grid{grid-template-columns:1fr}}
</style>
