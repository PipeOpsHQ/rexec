import { writable, derived } from "svelte/store";

// Types
export interface User {
  id: string;
  email: string;
  name: string;
  avatar?: string;
  tier: "guest" | "free" | "pro" | "enterprise";
  isGuest: boolean;
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
        const user: User = {
          id: data.user_id,
          email: email,
          name: data.name || "Guest User",
          tier: "guest",
          isGuest: true,
        };

        this.login(data.token, user);
        return { success: true };
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
        if (!response.ok) throw new Error("Failed to get OAuth URL");
        const data = await response.json();
        return data.url;
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
        const user: User = {
          id: data.user_id,
          email: data.email || "",
          name: data.name || data.email || "User",
          avatar: data.avatar,
          tier: data.tier || "free",
          isGuest: false,
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
        const user: User = {
          id: data.id,
          email: data.email || "",
          name: data.name || data.email || "User",
          avatar: data.avatar,
          tier: data.tier || "free",
          isGuest: data.is_guest || false,
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

    // Check if token is valid
    async validateToken() {
      const token = localStorage.getItem("rexec_token");
      if (!token) return false;

      try {
        const response = await fetch("/api/profile", {
          headers: { Authorization: `Bearer ${token}` },
        });
        return response.ok;
      } catch {
        return false;
      }
    },
  };
}

// Export the store
export const auth = createAuthStore();

// Derived stores for convenience
export const isAuthenticated = derived(auth, ($auth) => !!$auth.token);
export const isGuest = derived(auth, ($auth) => $auth.user?.isGuest ?? false);
export const userTier = derived(auth, ($auth) => $auth.user?.tier ?? "guest");
export const token = derived(auth, ($auth) => $auth.token);
