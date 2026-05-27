<template>
  <section class="settings-page">
    <div class="toolbar">
      <div>
        <h2 class="page-title">系统设置</h2>
        <p class="page-subtitle">管理登录安全、数据清理和系统运行状态。</p>
      </div>
    </div>

    <n-grid cols="1 l:2" responsive="screen" :x-gap="16" :y-gap="16">
      <n-grid-item>
        <n-card class="settings-card compact-card" :bordered="false">
          <template #header>
            <div class="card-title">
              <span>登录安全</span>
              <n-tag :type="security.login_captcha_enabled ? 'success' : 'default'" round>
                {{ security.login_captcha_enabled ? "已启用" : "未启用" }}
              </n-tag>
            </div>
          </template>
          <div class="security-setting">
            <div class="setting-icon">验</div>
            <div class="setting-copy">
              <strong>登录验证码</strong>
              <p>登录页增加滑块安全验证，减少账号密码被连续尝试的风险。</p>
            </div>
            <div class="setting-actions">
              <n-switch v-model:value="security.login_captcha_enabled" size="large" />
              <n-button type="primary" :loading="savingSecurity" @click="saveSecurity">保存设置</n-button>
            </div>
          </div>
        </n-card>
      </n-grid-item>

      <n-grid-item>
        <n-card class="settings-card compact-card" title="数据管理" :bordered="false">
          <n-form label-placement="top" :show-require-mark="false">
            <div class="cleanup-form">
              <n-form-item label="清理对象">
                <n-select v-model:value="cleanup.target" :options="cleanupTargetOptions" />
              </n-form-item>
              <n-form-item label="清理时间">
                <n-date-picker
                  v-model:value="cleanup.before"
                  type="datetime"
                  clearable
                  class="full-control"
                  placeholder="删除此时间之前的数据"
                />
              </n-form-item>
            </div>
            <div class="cleanup-action">
              <n-popconfirm positive-text="确认清理" negative-text="取消" @positive-click="runCleanup">
                <template #trigger>
                  <n-button type="warning" :loading="cleaning" :disabled="!cleanup.before">清理历史数据</n-button>
                </template>
                将删除所选时间之前的数据，操作不可恢复，确定继续吗？
              </n-popconfirm>
              <span v-if="lastCleanup">
                已清理 {{ targetLabel(lastCleanup.target) }} {{ lastCleanup.deleted_records }} 条
              </span>
            </div>
          </n-form>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-card class="settings-card site-card" :bordered="false">
      <template #header>
        <div class="card-title">
          <span>网站首页配置</span>
          <n-tag round :bordered="false">公开展示</n-tag>
        </div>
      </template>
      <n-form label-placement="top" :show-require-mark="false">
        <div class="site-form">
          <n-form-item label="网站名称">
            <n-input v-model:value="site.site_name" maxlength="40" show-count placeholder="例如 XNAuth 汐念验证" />
          </n-form-item>
          <n-form-item label="ICP备案号">
            <n-input v-model:value="site.icp_number" maxlength="80" show-count placeholder="例如 粤ICP备xxxxxxxx号" />
          </n-form-item>
        </div>
        <div class="footer-link-editor">
          <div class="editor-head">
            <div>
              <strong>首页底部快捷入口</strong>
              <span>最多 6 个，可填写站内路径或完整 URL。</span>
            </div>
            <n-button secondary type="primary" :disabled="site.footer_links.length >= 6" @click="addFooterLink">新增入口</n-button>
          </div>
          <div class="footer-link-list">
            <div v-for="(item, index) in site.footer_links" :key="index" class="footer-link-row">
              <n-input v-model:value="item.label" maxlength="24" placeholder="入口名称" />
              <n-input v-model:value="item.url" maxlength="200" placeholder="/login 或 https://example.com" />
              <n-button secondary type="error" @click="removeFooterLink(index)">删除</n-button>
            </div>
          </div>
        </div>
        <div class="site-actions">
          <n-button type="primary" :loading="savingSite" @click="saveSite">保存网站配置</n-button>
        </div>
      </n-form>
    </n-card>

    <n-card class="settings-card status-card" :bordered="false">
      <template #header>
        <div class="card-title">
          <span>系统性能状态</span>
          <n-tag :type="statusData?.database.status === 'ok' ? 'success' : 'error'" round>
            {{ statusData?.database.status === "ok" ? "运行正常" : "数据库异常" }}
          </n-tag>
        </div>
      </template>
      <template #header-extra>
        <n-button size="small" secondary :loading="statusLoading" @click="loadStatus">刷新</n-button>
      </template>

      <n-grid cols="1 m:2 l:4" responsive="screen" :x-gap="12" :y-gap="12">
        <n-grid-item v-for="item in statusItems" :key="item.label">
          <div class="metric-box">
            <span class="metric-label">{{ item.label }}</span>
            <strong class="metric-value">{{ item.value }}</strong>
            <span class="metric-hint">{{ item.hint }}</span>
          </div>
        </n-grid-item>
      </n-grid>
    </n-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import {
  NButton,
  NCard,
  NDatePicker,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NInput,
  NPopconfirm,
  NSelect,
  NSwitch,
  NTag,
  useMessage
} from "naive-ui";
import {
  cleanupData,
  securitySettings,
  siteSettings,
  systemStatus,
  updateSecuritySettings,
  updateSiteSettings,
  type CleanupResp,
  type SiteSettings,
  type SystemStatus
} from "@/api/admin";

