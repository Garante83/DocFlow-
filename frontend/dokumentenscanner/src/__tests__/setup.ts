import { config } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import de from '../locales/de.json'
import en from '../locales/en.json'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'de',
  messages: { de, en }
})

config.global.plugins = [i18n]
