import axios from "axios";
import { useAuthStore } from "@/stores/auth";

export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface PageData<T> {
  list: T[];
  page: number;
  page_size: number;
  total: number;
}

export const http = axios.create({
  baseURL: "/api",
  timeout: 15000
});

http.interceptors.request.use((config) => {
  const token = localStorage.getItem("xnauth_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => {
    const body = response.data as ApiResponse;
    if (body && typeof body.code === "number" && body.code !== 0) {
      return Promise.reject(new Error(body.message || "request failed"));
    }
    return response;
  },
  (error) => {
    if (error.response?.status === 401) {
      const auth = useAuthStore();
      auth.logout();
    }
    const message = error.response?.data?.message || error.message || "request failed";
    return Promise.reject(new Error(message));
  }
);

export async function get<T>(url: string, params?: Record<string, unknown>): Promise<T> {
  const response = await http.get<ApiResponse<T>>(url, { params });
  return response.data.data;
}

export async function post<T>(url: string, body?: Record<string, unknown>): Promise<T> {
  const response = await http.post<ApiResponse<T>>(url, body ?? {});
  return response.data.data;
}

export async function put<T>(url: string, body?: Record<string, unknown>): Promise<T> {
  const response = await http.put<ApiResponse<T>>(url, body ?? {});
  return response.data.data;
}

export async function del<T>(url: string): Promise<T> {
  const response = await http.delete<ApiResponse<T>>(url);
  return response.data.data;
}

