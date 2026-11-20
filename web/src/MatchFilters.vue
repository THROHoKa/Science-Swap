<script setup lang="ts">
import { regionOptions, useExchangeLabels } from './exchange'
import type { Catalog, MatchFilter, Period, Profile } from './exchange'

defineProps<{
  profile: Profile
  catalog: Catalog
  periods: (Period & { key: string })[]
}>()
const model = defineModel<MatchFilter>({ required: true })
const { t, place, field, month } = useExchangeLabels()
</script>

<template>
  <div class="space-y-6">
    <div v-if="profile.institution" class="space-y-1">
      <p class="text-xs font-semibold tracking-[0.06em] text-slate-600 uppercase">{{ t('matches.homeInstitution') }}</p>
      <p class="font-semibold">{{ profile.institution.name }}</p>
      <p class="text-sm text-slate-600">{{ place(profile.institution) }}</p>
    </div>
    <fieldset class="space-y-2">
      <legend class="mb-2 text-sm font-semibold">{{ t('profile.field') }}</legend>
      <button type="button" class="chip rounded-lg text-left" :aria-pressed="!model.allFields" @click="model = { ...model, allFields: !model.allFields }">
        {{ model.allFields ? t('matches.allFields') : field(profile.field, catalog) }}
        <span class="ml-2" aria-hidden="true">{{ model.allFields ? '+' : '×' }}</span>
      </button>
    </fieldset>
    <fieldset class="space-y-2">
      <legend class="mb-2 text-sm font-semibold">{{ t('matches.region') }}</legend>
      <div class="flex flex-wrap gap-2">
        <button v-for="region in regionOptions" :key="region" type="button" class="chip"
          :aria-pressed="model.region === region" @click="model = { ...model, region }">{{ t(`regions.${region}`) }}</button>
      </div>
    </fieldset>
    <fieldset class="space-y-2">
      <legend class="mb-2 text-sm font-semibold">{{ t('matches.period') }}</legend>
      <button v-for="period in periods" :key="period.key" type="button" class="chip flex w-full flex-col items-start rounded-lg p-2 text-left"
        :aria-pressed="model.preset === period.key" @click="model = { ...model, preset: period.key }">
        <span class="font-semibold">{{ t(`periods.${period.key}`) }}</span>
        <span class="text-xs text-slate-600">{{ month(period.from) }} – {{ month(period.to) }}</span>
      </button>
    </fieldset>
    <fieldset class="space-y-2">
      <legend class="mb-2 text-sm font-semibold">{{ t('matches.minimum') }}</legend>
      <div class="flex flex-wrap gap-2">
        <button v-for="minimum in [1, 3, 6]" :key="minimum" type="button" class="chip flex-1 justify-center whitespace-nowrap"
          :aria-pressed="model.minimum === minimum" @click="model = { ...model, minimum }">{{ t('matches.minimumValue', { months: minimum }) }}</button>
      </div>
    </fieldset>
  </div>
</template>
