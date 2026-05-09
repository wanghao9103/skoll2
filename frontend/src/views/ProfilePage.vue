<template>
  <div class="page-card profile-card">
    <div class="profile-head">
      <el-avatar :size="72" src="https://api.dicebear.com/9.x/thumbs/svg?seed=skoll-admin" />
      <div>
        <h2>个人中心</h2>
        <p>管理员账号信息与偏好设置入口。</p>
      </div>
    </div>

    <el-descriptions :column="1" border>
      <el-descriptions-item label="用户名">{{ username }}</el-descriptions-item>
      <el-descriptions-item label="角色">admin</el-descriptions-item>
      <el-descriptions-item label="邮箱">admin@example.com</el-descriptions-item>
      <el-descriptions-item label="说明">当前为基础演示页面，后续可接入真实用户资料接口。</el-descriptions-item>
    </el-descriptions>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useAuthStore } from '../stores/auth'

const authStore = useAuthStore()
const username = computed(() => authStore.username || 'admin')
</script>
