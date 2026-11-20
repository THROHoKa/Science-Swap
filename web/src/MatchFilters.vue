<script setup lang="ts">
import { computed, useId } from 'vue'
import { isAnyTime, isValidPeriod, regionOptions, useExchangeLabels } from './exchange'
import type { Catalog, MatchFilter, Profile } from './exchange'
import MonthRangeFields from './MonthRangeFields.vue'

const props = defineProps<{
  profile: Profile
  catalog: Catalog
}>()
const model = defineModel<MatchFilter>({ required: true })
const { t, place, field, regionLabel } = useExchangeLabels()
const availableFields = computed(() => props.catalog.fields.filter(item => !model.value.fields.includes(item.code)))
const anyTime = computed(() => isAnyTime(model.value))
const validPeriod = computed(() => isValidPeriod(model.value))
const periodHint = useId()

function addField(event: Event) {
  const select = event.target as HTMLSelectElement
  model.value = { ...model.value, fields: [...model.value.fields, select.value] }
  select.value = ''
}
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
      <div v-if="model.fields.length" class="flex flex-wrap gap-2">
        <button v-for="code in model.fields" :key="code" type="button"
          class="chip max-w-full gap-2 rounded-lg border-indigo-600 bg-indigo-50 text-left text-indigo-800"
          :aria-label="t('matches.removeField', { field: field(code, catalog) })"
          @click="model = { ...model, fields: model.fields.filter(item => item !== code) }">
          <span>{{ field(code, catalog) }}</span><span aria-hidden="true">×</span>
        </button>
      </div>
      <p v-else class="text-sm text-slate-600">{{ t('matches.allFields') }}</p>
      <select v-if="availableFields.length" class="field text-sm" :aria-label="t('matches.addField')" @change="addField">
        <option value="" disabled selected>{{ t('matches.addField') }}</option>
        <option v-for="item in availableFields" :key="item.code" :value="item.code">{{ field(item.code, catalog) }}</option>
      </select>
    </fieldset>
    <fieldset class="space-y-2">
      <legend class="mb-2 text-sm font-semibold">{{ t('matches.region') }}</legend>
      <div class="flex flex-wrap gap-2">
        <button v-for="region in regionOptions" :key="region" type="button" class="chip"
          :aria-pressed="model.region === region" @click="model = { ...model, region }">{{ regionLabel(region) }}</button>
      </div>
    </fieldset>
    <fieldset class="space-y-2">
      <legend class="mb-2 text-sm font-semibold">{{ t('matches.period') }}</legend>
      <button type="button" class="chip w-full rounded-lg text-left font-semibold"
        :aria-pressed="anyTime" @click="model = { ...model, from: '', to: '' }">
        {{ t('matches.anyTime') }}
      </button>
      <MonthRangeFields :from="model.from" :to="model.to" :from-label="t('matches.from')" :to-label="t('matches.to')"
        :described-by="!anyTime && !validPeriod ? periodHint : undefined"
        @update:from="model = { ...model, from: $event }" @update:to="model = { ...model, to: $event }" />
      <p v-if="!anyTime && !validPeriod" :id="periodHint" class="text-sm text-slate-600" role="status">
        {{ t('common.errors.invalid_period') }}
      </p>
    </fieldset>
    <fieldset v-if="validPeriod" class="space-y-2">
      <legend class="mb-2 text-sm font-semibold">{{ t('matches.minimum') }}</legend>
      <div class="flex flex-wrap gap-2">
        <button v-for="minimum in [1, 3, 6]" :key="minimum" type="button" class="chip flex-1 justify-center whitespace-nowrap"
          :aria-pressed="model.minimum === minimum" @click="model = { ...model, minimum }">{{ t('matches.minimumValue', { months: minimum }) }}</button>
      </div>
    </fieldset>
  </div>
</template>
