<template>
  <section class="stats-page">
    <div class="stats-head">
      <div>
        <h2 class="page-title">数据统计</h2>
        <p class="page-subtitle">基于收集字段配置生成分布统计和数额统计，支持按设备在线状态过滤数据来源。</p>
      </div>
      <div class="stats-controls">
        <n-select v-model:value="source" class="source-select" :options="sourceOptions" />
        <n-button type="primary" secondary @click="load">
          <template #icon>
            <n-icon><RefreshOutline /></n-icon>
          </template>
          刷新
        </n-button>
      </div>
    </div>

    <n-spin :show="loading">
      <n-empty
        v-if="!fields.length"
        class="stats-empty"
        description="暂无可统计字段，请在收集字段中启用统计并选择统计方式"
      />

      <n-grid v-else cols="1 l:2" responsive="screen" :x-gap="16" :y-gap="16">
        <n-gi v-for="field in fields" :key="`${field.app_id}-${field.field_key}`">
          <n-card class="stat-card" :bordered="false">
            <div class="stat-card-head">
              <div>
                <span class="stat-app">应用 #{{ field.app_id }}</span>
                <h3>{{ field.field_name }}</h3>
                <p>{{ field.field_key }}</p>
              </div>
              <n-tag round :bordered="false" :type="field.stat_type === 'sum' ? 'warning' : 'info'">
                {{ field.stat_type === "sum" ? "数额统计" : "分布统计" }}
              </n-tag>
            </div>

            <StatChart :field="field" />

            <div class="stat-foot">
              <span>样本 {{ field.total_count }} 条</span>
              <strong v-if="field.stat_type === 'sum'">累计 {{ formatNumber(field.numeric_sum || 0) }}</strong>
              <small v-if="field.number_note">{{ field.number_note }}</small>
            </div>
          </n-card>
        </n-gi>
      </n-grid>
    </n-spin>
  </section>
</template>

