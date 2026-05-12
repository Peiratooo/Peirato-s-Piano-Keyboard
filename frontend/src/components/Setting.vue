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
                            <div class="desc">控制本地音源播放音量。</div>
                        </div>
                        <n-slider v-model:value="store.config.volume" :min="0" :max="127" @dragend="changeConfig"/>
                    </div>
                    <div class="setting-row">
                        <div>
                            <div class="label">默认力度</div>
                            <div class="desc">电脑键盘或鼠标触发时使用的 MIDI velocity。</div>
                        </div>
                        <n-slider v-model:value="store.config.velocity" :min="0" :max="127" @dragend="changeConfig"/>
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
                                v-for="item in store.keybordType"
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
                    <div class="example-card">
                        <ExampleKeyboard/>
                        <ExamplePedal class="example-pedal"/>
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
                <div class="tip-box">目前先保留你原来的按键配色方案，后续可以在这里加“主题预设”。</div>
            </section>

            <section v-else-if="store.controlMenu === 'soundfont'" class="content-card">
                <div class="section-title">音源管理</div>
                <div class="setting-grid">
                    <div class="setting-row vertical">
                        <div>
                            <div class="label">当前音源</div>
                            <div class="desc">前端选择 .sf2 文件并上传给 Go，Go 负责保存、加载和写入配置。</div>
                        </div>
                        <div class="soundfont-info">
                            <n-tag :bordered="false" :type="store.soundFontInfo.loaded ? 'success' : 'warning'">
                                {{ store.soundFontInfo.loaded ? '已加载' : '未加载' }}
                            </n-tag>
                            <div class="mono path-text">{{ store.soundFontInfo.path || '默认音源 / 暂无路径' }}</div>
                            <div v-if="store.soundFontInfo.error" class="error-text">{{ store.soundFontInfo.error }}</div>
                        </div>
                    </div>

                    <div class="soundfont-upload-card">
                        <input
                            ref="soundFontInput"
                            class="hidden-input"
                            type="file"
                            accept=".sf2,audio/x-soundfont"
                            @change="handleSoundFontFileChange"
                        />
                        <div>
                            <div class="label">导入用户音源</div>
                            <div class="desc">选择本地 .sf2 文件后会上传到后端保存，下次启动会自动加载该音源。</div>
                        </div>
                        <n-button type="primary" size="small" @click="chooseSoundFontFile">选择 .sf2 文件</n-button>
                    </div>

                    <div class="setting-actions">
                        <n-button type="primary" size="small" @click="reloadSoundFont">重新加载音源</n-button>
                        <n-button size="small" @click="restoreDefaultSoundFont">恢复默认音源</n-button>
                    </div>

                    <div class="tip-box">
                        当前不使用后端原生文件选择器：文件由前端选择，然后以 base64 传给 Go。这样后续 MIDI 文件和音源文件的导入方式保持一致。
                    </div>
                </div>
            </section>

            <MidiPracticePanel v-else-if="store.controlMenu === 'practice'" />

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
    NInput,
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
import MidiPracticePanel from './settings/MidiPracticePanel.vue'
import {readFileAsBase64} from '../services/backendMidiService'

const store = inject('store')
const changeConfig = inject('changeConfig')
const changeKeyboardType = inject('changeKeyboardType')
const changeDevice = inject('changeDevice')
const resetConfig = inject('resetConfig')
const setKeyColor = inject('setKeyColor')
const Keyboard = inject('Keyboard')

const colorIndex = ref('')
const showColorPicker = ref(false)
const soundFontInput = ref(null)

const menus = [
    {key: 'basic', label: '基础设置', icon: '⌘'},
    {key: 'devices', label: 'MIDI 设备', icon: '🎛'},
    {key: 'keyboard', label: '键盘显示', icon: '🎹'},
    {key: 'appearance', label: '外观主题', icon: '🎨'},
    {key: 'soundfont', label: '音源管理', icon: '🎧'},
    {key: 'practice', label: 'MIDI 练习', icon: '✨'},
    {key: 'about', label: '关于软件', icon: 'i'},
]

const inDeviceOptions = computed(() => buildDeviceOptions(store.devices.inMidiPool))
const outDeviceOptions = computed(() => buildDeviceOptions(store.devices.outMidiPool))

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

function chooseSoundFontFile() {
    soundFontInput.value?.click()
}

async function handleSoundFontFileChange(event) {
    const file = event.target.files?.[0]
    if (!file) return

    try {
        const encoded = await readFileAsBase64(file)
        store.soundFontInfo = await Keyboard.ImportSoundFontBase64(file.name, encoded)
    } catch (error) {
        store.soundFontInfo = {
            ...store.soundFontInfo,
            loaded: false,
            error: String(error?.message || error),
        }
    } finally {
        event.target.value = ''
    }
}

async function reloadSoundFont() {
    changeConfig()
    try {
        store.soundFontInfo = await Keyboard.ReloadSoundFont()
    } catch (error) {
        store.soundFontInfo = {
            ...store.soundFontInfo,
            loaded: false,
            error: String(error?.message || error),
        }
    }
}

async function restoreDefaultSoundFont() {
    try {
        store.soundFontInfo = await Keyboard.RestoreDefaultSoundFont()
    } catch (error) {
        store.soundFontInfo = {
            ...store.soundFontInfo,
            loaded: false,
            error: String(error?.message || error),
        }
    }
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
    grid-template-columns: repeat(2, minmax(180px, 1fr));
    gap: 14px;
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
