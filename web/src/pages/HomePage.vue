<template>
  <main class="public-home">
    <header class="public-header">
      <router-link class="home-brand" to="/">
        <span class="brand-mark">XN</span>
        <span>
          <strong>{{ displaySiteName }}</strong>
          <small>License & Auth Cloud</small>
        </span>
      </router-link>

      <div class="header-actions">
        <n-button circle secondary class="theme-button" :title="themeButtonTitle" @click="cycleTheme">
          <template #icon>
            <n-icon><component :is="themeIcon" /></n-icon>
          </template>
        </n-button>
        <n-button type="primary" @click="goLogin">进入后台</n-button>
      </div>
    </header>

    <section class="hero-section">
      <div class="hero-copy">
        <h1>私有化 License 安全授权网关</h1>
        <p>
          {{ displaySiteName }} 面向本地部署软件提供应用授权、设备准入、在线心跳和数据上报能力，
          用应用级服务端密钥、客户端设备私钥和请求响应体双向加密构建可运营的安全验证链路。
        </p>
        <div class="hero-actions">
          <n-button type="primary" size="large" @click="goLogin">开始配置</n-button>
          <a class="text-link" href="#workflow">查看接入流程</a>
        </div>
        <div class="hero-proof">
          <span>一应用一套服务端密钥</span>
          <span>请求与响应体双向加密</span>
          <span>设备私钥签名绑定</span>
        </div>
      </div>
    </section>

    <section id="security" class="security-section">
      <div class="section-heading">
        <span>Security Architecture</span>
        <h2>从卡密校验到响应返回，全程围绕密钥可信执行</h2>
        <p>系统不是只做一次接口验签，而是把应用识别、设备绑定、请求加密和响应回包收束到同一套密钥链路。即使流量被抓取，也很难直接复用明文请求或伪造合法设备。</p>
      </div>
      <div class="security-grid">
        <article v-for="item in securityItems" :key="item.title">
          <n-icon><component :is="item.icon" /></n-icon>
          <h3>{{ item.title }}</h3>
          <p>{{ item.desc }}</p>
        </article>
      </div>
    </section>

    <section id="capabilities" class="home-section split-section">
      <div class="section-heading compact">
        <span>Operations Console</span>
        <h2>安全链路之外，也要能高效运营授权资产</h2>
        <p>后台以应用为核心组织卡密、设备、会话、公告版本和上报数据，让授权策略、异常处理和客户现场排查都能在一个控制台完成。</p>
      </div>
      <div class="capability-list">
        <article v-for="item in capabilities" :key="item.title">
          <n-icon><component :is="item.icon" /></n-icon>
          <div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.desc }}</p>
          </div>
        </article>
      </div>
    </section>

    <section id="workflow" class="home-section workflow-section">
      <div class="section-heading">
        <span>Integration Flow</span>
        <h2>按真实客户端落地流程组织接入</h2>
        <p>先配置应用级公钥，再生成客户端设备密钥，首次验证完成绑定，后续通过心跳和加密上报持续维持授权状态。</p>
      </div>
      <div class="workflow-rail">
        <article v-for="step in workflow" :key="step.title">
          <span>{{ step.index }}</span>
          <h3>{{ step.title }}</h3>
          <p>{{ step.desc }}</p>
        </article>
      </div>
    </section>

    <section class="cta-section">
      <div>
        <h2>从一套应用级密钥开始接入</h2>
        <p>进入后台创建应用、生成服务端密钥、签发卡密，并用真实运行数据持续观察验证成功率、在线设备和异常请求。</p>
      </div>
      <n-button type="primary" size="large" @click="goLogin">进入后台</n-button>
    </section>

    <footer class="home-footer">
      <div class="footer-main">
        <router-link class="footer-brand" to="/">
          <span class="brand-mark small">XN</span>
          <strong>{{ displaySiteName }}</strong>
        </router-link>
        <div class="footer-links">
          <a v-for="link in displayFooterLinks" :key="`${link.label}-${link.url}`" :href="link.url">{{ link.label }}</a>
        </div>
      </div>
      <div class="footer-record">
        <span>Copyright © {{ currentYear }} {{ displaySiteName }}. All Rights Reserved.</span>
        <a v-if="site.icp_number" href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer">
          {{ site.icp_number }}
        </a>
      </div>
    </footer>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { NButton, NIcon } from "naive-ui";
