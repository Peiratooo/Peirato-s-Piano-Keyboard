import {computed, ref, onBeforeUnmount} from 'vue'
import {saveSong,getSongs,getCommunityFile,saveCommunityFile} from './storage'
import {pongSample, bestClockOffset, remoteEventDelay} from './roomTiming'

// Set VITE_API_BASE_URL to a TLS endpoint allowed to serve the Bilibili Toy origin.
// Own-account Bearer sessions are intentionally separate from Toy toyOpenId:
// toyOpenId is not a service authentication credential.
const API = String(import.meta.env.VITE_API_BASE_URL || 'http://127.0.0.1:8080').replace(/\/$/, '')
const WS = API.replace(/^http/, 'ws')

export function useCommunity(piano, notify = () => {}) {
  const token = ref(sessionStorage.getItem('pp-piano-account-token') || '')
  const user = ref(null), playingSong = ref(null)
  const room = ref(null)
  const connected = ref(false)
  const audience = ref(0)
  const queue = ref([])
  const performerId = ref(0)
  const performerName = ref('')
  const turnEndsAt = ref('')
  const clockOffsetMs = ref(0)
  const latestInvite = ref('')
  const errorText = ref('')
  const connectionState = ref('idle') // idle / connecting / connected / reconnecting
  const reconnectAttempts = ref(0)
  const latencyMs = ref(null)
  let socket = null, checkTimer = null, reconnectTimer = null
  let wantedRoom = null, generation = 0, lastSequence = 0, joinedOnce = false
  const pendingAudio = new Set(), rttSamples = []
  const inRoom = computed(() => !!room.value && connectionState.value !== 'idle')
  const isPerformer = computed(() => connected.value && inRoom.value && !!user.value &&
    (room.value.id === 'lobby' ? performerId.value === user.value.id : room.value.ownerId === user.value.id))
  const myPosition = computed(() => queue.value.find(x => x.userId === user.value?.id)?.position || 0)
  const remainingSeconds = ref(0)

  async function request(path, options = {}) {
    const headers = {...options.headers}
    if (token.value) headers.Authorization = `Bearer ${token.value}`
    const res = await fetch(API + path, {...options, headers, cache:'no-store'})
    if (!res.ok) {
      if(res.status===401)clearAccount()
      let message = `HTTP ${res.status}`
      try { message = (await res.json()).error || message } catch {}
      throw Error(message)
    }
    if (res.status === 204) return null
    return (res.headers.get('content-type') || '').includes('application/json') ? res.json() : res.blob()
  }
  const json = (path, method, payload) => request(path, {
    method, headers: {'Content-Type':'application/json'}, body: JSON.stringify(payload), keepalive:path==='/api/practice/finish',
  })

  function clearAccount(){
    token.value='';user.value=null;sessionStorage.removeItem('pp-piano-account-token')
    const wasPrivate=!!piano.selected?.ownerAccountId
    piano.library=piano.library.filter(song=>!song.ownerAccountId)
    if(wasPrivate){piano.stop();piano.selectSong(piano.library[0])}
  }
  async function loadAccountSongs(uid){const songs=await getSongs();piano.library=[...piano.library.filter(song=>!song.ownerAccountId),...songs.filter(song=>song.ownerAccountId===uid)]}
  async function auth(action, username, password) {
    const answer = await json('/api/auth/' + action, 'POST', {username,password})
    token.value = answer.token
    sessionStorage.setItem('pp-piano-account-token', answer.token)
    user.value = answer.user
    await loadAccountSongs(answer.user.id)
    return answer.user
  }
  async function restoreSession() {
    if (!token.value) return
    try { user.value = await request('/api/auth/me'); await loadAccountSongs(user.value.id) }
    catch(e) { notify(e.message) }
  }
  async function logout() {
    await client.beforeLogout?.()
    await request('/api/auth/logout', {method:'POST'}).finally(()=>{leave();piano.stop();clearAccount()})
  }
  function clearRemote() { piano.clearRemoteNotes() }
  function clearPendingAudio() {
    for (const timer of pendingAudio) clearTimeout(timer)
    pendingAudio.clear()
  }
  function stopConnection() {
    if (checkTimer) clearInterval(checkTimer)
    if (reconnectTimer) clearTimeout(reconnectTimer)
    checkTimer = reconnectTimer = null
    clearPendingAudio()
    const old = socket; socket = null
    if (old) {
      if (old.readyState === WebSocket.OPEN) {
        try { old.send(JSON.stringify({type:'room.leave'})) } catch {}
      }
      old.close(1000, 'Leave room')
    }
  }
  function leave() {
    generation++
    wantedRoom = null; joinedOnce = false; lastSequence = 0
    stopConnection(); clearRemote(); piano.releaseAllUsers()
    connected.value = false; connectionState.value = 'idle'; reconnectAttempts.value = 0
    room.value = null; audience.value = 0; queue.value = []; performerId.value = 0
    turnEndsAt.value = ''; remainingSeconds.value = 0
  }
  function send(message) {
    if (socket?.readyState !== WebSocket.OPEN || !connected.value) return false
    socket.send(JSON.stringify(message))
    return true
  }
  function updateCountdown() {
    remainingSeconds.value = turnEndsAt.value ?
      Math.max(0, Math.ceil((Date.parse(turnEndsAt.value) - (Date.now() + clockOffsetMs.value))/1000)) : 0
  }
  // A short jitter buffer aligns audible and visual note events across clients.
  // Use the server timestamp, corrected by the lowest-RTT clock observation.
  function queueRemoteEvent(message, action) {
    if (document.hidden) return
    const delay = remoteEventDelay(message.serverTime, clockOffsetMs.value)
    const timer = setTimeout(() => { pendingAudio.delete(timer); if (connected.value && !isPerformer.value) action() }, delay)
    pendingAudio.add(timer)
  }
  function applySnapshot(message) {
    if (!message.room || !wantedRoom || message.room.id !== wantedRoom.roomId) return
    const previousPerformer = isPerformer.value
    room.value = message.room; connected.value = true; connectionState.value = 'connected'
    reconnectAttempts.value = 0
    audience.value = message.audience; queue.value = message.queue || []
    performerId.value = message.performerId; performerName.value = message.performerName || ''
    turnEndsAt.value = message.turnEndsAt || ''
    if (message.serverTime && !rttSamples.length) clockOffsetMs.value = Date.parse(message.serverTime) - Date.now()
    if (Number.isSafeInteger(message.seq)) lastSequence = message.seq
    clearPendingAudio()
    if (previousPerformer && !isPerformer.value) piano.releaseAllUsers()
    if (isPerformer.value) clearRemote()
    else if (!document.hidden) piano.syncRemoteNotes(message.notes || [], !!message.sustain)
    else clearRemote()
    updateCountdown()
  }
  function handlePong(message) {
    const sample = pongSample(Number(message.clientTime), Date.now(), message.serverTime)
    if (!sample) return
    latencyMs.value = Math.round(sample.rtt / 2)
    rttSamples.push(sample)
    if (rttSamples.length > 8) rttSamples.shift()
    clockOffsetMs.value = bestClockOffset(rttSamples, clockOffsetMs.value)
    updateCountdown()
  }
  function scheduleReconnect() {
    if (!wantedRoom || reconnectTimer) return
    connectionState.value = 'reconnecting'
    reconnectAttempts.value++
    const delay = Math.min(12000, 500 * 2 ** Math.min(5, reconnectAttempts.value - 1))
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      if (wantedRoom && !document.hidden) void openSocket(false)
      else if (wantedRoom) scheduleReconnect()
    }, delay)
  }
  function openSocket(initial) {
    const epoch = generation
    const destination = wantedRoom
    return new Promise((resolve,reject) => {
      let settled = false
      const rejectOnce = error => { if (!settled) { settled = true; if (initial) reject(error); else notify(error.message) } }
      let ws
      try { ws = new WebSocket(WS + '/ws') } catch (e) { rejectOnce(e); if (!initial) scheduleReconnect(); return }
      socket = ws
      ws.onopen = () => {
        if (epoch !== generation) { ws.close(); return }
        ws.send(JSON.stringify({type:'join',roomId:destination.roomId,inviteCode:destination.inviteCode,token:token.value}))
        checkTimer = setInterval(() => {
          updateCountdown()
          if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({type:'heartbeat',clientTime:Date.now()}))
        }, 10000)
      }
      ws.onmessage = event => {
        if (epoch !== generation || ws !== socket) return
        let message; try { message = JSON.parse(event.data) } catch { return }
        if (message.type === 'error') {
          errorText.value = message.error
          // Invalid invitation / deleted room: do not reconnect forever.
          wantedRoom = null
          rejectOnce(Error(message.error))
          ws.close();return
        }
        if (message.type === 'pong') { handlePong(message); return }
        if (message.type === 'snapshot') {
          // A snapshot is authoritative, including held notes and pedal state.
          applySnapshot(message)
          joinedOnce = true
          if (!settled) { settled = true; resolve(message.room) }
          return
        }
        if (!connected.value) return
        if (Number.isSafeInteger(message.seq)) {
          if (message.seq <= lastSequence) return
          lastSequence = message.seq
        }
        if (message.type === 'clear') { clearPendingAudio(); clearRemote() }
        else if (message.type === 'note.on' && !isPerformer.value) queueRemoteEvent(message, () => piano.remoteNoteOn(message.note,message.velocity))
        else if (message.type === 'note.off' && !isPerformer.value) queueRemoteEvent(message, () => piano.remoteNoteOff(message.note))
        else if (message.type === 'pedal' && !isPerformer.value) queueRemoteEvent(message, () => piano.remotePedal(message.on))
        else if (message.type === 'room.closed') { notify('琴房已关闭'); leave() }
      }
      ws.onerror = () => { if (initial && !settled) rejectOnce(Error('WebSocket 连接失败，请检查网络或服务地址')) }
      ws.onclose = () => {
        if (epoch !== generation || ws !== socket) return
        socket = null
        if (checkTimer) clearInterval(checkTimer)
        checkTimer = null
        clearPendingAudio(); clearRemote()
        const hadSeat = isPerformer.value
        connected.value = false
        if (hadSeat) piano.releaseAllUsers()
        if (wantedRoom && joinedOnce) scheduleReconnect()
        else {
          connectionState.value = 'idle'
          room.value = null
          wantedRoom = null
          rejectOnce(Error('未能进入琴房'))
        }
      }
    })
  }
  async function join(roomId = 'lobby', inviteCode = '') {
    leave()
    piano.changeMode('free')
    piano.displayMode = 'waterfall'
    await piano.unlock() // user gesture required in iOS / Toy
    wantedRoom = {roomId: roomId || 'lobby', inviteCode}
    rttSamples.length = 0; latencyMs.value = null; clockOffsetMs.value = 0
    errorText.value = ''
    connectionState.value = 'connecting'
    const result = await openSocket(true)
    return result
  }
  function onVisibilityChange() {
    if (document.hidden) { clearPendingAudio(); clearRemote(); return }
    if (connected.value) send({type:'room.sync'})
    else if (wantedRoom && !socket) {
      if (reconnectTimer) clearTimeout(reconnectTimer)
      reconnectTimer = null
      void openSocket(false)
    }
  }
  if (typeof document !== 'undefined') document.addEventListener('visibilitychange', onVisibilityChange)
  const queueUp = () => send({type:'queue.join'})
  const leaveQueue = () => send({type:'queue.leave'})
  const yieldSeat = () => send({type:'seat.leave'})
  async function createRoom(title, visibility) {
    await piano.unlock()
    const result = await json('/api/rooms', 'POST', {title,visibility})
    latestInvite.value = result.inviteCode || ''
    await join(result.room.id)
    return result
  }
  async function randomRoom() {await piano.unlock();const {roomId} = await request('/api/rooms/random');return join(roomId)}
  async function upload(file, fields) {
    const data = new FormData();data.append('file',file)
    for(const [key,value] of Object.entries(fields))data.append(key,String(value))
    return request('/api/midis',{method:'POST',body:data})
  }
  function pickFile(accept){return new Promise(resolve=>{const input=document.createElement('input');input.type='file';input.accept=accept;input.onchange=()=>resolve(input.files?.[0]||null);input.oncancel=()=>resolve(null);input.click()})}
  const chooseMIDI=()=>pickFile('.mid,.midi')
  async function changeAvatar(){const file=await pickFile('.png,.jpg,.jpeg');if(!file)return;const data=new FormData();data.append('file',file);user.value=await request('/api/auth/avatar',{method:'POST',body:data})}
  async function loadSong(midi,mode='follow') {
    await piano.unlock()
    if(inRoom.value)leave()
    const ownerAccountId=midi.public&&midi.reviewStatus==='approved'?0:user.value?.id||0
    const key=`${ownerAccountId}:${midi.id}:${midi.sha256}`
    let blob=await getCommunityFile(key)
    if(!blob){blob=await request(`/api/midis/${midi.id}/file`);await saveCommunityFile(key,blob)}
    const old=piano.library.find(song=>song.serverMidiId===midi.id&&song.ownerAccountId===ownerAccountId&&song.communitySha256===midi.sha256)
    if(old){old.name=midi.title;piano.selectSong(old);await saveSong(old)}
    else{
      const file=new File([blob],midi.title+'.mid',{type:'audio/midi'})
      if(!await piano.importMIDI(file))throw Error('无法载入 MIDI')
      piano.selectSong({...piano.selected,serverMidiId:midi.id,name:midi.title,ownerAccountId,communitySha256:midi.sha256})
      piano.library=[piano.selected,...piano.library.filter(song=>song.id!==piano.selected.id&&(song.serverMidiId!==midi.id||song.ownerAccountId!==ownerAccountId))]
      await saveSong(piano.selected)
    }
    piano.changeMode(mode)
    if(!await piano.play())throw Error('无法开始播放，请先启用声音')
    playingSong.value=midi
    return true
  }
  async function download(midi){
    const blob=await request(`/api/midis/${midi.id}/file`)
    const url=URL.createObjectURL(blob),link=document.createElement('a');link.href=url;link.download=midi.title.replace(/[\\/:]/g,'_')+'.mid';document.body.append(link);link.click();link.remove();setTimeout(()=>URL.revokeObjectURL(url),1000)
    await json(`/api/midis/${midi.id}/events`,'POST',{type:'download'})
    return true
  }
  const unsubscribe = piano.subscribe(e => {
    if (!inRoom.value || !isPerformer.value || e.remote || e.source!=='user') return
    if(e.type === 'on' || e.type === 'off') send({type:`note.${e.type}`,note:e.note,velocity:e.velocity||80})
  })
  onBeforeUnmount(() => {unsubscribe();leave();document.removeEventListener('visibilitychange', onVisibilityChange)})
  const client={API,token,user,playingSong,room,connected,audience,queue,performerId,performerName,turnEndsAt,
    remainingSeconds,inRoom,isPerformer,myPosition,latestInvite,errorText,connectionState,reconnectAttempts,latencyMs,
    request,json,auth,restoreSession,logout,join,leave,send,queueUp,leaveQueue,yieldSeat,createRoom,randomRoom,upload,loadSong,download,chooseMIDI,changeAvatar}
  return client
}
