<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api, APIError } from './api'
import { academicPresets, academicYear, hasPeriod, indexMonth, monthIndex, useExchangeLabels } from './exchange'
import type { Candidate, Catalog, MatchFilter, Matches, Period, Profile } from './exchange'
import MatchFilters from './MatchFilters.vue'

const { t, month, place, field, destination } = useExchangeLabels()
const profile = ref<Profile>()
const catalog = ref<Catalog>()
const matches = ref<Matches>()
const error = ref('')
const loading = ref(true)
const sheet = ref<HTMLDialogElement | null>(null)
const pending = ref(new Set<string>())
const filters = ref<MatchFilter>({ allFields: false, region: 'world', preset: 'saved', minimum: 1 })
const samePeriod = (a: Period, b: Period) => a.from === b.from && a.to === b.to
const periods = computed(() => {
  const p = profile.value
  if (!p) return []
  if (!hasPeriod(p)) return [{ key: 'any', from: '', to: '' }, ...academicPresets(academicYear())]
  const [year, monthNumber] = p.from.split('-').map(Number)
  const anchor = year - Number(monthNumber < 9)
  const presets = academicPresets(anchor || academicYear())
  return presets.some(preset => samePeriod(preset, p)) ? presets : [{ key: 'saved', from: p.from, to: p.to }, ...presets]
})

onMounted(async () => {
  try {
    const [saved, options] = await Promise.all([api<Profile>('/profile'), api<Catalog>('/catalog')])
    if (!saved.institution) {
      error.value = 'profile_required'
      loading.value = false
      return
    }
    catalog.value = options
    profile.value = saved
    filters.value = { ...filters.value, preset: periods.value.find(period => samePeriod(period, saved))?.key ?? 'saved' }
  } catch {
    error.value = 'unavailable'
    loading.value = false
  }
})

watch([profile, filters], async ([p, f], _, cleanup) => {
  if (!p) return
  const controller = new AbortController()
  cleanup(() => controller.abort())
  const selected = periods.value.find(period => period.key === f.preset) ?? p
  const query = new URLSearchParams({
    from: selected.from, to: selected.to, region: f.region,
    field: f.allFields ? 'all' : 'same',
  })
  if (hasPeriod(selected)) query.set('minimum', String(f.minimum))
  error.value = ''
  loading.value = true
  try {
    matches.value = await api<Matches>('/matches?' + query, { signal: controller.signal })
  } catch (problem) {
    if (!controller.signal.aborted) error.value = problem instanceof APIError ? problem.code : 'unavailable'
  } finally {
    if (!controller.signal.aborted) loading.value = false
  }
})

const axis = computed(() => {
  const m = matches.value
  const dated = m ? [m, ...m.results].filter(hasPeriod) : []
  if (!dated.length) return { first: 0, span: 1, ticks: [] as string[] }
  const first = Math.min(...dated.map(period => monthIndex(period.from)))
  const last = Math.max(...dated.map(period => monthIndex(period.to)))
  const ticks = [...new Set([first, Math.round((first + last) / 2), last])].map(indexMonth)
  return { first, span: last - first + 1, ticks }
})

function bar(period: Period) {
  if (!hasPeriod(period)) return { '--bar-start': '0%', '--bar-width': '100%' }
  const first = monthIndex(period.from)
  return {
    '--bar-start': (first - axis.value.first) * 100 / axis.value.span + '%',
    '--bar-width': (monthIndex(period.to) - first + 1) * 100 / axis.value.span + '%',
  }
}

const initials = (name: string) => name.trim().split(/\s+/).slice(0, 2).map(part => [...part][0]?.toUpperCase() ?? '').join('')
const months = (from: string, to: string) => monthIndex(to) - monthIndex(from) + 1
const dateSpan = (period: Period) => `${month(period.from)} – ${month(period.to)}`
const overlapPeriod = (candidate: Candidate): Period => ({ from: candidate.overlapFrom, to: candidate.overlapTo })

function availability(candidate: Candidate, requested: Period) {
  const available = hasPeriod(candidate) ? t('matches.available', { period: dateSpan(candidate) }) : t('matches.flexible')
  const overlap = overlapPeriod(candidate)
  return hasPeriod(requested) && hasPeriod(overlap)
    ? t('matches.availability', { available, shared: dateSpan(overlap) })
    : available
}

