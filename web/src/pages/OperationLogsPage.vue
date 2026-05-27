<template>
  <ResourcePage
    title="操作日志"
    subtitle="查看后台状态变更和管理操作记录。"
    endpoint="/admin/operation-logs"
    :app-scoped="false"
    searchable
    search-placeholder="搜索模块、动作、目标、IP 或内容"
    :filters="filters"
    :fields="fields"
    :can-create="false"
    :can-edit="false"
  />
</template>

<script setup lang="ts">
import ResourcePage from "@/components/ResourcePage.vue";
import type { FieldConfig, ResourceFilter } from "@/types/resource";

const moduleOptions = [
  { label: "应用管理", value: "app" },
  { label: "卡密管理", value: "license" },
  { label: "设备管理", value: "device" },
  { label: "在线会话", value: "session" },
  { label: "公告管理", value: "announcement" },
  { label: "版本管理", value: "version" },
  { label: "收集字段", value: "collect_field" }
];

const filters: ResourceFilter[] = [
  {
    key: "module",
    label: "模块分类",
    placeholder: "全部分类",
    options: moduleOptions
  }
];

const fields: FieldConfig[] = [
  { key: "id", label: "ID", form: false, width: "70px" },
  { key: "admin_id", label: "管理员ID", form: false },
  { key: "module", label: "模块", type: "select", form: false, options: moduleOptions },
  { key: "action", label: "动作", form: false },
  { key: "target_id", label: "目标ID", form: false },
  { key: "client_ip", label: "客户端IP", form: false },
  { key: "created_at", label: "时间", type: "datetime", form: false }
];
</script>
