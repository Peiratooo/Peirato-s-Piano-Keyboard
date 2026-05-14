import {createRouter, createWebHashHistory} from 'vue-router'

const routes = [
    {
        path: '/',
        name: 'main-window',
        component: () => import('../views/MainWindow.vue'),
    },
    {
        path: '/control',
        name: 'control-center',
        component: () => import('../views/ControlCenterWindow.vue'),
    },
    {
        path: '/midi',
        name: 'midi-window',
        component: () => import('../views/MidiWindow.vue'),
    },
]

const router = createRouter({
    history: createWebHashHistory(),
    routes,
})

export default router
