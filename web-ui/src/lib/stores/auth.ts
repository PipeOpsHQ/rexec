import { writable, derived } from "svelte/store";

// Types
export interface User {
  id: string;
  email: string;
  name: string;
  avatar?: string;
  tier: "guest" | "free" | "pro" | "enterprise";
  isGuest: boolean;
  isAdmin?: boolean;
  expiresAt?: number; // Unix timestamp for guest session expiration
}

export interface AuthState {
  token: string | null;
  user: User | null;
  isLoading: boolean;
  error: string | null;
}

// Initial state
const initialState: AuthState = {
  token: null,
  user: null,
  isLoading: false,
  error: null,
};

// Load persisted auth from localStorage
function loadPersistedAuth(): AuthState {
  if (typeof window === "undefined") return initialState;

  try {
    const token = localStorage.getItem("rexec_token");
    const userJson = localStorage.getItem("rexec_user");

    if (token && userJson) {
      const user = JSON.parse(userJson) as User;
      return { ...initialState, token, user };
    }
  } catch (e) {
    console.error("Failed to load persisted auth:", e);
    localStorage.removeItem("rexec_token");
    localStorage.removeItem("rexec_user");
  }

  return initialState;
}

// Create the store
function createAuthStore() {
  const { subscribe, set, update } = writable<AuthState>(loadPersistedAuth());

  return {
    subscribe,

    // Set loading state
    setLoading(isLoading: boolean) {
      update((state) => ({ ...state, isLoading, error: null }));
    },

    // Set error
    setError(error: string) {
      update((state) => ({ ...state, isLoading: false, error }));
    },

    // Login with token and user
    login(token: string, user: User) {
      localStorage.setItem("rexec_token", token);
      localStorage.setItem("rexec_user", JSON.stringify(user));
      set({ token, user, isLoading: false, error: null });
    },

    // Guest login with email
    async guestLogin(email: string) {
      update((state) => ({ ...state, isLoading: true, error: null }));

      try {
        const response = await fetch("/api/auth/guest", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            email: email,
            username: `Guest_${Math.floor(Math.random() * 10000)}`,
          }),
        });

        if (!response.ok) {
          const data = await response.json();
          throw new Error(data.error || "Guest login failed");
        }

        const data = await response.json();
        // API returns nested response: { token: "...", user: {...}, guest: true, expires_in: seconds, ... }
        const userData = data.user || data;

        // Calculate expiration time from expires_in (seconds from now)
        const expiresAt = data.expires_in
          ? Math.floor(Date.now() / 1000) + data.expires_in
          : undefined;

        const user: User = {
          id: userData.id || data.user_id,
          email: userData.email || email,
          name: userData.username || userData.name || "Guest User",
          tier: userData.tier || "guest",
          isGuest: true,
          isAdmin: false,
          expiresAt,
        };

        this.login(data.token, user);
        return {
          success: true,
          expiresAt,
          returningGuest: data.returning_guest || false,
          message: data.message || "",
          containerCount: data.containers || 0,
        };
      } catch (e) {
        const error = e instanceof Error ? e.message : "Guest login failed";
        this.setError(error);
        return { success: false, error };
      }
    },

    // OAuth login - get URL
    async getOAuthUrl() {
      try {
        const response = await fetch("/api/auth/oauth/url");
        if (!response.ok) {
          const errorText = await response.text();
          console.error(
            "OAuth URL request failed:",
            response.status,
            errorText,
          );
          throw new Error("Failed to get OAuth URL");
        }
        const data = await response.json();
        // Backend returns auth_url
        return data.auth_url || data.url;
      } catch (e) {
        console.error("Failed to get OAuth URL:", e);
        return null;
      }
    },

    // OAuth exchange - exchange code for token
    async exchangeOAuthCode(code: string) {
      update((state) => ({ ...state, isLoading: true, error: null }));

      try {
        const response = await fetch("/api/auth/oauth/exchange", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ code }),
        });

        if (!response.ok) {
          const data = await response.json();
          throw new Error(data.error || "OAuth exchange failed");
        }

        const data = await response.json();
        // API returns nested response: { token: "...", user: {...} }
        const userData = data.user || data;
        const user: User = {
          id: userData.id || data.user_id,
          email: userData.email || "",
          name: userData.username || userData.name || userData.email || "User",
          avatar: userData.avatar,
          tier: userData.tier || "free",
          isGuest: false,
          isAdmin: userData.is_admin || userData.role === 'admin' || false,
        };

        this.login(data.token, user);
        return { success: true };
      } catch (e) {
        const error = e instanceof Error ? e.message : "OAuth login failed";
        this.setError(error);
        return { success: false, error };
      }
    },

    // Fetch user profile
    async fetchProfile() {
      update((state) => ({ ...state, isLoading: true }));

      try {
        const token = localStorage.getItem("rexec_token");
        if (!token) throw new Error("No token");

        const response = await fetch("/api/profile", {
          headers: { Authorization: `Bearer ${token}` },
        });

        if (!response.ok) {
          if (response.status === 401) {
            this.logout();
            throw new Error("Session expired");
          }
          throw new Error("Failed to fetch profile");
        }

        const data = await response.json();
        // API returns nested response: { user: {...}, stats: {...}, limits: {...} }
        const userData = data.user || data;

        // Get existing user from localStorage to preserve expiresAt for guests
        let existingExpiresAt: number | undefined;
        try {
          const existingUserJson = localStorage.getItem("rexec_user");
          if (existingUserJson) {
            const existingUser = JSON.parse(existingUserJson);
            existingExpiresAt = existingUser.expiresAt;
          }
        } catch {
          // Ignore parse errors
        }

        const user: User = {
          id: userData.id,
          email: userData.email || "",
          name: userData.username || userData.name || userData.email || "User",
          avatar: userData.avatar,
          tier: userData.tier || "free",
          isGuest: userData.tier === "guest",
          isAdmin: userData.is_admin || userData.role === 'admin' || false,
          // For guests, prefer localStorage expiresAt (from login) over profile response
          // because profile calculates from user.CreatedAt which may be stale for returning guests
          expiresAt:
            userData.tier === "guest"
              ? existingExpiresAt || userData.expires_at
              : userData.expires_at,
        };

        update((state) => ({
          ...state,
          user,
          isLoading: false,
          error: null,
        }));
        localStorage.setItem("rexec_user", JSON.stringify(user));

        return { success: true, user };
      } catch (e) {
        const error =
          e instanceof Error ? e.message : "Failed to fetch profile";
        update((state) => ({ ...state, isLoading: false }));
        return { success: false, error };
      }
    },

    // Logout
    logout() {
      localStorage.removeItem("rexec_token");
      localStorage.removeItem("rexec_user");
      set(initialState);
    },

    // Check if guest session has expired
    isSessionExpired(): boolean {
      const userJson = localStorage.getItem("rexec_user");
      if (!userJson) return false;

      try {
        const user = JSON.parse(userJson) as User;
        // Check both isGuest flag and tier === "guest" for backwards compatibility
        if ((user.isGuest || user.tier === "guest") && user.expiresAt) {
          const now = Math.floor(Date.now() / 1000);
          return now >= user.expiresAt;
        }
      } catch {
        // Ignore parse errors
      }
      return false;
    },

    // Manually enable admin mode (for secret key access)
    enableAdminMode() {
      update((state) => {
        if (!state.user) return state;
        const updatedUser = { ...state.user, isAdmin: true, tier: "pro" as const };
        localStorage.setItem("rexec_user", JSON.stringify(updatedUser));
        return { ...state, user: updatedUser };
      });
    },

    // Check if token is valid
    async validateToken() {
      const token = localStorage.getItem("rexec_token");
      if (!token) {

        return false;
      }
      // ... (rest of the function)
      // I will implement enableAdminMode before validateToken to keep it grouped with actions
      return false; // Just placeholder for matching
    },

    // Manually enable admin mode (for secret key access)
    enableAdminMode() {
      update((state) => {
        if (!state.user) return state;
        const updatedUser = { ...state.user, isAdmin: true, tier: "pro" as const };
        localStorage.setItem("rexec_user", JSON.stringify(updatedUser));
        return { ...state, user: updatedUser };
      });
    },

    async validateToken() {

      // Check if guest session has expired locally first
      if (this.isSessionExpired()) {

        this.logout();
        return false;
      }

      // Check local session validity for guests first
      const userJson = localStorage.getItem("rexec_user");
      let isGuestWithValidLocalSession = false;
      if (userJson) {
        try {
          const user = JSON.parse(userJson) as User;
          if ((user.isGuest || user.tier === "guest") && user.expiresAt) {
            const now = Math.floor(Date.now() / 1000);
            isGuestWithValidLocalSession = now < user.expiresAt;

          }
        } catch {
          // Ignore parse errors
        }
      }

      try {
        const response = await fetch("/api/profile", {
          headers: { Authorization: `Bearer ${token}` },
        });

        if (response.ok) {

          return true;
        }

        // On 401 Unauthorized, the token is definitely invalid
        if (response.status === 401) {
          this.logout();
          return false;
        }

        // For other errors (500, etc) or missing schema fields, 
        // we should arguably keep the user logged in to avoid disruption
        // unless it's a guest whose session is expired.
        
        // If it's a guest with valid local session, definitely keep them.
        if (isGuestWithValidLocalSession) {

          return true;
        }

        // For non-guests (or unknown status), if we have a token, assume it's valid
        // but the backend might be having issues.
        // Returning true here prevents the immediate logout loop on refresh.
        // The UI might show an error later if specific calls fail.
        return true;
      } catch (e) {


        // On network error, keep guests logged in if local session is valid
        if (isGuestWithValidLocalSession) {

          return true;
        }

        // For non-guests, if we have a token but network failed, assume it's valid for now
        // to prevent immediate logout on flaky connections or server restarts.
        return true;
      }
    },
  };
}

// Export the store
export const auth = createAuthStore();

// Derived stores for convenience
export const isAuthenticated = derived(auth, ($auth) => !!$auth.token);
export const isGuest = derived(
  auth,
  ($auth) => $auth.user?.isGuest || $auth.user?.tier === "guest" || false,
);
export const isAdmin = derived(auth, ($auth) => !!$auth.user?.isAdmin);
export const userTier = derived(auth, ($auth) => $auth.user?.tier ?? "guest");
export const token = derived(auth, ($auth) => $auth.token);
export const sessionExpiresAt = derived(
  auth,
  ($auth) => $auth.user?.expiresAt ?? null,
);

// Check if guest session is expired
export const isSessionExpired = derived(auth, ($auth) => {
  if (!$auth.user?.isGuest || !$auth.user?.expiresAt) return false;
  const now = Math.floor(Date.now() / 1000);
  return now >= $auth.user.expiresAt;
});

// Time remaining in seconds for guest session
export const sessionTimeRemaining = derived(auth, ($auth) => {
  if (!$auth.user?.isGuest || !$auth.user?.expiresAt) return null;
  const now = Math.floor(Date.now() / 1000);
  const remaining = $auth.user.expiresAt - now;
  return remaining > 0 ? remaining : 0;
});
