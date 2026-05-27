<template>
  <main class="login-page">
    <section class="login-panel" aria-label="登录后台">
      <n-card class="login-card" :bordered="false" content-style="padding: 0;">
        <div class="form-header">
          <div class="login-brand">
            <div class="logo">XN</div>
            <div>
              <strong>XNAuth</strong>
              <span>汐念验证</span>
            </div>
          </div>
          <n-button circle secondary class="theme-button" :title="themeButtonTitle" @click="cycleTheme">
            <template #icon>
              <n-icon><component :is="themeIcon" /></n-icon>
            </template>
          </n-button>
        </div>
        <div class="login-title">
          <h1>登录后台</h1>
          <p>使用管理员账号继续</p>
        </div>
        <n-form class="login-form" label-placement="top" :model="form" @submit.prevent="submit">
          <n-form-item label="管理员账号">
            <n-input v-model:value="form.username" size="large" placeholder="admin" />
          </n-form-item>
          <n-form-item label="密码">
            <n-input v-model:value="form.password" size="large" type="password" show-password-on="click" />
          </n-form-item>
          <n-form-item v-if="captchaState.enabled" label="安全验证">
            <div class="slider-wrap">
              <div
                ref="sliderTrack"
                class="slider-captcha"
                :class="{ verified: slider.verified, dragging: slider.dragging }"
                @pointerdown="startSlide"
              >
                <div class="slider-fill" :style="{ width: `${slider.progress}%` }"></div>
                <div class="slider-text">
                  {{ slider.verified ? "验证通过" : slider.verifying ? "验证中" : "拖动滑块到最右侧" }}
                </div>
                <button
                  class="slider-handle"
                  type="button"
                  :style="{ left: `calc(${slider.progress}% - ${slider.progress * 0.38}px)` }"
                  :disabled="slider.verified || slider.verifying"
                  @pointerdown.stop="startSlide"
                >
                  {{ slider.verified ? "✓" : "→" }}
                </button>
              </div>
            </div>
          </n-form-item>
          <n-button block size="large" type="primary" :loading="loading" @click="submit">登录</n-button>
        </n-form>
      </n-card>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NButton, NCard, NForm, NFormItem, NIcon, NInput, useMessage } from "naive-ui";
import { DesktopOutline, MoonOutline, SunnyOutline } from "@vicons/ionicons5";
import { captcha, verifyCaptcha } from "@/api/admin";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore, type ThemeMode } from "@/stores/theme";

const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
const message = useMessage();
const themeStore = useThemeStore();
const loading = ref(false);
const form = reactive({
  username: "admin",
  password: ""
});
const captchaState = reactive({
  enabled: false,
  id: "",
  token: ""
});
const sliderTrack = ref<HTMLElement | null>(null);
const slider = reactive({
  progress: 0,
  dragging: false,
  verifying: false,
  verified: false
});

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

function cycleTheme() {
  const next: Record<ThemeMode, ThemeMode> = {
    light: "dark",
    dark: "system",
    system: "light"
  };
  themeStore.setMode(next[themeStore.mode]);
}

onMounted(loadCaptcha);
onBeforeUnmount(stopSlide);

async function loadCaptcha() {
  try {
    const data = await captcha();
    captchaState.enabled = data.enabled;
    captchaState.id = data.captcha_id || "";
    captchaState.token = "";
    resetSlider();
  } catch (error) {
    message.error(error instanceof Error ? error.message : "验证加载失败");
  }
}

function resetSlider() {
  slider.progress = 0;
  slider.dragging = false;
  slider.verifying = false;
  slider.verified = false;
}

function startSlide(event: PointerEvent) {
  if (!captchaState.id || slider.verified || slider.verifying) return;
  slider.dragging = true;
  moveSlide(event);
  window.addEventListener("pointermove", moveSlide);
  window.addEventListener("pointerup", stopSlide);
}

function moveSlide(event: PointerEvent) {
  if (!slider.dragging || slider.verified || slider.verifying) return;
  const rect = sliderTrack.value?.getBoundingClientRect();
  if (!rect || rect.width <= 0) return;
  const next = Math.max(0, Math.min(100, ((event.clientX - rect.left) / rect.width) * 100));
  slider.progress = next;
  if (next >= 98) completeSlide();
}

function stopSlide() {
  window.removeEventListener("pointermove", moveSlide);
  window.removeEventListener("pointerup", stopSlide);
  if (slider.dragging && !slider.verified && !slider.verifying) {
    slider.progress = 0;
  }
  slider.dragging = false;
}

