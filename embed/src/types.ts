/**
 * Rexec Embed Widget Types
 * TypeScript definitions for the embeddable terminal widget
 */

// ========== Configuration Types ==========

/**
 * Main configuration options for the Rexec embed widget
 */
export interface RexecEmbedConfig {
  /**
   * API token for authentication. Required for authenticated sessions.
   * Get this from your Rexec account settings.
   */
  token?: string;

  /**
   * Container ID to connect to. Use this for existing containers.
   */
  container?: string;

  /**
   * Share code for joining a collaborative session.
   * When provided, connects as a guest to an existing shared session.
   */
  shareCode?: string;

  /**
   * Role/environment to create when starting a new container.
   * Options: 'ubuntu', 'debian', 'alpine', 'node', 'python', 'go', 'rust', etc.
   */
  role?: string;

  /**
   * Base OS image for the container. Defaults to 'ubuntu'.
   * Options: 'ubuntu', 'debian', 'alpine', 'fedora', 'archlinux', 'kali', etc.
   */
  image?: string;

  /**
   * Base URL for the Rexec API. Defaults to 'https://rexec.dev'
   */
  baseUrl?: string;

  /**
   * Terminal theme configuration
   */
  theme?: TerminalTheme | "dark" | "light";

  /**
   * Font size in pixels. Default: 14
   */
  fontSize?: number;

  /**
   * Font family for the terminal. Default: 'JetBrains Mono, Menlo, Monaco, monospace'
   */
  fontFamily?: string;

  /**
   * Cursor style. Default: 'block'
   */
  cursorStyle?: "block" | "underline" | "bar";

  /**
   * Whether the cursor should blink. Default: true
   */
  cursorBlink?: boolean;

  /**
   * Number of lines in scrollback buffer. Default: 5000
   */
  scrollback?: number;

  /**
   * Whether to enable WebGL renderer for better performance. Default: true
   */
  webgl?: boolean;

  /**
   * Whether to show the toolbar with session controls. Default: true
   */
  showToolbar?: boolean;

  /**
   * Whether to show connection status indicator. Default: true
   */
  showStatus?: boolean;

  /**
   * Whether to allow users to copy text. Default: true
   */
  allowCopy?: boolean;

  /**
   * Whether to allow users to paste text. Default: true
   */
  allowPaste?: boolean;

  /**
   * Callback when terminal is ready and connected
   */
  onReady?: (terminal: RexecTerminalInstance) => void;

  /**
   * Callback when connection state changes
   */
  onStateChange?: (state: ConnectionState) => void;

  /**
   * Callback when an error occurs
   */
  onError?: (error: RexecError) => void;

  /**
   * Callback when terminal data is received
   */
  onData?: (data: string) => void;

  /**
   * Callback when terminal is resized
   */
  onResize?: (cols: number, rows: number) => void;

  /**
   * Callback when terminal is disconnected
   */
  onDisconnect?: (reason: string) => void;

  /**
   * Auto-reconnect on connection loss. Default: true
   */
  autoReconnect?: boolean;

  /**
   * Maximum reconnection attempts. Default: 10
   */
  maxReconnectAttempts?: number;

  /**
   * Initial command to execute after connection. Optional.
   */
  initialCommand?: string;

  /**
   * Custom CSS class to add to the container
   */
  className?: string;

  /**
   * Fit terminal to container size. Default: true
   */
  fitToContainer?: boolean;
}

/**
 * Terminal color theme configuration
 */
export interface TerminalTheme {
  background: string;
  foreground: string;
  cursor: string;
  cursorAccent?: string;
  selectionBackground?: string;
  selectionForeground?: string;
  selectionInactiveBackground?: string;
  black: string;
  red: string;
  green: string;
  yellow: string;
  blue: string;
  magenta: string;
  cyan: string;
  white: string;
  brightBlack: string;
  brightRed: string;
  brightGreen: string;
  brightYellow: string;
  brightBlue: string;
  brightMagenta: string;
  brightCyan: string;
  brightWhite: string;
}

// ========== State Types ==========

/**
 * Connection state for the terminal
 */
