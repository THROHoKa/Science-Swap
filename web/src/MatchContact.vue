<script setup lang="ts">
import { ref } from 'vue'
import { api, APIError } from './api'
import { useExchangeLabels } from './exchange'

const props = defineProps<{ id: string; received: boolean }>()
const sent = defineModel<boolean>('sent', { required: true })
const emit = defineEmits<{ error: [code: string] }>()
const { t } = useExchangeLabels()
const pending = ref(false)

async function requestContact() {
  if (pending.value) return
  pending.value = true
  emit('error', '')
  try {
    await api('/interests/' + props.id, { method: 'PUT' })
    sent.value = true
  } catch (problem) {
    emit('error', problem instanceof APIError ? problem.code : 'unavailable')
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <p v-if="sent && received" class="text-sm font-semibold text-indigo-800">{{ t('matches.mutual') }}</p>
  <p v-else-if="sent" class="text-sm font-semibold text-indigo-800">{{ t('matches.sent') }}</p>
  <button v-else type="button" class="relative z-10 button button-primary w-full px-4 disabled:opacity-50 sm:w-auto"
    :disabled="pending" @click="requestContact">
    {{ t(received ? 'matches.accept' : 'matches.request') }}
  </button>
</template>
