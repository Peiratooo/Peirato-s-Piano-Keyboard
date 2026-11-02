<template>
    <main class="library-window" :style="theme.cssVariables">
        <header class="library-toolbar">
            <NButton quaternary @click="returnToPiano">返回钢琴</NButton>
            <nav aria-label="曲库导航">
                <NButton :type="view === 'library' ? 'primary' : 'default'" @click="view = 'library'">本地曲库</NButton>
                <NButton :type="view === 'practice' ? 'primary' : 'default'" @click="view = 'practice'">练习记录</NButton>
            </nav>
            <NButton quaternary @click="Keyboard.OpenControlCenter()">设置</NButton>
        </header>
        <div class="library-content">
            <LocalLibraryPanel v-if="view === 'library'" :session="session" @play="showPractice" />
            <PracticePanel v-else :session="session" @choose="view = 'library'" @play="returnToPiano" />
        </div>
        <WorkspacePlayer :session="session" @piano="returnToPiano" @toggle="togglePlayback" />
    </main>
</template>

<script setup>
import {computed, inject, onBeforeUnmount, onMounted, ref} from 'vue'
import {useRoute} from 'vue-router'
import {NButton} from 'naive-ui'
import {Events} from '@wailsio/runtime'
import LocalLibraryPanel from '../components/library/LocalLibraryPanel.vue'
import PracticePanel from '../components/library/PracticePanel.vue'
import WorkspacePlayer from '../components/library/WorkspacePlayer.vue'
import {useMidiSession} from '../services/midiSession'
import {getPianoUiTheme} from '../services/uiTheme'

const store = inject('store')
const Keyboard = inject('Keyboard')
const view = ref(useRoute().query.view === 'practice' ? 'practice' : 'library')
const theme = computed(() => getPianoUiTheme(store.config.colors))
const session = useMidiSession({store, Keyboard})

async function togglePlayback() {
    if (session.mode !== 'play' && !await session.setMode('play')) return
    await session.togglePlayback()
}

async function returnToPiano() {
    await Keyboard.ReturnToPiano()
}

function showPractice(mode) {
    if (mode === 'follow') void returnToPiano()
}

const offNavigate = Events.On('libraryNavigate', ({data}) => {
    view.value = data
    void session.refreshLibrary()
})
onMounted(() => session.initialize())
onBeforeUnmount(() => {
    offNavigate()
    session.dispose()
})
</script>

<style scoped>
.library-window { position: fixed; inset: 0; display: grid; grid-template-rows: auto minmax(0, 1fr) auto; background: #2b2b2b; color: #ededed; }
.library-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 14px 24px; border-bottom: 1px solid #ffffff12; }
.library-toolbar nav { display: flex; gap: 8px; }
.library-content { overflow: auto; padding: 24px; }
</style>
