<template>
 <div class="account-dialog" :style="themeStyle" data-piano-control role="region" aria-label="钢琴账号">
  <header><span>账号</span><button aria-label="关闭账号" @click="$emit('close')"><PhX :size="22"/></button></header>
  <AccountPanel :client="client" :desktop-code="desktopCode"/>
 </div>
</template>
<script setup>
import {PhX} from '@phosphor-icons/vue'
import AccountPanel from './AccountPanel.vue'
defineProps({client:{type:Object,required:true},themeStyle:{type:Object,required:true},desktopCode:{type:String,default:''}})
defineEmits(['close'])
</script>
<style scoped>
.account-dialog{width:520px;max-width:calc(100vw - 32px);max-height:calc(100dvh - 48px);overflow:auto;padding:26px;border:1px solid #ffffff1c;border-radius:16px;background:#26282b;color:#eee;box-shadow:0 24px 90px #0006;font-family:'Segoe UI','Microsoft YaHei',sans-serif}
header{display:flex;align-items:center;justify-content:space-between;padding-bottom:22px;font-size:16px;font-weight:600}header button{display:grid;place-items:center;border:0;background:transparent;color:#b6bac0;cursor:pointer;width:30px;height:30px;border-radius:6px}header button:hover{background:#ffffff0c;color:white}header button:focus-visible{outline:2px solid var(--ui-accent);outline-offset:2px}
.account-dialog :deep(.card){padding:0;border:0;background:none;gap:16px}.account-dialog :deep(h3){font-size:24px;font-weight:600;line-height:1.3}.account-dialog :deep(label){gap:8px;font-size:13px}.account-dialog :deep(input:not([type=checkbox])){min-height:43px;font-size:14px}.account-dialog :deep(.primary){min-height:40px;padding:8px 20px}.account-dialog :deep(.card+.card){border-top:1px solid #ffffff16;padding-top:22px;margin-top:8px}
@media(max-width:600px){.account-dialog{padding:22px;max-height:calc(100dvh - 24px)}}
</style>
