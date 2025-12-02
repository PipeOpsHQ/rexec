<script lang="ts">
    import { createEventDispatcher, onMount, onDestroy } from "svelte";
    import {
        auth,
        isAuthenticated,
        isGuest,
        sessionExpiresAt,
    } from "$stores/auth";
    import { terminal, sessionCount } from "$stores/terminal";
    import { toast } from "$stores/toast";

    const dispatch = createEventDispatcher<{
        home: void;
        create: void;
        settings: void;
        sshkeys: void;
        guest: void;
    }>();

    let showUserMenu = false;
    let timeRemaining = "";
    let countdownInterval: ReturnType<typeof setInterval> | null = null;
    let isOAuthLoading = false;

    function updateCountdown() {
        if (!$sessionExpiresAt) {
            timeRemaining = "";
            return;
        }

        const now = Math.floor(Date.now() / 1000);
        const remaining = $sessionExpiresAt - now;

        if (remaining <= 0) {
            timeRemaining = "Expired";
            // Close all terminal sessions when guest session expires
            terminal.closeAllSessionsForce();
            toast.error("Guest session expired. Please sign in again.");
            auth.logout();
            return;
        }

        const hours = Math.floor(remaining / 3600);
        const minutes = Math.floor((remaining % 3600) / 60);
        const seconds = remaining % 60;

        if (hours > 0) {
            timeRemaining = `${hours}h ${minutes}m`;
        } else if (minutes > 0) {
            timeRemaining = `${minutes}m ${seconds}s`;
        } else {
            timeRemaining = `${seconds}s`;
        }
    }

    onMount(() => {
        if ($isGuest && $sessionExpiresAt) {
            updateCountdown();
            countdownInterval = setInterval(updateCountdown, 1000);
        }
    });

    onDestroy(() => {
        if (countdownInterval) {
            clearInterval(countdownInterval);
        }
    });

    // Reactively start/stop countdown when guest status changes
    $: if ($isGuest && $sessionExpiresAt && !countdownInterval) {
        updateCountdown();
        countdownInterval = setInterval(updateCountdown, 1000);
    } else if (!$isGuest && countdownInterval) {
        clearInterval(countdownInterval);
        countdownInterval = null;
        timeRemaining = "";
    }

    function handleGuestClick() {
        dispatch("guest");
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

    function handleLogout() {
        showUserMenu = false;
        auth.logout();
    }

    function toggleUserMenu() {
        showUserMenu = !showUserMenu;
    }

    function closeUserMenu(event: MouseEvent) {
        const target = event.target as HTMLElement;
        if (!target.closest(".user-menu-container")) {
            showUserMenu = false;
        }
    }
</script>

<svelte:window on:click={closeUserMenu} />

<header class="header">
    <button class="logo" on:click={() => dispatch("home")}>
        <span class="logo-icon">R</span>
        <span class="logo-text">Rexec</span>
        {#if $sessionCount > 0}
            <span class="session-badge">{$sessionCount}</span>
        {/if}
    </button>

    <nav class="nav-actions">
        {#if $isAuthenticated}
            {#if $sessionCount > 0}
                <span class="terminal-status">
                    <span class="terminal-dot"></span>
                    {$sessionCount} Active
                </span>
            {/if}

            <div class="user-menu-container">
                <button
                    class="user-badge"
                    on:click|stopPropagation={toggleUserMenu}
                >
                    <span class="user-avatar">
                        {$auth.user?.name?.charAt(0).toUpperCase() || "U"}
                    </span>
                    <span class="user-name">
                        {$auth.user?.name || "User"}
                    </span>
                    {#if $isGuest}
                        <span class="guest-badge">Guest</span>
                        {#if timeRemaining}
                            <span
                                class="countdown-badge"
                                class:warning={timeRemaining.includes("m") &&
                                    !timeRemaining.includes("h")}
                                class:danger={timeRemaining.includes("s") &&
                                    !timeRemaining.includes("m")}
                            >
                                ⏱ {timeRemaining}
                            </span>
                        {/if}
                    {/if}
                    <span class="dropdown-arrow">▼</span>
                </button>

                {#if showUserMenu}
                    <div class="user-menu">
                        <div class="user-menu-header">
                            <span class="user-menu-name"
                                >{$auth.user?.name || "User"}</span
                            >
                            {#if $auth.user?.email}
                                <span class="user-menu-email"
                                    >{$auth.user.email}</span
                                >
                            {/if}
                            <span class="user-menu-tier">
                                {$auth.user?.tier?.toUpperCase() || "FREE"} Plan
                            </span>
                        </div>

                        <div class="user-menu-divider"></div>

                        {#if !$isGuest}
                            <button
                                class="user-menu-item"
                                on:click={() => {
                                    showUserMenu = false;
                                    dispatch("home");
                                }}
                            >
                                <span>📊</span>
                                Dashboard
                            </button>
                        {/if}
                        <button
                            class="user-menu-item"
                            class:disabled={$isGuest}
                            disabled={$isGuest}
                            title={$isGuest
                                ? "Sign in with PipeOps to access Settings"
                                : ""}
                            on:click={() => {
                                if (!$isGuest) {
                                    showUserMenu = false;
                                    dispatch("settings");
                                }
                            }}
                        >
                            <span>⚙️</span>
                            Settings
                            {#if $isGuest}<span class="lock-icon">🔒</span>{/if}
                        </button>
                        <button
                            class="user-menu-item"
                            class:disabled={$isGuest}
                            disabled={$isGuest}
                            title={$isGuest
                                ? "Sign in with PipeOps to manage SSH Keys"
                                : ""}
                            on:click={() => {
                                if (!$isGuest) {
                                    showUserMenu = false;
                                    dispatch("sshkeys");
                                }
                            }}
                        >
                            <span>🔑</span>
                            SSH Keys
                            {#if $isGuest}<span class="lock-icon">🔒</span>{/if}
                        </button>
                        {#if $isGuest}
                            <div class="user-menu-divider"></div>
                            <button
                                class="user-menu-item accent"
                                on:click={handleOAuthLogin}
                                disabled={isOAuthLoading}
                            >
                                {#if isOAuthLoading}
                                    <span class="btn-spinner-sm"></span>
                                    Connecting...
                                {:else}
                                    <span>🔗</span>
                                    Sign in with PipeOps
                                {/if}
                            </button>
                        {/if}

                        <div class="user-menu-divider"></div>

                        <button
                            class="user-menu-item danger"
                            on:click={handleLogout}
                        >
                            <span>🚪</span>
                            Sign Out
                        </button>
                    </div>
                {/if}
            </div>
        {:else}
            <button
                class="btn btn-secondary btn-sm"
                on:click={handleGuestClick}
            >
                Try as Guest
            </button>
            <button
                class="btn btn-primary btn-sm"
                on:click={handleOAuthLogin}
                disabled={isOAuthLoading}
            >
                {#if isOAuthLoading}
                    <span class="btn-spinner-sm"></span>
                    Connecting...
                {:else}
                    Sign in with PipeOps
                {/if}
            </button>
        {/if}
    </nav>
</header>

<style>
    .btn-spinner-sm {
        display: inline-block;
        width: 12px;
        height: 12px;
        border: 2px solid transparent;
        border-top-color: currentColor;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-right: 6px;
        vertical-align: middle;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    .btn:disabled,
    .user-menu-item:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 20px;
        border-bottom: 1px solid var(--border);
        background: rgba(5, 5, 5, 0.95);
        backdrop-filter: blur(5px);
        position: sticky;
        top: 0;
        z-index: 100;
    }

    .logo {
        display: flex;
        align-items: center;
        gap: 12px;
        font-weight: 700;
        font-size: 16px;
        color: var(--accent);
        text-decoration: none;
        text-transform: uppercase;
        letter-spacing: 1px;
        background: none;
        border: none;
        cursor: pointer;
        padding: 0;
    }

    .logo-icon {
        width: 24px;
        height: 24px;
        background: var(--accent);
        color: var(--bg);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 14px;
        font-weight: bold;
        box-shadow: var(--accent-glow);
    }

    .logo-text {
        color: var(--accent);
    }

    .session-badge {
        background: var(--accent);
        color: var(--bg);
        font-size: 10px;
        padding: 2px 6px;
        font-weight: 600;
    }

    .nav-actions {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .terminal-status {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.3);
        font-size: 11px;
        font-family: var(--font-mono);
        color: var(--accent);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .terminal-dot {
        width: 6px;
        height: 6px;
        background: var(--accent);
        border-radius: 50%;
        animation: pulse 1.5s ease-in-out infinite;
    }

    .user-menu-container {
        position: relative;
    }

    .user-badge {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 4px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        cursor: pointer;
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--text);
        transition: border-color 0.2s;
    }

    .user-badge:hover {
        border-color: var(--accent);
    }

    .user-avatar {
        width: 20px;
        height: 20px;
        background: var(--accent);
        color: var(--bg);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 10px;
        font-weight: bold;
    }

    .user-name {
        color: var(--text);
    }

    .guest-badge {
        background: var(--warning);
        color: var(--bg);
        font-size: 9px;
        padding: 1px 4px;
        text-transform: uppercase;
    }

    .countdown-badge {
        background: var(--bg-tertiary);
        color: var(--accent);
        font-size: 10px;
        padding: 2px 6px;
        font-family: var(--font-mono);
        border: 1px solid var(--border);
    }

    .countdown-badge.warning {
        color: var(--warning);
        border-color: var(--warning);
        background: rgba(255, 200, 0, 0.1);
    }

    .countdown-badge.danger {
        color: var(--danger);
        border-color: var(--danger);
        background: rgba(255, 0, 60, 0.1);
        animation: pulse 1s infinite;
    }

    @keyframes pulse {
        0%,
        100% {
            opacity: 1;
        }
        50% {
            opacity: 0.6;
        }
    }

    .dropdown-arrow {
        font-size: 8px;
        color: var(--text-muted);
        margin-left: 4px;
    }

    .user-menu {
        position: absolute;
        top: calc(100% + 8px);
        right: 0;
        min-width: 220px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        z-index: 200;
        animation: fadeIn 0.15s ease;
    }

    .user-menu-header {
        padding: 12px 16px;
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .user-menu-name {
        font-weight: 600;
        color: var(--text);
    }

    .user-menu-email {
        font-size: 11px;
        color: var(--text-muted);
    }

    .user-menu-tier {
        font-size: 10px;
        color: var(--accent);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .user-menu-divider {
        height: 1px;
        background: var(--border);
    }

    .user-menu-item {
        display: flex;
        align-items: center;
        gap: 10px;
        width: 100%;
        padding: 10px 16px;
        background: none;
        border: none;
        color: var(--text-secondary);
        font-size: 12px;
        font-family: var(--font-mono);
        text-align: left;
        cursor: pointer;
        transition: all 0.15s;
    }

    .user-menu-item:hover {
        background: var(--bg-tertiary);
        color: var(--text);
    }

    .user-menu-item.accent {
        color: var(--accent);
    }

    .user-menu-item.accent:hover {
        background: var(--accent-dim);
    }

    .user-menu-item.danger {
        color: var(--danger);
    }

    .user-menu-item.danger:hover {
        background: rgba(255, 0, 60, 0.1);
    }

    .user-menu-item.disabled {
        opacity: 0.5;
        cursor: not-allowed;
        color: var(--text-muted);
    }

    .user-menu-item.disabled:hover {
        background: none;
        color: var(--text-muted);
    }

    .lock-icon {
        margin-left: auto;
        font-size: 10px;
        opacity: 0.7;
    }

    @keyframes fadeIn {
        from {
            opacity: 0;
            transform: translateY(-4px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
</style>
