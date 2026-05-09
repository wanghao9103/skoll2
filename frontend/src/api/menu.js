import http from './http'

export function getMenus() {
  return http.get('/api/menus')
}
