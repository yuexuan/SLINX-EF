import { createRouter, createWebHistory } from 'vue-router'
import { configStore } from '@/store/config'
import Auth from '../view/auth/Login.vue'
import Panel from '../view/panel/Panel.vue'
import Sub from '../view/sub/Sub.vue'

const router = createRouter({
    history: createWebHistory(window.__PANEL_PATH__ || '/'),
    routes: [
        {
            path: '/login',
            component: Auth,
            meta: { title: 'SLINX-EF · 登录' },
        },
        {
            path: '/',
            component: Panel,
            children: [
                {
                    path: '',
                    redirect: '/dashboard',
                },
                {
                    path: 'dashboard',
                    component: () => import('../view/panel/dashboard/Dashboard.vue'),
                    meta: { title: 'SLINX-EF · 仪表盘' },
                },
                {
                    path: 'inbound',
                    component: () => import('../view/panel/inbound/Inbound.vue'),
                    meta: { title: 'SLINX-EF · 入站管理' },
                },
                {
                    path: 'user',
                    component: () => import('../view/panel/user/User.vue'),
                    meta: { title: 'SLINX-EF · 用户管理' },
                },
                {
                    path: 'endpoint',
                    component: () => import('../view/panel/endpoint/Endpoint.vue'),
                    meta: { title: 'SLINX-EF · 节点管理' },
                },
                {
                    path: 'board',
                    component: () => import('../view/panel/board/Board.vue'),
                    meta: { title: 'SLINX-EF · 面板对接' },
                },
                {
                    path: 'detect',
                    component: () => import('../view/panel/detect/Detect.vue'),
                    meta: { title: 'SLINX-EF · IP检测' },
                },
                {
                    path: 'core',
                    component: () => import('../view/panel/core/Core.vue'),
                    meta: { title: 'SLINX-EF · 核心配置' },
                },
                {
                    path: 'config',
                    component: () => import('../view/panel/config/Config.vue'),
                    meta: { title: 'SLINX-EF · 面板配置' },
                },
            ],
        },
        {
            path: '/sub/:token',
            component: Sub,
            meta: { title: 'SLINX-EF · 订阅' },
        },
        {
            path: '/:pathMatch(.*)*',
            redirect: '/login',
        },
    ],
})

router.beforeEach((to) => {
    document.title = (to.meta.title as string) || 'SLINX-EF'

    const token = localStorage.getItem('token')

    if (token && to.path === '/login') {
        return '/'
    }

    if (!token && to.path !== '/login' && !to.path.startsWith(configStore.SubPath)) {
        return '/login'
    }
})

export default router
