<script lang="ts">
    import { onMount, onDestroy, tick } from "svelte";
    import { terminal, type TerminalSession } from "$stores/terminal";
    import { toast } from "$stores/toast";

    export let session: TerminalSession;

    let containerElement: HTMLDivElement;
    let attachedToContainer: HTMLDivElement | null = null;

    // Synchronously check and attach terminal if needed
    async function ensureAttached() {
        if (!containerElement || !session?.terminal) return;

        // Only reattach if the container has changed
        if (attachedToContainer === containerElement) return;

        // Wait for DOM to be ready
        await tick();

        try {
            // Use the store's reattachTerminal method which properly handles
            // ResizeObserver cleanup and setup for the new container
            terminal.reattachTerminal(session.id, containerElement);
            attachedToContainer = containerElement;
        } catch (e) {
            console.error("Failed to attach terminal:", e);
            // Try to recover by recreating attachment
            attachedToContainer = null;
        }
    }

    onMount(async () => {
        if (containerElement && session) {
            if (session.terminal?.element) {
                // Terminal already exists, reattach to new container
                terminal.reattachTerminal(session.id, containerElement);
                attachedToContainer = containerElement;
            } else {
                // First time - let the store create and attach the terminal
                terminal.attachTerminal(session.id, containerElement);
                attachedToContainer = containerElement;
            }

            // Connect WebSocket if not already connected
            if (
                !session.ws ||
                (session.ws.readyState !== WebSocket.OPEN &&
                    session.ws.readyState !== WebSocket.CONNECTING)
            ) {
                terminal.connectWebSocket(session.id);
            }
        }
    });

    onDestroy(() => {
        // Don't dispose terminal - it's managed by the store
        attachedToContainer = null;
    });

    // Use reactive statement to handle container changes (dock/float switch)
    $: if (
        containerElement &&
        session?.terminal &&
        attachedToContainer !== containerElement
    ) {
        // Use async IIFE to handle the async ensureAttached
        (async () => {
            await ensureAttached();
        })();
    }

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
                session.ws.send(JSON.stringify({ type: "input", data: text }));
            }
        });
    }

    function handleCopyLink() {
        const url = `${window.location.origin}/terminal/${session.containerId}`;
        navigator.clipboard
            .writeText(url)
            .then(() => {
                toast.success("Terminal link copied to clipboard");
            })
            .catch(() => {
                toast.error("Failed to copy link");
            });
    }

    // Focus terminal when clicking on container
    function handleContainerClick() {
        if (session.terminal) {
            session.terminal.focus();
        }
    }

    // Reactive status
    $: status = session?.status || "disconnected";
    $: isConnected = status === "connected";
    $: isConnecting = status === "connecting";
    $: isDisconnected = status === "disconnected" || status === "error";
    $: isSettingUp = session?.isSettingUp || false;
    $: setupMessage = session?.setupMessage || "";
</script>

