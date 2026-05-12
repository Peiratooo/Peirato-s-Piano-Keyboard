<template>
    <div
        class="container"
        @mouseenter="store.showSetting = true"
        @mouseleave="store.showSetting = false"
    >
        <div class="drag-area">
            <transition enter-active-class="blurFadeIN" leave-active-class="blurFadeOUT">
                <div
                    class="panel main-panel"
                    v-if="store.showSetting"
                    :style="{left: store.menuBar ? '0' : '-515px'}"
                >
                    <button class="icon-button quit" title="退出" @click="Keyboard.Quit()">⏻</button>
                    <button class="icon-button" title="打开设置中心" @click="openControlCenter">⚙</button>
                    <button class="icon-button" title="停止所有声音" @click="allNotesOff">■</button>
                    <div class="slider volume" title="音量">
                        <span class="panel-icon">🔊</span>
                        <n-slider :tooltip="false" v-model:value="store.config.volume" :min="0" :max="127" @dragend="changeConfig"/>
                    </div>
                    <div class="slider velocity" title="力度">
                        <span class="panel-icon">🎹</span>
                        <n-slider :tooltip="false" v-model:value="store.config.velocity" :min="0" :max="127" @dragend="changeConfig"/>
                    </div>
                    <div class="slider opacity" title="透明度">
                        <span class="panel-icon">◩</span>
                        <n-slider :tooltip="false" v-model:value="store.config.opacity" :min="20" :max="100" @dragend="changeConfig"/>
                    </div>
                    <button
                        class="switcher"
                        @click="store.menuBar = !store.menuBar"
                        :style="{transform: store.menuBar ? 'rotate(0deg)' : 'rotate(180deg)'}"
                        title="收起/展开"
                    >‹</button>
                </div>
            </transition>

            <transition enter-active-class="blurFadeIN" leave-active-class="blurFadeOUT">
                <div
                    v-if="store.showSetting"
                    class="keyboard-setting panel"
                    :style="{right: store.keyboardMenu ? '0' : '-510px'}"
                >
                    <button
                        class="switcher"
                        @click="store.keyboardMenu = !store.keyboardMenu"
                        :style="{transform: store.keyboardMenu ? 'rotate(180deg)' : 'rotate(0deg)'}"
                        title="收起/展开"
                    >‹</button>
                    <div class="key-count">
                        <span class="panel-icon">🎚</span>
                        <n-radio-group v-model:value="store.config.keyboardType" @update:value="changeKeyboardType" :disabled="!store.keyboardMenu">
                            <n-radio-button
                                v-for="item in store.keybordType"
                                :key="item.value"
                                :value="item.value"
                                :label="item.label"
                                size="small"
                            />
                        </n-radio-group>
                    </div>
                    <div class="key-tips">
                        <n-select
                            v-model:value="store.config.keyLabel"
                            :options="store.labelMap"
                            size="small"
                            @update:value="changeConfig"
                            :disabled="!store.keyboardMenu"
                        />
                    </div>
                    <div class="show-pedal">
                        <n-radio-group v-model:value="store.config.showPedal" @update:value="changeKeyboardType" :disabled="!store.keyboardMenu">
                            <n-radio-button :value="true" label="显示踏板" size="small"/>
                            <n-radio-button :value="false" label="隐藏踏板" size="small"/>
                        </n-radio-group>
                    </div>
                </div>
            </transition>
        </div>

        <ClassicKeyboard v-if="store.keyboardLoaded"/>

        <Pedal
            v-if="store.loaded && store.config.showPedal"
            class="floating-card"
            @mousemove="dragPedal"
            :style="{left: pedalDrag.x + 'px', top: pedalDrag.y + 'px'}"
        />

        <Chord
            v-if="store.loaded"
            class="floating-card"
            @mousemove="dragChord"
            :style="{left: chordPos.x + 'px', top: chordPos.y + 'px'}"
        />
    </div>
</template>

