<script lang="ts">
    import { onMount, onDestroy, tick } from "svelte";
    import { terminal, sessionCount, isFloating } from "$stores/terminal";
    import { containers, type ProgressEvent } from "$stores/containers";
    import { toast } from "$stores/toast";
    import { api } from "$utils/api";
    import TerminalPanel from "./TerminalPanel.svelte";

    // Track view mode changes to force terminal re-render
    let viewModeKey = 0;

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
    let images: Array<{
        name: string;
        display_name: string;
        description: string;
        category: string;
        popular?: boolean;
    }> = [];

    // Image icons
    const imageIcons: Record<string, string> = {
        ubuntu: "🟠",
        debian: "🔴",
        alpine: "🔵",
        fedora: "🔵",
        centos: "🟣",
        rocky: "🟢",
        alma: "🟣",
        arch: "🔷",
        kali: "🐉",
        parrot: "🦜",
        mint: "🌿",
        elementary: "🪶",
        devuan: "🔘",
        blackarch: "🖤",
        manjaro: "🟩",
        opensuse: "🦎",
        tumbleweed: "🌀",
        gentoo: "🗿",
        void: "⬛",
        nixos: "❄️",
        slackware: "📦",
        busybox: "📦",
        amazonlinux: "🟧",
        oracle: "🔶",
        rhel: "🎩",
        openeuler: "🔵",
        clearlinux: "💎",
        photon: "☀️",
        raspberrypi: "🍓",
        scientific: "🔬",
        rancheros: "🐄",
        custom: "📦",
    };

    // Available roles
    const roles = [
        {
            id: "standard",
            name: "The Minimalist",
            icon: "🧘",
            desc: "I use Arch btw. Just give me a shell.",
        },
        {
            id: "node",
            name: "10x JS Ninja",
            icon: "🚀",
            desc: "Ship fast, break things, npm install everything.",
        },
        {
            id: "python",
            name: "Data Wizard",
            icon: "🧙‍♂️",
            desc: "Import antigravity. I speak in list comprehensions.",
        },
        {
            id: "go",
            name: "The Gopher",
            icon: "🐹",
            desc: "If err != nil { panic(err) }. Simplicity is key.",
        },
        {
            id: "neovim",
            name: "Neovim God",
            icon: "⌨️",
            desc: "My config is longer than your code. Mouse? What mouse?",
        },
        {
            id: "devops",
            name: "YAML Herder",
            icon: "☸️",
            desc: "I don't write code, I write config. Prod is my playground.",
        },
        {
            id: "overemployed",
            name: "The Overemployed",
            icon: "💼",
            desc: "Working 4 remote jobs. Need max efficiency.",
        },
    ];

    function getIcon(imageName: string): string {
        const lower = imageName.toLowerCase();
        for (const [key, icon] of Object.entries(imageIcons)) {
            if (lower.includes(key)) return icon;
        }
        return "🐧";
    }

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

        isCreating = true;
        progress = 0;
        progressMessage = "Starting...";
        progressStage = "initializing";

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

                // Defensive checks for container object
                const containerName = container?.name || name;
                const containerId = container?.id || container?.db_id;

                if (!containerId) {
                    toast.error(
                        "Container created but ID not found. Please refresh.",
                    );
                    return;
                }

                toast.success(`Terminal "${containerName}" created!`);
                // Create session and connect
                terminal.createSession(containerId, containerName);
            },
            // onError
            (error) => {
                isCreating = false;
                showCreatePanel = false;
                // Ensure error is a string
                const errorMsg =
                    typeof error === "string"
                        ? error
                        : error?.message ||
                          error?.error ||
                          "Failed to create terminal";
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

    // Actions
    async function toggleViewMode() {
        terminal.toggleViewMode();
        // Increment key to force re-render of terminal panels
        viewModeKey++;
        // Wait for DOM update then fit terminals with multiple attempts
        await tick();
        setTimeout(() => terminal.fitAll(), 100);
        setTimeout(() => terminal.fitAll(), 300);
        setTimeout(() => terminal.fitAll(), 500);
    }

    function minimize() {
        terminal.minimize();
    }

    function restore() {
        terminal.restore();
    }

    function closeAll() {
        if (confirm("Close all terminal sessions?")) {
            terminal.closeAllSessions();
        }
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

        // Get container info
        const containerId = session.containerId;
        const containerName = session.name;

        // Calculate window position (offset from cursor)
        const windowWidth = 800;
        const windowHeight = 600;
        const left = Math.max(0, x - 50);
        const top = Math.max(0, y - 30);

        // Open new window with the terminal
        const newWindow = window.open(
            `${window.location.origin}?terminal=${containerId}&name=${encodeURIComponent(containerName)}`,
            `terminal_${containerId}`,
            `width=${windowWidth},height=${windowHeight},left=${left},top=${top},menubar=no,toolbar=no,location=no,status=no`,
        );

        if (newWindow) {
            // Close the session in current window
            terminal.closeSession(sessionId);
            toast.success(`Popped out "${containerName}" to new window`);
        } else {
            toast.error("Failed to open new window. Check popup blocker.");
        }
    }

    // Window event listeners
    onMount(() => {
        window.addEventListener("mousemove", handleMouseMove);
        window.addEventListener("mouseup", handleMouseUp);
        window.addEventListener("mousemove", handleTabDragMove);
        window.addEventListener("mouseup", handleTabDragEnd);
    });

    onDestroy(() => {
        window.removeEventListener("mousemove", handleMouseMove);
        window.removeEventListener("mouseup", handleMouseUp);
        window.removeEventListener("mousemove", handleTabDragMove);
        window.removeEventListener("mouseup", handleTabDragEnd);
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
                        {#each sessions as [id, session] (id)}
                            <button
                                class="floating-tab"
                                class:active={id === activeId}
                                class:dragging={draggingTabId === id &&
                                    isDraggingTab}
                                on:click={() => setActive(id)}
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
                            on:click={openCreatePanel}
                            title="New Terminal"
                        >
                            + New
                        </button>
                    </div>

                    <div class="floating-actions">
                        <button on:click={toggleViewMode} title="Dock Terminal">
                            ⬒ Dock
                        </button>
                        <button on:click={minimize} title="Minimize">−</button>
                        <button on:click={closeAll} title="Close All">×</button>
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
                                    <div class="progress-bar">
                                        <div
                                            class="progress-fill"
                                            style="width: {progress}%"
                                        ></div>
                                    </div>
                                    <div class="progress-info">
                                        <span class="progress-stage"
                                            >{progressStage}</span
                                        >
                                        <span class="progress-percent"
                                            >{progress}%</span
                                        >
                                    </div>
                                    <p class="progress-message">
                                        {progressMessage}
                                    </p>
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
                                                    <span class="role-icon"
                                                        >{role.icon}</span
                                                    >
                                                    <span class="role-name"
                                                        >{role.name}</span
                                                    >
                                                </button>
                                            {/each}
                                        </div>
                                        <p class="role-desc">
                                            {roles.find(
                                                (r) => r.id === selectedRole,
                                            )?.desc}
                                        </p>
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
                                                    <span class="os-icon"
                                                        >{getIcon(
                                                            image.name,
                                                        )}</span
                                                    >
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
                                                <span class="os-icon">📦</span>
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
                        {#each sessions as [id, session] (`float-${viewModeKey}-${id}`)}
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
        <div class="docked-terminal">
            <!-- Header -->
            <div class="docked-header">
                <div class="docked-tabs">
                    {#each sessions as [id, session] (id)}
                        <button
                            class="docked-tab"
                            class:active={id === activeId}
                            class:dragging={draggingTabId === id &&
                                isDraggingTab}
                            on:click={() => setActive(id)}
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
                </div>

                <div class="docked-actions">
                    <button
                        class="btn btn-primary btn-sm"
                        on:click={openCreatePanel}
                    >
                        + New Terminal
                    </button>
                    <button
                        class="btn btn-secondary btn-sm"
                        on:click={toggleViewMode}
                    >
                        ⬔ Float
                    </button>
                    <button
                        class="btn btn-secondary btn-sm"
                        on:click={minimize}
                    >
                        − Minimize
                    </button>
                    <button class="btn btn-danger btn-sm" on:click={closeAll}>
                        × Close All
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
                                    on:click={closeCreatePanel}>× Cancel</button
                                >
                            {/if}
                        </div>

                        {#if isCreating}
                            <div class="create-progress">
                                <div class="progress-bar">
                                    <div
                                        class="progress-fill"
                                        style="width: {progress}%"
                                    ></div>
                                </div>
                                <div class="progress-info">
                                    <span class="progress-stage"
                                        >{progressStage}</span
                                    >
                                    <span class="progress-percent"
                                        >{progress}%</span
                                    >
                                </div>
                                <p class="progress-message">
                                    {progressMessage}
                                </p>
                                <div class="spinner"></div>
                            </div>
                        {:else}
                            <div class="create-panel-content docked-content">
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
                                                    (selectedRole = role.id)}
                                                title={role.desc}
                                            >
                                                <span class="role-icon"
                                                    >{role.icon}</span
                                                >
                                                <span class="role-name"
                                                    >{role.name}</span
                                                >
                                            </button>
                                        {/each}
                                    </div>
                                    <p class="role-desc">
                                        {roles.find(
                                            (r) => r.id === selectedRole,
                                        )?.desc}
                                    </p>
                                </div>

                                <!-- OS Selection -->
                                <div class="create-section">
                                    <h4>Operating System</h4>
                                    <div class="os-grid docked-grid">
                                        {#each images as image (image.name)}
                                            <button
                                                class="os-card"
                                                on:click={() =>
                                                    selectAndCreate(image.name)}
                                            >
                                                <span class="os-icon"
                                                    >{getIcon(image.name)}</span
                                                >
                                                <span class="os-name"
                                                    >{image.display_name ||
                                                        image.name}</span
                                                >
                                                {#if image.popular}
                                                    <span class="popular-badge"
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
                                            <span class="os-icon">📦</span>
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
                    {#each sessions as [id, session] (`dock-${viewModeKey}-${id}`)}
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

    /* Docked Terminal */
    .docked-terminal {
        position: fixed;
        top: 60px;
        left: 0;
        right: 0;
        bottom: 0;
        background: var(--bg-card);
        z-index: 998;
        display: flex;
        flex-direction: column;
    }

    .docked-header {
        display: flex;
        align-items: center;
        padding: 8px 16px;
        background: #111;
        border-bottom: 1px solid var(--border);
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
        background: #0a0a0a;
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
        color: var(--text);
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
        font-size: 12px;
        font-weight: 600;
        color: var(--text-muted);
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
        background: var(--bg-card);
        border: 1px solid var(--border);
        cursor: pointer;
        transition: all 0.15s;
        font-family: var(--font-mono);
    }

    .role-card:hover {
        border-color: var(--accent);
        background: var(--accent-dim);
    }

    .role-card.selected {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
        box-shadow: 0 0 10px rgba(0, 255, 65, 0.2);
    }

    .role-icon {
        font-size: 24px;
    }

    .role-name {
        font-size: 10px;
        color: var(--text);
        text-align: center;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 100%;
    }

    .role-desc {
        font-size: 11px;
        color: var(--text-muted);
        font-family: var(--font-mono);
        font-style: italic;
        margin: 4px 0 0 0;
        min-height: 16px;
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
        background: var(--bg-card);
        border: 1px solid var(--border);
        cursor: pointer;
        transition: all 0.15s;
        font-family: var(--font-mono);
        position: relative;
    }

    .os-card:hover {
        border-color: var(--accent);
        background: var(--accent-dim);
    }

    .os-card.selected {
        border-color: var(--accent);
        background: rgba(0, 217, 255, 0.1);
        box-shadow: 0 0 10px rgba(0, 217, 255, 0.2);
    }

    .os-icon {
        font-size: 24px;
    }

    .os-name {
        font-size: 10px;
        color: var(--text);
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
</style>
