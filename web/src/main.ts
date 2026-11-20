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

watchEffect(() => {
  document.documentElement.lang = i18n.global.locale.value
  document.title = i18n.global.t('app.name')
})

createApp(App).use(i18n).mount('#app')
