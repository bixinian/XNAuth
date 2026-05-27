<template>
  <section class="dashboard">
    <div class="dashboard-hero">
      <div class="hero-copy">
        <div class="hero-tags">
          <n-tag round :bordered="false" type="success">运行中</n-tag>
          <n-tag round :bordered="false">{{ scopeTypeLabel }}</n-tag>
        </div>
        <h2>{{ currentScopeName }}</h2>
        <p>{{ scopeDescription }}</p>
        <div class="hero-actions">
          <n-button type="primary" @click="go('/licenses')">
            <template #icon>
              <n-icon><Card /></n-icon>
            </template>
            管理卡密
          </n-button>
          <n-button secondary @click="go('/verify-logs')">
            <template #icon>
              <n-icon><DocumentText /></n-icon>
            </template>
            验证日志
          </n-button>
          <n-button secondary :loading="loading" @click="load">
            <template #icon>
              <n-icon><RefreshOutline /></n-icon>
            </template>
            刷新数据
          </n-button>
        </div>
      </div>

      <div class="control-panel">
        <div class="panel-head">
          <span>实时运营控制台</span>
          <strong>{{ refreshText }}</strong>
        </div>
        <div class="control-metrics">
          <div v-for="item in controlMetrics" :key="item.label" class="control-metric" :class="item.tone">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
            <small>{{ item.hint }}</small>
          </div>
        </div>
        <div class="trend-grid">
          <div class="trend-block">
            <div class="trend-head">
              <span>近 7 天验证趋势</span>
              <small>成功 / 失败</small>
            </div>
            <div class="trend-bars">
              <div v-for="item in trendItems" :key="item.date" class="trend-item" :title="`${item.date} 共 ${item.total} 次`">
                <div class="bar-stack">
                  <i class="success" :style="{ height: `${verifyBarHeight(item.success)}%` }"></i>
                  <i class="failed" :style="{ height: `${verifyBarHeight(item.failed)}%` }"></i>
                </div>
                <span>{{ item.label }}</span>
              </div>
            </div>
          </div>

          <div class="trend-block">
            <div class="trend-head">
              <span>近 7 天会话趋势</span>
              <small>新增 / 活跃</small>
            </div>
            <div class="trend-bars">
              <div v-for="item in sessionTrendItems" :key="item.date" class="trend-item" :title="`${item.date} 新增 ${item.created} 个，活跃 ${item.active} 个`">
                <div class="bar-stack">
                  <i class="created" :style="{ height: `${sessionBarHeight(item.created)}%` }"></i>
                  <i class="active" :style="{ height: `${sessionBarHeight(item.active)}%` }"></i>
                </div>
                <span>{{ item.label }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <n-grid cols="1 s:2 m:4" :x-gap="16" :y-gap="16" responsive="screen">
      <n-gi v-for="card in cards" :key="card.label">
        <n-card class="metric-card" :bordered="false">
          <div class="metric-top">
            <span class="metric-icon" :class="card.tone">
              <n-icon><component :is="card.icon" /></n-icon>
            </span>
            <n-tag size="small" round :bordered="false">{{ card.hint }}</n-tag>
          </div>
          <n-statistic :label="card.label" :value="card.value" />
        </n-card>
      </n-gi>
    </n-grid>

    <n-grid cols="1 m:2" :x-gap="16" :y-gap="16" responsive="screen">
      <n-gi>
        <n-card class="work-card" title="高频操作" :bordered="false">
          <div class="action-grid">
            <button v-for="item in quickActions" :key="item.path" class="quick-action" @click="go(item.path)">
              <n-icon><component :is="item.icon" /></n-icon>
              <span>
                <strong>{{ item.title }}</strong>
                <small>{{ item.desc }}</small>
              </span>
            </button>
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card class="work-card" title="客户端接入链路" :bordered="false">
          <ol class="flow-list">
            <li>客户端完成应用级公钥配置，并用加密信封提交授权验证。</li>
            <li>服务端绑定设备公钥，创建 session_token 并按心跳刷新在线状态。</li>
            <li>后台通过验证日志、实时在线、公告版本和数据统计持续运营。</li>
          </ol>
          <n-button class="doc-link" secondary type="primary" @click="go('/integration-docs')">
            <template #icon>
              <n-icon><DocumentText /></n-icon>
            </template>
            查看对接文档
          </n-button>
        </n-card>
      </n-gi>
    </n-grid>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { NButton, NCard, NGi, NGrid, NIcon, NStatistic, NTag, useMessage } from "naive-ui";
import { Apps, BarChart, Card, Clipboard, Desktop, DocumentText, People, RefreshOutline, ShieldCheckmark, Time } from "@vicons/ionicons5";
import { dashboardSummary, type DashboardSessionTrendItem, type DashboardSummary, type DashboardTrendItem } from "@/api/admin";
import { useAppStore } from "@/stores/app";

const appStore = useAppStore();
const message = useMessage();
const router = useRouter();
const loading = ref(false);
const summary = ref<DashboardSummary | null>(null);

const emptyTrend = Array.from({ length: 7 }, (_, index) => {
  const date = new Date();
  date.setDate(date.getDate() - (6 - index));
  const label = `${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  return { date: label, label, total: 0, success: 0, failed: 0, today: index === 6 };
});
const emptySessionTrend = Array.from({ length: 7 }, (_, index) => {
  const date = new Date();
  date.setDate(date.getDate() - (6 - index));
  const label = `${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  return { date: label, label, total: 0, created: 0, active: 0, today: index === 6 };
});

const currentScopeName = computed(() => appStore.currentApp?.app_name || "全部应用");
const scopeTypeLabel = computed(() => (appStore.currentAppId ? "应用视图" : "全局视图"));
const scopeDescription = computed(() =>
  appStore.currentAppId
    ? "聚合当前应用的授权、设备、验证和客户端上报状态，快速判断该应用的运行质量。"
    : "聚合系统内全部应用的授权、设备、验证和客户端上报状态，用于掌握整体运行情况。"
);
const metrics = computed(() => summary.value?.metrics || {
  apps: 0,
  active_apps: 0,
  licenses: 0,
  active_licenses: 0,
  devices: 0,
  online_sessions: 0,
  collect_records: 0,
  verify_today: 0,
  verify_success: 0,
  verify_failed: 0,
  verify_success_rate: 0
});
const trendItems = computed<DashboardTrendItem[]>(() => summary.value?.trend || emptyTrend);
const sessionTrendItems = computed<DashboardSessionTrendItem[]>(() => summary.value?.session_trend || emptySessionTrend);
const maxTrendValue = computed(() => Math.max(1, ...trendItems.value.map((item) => item.total)));
const maxSessionTrendValue = computed(() => Math.max(1, ...sessionTrendItems.value.flatMap((item) => [item.created, item.active])));
const refreshText = computed(() => {
  const generatedAt = summary.value?.scope.generated_at;
  if (!generatedAt) return "等待刷新";
  return `更新于 ${new Date(generatedAt).toLocaleTimeString()}`;
});
const controlMetrics = computed(() => [
  { label: "今日验证", value: formatNumber(metrics.value.verify_today), hint: "客户端授权请求", tone: "blue" },
  { label: "成功率", value: `${metrics.value.verify_success_rate.toFixed(1)}%`, hint: `${metrics.value.verify_success} 次成功`, tone: "green" },
  { label: "失败拦截", value: formatNumber(metrics.value.verify_failed), hint: "今日失败原因可查", tone: "red" },
  { label: "实时在线", value: formatNumber(metrics.value.online_sessions), hint: "基于心跳窗口", tone: "amber" }
]);
const cards = computed(() => [
  { label: "应用数量", value: appStore.currentAppId ? 1 : metrics.value.apps, hint: `${metrics.value.active_apps} 启用`, tone: "indigo", icon: Apps },
  { label: "卡密数量", value: metrics.value.licenses, hint: `${metrics.value.active_licenses} 激活`, tone: "blue", icon: Card },
  { label: "绑定设备", value: metrics.value.devices, hint: "正常设备", tone: "green", icon: Desktop },
  { label: "上报记录", value: metrics.value.collect_records, hint: "客户端数据", tone: "purple", icon: People }
]);

const quickActions = [
  { title: "新增卡密", desc: "生成授权并设置设备限制", path: "/licenses", icon: Card },
  { title: "查看统计", desc: "按收集字段分析客户端数据", path: "/data-stats", icon: BarChart },
  { title: "版本更新", desc: "维护下载地址和更新说明", path: "/versions", icon: Clipboard },
  { title: "安全对接", desc: "查看加密流程和接入案例", path: "/integration-docs", icon: ShieldCheckmark }
];

async function load() {
  loading.value = true;
  try {
    summary.value = await dashboardSummary({
      app_id: appStore.currentAppId || undefined
    });
  } catch (error) {
    message.error(error instanceof Error ? error.message : "加载概览失败");
  } finally {
    loading.value = false;
  }
}

function verifyBarHeight(value: number) {
  if (value <= 0) return 0;
  return Math.max(10, Math.round((value / maxTrendValue.value) * 100));
}

function sessionBarHeight(value: number) {
  if (value <= 0) return 0;
  return Math.max(10, Math.round((value / maxSessionTrendValue.value) * 100));
}

function formatNumber(value: number) {
  return Intl.NumberFormat("zh-CN").format(value);
}

function go(path: string) {
  router.push(path);
}

onMounted(load);
watch(() => appStore.currentAppId, load);
</script>

<style scoped>
.dashboard {
  --dash-hero-border: color-mix(in srgb, var(--app-primary) 18%, var(--app-border));
  --dash-hero-bg:
    radial-gradient(circle at 86% 16%, rgba(47, 107, 255, 0.16), transparent 28%),
    radial-gradient(circle at 16% 12%, rgba(20, 184, 121, 0.12), transparent 30%),
    linear-gradient(135deg, #ffffff 0%, #eef5ff 52%, #f8fbff 100%);
  --dash-hero-shadow: 0 24px 58px rgba(47, 86, 145, 0.14);
  --dash-grid-color: rgba(47, 107, 255, 0.06);
  --dash-grid-mask: linear-gradient(90deg, rgba(0, 0, 0, 0.38), transparent 82%);
  --dash-hero-title: var(--app-text-strong);
  --dash-hero-muted: #52627a;
  --dash-panel-border: rgba(47, 107, 255, 0.14);
  --dash-panel-bg:
    linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(248, 251, 255, 0.76)),
    rgba(255, 255, 255, 0.72);
  --dash-panel-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.84), 0 22px 48px rgba(47, 86, 145, 0.12);
  --dash-panel-title: var(--app-text-strong);
  --dash-panel-muted: #6b7890;
  --dash-metric-bg: rgba(255, 255, 255, 0.74);
  --dash-metric-border: rgba(47, 107, 255, 0.12);
  --dash-metric-label: #6a7890;
  --dash-metric-value: #172033;
  --dash-metric-hint: #7b8798;
  --dash-blob-opacity: 0.12;
  --dash-trend-bg: rgba(255, 255, 255, 0.62);
  --dash-trend-border: rgba(47, 107, 255, 0.12);
  display: grid;
  gap: 16px;
}

:global(:root[data-theme="dark"] .dashboard) {
  --dash-hero-border: color-mix(in srgb, var(--app-primary) 24%, transparent);
  --dash-hero-bg:
    radial-gradient(circle at 86% 16%, rgba(71, 139, 255, 0.4), transparent 26%),
    radial-gradient(circle at 18% 10%, rgba(43, 211, 145, 0.18), transparent 28%),
    linear-gradient(135deg, #0b1730 0%, #12315b 48%, #0b1c35 100%);
  --dash-hero-shadow: 0 26px 70px rgba(0, 0, 0, 0.28);
  --dash-grid-color: rgba(255, 255, 255, 0.05);
  --dash-grid-mask: linear-gradient(90deg, rgba(0, 0, 0, 0.8), transparent 76%);
  --dash-hero-title: #ffffff;
  --dash-hero-muted: rgba(226, 235, 255, 0.78);
  --dash-panel-border: rgba(255, 255, 255, 0.16);
  --dash-panel-bg:
    linear-gradient(180deg, rgba(255, 255, 255, 0.11), rgba(255, 255, 255, 0.055)),
    rgba(8, 19, 39, 0.72);
  --dash-panel-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.1), 0 30px 70px rgba(0, 0, 0, 0.24);
  --dash-panel-title: #ffffff;
  --dash-panel-muted: rgba(226, 235, 255, 0.58);
  --dash-metric-bg: rgba(255, 255, 255, 0.08);
  --dash-metric-border: rgba(255, 255, 255, 0.12);
  --dash-metric-label: rgba(226, 235, 255, 0.62);
  --dash-metric-value: #ffffff;
  --dash-metric-hint: rgba(226, 235, 255, 0.52);
  --dash-blob-opacity: 0.18;
  --dash-trend-bg: rgba(255, 255, 255, 0.075);
  --dash-trend-border: rgba(255, 255, 255, 0.12);
}

.dashboard-hero {
  position: relative;
  overflow: hidden;
  display: grid;
  grid-template-columns: minmax(0, 0.92fr) minmax(360px, 1.08fr);
  gap: 18px;
  padding: 28px;
  border: 1px solid var(--dash-hero-border);
  border-radius: 8px;
  background: var(--dash-hero-bg);
  box-shadow: var(--dash-hero-shadow);
}

.dashboard-hero::before {
  position: absolute;
  inset: 0;
  pointer-events: none;
  content: "";
  background-image:
    linear-gradient(var(--dash-grid-color) 1px, transparent 1px),
    linear-gradient(90deg, var(--dash-grid-color) 1px, transparent 1px);
  background-size: 36px 36px;
  mask-image: var(--dash-grid-mask);
}

.dashboard-hero > * {
  position: relative;
  z-index: 1;
}

.hero-copy {
  align-self: center;
}

.hero-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.hero-copy h2 {
  margin: 14px 0 10px;
  color: var(--dash-hero-title);
  font-size: 32px;
  font-weight: 900;
  line-height: 40px;
}

.hero-copy p {
  max-width: 620px;
  margin: 0;
  color: var(--dash-hero-muted);
  font-size: 14px;
  line-height: 24px;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 22px;
}

.control-panel {
  min-width: 0;
  padding: 18px;
  border: 1px solid var(--dash-panel-border);
  border-radius: 8px;
  background: var(--dash-panel-bg);
  box-shadow: var(--dash-panel-shadow);
  backdrop-filter: blur(18px);
}

.panel-head,
.trend-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.panel-head span,
.trend-head span {
  color: var(--dash-panel-title);
  font-size: 15px;
  font-weight: 800;
}

.panel-head strong,
.trend-head small {
  color: var(--dash-panel-muted);
  font-size: 12px;
  font-weight: 600;
}

.control-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 16px;
}

