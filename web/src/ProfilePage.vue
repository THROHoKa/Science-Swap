<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { api, APIError, json } from './api'
import { basicDestinations, continentCodes, sameDestination, teachingExperiences, useExchangeLabels } from './exchange'
import type { Catalog, Destination, Institution, Profile, Subdivision, TeachingArea } from './exchange'
import MonthRangeFields from './MonthRangeFields.vue'

const { t, locale, country, continent, place, field, destination } = useExchangeLabels()
const profile = ref<Profile>()
const teachingTitle = ref('')
const teachingExperience = ref<NonNullable<TeachingArea['experience']> | ''>('')
const teachingTitleInput = ref<HTMLInputElement | null>(null)
const canAddTeachingArea = computed(() => (profile.value?.teachingAreas.length ?? 0) < 20 &&
  Boolean(teachingTitle.value.trim() && teachingExperience.value))
const catalog = ref<Catalog>()
const query = ref('')
const suggestions = ref<Institution[]>([])
const searched = ref(false)
const searching = ref(false)
const searchError = ref(false)
const results = ref<HTMLElement | null>(null)
const error = ref('')
const saving = ref(false)
const addPlace = ref(false)
const location = ref('')
const subdivision = ref('')
const subdivisions = ref<Subdivision[]>([])
const subdivisionNames = ref<Record<string, string>>({})
const countries = computed(() => [...(catalog.value?.countries ?? [])].sort((a, b) => country(a).localeCompare(country(b), locale.value)))

onMounted(async () => {
  try {
    const [saved, options] = await Promise.all([api<Profile>('/profile'), api<Catalog>('/catalog')])
    catalog.value = options
    query.value = saved.institution?.name ?? ''
    profile.value = saved
    for (const pref of saved.preferences) {
      if (pref.scope !== 'subdivision') continue
      const [code] = pref.value.split(':')
      api<Subdivision[]>('/subdivisions?country=' + code).then(items => {
        for (const item of items) subdivisionNames.value[code + ':' + item.code] = item.name
      }).catch(() => {})
    }
  } catch {
    error.value = 'unavailable'
  }
})

watch(query, (value, _, cleanup) => {
  suggestions.value = []
  searched.value = false
  searchError.value = false
  searching.value = false
  if (value.trim().length < 2 || value === profile.value?.institution?.name) return
  const controller = new AbortController()
  searching.value = true
  const delay = window.setTimeout(async () => {
    try {
      suggestions.value = await api<Institution[]>('/institutions?q=' + encodeURIComponent(value.trim()), { signal: controller.signal })
      searched.value = true
    } catch {
      if (!controller.signal.aborted) searchError.value = true
    } finally {
      if (!controller.signal.aborted) searching.value = false
    }
  }, 200)
  cleanup(() => {
    window.clearTimeout(delay)
    controller.abort()
  })
})

watch(location, (value, _, cleanup) => {
  subdivisions.value = []
  subdivision.value = ''
  if (!value.startsWith('country:')) return
  const code = value.slice(8)
  const controller = new AbortController()
  api<Subdivision[]>('/subdivisions?country=' + code, { signal: controller.signal }).then(items => {
    subdivisions.value = items
    for (const item of items) subdivisionNames.value[code + ':' + item.code] = item.name
  }).catch(() => {})
  cleanup(() => controller.abort())
})

function chooseInstitution(institution: Institution) {
  if (!profile.value) return
  profile.value.institution = institution
  query.value = institution.name
  suggestions.value = []
  searched.value = false
  document.getElementById('institution')?.focus()
}

const locationChips = computed(() => {
  const i = profile.value?.institution
  if (!i) return []
  const groups = catalog.value?.regionGroups.filter(group => group.countries.includes(i.country)).map(group => group.code) ?? []
  return [...new Set([i.city, i.subdivision, country(i.country), ...groups, continent(i.continent)].filter(Boolean))]
})

