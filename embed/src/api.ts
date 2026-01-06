/**
 * Rexec Embed Widget API Client
 * HTTP and WebSocket communication with the Rexec API
 */

import type {
  CreateContainerResponse,
  JoinSessionResponse,
  ContainerInfoResponse,
  WsMessage,
} from "./types";

const WS_PROTOCOL_VERSION = "rexec.v1";
const WS_TOKEN_PROTOCOL_PREFIX = "rexec.token.";

/**
 * API Client for Rexec HTTP endpoints
 */
export class RexecApiClient {
  private baseUrl: string;
  private token: string | null;

  constructor(baseUrl: string = "https://rexec.dev", token?: string) {
    // Remove trailing slash
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.token = token || null;
  }

  /**
   * Set the authentication token
   */
  setToken(token: string): void {
    this.token = token;
  }

  /**
   * Get default headers for API requests
   */
  private getHeaders(): Headers {
    const headers = new Headers({
      "Content-Type": "application/json",
    });

    if (this.token) {
      headers.set("Authorization", `Bearer ${this.token}`);
    }

    return headers;
  }

  /**
   * Make an API request
   */
  private async request<T>(
    endpoint: string,
    options: RequestInit = {},
  ): Promise<{ data?: T; error?: string }> {
    const url = `${this.baseUrl}${endpoint}`;

    try {
      const response = await fetch(url, {
        ...options,
        headers: this.getHeaders(),
      });

      const contentType = response.headers.get("content-type");
      let data: T | undefined;
      let error: string | undefined;

      if (contentType?.includes("application/json")) {
        const json = await response.json();
        if (response.ok) {
          data = json as T;
        } else {
          error =
            json.error || json.message || `Request failed: ${response.status}`;
        }
      } else if (!response.ok) {
        error = `Request failed: ${response.status}`;
      }

      return { data, error };
    } catch (e) {
      return {
        error: e instanceof Error ? e.message : "Network error",
      };
    }
  }

  /**
   * Create a new container with the specified image and optional role
   */
  async createContainer(
    image: string = "ubuntu",
    role?: string,
  ): Promise<{ data?: CreateContainerResponse; error?: string }> {
    const body: { image: string; role?: string } = { image };
    if (role) {
      body.role = role;
    }
    return this.request<CreateContainerResponse>("/api/containers", {
      method: "POST",
      body: JSON.stringify(body),
    });
  }

  /**
   * Get container information
   */
  async getContainer(
    containerId: string,
  ): Promise<{ data?: ContainerInfoResponse; error?: string }> {
    return this.request<ContainerInfoResponse>(
      `/api/containers/${encodeURIComponent(containerId)}`,
    );
  }

  /**
   * Wait for a container to be ready (running status)
   * Polls the container status until it's running or an error occurs
   */
  async waitForContainer(
    containerId: string,
    options: {
      maxAttempts?: number;
      intervalMs?: number;
      onProgress?: (status: string, attempt: number) => void;
    } = {},
  ): Promise<{ data?: ContainerInfoResponse; error?: string }> {
    const maxAttempts = options.maxAttempts ?? 60; // 60 attempts = ~2 minutes
    const intervalMs = options.intervalMs ?? 2000; // Poll every 2 seconds

    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      const { data, error } = await this.getContainer(containerId);

      if (error) {
        // If container not found yet, keep waiting
        if (error.includes("404") || error.includes("not found")) {
          options.onProgress?.("creating", attempt);
          await this.sleep(intervalMs);
          continue;
        }
        return { error };
      }

      if (data) {
        const status = data.status?.toLowerCase() || "";
        options.onProgress?.(status, attempt);

        if (status === "running") {
          return { data };
        }

        if (status === "error" || status === "failed") {
          return { error: `Container failed to start: ${status}` };
        }

        // Container is still being created/configured
        if (
          status === "creating" ||
          status === "configuring" ||
          status === "starting" ||
          status === "pulling"
        ) {
          await this.sleep(intervalMs);
          continue;
        }

        // Unknown status, try a few more times
        if (attempt < maxAttempts) {
          await this.sleep(intervalMs);
          continue;
        }
      }

      await this.sleep(intervalMs);
    }

