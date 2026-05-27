<template>
  <section class="resource-page">
    <div class="resource-head">
      <div>
        <h2 class="page-title">{{ title }}</h2>
        <p v-if="subtitle" class="page-subtitle">{{ subtitle }}</p>
      </div>
      <div class="toolbar-actions">
        <n-input
          v-if="searchable"
          v-model:value="keyword"
          clearable
          size="medium"
          class="search-input"
          :placeholder="searchPlaceholder"
          @keyup.enter="reload"
          @clear="reload"
        >
          <template #prefix>
            <n-icon><SearchOutline /></n-icon>
          </template>
        </n-input>
        <template v-for="filter in filters" :key="filter.key">
          <n-select
            v-if="filter.type !== 'text'"
            v-model:value="filterValues[filter.key]"
            clearable
            class="filter-control"
            :placeholder="filter.placeholder || filter.label"
            :options="filterOptions(filter)"
            @update:value="onFilterChange"
          />
          <n-input
            v-else
            v-model:value="filterValues[filter.key]"
            clearable
            class="filter-control"
            :placeholder="filter.placeholder || filter.label"
            @keyup.enter="reload"
            @clear="reload"
          />
        </template>
        <n-button
          v-for="action in headerActions"
          :key="action.label"
          :type="action.type"
          :secondary="headerActionSecondary(action)"
          :ghost="action.ghost"
          @click="runHeaderAction(action)"
        >
          {{ action.label }}
        </n-button>
        <n-button v-if="canCreate" type="primary" @click="openCreate">
          <template #icon>
            <n-icon><AddOutline /></n-icon>
          </template>
          新增
        </n-button>
      </div>
    </div>

    <n-card class="resource-card" :bordered="false">
      <template #header>
        <div class="table-title">
          <span>数据列表</span>
          <n-tag size="small" round :bordered="false">{{ total }} 条</n-tag>
        </div>
      </template>
      <n-data-table
        remote
        size="small"
        :bordered="false"
        :single-line="false"
        :columns="columns"
        :data="rows"
        :loading="loading"
        :row-key="rowKey"
        :row-props="rowProps"
        :pagination="false"
        :scroll-x="scrollX"
      />
      <template #footer>
        <div class="pager">
          <span class="pager-meta">第 {{ page }} 页，共 {{ total }} 条</span>
          <n-pagination
            v-model:page="page"
            v-model:page-size="pageSize"
            show-size-picker
            :page-sizes="[10, 20, 50, 100]"
            :item-count="total"
            @update:page="load"
            @update:page-size="onPageSizeChange"
          />
        </div>
      </template>
    </n-card>

    <n-modal
      v-model:show="modalVisible"
      preset="card"
      :title="editing ? '编辑' : '新增'"
      class="form-modal"
      :style="{ width: 'min(760px, calc(100vw - 28px))' }"
    >
      <n-form label-placement="top" :model="formModel" :show-require-mark="false">
        <n-grid cols="1 s:1 m:2" responsive="screen" :x-gap="16">
          <n-form-item-gi
            v-for="field in formFields"
            :key="field.key"
            :span="field.type === 'textarea' ? '1 s:1 m:2' : 1"
            :label="field.label"
            :required="field.required"
          >
            <n-input-number
              v-if="field.type === 'number'"
              v-model:value="formModel[field.key]"
              clearable
              class="full"
              :min="field.min"
              :max="field.max"
              :precision="field.precision"
            />
            <n-select
              v-else-if="field.type === 'select'"
              v-model:value="formModel[field.key]"
              clearable
              :options="selectOptions(field)"
            />
            <n-date-picker
              v-else-if="field.type === 'datetime'"
              v-model:formatted-value="formModel[field.key]"
              clearable
              class="full"
              type="datetime"
              value-format="yyyy-MM-dd HH:mm:ss"
            />
            <n-input
              v-else-if="field.type === 'password'"
              v-model:value="formModel[field.key]"
              type="password"
              show-password-on="click"
              clearable
            />
            <n-input
              v-else-if="field.type === 'textarea'"
              v-model:value="formModel[field.key]"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 8 }"
            />
            <n-input-group v-else-if="field.key === 'license_key'" class="full license-copy-group">
              <n-input v-model:value="formModel[field.key]" clearable />
              <n-button
                secondary
                title="复制卡密"
                @pointerdown.prevent="copyFieldValue(field.key, $event)"
                @keydown.enter.prevent="copyFieldValue(field.key, $event)"
                @keydown.space.prevent="copyFieldValue(field.key, $event)"
              >
                <template #icon>
                  <n-icon><CopyOutline /></n-icon>
                </template>
                复制
              </n-button>
            </n-input-group>
            <n-input v-else v-model:value="formModel[field.key]" clearable />
          </n-form-item-gi>
        </n-grid>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="modalVisible = false">取消</n-button>
          <n-button type="primary" :loading="saving" @click="submit">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref, watch, type VNodeChild } from "vue";
