<template>
  <div class="login-page">
    <el-card class="login-card">
      <template #header>
        <div class="login-title">登录管理平台</div>
      </template>
      <el-form :model="form" @submit.prevent>
        <el-form-item label="账号">
          <el-input v-model="form.username" placeholder="请输入账号" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <el-button type="primary" style="width: 100%" @click="onSubmit">登录</el-button>
      </el-form>
      <div class="hint">默认账号：admin / admin123</div>
    </el-card>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const form = reactive({
  username: 'admin',
  password: 'admin123'
})

async function onSubmit() {
  try {
    await authStore.doLogin({ ...form })
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } catch (err) {
    ElMessage.error(err.message)
  }
}
</script>
