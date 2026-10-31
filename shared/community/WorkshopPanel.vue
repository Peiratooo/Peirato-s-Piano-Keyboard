<template>
 <section class="community workshop" aria-label="MIDI 工坊">
  <div v-if="error" class="error" role="alert">{{error}}</div><p v-if="message" class="message" role="status">{{message}}</p>
  <template v-if="editing">
   <header class="catalog-heading"><div><h1>{{editing.id?'编辑作品':'上传 MIDI'}}</h1><p>让每一段旋律，都有机会被听见</p></div><button @click="editing=null"><PhArrowLeft/>返回工坊</button></header>
   <form class="card upload-form" @submit.prevent="save">
    <div v-if="!editing.id" class="file-choice"><PhFileAudio :size="32"/><div><strong>{{file?.name||'选择你的 MIDI 文件'}}</strong><p class="muted">支持 .mid / .midi，单个文件最大 8MB</p></div><button type="button" @click="chooseFile">选择文件</button></div>
    <label>曲目标题<input v-model.trim="editing.title" maxlength="120" required placeholder="为作品起一个名字"/></label>
    <div class="grid"><label>作曲者<input v-model.trim="editing.composer" maxlength="120" placeholder="作曲者姓名"/></label><label>难度<select v-model.number="editing.difficulty"><option v-for="n in 5" :key="n" :value="n">{{difficultyNames[n]}} · {{n}} / 5</option></select></label></div>
    <label>作品介绍<textarea v-model.trim="editing.description" maxlength="1500" placeholder="介绍这首曲子，或分享你的演奏建议"/></label><label>标签<input v-model.trim="editing.tags" placeholder="用逗号分隔，最多 8 个标签" maxlength="300"/></label>
    <label class="check"><input type="checkbox" v-model="editing.public" :disabled="editing.reviewStatus==='blocked'"/>公开分享，审核通过后展示在工坊</label>
    <label v-if="editing.public" class="check"><input v-model="editing.rightsConfirmed" type="checkbox" required/>我确认拥有公开分享该 MIDI 的必要权利</label>
    <p v-if="editing.reviewStatus==='blocked'" class="danger">作品已下架，只有管理员可以解除下架。</p>
    <div class="row"><button class="primary" :disabled="busy||!client.user||(!editing.id&&!file)">{{busy?'正在保存…':'保存作品'}}</button><span class="muted">私密作品仅自己可见</span></div>
   </form>
  </template>
  <template v-else>
   <header class="catalog-heading"><div><h1>{{scope==='mine'?'我的上传':scope==='favorites'?'我的收藏':'MIDI 工坊'}}</h1><p>{{scope==='mine'?'整理自己的旋律，分享演奏的乐趣':scope==='favorites'?'喜欢的旋律，随时回来继续弹':'发现下一首想弹的曲子'}}</p></div><button class="primary upload-cta" @click="edit()"><PhUploadSimple :size="20"/>上传 MIDI</button></header>
   <form class="catalog-search" role="search" @submit.prevent="page=1;load()"><PhMagnifyingGlass :size="22"/><input v-model.trim="query" aria-label="搜索工坊" placeholder="搜索曲名、作曲者、上传者或标签" maxlength="60"/><button v-if="query" type="button" aria-label="清除搜索" @click="query='';page=1;load()"><PhX :size="18"/></button><button type="submit" class="search-submit">搜索</button></form>
   <div class="catalog-filters"><div class="genre-filters" aria-label="标签筛选"><button v-for="genre in genres" :key="genre" :class="{selected:tag===(genre==='全部'?'':genre)}" :aria-pressed="tag===(genre==='全部'?'':genre)" @click="filterTag(genre==='全部'?'':genre)">{{genre}}</button><label class="custom-tag"><PhTag :size="17"/><input v-model.trim="tagInput" aria-label="自定义标签筛选" placeholder="更多标签" maxlength="40" @change="filterTag(tagInput)" @keydown.enter.prevent="filterTag(tagInput)"/></label><button v-if="tag&&!genres.includes(tag)" class="selected" @click="filterTag('')">{{tag}}<PhX :size="13"/></button></div><div class="catalog-selects"><select v-model="difficulty" aria-label="筛选难度" @change="page=1;load()"><option value="">全部难度</option><option v-for="n in 5" :key="n" :value="n">{{difficultyNames[n]}} · {{n}} / 5</option></select><select v-model="sort" aria-label="排序方式" @change="page=1;load()"><option v-for="item in sorts" :key="item.id" :value="item.id">{{item.name}}</option></select></div></div>
   <div class="catalog-grid" :aria-busy="loading">
    <template v-if="loading&&!songs.length"><div v-for="n in 8" :key="n" class="catalog-skeleton" aria-hidden="true"><span></span><span></span><span></span></div></template>
    <article v-for="song in songs" :key="song.id" class="midi-card" :class="{chosen:playingId===song.id||selectedId===song.id}">
     <div class="midi-card-heading"><div><h2><button class="midi-card-title" @click="open(song)">{{song.title}}</button></h2><p class="midi-composer">{{song.composer||'未标注作曲者'}}</p></div><button class="card-preview" :disabled="busy" :aria-label="'试听 '+song.title" @click="play(song,'play')"><PhPlay :size="20" weight="fill"/></button></div>
     <div class="midi-meta"><span>{{time(song.durationMs)}}</span><span>{{difficultyNames[song.difficulty]}}</span><span>{{Math.round(song.tempoBpm)}} BPM</span></div>
     <div class="midi-tags"><button v-for="label in tags(song).slice(0,2)" :key="label" @click="filterTag(label)">{{label}}</button><span v-if="scope==='mine'" class="review-badge" :class="song.reviewStatus">{{status(song)}}</span></div>
     <footer class="midi-card-footer"><div class="midi-uploader"><img v-if="song.ownerAvatarUrl" :src="client.API+song.ownerAvatarUrl" alt=""/><PhUserCircle v-else :size="23" weight="duotone"/><span>{{song.ownerName}}</span></div><div class="card-interactions"><button :class="{selected:song.myReaction===1}" :disabled="busy" :aria-label="'点赞 '+song.title" :aria-pressed="song.myReaction===1" @click="vote(song,1)"><PhThumbsUp :size="16"/>{{song.likeCount}}</button><button :class="{selected:song.myFavorite}" :disabled="busy" :aria-label="'收藏 '+song.title" :aria-pressed="song.myFavorite" @click="favorite(song)"><PhBookmarkSimple :size="16" :weight="song.myFavorite?'fill':'regular'"/>{{song.favoriteCount}}</button></div></footer>
    </article>
   </div>
   <div v-if="!loading&&!songs.length" class="catalog-empty"><PhMusicNotes :size="42" weight="duotone"/><h2>{{query||tag||difficulty?'没有找到这段旋律':'这里还没有曲目'}}</h2><p>{{query||tag||difficulty?'试试其他关键词，或调整筛选条件。':scope==='favorites'?'点击曲目上的收藏按钮，把喜欢的旋律留在这里。':scope==='mine'?'上传你的第一首 MIDI，随时回来练习。':'你可以上传作品，审核通过后会展示在这里。'}}</p><button v-if="query||tag||difficulty" @click="clearFilters">清除筛选</button><button v-else-if="scope!=='favorites'" @click="edit()">上传 MIDI</button></div>
   <div class="catalog-pager"><span>{{total}} 首曲目<span v-if="loading"> · 正在更新…</span></span><div><button aria-label="刷新工坊" :disabled="loading" @click="load()"><PhArrowClockwise :size="18"/></button><span>{{page}} / {{pages}}</span><button aria-label="上一页" :disabled="loading||page<=1" @click="page--;load()"><PhCaretLeft :size="18"/></button><button aria-label="下一页" :disabled="loading||page>=pages" @click="page++;load()"><PhCaretRight :size="18"/></button></div></div>
  </template>
  <NModal v-if="detail" :show="true" :auto-focus="true" :close-on-esc="true" @update:show="show=>{if(!show)detail=null}">
   <section class="community song-detail" :style="themeStyle" data-piano-control aria-label="曲目详情"><header class="detail-heading"><span class="status">{{status(detail)}}</span><button aria-label="关闭曲目详情" @click="detail=null"><PhX :size="22"/></button></header><h1>{{detail.title}}</h1><p class="detail-composer">{{detail.composer||'未标注作曲者'}}</p><div class="identity"><img v-if="detail.ownerAvatarUrl" class="avatar" :src="client.API+detail.ownerAvatarUrl" alt="上传者头像"/><PhUserCircle v-else :size="32"/><strong>{{detail.ownerName}}</strong></div>
    <div class="detail-facts"><span><small>时长</small>{{time(detail.durationMs)}}</span><span><small>难度</small>{{difficultyNames[detail.difficulty]}} · {{detail.difficulty}}/5</span><span><small>速度</small>{{Math.round(detail.tempoBpm)}} BPM</span><span><small>拍号 / 调号</small>{{detail.timeSignature}} · {{detail.keySignature||'未标注'}}</span></div>
    <p class="description">{{detail.description||'上传者还没有填写作品介绍。'}}</p><div class="row"><button v-for="label in tags(detail)" :key="label" class="tag" @click="detail=null;filterTag(label)">{{label}}</button></div><p class="muted">{{detail.trackCount}} 音轨 · {{detail.noteCount}} 音符 · {{detail.measures}} 小节</p><p class="muted">音轨：{{tracks(detail)}}</p>
    <div class="metrics"><span><PhEye/>{{detail.viewCount}} 浏览</span><span><PhPlay/>{{detail.playCount}} 播放</span><span><PhDownloadSimple/>{{detail.downloadCount}} 下载</span></div>
    <div class="row detail-interactions"><button :disabled="busy" :class="{selected:detail.myReaction===1}" aria-label="点赞" :aria-pressed="detail.myReaction===1" @click="vote(detail,1)"><PhThumbsUp/>{{detail.likeCount}}</button><button :disabled="busy" :class="{selected:detail.myReaction===-1}" aria-label="点踩" :aria-pressed="detail.myReaction===-1" @click="vote(detail,-1)"><PhThumbsDown/>{{detail.dislikeCount}}</button><button :disabled="busy" :class="{selected:detail.myFavorite}" aria-label="收藏" :aria-pressed="detail.myFavorite" @click="favorite(detail)"><PhBookmarkSimple/>{{detail.favoriteCount}} 收藏</button><button v-if="!client.user" class="detail-login" @click="$emit('account')">登录后参与互动</button></div>
    <p v-if="error" class="error" role="alert">{{error}}</p><p v-if="message" class="message" role="status">{{message}}</p>
    <div class="row detail-actions"><button class="primary" :disabled="busy" @click="play(detail,'play')"><PhPlay weight="fill"/>试听</button><button :disabled="busy" @click="play(detail,'follow')"><PhPianoKeys/>跟弹练习</button><button :disabled="busy" @click="download(detail)"><PhDownloadSimple/>下载</button><button v-if="client.user&&detail.public" :disabled="busy" @click="report">举报</button></div>
    <div v-if="client.user?.id===detail.ownerId" class="row"><button @click="edit(detail)">编辑作品</button><button class="danger" :disabled="busy" @click="remove">删除作品</button></div>
   </section>
  </NModal>
 </section>
