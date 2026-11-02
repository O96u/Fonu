import { createRouter, createWebHistory } from 'vue-router'
import { api } from '../api/client'
import AppLayout from '../layouts/AppLayout.vue'
import LoginView from '../views/LoginView.vue'
import SetupView from '../views/SetupView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/setup', name: 'setup', component: SetupView, meta: { public: true } },
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
          component: () => import('../views/DashboardView.vue'),
          meta: { title: '仪表盘', description: '查看 Fonu 运行状态与关键指标' },
        },
        {
          path: 'ddns',
          name: 'ddns',
          component: () => import('../views/DdnsView.vue'),
          meta: { title: 'DDNS', description: '自动同步公网 IP 到 DNS 解析' },
        },
        { path: 'dnshe-domains', redirect: { name: 'ddns' } },
        {
          path: 'certificates',
          name: 'certificates',
          component: () => import('../views/CertificatesView.vue'),
          meta: { title: '证书管理', description: '通过 ACME 自动申请和续签证书' },
        },
        {
          path: 'proxies',
          name: 'proxies',
          component: () => import('../views/ProxyView.vue'),
          meta: { title: '反向代理', description: '管理通过域名访问的 NAS 服务' },
        },
        {
          path: 'frp',
          name: 'frp',
          component: () => import('../views/FrpView.vue'),
          meta: { title: '内网穿透', description: '通过 FRP 将公网流量转发到 Fonu Nginx' },
        },
        {
          path: 'logs',
          name: 'logs',
          component: () => import('../views/LogsView.vue'),
          meta: { title: '日志中心', description: '查看系统、访问与 Nginx 错误日志' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('../views/SettingsView.vue'),
          meta: { title: '系统设置', description: '配置系统参数与运行策略' },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.name === 'dashboard') {
    void import('../components/AnalyticsAccessMap.vue')
  }
  const status = await api.authStatus()
  if (!status.initialized && to.name !== 'setup') {
    return { name: 'setup' }
  }
  if (status.initialized && to.name === 'setup') {
    return { name: 'login' }
  }
  if (to.meta.requiresAuth && !status.authenticated) return { name: 'login' }
  if ((to.name === 'login' || to.name === 'setup') && status.authenticated) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