import {
  NButton,
  NCard,
  NDataTable,
  NDatePicker,
  NDropdown,
  NForm,
  NFormItemGi,
  NGrid,
  NIcon,
  NInput,
  NInputGroup,
  NInputNumber,
  NModal,
  NPagination,
  NSelect,
  NSpace,
  NTag,
  type DataTableColumns,
  type SelectOption,
  useDialog,
  useMessage
} from "naive-ui";
import { AddOutline, CopyOutline, SearchOutline } from "@vicons/ionicons5";
import { createRecord, deleteRecord, listRecords, updateRecord } from "@/api/admin";
import { useAppStore } from "@/stores/app";
import type { FieldConfig, HeaderAction, ResourceFilter, RowAction } from "@/types/resource";
import { displayValue, formatDateTime } from "@/utils/format";

const props = withDefaults(
  defineProps<{
    title: string;
    subtitle?: string;
    endpoint: string;
    fields: FieldConfig[];
    appScoped?: boolean;
    canCreate?: boolean;
    canEdit?: boolean;
    canDelete?: boolean;
    searchable?: boolean;
    searchPlaceholder?: string;
    filters?: ResourceFilter[];
    headerActions?: HeaderAction[];
    actions?: RowAction[];
    defaultParams?: Record<string, unknown>;
    createDefaults?: () => Record<string, unknown>;
    prepareCreateBody?: (body: Record<string, unknown>) => Record<string, unknown>;
  }>(),
  {
    subtitle: "",
    appScoped: true,
    canCreate: true,
    canEdit: true,
    canDelete: false,
    searchable: false,
    searchPlaceholder: "搜索",
    filters: () => [],
    headerActions: () => [],
    actions: () => [],
    defaultParams: () => ({})
  }
);

const appStore = useAppStore();
const message = useMessage();
const dialog = useDialog();
const loading = ref(false);
const saving = ref(false);
const rows = ref<Record<string, unknown>[]>([]);
const page = ref(1);
const pageSize = ref(20);
const total = ref(0);
const keyword = ref("");
const filterValues = ref<Record<string, any>>({});
const modalVisible = ref(false);
const editing = ref<Record<string, unknown> | null>(null);
const formModel = ref<Record<string, any>>({});

const appFieldOptions = computed(() => appStore.apps.map((item) => ({ label: item.app_name, value: item.id })));
const hasAppField = computed(() => props.fields.some((field) => field.key === "app_id"));
const appField = computed<FieldConfig>(() => ({
  key: "app_id",
  label: "所属应用",
  type: "select",
  required: true,
  width: "150px",
  options: appFieldOptions.value
}));
const tableFields = computed(() => {
  const fields = props.fields.filter((field) => field.table !== false);
  if (props.appScoped && !appStore.currentAppId && !hasAppField.value) return [appField.value, ...fields];
  return fields;
});
const formFields = computed(() => {
  const fields = props.fields
    .filter((field) => field.form !== false)
    .filter((field) => (editing.value ? !field.createOnly : !field.editOnly));
  if (props.appScoped && !hasAppField.value) return [appField.value, ...fields];
  return fields;
});
const visibleActions = computed(() => props.actions);
const scrollX = computed(() => {
  const fieldWidth = tableFields.value.reduce((sum, field) => sum + columnWidth(field), 0);
  return Math.max(860, fieldWidth + (hasActions.value ? actionColumnWidth.value : 0));
});
const hasActions = computed(() => props.canEdit || props.canDelete || visibleActions.value.length > 0);
const actionColumnWidth = computed(() => {
  const buttonCount = Number(props.canEdit) + Number(props.canDelete) + Math.min(visibleActions.value.length, 1);
  return Math.max(132, 46 + buttonCount * 50);
});