</template>
<script setup>
import {ref,computed,onMounted,watch} from 'vue'
import {NModal} from 'naive-ui'
import {PhMusicNotes,PhUserCircle,PhThumbsUp,PhThumbsDown,PhBookmarkSimple,PhEye,PhPlay,PhDownloadSimple,PhUploadSimple,PhMagnifyingGlass,PhX,PhCaretLeft,PhCaretRight,PhArrowClockwise,PhArrowLeft,PhTag,PhFileAudio,PhPianoKeys} from '@phosphor-icons/vue'
import './community.css'
import './workshop.css'
const props=defineProps({client:{type:Object,required:true},themeStyle:{type:Object,required:true},playingId:{type:String,default:''}}),emit=defineEmits(['play','account'])
const scope=defineModel('scope',{type:String,default:'public'})
const genres=['全部','古典','流行','练习曲'],difficultyNames=['','入门','初级','中级','进阶','专业']
const sorts=[{id:'latest',name:'最新上传'},{id:'likes',name:'最多点赞'},{id:'favorites',name:'最多收藏'},{id:'views',name:'最多浏览'},{id:'plays',name:'最多播放'},{id:'downloads',name:'最多下载'}]
const query=ref(''),tag=ref(''),tagInput=ref(''),difficulty=ref(''),sort=ref('latest'),page=ref(1),total=ref(0),songs=ref([]),detail=ref(null),selectedId=ref(''),editing=ref(null),file=ref(null),busy=ref(false),loading=ref(false),error=ref(''),message=ref('')
let listVersion=0
const pages=computed(()=>Math.max(1,Math.ceil(total.value/20)))
const tags=song=>song.tags?song.tags.split(','):[]
const tracks=song=>JSON.parse(song.trackNames||'[]').join('、')||'未标注'
const time=ms=>`${Math.floor(ms/60000)}:${String(Math.floor(ms/1000)%60).padStart(2,'0')}`
const status=song=>song.reviewStatus==='blocked'?'已下架':!song.public?'私密':({pending:'待审核',approved:'公开',rejected:'审核未通过'})[song.reviewStatus]
async function run(action){if(busy.value)return;busy.value=true;error.value='';message.value='';try{return await action()}catch(e){error.value=e.message||String(e)}finally{busy.value=false}}
async function load(){const version=++listVersion;loading.value=true;error.value='';const params=new URLSearchParams({scope:scope.value,q:query.value,tag:tag.value,difficulty:String(difficulty.value),sort:sort.value,page:String(page.value),pageSize:'20'});try{const data=await props.client.request('/api/midis?'+params);if(version===listVersion){songs.value=data.items;total.value=data.total}}catch(e){if(version===listVersion)error.value=e.message||String(e)}finally{if(version===listVersion)loading.value=false}}
function filterTag(value){tag.value=value;tagInput.value=genres.includes(value)?'':value;page.value=1;void load()}
function clearFilters(){query.value='';tag.value='';tagInput.value='';difficulty.value='';page.value=1;void load()}
function replaceSong(song){if(detail.value?.id===song.id)detail.value=song;const index=songs.value.findIndex(x=>x.id===song.id);if(index>=0)songs.value[index]=song}
async function open(song){selectedId.value=song.id;await run(async()=>{const data=await props.client.request('/api/midis/'+song.id);await props.client.json(`/api/midis/${song.id}/events`,'POST',{type:'view'});data.viewCount++;detail.value=data;replaceSong(data)})}
function edit(song){if(!props.client.user){emit('account');return}editing.value=song?{...song,rightsConfirmed:false}:{title:'',composer:'',description:'',tags:'',difficulty:3,public:false,rightsConfirmed:false};detail.value=null;file.value=null}
async function chooseFile(){await run(async()=>{file.value=await props.client.chooseMIDI();if(file.value&&!editing.value.title)editing.value.title=file.value.name.replace(/\.midi?$/i,'')})}
async function save(){await run(async()=>{const {title,composer,description,tags,difficulty,public:pub,rightsConfirmed}=editing.value;const fields={title,composer,description,tags,difficulty,public:pub,rightsConfirmed};if(editing.value.id)await props.client.json('/api/midis/'+editing.value.id,'PATCH',fields);else await props.client.upload(file.value,fields);editing.value=null;page.value=1;if(scope.value==='mine')await load();else scope.value='mine';message.value=pub?'已提交，等待公开审核':'已保存到私密曲库'})}
async function interaction(song,reaction,favorite){if(!props.client.user){emit('account');return}await run(async()=>{const updated=await props.client.json(`/api/midis/${song.id}/interaction`,'PUT',{reaction,favorite});replaceSong(updated);if(scope.value==='favorites'&&!updated.myFavorite)await load()})}
const vote=(song,value)=>interaction(song,song.myReaction===value?0:value,song.myFavorite)
const favorite=song=>interaction(song,song.myReaction,!song.myFavorite)
async function play(song,mode){await run(async()=>{await props.client.loadSong(song,mode);selectedId.value=song.id;detail.value=null;emit('play',mode)})}
async function download(song){await run(async()=>{const saved=await props.client.download(song);if(saved!==false){replaceSong({...song,downloadCount:song.downloadCount+1});message.value='下载完成'}})}
async function remove(){if(!confirm('删除「'+detail.value.title+'」？已保存的练习记录会保留。'))return;await run(async()=>{await props.client.request('/api/midis/'+detail.value.id,{method:'DELETE'});detail.value=null;await load()})}
async function report(){const reason=prompt('请输入举报原因（至少 5 个字）');if(!reason)return;await run(async()=>{await props.client.json('/api/reports','POST',{kind:'midi',resourceId:detail.value.id,reason});message.value='举报已提交'})}
watch(scope,()=>{editing.value=null;detail.value=null;page.value=1;songs.value=[];void load()})
watch(()=>props.client.user?.id,()=>{detail.value=null;editing.value=null;page.value=1;if(!props.client.user&&scope.value!=='public')scope.value='public';else void load()})
onMounted(load)
</script>
