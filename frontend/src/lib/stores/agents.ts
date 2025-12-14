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
  // API token for this agent (only available right after registration)
  token?: string;
}

interface AgentsState {
  agents: Agent[];
  loading: boolean;
  error: string | null;
  // Map of agent ID to their API tokens (for newly registered agents)
  agentTokens: Record<string, string>;
}

const API_BASE = '/api';

function createAgentsStore() {
  const { subscribe, set, update } = writable<AgentsState>({
    agents: [],
    loading: false,
    error: null,
    agentTokens: {},
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
        
        // API returns sorted list, so use it as is
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
        // Add new agent to the top of the list and save the API token
        update(s => ({
          ...s,
          agents: [{ ...agent, status: 'registered' }, ...s.agents],
          loading: false,
          // Store the API token for this agent (used in getInstallScript)
          agentTokens: agent.token 
            ? { ...s.agentTokens, [agent.id]: agent.token }
            : s.agentTokens,
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
      // Get the current state to check for agent-specific API token
      const state = get({ subscribe });
      const agentToken = state.agentTokens[agentId];
      
      // Prefer the agent-specific API token (never expires)
      // Fall back to user's JWT token (expires in 24h) if no agent token available
      const token = agentToken || get(auth).token || '';
      const baseUrl = window.location.origin;
      
      // If using agent token, include it with a note that it's permanent
      return `curl -sSL ${baseUrl}/install-agent.sh | bash -s -- --agent-id ${agentId} --token ${token}`;
    },
    
    // Get the API token for a specific agent (if available)
    getAgentToken(agentId: string): string | undefined {
      const state = get({ subscribe });
      return state.agentTokens[agentId];
    },

    // Update agent status from WebSocket event
    updateAgentStatus(agentId: string, status: 'online' | 'offline' | 'registered', agentData?: any) {
      update(s => ({
        ...s,
        agents: s.agents.map(agent => {
          if (agent.id === agentId) {
            return {
              ...agent,
              status,
              ...(agentData?.system_info && { system_info: agentData.system_info }),
              ...(agentData?.stats && { stats: agentData.stats }),
              ...(agentData?.connected_at && { connected_at: agentData.connected_at }),
              // Also update OS/Arch/Shell if provided by agentData and more specific
              ...(agentData?.os && { os: agentData.os }),
              ...(agentData?.arch && { arch: agentData.arch }),
              ...(agentData?.shell && { shell: agentData.shell }),
            };
          }
          return agent;
        }),
      }));
    },

    reset() {
      set({ agents: [], loading: false, error: null });
    },
  };
}

export const agents = createAgentsStore();

// Listen for container WebSocket events to update agent status
if (typeof window !== 'undefined') {
  window.addEventListener('container-agent-connected', ((e: CustomEvent) => {
    const agentId = e.detail.id?.replace('agent:', '');
    if (agentId) {
      // Pass the whole e.detail as agentData to updateAgentStatus
      agents.updateAgentStatus(agentId, 'online', e.detail);
    }
  }) as EventListener);

  window.addEventListener('container-agent-disconnected', ((e: CustomEvent) => {
    const agentId = e.detail.id?.replace('agent:', '');
    if (agentId) {
      agents.updateAgentStatus(agentId, 'offline');
    }
  }) as EventListener);
}
