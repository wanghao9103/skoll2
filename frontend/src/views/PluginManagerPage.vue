<template>
  <div class="page-card">
    <div class="tool-row">
      <el-input v-model="installForm.packageUrl" placeholder="插件包 URL，例如 plugin://sample-hello 或 https://cdn.example.com/livekit.zip" />
      <el-input v-model="installForm.checksum" placeholder="checksum，可选" />
      <el-button type="primary" @click="onInstall">安装插件</el-button>
      <el-button @click="loadData">刷新</el-button>
    </div>

    <el-table :data="plugins" style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="key" label="标识" min-width="140" />
      <el-table-column prop="version" label="版本" width="120" />
      <el-table-column prop="status" label="状态" width="130" />
      <el-table-column prop="apiPrefix" label="API 前缀" min-width="160" />
      <el-table-column label="操作" width="300">
        <template #default="scope">
          <el-space>
            <el-button size="small" @click="openConfig(scope.row.key)">配置</el-button>
            <el-button size="small" type="primary" @click="onUpgrade(scope.row.key)">升级</el-button>
            <el-button size="small" type="success" @click="onEnable(scope.row.key)">启用</el-button>
            <el-button size="small" type="warning" @click="onDisable(scope.row.key)">禁用</el-button>
            <el-button size="small" type="danger" @click="onUninstall(scope.row.key)">卸载</el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="configDialog.visible" width="720px" title="插件配置">
      <div class="config-toolbar">
        <div>插件：{{ configDialog.pluginKey || '-' }}</div>
        <el-button size="small" type="primary" @click="addConfigRow">新增配置</el-button>
      </div>

      <el-table :data="configDialog.rows" style="width: 100%">
        <el-table-column label="配置键" min-width="180">
          <template #default="scope">
            <el-input v-model="scope.row.key" placeholder="例如 livekit.url" />
          </template>
        </el-table-column>
        <el-table-column label="配置值" min-width="240">
          <template #default="scope">
            <el-input v-model="scope.row.value" placeholder="请输入配置值" />
          </template>
        </el-table-column>
        <el-table-column label="敏感" width="90">
          <template #default="scope">
            <el-switch v-model="scope.row.isSecret" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="scope">
            <el-button link type="danger" @click="removeConfigRow(scope.$index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <template #footer>
        <el-button @click="configDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="saveConfig">保存配置</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

import {
  disablePlugin,
  enablePlugin,
  getPluginConfig,
  getPluginList,
  installPlugin,
  savePluginConfig,
  upgradePlugin,
  uninstallPlugin
} from '../api/plugin'

const plugins = ref([])
const installForm = reactive({
  packageUrl: '',
  checksum: ''
})
const configDialog = reactive({
  visible: false,
  pluginKey: '',
  rows: []
})

function notifyPluginChanged(action, pluginKey) {
  window.dispatchEvent(new CustomEvent('plugins:changed', {
    detail: { action, pluginKey }
  }))
}

async function loadData() {
  try {
    const res = await getPluginList()
    plugins.value = res.data || []
  } catch (err) {
    ElMessage.error(err.message)
  }
}

async function onInstall() {
  if (!installForm.packageUrl) {
    ElMessage.warning('请输入插件包 URL')
    return
  }
  try {
    await installPlugin({ ...installForm })
    ElMessage.success('安装成功')
    const pluginKey = inferPluginKey(installForm.packageUrl)
    installForm.packageUrl = ''
    installForm.checksum = ''
    await loadData()
    notifyPluginChanged('install', pluginKey)
  } catch (err) {
    ElMessage.error(err.message)
  }
}

async function onEnable(pluginKey) {
  try {
    await enablePlugin({ pluginKey })
    ElMessage.success('启用成功')
    await loadData()
    notifyPluginChanged('enable', pluginKey)
  } catch (err) {
    ElMessage.error(err.message)
  }
}

async function onUpgrade(pluginKey) {
  try {
    const { value } = await ElMessageBox.prompt('请输入目标版本号，例如 1.1.0', '升级插件', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      inputPattern: /^\d+\.\d+\.\d+$/,
      inputErrorMessage: '版本号格式必须为 x.y.z'
    })

    await upgradePlugin({ pluginKey, targetVersion: value })
    ElMessage.success('升级成功')
    await loadData()
    notifyPluginChanged('upgrade', pluginKey)
  } catch (err) {
    if (err === 'cancel' || err === 'close') {
      return
    }
    ElMessage.error(err.message)
  }
}

async function onDisable(pluginKey) {
  try {
    await disablePlugin({ pluginKey })
    ElMessage.success('禁用成功')
    await loadData()
    notifyPluginChanged('disable', pluginKey)
  } catch (err) {
    ElMessage.error(err.message)
  }
}

async function onUninstall(pluginKey) {
  try {
    await uninstallPlugin({ pluginKey })
    ElMessage.success('卸载成功')
    await loadData()
    notifyPluginChanged('uninstall', pluginKey)
  } catch (err) {
    ElMessage.error(err.message)
  }
}

function addConfigRow() {
  configDialog.rows.push({
    key: '',
    value: '',
    isSecret: false
  })
}

function removeConfigRow(idx) {
  configDialog.rows.splice(idx, 1)
}

async function openConfig(pluginKey) {
  try {
    const res = await getPluginConfig(pluginKey)
    configDialog.pluginKey = pluginKey
    configDialog.rows = (res.data || []).map((item) => ({
      key: item.key,
      value: item.value,
      isSecret: !!item.isSecret
    }))
    configDialog.visible = true
  } catch (err) {
    ElMessage.error(err.message)
  }
}

async function saveConfig() {
  try {
    await savePluginConfig({
      pluginKey: configDialog.pluginKey,
      configs: configDialog.rows
    })
    ElMessage.success('配置保存成功')
    configDialog.visible = false
  } catch (err) {
    ElMessage.error(err.message)
  }
}

function inferPluginKey(packageUrl) {
  if (!packageUrl) {
    return ''
  }
  const segment = packageUrl.split('/').pop() || ''
  return segment.replace(/\.[^/.]+$/, '').toLowerCase()
}

onMounted(loadData)
</script>