const message = useMessage();
const savingSecurity = ref(false);
const savingSite = ref(false);
const cleaning = ref(false);
const statusLoading = ref(false);
const lastCleanup = ref<CleanupResp | null>(null);
const statusData = ref<SystemStatus | null>(null);
const security = reactive({
  login_captcha_enabled: false
});
const site = reactive<SiteSettings>({
  site_name: "XNAuth 汐念验证",
  icp_number: "",
  footer_links: []
});
const cleanup = reactive({
  target: "collect_records",
  before: null as number | null
});
const cleanupTargetOptions = [
  { label: "数据上报日志", value: "collect_records" },
  { label: "操作日志", value: "operation_logs" }
];

const statusItems = computed(() => {
  const status = statusData.value;
  if (!status) {
    return [
      { label: "运行时长", value: "-", hint: "服务启动后累计" },
      { label: "内存占用", value: "-", hint: "当前分配内存" },
      { label: "协程数量", value: "-", hint: "Go runtime" },
      { label: "数据库连接", value: "-", hint: "open / in use" }
    ];
  }
  return [
    { label: "运行时长", value: formatUptime(status.uptime_seconds), hint: "服务启动后累计" },
    { label: "内存占用", value: `${formatMB(status.memory.alloc_mb)} MB`, hint: `系统 ${formatMB(status.memory.sys_mb)} MB` },
    { label: "协程数量", value: `${status.goroutines}`, hint: `GC ${status.memory.gc_count} 次` },
    {
      label: "数据库连接",
      value: `${status.database.stats.open_connections} / ${status.database.stats.in_use}`,
      hint: `空闲 ${status.database.stats.idle}`
    }
  ];
});

onMounted(async () => {
  await Promise.all([loadSecurity(), loadSite(), loadStatus()]);
});

async function loadSecurity() {
  const data = await securitySettings();
  security.login_captcha_enabled = data.login_captcha_enabled;
}

async function loadSite() {
  const data = await siteSettings();
  site.site_name = data.site_name;
  site.icp_number = data.icp_number;
  site.footer_links = [...data.footer_links];
}

async function loadStatus() {
  statusLoading.value = true;
  try {
    statusData.value = await systemStatus();
  } catch (error) {
    message.error(error instanceof Error ? error.message : "状态获取失败");
  } finally {
    statusLoading.value = false;
  }
}

async function saveSecurity() {
  savingSecurity.value = true;
  try {
    const data = await updateSecuritySettings({
      login_captcha_enabled: security.login_captcha_enabled
    });
    security.login_captcha_enabled = data.login_captcha_enabled;
    message.success("安全设置已保存");
  } catch (error) {
    message.error(error instanceof Error ? error.message : "保存失败");
  } finally {
    savingSecurity.value = false;
  }
}

async function saveSite() {
  savingSite.value = true;
  try {
    const data = await updateSiteSettings({
      site_name: site.site_name.trim(),
      icp_number: site.icp_number.trim(),
      footer_links: site.footer_links
        .map((item) => ({ label: item.label.trim(), url: item.url.trim() }))
        .filter((item) => item.label && item.url)
    });
    site.site_name = data.site_name;
    site.icp_number = data.icp_number;
    site.footer_links = [...data.footer_links];
    message.success("网站配置已保存");
  } catch (error) {
    message.error(error instanceof Error ? error.message : "保存失败");
  } finally {
    savingSite.value = false;
  }
}

