import { createApp } from 'vue'
import { createBootstrap } from 'bootstrap-vue-next'
import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-vue-next/dist/bootstrap-vue-next.css'
import '../../shared/styles/main.css'
import DeviceApp from './DeviceApp.vue'

createApp(DeviceApp).use(createBootstrap()).mount('#app')
