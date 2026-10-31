// Pure browser SMF format 0/1 parser. No Wails bindings or network required.
const readText=(u,p,n)=>String.fromCharCode(...u.subarray(p,p+n))
const tempoDefault=500000
const clamp=(v,a,b)=>Math.max(a,Math.min(b,v))
const varInt=(bytes,point)=>{let v=0,b;do{if(point.p>=bytes.length)throw Error('MIDI 事件已截断');b=bytes[point.p++];v=(v<<7)|(b&127)}while(b&128);return v}
const u32=(v,p)=>v.getUint32(p,false)

export function parseMIDI(buffer,name='未命名 MIDI') {
  const u=new Uint8Array(buffer),v=new DataView(buffer)
  if(u.length<14||readText(u,0,4)!=='MThd')throw Error('请选择有效的 .mid / .midi 文件')
  const headerLength=u32(v,4),format=v.getUint16(8,false),tracks=v.getUint16(10,false),division=v.getUint16(12,false)
  if(![0,1].includes(format)||division&0x8000)throw Error('暂不支持此 MIDI 格式或 SMPTE 时间格式')
  if(headerLength<6||8+headerLength>u.length||tracks<1||tracks>128||division<1)throw Error('MIDI 文件参数无效')
  const events=[],tempo=[{tick:0,value:tempoDefault}],signatures=[{tick:0,nn:4,dd:4}]
  let pos=8+headerLength,maxTick=0
  for(let t=0;t<tracks;t++) {
    if(pos+8>u.length||readText(u,pos,4)!=='MTrk')throw Error('找不到 MIDI 轨道')
    const end=pos+8+u32(v,pos+4)
    if(end>u.length)throw Error('MIDI 轨道数据不完整')
    const point={p:pos+8};let tick=0,running=0,nameOfTrack=`轨道 ${t+1}`
    while(point.p<end) {
      tick+=varInt(u,point);maxTick=Math.max(tick,maxTick)
      let status=u[point.p]
      if(status&0x80) {point.p++;if(status<0xf0)running=status}
      else {if(!running)throw Error('MIDI running status 无效');status=running}
      if(status===0xff) {
        const type=u[point.p++],length=varInt(u,point),start=point.p
        if(start+length>end)throw Error('MIDI meta 事件不完整')
        if(type===0x51&&length===3)tempo.push({tick,value:(u[start]<<16)|(u[start+1]<<8)|u[start+2]})
        if(type===0x58&&length>=2)signatures.push({tick,nn:u[start],dd:2**u[start+1]})
        if(type===0x03&&length>0)nameOfTrack=readText(u,start,Math.min(length,80))
        point.p=start+length
      } else if(status===0xf0||status===0xf7) {
        point.p+=varInt(u,point)
        if(point.p>end)throw Error('MIDI SysEx 数据不完整')
      } else {
        const kind=status&0xf0,channel=status&15, a=u[point.p++]
        const b=(kind===0xc0||kind===0xd0)?0:u[point.p++]
        if(point.p>end)throw Error('MIDI 消息不完整')
        if(kind===0x90&&b)events.push({tick,t,channel,type:'on',note:a,velocity:b})
        else if(kind===0x80||(kind===0x90&&!b))events.push({tick,t,channel,type:'off',note:a,velocity:0})
        else if(kind===0xb0)events.push({tick,t,channel,type:'cc',controller:a,value:b})
        else if(kind===0xc0)events.push({tick,t,channel,type:'program',program:a})
      }
    }
    pos=end
  }
  const uniqueTempos=[]
  tempo.sort((a,b)=>a.tick-b.tick)
  for(const item of tempo) {
    const last=uniqueTempos.at(-1)
    if(last&&last.tick===item.tick)last.value=item.value
    else uniqueTempos.push({...item})
  }
  let elapsed=0
  for(let i=0;i<uniqueTempos.length;i++){
    if(i)elapsed+=(uniqueTempos[i].tick-uniqueTempos[i-1].tick)*uniqueTempos[i-1].value/(division*1000)
    uniqueTempos[i].ms=elapsed
  }
  const tickMs=(tick)=>{
    let l=0,r=uniqueTempos.length
    while(l+1<r){const m=(l+r)>>1;if(uniqueTempos[m].tick<=tick)l=m;else r=m}
    const x=uniqueTempos[l]
    return x.ms+(tick-x.tick)*x.value/(division*1000)
  }
  // OFF before ON at identical tick handles repeated attacks correctly.
  const order={off:0,cc:1,program:2,on:3}
  events.sort((a,b)=>a.tick-b.tick||(order[a.type]-order[b.type]))
  const pending=new Map(),sustained=new Map(),notes=[],controllers=[]
  const pedal=new Map(),channelProgram=new Map(),channelVolume=new Map(),expression=new Map()
  // Keep key release time (visual) and sustain-aware sound end time separate.
  const finishSustained=(channel,ms)=>{
    for(const [key,queue] of sustained)if(key.startsWith(`${channel}:`)){
      for(const index of queue)notes[index].soundEndMs=Math.max(ms,notes[index].endMs)
      sustained.delete(key)
    }
  }
  for(const e of events){
    const ms=tickMs(e.tick),key=`${e.t}:${e.channel}:${e.note}`,pedalKey=`${e.channel}:${e.note}`
    if(e.type==='on'){
      if(notes.length>=250000)throw Error('MIDI 音符过多，无法在浏览器安全加载')
      const index=notes.length
      notes.push({id:index,note:e.note,velocity:e.velocity,startMs:ms,endMs:ms+400,
        soundEndMs:ms+400,trackIndex:e.t,channel:e.channel,
        program:channelProgram.get(e.channel)||0,
        gain:(channelVolume.get(e.channel)??100)/127*(expression.get(e.channel)??127)/127,
        hand:'right'})
      const queue=pending.get(key)||[];queue.push(index);pending.set(key,queue)
    } else if(e.type==='off') {
      const queue=pending.get(key)||[]
      if(queue.length){
        const index=queue.shift(),n=notes[index]
        n.endMs=Math.max(ms,n.startMs+1)
        n.soundEndMs=n.endMs
        if(pedal.get(e.channel)){
          const held=sustained.get(pedalKey)||[];held.push(index);sustained.set(pedalKey,held)
        }
      }
      if(!queue.length)pending.delete(key)
    } else if(e.type==='cc'){
      controllers.push({...e,ms})
      if(e.controller===64){
        const next=e.value>=64
        if(!next&&pedal.get(e.channel))finishSustained(e.channel,ms)
        pedal.set(e.channel,next)
      }else if(e.controller===7)channelVolume.set(e.channel,e.value)
      else if(e.controller===11)expression.set(e.channel,e.value)
      else if(e.controller===120||e.controller===123){
        finishSustained(e.channel,ms)
        for(const [key,queue] of pending)if(key.split(':')[1]===String(e.channel)){
          for(const index of queue){notes[index].endMs=ms;notes[index].soundEndMs=ms}
          pending.delete(key)
        }
      }
    } else if(e.type==='program'){
      controllers.push({...e,ms})
      channelProgram.set(e.channel,e.program)
    }
  }
  // No spread into Math.max: large tracks exceed the JS argument limit.
  let durationMs=tickMs(maxTick)
  for(const n of notes)durationMs=Math.max(durationMs,n.endMs)
  for(const queue of pending.values())for(const index of queue){
    notes[index].endMs=durationMs;notes[index].soundEndMs=durationMs
  }
  for(const queue of sustained.values())for(const index of queue)notes[index].soundEndMs=durationMs
  const activeTracks=[...new Set(notes.map(n=>n.trackIndex))]
  const medians=new Map(activeTracks.map(t=>{
    const arr=notes.filter(n=>n.trackIndex===t).map(n=>n.note).sort((a,b)=>a-b)
    return [t,arr[Math.floor(arr.length/2)]||60]
  }))
  const sorted=[...activeTracks].sort((a,b)=>medians.get(a)-medians.get(b))
  for(const note of notes)note.hand=activeTracks.length>=2
    ? (note.trackIndex===sorted[0]?'left':'right')
    : (note.note<60?'left':'right')
  notes.sort((a,b)=>a.startMs-b.startMs||a.note-b.note)
  // actual measure boundaries from MIDI time signature and PPQ.
  signatures.sort((a,b)=>a.tick-b.tick)
  let currentTick=0,sigIndex=0,measureIndex=1
  const measures=[]
  for(let guard=0;guard<2500&&currentTick<=maxTick;guard++) {
    while(sigIndex+1<signatures.length&&signatures[sigIndex+1].tick<=currentTick)sigIndex++
    const sig=signatures[sigIndex]
    measures.push({index:measureIndex++,ms:tickMs(currentTick)})
    const next=sigIndex+1<signatures.length?signatures[sigIndex+1].tick:Infinity
    const step=division*4*(sig.nn||4)/(sig.dd||4)
    currentTick=Math.min(currentTick+step,next)
    if(step<=0)break
  }
  return {id:`import-${Date.now()}`,name,format,tracks,notes,controllers,measures,durationMs,
    noteCount:notes.length,tempoBpm:Math.round(60000000/uniqueTempos[0].value)}
}

