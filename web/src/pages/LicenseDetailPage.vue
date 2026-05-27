<template>
  <section>
    <div class="toolbar">
      <div>
        <h2 class="page-title">卡密详情</h2>
        <p class="page-subtitle">聚合展示卡密基础信息、绑定设备、在线会话、上报数据和验证日志。</p>
      </div>
      <div class="toolbar-actions">
        <n-button size="small" @click="router.push('/licenses')">返回列表</n-button>
        <n-button size="small" type="primary" @click="load">刷新</n-button>
      </div>
    </div>

    <n-spin :show="loading">
      <div class="detail-grid">
        <div class="panel detail-panel license-summary">
          <div class="summary-main">
            <div class="summary-copy">
              <span class="summary-label">卡密</span>
              <code class="license-key">{{ displayValue(license?.license_key) }}</code>
            </div>
            <n-tag round :bordered="false" :type="licenseStatusType">{{ licenseStatus }}</n-tag>
          </div>

          <div class="metric-grid">
            <div v-for="item in summaryMetrics" :key="item.label" class="metric-item">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>

          <div class="info-list">
            <div class="info-item">
              <span>ID</span>
              <strong>{{ displayValue(license?.id) }}</strong>
            </div>
            <div class="info-item">
              <span>应用 ID</span>
              <strong>{{ displayValue(license?.app_id) }}</strong>
            </div>
            <div class="info-item">
              <span>过期时间</span>
              <strong>{{ formatDateTime(license?.expire_at) }}</strong>
            </div>
            <div class="info-item info-item-wide">
              <span>备注</span>
              <strong>{{ displayValue(license?.remark) }}</strong>
            </div>
          </div>
        </div>

        <n-tabs type="line" animated>
          <n-tab-pane name="devices" tab="绑定设备">
            <SimpleTable :rows="devices" :columns="deviceColumns" />
          </n-tab-pane>
          <n-tab-pane name="sessions" tab="在线会话">
            <SimpleTable :rows="sessionDeviceRows" :columns="sessionColumns" />
          </n-tab-pane>
          <n-tab-pane name="latest" tab="最新上报">
            <div class="panel detail-panel">
              <div v-if="latestCollect.record" class="latest-meta-grid">
                <div class="latest-meta-item">
                  <span>记录ID</span>
                  <strong>{{ displayValue(latestCollect.record.id) }}</strong>
                </div>
                <div class="latest-meta-item">
                  <span>事件</span>
                  <strong>{{ displayValue(latestCollect.record.event) }}</strong>
                </div>
                <div class="latest-meta-item">
                  <span>客户端IP</span>
                  <strong>{{ displayValue(latestCollect.record.client_ip) }}</strong>
                </div>
                <div class="latest-meta-item">
                  <span>上报时间</span>
                  <strong>{{ formatDateTime(latestCollect.record.created_at) }}</strong>
                </div>
              </div>
              <div v-else class="empty-state">
                <n-empty description="暂无最新上报" />
              </div>
              <SimpleTable :rows="latestCollect.values" :columns="valueColumns" />
            </div>
          </n-tab-pane>
          <n-tab-pane name="records" tab="历史上报">
            <SimpleTable :rows="collectRecords" :columns="recordColumns" show-pager show-size-picker :page-size-options="detailPageSizes" />
          </n-tab-pane>
          <n-tab-pane name="logs" tab="验证日志">
            <SimpleTable :rows="verifyLogs" :columns="verifyColumns" show-pager show-size-picker :page-size-options="detailPageSizes" />
          </n-tab-pane>
        </n-tabs>
      </div>
    </n-spin>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, watch, type VNodeChild } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NButton,
  NEmpty,
  NIcon,
  NPagination,
  NSpace,
  NSpin,
  NTabPane,
  NTabs,
  NTag,
  NTable,
  useDialog,
  useMessage
} from "naive-ui";
import { CopyOutline } from "@vicons/ionicons5";
import { get, type PageData } from "@/api/http";
import { putAction } from "@/api/admin";
import { displayValue, formatDateTime, statusLabel } from "@/utils/format";

