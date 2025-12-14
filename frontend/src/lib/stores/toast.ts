import { writable, derived } from "svelte/store";

// Types
export type ToastType = "success" | "error" | "warning" | "info" | "loading";

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
  duration: number;
  dismissible: boolean;
}

export interface ToastState {
  toasts: Toast[];
}

// Initial state
const initialState: ToastState = {
  toasts: [],
};

// Generate unique ID
function generateId(): string {
  return `toast-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

// Create the store
function createToastStore() {
  const { subscribe, update } = writable<ToastState>(initialState);

  // Auto-dismiss timers
  const timers = new Map<string, ReturnType<typeof setTimeout>>();

  return {
    subscribe,

    // Show a toast
    show(
      message: string,
      type: ToastType = "info",
      options: { duration?: number; dismissible?: boolean; id?: string } = {},
    ): string {
      const {
        duration = type === "loading" ? 0 : 5000,
        dismissible = true,
        id = generateId(),
      } = options;

      // Remove existing toast with same ID
      this.dismiss(id);

      const toast: Toast = {
        id,
        type,
        message,
        duration,
        dismissible,
      };

      update((state) => ({
        ...state,
        toasts: [...state.toasts, toast],
      }));

      // Auto-dismiss after duration (if not 0)
      if (duration > 0) {
        const timer = setTimeout(() => {
          this.dismiss(id);
        }, duration);
        timers.set(id, timer);
      }

      return id;
    },

    // Convenience methods
    success(message: string, options?: { duration?: number; id?: string }) {
      const safeMessage = this.sanitizeMessage(message, "Operation successful");
      return this.show(safeMessage, "success", options);
    },

    error(message: string, options?: { duration?: number; id?: string }) {
      const safeMessage = this.sanitizeMessage(message, "An error occurred");
      return this.show(safeMessage, "error", { duration: 8000, ...options });
    },

    warning(message: string, options?: { duration?: number; id?: string }) {
      const safeMessage = this.sanitizeMessage(message, "Warning");
      return this.show(safeMessage, "warning", options);
    },

    info(message: string, options?: { duration?: number; id?: string }) {
      const safeMessage = this.sanitizeMessage(message, "Info");
      return this.show(safeMessage, "info", options);
    },

    // Helper to ensure message is never undefined/null/empty
    sanitizeMessage(message: unknown, fallback: string): string {
      if (
        typeof message === "string" &&
        message.trim() &&
        message !== "undefined"
      ) {
        return message;
      }
      if (message && typeof message === "object") {
        const obj = message as Record<string, unknown>;
        if (typeof obj.message === "string" && obj.message.trim()) {
          return obj.message;
        }
        if (typeof obj.error === "string" && obj.error.trim()) {
          return obj.error;
        }
      }
      return fallback;
    },

    loading(message: string, options?: { id?: string }) {
      return this.show(message, "loading", {
        duration: 0,
        dismissible: false,
        ...options,
      });
    },

    // Dismiss a toast
    dismiss(id: string) {
      // Clear auto-dismiss timer
      const timer = timers.get(id);
      if (timer) {
        clearTimeout(timer);
        timers.delete(id);
      }

      update((state) => ({
        ...state,
        toasts: state.toasts.filter((t) => t.id !== id),
      }));
    },

    // Dismiss all toasts
    dismissAll() {
      // Clear all timers
      timers.forEach((timer) => clearTimeout(timer));
      timers.clear();

      update(() => initialState);
    },

    // Update a toast (useful for loading -> success/error transitions)
    update(id: string, message: string, type: ToastType, duration = 5000) {
      // Clear existing timer
      const existingTimer = timers.get(id);
      if (existingTimer) {
        clearTimeout(existingTimer);
        timers.delete(id);
      }

      update((state) => ({
        ...state,
        toasts: state.toasts.map((t) =>
          t.id === id
            ? { ...t, message, type, duration, dismissible: true }
            : t,
        ),
      }));

      // Set new auto-dismiss timer
      if (duration > 0) {
        const timer = setTimeout(() => {
          this.dismiss(id);
        }, duration);
        timers.set(id, timer);
      }
    },

    // Promise-based toast (shows loading, then success/error)
    async promise<T>(
      promise: Promise<T>,
      messages: {
        loading: string;
        success: string | ((data: T) => string);
        error: string | ((err: Error) => string);
      },
    ): Promise<T> {
      const id = this.loading(messages.loading);

      try {
        const result = await promise;
        const successMessage =
          typeof messages.success === "function"
            ? messages.success(result)
            : messages.success;
        this.update(id, successMessage, "success");
        return result;
      } catch (err) {
        const errorMessage =
          typeof messages.error === "function"
            ? messages.error(err as Error)
            : messages.error;
        this.update(id, errorMessage, "error", 8000);
        throw err;
      }
    },
  };
}

// Export the store
export const toast = createToastStore();

// Derived store for active toasts
export const activeToasts = derived(toast, ($toast) => $toast.toasts);

// Derived store for toast count
export const toastCount = derived(toast, ($toast) => $toast.toasts.length);
