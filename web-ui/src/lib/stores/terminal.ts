import { writable, derived, get } from "svelte/store";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { token } from "./auth";

// Types
export type SessionStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "error";
export type ViewMode = "floating" | "docked" | "fullscreen";

export interface TerminalSession {
  id: string;
  containerId: string;
  name: string;
  terminal: Terminal;
  fitAddon: FitAddon;
  ws: WebSocket | null;
  status: SessionStatus;
  reconnectAttempts: number;
  reconnectTimer: ReturnType<typeof setTimeout> | null;
  pingInterval: ReturnType<typeof setInterval> | null;
  resizeObserver: ResizeObserver | null;
  isSettingUp: boolean;
  setupMessage: string;
  // Stats
  stats: {
    cpu: number;
    memory: number;
    memoryLimit: number;
  };
  // Detached window state (when popped out as separate floating window)
  isDetached: boolean;
  detachedPosition: { x: number; y: number };
  detachedSize: { width: number; height: number };
  detachedZIndex: number;
}

export interface TerminalState {
  sessions: Map<string, TerminalSession>;
  activeSessionId: string | null;
  viewMode: ViewMode;
  isMinimized: boolean;
  floatingPosition: { x: number; y: number };
  floatingSize: { width: number; height: number };
  topZIndex: number; // Track highest z-index for detached windows
}

// Constants
const WS_MAX_RECONNECT = 5;
const WS_RECONNECT_DELAY = 2000;
const WS_PING_INTERVAL = 25000;

const REXEC_BANNER =
  "\x1b[38;5;46m\r\n" +
  "  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗\r\n" +
  "  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝\r\n" +
  "  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║\r\n" +
  "  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║\r\n" +
  "  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗\r\n" +
  "  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝\r\n" +
  "\x1b[0m\x1b[38;5;243m  Terminal as a Service · rexec.dev\x1b[0m\r\n\r\n";

// Terminal configuration
const TERMINAL_OPTIONS = {
  cursorBlink: true,
  cursorStyle: "bar" as const,
  fontSize: 14,
  fontFamily: '"JetBrains Mono", "Fira Code", Menlo, Monaco, monospace',
  theme: {
    background: "#0a0a0a",
    foreground: "#e0e0e0",
    cursor: "#00ff41",
    cursorAccent: "#0a0a0a",
    selectionBackground: "rgba(0, 255, 65, 0.3)",
    black: "#0a0a0a",
    red: "#ff003c",
    green: "#00ff41",
    yellow: "#fcee0a",
    blue: "#00a0dc",
    magenta: "#ff00ff",
    cyan: "#00ffff",
    white: "#e0e0e0",
    brightBlack: "#666666",
    brightRed: "#ff5555",
    brightGreen: "#55ff55",
    brightYellow: "#ffff55",
    brightBlue: "#5555ff",
    brightMagenta: "#ff55ff",
    brightCyan: "#55ffff",
    brightWhite: "#ffffff",
  },
  allowProposedApi: true,
  scrollback: 5000,
};

// Initial state
const initialState: TerminalState = {
  sessions: new Map(),
  activeSessionId: null,
  viewMode: "floating",
  isMinimized: false,
  floatingPosition: { x: 100, y: 100 },
  floatingSize: { width: 700, height: 500 },
  topZIndex: 1000,
};

// Load persisted preferences
function loadPreferences(): Partial<TerminalState> {
  if (typeof window === "undefined") return {};

  try {
    const saved = localStorage.getItem("rexec_terminal_prefs");
    if (saved) {
      const prefs = JSON.parse(saved);
      return {
        viewMode: prefs.viewMode || "floating",
        floatingPosition:
          prefs.floatingPosition || initialState.floatingPosition,
        floatingSize: prefs.floatingSize || initialState.floatingSize,
      };
    }
  } catch (e) {
    console.error("Failed to load terminal preferences:", e);
  }

  return {};
}

// Save preferences
function savePreferences(state: TerminalState) {
  try {
    localStorage.setItem(
      "rexec_terminal_prefs",
      JSON.stringify({
        viewMode: state.viewMode,
        floatingPosition: state.floatingPosition,
        floatingSize: state.floatingSize,
      }),
    );
  } catch (e) {
    console.error("Failed to save terminal preferences:", e);
  }
}

// Generate session ID
function generateSessionId(): string {
  return `session-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

// Get WebSocket URL
function getWsUrl(): string {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}`;
}

