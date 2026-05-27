<template>
  <ResourcePage
    title="验证日志"
    subtitle="查看授权验证成功和失败原因。"
    endpoint="/admin/verify-logs"
    searchable
    search-placeholder="搜索卡密、机器码、IP、版本或失败原因"
    :filters="filters"
    :fields="fields"
    :can-create="false"
    :can-edit="false"
  />
</template>

<script setup lang="ts">
import ResourcePage from "@/components/ResourcePage.vue";
import type { FieldConfig, ResourceFilter } from "@/types/resource";

const resultOptions = [
  { label: "成功", value: 1 },
  { label: "失败", value: 2 }
];

const failReasonOptions = [
  { label: "应用不存在", value: "app_not_found" },
  { label: "应用已禁用", value: "app_disabled" },
  { label: "卡密不存在", value: "license_not_found" },
  { label: "卡密已过期", value: "license_expired" },
  { label: "卡密已冻结", value: "license_frozen" },
  { label: "卡密已封禁", value: "license_banned" },
  { label: "卡密状态异常", value: "license_invalid" },
  { label: "版本不支持", value: "version_not_supported" },
  { label: "设备已禁用", value: "device_disabled" },
  { label: "设备不存在", value: "device_not_found" },
  { label: "设备状态异常", value: "device_invalid" },
  { label: "设备密钥未绑定", value: "device_key_unbound" },
  { label: "设备密钥不匹配", value: "device_key_mismatch" },
  { label: "机器码不匹配", value: "machine_code_mismatch" },
  { label: "设备数超限", value: "max_devices_exceeded" },
  { label: "在线数超限", value: "max_online_exceeded" },
  { label: "会话不存在", value: "session_not_found" },
  { label: "会话已踢下线", value: "session_revoked" },
  { label: "会话状态异常", value: "session_invalid" },
  { label: "会话已超时", value: "session_timeout" },
  { label: "会话卡密不匹配", value: "session_license_mismatch" },
  { label: "会话设备不匹配", value: "session_device_mismatch" },
  { label: "请求参数无效", value: "invalid_params" },
  { label: "Nonce 不匹配", value: "nonce_mismatch" },
  { label: "Nonce 无效", value: "invalid_nonce" },
  { label: "重复请求", value: "replay_request" },
  { label: "服务端错误", value: "server_error" }
];

const filters: ResourceFilter[] = [
  {
    key: "result",
    label: "验证结果",
    placeholder: "全部结果",
    options: resultOptions
  },
  {
    key: "fail_reason",
    label: "失败原因",
    placeholder: "全部失败原因",
    options: failReasonOptions
  }
];

const fields: FieldConfig[] = [
  { key: "id", label: "ID", form: false, width: "70px" },
  { key: "license_id", label: "卡密ID", form: false },
  { key: "license_key", label: "卡密", form: false },
  { key: "machine_code_hash", label: "机器码哈希", form: false },
  {
    key: "result",
    label: "结果",
    type: "select",
    form: false,
    options: resultOptions
  },
  { key: "fail_reason", label: "失败原因", type: "select", form: false, options: failReasonOptions },
  { key: "client_ip", label: "客户端IP", form: false },
  { key: "created_at", label: "时间", type: "datetime", form: false }
];
</script>
