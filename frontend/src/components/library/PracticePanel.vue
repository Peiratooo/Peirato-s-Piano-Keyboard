<template>
    <section aria-labelledby="practice-heading">
        <header class="history-header">
            <h2 id="practice-heading">本机练习记录</h2>
            <div>
                <NButton size="small" :loading="loading" @click="load">刷新</NButton>
                <NButton size="small" type="primary" @click="$emit('choose')">选择曲目</NButton>
            </div>
        </header>
        <p v-if="error" role="alert">{{ error }}</p>
        <p v-else-if="!loading && !items.length">播放或跟弹后，练习记录会保存在本机。</p>
        <template v-if="items.length">
            <p>{{ items.length }} 首曲目 · {{ Math.floor(totals.seconds / 60) }} 分钟 · {{ totals.sessions }} 次练习</p>
            <div class="history-grid">
                <article v-for="item in items" :key="item.localId">
                    <h3>{{ item.title }}</h3>
                    <p>{{ item.sessions }} 次练习 · {{ Math.floor(item.seconds / 60) }} 分 {{ Math.floor(item.seconds % 60) }} 秒</p>
                    <p>上次练习 {{ new Date(item.lastPracticed).toLocaleString('zh-CN') }}</p>
                    <NButton size="small" :disabled="session.busy" @click="resume(item)">继续练习</NButton>
                </article>
            </div>
        </template>
    </section>
</template>

<script setup>
import {computed, inject, onMounted, ref} from 'vue'
import {NButton} from 'naive-ui'

const props = defineProps({session: {type: Object, required: true}})
const emit = defineEmits(['choose', 'play'])
const Keyboard = inject('Keyboard')
const items = ref([])
const loading = ref(false)
const error = ref('')
const totals = computed(() => items.value.reduce((sum, item) => ({
    seconds: sum.seconds + item.seconds,
    sessions: sum.sessions + item.sessions,
}), {seconds: 0, sessions: 0}))

async function load() {
    loading.value = true
    error.value = ''
    try {
        items.value = await Keyboard.GetLocalPractice()
    } catch (err) {
        error.value = err.message || String(err)
    } finally {
        loading.value = false
    }
}

async function resume(item) {
    error.value = ''
    const {session} = props
    await session.refreshLibrary()
    if (!session.library.some(song => song.id === item.localId && !song.missing)) {
        error.value = '本地文件不可用，请重新导入。'
        return
    }
    if (await session.selectSong(item.localId) && await session.setMode(item.lastMode || 'follow') && await session.start()) {
        emit('play')
    }
}

onMounted(load)
</script>

<style scoped>
.history-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.history-header > div { display: flex; gap: 8px; }
.history-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 12px; margin-top: 18px; }
.history-grid article { padding: 16px; background: #303030; border: 1px solid #ffffff12; border-radius: 8px; }
.history-grid h3 { margin: 0 0 10px; }
.history-grid p { color: #b9b9b9; font-size: 13px; }
</style>
