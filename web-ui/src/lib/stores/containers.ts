import { writable, derived, get } from "svelte/store";
import { token } from "./auth";

// Types
export interface Container {
  id: string;
  db_id?: string;
  user_id: string;
  name: string;
  image: string;
  status: "running" | "stopped" | "creating" | "error";
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

    // Create container with progress (SSE with fallback to regular endpoint)
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

      const body: Record<string, string> = { name, image };
      if (image === "custom" && customImage) {
        body.custom_image = customImage;
      }

      // Set creating state
      update((state) => ({
        ...state,
        creating: {
          name,
          image,
          progress: 0,
          message: "Starting...",
          stage: "initializing",
        },
      }));

      let sseWorking = false;
      let completed = false;

      // Fallback to regular endpoint if SSE doesn't send data within 5 seconds
      const fallbackTimeout = setTimeout(() => {
        if (!sseWorking && !completed) {
          console.log(
            "[SSE] No data received, falling back to regular endpoint",
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
      }, 5000);

      fetch("/api/containers/stream", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}`,
        },
        body: JSON.stringify(body),
      })
        .then(async (response) => {
          if (!response.ok) {
            clearTimeout(fallbackTimeout);
            // Try to get error message from response
            const text = await response.text();
            console.error(
              "[SSE] Response not OK:",
              response.status,
              text.slice(0, 200),
            );
            update((state) => ({ ...state, creating: null }));

            // Try to parse as JSON error
            try {
              const errData = JSON.parse(text);
              onError?.(errData.error || `Server error: ${response.status}`);
            } catch {
              onError?.(`Server error: ${response.status}`);
            }
            return;
          }

          // Check content type
          const contentType = response.headers.get("content-type");
          console.log("[SSE] Response content-type:", contentType);

          if (!contentType?.includes("text/event-stream")) {
            console.warn(
              "[SSE] Unexpected content-type, may not be SSE:",
              contentType,
            );
          }

          const reader = response.body?.getReader();
          const decoder = new TextDecoder();
          let buffer = ""; // Buffer to handle chunked SSE data

          function processLine(line: string) {
            // Mark SSE as working when we receive any data
            if (!sseWorking) {
              sseWorking = true;
              clearTimeout(fallbackTimeout);
              console.log("[SSE] Stream is working");
            }

            if (!line.startsWith("data: ")) {
              // Log non-data lines for debugging (could be comments or errors)
              if (line.trim() && !line.startsWith(":")) {
                console.debug("[SSE] Non-data line:", line.slice(0, 100));
              }
              return;
            }

            const jsonStr = line.slice(6).trim();

            // Skip empty data
            if (!jsonStr) {
              console.debug("[SSE] Empty data line");
              return;
            }

            // Check if it looks like JSON
            if (!jsonStr.startsWith("{")) {
              console.warn("[SSE] Data is not JSON:", jsonStr.slice(0, 200));
              // Could be HTML error page or other content
              if (jsonStr.includes("<!DOCTYPE") || jsonStr.includes("<html")) {
                console.error(
                  "[SSE] Received HTML instead of JSON - likely an error page",
                );
                if (!completed) {
                  completed = true;
                  update((state) => ({ ...state, creating: null }));
                  onError?.(
                    "Server returned an error page instead of stream data",
                  );
                }
              }
              return;
            }

            console.debug("[SSE] Processing event:", jsonStr.slice(0, 100));

            try {
              const event = JSON.parse(jsonStr) as ProgressEvent;
              console.debug("[SSE] Parsed event:", {
                stage: event.stage,
                progress: event.progress,
                complete: event.complete,
                error: event.error,
              });

              // Update creating state with progress
              update((state) => ({
                ...state,
                creating: state.creating
                  ? {
                      ...state.creating,
                      progress: event.progress || 0,
                      message: event.message || "",
                      stage: event.stage || "",
                    }
                  : null,
              }));

              onProgress?.(event);

              if (event.complete && event.container_id && !completed) {
                console.log(
                  "[SSE] Container creation complete:",
                  event.container_id,
                );
                completed = true;
                const container: Container = {
                  id: event.container_id,
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
              }

              if (event.error && !completed) {
                console.error("[SSE] Event error:", event.error);
                completed = true;
                update((state) => ({ ...state, creating: null }));
                onError?.(event.error);
              }
            } catch (e) {
              console.warn(
                "[SSE] Failed to parse event:",
                jsonStr.slice(0, 200),
                e,
              );
              // If we get repeated parse errors, something is wrong
              // Don't fail immediately, let the fallback timeout handle it
            }
          }

          function processBuffer() {
            // Process complete lines from buffer
            const lines = buffer.split("\n");

            // Keep incomplete last line in buffer
            buffer = lines.pop() || "";

            for (const line of lines) {
              const trimmed = line.trim();
              if (trimmed) {
                processLine(trimmed);
              }
            }
          }

          function read() {
            reader
              ?.read()
              .then(({ done, value }) => {
                if (done) {
                  console.debug(
                    "[SSE] Stream ended, completed:",
                    completed,
                    "remaining buffer:",
                    buffer.slice(0, 50),
                  );
                  // Process any remaining buffered data
                  if (buffer.trim()) {
                    processLine(buffer.trim());
                  }
                  // Stream ended - if not completed yet and no error, it might be a connection issue
                  // But don't trigger error if we successfully completed
                  if (!completed) {
                    console.warn("[SSE] Stream ended without completion event");
                  }
                  return;
                }

                const text = decoder.decode(value, { stream: true });
                buffer += text;
                processBuffer();

                read();
              })
              .catch((e) => {
                console.error("[SSE] Read error:", e, "completed:", completed);
                // Only report error if we haven't successfully completed
                if (!completed) {
                  completed = true;
                  update((state) => ({ ...state, creating: null }));
                  onError?.(e instanceof Error ? e.message : "Stream error");
                }
              });
          }

          read();
        })
        .catch((e) => {
          console.error("[SSE] Fetch error:", e);
          clearTimeout(fallbackTimeout);
          // If SSE never worked, try fallback
          if (!sseWorking && !completed) {
            console.log("[SSE] Fetch failed, trying fallback endpoint");
            this.createContainerFallback(
              name,
              image,
              customImage,
              onProgress,
              onComplete,
              onError,
            );
          } else if (!completed) {
            update((state) => {
              if (state.creating) {
                onError?.("Failed to create container");
                return { ...state, creating: null };
              }
              return state;
            });
          }
        });
    },

    // Fallback container creation without SSE (for platforms that don't support streaming)
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

      // Update progress to show we're using fallback
      update((state) => ({
        ...state,
        creating: state.creating
          ? {
              ...state.creating,
              progress: 10,
              message: "Creating container...",
              stage: "creating",
            }
          : null,
      }));

      onProgress?.({
        stage: "creating",
        message: "Creating container (this may take a moment)...",
        progress: 10,
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

        const container: Container = {
          id: data.id,
          db_id: data.db_id,
          user_id: data.user_id,
          name: data.name,
          image: data.image,
          status: data.status || "running",
          created_at: data.created_at,
          ip_address: data.ip_address,
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
          c.id === id ? { ...c, status: "creating" as const } : c,
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
          c.id === id ? { ...c, status: "creating" as const } : c,
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
