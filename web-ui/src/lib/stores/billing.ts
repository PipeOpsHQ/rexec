import { writable, derived } from "svelte/store";

export interface Invoice {
  id: string;
  number: string;
  status: string;
  amount_due: number;
  amount_paid: number;
  currency: string;
  created: string;
  period_start: string;
  period_end: string;
  invoice_pdf?: string;
  hosted_invoice_url?: string;
  description?: string;
}

export interface Subscription {
  tier: "free" | "pro" | "enterprise";
  status: "active" | "canceled" | "past_due" | "trialing" | "inactive";
  container_limit: number;
  current_period_end?: string;
}

export interface BillingState {
  subscription: Subscription | null;
  invoices: Invoice[];
  isLoading: boolean;
  error: string | null;
}

const initialState: BillingState = {
  subscription: null,
  invoices: [],
  isLoading: false,
  error: null,
};

function createBillingStore() {
  const { subscribe, set, update } = writable<BillingState>(initialState);

  async function getAuthHeaders(): Promise<HeadersInit> {
    const token = localStorage.getItem("rexec_token");
    return {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    };
  }

  return {
    subscribe,

    async fetchSubscription() {
      update((state) => ({ ...state, isLoading: true, error: null }));

      try {
        const response = await fetch("/api/billing/subscription", {
          headers: await getAuthHeaders(),
        });

        if (!response.ok) {
          throw new Error("Failed to fetch subscription");
        }

        const data = await response.json();
        update((state) => ({
          ...state,
          subscription: data,
          isLoading: false,
        }));

        return data;
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to fetch subscription";
        update((state) => ({ ...state, isLoading: false, error: message }));
        return null;
      }
    },

    async fetchInvoices() {
      update((state) => ({ ...state, isLoading: true, error: null }));

      try {
        const response = await fetch("/api/billing/history", {
          headers: await getAuthHeaders(),
        });

        if (!response.ok) {
          throw new Error("Failed to fetch billing history");
        }

        const data = await response.json();
        update((state) => ({
          ...state,
          invoices: data.invoices || [],
          isLoading: false,
        }));

        return data.invoices;
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to fetch billing history";
        update((state) => ({ ...state, isLoading: false, error: message }));
        return [];
      }
    },

    async createCheckout(tier: "pro" | "enterprise") {
      try {
        const response = await fetch("/api/billing/checkout", {
          method: "POST",
          headers: await getAuthHeaders(),
          body: JSON.stringify({ tier }),
        });

        if (!response.ok) {
          const error = await response.json();
          throw new Error(error.error || "Failed to create checkout session");
        }

        const data = await response.json();
        return { success: true, checkoutUrl: data.checkout_url };
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to create checkout";
        return { success: false, error: message };
      }
    },

    async openPortal() {
      try {
        const response = await fetch("/api/billing/portal", {
          method: "POST",
          headers: await getAuthHeaders(),
        });

        if (!response.ok) {
          const error = await response.json();
          throw new Error(error.error || "Failed to open billing portal");
        }

        const data = await response.json();
        return { success: true, portalUrl: data.portal_url };
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to open portal";
        return { success: false, error: message };
      }
    },

    async cancelSubscription() {
      try {
        const response = await fetch("/api/billing/cancel", {
          method: "POST",
          headers: await getAuthHeaders(),
        });

        if (!response.ok) {
          const error = await response.json();
          throw new Error(error.error || "Failed to cancel subscription");
        }

        const data = await response.json();
        return { success: true, message: data.message };
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to cancel subscription";
        return { success: false, error: message };
      }
    },

    reset() {
      set(initialState);
    },
  };
}

export const billing = createBillingStore();

// Derived stores for convenience
export const subscription = derived(billing, ($billing) => $billing.subscription);
export const invoices = derived(billing, ($billing) => $billing.invoices);
export const billingLoading = derived(billing, ($billing) => $billing.isLoading);
