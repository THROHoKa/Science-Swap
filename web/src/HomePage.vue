<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import MatchExample from './MatchExample.vue'

const { t } = useI18n()
const steps = ['profile', 'preferences', 'matches'] as const
</script>

<template>
  <main>
    <section
      class="mx-auto grid max-w-[1160px] items-center gap-10 px-6 pt-12 pb-14 md:grid-cols-2 md:gap-8 md:pt-20 md:pb-16 lg:gap-14 lg:pt-[88px] lg:pb-[72px]"
      aria-labelledby="home-title"
    >
      <div class="flex min-w-0 flex-col items-start gap-6">
        <p class="rounded-full bg-indigo-50 px-3 py-1.5 text-sm font-semibold text-indigo-800">
          {{ t('home.eyebrow') }}
        </p>
        <h1 id="home-title" class="max-w-[18ch] font-display text-[clamp(2.375rem,5vw,3.5rem)] leading-[1.05] font-bold tracking-[-0.02em]">
          {{ t('home.headline') }}
        </h1>
        <p class="max-w-[32rem] text-lg leading-relaxed text-slate-600 lg:text-[1.1875rem]">
          {{ t('home.description') }}
        </p>
        <div class="flex flex-wrap gap-3 pt-2">
          <a href="/login" class="button button-primary px-[22px]">
            {{ t('home.createProfileNow') }}
          </a>
          <a class="button button-secondary px-[22px]" href="#how-it-works">
            {{ t('home.howItWorks') }}
          </a>
        </div>
      </div>

      <MatchExample />
    </section>

    <section id="how-it-works" class="scroll-mt-6 border-y border-slate-200 bg-white" aria-labelledby="how-it-works-title">
      <div class="mx-auto max-w-[1160px] px-6 py-12 md:py-[72px]">
        <h2 id="how-it-works-title" class="font-display text-3xl leading-tight font-bold md:text-4xl">
          {{ t('home.howItWorks') }}
        </h2>
        <ol class="mt-8 grid gap-5 md:mt-10 md:grid-cols-3 lg:gap-6">
          <li v-for="(step, index) in steps" :key="step" class="flex min-w-0 flex-col items-start gap-3 rounded-xl border border-slate-200 p-6">
            <span class="grid size-9 place-items-center rounded-full bg-indigo-50 font-semibold text-indigo-800" aria-hidden="true">
              {{ index + 1 }}
            </span>
            <h3 class="text-lg leading-snug font-semibold">{{ t(`home.steps.${step}.title`) }}</h3>
            <p class="leading-relaxed text-slate-600">{{ t(`home.steps.${step}.description`) }}</p>
          </li>
        </ol>
      </div>
    </section>
  </main>
</template>
