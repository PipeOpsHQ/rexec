<script lang="ts">
    import { onMount, onDestroy, tick } from "svelte";
    import { terminal, type TerminalSession, type SplitPane } from "$stores/terminal";

    export let session: TerminalSession;
    export let pane: SplitPane;

    let containerElement: HTMLDivElement;
    let isAttached = false;

    onMount(async () => {
        if (containerElement && pane) {
            terminal.attachSplitPane(session.id, pane.id, containerElement);
            isAttached = true;
        }
    });

    onDestroy(() => {
        // Cleanup handled by store
        isAttached = false;
    });

    // Handle container changes
    $: if (containerElement && pane && !isAttached) {
        terminal.attachSplitPane(session.id, pane.id, containerElement);
        isAttached = true;
    }

    function handleClick() {
        terminal.setActivePaneId(session.id, pane.id);
        pane.terminal?.focus();
    }

    function handleClose() {
        terminal.closeSplitPane(session.id, pane.id);
    }

    $: isActive = session.activePaneId === pane.id;
    $: isConnected = pane.ws?.readyState === WebSocket.OPEN;
</script>

<div 
    class="split-pane" 
    class:active={isActive}
    onclick={handleClick}
    onkeydown={() => {}}
    role="textbox"
    tabindex="0"
>
    <div class="split-pane-header">
        <span class="pane-label">
            <span class="status-dot" class:connected={isConnected}></span>
            Session {pane.id.slice(-4)}
        </span>
        <button class="close-pane" onclick={(e) => { e.stopPropagation(); handleClose(); }} title="Close split">
            ×
        </button>
    </div>
    <div 
        class="split-pane-terminal"
        bind:this={containerElement}
    ></div>
</div>

<style>
    .split-pane {
        display: flex;
        flex-direction: column;
        height: 100%;
        width: 100%;
        position: relative;
        background: #0a0a0a;
        border: 1px solid var(--border);
    }

    .split-pane.active {
        border-color: var(--accent);
    }

    .split-pane-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 4px 8px;
        background: #111;
        border-bottom: 1px solid var(--border);
        flex-shrink: 0;
    }

    .pane-label {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 10px;
        color: var(--text-muted);
        font-family: var(--font-mono);
        text-transform: uppercase;
    }

    .status-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: var(--red);
    }

    .status-dot.connected {
        background: var(--green);
    }

    .close-pane {
        background: none;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 2px 6px;
        font-size: 14px;
        line-height: 1;
        transition: color 0.15s;
    }

    .close-pane:hover {
        color: var(--red);
    }

    .split-pane-terminal {
        flex: 1;
        width: 100%;
        height: 0;
        min-height: 50px;
        overflow: hidden;
        padding: 4px;
    }

    .split-pane-terminal :global(.xterm) {
        height: 100% !important;
        width: 100% !important;
    }

    .split-pane-terminal :global(.xterm-viewport) {
        overflow-y: auto !important;
    }

    .split-pane-terminal :global(.xterm-screen) {
        height: 100% !important;
        width: 100% !important;
    }
</style>
