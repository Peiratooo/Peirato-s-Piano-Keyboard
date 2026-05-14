import {defineStore} from 'pinia'
import {createInitialPlayerState} from '../services/midiPlaybackController'

const defaultConfig = {
    colors: {
        whiteKey: {label: '白键按下', color: '#9AF7B3'},
        blackKey: {label: '黑键按下', color: '#5FFF5F'},
        damperPedal: {label: '延音踏板踩下', color: '#e7b510'},
        softPedal: {label: '柔音踏板踩下', color: '#10e786'},
        sostenutoPedal: {label: '消音踏板踩下', color: '#1054e7'},
    },
    keyLabel: 'octave_key',
    keyboardType: 0,
    velocity: 80,
    volume: 80,
    opacity: 100,
    showPedal: true,
    midiChannel: 0,
    soundFontPath: '',
    version: '',
}

export const data = defineStore('data', {
    state: () => {
        return {
            keyboard: [],

            // activeKey：当前正在亮起/正在发声的音。它会受到延音、MIDI 播放、跟弹提示等影响。
            activeKey: {},
            // pressedKey：用户手指真实按住的音。和弦识别优先使用它，避免延音踏板污染和弦判断。
            pressedKey: {},
            // playbackKey / hintKey / wrongKey 为后续 MIDI 播放和跟弹模式预留。
            playbackKey: {},
            hintKey: {},
            wrongKey: {},

            // player 保存 MIDI 独立窗口的播放基础状态。解析和播放调度统一由 Go / Wails 服务负责。
            player: createInitialPlayerState(),
            // soundFontInfo 由后端音源服务维护，设置中心只负责展示和触发 reload / restore。
            soundFontInfo: {
                loaded: false,
                path: '',
                name: '',
                error: '',
            },

            keyMapping: {},
            chordsname: {},
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
            keybordType:[
                {label:'88', value: 0},
                {label:'84', value: 1},
                {label:'76', value: 2},
                {label:'72', value: 3},
                {label:'61', value: 4},
                {label:'37', value: 5},
            ],
            config: defaultConfig,
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
            this.playbackKey = {}
            this.hintKey = {}
            this.wrongKey = {}
        },
        resetPlayer() {
            this.player = createInitialPlayerState()
            this.playbackKey = {}
            this.hintKey = {}
            this.wrongKey = {}
        },
    },
})
