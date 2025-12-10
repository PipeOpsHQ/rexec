import { writable, derived, get } from 'svelte/store';
import { auth } from './auth';

export interface AgentSystemInfo {
  os: string;
  arch: string;
  num_cpu: number;
  hostname: string;
  memory: {
    total: number;
    available: number;
    free: number;
  };
  disk: {
    total: number;
    free: number;
    available: number;
  };
}

export interface AgentStats {
  cpu_percent: number;
  memory: number;
  memory_limit: number;
  disk_usage: number;
  disk_limit: number;
}

export interface Agent {
  id: string;
  name: string;
  description?: string;
  os: string;
  arch: string;
  shell: string;
  tags?: string[];
  status: 'online' | 'offline' | 'registered';
  connected_at?: string;
  last_ping?: string;
  created_at: string;
  system_info?: AgentSystemInfo;
  stats?: AgentStats;
}

interface AgentsState {
  agents: Agent[];
  loading: boolean;
  error: string | null;
}

const API_BASE = '/api';

function createAgentsStore() {
  const { subscribe, set, update } = writable<AgentsState>({
    agents: [],
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

    async fetchAgents(): Promise<void> {
      update(s => ({ ...s, loading: true, error: null }));
      try {
        const res = await fetch(`${API_BASE}/agents`, {
          headers: getAuthHeader(),
        });
        if (!res.ok) throw new Error('Failed to fetch agents');
        const agentsList = await res.json();
        
        // For online agents, fetch their status to get system_info and stats
        const enrichedAgents = await Promise.all(
          (agentsList || []).map(async (agent: Agent) => {
            if (agent.status === 'online') {
              try {
                const statusRes = await fetch(`${API_BASE}/agents/${agent.id}/status`, {
                  headers: getAuthHeader(),
                });
                if (statusRes.ok) {
                  const status = await statusRes.json();
                  return {
                    ...agent,
                    system_info: status.system_info,
                    stats: status.stats,
                  };
                }
              } catch {
                // Ignore status fetch errors
              }
            }
            return agent;
          })
        );
        
        update(s => ({ ...s, agents: enrichedAgents, loading: false }));
      } catch (err: any) {
        update(s => ({ ...s, error: err.message, loading: false }));
      }
    },

    async registerAgent(name: string, description?: string): Promise<Agent | null> {
      update(s => ({ ...s, loading: true, error: null }));
      try {
        const res = await fetch(`${API_BASE}/agents/register`, {
          method: 'POST',
          headers: {
            ...getAuthHeader(),
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ name, description }),
        });
        if (!res.ok) throw new Error('Failed to register agent');
        const agent = await res.json();
        update(s => ({
          ...s,
          agents: [...s.agents, { ...agent, status: 'registered' }],
          loading: false,
        }));
        return agent;
      } catch (err: any) {
        update(s => ({ ...s, error: err.message, loading: false }));
        return null;
      }
    },

    async deleteAgent(agentId: string): Promise<boolean> {
      try {
        const res = await fetch(`${API_BASE}/agents/${agentId}`, {
          method: 'DELETE',
          headers: getAuthHeader(),
        });
        if (!res.ok) throw new Error('Failed to delete agent');
        update(s => ({
          ...s,
          agents: s.agents.filter(a => a.id !== agentId),
        }));
        return true;
      } catch (err: any) {
        update(s => ({ ...s, error: err.message }));
        return false;
      }
    },

    getToken(): string {
      const authState = get(auth);
      return authState.token || '';
    },

    getInstallScript(agentId: string): string {
      const authState = get(auth);
      const token = authState.token || '';
      const baseUrl = window.location.origin;
      return `curl -sSL ${baseUrl}/install-agent.sh | bash -s -- --agent-id ${agentId} --token ${token}`;
    },

    reset() {
      set({ agents: [], loading: false, error: null });
    },
  };
}

export const agents = createAgentsStore();
