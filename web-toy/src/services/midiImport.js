import {parseMIDI} from './midi'
let worker=null,nextId=0
const pending=new Map()
export async function parseMIDIFile(file){
  if(!file||file.size>10*1024*1024)throw Error('MIDI 文件请小于 10 MB')
  const buffer=await file.arrayBuffer()
  const name=file.name.replace(/\.midi?$/i,'')
  if(typeof Worker==='undefined')return parseMIDI(buffer,name)
  if(!worker){
    try { worker=new Worker(new URL('./midiWorker.js',import.meta.url),{type:'module'}) }
    catch { return parseMIDI(buffer,name) }
    worker.onmessage=e=>{
      const item=pending.get(e.data.id)
      if(!item)return
      pending.delete(e.data.id)
      if(e.data.error)item.reject(new Error(e.data.error))
      else item.resolve(e.data.song)
    }
    worker.onerror=()=>{
      for(const item of pending.values())item.reject(new Error('MIDI 解析线程异常'))
      pending.clear();worker.terminate();worker=null
    }
  }
  const id=++nextId
  return new Promise((resolve,reject)=>{
    pending.set(id,{resolve,reject})
    worker.postMessage({id,buffer,name},[buffer])
  })
}
export function closeMidiWorker(){worker?.terminate();worker=null;pending.clear()}