interface Row {
  [key: string]: unknown;
}

interface TableColumn {
  key: string;
  label: string;
  width?: number | string;
  mono?: boolean;
  wrap?: boolean;
  ellipsis?: boolean;
  copy?: boolean;
  format?: (value: unknown) => string;
  render?: (row: Row) => VNodeChild;
}

const SimpleTable = defineComponent({
  props: {
    rows: { type: Array as () => Row[], required: true },
    columns: { type: Array as () => TableColumn[], required: true },
    pageSize: { type: Number, default: 8 },
    pageSizeOptions: { type: Array as () => number[], default: () => [5, 8, 10, 20] },
    showPager: { type: Boolean, default: false },
    showSizePicker: { type: Boolean, default: false }
  },
  setup(props) {
    const page = ref(1);
    const pageSize = ref(props.pageSize);
    const pageRows = computed(() => {
      const start = (page.value - 1) * pageSize.value;
      return props.rows.slice(start, start + pageSize.value);
    });
    watch(
      () => props.rows.length,
      () => {
        page.value = 1;
      }
    );
    watch(
      () => props.pageSize,
      (next) => {
        pageSize.value = next;
        page.value = 1;
      }
    );

    return () => {
      if (!props.rows.length) {
        return h("div", { class: "empty-state" }, [h(NEmpty, { description: "暂无数据" })]);
      }
      const shouldShowPager = props.showPager || props.rows.length > pageSize.value;
      return h("div", { class: "detail-table-block" }, [
        h("div", { class: "detail-table-scroll" }, [
          h(NTable, { bordered: false, singleLine: false, class: "nested-table" }, {
            default: () => [
              h("thead", [
                h("tr", props.columns.map((column) => h("th", { style: columnStyle(column) }, column.label)))
              ]),
              h(
                "tbody",
                pageRows.value.map((row) =>
                  h(
                    "tr",
                    props.columns.map((column) =>
                      h("td", { style: columnStyle(column) }, renderTableContent(column, row) as any)
                    )
                  )
                )
              )
            ]
          })
        ]),
        h(
          "div",
          { class: "mobile-row-list" },
          pageRows.value.map((row) =>
            h(
              "div",
              { class: "mobile-row-card" },
              props.columns.map((column) =>
                h("div", { class: "mobile-row-field" }, [
                  h("span", { class: "mobile-row-label" }, column.label),
                  h("div", { class: "mobile-row-value" }, renderTableContent(column, row) as any)
                ])
              )
            )
          )
        ),
        shouldShowPager
          ? h("div", { class: "nested-pager" }, [
              h("span", { class: "nested-pager-meta" }, `共 ${props.rows.length} 条，每页 ${pageSize.value} 条`),
              h(NPagination, {
                page: page.value,
                pageSize: pageSize.value,
                itemCount: props.rows.length,
                showSizePicker: props.showSizePicker,
                pageSizes: props.pageSizeOptions,
                pageSlot: 4,
                "onUpdate:page": (next: number) => {
                  page.value = next;
                },
                "onUpdate:pageSize": (next: number) => {
                  pageSize.value = next;
                  page.value = 1;
                }
              })
            ])
          : null
      ]);
    };
  }
});

const route = useRoute();
const router = useRouter();
const message = useMessage();
const dialog = useDialog();
const loading = ref(false);
const license = ref<Row | null>(null);
const devices = ref<Row[]>([]);
const sessions = ref<Row[]>([]);
const collectRecords = ref<Row[]>([]);
const verifyLogs = ref<Row[]>([]);
const latestCollect = ref<{ record: Row | null; values: Row[] }>({ record: null, values: [] });
const detailPageSizes = [5, 8, 10, 20];
let manualCopyInput: HTMLTextAreaElement | null = null;

