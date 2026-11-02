<template>
 <section ref="workspace" class="community-workspace" :class="{'native-titlebar':nativeTitlebar,'nav-open':navOpen}" :style="themeStyle" tabindex="-1" data-piano-control aria-label="钢琴工坊工作区">
  <button v-if="navOpen" class="workspace-nav-shade" aria-label="收起导航" @click="navOpen=false"></button>
  <aside class="workspace-sidebar" aria-label="主导航">
   <button class="workspace-brand" @click="$emit('piano')"><img :src="logoUrl" alt="" width="28" height="28"/><span>PP 钢琴键盘</span></button>
   <nav class="workspace-primary-nav">
    <button :class="{active:view==='workshop'}" @click="navigate('workshop')"><PhGlobe :size="21"/><span>工坊</span></button>
    <button :class="{active:view==='practice'}" @click="navigate('practice')"><PhPianoKeys :size="21"/><span>练习</span></button>
    <button :class="{active:view==='mine'}" @click="navigate('mine')"><PhUserCircle :size="21"/><span>我的</span></button>
   </nav>
   <nav v-if="slots.rooms||slots.admin&&client.user?.role==='admin'" class="workspace-extra-nav"><button v-if="slots.rooms" :class="{active:view==='rooms'}" @click="navigate('rooms')"><PhUsersThree :size="21"/><span>琴房</span></button><button v-if="slots.admin&&client.user?.role==='admin'" :class="{active:view==='admin'}" @click="navigate('admin')"><PhShieldCheck :size="21"/><span>内容审核</span></button></nav>
   <button class="workspace-settings" @click="navOpen=false;$emit('settings')"><PhGearSix :size="21"/><span>设置</span></button>
  </aside>
  <header class="workspace-topbar"><button class="workspace-mobile-menu" aria-label="展开主导航" :aria-expanded="navOpen" @click="navOpen=!navOpen"><PhList :size="24"/></button><button class="workspace-back" @click="$emit('piano')"><PhArrowLeft :size="18"/><span>返回钢琴</span></button>
   <div class="workspace-top-actions"><button class="workspace-account" aria-label="账号" @click="$emit('account')"><img v-if="client.user?.avatarUrl" :src="client.API+client.user.avatarUrl" alt=""/><PhUserCircle v-else :size="30"/><span>{{client.user?.nickname||'登录'}}</span></button><div v-if="slots['window-controls']" class="workspace-window-controls"><slot name="window-controls"/></div></div>
  </header>
  <main class="workspace-main" :inert="navOpen">
   <p v-if="client.offline" class="workspace-offline" role="status">工坊暂时无法连接。你仍可以使用本地曲库练习。</p>
   <div v-if="view==='mine'&&!client.user" class="workspace-mine-heading"><h1>我的</h1><p>登录后管理上传的作品与收藏。</p><button @click="$emit('account')">登录账号</button></div>
   <KeepAlive><WorkshopPanel v-if="view==='workshop'||view==='mine'&&client.user" v-model:scope="scope" :client="client" :theme-style="themeStyle" :playing-id="playingId" @play="played" @account="$emit('account')"/></KeepAlive>
   <PracticePanel v-if="view==='practice'" :client="client" @play="$emit('piano')" @practice="$emit('practice')" @account="$emit('account')"/>
   <slot v-if="view==='rooms'" name="rooms"/><slot v-if="view==='admin'" name="admin"/>
  </main>
  <WorkspacePlayer class="workspace-bottom-player" :session="session" :composer="client.playingSong?.id===playingId?client.playingSong.composer||'':''" @piano="$emit('piano')" @toggle="$emit('toggle-playback')"/>
 </section>
</template>
<script setup>
import {computed,nextTick,onActivated,onMounted,ref,useSlots,watch} from 'vue'
import {PhArrowLeft,PhPianoKeys,PhGlobe,PhGearSix,PhUserCircle,PhList,PhUsersThree,PhShieldCheck} from '@phosphor-icons/vue'
import WorkshopPanel from './WorkshopPanel.vue'
import PracticePanel from './PracticePanel.vue'
import WorkspacePlayer from './WorkspacePlayer.vue'
import './workspace.css'
const props=defineProps({client:{type:Object,required:true},session:{type:Object,required:true},themeStyle:{type:Object,required:true},initialView:{type:String,default:'workshop'},logoUrl:{type:String,required:true},nativeTitlebar:{type:Boolean,default:false}})
const emit=defineEmits(['piano','account','settings','toggle-playback','practice'])
const slots=useSlots(),workspace=ref(null),view=ref(props.initialView),scope=ref(props.initialView==='mine'?'mine':'public'),navOpen=ref(false)
const playingId=computed(()=>props.session.selectedSong?.communityMidiId||props.session.selectedSong?.serverMidiId||'')
function navigate(next){view.value=next;navOpen.value=false;if(next==='workshop')scope.value='public';else if(next==='mine')scope.value='mine'}
function played(mode){if(mode==='follow')emit('piano')}
watch(()=>props.initialView,navigate)
watch(()=>props.client.user?.id,()=>{if(!props.client.user){scope.value='public';if(view.value==='admin')view.value='workshop'}else if(view.value==='mine')scope.value='mine'})
const focusWorkspace=()=>nextTick(()=>workspace.value?.focus({preventScroll:true}))
onMounted(focusWorkspace);onActivated(()=>{navigate(props.initialView);focusWorkspace()})
</script>
