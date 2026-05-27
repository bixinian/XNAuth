<template>
  <section>
    <div class="toolbar">
      <div>
        <h2 class="page-title">收集记录详情</h2>
        <p class="page-subtitle">查看单次客户端上报的主记录和字段值。</p>
      </div>
      <div class="toolbar-actions">
        <n-button size="small" @click="router.push('/collect-records')">返回列表</n-button>
        <n-button size="small" type="primary" @click="load">刷新</n-button>
      </div>
    </div>

    <n-spin :show="loading">
      <div class="panel detail-panel">
        <n-descriptions v-if="record" bordered :column="2" label-placement="left">
          <n-descriptions-item label="记录ID">{{ record.id }}</n-descriptions-item>
          <n-descriptions-item label="应用ID">{{ record.app_id }}</n-descriptions-item>
          <n-descriptions-item label="卡密ID">{{ record.license_id }}</n-descriptions-item>
          <n-descriptions-item label="设备ID">{{ record.device_id }}</n-descriptions-item>
          <n-descriptions-item label="事件">{{ displayValue(record.event) }}</n-descriptions-item>
          <n-descriptions-item label="客户端IP">{{ displayValue(record.client_ip) }}</n-descriptions-item>
          <n-descriptions-item label="User-Agent">{{ displayValue(record.user_agent) }}</n-descriptions-item>
          <n-descriptions-item label="上报时间">{{ formatDateTime(record.created_at) }}</n-descriptions-item>
        </n-descriptions>

        <n-table :bordered="false" :single-line="false" class="value-table">
          <thead>
            <tr>
              <th>字段 Key</th>
              <th>字段值</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="values.length === 0">
              <td colspan="3" class="empty">暂无字段值</td>
            </tr>
            <tr v-for="item in values" :key="String(item.id)">
              <td>{{ item.field_key }}</td>
              <td>{{ item.field_value }}</td>
              <td>{{ formatDateTime(item.created_at) }}</td>
            </tr>
          </tbody>
        </n-table>
      </div>
    </n-spin>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { NButton, NDescriptions, NDescriptionsItem, NSpin, NTable, useMessage } from "naive-ui";
import { get } from "@/api/http";
import { displayValue, formatDateTime } from "@/utils/format";

interface Row {
  [key: string]: unknown;
}

const route = useRoute();
const router = useRouter();
const message = useMessage();
const loading = ref(false);
const record = ref<Row | null>(null);
const values = ref<Row[]>([]);

async function load() {
  loading.value = true;
  try {
    const data = await get<{ record: Row; values: Row[] }>(`/admin/collect/records/${route.params.id}`);
    record.value = data.record;
    values.value = data.values || [];
  } catch (error) {
    message.error(error instanceof Error ? error.message : "加载详情失败");
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<style scoped>
.detail-panel {
  padding: 18px;
}

.value-table {
  margin-top: 18px;
}

.empty {
  height: 72px;
  color: #6b7280;
  text-align: center;
}
</style>

