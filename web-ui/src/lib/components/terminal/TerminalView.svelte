<script lang="ts">
    import { onMount, onDestroy, tick } from "svelte";
    import { terminal, sessionCount, isFloating } from "$stores/terminal";
    import { containers, type ProgressEvent } from "$stores/containers";
    import { toast } from "$stores/toast";
    import { api } from "$utils/api";
    import TerminalPanel from "./TerminalPanel.svelte";
    import PlatformIcon from "../icons/PlatformIcon.svelte";

    // Track view mode changes to force terminal re-render
    let viewModeKey = 0;

    // Mobile detection
    let isMobile = false;
    $: {
        if (typeof window !== 'undefined') {
            isMobile = window.innerWidth < 768 || /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
        }
    }

    // Floating window state
    let isDragging = false;
    let isResizing = false;
    let dragOffset = { x: 0, y: 0 };

    // Tab drag-out state
    let draggingTabId: string | null = null;
    let tabDragStart: { x: number; y: number } | null = null;
    let isDraggingTab = false;

    // Get state from store
    $: isMinimized = $terminal.isMinimized;
    $: floatingPosition = $terminal.floatingPosition;
    $: floatingSize = $terminal.floatingSize;
    $: sessions = Array.from($terminal.sessions.entries());
    $: dockedSessions = sessions.filter(([_, s]) => !s.isDetached);
    $: detachedSessions = sessions.filter(([_, s]) => s.isDetached);
    $: activeId = $terminal.activeSessionId;

    // Inline create terminal state
    let showCreatePanel = false;
    let selectedImage = "";
    let customImage = "";
    let isCreating = false;
    let selectedRole = "standard"; // Default role
    let progress = 0;
    let progressMessage = "";
    let progressStage = "";

    // Progress steps for visual display
    const progressSteps = [
        { id: "validating", label: "Validating" },
        { id: "pulling", label: "Pulling Image" },
        { id: "creating", label: "Creating Container" },
        { id: "starting", label: "Starting" },
        { id: "configuring", label: "Configuring" },
        { id: "ready", label: "Ready" },
    ];

    // Get step status
    function getStepStatus(stepId: string): "pending" | "active" | "completed" {
        const stepOrder = progressSteps.map((s) => s.id);
        const currentIndex = stepOrder.indexOf(progressStage);
        const stepIndex = stepOrder.indexOf(stepId);

        if (stepIndex < currentIndex) return "completed";
        if (stepIndex === currentIndex) return "active";
        return "pending";
    }

    // Round progress to integer
    $: displayProgress = Math.round(progress);
    let images: Array<{
        name: string;
        display_name: string;
        description: string;
        category: string;
        popular?: boolean;
    }> = [];



    // Role to preferred OS mapping
    const roleToOS: Record<string, string> = {
        standard: "alpine", // Minimalist loves lightweight
        node: "ubuntu", // Best Node.js support
        python: "ubuntu", // Best Python/data science support
        go: "alpine", // Go's preferred container OS
        neovim: "arch", // Power users love Arch
        devops: "alpine", // Container standard
        overemployed: "alpine", // Fast startup
    };

    // Auto-select OS when role changes
    $: if (selectedRole && roleToOS[selectedRole]) {
        const preferredOS = roleToOS[selectedRole];
        // Only auto-select if images are loaded and the preferred OS exists
        if (
            images.length > 0 &&
            images.some((img) => img.name === preferredOS)
        ) {
            selectedImage = preferredOS;
        }
    }

    // Available roles with detailed descriptions
    const roles = [
        {
            id: "standard",
            name: "The Minimalist",
            desc: "I use Arch btw. Just give me a shell.",
            tools: ["bash", "git", "curl", "vim"],
            recommendedOS: "Alpine",
            useCase: "Quick tasks, scripting, and basic development",
        },
        {
            id: "node",
            name: "10x JS Ninja",
            desc: "Ship fast, break things, npm install everything.",
            tools: ["node", "npm", "yarn", "pnpm", "git"],
            recommendedOS: "Ubuntu",
            useCase: "Full-stack JavaScript/TypeScript development",
        },
        {
            id: "python",
            name: "Data Wizard",
            desc: "Import antigravity. I speak in list comprehensions.",
            tools: ["python3", "pip", "jupyter", "pandas", "numpy"],
            recommendedOS: "Ubuntu",
            useCase: "Data science, ML, and Python development",
        },
        {
            id: "go",
            name: "The Gopher",
            desc: "If err != nil { panic(err) }. Simplicity is key.",
            tools: ["go", "git", "make", "delve"],
            recommendedOS: "Alpine",
            useCase: "Go development, CLI tools, and microservices",
        },
        {
            id: "neovim",
            name: "Neovim God",
            desc: "My config is longer than your code. Mouse? What mouse?",
            tools: ["neovim", "tmux", "fzf", "ripgrep", "lazygit"],
            recommendedOS: "Arch",
            useCase: "Terminal-first development with powerful editing",
        },
        {
            id: "devops",
            name: "YAML Herder",
            desc: "I don't write code, I write config. Prod is my playground.",
            tools: ["kubectl", "docker", "terraform", "helm", "aws-cli"],
            recommendedOS: "Alpine",
            useCase: "Infrastructure, containers, and cloud operations",
        },
        {
            id: "overemployed",
            name: "The Overemployed",
            desc: "Working 4 remote jobs. Need max efficiency.",
            tools: ["tmux", "git", "ssh", "docker", "zsh"],
            recommendedOS: "Alpine",
            useCase: "Maximum productivity with minimal overhead",
        },
    ];

    // Get current selected role details
    $: currentRole = roles.find((r) => r.id === selectedRole);



    // Load available images when create panel opens
    async function loadImages() {
        if (images.length > 0) return; // Already loaded

        const { data, error } = await api.get<{
            images?: typeof images;
            popular?: typeof images;
        }>("/api/images?all=true");

        if (data) {
            images = data.images || data.popular || [];
        } else if (error) {
            toast.error("Failed to load images");
        }
    }

    // Generate random name
    function generateName(): string {
        const adjectives = [
            "swift",
            "bold",
            "calm",
            "dark",
            "eager",
            "fast",
            "grand",
            "happy",
            "keen",
            "light",
            "merry",
            "noble",
            "proud",
            "quick",
            "rare",
            "sharp",
        ];
        const nouns = [
            "ant",
            "bear",
            "cat",
            "fox",
            "hawk",
            "lion",
            "owl",
            "wolf",
            "tiger",
            "eagle",
            "shark",
            "cobra",
            "raven",
            "viper",
            "lynx",
            "orca",
        ];
        const adj = adjectives[Math.floor(Math.random() * adjectives.length)];
        const noun = nouns[Math.floor(Math.random() * nouns.length)];
        const num = Math.floor(Math.random() * 1000);
        return `${adj}-${noun}-${num}`;
    }

    // Handle image selection and creation
    async function selectAndCreate(imageName: string) {
        if (isCreating) return;

        selectedImage = imageName;

        // For custom image, show input (TODO: could add a modal)
        if (imageName === "custom") {
            const input = prompt(
                "Enter Docker image name (e.g., nginx:latest):",
            );
            if (!input || !input.trim()) {
                selectedImage = "";
                return;
            }
            customImage = input.trim();
        }

        // Start creation - set state first for immediate UI feedback
        isCreating = true;
        progress = 0;
        progressMessage = "Starting...";
        progressStage = "initializing";

        // Defer heavy work to next tick to allow UI to update
        await new Promise((resolve) => setTimeout(resolve, 0));

        const name = generateName();
        const image = selectedImage;
        const custom = selectedImage === "custom" ? customImage : undefined;

        containers.createContainerWithProgress(
            name,
            image,
            custom,
            selectedRole,
            // onProgress
            (event: ProgressEvent) => {
                progress = event.progress || 0;
                progressMessage = event.message || "";
                progressStage = event.stage || "";
                // Don't handle errors here - let onError handle them
            },
            // onComplete
            (container) => {
                isCreating = false;
                showCreatePanel = false;
                selectedImage = "";
                customImage = "";

                // Debug logging
                console.log(
                    "[TerminalView] Container creation complete:",
                    container,
                );

                // Defensive checks for container object
                if (!container) {
                    console.error(
                        "[TerminalView] Container object is null/undefined",
                    );
                    toast.error("Container creation failed. Please try again.");
                    return;
                }

                const containerName =
                    container.name || name || `Terminal-${Date.now()}`;
                const containerId = container.id || container.db_id;

                console.log("[TerminalView] Creating session with:", {
                    containerId,
                    containerName,
                });

                if (!containerId) {
                    console.error(
                        "[TerminalView] No container ID found in:",
                        container,
                    );
                    toast.error(
                        "Container created but ID not found. Please refresh.",
                    );
                    return;
                }

                toast.success(`Terminal "${containerName}" created!`);

                // Add a small delay to ensure container is fully ready
                // before attempting WebSocket connection
                setTimeout(() => {
                    console.log("[TerminalView] Creating session after delay");
                    terminal.createSession(containerId, containerName);
                }, 1000);
            },
            // onError
            (error) => {
                console.error(
                    "[TerminalView] Container creation error:",
                    error,
                );
                isCreating = false;
                showCreatePanel = false;
                // Ensure error is a string
                let errorMsg = "Failed to create terminal";
                if (typeof error === "string" && error.trim()) {
                    errorMsg = error;
                } else if (error && typeof error === "object") {
                    errorMsg =
                        error.message || error.error || JSON.stringify(error);
                }
                // Avoid showing 'undefined' or empty messages
                if (!errorMsg || errorMsg === "undefined") {
                    errorMsg = "Failed to create terminal. Please try again.";
                }
                toast.error(errorMsg);
            },
        );
    }

    function openCreatePanel() {
        showCreatePanel = true;
        loadImages();
    }

    function closeCreatePanel() {
        if (!isCreating) {
            showCreatePanel = false;
            selectedImage = "";
            customImage = "";
            selectedRole = "standard";
        }
    }

    // Floating drag handlers
    function handleMouseDown(event: MouseEvent) {
        if (
            event.target instanceof HTMLElement &&
            (event.target.tagName === "BUTTON" ||
                event.target.closest("button"))
        ) {
            return;
        }

        isDragging = true;
        dragOffset = {
            x: event.clientX - floatingPosition.x,
            y: event.clientY - floatingPosition.y,
        };
    }

    function handleMouseMove(event: MouseEvent) {
        if (isDragging) {
            const x = Math.max(
                0,
                Math.min(window.innerWidth - 100, event.clientX - dragOffset.x),
            );
            const y = Math.max(
                0,
                Math.min(
                    window.innerHeight - 100,
                    event.clientY - dragOffset.y,
                ),
            );
            terminal.setFloatingPosition(x, y);
        }

        if (isResizing) {
            const width = Math.max(400, event.clientX - floatingPosition.x);
            const height = Math.max(300, event.clientY - floatingPosition.y);
            terminal.setFloatingSize(width, height);
        }
    }

    function handleMouseUp() {
        if (isDragging || isResizing) {
            isDragging = false;
            isResizing = false;
            // Fit terminals after resize
            setTimeout(() => terminal.fitAll(), 50);
        }
    }

    function handleResizeStart(event: MouseEvent) {
        event.preventDefault();
        event.stopPropagation();
        isResizing = true;
    }

    // Toggle between floating and docked
    function toggleView() {
        // Don't allow floating on mobile
        if (isMobile && !$isFloating) {
            toast.add("Floating mode not available on mobile", "info");
            return;
        }
        terminal.toggleFloating();
        // Increment key to force re-render
        viewModeKey++;
        // Wait a tick then fit all terminals
        tick().then(() => terminal.fitAll());
    }
    function minimize() {
        terminal.minimize();
    }

    function restore() {
        terminal.restore();
    }

    function closeSession(sessionId: string) {
        terminal.closeSession(sessionId);
    }

    function setActive(sessionId: string) {
        terminal.setActiveSession(sessionId);
    }

    function getStatusClass(status: string): string {
        switch (status) {
            case "connected":
                return "status-connected";
            case "connecting":
                return "status-connecting";
            default:
                return "status-disconnected";
        }
    }

    // Tab drag-out handlers for popping terminal to new window
    function handleTabDragStart(event: MouseEvent, sessionId: string) {
        if (event.button !== 0) return; // Only left click

        draggingTabId = sessionId;
        tabDragStart = { x: event.clientX, y: event.clientY };
        isDraggingTab = false;

        event.preventDefault();
    }

    function handleTabDragMove(event: MouseEvent) {
        if (!draggingTabId || !tabDragStart) return;

        const dx = Math.abs(event.clientX - tabDragStart.x);
        const dy = Math.abs(event.clientY - tabDragStart.y);

        // Consider it a drag if moved more than 20px
        if (dx > 20 || dy > 20) {
            isDraggingTab = true;
        }
    }

    function handleTabDragEnd(event: MouseEvent) {
        if (draggingTabId && isDraggingTab) {
            // Pop the terminal out to a new browser window
            popOutTerminal(draggingTabId, event.clientX, event.clientY);
        }

        draggingTabId = null;
        tabDragStart = null;
        isDraggingTab = false;
    }

    function popOutTerminal(sessionId: string, x: number, y: number) {
        const session = $terminal.sessions.get(sessionId);
        if (!session) return;

        // Detach as in-page floating window
        const containerName = session.name;
        const left = Math.max(50, x - 50);
        const top = Math.max(50, y - 30);

        terminal.detachSession(sessionId, left, top);
        toast.success(`Detached "${containerName}" to floating window`);
    }

    // Detached window drag state
    let draggingDetachedId: string | null = null;
    let detachedDragOffset = { x: 0, y: 0 };
    let resizingDetachedId: string | null = null;

    // Detached window drag handlers
    function handleDetachedMouseDown(event: MouseEvent, sessionId: string) {
        if ((event.target as HTMLElement).closest(".detached-actions")) return;
        draggingDetachedId = sessionId;
        const session = $terminal.sessions.get(sessionId);
        if (session) {
            detachedDragOffset = {
                x: event.clientX - session.detachedPosition.x,
                y: event.clientY - session.detachedPosition.y,
            };
        }
        event.preventDefault();
    }

    function handleDetachedMouseMove(event: MouseEvent) {
        if (draggingDetachedId) {
            const x = Math.max(0, event.clientX - detachedDragOffset.x);
            const y = Math.max(0, event.clientY - detachedDragOffset.y);
            terminal.setDetachedPosition(draggingDetachedId, x, y);
        }
        if (resizingDetachedId) {
            const session = $terminal.sessions.get(resizingDetachedId);
            if (session) {
                const width = Math.max(
                    300,
                    event.clientX - session.detachedPosition.x,
                );
                const height = Math.max(
                    200,
                    event.clientY - session.detachedPosition.y,
                );
                terminal.setDetachedSize(resizingDetachedId, width, height);
            }
        }
    }

    function handleDetachedMouseUp() {
        if (draggingDetachedId) {
            terminal.fitSession(draggingDetachedId);
        }
        if (resizingDetachedId) {
            terminal.fitSession(resizingDetachedId);
        }
        draggingDetachedId = null;
        resizingDetachedId = null;
    }

    function handleDetachedResizeStart(event: MouseEvent, sessionId: string) {
        resizingDetachedId = sessionId;
        event.preventDefault();
        event.stopPropagation();
    }

    function dockSession(sessionId: string) {
        const session = $terminal.sessions.get(sessionId);
        if (session) {
            terminal.attachSession(sessionId);
            toast.success(`Docked "${session.name}" back to terminal panel`);
        }
    }

    // Window event listeners
    onMount(() => {
        // Listen for container deletions
        window.addEventListener("container-deleted", handleContainerDeleted);

        // Force docked mode on mobile
        if (isMobile && $isFloating) {
            terminal.toggleFloating();
        }

        // Re-check mobile on window resize
        const handleResize = () => {
            const wasMobile = isMobile;
            isMobile = window.innerWidth < 768 || /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);

            // If switched to mobile while floating, go to docked
            if (isMobile && !wasMobile && $isFloating) {
                terminal.toggleFloating();
                toast.add("Switched to docked mode for mobile", "info");
            }
        };
        window.addEventListener('resize', handleResize);

        window.addEventListener("mousemove", handleMouseMove);
        window.addEventListener("mouseup", handleMouseUp);
        window.addEventListener("mousemove", handleTabDragMove);
        window.addEventListener("mouseup", handleTabDragEnd);
        window.addEventListener("mousemove", handleDetachedMouseMove);
        window.addEventListener("mouseup", handleDetachedMouseUp);

        return () => {
            window.removeEventListener('resize', handleResize);
        };
    });

    onDestroy(() => {
        window.removeEventListener("mousemove", handleMouseMove);
        window.removeEventListener("mouseup", handleMouseUp);
        window.removeEventListener("mousemove", handleTabDragMove);
        window.removeEventListener("mouseup", handleTabDragEnd);
        window.removeEventListener("mousemove", handleDetachedMouseMove);
        window.removeEventListener("mouseup", handleDetachedMouseUp);
    });
