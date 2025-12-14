<script lang="ts">
    import { createEventDispatcher, onMount, onDestroy } from "svelte";
    import {
        auth,
        isAuthenticated,
        isGuest,
        isAdmin,
        sessionExpiresAt,
    } from "$stores/auth";
    import { toast } from "$stores/toast";
    import { theme } from "$stores/theme";
    import StatusIcon from "./icons/StatusIcon.svelte";
    import InstallButton from "./InstallButton.svelte";

    const dispatch = createEventDispatcher<{
        home: void;
        create: void;
        settings: void;
        sshkeys: void;
        snippets: void;
        billing: void;
        agents: void;
        cli: void;
        guest: void;
        pricing: void;
        guides: void;
        usecases: void;
        admin: void;
        account: void;
    }>();

    let showUserMenu = false;
    let showMobileMenu = false;
    let timeRemaining = "";
    let countdownInterval: ReturnType<typeof setInterval> | null = null;
    let isOAuthLoading = false;
    let sessionCountValue = 0;
    let terminalModulePromise: Promise<typeof import("$stores/terminal")> | null =
        null;
    let terminalModule: typeof import("$stores/terminal") | null = null;
    let unsubscribeSessionCount: (() => void) | null = null;

    async function ensureTerminalModule() {
        if (terminalModule) return terminalModule;
        if (!terminalModulePromise) {
            terminalModulePromise = import("$stores/terminal")
                .then((mod) => {
                    terminalModule = mod;
                    return mod;
                })
                .catch((err) => {
                    terminalModulePromise = null;
                    throw err;
                });
        }
        return terminalModulePromise;
    }

    async function ensureSessionCountSubscription() {
        if (unsubscribeSessionCount) return;
        try {
            const mod = await ensureTerminalModule();
            unsubscribeSessionCount = mod.sessionCount.subscribe((count) => {
                sessionCountValue = count;
            });
        } catch (e) {
            console.warn("[Header] Failed to load terminal store:", e);
        }
    }

    function toggleMobileMenu() {
        showMobileMenu = !showMobileMenu;
    }

    function closeMobileMenu() {
        showMobileMenu = false;
    }

    function updateCountdown() {
        if (!$sessionExpiresAt) {
            timeRemaining = "";
            return;
        }

        const now = Math.floor(Date.now() / 1000);
        const remaining = $sessionExpiresAt - now;

        if (remaining <= 0) {
            timeRemaining = "Expired";
            // Close all terminals when guest access expires
            ensureTerminalModule()
                .then(({ terminal }) => terminal.closeAllSessionsForce())
                .catch(() => {
                    // Ignore
                });
            toast.error("Guest access expired. Please sign in again.");
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
        if (unsubscribeSessionCount) {
            unsubscribeSessionCount();
            unsubscribeSessionCount = null;
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

    // Subscribe to terminal sessionCount only when authenticated (keeps landing bundle smaller)
    $: if ($isAuthenticated) {
        ensureSessionCountSubscription();
    } else if (unsubscribeSessionCount) {
        unsubscribeSessionCount();
        unsubscribeSessionCount = null;
        sessionCountValue = 0;
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

<svelte:window onclick={closeUserMenu} />

<header class="header">
    <button class="mobile-menu-btn" onclick={toggleMobileMenu}>
        <StatusIcon status="menu" size={20} />
    </button>

    <button class="logo" onclick={() => dispatch("home")}>
        <span class="logo-icon">R</span>
        <span class="logo-text">Rexec</span>
    </button>

    <nav class="nav-links">
        <button class="nav-link" onclick={() => dispatch("pricing")}>
            <StatusIcon status="pricing" size={14} />
            <span>Pricing</span>
        </button>
        <a class="nav-link" href="/marketplace">
            <StatusIcon status="snippet" size={14} />
            <span>Snippets</span>
        </a>
        <a class="nav-link" href="/docs">
            <StatusIcon status="book" size={14} />
            <span>Docs</span>
        </a>
    </nav>

    <nav class="nav-actions">
        <button
            class="theme-toggle"
            onclick={() => theme.toggle()}
            title={$theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
            {#if $theme === 'dark'}
                <StatusIcon status="sun" size={16} />
            {:else}
                <StatusIcon status="moon" size={16} />
            {/if}
        </button>
        <InstallButton />
        {#if $isAuthenticated}
            {#if sessionCountValue > 0}
                <span class="terminal-status">
                    <span class="terminal-dot"></span>
                    {sessionCountValue} Terminal{sessionCountValue > 1 ? 's' : ''}
                </span>
            {/if}

            <div class="user-menu-container">
                <button
                    class="user-badge"
                    onclick={(e) => { e.stopPropagation(); toggleUserMenu(); }}
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
                                <StatusIcon status="clock" size={10} />
                                {timeRemaining}
                            </span>
                        {/if}
                    {/if}
                    <span class="dropdown-arrow">
                        <StatusIcon status="chevron-down" size={10} />
                    </span>
                </button>

                {#if showUserMenu}
                    <div class="user-menu">
                        <div 
                            class="user-menu-header clickable"
                            onclick={() => {
                                showUserMenu = false;
                                dispatch("account");
                            }}
                        >
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

                        <!-- Navigation -->
                        {#if !$isGuest}
                            <button
                                class="user-menu-item"
                                onclick={() => {
                                    showUserMenu = false;
                                    dispatch("home");
                                }}
                            >
                                <StatusIcon status="chart" size={14} />
                                Dashboard
                            </button>
                        {/if}

                        <div class="user-menu-divider"></div>

                        <!-- Account Section -->
                        <div class="user-menu-section-label">Account</div>
                        <button
                            class="user-menu-item"
                            class:disabled={$isGuest}
                            disabled={$isGuest}
                            title={$isGuest
                                ? "Sign in with PipeOps to access Settings"
                                : ""}
                            onclick={() => {
                                if (!$isGuest) {
                                    showUserMenu = false;
                                    dispatch("settings");
                                }
                            }}
                        >
                            <StatusIcon status="settings" size={14} />
                            Settings
                            {#if $isGuest}<span class="lock-icon"><StatusIcon status="lock" size={12} /></span>{/if}
                        </button>
                        <button
                            class="user-menu-item"
                            class:disabled={$isGuest}
                            disabled={$isGuest}
                            title={$isGuest
                                ? "Sign in with PipeOps to manage SSH Keys"
                                : ""}
                            onclick={() => {
                                if (!$isGuest) {
                                    showUserMenu = false;
                                    dispatch("sshkeys");
                                }
                            }}
                        >
                            <StatusIcon status="key" size={14} />
                            SSH Keys
                            {#if $isGuest}<span class="lock-icon"><StatusIcon status="lock" size={12} /></span>{/if}
                        </button>
                        <button
                            class="user-menu-item"
                            class:disabled={$isGuest}
                            disabled={$isGuest}
                            title={$isGuest
                                ? "Sign in with PipeOps to access Billing"
                                : ""}
                            onclick={() => {
                                if (!$isGuest) {
                                    showUserMenu = false;
                                    dispatch("billing");
                                }
                            }}
                        >
                            <StatusIcon status="invoice" size={14} />
                            Billing
                            {#if $isGuest}<span class="lock-icon"><StatusIcon status="lock" size={12} /></span>{/if}
                        </button>

                        <div class="user-menu-divider"></div>

                        <!-- Tools Section -->
                        <div class="user-menu-section-label">Tools</div>
                        <button
                            class="user-menu-item"
                            class:disabled={$isGuest}
                            disabled={$isGuest}
                            onclick={() => {
                                if (!$isGuest) {
                                    showUserMenu = false;
                                    dispatch("snippets");
                                }
                            }}
                        >
                            <StatusIcon status="snippet" size={14} />
                            Snippets
                            {#if $isGuest}<span class="lock-icon"><StatusIcon status="lock" size={12} /></span>{/if}
                        </button>
                        <button
                            class="user-menu-item"
                            class:disabled={$isGuest}
                            disabled={$isGuest}
                            title={$isGuest
                                ? "Sign in with PipeOps to access CLI"
                                : ""}
                            onclick={() => {
                                if (!$isGuest) {
                                    showUserMenu = false;
                                    dispatch("cli");
                                }
                            }}
                        >
                            <StatusIcon status="terminal" size={14} />
                            CLI
                            {#if $isGuest}<span class="lock-icon"><StatusIcon status="lock" size={12} /></span>{/if}
                        </button>
                        <button
                            class="user-menu-item"
                            class:disabled={$isGuest}
                            disabled={$isGuest}
                            title={$isGuest
                                ? "Sign in with PipeOps to access Agents"
                                : ""}
                            onclick={() => {
                                if (!$isGuest) {
                                    showUserMenu = false;
                                    dispatch("agents");
                                }
                            }}
                        >
                            <StatusIcon status="server" size={14} />
                            Agents
                            {#if $isGuest}<span class="lock-icon"><StatusIcon status="lock" size={12} /></span>{/if}
                        </button>

                        <div class="user-menu-divider"></div>

                        <!-- Explore Section -->
                        <div class="user-menu-section-label">Explore</div>
                        <button
                            class="user-menu-item"
                            onclick={() => {
                                showUserMenu = false;
                                dispatch("pricing");
                            }}
                        >
                            <StatusIcon status="pricing" size={14} />
                            Pricing
                        </button>

                        {#if $isAdmin}
                            <div class="user-menu-divider"></div>
                            <button
                                class="user-menu-item"
                                onclick={() => {
                                    showUserMenu = false;
                                    dispatch("admin");
                                }}
                            >
                                <StatusIcon status="shield" size={14} />
                                Admin
                            </button>
                        {/if}

                        {#if $isGuest}
                            <div class="user-menu-divider"></div>
                            <button
                                class="user-menu-item accent"
                                onclick={handleOAuthLogin}
                                disabled={isOAuthLoading}
                            >
                                {#if isOAuthLoading}
                                    <span class="btn-spinner-sm"></span>
                                    Connecting...
                                {:else}
                                    <StatusIcon status="link" size={14} />
                                    Sign in with PipeOps
                                {/if}
                            </button>
                        {/if}

                        <div class="user-menu-divider"></div>

                        <button
                            class="user-menu-item danger"
                            onclick={handleLogout}
                        >
                            <StatusIcon status="logout" size={14} />
                            Sign Out
                        </button>
                    </div>
                {/if}
            </div>
        {:else}
            <button
                class="btn btn-secondary btn-sm"
                onclick={handleGuestClick}
            >
                Try as Guest
            </button>
            <button
                class="btn btn-primary btn-sm"
                onclick={handleOAuthLogin}
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

{#if showMobileMenu}
    <div class="mobile-menu-overlay" onclick={closeMobileMenu}>
        <div class="mobile-menu-content" onclick={(e) => e.stopPropagation()}>
            <div class="mobile-menu-header">
                <span class="logo-text">Menu</span>
                <button class="close-btn" onclick={closeMobileMenu}>
                    <StatusIcon status="close" size={20} />
                </button>
            </div>
            <div class="mobile-nav-links">
                <button class="mobile-nav-link" onclick={() => { closeMobileMenu(); dispatch("pricing"); }}>
                    <StatusIcon status="pricing" size={16} /> Pricing
                </button>
                <button class="mobile-nav-link" onclick={() => { closeMobileMenu(); dispatch("home"); }}>
                    <StatusIcon status="chart" size={16} /> Dashboard
                </button>
                <div class="user-menu-divider"></div>
                <button class="mobile-nav-link" onclick={() => { closeMobileMenu(); dispatch("create"); }}>
                    <StatusIcon status="plus" size={16} /> New Terminal
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    .theme-toggle {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 32px;
        height: 32px;
        padding: 0;
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text-muted);
        cursor: pointer;
        transition: all 0.2s;
    }

    .theme-toggle:hover {
        color: var(--accent);
        border-color: var(--accent);
    }

    .mobile-menu-btn {
        display: none;
        background: none;
        border: none;
        color: var(--text);
        cursor: pointer;
        padding: 4px;
        margin-right: 8px;
    }

    .mobile-menu-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: var(--overlay-light);
        z-index: 1000;
        display: flex;
        justify-content: flex-start;
        animation: fadeIn 0.2s ease;
    }

    .mobile-menu-content {
        width: 250px;
        background: var(--bg-card);
        height: 100%;
        border-right: 1px solid var(--border);
        display: flex;
        flex-direction: column;
        animation: slideIn 0.2s ease;
    }

    .mobile-menu-header {
        padding: 16px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-bottom: 1px solid var(--border);
    }

    .close-btn {
        background: none;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
    }

    .mobile-nav-links {
        display: flex;
        flex-direction: column;
        padding: 16px;
        gap: 12px;
    }

    .mobile-nav-link {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px;
        color: var(--text);
        text-decoration: none;
        border-radius: 6px;
        transition: background 0.15s;
        background: none;
        border: none;
        width: 100%;
        text-align: left;
        cursor: pointer;
        font-size: 14px;
    }

    .mobile-nav-link:hover {
        background: var(--bg-tertiary);
    }

    @keyframes slideIn {
        from { transform: translateX(-100%); }
        to { transform: translateX(0); }
    }

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
        background: var(--glass);
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

    .nav-links {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-left: auto;
        margin-right: 16px;
    }

    .nav-link {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 6px 12px;
        font-size: 12px;
        color: var(--text-muted);
        text-decoration: none;
        border: 1px solid transparent;
        transition: all 0.15s ease;
        background: none;
        cursor: pointer;
    }

    .nav-link:hover {
        color: var(--text);
        border-color: var(--border);
        background: rgba(255, 255, 255, 0.03);
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

    .user-menu-header.clickable {
        cursor: pointer;
        transition: background 0.15s;
    }

    .user-menu-header.clickable:hover {
        background: var(--bg-tertiary);
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

    .user-menu-section-label {
        padding: 8px 16px 4px;
        font-size: 10px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-muted);
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

    /* Mobile responsive styles */
    @media (max-width: 768px) {
        .header {
            padding: 10px 12px;
        }

        .mobile-menu-btn {
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .nav-links {
            display: none;
        }

        .logo-text {
            display: inline; /* Keep logo text visible on mobile */
            font-size: 14px;
        }

        .nav-actions {
            gap: 8px;
        }

        .terminal-status {
            padding: 3px 8px;
            font-size: 10px;
        }

        .user-badge {
            padding: 4px 8px;
            gap: 6px;
        }

        .user-name {
            display: none;
        }

        .guest-badge {
            font-size: 8px;
            padding: 1px 3px;
        }

        .countdown-badge {
            font-size: 9px;
            padding: 1px 4px;
        }

        .dropdown-arrow {
            display: none;
        }

        .btn-sm.btn-secondary {
            display: none; /* Hide 'Try as Guest' on mobile */
        }

        .user-menu {
            right: -8px;
            min-width: 200px;
        }

        .btn-sm {
            padding: 6px 10px;
            font-size: 11px;
        }
    }

    @media (max-width: 480px) {
        .header {
            padding: 8px 10px;
        }

        .logo-icon {
            width: 22px;
            height: 22px;
            font-size: 12px;
        }

        .terminal-status {
            display: none;
        }

        .user-badge {
            padding: 3px 6px;
            gap: 4px;
        }

        .user-avatar {
            width: 18px;
            height: 18px;
            font-size: 9px;
        }

        .guest-badge {
            font-size: 7px;
        }

        .countdown-badge {
            font-size: 8px;
            padding: 1px 3px;
        }

        .user-menu {
            position: fixed;
            left: 10px;
            right: 10px;
            top: 50px;
            min-width: auto;
            width: calc(100% - 20px);
        }
    }
</style>
