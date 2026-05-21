<template>
    <section class="content-card about-card">
        <Author />
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
import {inject} from "vue";

const resetConfig = inject('resetConfig')

</script>

<style lang="scss" scoped>
.about-card {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-direction: column;
}
</style>
