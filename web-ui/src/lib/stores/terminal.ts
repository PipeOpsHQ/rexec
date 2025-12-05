import { writable, derived, get } from "svelte/store";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { Unicode11Addon } from "@xterm/addon-unicode11";
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
  collabMode: "view" | "control" | null; // null if not a collab session
  collabRole: "owner" | "editor" | "viewer" | null;
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

// Constants - Production-grade terminal settings
const WS_MAX_RECONNECT = 15; // More attempts for poor networks
const WS_RECONNECT_BASE_DELAY = 100; // Start with 100ms for instant reconnect feel
const WS_RECONNECT_MAX_DELAY = 8000; // Max 8s between retries
const WS_PING_INTERVAL = 20000; // 20s ping to keep connection alive
const WS_SILENT_RECONNECT_THRESHOLD = 3; // Show message after 3 silent attempts

// Input/Output optimization constants
const INPUT_THROTTLE_MS = 0; // No throttle - send immediately for responsiveness
const OUTPUT_FLUSH_INTERVAL = 8; // ~120fps for smooth output
const OUTPUT_IMMEDIATE_THRESHOLD = 256; // Immediately write small outputs
const OUTPUT_MAX_BUFFER = 32 * 1024; // 32KB max buffer before force flush
const CHUNK_SIZE = 8192; // 8KB chunks for large pastes
const CHUNK_DELAY = 5; // 5ms between chunks - faster paste

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
  if (typeof window === "undefined") return 10;

  const width = window.innerWidth;
  const height = window.innerHeight;
  const minDim = Math.min(width, height);

  // Scale font size based on screen size
  // Small screens (< 768px): 9px
  // Medium screens (768-1200px): 10px
  // Large screens (1200-1600px): 11px
  // Extra large (> 1600px): 12px
  if (minDim < 768) return 9;
  if (width < 1200) return 10;
  if (width < 1600) return 11;
  return 12;
}

