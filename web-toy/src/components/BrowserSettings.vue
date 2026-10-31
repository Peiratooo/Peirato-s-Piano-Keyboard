<template>
  <div class="control-window" role="region" aria-label="完整设置">
    <NConfigProvider :theme="null" :theme-overrides="lightOverrides" :style="lightTheme.cssVariables">
      <div class="titlebar"><div><div class="title">PP 钢琴键盘 · 设置</div><div class="subtitle">自定义你的键盘显示、声音与 MIDI 设备。</div></div>
        <NButton quaternary circle aria-label="关闭完整设置" @click="$emit('close')"><template #icon><PhX :size="20"/></template></NButton></div>
      <div class="settings-shell">
        <aside class="setting-menu"><button v-for="item in menus" :key="item.key" class="menu-item" :class="{active:store.controlMenu===item.key}" @click="store.controlMenu=item.key">
          <span class="menu-icon">{{item.icon}}</span><span>{{item.label}}</span></button></aside>
        <main class="setting-content">
          <section v-if="store.controlMenu==='basic'" class="content-card">
            <div class="section-title">基础设置</div><div class="setting-grid">
              <div class="setting-row"><div><div class="label">钢琴音量</div><div class="desc">控制内置 SoundFont 合成器的总体播放音量。</div></div>
                <NSlider v-model:value="store.config.volume" :min="0" :max="100" :tooltip="false" @dragend="save" @keyup="save"/></div>
              <div class="setting-row"><div><div class="label">触键力度</div><div class="desc">设置键鼠演奏力度；MIDI 播放力度 = 原力度 × 此值 ÷ 127，127 保留原力度。</div></div>
                <NSlider v-model:value="store.config.velocity" :min="1" :max="127" :tooltip="false" @dragend="save" @keyup="save"/></div>
              <div class="setting-row"><div><div class="label">采样率</div><div class="desc">当前浏览器音频设备的采样率。</div></div><NInput :value="piano.audio.context?`${piano.audio.context.sampleRate} Hz`:'开始演奏后显示'" readonly/></div>
              <div class="setting-row"><div><div class="label">音频延迟</div><div class="desc">由浏览器管理，演奏采用交互延迟模式。</div></div><NInput :value="piano.audio.context?`${Math.round((piano.audio.context.baseLatency||0)*1000)} ms`:'开始演奏后显示'" readonly/></div>
            </div>
          </section>
          <section v-else-if="store.controlMenu==='devices'" class="content-card">
            <div class="section-title">MIDI 设备</div><div class="setting-grid">
              <div class="setting-row vertical"><div><div class="label">输入 MIDI 设备</div><div class="desc">电子琴 / MIDI 键盘按下时，主窗口会实时反馈。</div></div>
                <div class="device-line"><NInput :value="midiStatus" readonly/><NButton @click="$emit('connect-midi')">连接设备</NButton></div></div>
              <div class="setting-row"><div><div class="label">软件延音踏板</div><div class="desc">开启后，松开琴键仍会延音；关闭时释放。</div></div><NSwitch :value="store.devices.softwareSustain" @update:value="toggleSustain"/></div>
              <div class="tip-box">需要浏览器支持 Web MIDI。设备授权由浏览器管理。</div>
            </div>
          </section>
          <KeyboardDisplaySettings v-else-if="store.controlMenu==='keyboard'"/>
          <KeyboardMappingSettings v-else-if="store.controlMenu==='keymap'"/>
          <AppearanceSettings v-else-if="store.controlMenu==='appearance'"/>
          <section v-else-if="store.controlMenu==='soundfont'" class="content-card soundfont-page">
            <div class="soundfont-hero"><div><div class="eyebrow">SoundFont Library</div><div class="section-title hero-title">音源选择</div><div class="hero-desc">管理本地 .sf2 音色库，切换后会同步应用到钢琴播放与练习。</div></div>
              <div class="hero-actions"><NButton type="primary" size="small" @click="$emit('import-sf2')">导入音源</NButton></div></div>
            <div class="soundfont-toolbar"><div><div class="label">音源库</div><div class="desc">当前音源：{{soundFontName}}</div></div></div>
            <div class="soundfont-list"><div class="soundfont-item active"><div class="soundfont-icon default-icon"><span>SF2</span></div>
              <div class="soundfont-item-main"><div class="item-title-line"><span class="label">{{soundFontName}}</span><NTag size="small" type="success" :bordered="false">正在使用</NTag></div>
                <div class="desc">{{soundFontName==='轻量钢琴'?'默认合成音色':'本次会话导入的采样音源'}}</div></div></div></div>
            <p class="desc">SF2 文件保留在本机。重新打开页面后，可再次导入自选音源。</p>
          </section>
          <section v-else class="content-card about-page"><img :src="baseUrl+'appicon.png'" alt="PP 钢琴键盘 Logo" width="64" height="64"/>
            <div class="section-title">PP 钢琴键盘</div><p class="desc">自由演奏、MIDI 播放与跟弹练习。</p><div class="about-actions">
              <NButton @click="$emit('share')">分享 Toy</NButton><NButton @click="$emit('immerse')">横屏沉浸 / 全屏</NButton></div></section>
        </main>
      </div>
    </NConfigProvider>
  </div>
