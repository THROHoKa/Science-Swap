<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, APIError } from './api'
import { hasPeriod, useExchangeLabels } from './exchange'
import type { CandidateDetails, Catalog } from './exchange'
import MatchContact from './MatchContact.vue'

const props = defineProps<{ candidateId: string }>()
const { t, place, continent, field, month, destination } = useExchangeLabels()
const profile = ref<CandidateDetails>()
const catalog = ref<Catalog>()
const error = ref('')

onMounted(async () => {
  try {
    const [candidate, options] = await Promise.all([
      api<CandidateDetails>('/matches/' + encodeURIComponent(props.candidateId)),
      api<Catalog>('/catalog'),
    ])
    catalog.value = options
    profile.value = candidate
  } catch (problem) {
    error.value = problem instanceof APIError ? problem.code : 'unavailable'
  }
})
</script>

<template>
  <main class="mx-auto w-full max-w-[760px] space-y-4 px-4 py-8 sm:px-6">
    <a href="/matches" class="header-link inline-flex min-h-11 items-center gap-2 font-medium">
      <span aria-hidden="true">←</span> {{ t('matchProfile.back') }}
    </a>
    <p v-if="error" class="panel border-red-200 text-red-800" role="alert">{{ t(`common.errors.${error}`) }}</p>
    <p v-else-if="!profile" role="status">{{ t('common.loading') }}</p>

    <template v-if="profile && catalog">
      <header class="panel space-y-2">
        <h1 class="wrap-break-word font-display text-3xl font-bold">{{ [profile.title, profile.name].filter(Boolean).join(' ') }}</h1>
        <p class="font-semibold">{{ profile.institution.name }}</p>
        <p class="text-sm text-slate-600">{{ [place(profile.institution), continent(profile.institution.continent)].filter(Boolean).join(' · ') }}</p>
        <p v-if="profile.demo" class="summary-chip w-fit text-sm">{{ t('matches.demo') }}</p>
      </header>

      <dl class="panel grid gap-5 sm:grid-cols-2">
        <div>
          <dt class="font-semibold">{{ t('profile.field') }}</dt>
          <dd class="mt-1">{{ field(profile.field, catalog) }}</dd>
        </div>
        <div>
          <dt class="font-semibold">{{ t('matchProfile.availability') }}</dt>
          <dd class="mt-1">{{ hasPeriod(profile) ? `${month(profile.from)} – ${month(profile.to)}` : t('matches.flexible') }}</dd>
        </div>
        <div class="sm:col-span-2">
          <dt class="font-semibold">{{ t('profile.destinations') }}</dt>
          <dd class="mt-1">
            <ul v-if="profile.preferences.length" class="flex flex-wrap gap-2">
              <li v-for="item in profile.preferences" :key="item.scope + item.value" class="rounded-full bg-slate-100 px-3 py-1 text-sm">
                {{ destination(item, catalog) }}
              </li>
            </ul>
            <span v-else>{{ t('matchProfile.noDestinations') }}</span>
          </dd>
        </div>
      </dl>

      <section v-if="profile.teachingAreas.length" class="panel space-y-3">
        <h2 class="font-display text-xl font-bold">{{ t('teaching.heading') }}</h2>
        <ul class="divide-y divide-slate-200">
          <li v-for="(area, index) in profile.teachingAreas" :key="index" class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 py-3">
            <p class="min-w-0 wrap-break-word whitespace-pre-line font-semibold">{{ area.title }}</p>
            <p class="text-sm text-slate-600">{{ area.experience ? t(`teaching.levels.${area.experience}`) : t('teaching.unspecified') }}</p>
          </li>
        </ul>
      </section>

      <div class="panel space-y-2">
        <MatchContact v-model:sent="profile.sent" :id="profile.id" :received="profile.received" @error="error = $event" />
        <p v-if="profile.sent && profile.received" class="text-sm text-slate-600">{{ t('matches.emailLater') }}</p>
      </div>
    </template>
  </main>
</template>
