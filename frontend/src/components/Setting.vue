<template>
    <div class="settings-shell">
        <aside class="setting-menu">
            <button
                v-for="item in menus"
                :key="item.key"
                class="menu-item"
                :class="{active: store.controlMenu === item.key}"
                @click="store.controlMenu = item.key"
            >
                <span class="menu-icon">{{ item.icon }}</span>
                <span>{{ item.label }}</span>
            </button>
        </aside>

        <main class="setting-content">
            <section v-if="store.controlMenu === 'basic'" class="content-card">
                <div class="section-title">基础设置</div>
                <div class="setting-grid">
                    <div class="setting-row">
                        <div>
                            <div class="label">音量</div>
                            <div class="desc">只控制本软件内置音源的输出音量。</div>
                        </div>
                        <n-slider v-model:value="store.config.volume" :min="0" :max="100" @dragend="changeConfig"/>
                    </div>
                    <div class="setting-row">
                        <div>
                            <div class="label">默认力度</div>
                            <div class="desc">电脑键盘或鼠标触发音符时使用的按键力度。</div>
                        </div>
                        <n-slider v-model:value="store.config.velocity" :min="1" :max="127" @dragend="changeConfig"/>
                    </div>
                    <div class="setting-row">
                        <div>
                            <div class="label">MIDI 通道</div>
                            <div class="desc">输出到外部 MIDI 设备时使用的 channel，默认 0。</div>
                        </div>
                        <n-input-number v-model:value="store.config.midiChannel" :min="0" :max="15" size="small" @update:value="changeConfig"/>
                    </div>
                    <div class="setting-actions">
                        <n-button type="info" size="small" @click="resetConfig">恢复默认设置</n-button>
                    </div>
                </div>
            </section>

            <section v-else-if="store.controlMenu === 'devices'" class="content-card">
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
                            @update:value="changeDevice('out', $event)"
                        />
                    </div>
                    <div class="tip-box">
                        当前设备列表会跟随后端热插拔扫描自动刷新。设备切换失败时，优先检查设备是否被其他软件占用。
                    </div>
                </div>
            </section>

            <section v-else-if="store.controlMenu === 'keyboard'" class="content-card">
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
                        <n-select v-model:value="store.config.keyLabel" :options="store.labelMap" size="small" @update:value="changeConfig"/>
                    </div>
                    <div class="setting-row">
                        <div>
                            <div class="label">踏板显示</div>
                            <div class="desc">主窗口是否显示踏板状态浮层。</div>
                        </div>
                        <n-switch v-model:value="store.config.showPedal" @update:value="changeConfig"/>
                    </div>

                </div>
            </section>

            <section v-else-if="store.controlMenu === 'appearance'" class="content-card">
                <div class="section-title">外观主题</div>
                <div class="color-grid">
                    <div class="picker" v-for="(item, key) in store.config.colors" :key="key" @click="changeColorIndex(key)">
                        <div class="color-preview" :style="{background: item.color}"></div>
                        <div>
                            <div class="label">{{ item.label }}</div>
                            <div class="desc mono">{{ item.color }}</div>
                        </div>
                    </div>
                </div>
                <div class="example-card">
                    <ExampleKeyboard/>
                    <ExamplePedal class="example-pedal"/>
                </div>
            </section>

            <section v-else-if="store.controlMenu === 'soundfont'" class="content-card">
                <div class="section-title">音源管理</div>
                <div class="setting-grid">
                    <div class="setting-row vertical">
                        <div>
                            <div class="label">当前音源</div>
                            <div class="desc">SoundFont 文件由后端选择、验证和加载。</div>
                        </div>
                        <div class="soundfont-info">
                            <n-tag :bordered="false" :type="store.config.activeSoundFontId ? 'success' : 'info'">
                                {{ activeSoundFontName }}
                            </n-tag>
                            <div class="mono path-text">{{ activeSoundFontPath }}</div>
                            <div v-if="soundFontError" class="error-text">{{ soundFontError }}</div>
                        </div>
                    </div>

                    <div class="soundfont-upload-card">
                        <div>
                            <div class="label">导入用户音源</div>
                            <div class="desc">添加本地 .sf2 文件后会立即切换为当前音源。</div>
                        </div>
                        <n-button type="primary" size="small" @click="addSoundFont">选择 .sf2 文件</n-button>
                    </div>

                    <div class="soundfont-list">
                        <div class="soundfont-item" :class="{active: !store.config.activeSoundFontId}">
                            <div>
                                <div class="label">默认音源</div>
                                <div class="desc mono">assets/Yamaha-Grand-Lite-v2.0.sf2</div>
                            </div>
                            <n-button size="small" :disabled="!store.config.activeSoundFontId" @click="selectSoundFont('')">
                                使用
                            </n-button>
                        </div>

                        <div
                            v-for="sf in soundFonts"
                            :key="sf.id"
                            class="soundfont-item"
                            :class="{active: store.config.activeSoundFontId === sf.id, missing: sf.missing || sf.error}"
                        >
                            <div class="soundfont-item-main">
                                <div class="label">
                                    {{ sf.name }}
                                    <n-tag v-if="sf.missing" size="small" type="error" :bordered="false">缺失</n-tag>
                                    <n-tag v-else-if="sf.error" size="small" type="warning" :bordered="false">异常</n-tag>
                                </div>
                                <div class="desc mono path-text">{{ sf.path }}</div>
                                <div v-if="sf.error" class="error-text">{{ sf.error }}</div>
                                <div class="desc">{{ formatSize(sf.size) }}</div>
                            </div>
                            <div class="soundfont-actions">
                                <n-button size="small" :disabled="store.config.activeSoundFontId === sf.id" @click="selectSoundFont(sf.id)">
                                    使用
                                </n-button>
                                <n-button size="small" type="error" ghost @click="removeSoundFont(sf.id)">
                                    删除
                                </n-button>
                            </div>
                        </div>
                    </div>

                    <div class="setting-actions">
                        <n-button size="small" @click="refreshSoundFonts">刷新音源状态</n-button>
                    </div>

                    <div class="tip-box">
                        MIDI 播放 / 练习功能已移除；这里仅管理本地演奏使用的 SoundFont 音源。
                    </div>
                </div>
            </section>


            <section v-else-if="store.controlMenu === 'about'" class="content-card about-card">
                <Author/>
            </section>
        </main>

        <n-drawer v-model:show="showColorPicker" placement="right" width="225px" class="color-picker" show-mask="transparent">
            <div class="color-picker-body">
                <Chrome
                    v-if="colorIndex && store.config.colors[colorIndex]"
                    :model-value="store.config.colors[colorIndex].color"
                    :disable-alpha="true"
                    @update:model-value="updateKeyColor"
                />
            </div>
        </n-drawer>
    </div>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, ref, watch} from 'vue'
