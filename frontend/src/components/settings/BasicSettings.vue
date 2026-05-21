<template>
    <section class="content-card">
        <div class="section-title">基础设置</div>
        <div class="setting-grid">
            <div class="setting-row">
                <div>
                    <div class="label">钢琴音量</div>
                    <div class="desc">控制内置 SoundFont 合成器的默认触键音量。</div>
                </div>
                <n-slider v-model:value="store.config.volume" :min="0" :max="100" @dragend="changeConfig" :tooltip="false" />
            </div>

            <div class="setting-row">
                <div>
                    <div class="label">触键力度</div>
                    <div class="desc">电脑键盘或鼠标触发音符时使用的 MIDI velocity。</div>
                </div>
                <n-slider v-model:value="store.config.velocity" :min="1" :max="127" @dragend="changeConfig" :tooltip="false" />
            </div>

            <div class="setting-row">
                <div>
                    <div class="label">采样率</div>
                    <div class="desc">影响音源合成精度与设备兼容性，常用 44100 Hz。</div>
                </div>
                <n-select
                    v-model:value="store.config.sampleRate"
                    :options="sampleRateOptions"
                    size="small"
                    @update:value="changeConfig"
                />
            </div>

            <div class="setting-row">
                <div>
                    <div class="label">缓冲频率</div>
                    <div class="desc">数值越小响应越快，数值越大越稳定；设备性能较弱时建议调高。</div>
                </div>
                <n-select
                    v-model:value="store.config.bufferSize"
                    :options="bufferSizeOptions"
                    size="small"
                    @update:value="changeConfig"
                />
            </div>

            <div class="setting-row">
                <div>
                    <div class="label">MIDI 通道</div>
                    <div class="desc">输出到外部 MIDI 设备时使用的 channel，默认 0。</div>
                </div>
                <n-input-number v-model:value="store.config.midiChannel" :min="0" :max="15" size="small" @update:value="changeConfig" />
            </div>

            <div class="tip-box">
                修改采样率或缓冲频率后，重新加载音源或重启软件可确保底层音频设备完全按新参数初始化。
            </div>


        </div>
    </section>
</template>

<script setup>
import {inject} from 'vue'
import {NButton, NInputNumber, NSelect, NSlider} from 'naive-ui'

const store = inject('store')
const changeConfig = inject('changeConfig')

const sampleRateOptions = [
    {label: '22050 Hz · 轻量', value: 22050},
    {label: '44100 Hz · 标准', value: 44100},
    {label: '48000 Hz · 视频/专业设备常用', value: 48000},
]

const bufferSizeOptions = [
    {label: '512 · 更低延迟', value: 512},
    {label: '1024 · 平衡', value: 1024},
    {label: '2048 · 更稳定', value: 2048},
    {label: '4096 · 低性能设备', value: 4096},
]
</script>
