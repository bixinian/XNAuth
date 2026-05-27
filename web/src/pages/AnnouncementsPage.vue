<template>
  <ResourcePage
    title="公告管理"
    subtitle="配置客户端独立获取的公告内容和弹窗策略。"
    endpoint="/admin/announcements"
    :fields="fields"
    can-delete
  />
</template>

<script setup lang="ts">
import ResourcePage from "@/components/ResourcePage.vue";
import type { FieldConfig } from "@/types/resource";

const fields: FieldConfig[] = [
  { key: "id", label: "ID", form: false, width: "70px" },
  { key: "title", label: "标题", required: true },
  { key: "content", label: "内容", type: "textarea", required: true },
  { key: "notice_type", label: "类型" },
  {
    key: "popup",
    label: "弹窗",
    type: "select",
    options: [
      { label: "否", value: 0 },
      { label: "是", value: 1 }
    ]
  },
  {
    key: "enabled",
    label: "启用",
    type: "select",
    options: [
      { label: "停用", value: 0 },
      { label: "启用", value: 1 }
    ]
  },
  { key: "sort_order", label: "排序", type: "number" },
  { key: "start_at", label: "生效时间", type: "datetime" },
  { key: "end_at", label: "失效时间", type: "datetime" }
];
</script>