function toggleDestination(option: Destination, removeOnly = false) {
  if (!profile.value) return
  const current = profile.value.preferences
  if (current.some(item => sameDestination(item, option))) {
    if (removeOnly) profile.value.preferences = current.filter(item => !sameDestination(item, option))
    return
  }
  profile.value.preferences = option.scope === 'world' ? [option] : [...current.filter(item => item.scope !== 'world'), option]
}

function addDestination() {
  const [scope, value] = location.value.split(':')
  if (!value || (scope !== 'country' && scope !== 'continent')) return
  toggleDestination(subdivision.value ? { scope: 'subdivision', value: value + ':' + subdivision.value } : { scope, value })
  addPlace.value = false
  location.value = ''
}

function clearDates() {
  if (profile.value) Object.assign(profile.value, { from: '', to: '' })
}

function addTeachingArea() {
  const p = profile.value
  if (!p || !canAddTeachingArea.value || !teachingExperience.value) return
  p.teachingAreas.push({ title: teachingTitle.value.trim(), experience: teachingExperience.value })
  teachingTitle.value = ''
  teachingExperience.value = ''
  teachingTitleInput.value?.focus()
}

async function save() {
  const p = profile.value
  if (!p || saving.value) return
  error.value = ''
  if (!p.institution) {
    error.value = 'invalid_reference'
    document.getElementById('institution')?.focus()
    return
  }
  saving.value = true
  try {
    await api('/profile', json('PUT', {
      title: p.title, name: p.name, institutionId: p.institution.id, field: p.field,
      teachingAreas: p.teachingAreas, from: p.from, to: p.to, preferences: p.preferences,
    }))
    window.location.assign('/matches')
  } catch (problem) {
    error.value = problem instanceof APIError ? problem.code : 'unavailable'
    saving.value = false
    await nextTick()
    document.getElementById('profile-error')?.scrollIntoView({ block: 'nearest' })
  }
}

async function signOut() {
  try {
    await api('/session', { method: 'DELETE' })
    window.location.assign('/')
  } catch {
    error.value = 'unavailable'
  }
}
</script>

