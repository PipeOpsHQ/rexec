<script lang="ts">
    import { onMount } from "svelte";
    import { auth, isAuthenticated } from "$stores/auth";
    import { containers, startAutoRefresh, stopAutoRefresh } from "$stores/containers";
    import { terminal, hasSessions } from "$stores/terminal";
    import { toast } from "$stores/toast";

    // Components
    import Header from "$components/Header.svelte";
    import Landing from "$components/Landing.svelte";
    import Dashboard from "$components/Dashboard.svelte";
    import CreateTerminal from "$components/CreateTerminal.svelte";
    import Settings from "$components/Settings.svelte";
    import SSHKeys from "$components/SSHKeys.svelte";
    import TerminalView from "$components/terminal/TerminalView.svelte";
    import ToastContainer from "$components/ui/ToastContainer.svelte";
    import JoinSession from "$components/JoinSession.svelte";
    import Pricing from "$components/Pricing.svelte";
    import StatusIcon from "$components/icons/StatusIcon.svelte";

    // App state
    let currentView:
        | "landing"
        | "dashboard"
        | "create"
        | "settings"
        | "sshkeys"
        | "join" = "landing";
    let isLoading = true;
    let isInitialized = false; // Prevents reactive statements from firing before token validation
    let joinCode = ""; // For /join/:code route

    // Guest email modal state
    let showGuestModal = false;
    let guestEmail = "";
    let isGuestSubmitting = false;
    
    // Pricing modal state
    let showPricing = false;

    function openGuestModal() {
        guestEmail = "";
        showGuestModal = true;
    }

    function closeGuestModal() {
        showGuestModal = false;
        guestEmail = "";
    }

    function validateEmail(email: string): boolean {
        const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return re.test(email);
    }

    async function handleGuestSubmit() {
        if (!guestEmail.trim()) {
            toast.error("Please enter your email");
            return;
        }

        if (!validateEmail(guestEmail.trim())) {
            toast.error("Please enter a valid email");
            return;
        }

        isGuestSubmitting = true;
        const result = await auth.guestLogin(guestEmail.trim());
        isGuestSubmitting = false;

        if (result.success) {
            closeGuestModal();
            // Fetch containers for returning guests
            await containers.fetchContainers();

            // Show welcome message
            if (result.returningGuest) {
                if (result.containerCount > 0) {
                    toast.success(
                        `Welcome back! Found ${result.containerCount} terminal${result.containerCount > 1 ? "s" : ""} from your previous visit.`,
                    );
                } else {
                    toast.success("Welcome back!");
                }
            } else {
                toast.success("Guest access started! You have 1 hour.");
            }
        }
    }

    function handleGuestKeydown(event: KeyboardEvent) {
        if (event.key === "Enter" && !isGuestSubmitting) {
            handleGuestSubmit();
        } else if (event.key === "Escape") {
            closeGuestModal();
        }
    }

    // Handle OAuth callback (from URL params when redirected back)
    async function handleOAuthCallback() {
        const params = new URLSearchParams(window.location.search);
        const code = params.get("code");

        if (code) {
            const result = await auth.exchangeOAuthCode(code);
            if (result.success) {
                // Clear URL params
                window.history.replaceState({}, "", window.location.pathname);
                currentView = "dashboard";
                await containers.fetchContainers();
                toast.success("Successfully signed in!");
            }
        }
    }

    // Handle OAuth postMessage from popup window
    function handleOAuthMessage(event: MessageEvent) {
        // Verify origin
        if (event.origin !== window.location.origin) return;

        if (event.data?.type === "oauth_success" && event.data?.data) {
            const { token, user } = event.data.data;
            if (token && user) {
                auth.login(token, user);
                currentView = "dashboard";
                containers.fetchContainers();
                toast.success(`Welcome, ${user.username || user.email}!`);
            }
        } else if (event.data?.type === "oauth_error") {
            toast.error(event.data.message || "Authentication failed");
        }
    }

    // Handle URL-based terminal routing
    async function handleTerminalUrl() {
        const path = window.location.pathname;
        const params = new URLSearchParams(window.location.search);

        // Check for /join/:code route
        const joinMatch = path.match(/^\/join\/([A-Z0-9]{6})$/i);
        if (joinMatch) {
            joinCode = joinMatch[1].toUpperCase();
            
            // If not authenticated, store join code and show landing with join prompt
            if (!$isAuthenticated) {
                localStorage.setItem("pendingJoinCode", joinCode);
                currentView = "landing";
                // Show a message that they need to login
                setTimeout(() => {
                    toast.info("Please login or continue as guest to join the session");
                }, 500);
                return;
            }
            
            currentView = "join";
            return;
        }
        
        // Check for pending join after authentication
        const pendingJoin = localStorage.getItem("pendingJoinCode");
        if (pendingJoin && $isAuthenticated) {
            localStorage.removeItem("pendingJoinCode");
            joinCode = pendingJoin;
            currentView = "join";
            return;
        }

        // Check for popped-out terminal window (?terminal=containerId&name=containerName)
        const terminalParam = params.get("terminal");
        const nameParam = params.get("name");

        if (terminalParam && $isAuthenticated) {
            // Clear URL params
            window.history.replaceState({}, "", window.location.pathname);

            // This is a popped-out terminal window
            const containerId = terminalParam;
            const containerName = nameParam || "Terminal";

            // Set to docked mode for full-screen terminal experience
            terminal.setViewMode("docked");
            terminal.createSession(containerId, containerName);
            currentView = "dashboard";
            return;
        }

        // Handle path-based routing (/terminal/containerId)
        const match = path.match(
            /^\/(?:terminal\/)?([a-f0-9]{64}|[a-f0-9-]{36})$/i,
        );

        if (match && $isAuthenticated) {
            const containerId = match[1];
            // Fetch container info and create session - TerminalPanel handles WebSocket
            const result = await containers.getContainer(containerId);
            if (result.success && result.container) {
                // Set terminal to docked mode for direct URL access (full screen)
                terminal.setViewMode("docked");
                terminal.createSession(containerId, result.container.name);
            }
        }
    }

    // Initialize app
    onMount(() => {
        // Listen for OAuth messages from popup window
        window.addEventListener("message", handleOAuthMessage);

        // Run async initialization
        (async () => {
            // Check for OAuth callback
            await handleOAuthCallback();

            // Validate existing token - check localStorage directly to avoid reactive timing issues
            const storedToken = localStorage.getItem("rexec_token");
            const storedUser = localStorage.getItem("rexec_user");

            console.log(
                "[App] Init - stored token:",
                !!storedToken,
                "stored user:",
                !!storedUser,
            );

            if (storedToken && storedUser) {
                const isValid = await auth.validateToken();
                console.log("[App] Token validation result:", isValid);

                if (isValid) {
                    await auth.fetchProfile();
                    currentView = "dashboard";
                    await containers.fetchContainers();
                    startAutoRefresh(); // Start polling for container updates
                    await handleTerminalUrl();
                } else {
                    console.log("[App] Token invalid, logging out");
                    auth.logout();
                }
            }

            isLoading = false;
            isInitialized = true; // Mark as initialized after token validation
        })();

        // Cleanup on destroy
        return () => {
            window.removeEventListener("message", handleOAuthMessage);
            stopAutoRefresh(); // Stop polling when component unmounts
        };
    });

    // React to auth changes (only after initialization to prevent race conditions)
    $: if (isInitialized && $isAuthenticated && currentView === "landing") {
        const pendingJoin = localStorage.getItem("pendingJoinCode");
        if (pendingJoin) {
            console.log("[App] Found pending join code, redirecting to join view");
            localStorage.removeItem("pendingJoinCode");
            joinCode = pendingJoin;
            currentView = "join";
        } else {
            currentView = "dashboard";
        }
        containers.fetchContainers();
        startAutoRefresh(); // Start polling when authenticated
    }

    $: if (isInitialized && !$isAuthenticated && currentView !== "landing") {
        currentView = "landing";
        containers.reset();
        terminal.closeAllSessionsForce();
        stopAutoRefresh(); // Stop polling when logged out
    }

    // Navigation functions
    function goToDashboard() {
        currentView = "dashboard";
        window.history.pushState({}, "", "/");
    }

    function goToCreate() {
        currentView = "create";
    }

    function goToSettings() {
        currentView = "settings";
    }

    function goToSSHKeys() {
        currentView = "sshkeys";
    }

    function onContainerCreated(
        event: CustomEvent<{ id: string; name: string }>,
    ) {
        const { id, name } = event.detail;
        currentView = "dashboard";

        // Create session - TerminalPanel will handle WebSocket connection
        terminal.createSession(id, name);
    }

    // Handle browser navigation
    function handlePopState() {
        const path = window.location.pathname;
        if (path === "/" || path === "") {
            currentView = $isAuthenticated ? "dashboard" : "landing";
        }
    }
