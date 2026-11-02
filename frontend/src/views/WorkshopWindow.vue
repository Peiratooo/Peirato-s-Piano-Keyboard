<template>
 <main class="workshop-window" :style="theme.cssVariables">
  <CommunityWorkspace :client="community" :session="session" :theme-style="theme.cssVariables" :initial-view="view" logo-url="/appicon.png" @piano="returnToPiano" @account="accountOpen=true" @settings="Keyboard.OpenControlCenter()" @toggle-playback="togglePlayback" @practice="startPractice"/>
  <NModal v-model:show="accountOpen"><AccountDialog :client="community" :theme-style="theme.cssVariables" @close="accountOpen=false"/></NModal>
 </main>
</template>
<script setup>
import {computed,inject,onBeforeUnmount,onMounted,ref} from 'vue'
import {useRoute} from 'vue-router'
import {NModal} from 'naive-ui'
import {Events} from '@wailsio/runtime'
import CommunityWorkspace from '../../../shared/community/CommunityWorkspace.vue'
import AccountDialog from '../../../shared/community/AccountDialog.vue'
import {useMidiSession} from '../services/midiSession'
import {useDesktopCommunity} from '../services/community'
import {getPianoUiTheme} from '../services/uiTheme'
const store=inject('store'),Keyboard=inject('Keyboard'),route=useRoute()
const view=ref(route.query.view||'workshop'),accountOpen=ref(false)
const theme=computed(()=>getPianoUiTheme(store.config.colors)),session=useMidiSession({store,Keyboard})
const community=useDesktopCommunity({Keyboard,session,changeMode:mode=>session.setMode(mode)})
async function togglePlayback(){if(session.mode!=='play')await session.setMode('play');await session.togglePlayback()}
async function returnToPiano(){await Keyboard.ReturnToPiano()}
async function startPractice(){await session.setMode('follow');Events.Emit('mainMidiModeRequested',{mode:'follow',chooseSong:true});await Keyboard.ReturnToPiano()}
async function enter(next){view.value=next;if(session.mode!=='play'){if(session.state.status==='playing')await session.togglePlayback();await session.setMode('play')}}
const off=Events.On('communityNavigate',event=>void enter(event.data))
onMounted(async()=>{await community.restoreSession();await session.initialize();await enter(view.value)})
onBeforeUnmount(()=>{off();session.dispose()})
</script>
<style scoped>.workshop-window{height:100dvh;position:relative;background:#2b2b2b}</style>
