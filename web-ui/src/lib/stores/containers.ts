import { writable, derived, get } from "svelte/store";
import { token } from "./auth";

// Types
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

    // Reset store
    reset() {
      set(initialState);
    },

    // Fetch all containers
    async fetchContainers() {
      update((state) => ({ ...state, isLoading: true, error: null }));

      const { data, error } = await apiCall<{
        containers: Container[];
        count: number;
        limit: number;
      }>("/api/containers");

      if (error) {
        update((state) => ({ ...state, isLoading: false, error }));
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
    async createContainer(name: string, image: string, customImage?: string) {
      update((state) => ({ ...state, isLoading: true, error: null }));

      const body: Record<string, string> = { name, image };
      if (image === "custom" && customImage) {
        body.custom_image = customImage;
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

      // Add new container to store
      update((state) => ({
        ...state,
        containers: [data!, ...state.containers],
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
          progress: 0,
          message: "Connecting...",
          stage: "initializing",
        },
      }));

      const body: Record<string, string> = { name, image };
      if (image === "custom" && customImage) {
        body.custom_image = customImage;
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
                      onError?.(event.error);
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
                          };

                          update((state) => ({
                            ...state,
                            containers: [container, ...state.containers],
                            creating: null,
                          }));

                          onComplete?.(container);
                        })
                        .catch(() => {
                          // Even if fetch fails, container was created
                          const container: Container = {
                            id: event.container_id!,
                            db_id: event.container_id,
                            user_id: "",
                            name,
                            image,
                            status: "running",
                            created_at: new Date().toISOString(),
                          };

                          update((state) => ({
                            ...state,
                            containers: [container, ...state.containers],
                            creating: null,
                          }));

                          onComplete?.(container);
                        });
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
                  onError?.(e.message);
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
                user_id: containerData.user_id,
                name: containerData.name,
                image: containerData.image,
                status: "running",
                created_at: containerData.created_at,
                ip_address: containerData.ip_address,
              };

              update((state) => ({
                ...state,
                containers: [container, ...state.containers],
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
