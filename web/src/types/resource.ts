import type { ButtonProps } from "naive-ui";

export interface FieldOption {
  label: string;
  value: string | number;
}

export type FieldType = "text" | "password" | "number" | "textarea" | "select" | "datetime";

export interface FieldConfig {
  key: string;
  label: string;
  type?: FieldType;
  required?: boolean;
  table?: boolean;
  form?: boolean;
  createOnly?: boolean;
  editOnly?: boolean;
  min?: number;
  max?: number;
  precision?: number;
  width?: string;
  options?: FieldOption[];
}

export interface ResourceFilter {
  key: string;
  label: string;
  type?: "select" | "text";
  placeholder?: string;
  options?: FieldOption[];
}

export interface RowAction {
  label: string;
  type?: ButtonProps["type"];
  ghost?: boolean;
  confirm?: string;
  reload?: boolean;
  successMessage?: string | false;
  show?: (row: Record<string, unknown>) => boolean;
  run: (row: Record<string, unknown>) => Promise<void>;
}

export interface HeaderAction {
  label: string;
  type?: ButtonProps["type"];
  secondary?: boolean;
  ghost?: boolean;
  confirm?: string;
  reload?: boolean;
  successMessage?: string | false;
  run: () => Promise<unknown>;
}
