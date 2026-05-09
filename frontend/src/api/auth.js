import http from './http'

export function login(payload) {
  return http.post('/api/auth/login', payload)
}
