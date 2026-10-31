import {parseMIDI} from './midi'
self.onmessage=event=>{
 const {id,buffer,name}=event.data
 try{self.postMessage({id,song:parseMIDI(buffer,name)})}
 catch(error){self.postMessage({id,error:error?.message||String(error)})}
}
