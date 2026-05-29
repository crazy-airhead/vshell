import { createI18n } from 'vue-i18n'
import en from './en'
import zhCN from './zh-CN'

export type MessageSchema = typeof en

const savedLocale = localStorage.getItem('locale') || 'zh-CN'

const i18n = createI18n<[MessageSchema], 'en' | 'zh-CN'>({
  legacy: false,
  locale: savedLocale,
  fallbackLocale: 'en',
  messages: {
    en,
    'zh-CN': zhCN,
  },
})

export default i18n
