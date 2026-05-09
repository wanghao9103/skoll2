<template>
  <div class="page-card">
    <h2>{{ title }}</h2>
    <component
      :is="loadedComponent"
      v-if="loadedComponent"
      :plugin-key="pluginKey"
    />
    <div v-else class="remote-fallback">
      <p>远程插件组件加载失败，已回退到占位展示。</p>
      <p>插件标识：{{ pluginKey || '-' }}</p>
      <p>远程入口：{{ frontendEntry || '-' }}</p>
      <p>模块导出：{{ remoteModule || '-' }}</p>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'

const route = useRoute()
const loadedComponent = ref(null)

const pluginKey = computed(() => route.meta.pluginKey || '')
const frontendEntry = computed(() => route.meta.frontendEntry || '')
const remoteModule = computed(() => route.meta.remoteModule || './App')
const title = computed(() => route.meta.menuName || '远程插件页面')

async function loadRemoteComponent() {
  if (!frontendEntry.value) {
    return
  }

  try {
    const remoteUrl = resolveRemoteUrl(frontendEntry.value)
    const mod = await import(/* @vite-ignore */ remoteUrl)

    if (mod && mod.default) {
      loadedComponent.value = mod.default
      return
    }

    if (mod && mod[remoteModule.value]) {
      loadedComponent.value = mod[remoteModule.value]
      return
    }

    ElMessage.warning('远程组件未找到可渲染导出，已使用占位页')
  } catch (err) {
    ElMessage.warning(err.message || '远程组件加载失败，已使用占位页')
  }
}

function resolveRemoteUrl(rawUrl) {
  if (!rawUrl) {
    return ''
  }

  if (/^https?:\/\//i.test(rawUrl)) {
    return rawUrl
  }

  // In Vite dev mode, importing "/public" assets by bare absolute path
  // is treated as source import and can fail. Use a full URL to force runtime fetch.
  return new URL(rawUrl, window.location.origin).href
}

onMounted(loadRemoteComponent)
</script>
