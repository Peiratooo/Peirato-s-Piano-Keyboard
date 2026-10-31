// Browser replacement for the desktop Go/MeltySynth output. Velocity is never
// rewritten to increase loudness: output gain is a separate final stage.
const clamp = (n,a,b)=>Math.max(a,Math.min(b,n))

class Sf2Sampler {
  constructor(buffer, context) {
    const view = new DataView(buffer)
    const readText = (p,n) => Array.from(new Uint8Array(buffer,p,n)).map(c=>String.fromCharCode(c)).join('').replace(/\0.*$/,'')
    if (readText(0,4) !== 'RIFF' || readText(8,4) !== 'sfbk') throw new Error('不是有效的 SF2 文件')
    const chunks = {}
    const scan = (begin,end) => {
      for(let p=begin;p+8<=end;) {
        const name=readText(p,4), length=view.getUint32(p+4,true), data=p+8
        if(data+length>end) throw new Error('SF2 文件不完整')
        if(name==='LIST') scan(data+4,data+length)
        else if(['smpl','phdr','pbag','pgen','inst','ibag','igen','shdr'].includes(name)) chunks[name]={start:data,length}
        p=data+length+(length&1)
      }
    }
    scan(12,buffer.byteLength)
    if(!['smpl','phdr','pbag','pgen','inst','ibag','igen','shdr'].every(k=>chunks[k])) throw new Error('SF2 缺少采样或音色信息')
    this.samples = new Int16Array(buffer,chunks.smpl.start,Math.floor(chunks.smpl.length/2))
    this.view=view
    this.ctx=context
    this.cache=new Map()
    this.phdr=this.records(chunks.phdr,38,(p)=>({name:readText(p,20),preset:view.getUint16(p+20,true),bank:view.getUint16(p+22,true),bag:view.getUint16(p+24,true)}))
    this.pbag=this.records(chunks.pbag,4,p=>({gen:view.getUint16(p,true)}))
    this.pgen=this.records(chunks.pgen,4,p=>({op:view.getUint16(p,true),value:view.getUint16(p+2,true),amount:view.getInt16(p+2,true)}))
    this.inst=this.records(chunks.inst,22,p=>({name:readText(p,20),bag:view.getUint16(p+20,true)}))
    this.ibag=this.records(chunks.ibag,4,p=>({gen:view.getUint16(p,true)}))
    this.igen=this.records(chunks.igen,4,p=>({op:view.getUint16(p,true),value:view.getUint16(p+2,true),amount:view.getInt16(p+2,true)}))
    this.shdr=this.records(chunks.shdr,46,p=>({name:readText(p,20),start:view.getUint32(p+20,true),end:view.getUint32(p+24,true),loopStart:view.getUint32(p+28,true),loopEnd:view.getUint32(p+32,true),rate:view.getUint32(p+36,true),root:view.getUint8(p+40),correction:view.getInt8(p+41)}))
    this.presets=new Map()
    for(let i=0;i<this.phdr.length-1;i++) this.presets.set(`${this.phdr[i].bank}:${this.phdr[i].preset}`, i)
  }
  records(chunk,size,map) {const arr=[];for(let p=chunk.start;p+size<=chunk.start+chunk.length;p+=size)arr.push(map(p));return arr}
  zones(bags,gens,start,end) {
    const zones=[]
    for(let b=start;b<end;b++) {
      const z={}
      for(let i=bags[b].gen;i<(bags[b+1]?.gen??gens.length);i++) {
        const g=gens[i];z[g.op]=g
      }
      zones.push(z)
    }
    return zones
  }
  range(z,type,n) {const g=z[type];return !g || (n>= (g.value&255) && n<= (g.value>>8))}
  zoneFor(note,velocity,program=0) {
    const index=this.presets.get(`0:${program}`)??this.presets.get('0:0')??0
    const preset=this.phdr[index], next=this.phdr[index+1]
    const presetZones=this.zones(this.pbag,this.pgen,preset.bag,next.bag)
    let presetGlobal={}
    for(const p of presetZones) {
      if(!p[41]) {presetGlobal={...presetGlobal,...p}; continue}
      const pp={...presetGlobal,...p}
      if(!this.range(pp,43,note)||!this.range(pp,44,velocity)) continue
      const ins=this.inst[p[41].value],end=this.inst[p[41].value+1]
      if(!ins||!end)continue
      const zones=this.zones(this.ibag,this.igen,ins.bag,end.bag)
      let global={}
      for(const z of zones) {
        if(!z[53]) {global={...global,...z}; continue}
        const region={...pp,...global,...z}
        if(this.range(region,43,note)&&this.range(region,44,velocity)) return region
      }
    }
    return null
  }
  bufferFor(index) {
    if(this.cache.has(index)) return this.cache.get(index)
    const sample=this.shdr[index]
    if(!sample||sample.end<=sample.start||sample.end>this.samples.length) return null
    const length=sample.end-sample.start
    const out=this.ctx.createBuffer(1,length,sample.rate||44100)
    const data=out.getChannelData(0)
    for(let i=0;i<length;i++)data[i]=this.samples[sample.start+i]/32768
    this.cache.set(index,out)
    // bounded cache: prevents accumulating hundreds of decoded samples on mobile
    if(this.cache.size>48)this.cache.delete(this.cache.keys().next().value)
    return out
  }
  start(note,velocity,destination,now,program=0,channelGain=1) {
    const zone=this.zoneFor(note,velocity,program)
    if(!zone) return null
    const sh=this.shdr[zone[53].value], buffer=this.bufferFor(zone[53].value)
    if(!buffer||!sh) return null
    const node=this.ctx.createBufferSource(), gain=this.ctx.createGain()
    node.buffer=buffer
    const root=zone[58]?.value??sh.root
    const cents=((note-root)*100+(zone[51]?.amount||0)*100+(zone[52]?.amount||0)-sh.correction)
    node.playbackRate.value=Math.pow(2,cents/1200)
    if((zone[54]?.value||0)&1) {
      node.loop=true
      node.loopStart=clamp((sh.loopStart-sh.start)/sh.rate,0,buffer.duration)
      node.loopEnd=clamp((sh.loopEnd-sh.start)/sh.rate,node.loopStart+0.001,buffer.duration)
    }
    const attenuation=Math.pow(10,-(zone[48]?.value||0)/200)
    const attack=Math.max(.003,Math.pow(2,(zone[34]?.amount||-11000)/1200))
    gain.gain.setValueAtTime(0,now)
    gain.gain.linearRampToValueAtTime(velocity/127*.72*attenuation*channelGain,now+Math.min(attack,0.4))
    node.connect(gain).connect(destination)
    node.start(now)
    return {source:node,gain,release:(at=null,immediate=false)=>{
      const t=Math.max(this.ctx.currentTime,at??this.ctx.currentTime)
      gain.gain.cancelScheduledValues(t)
      gain.gain.setTargetAtTime(0,t,immediate?.003:.08)
      try {node.stop(t+(immediate?.035:.55))} catch {}
    }}
  }
}

