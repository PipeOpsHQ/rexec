/**
 * Rexec Embed Widget - Terminal Component
 * Wraps xterm.js with WebSocket connection management
 */

import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { Unicode11Addon } from "@xterm/addon-unicode11";
import { WebglAddon } from "@xterm/addon-webgl";
import "@xterm/xterm/css/xterm.css";

import type {
  RexecEmbedConfig,
  RexecTerminalInstance,
  ConnectionState,
  SessionInfo,
  ContainerStats,
  RexecError,
  RexecEventMap,
  WsMessage,
  TerminalTheme,
} from "./types";
import { getTheme } from "./themes";
import { RexecApiClient, TerminalWebSocket, generateSessionId } from "./api";

// Default terminal options
const DEFAULT_FONT_SIZE = 14;
const DEFAULT_FONT_FAMILY =
  'JetBrains Mono, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace';
const DEFAULT_SCROLLBACK = 5000;

/**
 * Simple event emitter for terminal events
 */
class EventEmitter {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private listeners: Map<string, Set<(...args: any[]) => void>> = new Map();

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  on(event: string, callback: (...args: any[]) => void): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event)!.add(callback);
    return () => this.off(event, callback);
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  off(event: string, callback: (...args: any[]) => void): void {
    this.listeners.get(event)?.delete(callback);
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  emit(event: string, ...args: any[]): void {
    this.listeners.get(event)?.forEach((callback) => {
      try {
        callback(...args);
      } catch (e) {
        console.error(`[Rexec] Error in event handler for ${event}:`, e);
      }
    });
  }

  removeAllListeners(): void {
    this.listeners.clear();
  }
}

/**
 * Main Rexec Terminal class
 */
export class RexecTerminal implements RexecTerminalInstance {
  // Configuration
  private config: Required<RexecEmbedConfig>;
  private container: HTMLElement;

  // xterm.js terminal
  private terminal: Terminal | null = null;
  private fitAddon: FitAddon | null = null;
  private webglAddon: WebglAddon | null = null;
  private resizeObserver: ResizeObserver | null = null;
  private intersectionObserver: IntersectionObserver | null = null;
  private isVisible: boolean = false;
  private pendingFit: boolean = false;

  // Connection
  private api: RexecApiClient;
  private ws: TerminalWebSocket | null = null;
  private sessionId: string;

  // State
  private _state: ConnectionState = "idle";
  private _session: SessionInfo | null = null;
  private _stats: ContainerStats | null = null;
  private destroyed = false;

  // Events
  private events = new EventEmitter();

  // Output buffering for performance
  private outputBuffer = "";
  private flushTimeout: ReturnType<typeof setTimeout> | null = null;

  constructor(element: HTMLElement | string, config: RexecEmbedConfig = {}) {
    // Resolve container element
    if (typeof element === "string") {
      const el = document.querySelector<HTMLElement>(element);
      if (!el) {
        throw new Error(`[Rexec] Element not found: ${element}`);
      }
      this.container = el;
    } else {
      this.container = element;
    }

    // Merge config with defaults
    this.config = {
      token: config.token ?? "",
      container: config.container ?? "",
      shareCode: config.shareCode ?? "",
      role: config.role ?? "",
      image: config.image ?? "ubuntu",
      baseUrl: config.baseUrl ?? this.detectBaseUrl(),
      theme: config.theme ?? "dark",
      fontSize: config.fontSize ?? DEFAULT_FONT_SIZE,
      fontFamily: config.fontFamily ?? DEFAULT_FONT_FAMILY,
      cursorStyle: config.cursorStyle ?? "block",
      cursorBlink: config.cursorBlink ?? true,
      scrollback: config.scrollback ?? DEFAULT_SCROLLBACK,
      webgl: config.webgl ?? true,
      showToolbar: config.showToolbar ?? true,
      showStatus: config.showStatus ?? true,
      allowCopy: config.allowCopy ?? true,
      allowPaste: config.allowPaste ?? true,
      onReady: config.onReady ?? (() => {}),
      onStateChange: config.onStateChange ?? (() => {}),
      onError: config.onError ?? (() => {}),
      onData: config.onData ?? (() => {}),
      onResize: config.onResize ?? (() => {}),
      onDisconnect: config.onDisconnect ?? (() => {}),
      autoReconnect: config.autoReconnect ?? true,
      maxReconnectAttempts: config.maxReconnectAttempts ?? 10,
      initialCommand: config.initialCommand ?? "",
      className: config.className ?? "",
      fitToContainer: config.fitToContainer ?? true,
    };

    // Initialize API client
    this.api = new RexecApiClient(
      this.config.baseUrl,
      this.config.token || undefined,
    );

    // Generate session ID
    this.sessionId = generateSessionId();

    // Register callbacks as event listeners
    if (this.config.onReady) this.on("ready", this.config.onReady);
    if (this.config.onStateChange)
      this.on("stateChange", this.config.onStateChange);
    if (this.config.onError) this.on("error", this.config.onError);
    if (this.config.onData) this.on("data", this.config.onData);
    if (this.config.onResize) this.on("resize", this.config.onResize);
    if (this.config.onDisconnect)
      this.on("disconnect", this.config.onDisconnect);

    // Initialize terminal
    this.init();
  }

