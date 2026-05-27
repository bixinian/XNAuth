<template>
  <ResourcePage
    title="收集字段"
    subtitle="后台只保存这里配置且启用的客户端上报字段；数额统计会按数字字段累加，非数字值将被忽略。"
    endpoint="/admin/collect/fields"
    :fields="fields"
    can-delete
  />
</template>

<script setup lang="ts">
import ResourcePage from "@/components/ResourcePage.vue";
import type { FieldConfig } from "@/types/resource";

const yesNo = [
  { label: "否", value: 0 },
  { label: "是", value: 1 }
];

const statTypes = [
  { label: "分布统计", value: "distribution" },
  { label: "数额统计", value: "sum" }
];

const fields: FieldConfig[] = [
  { key: "id", label: "ID", form: false, width: "70px" },
  { key: "field_key", label: "字段 Key", required: true },
  { key: "field_name", label: "字段名", required: true },
  { key: "enabled", label: "启用", type: "select", options: yesNo },
  { key: "show_in_list", label: "列表展示", type: "select", options: yesNo },
  { key: "stat_enabled", label: "参与统计", type: "select", options: yesNo },
  { key: "stat_type", label: "统计方式", type: "select", options: statTypes },
  { key: "sort_order", label: "排序", type: "number" }
];
</script>