<script setup>
import {inject, ref} from 'vue'
import {NRadioButton, NRadioGroup, NSelect, NSlider} from 'naive-ui'
import ClassicKeyboard from '../components/ClassicKeyboard.vue'
import Pedal from '../components/Pedal.vue'
import Chord from '../components/Chord.vue'

const store = inject('store')
const Keyboard = inject('Keyboard')
const changeConfig = inject('changeConfig')
const changeKeyboardType = inject('changeKeyboardType')

const pedalDrag = ref({
    x: window.innerWidth * 0.92,
    y: window.innerHeight * 0.7,
    xr: 0.92,
    yr: 0.7,
})

const chordPos = ref({
    x: window.innerWidth * 0.02,
    y: window.innerHeight * 0.65,
    xr: 0.02,
    yr: 0.65,
})

function openControlCenter() {
    Keyboard.OpenControlCenter()
}

function allNotesOff() {
    Keyboard.AllNotesOff()
    store.clearAllKeys()
}

function dragPedal(event) {
    if (event.buttons !== 1) return
    pedalDrag.value.x += event.movementX
    pedalDrag.value.y += event.movementY
    pedalDrag.value.xr = pedalDrag.value.x / window.innerWidth
    pedalDrag.value.yr = pedalDrag.value.y / window.innerHeight
}

function dragChord(event) {
    if (event.buttons !== 1) return
    chordPos.value.x += event.movementX
    chordPos.value.y += event.movementY
    chordPos.value.xr = chordPos.value.x / window.innerWidth
    chordPos.value.yr = chordPos.value.y / window.innerHeight
}
</script>

<style lang="scss" scoped>
.container {
    position: relative;
    border-radius: 6px;
    overflow: hidden;
}

.drag-area {
    position: absolute;
    left: 0;
    right: 0;
    height: 25vh;
    background-color: #1f1f1f00;
    z-index: 999;
    --wails-draggable: drag;
    cursor: move;
    transition: 500ms;
    display: flex;
}

.panel {
    --wails-draggable: no-drag;
    height: 44px;
    align-items: center;
    font-size: 24px;
    gap: 16px;
    display: flex;
    position: relative;
    padding-left: 16px;
    padding-right: 8px;
    background-color: rgba(180, 180, 180, 0.29);
    border: 1px solid #99999933;
    box-shadow: 0 0 8px #00000033;
    backdrop-filter: blur(8px);
    cursor: auto;
    transition: 200ms;
}

.main-panel {
    border-radius: 0 0 16px 0;
}

.icon-button,
.switcher {
    border: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: 200ms;
    --wails-draggable: no-drag;
}

.icon-button {
    width: 28px;
    height: 28px;
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.34);
    color: #333;
    font-size: 15px;
}

.icon-button:hover {
    background: rgba(255, 255, 255, 0.7);
}

.quit {
    background-color: rgba(194, 50, 50, 0.6);
    color: rgba(255, 255, 255, 0.8);
}

.quit:hover {
    background-color: rgba(155, 58, 58, 0.85);
    color: #fff;
}

.slider {
    width: 138px;
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 18px;
}

.panel-icon {
    opacity: 0.7;
    font-size: 15px;
}

.switcher {
    padding: 6px;
    border-radius: 6px;
    font-size: 18px;
    background: transparent;
    color: #333;
}

.switcher:hover {
    background-color: rgba(128, 128, 128, 0.1);
}

.keyboard-setting {
    position: absolute;
    right: 0;
    z-index: 999;
    display: flex;
    height: 44px;
    border-radius: 0 0 0 16px;
    padding-left: 8px;
}

.key-count {
    display: flex;
    align-items: center;
    gap: 8px;
}

.key-tips {
    width: 7rem;
}

.floating-card {
    position: absolute;
    z-index: 9999;
    user-select: none;
    cursor: grab;
}

.floating-card:active {
    cursor: grabbing;
}
</style>
