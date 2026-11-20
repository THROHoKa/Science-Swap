import { ref } from 'vue'

type DatabaseStatus =
  | { state: 'loading' }
  | { state: 'ready'; version: string }
  | { state: 'unavailable' }

export function useDatabaseStatus() {
  const database = ref<DatabaseStatus>({ state: 'loading' })

  async function refresh() {
    database.value = { state: 'loading' }
    try {
      const response = await fetch('/api/version')
      if (!response.ok) throw new Error('Could not fetch PostgreSQL version')
      const result: { version?: unknown } | null = await response.json()
      if (typeof result?.version !== 'string' || !result.version) {
        throw new Error('Invalid PostgreSQL version response')
      }
      database.value = { state: 'ready', version: result.version }
    } catch {
      database.value = { state: 'unavailable' }
    }
  }

  return { database, refresh }
}