import {Chrome} from '@ckpack/vue-color'
import {
    NButton,
    NDrawer,
    NInputNumber,
    NTag,
    NRadioButton,
    NRadioGroup,
    NSelect,
    NSlider,
    NSwitch,
} from 'naive-ui'
import ExampleKeyboard from './ExampleKeyboard.vue'
import ExamplePedal from './ExamplePedal.vue'
import Author from './Author.vue'

const store = inject('store')
const changeConfig = inject('changeConfig')
const changeKeyboardType = inject('changeKeyboardType')
const changeDevice = inject('changeDevice')
const resetConfig = inject('resetConfig')
const setKeyColor = inject('setKeyColor')
const Keyboard = inject('Keyboard')

const colorIndex = ref('')
const showColorPicker = ref(false)
const soundFontError = ref('')


const menus = [
    {key: 'basic', label: '基础设置', icon: '⌘'},
    {key: 'devices', label: 'MIDI 设备', icon: '🎛'},
    {key: 'keyboard', label: '键盘显示', icon: '🎹'},
    {key: 'appearance', label: '外观主题', icon: '🎨'},
    {key: 'soundfont', label: '音源管理', icon: '🎧'},
    {key: 'about', label: '关于软件', icon: 'i'},
]

const inDeviceOptions = computed(() => buildDeviceOptions(store.devices.inMidiPool))
const outDeviceOptions = computed(() => buildDeviceOptions(store.devices.outMidiPool))
const soundFonts = computed(() => store.config.soundFonts || [])
const activeSoundFont = computed(() => soundFonts.value.find((item) => item.id === store.config.activeSoundFontId))
const activeSoundFontName = computed(() => activeSoundFont.value?.name || '默认音源')
const activeSoundFontPath = computed(() => activeSoundFont.value?.path || 'assets/Yamaha-Grand-Lite-v2.0.sf2')

function buildDeviceOptions(pool = {}) {
    return Object.entries(pool).map(([key, device]) => ({
        label: device.name,
        value: Number(device.value ?? key),
    }))
}

function changeColorIndex(index) {
    colorIndex.value = index.replaceAll(' ', '')
    showColorPicker.value = true
}

function updateKeyColor(color) {
    store.config.colors[colorIndex.value].color = color.hex
    setKeyColor()
}






async function syncConfigFromBackend() {
    store.config = {...store.config, ...await Keyboard.SendConfig()}
}

async function addSoundFont() {
    try {
        soundFontError.value = ''
        await Keyboard.OpenSoundFontDialog()
        await syncConfigFromBackend()
    } catch (error) {
        soundFontError.value = String(error?.message || error)
    }
}

async function selectSoundFont(id) {
    try {
        soundFontError.value = ''
        await Keyboard.SelectSoundFontByID(id)
        await syncConfigFromBackend()
    } catch (error) {
        soundFontError.value = String(error?.message || error)
        await syncConfigFromBackend()
    }
}

async function removeSoundFont(id) {
    try {
        soundFontError.value = ''
        await Keyboard.RemoveSoundFontByID(id)
        await syncConfigFromBackend()
    } catch (error) {
        soundFontError.value = String(error?.message || error)
        await syncConfigFromBackend()
    }
}

async function refreshSoundFonts() {
    try {
        soundFontError.value = ''
        await Keyboard.RefreshSoundFonts()
        await syncConfigFromBackend()
    } catch (error) {
        soundFontError.value = String(error?.message || error)
    }
}

