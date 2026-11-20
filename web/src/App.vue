<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import HomePage from './HomePage.vue'
import LoginPage from './LoginPage.vue'
import MatchesPage from './MatchesPage.vue'
import MatchProfilePage from './MatchProfilePage.vue'
import ProfilePage from './ProfilePage.vue'
import SwapIcon from './SwapIcon.vue'
import { useDatabaseStatus } from './database'
import { saveLocale } from './locale'

defineProps<{ page: 'home' | 'login' | 'profile' | 'matches' | 'matchProfile'; candidateId: string }>()

const { t, locale } = useI18n()
const { database, refresh } = useDatabaseStatus()
const languages = ['de', 'en'] as const

watch(locale, saveLocale)
</script>

<template>
  <div class="flex min-h-dvh flex-col">
    <header class="border-b border-slate-200 bg-white">
      <nav class="mx-auto flex min-h-[4.5rem] max-w-[1160px] items-center justify-between gap-1 px-2 min-[360px]:gap-2 min-[360px]:px-3 sm:px-6" :aria-label="t('home.navigation')">
        <a class="flex min-h-11 shrink-0 items-center gap-2.5 rounded-lg font-display text-base font-bold sm:text-xl" href="/">
          <span class="grid size-8 place-items-center rounded-[10px] bg-indigo-600 text-white sm:size-9">
            <SwapIcon class="size-5" />
          </span>
          {{ t('app.name') }}
        </a>
        <div class="flex items-center gap-2 md:gap-3 lg:gap-5">
          <div v-if="page === 'home'" class="hidden items-center gap-2.5 text-sm font-medium md:flex lg:gap-4">
            <a href="#how-it-works" class="header-link">{{ t('home.howItWorks') }}</a>
            <a href="/login" class="header-link">
              {{ t('home.signIn') }}
            </a>
            <a href="/login" class="button button-primary rounded-[10px] px-4">
              {{ t('home.createProfile') }}
            </a>
          </div>
          <div v-else-if="page !== 'login'" class="hidden items-center gap-4 text-sm font-medium md:flex">
            <a href="/matches" class="header-link aria-[current=page]:text-indigo-800" :aria-current="page === 'matches' ? 'page' : undefined">{{ t('matches.navigation') }}</a>
            <a href="/profile" class="header-link aria-[current=page]:text-indigo-800" :aria-current="page === 'profile' ? 'page' : undefined">{{ t('profile.navigation') }}</a>
          </div>

          <div class="flex rounded-[10px] border border-slate-200" role="group" :aria-label="t('language.label')">
            <button
              v-for="language in languages"
              :key="language"
              type="button"
              :aria-label="t(`language.${language}`)"
              :aria-pressed="locale === language"
              class="min-h-11 min-w-11 rounded-[9px] px-1 text-sm font-semibold aria-pressed:bg-indigo-50 aria-pressed:text-indigo-800 hover:bg-slate-100"
              @click="locale = language"
            >
              {{ t(`language.short.${language}`) }}
            </button>
          </div>

          <a
            v-if="page !== 'login'"
            :href="page === 'home' ? '/login' : '/profile'"
            class="grid size-11 place-items-center rounded-[10px] border border-slate-200 text-slate-700 hover:bg-slate-100 md:hidden"
            :aria-label="t(page === 'home' ? 'home.signIn' : 'profile.navigation')"
          >
            <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" aria-hidden="true">
              <circle cx="12" cy="8" r="3" />
              <path d="M5 20v-1a5 5 0 0 1 5-5h4a5 5 0 0 1 5 5v1" />
            </svg>
          </a>
        </div>
      </nav>
    </header>

    <HomePage v-if="page === 'home'" />
    <LoginPage v-else-if="page === 'login'" />
    <ProfilePage v-else-if="page === 'profile'" />
    <MatchProfilePage v-else-if="page === 'matchProfile'" :candidate-id="candidateId" />
    <MatchesPage v-else />

    <footer v-if="page === 'home'" class="mt-auto">
      <div class="mx-auto flex max-w-[1160px] flex-wrap items-start justify-between gap-x-8 gap-y-3 px-6 py-8 text-sm leading-relaxed text-slate-600">
        <div class="space-y-2">
          <p class="font-semibold text-slate-900">{{ t('app.name') }}</p>
          <i18n-t keypath="attribution.text" scope="global" tag="p">
            <template #geonames>
              <a class="underline underline-offset-2 hover:text-indigo-800" href="https://www.geonames.org/export/">{{ t('attribution.geonames') }}</a>
            </template>
            <template #geoLicense>
              <a class="underline underline-offset-2 hover:text-indigo-800" href="https://creativecommons.org/licenses/by/4.0/">{{ t('attribution.geoLicense') }}</a>
            </template>
            <template #ror>
              <a class="underline underline-offset-2 hover:text-indigo-800" href="https://ror.readme.io/docs/data-dump">{{ t('attribution.ror') }}</a>
            </template>
            <template #rorLicense>
              <a class="underline underline-offset-2 hover:text-indigo-800" href="https://creativecommons.org/publicdomain/zero/1.0/">{{ t('attribution.rorLicense') }}</a>
            </template>
          </i18n-t>
        </div>

        <details class="max-w-full" @toggle.once="refresh">
          <summary class="min-h-11 cursor-pointer rounded-lg py-2 font-medium hover:text-indigo-800">
            {{ t('database.label') }}
          </summary>
          <p role="status" class="wrap-break-word">
            {{ database.state === 'ready' ? database.version : t(`database.${database.state}`) }}
          </p>
          <button v-if="database.state === 'unavailable'" type="button" class="header-link" @click="refresh">
            {{ t('database.retry') }}
          </button>
        </details>
      </div>
    </footer>
  </div>
</template>
