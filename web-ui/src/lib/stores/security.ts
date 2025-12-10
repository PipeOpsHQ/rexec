import { writable, derived } from "svelte/store";

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
      return {
        ...initialState,
        passcodeHash: parsed.passcodeHash || null,
        lockAfterMinutes: parsed.lockAfterMinutes || 5,
        passcodeSetupPromptDismissed: parsed.passcodeSetupPromptDismissed || false,
        lastActivity: Date.now(), // Reset on load
        isLocked: false, // Never start locked
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
  const { subscribe, set, update } = writable<SecurityState>(loadPersistedSecurity());

  // Persist to localStorage
  function persist(state: SecurityState) {
    if (typeof window === "undefined") return;
    localStorage.setItem(
      "rexec_security",
      JSON.stringify({
        passcodeHash: state.passcodeHash,
        lockAfterMinutes: state.lockAfterMinutes,
        passcodeSetupPromptDismissed: state.passcodeSetupPromptDismissed,
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
      return new Promise((resolve) => {
        const unsubscribe = subscribe((state) => {
          unsubscribe();
          hashPasscode(passcode).then((hash) => {
            resolve(hash === state.passcodeHash);
          });
        });
      });
    },

    // Lock the screen
    lock() {
      update((state) => ({ ...state, isLocked: true }));
    },

    // Unlock the screen
    unlock() {
      update((state) => ({
        ...state,
        isLocked: false,
        lastActivity: Date.now(),
      }));
    },

    // Update last activity
    updateActivity() {
      update((state) => ({ ...state, lastActivity: Date.now() }));
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
      let shouldLock = false;
      const unsubscribe = subscribe((state) => {
        unsubscribe();
        if (!state.passcodeHash || state.isLocked) {
          shouldLock = false;
          return;
        }
        const elapsed = Date.now() - state.lastActivity;
        const timeoutMs = state.lockAfterMinutes * 60 * 1000;
        shouldLock = elapsed >= timeoutMs;
      });
      return shouldLock;
    },

    // Get current state synchronously
    getState(): SecurityState {
      let currentState = initialState;
      const unsubscribe = subscribe((state) => {
        currentState = state;
      });
      unsubscribe();
      return currentState;
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
