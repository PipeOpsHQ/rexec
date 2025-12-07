import { writable, derived, get } from "svelte/store";
import { api, type ApiResponse } from "../utils/api";
import type { User } from "./auth";
import type { Container } from "./containers";

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

export interface AdminState {
  users: AdminUser[];
  containers: AdminContainer[];
  terminals: AdminTerminal[];
  isLoading: boolean;
  error: string | null;
}

const initialState: AdminState = {
  users: [],
  containers: [],
  terminals: [],
  isLoading: false,
  error: null,
};

function createAdminStore() {
  const { subscribe, set, update } = writable<AdminState>(initialState);

  return {
    subscribe,

    // Mock data generator for development
    _generateMockData() {
      const mockUsers: AdminUser[] = [
        {
          id: "u1",
          name: "Alice Admin",
          email: "alice@rexec.dev",
          tier: "pro",
          isGuest: false,
          isAdmin: true,
          createdAt: "2023-01-15T10:00:00Z",
          lastLogin: "2023-10-27T14:30:00Z",
          containerCount: 2,
        },
        {
          id: "u2",
          name: "Bob Builder",
          email: "bob@example.com",
          tier: "free",
          isGuest: false,
          isAdmin: false,
          createdAt: "2023-03-20T09:15:00Z",
          lastLogin: "2023-10-26T11:20:00Z",
          containerCount: 1,
        },
        {
          id: "u3",
          name: "Charlie Guest",
          email: "guest_123@rexec.dev",
          tier: "guest",
          isGuest: true,
          isAdmin: false,
          createdAt: "2023-10-27T15:00:00Z",
          lastLogin: "2023-10-27T15:00:00Z",
          containerCount: 0,
        },
      ];

      const mockContainers: AdminContainer[] = [
        {
          id: "c1",
          user_id: "u1",
          name: "alice-dev",
          image: "ubuntu:latest",
          status: "running",
          created_at: "2023-10-27T10:00:00Z",
          username: "Alice Admin",
          userEmail: "alice@rexec.dev",
          resources: { memory_mb: 512, cpu_shares: 1024, disk_mb: 10240 },
        },
        {
          id: "c2",
          user_id: "u1",
          name: "alice-prod",
          image: "node:18",
          status: "stopped",
          created_at: "2023-10-25T14:00:00Z",
          username: "Alice Admin",
          userEmail: "alice@rexec.dev",
          resources: { memory_mb: 1024, cpu_shares: 2048, disk_mb: 20480 },
        },
        {
          id: "c3",
          user_id: "u2",
          name: "bob-sandbox",
          image: "python:3.9",
          status: "running",
          created_at: "2023-10-26T11:30:00Z",
          username: "Bob Builder",
          userEmail: "bob@example.com",
          resources: { memory_mb: 512, cpu_shares: 1024, disk_mb: 5120 },
        },
      ];

      const mockTerminals: AdminTerminal[] = [
        {
          id: "t1",
          containerId: "c1",
          name: "alice-dev (Main)",
          status: "connected",
          userId: "u1",
          username: "Alice Admin",
          connectedAt: "2023-10-27T14:35:00Z",
        },
        {
          id: "t2",
          containerId: "c3",
          name: "bob-sandbox",
          status: "connected",
          userId: "u2",
          username: "Bob Builder",
          connectedAt: "2023-10-27T15:10:00Z",
        },
      ];

      update((state) => ({
        ...state,
        users: mockUsers,
        containers: mockContainers,
        terminals: mockTerminals,
        isLoading: false,
      }));
    },

    async fetchUsers() {
      update((state) => ({ ...state, isLoading: true, error: null }));
      // TODO: Replace with real API call
      // const { data, error } = await api.get<AdminUser[]>("/api/admin/users");
      await new Promise((resolve) => setTimeout(resolve, 500)); // Simulate delay
      this._generateMockData(); // Populate with mock data for now
    },

    async fetchContainers() {
      update((state) => ({ ...state, isLoading: true, error: null }));
      // TODO: Replace with real API call
      // const { data, error } = await api.get<AdminContainer[]>("/api/admin/containers");
      await new Promise((resolve) => setTimeout(resolve, 500));
      this._generateMockData();
    },

    async fetchTerminals() {
      update((state) => ({ ...state, isLoading: true, error: null }));
      // TODO: Replace with real API call
      // const { data, error } = await api.get<AdminTerminal[]>("/api/admin/terminals");
      await new Promise((resolve) => setTimeout(resolve, 500));
      this._generateMockData();
    },

    async deleteUser(userId: string) {
       // TODO: API Call
       update(state => ({
           ...state,
           users: state.users.filter(u => u.id !== userId)
       }));
       return { success: true };
    },
    
    async deleteContainer(containerId: string) {
        // TODO: API Call
        update(state => ({
            ...state,
            containers: state.containers.filter(c => c.id !== containerId)
        }));
        return { success: true };
    }
  };
}

export const admin = createAdminStore();
