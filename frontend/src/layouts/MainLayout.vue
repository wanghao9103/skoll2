<template>
  <div class="layout">
    <aside class="layout-aside">
      <div class="brand">Skoll2 Admin</div>
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
          <span>{{ authStore.username || 'admin' }}</span>
          <el-button type="danger" size="small" @click="onLogout">退出</el-button>
        </div>
      </header>
      <main class="layout-content">
        <router-view />
      </main>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
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

async function loadMenus() {
  try {
    const res = await getMenus()
    menus.value = res.data || []
    applyDynamicRoutes(menus.value)
  } catch (err) {
    ElMessage.error(err.message)
  }
}

function onLogout() {
  authStore.logout()
  router.push('/login')
}

onMounted(loadMenus)
</script>
