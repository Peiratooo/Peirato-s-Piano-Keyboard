// MIDI files and recording drafts stay on-device. Toy CloudStorage is only
// suitable for small settings, never for MIDI or large recording payloads.
let databasePromise
function db() {
  if (!('indexedDB' in window)) return Promise.reject(new Error('当前浏览器不支持本地存储'))
  if (!databasePromise) {
    databasePromise = new Promise((resolve, reject) => {
      const req = indexedDB.open('pp-piano-toy', 1)
      req.onupgradeneeded = () => {
        if (!req.result.objectStoreNames.contains('songs')) req.result.createObjectStore('songs', {keyPath: 'id'})
        if (!req.result.objectStoreNames.contains('draft')) req.result.createObjectStore('draft')
      }
      req.onsuccess = () => resolve(req.result)
      req.onerror = () => reject(req.error || new Error('无法打开本地数据库'))
      req.onblocked = () => reject(new Error('本地数据库被其他页面占用'))
    }).catch(error => { databasePromise = null; throw error })
  }
  return databasePromise
}
async function transaction(store, mode, run) {
  const database = await db()
  return new Promise((resolve, reject) => {
    let result
    let tx
    try {
      tx = database.transaction(store, mode)
      const request = run(tx.objectStore(store))
      request.onsuccess = () => { result = request.result }
      request.onerror = () => { /* tx.onabort reports the durable failure */ }
      // Request success is not enough: wait until the whole transaction commits.
      tx.oncomplete = () => resolve(result)
      tx.onerror = () => { /* onabort is authoritative */ }
      tx.onabort = () => reject(tx.error || new Error('本地存储提交失败，数据未保存'))
    } catch (error) {
      reject(error)
    }
  })
}
const clone = value => JSON.parse(JSON.stringify(value))
export const saveSong = song => transaction('songs', 'readwrite', s => s.put(clone(song)))
export const getSongs = () => transaction('songs', 'readonly', s => s.getAll())
export const removeSong = id => transaction('songs', 'readwrite', s => s.delete(id))
export const saveDraft = draft => transaction('draft', 'readwrite', s => s.put(clone(draft), 'last'))
export const getDraft = () => transaction('draft', 'readonly', s => s.get('last'))
export const clearDraft = () => transaction('draft', 'readwrite', s => s.delete('last'))

// Community binaries have their own store; existing local songs/drafts are untouched.
let communityDatabase
async function communityCache(mode,key,value){
 if(!communityDatabase)communityDatabase=new Promise((resolve,reject)=>{const request=indexedDB.open('pp-piano-community-files',1);request.onupgradeneeded=()=>request.result.createObjectStore('files');request.onsuccess=()=>resolve(request.result);request.onerror=()=>reject(request.error)})
 const database=await communityDatabase
 return new Promise((resolve,reject)=>{const tx=database.transaction('files',mode);const store=tx.objectStore('files');const request=mode==='readwrite'?store.put(value,key):store.get(key);let result;request.onsuccess=()=>result=request.result;tx.oncomplete=()=>resolve(result);tx.onabort=()=>reject(tx.error)})
}
export const getCommunityFile=key=>communityCache('readonly',key)
export const saveCommunityFile=(key,value)=>communityCache('readwrite',key,value)
