<template>
    <section class="content-card about-card">
        <Author />
        <div class="setting-actions">
            <n-button size="small" :loading="checking" :disabled="installing" @click="checkUpdate">检查更新</n-button>
            <n-button v-if="update?.available" size="small" type="primary" :loading="installing" @click="installUpdate">安装 {{ update.version }} 并重启</n-button>
            <span role="status">{{ updateStatus }}</span>
        </div>
        <div class="setting-actions">
            <n-popover trigger="hover">
                <template #trigger>
                    <n-button size="small" type="info">恢复默认设置</n-button>
                </template>
                <div>
                    此操作将恢复软件默认设置，并清空音源/Midi列表。
                    <n-button size="small" @click="resetConfig" type="warning">确定</n-button>
                </div>
            </n-popover>
        </div>
    </section>
</template>

<script setup>
import Author from '../Author.vue'
import {NButton,NPopover} from "naive-ui";
import {inject, ref} from "vue";
import {Keyboard} from '../../../bindings/main/service'

const resetConfig = inject('resetConfig')
const checking = ref(false)
const installing = ref(false)
const update = ref(null)
const updateStatus = ref('')

async function checkUpdate() {
    checking.value = true
    update.value = null
    try {
        update.value = await Keyboard.CheckUpdate()
        updateStatus.value = !update.value.supported ? '请重新安装包含 updater 的完整软件包' : update.value.available ? `发现新版本 ${update.value.version}` : '暂无更新'
    } catch (error) { updateStatus.value = `检查失败：${String(error)}` }
    finally { checking.value = false }
}

async function installUpdate() {
    installing.value = true
    updateStatus.value = '正在下载并校验更新，请稍候…'
    try { await Keyboard.InstallUpdate() }
    catch (error) { updateStatus.value = `更新失败：${String(error)}` }
    finally { installing.value = false }
}

</script>

<style lang="scss" scoped>
.about-card {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-direction: column;
}
</style>
