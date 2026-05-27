import { del, get, post, put, type PageData } from "./http";

export interface AppRecord {
  id: number;
  app_key: string;
  app_name: string;
  status: number;
  min_login_version_code?: number | null;
  force_update: number;
  secure_key_id?: string | null;
  secure_x25519_private_key?: string | null;
  secure_x25519_public_key?: string | null;
  secure_ed25519_private_key?: string | null;
  secure_ed25519_public_key?: string | null;
  remark?: string | null;
  created_at: string;
  updated_at: string;
}

export interface LoginResp {
  token: string;
  expire_at: number;
  user: {
    id: number;
    username: string;
  };
}

export interface AdminProfile {
  id: number;
  username: string;
  status: number;
  last_login_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CaptchaResp {
  enabled: boolean;
  type?: "slider";
  captcha_id?: string;
  expire_seconds?: number;
}

export interface CaptchaVerifyResp {
  enabled: boolean;
  verified: boolean;
  captcha_id?: string;
  captcha_token?: string;
}

export interface SecuritySettings {
  login_captcha_enabled: boolean;
}

export interface CleanupResp {
  target: string;
  before: string;
  deleted_records: number;
  deleted_values?: number;
  matched_records: number;
}

export interface SystemStatus {
  status: string;
  uptime_seconds: number;
  goroutines: number;
  memory: {
    alloc_mb: number;
    sys_mb: number;
    heap_alloc_mb: number;
    heap_inuse_mb: number;
    last_gc_unix: number;
    gc_count: number;
    next_gc_mb: number;
    total_alloc_mb: number;
  };
  database: {
    status: string;
    stats: {
      open_connections: number;
      in_use: number;
      idle: number;
      wait_count: number;
    };
  };
}

export interface FooterLink {
  label: string;
  url: string;
}

export interface SiteSettings {
  site_name: string;
  icp_number: string;
  footer_links: FooterLink[];
}

export interface DashboardTrendItem {
  date: string;
  label: string;
  total: number;
  success: number;
  failed: number;
  today: boolean;
}

export interface DashboardSessionTrendItem {
  date: string;
  label: string;
  total: number;
  created: number;
  active: number;
  today: boolean;
}

export interface DashboardSummary {
  scope: {
    app_id?: number | null;
    generated_at: string;
  };
  metrics: {
    apps: number;
    active_apps: number;
    licenses: number;
    active_licenses: number;
    devices: number;
    online_sessions: number;
    collect_records: number;
    verify_today: number;
    verify_success: number;
    verify_failed: number;
    verify_success_rate: number;
  };
  trend: DashboardTrendItem[];
  session_trend: DashboardSessionTrendItem[];
}

export function captcha() {
  return get<CaptchaResp>("/admin/captcha");
}

export function verifyCaptcha(captchaID: string, sliderValue: number) {
  return post<CaptchaVerifyResp>("/admin/captcha/verify", {
    captcha_id: captchaID,
    slider_value: sliderValue
  });
}

export function login(username: string, password: string, captchaID?: string, captchaToken?: string) {
  return post<LoginResp>("/admin/login", {
    username,
    password,
    captcha_id: captchaID,
    captcha_token: captchaToken
  });
}

export function profile() {
  return get<AdminProfile>("/admin/profile");
}

export function updateProfile(body: { username?: string; current_password: string; new_password?: string }) {
  return put<AdminProfile>("/admin/profile", body);
}

export function securitySettings() {
  return get<SecuritySettings>("/admin/settings/security");
}

export function updateSecuritySettings(body: SecuritySettings) {
  return put<SecuritySettings>("/admin/settings/security", { ...body });
}

export function cleanupData(body: { target: string; before: string }) {
  return post<CleanupResp>("/admin/settings/cleanup", body);
}

export function systemStatus() {
  return get<SystemStatus>("/admin/settings/status");
}

export function siteSettings() {
  return get<SiteSettings>("/admin/settings/site");
}

export function updateSiteSettings(body: SiteSettings) {
  return put<SiteSettings>("/admin/settings/site", { ...body });
}

export function publicSiteSettings() {
  return get<SiteSettings>("/public/site-settings");
}

export function dashboardSummary(params?: Record<string, unknown>) {
  return get<DashboardSummary>("/admin/dashboard/summary", params);
}

export function listApps(params?: Record<string, unknown>) {
  return get<PageData<AppRecord>>("/admin/apps", params);
}

export function generateAppSecurityKeys(id: number) {
  return put<AppRecord>(`/admin/apps/${id}/security-keys/generate`, {});
}

export function createRecord<T>(endpoint: string, body: Record<string, unknown>) {
  return post<T>(endpoint, body);
}

export function updateRecord<T>(endpoint: string, id: number, body: Record<string, unknown>) {
  return put<T>(`${endpoint}/${id}`, body);
}

export function deleteRecord<T>(endpoint: string, id: number) {
  return del<T>(`${endpoint}/${id}`);
}

export function deleteAction<T>(path: string) {
  return del<T>(path);
}

export function listRecords<T>(endpoint: string, params?: Record<string, unknown>) {
  return get<PageData<T>>(endpoint, params);
}

export function getRecord<T>(endpoint: string, id: number | string) {
  return get<T>(`${endpoint}/${id}`);
}

export function putAction<T>(path: string, body?: Record<string, unknown>) {
  return put<T>(path, body ?? {});
}
