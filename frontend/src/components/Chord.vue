<template>
    <transition enter-active-class="blurFadeIN" leave-active-class="blurFadeOUT">
        <div class="chord" v-if="Object.keys(displayChord).length > 0 && displayChord.type === 'chord'">
            <div class="top">
                <div class="left">{{ displayChord.chord }}</div>
                <div class="right">
                    <div class="symbols" v-if="displayChord.alternativeSymbols && displayChord.alternativeSymbols.length > 0">
                        <div class="symbol" v-for="item in displayChord.alternativeSymbols.slice(0, 3)" :key="item">
                            {{ item }}
                        </div>
                    </div>
                    <div class="synonym-notes" v-else>
                        <div class="notes">{{ displayChord.notes?.join(' ') }}</div>
                    </div>
                    <div class="cn">{{ displayChord.chinese }}</div>
                </div>
            </div>
            <div class="bottom">
                <div class="en">{{ displayChord.name }}</div>
            </div>
        </div>

        <div class="note" v-else-if="Object.keys(displayChord).length > 0 && displayChord.type === 'note'">
            {{ displayChord.note }}
        </div>

        <div class="note unknown" v-else-if="Object.keys(displayChord).length > 0 && displayChord.type === 'unknown'">
            <span v-for="(item,index) in displayChord.notes" :key="index">
                {{ item }}
            </span>
        </div>
    </transition>
</template>

<script setup>
import {computed, inject, ref, watch} from 'vue'
import {detectChord} from '../services/chordEngine'

const store = inject('store')

// rawChord 完全实时；displayChord 做轻微防抖，避免用户按下和弦时因为手指先后落键而频繁闪烁。
const rawChord = computed(() => detectChord(store.pressedKey, store.chordsname))
const displayChord = ref({})
let debounceTimer = null

watch(
    rawChord,
    (value) => {
        clearTimeout(debounceTimer)
        debounceTimer = setTimeout(() => {
            displayChord.value = value
        }, 50)
    },
    {immediate: true, deep: true},
)
</script>

<style lang="scss" scoped>
* {
    user-select: none;
}
.chord,.note {
    background-color: #eeeeee55;
    border-radius: 4px;
    display: flex;
    flex-direction: column;
    padding: 8px;
    backdrop-filter: blur(4px);
    box-shadow: 0 0 6px 0 rgba(0, 0, 0, 0.25);
    gap: 6px;

    .top {
        display: flex;
        height: 80%;
        gap: 8px;

        .left {
            width: 100%;
            font-size: 36px;
        }

        .right {
            padding: 3px 0;
            justify-content: space-between;
            display: flex;
            align-items: flex-end;
            flex-direction: column;
        }
    }
}

.note {
    font-size: 32px;
    padding: 12px 16px;
    color: #333333;
}

.unknown {
    font-size: 18px;
    display: flex;
    flex-direction: row;
}

.synonym-notes {
    font-size: 12px;
    display: flex;
    justify-content: space-between;
}

.symbols {
    display: flex;
    font-size: 12px;
    gap: 8px;
    color: #333333;
    white-space: nowrap;
}

.en, .cn {
    font-size: 12px;
    white-space: nowrap;
}
</style>