// Create the store
function createTerminalStore() {
  const state: TerminalState = { ...initialState, ...loadPreferences() };
  const { subscribe, update } = writable<TerminalState>(state);

  // Helper to get current state
  const getState = (): TerminalState => {
    let currentState: TerminalState;
    subscribe((s) => (currentState = s))();
    return currentState!;
  };

  // Helper to update a session
  const updateSession = (
    sessionId: string,
    updater: (session: TerminalSession) => TerminalSession,
  ) => {
    update((state) => {
      const session = state.sessions.get(sessionId);
      if (session) {
        // Create a new Map to ensure Svelte reactivity
        const newSessions = new Map(state.sessions);
        newSessions.set(sessionId, updater(session));
        return { ...state, sessions: newSessions };
      }
      return state;
    });
  };

  return {
    subscribe,

    // Get current state
    getState,

    // Create a new terminal session (reuses existing session for same container)
    createSession(containerId: string, name: string): string | null {
      const currentState = getState();

      // Validate inputs
      if (!containerId) {
        console.error("createSession: containerId is required");
        return null;
      }

      // Ensure name is a valid string
      const sessionName =
        name && typeof name === "string"
          ? name
          : `Terminal-${containerId.slice(0, 8)}`;

      // Check for guest limit (max 3 sessions)
      const authToken = get(token);
      if (!authToken) return null;

      // Check if we already have a session for this container
      const existingSession = Array.from(currentState.sessions.values()).find(
        (s) => s.containerId === containerId,
      );
      if (existingSession) {
        // Just activate the existing session
        update((state) => ({ ...state, activeSessionId: existingSession.id }));
        // Update URL to reflect active terminal
        this.updateUrl(containerId);
        return existingSession.id;
      }

      // Create a new tab
      return this.createNewTab(containerId, sessionName);
    },

    // Create a new tab (always creates a new session, even for same container)
    createNewTab(containerId: string, name: string): string | null {
      const authToken = get(token);
      if (!authToken) return null;

      // Validate inputs
      if (!containerId) {
        console.error("createNewTab: containerId is required");
        return null;
      }

      // Ensure name is a valid string
      const validName =
        name && typeof name === "string"
          ? name
          : `Terminal-${containerId.slice(0, 8)}`;

      // Count existing sessions for this container to number the tab
      const currentState = getState();
      const existingCount = Array.from(currentState.sessions.values()).filter(
        (s) => s.containerId === containerId,
      ).length;

      // Create terminal instance
      const terminal = new Terminal(TERMINAL_OPTIONS);
      const fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.loadAddon(new WebLinksAddon());

      const sessionId = generateSessionId();
      const tabName =
        existingCount > 0 ? `${validName} (${existingCount + 1})` : validName;
      const session: TerminalSession = {
        id: sessionId,
        containerId,
        name: tabName,
        terminal,
        fitAddon,
        ws: null,
        status: "connecting",
        reconnectAttempts: 0,
        reconnectTimer: null,
        pingInterval: null,
        resizeObserver: null,
        isSettingUp: false,
        setupMessage: "",
        stats: {
          cpu: 0,
          memory: 0,
          memoryLimit: 0,
        },
        isDetached: false,
        detachedPosition: { x: 150, y: 150 },
        detachedSize: { width: 600, height: 400 },
        detachedZIndex: 1000,
      };

      update((state) => {
        // Create a new Map to ensure Svelte reactivity
        const newSessions = new Map(state.sessions);
        newSessions.set(sessionId, session);
        return {
          ...state,
          sessions: newSessions,
          activeSessionId: sessionId,
          isMinimized: false,
        };
      });

      // Update URL to reflect active terminal
      this.updateUrl(containerId);

      return sessionId;
    },

    // Update browser URL for terminal routing
    updateUrl(containerId: string) {
      const newUrl = `/${containerId}`;
      if (window.location.pathname !== newUrl) {
        window.history.pushState({ containerId }, "", newUrl);
      }
    },

    // Clear URL when no terminals are active
    clearUrl() {
      if (window.location.pathname !== "/") {
        window.history.pushState({}, "", "/");
      }
    },

    // Connect WebSocket for a session
    connectWebSocket(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return;

      const authToken = get(token);
      if (!authToken) return;

      // Prevent duplicate connections
      if (
        session.ws &&
        (session.ws.readyState === WebSocket.OPEN ||
          session.ws.readyState === WebSocket.CONNECTING)
      ) {
        return;
      }

      // Clear existing timers
      if (session.reconnectTimer) clearTimeout(session.reconnectTimer);
      if (session.pingInterval) clearInterval(session.pingInterval);

      const wsUrl = `${getWsUrl()}/ws/terminal/${session.containerId}?token=${authToken}`;
      const ws = new WebSocket(wsUrl);

      updateSession(sessionId, (s) => ({ ...s, ws, status: "connecting" }));

      ws.onopen = () => {
        updateSession(sessionId, (s) => ({
          ...s,
          status: "connected",
          reconnectAttempts: 0,
        }));

        // Send initial resize first
        ws.send(
          JSON.stringify({
            type: "resize",
            cols: session.terminal.cols,
            rows: session.terminal.rows,
          }),
        );

        // Clear terminal and write banner after a short delay to ensure proper sizing
        setTimeout(() => {
          session.terminal.clear();
          session.terminal.write(REXEC_BANNER);
          session.terminal.writeln("\x1b[32m⚡ Connected\x1b[0m");
          session.terminal.writeln(
            "\x1b[38;5;243m  Type 'help' for tips & shortcuts\x1b[0m\r\n",
          );
        }, 100);

        // Send resize again after fit
        // Setup ping interval
        const pingInterval = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "ping" }));
          }
        }, WS_PING_INTERVAL);

        updateSession(sessionId, (s) => ({ ...s, pingInterval }));
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "output") {
            // Check for setup/installation indicators
            const data = msg.data as string;
            const setupPatterns = [
              /installing/i,
              /setting up/i,
              /configuring/i,
              /downloading/i,
              /unpacking/i,
              /processing/i,
              /apt.*get/i,
              /apk.*add/i,
              /yum.*install/i,
              /dnf.*install/i,
            ];

            const isSetupActivity = setupPatterns.some((pattern) =>
              pattern.test(data),
            );
            if (isSetupActivity) {
              // Extract a short message from the output
              const lines = data.split("\n").filter((l) => l.trim());
              const lastLine = lines[lines.length - 1] || "";
              const setupMsg =
                lastLine.slice(0, 50) + (lastLine.length > 50 ? "..." : "");
              updateSession(sessionId, (s) => ({
                ...s,
                isSettingUp: true,
                setupMessage: setupMsg,
              }));

              // Clear setup state after inactivity
              setTimeout(() => {
                updateSession(sessionId, (s) => ({
                  ...s,
                  isSettingUp: false,
                  setupMessage: "",
                }));
              }, 3000);
            }

            session.terminal.write(data);
          } else if (msg.type === "error") {
            session.terminal.writeln(`\r\n\x1b[31mError: ${msg.data}\x1b[0m`);
          } else if (msg.type === "ping") {
            ws.send(JSON.stringify({ type: "pong" }));
          } else if (msg.type === "setup") {
            // Explicit setup message from backend
            updateSession(sessionId, (s) => ({
              ...s,
              isSettingUp: true,
              setupMessage: msg.message || "Setting up environment...",
            }));
          } else if (msg.type === "setup_complete") {
            updateSession(sessionId, (s) => ({
              ...s,
              isSettingUp: false,
              setupMessage: "",
            }));
          } else if (msg.type === "stats") {
            // Handle stats updates
            try {
              const statsData = JSON.parse(msg.data);
              updateSession(sessionId, (s) => ({
                ...s,
                stats: {
                  cpu: statsData.cpu_percent || 0,
                  memory: statsData.memory || 0,
                  memoryLimit: statsData.memory_limit || 0,
                },
              }));
            } catch (e) {
              console.error("Failed to parse stats:", e);
            }
          }
        } catch {
          // Raw data fallback
          session.terminal.write(event.data);
        }
      };

      ws.onclose = (event) => {
        updateSession(sessionId, (s) => ({ ...s, status: "disconnected" }));

        const currentSession = getState().sessions.get(sessionId);
        if (!currentSession) return;

        if (currentSession.pingInterval) {
          clearInterval(currentSession.pingInterval);
        }

        // Don't reconnect if we intentionally closed or container is gone
        // Code 1000 = normal close, 1006 = abnormal (server rejected), 4000+ = custom codes
        const shouldNotReconnect = event.code === 1000 || 
          event.code === 1006 || 
          event.code >= 4000 ||
          currentSession.reconnectAttempts >= WS_MAX_RECONNECT;

        if (!shouldNotReconnect && currentSession.reconnectAttempts < WS_MAX_RECONNECT) {
          updateSession(sessionId, (s) => ({
            ...s,
            status: "connecting",
            reconnectAttempts: s.reconnectAttempts + 1,
          }));

          session.terminal.writeln(
            `\r\n\x1b[33mReconnecting (${currentSession.reconnectAttempts + 1}/${WS_MAX_RECONNECT})...\x1b[0m`,
          );

          const timer = setTimeout(() => {
            this.connectWebSocket(sessionId);
          }, WS_RECONNECT_DELAY);

          updateSession(sessionId, (s) => ({ ...s, reconnectTimer: timer }));
        } else {
          // Don't spam the terminal with reconnection messages
          if (currentSession.reconnectAttempts === 0) {
            session.terminal.writeln(
              "\r\n\x1b[31mConnection closed. Container may be stopped or unavailable.\x1b[0m",
            );
          } else {
            session.terminal.writeln(
              "\r\n\x1b[31mConnection lost. Click reconnect to try again.\x1b[0m",
            );
          }
          updateSession(sessionId, (s) => ({ ...s, status: "error" }));
        }
      };

      ws.onerror = (error) => {
        // WebSocket errors before connection opens usually mean container is unavailable
        // Don't write error message here - onclose will handle it
        console.error("[Terminal] WebSocket error:", error);
      };

      // Handle terminal input
      session.terminal.onData((data) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "input", data }));
        }
      });
    },

    // Reconnect a session
    reconnectSession(sessionId: string) {
      updateSession(sessionId, (s) => ({ ...s, reconnectAttempts: 0 }));
      this.connectWebSocket(sessionId);
    },

    // Close a session
    closeSession(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return;

      // Cleanup
      if (session.ws) session.ws.close();
      if (session.pingInterval) clearInterval(session.pingInterval);
      if (session.reconnectTimer) clearTimeout(session.reconnectTimer);
      if (session.resizeObserver) session.resizeObserver.disconnect();
      if (session.terminal) session.terminal.dispose();

      update((state) => {
        // Create a new Map to ensure Svelte reactivity
        const newSessions = new Map(state.sessions);
        newSessions.delete(sessionId);

        // Set new active session if needed (prefer docked sessions over detached)
        let newActiveId: string | null = null;
        if (state.activeSessionId === sessionId) {
          // Find another docked (non-detached) session to make active
          const remainingDocked = Array.from(newSessions.values()).filter(
            (s) => !s.isDetached,
          );
          newActiveId =
            remainingDocked.length > 0
              ? remainingDocked[remainingDocked.length - 1].id
              : null;
        } else {
          newActiveId = state.activeSessionId;
        }

        // Update URL based on remaining sessions
        if (newActiveId) {
          const newSession = newSessions.get(newActiveId);
          if (newSession) {
            this.updateUrl(newSession.containerId);
          }
        } else {
          this.clearUrl();
        }

        return {
          ...state,
          sessions: newSessions,
          activeSessionId: newActiveId,
        };
      });
    },

    // Close all docked sessions (not detached ones)
    closeAllSessions() {
      const state = getState();
      state.sessions.forEach((session, id) => {
        // Only close non-detached sessions
        if (!session.isDetached) {
          this.closeSession(id);
        }
      });
      // Only clear URL if no sessions remain
      const remaining = getState();
      if (remaining.sessions.size === 0) {
        this.clearUrl();
      }
    },

    // Close all sessions including detached (for logout/cleanup)
    closeAllSessionsForce() {
      const state = getState();
      state.sessions.forEach((_, id) => this.closeSession(id));
      this.clearUrl();
    },

    // Set active session
    setActiveSession(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (session) {
        this.updateUrl(session.containerId);
      }
      update((state) => ({ ...state, activeSessionId: sessionId }));
    },

    // Set view mode
    setViewMode(mode: ViewMode) {
      update((state) => {
        const newState = { ...state, viewMode: mode };
        savePreferences(newState);
        return newState;
      });
    },

    // Toggle view mode
    toggleViewMode() {
      const state = getState();
      this.setViewMode(state.viewMode === "floating" ? "docked" : "floating");
    },

    // Alias for toggleViewMode
    toggleFloating() {
      this.toggleViewMode();
    },

    // Toggle fullscreen mode
    toggleFullscreen() {
      const state = getState();
      if (state.viewMode === "fullscreen") {
        // Exit fullscreen - go back to docked
        this.setViewMode("docked");
      } else {
        // Enter fullscreen
        this.setViewMode("fullscreen");
      }
    },

    // Minimize
    minimize() {
      update((state) => ({ ...state, isMinimized: true }));
    },

    // Restore
    restore() {
      update((state) => ({ ...state, isMinimized: false }));
    },

    // Update floating position
    setFloatingPosition(x: number, y: number) {
      update((state) => {
        const newState = { ...state, floatingPosition: { x, y } };
        savePreferences(newState);
        return newState;
      });
    },

    // Update floating size
    setFloatingSize(width: number, height: number) {
      update((state) => {
        const newState = { ...state, floatingSize: { width, height } };
        savePreferences(newState);
        return newState;
      });
    },

    // Resize all terminals
    fitAll() {
      const state = getState();
      state.sessions.forEach((session) => {
        try {
          session.fitAddon.fit();

          // Send resize to server
          if (session.ws?.readyState === WebSocket.OPEN) {
            session.ws.send(
              JSON.stringify({
                type: "resize",
                cols: session.terminal.cols,
                rows: session.terminal.rows,
              }),
            );
          }
        } catch (e) {
          console.error("Failed to fit terminal:", e);
        }
      });
    },

    // Fit a specific session
    fitSession(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return;

      try {
        session.fitAddon.fit();

        if (session.ws?.readyState === WebSocket.OPEN) {
          session.ws.send(
            JSON.stringify({
              type: "resize",
              cols: session.terminal.cols,
              rows: session.terminal.rows,
            }),
          );
        }
      } catch (e) {
        console.error("Failed to fit terminal:", e);
      }
    },

    // Attach terminal to DOM element
    attachTerminal(sessionId: string, element: HTMLElement) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || !element) return;

      // Clean up old resize observer if exists
      if (session.resizeObserver) {
        session.resizeObserver.disconnect();
      }

      session.terminal.open(element);

      // Setup resize observer
      if (window.ResizeObserver) {
        const resizeObserver = new ResizeObserver(() => {
          this.fitSession(sessionId);
        });
        resizeObserver.observe(element);
        updateSession(sessionId, (s) => ({ ...s, resizeObserver }));
      }

      // Multiple fit attempts to ensure proper sizing
      setTimeout(() => this.fitSession(sessionId), 50);
      setTimeout(() => this.fitSession(sessionId), 150);
      setTimeout(() => this.fitSession(sessionId), 300);
    },

    // Re-attach terminal to a new DOM element (for dock/float switching)
    reattachTerminal(sessionId: string, element: HTMLElement) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || !element) return;

      // Clean up old resize observer
      if (session.resizeObserver) {
        session.resizeObserver.disconnect();
      }

      // Clear the new container
      element.innerHTML = "";

      // Instead of calling terminal.open() which clears the input buffer,
      // move the existing terminal DOM element to preserve all state
      const existingElement = session.terminal.element;
      if (existingElement) {
        // Move the terminal's existing DOM element to the new container
        // This preserves the terminal state including current input buffer
        element.appendChild(existingElement);
      } else {
        // Fallback: if somehow there's no element, open normally
        // This should rarely happen, only on first attachment
        session.terminal.open(element);
      }

      // Force a refresh to ensure proper rendering after reattachment
      session.terminal.refresh(0, session.terminal.rows - 1);

      // Setup new resize observer
      if (window.ResizeObserver) {
        const resizeObserver = new ResizeObserver(() => {
          this.fitSession(sessionId);
        });
        resizeObserver.observe(element);
        updateSession(sessionId, (s) => ({ ...s, resizeObserver }));
      }

      // Multiple fit attempts to ensure proper sizing after view mode switch
      setTimeout(() => {
        this.fitSession(sessionId);
        session.terminal.focus();
      }, 50);
      setTimeout(() => this.fitSession(sessionId), 150);
      setTimeout(() => this.fitSession(sessionId), 300);
      setTimeout(() => this.fitSession(sessionId), 500);
    },

    // Write to terminal
    writeToSession(sessionId: string, data: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (session) {
        session.terminal.write(data);
      }
    },

    // Write line to terminal
    writeLineToSession(sessionId: string, data: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (session) {
        session.terminal.writeln(data);
      }
    },

    // Get session by container ID
    getSessionByContainerId(containerId: string): TerminalSession | undefined {
      const state = getState();
      return Array.from(state.sessions.values()).find(
        (s) => s.containerId === containerId,
      );
    },

    // Check if container has active session
    hasActiveSession(containerId: string): boolean {
      return !!this.getSessionByContainerId(containerId);
    },

    // Get the container ID of the active session
    getActiveContainerId(): string | null {
      const state = getState();
      if (!state.activeSessionId) return null;
      const session = state.sessions.get(state.activeSessionId);
      return session?.containerId || null;
    },

    // Detach a session into its own floating window
    detachSession(sessionId: string, x?: number, y?: number) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || session.isDetached) return;

      // Calculate position (use provided coords or offset from main window)
      const posX = x ?? state.floatingPosition.x + 50;
      const posY = y ?? state.floatingPosition.y + 50;

      // Assign a new highest z-index
      const newZIndex = state.topZIndex + 1;

      update((s) => ({ ...s, topZIndex: newZIndex }));

      updateSession(sessionId, (s) => ({
        ...s,
        isDetached: true,
        detachedPosition: { x: posX, y: posY },
        detachedSize: { width: 600, height: 400 },
        detachedZIndex: newZIndex,
      }));

      // If this was the active session, switch to another non-detached session
      if (state.activeSessionId === sessionId) {
        const remaining = Array.from(state.sessions.values()).filter(
          (s) => s.id !== sessionId && !s.isDetached,
        );
        update((s) => ({
          ...s,
          activeSessionId: remaining.length > 0 ? remaining[0].id : null,
        }));
      }

      // Fit the terminal after detaching
      setTimeout(() => this.fitSession(sessionId), 100);
    },

    // Bring a detached window to front
    bringToFront(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || !session.isDetached) return;

      // Only update if not already on top
      if (session.detachedZIndex < state.topZIndex) {
        const newZIndex = state.topZIndex + 1;
        update((s) => ({ ...s, topZIndex: newZIndex }));
        updateSession(sessionId, (s) => ({
          ...s,
          detachedZIndex: newZIndex,
        }));
      }
    },

    // Attach a detached session back to the main terminal panel
    attachSession(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || !session.isDetached) return;

      updateSession(sessionId, (s) => ({
        ...s,
        isDetached: false,
      }));

      // Make it the active session
      update((s) => ({
        ...s,
        activeSessionId: sessionId,
        isMinimized: false,
      }));

      // Fit the terminal after attaching
      setTimeout(() => this.fitSession(sessionId), 100);
    },

    // Update detached window position
    setDetachedPosition(sessionId: string, x: number, y: number) {
      updateSession(sessionId, (s) => ({
        ...s,
        detachedPosition: { x, y },
      }));
    },

    // Update detached window size
    setDetachedSize(sessionId: string, width: number, height: number) {
      updateSession(sessionId, (s) => ({
        ...s,
        detachedSize: { width, height },
      }));
    },
  };
}