<template>
  <main class="mx-auto w-full max-w-[760px] space-y-6 px-4 py-8 sm:px-6 sm:py-12">
    <header class="flex flex-wrap items-start justify-between gap-x-4">
      <div class="space-y-2">
        <h1 class="font-display text-3xl font-bold sm:text-4xl">{{ t('profile.title') }}</h1>
        <p class="text-slate-600">{{ t('profile.description') }}</p>
      </div>
      <button v-if="profile" type="button" class="header-link text-sm underline" @click="signOut">{{ t('profile.signOut') }}</button>
    </header>
    <p v-if="error" id="profile-error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-red-800" role="alert">{{ t(`common.errors.${error}`) }}</p>
    <p v-if="!profile && !error" role="status">{{ t('common.loading') }}</p>

    <form v-if="profile && catalog" class="space-y-6" @submit.prevent="save">
      <fieldset class="panel space-y-4">
        <legend class="px-2 font-semibold">{{ t('profile.person') }}</legend>
        <div class="grid gap-4 sm:grid-cols-[1fr_3fr]">
          <label class="block space-y-1.5 text-sm font-semibold">
            <span>{{ t('profile.honorific') }}</span>
            <input v-model="profile.title" class="field text-base font-normal" maxlength="40" autocomplete="honorific-prefix">
          </label>
          <label class="block space-y-1.5 text-sm font-semibold">
            <span>{{ t('profile.name') }}</span>
            <input v-model="profile.name" class="field text-base font-normal" maxlength="120" autocomplete="name" required>
          </label>
        </div>
        <label class="block space-y-1.5 text-sm font-semibold">
          <span>{{ t('login.email') }}</span>
          <input :value="profile.email" class="field text-base font-normal text-slate-600" type="email" readonly>
        </label>
      </fieldset>

      <fieldset class="panel space-y-4">
        <legend class="px-2 font-semibold">{{ t('profile.academics') }}</legend>
        <div class="space-y-1.5">
          <label for="institution" class="block text-sm font-semibold">{{ t('profile.institution') }}</label>
          <input id="institution" v-model="query" class="field" autocomplete="off" maxlength="80"
            :placeholder="t('profile.search')" :aria-controls="suggestions.length ? 'institutions' : undefined"
            @input="profile.institution = null" @keydown.down.prevent="results?.querySelector('button')?.focus()">
          <p v-if="searching" class="text-sm text-slate-600" role="status">{{ t('common.loading') }}</p>
          <p v-else-if="searchError" class="text-sm text-red-700" role="status">{{ t('common.errors.unavailable') }}</p>
          <p v-else-if="searched && !suggestions.length && !profile.institution" class="text-sm text-slate-600" role="status">{{ t('profile.noInstitutions') }}</p>
          <ul v-if="suggestions.length" id="institutions" ref="results" class="max-h-64 divide-y divide-slate-100 overflow-auto rounded-lg border border-slate-200 shadow-sm" :aria-label="t('profile.suggestions')">
            <li v-for="institution in suggestions" :key="institution.id">
              <button type="button" class="block min-h-11 w-full px-3 py-2 text-left hover:bg-indigo-50 focus:bg-indigo-50" @click="chooseInstitution(institution)">
                <span class="block font-semibold">{{ institution.name }}</span>
                <span class="block text-sm text-slate-600">{{ place(institution) }}</span>
              </button>
            </li>
          </ul>
        </div>
        <div v-if="locationChips.length" class="space-y-2">
          <p class="text-sm font-semibold">{{ t('profile.detectedRegion') }}</p>
          <ul class="flex flex-wrap gap-2 text-sm">
            <li v-for="label in locationChips" :key="label" class="rounded-full bg-slate-100 px-3 py-1">{{ label }}</li>
          </ul>
        </div>
        <label class="block space-y-1.5 text-sm font-semibold">
          <span>{{ t('profile.field') }}</span>
          <select v-model="profile.field" class="field text-base font-normal" required>
            <option value="" disabled>{{ t('profile.chooseField') }}</option>
            <option v-for="item in catalog.fields" :key="item.code" :value="item.code">{{ field(item.code, catalog) }}</option>
          </select>
        </label>
      </fieldset>

      <fieldset class="panel space-y-4">
        <legend class="px-2 font-semibold">{{ t('profile.teaching') }}</legend>
        <ul v-if="profile.teachingAreas.length" class="space-y-2">
          <li v-for="(area, index) in profile.teachingAreas" :key="index" class="flex items-start justify-between gap-2 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2">
            <div class="min-w-0">
              <p class="wrap-break-word whitespace-pre-line font-semibold">{{ area.title }}</p>
              <p class="text-sm text-slate-600">{{ area.experience ? t(`teaching.levels.${area.experience}`) : t('teaching.unspecified') }}</p>
            </div>
            <button type="button" class="grid size-9 shrink-0 place-items-center rounded-md text-lg text-slate-600 hover:bg-slate-200"
              :aria-label="t('teaching.remove', { title: area.title })" @click="profile.teachingAreas.splice(index, 1)">
              <span aria-hidden="true">×</span>
            </button>
          </li>
        </ul>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="block space-y-1.5 text-sm font-semibold">
            <span>{{ t('teaching.title') }}</span>
            <input ref="teachingTitleInput" v-model="teachingTitle" class="field text-base font-normal" maxlength="500" @keydown.enter.prevent="addTeachingArea">
          </label>
          <label class="block space-y-1.5 text-sm font-semibold">
            <span>{{ t('teaching.experience') }}</span>
            <select v-model="teachingExperience" class="field text-base font-normal">
              <option value="" disabled>{{ t('teaching.chooseExperience') }}</option>
              <option v-for="experience in teachingExperiences" :key="experience" :value="experience">{{ t(`teaching.levels.${experience}`) }}</option>
            </select>
          </label>
        </div>
        <button type="button" class="button button-secondary px-4 disabled:opacity-50" :disabled="!canAddTeachingArea"
          @click="addTeachingArea">{{ t('teaching.add') }}</button>
        <p v-if="teachingTitle || teachingExperience" class="text-sm text-slate-600">{{ t('teaching.addBeforeSave') }}</p>
      </fieldset>

      <fieldset class="panel space-y-4">
        <legend class="px-2 font-semibold">{{ t('profile.exchange') }}</legend>
        <p id="optional-dates" class="text-sm text-slate-600">{{ t('profile.optionalDates') }}</p>
        <MonthRangeFields v-model:from="profile.from" v-model:to="profile.to" :from-label="t('profile.from')"
          :to-label="t('profile.to')" described-by="optional-dates" class="sm:grid-cols-2" />
        <button v-if="profile.from || profile.to" type="button" class="header-link text-sm underline" @click="clearDates">{{ t('profile.clearDates') }}</button>
        <div role="group" aria-labelledby="destinations-label" aria-describedby="optional-destinations" class="space-y-2">
          <p id="destinations-label" class="text-sm font-semibold">{{ t('profile.destinations') }}</p>
          <p id="optional-destinations" class="text-sm text-slate-600">{{ t('profile.optionalDestinations') }}</p>
          <div class="flex flex-wrap gap-2">
            <button v-for="preset in basicDestinations" :key="preset.scope + preset.value" type="button" class="chip"
              :aria-pressed="profile.preferences.some(item => sameDestination(item, preset))"
              @click="toggleDestination(preset, profile.preferences.some(item => sameDestination(item, preset)))">
              {{ destination(preset, catalog) }}
            </button>
            <button type="button" class="chip border-dashed" :aria-expanded="addPlace" @click="addPlace = !addPlace">{{ t('profile.addPlace') }}</button>
          </div>
          <div v-if="addPlace" class="grid gap-2 rounded-lg bg-slate-50 p-3">
            <label class="block space-y-1.5 text-sm font-semibold">
              <span>{{ t('profile.place') }}</span>
              <select v-model="location" class="field text-base font-normal">
                <option value="" disabled>{{ t('profile.choosePlace') }}</option>
                <optgroup :label="t('profile.continents')"><option v-for="code in continentCodes" :key="code" :value="'continent:' + code">{{ continent(code) }}</option></optgroup>
                <optgroup :label="t('profile.countries')"><option v-for="code in countries" :key="code" :value="'country:' + code">{{ country(code) }}</option></optgroup>
              </select>
            </label>
            <label v-if="subdivisions.length" class="block space-y-1.5 text-sm font-semibold">
              <span>{{ t('profile.subdivision') }}</span>
              <select v-model="subdivision" class="field text-base font-normal">
                <option value="">{{ t('profile.wholeCountry') }}</option>
                <option v-for="item in subdivisions" :key="item.code" :value="item.code">{{ item.name }}</option>
              </select>
            </label>
            <button type="button" class="button button-secondary w-fit px-4 disabled:opacity-50" :disabled="!location" @click="addDestination">{{ t('profile.add') }}</button>
          </div>
          <div class="flex flex-wrap gap-2">
            <template v-for="item in profile.preferences" :key="item.scope + item.value">
              <button v-if="!basicDestinations.some(preset => sameDestination(preset, item))" type="button" class="chip" aria-pressed="true"
                :aria-label="t('profile.removePlace', { place: subdivisionNames[item.value] || destination(item, catalog) })" @click="toggleDestination(item, true)">
                {{ subdivisionNames[item.value] || destination(item, catalog) }} <span class="ml-2" aria-hidden="true">×</span>
              </button>
            </template>
          </div>
        </div>
      </fieldset>
      <div class="flex sm:justify-end">
        <button type="submit" class="button button-primary min-h-12 w-full px-6 disabled:opacity-60 sm:w-auto" :disabled="saving">{{ t(saving ? 'common.loading' : 'profile.save') }}</button>
      </div>
    </form>
  </main>
</template>