const licenseID = computed(() => String(route.params.id));
const licenseStatus = computed(() => statusLabel(license.value?.status, {
  0: "未激活",
  1: "正常",
  2: "已过期",
  3: "已冻结",
  4: "已封禁"
}));
const licenseStatusType = computed(() => {
  const status = Number(license.value?.status);
  if (status === 1) return "success";
  if (status === 2 || status === 3) return "warning";
  if (status === 4) return "error";
  return "info";
});
const summaryMetrics = computed(() => [
  { label: "设备上限", value: displayValue(license.value?.max_devices) },
  { label: "实时在线", value: activeSessionCount.value },
  { label: "绑定设备", value: devices.value.length },
  { label: "最近会话", value: displayValue(latestSessionTime.value) }
]);

const activeSessionCount = computed(() => sessionDeviceRows.value.filter((row) => Number(row.status) === 1).length);
const latestSessionTime = computed(() => {
  const newest = [...sessions.value].sort((a, b) => timeValue(b.last_heartbeat_at) - timeValue(a.last_heartbeat_at))[0];
  return newest ? formatDateTime(newest.last_heartbeat_at) : "暂无";
});
const deviceByID = computed(() => {
  const result = new Map<string, Row>();
  for (const device of devices.value) {
    result.set(String(device.id), device);
  }
  return result;
});
const sessionDeviceRows = computed(() => {
  const rows = [...sessions.value].sort((a, b) => {
    const activeDiff = Number(b.status === 1) - Number(a.status === 1);
    if (activeDiff !== 0) return activeDiff;
    return timeValue(b.last_heartbeat_at) - timeValue(a.last_heartbeat_at);
  });
  const picked = new Map<string, Row>();
  for (const row of rows) {
    const deviceID = String(row.device_id ?? "");
    if (!deviceID || picked.has(deviceID)) continue;
    const device = deviceByID.value.get(deviceID);
    picked.set(deviceID, {
      ...row,
      device_name: device?.device_name || `设备 ${deviceID}`,
      machine_code_hash: device?.machine_code_hash
    });
  }
  return Array.from(picked.values());
});

const deviceColumns: TableColumn[] = [
  { key: "id", label: "ID", width: 54 },
  { key: "machine_code_hash", label: "机器码哈希", width: "34%", mono: true, ellipsis: true, copy: true },
  { key: "device_name", label: "设备名", width: "18%", wrap: true },
  { key: "status", label: "状态", width: 76, format: (value: unknown) => statusLabel(value, { 1: "正常", 2: "禁用", 3: "已解绑" }) },
  { key: "last_seen_at", label: "最后出现", width: 126, wrap: true, format: formatDateTime },
  { key: "actions", label: "操作", width: 96, render: (row: Row) => renderDeviceActions(row) }
];
const sessionColumns: TableColumn[] = [
  { key: "id", label: "ID", width: 54 },
  { key: "device_name", label: "设备", width: "22%", ellipsis: true },
  { key: "device_id", label: "设备ID", width: 72 },
  { key: "status", label: "状态", width: 96, render: (row: Row) => renderSessionStatus(row) },
  { key: "client_ip", label: "IP", width: 110, ellipsis: true },
  { key: "last_heartbeat_at", label: "最后心跳", width: 126, wrap: true, format: formatDateTime },
  { key: "actions", label: "操作", width: 90, render: (row: Row) => renderSessionActions(row) }
];
const recordColumns: TableColumn[] = [
  { key: "id", label: "ID", width: 54 },
  { key: "event", label: "事件", width: "38%", ellipsis: true },
  { key: "client_ip", label: "IP", width: 120, ellipsis: true },
  { key: "created_at", label: "时间", width: 132, wrap: true, format: formatDateTime }
];
const valueColumns: TableColumn[] = [
  { key: "field_key", label: "字段", width: "34%", mono: true, ellipsis: true },
  { key: "field_value", label: "值", width: "66%", ellipsis: true, copy: true }
];
const verifyColumns: TableColumn[] = [
  { key: "id", label: "ID", width: 54 },
  { key: "result", label: "结果", width: 76, format: (value: unknown) => statusLabel(value, { 1: "成功", 2: "失败" }) },
  { key: "fail_reason", label: "失败原因", width: "30%", ellipsis: true },
  { key: "client_ip", label: "IP", width: 120, ellipsis: true },
  { key: "created_at", label: "时间", width: 132, wrap: true, format: formatDateTime }
];

