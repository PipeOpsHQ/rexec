import { writable, derived, get } from "svelte/store";
import { token } from "./auth";
import { createRexecWebSocket } from "../utils/ws";

// Types
export interface ContainerResources {
  memory_mb: number;
  cpu_shares: number;
  disk_mb: number;
}

export interface PortForward {
  id: string;
  name: string;
  container_id: string;
  container_port: number;
  local_port: number;
  protocol: string;
  is_active: boolean;
  created_at: string;
  websocket_url: string;
  proxy_url: string;
}

export interface Container {
  id: string;
  db_id?: string;
  user_id: string;
  name: string;
  image: string;
  role?: string; // Role/environment: devops, node, python, go, etc.
  status:
  | "running"
  | "stopped"
  | "creating"
  | "starting"
  | "stopping"
  | "error";
  created_at: string;
  last_used_at?: string;
  idle_seconds?: number;
  ip_address?: string;
  resources?: ContainerResources;
  portForwards?: PortForward[]; // New field for port forwards
  session_type?: string; // Session type: container, agent, gpu, ssh, custom
  // Agent-specific fields
  os?: string;
  arch?: string;
  shell?: string;
  distro?: string; // Linux distribution (ubuntu, debian, fedora, etc.)
  hostname?: string;
  region?: string;
  description?: string; // Agent description
  stats?: {
    cpu_percent?: number;
    memory?: number;
    memory_limit?: number;
    disk_usage?: number;
    disk_limit?: number;
  };
}

export interface CreatingContainer {
  name: string;
  image: string;
  progress: number;
  message: string;
  stage: string;
}

export interface ContainersState {
  containers: Container[];
  isLoading: boolean;
  error: string | null;
  limit: number;
  creating: CreatingContainer | null;
}

// Initial state
const initialState: ContainersState = {
  containers: [],
  isLoading: false,
  error: null,
  limit: 5,
  creating: null,
};

// Helper to get auth token
function getToken(): string | null {
  return get(token);
}

// Helper for API calls
async function apiCall<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<{ data?: T; error?: string; status: number }> {
  const authToken = getToken();
  if (!authToken) {
    return { error: "Not authenticated", status: 401 };
  }

  try {
    const response = await fetch(endpoint, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
        ...options.headers,
      },
    });

    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
      return {
        error: data.error || `Request failed with status ${response.status}`,
        status: response.status,
      };
    }

    return { data, status: response.status };
  } catch (e) {
    return {
      error: e instanceof Error ? e.message : "Network error",
      status: 0,
    };
  }
}

