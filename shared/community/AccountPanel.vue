<template>
 <section class="community">
  <div v-if="error" role="alert" class="error">{{error}}</div>
  <form v-if="!client.user" class="card" @submit.prevent="submit">
   <h3>{{register?'创建钢琴账号':'登录钢琴账号'}}</h3>
   <p class="muted">在桌面和 Toy 使用同一个账号，保存自己的作品与收藏。</p>
   <label>账号<input v-model.trim="username" autocomplete="username" minlength="3" maxlength="24" required placeholder="3–24 位字母、数字或下划线"/></label>
   <label>密码<input v-model="password" type="password" :autocomplete="register?'new-password':'current-password'" minlength="10" maxlength="72" required placeholder="至少 10 位"/></label>
   <label v-if="client.desktop" class="check"><input v-model="remember" type="checkbox"/>记住登录（7 天）</label>
   <div class="row"><button class="primary" :disabled="busy">{{busy?'正在处理…':register?'注册并登录':'登录'}}</button><button type="button" @click="register=!register">{{register?'已有账号':'创建账号'}}</button><button v-if="client.desktop" type="button" :disabled="busy" @click="run(()=>client.beginToyLogin(remember))">通过 Toy 登录</button></div>
  </form>
  <div v-if="client.desktopGrant && !client.user" class="card">
   <h3>请在 Toy 中确认登录</h3><p>确认码 <strong>{{client.desktopGrant.userCode}}</strong></p><p class="muted">请核对网页上的确认码。授权 5 分钟内有效，完成后软件会自动登录。</p>
   <div class="row"><button @click="client.openToyLogin()">重新打开网页</button><button @click="client.cancelToyLogin()">取消</button></div>
  </div>
  <div v-if="client.user" class="card">
   <div class="row spread"><div class="identity"><img v-if="client.user.avatarUrl" class="avatar" :src="client.API+client.user.avatarUrl" alt="我的头像"/><PhUserCircle v-else :size="38"/><div><strong>{{client.user.nickname}}</strong><div class="muted">@{{client.user.username}}</div></div></div><button :disabled="busy" @click="run(()=>client.logout())">退出登录</button></div>
   <div class="row"><label class="grow">昵称<input v-model.trim="nickname" maxlength="24" placeholder="2–24 个字"/></label><button :disabled="busy" @click="saveProfile">保存昵称</button><button :disabled="busy" @click="run(()=>client.changeAvatar())">设置头像</button></div>
   <p class="muted">头像支持 PNG/JPEG，最大 2MB。资料由你自行设置。</p>
  </div>
  <div v-if="desktopCode" class="card">
   <h3>登录桌面 PP 钢琴键盘</h3><p>网页确认码 <strong>{{desktopCode}}</strong></p><p class="muted">请核对与你软件中显示的确认码一致，再授权。</p>
   <p v-if="authorized" class="message">{{authorized}}，可回到钢琴软件。</p>
   <div v-else class="row"><button class="primary" :disabled="!client.user||busy" @click="authorize(true)">允许登录</button><button :disabled="!client.user||busy" @click="authorize(false)">取消授权</button></div>
  </div>
 </section>
</template>
<script setup>
import {ref,watch,onBeforeUnmount} from 'vue'
import {PhUserCircle} from '@phosphor-icons/vue'
import './community.css'
const props=defineProps({client:{type:Object,required:true},desktopCode:{type:String,default:''}})
const username=ref(''),password=ref(''),nickname=ref(''),register=ref(false),remember=ref(true),busy=ref(false),error=ref(''),authorized=ref('')
watch(()=>props.client.user?.nickname,value=>nickname.value=value||'',{immediate:true})
async function run(action){busy.value=true;error.value='';try{return await action()}catch(e){error.value=e.message||String(e)}finally{busy.value=false}}
async function submit(){await run(async()=>{await props.client.auth(register.value?'register':'login',username.value,password.value,remember.value);password.value=''})}
async function saveProfile(){await run(async()=>{const u=await props.client.json('/api/auth/me','PATCH',{nickname:nickname.value});props.client.user=u})}
async function authorize(approve){await run(async()=>{await props.client.json('/api/auth/desktop/confirm','POST',{userCode:props.desktopCode,approve});authorized.value=approve?'已授权':'已取消';const url=new URL(location.href);url.searchParams.delete('desktopCode');history.replaceState(null,'',url)})}
onBeforeUnmount(()=>{password.value=''})
</script>
