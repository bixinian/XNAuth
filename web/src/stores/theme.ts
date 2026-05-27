import { defineStore } from "pinia";

export type ThemeMode = "light" | "dark" | "system";

const storageKey = "xnauth_theme_mode";
let initialized = false;

function storedMode(): ThemeMode {
  const value = localStorage.getItem(storageKey);
  if (value === "light" || value === "dark" || value === "system") return value;
  return "system";
}

function prefersDark() {
  return window.matchMedia?.("(prefers-color-scheme: dark)").matches || false;
}

export const useThemeStore = defineStore("theme", {
  state: () => ({
    mode: storedMode(),
    systemDark: prefersDark()
  }),
  getters: {
    resolvedMode: (state): "light" | "dark" => (state.mode === "system" ? (state.systemDark ? "dark" : "light") : state.mode),
    isDark(): boolean {
      return this.resolvedMode === "dark";
    }
  },
  actions: {
    initialize() {
      if (initialized) return;
      initialized = true;
      const query = window.matchMedia?.("(prefers-color-scheme: dark)");
      if (!query) return;
      this.systemDark = query.matches;
      query.addEventListener("change", (event) => {
        this.systemDark = event.matches;
      });
    },
    setMode(mode: ThemeMode) {
      this.mode = mode;
      localStorage.setItem(storageKey, mode);
    }
  }
});
