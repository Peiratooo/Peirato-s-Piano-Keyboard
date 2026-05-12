<template>
    <div class="pedal-container">
        <div class="box"></div>
        <div class="pedals">
            <div
                class="pedal"
                :class="item"
                v-for="item in pedals"
                :key="item"
                :style="{color: isPedalDown(item) ? store.config.colors[item].color : '#000000dd'}"
            >
                <n-icon :style="{filter: `drop-shadow(${isPedalDown(item) ? '0 0 4px ' + store.config.colors[item].color : '0 0 2px #ffffff00'})`}">
                    <svg t="1740250796854" class="icon" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" p-id="6026" width="512" height="512"><path d="M256.256 282.048l-21.312-192a42.688 42.688 0 0 1 42.368-47.36h469.376a42.688 42.688 0 0 1 42.368 47.36l-21.312 192a42.688 42.688 0 0 1-17.792 30.144l-181.376 128a42.688 42.688 0 0 1-13.888 6.4v159.296q17.856 7.808 32.704 22.656 31.296 31.296 31.296 75.52v170.624q0 44.16-31.296 75.392-31.232 31.232-75.392 31.232t-75.456-31.232q-31.232-31.232-31.232-75.392V704q0-44.16 31.232-75.456 14.912-14.848 32.768-22.656V446.656a42.624 42.624 0 0 1-13.888-6.464l-181.376-128a42.688 42.688 0 0 1-17.792-30.144z" p-id="6027"></path></svg>
                </n-icon>
            </div>
        </div>
    </div>
</template>

<script setup>
import {NIcon} from 'naive-ui'
import {inject} from 'vue'

const store = inject('store')
const pedals = ['softPedal', 'sostenutoPedal', 'damperPedal']

function isPedalDown(pedalName) {
    const selectedID = store.devices.selectedInDevice
    if (selectedID === -1) return false
    return Boolean(store.devices.pedalStatus?.[selectedID]?.[pedalName])
}
</script>

<style lang="scss" scoped>
.pedal-container {
    position: absolute;
    width: 92px;
    height: 50px;
    padding: 0 4px;
    display: flex;
    overflow: hidden;
    border: 1px solid #55555522;
    box-shadow:  0 0 8px rgba(0, 0, 0, 0.25);
    background-color: rgb(222, 222, 222,0.5);
    backdrop-filter: blur(2px);
    border-radius: 6px;

    .box {
        z-index: 99;
        position: absolute;
        background-color: #222;
        left: 0;
        right: 0;
        height:14px;
        backdrop-filter: blur(8px);
    }

    .pedals {
        display: flex;
        justify-content: space-around;
        width: 100%;
        margin-top: 2px;
        font-size: 32px;
    }

    .pedal {
        display: flex;
        justify-content: center;
        align-items: center;
        transition: 150ms !important;
        width: 8px;
    }
}
</style>