  // ========== Public Properties ==========

  get state(): ConnectionState {
    return this._state;
  }

  get session(): SessionInfo | null {
    return this._session;
  }

  get stats(): ContainerStats | null {
    return this._stats;
  }

  // ========== Public Methods ==========

  write(data: string): void {
    if (!this.ws || !this.ws.isConnected()) {
      console.warn("[Rexec] Cannot write: not connected");
      return;
    }
    this.ws.sendRaw(data);
  }

  writeln(data: string): void {
    this.write(data + "\r");
  }

  clear(): void {
    this.terminal?.clear();
  }

  fit(): void {
    if (this.fitAddon && this.terminal) {
      try {
        // Get container dimensions for debugging
        const rect = this.container.getBoundingClientRect();
        console.log(
          "[Rexec SDK] fit() called, container size:",
          rect.width,
          "x",
          rect.height,
          "visible:",
          this.isVisible,
        );

        // Ensure container has dimensions before fitting
        if (rect.width === 0 || rect.height === 0) {
          console.warn(
            "[Rexec SDK] Container has zero dimensions, skipping fit",
          );
          this.pendingFit = true;
          return;
        }

        // If container is not visible, mark as pending
        if (!this.isVisible) {
          console.log(
            "[Rexec SDK] Container not visible, marking fit as pending",
          );
          this.pendingFit = true;
          // Still try to fit in case visibility detection is wrong
        }

        this.fitAddon.fit();
        const dims = this.fitAddon.proposeDimensions();
        console.log("[Rexec SDK] fit() proposed dimensions:", dims);
        if (dims) {
          this.ws?.sendResize(dims.cols, dims.rows);
          this.events.emit("resize", dims.cols, dims.rows);
        }
      } catch (e) {
        console.error("[Rexec SDK] fit() error:", e);
        // Ignore fit errors (can happen if element is hidden)
      }
    }
  }

  focus(): void {
    this.terminal?.focus();
  }

  blur(): void {
    this.terminal?.blur();
  }

  async reconnect(): Promise<void> {
    this.disconnect();
    await this.connect();
  }

  disconnect(): void {
    this.ws?.close();
    this.ws = null;
    this.setState("disconnected");
  }

  destroy(): void {
    if (this.destroyed) return;
    this.destroyed = true;

    // Clear output buffer
    if (this.flushTimeout) {
      clearTimeout(this.flushTimeout);
      this.flushTimeout = null;
    }

    // Disconnect WebSocket
    this.disconnect();

    // Dispose resize observer
    this.resizeObserver?.disconnect();
    this.resizeObserver = null;

    // Dispose intersection observer
    this.intersectionObserver?.disconnect();
    this.intersectionObserver = null;

    // Dispose WebGL addon
    this.webglAddon?.dispose();
    this.webglAddon = null;

    // Dispose fit addon
    this.fitAddon?.dispose();
    this.fitAddon = null;

    // Dispose terminal
    this.terminal?.dispose();
    this.terminal = null;

    // Clear container
    this.container.innerHTML = "";

    // Remove all event listeners
    this.events.removeAllListeners();
  }

