<script setup lang="ts">
defineProps<{ fromLabel: string; toLabel: string; describedBy?: string }>()
const from = defineModel<string>('from', { required: true })
const to = defineModel<string>('to', { required: true })
</script>

<template>
  <div class="grid gap-4">
    <label class="block min-w-0 space-y-1.5 text-sm font-semibold">
      <span>{{ fromLabel }}</span>
      <input v-model="from" class="field text-base font-normal" type="month" :required="Boolean(to)" :aria-describedby="describedBy">
    </label>
    <label class="block min-w-0 space-y-1.5 text-sm font-semibold">
      <span>{{ toLabel }}</span>
      <input v-model="to" class="field text-base font-normal" type="month" :min="from" :required="Boolean(from)" :aria-describedby="describedBy">
    </label>
  </div>
</template>
