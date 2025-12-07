<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { auth } from "$stores/auth";
    import { toast } from "$stores/toast";
    import StatusIcon from "./icons/StatusIcon.svelte";

    const dispatch = createEventDispatcher<{
        guest: void;
        navigate: { view: string };
    }>();

    let isOAuthLoading = false;

    function handleGuestClick() {
        dispatch("guest");
    }

    function navigateTo(view: string) {
        dispatch("navigate", { view });
    }

    async function handleOAuthLogin() {
        if (isOAuthLoading) return;

        isOAuthLoading = true;
        try {
            const url = await auth.getOAuthUrl();
            if (url) {
                window.location.href = url;
            } else {
                toast.error(
                    "Unable to connect to PipeOps. Please try again later.",
                );
                isOAuthLoading = false;
            }
        } catch (e) {
            toast.error("Failed to connect to PipeOps. Please try again.");
            isOAuthLoading = false;
        }
    }
</script>

<div class="landing">
    <div class="landing-content">
        <div class="landing-badge">
            <span class="dot"></span>
            <span>Terminal as a Service</span>
        </div>

        <h1>
            Instant <span class="accent">Linux</span> Terminals
            <br />
            In Your Browser
        </h1>

        <p class="description">
            Create your first terminal to access a cloud environment, GPU workspace, 
            or connect to remote resources. No setup required.
        </p>

        <div class="landing-actions">
            <button class="btn btn-primary btn-lg" on:click={handleGuestClick}>
                Try Now — No Sign Up
            </button>
            <button
                class="btn btn-secondary btn-lg"
                on:click={handleOAuthLogin}
                disabled={isOAuthLoading}
            >
                {#if isOAuthLoading}
                    <span class="btn-spinner"></span>
                    Connecting...
                {:else}
                    Sign in with PipeOps
                {/if}
            </button>
        </div>

        <div class="landing-links">
            <button class="link-btn" on:click={() => navigateTo('use-cases')}>
                <StatusIcon status="bolt" size={14} /> Use Cases
            </button>
            <span class="divider"></span>
            <button class="link-btn" on:click={() => navigateTo('guides')}>
                <StatusIcon status="book" size={14} /> Product Guide
            </button>
        </div>

        <div class="terminal-preview">
            <div class="terminal-preview-header">
                <span class="terminal-dot dot-red"></span>
                <span class="terminal-dot dot-yellow"></span>
                <span class="terminal-dot dot-green"></span>
                <span class="terminal-title">ubuntu-24 — rexec</span>
            </div>
            <div class="terminal-preview-body">
                <div class="terminal-line">
                    <span class="prompt">root@rexec:~#</span>
                    <span class="command">whoami</span>
                </div>
                <div class="terminal-output">root</div>
                <div class="terminal-line">
                    <span class="prompt">root@rexec:~#</span>
                    <span class="command">uname -a</span>
                </div>
                <div class="terminal-output">
                    Linux rexec 6.5.0-44-generic #44-Ubuntu SMP x86_64 GNU/Linux
                </div>
                <div class="terminal-line">
                    <span class="prompt">root@rexec:~#</span>
                    <span class="cursor">_</span>
                </div>
            </div>
        </div>

        <div class="features">
            <div class="feature">
                <span class="feature-icon"><StatusIcon status="bolt" size={24} /></span>
                <h3>Instant</h3>
                <p>Rexec terminals launch in seconds with pre-configured shells</p>
            </div>
            <div class="feature">
                <span class="feature-icon"><StatusIcon status="connected" size={24} /></span>
                <h3>Isolated</h3>
                <p>Each terminal is fully isolated with its own filesystem</p>
            </div>
            <div class="feature">
                <span class="feature-icon"><StatusIcon status="terminal" size={24} /></span>
                <h3>Accessible</h3>
                <p>Access from any browser, anywhere. SSH support included</p>
            </div>
        </div>
    </div>
</div>

<style>
    .landing {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        min-height: calc(100vh - 120px);
        text-align: center;
        border: 1px solid var(--border);
        background: rgba(10, 10, 10, 0.5);
        position: relative;
        padding: 40px;
    }

    .landing::before {
        content: "";
        position: absolute;
        top: -1px;
        left: -1px;
        width: 10px;
        height: 10px;
        border-top: 2px solid var(--accent);
        border-left: 2px solid var(--accent);
    }

    .landing::after {
        content: "";
        position: absolute;
        bottom: -1px;
        right: -1px;
        width: 10px;
        height: 10px;
        border-bottom: 2px solid var(--accent);
        border-right: 2px solid var(--accent);
    }

    .landing-content {
        max-width: 800px;
        width: 100%;
    }

    .landing-badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 4px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        font-size: 11px;
        color: var(--text-secondary);
        margin-bottom: 24px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .landing-badge .dot {
        width: 6px;
        height: 6px;
        background: var(--accent);
        animation: blink 1s step-end infinite;
    }

    h1 {
        font-size: 36px;
        font-weight: 700;
        margin-bottom: 20px;
        text-transform: uppercase;
        letter-spacing: 2px;
        line-height: 1.3;
    }

    h1 .accent {
        color: var(--accent);
        text-shadow: var(--accent-glow);
    }

    .description {
        font-size: 14px;
        color: var(--text-muted);
        max-width: 500px;
        margin: 0 auto 40px;
        line-height: 1.6;
    }

    .landing-actions {
        display: flex;
        gap: 16px;
        justify-content: center;
        margin-bottom: 40px;
    }

    .landing-links {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 16px;
        margin-bottom: 40px;
    }

    .link-btn {
        background: none;
        border: none;
        color: var(--text-secondary);
        font-size: 13px;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        border-radius: 6px;
        transition: all 0.2s;
        border: 1px solid transparent;
    }

    .link-btn:hover {
        color: var(--text);
        background: var(--bg-card);
        border-color: var(--border);
    }

    .divider {
        width: 4px;
        height: 4px;
        background: var(--border);
        border-radius: 50%;
    }

    .btn-spinner {
        display: inline-block;
        width: 14px;
        height: 14px;
        border: 2px solid transparent;
        border-top-color: currentColor;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-right: 8px;
        vertical-align: middle;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    .btn:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .terminal-preview {
        width: 100%;
        max-width: 600px;
        margin: 0 auto 40px;
        background: #000;
        border: 1px solid var(--border);
        text-align: left;
    }

    .terminal-preview-header {
        display: flex;
        align-items: center;
        padding: 8px 12px;
        background: #111;
        border-bottom: 1px solid var(--border);
        gap: 6px;
    }

    .terminal-dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        background: var(--border);
    }

    .terminal-dot.dot-red {
        background: #ff5f56;
    }

    .terminal-dot.dot-yellow {
        background: #ffbd2e;
    }

    .terminal-dot.dot-green {
        background: #27c93f;
    }

    .terminal-title {
        flex: 1;
        text-align: center;
        font-size: 11px;
        color: var(--text-muted);
    }

    .terminal-preview-body {
        padding: 16px;
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .terminal-line {
        margin-bottom: 4px;
    }

    .prompt {
        color: var(--accent);
        margin-right: 8px;
    }

    .command {
        color: var(--text);
    }

    .terminal-output {
        color: var(--text-muted);
        margin-bottom: 8px;
        padding-left: 16px;
    }

    .cursor {
        background: var(--accent);
        color: var(--bg);
        animation: blink 1s step-end infinite;
    }

    .features {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 16px;
    }

    .feature {
        padding: 20px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        text-align: left;
        transition: border-color 0.2s;
    }

    .feature:hover {
        border-color: var(--accent);
    }

    .feature-icon {
        font-size: 24px;
        display: block;
        margin-bottom: 12px;
    }

    .feature h3 {
        font-size: 14px;
        text-transform: uppercase;
        margin-bottom: 8px;
        color: var(--text);
        letter-spacing: 0.5px;
    }

    .feature p {
        font-size: 12px;
        color: var(--text-muted);
        line-height: 1.5;
    }

    @keyframes blink {
        0%,
        100% {
            opacity: 1;
        }
        50% {
            opacity: 0;
        }
    }

    @media (max-width: 768px) {
        h1 {
            font-size: 24px;
        }

        .landing-actions {
            flex-direction: column;
        }

        .features {
            grid-template-columns: 1fr;
        }
    }
</style>