import {computed,reactive,ref,shallowRef,onBeforeUnmount} from 'vue'
import {PianoAudio} from '../services/audio'
import {encodeRecording,exampleSong,parseMIDI} from '../services/midi'
import {parseMIDIFile,closeMidiWorker} from '../services/midiImport'
import {pitchName} from '../services/keyboardLayout'

const clamp=(v,a,b)=>Math.max(a,Math.min(b,v))
export function usePiano(notify=()=>{}) {
  const audio=new PianoAudio()
  const mode=ref('free')
  const displayMode=ref('waterfall')
  const library=ref([exampleSong()])
  const selected=shallowRef(library.value[0])
  const volume=ref(80), velocity=ref(80)
  const handMode=ref('both'),speed=ref(1),loop=ref(true)
  const range=reactive({a:0,b:selected.value.durationMs})
  const state=reactive({status:'idle',currentMs:0,leadInMs:0,waiting:false,step:null,wrongNote:-1})
  const pressed=reactive({}),pressedHands=reactive({}),playbackHands=reactive({}),heldSources=new Map()
  const recording=reactive({status:'idle',startedAt:0,events:[],durationMs:0,noteCount:0,backupSaved:false})
  const recordingPreviewing=ref(false)
  let recordingPreviewTimer
  const listeners=new Set()
  const fire=(event)=>{for(const fn of listeners)fn(event)}
  let baseTime=0,baseMs=0,leadStarted=0,tickId=null,stepIndex=0,steps=[],visualCursor=0,audioCursor=0,active=new Map()
  const clockNow=(fallback=performance.now())=>audio.context?audio.context.currentTime*1000:fallback
  const noteIndexAt=ms=>{
    const all=notes();let low=0,high=all.length
    while(low<high){const mid=(low+high)>>1;if(all[mid].startMs<ms)low=mid+1;else high=mid}
    return low
  }
  function resetCursors(ms){visualCursor=audioCursor=noteIndexAt(ms)}
  const notes=()=>selected.value?.notes||[]
  const duration=computed(()=>selected.value?.durationMs||0)
  const statusText=computed(()=>state.status==='paused'?'已暂停':state.leadInMs>0?'音符进入中':state.waiting?'等待按键':state.status==='playing'?(mode.value==='follow'?'跟弹练习中':'正在播放'):'已就绪')
  const activeSong=computed(()=>selected.value)
  let wasHidden=false

  async function unlock(){try{await audio.unlock();return true}catch(e){notify('音频启动失败：'+e.message);return false}}
  function putNote(note,source,down,vel=80,extra={}){
    if(source==='user'){
      if(down){pressed[note]=true;pressedHands[note]=extra.hand}
      else{delete pressed[note];delete pressedHands[note]}
    }else{
      const held=[...active.values()].filter(n=>n.note===note)
      if(held.length)playbackHands[note]=held.some(n=>n.hand==='right')?'right':held[0].hand
      else if(down)playbackHands[note]=extra.hand||'right'
      else delete playbackHands[note]
    }
    fire({type:down?'on':'off',note,velocity:vel,source,time:performance.now(),...extra})
  }
  function userNoteOn(note,vel=velocity.value,source='keyboard'){
    note=Number(note);if(!Number.isInteger(note)||note<0||note>127)return
    const key=`${source}:${note}`
    if(heldSources.has(key))return
    const first=!([...heldSources.keys()].some(k=>k.endsWith(`:${note}`)))
    heldSources.set(key,true)
    if(first){
      void unlock().then(ok=>{if(ok&&[...heldSources.keys()].some(k=>k.endsWith(`:${note}`)))audio.noteOn(note,vel,'user')})
      const hand=state.step?.notes.find(n=>n.note===note)?.hand||playbackHands[note]||'right'
      putNote(note,'user',true,vel,{hand})
      if(recording.status==='recording'){
        recording.events.push({type:'on',note,velocity:vel,ms:performance.now()-recording.startedAt})
        recording.noteCount++
        recording.backupSaved=false
      }
    }
    checkFollow(note)
  }
  function userNoteOff(note,source='keyboard'){
    note=Number(note);const key=`${source}:${note}`
    if(!heldSources.has(key))return
    heldSources.delete(key)
    // A single hardware/pointer/keyboard owner can release without cutting another.
    if(![...heldSources.keys()].some(k=>k.endsWith(`:${note}`))){
      audio.noteOff(note,'user');putNote(note,'user',false)
      if(recording.status==='recording'){
        recording.events.push({type:'off',note,ms:performance.now()-recording.startedAt})
        recording.backupSaved=false
      }
    }
  }
  function releaseSource(source){
    for(const key of [...heldSources.keys()])if(key.startsWith(source+':')){
      userNoteOff(Number(key.slice(source.length+1)),source)
    }
  }
  function releaseAllUsers(){
    for(const key of [...heldSources.keys()]){const sep=key.lastIndexOf(':');userNoteOff(Number(key.slice(sep+1)),key.slice(0,sep))}
  }
  function toggleSustain(on){
    if(audio.sustain===!!on)return
    audio.setSustain(on)
    if(recording.status==='recording'){
      recording.events.push({type:'pedal',value:on?127:0,ms:performance.now()-recording.startedAt})
      recording.backupSaved=false
    }
  }
  // Remote performance is read-only: never record or send these notes back to WS.
  const remoteHeld=new Set()
  function remoteNoteOn(note,vel=80){
    if(!Number.isInteger(note)||note<0||note>127)return
    if(remoteHeld.has(note))remoteNoteOff(note)
    remoteHeld.add(note);void unlock().then(ok=>{if(ok&&remoteHeld.has(note))audio.noteOn(note,vel,'remote')})
    putNote(note,'midi',true,vel,{remote:true})
  }
  function remoteNoteOff(note){
    if(!remoteHeld.has(note))return
    remoteHeld.delete(note);audio.noteOff(note,'remote')
    putNote(note,'midi',false,80,{remote:true})
  }
  function remotePedal(on){audio.setSustain(!!on)}
  function syncRemoteNotes(notes, pedal){
    const expected=new Map((notes||[]).filter(n=>Number.isInteger(n.note)&&n.note>=0&&n.note<=127).map(n=>[n.note,n.velocity]))
    for(const note of [...remoteHeld])if(!expected.has(note))remoteNoteOff(note)
    for(const [note,vel] of expected)if(!remoteHeld.has(note))remoteNoteOn(note,vel)
    remotePedal(pedal)
  }
  function clearRemoteNotes(){
    for(const note of [...remoteHeld])remoteNoteOff(note)
    audio.allOff('remote');audio.setSustain(false)
  }
  function setVolume(v){volume.value=clamp(Number(v),0,100);audio.setVolume(volume.value/100)}
  function setVelocity(v){velocity.value=clamp(Number(v),1,127)}
  async function importSF2(file){try{const message=await audio.loadSF2(file);notify('音源已载入 · '+message);return true}catch(e){notify('音源导入失败：'+e.message);return false}}

  function buildSteps(){
    steps=[]
    for(const n of notes()){
      if(n.startMs<range.a||n.startMs>=range.b)continue
      if(handMode.value!=='both'&&n.hand!==handMode.value)continue
      const last=steps.at(-1)
      if(last&&n.startMs-last.ms<=80)last.notes.push(n)
      else steps.push({ms:n.startMs,notes:[n],accepted:new Set()})
    }
    stepIndex=0
    state.step=steps[stepIndex]||null
  }
  function getClock(now=performance.now()) {
    if(state.status!=='playing'||state.waiting)return {ms:state.currentMs,lead:state.leadInMs}
    let elapsed=clockNow(now)-baseTime
    if(elapsed<leadStarted) return {ms:baseMs,lead:(leadStarted-elapsed)*speed.value}
    const ms=baseMs+(elapsed-leadStarted)*speed.value
    return {ms:clamp(ms,range.a,range.b),lead:0}
  }
  function visualClock(now=performance.now()){
    const {ms,lead}=getClock(now)
    return mode.value!=='free'&&state.status==='playing' ? ms-lead : ms
  }
  // Web Audio is the playback clock. Keep audio one interval ahead of the
  // visual playhead: main-thread stalls don't shift note attack timestamps.
  const audioLookaheadMs=110
  function scheduleAudio(ms){
    if(!audio.context||state.status!=='playing'||state.leadInMs>0)return
    const all=notes()
    let limit=Math.min(range.b,ms+audioLookaheadMs*speed.value)
    if(mode.value==='follow'&&steps[stepIndex])limit=Math.min(limit,steps[stepIndex].ms)
    const now=audio.context.currentTime
    while(audioCursor<all.length&&all[audioCursor].startMs<=limit+.001){
      const n=all[audioCursor++]
      if(n.startMs<range.a||n.startMs>=range.b||!shouldAccompany(n))continue
      const startAt=now+Math.max(0,n.startMs-ms)/speed.value/1000
      const end=Math.min(n.soundEndMs??n.endMs,range.b)
      const endAt=now+Math.max(0,end-ms)/speed.value/1000
      if(endAt>startAt)audio.scheduleMidiNote(n.note,Math.round(n.velocity*velocity.value/127),`midi-${n.id}`,startAt,endAt,n.program||0,n.gain??1)
    }
    audio.pruneScheduled(now)
  }
  function restoreAt(ms){
    // Resume the sound of notes held at the seek target. Key visuals end at
    // NoteOff, while soundEndMs can extend through the sustain pedal.
    const all=notes(),now=audio.context?.currentTime||0
    for(const n of all){
      if(n.startMs>=ms)break
      if((n.soundEndMs??n.endMs)<=ms||!shouldAccompany(n)||n.startMs>=range.b)continue
      const end=Math.min(n.soundEndMs??n.endMs,range.b)
      if(end>ms)audio.scheduleMidiNote(n.note,Math.round(n.velocity*velocity.value/127),`midi-${n.id}`,now,now+(end-ms)/speed.value/1000,n.program||0,n.gain??1)
      if(n.endMs>ms){active.set(n.id,n);putNote(n.note,'midi',true,n.velocity,{hand:n.hand})}
    }
    scheduleAudio(ms)
  }
  function clearPlaybackVoices(){
    active.clear()
    audio.allOff('midi') // cancels also future-scheduled voices
    for(const key of Object.keys(playbackHands))delete playbackHands[key]
  }
  function clearTick(){if(tickId){clearInterval(tickId);tickId=null}}
  function stop(){clearTick();clearPlaybackVoices();state.status='idle';state.waiting=false;state.leadInMs=0
    state.currentMs=range.a;state.step=null;state.wrongNote=-1;return true}
  function pause(){if(state.status!=='playing')return
    const clock=getClock();state.currentMs=clock.ms;state.leadInMs=clock.lead;state.status='paused';clearTick();clearPlaybackVoices()}
  function resume(){if(state.status!=='paused')return
    baseMs=state.currentMs;leadStarted=state.leadInMs/Math.max(.25,speed.value)
    baseTime=clockNow();state.status='playing';if(!leadStarted)restoreAt(baseMs);tickId=setInterval(tick,12)}
  function shouldAccompany(n){
    if(mode.value==='play')return handMode.value==='both'||n.hand===handMode.value
    return handMode.value==='left'?n.hand!=='left':handMode.value==='right'?n.hand!=='right':false
  }
  function tick(){
    if(state.status!=='playing')return
    const clock=getClock();let ms=clock.ms
    state.leadInMs=clock.lead
    if(clock.lead>0){state.currentMs=baseMs;return}
    if(mode.value==='follow'&&steps[stepIndex]){
      const next=steps[stepIndex]
      if(ms>=next.ms){ms=next.ms;state.waiting=true;state.step=next}
    }
    state.currentMs=ms
    // Audio is scheduled ahead; only the key visuals follow the current time.
    scheduleAudio(ms)
    for(const [key,n] of active){
      if(n.endMs<=ms+.001){active.delete(key);putNote(n.note,'midi',false)}
    }
    const all=notes()
    while(visualCursor<all.length&&all[visualCursor].startMs<=ms+.001){
      const n=all[visualCursor++]
      if(n.startMs>=range.a&&n.startMs<range.b&&shouldAccompany(n)){
        active.set(n.id,n)
        putNote(n.note,'midi',true,n.velocity,{hand:n.hand})
      }
    }
    if(ms>=range.b-.001){
      if(loop.value){
        clearPlaybackVoices();resetCursors(range.a);buildSteps()
        baseMs=range.a;baseTime=clockNow();leadStarted=0
        state.currentMs=range.a;state.waiting=false;state.leadInMs=0;restoreAt(range.a)
      }else stop()
    }
  }
  async function play(){
    if(!selected.value||!await unlock())return false
    if(state.status==='paused'){resume();return true}
    const requestedMs=state.currentMs
    stop()
    state.currentMs=clamp(requestedMs,range.a,range.b)
    if(mode.value==='follow'){buildSteps();if(!steps.length){notify('当前区间没有可练习的音符');return false}}
    baseMs=clamp(state.currentMs,range.a,range.b)
    // Full song starts get a 3s visual lead-in; A/B and seeking start immediately.
    const whole=range.a<=.5&&range.b>=duration.value-.5
    leadStarted=whole&&baseMs<=.5?3000:0
    baseTime=clockNow()
    state.leadInMs=leadStarted*speed.value;state.waiting=false;state.status='playing'
    resetCursors(baseMs)
    if(!leadStarted)restoreAt(baseMs)
    tickId=setInterval(tick,12)
    return true
  }
  function togglePlayback(){return state.status==='playing'?pause():state.status==='paused'?resume():play()}
  function seek(ms){
    const wasPlaying=state.status==='playing',wasPaused=state.status==='paused'
    clearPlaybackVoices();state.currentMs=clamp(Number(ms)||0,range.a,range.b)
    state.leadInMs=0;state.waiting=false
    buildSteps()
    if(mode.value==='follow'){
      stepIndex=steps.findIndex(s=>s.ms>=state.currentMs)
      if(stepIndex<0)stepIndex=steps.length
      state.step=steps[stepIndex]||null
    }
    resetCursors(state.currentMs)
    baseMs=state.currentMs;baseTime=clockNow();leadStarted=0
    if(wasPaused)state.status='paused'
    if(wasPlaying){state.status='playing';restoreAt(state.currentMs)}
  }
  function setRange(a,b){
    const dur=duration.value
    const nextA=clamp(Number(a)||0,0,Math.max(0,dur-1))
    const nextB=clamp(Number(b)||dur,nextA+1,dur)
    range.a=nextA;range.b=nextB
    if(state.status==='playing'||state.status==='paused')seek(clamp(state.currentMs,nextA,nextB))
    else state.currentMs=clamp(state.currentMs,nextA,nextB)
  }
  function changeMode(next){
    if(!['free','play','follow'].includes(next))return
    stop();mode.value=next
    if(next!=='free')displayMode.value='waterfall'
  }
  function setSpeed(value){
    const next=clamp(Number(value)||1,.25,5)
    if(next===speed.value)return
    const playing=state.status==='playing'
    if(playing){
      const now=getClock()
      baseMs=now.ms;baseTime=clockNow();leadStarted=now.lead/Math.max(.25,speed.value)
      clearPlaybackVoices();resetCursors(baseMs)
      state.currentMs=baseMs
    }
    speed.value=next
    if(playing){state.leadInMs=leadStarted*next;if(!leadStarted)restoreAt(baseMs)}
  }
  function setHand(next){handMode.value=next;if(state.status!=='idle')seek(state.currentMs)}
  function checkFollow(note){
    if(mode.value!=='follow'||state.status!=='playing'||!state.waiting)return
    const step=steps[stepIndex]
    if(!step)return
    if(!step.notes.some(n=>n.note===note)){
      state.wrongNote=note
      setTimeout(()=>{if(state.wrongNote===note)state.wrongNote=-1},350)
      return
    }
    step.accepted.add(note)
    if(step.notes.every(n=>step.accepted.has(n.note))){
      stepIndex++;state.step=steps[stepIndex]||null;state.waiting=false
      baseMs=state.currentMs;baseTime=clockNow();leadStarted=0
    }
  }
  function selectSong(song){stop();selected.value=song;range.a=0;range.b=song?.durationMs??0;state.currentMs=0;displayMode.value='waterfall'}
  async function importMIDI(file){
    if(!file||file.size>10*1024*1024){notify('MIDI 文件请小于 10 MB');return false}
    try{
      const song=await parseMIDIFile(file)
      if(!song.notes.length)throw Error('文件中没有可演奏的音符')
      library.value=[...library.value.filter(x=>x.id!==song.id),song]
      selectSong(song);changeMode('play');return true
    } catch(e){notify(e.message);return false}
  }
  function startRecording(){if(mode.value!=='free')changeMode('free');
    recording.status='recording';recording.events=[];recording.noteCount=0;recording.durationMs=0;recording.backupSaved=false;recording.startedAt=performance.now()}
  function stopRecording(){if(recording.status!=='recording')return
    recording.durationMs=performance.now()-recording.startedAt
    for(const k of Object.keys(pressed))recording.events.push({type:'off',note:Number(k),ms:recording.durationMs})
    recording.status=recording.noteCount?'draft':'idle';recording.backupSaved=false}
  function stopRecordingPreview(){clearTimeout(recordingPreviewTimer);audio.allOff('recording-preview');recordingPreviewing.value=false}
  async function previewRecording(){
    if(recording.status!=='draft'||!await unlock())return
    stopRecordingPreview()
    const song=parseMIDI(await encodeRecording(recording.events).arrayBuffer(),'录制试听')
    const now=audio.context.currentTime
    for(const note of song.notes)audio.scheduleMidiNote(note.note,Math.round(note.velocity*velocity.value/127),`recording-preview-${note.id}`,now+note.startMs/1000,now+note.endMs/1000)
    recordingPreviewing.value=true
    recordingPreviewTimer=setTimeout(stopRecordingPreview,song.durationMs+150)
  }
  function discardRecording(){recording.status='idle';recording.events=[];recording.noteCount=0;recording.durationMs=0;recording.backupSaved=false}
  function subscribe(fn){listeners.add(fn);return()=>listeners.delete(fn)}
  function setHidden(hidden){
    if(hidden&&!wasHidden){releaseAllUsers();if(state.status==='playing')pause()}
    wasHidden=hidden
  }
  onBeforeUnmount(()=>{stopRecordingPreview();stop();releaseAllUsers();audio.close();closeMidiWorker()})
  return {audio,mode,displayMode,library,selected,activeSong,duration,state,range,handMode,speed,loop,
    volume,velocity,pressed,pressedHands,playbackHands,recording,statusText,
    unlock,userNoteOn,userNoteOff,releaseSource,releaseAllUsers,toggleSustain,setVolume,setVelocity,importSF2,
    play,togglePlayback,pause,stop,seek,setRange,changeMode,setSpeed,setHand,selectSong,importMIDI,
    startRecording,stopRecording,previewRecording,stopRecordingPreview,recordingPreviewing,discardRecording,subscribe,visualClock,getClock,setHidden,
    remoteNoteOn,remoteNoteOff,remotePedal,syncRemoteNotes,clearRemoteNotes}
}