async function list(path: string, params: Record<string, unknown>) {
  const data = await get<PageData<Row>>(path, { page: 1, page_size: 100, ...params });
  return data.list || [];
}

function renderTableContent(column: TableColumn, row: Row) {
  if (column.render) return column.render(row);
  const value = column.format ? column.format(row[column.key]) : displayValue(row[column.key]);
  if (column.copy) {
    return h("div", { class: "copy-cell" }, [
      renderTextNode(column, value),
      h(
        NButton,
        {
          size: "tiny",
          circle: true,
          quaternary: true,
          class: "cell-copy-button",
          title: "复制完整内容",
          onClick: (event: MouseEvent) => {
            event.stopPropagation();
            copyCellValue(value);
          }
        },
        {
          icon: () => h(NIcon, null, { default: () => h(CopyOutline) })
        }
      )
    ]);
  }
  return renderTextNode(column, value);
}

function renderTextNode(column: TableColumn, value: string) {
  if (column.mono || column.wrap || column.ellipsis || isLongColumn(column.key)) {
    return h(column.mono ? "code" : "span", { class: cellClass(column), title: String(value) }, value);
  }
  return h("span", { class: "table-cell-text" }, value);
}

function cellClass(column: TableColumn) {
  return {
    "table-cell-text": true,
    "table-cell-code": column.mono,
    "table-cell-wrap": column.wrap,
    "table-cell-ellipsis": column.ellipsis || (!column.wrap && isLongColumn(column.key))
  };
}

function isLongColumn(key: string) {
  return ["machine_code_hash", "field_value", "fail_reason", "license_key", "content", "remark"].includes(key);
}

function columnStyle(column: TableColumn) {
  if (!column.width) return {};
  return { width: typeof column.width === "number" ? `${column.width}px` : column.width };
}

async function copyCellValue(value: string) {
  const text = String(value ?? "").trim();
  if (!text || text === "-") {
    message.warning("暂无可复制内容");
    return;
  }
  try {
    await copyText(text);
    message.success("已复制完整内容");
  } catch {
    selectManualCopyText(text);
    message.warning("浏览器限制自动复制，已选中完整内容，可按 Ctrl+C");
  }
}

async function copyText(text: string) {
  window.focus();
  if (legacyCopyText(text)) return;
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      if (legacyCopyText(text)) return;
    }
  }
  throw new Error("复制失败");
}

function legacyCopyText(text: string) {
  const input = document.createElement("textarea");
  input.value = text;
  input.setAttribute("readonly", "readonly");
  input.style.position = "fixed";
  input.style.opacity = "0";
  input.style.pointerEvents = "none";
  document.body.appendChild(input);
  input.focus({ preventScroll: true });
  input.select();
  input.setSelectionRange(0, input.value.length);
  const copied = executeCopyCommand(text);
  document.body.removeChild(input);
  return copied;
}

function executeCopyCommand(text: string) {
  let copied = false;
  const onCopy = (event: ClipboardEvent) => {
    event.clipboardData?.setData("text/plain", text);
    event.preventDefault();
    copied = true;
  };
  document.addEventListener("copy", onCopy);
  const commandSucceeded = document.execCommand("copy");
  document.removeEventListener("copy", onCopy);
  return copied || commandSucceeded;
}

