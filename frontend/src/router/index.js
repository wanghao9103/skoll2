import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

import LoginPage from '../views/LoginPage.vue'
import MainLayout from '../layouts/MainLayout.vue'
import DashboardPage from '../views/DashboardPage.vue'
import PluginManagerPage from '../views/PluginManagerPage.vue'
import PluginPage from '../views/PluginPage.vue'
import RemotePluginPage from '../views/RemotePluginPage.vue'

const componentMap = {
  DashboardPage,
  PluginManagerPage,
  PluginPage,
  RemotePluginPage
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginPage,
      meta: { public: true }
    },
    {
      path: '/',
      name: 'root',
      component: MainLayout,
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: DashboardPage
        },
        {
          path: 'plugins',
          name: 'plugins',
          component: PluginManagerPage
        }
      ]
    }
  ]
})

export function applyDynamicRoutes(menus = []) {
  const added = []
  menus.forEach((menu) => {
    const path = (menu.path || '').replace(/^\//, '')
    if (!path || path === 'dashboard' || path === 'plugins') {
      return
    }

    const routeName = `menu-${path.replace(/\//g, '-')}`
    if (router.hasRoute(routeName)) {
      return
    }

    const component = componentMap[menu.component] || PluginPage
    router.addRoute('root', {
      path,
      name: routeName,
      component,
      meta: {
        menuName: menu.name,
        rawPath: menu.path,
        pluginKey: menu.pluginKey,
        frontendEntry: menu.frontendEntry,
        remoteModule: menu.remoteModule
      }
    })
    added.push(routeName)
  })
  return added
}

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  if (to.meta.public) {
    next()
    return
  }

  if (!authStore.token) {
    next('/login')
    return
  }

  next()
})

export default router
