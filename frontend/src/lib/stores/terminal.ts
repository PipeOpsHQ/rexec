import { writable, derived, get } from "svelte/store";
import type { Terminal } from "@xterm/xterm";
import type { FitAddon } from "@xterm/addon-fit";
import type { WebglAddon } from "@xterm/addon-webgl";
import { token } from "./auth";
import { toast } from "./toast";
import { theme } from "./theme";
import { loadXtermCore, loadXtermWebgl } from "$utils/xterm";
import { createRexecWebSocket } from "$utils/ws";
import { trackEvent } from "$lib/analytics";

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
  status: "connecting" | "connected" | "disconnected" | "error";
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
  hasConnectedOnce: boolean; // Track if session has ever successfully connected
  // Agent session (external machine)
  isAgentSession: boolean;
  agentId: string | null;
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
    diskUsage?: number; // Real-time storage usage
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
const OUTPUT_FLUSH_INTERVAL = 8; // ~120fps for smooth output
const OUTPUT_IMMEDIATE_THRESHOLD = 256; // Immediately write small outputs
const OUTPUT_MAX_BUFFER = 32 * 1024; // 32KB max buffer before force flush

const REXEC_BANNER =
  "\x1b[38;5;46m\r\n" +
  "  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗\r\n" +
  "  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝\r\n" +
  "  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║\r\n" +
  "  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║\r\n" +
  "  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗\r\n" +
  "  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝\r\n" +
  "\x1b[0m\x1b[38;5;243m  Terminal as a Service · rexec.dev\x1b[0m\r\n" +
  "\x1b[38;5;243m  Run 'rexec tools' to see available tools\x1b[0m\r\n\r\n";

// Helper to handle custom key events (prevent browser defaults, map macOS keys)
function createCustomKeyHandler(term: Terminal) {
  return (event: KeyboardEvent): boolean => {
    if (event.type !== "keydown") return true;

    const platform =
      typeof navigator !== "undefined" ? navigator.platform || "" : "";
    const userAgent =
      typeof navigator !== "undefined" ? navigator.userAgent : "";
    const isMac =
      /Mac|iPod|iPhone|iPad/.test(platform) || /Macintosh/.test(userAgent);
    const isFirefox = /Firefox/.test(userAgent);
    const isSafari = /Safari/.test(userAgent) && !/Chrome/.test(userAgent);

    // Firefox-specific: Handle Backquote key (`) which Firefox sometimes mishandles
    if (
      isFirefox &&
      event.code === "Backquote" &&
      !event.ctrlKey &&
      !event.metaKey &&
      !event.altKey
    ) {
      return true; // Let xterm handle it
    }

    // Prevent browser defaults for common terminal shortcuts on ALL platforms
    // Ctrl+S (save/xoff), Ctrl+P (print/up), Ctrl+F (find/forward), Ctrl+R (refresh/search), etc.
    if (event.ctrlKey && !event.altKey && !event.metaKey && !event.shiftKey) {
      const key = event.key.toLowerCase();
      // Block browser defaults for keys essential to terminal usage
      const preventedKeys = [
        "s",
        "p",
        "f",
        "r",
        "o",
        "g",
        "b",
        "h",
        "i",
        "u",
        "l",
        "k",
        "j",
      ];
      if (preventedKeys.includes(key)) {
        event.preventDefault();
        // Return true to let xterm process it
        return true;
      }

      // Ctrl+C: Special handling - copy if there's terminal selection, else interrupt
      if (key === "c") {
        // Check terminal's own selection (not window.getSelection)
        if (term.hasSelection()) {
          // Has selection in terminal - copy to clipboard
          const selection = term.getSelection();
          if (selection) {
            navigator.clipboard.writeText(selection).catch(() => {});
          }
          event.preventDefault();
          return false;
        }
        // No selection - this is Ctrl+C for interrupt, let terminal handle it
        event.preventDefault();
        return true;
      }
    }

    // Safari-specific: Prevent default on certain key combos that Safari intercepts
    if (isSafari) {
      if (
        event.ctrlKey &&
        ["a", "e", "k", "u", "w"].includes(event.key.toLowerCase())
      ) {
        event.preventDefault();
        return true;
      }
    }

    // macOS specific handling: Map Cmd+Key -> Ctrl+Key
    if (
      isMac &&
      event.metaKey &&
      !event.ctrlKey &&
      !event.altKey &&
      !event.shiftKey
    ) {
      const key = event.key.toLowerCase();

      // Cmd+C: Copy if terminal has selection, else send Ctrl+C interrupt
      if (key === "c") {
        if (term.hasSelection()) {
          // Has selection in terminal - copy to clipboard
          const selection = term.getSelection();
          if (selection) {
            navigator.clipboard.writeText(selection).catch(() => {});
          }
          event.preventDefault();
          return false;
        }
        // No selection - send Ctrl+C to interrupt running process
        event.preventDefault();
        event.stopPropagation();
        term.input("\x03"); // Ctrl+C
        return false;
      }

      // Preserve standard clipboard shortcuts (Cmd+V, Cmd+X, Cmd+A)
      if (key === "v" || key === "x" || key === "a") return true;

      // Ghostty-style shortcuts to pass through to UI handler (TerminalView.svelte)
      // d = split, t = tab, w = close, n = new window
      const reserved = ["d", "t", "w", "n", "enter"];
      if (reserved.includes(key)) {
        return false;
      }

      // Prevent Cmd+Q (quit browser) from being sent to terminal
      if (key === "q") {
        return false;
      }

      // Map A-Z keys: Cmd+X -> Ctrl+X
      if (key.length === 1 && key >= "a" && key <= "z") {
        // Calculate control character (a=1, z=26)
        const charCode = key.charCodeAt(0) - 96;

        // Prevent default browser actions (e.g., Cmd+S, Cmd+F, Cmd+P)
        event.preventDefault();
        event.stopPropagation();

        // Send control character to terminal
        term.input(String.fromCharCode(charCode));

        // Stop xterm from processing the original event
        return false;
      }
    }

    // Windows/Linux: Handle Alt key combinations that might conflict with browser
    if (!isMac && event.altKey && !event.ctrlKey && !event.metaKey) {
      // Alt+F4, Alt+Tab etc should go to OS, not terminal
      if (["F4", "Tab"].includes(event.key)) {
        return false;
      }
      // Let terminal handle other Alt combos (for readline alt-b, alt-f, etc.)
      return true;
    }

    return true;
  };
}

