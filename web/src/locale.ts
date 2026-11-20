const locales = ['de', 'en'] as const
type Locale = (typeof locales)[number]

const languageStorageKey = 'science-swap.language'

function supportedLocale(value: string | null): Locale | undefined {
  return locales.find((locale) => locale === value)
}

export function initialLocale(): Locale {
  try {
    const saved = supportedLocale(localStorage.getItem(languageStorageKey))
    if (saved) return saved
  } catch {
    // The language switcher still works if browser storage is unavailable.
  }
  for (const preference of navigator.languages) {
    const locale = supportedLocale(preference.toLowerCase().split('-')[0])
    if (locale) return locale
  }
  return 'en'
}

export function saveLocale(locale: string) {
  try {
    localStorage.setItem(languageStorageKey, locale)
  } catch {
    // The current selection remains active for this page.
  }
}
