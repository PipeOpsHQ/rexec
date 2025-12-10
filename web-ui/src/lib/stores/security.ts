import { writable, derived, get } from "svelte/store";

// Types
export interface SecurityState {
  passcodeHash: string | null;
  isLocked: boolean;
  lockAfterMinutes: number;
  lastActivity: number;
  passcodeSetupPromptDismissed: boolean;
}

// Initial state
const initialState: SecurityState = {
  passcodeHash: null,
  isLocked: false,
  lockAfterMinutes: 5, // Default: lock after 5 minutes of inactivity
  lastActivity: Date.now(),
  passcodeSetupPromptDismissed: false,
};

// Load persisted security settings from localStorage
function loadPersistedSecurity(): SecurityState {
  if (typeof window === "undefined") return initialState;

  try {
    const stored = localStorage.getItem("rexec_security");
    if (stored) {
      const parsed = JSON.parse(stored);
      const hasPasscode = !!parsed.passcodeHash;
      const wasLocked = parsed.isLocked === true;
      const lastActivity = parsed.lastActivity || Date.now();
      const lockAfterMinutes = parsed.lockAfterMinutes || 5;
      
      // Calculate if should be locked based on:
      // 1. Was already locked before reload
      // 2. Or has passcode and was inactive too long
      const elapsed = Date.now() - lastActivity;
      const timeoutMs = lockAfterMinutes * 60 * 1000;
      const shouldBeLocked = hasPasscode && (wasLocked || elapsed >= timeoutMs);
      
      return {
        ...initialState,
        passcodeHash: parsed.passcodeHash || null,
        lockAfterMinutes: lockAfterMinutes,
        passcodeSetupPromptDismissed: parsed.passcodeSetupPromptDismissed || false,
        lastActivity: wasLocked ? lastActivity : Date.now(), // Keep old lastActivity if was locked
        isLocked: shouldBeLocked,
      };
    }
  } catch (e) {
    console.error("Failed to load security settings:", e);
  }

  return initialState;
}

// Simple hash function for passcode (not cryptographically secure, but sufficient for screen lock)
async function hashPasscode(passcode: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(passcode + "rexec_salt_v1");
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}

// Create the store
function createSecurityStore() {
  const store = writable<SecurityState>(loadPersistedSecurity());
  const { subscribe, set, update } = store;

  // Persist to localStorage
  function persist(state: SecurityState) {
    if (typeof window === "undefined") return;
    localStorage.setItem(
      "rexec_security",
      JSON.stringify({
        passcodeHash: state.passcodeHash,
        lockAfterMinutes: state.lockAfterMinutes,
        passcodeSetupPromptDismissed: state.passcodeSetupPromptDismissed,
        isLocked: state.isLocked,
        lastActivity: state.lastActivity,
      })
    );
  }

  return {
    subscribe,

    // Set passcode
    async setPasscode(passcode: string) {
      const hash = await hashPasscode(passcode);
      update((state) => {
        const newState = { ...state, passcodeHash: hash };
        persist(newState);
        return newState;
      });
    },

    // Remove passcode
    removePasscode() {
      update((state) => {
        const newState = { ...state, passcodeHash: null };
        persist(newState);
        return newState;
      });
    },

    // Verify passcode
    async verifyPasscode(passcode: string): Promise<boolean> {
      const state = get(store);
      const hash = await hashPasscode(passcode);
      return hash === state.passcodeHash;
    },

    // Lock the screen
    lock() {
      update((state) => {
        const newState = { ...state, isLocked: true };
        persist(newState);
        return newState;
      });
    },

    // Unlock the screen
    unlock() {
      update((state) => {
        const newState = {
          ...state,
          isLocked: false,
          lastActivity: Date.now(),
        };
        persist(newState);
        return newState;
      });
    },

    // Update last activity
    updateActivity() {
      update((state) => {
        const newState = { ...state, lastActivity: Date.now() };
        // Throttle persist to avoid too many writes
        if (typeof window !== "undefined") {
          const lastPersist = (window as any).__rexec_last_activity_persist || 0;
          const now = Date.now();
          if (now - lastPersist > 30000) { // Only persist every 30 seconds
            persist(newState);
            (window as any).__rexec_last_activity_persist = now;
          }
        }
        return newState;
      });
    },

    // Set lock timeout
    setLockTimeout(minutes: number) {
      update((state) => {
        const newState = { ...state, lockAfterMinutes: minutes };
        persist(newState);
        return newState;
      });
    },

    // Dismiss passcode setup prompt
    dismissSetupPrompt() {
      update((state) => {
        const newState = { ...state, passcodeSetupPromptDismissed: true };
        persist(newState);
        return newState;
      });
    },

    // Reset setup prompt dismissal
    resetSetupPrompt() {
      update((state) => {
        const newState = { ...state, passcodeSetupPromptDismissed: false };
        persist(newState);
        return newState;
      });
    },

    // Check if should lock based on inactivity
    checkInactivity(): boolean {
      const state = get(store);
      if (!state.passcodeHash || state.isLocked) {
        return false;
      }
      const elapsed = Date.now() - state.lastActivity;
      const timeoutMs = state.lockAfterMinutes * 60 * 1000;
      return elapsed >= timeoutMs;
    },

    // Get current state synchronously
    getState(): SecurityState {
      return get(store);
    },
  };
}

// Export the store
export const security = createSecurityStore();

// Derived stores
export const hasPasscode = derived(security, ($security) => !!$security.passcodeHash);
export const isLocked = derived(security, ($security) => $security.isLocked);
export const shouldPromptPasscode = derived(
  security,
  ($security) => !$security.passcodeHash && !$security.passcodeSetupPromptDismissed
);
