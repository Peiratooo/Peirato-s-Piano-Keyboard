<template><span style="display:none"></span></template>
<script setup>
import {useMessage,useNotification} from 'naive-ui'
const message = useMessage()
const notification = useNotification()

window.$message = message
window.$notify = {
    error(title, content) {
        notification.error({
            title,
            content,
            duration: 5200,
            keepAliveOnHover: true,
        })
    },
    success(title, content) {
        notification.success({
            title,
            content,
            duration: 2800,
            keepAliveOnHover: true,
        })
    },
    warning(title, content) {
        notification.warning({
            title,
            content,
            duration: 4200,
            keepAliveOnHover: true,
        })
    },
    info(title, content) {
        notification.info({
            title,
            content,
            duration: 3200,
            keepAliveOnHover: true,
        })
    },
}
</script>
