import {defineStore} from 'pinia'

const safeConfig = {
    colors: {},
    keyLabel: '',
    keyboardType: 0,
    velocity: 80,
    volume: 80,
    opacity: 100,
    showPedal: true,
    midiChannel: 0,
    activeSoundFontId: '',
    soundFonts: [],
    version: '',
}

export const data = defineStore('data', {
    state: () => {
        return {
            keyboard: [],

            // activeKey：当前正在亮起/正在发声的音。它会受到延音踏板影响。
            activeKey: {},
            // pressedKey：用户手指真实按住的音。和弦识别优先使用它，避免延音踏板污染和弦判断。
            pressedKey: {},

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
    getters: {},
    actions: {
        setKeyState(key, pressed) {
            this.activeKey[key] = pressed
            this.pressedKey[key] = pressed
        },
        clearAllKeys() {
            this.activeKey = {}
            this.pressedKey = {}
        },
    },
})