import {
  Analytics,
  Card,
  CloudUpload,
  DesktopOutline,
  Key,
  LockClosed,
  MoonOutline,
  Pulse,
  ShieldCheckmark,
  SunnyOutline
} from "@vicons/ionicons5";
import { publicSiteSettings, type FooterLink, type SiteSettings } from "@/api/admin";
import { useThemeStore, type ThemeMode } from "@/stores/theme";

const router = useRouter();
const themeStore = useThemeStore();

const site = reactive<SiteSettings>({
  site_name: "XNAuth 汐念验证",
  icp_number: "",
  footer_links: []
});

const displaySiteName = computed(() => site.site_name || "XNAuth 汐念验证");
const currentYear = new Date().getFullYear();
const displayFooterLinks = computed<FooterLink[]>(() => site.footer_links);
const themeIcon = computed(() => {
  if (themeStore.mode === "light") return SunnyOutline;
  if (themeStore.mode === "dark") return MoonOutline;
  return DesktopOutline;
});
const themeButtonTitle = computed(() => {
  if (themeStore.mode === "light") return "当前亮色主题，点击切换暗色";
  if (themeStore.mode === "dark") return "当前暗色主题，点击跟随系统";
  return "当前跟随系统，点击切换亮色";
});

const securityItems = [
  {
    title: "应用级密钥域",
    desc: "每个应用独立维护服务端 X25519/Ed25519 密钥，应用之间的通信信任边界互不复用。",
    icon: ShieldCheckmark
  },
  {
    title: "请求响应双向加密",
    desc: "客户端与服务端协商 AES-GCM 业务密钥，请求体和响应体都走密文信封，降低抓包分析价值。",
    icon: LockClosed
  },
  {
    title: "设备私钥身份",
    desc: "首次验证绑定设备公钥，后续请求必须由设备私钥签名，解绑指定设备时同步清理对应公钥。",
    icon: Key
  },
  {
    title: "密文响应回包",
    desc: "授权结果、公告版本和业务响应可通过加密信封返回客户端，减少响应数据明文暴露。",
    icon: CloudUpload
  }
];

const capabilities = [
  {
    title: "授权资产管理",
    desc: "卡密支持设备上限、到期时间、封禁冻结、删除和过期清理，适合批量交付和售后管控。",
    icon: Card
  },
  {
    title: "实时在线控制",
    desc: "基于心跳窗口判断实时在线，卡密详情可查看当前或最近设备，并支持强制踢下线。",
    icon: Pulse
  },
  {
    title: "公告版本下发",
    desc: "按应用维护公告、最新版本、最低支持版本和强制更新策略，客户端启动即可同步策略。",
    icon: CloudUpload
  },
  {
    title: "客户端数据洞察",
    desc: "自定义收集字段并按在线设备、离线设备和全部设备分类统计，辅助识别客户环境状态。",
    icon: Analytics
  }
];

const workflow = [
  { index: "01", title: "配置应用密钥", desc: "后台为应用生成独立服务端密钥对，客户端内置该应用对应的服务端公钥。" },
  { index: "02", title: "签发授权卡密", desc: "设置设备上限、到期时间和状态，把授权交付给客户部署环境。" },
  { index: "03", title: "加密首次验证", desc: "客户端生成设备密钥并提交密文信封，服务端解密、验签、绑定设备并返回会话。" },
  { index: "04", title: "心跳与上报运营", desc: "客户端持续心跳、读取公告版本并加密上报业务字段，后台沉淀运行视图。" }
];

