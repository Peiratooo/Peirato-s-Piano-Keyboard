<template>
    <div class="author-card">
        <div class="left">
            <div class="author">
                <div class="avatar">
                    <n-avatar src="/avatar.jpg" :size="72"/>
                </div>
                <div class="info">
                    <div class="name">Peirato</div>
                    <div class="social">
                        <button class="item" v-for="(item,index) in platforms" :key="index" :aria-label="item.name" :title="item.name" @click="openPlatform(index)">
                            <img :src="item.icon" :alt="item.name" width="28" height="28" />
                        </button>
                    </div>
                </div>
            </div>
            <div class="desc">
                <div class="text">
                    {{desc[0]}}
                </div>

            </div>
        </div>
        <div class="right">
            <div class="title">支持独立开发</div>
            <div class="qrcodes">
                <div class="qrcode" v-for="item in donate" :key="item.name">
                    <n-qr-code :padding="0"  style="box-sizing: content-box" :size="88" :value="item.url" :icon-src="item.icon" :icon-size="24"/>
                    <span class="qr-label">{{ item.name }}</span>
                </div>
            </div>
        </div>
        <n-modal v-model:show="showSocial.show">
            <div class="platform">
                <n-qr-code :padding="0" style="box-sizing: content-box" :size="168" :value="platforms[showSocial.index].url" :icon-src="platforms[showSocial.index].icon" :icon-size="32" />
            </div>
        </n-modal>
    </div>
</template>

<script setup>
import {NQrCode,NAvatar,NModal} from "naive-ui";
import {inject, ref} from "vue";

const Keyboard = inject("Keyboard")



const showSocial = ref({
    show:false,
    index:-1,
})

function openPlatform(index) {
    if (platforms[index].method === 'url') Keyboard.OpenUrl(platforms[index].url)
    else showSocial.value = {show: true, index}
}

const platforms = [
    {
        name: "Bilibili",
        icon: "/bilibili.png",
        url: "https://space.bilibili.com/7277347",
        method: "url"
    },
    {
        name: "抖音",
        icon: "/tiktok.png",
        url: "https://www.douyin.com/user/MS4wLjABAAAAENe7s0M7uUpWwOqCXhoRiBD85CeYolcPcFltgjYW-hw",
        method: "url"
    },
    {
        name: "微信",
        icon: "/wechat.png",
        url: "https://u.wechat.com/MBX8JgLr7FANfm5l5RYByAg",
        method: "qrcode"
    },
    {
        name: "GitHub",
        icon: "/github.png",
        url: "https://github.com/Peiratooo",
        method: "url"
    },
]

const desc = [
    "一个简单、免费、无广告的钢琴键盘挂件"
]

const donate = [
    {
        name: "支付宝",
        icon: "/alipay.png",
        url: "https://qr.alipay.com/fkx17082m1wjekpkpwhif58",
        method: "qrcode"
    },
    {
        name: "微信",
        icon: "/wechat.png",
        url: "wxp://f2f0umI-AZys25kpxmHZYNmRW4SbywxsqwZmYYGVLWb_MfU",
        method: "qrcode"
    },
]

</script>

<style lang="scss" scoped>
.author-card { display:flex; align-items:center; justify-content:space-between; gap:24px; width:100%; box-sizing:border-box; }
.author { display:flex; align-items:center; gap:14px; }
.name { font-size:24px; font-weight:700; letter-spacing:-.5px; color:#1e293b; }
.social { display:flex; gap:8px; margin-top:8px; }
.item { display:flex; padding:0; border:0; background:transparent; border-radius:7px; overflow:hidden; cursor:pointer; transition:transform .15s; }
.item:hover { transform:translateY(-2px); }
.item:focus-visible { outline:2px solid #2563eb; outline-offset:3px; }
.desc { margin-top:16px; color:#64748b; font-size:12px; line-height:1.8; }
.title { font-size:12px; color:#64748b; margin-bottom:12px; }
.qrcodes { display:flex; gap:16px; }
.qrcode { display:flex; flex-direction:column; gap:7px; align-items:center; }
.qr-label { font-size:11px; color:#64748b; }
.platform { background:#fff; border-radius:16px; padding:24px; }
@media(max-width:650px) { .author-card { align-items:flex-start; flex-direction:column; } }
@media(prefers-reduced-motion:reduce) { .item { transition:none; } }
</style>
