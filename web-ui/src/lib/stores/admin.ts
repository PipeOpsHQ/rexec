import { writable, derived, get } from "svelte/store";
import { api, type ApiResponse } from "../utils/api";
import type { User } from "./auth";
import type { Container } from "./containers";
import { PUBLIC_API_URL } from "$env/static/public";
import { browser } from "$app/environment";

// Types
export interface AdminUser extends User {
  createdAt: string;
  lastLogin?: string;
  containerCount: number;
}

export interface AdminContainer extends Container {
  username: string; // Owner's username
  userEmail: string; // Owner's email
}

export interface AdminTerminal {
  id: string;
  containerId: string;
  name: string;
  status: "connected" | "disconnected" | "error";
  userId: string;
  username: string;
  connectedAt: string;
}

// Backend AdminEvent interface
export interface AdminEvent<T = any> {
  type:
    | "user_created"
    | "user_updated"
    | "user_deleted"
    | "container_created"
    | "container_updated"
    | "container_deleted"
    | "session_created"
    | "session_updated"
    | "session_deleted";
  payload: T;
  timestamp: string; // ISO 8601 string
}

export interface AdminState {
  users: AdminUser[];
  containers: AdminContainer[];
  terminals: AdminTerminal[];
  isLoading: boolean;
  error: string | null;
  ws: WebSocket | null;
  wsConnected: boolean;
  wsReconnectAttempts: number;
  wsMaxReconnectAttempts: number;
  wsReconnectInterval: number; // in milliseconds
}

const initialState: AdminState = {
  users: [],
  containers: [],
  terminals: [],
  isLoading: false,
  error: null,
  ws: null,
  wsConnected: false,
  wsReconnectAttempts: 0,
  wsMaxReconnectAttempts: 10,
  wsReconnectInterval: 1000, // 1 second
};