</script>

{#if $sessionCount > 0}
    {#if $isFloating}
        <!-- Floating Terminal -->
        <div class="floating-container">
            <div
                class="floating-terminal"
                class:minimized={isMinimized}
                class:focused={true}
                style="left: {floatingPosition.x}px; top: {floatingPosition.y}px; width: {floatingSize.width}px; height: {floatingSize.height}px;"
            >
                <!-- Header -->
                <div
                    class="floating-header"
                    on:mousedown={handleMouseDown}
                    role="toolbar"
                    tabindex="-1"
                >
                    <div class="floating-tabs">
                        {#each dockedSessions as [id, session] (id)}
                            <button
                                class="floating-tab"
                                class:active={id === activeId &&
                                    !showCreatePanel}
                                class:dragging={draggingTabId === id &&
                                    isDraggingTab}
                                on:click={() => {
                                    showCreatePanel = false;
                                    setActive(id);
                                }}
                                on:mousedown={(e) => handleTabDragStart(e, id)}
                                title="Drag out to pop to new window"
                            >
                                <span
                                    class="status-dot {getStatusClass(
                                        session.status,
                                    )}"
                                ></span>
                                <span class="tab-name">{session.name}</span>
                                <button
                                    class="tab-close"
                                    on:click|stopPropagation={() =>
                                        closeSession(id)}
                                    title="Close"
                                >
                                    ×
                                </button>
                            </button>
                        {/each}
                        <button
                            class="new-tab-btn"
                            class:active={showCreatePanel}
                            on:click={openCreatePanel}
                            title="New Terminal"
                        >
                            +
                        </button>
                    </div>

                    <div class="floating-actions">
                        <button on:click={toggleView} title="Toggle View">
                            ⬒
                        </button>
                        <button on:click={minimize} title="Minimize">−</button>
                        <button
                            on:click={() => activeId && closeSession(activeId)}
                            title="Close Current Terminal"
                            disabled={!activeId}
                        >×</button>
                    </div>
                </div>

                <!-- Body -->
                <div class="floating-body">
                    {#if showCreatePanel}
                        <!-- Inline Create Panel -->
                        <div class="create-panel">
                            <div class="create-panel-header">
                                <h3>New Terminal</h3>
                                {#if !isCreating}
                                    <button
                                        class="close-create"
                                        on:click={closeCreatePanel}>×</button
                                    >
                                {/if}
                            </div>

                            {#if isCreating}
                                <div class="create-progress">
                                    <div class="progress-header-inline">
                                        <span class="progress-percent"
                                            >{displayProgress}%</span
                                        >
                                    </div>
                                    <div class="progress-bar">
                                        <div
                                            class="progress-fill"
                                            style="width: {displayProgress}%"
                                        ></div>
                                    </div>
                                    <div class="progress-steps-inline">
                                        {#each progressSteps as step}
                                            <div
                                                class="progress-step-inline {getStepStatus(
                                                    step.id,
                                                )}"
                                            >
                                                <span class="step-indicator"></span>
                                                <span class="step-label"
                                                    >{step.label}</span
                                                >
                                            </div>
                                        {/each}
                                    </div>
                                    <p class="progress-message">
                                        {progressMessage}
                                    </p>
                                    {#if currentRole && progressStage === "configuring"}
                                        <div class="installing-tools-inline">
                                            <p class="installing-label">
                                                Installing {currentRole.name} tools:
                                            </p>
                                            <div class="tools-installing">
                                                {#each currentRole.tools as tool}
                                                    <span
                                                        class="tool-badge-installing"
                                                        >{tool}</span
                                                    >
                                                {/each}
                                            </div>
                                        </div>
                                    {/if}
                                    <div class="spinner"></div>
                                </div>
                            {:else}
                                <div class="create-panel-content">
                                    <!-- Role Selection -->
                                    <div class="create-section">
                                        <h4>Environment</h4>
                                        <div class="role-grid">
                                            {#each roles as role}
                                                <button
                                                    class="role-card"
                                                    class:selected={selectedRole ===
                                                        role.id}
                                                    on:click={() =>
                                                        (selectedRole =
                                                            role.id)}
                                                    title={role.desc}
                                                >
                                                    <PlatformIcon platform={role.id} size={24} />
                                                    <span class="role-name"
                                                        >{role.name}</span
                                                    >
                                                </button>
                                            {/each}
                                        </div>
                                        {#if currentRole}
                                            <div class="role-info-compact">
                                                <div class="role-header-row">
                                                    <PlatformIcon platform={currentRole.id} size={16} />
                                                    <span class="role-name-sm">{currentRole.name}</span>
                                                    <span class="role-os-badge"><PlatformIcon platform={currentRole.recommendedOS.toLowerCase()} size={16} /> {currentRole.recommendedOS}</span>
                                                </div>
                                                <div class="role-tools">
                                                    {#each currentRole.tools as tool}
                                                        <span class="tool-badge">{tool}</span>
                                                    {/each}
                                                </div>
                                            </div>
                                        {/if}
                                    </div>

                                    <!-- OS Selection -->
                                    <div class="create-section">
                                        <h4>Operating System</h4>
                                        <div class="os-grid">
                                            {#each images as image (image.name)}
                                                <button
                                                    class="os-card"
                                                    on:click={() =>
                                                        selectAndCreate(
                                                            image.name,
                                                        )}
                                                >
                                                    <PlatformIcon platform={image.name} size={24} />
                                                    <span class="os-name"
                                                        >{image.display_name ||
                                                            image.name}</span
                                                    >
                                                </button>
                                            {/each}
                                            <button
                                                class="os-card"
                                                on:click={() =>
                                                    selectAndCreate("custom")}
                                            >
                                                <PlatformIcon platform="custom" size={24} />
                                                <span class="os-name"
                                                    >Custom</span
                                                >
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            {/if}
                        </div>
                    {:else}
                        {#each dockedSessions as [id, session] (`float-${viewModeKey}-${id}`)}
                            <div
                                class="terminal-panel"
                                class:active={id === activeId}
                            >
                                <TerminalPanel {session} />
                            </div>
                        {/each}
                    {/if}
                </div>

                <!-- Resize Handle -->
                <div
                    class="resize-handle"
                    on:mousedown={handleResizeStart}
                    role="separator"
                    tabindex="-1"
                    on:keydown={() => {}}
                ></div>
            </div>
        </div>

        <!-- Minimized bar -->
        {#if isMinimized}
            <div class="minimized-bar">
                <button class="restore-btn" on:click={restore}>
                    <span class="restore-icon">↑</span>
                    <span
                        >{$sessionCount} Terminal{$sessionCount > 1
                            ? "s"
                            : ""}</span
                    >
                </button>
            </div>
        {/if}
    {:else}
        <!-- Docked Terminal -->
        {#if isMinimized}
            <div class="minimized-bar docked-minimized">
                <button class="restore-btn" on:click={restore}>
                    <span class="restore-icon">↑</span>
                    <span
                        >{$sessionCount} Terminal{$sessionCount > 1
                            ? "s"
                            : ""}</span
                    >
                </button>
            </div>
        {:else}
            <div class="docked-terminal">
                <!-- Header -->
                <div class="docked-header">
                    <div class="docked-tabs">
                        {#each dockedSessions as [id, session] (id)}
                            <button
                                class="docked-tab"
                                class:active={id === activeId &&
                                    !showCreatePanel}
                                class:dragging={draggingTabId === id &&
                                    isDraggingTab}
                                on:click={() => {
                                    showCreatePanel = false;
                                    setActive(id);
                                }}
                                on:mousedown={(e) => handleTabDragStart(e, id)}
                                title="Drag out to pop to new window"
                            >
                                <span
                                    class="status-dot {getStatusClass(
                                        session.status,
                                    )}"
                                ></span>
                                <span class="tab-name">{session.name}</span>
                                <button
                                    class="tab-close"
                                    on:click|stopPropagation={() =>
                                        closeSession(id)}
                                    title="Close"
                                >
                                    ×
                                </button>
                            </button>
                        {/each}
                        <button
                            class="docked-tab new-tab-btn"
                            class:active={showCreatePanel}
                            on:click={openCreatePanel}
                            title="New Terminal"
                        >
                            +
                        </button>
                    </div>

                    <div class="docked-actions">
                        <button
                            class="btn btn-secondary btn-sm"
                            on:click={toggleView}
                            title="Float"
                        >
                            ⬔
                        </button>
                        <button
                            class="btn btn-secondary btn-sm"
                            on:click={minimize}
                            title="Minimize"
                        >
                            −
                        </button>
                        <button
                            class="btn btn-danger btn-sm"
                            on:click={() => activeId && closeSession(activeId)}
                            title="Close Current Terminal"
                            disabled={!activeId}
                        >
                            ×
                        </button>
                    </div>
                </div>

                <!-- Body -->
                <div class="docked-body">
                    {#if showCreatePanel}
                        <!-- Inline Create Panel for Docked Mode -->
                        <div class="create-panel docked-create">
                            <div class="create-panel-header">
                                <h3>New Terminal</h3>
                                {#if !isCreating}
                                    <button
                                        class="close-create"
                                        on:click={closeCreatePanel}
                                        >× Cancel</button
                                    >
                                {/if}
                            </div>

                            {#if isCreating}
                                <div class="create-progress">
                                    <div class="progress-header-inline">
                                        <span class="progress-percent"
                                            >{displayProgress}%</span
                                        >
                                    </div>
                                    <div class="progress-bar">
                                        <div
                                            class="progress-fill"
                                            style="width: {displayProgress}%"
                                        ></div>
                                    </div>
                                    <div class="progress-steps-inline">
                                        {#each progressSteps as step}
                                            <div
                                                class="progress-step-inline {getStepStatus(
                                                    step.id,
                                                )}"
                                            >
                                                <span class="step-icon"
                                                    >{step.icon}</span
                                                >
                                                <span class="step-label"
                                                    >{step.label}</span
                                                >
                                            </div>
                                        {/each}
                                    </div>
                                    <p class="progress-message">
                                        {progressMessage}
                                    </p>
                                    {#if currentRole && progressStage === "configuring"}
                                        <div class="installing-tools-inline">
                                            <p class="installing-label">
                                                Installing {currentRole.name} tools:
                                            </p>
                                            <div class="tools-installing">
                                                {#each currentRole.tools as tool}
                                                    <span
                                                        class="tool-badge-installing"
                                                        >{tool}</span
                                                    >
                                                {/each}
                                            </div>
                                        </div>
                                    {/if}
                                    <div class="spinner"></div>
                                </div>
                            {:else}
                                <div
                                    class="create-panel-content docked-content"
                                >
                                    <!-- Role Selection -->
                                    <div class="create-section">
                                        <h4>Environment</h4>
                                        <div class="role-grid">
                                            {#each roles as role}
                                                <button
                                                    class="role-card"
                                                    class:selected={selectedRole ===
                                                        role.id}
                                                    on:click={() =>
                                                        (selectedRole =
                                                            role.id)}
                                                    title={role.desc}
                                                >
                                                    <PlatformIcon platform={role.id} size={24} />
                                                    <span class="role-name"
                                                        >{role.name}</span
                                                    >
                                                </button>
                                            {/each}
                                        </div>
                                        {#if currentRole}
                                            <div class="role-info-compact">
                                                <div class="role-header-row">
                                                    <PlatformIcon platform={currentRole.id} size={16} />
                                                    <span class="role-name-sm">{currentRole.name}</span>
                                                    <span class="role-os-badge"><PlatformIcon platform={currentRole.recommendedOS.toLowerCase()} size={16} /> {currentRole.recommendedOS}</span>
                                                </div>
                                                <div class="role-tools">
                                                    {#each currentRole.tools as tool}
                                                        <span class="tool-badge">{tool}</span>
                                                    {/each}
                                                </div>
                                            </div>
                                        {/if}
                                    </div>

                                    <!-- OS Selection -->
                                    <div class="create-section">
                                        <h4>Operating System</h4>
                                        <div class="os-grid docked-grid">
                                            {#each images as image (image.name)}
                                                <button
                                                    class="os-card"
                                                    on:click={() =>
                                                        selectAndCreate(
                                                            image.name,
                                                        )}
                                                >
                                                    <PlatformIcon platform={image.name} size={24} />
                                                    <span class="os-name"
                                                        >{image.display_name ||
                                                            image.name}</span
                                                    >
                                                    {#if image.popular}
                                                        <span
                                                            class="popular-badge"
                                                            >Popular</span
                                                        >
                                                    {/if}
                                                </button>
                                            {/each}
                                            <button
                                                class="os-card"
                                                on:click={() =>
                                                    selectAndCreate("custom")}
                                            >
                                                <PlatformIcon platform="custom" size={24} />
                                                <span class="os-name"
                                                    >Custom Image</span
                                                >
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            {/if}
                        </div>
                    {:else}
                        {#each dockedSessions as [id, session] (`dock-${viewModeKey}-${id}`)}
                            <div
                                class="terminal-panel"
                                class:active={id === activeId}
                            >
                                <TerminalPanel {session} />
                            </div>
                        {/each}
                    {/if}
                </div>
            </div>
        {/if}
    {/if}
{/if}

<!-- Detached Floating Windows -->
{#each detachedSessions as [id, session] (id)}
    <div
        class="detached-window"
        style="left: {session.detachedPosition.x}px; top: {session
            .detachedPosition.y}px; width: {session.detachedSize
            .width}px; height: {session.detachedSize.height}px; z-index: {session.detachedZIndex};"
        on:mousedown={() => terminal.bringToFront(id)}
        role="dialog"
        tabindex="-1"
    >
        <div
            class="detached-header"
            on:mousedown={(e) => handleDetachedMouseDown(e, id)}
            on:dblclick={() => dockSession(id)}
            role="toolbar"
            tabindex="-1"
        >
            <span class="detached-title">{session.name}</span>
            <span class="detached-status status-{session.status}"></span>
            <div class="detached-actions">
                <button
                    on:click={() => dockSession(id)}
                    title="Dock back to terminal panel"
                >
                    ⬒
                </button>
                <button
                    on:click={() => {
                        terminal.closeSession(id);
                        toast.success(`Closed "${session.name}"`);
                    }}
                    title="Close terminal"
                >
                    ×
                </button>
            </div>
        </div>
        <div class="detached-body" id="detached-terminal-{id}">
            <TerminalPanel {session} />
        </div>
        <div
            class="detached-resize-handle"
            on:mousedown={(e) => handleDetachedResizeStart(e, id)}
            role="button"
            tabindex="-1"
        ></div>
    </div>
{/each}

<style>
    /* Floating Container */
    .floating-container {
        position: fixed;
        inset: 0;
        pointer-events: none;
        z-index: 1000;
    }

    .floating-terminal {
        position: absolute;
        display: flex;
        flex-direction: column;
        background: var(--bg-card);
        border: 1px solid var(--border);
        box-shadow:
            0 0 40px rgba(0, 0, 0, 0.9),
            0 0 1px var(--accent);
        pointer-events: auto;
        overflow: hidden;
        min-width: 400px;
        min-height: 300px;
    }

    .floating-terminal.focused {
        border-color: var(--accent);
        box-shadow:
            0 0 40px rgba(0, 0, 0, 0.9),
            0 0 10px rgba(0, 255, 65, 0.2);
    }

    .floating-terminal.minimized {
        display: none;
    }

    .floating-header {
        display: flex;
        align-items: center;
        padding: 6px 12px;
        background: #111;
        border-bottom: 1px solid var(--border);
        cursor: move;
        user-select: none;
        gap: 8px;
    }

    .floating-tabs {
        display: flex;
        flex: 1;
        gap: 2px;
        overflow-x: auto;
        scrollbar-width: none;
    }

    .floating-tabs::-webkit-scrollbar {
        display: none;
    }

    .floating-tab {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        background: transparent;
        border: 1px solid transparent;
        color: var(--text-muted);
        font-size: 11px;
        font-family: var(--font-mono);
        cursor: pointer;
        white-space: nowrap;
        transition: all 0.15s ease;
    }

    .floating-tab:hover {
        background: rgba(255, 255, 255, 0.05);
        color: var(--text-secondary);
    }

    .floating-tab.active {
        background: rgba(0, 255, 65, 0.1);
        border-color: var(--accent);
        color: var(--accent);
    }

    .floating-tab.dragging,
    .docked-tab.dragging {
        opacity: 0.5;
        transform: scale(0.95);
        cursor: grabbing;
    }

    .floating-tab:not(.dragging),
    .docked-tab:not(.dragging) {
        cursor: grab;
    }

    .floating-actions {
        display: flex;
        gap: 4px;
        align-items: center;
    }

    .floating-actions button {
        background: none;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 4px 8px;
        font-size: 12px;
        font-family: var(--font-mono);
        transition: color 0.15s ease;
    }

    .floating-actions button:hover {
        color: var(--text);
    }

    .floating-body {
        flex: 1;
        overflow: hidden;
        background: #0a0a0a;
        position: relative;
    }

    .resize-handle {
        position: absolute;
        bottom: 0;
        right: 0;
        width: 16px;
        height: 16px;
        cursor: nwse-resize;
        background: linear-gradient(135deg, transparent 50%, var(--border) 50%);
    }

    .resize-handle:hover {
        background: linear-gradient(135deg, transparent 50%, var(--accent) 50%);
    }

    /* Docked Terminal Container */
    .docked-terminal {
        position: fixed;
        bottom: 0;
        left: 0;
        right: 0;
        background: var(--bg);
        border-top: 1px solid var(--border);
        z-index: 1000;
        display: flex;
        flex-direction: column;
    }

    /* Mobile-specific styles */
    @media (max-width: 768px) {
        .docked-terminal {
            /* Full height on mobile for better usability */
            height: 100vh;
            top: 0;
            border-top: none;
        }

        .docked-toolbar {
            /* Larger touch targets */
            padding: 12px;
            min-height: 48px;
        }

        .docked-toolbar button {
            min-width: 44px;
            min-height: 44px;
            font-size: 18px;
        }

        .docked-tabs {
            /* Better spacing for touch */
            gap: 8px;
            padding: 8px;
        }

        .docked-tab {
            min-height: 44px;
            padding: 8px 16px;
            font-size: 14px;
        }

        /* Hide floating toggle on mobile */
        .docked-toolbar button[title="Toggle View"] {
            display: none;
        }
    }

    .docked-header {
        display: flex;
        align-items: center;
        gap: 16px;
    }

    .docked-tabs {
        display: flex;
        gap: 4px;
        overflow-x: auto;
        padding-right: 8px;
        align-items: center;
    }

    .docked-tab {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 6px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 4px 4px 0 0;
        color: var(--text-secondary);
        font-size: 13px;
        cursor: pointer;
        transition: all 0.15s ease;
        border-bottom: none;
        margin-bottom: -1px;
        white-space: nowrap;
    }

    .docked-tab:hover {
        background: var(--bg-secondary);
        color: var(--text);
    }

    .docked-tab.active {
        background: var(--bg);
        border-color: var(--accent);
        border-bottom-color: var(--bg);
        color: var(--text);
    }

    .docked-tabs .new-tab-btn {
        background: transparent;
        border: 1px dashed var(--border);
        color: var(--text-muted);
        padding: 4px 8px;
        border-radius: 4px;
        font-size: 12px;
        cursor: pointer;
        transition: all 0.15s ease;
        margin-left: 4px;
        height: 24px;
        display: flex;
        align-items: center;
    }

    .docked-tabs .new-tab-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
        background: rgba(0, 217, 255, 0.1);
    }

    .docked-tabs .new-tab-btn.active {
        border-style: solid;
        border-color: var(--accent);
        color: var(--accent);
        background: var(--bg);
        border-bottom-color: var(--bg);
    }

    /* Floating new tab button */
    .floating-tabs .new-tab-btn {
        background: transparent;
        border: none;
        color: var(--text-muted);
        padding: 6px 10px;
        font-size: 16px;
        cursor: pointer;
        transition: all 0.15s ease;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .floating-tabs .new-tab-btn:hover {
        color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
    }

    .floating-tabs .new-tab-btn.active {
        color: var(--accent);
        background: rgba(0, 255, 65, 0.15);
    }

    .docked-actions {
        display: flex;
        gap: 8px;
    }

    .docked-body {
        flex: 1;
        position: relative;
        overflow: hidden;
        background: #050505;
    }

    /* Common Styles */
    .terminal-panel {
        position: absolute;
        inset: 0;
        display: none;
        overflow: hidden;
    }

    .terminal-panel.active {
        display: flex;
        flex-direction: column;
    }

    .status-dot {
        width: 6px;
        height: 6px;
    }

    .status-connected {
        background: var(--green);
    }

    .status-connecting {
        background: var(--yellow);
        animation: pulse 1s infinite;
    }

    .status-disconnected {
        background: var(--red);
    }

    .tab-name {
        max-width: 120px;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .tab-close {
        background: none;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 0 4px;
        font-size: 14px;
        line-height: 1;
    }

    .tab-close:hover {
        color: var(--red);
    }

    /* Minimized Bar */
    .minimized-bar {
        position: fixed;
        bottom: 16px;
        right: 16px;
        z-index: 1001;
        pointer-events: auto;
    }

    .minimized-bar.docked-minimized {
        bottom: 0;
        left: 0;
        right: 0;
        border-radius: 0;
        display: flex;
        justify-content: center;
        background: var(--bg-elevated);
        border-top: 1px solid var(--border);
        padding: 8px;
    }

    .restore-btn {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        background: var(--bg-card);
        border: 1px solid var(--accent);
        color: var(--accent);
        font-family: var(--font-mono);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .restore-btn:hover {
        background: var(--accent);
        color: var(--bg);
    }

    .restore-icon {
        font-size: 14px;
    }

    /* Create Panel Styles */
    .create-panel {
        position: absolute;
        inset: 0;
        background: #0d0d0d;
        display: flex;
        flex-direction: column;
        padding: 16px;
        overflow-y: auto;
    }

    .create-panel-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
        padding-bottom: 12px;
        border-bottom: 1px solid var(--border);
    }

    .create-panel-header h3 {
        margin: 0;
        font-size: 14px;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--accent);
    }

    .close-create {
        background: none;
        border: 1px solid var(--border);
        color: var(--text-muted);
        padding: 4px 10px;
        font-size: 11px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s;
    }

    .close-create:hover {
        border-color: var(--red);
        color: var(--red);
    }

    /* Create Panel Content */
    .create-panel-content {
        display: flex;
        flex-direction: column;
        gap: 24px;
        padding: 16px;
        overflow-y: auto;
        max-height: 100%;
    }

    .create-panel-content.docked-content {
        max-width: 900px;
        margin: 0 auto;
        width: 100%;
    }

    .create-section {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .create-section h4 {
        font-size: 13px;
        font-weight: 600;
        color: var(--accent);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        margin: 0;
        font-family: var(--font-mono);
    }

    .role-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
        gap: 8px;
    }

    .role-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 6px;
        padding: 12px 8px;
        background: #1a1a1a;
        border: 1px solid #333;
        cursor: pointer;
        transition: all 0.15s;
        font-family: var(--font-mono);
    }

    .role-card:hover {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
    }

    .role-card.selected {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.15);
        box-shadow: 0 0 10px rgba(0, 255, 65, 0.2);
    }

    .role-icon {
        font-size: 28px;
        filter: drop-shadow(0 0 4px rgba(255, 255, 255, 0.3));
    }

    .role-name {
        font-size: 11px;
        color: #e0e0e0;
        text-align: center;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 100%;
    }

    /* Hero Stat Compact (Anime Style) */
    .hero-stat-compact {
        margin-top: 10px;
        position: relative;
        z-index: 50;
        overflow: visible;
    }

    .hero-identity {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 10px;
        background: linear-gradient(
            135deg,
            var(--bg-elevated) 0%,
            rgba(0, 255, 65, 0.05) 100%
        );
        border: 1px solid var(--border);
        border-radius: 6px;
        border-left: 2px solid var(--accent);
    }

    .hero-icon-sm {
        font-size: 20px;
        filter: drop-shadow(0 0 3px var(--accent));
    }

    .hero-title-sm {
        flex: 1;
        font-size: 11px;
        font-weight: 600;
        color: var(--text);
        font-family: var(--font-mono);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .stat-toggle-sm {
        display: flex;
        align-items: center;
        gap: 3px;
        padding: 4px 8px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 3px;
        cursor: pointer;
        transition: all 0.15s;
        font-family: var(--font-mono);
        font-size: 9px;
        color: var(--text);
    }

    .stat-toggle-sm:hover {
        border-color: var(--accent);
        background: var(--accent-dim);
    }

    .stat-toggle-icon {
        font-size: 7px;
        color: var(--accent);
    }

    /* Hero Stat Popover (Small) */
    .hero-stat-popover-sm {
        position: absolute;
        top: calc(100% + 6px);
        left: 0;
        width: 260px;
        background: var(--bg-elevated);
        border: 1px solid var(--accent);
        border-radius: 6px;
        padding: 10px;
        z-index: 1000;
        box-shadow:
            0 4px 16px rgba(0, 0, 0, 0.8),
            0 0 12px rgba(0, 255, 65, 0.2);
        animation: popoverSlide 0.15s ease-out;
    }

    @keyframes popoverSlide {
        from {
            opacity: 0;
            transform: translateY(-6px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .stat-header-sm {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
        padding-bottom: 6px;
        border-bottom: 1px solid var(--border);
    }

    .stat-class-sm {
        font-size: 10px;
        font-weight: 600;
        color: var(--accent);
        font-family: var(--font-mono);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .stat-level-sm {
        font-size: 8px;
        color: var(--warning, #ffd93d);
        font-family: var(--font-mono);
        padding: 2px 5px;
        background: rgba(255, 217, 61, 0.1);
        border-radius: 2px;
    }

    .stat-bars-sm {
        display: flex;
        flex-direction: column;
        gap: 4px;
        margin-bottom: 8px;
    }

    .stat-row-sm {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .stat-label-sm {
        font-size: 10px;
        width: 24px;
        flex-shrink: 0;
    }

    .stat-bar-track-sm {
        flex: 1;
        height: 5px;
        background: var(--bg-tertiary);
        border-radius: 2px;
        overflow: hidden;
    }

    .stat-bar-fill-sm {
        height: 100%;
        background: linear-gradient(90deg, var(--accent) 0%, #00ff88 100%);
        border-radius: 2px;
        transition: width 0.3s ease;
        box-shadow: 0 0 4px var(--accent);
    }

    .stat-bar-fill-sm.defense {
        background: linear-gradient(90deg, #00d9ff 0%, #0099ff 100%);
        box-shadow: 0 0 4px #00d9ff;
    }

    .stat-bar-fill-sm.speed {
        background: linear-gradient(90deg, #ff6b6b 0%, #ffd93d 100%);
        box-shadow: 0 0 4px #ff6b6b;
    }

    .stat-info-row-sm {
        display: flex;
        gap: 8px;
        margin-bottom: 8px;
    }

    .stat-info-sm {
        flex: 1;
        font-size: 9px;
        color: var(--text-muted);
        font-family: var(--font-mono);
        padding: 4px 6px;
        background: var(--bg-tertiary);
        border-radius: 3px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .abilities-row-sm {
        display: flex;
        flex-wrap: wrap;
        gap: 3px;
    }

    .ability-tag-sm {
        padding: 2px 5px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid var(--accent);
        border-radius: 2px;
        font-size: 8px;
        color: var(--accent);
        font-family: var(--font-mono);
    }

    /* Role info compact - simple tool badges */
    .role-info-compact {
        margin-top: 10px;
        padding: 10px;
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: 6px;
        border-left: 2px solid var(--accent);
    }

    .role-header-row {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;
    }

    .role-icon-sm {
        font-size: 18px;
    }

    .role-name-sm {
        font-size: 12px;
        font-weight: 600;
        color: var(--text);
        font-family: var(--font-mono);
    }

    .role-os-badge {
        margin-left: auto;
        font-size: 10px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    .role-tools {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
    }

    .tool-badge {
        padding: 3px 8px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.3);
        border-radius: 3px;
        font-size: 10px;
        color: var(--accent);
        font-family: var(--font-mono);
    }

    .os-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
        gap: 8px;
    }

    .docked-grid {
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        gap: 12px;
        max-width: 800px;
        margin: 0 auto;
    }

    .docked-create {
        align-items: center;
        justify-content: center;
    }

    .docked-create .create-panel-header {
        width: 100%;
        max-width: 800px;
    }

    .os-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 6px;
        padding: 12px 8px;
        background: #1a1a1a;
        border: 1px solid #333;
        cursor: pointer;
        transition: all 0.15s;
        font-family: var(--font-mono);
        position: relative;
    }

    .os-card:hover {
        border-color: #00d9ff;
        background: rgba(0, 217, 255, 0.1);
    }

    .os-card.selected {
        border-color: #00d9ff;
        background: rgba(0, 217, 255, 0.15);
        box-shadow: 0 0 10px rgba(0, 217, 255, 0.2);
    }

    .os-icon {
        font-size: 28px;
        filter: drop-shadow(0 0 4px rgba(255, 255, 255, 0.3));
    }

    .os-name {
        font-size: 11px;
        color: #e0e0e0;
        text-align: center;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 100%;
    }

    .popular-badge {
        position: absolute;
        top: 2px;
        right: 2px;
        font-size: 7px;
        padding: 1px 3px;
        background: var(--accent);
        color: var(--bg);
        text-transform: uppercase;
    }

    /* Create Progress */
    .create-progress {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 40px 20px;
        text-align: center;
    }

    .create-progress .progress-bar {
        width: 100%;
        max-width: 300px;
        height: 4px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        margin-bottom: 12px;
        overflow: hidden;
    }

    .create-progress .progress-fill {
        height: 100%;
        background: var(--accent);
        transition: width 0.3s ease;
    }

    .create-progress .progress-info {
        display: flex;
        justify-content: space-between;
        width: 100%;
        max-width: 300px;
        margin-bottom: 8px;
    }

    .create-progress .progress-stage {
        font-size: 11px;
        text-transform: uppercase;
        color: var(--accent);
    }

    .create-progress .progress-percent {
        font-size: 11px;
        color: var(--text-muted);
    }

    .create-progress .progress-message {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0 0 16px;
    }

    .spinner {
        width: 24px;
        height: 24px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
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

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    /* Progress Steps Inline */
    .progress-header-inline {
        margin-bottom: 8px;
    }

    .progress-header-inline .progress-percent {
        font-size: 16px;
        font-weight: 600;
        color: var(--accent);
    }

    .progress-steps-inline {
        display: flex;
        flex-wrap: wrap;
        justify-content: center;
        gap: 6px;
        margin: 12px 0;
    }

    .progress-step-inline {
        display: flex;
        align-items: center;
        gap: 3px;
        padding: 4px 8px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 4px;
        font-size: 10px;
        font-family: var(--font-mono);
        transition: all 0.2s;
    }

    .progress-step-inline.pending {
        opacity: 0.4;
        color: var(--text-muted);
    }

    .progress-step-inline.active {
        border-color: var(--accent);
        color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
        animation: pulse 1.5s infinite;
    }

    .progress-step-inline.completed {
        border-color: var(--green);
        color: var(--green);
    }

    .progress-step-inline .step-indicator {
        width: 12px;
        height: 12px;
        border-radius: 50%;
        border:2px solid var(--border);
        background: transparent;
        flex-shrink: 0;
    }

    .progress-step-inline.active .step-indicator {
        border-color: var(--accent);
        background: var(--accent);
        animation: pulse 1s infinite;
    }

    .progress-step-inline.completed .step-indicator {
        border-color: var(--green);
        background: var(--green);
    }

    .progress-step-inline .step-label {
        font-size: 9px;
        text-transform: uppercase;
    }

    .installing-tools-inline {
        margin: 12px 0;
        padding: 10px;
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: 6px;
    }

    .installing-tools-inline .installing-label {
        font-size: 10px;
        color: var(--text-muted);
        margin: 0 0 8px 0;
        font-family: var(--font-mono);
    }

    .tools-installing {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
    }

    .tool-badge-installing {
        padding: 3px 6px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid var(--accent);
        border-radius: 3px;
        font-size: 9px;
        color: var(--accent);
        font-family: var(--font-mono);
        animation: pulse 1.5s infinite;
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

    /* Detached Window Styles */
    .detached-window {
        position: fixed;
        display: flex;
        flex-direction: column;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
        pointer-events: auto;
        overflow: hidden;
    }

    .detached-header {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background: var(--bg-elevated);
        border-bottom: 1px solid var(--border);
        cursor: grab;
        user-select: none;
    }

    .detached-header:active {
        cursor: grabbing;
    }

    .detached-title {
        flex: 1;
        font-size: 12px;
        font-weight: 500;
        color: var(--text);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .detached-status {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        flex-shrink: 0;
    }

    .detached-status.status-connected {
        background: var(--green);
        box-shadow: 0 0 6px var(--green);
    }

    .detached-status.status-connecting {
        background: var(--warning);
        animation: pulse 1s infinite;
    }

    .detached-status.status-disconnected,
    .detached-status.status-error {
        background: var(--danger);
    }

    .detached-actions {
        display: flex;
        gap: 4px;
    }

    .detached-actions button {
        background: none;
        border: none;
        color: var(--text-muted);
        font-size: 14px;
        padding: 4px 6px;
        cursor: pointer;
        border-radius: 4px;
        transition: all 0.15s;
    }

    .detached-actions button:hover {
        background: rgba(255, 255, 255, 0.1);
        color: var(--text);
    }

    .detached-body {
        flex: 1;
        overflow: hidden;
        background: var(--bg-terminal);
    }

    .detached-resize-handle {
        position: absolute;
        bottom: 0;
        right: 0;
        width: 16px;
        height: 16px;
        cursor: se-resize;
        background: linear-gradient(
            135deg,
            transparent 50%,
            var(--border) 50%,
            var(--border) 60%,
            transparent 60%,
            transparent 70%,
            var(--border) 70%,
            var(--border) 80%,
            transparent 80%
        );
        opacity: 0.5;
        transition: opacity 0.15s;
    }

    .detached-resize-handle:hover {
        opacity: 1;
    }
</style>
