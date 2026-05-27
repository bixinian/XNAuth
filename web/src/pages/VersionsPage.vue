<template>
  <ResourcePage
    title="版本管理"
    subtitle="维护应用版本号、下载地址和更新说明；最低登录版本与强制更新策略在应用管理中配置。"
    endpoint="/admin/versions"
    :fields="fields"
    can-delete
  />
</template>

<script setup lang="ts">
import ResourcePage from "@/components/ResourcePage.vue";
import type { FieldConfig } from "@/types/resource";

const fields: FieldConfig[] = [
  { key: "id", label: "ID", form: false, width: "70px" },
  { key: "version_name", label: "版本名", required: true },
  { key: "version_code", label: "版本号", type: "number", required: true },
  {
    key: "enabled",
    label: "启用",
    type: "select",
    options: [
      { label: "停用", value: 0 },
      { label: "启用", value: 1 }
    ]
  },
  { key: "download_url", label: "下载地址" },
  { key: "file_hash", label: "文件 Hash" },
  { key: "file_size", label: "文件大小", type: "number" },
  { key: "changelog", label: "更新说明", type: "textarea" },
  { key: "released_at", label: "发布时间", type: "datetime" }
];
</script>
