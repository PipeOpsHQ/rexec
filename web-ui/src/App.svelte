<script lang="ts">
    import { onMount } from "svelte";
    import { get } from "svelte/store";
    import { auth, isAuthenticated, isAdmin, token } from "$stores/auth";
    import {
        containers,
        startAutoRefresh,
        stopAutoRefresh,
    } from "$stores/containers";
    import { terminal, hasSessions } from "$stores/terminal";
    import { toast } from "$stores/toast";
    import { collab } from "$stores/collab";

    // Components
    import Header from "$components/Header.svelte";
    import Landing from "$components/Landing.svelte";
    import Dashboard from "$components/Dashboard.svelte";
    import AdminDashboard from "$components/AdminDashboard.svelte";
    import CreateTerminal from "$components/CreateTerminal.svelte";
    import Settings from "$components/Settings.svelte";
    import SSHKeys from "$components/SSHKeys.svelte";
    import TerminalView from "$components/terminal/TerminalView.svelte";
    import ToastContainer from "$components/ui/ToastContainer.svelte";
    import JoinSession from "$components/JoinSession.svelte";
    import Pricing from "$components/Pricing.svelte";
    import StatusIcon from "$components/icons/StatusIcon.svelte";
    import Guides from "$components/Guides.svelte";
    import UseCases from "$components/UseCases.svelte";
    import UseCaseDetail from "$components/UseCaseDetail.svelte";
    import SnippetsPage from "$components/SnippetsPage.svelte";
    import MarketplacePage from "$components/MarketplacePage.svelte";
    import NotFound from "$components/NotFound.svelte";
    import Promo from "$components/Promo.svelte";
    import Billing from "$components/Billing.svelte";
    import ScreenLock from "$components/ScreenLock.svelte";
    import AgentDocs from "$components/AgentDocs.svelte";
    import CLIDocs from "$components/CLIDocs.svelte";
    import CLILogin from "$components/CLILogin.svelte";
    import Account from "$components/Account.svelte";
    import AccountLayout from "$components/AccountLayout.svelte";
    import APITokens from "$components/APITokens.svelte";

    // App state
    let currentView:
        | "landing"
        | "dashboard"
        | "admin"
        | "create"
        | "settings"
        | "sshkeys"
        | "snippets"
        | "marketplace"
        | "join"
        | "guides"
        | "use-cases"
        | "use-case-detail"
        | "pricing"
        | "promo"
        | "billing"
        | "agent-docs"
        | "cli-docs"
        | "cli-login"
        | "account"
        | "account-settings"
        | "account-ssh"
        | "account-billing"
        | "account-snippets"
        | "account-api"
        | "404" = "landing";
    let accountSection: string | null = null; // Track which account sub-section we're in
    let isLoading = true;
    let isInitialized = false; // Prevents reactive statements from firing before token validation
    let joinCode = ""; // For /join/:code route
    let useCaseSlug = ""; // For /use-cases/:slug route

    // Guest email modal state
    let showGuestModal = false;
    let guestEmail = "";
    let isGuestSubmitting = false;

    // Pricing modal state
    let showPricing = false;

    // SEO & Meta Management
    $: {
        if (typeof document !== "undefined") {
            // Title
            if (currentView === "admin") {
                document.title = "Admin Dashboard - Rexec";
            } else if (currentView === "pricing") {
                document.title = "Pricing - Rexec";
            } else if (currentView === "dashboard") {
                document.title = "Dashboard - Rexec";
            } else if (currentView === "landing") {
                document.title = "Rexec - Terminal as a Service";
            } else if (currentView === "promo") {
                document.title = "Rexec - The Future of Terminals";
            }

            // Robots meta
            let robotsMeta = document.querySelector('meta[name="robots"]');
            if (!robotsMeta) {
                robotsMeta = document.createElement("meta");
                robotsMeta.setAttribute("name", "robots");
                document.head.appendChild(robotsMeta);
            }

            if (currentView === "admin") {
                robotsMeta.setAttribute("content", "noindex, nofollow");
            } else {
                robotsMeta.setAttribute("content", "index, follow");
            }

            // Description meta
            let descMeta = document.querySelector('meta[name="description"]');
            if (!descMeta) {
                descMeta = document.createElement("meta");
                descMeta.setAttribute("name", "description");
                document.head.appendChild(descMeta);
            }

            if (currentView === "pricing") {
                descMeta.setAttribute(
                    "content",
                    "Simple, transparent pricing for instant Linux terminals. Scale your infrastructure as you grow.",
                );
            } else if (currentView === "landing" || currentView === "promo") {
                descMeta.setAttribute(
                    "content",
                    "Launch secure Linux terminals instantly in your browser. No setup required. Perfect for demos, training, and quick tasks.",
                );
            }
        }
    }

    function openGuestModal() {
        if ($isAuthenticated) {
            toast.info("You are already logged in.");
            return;
        }
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
            let { token, user: rawUser } = event.data.data;

            // Fix for nested token object issue [object Object]
            if (typeof token === "object" && token !== null) {
                token = token.token || token.access_token || "";
            }

            if (token && typeof token === "string" && rawUser) {
                // Ensure user object has all required fields
                const user = {
                    ...rawUser,
                    name:
                        rawUser.name ||
                        rawUser.username ||
                        rawUser.email ||
                        "User",
                    tier: rawUser.tier || "free",
                    isGuest: rawUser.tier === "guest",
                };

                auth.login(token, user);
                
                // Check for CLI callback - redirect with token
                const cliCallback = localStorage.getItem("cli_callback");
                if (cliCallback) {
                    localStorage.removeItem("cli_callback");
                    window.location.href = `${cliCallback}?token=${encodeURIComponent(token)}`;
                    return;
                }
                
                currentView = "dashboard";
                containers.fetchContainers();
                toast.success(`Welcome, ${user.name}!`);
            } else {
                console.error("Invalid token format received", event.data.data);
                toast.error("Authentication failed: Invalid token format");
            }
        } else if (event.data?.type === "oauth_error") {
            toast.error(event.data.message || "Authentication failed");
        }
    }

    // Helper to connect to an agent
    async function connectToAgent(agentId: string) {
        try {
            const authToken = get(token);
            const response = await fetch(`/api/agents/${agentId}/status`, {
                headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
            });
            if (response.ok) {
                const status = await response.json();
                if (status.status === "online") {
                    terminal.setViewMode("docked");
                    terminal.createAgentSession(agentId, `Agent ${agentId.slice(0, 8)}`);
                    currentView = "dashboard";
                } else {
                    toast.error("Agent is offline");
                    currentView = "dashboard";
                }
            } else {
                toast.error("Agent not found");
                currentView = "dashboard";
            }
        } catch (err) {
            console.error("Failed to fetch agent status:", err);
            toast.error("Failed to connect to agent");
            currentView = "dashboard";
        }
    }

    // Handle URL-based terminal routing
    async function handleTerminalUrl() {
        const path = window.location.pathname;
        const params = new URLSearchParams(window.location.search);

        // Check for /admin route
        if (path === "/admin" || path === "/admin/") {
            // Ensure auth logic runs first (handled by onMount), then check role
            // But here we just set view, auth check happens in goToAdmin or reactive block
            currentView = "admin";
            return;
        }

        // Check for /pricing route
        if (path === "/pricing") {
            currentView = "pricing";
            return;
        }

        // Check for /billing route
        if (path === "/billing") {
            if (!$isAuthenticated) {
                currentView = "landing";
                return;
            }
            currentView = "billing";
            return;
        }

        // Check for /billing/success route (Stripe checkout success)
        if (path === "/billing/success") {
            // Stripe redirects here after successful checkout
            // Show billing page - the webhook will have updated the tier
            toast.success("Payment successful! Your plan has been upgraded.");
            window.history.replaceState({}, "", "/billing");
            if ($isAuthenticated) {
                currentView = "billing";
                // Refresh user profile to get updated tier
                auth.fetchProfile();
            } else {
                currentView = "landing";
            }
            return;
        }

        // Check for /billing/cancel route (Stripe checkout cancelled)
        if (path === "/billing/cancel") {
            toast.info("Checkout cancelled. No changes were made to your plan.");
            window.history.replaceState({}, "", "/billing");
            if ($isAuthenticated) {
                currentView = "billing";
            } else {
                currentView = "landing";
            }
            return;
        }

        // Check for /promo route
        if (path === "/promo") {
            currentView = "promo";
            return;
        }

        // Check for /guides route (formerly ai-tools)
        if (path === "/guides" || path === "/ai-tools") {
            currentView = "guides";
            if (path === "/ai-tools") {
                window.history.replaceState({}, "", "/guides");
            }
            return;
        }

        // Check for /use-cases route (formerly agentic)
        if (path === "/use-cases" || path === "/agentic") {
            currentView = "use-cases";
            if (path === "/agentic") {
                window.history.replaceState({}, "", "/use-cases");
            }
            return;
        }

        // Check for /use-cases/:slug route
        const useCaseMatch = path.match(/^\/use-cases\/([a-z0-9-]+)$/i);
        if (useCaseMatch) {
            useCaseSlug = useCaseMatch[1].toLowerCase();
            currentView = "use-case-detail";
            return;
        }

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
                    toast.info(
                        "Please login or continue as guest to join the session",
                    );
                }, 500);
                return;
            }

            currentView = "join";
            return;
        }

        // Check for /marketplace route
        if (path === "/marketplace") {
            currentView = "marketplace";
            return;
        }

        // Check for /settings route - redirect to /account/settings
        if (path === "/settings") {
            if (!$isAuthenticated) {
                currentView = "landing";
                return;
            }
            window.history.replaceState({}, "", "/account/settings");
            currentView = "account-settings";
            accountSection = "settings";
            return;
        }

        // Check for /sshkeys route - redirect to /account/ssh
        if (path === "/sshkeys") {
            if (!$isAuthenticated) {
                currentView = "landing";
                return;
            }
            window.history.replaceState({}, "", "/account/ssh");
            currentView = "account-ssh";
            accountSection = "ssh";
            return;
        }

        // Check for /snippets route - redirect to /account/snippets
        if (path === "/snippets") {
            if (!$isAuthenticated) {
                currentView = "landing";
                return;
            }
            window.history.replaceState({}, "", "/account/snippets");
            currentView = "account-snippets";
            accountSection = "snippets";
            return;
        }

        // Check for /docs/agent or /agents route
        if (path === "/docs/agent" || path === "/agents") {
            currentView = "agent-docs";
            return;
        }

        // Check for /docs/cli route
        if (path === "/docs/cli") {
            currentView = "cli-docs";
            return;
        }

        // Check for /account routes
        if (path.startsWith("/account")) {
            // Use get() to check current store value synchronously
            if (!get(isAuthenticated)) {
                currentView = "landing";
                return;
            }

            // Handle account sub-routes
            if (path === "/account/settings") {
                currentView = "account-settings";
                accountSection = "settings";
            } else if (path === "/account/ssh" || path === "/account/sshkeys") {
                currentView = "account-ssh";
                accountSection = "ssh";
            } else if (path === "/account/billing") {
                currentView = "account-billing";
                accountSection = "billing";
            } else if (path === "/account/snippets") {
                currentView = "account-snippets";
                accountSection = "snippets";
            } else if (path === "/account/api" || path === "/account/tokens") {
                currentView = "account-api";
                accountSection = "api";
            } else if (path === "/account" || path === "/profile") {
                currentView = "account";
                accountSection = null;
            } else {
                // Unknown account sub-route
                currentView = "404";
            }
            return;
        }

        // Check for /cli-login route (CLI login with callback)
        if (path === "/cli-login") {
            const callback = params.get("callback");
            if ($isAuthenticated && callback) {
                // User is already logged in, redirect to callback with token
                const token = localStorage.getItem("auth_token");
                if (token) {
                    window.location.href = `${callback}?token=${encodeURIComponent(token)}`;
                    return;
                }
            }
            // Store callback and show login
            if (callback) {
                localStorage.setItem("cli_callback", callback);
            }
            currentView = "cli-login";
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

        // Check for pending agent redirect after authentication
        const pendingAgent = localStorage.getItem("pendingAgentId");
        if (pendingAgent && $isAuthenticated) {
            localStorage.removeItem("pendingAgentId");
            window.history.replaceState({}, "", `/agent:${pendingAgent}`);
            await connectToAgent(pendingAgent);
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

        // Handle agent URL routing (/agent:agentId)
        const agentMatch = path.match(/^\/agent:([a-f0-9-]{36})$/i);
        if (agentMatch) {
            const agentId = agentMatch[1];
            if ($isAuthenticated) {
                await connectToAgent(agentId);
            } else {
                localStorage.setItem("pendingAgentId", agentId);
                // Fall through to landing/login
                setTimeout(() => {
                    toast.info("Please login to access this agent");
                }, 500);
            }
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
            } else {
                // Container not found
                currentView = "404";
            }
            return;
        }

        // Check for unknown paths - show 404
        const knownPaths = [
            "/",
            "/ui/dashboard",
            "/admin",
            "/pricing",
            "/billing",
            "/billing/success",
            "/billing/cancel",
            "/guides",
            "/ai-tools",
            "/use-cases",
            "/agentic",
            "/snippets",
            "/marketplace",
            "/settings",
            "/sshkeys",
            "/promo",
            "/agents",
            "/docs/agent",
            "/docs/cli",
            "/account",
            "/account/settings",
            "/account/ssh",
            "/account/sshkeys",
            "/account/billing",
            "/account/snippets",
            "/account/api",
            "/account/tokens",
            "/profile",
        ];
        const isKnownPath =
            knownPaths.includes(path) ||
            path.startsWith("/use-cases/") ||
            path.startsWith("/join/") ||
            path.startsWith("/terminal/") ||
            path.startsWith("/agent:") ||
            path.match(/^\/[a-f0-9]{64}$/i) ||
            path.match(/^\/[a-f0-9-]{36}$/i);

        if (!isKnownPath && path !== "/") {
            currentView = "404";
        }
    }

    // Initialize app
    onMount(() => {
        // Listen for OAuth messages from popup window
        window.addEventListener("message", handleOAuthMessage);

        // Subscribe to collab session events - close terminal when session ends
        const unsubscribeCollab = collab.onMessage((msg) => {
            if (msg.type === "ended" || msg.type === "expired") {
                // Find and close all collab terminal sessions
                const state = terminal.getState();
                state.sessions.forEach((session, sessionId) => {
                    if (session.isCollabSession) {
                        terminal.closeSession(sessionId);
                        toast.info(
                            msg.type === "expired"
                                ? "Shared session expired"
                                : "Shared session ended by owner",
                        );
                    }
                });
            }
        });

        // Run async initialization
        (async () => {
            // Check for admin key in URL (Build-time env var protection)
            const params = new URLSearchParams(window.location.search);
            const adminKey = params.get("admin_key");
            const expectedKey = import.meta.env.VITE_ADMIN_SECRET;

            if (adminKey && expectedKey && adminKey === expectedKey) {
                // If user is not logged in, login as guest first
                if (!auth.validateToken()) {
                    await auth.guestLogin(
                        `admin_${Math.floor(Math.random() * 1000)}@rexec.dev`,
                    );
                }

                // Enable admin mode
                auth.enableAdminMode();
                toast.success("Admin mode enabled");

                // Remove key from URL
                params.delete("admin_key");
                const newQuery = params.toString();
                const newPath =
                    window.location.pathname + (newQuery ? "?" + newQuery : "");
                window.history.replaceState({}, "", newPath);
            }

            // Check for OAuth callback
            await handleOAuthCallback();

            // Validate existing token - check localStorage directly to avoid reactive timing issues
            const storedToken = localStorage.getItem("rexec_token");
            const storedUser = localStorage.getItem("rexec_user");

            if (storedToken && storedUser) {
                const isValid = await auth.validateToken();

                if (isValid) {
                    try {
                        await auth.fetchProfile();
                    } catch (e) {
                        // If profile fetch fails (e.g. backend error), fallback to stored user
                        console.warn(
                            "Profile fetch failed, using stored user data",
                            e,
                        );
                        try {
                            const user = JSON.parse(storedUser);
                            auth.login(storedToken, user);
                        } catch (parseError) {
                            console.error(
                                "Failed to parse stored user",
                                parseError,
                            );
                        }
                    }

                    currentView = "dashboard";
                    await containers.fetchContainers();
                    startAutoRefresh(); // Start polling for container updates
                } else {
                    auth.logout();
                }
            }

            // Check URL for public routes (guides, use-cases) or terminal deep links
            await handleTerminalUrl();

            isLoading = false;
            isInitialized = true; // Mark as initialized after token validation
        })();

        // Cleanup on destroy
        return () => {
            window.removeEventListener("message", handleOAuthMessage);
            unsubscribeCollab();
            stopAutoRefresh(); // Stop polling when component unmounts
        };
    });

    // React to auth changes (only after initialization to prevent race conditions)
    $: if (isInitialized && $isAuthenticated && currentView === "landing") {
        const pendingJoin = localStorage.getItem("pendingJoinCode");
        if (pendingJoin) {
            localStorage.removeItem("pendingJoinCode");
            joinCode = pendingJoin;
            currentView = "join";
        } else {
            currentView = "dashboard";
        }
        containers.fetchContainers();
        startAutoRefresh(); // Start polling when authenticated
    }

        $: if (isInitialized && !$isAuthenticated && 

               currentView !== "landing" && 
               currentView !== "promo" &&
               currentView !== "guides" && 
        currentView !== "use-cases" &&
        currentView !== "use-case-detail" &&
        currentView !== "marketplace" &&
        currentView !== "agent-docs" &&
        currentView !== "cli-docs" &&
        currentView !== "account" &&
        currentView !== "join" &&
        currentView !== "pricing" &&
        currentView !== "404"
    ) {
        currentView = "landing";
        containers.reset();
        terminal.closeAllSessionsForce();
        stopAutoRefresh(); // Stop polling when logged out
    }

    // Navigation functions
    function goToDashboard() {
        currentView = $isAuthenticated ? "dashboard" : "landing";
        window.history.pushState({}, "", "/");
    }

    function goToCreate() {
        currentView = "create";
    }

    function goToSettings() {
        currentView = "account-settings";
        accountSection = "settings";
        window.history.pushState({}, "", "/account/settings");
    }

    function goToSSHKeys() {
        currentView = "account-ssh";
        accountSection = "ssh";
        window.history.pushState({}, "", "/account/ssh");
    }

    function goToSnippets() {
        currentView = "account-snippets";
        accountSection = "snippets";
        window.history.pushState({}, "", "/account/snippets");
    }

    function goToMarketplace() {
        currentView = "marketplace";
        window.history.pushState({}, "", "/marketplace");
    }

    function goToAgents() {
        currentView = "agent-docs";
        window.history.pushState({}, "", "/agents");
    }

    function goToCLI() {
        currentView = "cli-docs";
        window.history.pushState({}, "", "/docs/cli");
    }

    function goToBilling() {
        currentView = "billing";
        window.history.pushState({}, "", "/billing");
    }

    function goToAccount() {
        currentView = "account";
        window.history.pushState({}, "", "/account");
    }

    function goToAdmin() {
        if ($isAdmin) {
            currentView = "admin";
            window.history.pushState({}, "", "/admin");
        } else {
            toast.error("Access denied");
            currentView = "dashboard";
        }
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
        } else if (path === "/guides" || path === "/ai-tools") {
            currentView = "guides";
        } else if (path === "/use-cases" || path === "/agentic") {
            currentView = "use-cases";
        } else if (path.startsWith("/use-cases/")) {
            const match = path.match(/^\/use-cases\/([a-z0-9-]+)$/i);
            if (match) {
                useCaseSlug = match[1].toLowerCase();
                currentView = "use-case-detail";
            }
        } else if (path === "/pricing") {
            currentView = "pricing";
        } else if (path === "/billing") {
            currentView = $isAuthenticated ? "billing" : "landing";
        } else if (path === "/admin") {
            if ($isAuthenticated && $isAdmin) {
                currentView = "admin";
            } else {
                // If direct navigation to /admin but not authed/admin, redirect or show dashboard which handles it
                // Actually better to let the auth logic in onMount handle redirection if needed
                // But for popstate (back button), we might need to be careful
                currentView = "admin"; // Let the view render (or show access denied)
            }
        } else if (path === "/snippets") {
            if ($isAuthenticated) {
                window.history.replaceState({}, "", "/account/snippets");
                currentView = "account-snippets";
                accountSection = "snippets";
            } else {
                currentView = "landing";
            }
        } else if (path === "/settings") {
            if ($isAuthenticated) {
                window.history.replaceState({}, "", "/account/settings");
                currentView = "account-settings";
                accountSection = "settings";
            } else {
                currentView = "landing";
            }
        } else if (path === "/sshkeys") {
            if ($isAuthenticated) {
                window.history.replaceState({}, "", "/account/ssh");
                currentView = "account-ssh";
                accountSection = "ssh";
            } else {
                currentView = "landing";
            }
        } else if (path === "/agents" || path === "/docs/agent") {
            currentView = "agent-docs";
        } else if (path === "/docs/cli") {
            currentView = "cli-docs";
        } else if (path.startsWith("/account")) {
            if ($isAuthenticated) {
                if (path === "/account/settings") {
                    currentView = "account-settings";
                    accountSection = "settings";
                } else if (path === "/account/ssh" || path === "/account/sshkeys") {
                    currentView = "account-ssh";
                    accountSection = "ssh";
                } else if (path === "/account/billing") {
                    currentView = "account-billing";
                    accountSection = "billing";
                } else if (path === "/account/snippets") {
                    currentView = "account-snippets";
                    accountSection = "snippets";
                } else if (path === "/account/api" || path === "/account/tokens") {
                    currentView = "account-api";
                    accountSection = "api";
                } else {
                    currentView = "account";
                    accountSection = null;
                }
            } else {
                currentView = "landing";
            }
        } else if (path === "/profile") {
            currentView = $isAuthenticated ? "account" : "landing";
        } else if (path === "/marketplace") {
            currentView = "marketplace";
        }
    }
