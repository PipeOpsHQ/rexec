import { writable, get } from 'svelte/store';
import { auth } from './auth';

export interface APIToken {
  id: string;
  name: string;
  token_prefix: string;
  scopes: string[];
  last_used_at?: string;
  expires_at?: string;
  created_at: string;
}

interface TokensState {
  tokens: APIToken[];
  loading: boolean;
  error: string | null;
}

const API_BASE = '/api';

function createTokensStore() {
  const { subscribe, set, update } = writable<TokensState>({
    tokens: [],
    loading: false,
    error: null,
  });

  function getAuthHeader(): HeadersInit {
    const authState = get(auth);
    if (authState.token) {
      return { Authorization: `Bearer ${authState.token}` };
    }
    return {};
  }

  return {
    subscribe,

    async fetchTokens(): Promise<void> {
      update(s => ({ ...s, loading: true, error: null }));
      try {
        const res = await fetch(`${API_BASE}/tokens`, {
          headers: getAuthHeader(),
        });
        if (!res.ok) throw new Error('Failed to fetch tokens');
        const data = await res.json();
        update(s => ({ ...s, tokens: data.tokens || [], loading: false }));
      } catch (err: any) {
        update(s => ({ ...s, error: err.message, loading: false }));
      }
    },

    async createToken(name: string, scopes: string[] = ['read', 'write'], expiresIn?: number): Promise<{ token: string; id: string } | null> {
      update(s => ({ ...s, loading: true, error: null }));
      try {
        const res = await fetch(`${API_BASE}/tokens`, {
          method: 'POST',
          headers: {
            ...getAuthHeader(),
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ name, scopes, expires_in: expiresIn }),
        });
        if (!res.ok) throw new Error('Failed to create token');
        const data = await res.json();
        
        // Refresh the tokens list
        await this.fetchTokens();
        
        return { token: data.token, id: data.id };
      } catch (err: any) {
        update(s => ({ ...s, error: err.message, loading: false }));
        return null;
      }
    },

    async revokeToken(tokenId: string): Promise<boolean> {
      update(s => ({ ...s, loading: true, error: null }));
      try {
        const res = await fetch(`${API_BASE}/tokens/${tokenId}`, {
          method: 'DELETE',
          headers: getAuthHeader(),
        });
        if (!res.ok) throw new Error('Failed to revoke token');
        
        // Remove from local state
        update(s => ({
          ...s,
          tokens: s.tokens.filter(t => t.id !== tokenId),
          loading: false,
        }));
        
        return true;
      } catch (err: any) {
        update(s => ({ ...s, error: err.message, loading: false }));
        return false;
      }
    },

    async deleteToken(tokenId: string): Promise<boolean> {
      update(s => ({ ...s, loading: true, error: null }));
      try {
        const res = await fetch(`${API_BASE}/tokens/${tokenId}/permanent`, {
          method: 'DELETE',
          headers: getAuthHeader(),
        });
        if (!res.ok) throw new Error('Failed to delete token');
        
        // Remove from local state
        update(s => ({
          ...s,
          tokens: s.tokens.filter(t => t.id !== tokenId),
          loading: false,
        }));
        
        return true;
      } catch (err: any) {
        update(s => ({ ...s, error: err.message, loading: false }));
        return false;
      }
    },

    reset() {
      set({ tokens: [], loading: false, error: null });
    },
  };
}

export const tokens = createTokensStore();
