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
                    <div class="desc">选择发声目标：不发音、仅软件音源，或仅发送到指定外部 MIDI 设备。</div>
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
const outDeviceOptions = computed(() => buildDeviceOptions(store.devices.outMidiPool, true))

function buildDeviceOptions(pool = {}, isOutput = false) {
    return Object.entries(pool)
        .map(([key, device]) => {
            const value = Number(device.value ?? key)
            const specialOutput = isOutput ? getSpecialOutputOption(value) : null
            return {
                label: specialOutput?.label || device.name || `MIDI 设备 ${key}`,
                value,
                order: specialOutput?.order ?? 10,
            }
        })
        .sort((a, b) => a.order - b.order || a.label.localeCompare(b.label, 'zh-Hans-CN'))
        .map(({label, value}) => ({label, value}))
}

function getSpecialOutputOption(value) {
    if (value === -1) return {label: '无（不发音）', order: 0}
    if (value === -2) return {label: '软件音源', order: 1}
    return null
}
</script>