onMounted(async () => {
  themeStore.initialize();
  try {
    const data = await publicSiteSettings();
    site.site_name = data.site_name;
    site.icp_number = data.icp_number;
    site.footer_links = [...data.footer_links];
    document.title = data.site_name || "XNAuth 汐念验证";
  } catch {
    document.title = "XNAuth 汐念验证";
  }
});

function cycleTheme() {
  const next: Record<ThemeMode, ThemeMode> = {
    light: "dark",
    dark: "system",
    system: "light"
  };
  themeStore.setMode(next[themeStore.mode]);
}

function goLogin() {
  router.push({ name: "login" });
}
</script>

<style scoped>
.public-home {
  --home-bg: #f7faff;
  --home-bg-top: #eef5ff;
  --home-bg-mid: #ffffff;
  --home-bg-bottom: #f7faff;
  --home-bg-wash: color-mix(in srgb, var(--app-primary) 10%, transparent);
  --home-hero-surface: rgba(255, 255, 255, 0.88);
  --home-hero-border: rgba(47, 107, 255, 0.16);
  --home-card: rgba(255, 255, 255, 0.9);
  --home-card-strong: #ffffff;
  --home-soft: #edf4ff;
  --home-line: rgba(47, 107, 255, 0.18);
  --home-security-bg:
    linear-gradient(135deg, rgba(47, 107, 255, 0.1), transparent 44%),
    linear-gradient(180deg, #f7fbff 0%, #edf5ff 100%);
  --home-security-border: rgba(47, 107, 255, 0.12);
  --home-security-kicker: var(--app-primary);
  --home-security-title: var(--app-text-strong);
  --home-security-muted: var(--app-muted);
  --home-security-card: rgba(255, 255, 255, 0.78);
  --home-security-card-border: rgba(47, 107, 255, 0.14);
  --home-security-icon-bg: var(--app-primary-soft);
  --home-security-icon: var(--app-primary);
  min-height: 100vh;
  min-height: 100dvh;
  overflow-x: hidden;
  background:
    linear-gradient(135deg, var(--home-bg-wash) 0%, transparent 34%),
    linear-gradient(180deg, var(--home-bg-top) 0%, var(--home-bg-mid) 42%, var(--home-bg-bottom) 100%),
    var(--home-bg);
  color: var(--app-text);
}

:global(:root[data-theme="dark"] .public-home) {
  --home-bg: #07101d;
  --home-bg-top: #07101d;
  --home-bg-mid: #0c1524;
  --home-bg-bottom: #07101d;
  --home-bg-wash: color-mix(in srgb, var(--app-primary) 12%, transparent);
  --home-hero-surface: rgba(15, 24, 41, 0.78);
  --home-hero-border: rgba(120, 162, 255, 0.2);
  --home-card: rgba(17, 28, 48, 0.78);
  --home-card-strong: rgba(17, 28, 48, 0.92);
  --home-soft: rgba(120, 162, 255, 0.12);
  --home-line: rgba(120, 162, 255, 0.24);
  --home-security-bg:
    linear-gradient(135deg, rgba(47, 107, 255, 0.22), transparent 44%),
    linear-gradient(180deg, #091427 0%, #0f1d32 100%);
  --home-security-border: rgba(255, 255, 255, 0.1);
  --home-security-kicker: #39d7a5;
  --home-security-title: #ffffff;
  --home-security-muted: rgba(226, 235, 255, 0.72);
  --home-security-card: rgba(255, 255, 255, 0.06);
  --home-security-card-border: rgba(255, 255, 255, 0.12);
  --home-security-icon-bg: rgba(57, 215, 165, 0.14);
  --home-security-icon: #39d7a5;
}

.public-header {
  position: sticky;
  top: 0;
  z-index: 20;
  display: grid;
  width: min(1180px, calc(100% - 40px));
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 18px;
  margin: 0 auto;
  padding: 18px 0;
  backdrop-filter: blur(18px);
}

.home-brand,
.footer-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.brand-mark {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--app-primary), var(--app-success));
  color: #ffffff;
  font-size: 13px;
  font-weight: 900;
  box-shadow: 0 18px 42px color-mix(in srgb, var(--app-primary) 28%, transparent);
}