async function completeSlide() {
  if (slider.verifying || slider.verified) return;
  slider.dragging = false;
  slider.progress = 100;
  slider.verifying = true;
  stopSlide();
  try {
    const data = await verifyCaptcha(captchaState.id, 100);
    captchaState.token = data.captcha_token || "";
    slider.verified = Boolean(data.verified && captchaState.token);
    if (!slider.verified) throw new Error("验证失败");
  } catch (error) {
    message.error(error instanceof Error ? error.message : "验证失败");
    await loadCaptcha();
  } finally {
    slider.verifying = false;
  }
}

async function submit() {
  if (!form.username || !form.password) {
    message.warning("请输入账号和密码");
    return;
  }
  if (captchaState.enabled && !captchaState.token) {
    message.warning("请先完成滑块验证");
    return;
  }
  loading.value = true;
  try {
    await auth.login(form.username, form.password, captchaState.id, captchaState.token);
    const redirect = typeof route.query.redirect === "string" ? route.query.redirect : "/dashboard";
    router.push(redirect);
  } catch (error) {
    message.error(error instanceof Error ? error.message : "登录失败");
    if (captchaState.enabled) await loadCaptcha();
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.login-page {
  display: grid;
  min-height: 100vh;
  min-height: 100dvh;
  place-items: center;
  padding: 24px;
  background:
    radial-gradient(circle at top left, color-mix(in srgb, var(--app-primary) 14%, transparent), transparent 34%),
    linear-gradient(180deg, var(--app-bg-accent), transparent 42%),
    var(--app-bg);
}

.login-panel {
  width: min(420px, 100%);
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border-radius: 8px;
  background: linear-gradient(135deg, #1e2a44, #2f6bff);
  color: #ffffff;
  font-size: 14px;
  font-weight: 800;
  box-shadow: 0 12px 24px rgba(47, 107, 255, 0.24);
}

.login-brand strong,
.login-brand span {
  display: block;
}

.login-brand strong {
  color: var(--app-text-strong);
  font-size: 17px;
  line-height: 23px;
}

.login-brand span {
  color: var(--app-muted);
  font-size: 12px;
  line-height: 18px;
}

.login-card {
  overflow: hidden;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--app-primary) 5%, transparent), transparent 42%),
    var(--app-surface);
  box-shadow: var(--app-shadow-md);
}

.form-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 26px 28px 0;
}

.theme-button {
  flex: 0 0 auto;
  color: var(--app-primary);
}

.login-title {
  padding: 28px 28px 0;
}

.login-title h1 {
  margin: 0;
  color: var(--app-text-strong);
  font-size: 26px;
  font-weight: 800;
  line-height: 34px;
}

.login-title p {
  margin: 6px 0 0;
  color: var(--app-muted);
  font-size: 13px;
  line-height: 20px;
}

.login-form {
  padding: 24px 28px 28px;
}

.slider-wrap {
  display: grid;
  gap: 8px;
  width: 100%;
}

.slider-captcha {
  position: relative;
  height: 44px;
  overflow: hidden;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
  cursor: pointer;
  touch-action: none;
  user-select: none;
}

.slider-fill {
  position: absolute;
  inset: 0 auto 0 0;
  background: linear-gradient(90deg, color-mix(in srgb, var(--app-primary) 18%, transparent), color-mix(in srgb, var(--app-primary) 34%, transparent));
  transition: width 0.12s ease;
}

.slider-captcha.dragging .slider-fill {
  transition: none;
}

.slider-captcha.verified .slider-fill {
  background: linear-gradient(90deg, rgba(20, 184, 121, 0.18), rgba(20, 184, 121, 0.34));
}

.slider-text {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: var(--app-muted);
  font-size: 13px;
  font-weight: 700;
  pointer-events: none;
}

.slider-captcha.verified .slider-text {
  color: var(--app-success);
}

.slider-handle {
  position: absolute;
  top: 50%;
  left: 0;
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border: 0;
  border-radius: 8px;
  background: var(--app-primary);
  color: #ffffff;
  cursor: grab;
  font-size: 18px;
  font-weight: 900;
  transform: translateY(-50%);
  transition: left 0.12s ease, background 0.12s ease;
}

.slider-captcha.dragging .slider-handle {
  cursor: grabbing;
  transition: none;
}

.slider-captcha.verified .slider-handle {
  background: var(--app-success);
}

.slider-handle:disabled {
  cursor: default;
}

@media (max-width: 560px) {
  .login-page {
    padding: 14px;
  }

  .form-header {
    padding: 22px 22px 0;
  }

  .login-title {
    padding: 24px 22px 0;
  }

  .login-form {
    padding: 22px;
  }

  .slider-captcha {
    height: 46px;
  }
}
</style>
