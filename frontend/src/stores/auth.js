import { defineStore } from 'pinia'
import { login } from '../api/auth'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    username: localStorage.getItem('username') || ''
  }),
  actions: {
    async doLogin(payload) {
      const res = await login(payload)
      this.token = res.data.token
      this.username = payload.username
      localStorage.setItem('token', this.token)
      localStorage.setItem('username', this.username)
    },
    logout() {
      this.token = ''
      this.username = ''
      localStorage.removeItem('token')
      localStorage.removeItem('username')
    }
  }
})
