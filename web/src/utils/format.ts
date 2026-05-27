export function displayValue(value: unknown): string {
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "boolean") return value ? "是" : "否";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

export function formatDateTime(value: unknown): string {
  if (!value) return "-";
  const text = String(value);
  return text.replace("T", " ").replace(/\.\d+Z$/, "").replace(/([+-]\d{2}:\d{2})$/, "");
}

export function statusLabel(value: unknown, map: Record<number, string>): string {
  const key = Number(value);
  return map[key] || displayValue(value);
}
