import {computed, onBeforeUnmount, provide, reactive, ref, watch, watchEffect} from 'vue'
import {changeColor, composite, rgba} from 'seemly'
import {data} from '../../../frontend/src/store'
import {colorPalettes} from '../../../frontend/src/services/themes'
import {getKeyLabelColor, getPianoUiTheme} from '../../../frontend/src/services/uiTheme'
import {acceptLivePerformance} from '../../../frontend/src/services/livePerformance'
import {setKeyboardMappingLabels} from '../../../frontend/src/services/computerKeyboard'
import defaultKeymap from '../services/defaultKeymap.json'
import {chooseFile, downloadFile} from '../services/files'
import {encodeRecording, parseMIDI} from '../services/midi'
import {saveSong, getSongs, removeSong, saveDraft, getDraft, clearDraft} from '../services/storage'
import {loadToyPreferences, saveToyPreferences} from '../services/toy'

// Browser storage and audio drive the same presentation components as Wails.
export function usePianoPresentation(piano, community, notify) {
  const store = data()
  const colors = Object.fromEntries(Object.entries(colorPalettes.find(p => p.key === 'fresh').colors)
    .map(([key, color]) => [key, {color}]))
  Object.assign(store.config, {colors, keyLabel: 'octave_key', keyboardType: 0, showPedal: true,
    volume: 80, velocity: 80, keymapProfiles: [{id: 'default', name: '默认方案', mapping: {...defaultKeymap}}]})
  const uiTheme = computed(() => getPianoUiTheme(store.config.colors))
  const importing = ref(false), loading = ref(false), previewPosition = ref(null), clockRevision = ref(0)
  const sustainActive = ref(false), midiDeviceStatus = ref('未连接'), soundFontName = ref('轻量钢琴')
  const recordingName = ref(''), recordingBusy = ref(false)
  let prefsTimer, checkpointTimer, draftQueue = Promise.resolve(), ready = false

  function saveConfig() {
    clearTimeout(prefsTimer)
    prefsTimer = setTimeout(() => saveToyPreferences({config: JSON.parse(JSON.stringify(store.config))}), 300)
  }
  function setKeyColor() {
    for (const [key, {color}] of Object.entries(store.config.colors)) {
      const style = document.documentElement.style, [r, g, b] = rgba(color)
      style.setProperty('--' + key, color)
      style.setProperty('--' + key + '-o', `rgba(${Math.round(r * .1)}, ${Math.round(g * .1)}, ${Math.round(b * .1)}, .4)`)
      if (key.includes('Key')) {
        style.setProperty('--' + key + '-label', getKeyLabelColor(color))
        style.setProperty('--' + key + '-sustained-label', getKeyLabelColor(composite(key.startsWith('black') ? '#333333' : '#f6f6f6', changeColor(color, {alpha: .38}))))
      }
    }
  }
  watch(() => store.config.volume, value => piano.setVolume(value), {immediate: true})
  watch(() => store.config.velocity, value => piano.setVelocity(value), {immediate: true})
  watch(() => store.config.colors, setKeyColor, {deep: true, immediate: true})
  watch(() => [store.activeKeyMapping, store.keyboardLoaded], ([mapping]) => {
    setKeyboardMappingLabels(store.keyboardConfig, mapping)
  }, {deep: true})

  const session = reactive({
    get library() { return piano.library }, get selectedId() { return piano.selected?.id || '' },
    get selectedSong() { return piano.selected }, get timeline() { return piano.selected },
    get durationMs() { return piano.duration }, get mode() { return piano.mode },
    get loading() { return loading.value }, get importing() { return importing.value }, get busy() { return importing.value || loading.value },
    get options() { return {speed: piano.speed, handMode: piano.handMode, loop: piano.loop, leftMs: piano.range.a, rightMs: piano.range.b} },
    get state() { return {...piano.state, mode: piano.mode, handMode: piano.handMode, speed: piano.speed,
      currentStep: piano.state.step ? {...piano.state.step, notes: piano.state.step.notes.filter(n => !piano.state.step.accepted?.has(n.note))} : null,
      currentMs: previewPosition.value ?? piano.state.currentMs, scrubbing: previewPosition.value !== null,
      leftMs: piano.range.a, rightMs: piano.range.b, receivedAt: performance.now(), clockRevision: clockRevision.value} },
    togglePlayback: () => piano.togglePlayback(), stop: () => piano.stop(),
    setSpeed: value => { clockRevision.value++; piano.setSpeed(value) },
    setHand: value => { clockRevision.value++; piano.setHand(value) },
    setLoop: value => { piano.loop = value }, setRange: (a, b) => piano.setRange(a, b),
    previewSeek: value => { previewPosition.value = Math.max(piano.range.a, Math.min(piano.range.b, value)); return previewPosition.value },
    endSeekPreview: () => { previewPosition.value = null; clockRevision.value++ },
    seek: value => { piano.seek(value); clockRevision.value++; return true },
    async selectSong(id) {
      const song = piano.library.find(item => item.id === id)
      if (!song) return false
      if (community.inRoom) community.leave()
      piano.selectSong(song)
      return true
    },
    async importSong() {
      const file = await chooseFile('.mid,.midi,audio/midi')
      if (!file) return false
      importing.value = true
      if (community.inRoom) community.leave()
      const imported = await piano.importMIDI(file)
      if (imported) await saveSong(piano.selected).catch(error => notify('曲目已打开，但保存曲库失败：' + error.message))
      importing.value = false
      return imported
    },
    async removeSong(id) {
      await removeSong(id)
      if (piano.selected?.id === id) { piano.stop(); piano.selected = null; piano.range.a = piano.range.b = 0 }
      piano.library = piano.library.filter(song => song.id !== id)
    },
  })
  watchEffect(() => {
    store.mainMode = piano.mode
    store.mainDisplayMode = piano.displayMode
    store.midiPlayerState = session.state
    store.midiPlaybackKey = Object.fromEntries(Object.entries(piano.playbackHands).filter(([, hand]) => hand !== 'left').map(([note]) => [note, true]))
    store.midiPlaybackLeftKey = Object.fromEntries(Object.entries(piano.playbackHands).filter(([, hand]) => hand === 'left').map(([note]) => [note, true]))
    const hints = piano.mode === 'follow' && !session.state.scrubbing && piano.state.status === 'playing' ? session.state.currentStep?.notes || [] : []
    store.midiHintKey = Object.fromEntries(hints.filter(n => n.hand !== 'left').map(n => [n.note, true]))
    store.midiHintLeftKey = Object.fromEntries(hints.filter(n => n.hand === 'left').map(n => [n.note, true]))
  })

  let sequence = 0
  const voices = new Map()
  function liveSignal(type, voice, sustained = false) {
    acceptLivePerformance({type, ...voice, sequence: ++sequence, timestampMs: performance.now(), sustained})
  }
  const offPiano = piano.subscribe(event => {
    if (event.source !== 'user' && !event.remote) return
    const key = `${event.source}:${event.note}`
    if (event.type === 'on') {
      const voice = {id: `${key}:${sequence + 1}`, source: event.source, channel: 0, note: event.note, velocity: event.velocity, pressed: true}
      if (voices.has(key)) liveSignal('noteEnd', voices.get(key))
      voices.set(key, voice)
      liveSignal('noteOn', voice)
      store.backendPressedKey[event.note] = true
      store.setPressedHand(event.note, event.hand)
      store.activeKey[event.note] = true
    } else {
      const voice = voices.get(key)
      if (voice) {
        voice.pressed = false
        liveSignal('noteOff', voice, sustainActive.value)
        if (!sustainActive.value) voices.delete(key)
      }
      store.backendPressedKey[event.note] = false
      if (!store.pressedKey[event.note]) delete store.pressedKeyHands[event.note]
      if (!sustainActive.value) store.activeKey[event.note] = false
    }
  })
  function toggleSustain(on = !sustainActive.value) {
    if (community.inRoom && !community.isPerformer) return
    sustainActive.value = !!on
    store.devices.softwareSustain = !!on
    piano.toggleSustain(on)
    if (community.isPerformer) community.send({type: 'pedal', on})
    if (!on) for (const [key, voice] of voices) if (!voice.pressed) {
      liveSignal('noteEnd', voice)
      store.activeKey[voice.note] = false
      voices.delete(key)
    }
  }
  provide('store', store)
  provide('changeConfig', saveConfig)
  provide('changeKeyboardType', saveConfig)
  provide('setKeyColor', setKeyColor)
  provide('toggleSustain', toggleSustain)
  provide('applyConfig', config => { Object.assign(store.config, config); saveConfig() })
  provide('setInteractiveKey', (note, pressed) => {
    if (community.inRoom && !community.isPerformer) return
    const previous = store.interactiveKeys[note] || 0
    store.setKeyState(note, pressed)
    if (pressed && !previous) piano.userNoteOn(note, store.config.velocity, 'pointer')
    else if (!pressed && !store.interactiveKeys[note]) piano.userNoteOff(note, 'pointer')
  })
  provide('subscribeKeyboardFeedback', listener => piano.subscribe(event => {
    if (event.type === 'on') listener({type: 'attack', note: event.note, hand: event.hand})
  }))

  async function persistDraft() {
    if (!['recording', 'draft'].includes(piano.recording.status)) return
    const snapshot = JSON.parse(JSON.stringify({...piano.recording, backupSaved: true}))
    const task = draftQueue.then(() => saveDraft(snapshot))
    draftQueue = task.catch(() => {})
    await task.then(() => {
      if (piano.recording.events.length === snapshot.events.length) piano.recording.backupSaved = true
    }, error => { piano.recording.backupSaved = false; notify('草稿备份失败：' + error.message) })
  }
  const recording = reactive({
    get state() { return piano.recording }, get durationMs() { return piano.recording.durationMs },
    get busy() { return recordingBusy.value }, get available() { return !community.inRoom },
    get previewing() { return piano.recordingPreviewing },
    get name() { return recordingName.value }, set name(value) { recordingName.value = value },
    start() { piano.startRecording(); void persistDraft() },
    async stop() { piano.stopRecording(); await persistDraft() },
    preview: () => piano.previewRecording(), stopPreview: () => piano.stopRecordingPreview(),
    async save() {
      const blob = encodeRecording(piano.recording.events)
      const name = recordingName.value.trim() || `我的录制 ${new Date().toLocaleString('zh-CN')}`
      const song = parseMIDI(await blob.arrayBuffer(), name)
      song.id = 'recording-' + Date.now()
      downloadFile(blob, name.replace(/[/:]/g, '-') + '.mid')
      await saveSong(song)
      piano.library = [...piano.library, song]
      await persistDraft()
      notify('已请求下载 MIDI，草稿保留以便再次保存。')
    },
    async discard() { await draftQueue; await clearDraft(); piano.stopRecordingPreview(); piano.discardRecording() },
  })

  function profile(id) { return store.config.keymapProfiles.find(item => item.id === id) }
  provide('Keyboard', {
    AddKeymapProfile() {
      const item = {id: crypto.randomUUID(), name: '新方案', mapping: {}}
      store.config.keymapProfiles.push(item); store.config.activeKeymapProfileId = item.id
      saveConfig()
    },
    SelectKeymapProfile(id) { store.config.activeKeymapProfileId = id; saveConfig() },
    RenameKeymapProfile(id, name) { profile(id).name = name; saveConfig() },
    DeleteKeymapProfile(id) {
      store.config.keymapProfiles = store.config.keymapProfiles.filter(item => item.id !== id)
      store.config.activeKeymapProfileId = 'default'; saveConfig()
    },
    BindKeymapKey(id, key, note) { profile(id).mapping[key] = note; saveConfig() },
    RemoveKeymapBinding(id, key) { delete profile(id).mapping[key]; saveConfig() },
    async ImportKeymapProfile() {
      const file = await chooseFile('.json')
      if (!file) return
      const value = JSON.parse(await file.text())
      if (value.format !== 'peirato-keymap-profile' || value.version !== 1 || typeof value.name !== 'string' || !value.mapping ||
        Object.entries(value.mapping).some(([key, note]) => !key || !Number.isInteger(note) || note < 0 || note > 127)) throw Error('按键方案格式无效')
      const item = {id: crypto.randomUUID(), name: value.name.slice(0, 80), mapping: value.mapping}
      store.config.keymapProfiles.push(item); store.config.activeKeymapProfileId = item.id; saveConfig()
    },
    ExportActiveKeymapProfile() {
      const item = store.activeKeymapProfile
      downloadFile(new Blob([JSON.stringify({format: 'peirato-keymap-profile', version: 1, name: item.name, mapping: item.mapping}, null, 2)], {type: 'application/json'}), item.name + '.json')
    },
  })
  async function importSF2() {
    const file = await chooseFile('.sf2')
    if (file && await piano.importSF2(file)) soundFontName.value = file.name
  }
  async function initialize() {
    const [keyboard, chordNames, preferences] = await Promise.all([
      fetch(import.meta.env.BASE_URL + 'keyboard_config.json').then(r => r.json()),
      fetch(import.meta.env.BASE_URL + 'ChordNames.json').then(r => r.json()), loadToyPreferences(),
    ])
    const saved = preferences.config
    if (saved) Object.assign(store.config, saved)
    store.keyboardConfig = keyboard.map(key => ({...key, octave_key: key.note === 'C' ? key.note + key.octave : '', pitch: key.pitch || null, keyboard: ''}))
    store.chordsName = chordNames
    store.keyboardLoaded = store.loaded = true
    await community.restoreSession()
    const songs = await getSongs()
    piano.library = [...piano.library, ...songs.filter(song => song.notes?.length && !piano.library.some(item => item.id === song.id) && (!song.ownerAccountId || song.ownerAccountId === community.user?.id))]
    const draft = await getDraft()
    if (draft?.events?.length) { Object.assign(piano.recording, draft, {status: 'draft', backupSaved: true}); notify('已恢复上次录制草稿') }
    ready = true
    let checkpoint = 0
    checkpointTimer = setInterval(() => {
      if (piano.recording.status === 'recording') {
        piano.recording.durationMs = performance.now() - piano.recording.startedAt
        if (++checkpoint % 15 === 0) void persistDraft()
      }
    }, 1000)
  }
  onBeforeUnmount(() => {
    clearTimeout(prefsTimer); clearInterval(checkpointTimer); offPiano()
    if (ready) saveToyPreferences({config: JSON.parse(JSON.stringify(store.config))})
    if (piano.recording.status === 'recording') { piano.stopRecording(); void persistDraft() }
  })
  return {store, session, recording, uiTheme, saveConfig, initialize, sustainActive, toggleSustain, midiDeviceStatus, soundFontName, importSF2, persistDraft}
}