</script>

<svelte:window on:popstate={handlePopState} />

<div class="app">
    {#if isLoading}
        <div class="loading-screen">
            <div class="spinner-large"></div>
            <p>Loading...</p>
        </div>
    {:else}
        <Header
            on:home={goToDashboard}
            on:create={goToCreate}
            on:settings={goToSettings}
            on:sshkeys={goToSSHKeys}
            on:guest={openGuestModal}
            on:pricing={() => showPricing = true}
        />

        <main class="main" class:has-terminal={$hasSessions}>
            {#if currentView === "landing"}
                <Landing on:guest={openGuestModal} />
            {:else if currentView === "dashboard"}
                <Dashboard
                    on:create={goToCreate}
                    on:connect={(e) => {
                        // Only create session - TerminalPanel will handle WebSocket connection
                        terminal.createSession(e.detail.id, e.detail.name);
                    }}
                />
            {:else if currentView === "create"}
                <CreateTerminal
                    on:cancel={goToDashboard}
                    on:created={onContainerCreated}
                />
            {:else if currentView === "settings"}
                <Settings on:back={goToDashboard} />
            {:else if currentView === "sshkeys"}
                <SSHKeys on:back={goToDashboard} />
            {:else if currentView === "join"}
                <JoinSession code={joinCode} on:joined={(e) => {
                    // Use createCollabSession for shared terminals to track mode/role
                    terminal.createCollabSession(
                        e.detail.containerId, 
                        e.detail.containerName,
                        e.detail.mode || 'control',
                        e.detail.role || 'viewer'
                    );
                    currentView = "dashboard";
                }} on:cancel={goToDashboard} />
            {/if}
        </main>

        <!-- Terminal overlay (floating or docked) -->
        {#if $hasSessions}
            <TerminalView />
        {/if}

        <!-- Toast notifications -->
        <ToastContainer />
        
        <!-- Pricing Modal -->
        <Pricing bind:isOpen={showPricing} on:close={() => showPricing = false} />

        <!-- Guest Email Modal -->
        {#if showGuestModal}
            <div
                class="modal-overlay"
                on:click={closeGuestModal}
                on:keydown={handleGuestKeydown}
                role="presentation"
            >
                <div
                    class="modal"
                    on:click|stopPropagation
                    role="dialog"
                    aria-modal="true"
                    aria-labelledby="guest-modal-title"
                >
                    <div class="modal-header">
                        <h2 id="guest-modal-title">Get Started</h2>
                        <button
                            class="modal-close"
                            on:click={closeGuestModal}
                            aria-label="Close">×</button
                        >
                    </div>

                    <div class="modal-body">
                        <p class="modal-description">
                            Enter your email to start your free guest access.
                            We'll use this to save your work and send you
                            updates.
                        </p>

                        <div class="form-group">
                            <label for="guest-email">Email Address</label>
                            <input
                                type="email"
                                id="guest-email"
                                bind:value={guestEmail}
                                on:keydown={handleGuestKeydown}
                                placeholder="you@example.com"
                                disabled={isGuestSubmitting}
                            />
                        </div>

                        <p class="modal-hint">
                            <StatusIcon status="validating" size={14} /> Guest access lasts 30 minutes. Sign in with
                            PipeOps for unlimited access.
                        </p>
                    </div>

                    <div class="modal-footer">
                        <button
                            class="btn btn-secondary"
                            on:click={closeGuestModal}
                            disabled={isGuestSubmitting}
                        >
                            Cancel
                        </button>
                        <button
                            class="btn btn-primary"
                            on:click={handleGuestSubmit}
                            disabled={isGuestSubmitting || !guestEmail.trim()}
                        >
                            {isGuestSubmitting
                                ? "Starting..."
                                : "Start Terminal"}
                        </button>
                    </div>
                </div>
            </div>
        {/if}
    {/if}
</div>

<style>
    .app {
        min-height: 100vh;
        display: flex;
        flex-direction: column;
    }

    .main {
        flex: 1;
        max-width: 1400px;
        margin: 0 auto;
        padding: 20px;
        width: 100%;
    }

    .main.has-terminal {
        padding-bottom: calc(45vh + 20px);
    }

    @media (max-width: 768px) {
        .main.has-terminal {
            padding-bottom: 20px;
        }
    }

    .loading-screen {
        position: fixed;
        inset: 0;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 16px;
        background: var(--bg);
        background-image: var(--bg-grid);
        background-size: 20px 20px;
        z-index: 9999;
    }

    .loading-screen p {
        color: var(--text-muted);
        font-size: 14px;
    }

    .spinner-large {
        width: 40px;
        height: 40px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    /* Guest Modal Styles */
    .modal-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.85);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 2000;
        animation: fadeIn 0.15s ease;
    }

    .modal {
        background: var(--bg-card);
        border: 1px solid var(--border);
        width: 100%;
        max-width: 420px;
        animation: slideIn 0.2s ease;
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
    }

    .modal-header h2 {
        font-size: 16px;
        text-transform: uppercase;
        letter-spacing: 1px;
        margin: 0;
    }

    .modal-close {
        background: none;
        border: none;
        color: var(--text-muted);
        font-size: 24px;
        cursor: pointer;
        padding: 0;
        line-height: 1;
    }

    .modal-close:hover {
        color: var(--text);
    }

    .modal-body {
        padding: 20px;
    }

    .modal-description {
        font-size: 13px;
        color: var(--text-secondary);
        margin: 0 0 20px;
        line-height: 1.5;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
        margin-bottom: 16px;
    }

    .form-group label {
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-secondary);
    }

    .form-group input {
        width: 100%;
        padding: 12px 14px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 14px;
    }

    .form-group input:focus {
        outline: none;
        border-color: var(--accent);
    }

    .form-group input:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .modal-hint {
        font-size: 11px;
        color: var(--text-muted);
        margin: 0;
        padding: 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        padding: 16px 20px;
        border-top: 1px solid var(--border);
    }

    @keyframes fadeIn {
        from {
            opacity: 0;
        }
        to {
            opacity: 1;
        }
    }

    @keyframes slideIn {
        from {
            opacity: 0;
            transform: translateY(-20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    @media (max-width: 600px) {
        .modal {
            margin: 16px;
        }
    }
</style>