function selectManualCopyText(text: string) {
  manualCopyInput?.remove();
  manualCopyInput = document.createElement("textarea");
  manualCopyInput.value = text;
  manualCopyInput.setAttribute("readonly", "readonly");
  manualCopyInput.style.position = "fixed";
  manualCopyInput.style.left = "0";
  manualCopyInput.style.top = "0";
  manualCopyInput.style.width = "1px";
  manualCopyInput.style.height = "1px";
  manualCopyInput.style.opacity = "0.01";
  manualCopyInput.style.pointerEvents = "none";
  document.body.appendChild(manualCopyInput);
  manualCopyInput.focus({ preventScroll: true });
  manualCopyInput.select();
  manualCopyInput.setSelectionRange(0, manualCopyInput.value.length);
  window.setTimeout(() => {
    manualCopyInput?.remove();
    manualCopyInput = null;
  }, 8000);
}

async function load() {
  loading.value = true;
  try {
    const [base, deviceList, sessionList, latest, recordList, logList] = await Promise.all([
      get<Row>(`/admin/licenses/${licenseID.value}`),
      list("/admin/devices", { license_id: licenseID.value }),
      list("/admin/sessions", { license_id: licenseID.value }),
      get<{ record: Row | null; values: Row[] }>(`/admin/licenses/${licenseID.value}/collect/latest`),
      list(`/admin/licenses/${licenseID.value}/collect/records`, {}),
      list("/admin/verify-logs", { license_id: licenseID.value })
    ]);
    license.value = base;
    devices.value = deviceList;
    sessions.value = sessionList;
    latestCollect.value = latest;
    collectRecords.value = recordList;
    verifyLogs.value = logList;
  } catch (error) {
    message.error(error instanceof Error ? error.message : "加载详情失败");
  } finally {
    loading.value = false;
  }
}

function renderDeviceActions(row: Row) {
  const status = Number(row.status);
  const nodes: VNodeChild[] = [];
  if (status !== 1) {
    nodes.push(h(NButton, { size: "tiny", secondary: true, type: "success", onClick: () => updateDeviceStatus(row, 1) }, { default: () => "启用" }));
  }
  if (status !== 2) {
    nodes.push(h(NButton, { size: "tiny", secondary: true, type: "warning", onClick: () => updateDeviceStatus(row, 2) }, { default: () => "禁用" }));
  }
  if (status !== 3) {
    nodes.push(h(NButton, { size: "tiny", secondary: true, type: "error", onClick: () => unbindDevice(row) }, { default: () => "解绑" }));
  }
  return h(NSpace, { size: 6, wrap: false }, { default: () => nodes });
}

function renderSessionStatus(row: Row) {
  const status = Number(row.status);
  return h(
    NTag,
    { size: "small", round: true, bordered: false, type: sessionStatusType(status) },
    { default: () => statusLabel(status, { 1: "实时在线", 2: "已离线", 3: "已踢下线", 4: "心跳超时" }) }
  );
}

function renderSessionActions(row: Row) {
  if (Number(row.status) !== 1) {
    return h("span", { class: "cell-muted" }, "无操作");
  }
  return h(
    NButton,
    { size: "tiny", secondary: true, type: "error", onClick: () => revokeSession(row) },
    { default: () => "踢下线" }
  );
}

function sessionStatusType(status: number) {
  if (status === 1) return "success";
  if (status === 3) return "error";
  if (status === 4) return "warning";
  return "default";
}

async function updateDeviceStatus(row: Row, status: number) {
  try {
    await putAction(`/admin/devices/${row.id}/status`, { status });
    message.success(status === 1 ? "设备已启用" : "设备已禁用");
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : "操作失败");
  }
}

