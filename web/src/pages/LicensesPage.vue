<template>
  <ResourcePage
    title="卡密管理"
    subtitle="管理卡密状态、设备上限、当前在线和到期时间。"
    endpoint="/admin/licenses"
    searchable
    search-placeholder="搜索卡密"
    :fields="fields"
    :header-actions="headerActions"
    :actions="actions"
    :create-defaults="createDefaults"
    :prepare-create-body="prepareCreateBody"
    can-delete
  />
</template>

<script setup lang="ts">
import ResourcePage from "@/components/ResourcePage.vue";
import { deleteAction, putAction } from "@/api/admin";
import { useAppStore } from "@/stores/app";
import { useRouter } from "vue-router";
import type { FieldConfig, HeaderAction, RowAction } from "@/types/resource";

const router = useRouter();
const appStore = useAppStore();

const durationOptions = [
  { label: "天后过期", value: "day" },
  { label: "月后过期", value: "month" },
  { label: "年后过期", value: "year" },
  { label: "永久", value: "permanent" }
];

const fields: FieldConfig[] = [
  { key: "id", label: "ID", form: false, width: "70px" },
  { key: "license_key", label: "卡密", form: true, required: true },
  {
    key: "status",
    label: "状态",
    type: "select",
    form: false,
    options: [
      { label: "未激活", value: 0 },
      { label: "正常", value: 1 },
      { label: "已过期", value: 2 },
      { label: "已冻结", value: 3 },
      { label: "已封禁", value: 4 }
    ]
  },
  {
    key: "duration_unit",
    label: "有效期类型",
    type: "select",
    table: false,
    createOnly: true,
    required: true,
    options: durationOptions
  },
  { key: "duration_value", label: "有效期数值", type: "number", table: false, createOnly: true, min: 1, precision: 0 },
  { key: "max_devices", label: "设备上限", type: "number", min: 1, precision: 0 },
  { key: "current_online", label: "实时在线", form: false },
  { key: "expire_at", label: "过期时间", type: "datetime", editOnly: true },
  { key: "remark", label: "备注", type: "textarea" }
];

const actions: RowAction[] = [
  {
    label: "详情",
    reload: false,
    successMessage: false,
    run: async (row) => {
      await router.push(`/licenses/${row.id}`);
    }
  },
  { label: "解冻", type: "success", run: (row) => putAction(`/admin/licenses/${row.id}/status`, { status: 1 }) },
  { label: "冻结", type: "warning", run: (row) => putAction(`/admin/licenses/${row.id}/status`, { status: 3 }) },
  { label: "封禁", type: "error", ghost: true, run: (row) => putAction(`/admin/licenses/${row.id}/status`, { status: 4 }) }
];

const headerActions: HeaderAction[] = [
  {
    label: "清理过期",
    type: "warning",
    confirm: "确定删除已过期卡密吗？此操作不可恢复。",
    successMessage: "已清理过期卡密",
    run: () => {
      const query = appStore.currentAppId ? `?app_id=${appStore.currentAppId}` : "";
      return deleteAction(`/admin/licenses/expired${query}`);
    }
  }
];

function createDefaults() {
  return {
    license_key: randomLicenseKey(),
    duration_unit: "day",
    duration_value: 1,
    max_devices: 1
  };
}

function prepareCreateBody(body: Record<string, unknown>) {
  const next = { ...body };
  const unit = String(next.duration_unit || "day");
  const amount = Math.max(1, Math.floor(Number(next.duration_value) || 1));

  delete next.duration_unit;
  delete next.duration_value;
  delete next.current_online;
  next.max_online = Math.max(1, Math.floor(Number(next.max_devices) || 1));

  if (unit === "permanent") {
    delete next.expire_at;
    return next;
  }

  const expireAt = new Date();
  if (unit === "month") {
    expireAt.setMonth(expireAt.getMonth() + amount);
  } else if (unit === "year") {
    expireAt.setFullYear(expireAt.getFullYear() + amount);
  } else {
    expireAt.setDate(expireAt.getDate() + amount);
  }
  next.expire_at = formatLocalDateTime(expireAt);
  return next;
}

function randomLicenseKey(length = 20) {
  const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
  const bytes = new Uint8Array(length);
  if (window.crypto?.getRandomValues) {
    window.crypto.getRandomValues(bytes);
  } else {
    for (let index = 0; index < bytes.length; index += 1) {
      bytes[index] = Math.floor(Math.random() * 256);
    }
  }
  return Array.from(bytes, (byte) => letters[byte % letters.length]).join("");
}

function formatLocalDateTime(date: Date) {
  const pad = (input: number) => String(input).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(
    date.getMinutes()
  )}:${pad(date.getSeconds())}`;
}
</script>
