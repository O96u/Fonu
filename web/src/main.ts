import { createApp } from 'vue'
import App from './App.vue'
import router from './router'

performance.mark('fonu:entry')
const app = createApp(App)

app.config.errorHandler = (err, _instance, info) => {
  console.error('[fonu]', info, err)
}

app.use(router)
router.isReady().then(() => {
  performance.mark('fonu:route-ready')
  app.mount('#app')
  performance.mark('fonu:mounted')
  performance.measure('fonu:route-init', 'fonu:entry', 'fonu:route-ready')
  performance.measure('fonu:mount', 'fonu:route-ready', 'fonu:mounted')
}).catch((err: unknown) => {
  console.error('[fonu] startup', err)
  const text = document.getElementById('fonu-boot-text')
  const retry = document.getElementById('fonu-boot-retry')
  if (text) text.textContent = 'Fonu 初始化失败，请重新加载。'
  if (retry) retry.hidden = false
})
