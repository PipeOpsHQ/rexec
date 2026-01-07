<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { collab } from "../stores/collab";
    import { containers } from "../stores/containers";
    import { slide, fade } from "svelte/transition";
    import PlatformIcon from "./icons/PlatformIcon.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    export let containerId: string;
    export let isOpen = false;
    export let compact = false;

    const dispatch = createEventDispatcher();

    let mode: "view" | "control" = "view";
    let maxUsers = 5;
    let isStarting = false;
    let shareCode = "";
    let shareUrl = "";
    let copied = false;

    $: session = $collab.activeSession;
    $: participants = $collab.participants;

    // Get terminal info for display
    $: terminal = $containers.containers.find((c) => c.id === containerId);
    $: terminalName = terminal?.name || containerId.slice(0, 12);
    $: terminalOS = terminal?.image || "unknown";

    async function startSession() {
        isStarting = true;
        const result = await collab.startSession(containerId, mode, maxUsers);
        isStarting = false;

        if (result) {
            shareCode = result.shareCode;
            shareUrl = `${window.location.origin}/join/${shareCode}`;
            collab.connectWebSocket(shareCode);
        }
    }

    async function endSession() {
        if (session) {
            await collab.endSession(session.id);
        }
        close();
    }

    function copyLink() {
        navigator.clipboard.writeText(shareUrl);
        copied = true;
        setTimeout(() => (copied = false), 2000);
    }

    function close() {
        isOpen = false;
        dispatch("close");
    }
</script>

