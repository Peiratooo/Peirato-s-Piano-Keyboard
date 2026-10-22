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
        redirect: to => ({path: '/', query: {...to.query, midiMode: 'play'}}),
    },
]

const router = createRouter({
    history: createWebHashHistory(),
    routes,
})

export default router
