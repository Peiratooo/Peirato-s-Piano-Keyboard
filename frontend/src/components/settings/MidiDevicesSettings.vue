<template>
    <section class="content-card">
        <div class="section-title">MIDI 设备</div>
        <div class="setting-grid">
            <div class="setting-row vertical">
                <div>
                    <div class="label">输入 MIDI 设备</div>
                    <div class="desc">电子琴 / MIDI 键盘按下时，主窗口会实时反馈。</div>
                </div>
                <n-select
                    size="small"
                    v-model:value="store.devices.selectedInDevice"
                    :options="inDeviceOptions"
                    :disabled="!inDeviceOptions.length"
                    @update:value="changeDevice('in', $event)"
                />
            </div>

            <div class="setting-row vertical">
                <div>
                    <div class="label">输出 MIDI 设备</div>
                    <div class="desc">电脑键盘或鼠标按下时，可以同步发送 MIDI 到外部设备。</div>
                </div>
                <n-select
                    size="small"
                    v-model:value="store.devices.selectedOutDevice"
                    :options="outDeviceOptions"
                    :disabled="!outDeviceOptions.length"
                    @update:value="changeDevice('out', $event)"
                />
            </div>

            <div class="tip-box">
                当前设备列表会跟随后端热插拔扫描自动刷新。设备切换失败时，优先检查设备是否被其他软件占用。
            </div>
        </div>
    </section>
</template>

<script setup>
import {computed, inject} from 'vue'
import {NSelect} from 'naive-ui'

const store = inject('store')
const changeDevice = inject('changeDevice')

const inDeviceOptions = computed(() => buildDeviceOptions(store.devices.inMidiPool))
const outDeviceOptions = computed(() => buildDeviceOptions(store.devices.outMidiPool))

function buildDeviceOptions(pool = {}) {
    return Object.entries(pool)
        .map(([key, device]) => ({
            label: device.name || `MIDI 设备 ${key}`,
            value: Number(device.value ?? key),
        }))
        .sort((a, b) => a.label.localeCompare(b.label, 'zh-Hans-CN'))
}
</script>