  getDimensions(): { cols: number; rows: number } {
    if (!this.terminal) {
      return { cols: 80, rows: 24 };
    }
    return {
      cols: this.terminal.cols,
      rows: this.terminal.rows,
    };
  }

  async copySelection(): Promise<boolean> {
    if (!this.config.allowCopy) return false;
    const selection = this.terminal?.getSelection();
    if (selection) {
      try {
        await navigator.clipboard.writeText(selection);
        return true;
      } catch {
        return false;
      }
    }
    return false;
  }

  async paste(): Promise<void> {
    if (!this.config.allowPaste) return;
    try {
      const text = await navigator.clipboard.readText();
      if (text) {
        this.write(text);
      }
    } catch {
      // Clipboard access denied
    }
  }

  selectAll(): void {
    this.terminal?.selectAll();
  }

  scrollToBottom(): void {
    this.terminal?.scrollToBottom();
  }

  setFontSize(size: number): void {
    if (this.terminal) {
      this.terminal.options.fontSize = Math.max(8, Math.min(32, size));
      this.fit();
    }
  }

  setTheme(theme: TerminalTheme | "dark" | "light"): void {
    if (this.terminal) {
      this.terminal.options.theme = getTheme(theme);
    }
  }

  on<K extends keyof RexecEventMap>(
    event: K,
    callback: RexecEventMap[K],
  ): () => void {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return this.events.on(event as string, callback as any);
  }

  off<K extends keyof RexecEventMap>(
    event: K,
    callback: RexecEventMap[K],
  ): void {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    this.events.off(event as string, callback as any);
  }

  // ========== Private Methods ==========

  /**
   * Detect base URL from script src or current page
   */
  private detectBaseUrl(): string {
    // Try to detect from the script tag that loaded us
    const scripts = document.getElementsByTagName("script");
    for (const script of scripts) {
      const src = script.src;
      if (src && (src.includes("rexec") || src.includes("embed"))) {
        try {
          const url = new URL(src);
          return `${url.protocol}//${url.host}`;
        } catch {
          // Ignore invalid URLs
        }
      }
    }
    // Fall back to current origin or default
    if (typeof window !== "undefined" && window.location.origin !== "null") {
      return window.location.origin;
    }
    return "https://rexec.dev";
  }

  /**
   * Initialize the terminal
   */
  private async init(): Promise<void> {
    this.setupContainer();
    this.createTerminal();
    // Note: setupResizeObserver is now called in createTerminal after terminal.open()
    await this.connect();
  }

