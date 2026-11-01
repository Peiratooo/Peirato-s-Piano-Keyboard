<template>
    <section class="quick-settings" data-piano-control aria-label="设置">
        <header>
            <div><PhGearSix :size="20" /><h2>设置</h2></div>
            <button type="button" aria-label="关闭设置" @click="$emit('close')"><PhX :size="18" /></button>
        </header>
        <div class="settings-body">
            <label id="quick-volume-label" class="setting-label" for="quick-volume" @click="focusControl('quick-volume')">钢琴音量 <output>{{ store.config.volume }}%</output></label>
            <n-slider v-model:value="store.config.volume" v-piano-label="{id: 'quick-volume'}"
                :min="0" :max="100" :tooltip="false" @dragend="save" @keyup="saveSliderKey" />
            <label id="quick-velocity-label" class="setting-label" for="quick-velocity" @click="focusControl('quick-velocity')">触键力度 <output>{{ store.config.velocity }}</output></label>
            <n-slider v-model:value="store.config.velocity" v-piano-label="{id: 'quick-velocity', description: 'quick-velocity-hint'}"
                :min="1" :max="127" :tooltip="false" @dragend="save" @keyup="saveSliderKey" />
            <p id="quick-velocity-hint" class="setting-hint">键鼠演奏力度；MIDI 按此值 ÷ 127 缩放</p>
            <div class="setting-divider"></div>
            <label id="quick-range-label" class="setting-label" for="quick-range" @click="focusControl('quick-range')">键盘范围</label>
            <n-select v-model:value="store.config.keyboardType" :options="keyboardOptions" size="small"
                v-piano-label="{id: 'quick-range', select: true, expanded: selectionOpen.range}"
                :menu-props="selectMenuProps('quick-range')" @update:show="selectionOpen.range = $event" @update:value="changeKeyboardType" />
            <label id="quick-label-label" class="setting-label" for="quick-label" @click="focusControl('quick-label')">琴键标签</label>
            <n-select v-model:value="store.config.keyLabel" :options="labelOptions" size="small"
                v-piano-label="{id: 'quick-label', select: true, expanded: selectionOpen.label}"
                :menu-props="selectMenuProps('quick-label')" @update:show="selectionOpen.label = $event" @update:value="save" />
            <template v-if="store.config.keyLabel === 'pitch'">
                <label id="quick-tonic-label" class="setting-label" for="quick-tonic" @click="focusControl('quick-tonic')">唱名调性</label>
                <n-select v-model:value="store.config.keyTonic" :options="tonicOptions" size="small"
                    v-piano-label="{id: 'quick-tonic', select: true, expanded: selectionOpen.tonic}"
                    :menu-props="selectMenuProps('quick-tonic')" @update:show="selectionOpen.tonic = $event" @update:value="save" />
            </template>
            <div class="toggle-row">
                <label id="quick-pedal-label" for="quick-pedal" @click.prevent="togglePedal">显示踏板状态</label>
                <n-switch id="quick-pedal" v-model:value="store.config.showPedal" size="small"
                    aria-labelledby="quick-pedal-label" @update:value="save" />
            </div>
        </div>
        <button class="advanced-settings" type="button" @click="$emit('open-full')">
            打开完整设置 <PhArrowUpRight :size="17" />
        </button>
    </section>
</template>

<script setup>
import {computed, inject, reactive} from 'vue'
import {NSelect, NSlider, NSwitch} from 'naive-ui'
import {PhArrowUpRight, PhGearSix, PhX} from '@phosphor-icons/vue'
import {tonicOptions} from '../services/keyDisplay'

defineEmits(['close', 'open-full'])
const store = inject('store')
const save = inject('changeConfig')
const changeKeyboardType = inject('changeKeyboardType')
const keyboardOptions = computed(() => store.keyboardOptions.map(item => ({...item, label: `${item.label} 键`})))
const labelOptions = computed(() => [{label: '不显示', value: ''}, ...store.labelMap])
const selectionOpen = reactive({range: false, label: false, tonic: false})

// Naive's slider/select labels need to be placed on their inner focus target.
// The combobox role also lets the shared piano input guard recognize selection.
function labelControl(element, {value}) {
    const target = element.querySelector('[role="slider"], .n-base-selection-label[tabindex]')
    if (!target) return
    target.id = value.id
    target.setAttribute('aria-labelledby', `${value.id}-label`)
    if (value.description) target.setAttribute('aria-describedby', value.description)
    if (value.select) {
        target.setAttribute('role', 'combobox')
        target.setAttribute('aria-haspopup', 'listbox')
        target.setAttribute('aria-controls', `${value.id}-options`)
        target.setAttribute('aria-expanded', String(!!value.expanded))
    }
}
const vPianoLabel = {mounted: labelControl, updated: labelControl}

function selectMenuProps(id) {
    return {id: `${id}-options`, role: 'listbox', 'aria-labelledby': `${id}-label`, 'data-piano-control': ''}
}
function focusControl(id) {
    document.getElementById(id)?.focus()
}
function saveSliderKey(event) {
    if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(event.key)) save()
}
function togglePedal() {
    focusControl('quick-pedal')
    store.config.showPedal = !store.config.showPedal
    save()
}
</script>

<style scoped>
.quick-settings { width: 320px; display: flex; flex-direction: column; overflow: hidden; color: #eeeeee; background: #2a2a2af5; border: 1px solid #ffffff1c; border-radius: 14px; box-shadow: 0 18px 55px #0005; backdrop-filter: blur(20px); color-scheme: dark; }
header { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 16px 20px 8px; }
header > div { display: flex; align-items: center; gap: 10px; }
h2 { font-size: 16px; font-weight: 600; }
button { border: 0; cursor: pointer; color: inherit; font: inherit; background: transparent; }
header button { display: grid; place-items: center; width: 28px; height: 28px; border-radius: 6px; color: #b8b8b8; }
header button:hover { color: white; background: #ffffff12; }
.settings-body { padding: 0 22px 10px; overflow: auto; min-height: 0; }
.setting-label { display: flex; align-items: center; justify-content: space-between; margin: 11px 0 8px; font-size: 13px; color: #d7d7d7; }
output { font-variant-numeric: tabular-nums; color: #c5c5c5; font-size: 12px; }
.setting-hint { margin-top: 6px; color: #999999; font-size: 11px; line-height: 1.5; }
.setting-divider { height: 1px; background: #ffffff10; margin: 14px 0 2px; }
.toggle-row { display: flex; align-items: center; justify-content: space-between; margin: 18px 0 10px; font-size: 13px; color: #d7d7d7; cursor: pointer; }
.toggle-row label { cursor: pointer; }
.advanced-settings { flex-shrink: 0; display: flex; justify-content: space-between; align-items: center; padding: 18px 22px; font-size: 12px; color: #bbbbbb; border-top: 1px solid #ffffff10; }
.advanced-settings:hover { background: #ffffff06; color: #eeeeee; }
button:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 3px; }
</style>
