import { defineStore } from "pinia";
import { listApps, type AppRecord } from "@/api/admin";

function initialAppScope() {
  const stored = localStorage.getItem("xnauth_app_scope_id");
  return stored === null ? 0 : Number(stored) || 0;
}

export const useAppStore = defineStore("app", {
  state: () => ({
    apps: [] as AppRecord[],
    currentAppId: initialAppScope(),
    loading: false
  }),
  getters: {
    currentApp: (state) => state.apps.find((item) => item.id === state.currentAppId) || null
  },
  actions: {
    async loadApps() {
      this.loading = true;
      try {
        const data = await listApps({ page: 1, page_size: 100, status: 1 });
        this.apps = data.list;
        localStorage.removeItem("xnauth_current_app_id");
        if (this.currentAppId && !data.list.some((item) => item.id === this.currentAppId)) {
          this.setCurrentApp(0);
        }
      } finally {
        this.loading = false;
      }
    },
    setCurrentApp(appId: number) {
      this.currentAppId = appId;
      localStorage.setItem("xnauth_app_scope_id", String(appId));
    }
  }
});