  /**
   * Set up the container element
   */
  private setupContainer(): void {
    // Add Rexec class
    this.container.classList.add("rexec-embed");
    if (this.config.className) {
      this.container.classList.add(this.config.className);
    }

    // Ensure container has proper styling
    const style = window.getComputedStyle(this.container);
    if (style.position === "static") {
      this.container.style.position = "relative";
    }

    // Add minimal required styles
    if (!document.getElementById("rexec-embed-styles")) {
      const styleEl = document.createElement("style");
      styleEl.id = "rexec-embed-styles";
      styleEl.textContent = `
        .rexec-embed {
          width: 100%;
          height: 100%;
          min-height: 300px;
          overflow: hidden;
          background: #0d1117;
          position: relative;
          display: flex;
          flex-direction: column;
        }
        .rexec-embed .terminal-wrapper {
          width: 100%;
          flex: 1;
          min-height: 0;
          position: relative;
          display: flex;
          flex-direction: column;
        }
        .rexec-embed .xterm {
          padding: 8px;
          padding-bottom: 28px;
          flex: 1;
          min-height: 0;
        }
        .rexec-embed .xterm-screen {
          width: 100% !important;
          height: 100% !important;
        }
        .rexec-embed .xterm-viewport {
          width: 100% !important;
        }
        .rexec-embed .xterm-helper-textarea {
          position: absolute !important;
          opacity: 0 !important;
          left: -9999px !important;
          top: 0 !important;
          width: 0 !important;
          height: 0 !important;
          z-index: -10 !important;
          pointer-events: none !important;
        }
        .rexec-embed .xterm-screen {
          cursor: text;
        }
        .rexec-embed .terminal-wrapper:focus-within .xterm-cursor {
          animation: blink 1s step-end infinite;
        }
        @keyframes blink {
          50% { opacity: 0; }
        }
        .rexec-embed .xterm-viewport::-webkit-scrollbar {
          width: 8px;
        }
        .rexec-embed .xterm-viewport::-webkit-scrollbar-thumb {
          background: rgba(255, 255, 255, 0.2);
          border-radius: 4px;
        }
        .rexec-embed .xterm-viewport::-webkit-scrollbar-track {
          background: transparent;
        }
        .rexec-embed .status-overlay {
          position: absolute;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          color: #58a6ff;
          font-family: system-ui, sans-serif;
          font-size: 14px;
          text-align: center;
          z-index: 10;
          pointer-events: none;
        }
        .rexec-embed .status-overlay .spinner {
          width: 24px;
          height: 24px;
          border: 2px solid rgba(88, 166, 255, 0.3);
          border-top-color: #58a6ff;
          border-radius: 50%;
          animation: rexec-spin 1s linear infinite;
          margin: 0 auto 8px;
        }
        @keyframes rexec-spin {
          to { transform: rotate(360deg); }
        }
        .rexec-embed .rexec-branding {
          position: absolute;
          bottom: 0;
          left: 0;
          right: 0;
          height: 24px;
          background: linear-gradient(to top, rgba(13, 17, 23, 0.95) 0%, rgba(13, 17, 23, 0.8) 70%, transparent 100%);
          display: flex;
          align-items: center;
          justify-content: flex-end;
          padding: 0 10px;
          z-index: 5;
          pointer-events: auto;
        }
        .rexec-embed .rexec-branding a {
          display: flex;
          align-items: center;
          gap: 6px;
          text-decoration: none;
          color: rgba(255, 255, 255, 0.5);
          font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
          font-size: 11px;
          font-weight: 500;
          transition: color 0.2s, transform 0.2s;
        }
        .rexec-embed .rexec-branding a:hover {
          color: #00ff41;
          transform: translateY(-1px);
        }
        .rexec-embed .rexec-branding .rexec-logo {
          width: 14px;
          height: 14px;
          fill: currentColor;
        }
        .rexec-embed .rexec-branding .powered-text {
          opacity: 0.7;
        }
        .rexec-embed .rexec-branding .rexec-name {
          color: #00ff41;
          font-weight: 600;
          letter-spacing: 0.5px;
        }
        .rexec-embed .rexec-branding a:hover .rexec-name {
          text-shadow: 0 0 8px rgba(0, 255, 65, 0.5);
        }
      `;
      document.head.appendChild(styleEl);
    }

    // Create terminal wrapper
    const wrapper = document.createElement("div");
    wrapper.className = "terminal-wrapper";
    wrapper.setAttribute("tabindex", "0");
    this.container.appendChild(wrapper);

    // Add Rexec branding
    const branding = document.createElement("div");
    branding.className = "rexec-branding";
    branding.innerHTML = `
      <a href="https://rexec.sh" target="_blank" rel="noopener noreferrer" title="Powered by Rexec - Terminal as a Service">
        <span class="powered-text">Powered by</span>
        <svg class="rexec-logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M4 4h16v2H4V4zm0 4h10v2H4V8zm0 4h16v2H4v-2zm0 4h10v2H4v-2zm12 0h4v4h-4v-4z"/>
        </svg>
        <span class="rexec-name">Rexec</span>
      </a>
    `;
    this.container.appendChild(branding);

    // Add click handler to focus terminal
    this.container.addEventListener("click", (e) => {
      // Don't focus if clicking on branding link
      if ((e.target as HTMLElement).closest(".rexec-branding")) return;
      this.terminal?.focus();
    });
  }