function addFooterLink() {
  if (site.footer_links.length >= 6) return;
  site.footer_links.push({ label: "", url: "" });
}

function removeFooterLink(index: number) {
  site.footer_links.splice(index, 1);
}

async function runCleanup() {
  if (!cleanup.before) {
    message.warning("请选择清理时间");
    return;
  }
  cleaning.value = true;
  try {
    lastCleanup.value = await cleanupData({
      target: cleanup.target,
      before: new Date(cleanup.before).toISOString()
    });
    message.success(`已清理 ${lastCleanup.value.deleted_records} 条数据`);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "清理失败");
  } finally {
    cleaning.value = false;
  }
}

function targetLabel(value: string) {
  return cleanupTargetOptions.find((item) => item.value === value)?.label || value;
}

function formatUptime(seconds: number) {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}天 ${hours}小时`;
  if (hours > 0) return `${hours}小时 ${minutes}分`;
  return `${minutes}分`;
}

function formatMB(value: number) {
  return value.toFixed(value >= 10 ? 1 : 2);
}
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-card {
  height: 100%;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
  box-shadow: var(--app-shadow-sm);
}

.compact-card :deep(.n-card__content) {
  padding-top: 8px;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: space-between;
}

.security-setting {
  display: grid;
  grid-template-columns: 46px minmax(0, 1fr) auto;
  align-items: center;
  gap: 14px;
  padding: 4px 0;
}

.setting-icon {
  display: grid;
  width: 46px;
  height: 46px;
  place-items: center;
  border-radius: 8px;
  background: var(--app-primary-soft);
  color: var(--app-primary);
  font-size: 18px;
  font-weight: 800;
}

.setting-copy {
  min-width: 0;
}

.setting-copy strong {
  display: block;
  color: var(--app-text-strong);
  font-size: 16px;
  line-height: 22px;
}

.setting-copy p {
  margin: 5px 0 0;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 20px;
}

.setting-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.cleanup-form {
  display: grid;
  grid-template-columns: minmax(160px, 0.9fr) minmax(220px, 1.1fr);
  gap: 12px;
}

.cleanup-action {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.cleanup-action span {
  color: var(--app-muted);
  font-size: 13px;
}

.site-card :deep(.n-card__content) {
  padding-top: 10px;
}

.site-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.footer-link-editor {
  display: grid;
  gap: 12px;
  margin-top: 2px;
}

.editor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.editor-head strong,
.editor-head span {
  display: block;
}

.editor-head strong {
  color: var(--app-text-strong);
  font-size: 15px;
  line-height: 22px;
}

.editor-head span {
  margin-top: 2px;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 18px;
}

.footer-link-list {
  display: grid;
  gap: 10px;
}

.footer-link-row {
  display: grid;
  grid-template-columns: minmax(140px, 0.45fr) minmax(220px, 1fr) auto;
  gap: 10px;
  align-items: center;
}

.site-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.full-control {
  width: 100%;
}

.metric-box {
  display: flex;
  min-height: 116px;
  flex-direction: column;
  justify-content: space-between;
  gap: 10px;
  padding: 16px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--app-primary) 4%, transparent), transparent),
    var(--app-surface-2);
}

.metric-label {
  color: var(--app-muted);
  font-size: 13px;
  line-height: 18px;
}

.metric-value {
  color: var(--app-text-strong);
  font-size: 24px;
  font-weight: 800;
  line-height: 32px;
  word-break: keep-all;
}

.metric-hint {
  color: var(--app-muted);
  font-size: 12px;
  line-height: 18px;
}

@media (max-width: 760px) {
  .security-setting,
  .cleanup-form,
  .site-form,
  .footer-link-row {
    grid-template-columns: 1fr;
  }

  .editor-head {
    align-items: stretch;
    flex-direction: column;
  }

  .site-actions {
    justify-content: flex-start;
  }

  .setting-icon {
    display: none;
  }

  .setting-actions {
    display: grid;
    grid-template-columns: auto 1fr;
    width: 100%;
  }

  .setting-actions .n-button {
    justify-self: start;
  }

  .metric-box {
    min-height: 94px;
  }
}
</style>