const columns = computed<DataTableColumns<Record<string, unknown>>>(() => {
  const base: DataTableColumns<Record<string, unknown>> = tableFields.value.map((field) => ({
    title: field.label,
    key: field.key,
    width: columnWidth(field),
    ellipsis: {
      tooltip: true
    },
    render: (row) => renderField(field, row)
  }));
  if (!hasActions.value) return base;
  base.push({
    title: "操作",
    key: "actions",
    width: actionColumnWidth.value,
    fixed: "right",
    render: (row) => renderActions(row)
  });
  return base;
});

async function load() {
  loading.value = true;
  try {
    const params: Record<string, unknown> = {
      page: page.value,
      page_size: pageSize.value,
      ...props.defaultParams
    };
    if (props.appScoped && appStore.currentAppId) {
      params.app_id = appStore.currentAppId;
    }
    if (props.searchable && keyword.value.trim()) {
      params.keyword = keyword.value.trim();
      params.license_key = keyword.value.trim();
      params.machine_code_hash = keyword.value.trim();
    }
    for (const [key, value] of Object.entries(filterValues.value)) {
      if (value === "" || value === null || value === undefined) continue;
      params[key] = value;
    }
    const data = await listRecords<Record<string, unknown>>(props.endpoint, params);
    rows.value = data.list || [];
    total.value = data.total || 0;
  } catch (error) {
    message.error(error instanceof Error ? error.message : "加载失败");
  } finally {
    loading.value = false;
  }
}

function reload() {
  page.value = 1;
  load();
}

function onPageSizeChange() {
  page.value = 1;
  load();
}

function onFilterChange() {
  reload();
}

function openCreate() {
  editing.value = null;
  formModel.value = props.createDefaults ? { ...props.createDefaults() } : {};
  if (props.appScoped && appStore.currentAppId) {
    formModel.value.app_id = appStore.currentAppId;
  }
  modalVisible.value = true;
}

function openEdit(row: Record<string, unknown>) {
  editing.value = row;
  formModel.value = normalizeFormModel(row);
  modalVisible.value = true;
}

async function copyFieldValue(key: string, event?: Event) {
  const text = String(formModel.value[key] ?? "").trim();
  if (!text) {
    message.warning("没有可复制的卡密");
    return;
  }
  const sourceInput = findSourceInput(event);
  try {
    await copyText(text, sourceInput);
    message.success("已复制卡密");
  } catch (error) {
    sourceInput?.select();
    message.warning(error instanceof Error ? error.message : "复制失败");
  }
}

async function submit() {
  saving.value = true;
  try {
    let body = normalizeBody(formModel.value);
    if (!editing.value && props.prepareCreateBody) {
      body = normalizeBody(props.prepareCreateBody(body));
    }
    if (props.appScoped && !body.app_id) {
      message.warning("请先选择所属应用");
      return;
    }
    if (editing.value) {
      await updateRecord(props.endpoint, Number(editing.value.id), body);
      message.success("已更新");
    } else {
      await createRecord(props.endpoint, body);
      message.success("已创建");
    }
    modalVisible.value = false;
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : "保存失败");
  } finally {
    saving.value = false;
  }
}

function remove(row: Record<string, unknown>) {
  dialog.warning({
    title: "确认删除",
    content: `确定删除 ID ${row.id} 吗？`,
    positiveText: "删除",
    negativeText: "取消",
    onPositiveClick: async () => {
      await deleteRecord(props.endpoint, Number(row.id));
      message.success("已删除");
      load();
    }
  });
}