<div class="terminal-panel-wrapper">
    <!-- Toolbar -->
    <div class="terminal-toolbar">
        <div class="toolbar-left">
            <span class="terminal-name">{session.name}</span>
            <span
                class="terminal-status"
                class:connected={isConnected}
                class:connecting={isConnecting}
                class:disconnected={isDisconnected}
            >
                <span class="status-indicator"></span>
                {status}
            </span>
            {#if isSettingUp}
                <span class="setup-indicator">
                    <span class="setup-spinner"></span>
                    Installing...
                </span>
            {/if}
        </div>

        <div class="toolbar-actions">
            {#if isDisconnected}
                <button
                    class="toolbar-btn reconnect-btn"
                    on:click={handleReconnect}
                    title="Reconnect"
                >
                    <svg
                        class="toolbar-icon"
                        viewBox="0 0 16 16"
                        fill="currentColor"
                    >
                        <path
                            d="M11.534 7h3.932a.25.25 0 0 1 .192.41l-1.966 2.36a.25.25 0 0 1-.384 0l-1.966-2.36a.25.25 0 0 1 .192-.41zm-11 2h3.932a.25.25 0 0 0 .192-.41L2.692 6.23a.25.25 0 0 0-.384 0L.342 8.59A.25.25 0 0 0 .534 9z"
                        />
                        <path
                            fill-rule="evenodd"
                            d="M8 3c-1.552 0-2.94.707-3.857 1.818a.5.5 0 1 1-.771-.636A6.002 6.002 0 0 1 13.917 7H12.9A5.002 5.002 0 0 0 8 3zM3.1 9a5.002 5.002 0 0 0 8.757 2.182.5.5 0 1 1 .771.636A6.002 6.002 0 0 1 2.083 9H3.1z"
                        />
                    </svg>
                    Reconnect
                </button>
            {/if}
            <button
                class="toolbar-btn"
                on:click={handleCopyLink}
                title="Copy Terminal Link"
            >
                <svg
                    class="toolbar-icon"
                    viewBox="0 0 16 16"
                    fill="currentColor"
                >
                    <path
                        d="M4.715 6.542 3.343 7.914a3 3 0 1 0 4.243 4.243l1.828-1.829A3 3 0 0 0 8.586 5.5L8 6.086a1.002 1.002 0 0 0-.154.199 2 2 0 0 1 .861 3.337L6.88 11.45a2 2 0 1 1-2.83-2.83l.793-.792a4.018 4.018 0 0 1-.128-1.287z"
                    />
                    <path
                        d="M6.586 4.672A3 3 0 0 0 7.414 9.5l.775-.776a2 2 0 0 1-.896-3.346L9.12 3.55a2 2 0 1 1 2.83 2.83l-.793.792c.112.42.155.855.128 1.287l1.372-1.372a3 3 0 1 0-4.243-4.243L6.586 4.672z"
                    />
                </svg>
                Link
            </button>
            <button
                class="toolbar-btn"
                on:click={handleCopy}
                title="Copy Selection"
            >
                <svg
                    class="toolbar-icon"
                    viewBox="0 0 16 16"
                    fill="currentColor"
                >
                    <path
                        d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1h1a1 1 0 0 1 1 1V14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3.5a1 1 0 0 1 1-1h1v-1z"
                    />
                    <path
                        d="M9.5 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3zm-3-1A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3z"
                    />
                </svg>
                Copy
            </button>
            <button class="toolbar-btn" on:click={handlePaste} title="Paste">
                <svg
                    class="toolbar-icon"
                    viewBox="0 0 16 16"
                    fill="currentColor"
                >
                    <path
                        d="M3.5 2a.5.5 0 0 0-.5.5v12a.5.5 0 0 0 .5.5h9a.5.5 0 0 0 .5-.5v-12a.5.5 0 0 0-.5-.5H12a.5.5 0 0 1 0-1h.5A1.5 1.5 0 0 1 14 2.5v12a1.5 1.5 0 0 1-1.5 1.5h-9A1.5 1.5 0 0 1 2 14.5v-12A1.5 1.5 0 0 1 3.5 1H4a.5.5 0 0 1 0 1h-.5z"
                    />
                    <path
                        d="M10 .5a.5.5 0 0 0-.5-.5h-3a.5.5 0 0 0-.5.5.5.5 0 0 1-.5.5.5.5 0 0 0-.5.5V2a.5.5 0 0 0 .5.5h5A.5.5 0 0 0 11 2v-.5a.5.5 0 0 0-.5-.5.5.5 0 0 1-.5-.5z"
                    />
                </svg>
                Paste
            </button>
            <button
                class="toolbar-btn"
                on:click={handleClear}
                title="Clear Terminal"
            >
                <svg
                    class="toolbar-icon"
                    viewBox="0 0 16 16"
                    fill="currentColor"
                >
                    <path
                        d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"
                    />
                    <path
                        fill-rule="evenodd"
                        d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"
                    />
                </svg>
                Clear
            </button>
        </div>
    </div>

    <!-- Terminal Container -->
    <div
        class="terminal-container"
        bind:this={containerElement}
        on:click={handleContainerClick}
        on:keydown={() => {}}
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
            <svg
                class="disconnected-icon"
                viewBox="0 0 16 16"
                fill="currentColor"
            >
                <path
                    d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767L8.982 1.566zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5zm.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"
                />
            </svg>
            <span>Disconnected</span>
            <button class="reconnect-btn" on:click={handleReconnect}>
                <svg
                    class="reconnect-icon"
                    viewBox="0 0 16 16"
                    fill="currentColor"
                >
                    <path
                        d="M11.534 7h3.932a.25.25 0 0 1 .192.41l-1.966 2.36a.25.25 0 0 1-.384 0l-1.966-2.36a.25.25 0 0 1 .192-.41zm-11 2h3.932a.25.25 0 0 0 .192-.41L2.692 6.23a.25.25 0 0 0-.384 0L.342 8.59A.25.25 0 0 0 .534 9z"
                    />
                    <path
                        fill-rule="evenodd"
                        d="M8 3c-1.552 0-2.94.707-3.857 1.818a.5.5 0 1 1-.771-.636A6.002 6.002 0 0 1 13.917 7H12.9A5.002 5.002 0 0 0 8 3zM3.1 9a5.002 5.002 0 0 0 8.757 2.182.5.5 0 1 1 .771.636A6.002 6.002 0 0 1 2.083 9H3.1z"
                    />
                </svg>
                Reconnect
            </button>
        </div>
    {/if}

    {#if isSettingUp}
        <div class="setup-overlay">
            <div class="setup-content">
                <div class="setup-spinner-large"></div>
                <span class="setup-title">Installing packages...</span>
                <span class="setup-detail">{setupMessage}</span>
            </div>
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
        display: flex;
        align-items: center;
        gap: 4px;
        background: none;
        border: 1px solid transparent;
        color: var(--text-muted);
        font-size: 11px;
        font-family: var(--font-mono);
        padding: 4px 8px;
        cursor: pointer;
        transition: all 0.15s;
    }

    .toolbar-icon {
        width: 12px;
        height: 12px;
        flex-shrink: 0;
    }

    .toolbar-btn:hover:not(:disabled) {
        color: var(--text);
        background: var(--bg-tertiary);
        border-color: var(--border);
    }

    .toolbar-btn:disabled {
        cursor: default;
        opacity: 0.8;
    }

    .toolbar-btn.reconnect-btn {
        color: var(--red);
        border-color: var(--red);
        background: rgba(255, 0, 60, 0.1);
    }

    .toolbar-btn.reconnect-btn:hover {
        background: var(--red);
        color: var(--bg);
    }

    /* Terminal Container */
    .terminal-container {
        flex: 1;
        width: 100%;
        height: 0;
        min-height: 100px;
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
        width: 100% !important;
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
        width: 20px;
        height: 20px;
        color: var(--red);
    }

    .reconnect-btn {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .reconnect-icon {
        width: 14px;
        height: 14px;
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
        0%,
        100% {
            opacity: 1;
        }
        50% {
            opacity: 0.5;
        }
    }

    /* Setup Indicator */
    .setup-indicator {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 10px;
        text-transform: uppercase;
        padding: 2px 8px;
        background: rgba(0, 200, 255, 0.1);
        border: 1px solid var(--cyan, #00c8ff);
        color: var(--cyan, #00c8ff);
        animation: fadeIn 0.2s ease;
    }

    .setup-spinner {
        width: 8px;
        height: 8px;
        border: 1.5px solid rgba(0, 200, 255, 0.3);
        border-top-color: var(--cyan, #00c8ff);
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    /* Setup Overlay */
    .setup-overlay {
        position: absolute;
        bottom: 16px;
        right: 16px;
        z-index: 10;
        animation: fadeIn 0.2s ease;
    }

    .setup-content {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 16px;
        background: rgba(0, 200, 255, 0.1);
        border: 1px solid var(--cyan, #00c8ff);
        backdrop-filter: blur(4px);
    }

    .setup-spinner-large {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(0, 200, 255, 0.3);
        border-top-color: var(--cyan, #00c8ff);
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    .setup-title {
        font-size: 12px;
        color: var(--cyan, #00c8ff);
        font-weight: 500;
    }

    .setup-detail {
        font-size: 11px;
        color: var(--text-muted);
        max-width: 200px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    @keyframes fadeIn {
        from {
            opacity: 0;
        }
        to {
            opacity: 1;
        }
    }
</style>