function createAdminStore() {
  const { subscribe, set, update } = writable<AdminState>(initialState);

  let ws: WebSocket | null = null;
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

  async function connectWebSocket() {
    update((state) => ({
      ...state,
      wsConnected: false,
      error: null,
    }));

    if (!browser) return;

    const token = localStorage.getItem("rexec_token"); // Assuming token is stored here
    if (!token) {
      console.warn("Admin WebSocket: No authentication token found.");
      update((state) => ({ ...state, error: "Authentication token missing." }));
      return;
    }

    const wsProtocol = PUBLIC_API_URL.startsWith("https") ? "wss" : "ws";
    const wsUrl = `${wsProtocol}://${
      new URL(PUBLIC_API_URL).host
    }/ws/admin/events?token=${token}`;

    ws = new WebSocket(wsUrl);
    update((state) => ({ ...state, ws }));

    ws.onopen = () => {
      console.log("Admin WebSocket: Connected");
      update((state) => ({
        ...state,
        wsConnected: true,
        wsReconnectAttempts: 0,
        wsReconnectInterval: initialState.wsReconnectInterval,
      }));
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
        reconnectTimeout = null;
      }
    };

    ws.onmessage = (event) => {
      const adminEvent: AdminEvent = JSON.parse(event.data);
      update((state) => {
        let newUsers = [...state.users];
        let newContainers = [...state.containers];
        let newTerminals = [...state.terminals];

        switch (adminEvent.type) {
          case "user_created":
            newUsers = [...newUsers, adminEvent.payload as AdminUser];
            break;
          case "user_updated":
            newUsers = newUsers.map((u) =>
              u.id === (adminEvent.payload as AdminUser).id
                ? (adminEvent.payload as AdminUser)
                : u
            );
            break;
          case "user_deleted":
            newUsers = newUsers.filter(
              (u) => u.id !== (adminEvent.payload as AdminUser).id
            );
            break;
          case "container_created":
            newContainers = [
              ...newContainers,
              adminEvent.payload as AdminContainer,
            ];
            break;
          case "container_updated":
            newContainers = newContainers.map((c) =>
              c.id === (adminEvent.payload as AdminContainer).id
                ? (adminEvent.payload as AdminContainer)
                : c
            );
            break;
          case "container_deleted":
            newContainers = newContainers.filter(
              (c) => c.id !== (adminEvent.payload as AdminContainer).id
            );
            break;
          case "session_created":
            newTerminals = [
              ...newTerminals,
              adminEvent.payload as AdminTerminal,
            ];
            break;
          case "session_updated":
            newTerminals = newTerminals.map((t) =>
              t.id === (adminEvent.payload as AdminTerminal).id
                ? (adminEvent.payload as AdminTerminal)
                : t
            );
            break;
          case "session_deleted":
            newTerminals = newTerminals.filter(
              (t) => t.id !== (adminEvent.payload as AdminTerminal).id
            );
            break;
          default:
            console.warn("Admin WebSocket: Unknown event type", adminEvent.type);
        }

        return { ...state, users: newUsers, containers: newContainers, terminals: newTerminals };
      });
    };

    ws.onclose = (event) => {
      console.log(
        `Admin WebSocket: Disconnected (Code: ${event.code}, Reason: ${event.reason})`
      );
      update((state) => ({ ...state, wsConnected: false }));

      if (event.code !== 1000 && event.code !== 1001) {
        // Don't try to reconnect on normal closures (1000: Normal, 1001: Going Away)
        reconnect();
      }
    };

    ws.onerror = (error) => {
      console.error("Admin WebSocket: Error", error);
      update((state) => ({ ...state, error: "WebSocket error" }));
      ws?.close(); // Close to trigger onclose and reconnect logic
    };
  }

  function reconnect() {
    update((state) => {
      if (state.wsReconnectAttempts < state.wsMaxReconnectAttempts) {
        const newReconnectAttempts = state.wsReconnectAttempts + 1;
        const newReconnectInterval = state.wsReconnectInterval * 2; // Exponential backoff
        reconnectTimeout = setTimeout(
          connectWebSocket,
          newReconnectInterval
        ) as ReturnType<typeof setTimeout>;
        console.log(
          `Admin WebSocket: Reconnecting in ${
            newReconnectInterval / 1000
          }s (Attempt ${newReconnectAttempts})`
        );
        return {
          ...state,
          wsReconnectAttempts: newReconnectAttempts,
          wsReconnectInterval: newReconnectInterval,
        };
      } else {
        console.error(
          "Admin WebSocket: Max reconnect attempts reached. Please refresh."
        );
        return { ...state, error: "Max reconnect attempts reached." };
      }
    });
  }

  function disconnectWebSocket() {
    if (ws) {
      ws.close(1000, "Component unmounted"); // Normal closure
      ws = null;
    }
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout);
      reconnectTimeout = null;
    }
    update((state) => ({ ...state, wsConnected: false, ws: null }));
  }

  return {
    subscribe,
    fetchUsers: async () => {
      update((state) => ({ ...state, isLoading: true, error: null }));
      const { data, error } = await api.get<AdminUser[]>("/api/admin/users");

      if (error) {
        update((state) => ({ ...state, isLoading: false, error }));
        return;
      }
      update((state) => ({ ...state, users: data || [], isLoading: false }));
    },

    fetchContainers: async () => {
      update((state) => ({ ...state, isLoading: true, error: null }));
      const { data, error } = await api.get<AdminContainer[]>(
        "/api/admin/containers"
      );

      if (error) {
        update((state) => ({ ...state, isLoading: false, error }));
        return;
      }
      update((state) => ({
        ...state,
        containers: data || [],
        isLoading: false,
      }));
    },

    fetchTerminals: async () => {
      update((state) => ({ ...state, isLoading: true, error: null }));
      const { data, error } = await api.get<AdminTerminal[]>(
        "/api/admin/terminals"
      );

      if (error) {
        update((state) => ({ ...state, isLoading: false, error }));
        return;
      }
      update((state) => ({
        ...state,
        terminals: data || [],
        isLoading: false,
      }));
    },

    deleteUser: async (userId: string) => {
      const { error } = await api.delete(`/api/admin/users/${userId}`);
      if (error) return { success: false, error };
      // WS event will handle updating the store
      return { success: true };
    },

    deleteContainer: async (containerId: string) => {
      const { error } = await api.delete(`/api/admin/containers/${containerId}`);
      if (error) return { success: false, error };
      // WS event will handle updating the store
      return { success: true };
    },

    startAdminEvents: () => {
      connectWebSocket();
    },

    stopAdminEvents: () => {
      disconnectWebSocket();
    },
  };
}

export const admin = createAdminStore();