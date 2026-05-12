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
]

const router = createRouter({
    history: createWebHashHistory(),
    routes,
})

router.beforeEach((to, from, next) => {
    next()
})

export default router
