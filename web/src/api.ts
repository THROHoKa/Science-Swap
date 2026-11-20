export class APIError extends Error {
  constructor(readonly code: string) {
    super(code)
  }
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch('/api' + path, options)
  if (response.status === 401) window.location.replace('/login')
  if (!response.ok) {
    const details = await response.json().catch(() => ({}))
    throw new APIError(details.error ?? 'unavailable')
  }
  if (response.status === 204) return undefined as T
  return response.json()
}

export function json(method: string, body: unknown): RequestInit {
  return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
}
