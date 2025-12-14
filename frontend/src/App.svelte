<script lang="ts">
    import { onDestroy, onMount } from "svelte";
    import { get } from "svelte/store";
    import { auth, isAuthenticated, isAdmin, token } from "$stores/auth";
    import {
        containers,
        startAutoRefresh,
        stopAutoRefresh,
    } from "$stores/containers";
    import { toast } from "$stores/toast";
    import { collab } from "$stores/collab";
    import { theme } from "$stores/theme";

    // Components (eager)
    import Header from "$components/Header.svelte";
    import Landing from "$components/Landing.svelte";
    import ToastContainer from "$components/ui/ToastContainer.svelte";
    import StatusIcon from "$components/icons/StatusIcon.svelte";

    // Lazy terminal store (keeps marketing bundle small)
    type TerminalStoreModule = typeof import("$stores/terminal");
    type TerminalViewMode = "floating" | "docked" | "fullscreen";
    let terminalStoreModule: TerminalStoreModule | null = null;
    let terminalStorePromise: Promise<TerminalStoreModule> | null = null;
    let hasTerminalSessions = false;
    let unsubscribeHasSessions: (() => void) | null = null;

    async function ensureTerminalStore() {
        if (terminalStoreModule) return terminalStoreModule;
        if (!terminalStorePromise) {
            terminalStorePromise = import("$stores/terminal")
                .then((mod) => {
                    terminalStoreModule = mod;
                    if (!unsubscribeHasSessions) {
                        unsubscribeHasSessions = mod.hasSessions.subscribe(
                            (value) => {
                                hasTerminalSessions = value;
                            },
                        );
                    }
                    return mod;
                })
                .catch((err) => {
                    terminalStorePromise = null;
                    throw err;
                });
        }
        return terminalStorePromise;
    }

    async function openTerminalForContainer(
        containerId: string,
        name: string,
        viewMode?: TerminalViewMode,
    ) {
        try {
            const { terminal } = await ensureTerminalStore();
            if (viewMode) terminal.setViewMode(viewMode);
            await terminal.createSession(containerId, name);
        } catch (e) {
            console.error("[App] Failed to open terminal:", e);
            toast.error("Failed to open terminal. Please try again.");
        }
    }

    async function openTerminalForAgent(
        agentId: string,
        name: string,
        viewMode?: TerminalViewMode,
    ) {
        try {
            const { terminal } = await ensureTerminalStore();
            if (viewMode) terminal.setViewMode(viewMode);
            await terminal.createAgentSession(agentId, name);
        } catch (e) {
            console.error("[App] Failed to open agent terminal:", e);
            toast.error("Failed to connect to agent terminal.");
        }
    }

    type CollabMode = "view" | "control";
    type CollabRole = "owner" | "editor" | "viewer";

    async function openTerminalForCollab(
        containerId: string,
        name: string,
        mode: CollabMode,
        role: CollabRole,
    ) {
        try {
            const { terminal } = await ensureTerminalStore();
            await terminal.createCollabSession(containerId, name, mode, role);
        } catch (e) {
            console.error("[App] Failed to join shared session:", e);
            toast.error("Failed to join session. Please try again.");
        }
    }

    // Lazy component loading (code-split views so marketing stays lightweight)
    type LazyComponentKey =
        | "dashboard"
        | "adminDashboard"
        | "createTerminal"
        | "settings"
        | "sshKeys"
        | "snippetsPage"
        | "marketplacePage"
        | "agentDocs"
        | "cliDocs"
        | "docs"
        | "cliLogin"
        | "account"
        | "accountLayout"
        | "billing"
        | "recordingsPage"
        | "apiTokens"
        | "joinSession"
        | "guides"
        | "useCases"
        | "useCaseDetail"
        | "pricing"
        | "promo"
        | "notFound"
        | "terminalView"
        | "screenLock";

    type LazyComponentModule = { default: any };
    type LazyLoader = () => Promise<LazyComponentModule>;

    const componentLoaders: Record<LazyComponentKey, LazyLoader> = {
        dashboard: () => import("$components/Dashboard.svelte"),
        adminDashboard: () => import("$components/AdminDashboard.svelte"),
        createTerminal: () => import("$components/CreateTerminal.svelte"),
        settings: () => import("$components/Settings.svelte"),
        sshKeys: () => import("$components/SSHKeys.svelte"),
        snippetsPage: () => import("$components/SnippetsPage.svelte"),
        marketplacePage: () => import("$components/MarketplacePage.svelte"),
        agentDocs: () => import("$components/AgentDocs.svelte"),
        cliDocs: () => import("$components/CLIDocs.svelte"),
        docs: () => import("$components/Docs.svelte"),
        cliLogin: () => import("$components/CLILogin.svelte"),
        account: () => import("$components/Account.svelte"),
        accountLayout: () => import("$components/AccountLayout.svelte"),
        billing: () => import("$components/Billing.svelte"),
        recordingsPage: () => import("$components/RecordingsPage.svelte"),
        apiTokens: () => import("$components/APITokens.svelte"),
        joinSession: () => import("$components/JoinSession.svelte"),
        guides: () => import("$components/Guides.svelte"),
        useCases: () => import("$components/UseCases.svelte"),
        useCaseDetail: () => import("$components/UseCaseDetail.svelte"),
        pricing: () => import("$components/Pricing.svelte"),
        promo: () => import("$components/Promo.svelte"),
        notFound: () => import("$components/NotFound.svelte"),
        terminalView: () => import("$components/terminal/TerminalView.svelte"),
        screenLock: () => import("$components/ScreenLock.svelte"),
    };

    let lazyComponents: Partial<Record<LazyComponentKey, any>> = {};
    const componentPromises = new Map<LazyComponentKey, Promise<any>>();

    async function ensureComponent(key: LazyComponentKey) {
        if (lazyComponents[key]) return lazyComponents[key];
        const existing = componentPromises.get(key);
        if (existing) return existing;

        const promise = componentLoaders[key]()
            .then((mod) => {
                const component = mod.default;
                lazyComponents = { ...lazyComponents, [key]: component };
                componentPromises.delete(key);
                return component;
            })
            .catch((err) => {
                componentPromises.delete(key);
                throw err;
            });

        componentPromises.set(key, promise);
        return promise;
    }

    function preloadComponent(key: LazyComponentKey) {
        ensureComponent(key).catch((err) => {
            console.error(`[App] Failed to load component '${key}':`, err);
        });
    }

    onDestroy(() => {
        if (unsubscribeHasSessions) {
            unsubscribeHasSessions();
            unsubscribeHasSessions = null;
        }
    });

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
        | "account-recordings"
        | "account-api"
        | "docs"
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

    // SEO & Meta Management - Page-specific metadata
    interface PageSEO {
        title: string;
        description: string;
        robots?: string;
        ogTitle?: string;
        keywords?: string;
    }

    const seoConfig: Record<string, PageSEO> = {
        landing: {
            title: "Rexec - Terminal as a Service | Instant Linux Terminals",
            description: "Launch secure Linux terminals instantly in your browser. No setup required. Choose from Ubuntu, Debian, Alpine, and more. Perfect for developers, learning, and testing.",
            keywords: "terminal, linux, cloud terminal, web terminal, ubuntu, debian, docker, containers, developer tools",
        },
        promo: {
            title: "Rexec - The Future of Cloud Terminals",
            description: "Experience the next generation of cloud terminals. Instant access to powerful Linux environments with no setup required.",
        },
        dashboard: {
            title: "Dashboard - Rexec",
            description: "Manage your cloud terminals and containers. Create, monitor, and control your Linux environments.",
            robots: "noindex, nofollow",
        },
        admin: {
            title: "Admin Dashboard - Rexec",
            description: "Rexec administration panel",
            robots: "noindex, nofollow",
        },
        pricing: {
            title: "Pricing - Rexec | Simple, Transparent Plans",
            description: "Simple, transparent pricing for instant Linux terminals. Free tier available. Scale your infrastructure as you grow with Pro and Enterprise plans.",
            keywords: "pricing, plans, cloud terminal pricing, linux terminal cost",
        },
        billing: {
            title: "Billing - Rexec",
            description: "Manage your Rexec subscription and billing details.",
            robots: "noindex, nofollow",
        },
        guides: {
            title: "Guides & Tutorials - Rexec",
            description: "Learn how to use Rexec with AI tools like Claude, ChatGPT, and GitHub Copilot. Step-by-step tutorials for terminal automation.",
            keywords: "tutorials, guides, AI tools, Claude, ChatGPT, terminal automation",
        },
        "use-cases": {
            title: "Use Cases - Rexec | Terminal as a Service",
            description: "Discover how developers, teams, and enterprises use Rexec for development, testing, training, and AI agent workflows.",
            keywords: "use cases, agentic, AI agents, development, testing, training",
        },
        "use-case-detail": {
            title: "Use Case - Rexec",
            description: "Detailed use case for Rexec Terminal as a Service.",
        },
        docs: {
            title: "Documentation - Rexec",
            description: "Complete documentation for Rexec. Learn about features, security, API, CLI, and agent integration.",
            keywords: "documentation, docs, API, CLI, agent, security",
        },
        "cli-docs": {
            title: "CLI Documentation - Rexec",
            description: "Install and use the Rexec CLI for terminal access from your command line. SSH, exec, and manage containers.",
            keywords: "CLI, command line, terminal, SSH, rexec-cli",
        },
        "agent-docs": {
            title: "Agent Documentation - Rexec",
            description: "Connect your own servers to Rexec with the agent. Bring your own server (BYOS) for terminal access anywhere.",
            keywords: "agent, BYOS, bring your own server, remote terminal, rexec-agent",
        },
        marketplace: {
            title: "Marketplace - Rexec",
            description: "Browse and use pre-built terminal environments and configurations shared by the community.",
            keywords: "marketplace, templates, environments, community",
        },
        snippets: {
            title: "Snippets - Rexec",
            description: "Manage your command snippets. Save, organize, and quickly execute frequently used commands.",
            robots: "noindex, nofollow",
        },
        account: {
            title: "Account - Rexec",
            description: "Manage your Rexec account settings, profile, and preferences.",
            robots: "noindex, nofollow",
        },
        "account-settings": {
            title: "Settings - Rexec",
            description: "Configure your Rexec account settings, MFA, and connected agents.",
            robots: "noindex, nofollow",
        },
        "account-ssh": {
            title: "SSH Keys - Rexec",
            description: "Manage your SSH keys for secure terminal access.",
            robots: "noindex, nofollow",
        },
        "account-billing": {
            title: "Billing - Rexec",
            description: "Manage your subscription and billing details.",
            robots: "noindex, nofollow",
        },
        "account-snippets": {
            title: "Snippets - Rexec",
            description: "Manage your command snippets.",
            robots: "noindex, nofollow",
        },
        "account-recordings": {
            title: "Recordings - Rexec",
            description: "View and manage your terminal session recordings.",
            robots: "noindex, nofollow",
        },
        "account-api": {
            title: "API Tokens - Rexec",
            description: "Manage your API tokens for CLI and programmatic access.",
            robots: "noindex, nofollow",
        },
        settings: {
            title: "Settings - Rexec",
            description: "Configure your terminal settings and preferences.",
            robots: "noindex, nofollow",
        },
        sshkeys: {
            title: "SSH Keys - Rexec",
            description: "Manage your SSH keys for secure access.",
            robots: "noindex, nofollow",
        },
        create: {
            title: "Create Terminal - Rexec",
            description: "Create a new cloud terminal. Choose from Ubuntu, Debian, Alpine, Kali, and more.",
            robots: "noindex, nofollow",
        },
        join: {
            title: "Join Session - Rexec",
            description: "Join a collaborative terminal session.",
            robots: "noindex, nofollow",
        },
        "cli-login": {
            title: "CLI Login - Rexec",
            description: "Authenticate your CLI with Rexec.",
            robots: "noindex, nofollow",
        },
        "404": {
            title: "Page Not Found - Rexec",
            description: "The page you're looking for doesn't exist.",
            robots: "noindex, nofollow",
        },
    };

    function updateMeta(name: string, content: string) {
        let meta = document.querySelector(`meta[name="${name}"]`);
        if (!meta) {
            meta = document.createElement("meta");
            meta.setAttribute("name", name);
            document.head.appendChild(meta);
        }
        meta.setAttribute("content", content);
    }

    function updateOGMeta(property: string, content: string) {
        let meta = document.querySelector(`meta[property="${property}"]`);
        if (!meta) {
            meta = document.createElement("meta");
            meta.setAttribute("property", property);
            document.head.appendChild(meta);
        }
        meta.setAttribute("content", content);
    }

    // Explicit function to update SEO - can be called after route changes
    function updateSEO(view: string) {
        if (typeof document === "undefined") return;
        
        const seo = seoConfig[view] || seoConfig.landing;
        
        // Update title
        document.title = seo.title;
        
        // Update description
        updateMeta("description", seo.description);
        
        // Update robots
        updateMeta("robots", seo.robots || "index, follow");
        
        // Update keywords if provided
        if (seo.keywords) {
            updateMeta("keywords", seo.keywords);
        }
        
        // Update Open Graph tags
        updateOGMeta("og:title", seo.ogTitle || seo.title);
        updateOGMeta("og:description", seo.description);
        updateOGMeta("og:url", `https://rexec.pipeops.io${window.location.pathname}`);
        
        // Update Twitter tags
        updateMeta("twitter:title", seo.ogTitle || seo.title);
        updateMeta("twitter:description", seo.description);
        
        // Update canonical URL
        let canonical = document.querySelector('link[rel="canonical"]');
        if (canonical) {
            canonical.setAttribute("href", `https://rexec.pipeops.io${window.location.pathname}`);
        }
    }

    // Reactive SEO update when currentView changes
    $: if (typeof document !== "undefined" && currentView) {
        updateSEO(currentView);
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
                    void openTerminalForAgent(
                        agentId,
                        `Agent ${agentId.slice(0, 8)}`,
                        "docked",
                    );
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

        // Handle root path - show dashboard if authenticated, landing if not
        if (path === "/" || path === "") {
            if (get(isAuthenticated)) {
                currentView = "dashboard";
            } else {
                currentView = "landing";
            }
            return;
        }

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
            if (!get(isAuthenticated)) {
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
            if (get(isAuthenticated)) {
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
            if (get(isAuthenticated)) {
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

        // Check for /join/:code route (share codes use base64-URL: letters, numbers, -, _)
        const joinMatch = path.match(/^\/join\/([A-Z0-9_-]{6})$/i);
        if (joinMatch) {
            joinCode = joinMatch[1].toUpperCase();

            // If not authenticated, store join code and show landing with join prompt
            // Use get() instead of $ for synchronous access in async context
            if (!get(isAuthenticated)) {
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
            if (!get(isAuthenticated)) {
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
            if (!get(isAuthenticated)) {
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
            if (!get(isAuthenticated)) {
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

        // Check for /docs route (main documentation page)
        if (path === "/docs") {
            currentView = "docs";
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
            } else if (path === "/account/recordings") {
                currentView = "account-recordings";
                accountSection = "recordings";
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
            if (get(isAuthenticated) && callback) {
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
        if (pendingJoin && get(isAuthenticated)) {
            localStorage.removeItem("pendingJoinCode");
            joinCode = pendingJoin;
            currentView = "join";
            return;
        }

        // Check for pending agent redirect after authentication
        const pendingAgent = localStorage.getItem("pendingAgentId");
        if (pendingAgent && get(isAuthenticated)) {
            localStorage.removeItem("pendingAgentId");
            window.history.replaceState({}, "", `/agent:${pendingAgent}`);
            await connectToAgent(pendingAgent);
            return;
        }

        // Check for popped-out terminal window (?terminal=containerId&name=containerName)
        const terminalParam = params.get("terminal");
        const nameParam = params.get("name");

        if (terminalParam && get(isAuthenticated)) {
            // Clear URL params
            window.history.replaceState({}, "", window.location.pathname);

            // This is a popped-out terminal window
            const containerId = terminalParam;
            const containerName = nameParam || "Terminal";

            // Set to docked mode for full-screen terminal experience
            void openTerminalForContainer(containerId, containerName, "docked");
            currentView = "dashboard";
            return;
        }

        // Handle agent URL routing (/agent:agentId)
        const agentMatch = path.match(/^\/agent:([a-f0-9-]{36})$/i);
        if (agentMatch) {
            const agentId = agentMatch[1];
            if (get(isAuthenticated)) {
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

        if (match && get(isAuthenticated)) {
            const containerId = match[1];
            // Fetch container info and create session - TerminalPanel handles WebSocket
            const result = await containers.getContainer(containerId);
            if (result.success && result.container) {
                // Set terminal to docked mode for direct URL access (full screen)
                void openTerminalForContainer(
                    containerId,
                    result.container.name,
                    "docked",
                );
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
            "/docs",
            "/docs/agent",
            "/docs/cli",
            "/account",
            "/account/settings",
            "/account/ssh",
            "/account/sshkeys",
            "/account/billing",
            "/account/snippets",
            "/account/recordings",
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

        // Subscribe to theme changes and update terminal themes
        const unsubscribeTheme = theme.subscribe((currentTheme) => {
            terminalStoreModule?.terminal.updateTheme(currentTheme === "dark");
        });

        // Subscribe to collab session events - close terminal when session ends
        const unsubscribeCollab = collab.onMessage((msg) => {
            if (msg.type === "ended" || msg.type === "expired") {
                if (!terminalStoreModule) return;
                // Find and close all collab terminal sessions
                const state = terminalStoreModule.terminal.getState();
                state.sessions.forEach((session, sessionId) => {
                    if (session.isCollabSession) {
                        terminalStoreModule?.terminal.closeSession(sessionId);
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

                    // Check for CLI callback - redirect with token if present
                    const cliCallback = localStorage.getItem("cli_callback");
                    if (cliCallback) {
                        localStorage.removeItem("cli_callback");
                        window.location.href = `${cliCallback}?token=${encodeURIComponent(storedToken)}`;
                        return; // Stop further processing
                    }

                    // Fetch containers for authenticated users
                    await containers.fetchContainers();
                    startAutoRefresh(); // Start polling for container updates
                    
                    // Set default view to dashboard only if on root path
                    // handleTerminalUrl() will override for specific routes like /account
                    const currentPath = window.location.pathname;
                    if (currentPath === "/" || currentPath === "") {
                        currentView = "dashboard";
                    }
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
            unsubscribeTheme();
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
        currentView !== "docs" &&
        currentView !== "account" &&
        currentView !== "join" &&
        currentView !== "pricing" &&
        currentView !== "404"
    ) {
        currentView = "landing";
        containers.reset();
        terminalStoreModule?.terminal.closeAllSessionsForce();
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
        void openTerminalForContainer(id, name);
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
        } else if (path === "/docs") {
            currentView = "docs";
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
                } else if (path === "/account/recordings") {
                    currentView = "account-recordings";
                    accountSection = "recordings";
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

    // Preload view components when needed (keeps landing bundle small)
    $: if (hasTerminalSessions) preloadComponent("terminalView");
    $: if ($isAuthenticated) preloadComponent("screenLock");
    $: if (showPricing || currentView === "pricing") preloadComponent("pricing");

    $: {
        switch (currentView) {
            case "dashboard":
                preloadComponent("dashboard");
                break;
            case "admin":
                preloadComponent("adminDashboard");
                break;
            case "create":
                preloadComponent("createTerminal");
                break;
            case "settings":
                preloadComponent("settings");
                break;
            case "sshkeys":
                preloadComponent("sshKeys");
                break;
            case "snippets":
                preloadComponent("snippetsPage");
                break;
            case "marketplace":
                preloadComponent("marketplacePage");
                break;
            case "agent-docs":
                preloadComponent("agentDocs");
                break;
            case "cli-docs":
                preloadComponent("cliDocs");
                break;
            case "docs":
                preloadComponent("docs");
                break;
            case "cli-login":
                preloadComponent("cliLogin");
                break;
            case "account":
                preloadComponent("account");
                break;
            case "account-settings":
                preloadComponent("accountLayout");
                preloadComponent("settings");
                break;
            case "account-ssh":
                preloadComponent("accountLayout");
                preloadComponent("sshKeys");
                break;
            case "account-billing":
                preloadComponent("accountLayout");
                preloadComponent("billing");
                break;
            case "account-snippets":
                preloadComponent("accountLayout");
                preloadComponent("snippetsPage");
                break;
            case "account-recordings":
                preloadComponent("accountLayout");
                preloadComponent("recordingsPage");
                break;
            case "account-api":
                preloadComponent("accountLayout");
                preloadComponent("apiTokens");
                break;
            case "join":
                preloadComponent("joinSession");
                break;
            case "guides":
                preloadComponent("guides");
                break;
            case "use-cases":
                preloadComponent("useCases");
                break;
            case "use-case-detail":
                preloadComponent("useCaseDetail");
                break;
            case "pricing":
                preloadComponent("pricing");
                break;
            case "promo":
                preloadComponent("promo");
                break;
            case "billing":
                preloadComponent("billing");
                break;
            case "404":
                preloadComponent("notFound");
                break;
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

        <main class="main" class:has-terminal={hasTerminalSessions}>
            {#if currentView === "landing"}
                <Landing
                    on:guest={openGuestModal}
                    on:navigate={(e) => {
                        // @ts-ignore - view is checked elsewhere or we trust it matches types
                        currentView = e.detail.view;
                        window.history.pushState({}, "", "/" + e.detail.view);
                    }}
                />
            {:else if currentView === "dashboard"}
                {#if lazyComponents.dashboard}
                    <svelte:component
                        this={lazyComponents.dashboard}
                        on:create={goToCreate}
                        on:connect={(e) => {
                            if (e.detail.id.startsWith("agent:")) {
                                const agentId = e.detail.id.replace("agent:", "");
                                void openTerminalForAgent(agentId, e.detail.name);
                                return;
                            }
                            void openTerminalForContainer(e.detail.id, e.detail.name);
                        }}
                        on:showAgentDocs={goToAgents}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "admin"}
                {#if lazyComponents.adminDashboard}
                    <svelte:component this={lazyComponents.adminDashboard} />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "create"}
                {#if lazyComponents.createTerminal}
                    <svelte:component
                        this={lazyComponents.createTerminal}
                        on:cancel={goToDashboard}
                        on:created={onContainerCreated}
                        on:upgrade={() => {
                            currentView = "pricing";
                            window.history.pushState({}, "", "/pricing");
                        }}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "settings"}
                {#if lazyComponents.settings}
                    <svelte:component
                        this={lazyComponents.settings}
                        on:back={goToDashboard}
                        on:connectAgent={(e) => {
                            const { agentId, agentName } = e.detail;
                            void openTerminalForAgent(agentId, agentName);
                            currentView = "dashboard";
                            window.history.pushState({}, "", "/");
                            toast.success(`Connecting to ${agentName}...`);
                        }}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "sshkeys"}
                {#if lazyComponents.sshKeys}
                    <svelte:component
                        this={lazyComponents.sshKeys}
                        on:back={goToDashboard}
                        on:run={(e) => {
                            const command = e.detail.command;
                            const store = terminalStoreModule;
                            const activeSessionId =
                                store?.terminal.getState().activeSessionId;

                            if (activeSessionId && store) {
                                currentView = "dashboard";
                                setTimeout(() => {
                                    store.terminal.sendInput(
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
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "snippets"}
                {#if lazyComponents.snippetsPage}
                    <svelte:component
                        this={lazyComponents.snippetsPage}
                        on:back={goToDashboard}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "marketplace"}
                {#if lazyComponents.marketplacePage}
                    <svelte:component
                        this={lazyComponents.marketplacePage}
                        on:back={goToDashboard}
                        on:use={() => {
                            goToDashboard();
                        }}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "agent-docs"}
                {#if lazyComponents.agentDocs}
                    <svelte:component
                        this={lazyComponents.agentDocs}
                        onback={() => {
                            window.history.back();
                        }}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "cli-docs"}
                {#if lazyComponents.cliDocs}
                    <svelte:component
                        this={lazyComponents.cliDocs}
                        onback={() => {
                            window.history.back();
                        }}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "docs"}
                {#if lazyComponents.docs}
                    <svelte:component
                        this={lazyComponents.docs}
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "docs/cli") {
                                currentView = "cli-docs";
                                window.history.pushState({}, "", "/docs/cli");
                            } else if (view === "docs/agent") {
                                currentView = "agent-docs";
                                window.history.pushState({}, "", "/docs/agent");
                            }
                        }}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "cli-login"}
                {#if lazyComponents.cliLogin}
                    <svelte:component this={lazyComponents.cliLogin} />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "account"}
                {#if lazyComponents.account}
                    <svelte:component
                        this={lazyComponents.account}
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "dashboard") goToDashboard();
                            else if (view === "settings") {
                                currentView = "account-settings";
                                accountSection = "settings";
                                window.history.pushState(
                                    {},
                                    "",
                                    "/account/settings",
                                );
                            } else if (view === "sshkeys") {
                                currentView = "account-ssh";
                                accountSection = "ssh";
                                window.history.pushState({}, "", "/account/ssh");
                            } else if (view === "billing") {
                                currentView = "account-billing";
                                accountSection = "billing";
                                window.history.pushState(
                                    {},
                                    "",
                                    "/account/billing",
                                );
                            } else if (view === "snippets") {
                                currentView = "account-snippets";
                                accountSection = "snippets";
                                window.history.pushState(
                                    {},
                                    "",
                                    "/account/snippets",
                                );
                            } else if (view === "recordings") {
                                currentView = "account-recordings";
                                accountSection = "recordings";
                                window.history.pushState(
                                    {},
                                    "",
                                    "/account/recordings",
                                );
                            } else if (view === "pricing") {
                                currentView = "pricing";
                                window.history.pushState({}, "", "/pricing");
                            } else if (view === "admin") goToAdmin();
                            else if (view === "docs/cli") goToCLI();
                            else if (view === "docs/agent") goToAgents();
                        }}
                        on:logout={() => auth.logout()}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "account-settings"}
                {#if lazyComponents.accountLayout && lazyComponents.settings}
                    <svelte:component
                        this={lazyComponents.accountLayout}
                        section="settings"
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "dashboard") goToDashboard();
                        }}
                    >
                        <svelte:component
                            this={lazyComponents.settings}
                            on:back={() => {
                                currentView = "account";
                                accountSection = null;
                                window.history.pushState({}, "", "/account");
                            }}
                            on:connectAgent={(e) => {
                                const { agentId, agentName } = e.detail;
                                void openTerminalForAgent(agentId, agentName);
                                currentView = "dashboard";
                                window.history.pushState({}, "", "/");
                                toast.success(`Connecting to ${agentName}...`);
                            }}
                        />
                    </svelte:component>
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "account-ssh"}
                {#if lazyComponents.accountLayout && lazyComponents.sshKeys}
                    <svelte:component
                        this={lazyComponents.accountLayout}
                        section="ssh"
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "dashboard") goToDashboard();
                        }}
                    >
                        <svelte:component
                            this={lazyComponents.sshKeys}
                            on:back={() => {
                                currentView = "account";
                                accountSection = null;
                                window.history.pushState({}, "", "/account");
                            }}
                            on:run={(e) => {
                                const command = e.detail.command;
                                const store = terminalStoreModule;
                                const activeSessionId =
                                    store?.terminal.getState().activeSessionId;

                                if (activeSessionId && store) {
                                    currentView = "dashboard";
                                    window.history.pushState({}, "", "/");
                                    setTimeout(() => {
                                        store.terminal.sendInput(
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
                                    window.history.pushState({}, "", "/");
                                }
                            }}
                        />
                    </svelte:component>
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "account-billing"}
                {#if lazyComponents.accountLayout && lazyComponents.billing}
                    <svelte:component
                        this={lazyComponents.accountLayout}
                        section="billing"
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "dashboard") goToDashboard();
                        }}
                    >
                        <svelte:component
                            this={lazyComponents.billing}
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
                    </svelte:component>
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "account-snippets"}
                {#if lazyComponents.accountLayout && lazyComponents.snippetsPage}
                    <svelte:component
                        this={lazyComponents.accountLayout}
                        section="snippets"
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "dashboard") goToDashboard();
                        }}
                    >
                        <svelte:component
                            this={lazyComponents.snippetsPage}
                            on:back={() => {
                                currentView = "account";
                                accountSection = null;
                                window.history.pushState({}, "", "/account");
                            }}
                        />
                    </svelte:component>
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "account-recordings"}
                {#if lazyComponents.accountLayout && lazyComponents.recordingsPage}
                    <svelte:component
                        this={lazyComponents.accountLayout}
                        section="recordings"
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "dashboard") goToDashboard();
                        }}
                    >
                        <svelte:component this={lazyComponents.recordingsPage} />
                    </svelte:component>
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "account-api"}
                {#if lazyComponents.accountLayout && lazyComponents.apiTokens}
                    <svelte:component
                        this={lazyComponents.accountLayout}
                        section="api"
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "dashboard") goToDashboard();
                        }}
                    >
                        <svelte:component this={lazyComponents.apiTokens} />
                    </svelte:component>
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "join"}
                {#if lazyComponents.joinSession}
                    <svelte:component
                        this={lazyComponents.joinSession}
                        code={joinCode}
                        on:joined={(e) => {
                            void openTerminalForCollab(
                                e.detail.containerId,
                                e.detail.containerName,
                                e.detail.mode || "control",
                                e.detail.role || "viewer",
                            );
                            currentView = "dashboard";
                        }}
                        on:cancel={goToDashboard}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "guides"}
                {#if lazyComponents.guides}
                    <svelte:component
                        this={lazyComponents.guides}
                        on:tryNow={openGuestModal}
                        on:navigate={(e) => {
                            const view = e.detail.view;
                            if (view === "agentic") {
                                currentView = "use-cases";
                                window.history.pushState({}, "", "/use-cases");
                            } else if (view === "docs/cli") {
                                goToCLI();
                            } else if (view === "docs/agent") {
                                goToAgents();
                            }
                        }}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "use-cases"}
                {#if lazyComponents.useCases}
                    <svelte:component
                        this={lazyComponents.useCases}
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
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "use-case-detail"}
                {#if lazyComponents.useCaseDetail}
                    <svelte:component
                        this={lazyComponents.useCaseDetail}
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
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "pricing"}
                {#if lazyComponents.pricing}
                    <svelte:component this={lazyComponents.pricing} mode="page" />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "billing"}
                {#if lazyComponents.billing}
                    <svelte:component
                        this={lazyComponents.billing}
                        on:back={goToDashboard}
                        on:pricing={() => {
                            currentView = "pricing";
                            window.history.pushState({}, "", "/pricing");
                        }}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "promo"}
                {#if lazyComponents.promo}
                    <svelte:component
                        this={lazyComponents.promo}
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
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {:else if currentView === "404"}
                {#if lazyComponents.notFound}
                    <svelte:component
                        this={lazyComponents.notFound}
                        on:home={goToDashboard}
                    />
                {:else}
                    <div class="view-loading">Loading...</div>
                {/if}
            {/if}
        </main>

        <!-- Terminal overlay (floating or docked) -->
        {#if hasTerminalSessions}
            {#if lazyComponents.terminalView}
                <svelte:component this={lazyComponents.terminalView} />
            {/if}
        {/if}

        <!-- Screen Lock Security -->
        {#if $isAuthenticated && lazyComponents.screenLock}
            <svelte:component this={lazyComponents.screenLock} />
        {/if}

        <!-- Toast notifications -->
        <ToastContainer />

        <!-- Pricing Modal -->
        {#if showPricing && lazyComponents.pricing}
            <svelte:component
                this={lazyComponents.pricing}
                bind:isOpen={showPricing}
                on:close={() => (showPricing = false)}
            />
        {/if}

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

    .view-loading {
        padding: 24px 0;
        text-align: center;
        color: var(--text-muted);
        font-size: 14px;
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