.control-metric {
  position: relative;
  overflow: hidden;
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--dash-metric-border);
  border-radius: 8px;
  background: var(--dash-metric-bg);
}

.control-metric::after {
  position: absolute;
  right: -18px;
  bottom: -22px;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  content: "";
  opacity: var(--dash-blob-opacity);
}

.control-metric.blue::after {
  background: #67a4ff;
}

.control-metric.green::after {
  background: #2bd391;
}

.control-metric.red::after {
  background: #fb7185;
}

.control-metric.amber::after {
  background: #fbbf24;
}

.control-metrics span,
.control-metrics strong,
.control-metrics small {
  display: block;
}

.control-metrics span {
  color: var(--dash-metric-label);
  font-size: 12px;
}

.control-metrics strong {
  margin-top: 9px;
  color: var(--dash-metric-value);
  font-size: 26px;
  font-weight: 900;
  line-height: 32px;
}

.control-metrics small {
  margin-top: 4px;
  overflow: hidden;
  color: var(--dash-metric-hint);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.trend-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(280px, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.trend-block {
  padding: 14px;
  border: 1px solid var(--dash-trend-border);
  border-radius: 8px;
  background: var(--dash-trend-bg);
}

.trend-bars {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 10px;
  align-items: end;
  min-height: 132px;
  margin-top: 12px;
}

.trend-item {
  display: grid;
  min-width: 0;
  gap: 8px;
  justify-items: center;
}

.bar-stack {
  display: flex;
  width: 100%;
  height: 96px;
  align-items: end;
  justify-content: center;
  gap: 3px;
}

.bar-stack i {
  display: block;
  width: min(18px, 38%);
  min-height: 2px;
  border-radius: 6px 6px 2px 2px;
}

.bar-stack .success {
  background: linear-gradient(180deg, var(--app-primary), var(--app-success));
}

.bar-stack .failed {
  background: linear-gradient(180deg, var(--app-warning), var(--app-danger));
}

.bar-stack .created {
  background: linear-gradient(180deg, #67a4ff, var(--app-primary));
}

.bar-stack .active {
  background: linear-gradient(180deg, var(--app-success), #0ea5e9);
}

.trend-item span {
  color: var(--dash-panel-muted);
  font-size: 11px;
}

.metric-card,
.work-card {
  height: 100%;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
  box-shadow: var(--app-shadow-sm);
}

.metric-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.metric-icon {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border-radius: 8px;
  font-size: 22px;
}

.metric-icon.indigo {
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
  color: var(--app-primary);
}

.metric-icon.blue {
  background: var(--app-primary-soft);
  color: var(--app-primary);
}

.metric-icon.green {
  background: color-mix(in srgb, var(--app-success) 14%, transparent);
  color: var(--app-success);
}

.metric-icon.purple {
  background: rgba(124, 58, 237, 0.14);
  color: #8b5cf6;
}

.action-grid {
  display: grid;
  gap: 10px;
}

.quick-action {
  display: grid;
  grid-template-columns: 36px 1fr;
  gap: 12px;
  align-items: center;
  width: 100%;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
  color: var(--app-text);
  cursor: pointer;
  text-align: left;
}

.quick-action:hover {
  border-color: color-mix(in srgb, var(--app-primary) 42%, var(--app-border));
  background: var(--app-hover);
}

.quick-action .n-icon {
  color: var(--app-primary);
  font-size: 20px;
}

.quick-action strong,
.quick-action small {
  display: block;
}

.quick-action strong {
  color: var(--app-text-strong);
  font-size: 14px;
  line-height: 20px;
}

.quick-action small {
  color: var(--app-muted);
  font-size: 12px;
  line-height: 18px;
}

.flow-list {
  display: grid;
  gap: 12px;
  margin: 0;
  padding-left: 20px;
  color: var(--app-muted);
  font-size: 14px;
  line-height: 24px;
}

.doc-link {
  margin-top: 16px;
  width: 100%;
}

@media (max-width: 1080px) {
  .dashboard-hero {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 680px) {
  .dashboard-hero {
    padding: 20px;
  }

  .hero-copy h2 {
    font-size: 25px;
    line-height: 33px;
  }

  .control-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .trend-grid {
    grid-template-columns: 1fr;
  }

  .trend-bars {
    gap: 7px;
  }
}
</style>
