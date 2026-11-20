const demoSessionKey = 'science-swap.demo'

export function hasDemoSession(): boolean {
  try {
    return sessionStorage.getItem(demoSessionKey) === 'active'
  } catch {
    return false
  }
}

export function startDemoSession() {
  sessionStorage.setItem(demoSessionKey, 'active')
}

export function endDemoSession() {
  sessionStorage.removeItem(demoSessionKey)
}
