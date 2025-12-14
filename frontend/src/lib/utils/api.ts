import { get } from 'svelte/store';
import { token } from '$stores/auth';

// API base URL (empty for same-origin, can be configured for different environments)
export const API_BASE = '';

// Types
export interface ApiResponse<T = unknown> {
  data?: T;
  error?: string;
  status: number;
  ok: boolean;
}

export interface ApiError {
  error: string;
  message?: string;
  code?: string;
  details?: Record<string, unknown>;
}

// Get auth token
function getAuthToken(): string | null {
  return get(token);
}

// Build headers with auth
function buildHeaders(customHeaders?: HeadersInit): Headers {
  const headers = new Headers({
    'Content-Type': 'application/json',
  });

  const authToken = getAuthToken();
  if (authToken) {
    headers.set('Authorization', `Bearer ${authToken}`);
  }

  if (customHeaders) {
    const custom = new Headers(customHeaders);
    custom.forEach((value, key) => {
      headers.set(key, value);
    });
  }

  return headers;
}

// Generic fetch wrapper
export async function apiFetch<T = unknown>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const url = `${API_BASE}${endpoint}`;

  try {
    const response = await fetch(url, {
      ...options,
      headers: buildHeaders(options.headers),
    });

    let data: T | undefined;
    let error: string | undefined;

    // Try to parse JSON response
    const contentType = response.headers.get('content-type');
    if (contentType?.includes('application/json')) {
      try {
        const json = await response.json();
        if (response.ok) {
          data = json as T;
        } else {
          error = json.error || json.message || `Request failed with status ${response.status}`;
        }
      } catch {
        if (!response.ok) {
          error = `Request failed with status ${response.status}`;
        }
      }
    } else if (!response.ok) {
      error = `Request failed with status ${response.status}`;
    }

    return {
      data,
      error,
      status: response.status,
      ok: response.ok,
    };
  } catch (e) {
    return {
      error: e instanceof Error ? e.message : 'Network error',
      status: 0,
      ok: false,
    };
  }
}

// Convenience methods
export const api = {
  get<T = unknown>(endpoint: string, options?: RequestInit): Promise<ApiResponse<T>> {
    return apiFetch<T>(endpoint, { ...options, method: 'GET' });
  },

  post<T = unknown>(endpoint: string, body?: unknown, options?: RequestInit): Promise<ApiResponse<T>> {
    return apiFetch<T>(endpoint, {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
  },

  put<T = unknown>(endpoint: string, body?: unknown, options?: RequestInit): Promise<ApiResponse<T>> {
    return apiFetch<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    });
  },

  delete<T = unknown>(endpoint: string, options?: RequestInit): Promise<ApiResponse<T>> {
    return apiFetch<T>(endpoint, { ...options, method: 'DELETE' });
  },

  patch<T = unknown>(endpoint: string, body?: unknown, options?: RequestInit): Promise<ApiResponse<T>> {
    return apiFetch<T>(endpoint, {
      ...options,
      method: 'PATCH',
      body: body ? JSON.stringify(body) : undefined,
    });
  },
};

// SSE (Server-Sent Events) helper for streaming responses
export function streamFetch(
  endpoint: string,
  options: RequestInit = {},
  onMessage: (data: unknown) => void,
  onError?: (error: string) => void,
  onComplete?: () => void
): () => void {
  const controller = new AbortController();
  const url = `${API_BASE}${endpoint}`;

  fetch(url, {
    ...options,
    headers: buildHeaders(options.headers),
    signal: controller.signal,
  })
    .then(async (response) => {
      if (!response.ok) {
        const text = await response.text();
        try {
          const json = JSON.parse(text);
          onError?.(json.error || 'Stream request failed');
        } catch {
          onError?.('Stream request failed');
        }
        return;
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();

      if (!reader) {
        onError?.('No response body');
        return;
      }

      async function read() {
        try {
          const { done, value } = await reader!.read();

          if (done) {
            onComplete?.();
            return;
          }

          const text = decoder.decode(value);
          const lines = text.split('\n');

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              try {
                const data = JSON.parse(line.slice(6));
                onMessage(data);
              } catch {
                // Ignore parse errors for malformed SSE
              }
            }
          }

          read();
        } catch (e) {
          if (e instanceof Error && e.name !== 'AbortError') {
            onError?.(e.message);
          }
        }
      }

      read();
    })
    .catch((e) => {
      if (e instanceof Error && e.name !== 'AbortError') {
        onError?.(e.message);
      }
    });

  // Return abort function
  return () => controller.abort();
}

// Utility functions
export function escapeHtml(text: string): string {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return '0 Bytes';

  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];

  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${Math.floor(seconds)}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
}

export function formatRelativeTime(date: Date | string): string {
  const now = new Date();
  const then = typeof date === 'string' ? new Date(date) : date;
  const seconds = Math.floor((now.getTime() - then.getTime()) / 1000);

  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;

  return then.toLocaleDateString();
}

export function debounce<T extends (...args: unknown[]) => unknown>(
  fn: T,
  delay: number
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout>;

  return (...args: Parameters<T>) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn(...args), delay);
  };
}

export function throttle<T extends (...args: unknown[]) => unknown>(
  fn: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle = false;

  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      fn(...args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
}

// Copy to clipboard
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    // Fallback for older browsers
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.left = '-9999px';
    document.body.appendChild(textArea);
    textArea.select();
    try {
      document.execCommand('copy');
      return true;
    } catch {
      return false;
    } finally {
      document.body.removeChild(textArea);
    }
  }
}

// Generate random string
export function randomString(length = 8): string {
  return Math.random()
    .toString(36)
    .slice(2, 2 + length);
}

// Sleep/delay utility
export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// Check if WebSocket is supported
export function isWebSocketSupported(): boolean {
  return 'WebSocket' in window || 'MozWebSocket' in window;
}

// Get WebSocket URL from current page
export function getWebSocketUrl(path: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}${path}`;
}

// ========== Resource Formatting Utilities ==========

/**
 * Format memory value (in MB) to human readable string
 * Shows GB for values >= 1024MB
 */
export function formatMemory(mb: number): string {
  if (mb >= 1024) {
    const gb = mb / 1024;
    return gb % 1 === 0 ? `${gb}G` : `${gb.toFixed(1)}G`;
  }
  return `${mb}M`;
}

/**
 * Format storage value (in MB) to human readable string  
 * Shows GB for values >= 1024MB
 */
export function formatStorage(mb: number): string {
  if (mb >= 1024) {
    const gb = mb / 1024;
    return gb % 1 === 0 ? `${gb}G` : `${gb.toFixed(1)}G`;
  }
  return `${mb}M`;
}

/**
 * Format CPU shares to vCPU string
 * Converts cpu_shares (1000 = 1 vCPU) to readable format
 */
export function formatCPU(cpuShares: number): string {
  const vcpu = cpuShares / 1000;
  return vcpu % 1 === 0 ? `${vcpu} vCPU` : `${vcpu.toFixed(1)} vCPU`;
}

/**
 * Format memory bytes to readable string (for live stats)
 */
export function formatMemoryBytes(bytes: number): string {
  const mb = bytes / 1024 / 1024;
  if (mb >= 1024) {
    const gb = mb / 1024;
    return gb % 1 === 0 ? `${gb}G` : `${gb.toFixed(1)}G`;
  }
  return `${Math.round(mb)}M`;
}
