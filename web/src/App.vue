<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides" :locale="zhCN" :date-locale="dateZhCN">
    <n-global-style />
    <n-message-provider placement="top-right">
      <n-dialog-provider>
        <n-loading-bar-provider>
          <router-view />
        </n-loading-bar-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed, onMounted, watchEffect } from "vue";
import {
  darkTheme,
  dateZhCN,
  NConfigProvider,
  NDialogProvider,
  NGlobalStyle,
  NLoadingBarProvider,
  NMessageProvider,
  zhCN,
  type GlobalThemeOverrides
} from "naive-ui";
import { useThemeStore } from "@/stores/theme";

const themeStore = useThemeStore();

const baseOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: "#2f6bff",
    primaryColorHover: "#4f7fff",
    primaryColorPressed: "#2455db",
    primaryColorSuppl: "#2f6bff",
    infoColor: "#2f6bff",
    successColor: "#14b879",
    warningColor: "#f59e0b",
    errorColor: "#ef4444",
    borderRadius: "6px",
    borderRadiusSmall: "6px",
    fontFamily:
      "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, Microsoft YaHei, sans-serif",
    fontWeightStrong: "700"
  },
  Button: {
    borderRadiusTiny: "6px",
    borderRadiusSmall: "6px",
    borderRadiusMedium: "6px",
    borderRadiusLarge: "6px",
    fontWeight: "600"
  },
  Card: {
    borderRadius: "8px"
  },
  DataTable: {
    borderRadius: "8px",
    thFontWeight: "700"
  },
  Input: {
    borderRadius: "6px"
  },
  Select: {
    peers: {
      InternalSelection: {
        borderRadius: "6px"
      }
    }
  },
  Menu: {
    itemHeight: "44px",
    borderRadius: "6px",
    fontSize: "14px"
  },
  Modal: {
    borderRadius: "8px"
  }
};

const lightOverrides: GlobalThemeOverrides = {
  common: {
    bodyColor: "#f6f8fc",
    cardColor: "#ffffff",
    modalColor: "#ffffff",
    popoverColor: "#ffffff",
    tableColor: "#ffffff",
    textColorBase: "#121826",
    textColor1: "#121826",
    textColor2: "#334155",
    textColor3: "#667085",
    borderColor: "#e6eaf2",
    dividerColor: "#edf0f6"
  }
};

const darkOverrides: GlobalThemeOverrides = {
  common: {
    bodyColor: "#0c111d",
    cardColor: "#111827",
    modalColor: "#111827",
    popoverColor: "#111827",
    tableColor: "#111827",
    textColorBase: "#f8fafc",
    textColor1: "#f8fafc",
    textColor2: "#d7deea",
    textColor3: "#98a2b3",
    borderColor: "#263244",
    dividerColor: "#202b3d"
  }
};

const naiveTheme = computed(() => (themeStore.isDark ? darkTheme : null));
const themeOverrides = computed<GlobalThemeOverrides>(() => ({
  ...baseOverrides,
  common: {
    ...baseOverrides.common,
    ...(themeStore.isDark ? darkOverrides.common : lightOverrides.common)
  }
}));

onMounted(() => themeStore.initialize());

watchEffect(() => {
  document.documentElement.dataset.theme = themeStore.resolvedMode;
  document.documentElement.style.colorScheme = themeStore.resolvedMode;
});
</script>