function runAction(action: RowAction, row: Record<string, unknown>) {
  const execute = async () => {
    try {
      await action.run(row);
      if (action.successMessage !== false) {
        message.success(action.successMessage || "操作完成");
      }
      if (action.reload !== false) {
        load();
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : "操作失败");
    }
  };
  if (action.confirm) {
    dialog.warning({
      title: "确认操作",
      content: action.confirm,
      positiveText: action.label,
      negativeText: "取消",
      onPositiveClick: execute
    });
    return;
  }
  execute();
}

function runHeaderAction(action: HeaderAction) {
  const execute = async () => {
    try {
      await action.run();
      if (action.successMessage !== false) {
        message.success(action.successMessage || "操作完成");
      }
      if (action.reload !== false) {
        load();
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : "操作失败");
    }
  };
  if (action.confirm) {
    dialog.warning({
      title: "确认操作",
      content: action.confirm,
      positiveText: action.label,
      negativeText: "取消",
      onPositiveClick: execute
    });
    return;
  }
  execute();
}

function renderField(field: FieldConfig, row: Record<string, unknown>) {
  const value = row[field.key];
  if (field.key === "app_id") {
    return h("span", { class: "app-cell" }, appLabel(value));
  }
  if (field.type === "select") {
    return h(
      NTag,
      { size: "small", round: true, bordered: false, type: optionTagType(value) },
      { default: () => optionLabel(field, value) }
    );
  }
  if (field.type === "datetime") {
    return h("span", { class: "cell-muted" }, formatDateTime(value));
  }
  if (field.key === "license_key") {
    return h("code", { class: "license-code" }, displayValue(value));
  }
  return h("span", displayValue(value));
}

function renderActions(row: Record<string, unknown>) {
  const nodes: VNodeChild[] = [];
  if (props.canEdit) {
    nodes.push(h(NButton, { size: "tiny", secondary: true, onClick: () => openEdit(row) }, { default: () => "编辑" }));
  }
  if (props.canDelete) {
    nodes.push(h(NButton, { size: "tiny", type: "error", secondary: true, onClick: () => remove(row) }, { default: () => "删除" }));
  }
  const rowActions = visibleActions.value.filter((action) => !action.show || action.show(row));
  if (rowActions.length === 1) {
    const action = rowActions[0];
    nodes.push(
      h(NButton, { size: "tiny", type: action.type, secondary: !action.ghost, ghost: action.ghost, onClick: () => runAction(action, row) }, {
        default: () => action.label
      })
    );
  } else if (rowActions.length > 1) {
    nodes.push(
      h(
        NDropdown,
        {
          trigger: "click",
          options: rowActions.map((action, index) => ({ label: action.label, key: String(index) })),
          onSelect: (key: string) => runAction(rowActions[Number(key)], row)
        },
        {
          default: () => h(NButton, { size: "tiny", secondary: true }, { default: () => "更多" })
        }
      )
    );
  }
  return h(NSpace, { size: 6, wrap: false }, { default: () => nodes });
}

function columnWidth(field: FieldConfig) {
  if (!field.width) return 112;
  const parsed = Number.parseInt(String(field.width), 10);
  return Number.isFinite(parsed) ? parsed : 112;
}

function optionTagType(value: unknown) {
  const numeric = Number(value);
  if (numeric === 1) return "success";
  if (numeric === 2) return "warning";
  if (numeric === 3) return "warning";
  if (numeric === 4) return "error";
  if (numeric === 0) return "info";
  return "default";
}

function optionLabel(field: FieldConfig, value: unknown) {
  const found = field.options?.find((item) => item.value === value || String(item.value) === String(value));
  return found?.label || displayValue(value);
}

function selectOptions(field: FieldConfig): SelectOption[] {
  return (field.options || []).map((item) => ({ label: item.label, value: item.value }));
}

function filterOptions(filter: ResourceFilter): SelectOption[] {
  return (filter.options || []).map((item) => ({ label: item.label, value: item.value }));
}

function appLabel(value: unknown) {
  const found = appStore.apps.find((item) => String(item.id) === String(value));
  return found?.app_name || displayValue(value);
}