function unbindDevice(row: Row) {
  dialog.warning({
    title: "确认解绑",
    content: "只解绑当前设备：该机器码不再占用设备名额，并清空这台设备保存的公钥和在线会话。",
    positiveText: "解绑",
    negativeText: "取消",
    onPositiveClick: async () => {
      try {
        await putAction(`/admin/devices/${row.id}/unbind`);
        message.success("设备已解绑");
        await load();
      } catch (error) {
        message.error(error instanceof Error ? error.message : "解绑失败");
      }
    }
  });
}

function revokeSession(row: Row) {
  dialog.warning({
    title: "确认踢下线",
    content: "客户端下一次心跳会收到强制下线状态，并停止当前授权会话。",
    positiveText: "踢下线",
    negativeText: "取消",
    onPositiveClick: async () => {
      try {
        await putAction(`/admin/sessions/${row.id}/revoke`, { reason: "admin revoked from license detail" });
        message.success("已踢下线");
        await load();
      } catch (error) {
        message.error(error instanceof Error ? error.message : "踢下线失败");
      }
    }
  });
}

function timeValue(value: unknown) {
  if (!value) return 0;
  const parsed = new Date(String(value)).getTime();
  return Number.isFinite(parsed) ? parsed : 0;
}

onMounted(load);
</script>

<style scoped>
.detail-grid {
  display: grid;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  gap: 18px;
  overflow: hidden;
}

