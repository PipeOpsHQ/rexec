import { writable, derived, get } from "svelte/store";
import { token } from "./auth";

// Types
export interface ContainerResources {
  memory_mb: number;
  cpu_shares: number;
  disk_mb: number;
}

export interface Container {
  id: string;
  db_id?: string;
  user_id: string;
  name: string;
  image: string;
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
  limit: 2,
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
        limit: data?.limit || 2,
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

    // Create container with SSE progress streaming (falls back to polling if SSE fails)
    createContainerWithProgress(
      name: string,
      image: string,
      customImage?: string,
      role?: string,
      onProgress?: (event: ProgressEvent) => void,
      onComplete?: (container: Container) => void,
      onError?: (error: string) => void,
      resources?: { memory_mb?: number; cpu_shares?: number; disk_mb?: number },
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

      const body: Record<string, string | number> = { name, image };
      if (image === "custom" && customImage) {
        body.custom_image = customImage;
      }
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

      // Try SSE endpoint first, fall back to polling if it fails
      fetch("/api/containers/stream", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}`,
        },
        body: JSON.stringify(body),
      })
        .then(async (response) => {
          const contentType = response.headers.get("content-type") || "";

          // Check if we got an SSE response or something else (like Cloudflare error page)
          if (!contentType.includes("text/event-stream")) {
            // Not SSE - likely Cloudflare blocking. Fall back to polling.
            console.warn(
              "SSE not available (got " +
              contentType +
              "), falling back to polling",
            );
            // Use the polling fallback
            this.createContainerFallback(
              name,
              image,
              customImage,
              onProgress,
              onComplete,
              onError,
            );
            return;
          }

          if (!response.ok) {
            const errorData = await response.json().catch(() => ({}));
            throw new Error(
              errorData.error ||
              `Request failed with status ${response.status}`,
            );
          }

          const reader = response.body?.getReader();
          const decoder = new TextDecoder();

          if (!reader) {
            throw new Error("No response body");
          }

          let buffer = "";
          let receivedAnyData = false;

          const processEvents = () => {
            const lines = buffer.split("\n");
            buffer = lines.pop() || ""; // Keep incomplete line in buffer

            for (const line of lines) {
              if (line.startsWith("data: ")) {
                receivedAnyData = true;
                try {
                  const event: ProgressEvent = JSON.parse(line.slice(6));

                  // Update store state
                  update((state) => ({
                    ...state,
                    creating: state.creating
                      ? {
                        ...state.creating,
                        progress: event.progress || state.creating.progress,
                        message: event.message || state.creating.message,
                        stage: event.stage || state.creating.stage,
                      }
                      : null,
                  }));

                  // Call progress callback
                  onProgress?.(event);

                  // Handle completion
                  if (event.complete) {
                    if (event.error) {
                      update((state) => ({ ...state, creating: null }));
                      onError?.(
                        typeof event.error === "string"
                          ? event.error
                          : "Container creation failed",
                      );
                    } else if (event.container_id) {
                      // Fetch the created container details
                      fetch(`/api/containers/${event.container_id}`, {
                        headers: {
                          Authorization: `Bearer ${authToken}`,
                        },
                      })
                        .then((res) => res.json())
                        .then((containerData) => {
                          const container: Container = {
                            id:
                              containerData.id ||
                              containerData.docker_id ||
                              event.container_id!,
                            db_id: containerData.db_id || event.container_id,
                            user_id: containerData.user_id,
                            name: containerData.name || name,
                            image: containerData.image || image,
                            status: "running",
                            created_at:
                              containerData.created_at ||
                              new Date().toISOString(),
                            ip_address: containerData.ip_address,
                            resources: containerData.resources,
                          };

                          update((state) => ({
                            ...state,
                            containers: [container, ...state.containers.filter(c => c.id !== container.id && c.db_id !== container.id)],
                            creating: null,
                          }));

                          onComplete?.(container);
                        })
                        .catch(() => {
                          // Even if fetch fails, container was created - try to parse resources from detail
                          let resources: ContainerResources | undefined;
                          try {
                            if (event.detail) {
                              const detailData = JSON.parse(event.detail);
                              resources = detailData.resources;
                            }
                          } catch { /* ignore parse errors */ }
                          
                          const container: Container = {
                            id: event.container_id!,
                            db_id: event.container_id,
                            user_id: "",
                            name,
                            image,
                            status: "running",
                            created_at: new Date().toISOString(),
                            resources,
                          };

                          update((state) => ({
                            ...state,
                            containers: [container, ...state.containers.filter(c => c.id !== container.id && c.db_id !== container.id)],
                            creating: null,
                          }));

                          onComplete?.(container);
                        });
                    } else {
                      // Completion without container_id - shouldn't happen but handle it
                      update((state) => ({ ...state, creating: null }));
                      onError?.("Container created but no ID received");
                    }
                  }
                } catch {
                  // Ignore parse errors for malformed SSE
                }
              }
            }
          };

          const read = async (): Promise<void> => {
            try {
              const { done, value } = await reader.read();

              if (done) {
                // Process any remaining buffer
                if (buffer) {
                  buffer += "\n";
                  processEvents();
                }
                // If we never received any SSE data, the stream was blocked
                if (!receivedAnyData) {
                  console.warn(
                    "SSE stream ended without data, falling back to polling",
                  );
                  this.createContainerFallback(
                    name,
                    image,
                    customImage,
                    onProgress,
                    onComplete,
                    onError,
                  );
                }
                return;
              }

              buffer += decoder.decode(value, { stream: true });
              processEvents();

              return read();
            } catch (e) {
              if (e instanceof Error && e.name !== "AbortError") {
                // On stream error, try polling fallback if we haven't received data
                if (!receivedAnyData) {
                  console.warn(
                    "SSE stream error, falling back to polling:",
                    e.message,
                  );
                  this.createContainerFallback(
                    name,
                    image,
                    customImage,
                    onProgress,
                    onComplete,
                    onError,
                  );
                } else {
                  update((state) => ({ ...state, creating: null }));
                  onError?.(e instanceof Error ? e.message : "Stream error");
                }
              }
            }
          };

          return read();
        })
        .catch((e) => {
          // Network error or fetch failed - try polling fallback
          console.warn("SSE fetch failed, falling back to polling:", e.message);
          this.createContainerFallback(
            name,
            image,
            customImage,
            onProgress,
            onComplete,
            onError,
          );
        });
    },

    // Container creation with polling for async backend
    async createContainerFallback(
      name: string,
      image: string,
      customImage?: string,
      onProgress?: (event: ProgressEvent) => void,
      onComplete?: (container: Container) => void,
      onError?: (error: string) => void,
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
          progress: 5,
          message: "Requesting container...",
          stage: "requesting",
        },
      }));

      onProgress?.({
        stage: "requesting",
        message: "Requesting container...",
        progress: 5,
      });

      const body: Record<string, string> = { name, image };
      if (image === "custom" && customImage) {
        body.custom_image = customImage;
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
          onError?.(data.error || "Failed to create container");
          return;
        }

        // Container creation is async - poll for status
        const containerId = data.db_id || data.id;

        update((state) => ({
          ...state,
          creating: state.creating
            ? {
              ...state.creating,
              progress: 20,
              message: "Pulling image and creating container...",
              stage: "creating",
            }
            : null,
        }));

        onProgress?.({
          stage: "creating",
          message:
            "Pulling image and creating container (this may take a minute)...",
          progress: 20,
        });

        // Poll for container status
        const maxAttempts = 120; // 2 minutes max
        const pollInterval = 1000; // 1 second
        let attempts = 0;

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
                // Container might not be in Docker yet, keep polling
                setTimeout(pollStatus, pollInterval);
                return;
              }
              throw new Error("Failed to get container status");
            }

            const containerData = await statusResponse.json();
            const status = containerData.status;

            // Update progress based on attempts
            const progress = Math.min(
              20 + Math.floor((attempts / maxAttempts) * 70),
              90,
            );

            update((state) => ({
              ...state,
              creating: state.creating
                ? {
                  ...state.creating,
                  progress,
                  message:
                    status === "creating"
                      ? "Still creating..."
                      : `Status: ${status}`,
                  stage: status,
                }
                : null,
            }));

            if (status === "running") {
              // Container is ready!
              const container: Container = {
                id: containerData.id || containerData.docker_id || containerId,
                db_id: containerData.db_id || containerId,
                user_id: containerData.user_id || "",
                name: containerData.name || name,
                image: containerData.image || image,
                status: "running",
                created_at:
                  containerData.created_at || new Date().toISOString(),
                ip_address: containerData.ip_address,
                resources: containerData.resources,
              };

              update((state) => ({
                ...state,
                containers: [container, ...state.containers.filter(c => c.id !== container.id && c.db_id !== container.id)],
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
              update((state) => ({ ...state, creating: null }));
              onError?.("Container creation failed. Please try again.");
              return;
            }

            if (attempts >= maxAttempts) {
              update((state) => ({ ...state, creating: null }));
              onError?.(
                "Container creation timed out. Please check your containers list.",
              );
              return;
            }

            // Keep polling
            setTimeout(pollStatus, pollInterval);
          } catch (e) {
            if (attempts >= maxAttempts) {
              update((state) => ({ ...state, creating: null }));
              onError?.(
                e instanceof Error
                  ? e.message
                  : "Failed to check container status",
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
          e instanceof Error ? e.message : "Failed to create container",
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
    async deleteContainer(id: string) {
      const { error } = await apiCall(`/api/containers/${id}`, {
        method: "DELETE",
      });

      if (error) {
        return { success: false, error };
      }

      update((state) => ({
        ...state,
        containers: state.containers.filter((c) => c.id !== id),
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
}

// Export the store
export const containers = createContainersStore();

// WebSocket connection for real-time updates
let eventsSocket: WebSocket | null = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_DELAY = 3000;

// Get WebSocket URL
function getWebSocketUrl(): string {
  const currentToken = get(token);
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const host = window.location.host;
  return `${protocol}//${host}/api/containers/events?token=${currentToken}`;
}

// WebSocket connection status
export const wsConnected = writable(false);

// Start WebSocket connection for real-time container updates
export function startContainerEvents() {
  if (eventsSocket?.readyState === WebSocket.OPEN) return;

  const currentToken = get(token);
  if (!currentToken) return;

  try {
    eventsSocket = new WebSocket(getWebSocketUrl());

    eventsSocket.onopen = () => {
      console.log("[ContainerEvents] WebSocket connected");
      reconnectAttempts = 0;
      wsConnected.set(true);
    };

    eventsSocket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        handleContainerEvent(data);
      } catch (e) {
        console.error("[ContainerEvents] Failed to parse message:", e);
      }
    };

