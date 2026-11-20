import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export type Institution = {
  id: number
  name: string
  city: string
  country: string
  continent: string
  subdivision: string
}

export type Destination = {
  scope: 'world' | 'continent' | 'region_group' | 'country' | 'subdivision'
  value: string
}

export type Period = { from: string; to: string }
export const hasPeriod = (period: Period) => Boolean(period.from && period.to)
export type Profile = Period & {
  email: string
  title: string
  name: string
  institution: Institution | null
  field: string
  teaching: string
  preferences: Destination[]
}
type Label = { code: string; en: string; de: string }
export type Catalog = { fields: Label[]; regionGroups: (Label & { countries: string[] })[]; countries: string[] }
export type Subdivision = { code: string; name: string }
export type Candidate = Period & {
  id: string
  title: string
  name: string
  institution: Institution
  field: string
  teaching: string
  demo: boolean
  overlapFrom: string
  overlapTo: string
  preferences: Destination[]
  sent: boolean
  received: boolean
}
export type Matches = Period & { results: Candidate[] }
export type MatchFilter = { allFields: boolean; region: string; preset: string; minimum: number }

const continents: Record<string, string> = { AF: '002', AN: '010', AS: '142', EU: '150', NA: '003', OC: '009', SA: '005' }
export const continentCodes = Object.keys(continents)
export const regionOptions = ['world', 'europe', 'eu', 'dach'] as const
export const basicDestinations: Destination[] = [
  { scope: 'world', value: '' },
  { scope: 'continent', value: 'EU' },
  { scope: 'region_group', value: 'EU' },
  { scope: 'region_group', value: 'DACH' },
]

export function useExchangeLabels() {
  const { t, locale } = useI18n()
  const display = computed(() => new Intl.DisplayNames(locale.value, { type: 'region', fallback: 'none' }))
  const dates = computed(() => new Intl.DateTimeFormat(locale.value, { month: 'short', year: 'numeric', timeZone: 'UTC' }))
  const localized = (label: Label) => locale.value === 'de' ? label.de : label.en
  const country = (code: string) => code ? display.value.of(code) ?? code : ''
  const continent = (code: string) => continents[code] ? display.value.of(continents[code]) ?? t(`continents.${code}`) : ''
  const place = (i: Institution) => [...new Set([i.city, i.subdivision, country(i.country)].filter(Boolean))].join(' · ')
  const month = (value: string) => dates.value.format(new Date(value + '-01T00:00:00Z'))
  const field = (code: string, catalog: Catalog) => {
    const label = catalog.fields.find(item => item.code === code)
    return label ? `${label.code} · ${localized(label)}` : code
  }
  function destination(item: Destination, catalog: Catalog) {
    switch (item.scope) {
      case 'world': return t('regions.world')
      case 'continent': return continent(item.value)
      case 'country': return country(item.value)
      case 'region_group': {
        const label = catalog.regionGroups.find(group => group.code === item.value)
        return label ? localized(label) : item.value
      }
      case 'subdivision': return country(item.value.split(':')[0]) + ' · ' + item.value.split(':')[1]
    }
  }
  return { t, locale, country, continent, place, month, field, destination }
}

export function academicPresets(year: number) {
  const start = String(year).padStart(4, '0')
  const next = String(year + 1).padStart(4, '0')
  return [
    { key: 'year', from: start + '-09', to: next + '-08' },
    { key: 'winter', from: start + '-10', to: next + '-03' },
    { key: 'summer', from: next + '-04', to: next + '-09' },
  ]
}

export function academicYear() {
  const now = new Date()
  return now.getFullYear() + Number(now.getMonth() >= 8)
}

export function monthIndex(value: string) {
  const [year, month] = value.split('-').map(Number)
  return year * 12 + month - 1
}

export function indexMonth(value: number) {
  return String(Math.floor(value / 12)).padStart(4, '0') + '-' + String(value % 12 + 1).padStart(2, '0')
}

export const sameDestination = (a: Destination, b: Destination) => a.scope === b.scope && a.value === b.value
