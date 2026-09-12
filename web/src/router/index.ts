import { createRouter, createWebHistory } from 'vue-router'
import { api } from '../api/client'
import AppLayout from '../layouts/AppLayout.vue'
import LoginView from '../views/LoginView.vue'
import ProxyView from '../views/ProxyView.vue'
import DashboardView from '../views/DashboardView.vue'
import DdnsView from '../views/DdnsView.vue'
import CertificatesView from '../views/CertificatesView.vue'
import LogsView from '../views/LogsView.vue'
import SettingsView from '../views/SettingsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/setup', redirect: '/login' },
    { path: '/login', name: 'login', component: LoginView, meta: { public: true } },
    {
      path: '/',
      component: AppLayout,
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/dashboard' },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: DashboardView,
          meta: { title: '仪表盘', description: '查看 Fonu 运行状态与关键指标' },
        },
        {
          path: 'ddns',
          name: 'ddns',
          component: DdnsView,
          meta: { title: 'DDNS', description: '自动同步公网 IP 到 DNS 解析' },
        },
        {
          path: 'certificates',
          name: 'certificates',
          component: CertificatesView,
          meta: { title: 'HTTPS 证书', description: '通过 ACME 自动申请和续签证书' },
        },
        {
          path: 'proxies',
          name: 'proxies',
          component: ProxyView,
          meta: { title: '反代', description: '管理通过域名访问的 NAS 服务' },
        },
        {
          path: 'logs',
          name: 'logs',
          component: LogsView,
          meta: { title: '日志', description: '查看访问、错误与系统运行日志' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: SettingsView,
          meta: { title: '设置', description: '配置系统参数与运行策略' },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const status = await api.authStatus()
  if (to.meta.requiresAuth && !status.authenticated) return { name: 'login' }
  if (to.name === 'login' && status.authenticated) return { name: 'dashboard' }
  return true
})

export default router
