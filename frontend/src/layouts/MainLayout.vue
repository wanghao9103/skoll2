<template>
  <div class="layout">
    <aside class="layout-aside">
      <div class="brand-wrap">
        <div class="brand-dot"></div>
        <div class="brand">Skoll2 Admin</div>
      </div>
      <el-menu
        :default-active="activePath"
        class="menu"
        router
      >
        <el-menu-item
          v-for="item in menus"
          :key="item.path"
          :index="item.path"
        >
          <span>{{ item.name }}</span>
        </el-menu-item>
      </el-menu>
    </aside>

    <section class="layout-main">
      <header class="layout-header">
        <div class="title">插件化后台管理系统</div>
        <div class="right-tools">
          <el-dropdown trigger="click" @command="onUserCommand">
            <div class="user-chip">
              <el-avatar :size="34" src="https://api.dicebear.com/9.x/thumbs/svg?seed=skoll-admin" />
              <span>{{ authStore.username || 'admin' }}</span>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">个人中心</el-dropdown-item>
                <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>
      <main class="layout-content">
        <router-view />
      </main>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

import { getMenus } from '../api/menu'
import { useAuthStore } from '../stores/auth'
import { applyDynamicRoutes } from '../router'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const menus = ref([])
const activePath = computed(() => route.path)
const pluginChangedEvent = 'plugins:changed'

async function loadMenus() {
  try {
    const res = await getMenus()
    menus.value = res.data || []
    applyDynamicRoutes(menus.value)

    const allowedPaths = new Set(['/dashboard', '/plugins', '/profile'])
    menus.value.forEach((item) => {
      if (item.path) {
        allowedPaths.add(item.path)
      }
    })

    if (!allowedPaths.has(route.path)) {
      router.push('/plugins')
    }
  } catch (err) {
    ElMessage.error(err.message)
  }
}

function onLogout() {
  authStore.logout()
  router.push('/login')
}

function onUserCommand(command) {
  if (command === 'profile') {
    router.push('/profile')
    return
  }
  if (command === 'logout') {
    onLogout()
  }
}

function onPluginChanged() {
  loadMenus()
}

onMounted(() => {
  loadMenus()
  window.addEventListener(pluginChangedEvent, onPluginChanged)
})

onUnmounted(() => {
  window.removeEventListener(pluginChangedEvent, onPluginChanged)
})
</script>
