<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SwapIcon from './SwapIcon.vue'

const { t, locale } = useI18n()
const countryNames = computed(() => new Intl.DisplayNames(locale.value, { type: 'region' }))
const overlap = 7
const requested = 12
</script>

<template>
  <figure class="min-w-0">
    <div class="flex flex-col gap-5 rounded-2xl border border-slate-200 bg-white p-5 shadow-[0_12px_32px_rgba(15,23,42,0.08)] sm:p-6">
      <div class="flex items-center justify-between gap-2">
        <h2 class="text-[0.8125rem] font-semibold tracking-[0.06em] text-slate-600 uppercase">
          {{ t('home.example.heading') }}
        </h2>
        <p class="rounded-full bg-indigo-50 px-2.5 py-1 text-center text-[0.8125rem] font-semibold text-indigo-800">
          {{ t('home.example.status') }}
        </p>
      </div>

      <div class="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-3 sm:gap-4">
        <div>
          <p class="leading-snug font-semibold">{{ t('home.example.homeUniversity') }}</p>
          <p class="mt-1 text-sm leading-snug text-slate-600">
            {{ t('home.example.location', { region: t('home.example.homeRegion'), country: countryNames.of('DE') }) }}
          </p>
        </div>
        <SwapIcon class="size-7 text-indigo-600" />
        <div class="text-right">
          <p class="leading-snug font-semibold">{{ t('home.example.partnerUniversity') }}</p>
          <p class="mt-1 text-sm leading-snug text-slate-600">
            {{ t('home.example.location', { region: t('home.example.partnerRegion'), country: countryNames.of('NL') }) }}
          </p>
        </div>
      </div>

      <div class="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-2 text-sm text-slate-600" aria-hidden="true">
        <span>{{ t('home.example.you') }}</span>
        <div class="relative h-2.5 rounded-full bg-slate-100">
          <div class="absolute inset-y-0 left-[30%] w-[44%] rounded-full bg-indigo-200"></div>
          <div class="absolute inset-y-0 left-[30%] w-[26%] rounded-full bg-amber-400"></div>
        </div>
        <span>{{ t('home.example.partner') }}</span>
        <div class="relative h-2.5 rounded-full bg-slate-100">
          <div class="absolute inset-y-0 left-[4%] w-[52%] rounded-full bg-slate-300"></div>
          <div class="absolute inset-y-0 left-[30%] w-[26%] rounded-full bg-amber-400"></div>
        </div>
      </div>

      <div class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 border-t border-slate-200 pt-4">
        <p class="text-sm text-slate-600">{{ t('home.example.overlap') }}</p>
        <p class="font-display text-[1.75rem] leading-none font-bold" aria-hidden="true">
          {{ overlap }}/{{ requested }}
          <span class="text-sm font-medium text-slate-600">{{ t('home.example.months') }}</span>
        </p>
        <span class="sr-only">{{ t('home.example.overlapDescription', { overlap, requested }) }}</span>
      </div>
    </div>
    <figcaption class="mt-3 text-[0.8125rem] text-slate-500">{{ t('home.example.caption') }}</figcaption>
  </figure>
</template>
