<template>
    <div class="classic-keyboard" @mousedown="mouse.down = true" @mouseup="mouse.down=false;" @mouseleave="mouse.down=false;mouse.keyIndex=-1">
        <div class="key" v-for="(item, index) in store.keyboardConfig.slice(39,63)"
             :class="[item.color,item.note,(mouse.down && mouse.keyIndex===index) || [2,5,8,14,17,20].includes(index) ? keyColorMap[item.color]:'',]"
             :key="index" @mouseenter="mouse.keyIndex=index">
            <div class="label" v-if="activeLabel !== ''">
                {{item[activeLabel]}}
            </div>
        </div>
    </div>
</template>

<script setup>
import {inject, onMounted,ref} from "vue";

const store = inject("store")

const keyColorMap = {
    'black':"b-active",
    'white':"w-active",
    'black-l':"b-l-active",
    'white-l':"w-l-active",
}
const mouse = ref({
    down:false,
    keyIndex:-1,
})

const activeLabel = ref("")


const size = ref({
    wWidth:0,
    bWidth:0,
    gap:0,
    ratio:1.6
})

function resize() {
    size.value.wWidth = 38
    size.value.bWidth = size.value.wWidth / size.value.ratio
    size.value.gap = size.value.bWidth / size.value.ratio
    document.documentElement.style.setProperty('--example-black-key-width', size.value.bWidth+'px');
    document.documentElement.style.setProperty('--example-white-key-width', size.value.wWidth+'px');
    document.documentElement.style.setProperty('--example-white-key-offset', -size.value.bWidth+'px');
    document.documentElement.style.setProperty('--example-black-key-offset', -size.value.gap*0.7+'px');
}

onMounted(()=>{
    resize()
})

</script>

<style lang="scss" scoped>

.classic-keyboard {
    display: flex;
    flex-direction: row;
    height: 100%;
}

.key {
    position: relative;
    transition: 150ms;
    height: 100%;
    user-select: none;
}
.label {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 10%;
    display: flex;
    justify-content: center;
}
.white {
    height:100%;
    width:var(--example-white-key-width);
    z-index:1;
    border-left:1px solid #bbb;
    border-bottom:1px solid #bbb;
    border-right: 0 solid #bbb;
    border-radius:0 0 5px 5px;
    box-shadow:-1px 0 0 rgba(255,255,255,0.8) inset,0 0 5px #ccc inset,0 0 3px rgba(0,0,0,0.2);
    background-color:#eee;
    .label {
        font-size: 18px;
    }
}
.w-active {
    border-left:1px solid var(--whiteKey-o);
    border-bottom:1px solid var(--whiteKey-o);
    border-right: 1px solid var(--whiteKey-o);
    box-shadow:3px 0 3px rgba(0,0,0,0.1) inset,-3px 0 8px rgba(0,0,0,0.1) inset,0 0 3px rgba(0,0,0,0.2);
    background-color: var(--whiteKey);
}
.black {
    height:63%;
    width: var(--example-black-key-width);
    transform: translateX(var(--example-black-key-offset));
    z-index:3;
    border-bottom:1px solid #000;
    border-right: 0 solid #000;
    border-left:1px solid #000;
    border-radius:0 0 3px 3px;
    box-shadow:-1px -1px 2px rgba(255,255,255,0.2) inset,0 -5px 2px 3px rgba(0,0,0,0.6) inset,0 2px 4px rgba(0,0,0,0.5);
    background-color:#333;
    .label {
        font-size: 13px;
        color: #eee;
    }
}

.b-active {
    box-shadow:-1px -5px 2px rgba(131, 131, 131, 0.2) inset,0 -10px 10px 5px rgba(0,0,0,0.6) inset,0 1px 2px rgba(0,0,0,0.5);
    background-color: var(--blackKey);

    border-bottom:1px solid var(--blackKey-o);
    border-right: 0 solid var(--blackKey-o);
    border-left:1px solid var(--blackKey-o);
}


.A:first-child {
    margin: 0;
}
.B,.D,.E,.A,.G {
    margin-left: var(--example-white-key-offset);
}
</style>