    return { error: "Timeout waiting for container to be ready" };
  }

  /**
   * Sleep for a given number of milliseconds
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * Join a collaborative session via share code
   */
  async joinSession(
    shareCode: string,
  ): Promise<{ data?: JoinSessionResponse; error?: string }> {
    return this.request<JoinSessionResponse>(
      `/api/collab/join/${encodeURIComponent(shareCode)}`,
    );
  }

  /**
   * Start a new collaborative session for a container
   */
  async startCollabSession(
    containerId: string,
    mode: "view" | "control" = "view",
  ): Promise<{
    data?: { session_id: string; share_code: string; expires_at: string };
    error?: string;
  }> {
    return this.request("/api/collab/start", {
      method: "POST",
      body: JSON.stringify({ container_id: containerId, mode }),
    });
  }

  /**
   * Get WebSocket URL for terminal connection
   */
  getTerminalWsUrl(containerId: string, sessionId: string): string {
    const protocol = this.baseUrl.startsWith("https") ? "wss:" : "ws:";
    const host = this.baseUrl.replace(/^https?:\/\//, "");
    return `${protocol}//${host}/ws/terminal/${encodeURIComponent(containerId)}?id=${encodeURIComponent(sessionId)}`;
  }

  /**
   * Get WebSocket URL for agent terminal connection
   */
  getAgentTerminalWsUrl(agentId: string, sessionId: string): string {
    const protocol = this.baseUrl.startsWith("https") ? "wss:" : "ws:";
    const host = this.baseUrl.replace(/^https?:\/\//, "");
    return `${protocol}//${host}/ws/agent/${encodeURIComponent(agentId)}/terminal?id=${encodeURIComponent(sessionId)}`;
  }

  /**
   * Get WebSocket URL for collab session
   */
  getCollabWsUrl(shareCode: string): string {
    const protocol = this.baseUrl.startsWith("https") ? "wss:" : "ws:";
    const host = this.baseUrl.replace(/^https?:\/\//, "");
    return `${protocol}//${host}/ws/collab/${encodeURIComponent(shareCode)}`;
  }
}

/**
 * WebSocket protocols for Rexec authentication
 */
export function getRexecWebSocketProtocols(
  token: string | null,
): string[] | undefined {
  if (!token) return undefined;
  return [WS_PROTOCOL_VERSION, `${WS_TOKEN_PROTOCOL_PREFIX}${token}`];
}

/**
 * Create a WebSocket with Rexec authentication
 */
export function createRexecWebSocket(
  url: string,
  token: string | null,
): WebSocket {
  const protocols = getRexecWebSocketProtocols(token);
  return protocols ? new WebSocket(url, protocols) : new WebSocket(url);
}

/**
 * Terminal WebSocket connection manager
 */
export class TerminalWebSocket {
  private ws: WebSocket | null = null;
  private url: string;
  private token: string | null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts: number;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingInterval: ReturnType<typeof setInterval> | null = null;
  private autoReconnect: boolean;

  // Callbacks
  public onOpen: (() => void) | null = null;
  public onClose: ((code: number, reason: string) => void) | null = null;
  public onError: ((error: Event) => void) | null = null;
  public onMessage: ((message: WsMessage) => void) | null = null;
  public onReconnecting: ((attempt: number) => void) | null = null;

  constructor(
    url: string,
    token: string | null,
    options: {
      autoReconnect?: boolean;
      maxReconnectAttempts?: number;
    } = {},
  ) {
    this.url = url;
    this.token = token;
    this.autoReconnect = options.autoReconnect ?? true;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
  }

  /**
   * Connect to the WebSocket
   */
  connect(): void {
    if (
      this.ws &&
      (this.ws.readyState === WebSocket.OPEN ||
        this.ws.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    this.clearTimers();
    this.ws = createRexecWebSocket(this.url, this.token);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.startPingInterval();
      this.onOpen?.();
    };

    this.ws.onclose = (event) => {
      this.clearTimers();
      this.onClose?.(event.code, event.reason);

      // Attempt reconnect if not intentional close
      const isIntentionalClose = event.code === 1000;
      const isSessionEnded = event.code === 4000 || event.code === 4001;

      if (this.autoReconnect && !isIntentionalClose && !isSessionEnded) {
        this.attemptReconnect();
      }
    };

    this.ws.onerror = (error) => {
      this.onError?.(error);
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WsMessage = JSON.parse(event.data);
        // Ignore ping/pong messages - they're just keepalives
        if (message.type === "ping" || message.type === "pong") {
          return;
        }
        this.onMessage?.(message);
      } catch {
        // Only treat as output if it looks like actual terminal data
        // Ignore empty messages or system messages
        const data = event.data;
        if (
          typeof data === "string" &&
          data.length > 0 &&
          !data.startsWith("{") &&
          !data.startsWith("[")
        ) {
          this.onMessage?.({ type: "output", data });
        }
        // Otherwise silently ignore malformed JSON
      }
    };
  }

  /**
   * Send a message through the WebSocket
   */
  send(message: WsMessage): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  /**
   * Send raw data (for terminal input)
   */
  sendRaw(data: string): void {
    this.send({ type: "input", data });
  }

  /**
   * Send resize message
   */
  sendResize(cols: number, rows: number): void {
    this.send({ type: "resize", cols, rows });
  }

  /**
   * Send ping message
   */
  sendPing(): void {
    this.send({ type: "ping" });
  }

  /**
   * Close the WebSocket connection
   */
  close(code = 1000, reason = "User disconnected"): void {
    this.autoReconnect = false; // Prevent reconnect on intentional close
    this.clearTimers();
    if (this.ws) {
      this.ws.close(code, reason);
      this.ws = null;
    }
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  /**
   * Get current ready state
   */
  getReadyState(): number {
    return this.ws?.readyState ?? WebSocket.CLOSED;
  }

  /**
   * Clear all timers
   */
  private clearTimers(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  /**
   * Start ping interval to keep connection alive
   */
  private startPingInterval(): void {
    this.pingInterval = setInterval(() => {
      this.sendPing();
    }, 20000); // 20 seconds
  }

  /**
   * Attempt to reconnect
   */
  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      return;
    }

    this.reconnectAttempts++;
    const delay = Math.min(100 * Math.pow(2, this.reconnectAttempts), 8000);

    this.onReconnecting?.(this.reconnectAttempts);

    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }

  /**
   * Reset reconnect attempts counter
   */
  resetReconnectAttempts(): void {
    this.reconnectAttempts = 0;
  }

  /**
   * Update the URL (useful for reconnecting to a different session)
   */
  updateUrl(url: string): void {
    this.url = url;
  }
}

/**
 * Generate a unique session ID
 */
export function generateSessionId(): string {
  return `embed-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;
}