    eventsSocket.onclose = (event) => {
      console.log("[ContainerEvents] WebSocket closed:", event.code);
      eventsSocket = null;
      wsConnected.set(false);

      // Attempt to reconnect if not intentionally closed
      if (event.code !== 1000 && reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
        reconnectAttempts++;
        console.log(`[ContainerEvents] Reconnecting (attempt ${reconnectAttempts})...`);
        setTimeout(startContainerEvents, RECONNECT_DELAY);
      }
    };

    eventsSocket.onerror = (error) => {
      console.error("[ContainerEvents] WebSocket error:", error);
      wsConnected.set(false);
    };
  } catch (e) {
    console.error("[ContainerEvents] Failed to create WebSocket:", e);
  }
}

// Stop WebSocket connection
export function stopContainerEvents() {
  if (eventsSocket) {
    eventsSocket.close(1000, "User logged out");
    eventsSocket = null;
  }
  reconnectAttempts = 0;
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
  console.log("[ContainerEvents] Received event:", type, containerData);

  switch (type) {
    case "list":
      // Full container list received
      containers.update((state) => ({
        ...state,
        containers: containerData.containers || [],
        limit: containerData.limit || 2,
        isLoading: false,
      }));
      break;

    case "created":
      // New container created - also clear creating state
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
      // Container status changed - merge all data including resources
      containers.update((state) => {
        const newContainers = state.containers.map((c) => {
          if (matchesContainer(c, containerData)) {
            // Create a completely new object to ensure reactivity
            return { 
              ...c, 
              ...containerData,
              status: containerData.status || c.status,
              resources: containerData.resources || c.resources,
            };
          }
          return c;
        });
        
        // Return new state object
        return {
          ...state,
          containers: newContainers,
        };
      });
      break;

    case "deleted":
      // Container deleted
      containers.update((state) => ({
        ...state,
        containers: state.containers.filter((c) => !matchesContainer(c, containerData)),
      }));
      break;

    default:
      console.log("[ContainerEvents] Unknown event type:", type);
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
