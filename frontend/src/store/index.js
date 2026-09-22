import {defineStore} from 'pinia'

const safeConfig = {
    colors: {},
    keyLabel: '',
    keyTonic: 0,
    keyboardType: 0,
    velocity: 80,
    volume: 80,
    sampleRate: 44100,
    bufferSize: 2048,
    opacity: 100,
    showPedal: true,
    midiChannel: 0,
    activeSoundFontId: '',
    soundFonts: [],
    activeKeymapProfileId: 'default',
    keymapProfiles: [],
    version: '',
}

export const data = defineStore('data', {
    state: () => {
        return {
            keyboard: [],
            keyboardConfig: [],

            // activeKey：当前正在亮起/正在发声的音。它会受到延音踏板影响。
            activeKey: {},
            // pressedKey：用户手指真实按住的音。和弦识别优先使用它，避免延音踏板污染和弦判断。
            backendPressedKey: {},
            interactiveKeys: {},
            midiPlaybackKey: {},
            midiPlaybackLeftKey: {},
            midiHintKey: {},
            midiHintLeftKey: {},
            midiWindowOpen: false,
            midiPlayerState: {
                status: 'idle',
                mode: 'play',
                handMode: 'both',
                durationMs: 0,
                currentMs: 0,
                leftMs: 0,
                rightMs: 0,
                speed: 1,
                loop: true,
                waiting: false,
            },

            keyMapping: {},
            chordsName: {},
            keyboardLoaded: false,
            loaded: false,
            devices: {
                inMidiPool: {},
                outMidiPool: {},
                selectedInDevice: -1,
                selectedOutDevice: -1,
                pedalStatus: {}
            },
            noteName:{
                0:'A',
                1:'Bb',
                2:'B',
                3:'C',
                4:'Db',
                5:'D',
                6:'Eb',
                7:'E',
                8:'F',
                9:'Gb',
                10:'G',
                11:'Ab'
            },
            labelMap: [
                {label: '八度', value: 'octave_key'},
                {label:'音符名', value: 'note'},
                {label:'数字唱名法', value: 'pitch'},
                {label:'音调唱名法', value: 'tone'},
                {label:'键盘映射', value: 'keyboard'}
            ],
            scale:1,
            keyboardRange:[
                [0, 88],
                [3, 87],
                [6, 83],
                [8, 81],
                [27,88],
                [27,64],
            ],
            keyboardOptions:[
                {label:'88', value: 0},
                {label:'84', value: 1},
                {label:'76', value: 2},
                {label:'72', value: 3},
                {label:'61', value: 4},
                {label:'37', value: 5},
            ],
            config: {...safeConfig},
            controlMenu: 'basic',
            keyboardMenu:true,
            menuBar:true,
            showSetting:true,
            showAuthor:false,
        }
    },
    getters: {
        pressedKey(state) {
            const keys = {...state.backendPressedKey}
            for (const [note, count] of Object.entries(state.interactiveKeys)) {
                if (count > 0) keys[note] = true
            }
            return keys
        },
        activeKeymapProfile(state) {
            const profiles = state.config.keymapProfiles || []
            return profiles.find((profile) => profile.id === state.config.activeKeymapProfileId) || profiles[0] || null
        },
        activeKeyMapping() {
            return this.activeKeymapProfile?.mapping || {}
        },
        keymapProfileOptions(state) {
            return (state.config.keymapProfiles || []).map((profile) => ({
                label: profile.name || '未命名方案',
                value: profile.id,
            }))
        },
    },
    actions: {
        setKeyState(key, pressed) {
            this.interactiveKeys[key] = Math.max(0, (this.interactiveKeys[key] || 0) + (pressed ? 1 : -1))
        },
        setMidiVisualKey(key, hand, active, source = 'playback') {
            const isLeft = hand === 'left'
            const target = source === 'followHint'
                ? (isLeft ? this.midiHintLeftKey : this.midiHintKey)
                : (isLeft ? this.midiPlaybackLeftKey : this.midiPlaybackKey)
            target[key] = active
        },
        clearMidiHints() {
            this.midiHintKey = {}
            this.midiHintLeftKey = {}
        },
        clearMidiVisualKeys() {
            this.midiPlaybackKey = {}
            this.midiPlaybackLeftKey = {}
            this.clearMidiHints()
        },
        clearAllKeys() {
            this.activeKey = {}
            this.backendPressedKey = {}
            this.interactiveKeys = {}
            this.clearMidiVisualKeys()
        },
    },
})