export type ConnectionState =
  | "idle"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "disconnected"
  | "error";

/**
 * Session information
 */
export interface SessionInfo {
  id: string;
  containerId: string;
  containerName?: string;
  role?: string;
  mode?: "view" | "control";
  createdAt?: string;
  expiresAt?: string;
}

/**
 * Container statistics
 */
export interface ContainerStats {
  cpu: number;
  memory: number;
  memoryLimit: number;
  diskRead: number;
  diskWrite: number;
  diskUsage?: number;
  diskLimit: number;
  netRx: number;
  netTx: number;
}

// ========== Event Types ==========

/**
 * Error object for Rexec operations
 */
export interface RexecError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
  recoverable: boolean;
}

/**
 * WebSocket message types
 */
export interface WsMessage {
  type:
    | "input"
    | "output"
    | "resize"
    | "ping"
    | "pong"
    | "stats"
    | "error"
    | "connected"
    | "setup";
  data?: string;
  cols?: number;
  rows?: number;
}

// ========== Instance Types ==========

/**
 * Public API for the Rexec terminal instance
 */
export interface RexecTerminalInstance {
  /**
   * Current connection state
   */
  readonly state: ConnectionState;

  /**
   * Current session information
   */
  readonly session: SessionInfo | null;

  /**
   * Current container statistics
   */
  readonly stats: ContainerStats | null;

  /**
   * Write data to the terminal (send to server)
   */
  write(data: string): void;

  /**
   * Write a line to the terminal (adds newline)
   */
  writeln(data: string): void;

  /**
   * Clear the terminal screen
   */
  clear(): void;

  /**
   * Resize the terminal to fit the container
   */
  fit(): void;

  /**
   * Focus the terminal
   */
  focus(): void;

  /**
   * Blur the terminal
   */
  blur(): void;

  /**
   * Reconnect to the terminal session
   */
  reconnect(): Promise<void>;

  /**
   * Disconnect from the terminal session
   */
  disconnect(): void;

  /**
   * Destroy the terminal instance and clean up resources
   */
  destroy(): void;

  /**
   * Get the current terminal dimensions
   */
  getDimensions(): { cols: number; rows: number };

  /**
   * Copy selected text to clipboard
   */
  copySelection(): Promise<boolean>;

  /**
   * Paste text from clipboard
   */
  paste(): Promise<void>;

  /**
   * Select all text in terminal
   */
  selectAll(): void;

  /**
   * Scroll to the bottom of the terminal
   */
  scrollToBottom(): void;

  /**
   * Set the terminal font size
   */
  setFontSize(size: number): void;

  /**
   * Set the terminal theme
   */
  setTheme(theme: TerminalTheme | "dark" | "light"): void;

  /**
   * Register an event listener
   */
  on<K extends keyof RexecEventMap>(
    event: K,
    callback: RexecEventMap[K],
  ): () => void;

  /**
   * Remove an event listener
   */
  off<K extends keyof RexecEventMap>(
    event: K,
    callback: RexecEventMap[K],
  ): void;
}

/**
 * Event map for terminal events
 */
export interface RexecEventMap {
  ready: (terminal: RexecTerminalInstance) => void;
  stateChange: (state: ConnectionState) => void;
  data: (data: string) => void;
  resize: (cols: number, rows: number) => void;
  error: (error: RexecError) => void;
  disconnect: (reason: string) => void;
  stats: (stats: ContainerStats) => void;
}

// ========== API Types ==========

/**
 * API response for container creation
 */
export interface CreateContainerResponse {
  id: string;
  docker_id: string;
  name: string;
  status: string;
  role: string;
}

/**
 * API response for joining a collab session
 */
export interface JoinSessionResponse {
  session_id: string;
  container_id: string;
  container_name: string;
  mode: "view" | "control";
  role: "owner" | "editor" | "viewer";
  expires_at: string;
}

/**
 * API response for container info
 */
export interface ContainerInfoResponse {
  id: string;
  docker_id: string;
  name: string;
  status: string;
  role: string;
  image: string;
  created_at: string;
}
