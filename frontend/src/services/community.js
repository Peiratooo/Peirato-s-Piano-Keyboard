import {reactive,onBeforeUnmount} from 'vue'
import {Browser,Events} from '@wailsio/runtime'

export function useDesktopCommunity({Keyboard,session,changeMode}) {
 let timer=null,polling=false
 const client=reactive({API:'',user:null,offline:false,desktop:true,desktopGrant:null,playingSong:null})
 async function request(path,options={}) {
  const data=await Keyboard.CommunityRequest(path,options.method||'GET',options.body||'')
  client.offline=false
  return JSON.parse(data)
 }
 const json=(path,method,data)=>request(path,{method,body:JSON.stringify(data)})
 function stopPolling(){if(timer)clearInterval(timer);timer=null;client.desktopGrant=null}
 async function restoreSession(){const state=await Keyboard.GetCommunitySession();Object.assign(client,state,{API:state.api});await session.refreshLibrary()}
 async function auth(action,username,password,remember){stopPolling();const state=await Keyboard.CommunityLogin(action,username,password,remember);Object.assign(client,state,{API:state.api});await session.refreshLibrary();Events.Emit('communityAccountChanged');return client.user}
 async function logout(){await client.beforeLogout?.();try{await Keyboard.CommunityLogout()}finally{stopPolling();client.user=null;Events.Emit('communityAccountChanged');await session.stop();await session.refreshLibrary()}}
 async function beginToyLogin(remember){
  stopPolling();client.desktopGrant=await Keyboard.BeginCommunityToyLogin(remember)
  await Browser.OpenURL(client.desktopGrant.verificationUrl)
  timer=setInterval(async()=>{
   if(polling)return
   polling=true
   try{
    const result=await Keyboard.PollCommunityToyLogin()
    if(!result.pending){Object.assign(client,result,{API:result.api});stopPolling();await session.refreshLibrary();window.$message?.success('已通过 Toy 登录')}
   }catch(e){stopPolling();window.$message?.error(String(e.message||e))}finally{polling=false}
  },5000)
 }
 async function cancelToyLogin(){stopPolling();await Keyboard.CancelCommunityToyLogin()}
 const openToyLogin=()=>Browser.OpenURL(client.desktopGrant.verificationUrl)
 async function changeAvatar(){const user=await Keyboard.SetCommunityAvatar();if(user)client.user=user}
 const chooseMIDI=()=>Keyboard.ChooseCommunityMIDI()
 const upload=async(file,fields)=>JSON.parse(await Keyboard.UploadCommunityMIDI(file.path,JSON.stringify(fields)))
 async function loadSong(song,mode){
  const item=await Keyboard.LoadCommunityMIDI(song.id)
  await session.initialize();await session.refreshLibrary()
  if(!await session.selectSong(item.id)||!await changeMode(mode)||!await session.start())throw Error('无法开始播放，请检查 MIDI 或音源设置')
  client.playingSong=song
 }
 async function resumePractice(item){
  await session.initialize();await session.refreshLibrary()
  if(session.library.some(song=>song.id===item.localId)){
   if(!await session.selectSong(item.localId)||!await changeMode(item.lastMode||'follow')||!await session.start())throw Error('无法载入这首曲目，请重新选择')
  }else if(item.midiId){await loadSong(await request('/api/midis/'+item.midiId),item.lastMode||'follow')}
  else throw Error('本地文件已移除，请重新导入')
 }
 const getLocalPractice=()=>Keyboard.GetLocalPractice()
 const download=song=>Keyboard.DownloadCommunityMIDI(song.id)
 Object.assign(client,{request,json,restoreSession,auth,logout,beginToyLogin,cancelToyLogin,openToyLogin,changeAvatar,chooseMIDI,upload,loadSong,download,getLocalPractice,resumePractice})
 const offAccount=Events.On('communityAccountChanged',()=>void restoreSession())
 const unsubscribe=Events.On('communitySessionExpired',()=>{client.user=null;if(session.selectedSong?.communityAccountId)void session.stop();void session.refreshLibrary()})
 onBeforeUnmount(()=>{stopPolling();unsubscribe();offAccount();void Keyboard.CancelCommunityToyLogin()})
 return client
}