function formatSize(size) {
    const bytes = Number(size || 0)
    if (bytes <= 0) return ''
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

watch(showColorPicker, () => {
    if (!showColorPicker.value) {
        colorIndex.value = ''
        changeConfig()
    }
})

onBeforeUnmount(() => {
    changeConfig()
})
</script>

<style lang="scss" scoped>
.settings-shell {
    display: grid;
    grid-template-columns: 210px 1fr;
    gap: 18px;
    padding: 0 28px 28px;
    box-sizing: border-box;
}

.setting-menu {
    display: flex;
    flex-direction: column;
    gap: 8px;
    background: rgba(255, 255, 255, 0.66);
    border: 1px solid rgba(148, 163, 184, 0.22);
    border-radius: 22px;
    padding: 12px;
    box-shadow: 0 18px 50px rgba(15, 23, 42, 0.08);
    backdrop-filter: blur(18px);
}

.menu-item {
    display: flex;
    align-items: center;
    gap: 10px;
    border: 0;
    border-radius: 14px;
    padding: 11px 12px;
    text-align: left;
    cursor: pointer;
    background: transparent;
    color: #475569;
    transition: 180ms;
}

.menu-item:hover,
.menu-item.active {
    color: #0f172a;
    background: rgba(37, 99, 235, 0.09);
}

.menu-icon {
    width: 20px;
    text-align: center;
}

.setting-content {
    min-height: 520px;
}

.content-card {
    min-height: 520px;
    background: rgba(255, 255, 255, 0.76);
    border: 1px solid rgba(148, 163, 184, 0.22);
    border-radius: 24px;
    padding: 24px;
    box-shadow: 0 18px 50px rgba(15, 23, 42, 0.08);
    backdrop-filter: blur(18px);
    box-sizing: border-box;
}

.section-title {
    margin-bottom: 22px;
    font-size: 18px;
    font-weight: 700;
}

.setting-grid {
    display: flex;
    flex-direction: column;
    gap: 18px;
}

.setting-row {
    display: grid;
    grid-template-columns: 230px 1fr;
    align-items: center;
    gap: 22px;
}

.setting-row.vertical {
    grid-template-columns: 1fr;
    align-items: stretch;
    gap: 10px;
}

.label {
    font-size: 14px;
    font-weight: 650;
    color: #1e293b;
}

.desc {
    margin-top: 4px;
    font-size: 12px;
    line-height: 1.6;
    color: #64748b;
}

.mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}


.soundfont-info {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.soundfont-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.soundfont-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
    padding: 14px;
    border-radius: 16px;
    background: rgba(248, 250, 252, 0.9);
    border: 1px solid rgba(148, 163, 184, 0.22);
}

.soundfont-item.active {
    border-color: rgba(34, 197, 94, 0.42);
    background: rgba(240, 253, 244, 0.72);
}

.soundfont-item.missing {
    border-color: rgba(239, 68, 68, 0.28);
}

.soundfont-item-main {
    min-width: 0;
}

.soundfont-actions {
    display: flex;
    flex-shrink: 0;
    gap: 8px;
}

.soundfont-upload-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
    padding: 18px;
    border-radius: 18px;
    background: rgba(248, 250, 252, 0.9);
    border: 1px dashed rgba(37, 99, 235, 0.28);
}

.hidden-input {
    display: none;
}

.path-text {
    word-break: break-all;
    color: #475569;
}

.error-text {
    color: #b91c1c;
    font-size: 12px;
    line-height: 1.6;
}

.setting-actions {
    margin-top: 8px;
}

.tip-box,
.placeholder-card {
    border-radius: 16px;
    padding: 16px;
    background: rgba(37, 99, 235, 0.07);
    color: #475569;
    font-size: 13px;
    line-height: 1.7;
}

.placeholder-title {
    margin-bottom: 6px;
    font-size: 15px;
    font-weight: 700;
    color: #1e293b;
}

.color-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(180px, 1fr));
    gap: 12px;
    margin-bottom: 18px;
}

.picker {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px;
    border-radius: 18px;
    background: rgba(248, 250, 252, 0.9);
    border: 1px solid rgba(148, 163, 184, 0.18);
    cursor: pointer;
    transition: 180ms;
}

.picker:hover {
    transform: translateY(-1px);
    box-shadow: 0 12px 30px rgba(15, 23, 42, 0.08);
}

.color-preview {
    width: 38px;
    height: 38px;
    border-radius: 12px;
    box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.08);
}

.example-card {
    position: relative;
    height: 170px;
    overflow: hidden;
    border-radius: 18px;
    background: rgba(15, 23, 42, 0.05);
    padding: 18px;
}

.example-pedal {
    position: absolute;
    right: 18px;
    bottom: 18px;
    z-index: 99;
}

.color-picker-body {
    height: 100%;
    overflow: hidden;
}

.about-card {
    display: flex;
    align-items: center;
    justify-content: center;
}

:deep(.vc-chrome) {
    box-sizing: border-box !important;
    border-radius: 8px !important;
    height: 100% !important;
}
</style>
