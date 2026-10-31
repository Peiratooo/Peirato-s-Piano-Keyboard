import {computed,watch,onBeforeUnmount} from 'vue'

// Both players report the same lifecycle; no fabricated offline practice time.
export function useCommunityPractice(client,getPlayback,notify=()=>{}) {
 let sessionId=0,queue=Promise.resolve(),disposed=false
 async function finish(){if(sessionId){const id=sessionId;sessionId=0;await client.json('/api/practice/finish','POST',{sessionId:id})}}
 const playback=computed(getPlayback)
 const stop=watch([
  ()=>playback.value.midiId,()=>playback.value.mode,()=>playback.value.status,
  ()=>client.user?.id,()=>!!playback.value.hidden,
 ],(now,previous=[])=>{
  const [midiId,mode,status,userId,hidden]=now
  const [oldId,oldMode,oldStatus,oldUser,oldHidden]=previous
  const playing=!!midiId&&status==='playing'&&mode!=='free'
  const freshPlay=playing&&(oldId!==midiId||oldMode!==mode||!['playing','paused'].includes(oldStatus))
  queue=queue.catch(()=>{}).then(async()=>{
   if(disposed)return
   if(freshPlay)await client.json(`/api/midis/${midiId}/events`,'POST',{type:'play'}).catch(()=>{})
   const recording=playing&&!!userId&&!hidden
   if(sessionId&&(!recording||oldId!==midiId||oldMode!==mode||oldUser!==userId))await finish().catch(()=>{})
   if(recording&&!sessionId){const result=await client.json('/api/practice/start','POST',{midiId,mode});sessionId=result.id}
  }).catch(e=>notify('云端练习记录暂不可用：'+e.message))
 },{immediate:true})
 const timer=setInterval(()=>{if(sessionId)queue=queue.catch(()=>{}).then(()=>sessionId?client.json('/api/practice/heartbeat','POST',{sessionId}).catch(()=>{}):undefined)},20000)
 client.beforeLogout=()=>{queue=queue.catch(()=>{}).then(finish);return queue.catch(()=>{})}
 const onPageHide=()=>{void client.beforeLogout()}
 if(typeof window!=='undefined')window.addEventListener('pagehide',onPageHide)
 onBeforeUnmount(()=>{stop();clearInterval(timer);disposed=true;void client.beforeLogout();window.removeEventListener('pagehide',onPageHide)})
}