  /**
   * Create the xterm.js terminal
   */
  private createTerminal(): void {
    console.log("[Rexec SDK] createTerminal called");
    const wrapper = this.container.querySelector(".terminal-wrapper");
    if (!wrapper) {
      console.error("[Rexec SDK] No .terminal-wrapper found in container!");
      return;
    }
    console.log("[Rexec SDK] Found terminal wrapper:", wrapper);

    // Create terminal with options
    this.terminal = new Terminal({
      cursorBlink: this.config.cursorBlink,
      cursorStyle: this.config.cursorStyle,
      fontSize: this.config.fontSize,
      fontFamily: this.config.fontFamily,
      theme: getTheme(this.config.theme),
      scrollback: this.config.scrollback,
      allowProposedApi: true,
      convertEol: true,
      scrollOnUserInput: true,
      altClickMovesCursor: true,
      macOptionIsMeta: true,
      macOptionClickForcesSelection: true,
    });

    // Load addons
    this.fitAddon = new FitAddon();
    this.terminal.loadAddon(this.fitAddon);

    const unicode11Addon = new Unicode11Addon();
    this.terminal.loadAddon(unicode11Addon);
    this.terminal.unicode.activeVersion = "11";

    const webLinksAddon = new WebLinksAddon();
    this.terminal.loadAddon(webLinksAddon);

    // Open terminal in DOM
    console.log("[Rexec SDK] Opening terminal in wrapper");
    this.terminal.open(wrapper as HTMLElement);
    console.log("[Rexec SDK] Terminal opened, element:", this.terminal.element);

    // Try WebGL renderer for better performance
    if (this.config.webgl) {
      try {
        this.webglAddon = new WebglAddon();
        this.webglAddon.onContextLoss(() => {
          this.webglAddon?.dispose();
          this.webglAddon = null;
        });
        this.terminal.loadAddon(this.webglAddon);
      } catch (e) {
        console.warn("[Rexec] WebGL not available, using canvas renderer");
      }
    }

    // Initial fit - wait for container to have dimensions
    requestAnimationFrame(() => {
      console.log("[Rexec SDK] Running initial fit sequence");
      this.waitForDimensionsAndFit();
    });

    // Set up observers to catch container sizing and visibility changes
    this.setupResizeObserver();
    this.setupIntersectionObserver();

    // Write a test message to verify terminal works
    this.terminal.write(
      "\x1b[33m[Rexec SDK] Terminal initialized...\x1b[0m\r\n",
    );
    console.log("[Rexec SDK] Wrote test message to terminal");

    // Handle terminal data input
    this.terminal.onData((data) => {
      console.log("[Rexec SDK] Terminal input:", data.length, "chars");
      if (this.ws?.isConnected()) {
        this.ws.sendRaw(data);
      } else {
        console.warn("[Rexec SDK] WebSocket not connected, can't send input");
      }
    });

    // Handle terminal resize
    this.terminal.onResize(({ cols, rows }) => {
      this.ws?.sendResize(cols, rows);
      this.events.emit("resize", cols, rows);
    });

    // Handle paste
    if (this.config.allowPaste) {
      this.terminal.attachCustomKeyEventHandler((event) => {
        if (
          event.type === "keydown" &&
          event.key === "v" &&
          (event.ctrlKey || event.metaKey)
        ) {
          this.paste();
          return false;
        }
        if (
          event.type === "keydown" &&
          event.key === "c" &&
          (event.ctrlKey || event.metaKey) &&
          this.terminal?.hasSelection()
        ) {
          this.copySelection();
          return false;
        }
        return true;
      });
    }
  }

  /**
   * Set up resize observer for auto-fitting
   */
  private setupResizeObserver(): void {
    if (!this.config.fitToContainer) return;

    let resizeTimeout: ReturnType<typeof setTimeout> | null = null;
    let hasInitialFit = false;

    this.resizeObserver = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!entry) return;

      const { width, height } = entry.contentRect;
      console.log("[Rexec SDK] ResizeObserver triggered:", width, "x", height);

      // If container now has dimensions and we haven't done initial fit, do it now
      if (width > 0 && height > 0 && !hasInitialFit) {
        hasInitialFit = true;
        console.log("[Rexec SDK] Container has dimensions, doing initial fit");
        // Use setTimeout to ensure DOM is fully laid out
        setTimeout(() => this.fit(), 0);
        setTimeout(() => this.fit(), 100);
        return;
      }