async function requestContact(candidate: Candidate) {
  if (pending.value.has(candidate.id)) return
  pending.value.add(candidate.id)
  error.value = ''
  try {
    await api('/interests/' + candidate.id, { method: 'PUT' })
    candidate.sent = true
  } catch (problem) {
    error.value = problem instanceof APIError ? problem.code : 'unavailable'
  } finally {
    pending.value.delete(candidate.id)
  }
}
</script>

<template>
  <main class="mx-auto grid w-full max-w-[1240px] items-start gap-6 px-4 py-8 sm:px-6 lg:grid-cols-[minmax(16rem,20rem)_minmax(0,1fr)]">
    <aside v-if="profile && catalog" class="panel hidden lg:block">
      <MatchFilters v-model="filters" :profile="profile" :catalog="catalog" :periods="periods" />
    </aside>
    <section class="min-w-0 space-y-4" aria-labelledby="match-heading">
      <div class="flex flex-wrap items-baseline justify-between gap-2">
        <h1 id="match-heading" class="font-display text-3xl font-bold">{{ matches ? t('matches.count', matches.results.length) : t('matches.navigation') }}</h1>
        <p v-if="profile" class="text-sm text-slate-600">{{ t('matches.explanation') }}</p>
      </div>

      <div v-if="profile && catalog" class="flex min-w-0 items-center gap-2 lg:hidden">
        <button type="button" class="button button-primary shrink-0 rounded-full px-4" @click="sheet?.showModal()">{{ t('matches.filters') }}</button>
        <ul class="flex min-w-0 gap-2 overflow-x-auto py-1 text-sm">
          <li v-if="!filters.allFields" class="summary-chip">{{ field(profile.field, catalog) }}</li>
          <li class="summary-chip">{{ t(`regions.${filters.region}`) }}</li>
          <li class="summary-chip">{{ t(`periods.${filters.preset}`) }}</li>
          <li v-if="filters.preset !== 'any'" class="summary-chip">{{ t('matches.minimumValue', { months: filters.minimum }) }}</li>
        </ul>
      </div>

      <div v-if="error" class="panel border-red-200 text-red-800" role="alert">
        {{ t(`common.errors.${error}`) }}
        <a v-if="error === 'profile_required'" href="/profile" class="header-link ml-2 underline">{{ t('profile.navigation') }}</a>
      </div>
      <p v-if="loading" class="text-sm text-slate-600" role="status">{{ t('common.loading') }}</p>

      <template v-if="matches && catalog && profile">
        <div v-if="axis.ticks.length || !hasPeriod(matches)" class="panel p-4">
          <p v-if="!hasPeriod(matches)" class="text-sm font-semibold">{{ t('matches.anyDates') }}</p>
          <div v-if="axis.ticks.length" class="grid grid-cols-[3rem_1fr] items-center gap-x-3 gap-y-1 text-sm" aria-hidden="true">
            <span class="font-semibold">{{ t('matches.you') }}</span>
            <div class="track"><span class="range bg-indigo-400" :style="bar(matches)"></span></div>
            <div class="col-start-2 flex justify-between gap-1 text-xs text-slate-600">
              <span v-for="tick in axis.ticks" :key="tick">{{ month(tick) }}</span>
            </div>
          </div>
        </div>
        <p v-if="!matches.results.length && !loading" class="panel border-dashed text-center text-slate-600">{{ t('matches.empty') }}</p>
        <article v-for="candidate in matches.results" :key="candidate.id" class="panel space-y-4">
          <div class="flex items-start justify-between gap-3">
            <div class="flex min-w-0 gap-3">
              <span class="grid size-10 shrink-0 place-items-center rounded-full bg-indigo-100 text-sm font-semibold text-indigo-900 sm:size-12" aria-hidden="true">{{ initials(candidate.name) }}</span>
              <div class="min-w-0 space-y-1">
                <h2 class="wrap-break-word font-semibold sm:text-lg">{{ [candidate.title, candidate.name].filter(Boolean).join(' ') }}</h2>
                <p class="wrap-break-word text-sm">{{ candidate.institution.name }}</p>
                <p class="text-sm text-slate-600">{{ place(candidate.institution) }}</p>
              </div>
            </div>
            <div v-if="hasPeriod(matches) && candidate.overlapFrom" class="min-w-16 shrink-0 space-y-1 text-right sm:min-w-32">
              <p class="font-display text-2xl leading-none font-bold sm:text-3xl">{{ months(candidate.overlapFrom, candidate.overlapTo) }}/{{ months(matches.from, matches.to) }}</p>
              <p class="text-xs text-slate-600">{{ t('matches.overlap') }}</p>
              <progress class="overlap-progress hidden w-full sm:block" :value="months(candidate.overlapFrom, candidate.overlapTo)" :max="months(matches.from, matches.to)" :aria-label="t('matches.overlap')"></progress>
            </div>
            <div v-else-if="hasPeriod(candidate)" class="min-w-16 shrink-0 space-y-1 text-right sm:min-w-32">
              <p class="font-display text-2xl leading-none font-bold sm:text-3xl">{{ months(candidate.from, candidate.to) }}</p>
              <p class="text-xs text-slate-600">{{ t('matches.availableMonths') }}</p>
            </div>
          </div>
          <div class="flex flex-wrap gap-1.5 text-xs">
            <span class="summary-chip bg-slate-100 text-slate-700">{{ field(candidate.field, catalog) }}</span>
            <span v-if="candidate.field !== profile.field" class="summary-chip bg-amber-50 text-amber-800">{{ t('matches.otherField') }}</span>
            <span v-if="candidate.demo" class="summary-chip">{{ t('matches.demo') }}</span>
            <span v-if="candidate.preferences.length" class="summary-chip bg-slate-100 text-slate-700">{{ t('matches.seeks', { places: candidate.preferences.map(item => destination(item, catalog!)).join(', ') }) }}</span>
          </div>
          <p v-if="candidate.teaching" class="wrap-break-word text-sm text-slate-600">{{ candidate.teaching }}</p>
          <div v-if="axis.ticks.length" class="grid grid-cols-[3rem_1fr] items-center gap-x-3 gap-y-2 text-sm text-slate-600" aria-hidden="true">
            <span>{{ t('matches.you') }}</span>
            <div class="track">
              <span class="range bg-indigo-200" :style="bar(matches)"></span>
              <span v-if="candidate.overlapFrom" class="range bg-amber-500" :style="bar(overlapPeriod(candidate))"></span>
            </div>
            <span>{{ initials(candidate.name) }}</span>
            <div class="track">
              <span class="range bg-slate-300" :style="bar(candidate)"></span>
              <span v-if="candidate.overlapFrom" class="range bg-amber-500" :style="bar(overlapPeriod(candidate))"></span>
            </div>
          </div>
          <div class="flex flex-wrap items-center justify-between gap-2 border-t border-slate-100 pt-3">
            <p class="text-sm text-slate-600">{{ availability(candidate, matches) }}</p>
            <p v-if="candidate.sent && candidate.received" class="text-sm font-semibold text-indigo-800">{{ t('matches.mutual') }}</p>
            <p v-else-if="candidate.sent" class="text-sm font-semibold text-indigo-800">{{ t('matches.sent') }}</p>
            <button v-else type="button" class="button button-primary w-full px-4 disabled:opacity-50 sm:w-auto" :disabled="pending.has(candidate.id)" @click="requestContact(candidate)">
              {{ t(candidate.received ? 'matches.accept' : 'matches.request') }}
            </button>
          </div>
          <p v-if="candidate.sent && candidate.received" class="text-sm text-slate-600">{{ t('matches.emailLater') }}</p>
        </article>
      </template>
    </section>
    <dialog v-if="profile && catalog" ref="sheet" class="fixed inset-x-0 top-auto bottom-0 m-0 max-h-[85dvh] w-full max-w-none overflow-hidden rounded-t-2xl bg-white p-0 backdrop:bg-slate-950/45 lg:hidden" aria-labelledby="filter-title">
      <div class="flex max-h-[85dvh] flex-col">
        <div class="flex shrink-0 items-center justify-between gap-2 px-5 py-3">
          <h2 id="filter-title" class="font-display text-2xl font-bold">{{ t('matches.filters') }}</h2>
          <button type="button" class="chip" @click="sheet?.close()">{{ t('common.close') }}</button>
        </div>
        <div class="min-h-0 overflow-y-auto px-5 py-2">
          <MatchFilters v-model="filters" :profile="profile" :catalog="catalog" :periods="periods" />
        </div>
        <div class="shrink-0 border-t border-slate-200 p-4">
          <button type="button" class="button button-primary w-full px-4" @click="sheet?.close()">{{ t('matches.show', matches?.results.length ?? 0) }}</button>
        </div>
      </div>
    </dialog>
  </main>
</template>
