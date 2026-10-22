<template>
    <section class="content-card appearance-page">
        <div class="section-title compact-title">外观主题</div>

        <div class="theme-panel">
            <div class="theme-copy">
                <div class="label">主题方案</div>
                <div class="desc">{{ selectedPalette?.desc }}</div>
            </div>
            <n-select
                v-model:value="selectedPaletteKey"
                :options="paletteOptions"
                size="small"
                class="theme-select"
                @update:value="applyPaletteByKey"
            />
        </div>

        <div class="palette-preview">
            <span
                v-for="color in selectedPalette?.preview"
                :key="color"
                class="preview-swatch"
                :style="{background: color}"
            />
        </div>

        <div class="example-card">
            <ExampleKeyboard />
            <ExamplePedal class="example-pedal" />
        </div>

        <div class="color-grid">
            <div class="color-row" v-for="(item, key) in store.config.colors" :key="key">
                <div class="color-label">
                    <div >{{ item.label }}</div>

                </div>
                <n-color-picker
                    v-model:value="store.config.colors[key].color"
                    :show-alpha="false"
                    size="small"
                    :modes="['hex']"
                    @update:value="updateKeyColor"
                    @complete="changeConfig"
                />
            </div>
        </div>
    </section>
</template>

<script setup>
import {computed, inject, ref} from 'vue'
import {NCollapse, NCollapseItem, NColorPicker, NSelect} from 'naive-ui'
import ExampleKeyboard from '../ExampleKeyboard.vue'
import ExamplePedal from '../ExamplePedal.vue'
import {colorPalettes} from '../../services/themes'

const store = inject('store')
const changeConfig = inject('changeConfig')
const setKeyColor = inject('setKeyColor')

const selectedPaletteKey = ref('fresh')


const paletteOptions = computed(() => colorPalettes.map((palette) => ({
    label: palette.label,
    value: palette.key,
})))
const selectedPalette = computed(() => colorPalettes.find((palette) => palette.key === selectedPaletteKey.value) || colorPalettes[0])

function updateKeyColor() {
    setKeyColor?.()
}

function applyPaletteByKey(key) {
    const palette = colorPalettes.find((item) => item.key === key)
    if (!palette) return
    applyPalette(palette)
}

function applyPalette(palette) {
    for (const [key, color] of Object.entries(palette.colors)) {
        if (!store.config.colors[key]) continue
        store.config.colors[key] = {
            ...store.config.colors[key],
            color,
        }
    }
    setKeyColor?.()
    changeConfig?.()
}
</script>

<style lang="scss" scoped>
.appearance-page {
    display: flex;
    flex-direction: column;
    gap: 14px;
}

.compact-title {
    margin-bottom: 2px;
}

.theme-panel {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 220px;
    align-items: center;
    gap: 16px;
    padding: 14px;
    border-radius: 18px;
    background: var(--setting-subtle-bg);
    border: 1px solid rgba(148, 163, 184, 0.18);
}

.theme-select {
    width: 100%;
}

.palette-preview {
    display: flex;
    gap: 8px;
    padding: 0 2px;
}

.preview-swatch {
    width: 32px;
    height: 18px;
    border-radius: 999px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.08);
}

.example-card {
    position: relative;
    height: 150px;
    overflow: hidden;
    border-radius: 18px;
    background: rgba(15, 23, 42, 0.05);
    padding: 16px;
}

.example-pedal {
    position: absolute;
    right: 16px;
    bottom: 16px;
    z-index: 99;
}

.manual-collapse {
    :deep(.n-collapse-item__header-main) {
        font-size: 14px;
        font-weight: 650;
        color: var(--setting-text-main);
    }
}

.color-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(128px, 1fr));
    gap: 10px;
}

.color-row {
   display: flex;
    flex-direction: column;
    align-items: center;
    gap:6px;
    padding: 10px;
    border-radius: 14px;
    background: rgba(248, 250, 252, 0.78);
    border: 1px solid rgba(148, 163, 184, 0.16);
}

.color-label {
    min-width: 0;
}

@media (max-width: 760px) {
    .theme-panel,
    .color-grid,
    .color-row {
        grid-template-columns: minmax(0, 1fr);
    }

    .example-card {
        height: 135px;
    }
}
</style>
