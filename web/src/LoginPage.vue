<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { startDemoSession } from './demo-session'

const { t } = useI18n()
const input = ref<HTMLInputElement | null>(null)
const error = ref<'required' | 'invalid' | 'storage' | null>(null)

function continueDemo() {
  const field = input.value
  if (!field) return
  if (!field.validity.valid) {
    error.value = field.validity.valueMissing ? 'required' : 'invalid'
    field.focus()
    return
  }
  try {
    startDemoSession()
  } catch {
    error.value = 'storage'
    return
  }
  window.location.assign('/profile')
}
</script>

<template>
  <main class="flex flex-1 items-center justify-center px-6 py-10 sm:py-16">
    <section class="auth-card" aria-labelledby="login-title">
      <header class="space-y-2">
        <h1 id="login-title" class="font-display text-[1.75rem] leading-tight font-bold">{{ t('login.title') }}</h1>
        <p class="leading-relaxed text-slate-600">{{ t('login.description') }}</p>
      </header>

      <form class="space-y-4" novalidate @submit.prevent="continueDemo">
        <div class="space-y-1.5">
          <label for="email" class="block text-sm font-semibold">{{ t('login.email') }}</label>
          <input
            id="email"
            ref="input"
            class="block min-h-12 w-full rounded-[10px] border border-slate-300 bg-white px-3.5 placeholder:text-slate-500 aria-invalid:border-red-600"
            type="email"
            autocomplete="email"
            autocapitalize="none"
            :spellcheck="false"
            maxlength="254"
            required
            :placeholder="t('login.placeholder')"
            :aria-invalid="error === 'required' || error === 'invalid' ? 'true' : undefined"
            :aria-describedby="error ? 'login-error' : undefined"
            @input="error = null"
          />
          <p v-if="error" id="login-error" class="text-sm text-red-700" role="alert">{{ t(`login.error.${error}`) }}</p>
        </div>
        <button type="submit" class="button button-primary min-h-12 w-full rounded-[10px]">
          {{ t('login.continue') }}
        </button>
      </form>

      <p class="flex items-start gap-2.5 rounded-[10px] bg-indigo-50 px-3.5 py-3 text-sm leading-relaxed text-indigo-900">
        <svg class="mt-0.5 size-[18px] shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
          <circle cx="12" cy="12" r="9" />
          <path d="M12 8v5m0 3h.01" />
        </svg>
        <span>{{ t('login.demo') }}</span>
      </p>
    </section>
  </main>
</template>