      // Debounce subsequent resize events
      if (resizeTimeout) clearTimeout(resizeTimeout);
      resizeTimeout = setTimeout(() => this.fit(), 50);
    });

    this.resizeObserver.observe(this.container);

    // Also observe the terminal wrapper for size changes
    const wrapper = this.container.querySelector(".terminal-wrapper");
    if (wrapper) {
      this.resizeObserver.observe(wrapper);
    }
  }

  /**
   * Set up intersection observer to detect when terminal becomes visible
   * This handles cases where terminal is in a modal, tab, or hidden container
   */
  private setupIntersectionObserver(): void {
    console.log("[Rexec SDK] Setting up IntersectionObserver");

    this.intersectionObserver = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        if (!entry) return;

        const wasVisible = this.isVisible;
        this.isVisible = entry.isIntersecting && entry.intersectionRatio > 0;

        console.log(
          "[Rexec SDK] IntersectionObserver:",
          "visible:",
          this.isVisible,
          "ratio:",
          entry.intersectionRatio,
        );

        // Terminal just became visible
        if (this.isVisible && !wasVisible) {
          console.log("[Rexec SDK] Terminal became visible, triggering fit");
          // Multiple fits to handle dynamic sizing
          setTimeout(() => this.fit(), 0);
          setTimeout(() => this.fit(), 50);
          setTimeout(() => this.fit(), 150);
          setTimeout(() => this.fit(), 300);

          // Focus the terminal
          setTimeout(() => this.terminal?.focus(), 100);
        }

        // If we had a pending fit while hidden, do it now
        if (this.isVisible && this.pendingFit) {
          this.pendingFit = false;
          this.fit();
        }
      },
      {
        root: null, // viewport
        threshold: [0, 0.1, 0.5, 1.0], // trigger at multiple visibility levels
      },
    );

    this.intersectionObserver.observe(this.container);
  }

  /**
   * Wait for container to have dimensions, then fit
   */
  private waitForDimensionsAndFit(): void {
    const checkDimensions = () => {
      const rect = this.container.getBoundingClientRect();
      console.log(
        "[Rexec SDK] Checking dimensions:",
        rect.width,
        "x",
        rect.height,
      );

      if (rect.width > 0 && rect.height > 0) {
        console.log("[Rexec SDK] Container ready, fitting terminal");
        this.fit();
        return true;
      }
      return false;
    };

    // Try immediately
    if (checkDimensions()) return;

    // If not ready, poll until ready (up to 2 seconds)
    let attempts = 0;
    const maxAttempts = 20;
    const interval = setInterval(() => {
      attempts++;
      if (checkDimensions() || attempts >= maxAttempts) {
        clearInterval(interval);
        if (attempts >= maxAttempts) {
          console.warn(
            "[Rexec SDK] Container never got dimensions after",
            maxAttempts,
            "attempts",
          );
        }
      }
    }, 100);
  }

  /**
   * Connect to the terminal session
   */
  private async connect(): Promise<void> {
    if (this.destroyed) return;

    this.setState("connecting");
    this.showStatus("Connecting...");

    try {
      let containerId: string;
      let wsUrl: string;

      // Determine connection mode
      if (this.config.shareCode) {
        // Join collab session via share code
        const { data, error } = await this.api.joinSession(
          this.config.shareCode,
        );
        if (error || !data) {
          throw this.createError(
            "JOIN_FAILED",
            error || "Failed to join session",
          );
        }
        containerId = data.container_id;
        this._session = {
          id: data.session_id,
          containerId: data.container_id,
          containerName: data.container_name,
          mode: data.mode,
          expiresAt: data.expires_at,
        };
        wsUrl = this.api.getTerminalWsUrl(containerId, this.sessionId);
      } else if (this.config.container) {
        // Connect to existing container
        containerId = this.config.container;
        this._session = {
          id: this.sessionId,
          containerId,
        };
        wsUrl = this.api.getTerminalWsUrl(containerId, this.sessionId);
      } else if (this.config.role || this.config.image) {
        // Create new container with role and/or image
        this.showStatus("Creating container...");
        const { data: createData, error: createError } =
          await this.api.createContainer(
            this.config.image || "ubuntu",
            this.config.role,
          );
        if (createError || !createData) {
          throw this.createError(
            "CREATE_FAILED",
            createError || "Failed to create container",
          );
        }

        // Container creation is async - we get the DB ID back immediately
        // but need to wait for the container to actually be running
        const dbId = createData.id;
        this.showStatus("Waiting for container to start...");

        // Poll for container to be ready
        const { data: containerData, error: waitError } =
          await this.api.waitForContainer(dbId, {
            maxAttempts: 90, // Up to 3 minutes for slow roles
            intervalMs: 2000,
            onProgress: (status, _attempt) => {
              const statusMessages: Record<string, string> = {
                creating: "Creating container...",
                pulling: "Pulling image...",
                configuring: "Configuring environment...",
                starting: "Starting container...",
                running: "Container ready!",
              };
              const message =
                statusMessages[status] || `Preparing container (${status})...`;
              this.showStatus(message);
            },
          });

        if (waitError || !containerData) {
          throw this.createError(
            "CREATE_FAILED",
            waitError || "Container failed to start",
          );
        }

        // Now we have the actual running container
        containerId = containerData.docker_id || containerData.id;
        this._session = {
          id: this.sessionId,
          containerId,
          containerName: containerData.name,
          role: containerData.role,
        };
        wsUrl = this.api.getTerminalWsUrl(containerId, this.sessionId);
      } else {
        throw this.createError(
          "CONFIG_ERROR",
          "Must provide container, shareCode, role, or image",
        );
      }

      // Connect WebSocket
      this.connectWebSocket(wsUrl);
    } catch (e) {
      const error =
        e instanceof Error
          ? this.createError("CONNECT_ERROR", e.message)
          : (e as RexecError);
      this.handleError(error);
    }
  }

  /**
   * Connect WebSocket to the terminal
   */
  private connectWebSocket(url: string): void {
    console.log("[Rexec SDK] connectWebSocket called with URL:", url);
    this.ws = new TerminalWebSocket(url, this.config.token || null, {
      autoReconnect: this.config.autoReconnect,
      maxReconnectAttempts: this.config.maxReconnectAttempts,
    });

    this.ws.onOpen = () => {
      console.log("[Rexec SDK] WebSocket opened!");
      this.hideStatus();
      this.setState("connected");

      // Send initial resize
      const dims = this.getDimensions();
      this.ws?.sendResize(dims.cols, dims.rows);

      // Focus terminal so user can type immediately
      // Use setTimeout to ensure DOM is ready
      setTimeout(() => {
        this.terminal?.focus();
        // Double-check focus by also focusing the textarea directly
        const textarea = this.container.querySelector(
          ".xterm-helper-textarea",
        ) as HTMLTextAreaElement;
        if (textarea) {
          textarea.focus();
        }
      }, 100);

      // Send initial command if configured
      if (this.config.initialCommand) {
        setTimeout(() => {
          this.writeln(this.config.initialCommand);
        }, 500);
      }

      // Emit ready event
      this.events.emit("ready", this);
    };

    this.ws.onClose = (code, reason) => {
      console.log("[Rexec SDK] WebSocket closed:", code, reason);
      if (code !== 1000) {
        this.events.emit("disconnect", reason || "Connection closed");
      }
      if (this._state !== "reconnecting") {
        this.setState("disconnected");
      }
    };

    this.ws.onError = () => {
      // Errors are handled by onClose
    };

    this.ws.onReconnecting = (attempt) => {
      this.setState("reconnecting");
      this.showStatus(`Reconnecting... (${attempt})`);
    };

    this.ws.onMessage = (message) => {
      console.log(
        "[Rexec SDK] WS message received:",
        message.type,
        "data:",
        message.data?.substring?.(0, 50) || "(none)",
      );
      this.handleMessage(message);
    };

    console.log("[Rexec SDK] Calling ws.connect()");
    this.ws.connect();
  }

  /**
   * Handle incoming WebSocket message
   */
  private handleMessage(message: WsMessage): void {
    console.log(
      "[Rexec Terminal] handleMessage:",
      message.type,
      "data length:",
      message.data?.length || 0,
    );
    switch (message.type) {
      case "output":
        if (message.data) {
          console.log(
            "[Rexec Terminal] Writing output to terminal:",
            message.data.substring(0, 100),
          );
          this.writeToTerminal(message.data);
          this.events.emit("data", message.data);
        }
        break;

      case "connected":
        this.hideStatus();
        this.setState("connected");
        break;

      case "stats":
        if (message.data) {
          try {
            const statsData =
              typeof message.data === "string"
                ? JSON.parse(message.data)
                : message.data;
            this._stats = {
              cpu: statsData.cpu || 0,
              memory: statsData.memory || 0,
              memoryLimit: statsData.memory_limit || 0,
              diskRead: statsData.disk_read || 0,
              diskWrite: statsData.disk_write || 0,
              diskUsage: statsData.disk_usage,
              diskLimit: statsData.disk_limit || 0,
              netRx: statsData.net_rx || 0,
              netTx: statsData.net_tx || 0,
            };
            this.events.emit("stats", this._stats);
          } catch {
            // Ignore stats parse errors
          }
        }
        break;

      case "error":
        this.handleError(
          this.createError("SERVER_ERROR", message.data || "Server error"),
        );
        break;

      case "setup":
        this.showStatus(message.data || "Setting up...");
        break;

      default:
        // Unknown message type - treat as output if has data
        if (message.data && typeof message.data === "string") {
          this.writeToTerminal(message.data);
        }
    }
  }

  /**
   * Write data to terminal with buffering for performance
   */
  private writeToTerminal(data: string): void {
    console.log(
      "[Rexec Terminal] writeToTerminal called, terminal exists:",
      !!this.terminal,
      "data length:",
      data.length,
    );
    if (!this.terminal) {
      console.error("[Rexec Terminal] No terminal instance!");
      return;
    }

    // For small outputs, write immediately
    if (data.length < 256) {
      console.log("[Rexec Terminal] Writing small output directly");
      this.terminal.write(data);
      return;
    }

    // Buffer larger outputs
    this.outputBuffer += data;

    // Flush if buffer is large
    if (this.outputBuffer.length > 32 * 1024) {
      this.flushOutput();
      return;
    }

    // Schedule flush
    if (!this.flushTimeout) {
      this.flushTimeout = setTimeout(() => this.flushOutput(), 8);
    }
  }

  /**
   * Flush output buffer to terminal
   */
  private flushOutput(): void {
    if (this.flushTimeout) {
      clearTimeout(this.flushTimeout);
      this.flushTimeout = null;
    }

    if (this.outputBuffer && this.terminal) {
      this.terminal.write(this.outputBuffer);
      this.outputBuffer = "";
    }
  }

  /**
   * Update connection state
   */
  private setState(state: ConnectionState): void {
    if (this._state !== state) {
      this._state = state;
      this.events.emit("stateChange", state);
    }
  }

  /**
   * Show status overlay
   */
  private showStatus(message: string): void {
    if (!this.config.showStatus) return;

    let overlay = this.container.querySelector(".status-overlay");
    if (!overlay) {
      overlay = document.createElement("div");
      overlay.className = "status-overlay";
      this.container.appendChild(overlay);
    }
    overlay.innerHTML = `
      <div class="spinner"></div>
      <div>${message}</div>
      <div style="margin-top: 16px; display: flex; align-items: center; gap: 6px; opacity: 0.6;">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
          <path d="M4 4h16v2H4V4zm0 4h10v2H4V8zm0 4h16v2H4v-2zm0 4h10v2H4v-2zm12 0h4v4h-4v-4z"/>
        </svg>
        <span style="font-size: 12px; color: #00ff41; font-weight: 600; letter-spacing: 0.5px;">Rexec</span>
      </div>
    `;
  }

  /**
   * Hide status overlay
   */
  private hideStatus(): void {
    const overlay = this.container.querySelector(".status-overlay");
    if (overlay) {
      overlay.remove();
    }
  }

  /**
   * Create an error object
   */
  private createError(
    code: string,
    message: string,
    recoverable = false,
  ): RexecError {
    return { code, message, recoverable };
  }

  /**
   * Handle an error
   */
  private handleError(error: RexecError): void {
    this.setState("error");
    this.showStatus(`Error: ${error.message}`);
    this.events.emit("error", error);
  }
}
