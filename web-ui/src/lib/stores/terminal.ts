import { writable, derived, get } from "svelte/store";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { WebglAddon } from "@xterm/addon-webgl";
import { token } from "./auth";
import { toast } from "./toast";

// Types
export type SessionStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "error";
export type ViewMode = "floating" | "docked" | "fullscreen";

// Split pane configuration
export type SplitDirection = "horizontal" | "vertical";

export interface SplitPane {
  id: string;
  sessionId: string; // Reference to the parent session
  terminal: Terminal;
  fitAddon: FitAddon;
  webglAddon: WebglAddon | null;
  resizeObserver: ResizeObserver | null;
  ws: WebSocket | null; // Each split pane has its own independent WebSocket
  reconnectAttempts: number;
  reconnectTimer: ReturnType<typeof setTimeout> | null;
}

export interface SplitPaneLayout {
  direction: SplitDirection;
  panes: string[]; // Pane IDs
  sizes: number[]; // Percentage sizes for each pane
}

export interface TerminalSession {
  id: string;
  containerId: string;
  name: string;
  terminal: Terminal;
  fitAddon: FitAddon;
  webglAddon: WebglAddon | null;
  ws: WebSocket | null;
  status: SessionStatus;
  reconnectAttempts: number;
  reconnectTimer: ReturnType<typeof setTimeout> | null;
  pingInterval: ReturnType<typeof setInterval> | null;
  resizeObserver: ResizeObserver | null;
  isSettingUp: boolean;
  setupMessage: string;
  // Collaboration mode
  isCollabSession: boolean;
  collabMode: 'view' | 'control' | null; // null if not a collab session
  collabRole: 'owner' | 'editor' | 'viewer' | null;
  // Stats
  stats: {
    cpu: number;
    memory: number;
    memoryLimit: number;
    diskRead: number;
    diskWrite: number;
    diskLimit: number;
    netRx: number;
    netTx: number;
  };
  // Detached window state (when popped out as separate floating window)
  isDetached: boolean;
  detachedPosition: { x: number; y: number };
  detachedSize: { width: number; height: number };
  detachedZIndex: number;
  // Split pane support - multiple views into the same terminal
  splitPanes: Map<string, SplitPane>;
  splitLayout: SplitPaneLayout | null;
  activePaneId: string | null; // Which pane is focused
}

export interface TerminalState {
  sessions: Map<string, TerminalSession>;
  activeSessionId: string | null;
  viewMode: ViewMode;
  isMinimized: boolean;
  floatingPosition: { x: number; y: number };
  floatingSize: { width: number; height: number };
  dockedHeight: number; // Height in vh units for docked mode
  topZIndex: number; // Track highest z-index for detached windows
}

// Constants
const WS_MAX_RECONNECT = 10;           // More attempts for poor networks
const WS_RECONNECT_BASE_DELAY = 1000;  // Start with 1s delay
const WS_RECONNECT_MAX_DELAY = 30000;  // Max 30s between retries
const WS_PING_INTERVAL = 25000;
const WS_SILENT_RECONNECT_THRESHOLD = 3; // Show message only after N silent attempts

const REXEC_BANNER =
  "\x1b[38;5;46m\r\n" +
  "  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗\r\n" +
  "  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝\r\n" +
  "  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║\r\n" +
  "  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║\r\n" +
  "  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗\r\n" +
  "  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝\r\n" +
  "\x1b[0m\x1b[38;5;243m  Terminal as a Service · rexec.dev\x1b[0m\r\n\r\n";

// Calculate responsive font size based on screen dimensions
function getResponsiveFontSize(): number {
  if (typeof window === "undefined") return 13;
  
  const width = window.innerWidth;
  const height = window.innerHeight;
  const minDim = Math.min(width, height);
  
  // Scale font size based on screen size
  // Small screens (< 768px): 11px
  // Medium screens (768-1200px): 12px
  // Large screens (1200-1600px): 13px
  // Extra large (> 1600px): 14px
  if (minDim < 768) return 11;
  if (width < 1200) return 12;
  if (width < 1600) return 13;
  return 14;
}