// Export the store
export const terminal = createTerminalStore();

// Derived stores
export const activeSession = derived(terminal, ($terminal) =>
  $terminal.activeSessionId
    ? $terminal.sessions.get($terminal.activeSessionId)
    : null,
);

export const sessionCount = derived(
  terminal,
  ($terminal) => $terminal.sessions.size,
);

export const hasSessions = derived(
  terminal,
  ($terminal) => $terminal.sessions.size > 0,
);

// Get set of container IDs that have active (connected) terminal sessions
export const connectedContainerIds = derived(terminal, ($terminal) => {
  const connected = new Set<string>();
  // Use Array.from to ensure proper iteration and reactivity
  const sessions = Array.from($terminal.sessions.values());
  for (const session of sessions) {
    if (session.status === "connected" || session.status === "connecting") {
      connected.add(session.containerId);
    }
  }
  return connected;
});

// Check if a specific container has an active terminal connection
export function isContainerConnected(containerId: string): boolean {
  const state = get(terminal);
  for (const session of state.sessions.values()) {
    if (session.containerId === containerId && 
        (session.status === "connected" || session.status === "connecting")) {
      return true;
    }
  }
  return false;
}

export const isFloating = derived(
  terminal,
  ($terminal) => $terminal.viewMode === "floating",
);

export const isDocked = derived(
  terminal,
  ($terminal) => $terminal.viewMode === "docked",
);

export const isFullscreen = derived(
  terminal,
  ($terminal) => $terminal.viewMode === "fullscreen",
);
