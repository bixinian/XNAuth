import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";
import { useAuthStore } from "@/stores/auth";

const routes: RouteRecordRaw[] = [
  {
    path: "/",
    name: "home",
    component: () => import("@/pages/HomePage.vue"),
    meta: { public: true }
  },
  {
    path: "/login",
    name: "login",
    component: () => import("@/pages/LoginPage.vue"),
    meta: { public: true }
  },
  {
    path: "/",
    component: () => import("@/layouts/AdminLayout.vue"),
    children: [
      { path: "dashboard", name: "dashboard", component: () => import("@/pages/DashboardPage.vue") },
      { path: "apps", name: "apps", component: () => import("@/pages/AppsPage.vue") },
      { path: "licenses", name: "licenses", component: () => import("@/pages/LicensesPage.vue") },
      { path: "licenses/:id", name: "license-detail", component: () => import("@/pages/LicenseDetailPage.vue") },
      { path: "devices", redirect: "/licenses" },
      { path: "sessions", redirect: "/licenses" },
      { path: "announcements", name: "announcements", component: () => import("@/pages/AnnouncementsPage.vue") },
      { path: "versions", name: "versions", component: () => import("@/pages/VersionsPage.vue") },
      { path: "collect-fields", name: "collect-fields", component: () => import("@/pages/CollectFieldsPage.vue") },
      { path: "data-stats", name: "data-stats", component: () => import("@/pages/DataSummaryPage.vue") },
      { path: "collect-records", name: "collect-records", component: () => import("@/pages/CollectRecordsPage.vue") },
      { path: "collect-records/:id", name: "collect-record-detail", component: () => import("@/pages/CollectRecordDetailPage.vue") },
      { path: "verify-logs", name: "verify-logs", component: () => import("@/pages/VerifyLogsPage.vue") },
      { path: "operation-logs", name: "operation-logs", component: () => import("@/pages/OperationLogsPage.vue") },
      { path: "integration-docs", name: "integration-docs", component: () => import("@/pages/IntegrationDocsPage.vue") },
      { path: "users", name: "users", component: () => import("@/pages/UsersPage.vue") },
      { path: "settings", name: "settings", component: () => import("@/pages/SettingsPage.vue") }
    ]
  }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

router.beforeEach((to) => {
  const auth = useAuthStore();
  if (!to.meta.public && !auth.isLoggedIn) {
    return { name: "login", query: { redirect: to.fullPath } };
  }
  if (to.name === "login" && auth.isLoggedIn) {
    return { name: "dashboard" };
  }
  return true;
});

export default router;