.detail-grid :deep(.n-tabs),
.detail-grid :deep(.n-tabs-pane-wrapper),
.detail-grid :deep(.n-tabs-pane-wrapper .n-tabs-pane-wrapper-scroll-content),
.detail-grid :deep(.n-tab-pane) {
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.detail-grid :deep(.n-tabs-pane-wrapper) {
  overflow: hidden;
}

.detail-panel {
  padding: 18px;
}

.license-summary {
  display: grid;
  gap: 16px;
}

.summary-main {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.summary-copy {
  display: grid;
  min-width: 0;
  gap: 7px;
}

.summary-label,
.metric-item span,
.info-item span {
  color: var(--app-muted);
  font-size: 12px;
  font-weight: 700;
  line-height: 18px;
}

.summary-main :deep(.n-tag) {
  align-self: flex-start;
}

.license-key {
  display: block;
  max-width: 100%;
  overflow-wrap: anywhere;
  color: var(--app-text-strong);
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  font-size: 18px;
  font-weight: 700;
  line-height: 28px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.metric-item {
  display: grid;
  gap: 6px;
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.metric-item strong {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--app-text-strong);
  font-size: 22px;
  font-weight: 800;
  line-height: 28px;
}

.info-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border: 1px solid var(--app-border);
  border-radius: 8px;
  overflow: hidden;
}

.info-item {
  display: grid;
  min-width: 0;
  gap: 5px;
  padding: 12px;
  border-right: 1px solid var(--app-border);
  border-bottom: 1px solid var(--app-border);
}

.info-item:nth-child(2n),
.info-item:last-child {
  border-right: 0;
}

.info-item:nth-last-child(-n + 2) {
  border-bottom: 0;
}

.info-item-wide {
  grid-column: span 1;
}

.info-item strong {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--app-text);
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
}

.detail-grid :deep(.detail-table-block) {
  display: grid;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  gap: 10px;
  margin-top: 12px;
  overflow: hidden;
}

.detail-grid :deep(.detail-table-scroll) {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.detail-grid :deep(.nested-table) {
  width: 100%;
  min-width: 0;
  table-layout: fixed;
}

.detail-grid :deep(.nested-table th),
.detail-grid :deep(.nested-table td) {
  min-width: 0;
  vertical-align: middle;
}

.detail-grid :deep(.nested-pager) {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.detail-grid :deep(.nested-pager-meta) {
  color: var(--app-muted);
  font-size: 12px;
}

.detail-grid :deep(.mobile-row-list) {
  display: none;
}

.latest-meta-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.latest-meta-item {
  display: grid;
  min-width: 0;
  gap: 6px;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.latest-meta-item span {
  color: var(--app-muted);
  font-size: 12px;
  font-weight: 700;
  line-height: 18px;
}

.latest-meta-item strong {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--app-text);
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
}

.detail-grid :deep(.table-cell-text) {
  display: block;
  max-width: 100%;
  min-width: 0;
  line-height: 20px;
}

.detail-grid :deep(.copy-cell) {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 4px;
}

.detail-grid :deep(.copy-cell .table-cell-text) {
  flex: 1 1 auto;
}

.detail-grid :deep(.cell-copy-button) {
  flex: 0 0 auto;
  color: var(--app-muted);
}

.detail-grid :deep(.table-cell-wrap) {
  overflow-wrap: anywhere;
  word-break: break-word;
  white-space: normal;
}

.detail-grid :deep(.table-cell-ellipsis) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-grid :deep(.table-cell-code) {
  color: var(--app-text);
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  font-size: 12px;
}

.detail-grid :deep(.empty-state) {
  display: grid;
  min-height: 128px;
  place-items: center;
  margin-top: 12px;
  border: 1px dashed var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.detail-grid :deep(.cell-muted) {
  color: var(--app-muted);
  font-size: 12px;
}

@media (max-width: 760px) {
  .detail-panel {
    padding: 14px;
  }

  .summary-main {
    align-items: stretch;
    flex-direction: column;
  }

  .license-key {
    font-size: 16px;
    line-height: 25px;
  }

  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric-item strong {
    font-size: 18px;
    line-height: 24px;
  }

  .info-list {
    grid-template-columns: 1fr;
  }

  .info-item,
  .info-item:nth-child(2n),
  .info-item:nth-last-child(-n + 2) {
    border-right: 0;
    border-bottom: 1px solid var(--app-border);
  }

  .info-item:last-child {
    border-bottom: 0;
  }

  .latest-meta-grid {
    grid-template-columns: 1fr;
    gap: 8px;
    margin-bottom: 10px;
  }

  .latest-meta-item {
    grid-template-columns: minmax(74px, 28%) minmax(0, 1fr);
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
  }

  .detail-grid :deep(.detail-table-scroll) {
    display: none;
  }

  .detail-grid :deep(.mobile-row-list) {
    display: grid;
    gap: 10px;
  }

  .detail-grid :deep(.mobile-row-card) {
    display: grid;
    gap: 0;
    overflow: hidden;
    border: 1px solid var(--app-border);
    border-radius: 8px;
    background: var(--app-surface);
  }

  .detail-grid :deep(.mobile-row-field) {
    display: grid;
    grid-template-columns: minmax(74px, 28%) minmax(0, 1fr);
    min-width: 0;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--app-border);
  }

  .detail-grid :deep(.mobile-row-field:last-child) {
    border-bottom: 0;
  }

  .detail-grid :deep(.mobile-row-label) {
    min-width: 0;
    color: var(--app-muted);
    font-size: 12px;
    font-weight: 700;
    line-height: 18px;
  }

  .detail-grid :deep(.mobile-row-value) {
    min-width: 0;
    color: var(--app-text);
    font-size: 13px;
    line-height: 20px;
  }

  .detail-grid :deep(.mobile-row-value .n-space) {
    flex-wrap: wrap !important;
    gap: 6px !important;
  }

  .detail-grid :deep(.nested-pager) {
    align-items: stretch;
    flex-direction: column;
  }

  .detail-grid :deep(.nested-pager .n-pagination) {
    justify-content: flex-start;
    row-gap: 8px;
  }

  .detail-grid :deep(.nested-pager .n-pagination .n-pagination-prefix),
  .detail-grid :deep(.nested-pager .n-pagination .n-pagination-suffix) {
    margin-inline: 0;
  }

  .detail-grid :deep(.table-cell-code) {
    font-size: 11px;
    line-height: 18px;
  }
}
</style>