// One-track SMF: retain the real input velocities, timings and sustain CCs.
const be16=n=>[(n>>8)&255,n&255]
const be32=n=>[(n>>>24)&255,(n>>>16)&255,(n>>>8)&255,n&255]
const variable=n=>{let b=n&127;while(n>>=7)b=(b<<8)|((n&127)|128);const o=[];while(true){o.push(b&255);if(b&128)b>>=8;else break}return o}
export function encodeRecording(events) {
  const ppq=480,perMs=.96
  const sorted=[...events].sort((a,b)=>a.ms-b.ms||((a.type==='off'?0:1)-(b.type==='off'?0:1)))
  let last=0
  const track=[0,0xff,0x51,3,7,161,32] // 120 BPM
  for(const e of sorted){
    const tick=Math.max(last,Math.round(e.ms*perMs))
    track.push(...variable(tick-last));last=tick
    if(e.type==='on')track.push(0x90,e.note&127,e.velocity&127)
    else if(e.type==='off')track.push(0x80,e.note&127,0)
    else if(e.type==='pedal')track.push(0xb0,64,e.value&127)
  }
  track.push(0,0xff,0x2f,0)
  const out=new Uint8Array([
    ...new TextEncoder().encode('MThd'),...be32(6),...be16(0),...be16(1),...be16(ppq),
    ...new TextEncoder().encode('MTrk'),...be32(track.length),...track,
  ])
  return new Blob([out],{type:'audio/midi'})
}

export function exampleSong() {
  const sequence=[60,60,67,67,69,69,67,65,65,64,64,62,62,60,67,67,65,65,64,64,62,67,67,65,65,64,64,62,60,60,67,67,69,69,67,65,65,64,64,62,62,60]
  const notes=sequence.map((note,i)=>({id:i,note,velocity:77,startMs:i*450,endMs:i*450+385,channel:0,trackIndex:0,hand:note<65?'left':'right'}))
  const durationMs=notes.at(-1).endMs+450
  const measures=Array.from({length:Math.ceil(durationMs/1800)},(_,i)=>({index:i+1,ms:i*1800}))
  return {id:'builtin-demo',name:'小星星 · 演奏示例',notes,controllers:[],measures,durationMs,noteCount:notes.length,tracks:1,tempoBpm:120}
}