// Calculate responsive font size based on screen dimensions
function getResponsiveFontSize(): number {
  if (typeof window === "undefined") return 11;

  const width = window.innerWidth;

  // Scale font size based on screen size - compact for better fit
  // Small screens (< 768px): 10px
  // Medium screens (768-1200px): 11px
  // Large screens (1200-1600px): 12px
  // Extra large (> 1600px): 12px
  if (width < 768) return 10;
  if (width < 1200) return 11;
  return 12;
}

// Current zoom level (persisted font size)
let currentFontSize = getResponsiveFontSize();
const MIN_FONT_SIZE = 8;
const MAX_FONT_SIZE = 24;
const ZOOM_STEP = 1;

// Terminal themes for dark and light mode
const DARK_TERMINAL_THEME = {
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
};

const LIGHT_TERMINAL_THEME = {
  background: "#ffffff",
  foreground: "#1a1a1a",
  cursor: "#00a830",
  cursorAccent: "#ffffff",
  selectionBackground: "rgba(0, 168, 48, 0.3)",
  selectionForeground: "#1a1a1a",
  selectionInactiveBackground: "rgba(0, 168, 48, 0.15)",
  black: "#1a1a1a",
  red: "#dc2626",
  green: "#00a830",
  yellow: "#ca8a04",
  blue: "#2563eb",
  magenta: "#9333ea",
  cyan: "#0891b2",
  white: "#e5e5e5",
  brightBlack: "#737373",
  brightRed: "#ef4444",
  brightGreen: "#22c55e",
  brightYellow: "#eab308",
  brightBlue: "#3b82f6",
  brightMagenta: "#a855f7",
  brightCyan: "#06b6d4",
  brightWhite: "#ffffff",
};

// Get current terminal theme based on app theme
function getCurrentTerminalTheme() {
  const currentTheme = get(theme);
  return currentTheme === "light" ? LIGHT_TERMINAL_THEME : DARK_TERMINAL_THEME;
}

