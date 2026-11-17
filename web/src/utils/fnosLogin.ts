const SKIP_KEY = 'fonu.skipFnosLogin'

export function skipFnosAutoLogin(): boolean {
  try {
    return sessionStorage.getItem(SKIP_KEY) === '1'
  } catch {
    return false
  }
}

export function setSkipFnosAutoLogin() {
  try {
    sessionStorage.setItem(SKIP_KEY, '1')
  } catch {
    // sessionStorage can be unavailable in private browsing.
  }
}

export function clearSkipFnosAutoLogin() {
  try {
    sessionStorage.removeItem(SKIP_KEY)
  } catch {
    // sessionStorage can be unavailable in private browsing.
  }
}