</template>
<script setup>
import {computed, inject, onBeforeUnmount} from 'vue'
import {NButton,NConfigProvider,NInput,NSlider,NSwitch,NTag} from 'naive-ui'
import {PhX} from '@phosphor-icons/vue'
import KeyboardDisplaySettings from '../../../frontend/src/components/settings/KeyboardDisplaySettings.vue'
import KeyboardMappingSettings from '../../../frontend/src/components/settings/KeyboardMappingSettings.vue'
import AppearanceSettings from '../../../frontend/src/components/settings/AppearanceSettings.vue'
import {getPianoUiTheme} from '../../../frontend/src/services/uiTheme'
const props=defineProps({store:{type:Object,required:true},piano:{type:Object,required:true},midiStatus:String,soundFontName:String})
defineEmits(['close','connect-midi','import-sf2','share','immerse'])
const save=inject('changeConfig'),toggleSustain=inject('toggleSustain')
const baseUrl=import.meta.env.BASE_URL
const lightTheme=computed(()=>getPianoUiTheme(props.store.config.colors,'light'))
const lightOverrides=computed(()=>({common:{primaryColor:lightTheme.value.accent,primaryColorHover:lightTheme.value.accentHover,primaryColorPressed:lightTheme.value.accentPressed,primaryColorSuppl:lightTheme.value.accent},Button:lightTheme.value.naiveOverrides.Button}))
const menus=[{key:'basic',label:'基础设置',icon:'⌘'},{key:'devices',label:'MIDI 设备',icon:'🎛'},{key:'keyboard',label:'键盘显示',icon:'🎹'},{key:'keymap',label:'按键方案',icon:'⌨'},{key:'appearance',label:'外观主题',icon:'🎨'},{key:'soundfont',label:'音源管理',icon:'🎧'},{key:'about',label:'关于软件',icon:'i'}]
onBeforeUnmount(save)
</script>
<style lang="scss" src="../../../shared/piano/settings.scss"></style>
<style lang="scss" scoped src="../../../shared/piano/soundfont.scss"></style>
<style scoped>
.control-window{width:min(1180px,calc(100vw - 48px));max-height:calc(100dvh - 48px);overflow:auto;background:#f3f3f3;color:#303030;border-radius:12px;user-select:none}
.titlebar{display:flex;align-items:center;justify-content:space-between;padding:22px 28px 16px}.title{font-size:22px;font-weight:700;letter-spacing:-.02em}.subtitle{margin-top:6px;font-size:13px;color:#666}
.device-line,.about-actions{display:flex;align-items:center;gap:10px}.about-page>img{margin-bottom:20px}.about-actions{margin-top:24px;flex-wrap:wrap}
@media(max-width:600px){.control-window{width:calc(100vw - 16px);max-height:calc(100dvh - 16px)}.titlebar{padding:20px 18px 16px}.title{font-size:20px}.subtitle{line-height:1.5}}
</style>
