<script lang="ts">
  import { onMount, onDestroy, afterUpdate } from 'svelte';
  import { terminal, type TerminalSession } from '$stores/terminal';

  export let session: TerminalSession;

  let containerElement: HTMLDivElement;
  let currentContainerId: string | null = null;

  // Re-attach terminal when container element changes or becomes available
  function attachIfNeeded() {
    if (!containerElement || !session) return;

    // Check if terminal needs to be attached to this container
    const terminalParent = session.terminal?.element?.parentElement;

    if (terminalParent !== containerElement) {
      // Terminal is not attached to this container, re-attach it
      // First, clear the container
      containerElement.innerHTML = '';

      // Re-open terminal in this container
      if (session.terminal) {
        session.terminal.open(containerElement);

        // Fit after re-attachment
        setTimeout(() => {
          terminal.fitSession(session.id);
        }, 50);
      }

      currentContainerId = session.id;
    }
  }

  onMount(() => {
    if (containerElement && session) {
      // Check if terminal is already initialized
      if (session.terminal?.element) {
        // Re-attach to this container
        attachIfNeeded();
      } else {
        // First time attachment
        terminal.attachTerminal(session.id, containerElement);
      }

      // Connect WebSocket if not already connected or connecting
      if (!session.ws || (session.ws.readyState !== WebSocket.OPEN && session.ws.readyState !== WebSocket.CONNECTING)) {
        terminal.connectWebSocket(session.id);
      }
    }
  });

  // Re-attach after DOM updates (handles dock/float switching)
  afterUpdate(() => {
    attachIfNeeded();
  });

  onDestroy(() => {
    // Don't dispose terminal here - it's managed by the store
    // Just clean up local references
    currentContainerId = null;
  });

  // Actions
  function handleReconnect() {
    terminal.reconnectSession(session.id);
  }

  function handleClear() {
    if (session.terminal) {
      session.terminal.clear();
    }
  }

  function handleCopy() {
    if (session.terminal) {
      const selection = session.terminal.getSelection();
      if (selection) {
        navigator.clipboard.writeText(selection);
      }
    }
  }

  function handlePaste() {
    navigator.clipboard.readText().then((text) => {
      if (session.ws && session.ws.readyState === WebSocket.OPEN) {
        session.ws.send(JSON.stringify({ type: 'input', data: text }));
      }
    });
  }

  // Focus terminal when clicking on container
  function handleContainerClick() {
    if (session.terminal) {
      session.terminal.focus();
    }
  }

  // Reactive status
  $: status = session?.status || 'disconnected';
  $: isConnected = status === 'connected';
  $: isConnecting = status === 'connecting';
  $: isDisconnected = status === 'disconnected' || status === 'error';
</script>

<div class="terminal-panel-wrapper">
  <!-- Toolbar -->
  <div class="terminal-toolbar">
    <div class="toolbar-left">
      <span class="terminal-name">{session.name}</span>
      <span class="terminal-status" class:connected={isConnected} class:connecting={isConnecting} class:disconnected={isDisconnected}>
        <span class="status-indicator"></span>
        {status}
      </span>
    </div>

    <div class="toolbar-actions">
      {#if isDisconnected}
        <button class="toolbar-btn" on:click={handleReconnect} title="Reconnect">
          ↻ Reconnect
        </button>
      {/if}
      <button class="toolbar-btn" on:click={handleCopy} title="Copy Selection">
        📋 Copy
      </button>
      <button class="toolbar-btn" on:click={handlePaste} title="Paste">
        📥 Paste
      </button>
      <button class="toolbar-btn" on:click={handleClear} title="Clear Terminal">
        🗑 Clear
      </button>
    </div>
  </div>

  <!-- Terminal Container -->
  <div
    class="terminal-container"
    bind:this={containerElement}
    on:click={handleContainerClick}
    role="textbox"
    tabindex="0"
  ></div>

  <!-- Connection overlay -->
  {#if isConnecting}
    <div class="connection-overlay">
      <div class="connection-spinner"></div>
      <span>Connecting...</span>
    </div>
  {/if}

  {#if isDisconnected}
    <div class="disconnected-overlay">
      <span class="disconnected-icon">⚠</span>
      <span>Disconnected</span>
      <button class="reconnect-btn" on:click={handleReconnect}>
        ↻ Reconnect
      </button>
    </div>
  {/if}
</div>

<style>
  .terminal-panel-wrapper {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
    position: relative;
    background: #0a0a0a;
  }

  /* Toolbar */
  .terminal-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 12px;
    background: #111;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .terminal-name {
    font-size: 12px;
    color: var(--text);
    font-weight: 500;
  }

  .terminal-status {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 10px;
    text-transform: uppercase;
    padding: 2px 8px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
  }

  .status-indicator {
    width: 6px;
    height: 6px;
  }

  .terminal-status.connected {
    border-color: var(--green);
    color: var(--green);
  }

  .terminal-status.connected .status-indicator {
    background: var(--green);
  }

  .terminal-status.connecting {
    border-color: var(--yellow);
    color: var(--yellow);
  }

  .terminal-status.connecting .status-indicator {
    background: var(--yellow);
    animation: pulse 1s infinite;
  }

  .terminal-status.disconnected {
    border-color: var(--red);
    color: var(--red);
  }

  .terminal-status.disconnected .status-indicator {
    background: var(--red);
  }

  .toolbar-actions {
    display: flex;
    gap: 4px;
  }

  .toolbar-btn {
    background: none;
    border: 1px solid transparent;
    color: var(--text-muted);
    font-size: 11px;
    font-family: var(--font-mono);
    padding: 4px 8px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .toolbar-btn:hover {
    color: var(--text);
    background: var(--bg-tertiary);
    border-color: var(--border);
  }

  /* Terminal Container */
  .terminal-container {
    flex: 1;
    width: 100%;
    overflow: hidden;
    padding: 8px;
  }

  .terminal-container:focus {
    outline: none;
  }

  .terminal-container :global(.xterm) {
    height: 100% !important;
    width: 100% !important;
  }

  .terminal-container :global(.xterm-viewport) {
    overflow-y: auto !important;
  }

  .terminal-container :global(.xterm-screen) {
    height: 100% !important;
  }

  /* Connection Overlay */
  .connection-overlay {
    position: absolute;
    inset: 0;
    top: 40px; /* Below toolbar */
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    background: rgba(10, 10, 10, 0.9);
    z-index: 10;
  }

  .connection-spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .connection-overlay span {
    color: var(--text-muted);
    font-size: 13px;
  }

  /* Disconnected Overlay */
  .disconnected-overlay {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
    background: rgba(255, 0, 60, 0.1);
    border: 1px solid var(--red);
    z-index: 10;
  }

  .disconnected-icon {
    font-size: 16px;
  }

  .disconnected-overlay span {
    color: var(--red);
    font-size: 12px;
  }

  .reconnect-btn {
    background: var(--red);
    border: none;
    color: var(--bg);
    font-size: 11px;
    font-family: var(--font-mono);
    padding: 4px 10px;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .reconnect-btn:hover {
    opacity: 0.9;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }
</style>
