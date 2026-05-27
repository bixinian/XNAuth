<template>
  <ResourcePage
    title="应用管理"
    subtitle="创建授权入口，并配置应用级登录策略。"
    endpoint="/admin/apps"
    :app-scoped="false"
    searchable
    search-placeholder="搜索应用名或 AppKey"
    :fields="fields"
    :actions="actions"
    can-delete
  />
</template>

<script setup lang="ts">
import ResourcePage from "@/components/ResourcePage.vue";
import { generateAppSecurityKeys } from "@/api/admin";
import type { FieldConfig, RowAction } from "@/types/resource";

const fields: FieldConfig[] = [
  { key: "id", label: "ID", form: false, width: "70px" },
  { key: "app_key", label: "AppKey", required: true },
  { key: "app_name", label: "应用名称", required: true },
  {
    key: "status",
    label: "状态",
    type: "select",
    options: [
      { label: "启用", value: 1 },
      { label: "停用", value: 2 }
    ]
  },
  { key: "min_login_version_code", label: "最低登录版本", type: "number" },
  {
    key: "force_update",
    label: "强制更新",
    type: "select",
    options: [
      { label: "否", value: 0 },
      { label: "是", value: 1 }
    ]
  },
  { key: "secure_key_id", label: "安全 Key ID", width: "180px" },
  { key: "secure_x25519_public_key", label: "X25519 公钥", type: "textarea", table: false },
  { key: "secure_x25519_private_key", label: "X25519 私钥", type: "textarea", table: false },
  { key: "secure_ed25519_public_key", label: "Ed25519 公钥", type: "textarea", table: false },
  { key: "secure_ed25519_private_key", label: "Ed25519 私钥", type: "textarea", table: false },
  { key: "remark", label: "备注", type: "textarea" },
  { key: "created_at", label: "创建时间", type: "datetime", form: false }
];

const actions: RowAction[] = [
  {
    label: "生成密钥",
    type: "primary",
    confirm: "重新生成应用级服务端密钥后，客户端需要更新内置的应用公钥配置；旧公钥请求将无法通过。确定继续吗？",
    successMessage: "已生成应用密钥",
    run: async (row) => {
      await generateAppSecurityKeys(Number(row.id));
    }
  }
];
</script>
