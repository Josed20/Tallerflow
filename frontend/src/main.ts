import { createApp } from 'vue'
import { VueQueryPlugin } from '@tanstack/vue-query'
import App from './App.vue'
import { pinia } from './app/pinia'
import { router } from './app/router'
import './styles/main.css'

createApp(App).use(pinia).use(router).use(VueQueryPlugin).mount('#app')
