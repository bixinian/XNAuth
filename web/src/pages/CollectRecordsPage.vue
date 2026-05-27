<template>
  <ResourcePage
    title="收集记录"
    subtitle="按应用查看客户端上报记录，详情可在后续卡密详情页聚合。"
    endpoint="/admin/collect/records"
    :fields="fields"
    :can-create="false"
    :can-edit="false"
    searchable
    search-placeholder="搜索卡密、机器码、事件或 IP"
    :actions="actions"
  />
</template>

<script setup lang="ts">
import ResourcePage from "@/components/ResourcePage.vue";
import { useRouter } from "vue-router";
import type { FieldConfig } from "@/types/resource";
import type { RowAction } from "@/types/resource";

const router = useRouter();

const fields: FieldConfig[] = [
  { key: "id", label: "ID", form: false, width: "70px" },
  { key: "license_id", label: "卡密ID", form: false },
  { key: "device_id", label: "设备ID", form: false },
  { key: "license_key", label: "卡密", form: false },
  { key: "event", label: "事件", form: false },
  { key: "client_ip", label: "客户端IP", form: false },
  { key: "created_at", label: "上报时间", type: "datetime", form: false }
];

const actions: RowAction[] = [
  {
    label: "详情",
    reload: false,
    successMessage: false,
    run: async (row) => {
      await router.push(`/collect-records/${row.id}`);
    }
  }
];
</script>
