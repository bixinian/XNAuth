<template>
  <section class="account-page">
    <div class="toolbar">
      <div>
        <h2 class="page-title">账号设置</h2>
        <p class="page-subtitle">维护当前系统管理员账号和登录密码。</p>
      </div>
    </div>

    <n-grid cols="1 m:2" responsive="screen" :x-gap="16" :y-gap="16">
      <n-grid-item>
        <n-card class="account-card" :bordered="false">
          <div class="account-head">
            <div class="account-avatar">{{ userInitial }}</div>
            <div>
              <div class="account-name">{{ profileData?.username || "admin" }}</div>
              <n-tag :type="profileData?.status === 1 ? 'success' : 'warning'" round>
                {{ profileData?.status === 1 ? "启用中" : "已禁用" }}
              </n-tag>
            </div>
          </div>

          <div class="info-list">
            <div>
              <span>账号 ID</span>
              <strong>{{ profileData?.id || "-" }}</strong>
            </div>
            <div>
              <span>最后登录</span>
              <strong>{{ formatTime(profileData?.last_login_at) }}</strong>
            </div>
            <div>
              <span>创建时间</span>
              <strong>{{ formatTime(profileData?.created_at) }}</strong>
            </div>
          </div>
        </n-card>
      </n-grid-item>

      <n-grid-item>
        <n-card class="account-card" title="登录安全" :bordered="false">
          <n-form label-placement="top" :show-require-mark="false">
            <n-form-item label="管理员账号">
              <n-input v-model:value="form.username" placeholder="请输入管理员账号" clearable />
            </n-form-item>
            <n-form-item label="当前密码">
              <n-input
                v-model:value="form.currentPassword"
                type="password"
                show-password-on="click"
                placeholder="修改账号或密码时需要输入"
                clearable
              />
            </n-form-item>
            <n-form-item label="新密码">
              <n-input
                v-model:value="form.newPassword"
                type="password"
                show-password-on="click"
                placeholder="留空则不修改密码"
                clearable
              />
            </n-form-item>
            <n-form-item label="确认新密码">
              <n-input
                v-model:value="form.confirmPassword"
                type="password"
                show-password-on="click"
                placeholder="再次输入新密码"
                clearable
              />
            </n-form-item>
            <n-space justify="end">
              <n-button @click="resetForm">重置</n-button>
              <n-button type="primary" :loading="saving" @click="saveProfile">保存</n-button>
            </n-space>
          </n-form>
        </n-card>
      </n-grid-item>
    </n-grid>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { NButton, NCard, NForm, NFormItem, NGrid, NGridItem, NInput, NSpace, NTag, useMessage } from "naive-ui";
import { profile, updateProfile, type AdminProfile } from "@/api/admin";
import { useAuthStore } from "@/stores/auth";

const message = useMessage();
const auth = useAuthStore();
const profileData = ref<AdminProfile | null>(null);
const saving = ref(false);
const form = reactive({
  username: "",
  currentPassword: "",
  newPassword: "",
  confirmPassword: ""
});

const userInitial = computed(() => (profileData.value?.username || "A").slice(0, 1).toUpperCase());

onMounted(loadProfile);

async function loadProfile() {
  profileData.value = await profile();
  resetForm();
}

function resetForm() {
  form.username = profileData.value?.username || "";
  form.currentPassword = "";
  form.newPassword = "";
  form.confirmPassword = "";
}

async function saveProfile() {
  const username = form.username.trim();
  const newPassword = form.newPassword.trim();
  const usernameChanged = username !== (profileData.value?.username || "");
  const passwordChanged = newPassword !== "";

  if (!username) {
    message.warning("请输入管理员账号");
    return;
  }
  if (!usernameChanged && !passwordChanged) {
    message.info("没有需要保存的修改");
    return;
  }
  if (!form.currentPassword) {
    message.warning("请输入当前密码");
    return;
  }
  if (passwordChanged && newPassword.length < 6) {
    message.warning("新密码至少 6 位");
    return;
  }
  if (passwordChanged && newPassword !== form.confirmPassword.trim()) {
    message.warning("两次输入的新密码不一致");
    return;
  }

  saving.value = true;
  try {
    profileData.value = await updateProfile({
      username,
      current_password: form.currentPassword,
      new_password: passwordChanged ? newPassword : undefined
    });
    await auth.loadProfile();
    resetForm();
    message.success("账号设置已保存");
  } catch (error) {
    message.error(error instanceof Error ? error.message : "保存失败");
  } finally {
    saving.value = false;
  }
}

function formatTime(value?: string | null) {
  if (!value) return "-";
  return value.replace("T", " ").replace("+08:00", "").slice(0, 19);
}
</script>

<style scoped>
.account-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.account-card {
  height: 100%;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
  box-shadow: var(--app-shadow-sm);
}

.account-head {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 22px;
}

.account-avatar {
  display: grid;
  width: 52px;
  height: 52px;
  place-items: center;
  border-radius: 8px;
  color: #fff;
  background: linear-gradient(135deg, #2f6bff, #14b879);
  font-size: 22px;
  font-weight: 800;
}

.account-name {
  margin-bottom: 8px;
  color: var(--app-text-strong);
  font-size: 22px;
  font-weight: 800;
}

.info-list {
  display: grid;
  gap: 10px;
}

.info-list > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 0;
  border-top: 1px solid var(--app-border);
}

.info-list span {
  color: var(--app-muted);
  font-size: 13px;
}

.info-list strong {
  color: var(--app-text-strong);
  font-size: 14px;
  text-align: right;
  word-break: break-word;
}

@media (max-width: 760px) {
  .info-list > div {
    align-items: flex-start;
    flex-direction: column;
  }

  .info-list strong {
    text-align: left;
  }
}
</style>
