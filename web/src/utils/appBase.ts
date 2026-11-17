/** Public path registered with the fnOS unified gateway. */
export const FNOS_GATEWAY_PREFIX = '/app/fonu'

export function appBasePath(): string {
  if (typeof window === 'undefined') return ''
  const path = window.location.pathname
  if (path === FNOS_GATEWAY_PREFIX || path.startsWith(`${FNOS_GATEWAY_PREFIX}/`)) {
    return FNOS_GATEWAY_PREFIX
  }
  return ''
}