// Terminal configuration - Production-grade like Google Cloud Shell
const TERMINAL_OPTIONS = {
  cursorBlink: true,
  cursorStyle: "bar" as const,
  cursorWidth: 2,
  fontSize: currentFontSize,
  fontFamily:
    '"JetBrains Mono", "Fira Code", "SF Mono", Menlo, Monaco, "Courier New", monospace',
  fontWeight: "400" as const,
  fontWeightBold: "600" as const,
  letterSpacing: 0,
  lineHeight: 1.2,
  theme: getCurrentTerminalTheme(),
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
    async createSession(
      containerId: string,
      name: string,
    ): Promise<string | null> {
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
      return await this.createNewTab(containerId, sessionName);
    },

    // Create a new tab (always creates a new session, even for same container)
    // OPTIMIZED: Starts WebSocket connection in parallel with xterm loading for instant terminal access
    async createNewTab(
      containerId: string,
      name: string,
    ): Promise<string | null> {
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

      // Generate session ID immediately so we can start WebSocket connection in parallel
      const sessionId = generateSessionId();
      const tabName =
        existingCount > 0 ? `${validName} (${existingCount + 1})` : validName;

      // Create session placeholder with null terminal - will be populated after xterm loads
      // This allows WebSocket to connect immediately while xterm loads in parallel
      const session: TerminalSession = {
        id: sessionId,
        containerId,
        name: tabName,
        terminal: null as unknown as Terminal, // Temporarily null, set after xterm loads
        fitAddon: null as unknown as FitAddon, // Temporarily null, set after xterm loads
        webglAddon: null,
        ws: null,
        status: "connecting",
        reconnectAttempts: 0,
        reconnectTimer: null,
        pingInterval: null,
        resizeObserver: null,
        isSettingUp: false,
        setupMessage: "",
        hasConnectedOnce: false,
        isAgentSession: false,
        agentId: null,
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

      // Add session to state immediately so UI updates
      update((state) => {
        const newSessions = new Map(state.sessions);
        newSessions.set(sessionId, session);
        return {
          ...state,
          sessions: newSessions,
          activeSessionId: sessionId,
          isMinimized: false,
        };
      });

      // Update URL immediately
      this.updateUrl(containerId);

      // Start WebSocket connection AND xterm loading in parallel
      // WebSocket connects immediately; xterm loads in background
      const connectWebSocketAsync = () => {
        // Connect WebSocket immediately - messages will be buffered until terminal is ready
        this.connectWebSocket(sessionId);
      };

      const loadXtermAsync = async () => {
        // Load xterm modules
        let XtermTerminal: (typeof import("@xterm/xterm"))["Terminal"];
        let XtermFitAddon: (typeof import("@xterm/addon-fit"))["FitAddon"];
        let XtermUnicode11Addon: (typeof import("@xterm/addon-unicode11"))["Unicode11Addon"];
        let XtermWebLinksAddon: (typeof import("@xterm/addon-web-links"))["WebLinksAddon"];
        try {
          ({
            Terminal: XtermTerminal,
            FitAddon: XtermFitAddon,
            Unicode11Addon: XtermUnicode11Addon,
            WebLinksAddon: XtermWebLinksAddon,
          } = await loadXtermCore());
        } catch (e) {
          console.error("[Terminal] Failed to load xterm:", e);
          toast.error("Failed to load terminal. Please refresh and try again.");
          // Close the session on failure
          this.closeSession(sessionId);
          return false;
        }

        let terminal: Terminal;
        let fitAddon: FitAddon;
        try {
          // Ensure we use the current theme state, not the module-load-time state
          const currentOptions = {
            ...TERMINAL_OPTIONS,
            theme: getCurrentTerminalTheme(),
          };
          terminal = new XtermTerminal(currentOptions);
          fitAddon = new XtermFitAddon();
          terminal.loadAddon(fitAddon);
          terminal.loadAddon(new XtermWebLinksAddon());
          // Unicode11 addon for proper Unicode character widths (emojis, CJK, etc.)
          const unicode11Addon = new XtermUnicode11Addon();
          terminal.loadAddon(unicode11Addon);
          terminal.unicode.activeVersion = "11";

          // Enable mouse events for TUI apps (opencode, vim, tmux, etc.)
          // DECSET 1000: X10 mouse mode (basic)
          // DECSET 1002: Cell motion mouse tracking
          // DECSET 1003: All motion mouse tracking (needed for opencode)
          // DECSET 1006: SGR mouse mode (preferred, more accurate)
          // DECSET 1007: Alternate scroll mode
          // DECSET 1015: URXVT mouse mode (fallback)
          terminal.options.mouseSupport = true;
          // Enable all mouse modes for maximum compatibility
          terminal.write("\x1b[?1000h"); // X10 mouse mode
          terminal.write("\x1b[?1002h"); // Cell motion mouse tracking
          terminal.write("\x1b[?1003h"); // All motion mouse tracking (for opencode)
          terminal.write("\x1b[?1006h"); // SGR mouse mode (preferred)
          terminal.write("\x1b[?1007h"); // Alternate scroll mode
          terminal.write("\x1b[?1015h"); // URXVT mouse mode (fallback)

          // Attach custom key handler (browser overrides + macOS mapping)
          terminal.attachCustomKeyEventHandler(
            createCustomKeyHandler(terminal),
          );
        } catch (e) {
          console.error("[Terminal] Failed to initialize terminal:", e);
          toast.error("Failed to initialize terminal. Please try again.");
          this.closeSession(sessionId);
          return false;
        }

        // Update session with the real terminal and fitAddon
        updateSession(sessionId, (s) => ({
          ...s,
          terminal,
          fitAddon,
        }));

        return true;
      };

      // Start both operations in parallel - WebSocket first (instant), then xterm (may take time)
      connectWebSocketAsync();

      // Load xterm in background - don't await here to return sessionId immediately
      loadXtermAsync().catch((e) => {
        console.error("[Terminal] Xterm load failed:", e);
      });

      return sessionId;
    },

    // Create a collab session (joined via share link)
    async createCollabSession(
      containerId: string,
      name: string,
      mode: "view" | "control",
      role: "owner" | "editor" | "viewer",
    ): Promise<string | null> {
      const sessionId = await this.createSession(containerId, name);
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

    // Create an agent session (external machine connected via agent)
    async createAgentSession(
      agentId: string,
      name: string,
    ): Promise<string | null> {
      const authToken = get(token);
      if (!authToken) return null;

      // Use agentId as the containerId for consistency
      const sessionId = await this.createNewTab(
        `agent:${agentId}`,
        `[Agent] ${name}`,
      );
      if (!sessionId) return null;

      // Mark as agent session
      updateSession(sessionId, (s) => ({
        ...s,
        isAgentSession: true,
        agentId: agentId,
      }));

      return sessionId;
    },

    // Update browser URL for terminal routing
    updateUrl(containerId: string) {
      const newUrl = `/terminal/${containerId}`;
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
    // OPTIMIZED: Handles case where terminal may not be loaded yet (parallel loading)
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

      // Build WebSocket URL - different endpoint for agent sessions
      const agentId =
        session.agentId ||
        (session.containerId.startsWith("agent:")
          ? session.containerId.slice("agent:".length)
          : null);
      let wsUrl: string;
      if (session.isAgentSession || agentId) {
        if (!agentId) return;
        wsUrl = `${getWsUrl()}/ws/agent/${encodeURIComponent(agentId)}/terminal?id=${encodeURIComponent(sessionId)}`;
      } else {
        wsUrl = `${getWsUrl()}/ws/terminal/${encodeURIComponent(session.containerId)}?id=${encodeURIComponent(sessionId)}`;
      }
      const ws = createRexecWebSocket(wsUrl, authToken);

      updateSession(sessionId, (s) => ({ ...s, ws, status: "connecting" }));

      // Helper to get current session state (terminal may be loaded after WS connects)
      const getCurrentSession = () => getState().sessions.get(sessionId);

      // Helper to wait for terminal to be ready (with timeout)
      // Increased timeout for mobile networks which may be slower to load xterm modules
      const isMobile =
        /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
          navigator.userAgent,
        );
      const defaultMaxWait = isMobile ? 15000 : 8000; // 15s for mobile, 8s for desktop

      const waitForTerminal = (
        callback: (terminal: Terminal) => void,
        maxWait = defaultMaxWait,
      ) => {
        const startTime = Date.now();
        const check = () => {
          const currentSession = getCurrentSession();
          if (currentSession?.terminal) {
            callback(currentSession.terminal);
          } else if (Date.now() - startTime < maxWait) {
            // Check again in 50ms
            setTimeout(check, 50);
          } else {
            console.warn("[Terminal] Timeout waiting for terminal to be ready");
          }
        };
        check();
      };

      ws.onopen = () => {
        // Check if this is a reconnect (hasConnectedOnce tracks if we've ever connected)
        const currentSession = getCurrentSession();
        const isReconnect = currentSession?.hasConnectedOnce === true;

        // Set to connected immediately - shell setup happens in background
        updateSession(sessionId, (s) => ({
          ...s,
          status: "connected",
          reconnectAttempts: 0,
          hasConnectedOnce: true,
        }));

        // Track terminal connection
        trackEvent("terminal_connected", {
          containerId: currentSession?.containerId,
          sessionId: sessionId,
          isReconnect: isReconnect,
          isAgent: currentSession?.isAgentSession || false,
        });

        // Wait for terminal to be ready, then do terminal-specific initialization
        waitForTerminal((terminal) => {
          const sess = getCurrentSession();
          if (!sess) return;

          // Fit terminal first to get accurate dimensions
          try {
            sess.fitAddon?.fit();
          } catch (e) {
            // Ignore fit errors on initial connection
          }

          // Send resize with correct dimensions
          ws.send(
            JSON.stringify({
              type: "resize",
              cols: terminal.cols || 80,
              rows: terminal.rows || 24,
            }),
          );

          if (isReconnect) {
            // On reconnect, don't clear/reset - just send Ctrl+L to refresh the display
            // This preserves TUI apps like opencode that are running in tmux
            terminal.writeln("\r\n\x1b[32m› Reconnected\x1b[0m\r\n");
            // Send Ctrl+L (form feed) to trigger a screen redraw in tmux/shell
            ws.send(JSON.stringify({ type: "input", data: "\x0c" }));
          } else {
            // First connect - clear terminal and show banner
            terminal.clear();
            // Reset terminal state: clear screen, reset attributes, show cursor
            terminal.write("\x1b[0m\x1b[?25h\x1b[H\x1b[2J");
            terminal.write(REXEC_BANNER);
            terminal.writeln("\x1b[32m› Connected\x1b[0m");
            terminal.writeln(
              "\x1b[38;5;243m  Type 'help' for tips & shortcuts · Ctrl+C to interrupt\x1b[0m",
            );
            // Don't show "Starting shell..." here - wait for backend to send shell_starting message
          }
        });

        // Send default resize immediately even if terminal isn't ready
        // (ensures backend doesn't hang waiting for resize)
        const sess = getCurrentSession();
        if (!sess?.terminal) {
          ws.send(
            JSON.stringify({
              type: "resize",
              cols: 80,
              rows: 24,
            }),
          );
        }

        // Setup ping interval
        const pingInterval = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "ping" }));
          }
        }, WS_PING_INTERVAL);

        updateSession(sessionId, (s) => ({ ...s, pingInterval }));
      };

      // Output buffer for batching writes - optimized for smooth rendering
      // Also buffers output when terminal isn't ready yet
      let outputBuffer = "";
      let flushTimeout: ReturnType<typeof setTimeout> | null = null;
      let rafId: number | null = null;
      let lastFlushTime = 0;
      let hasReceivedFirstOutput = false;

      // Filter problematic escape sequences from output to prevent artifacts
      const sanitizeOutput = (data: string): string => {
        // Filter mouse tracking sequences
        let result = data.replace(/\x1b\[<\d+;\d+;\d+[Mm]/g, "");
        // Filter OSC (Operating System Command) query responses that leak into input
        // These are sequences like ESC ] <number> ; <data> BEL or ESC ] <number> ; <data> ESC \
        // Common ones: OSC 10/11 (foreground/background color queries)
        result = result.replace(/\x1b\]\d+;[^\x07\x1b]*(?:\x07|\x1b\\)/g, "");
        // Also filter partial/malformed OSC sequences that might appear at boundaries
        result = result.replace(/\]\d+;rgb:[0-9a-f/]+/gi, "");
        return result;
      };

      // Immediate write for small, interactive output (like keystrokes)
      const writeImmediate = (data: string) => {
        const currentSession = getCurrentSession();
        if (currentSession?.terminal) {
          // Clear "Starting shell..." on first output
          if (!hasReceivedFirstOutput) {
            hasReceivedFirstOutput = true;
            // Clear line and move to start, then write newline
            currentSession.terminal.write("\r\x1b[K\r\n");
          }
          const sanitized = sanitizeOutput(data);
          if (sanitized) {
            currentSession.terminal.write(sanitized);
          }
        } else {
          // Terminal not ready - buffer the data
          outputBuffer += data;
        }
      };

      // Flush buffer using requestAnimationFrame for smooth rendering
      const flushBuffer = () => {
        const currentSession = getCurrentSession();
        if (outputBuffer && currentSession?.terminal) {
          // Clear "Starting shell..." on first output
          if (!hasReceivedFirstOutput) {
            hasReceivedFirstOutput = true;
            // Clear line and move to start, then write newline
            currentSession.terminal.write("\r\x1b[K\r\n");
          }
          const sanitized = sanitizeOutput(outputBuffer);
          if (sanitized) {
            currentSession.terminal.write(sanitized);
          }
          outputBuffer = "";
          lastFlushTime = performance.now();
        } else if (outputBuffer && !currentSession?.terminal) {
          // Terminal still not ready - reschedule flush
          setTimeout(flushBuffer, 50);
          return;
        }
        flushTimeout = null;
        rafId = null;
      };

      // Schedule buffer flush - uses RAF for smooth rendering with timeout fallback
      // The timeout guarantees flush even when RAF is throttled (inactive tab, etc.)
      const scheduleFlush = () => {
        if (flushTimeout) return;

        const timeSinceLastFlush = performance.now() - lastFlushTime;

        // If we haven't flushed recently, try RAF for smooth render
        if (timeSinceLastFlush > OUTPUT_FLUSH_INTERVAL) {
          // Use RAF but with a guaranteed timeout fallback
          rafId = requestAnimationFrame(() => {
            if (flushTimeout) {
              clearTimeout(flushTimeout);
              flushTimeout = null;
            }
            flushBuffer();
          });
          // Fallback timeout in case RAF doesn't fire (throttled tab, etc.)
          flushTimeout = setTimeout(() => {
            if (rafId) {
              cancelAnimationFrame(rafId);
              rafId = null;
            }
            flushBuffer();
          }, 50); // 50ms max delay guarantees prompt appears
        } else {
          // Schedule for next frame with guaranteed timeout
          flushTimeout = setTimeout(() => {
            flushBuffer();
          }, OUTPUT_FLUSH_INTERVAL);
        }
      };

      ws.onmessage = async (event) => {
        // Handle Blob data from WebSocket (common with agent connections)
        let eventData = event.data;
        if (eventData instanceof Blob) {
          eventData = await eventData.text();
        }

        try {
          const msg = JSON.parse(eventData);
          if (msg.type === "output") {
            const data = msg.data as string;

            // Check for explicit status updates from backend script
            const statusMatch = data.match(/\[\[REXEC_STATUS\]\](.*)/);
            if (statusMatch) {
              const statusMsg = statusMatch[1].trim();
              if (statusMsg === "Setup complete.") {
                // Setup complete - clear status and refresh terminal immediately
                updateSession(sessionId, (s) => ({
                  ...s,
                  isSettingUp: false,
                  setupMessage: "",
                }));

                // Auto-reload: send Enter to get a fresh prompt with tools loaded
                const currentSession = getCurrentSession();
                if (
                  currentSession?.ws?.readyState === WebSocket.OPEN &&
                  currentSession.terminal
                ) {
                  // Clear terminal and show refreshed state
                  currentSession.terminal.writeln(
                    "\r\n\x1b[32m✓ Environment ready! Press Enter for a fresh prompt.\x1b[0m\r\n",
                  );
                  // Send Enter key to trigger a fresh shell prompt
                  currentSession.ws.send(
                    JSON.stringify({ type: "input", data: "\n" }),
                  );
                }
              } else {
                updateSession(sessionId, (s) => ({
                  ...s,
                  isSettingUp: true,
                  setupMessage: statusMsg,
                }));
              }
              // Don't show the internal status tag line in the terminal if possible,
              // but since it's mixed with other output, we might leave it or filter it.
              // Filtering it from the buffer prevents it from showing in the terminal.
              // Let's filter it out from the display output.
              const cleanData = data.replace(/\[\[REXEC_STATUS\]\].*\n?/g, "");
              if (cleanData) {
                // Buffer the rest
                outputBuffer += cleanData;
                if (outputBuffer.length >= OUTPUT_MAX_BUFFER) {
                  if (flushTimeout) clearTimeout(flushTimeout);
                  if (rafId) cancelAnimationFrame(rafId);
                  flushBuffer();
                } else {
                  scheduleFlush();
                }
              }
              return;
            }

            // Small interactive output (single chars, escape sequences) - write immediately
            // This makes typing feel instant like a native terminal
            if (data.length <= OUTPUT_IMMEDIATE_THRESHOLD) {
              writeImmediate(data);
              return;
            }

            // Check for setup/installation indicators (fallback)
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

              // Clear immediately - setup message auto-clears when setup completes
              // No artificial delay needed
              updateSession(sessionId, (s) => ({
                ...s,
                isSettingUp: false,
                setupMessage: "",
              }));
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
            const currentSession = getCurrentSession();
            if (currentSession?.terminal) {
              currentSession.terminal.writeln(
                `\r\n\x1b[31mError: ${msg.data}\x1b[0m`,
              );
            }
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
          } else if (
            msg.type === "shell_ready" ||
            msg.type === "shell_started"
          ) {
            // Shell is now ready for input - update status to connected
            // shell_ready: sent by container terminals after exec attach
            // shell_started: sent by agent terminals after PTY is ready
            updateSession(sessionId, (s) => ({
              ...s,
              status: "connected",
            }));
          } else if (msg.type === "stats") {
            // Handle stats updates
            try {
              // Stats can come as string (from container) or object (from agent)
              const statsData =
                typeof msg.data === "string" ? JSON.parse(msg.data) : msg.data;
              updateSession(sessionId, (s) => ({
                ...s,
                stats: {
                  cpu: statsData.cpu_percent || 0,
                  memory: statsData.memory || 0,
                  memoryLimit: statsData.memory_limit || 0,
                  diskRead: statsData.disk_read || 0,
                  diskWrite: statsData.disk_write || 0,
                  diskUsage: statsData.disk_usage || 0,
                  diskLimit: statsData.disk_limit || 0,
                  netRx: statsData.net_rx || 0,
                  netTx: statsData.net_tx || 0,
                },
              }));
            } catch (e) {
              console.error("Failed to parse stats:", e);
            }
          } else if (msg.type === "container_status") {
            // Container status update (e.g., configuring -> running)
            // This helps with multi-replica scenarios and status refresh
            const newStatus = msg.data as string;
            // Update container status in containers store if available
            import("./containers")
              .then(({ containers }) => {
                // Find container by ID and update status
                containers.update((state) => {
                  const containerIndex = state.containers.findIndex(
                    (c) =>
                      c.id === session.containerId ||
                      c.db_id === session.containerId,
                  );
                  if (containerIndex >= 0) {
                    const container = state.containers[containerIndex];
                    if (container.status !== newStatus) {
                      const updatedContainers = [...state.containers];
                      updatedContainers[containerIndex] = {
                        ...container,
                        status: newStatus as any,
                      };
                      return { ...state, containers: updatedContainers };
                    }
                  }
                  return state;
                });
              })
              .catch(() => {
                // Containers store might not be loaded, ignore
              });
          } else if (msg.type === "shell_starting") {
            // Show "Starting shell..." right before actual shell startup (sent from backend)
            // This will be cleared when first output arrives
            const currentSession = getCurrentSession();
            if (currentSession?.terminal) {
              currentSession.terminal.write(
                "\x1b[38;5;243m› Starting shell...\x1b[0m",
              );
            }
          } else if (msg.type === "shell_started") {
            // Shell is ready - no action needed, output will follow
          } else if (msg.type === "shell_stopped") {
            const currentSession = getCurrentSession();
            if (currentSession?.terminal) {
              currentSession.terminal.writeln(
                "\r\n\x1b[33m› Shell session ended\x1b[0m",
              );
            }
          } else if (msg.type === "shell_error") {
            const errorData = msg.data as { error?: string } | undefined;
            const errorMsg = errorData?.error || "Shell error";
            const currentSession = getCurrentSession();
            if (currentSession?.terminal) {
              currentSession.terminal.writeln(
                `\r\n\x1b[31m› Error: ${errorMsg}\x1b[0m`,
              );
            }
          }
        } catch {
          // Raw data fallback - also buffer this
          outputBuffer += eventData;
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
          if (currentSession.terminal) {
            currentSession.terminal.writeln(
              "\r\n\x1b[33m⟳ Container restarting, reconnecting...\x1b[0m",
            );
          }

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
          if (
            attemptNum > WS_SILENT_RECONNECT_THRESHOLD &&
            currentSession.terminal
          ) {
            currentSession.terminal.writeln(
              `\r\n\x1b[33m⟳ Reconnecting (${attemptNum}/${WS_MAX_RECONNECT})...\x1b[0m`,
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
            if (currentSession.terminal) {
              currentSession.terminal.writeln(
                "\r\n\x1b[31m✖ Terminal session ended. Terminal may have been stopped or removed.\x1b[0m",
              );
            }
            updateSession(sessionId, (s) => ({ ...s, status: "error" }));
          } else if (maxAttemptsReached) {
            if (currentSession.terminal) {
              currentSession.terminal.writeln(
                "\r\n\x1b[31m✖ Connection lost after multiple attempts. Click \x1b[33m⟳\x1b[31m to reconnect.\x1b[0m",
              );
            }
            updateSession(sessionId, (s) => ({ ...s, status: "error" }));
          }
          // If intentional close (code 1000), don't show any message
        }
      };

      ws.onerror = () => {
        // WebSocket errors are handled by onclose - just log silently
      };

      // Handle terminal input with chunking for large pastes
      let inputQueue: string[] = [];
      let isProcessingQueue = false;
      const CHUNK_SIZE = 32768; // 32KB chunks for large pastes
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

      // Attach input handlers once terminal is ready
      // This is deferred because terminal may be null during parallel loading
      waitForTerminal((terminal) => {
        terminal.onData((data) => {
          if (ws.readyState !== WebSocket.OPEN) return;

          // Block input if this is a view-only collab session
          const currentSession = getState().sessions.get(sessionId);
          if (
            currentSession?.isCollabSession &&
            currentSession?.collabMode === "view"
          ) {
            return;
          }

          // Send input immediately for responsiveness
          // Large pastes are chunked to avoid WebSocket message size limits
          if (data.length > CHUNK_SIZE) {
            // Large paste - chunk it
            for (let i = 0; i < data.length; i += CHUNK_SIZE) {
              inputQueue.push(data.slice(i, i + CHUNK_SIZE));
            }
            processInputQueue();
          } else {
            // Normal input - send immediately (no buffering for instant feel)
            ws.send(JSON.stringify({ type: "input", data: data }));
          }
        });

        // Handle binary data efficiently (for file transfers, etc.)
        terminal.onBinary((data) => {
          if (ws.readyState !== WebSocket.OPEN) return;
          ws.send(JSON.stringify({ type: "input", data: data }));
        });
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

          // Connect immediately - backend sends container_id as soon as it's ready
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
        return;
      }

      // Update all sessions first with new container ID
      sessionsToUpdate.forEach((sessionId) => {
        updateSession(sessionId, (s) => ({
          ...s,
          containerId: newContainerId,
          status: "connecting",
          reconnectAttempts: 0,
        }));
      });

      // Connect immediately - state updates are synchronous
      sessionsToUpdate.forEach((sessionId) => {
        this.connectWebSocket(sessionId);
      });

      // Reconnect split panes
      splitPanesToReconnect.forEach(({ sessionId, paneId }) => {
        this.connectSplitPaneWebSocket(sessionId, paneId);
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
      this.setFontSize(newFontSize);
    },

    // Set specific font size for all terminals
    setFontSize(newFontSize: number) {
      currentFontSize = Math.max(
        MIN_FONT_SIZE,
        Math.min(MAX_FONT_SIZE, newFontSize),
      );
      const state = getState();
      state.sessions.forEach((session) => {
        try {
          session.terminal.options.fontSize = currentFontSize;
          session.fitAddon.fit();
          // Also update split panes
          session.splitPanes.forEach((pane) => {
            pane.terminal.options.fontSize = currentFontSize;
            pane.fitAddon.fit();
          });
        } catch (e) {
          console.error("Failed to update font size:", e);
        }
      });
    },

    // Zoom in (increase font size)
    zoomIn() {
      this.setFontSize(currentFontSize + ZOOM_STEP);
    },

    // Zoom out (decrease font size)
    zoomOut() {
      this.setFontSize(currentFontSize - ZOOM_STEP);
    },

    // Reset zoom to default
    resetZoom() {
      this.setFontSize(getResponsiveFontSize());
    },

    // Get current font size
    getFontSize(): number {
      return currentFontSize;
    },

    // Update terminal theme (called when app theme changes)
    updateTheme(isDarkMode: boolean) {
      const newTheme = isDarkMode ? DARK_TERMINAL_THEME : LIGHT_TERMINAL_THEME;
      const state = getState();

      // Update all terminal sessions
      state.sessions.forEach((session) => {
        session.terminal.options.theme = newTheme;
        // Force terminal to refresh with new theme
        session.terminal.refresh(0, session.terminal.rows - 1);

        // Update split panes as well
        session.splitPanes.forEach((pane) => {
          pane.terminal.options.theme = newTheme;
          pane.terminal.refresh(0, pane.terminal.rows - 1);
        });
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
          const CHUNK_SIZE = 32768;
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

      // Try to load WebGL addon for GPU-accelerated rendering (best-effort)
      // This significantly improves performance with large outputs.
      loadXtermWebgl()
        .then(({ WebglAddon: XtermWebglAddon }) => {
          const current = getState().sessions.get(sessionId);
          if (!current) return;
          if (current.webglAddon) return;

          try {
            const webglAddon = new XtermWebglAddon();
            webglAddon.onContextLoss(() => {
              webglAddon.dispose();
              updateSession(sessionId, (s) => ({ ...s, webglAddon: null }));
            });
            current.terminal.loadAddon(webglAddon);
            updateSession(sessionId, (s) => ({ ...s, webglAddon }));
          } catch (e) {
            console.warn(
              "WebGL addon not available, using canvas renderer:",
              e,
            );
          }
        })
        .catch(() => {
          // Ignore (WebGL module may not be supported in some environments)
        });

      // Setup resize observer with debouncing to prevent rapid resize events
      if (window.ResizeObserver) {
        let resizeTimeout: ReturnType<typeof setTimeout> | null = null;
        const resizeObserver = new ResizeObserver(() => {
          // Debounce resize events to prevent terminal corruption
          if (resizeTimeout) clearTimeout(resizeTimeout);
          resizeTimeout = setTimeout(() => {
            this.fitSession(sessionId);
          }, 50); // 50ms debounce
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
          const CHUNK_SIZE = 32768;
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

      // Setup new resize observer with debouncing
      if (window.ResizeObserver) {
        let resizeTimeout: ReturnType<typeof setTimeout> | null = null;
        const resizeObserver = new ResizeObserver(() => {
          // Debounce resize events to prevent terminal corruption
          if (resizeTimeout) clearTimeout(resizeTimeout);
          resizeTimeout = setTimeout(() => {
            this.fitSession(sessionId);
          }, 50); // 50ms debounce
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

    // Send Ctrl+C to the active terminal
    sendCtrlC(sessionId: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || !session.ws || session.ws.readyState !== WebSocket.OPEN)
        return;

      // Ctrl+C is ASCII 0x03
      session.ws.send(JSON.stringify({ type: "input", data: "\x03" }));
    },

    // Send text input to a session (for snippets/macros)
    sendInput(sessionId: string, data: string) {
      const state = getState();
      const session = state.sessions.get(sessionId);

      if (session && session.ws && session.ws.readyState === WebSocket.OPEN) {
        session.ws.send(JSON.stringify({ type: "input", data: data }));
      }
    },

    // Navigate between split panes
    navigateSplitPanes(
      sessionId: string,
      direction: "left" | "right" | "up" | "down",
    ) {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session || !session.splitLayout || session.splitPanes.size === 0)
        return;

      const currentActivePaneId = session.activePaneId || "main";
      const currentLayout = session.splitLayout;
      const panes = currentLayout.panes; // Cache for easier access
      const currentIndex = panes.indexOf(currentActivePaneId);

      if (currentIndex === -1) {
        console.warn(
          `[Terminal] Active pane ${currentActivePaneId} not found in layout.`,
        );
        return;
      }

      let nextIndex = currentIndex;

      // Determine next index based on direction and layout
      // For horizontal splits, left/right navigate along the row
      // For vertical splits, up/down navigate along the column
      if (currentLayout.direction === "horizontal") {
        if (direction === "left") {
          nextIndex = (currentIndex - 1 + panes.length) % panes.length;
        } else if (direction === "right") {
          nextIndex = (currentIndex + 1) % panes.length;
        }
      } else {
        // Default to vertical if not explicitly horizontal
        if (direction === "up") {
          nextIndex = (currentIndex - 1 + panes.length) % panes.length;
        } else if (direction === "down") {
          nextIndex = (currentIndex + 1) % panes.length;
        }
      }

      const nextActivePaneId = panes[nextIndex];

      if (nextActivePaneId && nextActivePaneId !== currentActivePaneId) {
        updateSession(sessionId, (s) => ({
          ...s,
          activePaneId: nextActivePaneId,
        }));
        // Focus the new active terminal
        requestAnimationFrame(() => {
          if (nextActivePaneId === "main") {
            session.terminal.focus();
          } else {
            const nextPane = session.splitPanes.get(nextActivePaneId);
            nextPane?.terminal.focus();
          }
        });
      }
    },

    // ============ SPLIT PANE METHODS ============

    // Generate a unique pane ID
    _generatePaneId(): string {
      return `pane-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    },

    // Split the current terminal view (creates a new INDEPENDENT terminal session to the same container)
    async splitPane(
      sessionId: string,
      direction: SplitDirection = "horizontal",
    ): Promise<string | null> {
      const state = getState();
      const session = state.sessions.get(sessionId);
      if (!session) return null;

      const authToken = get(token);
      if (!authToken) return null;

      // Create a new terminal instance with its OWN WebSocket connection
      let XtermTerminal: (typeof import("@xterm/xterm"))["Terminal"];
      let XtermFitAddon: (typeof import("@xterm/addon-fit"))["FitAddon"];
      let XtermUnicode11Addon: (typeof import("@xterm/addon-unicode11"))["Unicode11Addon"];
      let XtermWebLinksAddon: (typeof import("@xterm/addon-web-links"))["WebLinksAddon"];
      try {
        ({
          Terminal: XtermTerminal,
          FitAddon: XtermFitAddon,
          Unicode11Addon: XtermUnicode11Addon,
          WebLinksAddon: XtermWebLinksAddon,
        } = await loadXtermCore());
      } catch (e) {
        console.error("[Terminal] Failed to load xterm:", e);
        toast.error("Failed to open split pane. Please try again.");
        return null;
      }
      let newTerminal: Terminal;
      let newFitAddon: FitAddon;
      try {
        // Ensure we use the current theme state
        const currentOptions = {
          ...TERMINAL_OPTIONS,
          theme: getCurrentTerminalTheme(),
        };
        newTerminal = new XtermTerminal(currentOptions);
        newFitAddon = new XtermFitAddon();
        newTerminal.loadAddon(newFitAddon);
        newTerminal.loadAddon(new XtermWebLinksAddon());
        // Unicode11 addon for proper Unicode character widths
        const unicode11Addon = new XtermUnicode11Addon();
        newTerminal.loadAddon(unicode11Addon);
        newTerminal.unicode.activeVersion = "11";

        // Enable mouse events for TUI apps (opencode, vim, tmux, etc.)
        // Same mouse modes as main terminal
        newTerminal.options.mouseSupport = true;
        newTerminal.write("\x1b[?1000h"); // X10 mouse mode
        newTerminal.write("\x1b[?1002h"); // Cell motion mouse tracking
        newTerminal.write("\x1b[?1003h"); // All motion mouse tracking (for opencode)
        newTerminal.write("\x1b[?1006h"); // SGR mouse mode (preferred)
        newTerminal.write("\x1b[?1007h"); // Alternate scroll mode
        newTerminal.write("\x1b[?1015h"); // URXVT mouse mode (fallback)

        // Attach custom key handler (browser overrides + macOS mapping)
        newTerminal.attachCustomKeyEventHandler(
          createCustomKeyHandler(newTerminal),
        );
      } catch (e) {
        console.error("[Terminal] Failed to initialize split pane:", e);
        toast.error("Failed to open split pane. Please try again.");
        return null;
      }

      const paneId = this._generatePaneId();
      const newPane: SplitPane = {
        id: paneId,
        sessionId,
        terminal: newTerminal,
        fitAddon: newFitAddon,
        webglAddon: null,
        resizeObserver: null,
        ws: null, // Each pane gets its own WebSocket
        status: "connecting",
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

      // Update status to connecting and show message
      updateSession(sessionId, (s) => {
        const newPanes = new Map(s.splitPanes);
        const p = newPanes.get(paneId);
        if (p) {
          newPanes.set(paneId, { ...p, status: "connecting" });
        }
        return { ...s, splitPanes: newPanes };
      });

      // Create independent WebSocket connection for this pane with unique ID
      // Add newSession=true to tell backend to create a fresh tmux session (not resume main)
      const agentId =
        session.agentId ||
        (session.containerId.startsWith("agent:")
          ? session.containerId.slice("agent:".length)
          : null);
      let wsUrl: string;
      if (session.isAgentSession || agentId) {
        if (!agentId) return;
        wsUrl = `${getWsUrl()}/ws/agent/${encodeURIComponent(agentId)}/terminal?id=${encodeURIComponent(paneId)}&newSession=true`;
      } else {
        wsUrl = `${getWsUrl()}/ws/terminal/${encodeURIComponent(session.containerId)}?id=${encodeURIComponent(paneId)}&newSession=true`;
      }
      const ws = createRexecWebSocket(wsUrl, authToken);

      ws.onopen = () => {
        // Set to connected immediately - shell setup happens in background
        updateSession(sessionId, (s) => {
          const newPanes = new Map(s.splitPanes);
          const p = newPanes.get(paneId);
          if (p) {
            newPanes.set(paneId, {
              ...p,
              status: "connected",
              reconnectAttempts: 0,
            });
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

      // Track if we're in a TUI app that needs mouse support (opencode, vim, tmux, etc.)
      // Detect TUI apps by checking for common patterns in output
      let isTUIAppSplit = false;
      let tuiAppDetectionBufferSplit = "";
      const TUI_INDICATORS_SPLIT = [
        "opencode",
        "vim",
        "nvim",
        "tmux",
        "screen",
        "less",
        "more",
        "top",
        "htop",
        "nano",
        "emacs",
      ];

      const detectTUIAppSplit = (data: string): boolean => {
        // Check last 2KB of output for TUI indicators
        tuiAppDetectionBufferSplit = (tuiAppDetectionBufferSplit + data).slice(-2048);
        const lower = tuiAppDetectionBufferSplit.toLowerCase();
        return TUI_INDICATORS_SPLIT.some((indicator) => lower.includes(indicator));
      };

      // Filter problematic escape sequences from output to prevent artifacts
      // BUT preserve mouse tracking sequences for TUI apps that need them
      const sanitizeOutput = (data: string): string => {
        // Update TUI app detection
        isTUIAppSplit = detectTUIAppSplit(data) || isTUIAppSplit;

        let result = data;

        // Only filter mouse tracking sequences if NOT in a TUI app
        // TUI apps like opencode, vim, tmux need mouse tracking to work properly
        if (!isTUIAppSplit) {
          // Filter mouse tracking sequences (SGR format: ESC[<button;x;yM or ESC[<button;x;ym)
          result = result.replace(/\x1b\[<\d+;\d+;\d+[Mm]/g, "");
        }
        // Always filter OSC (Operating System Command) query responses that leak into input
        // These are sequences like ESC ] <number> ; <data> BEL or ESC ] <number> ; <data> ESC \
        // Common ones: OSC 10/11 (foreground/background color queries)
        result = result.replace(/\x1b\]\d+;[^\x07\x1b]*(?:\x07|\x1b\\)/g, "");
        // Also filter partial/malformed OSC sequences that might appear at boundaries
        result = result.replace(/\]\d+;rgb:[0-9a-f/]+/gi, "");
        return result;
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

      // Schedule buffer flush - uses RAF for smooth rendering with timeout fallback
      const scheduleFlush = () => {
        if (flushTimeout) return;

        const timeSinceLastFlush = performance.now() - lastFlushTime;

        if (timeSinceLastFlush > OUTPUT_FLUSH_INTERVAL) {
          // Use RAF but with a guaranteed timeout fallback
          rafId = requestAnimationFrame(() => {
            if (flushTimeout) {
              clearTimeout(flushTimeout);
              flushTimeout = null;
            }
            flushBuffer();
          });
          // Fallback timeout in case RAF doesn't fire (throttled tab, etc.)
          flushTimeout = setTimeout(() => {
            if (rafId) {
              cancelAnimationFrame(rafId);
              rafId = null;
            }
            flushBuffer();
          }, 50); // 50ms max delay guarantees prompt appears
        } else {
          flushTimeout = setTimeout(() => {
            flushBuffer();
          }, OUTPUT_FLUSH_INTERVAL);
        }
      };

      ws.onmessage = async (event) => {
        // Handle Blob data from WebSocket (common with agent connections)
        let eventData = event.data;
        if (eventData instanceof Blob) {
          eventData = await eventData.text();
        }

        try {
          const msg = JSON.parse(eventData);
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
          } else if (
            msg.type === "shell_ready" ||
            msg.type === "shell_started"
          ) {
            // Shell is now ready for input - update status to connected
            // shell_ready: sent by container terminals after exec attach
            // shell_started: sent by agent terminals after PTY is ready
            updateSession(sessionId, (s) => {
              const newPanes = new Map(s.splitPanes);
              const p = newPanes.get(paneId);
              if (p) {
                newPanes.set(paneId, { ...p, status: "connected" });
              }
              return { ...s, splitPanes: newPanes };
            });
          }
        } catch {
          outputBuffer += eventData;
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

          // Update reconnect attempts and status
          updateSession(sessionId, (s) => {
            const newPanes = new Map(s.splitPanes);
            const p = newPanes.get(paneId);
            if (p) {
              newPanes.set(paneId, {
                ...p,
                status: "connecting",
                reconnectAttempts: attemptNum,
              });
            }
            return { ...s, splitPanes: newPanes };
          });

          // Only show message after silent threshold is exceeded
          if (attemptNum > WS_SILENT_RECONNECT_THRESHOLD) {
            pane.terminal.writeln(
              `\r\n\x1b[33m⟳ Split session reconnecting (${attemptNum}/${WS_MAX_RECONNECT})...\x1b[0m`,
            );
          } else {
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
          // Update status to disconnected/error
          updateSession(sessionId, (s) => {
            const newPanes = new Map(s.splitPanes);
            const p = newPanes.get(paneId);
            if (p) {
              newPanes.set(paneId, {
                ...p,
                status: isContainerGone ? "error" : "disconnected",
              });
            }
            return { ...s, splitPanes: newPanes };
          });

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

      ws.onerror = () => {};

      // Handle terminal input for split pane
      pane.terminal.onData((data) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify({ type: "input", data: data }));
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

      // Try to load WebGL addon (best-effort)
      loadXtermWebgl()
        .then(({ WebglAddon: XtermWebglAddon }) => {
          const current = getState().sessions.get(sessionId);
          const currentPane = current?.splitPanes.get(paneId);
          if (!currentPane) return;
          if (currentPane.webglAddon) return;

          try {
            const webglAddon = new XtermWebglAddon();
            webglAddon.onContextLoss(() => webglAddon.dispose());
            currentPane.terminal.loadAddon(webglAddon);
            currentPane.webglAddon = webglAddon;
          } catch (e) {
            console.warn("WebGL addon not available for split pane:", e);
          }
        })
        .catch(() => {
          // Ignore
        });

      // Setup resize observer with debouncing
      if (window.ResizeObserver) {
        let resizeTimeout: ReturnType<typeof setTimeout> | null = null;
        const resizeObserver = new ResizeObserver(() => {
          // Debounce resize events to prevent terminal corruption
          if (resizeTimeout) clearTimeout(resizeTimeout);
          resizeTimeout = setTimeout(() => {
            this.fitSplitPane(sessionId, paneId);
          }, 50); // 50ms debounce
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

  // Setup keyboard shortcuts for zoom (Cmd/Ctrl + Plus/Minus)
  window.addEventListener("keydown", (e: KeyboardEvent) => {
    const isMod = e.metaKey || e.ctrlKey;
    if (!isMod) return;

    // Zoom in: Cmd/Ctrl + Plus or Cmd/Ctrl + =
    if (e.key === "+" || e.key === "=") {
      e.preventDefault();
      terminal.zoomIn();
    }
    // Zoom out: Cmd/Ctrl + Minus
    else if (e.key === "-") {
      e.preventDefault();
      terminal.zoomOut();
    }
    // Reset zoom: Cmd/Ctrl + 0
    else if (e.key === "0") {
      e.preventDefault();
      terminal.resetZoom();
    }
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