// Create the store
function createContainersStore() {
  const { subscribe, set, update } = writable<ContainersState>(initialState);

  return {
    subscribe,
    update, // Expose update for WebSocket handler

    // Reset store
    reset() {
      set(initialState);
    },

    // Fetch all containers
    async fetchContainers(silent = false) {
      if (!silent) {
        update((state) => ({ ...state, isLoading: true, error: null }));
      }

      // Server now returns both containers AND online agents in one call
      const { data, error } = await apiCall<{
        containers: Container[];
        count: number;
        limit: number;
      }>("/api/containers");

      if (error) {
        update((state) => ({ ...state, isLoading: false, error: silent ? state.error : error }));
        return { success: false, error };
      }

      update((state) => ({
        ...state,
        containers: data?.containers || [],
        limit: data?.limit || 5,
        isLoading: false,
        error: null,
      }));

      return { success: true, containers: data?.containers || [] };
    },

    // Get a single container
    async getContainer(id: string) {
      const { data, error } = await apiCall<Container>(`/api/containers/${id}`);

      if (error) {
        return { success: false, error };
      }

      // Update container in store if it exists
      update((state) => ({
        ...state,
        containers: state.containers.map((c) =>
          c.id === id ? { ...c, ...data } : c,
        ),
      }));

      return { success: true, container: data };
    },

    // Create a new container
    async createContainer(
      name: string,
      image: string,
      customImage?: string,
      role?: string,
    ) {
      update((state) => ({ ...state, isLoading: true, error: null }));

      const body: Record<string, string> = { name, image };
      if (image === "custom" && customImage) {
        body.custom_image = customImage;
      }
      if (role) {
        body.role = role;
      }

      const { data, error, status } = await apiCall<Container>(
        "/api/containers",
        {
          method: "POST",
          body: JSON.stringify(body),
        },
      );

      if (error) {
        update((state) => ({ ...state, isLoading: false, error }));
        return { success: false, error, status };
      }

      // Add new container to store (dedupe by id)
      update((state) => ({
        ...state,
        containers: [data!, ...state.containers.filter(c => c.id !== data!.id && c.db_id !== data!.id)],
        isLoading: false,
        error: null,
      }));

      return { success: true, container: data };
    },

    // Create container with progress via WebSocket events (uses polling as primary method)
    createContainerWithProgress(
      name: string,
      image: string,
      customImage?: string,
      _role?: string, // Role is sent to backend but not used client-side
      onProgress?: (event: ProgressEvent) => void,
      onComplete?: (container: Container) => void,
      onError?: (error: string) => void,
      resources?: { memory_mb?: number; cpu_shares?: number; disk_mb?: number },
      shellOptions?: { use_tmux?: boolean },
    ) {
      const authToken = getToken();
      if (!authToken) {
        onError?.("Not authenticated");
        return;
      }

      // Set creating state
      update((state) => ({
        ...state,
        creating: {
          name,
          image,
          progress: 0,
          message: "Connecting...",
          stage: "initializing",
        },
      }));

      // Set up WebSocket event listeners for progress updates
      const handleProgress = (e: CustomEvent) => {
        const data = e.detail;
        // Call the onProgress callback with WebSocket progress data
        onProgress?.({
          stage: data.stage,
          message: data.message,
          progress: data.progress,
          complete: data.complete,
        });
      };

      const handleCreated = (e: CustomEvent) => {
        const container = e.detail;
        cleanup();
        
        const containerObj: Container = {
          id: container.id,
          db_id: container.db_id || container.id,
          user_id: container.user_id,
          name: container.name || name,
          image: container.image || image,
          status: "running",
          created_at: container.created_at || new Date().toISOString(),
          ip_address: container.ip_address,
          resources: container.resources,
        };

        update((state) => ({
          ...state,
          containers: [containerObj, ...state.containers.filter(c => {
            // Robust deduplication: check all ID combinations
            if (c.id === containerObj.id) return false;
            if (c.db_id && c.db_id === containerObj.db_id) return false;
            if (c.id === containerObj.db_id) return false; // Existing ID matches new DB_ID
            return true;
          })],
          creating: null,
        }));

        onProgress?.({
          stage: "ready",
          message: "Terminal ready!",
          progress: 100,
          complete: true,
          container_id: container.id,
        });

        onComplete?.(containerObj);
      };

      const handleError = (e: CustomEvent) => {
        cleanup();
        update((state) => ({ ...state, creating: null }));
        onError?.(e.detail?.error || "Terminal creation failed");
      };

      const cleanup = () => {
        if (typeof window !== 'undefined') {
          window.removeEventListener('container-progress', handleProgress as EventListener);
          window.removeEventListener('container-created', handleCreated as EventListener);
          window.removeEventListener('container-error', handleError as EventListener);
        }
      };

      if (typeof window !== 'undefined') {
        window.addEventListener('container-progress', handleProgress as EventListener);
        window.addEventListener('container-created', handleCreated as EventListener);
        window.addEventListener('container-error', handleError as EventListener);
      }

      // Use polling-based creation (which the backend sends WebSocket progress events for)
      this.createContainerFallback(
        name,
        image,
        customImage,
        _role,
        onProgress,
        onComplete,
        onError,
        resources,
        shellOptions,
      );
    },

    // Container creation with polling for async backend
    async createContainerFallback(
      name: string,
      image: string,
      customImage?: string,
      role?: string,
      onProgress?: (event: ProgressEvent) => void,
      onComplete?: (container: Container) => void,
      onError?: (error: string) => void,
      resources?: { memory_mb?: number; cpu_shares?: number; disk_mb?: number },
      shellOptions?: { use_tmux?: boolean },
    ) {
      const authToken = getToken();
      if (!authToken) {
        onError?.("Not authenticated");
        return;
      }

      // Set creating state - use the same stages as SSE
      update((state) => ({
        ...state,
        creating: {
          name,
          image,
          progress: 5,
          message: "Validating request...",
          stage: "validating",
        },
      }));

      onProgress?.({
        stage: "validating",
        message: "Validating request...",
        progress: 5,
      });

      const body: Record<string, any> = { name, image };
      if (image === "custom" && customImage) {
        body.custom_image = customImage;
      }
      // Add role if specified
      if (role) {
        body.role = role;
      }
      // Add resource customization if provided
      if (resources?.memory_mb) {
        body.memory_mb = resources.memory_mb;
      }
      if (resources?.cpu_shares) {
        body.cpu_shares = resources.cpu_shares;
      }
      if (resources?.disk_mb) {
        body.disk_mb = resources.disk_mb;
      }
      // Add shell options if provided
      if (shellOptions) {
        body.shell = {
          use_tmux: shellOptions.use_tmux
        };
      }

      try {
        const response = await fetch("/api/containers", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${authToken}`,
          },
          body: JSON.stringify(body),
        });

        const data = await response.json();

        if (!response.ok) {
          update((state) => ({ ...state, creating: null }));
          onError?.(data.error || "Failed to create terminal");
          return;
        }

        // Container creation is async - poll for status
        const containerId = data.db_id || data.id;

        // Initial progress update - validation complete, now pulling
        // This ensures immediate UI update without waiting for WebSocket
        update((state) => ({
          ...state,
          creating: state.creating
            ? {
              ...state.creating,
              progress: 15,
              message: "Pulling image...",
              stage: "pulling",
            }
            : null,
        }));

        // Notify callback immediately so UI updates
        onProgress?.({
          stage: "pulling",
          message: "Pulling image...",
          progress: 15,
        });

        // Poll for container status (as backup/timeout mechanism)
        // WebSocket events will provide real-time progress updates
        const maxAttempts = 120; // 2 minutes max
        const pollInterval = 1000; // 1 second
        let attempts = 0;

        // Stage definitions (used for fallback when WebSocket isn't working)
        const stageConfig: Record<string, { progress: number; message: string }> = {
          pulling: { progress: 15, message: "Pulling image..." },
          creating: { progress: 35, message: "Creating container..." },
          starting: { progress: 55, message: "Starting container..." },
          configuring: { progress: 75, message: "Configuring environment..." },
          ready: { progress: 100, message: "Ready!" },
        };

        // Track if we're receiving WebSocket progress events
        let receivedWsProgress = false;
        const wsProgressHandler = () => { receivedWsProgress = true; };
        if (typeof window !== 'undefined') {
          window.addEventListener('container-progress', wsProgressHandler);
        }

        const pollStatus = async (): Promise<void> => {
          attempts++;

          try {
            const statusResponse = await fetch(
              `/api/containers/${containerId}`,
              {
                headers: {
                  Authorization: `Bearer ${authToken}`,
                },
              },
            );

            if (!statusResponse.ok) {
              if (statusResponse.status === 404 && attempts < 5) {
                // Container might not be in Docker yet - still pulling
                setTimeout(pollStatus, pollInterval);
                return;
              }
              throw new Error("Failed to get terminal status");
            }

            const containerData = await statusResponse.json();
            const status = containerData.status;

            // Check if creating state was cleared (by WebSocket completing)
            let creatingCleared = false;
            const unsub = subscribe((state) => {
              if (!state.creating) {
                creatingCleared = true;
              }
            });
            unsub();
            
            if (creatingCleared) {
              // Already completed via WebSocket event
              if (typeof window !== 'undefined') {
                window.removeEventListener('container-progress', wsProgressHandler);
              }
              return;
            }

            // Update progress based on container status (fallback when WebSocket isn't working)
            if (!receivedWsProgress && status in stageConfig) {
              const stageInfo = stageConfig[status];
              update((state) => ({
                ...state,
                creating: state.creating
                  ? {
                    ...state.creating,
                    progress: stageInfo.progress,
                    message: stageInfo.message,
                    stage: status,
                  }
                  : null,
              }));
              onProgress?.({
                stage: status,
                message: stageInfo.message,
                progress: stageInfo.progress,
              });
            }

            // If stuck in configuring for too long (30+ seconds), treat as ready
            // Shell setup continues in background but user can connect
            if (status === "configuring" && attempts >= 30) {
              if (typeof window !== 'undefined') {
                window.removeEventListener('container-progress', wsProgressHandler);
              }
              
              const container: Container = {
                id: containerData.id || containerData.docker_id || containerId,
                db_id: containerData.db_id || containerId,
                user_id: containerData.user_id,
                name: containerData.name || name,
                image: containerData.image || image,
                status: "running",
                created_at: containerData.created_at || new Date().toISOString(),
                ip_address: containerData.ip_address,
                resources: containerData.resources,
              };

              update((state) => ({
                ...state,
                containers: [container, ...state.containers.filter(c => {
                  if (c.id === container.id) return false;
                  if (c.db_id && c.db_id === container.db_id) return false;
                  if (c.id === container.db_id) return false;
                  return true;
                })],
                creating: null,
              }));

              onProgress?.({
                stage: "ready",
                message: "Terminal ready (setup continuing in background)",
                progress: 100,
                complete: true,
              });

              onComplete?.(container);
              return;
            }

            if (status === "running") {
              // Container is running - complete immediately
              // Shell setup runs in background, user can connect right away
              
              if (typeof window !== 'undefined') {
                window.removeEventListener('container-progress', wsProgressHandler);
              }
              
              const container: Container = {
                id: containerData.id || containerData.docker_id || containerId,
                db_id: containerData.db_id || containerId,
                user_id: containerData.user_id || "",
                name: containerData.name || name,
                image: containerData.image || image,
                status: "running",
                created_at: containerData.created_at || new Date().toISOString(),
                ip_address: containerData.ip_address,
                resources: containerData.resources,
              };

              update((state) => ({
                ...state,
                containers: [container, ...state.containers.filter(c => {
                  if (c.id === container.id) return false;
                  if (c.db_id && c.db_id === container.db_id) return false;
                  if (c.id === container.db_id) return false;
                  return true;
                })],
                creating: null,
              }));

              onProgress?.({
                stage: "ready",
                message: "Terminal ready!",
                progress: 100,
                complete: true,
                container_id: container.id,
              });

              onComplete?.(container);
              return;
            }

            if (status === "error") {
              if (typeof window !== 'undefined') {
                window.removeEventListener('container-progress', wsProgressHandler);
              }
              update((state) => ({ ...state, creating: null }));
              onError?.("Terminal creation failed. Please try again.");
              return;
            }

            if (attempts >= maxAttempts) {
              if (typeof window !== 'undefined') {
                window.removeEventListener('container-progress', wsProgressHandler);
              }
              update((state) => ({ ...state, creating: null }));
              onError?.(
                "Terminal creation timed out. Please check your terminals list.",
              );
              return;
            }

            // Keep polling
            setTimeout(pollStatus, pollInterval);
          } catch (e) {
            if (attempts >= maxAttempts) {
              if (typeof window !== 'undefined') {
                window.removeEventListener('container-progress', wsProgressHandler);
              }
              update((state) => ({ ...state, creating: null }));
              onError?.(
                e instanceof Error
                  ? e.message
                  : "Failed to check terminal status",
              );
            } else {
              // Retry on error
              setTimeout(pollStatus, pollInterval);
            }
          }
        };

        // Start polling
        setTimeout(pollStatus, pollInterval);
      } catch (e) {
        update((state) => ({ ...state, creating: null }));
        onError?.(
          e instanceof Error ? e.message : "Failed to create terminal",
        );
      }
    },

    // Start a container
    async startContainer(id: string) {
      update((state) => ({
        ...state,
        containers: state.containers.map((c) =>
          c.id === id ? { ...c, status: "starting" as const } : c,
        ),
      }));

      const { data, error } = await apiCall<{
        id: string;
        status: string;
        recreated?: boolean;
      }>(`/api/containers/${id}/start`, { method: "POST" });

      if (error) {
        update((state) => ({
          ...state,
          containers: state.containers.map((c) =>
            c.id === id ? { ...c, status: "stopped" as const } : c,
          ),
        }));
        return { success: false, error };
      }

      // Handle recreated container (new ID)
      if (data?.recreated && data.id !== id) {
        // Update any active terminal sessions for this container
        import("./terminal")
          .then(({ terminal }) => terminal.updateSessionContainerId(id, data.id))
          .catch(() => {
            // Ignore - terminal store may not be loaded in some flows
          });
        
        update((state) => ({
          ...state,
          containers: state.containers.map((c) =>
            c.id === id ? { ...c, id: data.id, status: "running" as const } : c,
          ),
        }));
        return { success: true, newId: data.id, recreated: true };
      }

      update((state) => ({
        ...state,
        containers: state.containers.map((c) =>
          c.id === id ? { ...c, status: "running" as const } : c,
        ),
      }));

      return { success: true };
    },

    // Stop a container
    async stopContainer(id: string) {
      update((state) => ({
        ...state,
        containers: state.containers.map((c) =>
          c.id === id ? { ...c, status: "stopping" as const } : c,
        ),
      }));

      const { error } = await apiCall(`/api/containers/${id}/stop`, {
        method: "POST",
      });

      if (error) {
        update((state) => ({
          ...state,
          containers: state.containers.map((c) =>
            c.id === id ? { ...c, status: "running" as const } : c,
          ),
        }));
        return { success: false, error };
      }

      update((state) => ({
        ...state,
        containers: state.containers.map((c) =>
          c.id === id ? { ...c, status: "stopped" as const } : c,
        ),
      }));

      return { success: true };
    },

    // Delete a container
    async deleteContainer(id: string, dbId?: string) {
      // Use db_id as fallback if id is empty (for error state containers)
      const deleteId = id || dbId;
      if (!deleteId) {
        return { success: false, error: "No terminal ID provided" };
      }
      
      const { error } = await apiCall(`/api/containers/${deleteId}`, {
        method: "DELETE",
      });

      if (error) {
        return { success: false, error };
      }

      update((state) => ({
        ...state,
        containers: state.containers.filter((c) => c.id !== id && c.id !== dbId && c.db_id !== id && c.db_id !== dbId),
      }));

      return { success: true };
    },

    // Update container status locally
    updateStatus(id: string, status: Container["status"]) {
      update((state) => ({
        ...state,
        containers: state.containers.map((c) =>
          c.id === id ? { ...c, status } : c,
        ),
      }));
    },

    // Clear creating state (useful after settings updates)
    clearCreating() {
      update((state) => ({ ...state, creating: null }));
    },
  };
}

// Progress event type
export interface ProgressEvent {
  stage: string;
  message: string;
  progress: number;
  detail?: string;
  error?: string;
  complete?: boolean;
  container_id?: string;
  container?: {
    id: string;
    db_id?: string;
    user_id: string;
    name: string;
    image: string;
    status: string;
    created_at: string;
    ip_address?: string;
    resources?: ContainerResources;
    guest?: boolean;
    expires_at?: string;
    session_limit_seconds?: number;
  };
}

// Export the store
export const containers = createContainersStore();

// WebSocket connection for real-time updates
let eventsSocket: WebSocket | null = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 15;           // More attempts for resilience
const RECONNECT_BASE_DELAY = 250;            // Start with 250ms for fast reconnect
const RECONNECT_MAX_DELAY = 30000;           // Max 30s between retries
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let pingInterval: ReturnType<typeof setInterval> | null = null;
let lastPongTime = 0;

// Get WebSocket URL
function getWebSocketUrl(): string {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const host = window.location.host;
  return `${protocol}//${host}/api/containers/events`;
}

// WebSocket connection status
export const wsConnected = writable(false);

// Calculate exponential backoff delay
function getReconnectDelay(): number {
  return Math.min(RECONNECT_BASE_DELAY * Math.pow(1.5, reconnectAttempts), RECONNECT_MAX_DELAY);
}

// Start WebSocket connection for real-time container updates
export function startContainerEvents() {
  // Clear any pending reconnect
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  
  if (eventsSocket?.readyState === WebSocket.OPEN || eventsSocket?.readyState === WebSocket.CONNECTING) {
    return;
  }

  const currentToken = get(token);
  if (!currentToken) return;

  try {
    eventsSocket = createRexecWebSocket(getWebSocketUrl(), currentToken);

    eventsSocket.onopen = () => {
      wsConnected.set(true);
      reconnectAttempts = 0; // Reset on successful connect
      lastPongTime = Date.now();

      // Start client-side ping to keep connection alive
      if (pingInterval) clearInterval(pingInterval);
      pingInterval = setInterval(() => {
        if (eventsSocket?.readyState === WebSocket.OPEN) {
          try {
            eventsSocket.send(JSON.stringify({ type: "ping" }));
          } catch {
            // Connection might be closing
          }

          // Check if we haven't received activity in too long (2.5 minutes)
          if (lastPongTime && Date.now() - lastPongTime > 150000) {
            console.warn("[ContainerEvents] No activity in 2.5 minutes, reconnecting...");
            eventsSocket?.close(4000, "Ping timeout");
          }
        }
      }, 25000); // Ping every 25s
    };

    eventsSocket.onmessage = (event) => {
      // Any message counts as activity
      lastPongTime = Date.now();

      try {
        const data = JSON.parse(event.data);

        // Handle pong response silently
        if (data.type === "pong") {
          return;
        }

        handleContainerEvent(data);
      } catch (e) {
        console.error("[ContainerEvents] Failed to parse message:", e);
      }
    };

    eventsSocket.onclose = (event) => {
      eventsSocket = null;
      wsConnected.set(false);

      // Clear ping interval
      if (pingInterval) {
        clearInterval(pingInterval);
        pingInterval = null;
      }

      // Don't reconnect if intentionally closed or auth issue
      const isIntentionalClose = event.code === 1000;
      const isAuthError = event.code === 4001 || event.code === 4003;
      
      if (isIntentionalClose || isAuthError) {
        reconnectAttempts = 0;
        return;
      }

      // Attempt silent reconnect with exponential backoff
      if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
        reconnectAttempts++;
        const delay = getReconnectDelay();
        reconnectTimer = setTimeout(startContainerEvents, delay);
      } else {
        // Reset after a longer delay to allow manual refresh or page reload
        reconnectTimer = setTimeout(() => {
          reconnectAttempts = 0;
          startContainerEvents();
        }, 60000); // Retry after 1 minute
      }
    };

    eventsSocket.onerror = () => {
      // Error will trigger onclose, no need to handle separately
    };
  } catch (e) {
    console.error("[ContainerEvents] Failed to create WebSocket:", e);
    // Retry after delay
    if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
      reconnectAttempts++;
      reconnectTimer = setTimeout(startContainerEvents, getReconnectDelay());
    }
  }
}

// Stop WebSocket connection
export function stopContainerEvents() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (pingInterval) {
    clearInterval(pingInterval);
    pingInterval = null;
  }
  if (eventsSocket) {
    eventsSocket.close(1000, "User logged out");
    eventsSocket = null;
  }
  reconnectAttempts = 0;
  wsConnected.set(false);
}

// Helper to match container by any ID field
function matchesContainer(container: Container, eventData: any): boolean {
  const eventId = eventData.id;
  const eventDbId = eventData.db_id;
  
  // Match by docker ID or db_id
  if (eventId && (container.id === eventId || container.db_id === eventId)) return true;
  if (eventDbId && (container.id === eventDbId || container.db_id === eventDbId)) return true;
  
  return false;
}

// Handle incoming container events
function handleContainerEvent(event: {
  type: string;
  container: any;
  timestamp: string;
}) {
  const { type, container: containerData } = event;

  switch (type) {
    case "list":
      // Full container list received - use smart merge to avoid card rearrangement
      containers.update((state) => {
        const newContainers = containerData.containers || [];
        const existingContainers = state.containers;

        // If this is the first load or we have no existing containers, just use the new list
        if (existingContainers.length === 0) {
          return {
            ...state,
            containers: newContainers,
            limit: containerData.limit || 5,
            isLoading: false,
          };
        }

        // Create a map for quick lookup of new container data by ID
        const newContainerMap = new Map<string, Container>();
        for (const c of newContainers) {
          newContainerMap.set(c.id, c);
        }

        const existingIds = new Set(existingContainers.map((c) => c.id));

        // Update existing containers IN PLACE (preserving their exact order)
        // Only update properties, don't recreate objects unless data changed
        const updatedContainers = existingContainers
          .filter((existing) => newContainerMap.has(existing.id)) // Remove deleted ones
          .map((existing) => {
            const updated = newContainerMap.get(existing.id);
            if (!updated) return existing;
            // Only create new object if something actually changed
            if (existing.status === updated.status &&
                existing.name === updated.name &&
                JSON.stringify(existing.stats) === JSON.stringify(updated.stats)) {
              return existing; // No change, keep same reference
            }
            return { ...existing, ...updated };
          });

        // Add any new containers AT THE END (not beginning) to avoid re-ordering
        const addedContainers = newContainers.filter((n: Container) => !existingIds.has(n.id));

        return {
          ...state,
          containers: [...updatedContainers, ...addedContainers],
          limit: containerData.limit || 5,
          isLoading: false,
        };
      });
      break;

    case "progress":
      // Update creating state with progress
      containers.update((state) => {
        // Only update if we're currently creating something
        if (!state.creating) return state;
        
        return {
          ...state,
          creating: {
            ...state.creating,
            progress: containerData.progress || state.creating.progress,
            message: containerData.message || state.creating.message,
            stage: containerData.stage || state.creating.stage,
          },
        };
      });
      
      // Dispatch progress event for any listeners (e.g., createContainerWithProgress callbacks)
      if (typeof window !== 'undefined') {
        window.dispatchEvent(new CustomEvent('container-progress', {
          detail: {
            id: containerData.id,
            stage: containerData.stage,
            message: containerData.message,
            progress: containerData.progress,
            complete: containerData.complete,
          }
        }));
      }
      
      // If this is a completion event with container data, dispatch to any listeners
      if (containerData.complete && containerData.container) {
        // Dispatch a custom event for components listening for container creation
        if (typeof window !== 'undefined') {
          window.dispatchEvent(new CustomEvent('container-created', {
            detail: containerData.container
          }));
        }
      } else if (containerData.complete && containerData.error) {
        // Dispatch error event
        if (typeof window !== 'undefined') {
          window.dispatchEvent(new CustomEvent('container-error', {
            detail: { error: containerData.error, id: containerData.id }
          }));
        }
      }
      break;

    case "created":
      // New container created - prepend to list
      containers.update((state) => ({
        ...state,
        containers: [
          containerData,
          ...state.containers.filter((c) => !matchesContainer(c, containerData)),
        ],
        creating: null, // Clear creating state
      }));
      break;

    case "started":
    case "stopped":
    case "updated":
      // Container status changed - update in place without recreating array if possible
      containers.update((state) => {
        const targetIndex = state.containers.findIndex((c) => matchesContainer(c, containerData));
        if (targetIndex === -1) return state; // Container not found

        const existing = state.containers[targetIndex];
        const newStatus = containerData.status || existing.status;

        // Check if anything actually changed
        if (existing.status === newStatus &&
            existing.name === (containerData.name || existing.name)) {
          return state; // No change
        }

        // Only update the specific container
        const updatedContainers = [...state.containers];
        updatedContainers[targetIndex] = {
          ...existing,
          ...containerData,
          status: newStatus,
          resources: containerData.resources || existing.resources,
        };

        return { ...state, containers: updatedContainers };
      });
      break;

    case "deleted":
      // Container deleted
      containers.update((state) => ({
        ...state,
        containers: state.containers.filter((c) => !matchesContainer(c, containerData)),
      }));
      break;

    case "agent_connected":
      // Agent connected - update in place if exists (reconnecting), or add if new
      containers.update((state) => {
        const existingIndex = state.containers.findIndex((c) => c.id === containerData.id);
        if (existingIndex >= 0) {
          // Agent exists - update in place, keeping position
          const updatedContainers = [...state.containers];
          updatedContainers[existingIndex] = {
            ...updatedContainers[existingIndex],
            ...containerData,
            status: "running", // Mark as online/running
          };
          return { ...state, containers: updatedContainers };
        } else {
          // New agent - add to beginning of list
          return {
            ...state,
            containers: [containerData, ...state.containers],
          };
        }
      });
      // Dispatch event for agents store
      if (typeof window !== 'undefined') {
        window.dispatchEvent(new CustomEvent('container-agent-connected', { detail: containerData }));
      }
      break;

    case "agent_disconnected":
      // Agent disconnected - update status to offline but keep in list at same position
      containers.update((state) => {
        const targetIndex = state.containers.findIndex((c) => c.id === containerData.id);
        if (targetIndex === -1) return state; // Agent not found

        const existing = state.containers[targetIndex];
        if (existing.status === "offline") return state; // Already offline, no change

        // Only update the specific container
        const updatedContainers = [...state.containers];
        updatedContainers[targetIndex] = {
          ...existing,
          status: "offline",
          stats: undefined,
        };

        return { ...state, containers: updatedContainers };
      });
      // Dispatch event for agents store
      if (typeof window !== 'undefined') {
        window.dispatchEvent(new CustomEvent('container-agent-disconnected', { detail: containerData }));
      }
      break;

    case "agent_stats":
      // Agent stats updated - update the stats for the matching agent
      // Use minimal updates to avoid triggering re-renders
      containers.update((state) => {
        const targetIndex = state.containers.findIndex((c) => c.id === containerData.id);
        if (targetIndex === -1) return state; // Agent not found, no change

        const existing = state.containers[targetIndex];
        // Check if stats actually changed
        if (JSON.stringify(existing.stats) === JSON.stringify(containerData.stats)) {
          return state; // No change, return same state reference
        }

        // Only update the specific container, keep array structure stable
        const updatedContainers = [...state.containers];
        updatedContainers[targetIndex] = { ...existing, stats: containerData.stats };

        return {
          ...state,
          containers: updatedContainers,
        };
      });
      break;

    default:
      break;
  }
}

// Legacy polling functions (kept for fallback)
let refreshInterval: ReturnType<typeof setInterval> | null = null;

export function startAutoRefresh() {
  // Use WebSocket for real-time updates
  startContainerEvents();
}

export function stopAutoRefresh() {
  stopContainerEvents();

  if (refreshInterval) {
    clearInterval(refreshInterval);
    refreshInterval = null;
  }
}

// Derived stores
export const runningContainers = derived(containers, ($containers) =>
  $containers.containers.filter((c) => c.status === "running"),
);

export const stoppedContainers = derived(containers, ($containers) =>
  $containers.containers.filter((c) => c.status === "stopped"),
);

export const containerCount = derived(
  containers,
  ($containers) => $containers.containers.length,
);

export const isAtLimit = derived(
  containers,
  ($containers) => $containers.containers.length >= $containers.limit,
);

export const isCreating = derived(
  containers,
  ($containers) => $containers.creating !== null,
);

export const creatingContainer = derived(
  containers,
  ($containers) => $containers.creating,
);