<script setup lang="ts">
import { defineComponent, h, nextTick, onBeforeUnmount, onMounted, ref, watch, type PropType } from "vue";
import { BarChart as EChartBarChart, PieChart } from "echarts/charts";
import { GridComponent, LegendComponent, TitleComponent, TooltipComponent } from "echarts/components";
import { init, use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import type { EChartsOption } from "echarts";
import { NButton, NCard, NEmpty, NGi, NGrid, NIcon, NSelect, NSpin, NTag, useMessage } from "naive-ui";
import { RefreshOutline } from "@vicons/ionicons5";
import { get } from "@/api/http";
import { useAppStore } from "@/stores/app";

interface StatItem {
  label: string;
  value: number;
  count?: number;
}

interface StatField {
  field_id: number;
  app_id: number;
  field_key: string;
  field_name: string;
  stat_type: "distribution" | "sum";
  total_count: number;
  numeric_sum?: number;
  number_note?: string;
  items: StatItem[];
}

interface StatsResp {
  source: string;
  fields: StatField[];
}

use([EChartBarChart, PieChart, GridComponent, LegendComponent, TitleComponent, TooltipComponent, CanvasRenderer]);

const StatChart = defineComponent({
  name: "StatChart",
  props: {
    field: { type: Object as PropType<StatField>, required: true }
  },
  setup(props) {
    const chartEl = ref<HTMLElement | null>(null);
    let chart: ReturnType<typeof init> | null = null;
    let observer: ResizeObserver | null = null;
    let resizeHandler: (() => void) | null = null;

    function render() {
      if (!chartEl.value) return;
      const instance = chart || init(chartEl.value);
      chart = instance;
      instance.setOption(chartOption(props.field), true);
      instance.resize();
    }

    onMounted(async () => {
      await nextTick();
      render();
      if (chartEl.value) {
        const ResizeObserverCtor = window.ResizeObserver;
        if (typeof ResizeObserverCtor !== "undefined") {
          observer = new ResizeObserverCtor(() => chart?.resize());
          observer.observe(chartEl.value);
        } else {
          resizeHandler = () => chart?.resize();
          window.addEventListener("resize", resizeHandler);
        }
      }
    });

    watch(
      () => props.field,
      async () => {
        await nextTick();
        render();
      },
      { deep: true }
    );

    onBeforeUnmount(() => {
      observer?.disconnect();
      if (resizeHandler) window.removeEventListener("resize", resizeHandler);
      chart?.dispose();
      chart = null;
    });

    return () => h("div", { ref: chartEl, class: "stat-chart" });
  }
});

const appStore = useAppStore();
const message = useMessage();
const loading = ref(false);
const source = ref<"all" | "online" | "offline">("all");
const fields = ref<StatField[]>([]);

const sourceOptions = [
  { label: "全部设备", value: "all" },
  { label: "在线设备", value: "online" },
  { label: "离线设备", value: "offline" }
];

async function load() {
  loading.value = true;
  try {
    const data = await get<StatsResp>("/admin/collect/summary", {
      source: source.value,
      app_id: appStore.currentAppId || undefined
    });
    fields.value = data.fields || [];
  } catch (error) {
    const detail = error instanceof Error ? error.message : "加载统计失败";
    message.error(detail === "Network Error" ? "加载统计失败，请确认手机浏览器没有拦截统计接口" : detail);
  } finally {
    loading.value = false;
  }
}

function chartOption(field: StatField): EChartsOption {
  const textColor = cssVar("--app-text");
  const mutedColor = cssVar("--app-muted");
  const borderColor = cssVar("--app-border");
  const primaryColor = cssVar("--app-primary");

  if (!field.items.length || field.total_count === 0) {
    return {
      title: {
        text: "暂无数据",
        left: "center",
        top: "middle",
        textStyle: { color: mutedColor, fontSize: 13, fontWeight: 500 }
      }
    };
  }

  if (field.stat_type === "sum") {
    return {
      color: [primaryColor],
      tooltip: { trigger: "axis" },
      grid: { left: 18, right: 18, top: 28, bottom: 28, containLabel: true },
      xAxis: {
        type: "category",
        data: [field.field_name],
        axisLine: { lineStyle: { color: borderColor } },
        axisLabel: { color: mutedColor }
      },
      yAxis: {
        type: "value",
        splitLine: { lineStyle: { color: borderColor } },
        axisLabel: { color: mutedColor }
      },
      series: [
        {
          type: "bar",
          barMaxWidth: 48,
          data: [field.numeric_sum || 0],
          label: {
            show: true,
            position: "top",
            color: textColor,
            formatter: (params: any) => formatNumber(Number(params.value || 0))
          }
        }
      ]
    };
  }

  return {
    color: ["#2f6bff", "#14b879", "#f59e0b", "#8b5cf6", "#ef4444", "#06b6d4", "#64748b"],
    tooltip: { trigger: "item", formatter: "{b}<br/>数量：{c}<br/>占比：{d}%" },
    legend: {
      type: "scroll",
      bottom: 0,
      textStyle: { color: mutedColor }
    },
    series: [
      {
        name: field.field_name,
        type: "pie",
        radius: ["44%", "70%"],
        center: ["50%", "43%"],
        avoidLabelOverlap: true,
        label: {
          color: textColor,
          formatter: "{b}: {c}"
        },
        data: field.items.map((item) => ({ name: item.label, value: item.value }))
      }
    ]
  };
}

function cssVar(name: string) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(value);
}

onMounted(load);
watch(() => [appStore.currentAppId, source.value], load);
</script>

<style scoped>
.stats-page {
  display: grid;
  gap: 16px;
}

.stats-head {
  display: flex;
  min-width: 0;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
}

.stats-head > div:first-child {
  min-width: 0;
}

.stats-controls {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.source-select {
  width: 160px;
}

.stats-empty {
  min-height: 260px;
  border: 1px dashed var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
}

.stat-card {
  height: 100%;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
  box-shadow: var(--app-shadow-sm);
}

.stat-card-head {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.stat-card-head h3 {
  margin: 5px 0 2px;
  color: var(--app-text-strong);
  font-size: 17px;
  line-height: 24px;
}

.stat-card-head p,
.stat-app,
.stat-foot,
.stat-foot small {
  color: var(--app-muted);
  font-size: 12px;
}

.stat-card-head p {
  margin: 0;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
}

.stat-chart {
  width: 100%;
  height: 300px;
  margin-top: 8px;
}

.stat-foot {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--app-border);
}

.stat-foot strong {
  color: var(--app-text-strong);
  font-size: 14px;
}

.stat-foot small {
  flex: 1 1 100%;
  line-height: 18px;
}

@media (max-width: 760px) {
  .stats-head {
    align-items: stretch;
    flex-direction: column;
  }

  .stats-controls {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .source-select {
    width: 100%;
  }

  .stat-card-head,
  .stat-foot {
    align-items: stretch;
    flex-direction: column;
  }

  .stat-chart {
    height: 260px;
  }
}
</style>
