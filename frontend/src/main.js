import { createApp, h, onMounted, ref } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import App from './App.vue'
import router from './router'
import './styles.css'

// Expose host Vue runtime helpers for remote plugins in /public.
window.__SKOLL_VUE__ = { h, onMounted, ref }

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlus)

app.mount('#app')