{#if isOpen}
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div
        class="panel-overlay"
        onclick={(e) => {
            if (e.target === e.currentTarget) close();
        }}
        transition:fade={{ duration: 150 }}
    >
        <div
            class="collab-panel"
            class:compact
            transition:slide={{ duration: 200, axis: "x" }}
        >
            <div class="panel-header">
                <div class="header-left">
                    <span class="icon-wrapper">
                        <svg
                            width="16"
                            height="16"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                        >
                            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"
                            ></path>
                            <circle cx="9" cy="7" r="4"></circle>
                            <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
                            <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
                        </svg>
                    </span>
                    <span class="title">Live Collaboration</span>
                </div>
                <button class="close-btn" onclick={close}>
                    <svg
                        width="18"
                        height="18"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                    >
                        <line x1="18" y1="6" x2="6" y2="18"></line>
                        <line x1="6" y1="6" x2="18" y2="18"></line>
                    </svg>
                </button>
            </div>

            <div class="panel-body">
                {#if !session}
                    <!-- START SESSION VIEW -->
                    <div class="start-view" in:fade={{ duration: 200 }}>
                        <div class="terminal-summary">
                            <div class="terminal-icon">
                                <PlatformIcon platform={terminalOS} size={24} />
                            </div>
                            <div class="terminal-meta">
                                <span class="t-label">Sharing Terminal</span>
                                <span class="t-name">{terminalName}</span>
                            </div>
                        </div>

                        <div class="section">
                            <span class="section-label">Access Mode</span>
                            <div class="mode-cards">
                                <button
                                    class="mode-card"
                                    class:selected={mode === "view"}
                                    onclick={() => (mode = "view")}
                                >
                                    <div class="card-header">
                                        <span class="card-icon view">
                                            <svg
                                                width="16"
                                                height="16"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                stroke-width="2"
                                            >
                                                <path
                                                    d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"
                                                />
                                                <circle cx="12" cy="12" r="3" />
                                            </svg>
                                        </span>
                                        <span class="card-title">View Only</span
                                        >
                                    </div>
                                    <p class="card-desc">
                                        Collaborators can watch but cannot
                                        interact.
                                    </p>
                                </button>

                                <button
                                    class="mode-card"
                                    class:selected={mode === "control"}
                                    onclick={() => (mode = "control")}
                                >
                                    <div class="card-header">
                                        <span class="card-icon control">
                                            <svg
                                                width="16"
                                                height="16"
                                                viewBox="0 0 24 24"
                                                fill="none"
                                                stroke="currentColor"
                                                stroke-width="2"
                                            >
                                                <polygon
                                                    points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"
                                                />
                                            </svg>
                                        </span>
                                        <span class="card-title"
                                            >Full Control</span
                                        >
                                    </div>
                                    <p class="card-desc">
                                        Collaborators can type and execute
                                        commands.
                                    </p>
                                </button>
                            </div>
                        </div>

                        <div class="section">
                            <div class="slider-header">
                                <span class="section-label"
                                    >Max Participants</span
                                >
                                <span class="slider-val">{maxUsers}</span>
                            </div>
                            <input
                                type="range"
                                min="2"
                                max="10"
                                bind:value={maxUsers}
                                class="slider"
                            />
                        </div>

                        <div class="actions">
                            <button
                                class="btn-primary start-btn"
                                onclick={startSession}
                                disabled={isStarting}
                            >
                                {#if isStarting}
                                    <span class="spinner"></span> Starting...
                                {:else}
                                    Start Session
                                {/if}
                            </button>
                        </div>
                    </div>
                {:else}
                    <!-- ACTIVE SESSION VIEW -->
                    <div class="active-view" in:fade={{ duration: 200 }}>
                        <div
                            class="live-banner"
                            class:control={mode === "control"}
                        >
                            <span class="live-indicator">
                                <span class="pulse-dot"></span>
                                LIVE
                            </span>
                            <span class="session-mode">
                                {mode === "control"
                                    ? "Full Control Enabled"
                                    : "View Only Mode"}
                            </span>
                        </div>

                        <div class="section share-section">
                            <span class="section-label">Share Link</span>
                            <div class="share-box">
                                <input
                                    class="share-input"
                                    readonly
                                    value={shareUrl}
                                    onclick={(e) => e.currentTarget.select()}
                                />
                                <button
                                    class="copy-btn"
                                    onclick={copyLink}
                                    class:copied
                                >
                                    {#if copied}
                                        <span>✓ Copied</span>
                                    {:else}
                                        <svg
                                            width="14"
                                            height="14"
                                            viewBox="0 0 24 24"
                                            fill="none"
                                            stroke="currentColor"
                                            stroke-width="2"
                                        >
                                            <rect
                                                x="9"
                                                y="9"
                                                width="13"
                                                height="13"
                                                rx="2"
                                                ry="2"
                                            ></rect>
                                            <path
                                                d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
                                            ></path>
                                        </svg>
                                    {/if}
                                </button>
                            </div>

                            <!-- How it works info -->
                            <div class="how-it-works">
                                <div class="how-it-works-header">
                                    <StatusIcon status="info" size={14} />
                                    <span>How Collaboration Works</span>
                                </div>
                                <ul class="how-it-works-list">
                                    <li>
                                        <span class="step-num">1</span>
                                        <span
                                            >Share the link above with your
                                            collaborators</span
                                        >
                                    </li>
                                    <li>
                                        <span class="step-num">2</span>
                                        <span
                                            >They'll join instantly - no account
                                            required for viewing</span
                                        >
                                    </li>
                                    <li>
                                        <span class="step-num">3</span>
                                        <span
                                            >{mode === "control"
                                                ? "Everyone can type commands in real-time"
                                                : "Collaborators watch as you work"}</span
                                        >
                                    </li>
                                    <li>
                                        <span class="step-num">4</span>
                                        <span
                                            >Session expires in 60 minutes or
                                            when you end it</span
                                        >
                                    </li>
                                </ul>
                                <div class="how-it-works-tip">
                                    <strong>Tip:</strong>
                                    {mode === "control"
                                        ? "Great for pair programming and interactive debugging!"
                                        : "Perfect for demos, teaching, and code reviews!"}
                                </div>
                            </div>
                        </div>

                        <div class="section participants-section">
                            <div class="section-header">
                                <span class="section-label">Participants</span>
                                <span class="p-count"
                                    >{participants.length} / {maxUsers}</span
                                >
                            </div>

                            <div class="participants-list">
                                {#each participants as p}
                                    <div class="participant-row">
                                        <div
                                            class="p-avatar"
                                            style="background-color: {p.color}"
                                        >
                                            {p.username
                                                .slice(0, 1)
                                                .toUpperCase()}
                                        </div>
                                        <div class="p-info">
                                            <span class="p-name">
                                                {p.username}
                                                {#if p.role === "owner"}<span
                                                        class="p-badge owner"
                                                        >HOST</span
                                                    >{/if}
                                            </span>
                                            <span class="p-role">{p.role}</span>
                                        </div>
                                        <div class="p-status">
                                            <span class="p-dot"></span>
                                        </div>
                                    </div>
                                {:else}
                                    <div class="empty-state">
                                        Waiting for others to join...
                                    </div>
                                {/each}
                            </div>
                        </div>

                        <div class="danger-zone">
                            <button class="btn-danger" onclick={endSession}>
                                End Session
                            </button>
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </div>
{/if}

<style>
    /* Base & Layout */
    .panel-overlay {
        position: fixed;
        inset: 0;
        background: var(--overlay-light);
        backdrop-filter: blur(2px);
        display: flex;
        justify-content: flex-end; /* Side panel style */
        z-index: 9999;
    }

    .collab-panel {
        width: 360px;
        height: 100%;
        background: var(--bg-card);
        border-left: 1px solid var(--border);
        display: flex;
        flex-direction: column;
        box-shadow: var(--shadow-soft);
    }

    .collab-panel.compact {
        width: 300px;
    }

    /* Header */
    .panel-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
        background: var(--bg-secondary);
    }

    .header-left {
        display: flex;
        align-items: center;
        gap: 10px;
        color: var(--text);
    }

    .icon-wrapper {
        color: var(--accent);
        display: flex;
    }

    .title {
        font-size: 14px;
        font-weight: 600;
        letter-spacing: 0.5px;
        text-transform: uppercase;
    }

    .close-btn {
        background: transparent;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 4px;
        border-radius: 4px;
        transition: all 0.2s;
        display: flex;
    }

    .close-btn:hover {
        background: var(--bg-tertiary);
        color: var(--text);
    }

    /* Body */
    .panel-body {
        flex: 1;
        overflow-y: auto;
        padding: 20px;
        display: flex;
        flex-direction: column;
    }

    .section {
        margin-bottom: 24px;
    }

    .section-label {
        display: block;
        font-size: 11px;
        text-transform: uppercase;
        color: var(--text-muted);
        font-weight: 600;
        margin-bottom: 10px;
        letter-spacing: 0.5px;
    }

    /* Start View Components */
    .terminal-summary {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 6px;
        margin-bottom: 24px;
    }

    .terminal-icon {
        width: 36px;
        height: 36px;
        background: var(--bg-card-hover);
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .terminal-meta {
        display: flex;
        flex-direction: column;
    }

    .t-label {
        font-size: 10px;
        color: #a0a0a0;
    }

    .t-name {
        font-size: 13px;
        color: #fff;
        font-weight: 500;
    }

    .mode-cards {
        display: grid;
        gap: 10px;
    }

    .mode-card {
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 6px;
        padding: 12px;
        text-align: left;
        cursor: pointer;
        transition: all 0.2s;
    }

    .mode-card:hover {
        border-color: var(--text-muted);
        background: var(--bg-card-hover);
    }

    .mode-card.selected {
        border-color: var(--accent);
        background: var(--accent-dim);
    }

    .card-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 4px;
    }

    .card-icon {
        font-size: 14px;
    }

    .card-title {
        font-size: 13px;
        font-weight: 600;
        color: var(--text);
    }

    .mode-card.selected .card-title {
        color: var(--accent);
    }

    .card-desc {
        font-size: 11px;
        color: var(--text-muted);
        margin: 0;
        line-height: 1.4;
    }

    .slider-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;
    }

    .slider-val {
        font-size: 12px;
        color: var(--accent);
        font-weight: 600;
    }

    .slider {
        width: 100%;
        height: 4px;
        background: var(--border);
        border-radius: 2px;
        -webkit-appearance: none;
    }

    .slider::-webkit-slider-thumb {
        -webkit-appearance: none;
        width: 16px;
        height: 16px;
        background: var(--accent);
        border-radius: 50%;
        cursor: pointer;
        border: 2px solid var(--bg-card);
    }

    .btn-primary {
        width: 100%;
        padding: 12px;
        background: var(--accent);
        color: var(--bg);
        border: none;
        border-radius: 6px;
        font-size: 13px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        cursor: pointer;
        transition: all 0.2s;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
    }

    .btn-primary:hover:not(:disabled) {
        background: var(--accent-hover);
        transform: translateY(-1px);
    }

    .btn-primary:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    /* Active View Components */
    .live-banner {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 10px 16px;
        background: rgba(0, 255, 157, 0.1);
        border: 1px solid rgba(0, 255, 157, 0.2);
        border-radius: 6px;
        margin-bottom: 24px;
    }

    .live-banner.control {
        background: rgba(255, 166, 0, 0.1);
        border-color: rgba(255, 166, 0, 0.2);
    }

    .live-indicator {
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--accent, #00ff9d);
        font-weight: 700;
        font-size: 11px;
        letter-spacing: 1px;
    }

    .live-banner.control .live-indicator {
        color: #ffa600;
    }

    .pulse-dot {
        width: 8px;
        height: 8px;
        background: currentColor;
        border-radius: 50%;
        animation: pulse 1.5s infinite;
    }

    @keyframes pulse {
        0% {
            opacity: 1;
            transform: scale(1);
        }
        50% {
            opacity: 0.5;
            transform: scale(0.9);
        }
        100% {
            opacity: 1;
            transform: scale(1);
        }
    }

    .session-mode {
        font-size: 10px;
        color: #aaa;
        text-transform: uppercase;
    }

    .share-box {
        display: flex;
        gap: 8px;
    }

    .share-input {
        flex: 1;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        padding: 10px 12px;
        border-radius: 4px;
        color: var(--text);
        font-family: var(--font-mono, monospace);
        font-size: 12px;
        outline: none;
    }

    .share-input:focus {
        border-color: var(--accent);
    }

    .copy-btn {
        background: var(--bg-card-hover);
        border: none;
        border-radius: 4px;
        color: var(--text);
        padding: 0 12px;
        font-size: 11px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
        min-width: 80px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .copy-btn:hover {
        background: var(--bg-tertiary);
    }

    .copy-btn.copied {
        background: var(--accent);
        color: var(--bg);
    }

    /* How it works section */
    .how-it-works {
        margin-top: 16px;
        padding: 14px;
        background: linear-gradient(135deg, var(--accent-dim), transparent);
        border: 1px solid var(--accent-dim);
        border-radius: 8px;
    }

    .how-it-works-header {
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--accent);
        font-size: 12px;
        font-weight: 600;
        margin-bottom: 12px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .how-it-works-list {
        list-style: none;
        padding: 0;
        margin: 0 0 12px 0;
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .how-it-works-list li {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        font-size: 11px;
        color: var(--text-secondary);
        line-height: 1.4;
    }

    .step-num {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 18px;
        height: 18px;
        min-width: 18px;
        background: var(--accent-dim);
        color: var(--accent);
        border-radius: 50%;
        font-size: 10px;
        font-weight: 700;
    }

    .how-it-works-tip {
        font-size: 10px;
        color: var(--text-muted);
        padding-top: 10px;
        border-top: 1px solid var(--border-muted);
        line-height: 1.4;
    }

    .how-it-works-tip strong {
        color: var(--text-secondary);
    }

    .participants-section {
        flex: 1;
        display: flex;
        flex-direction: column;
        min-height: 0;
    }

    .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
    }

    .p-count {
        font-size: 11px;
        color: var(--text-muted);
    }

    .participants-list {
        flex: 1;
        overflow-y: auto;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 6px;
        padding: 4px;
    }

    .participant-row {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 8px;
        border-bottom: 1px solid var(--border-muted);
    }

    .participant-row:last-child {
        border-bottom: none;
    }

    .p-avatar {
        width: 28px;
        height: 28px;
        border-radius: 50%;
        background: #333;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 11px;
        font-weight: 700;
        color: #000;
    }

    .p-info {
        flex: 1;
        display: flex;
        flex-direction: column;
    }

    .p-name {
        font-size: 12px;
        color: #e0e0e0;
        font-weight: 500;
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .p-badge {
        font-size: 8px;
        padding: 1px 4px;
        border-radius: 2px;
        background: #333;
        color: #aaa;
    }

    .p-badge.owner {
        background: rgba(255, 215, 0, 0.15);
        color: #ffd700;
    }

    .p-role {
        font-size: 10px;
        color: #a0a0a0;
        text-transform: capitalize;
    }

    .p-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: #00ff9d;
        box-shadow: 0 0 4px rgba(0, 255, 157, 0.4);
    }

    .empty-state {
        padding: 20px;
        text-align: center;
        color: #a0a0a0;
        font-size: 12px;
        font-style: italic;
    }

    .danger-zone {
        margin-top: auto;
        padding-top: 20px;
        border-top: 1px solid #1e1e24;
    }

    .btn-danger {
        width: 100%;
        padding: 10px;
        background: rgba(255, 68, 68, 0.1);
        border: 1px solid rgba(255, 68, 68, 0.2);
        color: #ff4444;
        border-radius: 6px;
        font-size: 12px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-danger:hover {
        background: rgba(255, 68, 68, 0.2);
        border-color: rgba(255, 68, 68, 0.4);
    }

    .spinner {
        display: inline-block;
        width: 12px;
        height: 12px;
        border: 2px solid rgba(0, 0, 0, 0.3);
        border-radius: 50%;
        border-top-color: #000;
        animation: spin 1s ease-in-out infinite;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }
</style>