// Terminal configuration - optimized for large data handling
const TERMINAL_OPTIONS = {
  cursorBlink: true,
  cursorStyle: "bar" as const,
  fontSize: getResponsiveFontSize(),
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
  scrollback: 50000,            // Large scrollback for heavy output (e.g., opencode)
  fastScrollModifier: "alt" as const,    // Alt+scroll for fast scrolling
  fastScrollSensitivity: 15,    // Faster scroll speed
  scrollSensitivity: 5,         // Improved normal scroll speed
  smoothScrollDuration: 0,      // Disable smooth scrolling for performance
  windowsMode: false,           // Optimize for non-Windows
  convertEol: false,            // Don't convert line endings
  rightClickSelectsWord: true,
  drawBoldTextInBrightColors: true,
  minimumContrastRatio: 1,      // Don't adjust colors (faster)
  // Performance optimizations for large data
  rescaleOverlappingGlyphs: true,
  scrollOnUserInput: true,      // Auto-scroll on new input
  linkHandler: undefined,       // Disable link detection for performance
};

// Initial state
const initialState: TerminalState = {
  sessions: new Map(),
  activeSessionId: null,
  viewMode: "floating",
  isMinimized: false,
  floatingPosition: { x: 100, y: 100 },
  floatingSize: { width: 700, height: 500 },
  dockedHeight: 45, // 45vh default
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
        dockedHeight: prefs.dockedHeight || initialState.dockedHeight,
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
        dockedHeight: state.dockedHeight,
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
      // WebGL addon will be loaded after terminal is attached to DOM

      const sessionId = generateSessionId();
      const tabName =
        existingCount > 0 ? `${validName} (${existingCount + 1})` : validName;
      const session: TerminalSession = {
        id: sessionId,
        containerId,
        name: tabName,
        terminal,
        fitAddon,
        webglAddon: null, // Will be set when attached to DOM
        ws: null,
        status: "connecting",
        reconnectAttempts: 0,
        reconnectTimer: null,
        pingInterval: null,
        resizeObserver: null,
        isSettingUp: false,
        setupMessage: "",
        isCollabSession: false,
        collabMode: null,
        collabRole: null,
        stats: {
          cpu: 0,
          memory: 0,
          memoryLimit: 0,
          diskRead: 0,
          diskWrite: 0,
          diskLimit: 0,
          netRx: 0,
          netTx: 0,
        },
        isDetached: false,
        detachedPosition: { x: 150, y: 150 },
        detachedSize: { width: 600, height: 400 },
        detachedZIndex: 1000,
        // Split pane support
        splitPanes: new Map(),
        splitLayout: null,
        activePaneId: null,
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

    // Create a collab session (joined via share link)
    createCollabSession(containerId: string, name: string, mode: 'view' | 'control', role: 'owner' | 'editor' | 'viewer') {
      const sessionId = this.createSession(containerId, name);
      
      // Mark as collab session with mode/role
      updateSession(sessionId, (s) => ({
        ...s,
        isCollabSession: true,
        collabMode: mode,
        collabRole: role,
      }));
      
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

        // Clear terminal and write banner immediately
        session.terminal.clear();
        session.terminal.write(REXEC_BANNER);
        session.terminal.writeln("\x1b[32m› Connected\x1b[0m");
        session.terminal.writeln(
          "\x1b[38;5;243m  Type 'help' for tips & shortcuts\x1b[0m\r\n",
        );

        // Send resize again after fit
        // Setup ping interval
        const pingInterval = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "ping" }));
          }
        }, WS_PING_INTERVAL);

        updateSession(sessionId, (s) => ({ ...s, pingInterval }));
      };

      // Output buffer for batching writes (prevents browser freeze on large outputs)
      let outputBuffer = '';
      let flushTimeout: ReturnType<typeof setTimeout> | null = null;
      const FLUSH_INTERVAL = 16; // ~60fps
      const MAX_BUFFER_SIZE = 64 * 1024; // 64KB max before force flush
      
      // Filter corrupted mouse tracking data from output
      // Pattern matches raw mouse coord spam like "35;166;10M35;165;10M..." and full sequences
      const sanitizeOutput = (data: string): string => {
        // Remove mouse tracking spam (fragmented SGR mouse sequences)
        // Includes handling for the CSI header \x1b[< to avoid leaving dangling escape codes
        return data.replace(/(?:\x1b\[<)?(?:\d+;\d+;\d+[Mm])+/g, '');
      };
      
      const flushBuffer = () => {
        if (outputBuffer && session.terminal) {
          // Sanitize before writing to terminal
          const sanitized = sanitizeOutput(outputBuffer);
          if (sanitized) {
            // Write to main terminal only (split panes have their own WebSocket connections)
            session.terminal.write(sanitized);
          }
          outputBuffer = '';
        }
        flushTimeout = null;
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

            // Buffer output for batched writes (prevents freeze on large outputs)
            outputBuffer += data;
            
            // Force flush if buffer is too large
            if (outputBuffer.length >= MAX_BUFFER_SIZE) {
              if (flushTimeout) {
                clearTimeout(flushTimeout);
              }
              flushBuffer();
            } else if (!flushTimeout) {
              // Schedule flush on next animation frame
              flushTimeout = setTimeout(flushBuffer, FLUSH_INTERVAL);
            }
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
                  diskRead: statsData.disk_read || 0,
                  diskWrite: statsData.disk_write || 0,
                  diskLimit: statsData.disk_limit || 0,
                  netRx: statsData.net_rx || 0,
                  netTx: statsData.net_tx || 0,
                },
              }));
            } catch (e) {
              console.error("Failed to parse stats:", e);
            }
          }
        } catch {
          // Raw data fallback - also buffer this
          outputBuffer += event.data;
          if (outputBuffer.length >= MAX_BUFFER_SIZE) {
            if (flushTimeout) {
              clearTimeout(flushTimeout);
            }
            flushBuffer();
          } else if (!flushTimeout) {
            flushTimeout = setTimeout(flushBuffer, FLUSH_INTERVAL);
          }
        }
      };

      ws.onclose = (event) => {
        const currentSession = getState().sessions.get(sessionId);
        if (!currentSession) return;

        if (currentSession.pingInterval) {
          clearInterval(currentSession.pingInterval);
        }

        // Determine if we should attempt reconnection
        // Code 1000 = normal close (intentional), 4000+ = custom error codes (container gone, auth failed, etc)
        // Code 1006 = abnormal close (network issue) - SHOULD reconnect
        const isIntentionalClose = event.code === 1000;
        const isContainerGone = event.code >= 4000;
        const maxAttemptsReached = currentSession.reconnectAttempts >= WS_MAX_RECONNECT;
        
        const shouldReconnect = !isIntentionalClose && !isContainerGone && !maxAttemptsReached;

        if (shouldReconnect) {
          const attemptNum = currentSession.reconnectAttempts + 1;
          
          // Calculate exponential backoff delay
          const baseDelay = WS_RECONNECT_BASE_DELAY;
          const delay = Math.min(baseDelay * Math.pow(1.5, currentSession.reconnectAttempts), WS_RECONNECT_MAX_DELAY);
          
          // Update status to connecting (not error/disconnected during silent reconnect)
          updateSession(sessionId, (s) => ({
            ...s,
            status: "connecting",
            reconnectAttempts: attemptNum,
          }));

          // Only show message after silent threshold is exceeded
          if (attemptNum > WS_SILENT_RECONNECT_THRESHOLD) {
            session.terminal.writeln(
              `\r\n\x1b[33m⟳ Reconnecting (${attemptNum}/${WS_MAX_RECONNECT})...\x1b[0m`,
            );
          } else {
            // Silent reconnect - just log to console
            console.log(`[Terminal] Silent reconnect attempt ${attemptNum}/${WS_MAX_RECONNECT} for ${sessionId}`);
          }

          const timer = setTimeout(() => {
            this.connectWebSocket(sessionId);
          }, delay);

          updateSession(sessionId, (s) => ({ ...s, reconnectTimer: timer }));
        } else {
          // Set to disconnected/error state
          updateSession(sessionId, (s) => ({ ...s, status: "disconnected" }));
          
          // Show appropriate message based on reason
          if (isContainerGone) {
            session.terminal.writeln(
              "\r\n\x1b[31m✖ Terminal session ended. Container may have been stopped or removed.\x1b[0m",
            );
            updateSession(sessionId, (s) => ({ ...s, status: "error" }));
          } else if (maxAttemptsReached) {
            session.terminal.writeln(
              "\r\n\x1b[31m✖ Connection lost after multiple attempts. Click \x1b[33m⟳\x1b[31m to reconnect.\x1b[0m",
            );
            updateSession(sessionId, (s) => ({ ...s, status: "error" }));
          }
          // If intentional close (code 1000), don't show any message
        }
      };

      ws.onerror = () => {
        // WebSocket errors are handled by onclose - just log silently
        console.log("[Terminal] WebSocket error - will attempt reconnect");
      };

      // Handle terminal input with chunking for large pastes
      let inputQueue: string[] = [];
      let isProcessingQueue = false;
      const CHUNK_SIZE = 4096; // 4KB chunks for large pastes
      const CHUNK_DELAY = 10; // 10ms between chunks

      const processInputQueue = async () => {
        if (isProcessingQueue || inputQueue.length === 0) return;
        isProcessingQueue = true;

        while (inputQueue.length > 0 && ws.readyState === WebSocket.OPEN) {
          const chunk = inputQueue.shift()!;
          ws.send(JSON.stringify({ type: "input", data: chunk }));
          // Small delay between chunks to prevent overwhelming the connection
          if (inputQueue.length > 0) {
            await new Promise((resolve) => setTimeout(resolve, CHUNK_DELAY));
          }
        }
        isProcessingQueue = false;
      };

      session.terminal.onData((data) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        
        // Block input if this is a view-only collab session
        const currentSession = getState().sessions.get(sessionId);
        if (currentSession?.isCollabSession && currentSession?.collabMode === 'view') {
          return;
        }

        // For large pastes, chunk the data
        if (data.length > CHUNK_SIZE) {
          for (let i = 0; i < data.length; i += CHUNK_SIZE) {
            inputQueue.push(data.slice(i, i + CHUNK_SIZE));
          }
          processInputQueue();
        } else {
          // Small inputs go directly
          ws.send(JSON.stringify({ type: "input", data: data }));
        }
      });
    },

    // Reset terminal state (fix garbled output)
    resetSession(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || !session.terminal) return;

      // Send full reset sequence
      session.terminal.write('\x1bc'); // Full reset
      session.terminal.clear(); // Clear buffer
      
      // Re-write banner
      session.terminal.write(REXEC_BANNER);
      session.terminal.writeln("\x1b[32m› Terminal Reset\x1b[0m");
      
      // Resend resize to ensure backend sync
      if (session.ws && session.ws.readyState === WebSocket.OPEN) {
        session.ws.send(
          JSON.stringify({
            type: "resize",
            cols: session.terminal.cols,
            rows: session.terminal.rows,
          }),
        );
        // Request a redraw by sending a space and backspace (hacky but effective for some shells)
        // Or just let the user press enter
      }
      
      toast.success("Terminal reset");
    },

    // Reconnect a session
    reconnectSession(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      
      // Close existing WebSocket if any
      if (session?.ws) {
        session.ws.close();
      }
      
      updateSession(sessionId, (s) => ({ ...s, reconnectAttempts: 0, status: 'connecting' }));
      this.connectWebSocket(sessionId);
    },

    // Update a session's container ID (used when container is recreated with new ID)
    updateSessionContainerId(oldContainerId: string, newContainerId: string) {
      const state = getState();
      
      // Find all sessions by old container ID
      const sessionsToUpdate: string[] = [];
      
      for (const [id, session] of state.sessions) {
        if (session.containerId === oldContainerId) {
          sessionsToUpdate.push(id);
        }
      }
      
      if (sessionsToUpdate.length === 0) {
        console.log(`[Terminal] No session found for container ${oldContainerId}`);
        return;
      }
      
      console.log(`[Terminal] Updating ${sessionsToUpdate.length} sessions from ${oldContainerId} to ${newContainerId}`);
      
      sessionsToUpdate.forEach(sessionId => {
        const session = state.sessions.get(sessionId);
        if (!session) return;

        // Close old WebSocket
        if (session.ws) {
          session.ws.close();
        }
        
        // Update the session with new container ID
        updateSession(sessionId, (s) => ({
          ...s,
          containerId: newContainerId,
          status: "connecting",
          reconnectAttempts: 0,
        }));
        
        // Connect to new container
        this.connectWebSocket(sessionId);

        // Reconnect split panes
        session.splitPanes.forEach((pane, paneId) => {
          if (pane.ws) {
            pane.ws.close();
          }
          this.connectSplitPaneWebSocket(sessionId, paneId);
        });
      });
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
      if (session.webglAddon) session.webglAddon.dispose();
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

    // Update docked height (in vh units)
    setDockedHeight(height: number) {
      update((state) => {
        // Clamp between 20vh and 90vh
        const clampedHeight = Math.max(20, Math.min(90, height));
        const newState = { ...state, dockedHeight: clampedHeight };
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

    // Update font size for all terminals (called on window resize)
    updateFontSize() {
      const newFontSize = getResponsiveFontSize();
      const state = getState();
      state.sessions.forEach((session) => {
        try {
          session.terminal.options.fontSize = newFontSize;
          session.fitAddon.fit();
          // Also update split panes
          session.splitPanes.forEach((pane) => {
            pane.terminal.options.fontSize = newFontSize;
            pane.fitAddon.fit();
          });
        } catch (e) {
          console.error("Failed to update font size:", e);
        }
      });
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

      // Try to load WebGL addon for GPU-accelerated rendering
      // This significantly improves performance with large outputs
      try {
        const webglAddon = new WebglAddon();
        webglAddon.onContextLoss(() => {
          // WebGL context lost, dispose and fall back to canvas renderer
          webglAddon.dispose();
          updateSession(sessionId, (s) => ({ ...s, webglAddon: null }));
        });
        session.terminal.loadAddon(webglAddon);
        updateSession(sessionId, (s) => ({ ...s, webglAddon }));
      } catch (e) {
        // WebGL not available, terminal will use canvas renderer
        console.warn("WebGL addon not available, using canvas renderer:", e);
      }

      // Setup resize observer
      if (window.ResizeObserver) {
        const resizeObserver = new ResizeObserver(() => {
          this.fitSession(sessionId);
        });
        resizeObserver.observe(element);
        updateSession(sessionId, (s) => ({ ...s, resizeObserver }));
      }

      // Single fit call - ResizeObserver handles subsequent fits
      requestAnimationFrame(() => this.fitSession(sessionId));
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

      // Single fit call after reattachment - ResizeObserver handles the rest
      requestAnimationFrame(() => {
        this.fitSession(sessionId);
        session.terminal.focus();
      });
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
      requestAnimationFrame(() => this.fitSession(sessionId));
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
      requestAnimationFrame(() => this.fitSession(sessionId));
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

    // ============ SPLIT PANE METHODS ============

    // Generate a unique pane ID
    _generatePaneId(): string {
      return `pane-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    },

    // Split the current terminal view (creates a new INDEPENDENT terminal session to the same container)
    splitPane(sessionId: string, direction: SplitDirection = "horizontal"): string | null {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return null;

      const authToken = get(token);
      if (!authToken) return null;

      // Create a new terminal instance with its OWN WebSocket connection
      const newTerminal = new Terminal(TERMINAL_OPTIONS);
      const newFitAddon = new FitAddon();
      newTerminal.loadAddon(newFitAddon);
      newTerminal.loadAddon(new WebLinksAddon());

      const paneId = this._generatePaneId();
      const newPane: SplitPane = {
        id: paneId,
        sessionId,
        terminal: newTerminal,
        fitAddon: newFitAddon,
        webglAddon: null,
        resizeObserver: null,
        ws: null, // Each pane gets its own WebSocket
        reconnectAttempts: 0,
        reconnectTimer: null,
      };

      // Update session with new pane
      updateSession(sessionId, (s) => {
        const newPanes = new Map(s.splitPanes);
        newPanes.set(paneId, newPane);

        // If this is the first split, set up the layout
        let layout = s.splitLayout;
        if (!layout) {
          // First split - create initial layout with main terminal + new pane
          layout = {
            direction,
            panes: ["main", paneId],
            sizes: [50, 50],
          };
        } else if (layout.direction === direction) {
          // Same direction - add to existing layout
          const numPanes = layout.panes.length;
          const newSize = 100 / (numPanes + 1);
          layout = {
            ...layout,
            panes: [...layout.panes, paneId],
            sizes: layout.sizes.map(() => newSize).concat([newSize]),
          };
        } else {
          // Different direction - need to nest (simplified: just add to end)
          const numPanes = layout.panes.length;
          const newSize = 100 / (numPanes + 1);
          layout = {
            direction,
            panes: [...layout.panes, paneId],
            sizes: layout.sizes.map(() => newSize).concat([newSize]),
          };
        }

        return {
          ...s,
          splitPanes: newPanes,
          splitLayout: layout,
          activePaneId: paneId,
        };
      });

      // Connect the split pane to its own WebSocket session
      this.connectSplitPaneWebSocket(sessionId, paneId);

      return paneId;
    },

    // Connect a split pane to its own independent WebSocket session
    connectSplitPaneWebSocket(sessionId: string, paneId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return;

      const pane = session.splitPanes.get(paneId);
      if (!pane) return;

      const authToken = get(token);
      if (!authToken) return;

      // Prevent duplicate connections
      if (
        pane.ws &&
        (pane.ws.readyState === WebSocket.OPEN ||
          pane.ws.readyState === WebSocket.CONNECTING)
      ) {
        return;
      }

      // Clear existing timer
      if (pane.reconnectTimer) clearTimeout(pane.reconnectTimer);

      // Create independent WebSocket connection for this pane
      const wsUrl = `${getWsUrl()}/ws/terminal/${session.containerId}?token=${authToken}`;
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        // Reset reconnect attempts on successful connection
        updateSession(sessionId, (s) => {
          const newPanes = new Map(s.splitPanes);
          const p = newPanes.get(paneId);
          if (p) {
            newPanes.set(paneId, { ...p, reconnectAttempts: 0 });
          }
          return { ...s, splitPanes: newPanes };
        });

        // Send initial resize
        ws.send(
          JSON.stringify({
            type: "resize",
            cols: pane.terminal.cols,
            rows: pane.terminal.rows,
          }),
        );

        // Show connected message in split pane
        pane.terminal.writeln("\x1b[32m› Split session connected\x1b[0m\r\n");
      };

      // Output buffer for batching writes
      let outputBuffer = '';
      let flushTimeout: ReturnType<typeof setTimeout> | null = null;
      const FLUSH_INTERVAL = 16;
      const MAX_BUFFER_SIZE = 64 * 1024;

      const flushBuffer = () => {
        if (outputBuffer && pane.terminal) {
          // Filter mouse tracking spam
          const sanitized = outputBuffer.replace(/(?:\d+;\d+;\d+[Mm])+/g, '');
          if (sanitized) {
            pane.terminal.write(sanitized);
          }
          outputBuffer = '';
        }
        flushTimeout = null;
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "output") {
            outputBuffer += msg.data;
            if (outputBuffer.length >= MAX_BUFFER_SIZE) {
              if (flushTimeout) clearTimeout(flushTimeout);
              flushBuffer();
            } else if (!flushTimeout) {
              flushTimeout = setTimeout(flushBuffer, FLUSH_INTERVAL);
            }
          } else if (msg.type === "error") {
            pane.terminal.writeln(`\r\n\x1b[31mError: ${msg.data}\x1b[0m`);
          } else if (msg.type === "ping") {
            ws.send(JSON.stringify({ type: "pong" }));
          }
        } catch {
          outputBuffer += event.data;
          if (outputBuffer.length >= MAX_BUFFER_SIZE) {
            if (flushTimeout) clearTimeout(flushTimeout);
            flushBuffer();
          } else if (!flushTimeout) {
            flushTimeout = setTimeout(flushBuffer, FLUSH_INTERVAL);
          }
        }
      };

      ws.onclose = (event) => {
        const currentSession = getState().sessions.get(sessionId);
        if (!currentSession) return;
        
        const currentPane = currentSession.splitPanes.get(paneId);
        if (!currentPane) return;

        // Determine if we should attempt reconnection
        const isIntentionalClose = event.code === 1000;
        const isContainerGone = event.code >= 4000;
        const maxAttemptsReached = currentPane.reconnectAttempts >= WS_MAX_RECONNECT;
        
        const shouldReconnect = !isIntentionalClose && !isContainerGone && !maxAttemptsReached;

        if (shouldReconnect) {
          const attemptNum = currentPane.reconnectAttempts + 1;
          
          // Calculate exponential backoff delay
          const baseDelay = WS_RECONNECT_BASE_DELAY;
          const delay = Math.min(baseDelay * Math.pow(1.5, currentPane.reconnectAttempts), WS_RECONNECT_MAX_DELAY);

          // Update reconnect attempts
          updateSession(sessionId, (s) => {
            const newPanes = new Map(s.splitPanes);
            const p = newPanes.get(paneId);
            if (p) {
              newPanes.set(paneId, { ...p, reconnectAttempts: attemptNum });
            }
            return { ...s, splitPanes: newPanes };
          });

          // Only show message after silent threshold is exceeded
          if (attemptNum > WS_SILENT_RECONNECT_THRESHOLD) {
            pane.terminal.writeln(
              `\r\n\x1b[33m⟳ Split session reconnecting (${attemptNum}/${WS_MAX_RECONNECT})...\x1b[0m`,
            );
          } else {
            console.log(`[Terminal] Split pane silent reconnect attempt ${attemptNum}/${WS_MAX_RECONNECT} for ${paneId}`);
          }

          const timer = setTimeout(() => {
            this.connectSplitPaneWebSocket(sessionId, paneId);
          }, delay);

          // Save timer
          updateSession(sessionId, (s) => {
            const newPanes = new Map(s.splitPanes);
            const p = newPanes.get(paneId);
            if (p) {
              newPanes.set(paneId, { ...p, reconnectTimer: timer });
            }
            return { ...s, splitPanes: newPanes };
          });
        } else {
          // Show appropriate message based on reason
          if (isContainerGone) {
            pane.terminal.writeln(
              "\r\n\x1b[31m✖ Split session ended. Container may have been stopped or removed.\x1b[0m",
            );
          } else if (maxAttemptsReached) {
            pane.terminal.writeln(
              "\r\n\x1b[31m✖ Split session connection lost after multiple attempts.\x1b[0m",
            );
          } else {
             pane.terminal.writeln("\r\n\x1b[33m› Split session disconnected\x1b[0m");
          }
        }
      };

      ws.onerror = () => {
        console.log("[Terminal] Split pane WebSocket error - will attempt reconnect");
      };

      // Handle terminal input for split pane
      const mouseTrackingRegex = /\x1b\[<[\d;]+[Mm]/g;
      const mouseSpamRegex = /(?:\d+;\d+;\d+[Mm])+/g;

      pane.terminal.onData((data) => {
        if (ws.readyState !== WebSocket.OPEN) return;

        let filteredData = data.replace(mouseTrackingRegex, '');
        filteredData = filteredData.replace(mouseSpamRegex, '');
        if (!filteredData) return;

        ws.send(JSON.stringify({ type: "input", data: filteredData }));
      });

      // Store the WebSocket on the pane
      pane.ws = ws;

      // Update the pane in the session
      updateSession(sessionId, (s) => {
        const newPanes = new Map(s.splitPanes);
        newPanes.set(paneId, pane);
        return { ...s, splitPanes: newPanes };
      });
    },

    // Close a split pane
    closeSplitPane(sessionId: string, paneId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return;

      const pane = session.splitPanes.get(paneId);
      if (!pane) return;

      // Cleanup pane resources
      if (pane.reconnectTimer) clearTimeout(pane.reconnectTimer);
      if (pane.ws) pane.ws.close();
      if (pane.resizeObserver) pane.resizeObserver.disconnect();
      if (pane.webglAddon) pane.webglAddon.dispose();
      if (pane.terminal) pane.terminal.dispose();

      updateSession(sessionId, (s) => {
        const newPanes = new Map(s.splitPanes);
        newPanes.delete(paneId);

        // Update layout
        let layout = s.splitLayout;
        if (layout) {
          const paneIndex = layout.panes.indexOf(paneId);
          if (paneIndex !== -1) {
            const newPanesList = layout.panes.filter((p) => p !== paneId);
            if (newPanesList.length <= 1) {
              // Only main pane left, remove split layout
              layout = null;
            } else {
              // Redistribute sizes evenly
              const newSizes = layout.sizes.filter((_, i) => i !== paneIndex);
              const totalRemaining = newSizes.reduce((a, b) => a + b, 0);
              layout = {
                ...layout,
                panes: newPanesList,
                sizes: newSizes.map((s) => (s / totalRemaining) * 100),
              };
            }
          }
        }

        return {
          ...s,
          splitPanes: newPanes,
          splitLayout: layout,
          activePaneId: s.activePaneId === paneId ? null : s.activePaneId,
        };
      });
    },

    // Set active pane within a session
    setActivePaneId(sessionId: string, paneId: string | null) {
      updateSession(sessionId, (s) => ({
        ...s,
        activePaneId: paneId,
      }));
    },

    // Attach a split pane terminal to a DOM element
    attachSplitPane(sessionId: string, paneId: string, element: HTMLElement) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return;

      const pane = session.splitPanes.get(paneId);
      if (!pane || !element) return;

      // Clean up old resize observer
      if (pane.resizeObserver) {
        pane.resizeObserver.disconnect();
      }

      pane.terminal.open(element);

      // Try to load WebGL addon
      try {
        const webglAddon = new WebglAddon();
        webglAddon.onContextLoss(() => {
          webglAddon.dispose();
        });
        pane.terminal.loadAddon(webglAddon);
        pane.webglAddon = webglAddon;
      } catch (e) {
        console.warn("WebGL addon not available for split pane:", e);
      }

      // Setup resize observer
      if (window.ResizeObserver) {
        const resizeObserver = new ResizeObserver(() => {
          this.fitSplitPane(sessionId, paneId);
        });
        resizeObserver.observe(element);
        pane.resizeObserver = resizeObserver;
      }

      // Initial fit
      requestAnimationFrame(() => this.fitSplitPane(sessionId, paneId));
    },

    // Fit a split pane terminal
    fitSplitPane(sessionId: string, paneId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return;

      const pane = session.splitPanes.get(paneId);
      if (!pane) return;

      try {
        pane.fitAddon.fit();
      } catch (e) {
        console.error("Failed to fit split pane:", e);
      }
    },

    // Update split pane sizes (during resize)
    setSplitPaneSizes(sessionId: string, sizes: number[]) {
      updateSession(sessionId, (s) => {
        if (!s.splitLayout) return s;
        return {
          ...s,
          splitLayout: {
            ...s.splitLayout,
            sizes,
          },
        };
      });
    },

    // Toggle split direction
    toggleSplitDirection(sessionId: string) {
      updateSession(sessionId, (s) => {
        if (!s.splitLayout) return s;
        return {
          ...s,
          splitLayout: {
            ...s.splitLayout,
            direction: s.splitLayout.direction === "horizontal" ? "vertical" : "horizontal",
          },
        };
      });
      // Fit all panes after direction change
      requestAnimationFrame(() => {
        const session = getState().sessions.get(sessionId);
        if (session) {
          this.fitSession(sessionId);
          session.splitPanes.forEach((_, paneId) => {
            this.fitSplitPane(sessionId, paneId);
          });
        }
      });
    },
  };
}

// Export the store
export const terminal = createTerminalStore();

// Setup window resize listener for responsive font size
if (typeof window !== "undefined") {
  let resizeTimeout: ReturnType<typeof setTimeout>;
  window.addEventListener("resize", () => {
    // Debounce resize events
    clearTimeout(resizeTimeout);
    resizeTimeout = setTimeout(() => {
      terminal.updateFontSize();
    }, 150);
  });
}

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
