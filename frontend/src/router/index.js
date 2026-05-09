import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

import LoginPage from '../views/LoginPage.vue'
import MainLayout from '../layouts/MainLayout.vue'
import DashboardPage from '../views/DashboardPage.vue'
import PluginManagerPage from '../views/PluginManagerPage.vue'
import PluginPage from '../views/PluginPage.vue'
import RemotePluginPage from '../views/RemotePluginPage.vue'
import ProfilePage from '../views/ProfilePage.vue'

const componentMap = {
  DashboardPage,
  PluginManagerPage,
  PluginPage,
  RemotePluginPage,
  ProfilePage
}

const dynamicRouteNames = new Set()

function toRouteName(path) {
  return `menu-${path.replace(/^\//, '').replace(/\//g, '-')}`
}

function isStaticPath(path) {
  return path === '/dashboard' || path === '/plugins' || path === '/profile'
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
        },
        {
          path: 'profile',
          name: 'profile',
          component: ProfilePage
        }
      ]
    }
  ]
})

export function applyDynamicRoutes(menus = []) {
  const desiredNames = new Set()

  menus.forEach((menu) => {
    const fullPath = menu.path || ''
    if (!fullPath || isStaticPath(fullPath)) {
      return
    }

    const childPath = fullPath.replace(/^\//, '')
    if (!childPath) {
      return
    }

    const routeName = toRouteName(fullPath)
    desiredNames.add(routeName)

    if (router.hasRoute(routeName)) {
      return
    }

    const component = componentMap[menu.component] || PluginPage
    router.addRoute('root', {
      path: childPath,
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
    dynamicRouteNames.add(routeName)
  })

  for (const routeName of [...dynamicRouteNames]) {
    if (desiredNames.has(routeName)) {
      continue
    }
    if (router.hasRoute(routeName)) {
      router.removeRoute(routeName)
    }
    dynamicRouteNames.delete(routeName)
  }

  return [...desiredNames]
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
