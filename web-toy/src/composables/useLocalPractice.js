import {watch,onBeforeUnmount} from 'vue'
const key='pp-piano-practice-history'
export function useLocalPractice(piano,notify){
 let items=[]
 try{items=JSON.parse(localStorage.getItem(key)||'[]')}catch{notify('本地练习记录无法读取')}
 let previous={status:'idle'},last=performance.now()
 function record(){
  const now=performance.now(),song=piano.selected,state={id:song?.id,status:piano.state.status,mode:piano.mode}
  const old=items.find(item=>item.localId===previous.id)
  if(old&&previous.status==='playing'){old.seconds+=(now-last)/1000;old.lastPracticed=new Date().toISOString()}
  if(state.id&&state.mode!=='free'&&state.status==='playing'&&(state.id!==previous.id||state.mode!==previous.mode||!['playing','paused'].includes(previous.status))){
   let item=items.find(item=>item.localId===state.id)
   if(!item){item={localId:state.id,midiId:song.serverMidiId||'',accountId:song.ownerAccountId||0,title:song.name,sessions:0,seconds:0};items.unshift(item)}
   item.sessions++;item.lastMode=state.mode;item.lastPracticed=new Date().toISOString()
  }
  const current=items.find(item=>item.localId===state.id)
  if(current&&state.status==='playing'){current.lastMode=state.mode;current.lastPracticed=new Date().toISOString()}
  previous=state;last=now
  try{localStorage.setItem(key,JSON.stringify(items))}catch{notify('本地练习记录未能保存，请检查可用空间')}
 }
 const stop=watch(()=>[piano.selected?.id,piano.mode,piano.state.status],record,{flush:'sync'})
 const timer=setInterval(record,15000)
 window.addEventListener('pagehide',record)
 onBeforeUnmount(()=>{record();stop();clearInterval(timer);window.removeEventListener('pagehide',record)})
 return ()=>{record();return items.slice().sort((a,b)=>new Date(b.lastPracticed)-new Date(a.lastPracticed))}
}
