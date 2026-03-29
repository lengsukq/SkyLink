import { createApp } from 'vue'
import App from '../App.vue'
import router from '../router'
import '../styles/tailwind.css'
import '../styles/layout.css'

export function createSkyLinkApp() {
  const app = createApp(App)
  app.use(router)
  return app
}
