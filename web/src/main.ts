import { createApp } from 'vue'
import naive from 'naive-ui'
import App from './App.vue'
import router from './router'

const app = createApp(App)

app.config.errorHandler = (err, _instance, info) => {
  console.error('[fonu]', info, err)
}

app.use(router).use(naive).mount('#app')
