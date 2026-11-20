<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDatabaseStatus } from './database'
import { saveLocale } from './locale'

const { t, locale } = useI18n()
const { database, refresh } = useDatabaseStatus()

watch(locale, saveLocale)
onMounted(refresh)
</script>

<template>
  <div class="mx-auto flex min-h-dvh w-full max-w-5xl flex-col px-6 sm:px-10">
    <header class="flex items-center justify-between gap-3 border-b border-slate-300/60 py-5">
      <a class="rounded text-lg font-semibold text-slate-900" href="/">
        {{ t('app.name') }}
      </a>
      <label class="flex items-center gap-2 text-sm text-slate-600">
        <span class="sr-only sm:not-sr-only">{{ t('language.label') }}</span>
        <select v-model="locale" class="rounded-lg border border-slate-300 bg-white px-2 py-1.5 text-slate-900">
          <option value="en">{{ t('language.en') }}</option>
          <option value="de">{{ t('language.de') }}</option>
        </select>
      </label>
    </header>

    <main class="mx-auto flex w-full max-w-xl flex-1 flex-col justify-center py-16">
      <h1 class="text-5xl leading-tight tracking-tight text-slate-950 sm:text-7xl">
        {{ t('app.greeting') }}
      </h1>
      <section
        class="mt-8 rounded-2xl border border-slate-300/70 bg-white p-6 shadow-sm sm:p-7"
        aria-labelledby="database-heading"
      >
        <h2 id="database-heading" class="text-sm font-medium text-slate-600">
          {{ t('database.label') }}
        </h2>
        <div class="mt-3 min-h-9" role="status">
          <p v-if="database.state === 'ready'" class="flex items-baseline gap-3 text-slate-950">
            <span class="size-2.5 shrink-0 self-center rounded-full bg-emerald-600" aria-hidden="true"></span>
            <span class="font-mono text-xl break-words sm:text-2xl">{{ database.version }}</span>
          </p>
          <p v-else class="text-base text-slate-700">
            {{ t(database.state === 'loading' ? 'database.loading' : 'database.unavailable') }}
          </p>
        </div>
        <button
          v-if="database.state === 'unavailable'"
          type="button"
          class="mt-3 rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium hover:bg-slate-100"
          @click="refresh"
        >
          {{ t('database.retry') }}
        </button>
      </section>
    </main>
  </div>
</template>
