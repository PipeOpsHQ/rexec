import { writable, derived, get } from "svelte/store";
import { auth } from "$stores/auth";

export interface SecurityState {
  enabled: boolean;
  isLocked: boolean;
  lockAfterMinutes: number;
  lastActivity: number;
  passcodeSetupPromptDismissed: boolean;
  singleSessionMode: boolean;
}

const initialState: SecurityState = {
  enabled: false,
  isLocked: false,
  lockAfterMinutes: 5,
  lastActivity: Date.now(),
  passcodeSetupPromptDismissed: false,
  singleSessionMode: false,
};

function loadPersistedSecurity(): SecurityState {
  if (typeof window === "undefined") return initialState;
  try {
    const stored = localStorage.getItem("rexec_security");
    if (stored) {
      const parsed = JSON.parse(stored);
      return {
        ...initialState,
        isLocked: parsed.isLocked === true,
        lockAfterMinutes: parsed.lockAfterMinutes || 5,
        lastActivity: parsed.lastActivity || Date.now(),
        passcodeSetupPromptDismissed: parsed.passcodeSetupPromptDismissed || false,
      };
    }
  } catch (e) {
    console.error("Failed to load security settings:", e);
  }
  return initialState;
}

function createSecurityStore() {
  const store = writable<SecurityState>(loadPersistedSecurity());
  const { subscribe, update } = store;

  function persistLocal(state: SecurityState) {
    if (typeof window === "undefined") return;
    localStorage.setItem(
      "rexec_security",
      JSON.stringify({
        isLocked: state.isLocked,
        lockAfterMinutes: state.lockAfterMinutes,
        lastActivity: state.lastActivity,
        passcodeSetupPromptDismissed: state.passcodeSetupPromptDismissed,
      })
    );
  }

  async function authedFetch(input: RequestInfo, init: RequestInit = {}) {
    const token = get(auth).token || localStorage.getItem("rexec_token");
    const headers = new Headers(init.headers || {});
    headers.set("Content-Type", "application/json");
    if (token) headers.set("Authorization", `Bearer ${token}`);
    return fetch(input, { ...init, headers });
  }

  return {
    subscribe,

    async refreshFromServer() {
      const res = await authedFetch("/api/security");
      if (res.status === 423) {
        update((state) => {
          const newState = { ...state, isLocked: true };
          persistLocal(newState);
          return newState;
        });
        return;
      }
      if (!res.ok) return;
      const data = await res.json();
      update((state) => {
        const newState = {
          ...state,
          enabled: !!data.enabled,
          lockAfterMinutes: data.lock_after_minutes || state.lockAfterMinutes,
          singleSessionMode: !!data.single_session_mode,
        };
        persistLocal(newState);
        return newState;
      });
    },

    async setPasscode(newPasscode: string, currentPasscode?: string, lockAfterMinutes?: number) {
      const res = await authedFetch("/api/security/passcode", {
        method: "POST",
        body: JSON.stringify({
          new_passcode: newPasscode,
          current_passcode: currentPasscode,
          lock_after_minutes: lockAfterMinutes,
        }),
      });
      if (res.status === 423) {
        this.lockLocal();
        return { success: false, error: "session_locked" };
      }
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        return { success: false, error: data.error || "Failed to set passcode" };
      }
      update((state) => {
        const newState = {
          ...state,
          enabled: true,
          lockAfterMinutes: data.lock_after_minutes || state.lockAfterMinutes,
        };
        persistLocal(newState);
        return newState;
      });
      return { success: true };
    },

    async removePasscode(currentPasscode: string) {
      const res = await authedFetch("/api/security/passcode", {
        method: "DELETE",
        body: JSON.stringify({ current_passcode: currentPasscode }),
      });
      if (res.status === 423) {
        this.lockLocal();
        return { success: false, error: "session_locked" };
      }
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        return { success: false, error: data.error || "Failed to disable screen lock" };
      }
      update((state) => {
        const newState = { ...state, enabled: false, isLocked: false };
        persistLocal(newState);
        return newState;
      });
      return { success: true };
    },

    async updateLockTimeout(minutes: number) {
      const res = await authedFetch("/api/security", {
        method: "PATCH",
        body: JSON.stringify({ lock_after_minutes: minutes }),
      });
      if (res.status === 423) {
        this.lockLocal();
        return { success: false, error: "session_locked" };
      }
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        return { success: false, error: data.error || "Failed to update lock timeout" };
      }
      update((state) => {
        const newState = { ...state, lockAfterMinutes: minutes };
        persistLocal(newState);
        return newState;
      });
      return { success: true };
    },

    async lock() {
      const res = await authedFetch("/api/security/lock", { method: "POST" });
      if (res.ok || res.status === 423) {
        this.lockLocal();
      }
    },

    lockLocal() {
      update((state) => {
        const newState = { ...state, isLocked: true };
        persistLocal(newState);
        return newState;
      });
    },

    async unlockWithPasscode(passcode: string) {
      const res = await authedFetch("/api/security/unlock", {
        method: "POST",
        body: JSON.stringify({ passcode }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        return { success: false, error: data.error || "Incorrect passcode" };
      }

      const newToken = data.token as string | undefined;
      if (newToken) {
        const currentAuth = get(auth);
        if (currentAuth.user) {
          auth.login(newToken, {
            ...currentAuth.user,
            tier: data.user?.tier ?? currentAuth.user.tier,
            isAdmin: data.user?.is_admin ?? currentAuth.user.isAdmin,
            subscriptionActive: data.user?.subscription_active ?? currentAuth.user.subscriptionActive,
            allowedIPs: data.user?.allowed_ips ?? currentAuth.user.allowedIPs,
            mfaEnabled: data.user?.mfa_enabled ?? currentAuth.user.mfaEnabled,
          });
        } else if (data.user) {
          auth.login(newToken, {
            id: data.user.id,
            email: data.user.email,
            username: data.user.username,
            name: data.user.username,
            tier: data.user.tier,
            isGuest: data.user.tier === "guest",
            isAdmin: data.user.is_admin,
            subscriptionActive: data.user.subscription_active,
            allowedIPs: data.user.allowed_ips || [],
            mfaEnabled: data.user.mfa_enabled || false,
          });
        }
      }

      update((state) => {
        const newState = { ...state, isLocked: false, lastActivity: Date.now() };
        persistLocal(newState);
        return newState;
      });

      return { success: true };
    },

    updateActivity() {
      update((state) => {
        const newState = { ...state, lastActivity: Date.now() };
        if (typeof window !== "undefined") {
          const lastPersist = (window as any).__rexec_last_activity_persist || 0;
          const now = Date.now();
          if (now - lastPersist > 30000) {
            persistLocal(newState);
            (window as any).__rexec_last_activity_persist = now;
          }
        }
        return newState;
      });
    },

    dismissSetupPrompt() {
      update((state) => {
        const newState = { ...state, passcodeSetupPromptDismissed: true };
        persistLocal(newState);
        return newState;
      });
    },

    resetSetupPrompt() {
      update((state) => {
        const newState = { ...state, passcodeSetupPromptDismissed: false };
        persistLocal(newState);
        return newState;
      });
    },

    async setSingleSessionMode(enabled: boolean): Promise<{ success: boolean; error?: string }> {
      const res = await authedFetch("/api/security/single-session", {
        method: "POST",
        body: JSON.stringify({ enabled }),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        return { success: false, error: data.error || "Failed to update setting" };
      }
      update((state) => {
        const newState = { ...state, singleSessionMode: enabled };
        persistLocal(newState);
        return newState;
      });
      return { success: true };
    },

    checkInactivity(): boolean {
      const state = get(store);
      if (!state.enabled || state.isLocked) return false;
      const elapsed = Date.now() - state.lastActivity;
      const timeoutMs = state.lockAfterMinutes * 60 * 1000;
      return elapsed >= timeoutMs;
    },

    getState(): SecurityState {
      return get(store);
    },
  };
}

export const security = createSecurityStore();

export const hasPasscode = derived(security, ($security) => $security.enabled);
export const isLocked = derived(security, ($security) => $security.isLocked);
export const shouldPromptPasscode = derived(
  security,
  ($security) => !$security.enabled && !$security.passcodeSetupPromptDismissed
);
