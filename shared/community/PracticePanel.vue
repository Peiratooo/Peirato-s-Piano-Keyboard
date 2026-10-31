<template>
 <section class="community practice-workspace" aria-label="我的练习"><header class="catalog-heading"><div><h1>我的练习</h1><p>每一次回到琴键前，都是一点进步</p></div><button :disabled="busy||!client.user" @click="load"><PhArrowClockwise :size="18"/>刷新</button></header><p v-if="error" class="error" role="alert">{{error}}</p>
  <div v-if="!client.user" class="catalog-empty"><PhChartBar :size="44" weight="duotone"/><h2>记录每一段进步</h2><p>登录后，在 App 和网页之间同步你的练习记录。</p><button class="primary" @click="$emit('account')">登录账号</button></div>
  <template v-else><div class="practice-summary"><div><strong>{{items.length}}</strong><span>练习曲目</span></div><div><strong>{{Math.floor(seconds/60)}}</strong><span>有效练习分钟</span></div><div><strong>{{items.reduce((sum,item)=>sum+item.sessions,0)}}</strong><span>练习次数</span></div></div><div class="practice-grid"><article v-for="item in items" :key="item.midiId" class="card"><h3>{{item.title}}</h3><p class="muted">{{item.sessions}} 次练习 · {{Math.floor(item.seconds/60)}} 分 {{item.seconds%60}} 秒</p><p class="muted">最近练习 {{new Date(item.lastPracticed).toLocaleString('zh-CN')}}</p><button :disabled="busy" @click="resume(item)"><PhPlay/>继续练习</button></article></div><div v-if="!items.length" class="catalog-empty"><PhPianoKeys :size="44" weight="duotone"/><h2>从一首曲子开始</h2><p>在工坊选择一首 MIDI 开始跟弹，练习记录会自动保存在这里。</p></div></template>
 </section>
</template>
<script setup>
import {ref,computed,onMounted,watch} from 'vue'
import {PhArrowClockwise,PhChartBar,PhPlay,PhPianoKeys} from '@phosphor-icons/vue'
import './community.css'
import './workshop.css'
const props=defineProps({client:{type:Object,required:true}}),emit=defineEmits(['play','account'])
const items=ref([]),busy=ref(false),error=ref(''),seconds=computed(()=>items.value.reduce((sum,x)=>sum+x.seconds,0))
async function run(action){busy.value=true;error.value='';try{await action()}catch(e){error.value=e.message||String(e)}finally{busy.value=false}}
async function load(){if(!props.client.user){items.value=[];return}await run(async()=>{const data=await props.client.request('/api/practice/stats');items.value=data.items||[]})}
async function resume(item){await run(async()=>{const song=await props.client.request('/api/midis/'+item.midiId);await props.client.loadSong(song,item.lastMode||'follow');emit('play')})}
onMounted(load);watch(()=>props.client.user?.id,load)
</script>
<style scoped>
.practice-summary{display:flex;gap:60px;margin:5px 0 32px;padding:22px 0;border-top:1px solid #ffffff0e;border-bottom:1px solid #ffffff0e}.practice-summary>div{display:flex;flex-direction:column;gap:4px}.practice-summary strong{font-size:30px;font-weight:600}.practice-summary span{color:#949ba5;font-size:13px}.practice-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:20px}.practice-grid .card{padding:22px;background:#292b2e;border-color:#ffffff12;gap:15px}.practice-grid .card h3{font-size:19px}.practice-grid .card button{align-self:flex-start;margin-top:5px}@media(max-width:600px){.practice-summary{gap:26px}.practice-summary strong{font-size:26px}}
</style>
