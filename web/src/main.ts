import { createApp } from 'vue'
import naive from 'naive-ui'
import App from './App.vue'
import router from './router'

const app = createApp(App)

app.config.errorHandler = (err, _instance, info) => {
  console.error('[fonu]', info, err)
}

app.use(router).use(naive)
router.isReady().then(() => app.mount('#app')).catch((err: unknown) => {
  console.error('[fonu] startup', err)
  const text = document.getElementById('fonu-boot-text')
  const retry = document.getElementById('fonu-boot-retry')
  if (text) text.textContent = 'Fonu 初始化失败，请重新加载。'
  if (retry) retry.hidden = false
})
