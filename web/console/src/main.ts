import { createApp } from 'vue'
import { createBootstrap } from 'bootstrap-vue-next'
import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-icons/font/bootstrap-icons.css'
import 'bootstrap-vue-next/dist/bootstrap-vue-next.css'
import '../../shared/styles/main.css'
import App from './App.vue'
import router from './router'

createApp(App).use(createBootstrap()).use(router).mount('#app')
