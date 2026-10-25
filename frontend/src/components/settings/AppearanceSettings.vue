<template>
    <section v-if="store.loaded" class="content-card appearance-page">
        <div class="appearance-heading"><div class="section-title">外观主题</div><span>{{ colorPalettes.length }} 组预设</span></div>

        <div class="theme-panel">
            <div class="theme-copy">
                <div class="label">主题方案</div>
                <div class="desc">{{ selectedPalette ? selectedPalette.desc : '当前为自定义配色，可继续调色或选择预设。' }}</div>
            </div>
            <n-select
                :value="selectedPalette ? selectedPalette.key : 'custom'"
                :options="paletteOptions"
                :render-label="renderPaletteLabel"
                :virtual-scroll="false"
                filterable
                aria-label="配色方案"
                class="theme-select"
                @update:value="applyPaletteByKey"
            />
        </div>

        <PalettePreview :colors="store.config.colors" />

        <n-collapse class="custom-colors" arrow-placement="right">
            <n-collapse-item name="colors" title="自定义颜色">
                <template #header-extra><span class="custom-summary">微调琴键与踏板</span></template>
                <div class="color-groups">
                    <section v-for="group in colorGroups" :key="group.label" class="color-group" :class="{'pedal-colors': group.label === '踏板'}">
                        <h3>{{ group.label }}</h3>
                        <div class="color-fields">
                            <div v-for="field in group.fields" :key="field.key" class="color-field">
                                <div :id="'color-' + field.key" class="color-label">{{ field.label }}</div>
                                <n-color-picker
                                    v-model:value="store.config.colors[field.key].color"
                                    :aria-labelledby="'color-' + field.key"
                                    :show-alpha="false"
                                    :modes="['hex']"
                                    @update:value="setKeyColor"
                                    @complete="changeConfig"
                                />
                            </div>
                        </div>
                    </section>
                </div>
            </n-collapse-item>
        </n-collapse>
    </section>
</template>

<script setup>
import {computed, h, inject} from 'vue'
import {NCollapse, NCollapseItem, NColorPicker, NSelect} from 'naive-ui'
import PalettePreview from '../PalettePreview.vue'
import {colorPalettes, paletteGroups} from '../../services/themes'

const store = inject('store')
const changeConfig = inject('changeConfig')
const setKeyColor = inject('setKeyColor')

const colorGroups = [
    {label: '左手', fields: [{key: 'whiteKeyLeft', label: '白键'}, {key: 'blackKeyLeft', label: '黑键'}]},
    {label: '右手', fields: [{key: 'whiteKey', label: '白键'}, {key: 'blackKey', label: '黑键'}]},
    {label: '踏板', fields: [{key: 'damperPedal', label: '延音'}, {key: 'softPedal', label: '柔音'}, {key: 'sostenutoPedal', label: '持音'}]},
]
const selectedPalette = computed(() => {
    const current = Object.fromEntries(Object.entries(store.config.colors).map(([key, value]) => [key, value.color.toUpperCase()]))
    return colorPalettes.find(palette => Object.entries(palette.colors).every(([key, color]) => current[key] === color))
})
const paletteOptions = computed(() => [
    ...(!selectedPalette.value ? [{label: '自定义配色', value: 'custom'}] : []),
    ...paletteGroups.map(group => ({type: 'group', key: group, label: group,
        children: colorPalettes.filter(palette => palette.group === group).map(palette => ({
            label: palette.label + (palette.recommended ? ' · 推荐' : ''), value: palette.key, palette,
        })),
    })),
])

function renderPaletteLabel(option) {
    if (!option.palette) return option.label
    return h('span', {style: {display: 'flex', alignItems: 'center', gap: '10px', width: '100%'}}, [
        h('span', {style: {flex: 1}}, option.label),
        h('span', {'aria-hidden': true, style: {display: 'flex', alignItems: 'center', gap: '4px', fontSize: '10px', color: '#777'}}, [
            '左', h('span', {style: {width: '10px', height: '10px', borderRadius: '3px', background: option.palette.colors.whiteKeyLeft}}),
            '右', h('span', {style: {width: '10px', height: '10px', borderRadius: '3px', background: option.palette.colors.whiteKey}}),
        ]),
    ])
}

function applyPaletteByKey(key) {
    if (key === 'custom') return
    const palette = colorPalettes.find((item) => item.key === key)
    for (const [key, color] of Object.entries(palette.colors)) {
        store.config.colors[key].color = color
    }
    setKeyColor()
    changeConfig()
}
</script>

<style lang="scss" scoped>
.appearance-page {
    display: flex;
    flex-direction: column;
    gap: 24px;
}

.appearance-page .section-title { margin: 0; }
.appearance-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.appearance-heading > span { color: #777; font-size: 12px; }

.theme-panel {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(240px, 44%);
    align-items: center;
    gap: 24px;
}

.theme-select {
    width: 100%;
}

.custom-colors { border-top: 1px solid #e8e8e8; padding-top: 18px; }
.custom-summary { color: #777; font-size: 12px; margin-right: 10px; }
.color-groups {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 24px 32px;
    padding-top: 8px;
}
.color-group { min-width: 0; }
.color-group h3 { margin: 0 0 12px; color: #454545; font-size: 13px; font-weight: 600; }
.color-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.color-field {
    display: flex;
    flex-direction: column;
    min-width: 0;
    gap: 8px;
}
.color-label { font-size: 12px; color: #666; }
.pedal-colors { grid-column: 1 / -1; padding-top: 20px; border-top: 1px solid #ededed; }
.pedal-colors .color-fields { grid-template-columns: repeat(3, minmax(0, 1fr)); }

@media (max-width: 760px) {
    .theme-panel {
        grid-template-columns: minmax(0, 1fr);
    }
}
@media (max-width: 560px) {
    .color-groups { grid-template-columns: minmax(0, 1fr); }
    .custom-summary { display: none; }
}
</style>
