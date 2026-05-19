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

const store = inject('store')
const changeConfig = inject('changeConfig')
const setKeyColor = inject('setKeyColor')

const selectedPaletteKey = ref('fresh')

const colorPalettes = [
    {
        key: 'fresh',
        label: '清透薄荷',
        desc: '清爽明亮，接近默认风格。',
        preview: ['#9AF7B3', '#5FFF5F', '#f7e89a', '#ffd25f', '#10e786'],
        colors: {
            whiteKey: '#9AF7B3',
            blackKey: '#5FFF5F',
            whiteKeyLeft: '#f7e89a',
            blackKeyLeft: '#ffd25f',
            damperPedal: '#e7b510',
            softPedal: '#10e786',
            sostenutoPedal: '#1054e7',
        },
    },
    {
        key: 'apple-blue',
        label: '苹果蓝紫',
        desc: '现代冷色，高亮清晰。',
        preview: ['#7DD3FC', '#38BDF8', '#C4B5FD', '#8B5CF6', '#34D399'],
        colors: {
            whiteKey: '#7DD3FC',
            blackKey: '#38BDF8',
            whiteKeyLeft: '#C4B5FD',
            blackKeyLeft: '#8B5CF6',
            damperPedal: '#F59E0B',
            softPedal: '#34D399',
            sostenutoPedal: '#60A5FA',
        },
    },
    {
        key: 'warm-stage',
        label: '暖色舞台',
        desc: '偏演奏氛围，温暖。',
        preview: ['#FDBA74', '#FB923C', '#FDE68A', '#F59E0B', '#A78BFA'],
        colors: {
            whiteKey: '#FDBA74',
            blackKey: '#FB923C',
            whiteKeyLeft: '#FDE68A',
            blackKeyLeft: '#F59E0B',
            damperPedal: '#FACC15',
            softPedal: '#4ADE80',
            sostenutoPedal: '#A78BFA',
        },
    },
    {
        key: 'soft-pro',
        label: '低饱和专业',
        desc: '更克制，适合长时间练习。',
        preview: ['#A7F3D0', '#6EE7B7', '#BFDBFE', '#93C5FD', '#FCD34D'],
        colors: {
            whiteKey: '#A7F3D0',
            blackKey: '#6EE7B7',
            whiteKeyLeft: '#BFDBFE',
            blackKeyLeft: '#93C5FD',
            damperPedal: '#FCD34D',
            softPedal: '#86EFAC',
            sostenutoPedal: '#93C5FD',
        },
    },
    {
        key: 'neon-practice',
        label: '霓虹练习',
        desc: '对比更强，适合快速观察按键反馈。',
        preview: ['#22D3EE', '#06B6D4', '#F0ABFC', '#D946EF', '#FDE047'],
        colors: {
            whiteKey: '#22D3EE',
            blackKey: '#06B6D4',
            whiteKeyLeft: '#F0ABFC',
            blackKeyLeft: '#D946EF',
            damperPedal: '#FDE047',
            softPedal: '#4ADE80',
            sostenutoPedal: '#60A5FA',
        },
    },
    {
        key: 'classic-ivory',
        label: '古典象牙',
        desc: '柔和暖白，适合浅色界面。',
        preview: ['#FDECC8', '#E9B872', '#D8F3DC', '#95D5B2', '#8ECAE6'],
        colors: {
            whiteKey: '#FDECC8',
            blackKey: '#E9B872',
            whiteKeyLeft: '#D8F3DC',
            blackKeyLeft: '#95D5B2',
            damperPedal: '#FFB703',
            softPedal: '#52B788',
            sostenutoPedal: '#8ECAE6',
        },
    },
    {
        key: 'studio-focus',
        label: '录音棚聚焦',
        desc: '灰蓝基调，状态色更稳重。',
        preview: ['#CBD5E1', '#94A3B8', '#BAE6FD', '#38BDF8', '#FBBF24'],
        colors: {
            whiteKey: '#CBD5E1',
            blackKey: '#94A3B8',
            whiteKeyLeft: '#BAE6FD',
            blackKeyLeft: '#38BDF8',
            damperPedal: '#FBBF24',
            softPedal: '#6EE7B7',
            sostenutoPedal: '#93C5FD',
        },
    },
]

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