function normalizeBody(body: Record<string, unknown>) {
  const next: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(body)) {
    if (value === "") continue;
    next[key] = value;
  }
  return next;
}

async function copyText(text: string, sourceInput?: HTMLInputElement | null) {
  if (legacyCopyText(text, sourceInput)) return;
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      throw new Error("浏览器限制自动复制，已选中卡密，可按 Ctrl+C");
    }
  }
  throw new Error("浏览器限制自动复制，已选中卡密，可按 Ctrl+C");
}

function legacyCopyText(text: string, sourceInput?: HTMLInputElement | null) {
  if (sourceInput) {
    sourceInput.focus({ preventScroll: true });
    sourceInput.select();
    sourceInput.setSelectionRange(0, sourceInput.value.length);
    if (executeCopyCommand(text)) return true;
  }

  const input = document.createElement("textarea");
  input.value = text;
  input.setAttribute("readonly", "readonly");
  input.style.position = "fixed";
  input.style.opacity = "0";
  input.style.pointerEvents = "none";
  document.body.appendChild(input);
  input.focus();
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

function findSourceInput(event?: Event) {
  const target = event?.currentTarget;
  if (!(target instanceof HTMLElement)) return null;
  return target.closest(".license-copy-group")?.querySelector("input") ?? null;
}

function normalizeFormModel(row: Record<string, unknown>) {
  const next: Record<string, unknown> = { ...row };
  for (const field of formFields.value) {
    if (field.type === "datetime") {
      next[field.key] = toDatePickerValue(next[field.key]);
    }
  }
  return next;
}

function toDatePickerValue(value: unknown) {
  if (!value) return null;
  const text = String(value);
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}(:\d{2})?$/.test(text)) {
    return text.length === 16 ? `${text}:00` : text;
  }
  const date = new Date(text);
  if (Number.isNaN(date.getTime())) return text;
  const pad = (input: number) => String(input).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(
    date.getMinutes()
  )}:${pad(date.getSeconds())}`;
}

function headerActionSecondary(action: HeaderAction) {
  if (typeof action.secondary === "boolean") return action.secondary;
  return action.type !== "primary";
}

function rowKey(row: Record<string, unknown>) {
  return String(row.id);
}

function rowProps(row: Record<string, unknown>) {
  return {
    onDblclick: () => {
      if (props.canEdit) openEdit(row);
    }
  };
}

watch(
  () => appStore.currentAppId,
  () => {
    if (props.appScoped) reload();
  }
);

onMounted(load);

defineExpose({ reload });
</script>

<style scoped>
.resource-page {
  display: grid;
  width: 100%;
  min-width: 0;
  gap: 16px;
}

.resource-head {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.resource-head > div:first-child {
  min-width: 0;
}

.resource-card {
  width: 100%;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
  box-shadow: var(--app-shadow-sm);
}

.table-title {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--app-text);
  font-size: 15px;
  font-weight: 700;
}

.search-input {
  width: 260px;
  min-width: 0;
}

.filter-control {
  width: 150px;
  min-width: 0;
}

.toolbar-actions {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.app-cell {
  color: var(--app-text);
  font-size: 13px;
  font-weight: 600;
}

.cell-muted {
  color: var(--app-muted);
}

.license-code {
  color: var(--app-text);
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  font-size: 12px;
}

.pager {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.pager-meta {
  color: var(--app-muted);
  font-size: 12px;
}

.form-modal {
  max-width: 760px;
}

.full {
  width: 100%;
}

.license-copy-group {
  display: flex;
}

.license-copy-group .n-input {
  flex: 1 1 auto;
}

@media (max-width: 760px) {
  .resource-head {
    align-items: stretch;
    flex-direction: column;
    gap: 12px;
  }

  .search-input {
    width: 100%;
  }

  .filter-control {
    width: 100%;
  }

  .toolbar-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
    min-width: 0;
    gap: 8px;
  }

  .toolbar-actions .search-input {
    grid-column: 1 / -1;
  }

  .toolbar-actions .n-button {
    width: 100%;
    min-width: 40px;
  }

  .pager {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
