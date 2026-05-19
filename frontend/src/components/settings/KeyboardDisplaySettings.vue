<template>
    <section class="content-card">
        <div class="section-title">键盘显示</div>
        <div class="setting-grid">
            <div class="setting-row">
                <div>
                    <div class="label">键盘范围</div>
                    <div class="desc">主窗口显示的琴键数量。</div>
                </div>
                <n-radio-group v-model:value="store.config.keyboardType" @update:value="changeKeyboardType">
                    <n-radio-button
                        v-for="item in store.keyboardOptions"
                        :key="item.value"
                        :value="item.value"
                        :label="item.label"
                        size="small"
                    />
                </n-radio-group>
            </div>

            <div class="setting-row">
                <div>
                    <div class="label">琴键标签</div>
                    <div class="desc">显示八度、音名、唱名或电脑键盘映射。</div>
                </div>
                <n-select v-model:value="store.config.keyLabel" :options="store.labelMap" size="small" @update:value="changeConfig" />
            </div>

            <div class="setting-row">
                <div>
                    <div class="label">踏板显示</div>
                    <div class="desc">主窗口是否显示踏板状态浮层。</div>
                </div>
                <n-switch v-model:value="store.config.showPedal" @update:value="changeConfig" />
            </div>
        </div>
    </section>
</template>

<script setup>
import {inject} from 'vue'
import {NRadioButton, NRadioGroup, NSelect, NSwitch} from 'naive-ui'

const store = inject('store')
const changeConfig = inject('changeConfig')
const changeKeyboardType = inject('changeKeyboardType')
</script>
