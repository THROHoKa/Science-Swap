import { createApp, watchEffect } from 'vue'
import { createI18n } from 'vue-i18n'
import App from './App.vue'
import { initialLocale } from './locale'
import de from './locales/de.json'
import en from './locales/en.json'
import './style.css'

const i18n = createI18n({
  legacy: false,
  locale: initialLocale(),
  fallbackLocale: 'en',
  messages: { de, en },
})

const path = window.location.pathname
const page = path === '/login' ? 'login' : path === '/profile' ? 'profile' : path === '/matches' ? 'matches' : 'home'

watchEffect(() => {
  const { t, locale } = i18n.global
  document.documentElement.lang = locale.value
  document.title = page === 'home' ? t('app.name') : t(`${page}.pageTitle`)
})

createApp(App, { page }).use(i18n).mount('#app')
