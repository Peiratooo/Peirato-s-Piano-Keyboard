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
 async function restoreSession(){Object.assign(client,await Keyboard.GetCommunitySession());await session.refreshLibrary()}
 async function auth(action,username,password,remember){stopPolling();Object.assign(client,await Keyboard.CommunityLogin(action,username,password,remember));await session.refreshLibrary();return client.user}
 async function logout(){await client.beforeLogout?.();try{await Keyboard.CommunityLogout()}finally{stopPolling();client.user=null;await session.stop();await session.refreshLibrary()}}
 async function beginToyLogin(remember){
  stopPolling();client.desktopGrant=await Keyboard.BeginCommunityToyLogin(remember)
  await Browser.OpenURL(client.desktopGrant.verificationUrl)
  timer=setInterval(async()=>{
   if(polling)return
   polling=true
   try{
    const result=await Keyboard.PollCommunityToyLogin()
    if(!result.pending){Object.assign(client,result);stopPolling();await session.refreshLibrary();window.$message?.success('已通过 Toy 登录')}
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
 const download=song=>Keyboard.DownloadCommunityMIDI(song.id)
 Object.assign(client,{request,json,restoreSession,auth,logout,beginToyLogin,cancelToyLogin,openToyLogin,changeAvatar,chooseMIDI,upload,loadSong,download})
 const unsubscribe=Events.On('communitySessionExpired',()=>{client.user=null;if(session.selectedSong?.communityAccountId)void session.stop();void session.refreshLibrary()})
 onBeforeUnmount(()=>{stopPolling();unsubscribe();void Keyboard.CancelCommunityToyLogin()})
 return client
}
