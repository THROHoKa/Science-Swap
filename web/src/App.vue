<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import HomePage from './HomePage.vue'
import SwapIcon from './SwapIcon.vue'
import { useDatabaseStatus } from './database'
import { saveLocale } from './locale'

const { t, locale } = useI18n()
const { database, refresh } = useDatabaseStatus()
const languages = ['de', 'en'] as const
const entryDialog = ref<HTMLDialogElement | null>(null)
const entryMode = ref<'profile' | 'signIn'>('profile')

watch(locale, saveLocale)

async function openEntry(mode: 'profile' | 'signIn') {
  entryMode.value = mode
  await nextTick()
  entryDialog.value?.showModal()
}
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
          <div class="hidden items-center gap-2.5 text-sm font-medium md:flex lg:gap-4">
            <a href="#how-it-works" class="header-link">{{ t('home.howItWorks') }}</a>
            <button type="button" class="header-link" @click="openEntry('signIn')">
              {{ t('home.signIn') }}
            </button>
            <button type="button" class="button button-primary rounded-[10px] px-4" @click="openEntry('profile')">
              {{ t('home.createProfile') }}
            </button>
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

          <button
            type="button"
            class="grid size-11 place-items-center rounded-[10px] border border-slate-200 text-slate-700 hover:bg-slate-100 md:hidden"
            :aria-label="t('home.signIn')"
            @click="openEntry('signIn')"
          >
            <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" aria-hidden="true">
              <circle cx="12" cy="8" r="3" />
              <path d="M5 20v-1a5 5 0 0 1 5-5h4a5 5 0 0 1 5 5v1" />
            </svg>
          </button>
        </div>
      </nav>
    </header>

    <HomePage @create-profile="openEntry('profile')" />

    <footer class="mt-auto">
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

    <dialog
      ref="entryDialog"
      class="m-auto w-11/12 max-w-md rounded-2xl border border-slate-200 bg-white p-6 text-slate-900 shadow-xl backdrop:bg-slate-950/45"
      aria-labelledby="entry-title"
      aria-describedby="entry-description"
    >
      <h2 id="entry-title" class="font-display text-2xl font-bold">
        {{ t(entryMode === 'profile' ? 'home.createProfile' : 'home.signIn') }}
      </h2>
      <p id="entry-description" class="mt-3 leading-relaxed text-slate-600">{{ t('home.entry.unavailable') }}</p>
      <form method="dialog" class="mt-5 flex justify-end">
        <button type="submit" class="button button-primary px-4">{{ t('home.entry.close') }}</button>
      </form>
    </dialog>
  </div>
</template>
