/** Lab console gate aligned with Dashboard login (admin / admin). */
export const LAB_AUTH_KEY = 'aegisflux.labAuth'
export const LAB_AUTH_VALUE = 'admin'

export function readLabAuthenticated(): boolean {
  if (typeof window === 'undefined') return false
  return window.localStorage.getItem(LAB_AUTH_KEY) === LAB_AUTH_VALUE
}
