<template>
 <section ref="workspace" class="community-workspace" :class="{'native-titlebar':nativeTitlebar,'nav-open':navOpen}" :style="themeStyle" tabindex="-1" data-piano-control aria-label="钢琴工坊工作区">
  <button v-if="navOpen" class="workspace-nav-shade" aria-label="收起导航" @click="navOpen=false"></button>
  <aside class="workspace-sidebar" aria-label="主导航">
   <button class="workspace-brand" @click="$emit('piano')"><img :src="logoUrl" alt="" width="34" height="34"/><span>PP 钢琴键盘</span></button>
   <nav class="workspace-primary-nav"><button @click="$emit('piano')"><PhMusicNotes :size="23"/><span>演奏</span></button><button :class="{active:view==='workshop'}" :aria-current="view==='workshop'?'page':undefined" @click="navigate('workshop')"><PhPianoKeys :size="23"/><span>MIDI 工坊</span></button><button :class="{active:view==='practice'}" :aria-current="view==='practice'?'page':undefined" @click="navigate('practice')"><PhChartBar :size="23"/><span>我的练习</span></button></nav>
   <div class="workspace-nav-caption">曲库</div>
   <nav class="workspace-library-nav"><button v-for="item in scopes" :key="item.id" :class="{active:view==='workshop'&&scope===item.id}" :aria-current="view==='workshop'&&scope===item.id?'page':undefined" @click="selectScope(item.id)"><component :is="item.icon" :size="23"/><span>{{item.name}}</span></button></nav>
   <nav v-if="slots.rooms||slots.admin&&client.user?.role==='admin'" class="workspace-extra-nav"><button v-if="slots.rooms" :class="{active:view==='rooms'}" @click="navigate('rooms')"><PhUsersThree :size="23"/><span>琴房</span></button><button v-if="slots.admin&&client.user?.role==='admin'" :class="{active:view==='admin'}" @click="navigate('admin')"><PhShieldCheck :size="23"/><span>内容审核</span></button></nav>
   <button class="workspace-settings" @click="navOpen=false;$emit('settings')"><PhGearSix :size="23"/><span>设置</span></button>
  </aside>
  <header class="workspace-topbar"><button class="workspace-mobile-menu" aria-label="展开主导航" :aria-expanded="navOpen" @click="navOpen=!navOpen"><PhList :size="24"/></button><span class="workspace-mobile-title">PP 钢琴键盘</span>
   <div class="workspace-top-actions"><button class="workspace-account" aria-label="账号" @click="$emit('account')"><img v-if="client.user?.avatarUrl" :src="client.API+client.user.avatarUrl" alt=""/><PhUserCircle v-else :size="34" weight="duotone"/><span>{{client.user?.nickname||'登录'}}</span></button><div v-if="slots['window-controls']" class="workspace-window-controls"><slot name="window-controls"/></div></div>
  </header>
  <main class="workspace-main" :inert="navOpen">
   <p v-if="client.offline" class="workspace-offline" role="status">社区服务暂时无法连接，本地曲库和演奏仍可使用。</p>
   <KeepAlive><WorkshopPanel v-if="view==='workshop'" v-model:scope="scope" :client="client" :theme-style="themeStyle" :playing-id="playingId" @play="played" @account="$emit('account')"/></KeepAlive>
   <PracticePanel v-if="view==='practice'" :client="client" @play="$emit('piano')" @account="$emit('account')"/>
   <slot v-if="view==='rooms'" name="rooms"/>
   <slot v-if="view==='admin'" name="admin"/>
  </main>
  <WorkspacePlayer class="workspace-bottom-player" :session="session" :composer="client.playingSong?.id===playingId?client.playingSong.composer||'':''" @piano="$emit('piano')" @toggle="$emit('toggle-playback')"/>
 </section>
</template>
<script setup>
import {computed,nextTick,onActivated,onMounted,ref,useSlots,watch} from 'vue'
import {PhMusicNotes,PhPianoKeys,PhChartBar,PhGlobe,PhUploadSimple,PhBookmarkSimple,PhGearSix,PhUserCircle,PhList,PhUsersThree,PhShieldCheck} from '@phosphor-icons/vue'
import WorkshopPanel from './WorkshopPanel.vue'
import PracticePanel from './PracticePanel.vue'
import WorkspacePlayer from './WorkspacePlayer.vue'
import './workspace.css'
const props=defineProps({client:{type:Object,required:true},session:{type:Object,required:true},themeStyle:{type:Object,required:true},initialView:{type:String,default:'workshop'},logoUrl:{type:String,required:true},nativeTitlebar:{type:Boolean,default:false}})
const emit=defineEmits(['piano','account','settings','toggle-playback'])
const slots=useSlots(),workspace=ref(null),view=ref(props.initialView),scope=ref('public'),navOpen=ref(false)
const playingId=computed(()=>props.session.selectedSong?.communityMidiId||props.session.selectedSong?.serverMidiId||'')
const scopes=[{id:'public',name:'公开工坊',icon:PhGlobe},{id:'mine',name:'我的上传',icon:PhUploadSimple},{id:'favorites',name:'我的收藏',icon:PhBookmarkSimple}]
function navigate(next){view.value=next;navOpen.value=false}
function selectScope(next){if(next!=='public'&&!props.client.user){navOpen.value=false;emit('account');return}scope.value=next;navigate('workshop')}
function played(mode){if(mode==='follow')emit('piano')}
watch(()=>props.initialView,next=>navigate(next))
watch(()=>props.client.user?.id,()=>{if(!props.client.user){scope.value='public';if(view.value==='admin')view.value='workshop'}})
const focusWorkspace=()=>nextTick(()=>workspace.value?.focus({preventScroll:true}))
onMounted(focusWorkspace);onActivated(()=>{navigate(props.initialView);focusWorkspace()})
</script>