// Terminal configuration - Production-grade like Google Cloud Shell
const TERMINAL_OPTIONS = {
  cursorBlink: true,
  cursorStyle: "bar" as const,
  cursorWidth: 2,
  fontSize: getResponsiveFontSize(),
  fontFamily:
    '"JetBrains Mono", "Fira Code", "SF Mono", Menlo, Monaco, "Courier New", monospace',
  fontWeight: "400" as const,
  fontWeightBold: "600" as const,
  letterSpacing: 0,
  lineHeight: 1.2,
  theme: {
    background: "#0a0a0a",
    foreground: "#e0e0e0",
    cursor: "#00ff41",
    cursorAccent: "#0a0a0a",
    selectionBackground: "rgba(0, 255, 65, 0.3)",
    selectionForeground: "#ffffff",
    selectionInactiveBackground: "rgba(0, 255, 65, 0.15)",
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
  scrollback: 100000, // 100K lines - handles heavy output like builds/logs
  fastScrollModifier: "alt" as const,
  fastScrollSensitivity: 20,
  scrollSensitivity: 3,
  smoothScrollDuration: 0, // Instant scroll for responsiveness
  windowsMode: false,
  convertEol: false,
  rightClickSelectsWord: true,
  drawBoldTextInBrightColors: true,
  minimumContrastRatio: 1,
  // Performance optimizations
  rescaleOverlappingGlyphs: true,
  scrollOnUserInput: true,
  altClickMovesCursor: true, // Alt+click to move cursor (like iTerm2)
  macOptionIsMeta: true, // Option key as Meta for proper terminal shortcuts
  macOptionClickForcesSelection: true,
  // Accessibility
  screenReaderMode: false,
  // Bell
  bellStyle: "none" as const, // Silent bell
  // Tab stops
  tabStopWidth: 8,
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
      // Unicode11 addon for proper Unicode character widths (emojis, CJK, etc.)
      const unicode11Addon = new Unicode11Addon();
      terminal.loadAddon(unicode11Addon);
      terminal.unicode.activeVersion = "11";
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
    createCollabSession(
      containerId: string,
      name: string,
      mode: "view" | "control",
      role: "owner" | "editor" | "viewer",
    ): string | null {
      const sessionId = this.createSession(containerId, name);
      if (!sessionId) return null;

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

      const wsUrl = `${getWsUrl()}/ws/terminal/${session.containerId}?token=${authToken}&id=${sessionId}`;
      const ws = new WebSocket(wsUrl);

      updateSession(sessionId, (s) => ({ ...s, ws, status: "connecting" }));

      ws.onopen = () => {
        updateSession(sessionId, (s) => ({
          ...s,
          status: "connected",
          reconnectAttempts: 0,
        }));

        // Fit terminal first to get accurate dimensions
        try {
          session.fitAddon.fit();
        } catch (e) {
          // Ignore fit errors on initial connection
        }

        // Send resize with correct dimensions
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

        // Setup ping interval
        const pingInterval = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "ping" }));
          }
        }, WS_PING_INTERVAL);

        updateSession(sessionId, (s) => ({ ...s, pingInterval }));
      };

      // Output buffer for batching writes - optimized for smooth rendering
      let outputBuffer = "";
      let flushTimeout: ReturnType<typeof setTimeout> | null = null;
      let rafId: number | null = null;
      let lastFlushTime = 0;

      // Filter mouse tracking sequences from output
      const sanitizeOutput = (data: string): string => {
        // Only remove complete SGR mouse sequences
        return data.replace(/\x1b\[<\d+;\d+;\d+[Mm]/g, "");
      };

      // Immediate write for small, interactive output (like keystrokes)
      const writeImmediate = (data: string) => {
        if (session.terminal) {
          const sanitized = sanitizeOutput(data);
          if (sanitized) {
            session.terminal.write(sanitized);
          }
        }
      };

      // Flush buffer using requestAnimationFrame for smooth rendering
      const flushBuffer = () => {
        if (outputBuffer && session.terminal) {
          const sanitized = sanitizeOutput(outputBuffer);
          if (sanitized) {
            session.terminal.write(sanitized);
          }
          outputBuffer = "";
          lastFlushTime = performance.now();
        }
        flushTimeout = null;
        rafId = null;
      };

      // Schedule buffer flush - uses RAF for smooth 60fps rendering
      const scheduleFlush = () => {
        if (flushTimeout || rafId) return;

        const timeSinceLastFlush = performance.now() - lastFlushTime;

        // If we haven't flushed recently, use RAF for immediate smooth render
        if (timeSinceLastFlush > OUTPUT_FLUSH_INTERVAL) {
          rafId = requestAnimationFrame(flushBuffer);
        } else {
          // Otherwise schedule for next frame
          flushTimeout = setTimeout(() => {
            rafId = requestAnimationFrame(flushBuffer);
          }, OUTPUT_FLUSH_INTERVAL);
        }
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "output") {
            const data = msg.data as string;

            // Small interactive output (single chars, escape sequences) - write immediately
            // This makes typing feel instant like a native terminal
            if (data.length <= OUTPUT_IMMEDIATE_THRESHOLD) {
              writeImmediate(data);
              return;
            }

            // Check for setup/installation indicators (only for larger outputs)
            const setupPatterns = [
              /installing/i,
              /setting up/i,
              /configuring/i,
              /downloading/i,
              /unpacking/i,
              /processing/i,
            ];

            const isSetupActivity = setupPatterns.some((pattern) =>
              pattern.test(data),
            );
            if (isSetupActivity) {
              const lines = data.split("\n").filter((l) => l.trim());
              const lastLine = lines[lines.length - 1] || "";
              const setupMsg =
                lastLine.slice(0, 50) + (lastLine.length > 50 ? "..." : "");
              updateSession(sessionId, (s) => ({
                ...s,
                isSettingUp: true,
                setupMessage: setupMsg,
              }));

              setTimeout(() => {
                updateSession(sessionId, (s) => ({
                  ...s,
                  isSettingUp: false,
                  setupMessage: "",
                }));
              }, 3000);
            }

            // Buffer larger outputs for batched writes
            outputBuffer += data;

            // Force flush if buffer is too large
            if (outputBuffer.length >= OUTPUT_MAX_BUFFER) {
              if (flushTimeout) clearTimeout(flushTimeout);
              if (rafId) cancelAnimationFrame(rafId);
              flushBuffer();
            } else {
              scheduleFlush();
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
          if (outputBuffer.length >= OUTPUT_MAX_BUFFER) {
            if (flushTimeout) clearTimeout(flushTimeout);
            if (rafId) cancelAnimationFrame(rafId);
            flushBuffer();
          } else {
            scheduleFlush();
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
        // Code 1000 = normal close (intentional)
        // Code 4100 = container_restart_required - need to look up new container ID
        // Code 4000+ (other) = container gone, auth failed, etc
        // Code 1006 = abnormal close (network issue) - SHOULD reconnect
        const isIntentionalClose = event.code === 1000;
        const isContainerRestartRequired = event.code === 4100;
        const isContainerGone = event.code >= 4000 && event.code !== 4100;
        const maxAttemptsReached =
          currentSession.reconnectAttempts >= WS_MAX_RECONNECT;

        // Handle container restart - fetch new container ID and reconnect
        if (isContainerRestartRequired) {
          session.terminal.writeln(
            "\r\n\x1b[33m⟳ Container restarting, reconnecting...\x1b[0m",
          );

          // Fetch the container info to get the new ID
          this.refreshContainerAndReconnect(
            sessionId,
            currentSession.containerId,
          );
          return;
        }

        const shouldReconnect =
          !isIntentionalClose && !isContainerGone && !maxAttemptsReached;

        if (shouldReconnect) {
          const attemptNum = currentSession.reconnectAttempts + 1;

          // First attempt is instant, then exponential backoff
          const baseDelay = WS_RECONNECT_BASE_DELAY;
          const delay =
            attemptNum === 1
              ? 0
              : Math.min(
                  baseDelay *
                    Math.pow(1.5, currentSession.reconnectAttempts - 1),
                  WS_RECONNECT_MAX_DELAY,
                );

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
            console.log(
              `[Terminal] Silent reconnect attempt ${attemptNum}/${WS_MAX_RECONNECT} for ${sessionId} (delay: ${delay}ms)`,
            );
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
              "\r\n\x1b[31m✖ Terminal session ended. Terminal may have been stopped or removed.\x1b[0m",
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
        if (
          currentSession?.isCollabSession &&
          currentSession?.collabMode === "view"
        ) {
          return;
        }

        // Filter out SGR mouse tracking sequences from input
        const mouseTrackingRegex = /\x1b\[<\d+;\d+;\d+[Mm]/g;
        const filteredData = data.replace(mouseTrackingRegex, "");
        if (!filteredData) return;

        // Send input immediately for responsiveness
        // Large pastes are chunked to avoid WebSocket message size limits
        if (filteredData.length > CHUNK_SIZE) {
          // Large paste - chunk it
          for (let i = 0; i < filteredData.length; i += CHUNK_SIZE) {
            inputQueue.push(filteredData.slice(i, i + CHUNK_SIZE));
          }
          processInputQueue();
        } else {
          // Normal input - send immediately (no buffering for instant feel)
          ws.send(JSON.stringify({ type: "input", data: filteredData }));
        }
      });

      // Handle binary data efficiently (for file transfers, etc.)
      session.terminal.onBinary((data) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify({ type: "input", data: data }));
      });
    },

    // Reset terminal state (fix garbled output)
    resetSession(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || !session.terminal) return;

      // Fit terminal first
      try {
        session.fitAddon.fit();
      } catch (e) {
        // Ignore fit errors
      }

      // Send full reset sequence
      session.terminal.write("\x1bc"); // Full reset
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

      updateSession(sessionId, (s) => ({
        ...s,
        reconnectAttempts: 0,
        status: "connecting",
      }));
      this.connectWebSocket(sessionId);
    },

    // Refresh container info and reconnect with potentially new container ID
    async refreshContainerAndReconnect(
      sessionId: string,
      oldContainerId: string,
    ) {
      const authToken = get(token);
      if (!authToken) return;

      try {
        // Start the container via API - this will recreate if needed and return new ID
        const response = await fetch(
          `/api/containers/${oldContainerId}/start`,
          {
            method: "POST",
            headers: {
              Authorization: `Bearer ${authToken}`,
            },
          },
        );

        if (response.ok) {
          const data = await response.json();
          const newContainerId = data.id || oldContainerId;

          if (newContainerId !== oldContainerId) {
            console.log(
              `[Terminal] Container recreated: ${oldContainerId} -> ${newContainerId}`,
            );
            // Update session with new container ID
            updateSession(sessionId, (s) => ({
              ...s,
              containerId: newContainerId,
              reconnectAttempts: 0,
              status: "connecting",
            }));
          } else {
            updateSession(sessionId, (s) => ({
              ...s,
              reconnectAttempts: 0,
              status: "connecting",
            }));
          }

          // Small delay to ensure container is ready
          await new Promise((resolve) => setTimeout(resolve, 500));

          // Reconnect
          this.connectWebSocket(sessionId);
        } else {
          console.error(
            "[Terminal] Failed to start container:",
            await response.text(),
          );
          const state = getState();
          const session = state.sessions.get(sessionId);
          if (session) {
            session.terminal.writeln(
              "\r\n\x1b[31m✖ Failed to restart container. Please try again.\x1b[0m",
            );
          }
          updateSession(sessionId, (s) => ({ ...s, status: "error" }));
        }
      } catch (e) {
        console.error("[Terminal] Error refreshing container:", e);
        updateSession(sessionId, (s) => ({ ...s, status: "error" }));
      }
    },

    // Update a session's container ID (used when container is recreated with new ID)
    updateSessionContainerId(oldContainerId: string, newContainerId: string) {
      const state = getState();

      // Find all sessions by old container ID
      const sessionsToUpdate: string[] = [];
      const splitPanesToReconnect: Array<{
        sessionId: string;
        paneId: string;
      }> = [];

      for (const [id, session] of state.sessions) {
        if (session.containerId === oldContainerId) {
          sessionsToUpdate.push(id);

          // Collect split panes to reconnect
          session.splitPanes.forEach((pane, paneId) => {
            if (pane.ws) {
              pane.ws.close();
            }
            splitPanesToReconnect.push({ sessionId: id, paneId });
          });

          // Close old WebSocket immediately
          if (session.ws) {
            session.ws.close();
          }
        }
      }

      if (sessionsToUpdate.length === 0) {
        console.log(
          `[Terminal] No session found for container ${oldContainerId}`,
        );
        return;
      }

      console.log(
        `[Terminal] Updating ${sessionsToUpdate.length} sessions from ${oldContainerId} to ${newContainerId}`,
      );

      // Update all sessions first with new container ID
      sessionsToUpdate.forEach((sessionId) => {
        updateSession(sessionId, (s) => ({
          ...s,
          containerId: newContainerId,
          status: "connecting",
          reconnectAttempts: 0,
        }));
      });

      // Use setTimeout to ensure state is updated before reconnecting
      setTimeout(() => {
        sessionsToUpdate.forEach((sessionId) => {
          this.connectWebSocket(sessionId);
        });

        // Reconnect split panes
        splitPanesToReconnect.forEach(({ sessionId, paneId }) => {
          this.connectSplitPaneWebSocket(sessionId, paneId);
        });
      }, 100);
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

      // Add paste event handler to ensure Cmd+V/Ctrl+V works correctly
      // xterm.js textarea sometimes doesn't receive paste events properly
      // Note: We use 'capture: true' and check if the terminal's textarea is focused
      // to avoid interfering with TUI applications that handle their own input
      element.addEventListener("paste", (e: ClipboardEvent) => {
        // Only handle paste if it's going to the terminal container, not internal TUI elements
        const target = e.target as HTMLElement;
        const isTerminalTextarea = target.classList.contains(
          "xterm-helper-textarea",
        );

        // If it's the xterm textarea, let xterm handle it naturally first
        // We only intervene if the paste didn't work (fallback)
        if (!isTerminalTextarea) {
          return; // Let the event propagate normally
        }

        const text = e.clipboardData?.getData("text");
        if (text && session.ws?.readyState === WebSocket.OPEN) {
          // Use the chunking logic for large pastes
          const CHUNK_SIZE = 4096;
          if (text.length > CHUNK_SIZE) {
            e.preventDefault();
            // Send in chunks with small delays
            const chunks: string[] = [];
            for (let i = 0; i < text.length; i += CHUNK_SIZE) {
              chunks.push(text.slice(i, i + CHUNK_SIZE));
            }
            let i = 0;
            const sendChunk = () => {
              if (
                i < chunks.length &&
                session.ws?.readyState === WebSocket.OPEN
              ) {
                session.ws.send(
                  JSON.stringify({ type: "input", data: chunks[i] }),
                );
                i++;
                if (i < chunks.length) {
                  setTimeout(sendChunk, 10);
                }
              }
            };
            sendChunk();
          }
          // For normal-sized pastes, let xterm.js handle it naturally
        }
      });

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

      // Add paste event handler to the new container
      // Only intercept large pastes, let xterm handle normal ones
      element.addEventListener("paste", (e: ClipboardEvent) => {
        const target = e.target as HTMLElement;
        const isTerminalTextarea = target.classList.contains(
          "xterm-helper-textarea",
        );

        if (!isTerminalTextarea) {
          return;
        }

        const text = e.clipboardData?.getData("text");
        if (text && session.ws?.readyState === WebSocket.OPEN) {
          const CHUNK_SIZE = 4096;
          if (text.length > CHUNK_SIZE) {
            e.preventDefault();
            const chunks: string[] = [];
            for (let i = 0; i < text.length; i += CHUNK_SIZE) {
              chunks.push(text.slice(i, i + CHUNK_SIZE));
            }
            let i = 0;
            const sendChunk = () => {
              if (
                i < chunks.length &&
                session.ws?.readyState === WebSocket.OPEN
              ) {
                session.ws.send(
                  JSON.stringify({ type: "input", data: chunks[i] }),
                );
                i++;
                if (i < chunks.length) {
                  setTimeout(sendChunk, 10);
                }
              }
            };
            sendChunk();
          }
        }
      });

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
    splitPane(
      sessionId: string,
      direction: SplitDirection = "horizontal",
    ): string | null {
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
      // Unicode11 addon for proper Unicode character widths
      const unicode11Addon = new Unicode11Addon();
      newTerminal.loadAddon(unicode11Addon);
      newTerminal.unicode.activeVersion = "11";

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

      // Create independent WebSocket connection for this pane with unique ID
      const wsUrl = `${getWsUrl()}/ws/terminal/${session.containerId}?token=${authToken}&id=${paneId}`;
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

      // Output buffer for batching writes - optimized like main terminal
      let outputBuffer = "";
      let flushTimeout: ReturnType<typeof setTimeout> | null = null;
      let rafId: number | null = null;
      let lastFlushTime = 0;

      const sanitizeOutput = (data: string): string => {
        return data.replace(/\x1b\[<\d+;\d+;\d+[Mm]/g, "");
      };

      const writeImmediate = (data: string) => {
        if (pane.terminal) {
          const sanitized = sanitizeOutput(data);
          if (sanitized) {
            pane.terminal.write(sanitized);
          }
        }
      };

      const flushBuffer = () => {
        if (outputBuffer && pane.terminal) {
          const sanitized = sanitizeOutput(outputBuffer);
          if (sanitized) {
            pane.terminal.write(sanitized);
          }
          outputBuffer = "";
          lastFlushTime = performance.now();
        }
        flushTimeout = null;
        rafId = null;
      };

      const scheduleFlush = () => {
        if (flushTimeout || rafId) return;
        const timeSinceLastFlush = performance.now() - lastFlushTime;
        if (timeSinceLastFlush > OUTPUT_FLUSH_INTERVAL) {
          rafId = requestAnimationFrame(flushBuffer);
        } else {
          flushTimeout = setTimeout(() => {
            rafId = requestAnimationFrame(flushBuffer);
          }, OUTPUT_FLUSH_INTERVAL);
        }
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === "output") {
            const data = msg.data as string;

            // Small outputs - write immediately for responsiveness
            if (data.length <= OUTPUT_IMMEDIATE_THRESHOLD) {
              writeImmediate(data);
              return;
            }

            outputBuffer += data;
            if (outputBuffer.length >= OUTPUT_MAX_BUFFER) {
              if (flushTimeout) clearTimeout(flushTimeout);
              if (rafId) cancelAnimationFrame(rafId);
              flushBuffer();
            } else {
              scheduleFlush();
            }
          } else if (msg.type === "error") {
            pane.terminal.writeln(`\r\n\x1b[31mError: ${msg.data}\x1b[0m`);
          } else if (msg.type === "ping") {
            ws.send(JSON.stringify({ type: "pong" }));
          }
        } catch {
          outputBuffer += event.data;
          if (outputBuffer.length >= OUTPUT_MAX_BUFFER) {
            if (flushTimeout) clearTimeout(flushTimeout);
            if (rafId) cancelAnimationFrame(rafId);
            flushBuffer();
          } else {
            scheduleFlush();
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
        const maxAttemptsReached =
          currentPane.reconnectAttempts >= WS_MAX_RECONNECT;

        const shouldReconnect =
          !isIntentionalClose && !isContainerGone && !maxAttemptsReached;

        if (shouldReconnect) {
          const attemptNum = currentPane.reconnectAttempts + 1;

          // First attempt is instant, then exponential backoff
          const baseDelay = WS_RECONNECT_BASE_DELAY;
          const delay =
            attemptNum === 1
              ? 0
              : Math.min(
                  baseDelay * Math.pow(1.5, currentPane.reconnectAttempts - 1),
                  WS_RECONNECT_MAX_DELAY,
                );

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
            console.log(
              `[Terminal] Split pane silent reconnect attempt ${attemptNum}/${WS_MAX_RECONNECT} for ${paneId} (delay: ${delay}ms)`,
            );
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
              "\r\n\x1b[31m✖ Split session ended. Terminal may have been stopped or removed.\x1b[0m",
            );
          } else if (maxAttemptsReached) {
            pane.terminal.writeln(
              "\r\n\x1b[31m✖ Split session connection lost after multiple attempts.\x1b[0m",
            );
          } else {
            pane.terminal.writeln(
              "\r\n\x1b[33m› Split session disconnected\x1b[0m",
            );
          }
        }
      };

      ws.onerror = () => {
        console.log(
          "[Terminal] Split pane WebSocket error - will attempt reconnect",
        );
      };

      // Handle terminal input for split pane
      // Only filter complete SGR mouse sequences
      const mouseTrackingRegex = /\x1b\[<\d+;\d+;\d+[Mm]/g;

      pane.terminal.onData((data) => {
        if (ws.readyState !== WebSocket.OPEN) return;

        let filteredData = data.replace(mouseTrackingRegex, "");
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
            direction:
              s.splitLayout.direction === "horizontal"
                ? "vertical"
                : "horizontal",
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
    if (
      session.containerId === containerId &&
      (session.status === "connected" || session.status === "connecting")
    ) {
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
