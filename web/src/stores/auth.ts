import { defineStore } from "pinia";
import { login, profile } from "@/api/admin";
import router from "@/router";

interface UserInfo {
  id: number;
  username: string;
}

export const useAuthStore = defineStore("auth", {
  state: () => ({
    token: localStorage.getItem("xnauth_token") || "",
    user: null as UserInfo | null
  }),
  getters: {
    isLoggedIn: (state) => Boolean(state.token)
  },
  actions: {
    async login(username: string, password: string, captchaID?: string, captchaToken?: string) {
      const data = await login(username, password, captchaID, captchaToken);
      this.token = data.token;
      this.user = data.user;
      localStorage.setItem("xnauth_token", data.token);
    },
    async loadProfile() {
      if (!this.token) return;
      this.user = await profile();
    },
    logout() {
      this.token = "";
      this.user = null;
      localStorage.removeItem("xnauth_token");
      if (router.currentRoute.value.name !== "login") {
        router.push({ name: "login" });
      }
    }
  }
});