.brand-mark.small {
  width: 34px;
  height: 34px;
  font-size: 11px;
}

.home-brand strong,
.home-brand small {
  display: block;
}

.home-brand strong {
  color: var(--app-text-strong);
  font-size: 18px;
  line-height: 22px;
}

.home-brand small {
  color: var(--app-muted);
  font-size: 12px;
  line-height: 18px;
}

.header-actions {
  display: flex;
  justify-self: end;
  align-items: center;
  gap: 10px;
}

.theme-button {
  color: var(--app-primary);
}

.hero-section {
  width: min(1180px, calc(100% - 40px));
  margin: 0 auto;
  padding: 106px 0 118px;
}

.hero-copy {
  max-width: 900px;
}

.hero-copy h1 {
  max-width: 820px;
  margin: 0;
  color: var(--app-text-strong);
  font-size: clamp(46px, 6.6vw, 82px);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1.02;
}

.hero-copy p {
  max-width: 680px;
  margin: 24px 0 0;
  color: var(--app-muted);
  font-size: 16px;
  line-height: 30px;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  align-items: center;
  margin-top: 34px;
}

.text-link {
  display: inline-flex;
  min-height: 40px;
  align-items: center;
  padding: 0 2px;
  color: var(--app-primary);
  font-size: 14px;
  font-weight: 800;
}

.hero-proof {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 34px;
}

.hero-proof span {
  padding: 9px 12px;
  border: 1px solid var(--home-line);
  border-radius: 999px;
  background: color-mix(in srgb, var(--home-soft) 82%, transparent);
  color: var(--app-text);
  font-size: 13px;
  font-weight: 700;
}

.home-section,
.cta-section,
.home-footer {
  width: min(1180px, calc(100% - 40px));
  margin: 0 auto;
}

.security-section {
  width: 100%;
  margin: 0;
  padding: 78px 0;
  border-top: 1px solid var(--home-security-border);
  border-bottom: 1px solid var(--home-security-border);
  background: var(--home-security-bg);
}

.security-section .section-heading,
.security-section .security-grid {
  width: min(1180px, calc(100% - 40px));
  margin-right: auto;
  margin-left: auto;
}

.security-section .section-heading {
  max-width: min(1180px, calc(100% - 40px));
}

.security-section .section-heading span {
  color: var(--home-security-kicker);
}

.security-section .section-heading h2 {
  max-width: 760px;
  color: var(--home-security-title);
}

.security-section .section-heading p {
  max-width: 760px;
  color: var(--home-security-muted);
}

.section-heading {
  max-width: 760px;
  margin-bottom: 30px;
}

.section-heading.compact {
  margin-bottom: 0;
}

.section-heading span {
  color: var(--app-primary);
  font-size: 12px;
  font-weight: 900;
  letter-spacing: 0;
  text-transform: uppercase;
}

.section-heading h2,
.cta-section h2 {
  margin: 12px 0 0;
  color: var(--app-text-strong);
  font-size: clamp(30px, 4vw, 50px);
  font-weight: 950;
  letter-spacing: 0;
  line-height: 1.12;
}

.section-heading p,
.cta-section p {
  margin: 18px 0 0;
  color: var(--app-muted);
  font-size: 15px;
  line-height: 28px;
}

.security-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.security-grid article,
.capability-list article,
.workflow-rail article {
  min-width: 0;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--home-card);
  box-shadow: var(--app-shadow-sm);
}

.security-section .security-grid article {
  border-color: var(--home-security-card-border);
  background: var(--home-security-card);
  box-shadow: none;
}

