<template>
  <n-layout class="admin-shell" :has-sider="!isMobile">
    <n-layout-sider
      v-if="!isMobile"
      class="sidebar"
      collapse-mode="width"
      :collapsed-width="76"
      :width="260"
      :collapsed="collapsed"
      :native-scrollbar="false"
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="brand" :class="{ compact: collapsed }">
        <div class="brand-mark">XN</div>
        <div v-if="!collapsed" class="brand-copy">
          <strong>XNAuth</strong>
          <span>汐念验证</span>
        </div>
      </div>
      <n-menu
        class="side-menu"
        :collapsed="collapsed"
        :collapsed-width="76"
        :options="menuOptions"
        :value="activeKey"
        @update:value="handleMenu"
      />
    </n-layout-sider>

    <n-drawer v-model:show="mobileMenuVisible" placement="left" :width="292">
      <n-drawer-content :native-scrollbar="false" closable>
        <template #header>
          <div class="drawer-brand">
            <div class="brand-mark">XN</div>
            <div>
              <strong>XNAuth</strong>
              <span>汐念验证</span>
            </div>
          </div>
        </template>
        <n-menu :options="menuOptions" :value="activeKey" @update:value="handleMenu" />
      </n-drawer-content>
    </n-drawer>

    <n-layout class="main-layout">
      <n-layout-header class="topbar">
        <div class="topbar-main">
          <n-button v-if="isMobile" circle secondary class="icon-button" @click="mobileMenuVisible = true">
            <template #icon>
              <n-icon><MenuOutline /></n-icon>
            </template>
          </n-button>
          <div v-if="!isMobile" class="topbar-context">
            <n-icon><ShieldCheckmarkOutline /></n-icon>
            <span>管理后台</span>
          </div>
        </div>

        <div class="topbar-tools">
          <n-button circle secondary class="icon-button theme-icon-button" :title="themeButtonTitle" @click="cycleTheme">
            <template #icon>
              <n-icon><component :is="themeIcon" /></n-icon>
            </template>
          </n-button>
          <n-dropdown :options="userOptions" @select="handleUserAction">
            <n-button secondary class="user-button">
              <span class="avatar">{{ userInitial }}</span>
              <span class="user-name">{{ auth.user?.username || "管理员" }}</span>
            </n-button>
          </n-dropdown>
        </div>

        <div class="topbar-controls">
          <n-dropdown trigger="click" :options="appScopeOptions" @select="onAppChange">
            <n-button secondary class="scope-button" :title="scopeButtonTitle">
              <template #icon>
                <n-icon><Apps /></n-icon>
              </template>
              <span class="scope-text">{{ currentScopeLabel }}</span>
              <n-icon class="scope-caret"><ChevronDownOutline /></n-icon>
            </n-button>
          </n-dropdown>
        </div>
      </n-layout-header>

      <n-layout-content class="content">
        <div class="content-inner">
          <router-view />
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { computed, h, onBeforeUnmount, onMounted, ref } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import {
  NButton,
  NDrawer,
  NDrawerContent,
  NDropdown,
  NIcon,
  NLayout,
  NLayoutContent,
  NLayoutHeader,
  NLayoutSider,
  NMenu,
  type MenuOption
} from "naive-ui";
import {
  Apps,
  BarChart,
  Card,
  ChevronDownOutline,
  Clipboard,
  CloudUpload,
  DocumentText,
  Home,
  LogOutOutline,
  Megaphone,
  MenuOutline,
  MoonOutline,
  People,
  Settings,
  ShieldCheckmarkOutline,
  SunnyOutline,
  DesktopOutline
} from "@vicons/ionicons5";
import { useAuthStore } from "@/stores/auth";
import { useAppStore } from "@/stores/app";
import { useThemeStore, type ThemeMode } from "@/stores/theme";

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();
const appStore = useAppStore();
const themeStore = useThemeStore();
const collapsed = ref(false);
const mobileMenuVisible = ref(false);
const isMobile = ref(false);

