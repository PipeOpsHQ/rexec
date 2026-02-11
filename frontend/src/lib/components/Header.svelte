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
    import { collab, type CollabInvitation } from "$stores/collab";
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
        github: void;
        guides: void;
        usecases: void;
        admin: void;
        account: void;
    }>();

    let showUserMenu = false;
    let showMobileMenu = false;
    let showInvitationsMenu = false;
    let timeRemaining = "";
    let countdownInterval: ReturnType<typeof setInterval> | null = null;
    let invitationsInterval: ReturnType<typeof setInterval> | null = null;
    let isOAuthLoading = false;
    let sessionCountValue = 0;
    let pendingInvitations: CollabInvitation[] = [];
    let terminalModulePromise: Promise<
        typeof import("$stores/terminal")
    > | null = null;
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
        // Fetch invitations on mount for authenticated non-guest users
        if ($isAuthenticated && !$isGuest) {
            fetchInvitations();
            // Poll for invitations every 30 seconds
            invitationsInterval = setInterval(fetchInvitations, 30000);
        }
    });

    onDestroy(() => {
        if (countdownInterval) {
            clearInterval(countdownInterval);
        }
        if (invitationsInterval) {
            clearInterval(invitationsInterval);
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

    // Start/stop invitation polling based on auth state
    $: if ($isAuthenticated && !$isGuest && !invitationsInterval) {
        fetchInvitations();
        invitationsInterval = setInterval(fetchInvitations, 30000);
    } else if ((!$isAuthenticated || $isGuest) && invitationsInterval) {
        clearInterval(invitationsInterval);
        invitationsInterval = null;
        pendingInvitations = [];
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
        if (!target.closest(".invitations-menu-container")) {
            showInvitationsMenu = false;
        }
    }

    async function fetchInvitations() {
        if ($isAuthenticated && !$isGuest) {
            pendingInvitations = await collab.getMyInvitations();
        }
    }

    async function handleAcceptInvitation(invitation: CollabInvitation) {
        const result = await collab.respondToInvitation(invitation.id, "accept");
        if (result.success && result.shareCode) {
            showInvitationsMenu = false;
            toast.success(`Joined ${invitation.containerName}'s session`);
            // Navigate to the terminal with the share code
            window.location.href = `/join/${result.shareCode}`;
        } else {
            toast.error(result.error || "Failed to accept invitation");
        }
        await fetchInvitations();
    }

    async function handleDeclineInvitation(invitation: CollabInvitation) {
        const result = await collab.respondToInvitation(invitation.id, "decline");
        if (result.success) {
            toast.info("Invitation declined");
        } else {
            toast.error(result.error || "Failed to decline invitation");
        }
        await fetchInvitations();
    }

    function toggleInvitationsMenu() {
        showInvitationsMenu = !showInvitationsMenu;
        if (showInvitationsMenu) {
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
        <a class="nav-link" href="https://github.com/PipeOpsHQ/rexec" target="_blank" rel="noopener noreferrer">
            <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
            </svg>
            <span>GitHub</span>
        </a>
        <a class="nav-link" href="/marketplace">
            <StatusIcon status="snippet" size={14} />
            <span>Snippets</span>
        </a>
        <a class="nav-link" href="/resources">
            <StatusIcon status="video" size={14} />
            <span>Resources</span>
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
            title={$theme === "dark"
                ? "Switch to light mode"
                : "Switch to dark mode"}
        >
            {#if $theme === "dark"}
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
                    {sessionCountValue} Terminal{sessionCountValue > 1
                        ? "s"
                        : ""}
                </span>
            {/if}

            <!-- Invitations Bell -->
            {#if !$isGuest}
                <div class="invitations-menu-container">
                    <button
                        class="invitations-bell"
                        class:has-invitations={pendingInvitations.length > 0}
                        onclick={(e) => {
                            e.stopPropagation();
                            toggleInvitationsMenu();
                        }}
                        title="Collaboration Invitations"
                    >
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"></path>
                            <path d="M13.73 21a2 2 0 0 1-3.46 0"></path>
                        </svg>
                        {#if pendingInvitations.length > 0}
                            <span class="invitation-badge">{pendingInvitations.length}</span>
                        {/if}
                    </button>

                    {#if showInvitationsMenu}
                        <div class="invitations-menu">
                            <div class="invitations-header">
                                <span>Collaboration Invitations</span>
                            </div>
                            {#if pendingInvitations.length === 0}
                                <div class="no-invitations">
                                    No pending invitations
                                </div>
                            {:else}
                                {#each pendingInvitations as invitation}
                                    <div class="invitation-item">
                                        <div class="invitation-info">
                                            <span class="invitation-from">{invitation.invitedBy}</span>
                                            <span class="invitation-text">invited you to</span>
                                            <span class="invitation-terminal">{invitation.containerName}</span>
                                            <span class="invitation-mode" class:control={invitation.mode === "control"}>
                                                {invitation.mode === "control" ? "Full Control" : "View Only"}
                                            </span>
                                        </div>
                                        <div class="invitation-actions">
                                            <button class="inv-btn accept" onclick={() => handleAcceptInvitation(invitation)}>
                                                Accept
                                            </button>
                                            <button class="inv-btn decline" onclick={() => handleDeclineInvitation(invitation)}>
                                                Decline
                                            </button>
                                        </div>
                                    </div>
                                {/each}
                            {/if}
                        </div>
                    {/if}
                </div>
            {/if}

            <div class="user-menu-container">
                <button
                    class="user-badge"
                    onclick={(e) => {
                        e.stopPropagation();
                        toggleUserMenu();
                    }}
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
                            {#if $isGuest}<span class="lock-icon"
                                    ><StatusIcon
                                        status="lock"
                                        size={12}
                                    /></span
                                >{/if}
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
                            {#if $isGuest}<span class="lock-icon"
                                    ><StatusIcon
                                        status="lock"
                                        size={12}
                                    /></span
                                >{/if}
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
                            {#if $isGuest}<span class="lock-icon"
                                    ><StatusIcon
                                        status="lock"
                                        size={12}
                                    /></span
                                >{/if}
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
                            {#if $isGuest}<span class="lock-icon"
                                    ><StatusIcon
                                        status="lock"
                                        size={12}
                                    /></span
                                >{/if}
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
                            {#if $isGuest}<span class="lock-icon"
                                    ><StatusIcon
                                        status="lock"
                                        size={12}
                                    /></span
                                >{/if}
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
                            {#if $isGuest}<span class="lock-icon"
                                    ><StatusIcon
                                        status="lock"
                                        size={12}
                                    /></span
                                >{/if}
                        </button>

                        <div class="user-menu-divider"></div>

                        <!-- Explore Section -->
                        <div class="user-menu-section-label">Explore</div>
                        <a
                            class="user-menu-item"
                            href="https://github.com/PipeOpsHQ/rexec"
                            target="_blank"
                            rel="noopener noreferrer"
                            onclick={() => { showUserMenu = false; }}
                        >
                            <svg viewBox="0 0 24 24" fill="currentColor" width="14" height="14">
                                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                            </svg>
                            GitHub
                        </a>

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
                <a
                    class="mobile-nav-link"
                    href="https://github.com/PipeOpsHQ/rexec"
                    target="_blank"
                    rel="noopener noreferrer"
                    onclick={closeMobileMenu}
                >
                    <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
                        <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                    </svg>
                    GitHub
                </a>
                <button
                    class="mobile-nav-link"
                    onclick={() => {
                        closeMobileMenu();
                        dispatch("home");
                    }}
                >
                    <StatusIcon status="chart" size={16} /> Dashboard
                </button>
                <div class="user-menu-divider"></div>
                <button
                    class="mobile-nav-link"
                    onclick={() => {
                        closeMobileMenu();
                        dispatch("create");
                    }}
                >
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
        from {
            transform: translateX(-100%);
        }
        to {
            transform: translateX(0);
        }
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

    /* Invitations Menu Styles */
    .invitations-menu-container {
        position: relative;
    }

    .invitations-bell {
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
        position: relative;
    }

    .invitations-bell:hover {
        color: var(--accent);
        border-color: var(--accent);
    }

    .invitations-bell.has-invitations {
        color: var(--accent);
        border-color: var(--accent);
    }

    .invitation-badge {
        position: absolute;
        top: -4px;
        right: -4px;
        min-width: 16px;
        height: 16px;
        background: var(--danger);
        color: white;
        font-size: 10px;
        font-weight: 700;
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 0 4px;
    }

    .invitations-menu {
        position: absolute;
        top: calc(100% + 8px);
        right: 0;
        min-width: 300px;
        max-width: 360px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        z-index: 200;
        animation: fadeIn 0.15s ease;
    }

    .invitations-header {
        padding: 12px 16px;
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text);
        border-bottom: 1px solid var(--border);
    }

    .no-invitations {
        padding: 20px;
        text-align: center;
        color: var(--text-muted);
        font-size: 12px;
    }

    .invitation-item {
        padding: 12px 16px;
        border-bottom: 1px solid var(--border);
    }

    .invitation-item:last-child {
        border-bottom: none;
    }

    .invitation-info {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 4px;
        margin-bottom: 10px;
        font-size: 12px;
        line-height: 1.5;
    }

    .invitation-from {
        font-weight: 600;
        color: var(--accent);
    }

    .invitation-text {
        color: var(--text-muted);
    }

    .invitation-terminal {
        font-weight: 600;
        color: var(--text);
    }

    .invitation-mode {
        font-size: 10px;
        padding: 2px 6px;
        background: rgba(0, 255, 157, 0.1);
        color: var(--accent);
        border-radius: 3px;
        margin-left: 4px;
    }

    .invitation-mode.control {
        background: rgba(255, 166, 0, 0.1);
        color: #ffa600;
    }

    .invitation-actions {
        display: flex;
        gap: 8px;
    }

    .inv-btn {
        flex: 1;
        padding: 6px 12px;
        border: none;
        border-radius: 4px;
        font-size: 11px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }

    .inv-btn.accept {
        background: var(--accent);
        color: var(--bg);
    }

    .inv-btn.accept:hover {
        background: var(--accent-hover);
    }

    .inv-btn.decline {
        background: var(--bg-tertiary);
        color: var(--text-muted);
        border: 1px solid var(--border);
    }

    .inv-btn.decline:hover {
        background: rgba(255, 68, 68, 0.1);
        color: #ff6b6b;
        border-color: rgba(255, 68, 68, 0.3);
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

        .mobile-menu-content {
            width: 220px;
        }

        .mobile-menu-header {
            padding: 12px;
        }

        .mobile-nav-links {
            padding: 12px;
            gap: 8px;
        }

        .mobile-nav-link {
            padding: 8px;
            font-size: 13px;
            gap: 10px;
        }
    }

    @media (max-width: 360px) {
        .header {
            padding: 6px 8px;
        }

        .mobile-menu-btn {
            margin-right: 4px;
            padding: 2px;
        }

        .mobile-menu-btn :global(svg) {
            width: 16px;
            height: 16px;
        }

        .logo {
            gap: 6px;
        }

        .logo-icon {
            width: 18px;
            height: 18px;
            font-size: 10px;
        }

        .logo-text {
            font-size: 12px;
        }

        .nav-actions {
            gap: 4px;
        }

        .theme-toggle {
            padding: 4px;
        }

        .theme-toggle :global(svg) {
            width: 14px;
            height: 14px;
        }

        .user-badge {
            padding: 2px 4px;
            gap: 3px;
        }

        .user-avatar {
            width: 16px;
            height: 16px;
            font-size: 8px;
        }

        .guest-badge {
            font-size: 6px;
            padding: 1px 2px;
        }

        .countdown-badge {
            font-size: 7px;
            padding: 1px 2px;
        }

        .countdown-badge :global(svg) {
            width: 8px;
            height: 8px;
        }

        .user-menu {
            left: 6px;
            right: 6px;
            top: 44px;
            width: calc(100% - 12px);
        }

        .user-menu-header {
            padding: 10px;
        }

        .user-menu-name {
            font-size: 12px;
        }

        .user-menu-email {
            font-size: 10px;
        }

        .user-menu-tier {
            font-size: 9px;
            padding: 2px 5px;
        }

        .user-menu-item {
            padding: 8px 10px;
            font-size: 12px;
            gap: 8px;
        }

        .user-menu-section-label {
            font-size: 9px;
            padding: 4px 10px 2px;
        }

        .mobile-menu-content {
            width: 180px;
        }

        .mobile-menu-header {
            padding: 10px;
        }

        .mobile-menu-header .logo-text {
            font-size: 14px;
        }

        .close-btn :global(svg) {
            width: 16px;
            height: 16px;
        }

        .mobile-nav-links {
            padding: 10px;
            gap: 6px;
        }

        .mobile-nav-link {
            padding: 6px 8px;
            font-size: 12px;
            gap: 8px;
        }

        .mobile-nav-link :global(svg) {
            width: 14px;
            height: 14px;
        }
    }
</style>