.security-section .security-grid .n-icon {
  background: var(--home-security-icon-bg);
  color: var(--home-security-icon);
}

.security-section .security-grid h3 {
  color: var(--home-security-title);
}

.security-section .security-grid p {
  color: var(--home-security-muted);
}

.security-grid article {
  min-height: 240px;
  padding: 24px;
}

.security-grid .n-icon,
.capability-list .n-icon {
  display: grid;
  width: 46px;
  height: 46px;
  place-items: center;
  border-radius: 8px;
  background: var(--app-primary-soft);
  color: var(--app-primary);
  font-size: 24px;
}

.security-grid h3,
.capability-list h3,
.workflow-rail h3 {
  margin: 22px 0 10px;
  color: var(--app-text-strong);
  font-size: 18px;
}

.security-grid p,
.capability-list p,
.workflow-rail p {
  margin: 0;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 24px;
}

.home-section {
  padding: 72px 0;
  border-top: 1px solid var(--app-border);
}

.split-section {
  display: grid;
  grid-template-columns: 0.86fr 1.14fr;
  gap: 42px;
  align-items: start;
}

.capability-list {
  display: grid;
  gap: 12px;
}

.capability-list article {
  display: grid;
  grid-template-columns: 50px 1fr;
  gap: 16px;
  align-items: center;
  padding: 18px;
}

.capability-list h3 {
  margin: 0 0 6px;
}

.workflow-rail {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.workflow-rail article {
  padding: 20px;
}

.workflow-rail span {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-primary) 12%, transparent);
  color: var(--app-primary);
  font-weight: 900;
}

.cta-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 70px;
  padding: 34px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 12%, var(--app-surface)), var(--app-surface)),
    var(--app-surface);
  box-shadow: var(--app-shadow-md);
}

.cta-section h2 {
  font-size: clamp(26px, 3vw, 36px);
}

.home-footer {
  padding: 28px 0 38px;
  border-top: 1px solid var(--app-border);
}

.footer-main,
.footer-record {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
}

.footer-brand strong {
  color: var(--app-text-strong);
  font-size: 15px;
}

.footer-links {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px 14px;
}

.footer-links a,
.footer-record,
.footer-record a {
  color: var(--app-muted);
  font-size: 13px;
}

.footer-links a:hover,
.footer-record a:hover {
  color: var(--app-primary);
}

.footer-record {
  margin-top: 18px;
}

@media (max-width: 1080px) {
  .split-section {
    grid-template-columns: 1fr;
  }

  .hero-section {
    padding-top: 68px;
  }

  .security-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .workflow-rail {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .public-header,
  .hero-section,
  .home-section,
  .cta-section,
  .home-footer {
    width: min(100% - 28px, 1180px);
  }

  .public-header {
    padding: 14px 0;
  }

  .home-brand small {
    display: none;
  }

  .brand-mark {
    width: 38px;
    height: 38px;
  }

  .header-actions {
    gap: 8px;
  }

  .header-actions :deep(.n-button:not(.theme-button)) {
    padding: 0 12px;
  }

  .hero-section {
    gap: 34px;
    padding: 52px 0 64px;
  }

  .hero-copy h1 {
    font-size: 44px;
  }

  .hero-copy p {
    font-size: 15px;
    line-height: 28px;
  }

  .hero-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .text-link {
    justify-content: center;
  }

  .security-section .section-heading,
  .security-section .security-grid {
    width: min(100% - 28px, 1180px);
  }

  .security-grid,
  .workflow-rail {
    grid-template-columns: 1fr;
  }

  .security-section,
  .home-section {
    padding: 56px 0;
  }

  .capability-list article {
    grid-template-columns: 1fr;
  }

  .cta-section,
  .footer-main,
  .footer-record {
    align-items: stretch;
    flex-direction: column;
  }

  .footer-links {
    justify-content: flex-start;
  }
}
</style>