export class PianoAudio {
  constructor() {
    this.context=null;this.master=null;this.limiter=null;this.sampler=null
    this.voices=new Map();this.sustain=false;this.heldReleased=new Set();this.program=0;this.volume=.78
  }
  async unlock() {
    if(!this.context) {
      const Ctx=window.AudioContext||window.webkitAudioContext
      if(!Ctx)throw new Error('当前浏览器不支持 Web Audio')
      this.context=new Ctx({latencyHint:'interactive'})
      this.master=this.context.createGain();this.master.gain.value=this.volume
      this.limiter=this.context.createDynamicsCompressor()
      this.limiter.threshold.value=-2
      this.limiter.knee.value=0
      this.limiter.ratio.value=16
      this.limiter.attack.value=.003
      this.limiter.release.value=.12
      this.master.connect(this.limiter).connect(this.context.destination)
    }
    if(this.context.state!=='running')await this.context.resume()
    return this.context
  }
  setVolume(value) {this.volume=clamp(value,0,1);if(this.master)this.master.gain.value=this.volume}
  async loadSF2(file) {
    await this.unlock()
    const data=await file.arrayBuffer()
    const sampler=new Sf2Sampler(data,this.context)
    this.allOff();this.sampler=sampler
    return `${sampler.presets.size} 个预置音色 · ${Math.round(data.byteLength/1048576)} MB`
  }
  noteOn(note,velocity=80,source='user',at=null,program=this.program,channelGain=1) {
    if(!this.context)return
    if(note<0||note>127||velocity<=0)return
    const time=Math.max(this.context.currentTime,at??this.context.currentTime)
    this.noteOff(note,source,true)
    const key=`${source}:${note}`
    const sampled=this.sampler?.start(note,velocity,this.master,time,program,channelGain)
    if(sampled) {this.voices.set(key,sampled);return}
    // Compact browser piano fallback until the user imports their SF2.
    const freq=440*Math.pow(2,(note-69)/12)
    const gain=this.context.createGain()
    gain.gain.setValueAtTime(0,time)
    gain.gain.linearRampToValueAtTime((velocity/127)**1.55*.36*channelGain,time+.006)
    gain.gain.exponentialRampToValueAtTime(.0008,time+3.8)
    const oscillators=[]
    for(const [part,level] of [[1,1],[2,.3],[3,.12],[4,.06]]) {
      const osc=this.context.createOscillator(),amp=this.context.createGain()
      osc.type='sine';osc.frequency.value=freq*part; amp.gain.value=level
      osc.connect(amp).connect(gain);osc.start(time);osc.stop(time+4.4)
      oscillators.push(osc)
    }
    gain.connect(this.master)
    this.voices.set(key,{source:oscillators[0],gain,release:(at=null,immediate=false)=>{
      const t=Math.max(this.context.currentTime,at??this.context.currentTime)
      gain.gain.cancelScheduledValues(t)
      gain.gain.setTargetAtTime(0,t,immediate?.003:.08)
      oscillators.forEach(o=>{try{o.stop(t+(immediate?.035:.45))}catch{}})
    }})
  }
  noteOff(note,source='user',force=false,at=null) {
    const key=`${source}:${note}`
    if(this.sustain&&!force&&source==='user'){this.heldReleased.add(key);return}
    const voice=this.voices.get(key)
    if(voice){voice.release(at,force);this.voices.delete(key)}
    this.heldReleased.delete(key)
  }
  // Schedule a complete voice directly on the Web Audio clock instead of
  // depending on the JS event-loop interval for each attack and release.
  // Scheduled voices remain cancellable on pause, seek, looping and unload.
  scheduleMidiNote(note,velocity,source,startAt,endAt,program=0,channelGain=1) {
    this.noteOn(note,velocity,source,startAt,program,channelGain)
    const key=`${source}:${note}`,voice=this.voices.get(key)
    if (!voice) return
    voice.release(endAt,false)
    voice.scheduledUntil=endAt+.6
  }
  pruneScheduled(now=this.context?.currentTime||0) {
    for(const [key,voice] of this.voices) {
      if(key.startsWith('midi-')&&voice.scheduledUntil&&voice.scheduledUntil<now){
        this.voices.delete(key)
      }
    }
  }
  setSustain(on) {
    this.sustain=!!on
    if(!on)for(const key of [...this.heldReleased]) {
      const at=key.lastIndexOf(':');this.noteOff(Number(key.slice(at+1)),key.slice(0,at),true)
    }
  }
  allOff(source=null) {
    for(const key of [...this.voices.keys()]) {
      if(source&&!key.startsWith(source+':')&&!(source==='midi'&&key.startsWith('midi-')))continue
      const p=key.lastIndexOf(':');this.noteOff(Number(key.slice(p+1)),key.slice(0,p),true)
    }
  }
  close() {this.allOff();void this.context?.close();this.context=null}
}
