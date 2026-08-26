<template>
  <main class="login-page">
    <section class="login-card" aria-label="Backplane Admin">
      <header class="login-header">
        <div class="brand-lockup">
          <div class="brand-mark">B</div>
          <span>Backplane Admin</span>
        </div>
        <p>后台管理系统</p>
      </header>

      <n-form
        ref="formRef"
        class="login-form"
        :model="form"
        :rules="rules"
        :show-label="false"
        :show-feedback="false"
        size="large"
        @keyup.enter="handleLogin"
      >
        <n-form-item path="username">
          <n-input v-model:value="form.username" placeholder="用户名" autocomplete="username">
            <template #prefix><n-icon :component="PersonOutline" /></template>
          </n-input>
        </n-form-item>
        <n-form-item path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="密码"
            autocomplete="current-password"
          >
            <template #prefix><n-icon :component="LockClosedOutline" /></template>
          </n-input>
        </n-form-item>
        <n-button type="primary" size="large" block :loading="loading" @click="handleLogin">
          登录
        </n-button>
      </n-form>

      <p v-if="isDev" class="login-hint">默认账号：admin / admin123</p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NIcon, NInput, type FormInst, type FormRules } from 'naive-ui'
import { LockClosedOutline, PersonOutline } from '@vicons/ionicons5'
import { useUserStore } from '@/store/user'
import { message } from '@/utils/ui'

const router = useRouter()
const userStore = useUserStore()
const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const isDev = import.meta.env.DEV
const form = reactive({ username: '', password: '' })
const rules: FormRules = {
  username: { required: true, message: '请输入用户名', trigger: ['input', 'blur'] },
  password: { required: true, message: '请输入密码', trigger: ['input', 'blur'] },
}

async function handleLogin() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  loading.value = true
  try {
    await userStore.login(form.username, form.password)
    message.success('登录成功')
    await router.push('/')
  } catch {
    // The shared interceptor displays request errors.
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 100vh;
  padding: 24px;
  background: #f1f4f6;
}

.login-card {
  width: min(100%, 400px);
  padding: 36px 40px;
  border: 1px solid #e1e6ea;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 12px 32px rgba(16, 24, 40, 0.08);
}

.brand-lockup {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 11px;
  color: #17202a;
  font-size: 24px;
  font-weight: 700;
}

.brand-mark {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 7px;
  background: #14b8a6;
  color: #062c2a;
  font-weight: 800;
}

.login-header {
  margin-bottom: 28px;
  text-align: center;
}

.login-header p {
  margin: 9px 0 0;
  color: #8a949e;
  font-size: 14px;
}

.login-form :deep(.n-form-item) {
  margin-bottom: 16px;
}

.login-form :deep(.n-form-item:last-of-type) {
  margin-bottom: 20px;
}

.login-hint {
  margin: 18px 0 0;
  color: #98a2b3;
  font-size: 12px;
  text-align: center;
}

@media (max-width: 480px) {
  .login-card {
    padding: 32px 24px;
  }
}
</style>
