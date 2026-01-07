import { writable, get } from "svelte/store";
import { auth } from "./auth";
import { createRexecWebSocket } from "$utils/ws";
import { trackEvent } from "$lib/analytics";

// Types
export interface CollabSession {
  id: string;
  shareCode: string;
  containerId: string;
  containerName: string;
  mode: "view" | "control";
  role: "owner" | "editor" | "viewer";
  expiresAt: string;
  participants: CollabParticipant[];
}

export interface CollabParticipant {
  userId: string;
  username: string;
  role: string;
  color: string;
  cursor?: { x: number; y: number };
}

export interface CollabMessage {
  type:
    | "join"
    | "leave"
    | "cursor"
    | "selection"
    | "input"
    | "output"
    | "sync"
    | "participants"
    | "ended"
    | "expired";
  userId?: string;
  username?: string;
  role?: string;
  color?: string;
  data?: any;
  timestamp: number;
}

// Store state
interface CollabState {
  activeSession: CollabSession | null;
  participants: CollabParticipant[];
  isConnected: boolean;
  isConnecting: boolean;
  error: string | null;
}

function createCollabStore() {
  const { subscribe, set, update } = writable<CollabState>({
    activeSession: null,
    participants: [],
    isConnected: false,
    isConnecting: false,
    error: null,
  });

  let ws: WebSocket | null = null;
  let messageHandlers: ((msg: CollabMessage) => void)[] = [];
  let reconnectAttempts = 0;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let currentShareCode: string | null = null;

  const MAX_RECONNECT_ATTEMPTS = 10;
  const RECONNECT_BASE_DELAY = 1000;
  const RECONNECT_MAX_DELAY = 30000;

  const API_BASE = "";

  function getReconnectDelay(): number {
    return Math.min(
      RECONNECT_BASE_DELAY * Math.pow(1.5, reconnectAttempts),
      RECONNECT_MAX_DELAY,
    );
  }

  async function startSession(
    containerId: string,
    mode: "view" | "control" = "view",
    maxUsers: number = 5,
  ): Promise<CollabSession | null> {
    const token = get(auth).token;
    if (!token) return null;

    try {
      const res = await fetch(`${API_BASE}/api/collab/start`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          container_id: containerId,
          mode,
          max_users: maxUsers,
        }),
      });

      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.error || "Failed to start session");
      }

      const data = await res.json();
      const session: CollabSession = {
        id: data.session_id,
        shareCode: data.share_code,
        containerId,
        containerName: data.container_name || containerId.slice(0, 12),
        mode,
        role: "owner",
        expiresAt: data.expires_at,
        participants: [],
      };

      update((s) => ({ ...s, activeSession: session }));

      // Track collab session started
      trackEvent("collab_session_started", {
        mode: mode,
        maxUsers: maxUsers,
        containerId: containerId,
        shareCode: data.share_code,
      });

      return session;
    } catch (err: any) {
      update((s) => ({ ...s, error: err.message }));
      return null;
    }
  }

  async function joinSession(shareCode: string): Promise<CollabSession | null> {
    const token = get(auth).token;
    if (!token) return null;

    update((s) => ({ ...s, isConnecting: true, error: null }));

    try {
      const res = await fetch(`${API_BASE}/api/collab/join/${shareCode}`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (!res.ok) {
        const error = await res.json();
        throw new Error(error.error || "Failed to join session");
      }

      const data = await res.json();
      const session: CollabSession = {
        id: data.session_id,
        shareCode,
        containerId: data.container_id,
        containerName: data.container_name || data.container_id.slice(0, 12),
        mode: data.mode,
        role: data.role,
        expiresAt: data.expires_at,
        participants: [],
      };

      update((s) => ({ ...s, activeSession: session, isConnecting: false }));
      return session;
    } catch (err: any) {
      update((s) => ({ ...s, error: err.message, isConnecting: false }));
      return null;
    }
  }

  function connectWebSocket(shareCode: string) {
    const token = get(auth).token;
    if (!token) return;

    // Clear any pending reconnect
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }

    // Store share code for reconnection
    currentShareCode = shareCode;

    update((s) => ({ ...s, isConnecting: true }));

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/ws/collab/${shareCode}`;

    ws = createRexecWebSocket(wsUrl, token);

    ws.onopen = () => {
      reconnectAttempts = 0;
      update((s) => ({
        ...s,
        isConnected: true,
        isConnecting: false,
        error: null,
      }));
    };

    ws.onmessage = (event) => {
      try {
        const msg: CollabMessage = JSON.parse(event.data);
        handleMessage(msg);
      } catch (err) {
        console.error("[Collab] Failed to parse message:", err);
      }
    };

    ws.onclose = (event) => {
      ws = null;
      update((s) => ({ ...s, isConnected: false }));

      // Don't reconnect if session ended or intentionally closed
      const isIntentionalClose = event.code === 1000;
      const isSessionEnded = event.code === 4000 || event.code === 4001;

      if (isIntentionalClose || isSessionEnded) {
        reconnectAttempts = 0;
        currentShareCode = null;
        return;
      }

      // Attempt silent reconnect
      if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS && currentShareCode) {
        reconnectAttempts++;
        const delay = getReconnectDelay();

        reconnectTimer = setTimeout(() => {
          if (currentShareCode) {
            connectWebSocket(currentShareCode);
          }
        }, delay);
      } else {
        update((s) => ({
          ...s,
          error: "Connection lost. Please rejoin the session.",
        }));
      }
    };

    ws.onerror = () => {
      // Errors are handled by onclose - just log silently

      update((s) => ({ ...s, isConnecting: false }));
    };
  }

  function handleMessage(msg: CollabMessage) {
    switch (msg.type) {
      case "join":
        update((s) => ({
          ...s,
          participants: [
            ...s.participants,
            {
              userId: msg.userId!,
              username: msg.username!,
              role: msg.role!,
              color: msg.color!,
            },
          ],
        }));
        break;

      case "leave":
        update((s) => ({
          ...s,
          participants: s.participants.filter((p) => p.userId !== msg.userId),
        }));
        break;

      case "participants":
        update((s) => ({
          ...s,
          participants: msg.data as CollabParticipant[],
        }));
        break;

      case "cursor":
        update((s) => ({
          ...s,
          participants: s.participants.map((p) =>
            p.userId === msg.userId ? { ...p, cursor: msg.data } : p,
          ),
        }));
        break;

      case "ended":
      case "expired":
        disconnect();
        update((s) => ({
          ...s,
          activeSession: null,
          error:
            msg.type === "expired"
              ? "Session expired"
              : "Session ended by owner",
        }));
        break;
    }

    // Notify all handlers
    messageHandlers.forEach((handler) => handler(msg));
  }

  function sendMessage(type: string, data?: any) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type, data, timestamp: Date.now() }));
    }
  }

  function sendCursorPosition(x: number, y: number) {
    sendMessage("cursor", { x, y });
  }

  function sendInput(input: string) {
    // Check if user can send input (owner or editor only)
    const state = get({ subscribe });
    if (state.activeSession?.role === "viewer") {
      return;
    }
    sendMessage("input", input);
  }

  function canSendInput(): boolean {
    const state = get({ subscribe });
    return state.activeSession?.role !== "viewer";
  }

  function onMessage(handler: (msg: CollabMessage) => void) {
    messageHandlers.push(handler);
    return () => {
      messageHandlers = messageHandlers.filter((h) => h !== handler);
    };
  }

  async function endSession(sessionId: string): Promise<boolean> {
    const token = get(auth).token;
    if (!token) return false;

    try {
      const res = await fetch(`${API_BASE}/api/collab/sessions/${sessionId}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        disconnect();
        update((s) => ({ ...s, activeSession: null }));
        return true;
      }
    } catch (err) {
      console.error("[Collab] Failed to end session:", err);
    }
    return false;
  }

  async function getActiveSessions(): Promise<any[]> {
    const token = get(auth).token;
    if (!token) return [];

    try {
      const res = await fetch(`${API_BASE}/api/collab/sessions`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (res.ok) {
        const data = await res.json();
        return data.sessions || [];
      }
    } catch (err) {
      console.error("[Collab] Failed to get sessions:", err);
    }
    return [];
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    currentShareCode = null;
    reconnectAttempts = 0;
    if (ws) {
      ws.close(1000, "User disconnected");
      ws = null;
    }
    update((s) => ({
      ...s,
      isConnected: false,
      participants: [],
    }));
  }

  function reset() {
    disconnect();
    set({
      activeSession: null,
      participants: [],
      isConnected: false,
      isConnecting: false,
      error: null,
    });
  }

  return {
    subscribe,
    startSession,
    joinSession,
    connectWebSocket,
    sendCursorPosition,
    sendInput,
    canSendInput,
    onMessage,
    endSession,
    getActiveSessions,
    disconnect,
    reset,
  };
}

export const collab = createCollabStore();