const navItems = [
  { label: "首页概览", key: "dashboard", to: "/dashboard", icon: Home, description: "授权、设备、在线和上报数据总览" },
  { label: "应用管理", key: "apps", to: "/apps", icon: Apps, description: "维护应用标识、名称和启用状态" },
  { label: "卡密管理", key: "licenses", to: "/licenses", icon: Card, description: "创建卡密并管理状态、设备上限和在线上限" },
  { label: "公告管理", key: "announcements", to: "/announcements", icon: Megaphone, description: "配置客户端公告和弹窗策略" },
  { label: "版本管理", key: "versions", to: "/versions", icon: CloudUpload, description: "配置版本发布、最低支持版本和强更" },
  { label: "收集字段", key: "collect-fields", to: "/collect-fields", icon: Clipboard, description: "配置客户端上报字段和列表展示" },
  { label: "数据统计", key: "data-stats", to: "/data-stats", icon: BarChart, description: "按设备来源查看上报字段统计图" },
  { label: "收集记录", key: "collect-records", to: "/collect-records", icon: People, description: "查看客户端信息上报历史" },
  { label: "验证日志", key: "verify-logs", to: "/verify-logs", icon: DocumentText, description: "追踪授权验证成功和失败原因" },
  { label: "操作日志", key: "operation-logs", to: "/operation-logs", icon: BarChart, description: "审计后台管理操作" },
  { label: "对接文档", key: "integration-docs", to: "/integration-docs", icon: DocumentText, description: "查看客户端安全接入和接口动作说明" },
  { label: "系统设置", key: "settings", to: "/settings", icon: Settings, description: "查看服务配置和客户端接入提示" }
];

function icon(component: unknown) {
  return () => h(NIcon, { size: 18 }, { default: () => h(component as never) });
}

const menuOptions = computed<MenuOption[]>(() =>
  navItems.map((item) => ({
    label: () => h(RouterLink, { to: item.to }, { default: () => item.label }),
    key: item.key,
    icon: icon(item.icon)
  }))
);

const routeNameMap: Record<string, string> = {
  "license-detail": "licenses",
  "collect-record-detail": "collect-records"
};

const activeKey = computed(() => routeNameMap[String(route.name)] || String(route.name || "dashboard"));
const appScopeOptions = computed(() => [
  { label: "全部应用", key: 0, icon: icon(Apps) },
  ...appStore.apps.map((item) => ({ label: `${item.app_name} #${item.id}`, key: item.id, icon: icon(Apps) }))
]);
const currentScopeLabel = computed(() => appStore.currentApp?.app_name || "全部应用");
const scopeButtonTitle = computed(() => `当前统计范围：${currentScopeLabel.value}`);
const userInitial = computed(() => (auth.user?.username || "A").slice(0, 1).toUpperCase());
const userOptions = [
  {
    label: "账号设置",
    key: "users",
    icon: icon(People)
  },
  {
    label: "退出登录",
    key: "logout",
    icon: icon(LogOutOutline)
  }
];

const themeIcon = computed(() => {
  if (themeStore.mode === "light") return SunnyOutline;
  if (themeStore.mode === "dark") return MoonOutline;
  return DesktopOutline;
});
const themeButtonTitle = computed(() => {
  if (themeStore.mode === "light") return "当前亮色主题，点击切换暗色";
  if (themeStore.mode === "dark") return "当前暗色主题，点击跟随系统";
  return "当前跟随系统主题，点击切换亮色";
});

function handleMenu(key: string) {
  mobileMenuVisible.value = false;
  router.push({ name: key });
}

function handleUserAction(key: string) {
  if (key === "users") router.push({ name: "users" });
  if (key === "logout") auth.logout();
}

function onAppChange(value: string | number) {
  appStore.setCurrentApp(Number(value) || 0);
}

function cycleTheme() {
  const next: Record<ThemeMode, ThemeMode> = {
    light: "dark",
    dark: "system",
    system: "light"
  };
  themeStore.setMode(next[themeStore.mode]);
}

let mediaQuery: MediaQueryList | null = null;
function syncMobile() {
  isMobile.value = mediaQuery?.matches || false;
  if (!isMobile.value) mobileMenuVisible.value = false;
}

onMounted(async () => {
  mediaQuery = window.matchMedia("(max-width: 980px)");
  syncMobile();
  mediaQuery.addEventListener("change", syncMobile);
  await Promise.all([auth.loadProfile(), appStore.loadApps()]);
});

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener("change", syncMobile);
});
</script>

