import {computed, ref} from 'vue'
import {readFileAsBase64} from '../services/backendMidiService'

const STORAGE_KEY = 'peirato-piano-midi-library-v1'

function createId(path) {
    return String(path || '').trim().toLowerCase()
}

function normalizeMidiPath(file) {
    // Wails WebView 在开启文件能力后通常会给 File 对象补充 path。
    // 浏览器安全模型不会提供绝对路径，所以这里必须防御：没有绝对路径时不写入 MIDI 目录。
    const path = file?.path || file?.fullPath || file?.webkitRelativePath || ''
    return String(path || '').trim()
}

function isLikelyAbsolutePath(path) {
    return /^([a-zA-Z]:\\|\\\\|\/)/.test(path || '')
}

function loadStoredItems() {
    try {
        const raw = localStorage.getItem(STORAGE_KEY)
        const list = raw ? JSON.parse(raw) : []
        return Array.isArray(list) ? list.filter((item) => item?.path) : []
    } catch {
        return []
    }
}

function saveStoredItems(items) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
}

const midiItems = ref(loadStoredItems())
const selectedMidiId = ref(midiItems.value[0]?.id || '')

export function useMidiLibrary(Keyboard, store) {
    const selectedMidi = computed(() => midiItems.value.find((item) => item.id === selectedMidiId.value) || null)

    function persist() {
        midiItems.value = [...midiItems.value]
        saveStoredItems(midiItems.value)
    }

    async function importMidiFile(file) {
        const absolutePath = normalizeMidiPath(file)
        if (!isLikelyAbsolutePath(absolutePath)) {
            throw new Error('当前系统没有返回 MIDI 绝对路径，请在 Wails 窗口中选择文件，或改用拖拽/系统文件选择能力。')
        }

        const id = createId(absolutePath)
        const existed = midiItems.value.find((item) => item.id === id)
        if (!existed) {
            midiItems.value.unshift({
                id,
                name: file.name || absolutePath.split(/[\\/]/).pop(),
                path: absolutePath,
                createdAt: Date.now(),
            })
            persist()
        }

        selectedMidiId.value = id
        const parsed = await loadMidiFromFile(file, absolutePath)
        return {status: existed ? 'exists' : 'created', parsed}
    }

    async function loadMidiFromFile(file, absolutePath = '') {
        const encoded = await readFileAsBase64(file)
        const parsed = await Keyboard.LoadMidiFileBase64(file.name, encoded)
        store.player.filePath = absolutePath || normalizeMidiPath(file)
        return parsed
    }

    function selectMidi(id) {
        selectedMidiId.value = id
    }

    function removeMidi(id) {
        midiItems.value = midiItems.value.filter((item) => item.id !== id)
        if (selectedMidiId.value === id) {
            selectedMidiId.value = midiItems.value[0]?.id || ''
        }
        persist()
    }

    return {
        midiItems,
        selectedMidiId,
        selectedMidi,
        importMidiFile,
        loadMidiFromFile,
        selectMidi,
        removeMidi,
        isLikelyAbsolutePath,
    }
}