</script>

<svelte:window onpopstate={handlePopState} />

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
            on:snippets={goToSnippets}
            on:billing={goToBilling}
            on:agents={goToAgents}
            on:cli={goToCLI}
            on:account={goToAccount}
            on:guest={openGuestModal}
            on:pricing={() => {
                currentView = "pricing";
                window.history.pushState({}, "", "/pricing");
            }}
            on:admin={goToAdmin}
        />

        <main class="main" class:has-terminal={$hasSessions}>
            {#if currentView === "landing"}
                <Landing
                    on:guest={openGuestModal}
                    on:navigate={(e) => {
                        // @ts-ignore - view is checked elsewhere or we trust it matches types
                        currentView = e.detail.view;
                        window.history.pushState({}, "", "/" + e.detail.view);
                    }}
                />
                {:else if currentView === "dashboard"}                <Dashboard
                    on:create={goToCreate}
                    on:connect={(e) => {
                        // Check if this is an agent connection (id starts with 'agent:')
                        if (e.detail.id.startsWith('agent:')) {
                            const agentId = e.detail.id.replace('agent:', '');
                            terminal.createAgentSession(agentId, e.detail.name);
                        } else {
                            // Regular container connection
                            terminal.createSession(e.detail.id, e.detail.name);
                        }
                    }}
                    on:showAgentDocs={goToAgents}
                />
            {:else if currentView === "admin"}
                <AdminDashboard />
            {:else if currentView === "create"}
                <CreateTerminal
                    on:cancel={goToDashboard}
                    on:created={onContainerCreated}
                    on:upgrade={() => {
                        currentView = "pricing";
                        window.history.pushState({}, "", "/pricing");
                    }}
                />
            {:else if currentView === "settings"}
                <Settings 
                    on:back={goToDashboard} 
                    on:connectAgent={(e) => {
                        const { agentId, agentName } = e.detail;
                        // Create a terminal session for the agent
                        terminal.createAgentSession(agentId, agentName);
                        currentView = "dashboard";
                        window.history.pushState({}, "", "/");
                        toast.success(`Connecting to ${agentName}...`);
                    }}
                />
            {:else if currentView === "sshkeys"}
                <SSHKeys
                    on:back={goToDashboard}
                    on:run={(e) => {
                        const command = e.detail.command;
                        const activeSessionId =
                            terminal.getState().activeSessionId;

                        if (activeSessionId) {
                            currentView = "dashboard";
                            // Small delay to ensure view switch
                            setTimeout(() => {
                                terminal.sendInput(
                                    activeSessionId,
                                    command + "\n",
                                );
                                toast.success("Running SSH command...");
                            }, 50);
                        } else {
                            toast.error(
                                "No active terminal. Please create one first.",
                            );
                            currentView = "dashboard";
                        }
                    }}
                />
            {:else if currentView === "snippets"}
                <SnippetsPage on:back={goToDashboard} />
            {:else if currentView === "marketplace"}
                <MarketplacePage 
                    on:back={goToDashboard} 
                    on:use={(e) => {
                        // Copy to clipboard handled in component
                        goToDashboard();
                    }}
                />
            {:else if currentView === "agent-docs"}
                <AgentDocs 
                    onback={() => {
                        window.history.back();
                    }}
                />
            {:else if currentView === "cli-docs"}
                <CLIDocs 
                    onback={() => {
                        window.history.back();
                    }}
                />
            {:else if currentView === "cli-login"}
                <CLILogin />
            {:else if currentView === "account"}
                <Account
                    on:navigate={(e) => {
                        // Handle internal navigation from Account page
                        const view = e.detail.view;
                        if (view === 'dashboard') goToDashboard();
                        else if (view === 'settings') {
                            currentView = "account-settings";
                            accountSection = "settings";
                            window.history.pushState({}, "", "/account/settings");
                        }
                        else if (view === 'sshkeys') {
                            currentView = "account-ssh";
                            accountSection = "ssh";
                            window.history.pushState({}, "", "/account/ssh");
                        }
                        else if (view === 'billing') {
                            currentView = "account-billing";
                            accountSection = "billing";
                            window.history.pushState({}, "", "/account/billing");
                        }
                        else if (view === 'snippets') {
                            currentView = "account-snippets";
                            accountSection = "snippets";
                            window.history.pushState({}, "", "/account/snippets");
                        }
                        else if (view === 'pricing') {
                            currentView = "pricing";
                            window.history.pushState({}, "", "/pricing");
                        }
                        else if (view === 'admin') goToAdmin();
                        else if (view === 'docs/cli') goToCLI();
                        else if (view === 'docs/agent') goToAgents();
                    }}
                    on:logout={() => auth.logout()}
                />
            {:else if currentView === "account-settings"}
                <AccountLayout section="settings" on:navigate={(e) => {
                    const view = e.detail.view;
                    if (view === 'dashboard') goToDashboard();
                }}>
                    <Settings
                        on:back={() => {
                            currentView = "account";
                            accountSection = null;
                            window.history.pushState({}, "", "/account");
                        }}
                        on:connectAgent={(e) => {
                            const { agentId, agentName } = e.detail;
                            terminal.createAgentSession(agentId, agentName);
                            currentView = "dashboard";
                            window.history.pushState({}, "", "/");
                            toast.success(`Connecting to ${agentName}...`);
                        }}
                    />
                </AccountLayout>
            {:else if currentView === "account-ssh"}
                <AccountLayout section="ssh" on:navigate={(e) => {
                    const view = e.detail.view;
                    if (view === 'dashboard') goToDashboard();
                }}>
                    <SSHKeys
                        on:back={() => {
                            currentView = "account";
                            accountSection = null;
                            window.history.pushState({}, "", "/account");
                        }}
                        on:run={(e) => {
                            const command = e.detail.command;
                            const activeSessionId = terminal.getState().activeSessionId;

                            if (activeSessionId) {
                                currentView = "dashboard";
                                window.history.pushState({}, "", "/");
                                setTimeout(() => {
                                    terminal.sendInput(activeSessionId, command + "\n");
                                    toast.success("Running SSH command...");
                                }, 50);
                            } else {
                                toast.error("No active terminal. Please create one first.");
                                currentView = "dashboard";
                                window.history.pushState({}, "", "/");
                            }
                        }}
                    />
                </AccountLayout>
            {:else if currentView === "account-billing"}
                <AccountLayout section="billing" on:navigate={(e) => {
                    const view = e.detail.view;
                    if (view === 'dashboard') goToDashboard();
                }}>
                    <Billing
                        on:back={() => {
                            currentView = "account";
                            accountSection = null;
                            window.history.pushState({}, "", "/account");
                        }}
                        on:pricing={() => {
                            currentView = "pricing";
                            window.history.pushState({}, "", "/pricing");
                        }}
                    />
                </AccountLayout>
            {:else if currentView === "account-snippets"}
                <AccountLayout section="snippets" on:navigate={(e) => {
                    const view = e.detail.view;
                    if (view === 'dashboard') goToDashboard();
                }}>
                    <SnippetsPage
                        on:back={() => {
                            currentView = "account";
                            accountSection = null;
                            window.history.pushState({}, "", "/account");
                        }}
                    />
                </AccountLayout>
            {:else if currentView === "account-api"}
                <AccountLayout section="api" on:navigate={(e) => {
                    const view = e.detail.view;
                    if (view === 'dashboard') goToDashboard();
                }}>
                    <APITokens />
                </AccountLayout>
            {:else if currentView === "join"}
                <JoinSession
                    code={joinCode}
                    on:joined={(e) => {
                        // Use createCollabSession for shared terminals to track mode/role
                        terminal.createCollabSession(
                            e.detail.containerId,
                            e.detail.containerName,
                            e.detail.mode || "control",
                            e.detail.role || "viewer",
                        );
                        currentView = "dashboard";
                    }}
                    on:cancel={goToDashboard}
                />
            {:else if currentView === "guides"}
                <Guides
                    on:tryNow={openGuestModal}
                    on:navigate={(e) => {
                        const view = e.detail.view;
                        if (view === "agentic") {
                            // Legacy handling
                            currentView = "use-cases";
                            window.history.pushState({}, "", "/use-cases");
                        } else if (view === "docs/cli") {
                            goToCLI();
                        } else if (view === "docs/agent") {
                            goToAgents();
                        }
                    }}
                />
            {:else if currentView === "use-cases"}
                <UseCases
                    on:tryNow={openGuestModal}
                    on:navigate={(e) => {
                        useCaseSlug = e.detail.slug;
                        currentView = "use-case-detail";
                        window.history.pushState(
                            {},
                            "",
                            `/use-cases/${e.detail.slug}`,
                        );
                    }}
                />
            {:else if currentView === "use-case-detail"}
                <UseCaseDetail
                    slug={useCaseSlug}
                    on:back={() => {
                        currentView = "use-cases";
                        window.history.pushState({}, "", "/use-cases");
                    }}
                    on:tryNow={openGuestModal}
                    on:navigate={(e) => {
                        useCaseSlug = e.detail.slug;
                        window.history.pushState(
                            {},
                            "",
                            `/use-cases/${e.detail.slug}`,
                        );
                    }}
                />
            {:else if currentView === "pricing"}
                <Pricing mode="page" />
            {:else if currentView === "billing"}
                <Billing 
                    on:back={goToDashboard}
                    on:pricing={() => {
                        currentView = "pricing";
                        window.history.pushState({}, "", "/pricing");
                    }}
                />
            {:else if currentView === "promo"}
                <Promo 
                    on:guest={openGuestModal}
                    on:navigate={(e) => {
                        if (e.detail.view === "use-cases") {
                            window.history.pushState({}, "", "/use-cases");
                            currentView = "use-cases";
                        } else if (e.detail.view === "guides") {
                            window.history.pushState({}, "", "/guides");
                            currentView = "guides";
                        } else if (e.detail.view === "pricing") {
                            window.history.pushState({}, "", "/pricing");
                            currentView = "pricing";
                        }
                    }}
                />
            {:else if currentView === "404"}
                <NotFound on:home={goToDashboard} />
            {/if}
        </main>

        <!-- Terminal overlay (floating or docked) -->
        {#if $hasSessions}
            <TerminalView />
        {/if}

        <!-- Screen Lock Security -->
        <ScreenLock />

        <!-- Toast notifications -->
        <ToastContainer />

        <!-- Pricing Modal -->
        <Pricing
            bind:isOpen={showPricing}
            on:close={() => (showPricing = false)}
        />

        <!-- Guest Email Modal -->
        {#if showGuestModal}
            <div
                class="modal-overlay"
                on:click={(e) =>
                    e.target === e.currentTarget && closeGuestModal()}
                on:keydown={handleGuestKeydown}
                role="presentation"
            >
                <div
                    class="modal"
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
                            <StatusIcon status="validating" size={14} /> Guest access
                            lasts 1 hour. Sign in with PipeOps for unlimited access.
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