<style scoped>
.admin-shell {
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--app-bg);
  overflow-x: hidden;
  touch-action: pan-y;
}

.sidebar {
  border-right: 1px solid var(--app-border);
  background: var(--app-sidebar-bg);
  box-shadow: var(--sidebar-shadow);
}

.brand,
.drawer-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand {
  height: 72px;
  padding: 0 18px;
}

.brand.compact {
  justify-content: center;
  padding: 0;
}

.brand-mark {
  display: grid;
  width: 40px;
  height: 40px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 8px;
  background: linear-gradient(135deg, #1e2a44, #2f6bff);
  color: #ffffff;
  font-size: 13px;
  font-weight: 800;
  box-shadow: 0 12px 24px rgba(47, 107, 255, 0.24);
}

.brand-copy strong,
.drawer-brand strong {
  display: block;
  color: var(--app-text);
  font-size: 17px;
  line-height: 22px;
}

.brand-copy span,
.drawer-brand span {
  display: block;
  color: var(--app-muted);
  font-size: 12px;
  line-height: 18px;
}

.side-menu {
  padding: 8px 12px 20px;
}

.main-layout {
  min-width: 0;
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--app-bg);
  overflow-x: hidden;
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  display: grid;
  min-height: 64px;
  grid-template-columns: auto minmax(0, 1fr) auto;
  grid-template-areas: "main controls tools";
  align-items: center;
  gap: 10px;
  padding: 10px 24px;
  border-bottom: 1px solid var(--app-border);
  background: color-mix(in srgb, var(--app-surface) 88%, transparent);
  backdrop-filter: blur(18px);
}

.topbar-main {
  grid-area: main;
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 14px;
}

.topbar-controls {
  grid-area: controls;
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-start;
  justify-self: start;
  gap: 10px;
}

.topbar-tools {
  grid-area: tools;
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.topbar-context {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  padding: 0 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
  color: var(--app-muted);
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
}

.icon-button {
  flex: 0 0 auto;
}

.scope-button {
  max-width: min(30vw, 240px);
  min-width: 118px;
  justify-content: flex-start;
  padding: 0 11px;
}

.scope-text {
  display: block;
  min-width: 0;
  max-width: 142px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scope-caret {
  margin-left: auto;
  color: var(--app-muted);
  font-size: 15px;
}

.theme-icon-button {
  color: var(--app-primary);
}

.user-button {
  min-width: 104px;
}

.avatar {
  display: inline-grid;
  width: 22px;
  height: 22px;
  margin-right: 6px;
  place-items: center;
  border-radius: 50%;
  background: var(--app-primary-soft);
  color: var(--app-primary);
  font-size: 12px;
  font-weight: 800;
}

.content {
  min-height: calc(100vh - 64px);
  min-height: calc(100dvh - 64px);
  background:
    linear-gradient(180deg, var(--app-bg-accent), transparent 280px),
    var(--app-bg);
  overflow-x: hidden;
  overscroll-behavior-y: auto;
  touch-action: pan-y;
}

.content-inner {
  width: min(100%, 1440px);
  margin: 0 auto;
  padding: 24px;
}

@media (max-width: 1180px) {
  .topbar {
    grid-template-columns: auto minmax(0, 1fr) auto;
    grid-template-areas: "main controls tools";
  }
}

@media (max-width: 760px) {
  .topbar {
    min-height: 56px;
    gap: 6px;
    padding: 8px 10px;
  }

  .topbar-main {
    gap: 6px;
  }

  .topbar-controls {
    width: auto;
    justify-content: flex-start;
    justify-self: start;
    gap: 6px;
  }

  .topbar-tools {
    gap: 6px;
  }

  .scope-button {
    width: min(36vw, 156px);
    max-width: none;
    min-width: 0;
  }

  .scope-text {
    max-width: min(22vw, 92px);
  }

  .user-button {
    width: 38px;
    min-width: 38px;
    padding: 0;
    justify-content: center;
  }

  .user-name {
    display: none;
  }

  .avatar {
    margin-right: 0;
  }

  .content-inner {
    padding: 14px;
  }
}

@media (max-width: 420px) {
  .scope-button {
    width: min(42vw, 132px);
  }

  .scope-text {
    max-width: min(28vw, 74px);
  }
}
</style>
