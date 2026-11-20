<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { endDemoSession } from './demo-session'

const { t } = useI18n()

function leaveDemo() {
  try {
    endDemoSession()
  } finally {
    window.location.replace('/')
  }
}
</script>

<template>
  <main class="flex flex-1 items-center justify-center px-6 py-10 sm:py-16">
    <section class="auth-card" aria-labelledby="demo-profile-title">
      <header class="space-y-2">
        <h1 id="demo-profile-title" class="font-display text-[1.75rem] leading-tight font-bold">{{ t('demoProfile.title') }}</h1>
        <p class="leading-relaxed text-slate-600">{{ t('demoProfile.description') }}</p>
      </header>
      <button type="button" class="button button-secondary self-start px-4" @click="leaveDemo">
        {{ t('demoProfile.leave') }}
      </button>
    </section>
  </main>
</template>
