<script lang="ts">
    import { createEventDispatcher, tick } from "svelte";
    import { writable } from "svelte/store";
    import {
        containers,
        isCreating,
        creatingContainer,
        wsConnected,
        type Container,
    } from "$stores/containers";
    import { auth } from "$stores/auth";
    import { terminal, connectedContainerIds } from "$stores/terminal";
    import { toast } from "$stores/toast";
    import { formatRelativeTime, formatMemory, formatStorage, formatCPU } from "$utils/api";
    import ConfirmModal from "./ConfirmModal.svelte";
    import TerminalSettingsModal from "./TerminalSettingsModal.svelte";
    import PlatformIcon from "./icons/PlatformIcon.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    const dispatch = createEventDispatcher<{
        create: void;
        connect: { id: string; name: string };
    }>();

    // Session type icons and labels - using StatusIcon types
    const sessionTypeInfo: Record<string, { icon: string; label: string; color: string }> = {
        container: { icon: "terminal", label: "Terminal", color: "#00d4ff" },
        gpu: { icon: "gpu", label: "GPU Session", color: "#ff6b6b" },
        ssh: { icon: "connected", label: "SSH Target", color: "#ffd93d" },
        custom: { icon: "wrench", label: "Custom", color: "#a29bfe" },
    };

    function getSessionType(container: Container): string {
        // For now, all are containers. Future: detect from container metadata
        return container.session_type || "container";
    }

    // Track containers being deleted
    // (moved to loadingStates)

    // Confirm modal state
    let showDeleteConfirm = false;
    let containerToDelete: Container | null = null;

    // Settings modal state
    let showSettingsModal = false;
    let settingsContainer: Container | null = null;
    
    // Shortcuts modal state
    let showShortcutsModal = false;

    function toggleShortcuts() {
        showShortcutsModal = !showShortcutsModal;
    }

    function openSettings(container: Container) {
        settingsContainer = container;
        showSettingsModal = true;
    }

    function closeSettings() {
        showSettingsModal = false;
        settingsContainer = null;
    }

    // Reactive connected container IDs - direct subscription for proper reactivity
    $: connectedIds = $connectedContainerIds;

    // Check if container has active terminal connection (reactive version)
    function isConnected(containerId: string): boolean {
        return connectedIds.has(containerId);
    }

    // Track loading states for containers - use a reactive store pattern
    const loadingStatesStore = writable<Record<string, 'starting' | 'stopping' | 'deleting' | null>>({});
    $: loadingStates = $loadingStatesStore;
    
    // Track the last known status to detect WebSocket updates
    let lastKnownStatus: Record<string, string> = {};

    function setLoading(id: string, state: 'starting' | 'stopping' | 'deleting' | null) {
        loadingStatesStore.update(states => {
            const newStates = { ...states };
            if (state) {
                newStates[id] = state;
            } else {
                delete newStates[id];
            }
            return newStates;
        });
    }

    // These are now reactive getters that depend on the store subscription
    $: getLoadingState = (id: string): 'starting' | 'stopping' | 'deleting' | null => {
        return $loadingStatesStore[id] || null;
    };

    $: isContainerLoading = (id: string): boolean => {
        return !!$loadingStatesStore[id];
    };

    // Clear loading state when container status changes via WebSocket
    $: {
        for (const container of $containers.containers) {
            const prevStatus = lastKnownStatus[container.id];
            const currentStatus = container.status;
            
            // If status changed, clear any loading state
            if (prevStatus && prevStatus !== currentStatus) {
                if (loadingStates[container.id]) {
                    setLoading(container.id, null);
                }
            }
            lastKnownStatus[container.id] = currentStatus;
        }
        
        // Clean up deleted containers from lastKnownStatus
        const currentIds = new Set($containers.containers.map(c => c.id));
        for (const id of Object.keys(lastKnownStatus)) {
            if (!currentIds.has(id)) {
                delete lastKnownStatus[id];
                if (loadingStates[id]) {
                    setLoading(id, null);
                }
            }
        }
    }

    // Container actions
    async function handleStart(container: Container) {
        setLoading(container.id, 'starting');
        const toastId = toast.loading(`Starting ${container.name}...`);
        const result = await containers.startContainer(container.id);
        setLoading(container.id, null);

        if (result.success) {
            toast.update(toastId, `${container.name} started`, "success");
            if (result.recreated) {
                toast.info(
                    "Terminal was recreated. Your data volume was preserved.",
                );
            }
        } else {
            toast.update(
                toastId,
                result.error || "Failed to start terminal",
                "error",
            );
        }
    }

    async function handleStop(container: Container) {
        setLoading(container.id, 'stopping');
        const toastId = toast.loading(`Stopping ${container.name}...`);
        const result = await containers.stopContainer(container.id);
        setLoading(container.id, null);

        if (result.success) {
            toast.update(toastId, `${container.name} stopped`, "success");
        } else {
            toast.update(
                toastId,
                result.error || "Failed to stop terminal",
                "error",
            );
        }
    }

    function handleDelete(container: Container) {
        containerToDelete = container;
        showDeleteConfirm = true;
    }

    async function confirmDelete() {
        if (!containerToDelete) return;

        const container = containerToDelete;
        containerToDelete = null;
        showDeleteConfirm = false;

        // Set loading state immediately and force UI update
        const loadingId = container.id || container.db_id;
        if (loadingId) setLoading(loadingId, 'deleting');
        
        // Use tick to ensure DOM updates before API call
        await tick();
        
        const toastId = toast.loading(`Deleting ${container.name}...`);
        const result = await containers.deleteContainer(container.id, container.db_id);
        if (loadingId) setLoading(loadingId, null);

        if (result.success) {
            toast.update(toastId, `${container.name} deleted`, "success");
        } else {
            toast.update(
                toastId,
                result.error || "Failed to delete terminal",
                "error",
            );
        }
    }

    function cancelDelete() {
        containerToDelete = null;
    }

    // Track containers being connected
    let connectingIds: Set<string> = new Set();

    function handleConnect(container: Container) {
        // Mark as connecting immediately
        connectingIds.add(container.id);
        connectingIds = new Set(connectingIds); // Trigger reactivity
        dispatch("connect", { id: container.id, name: container.name });
    }

    // Reactively clear connecting state when container becomes connected
    $: {
        // When connectedIds changes, check if any connecting containers are now connected
        for (const id of connectingIds) {
            if (connectedIds.has(id)) {
                connectingIds.delete(id);
                connectingIds = new Set(connectingIds);
            }
        }
    }

    function isConnecting(containerId: string): boolean {
        // Not connecting if already connected
        if (connectedIds.has(containerId)) return false;
        return connectingIds.has(containerId);
    }

    function hasActiveSession(containerId: string): boolean {
        return terminal.hasActiveSession(containerId);
    }

    function getStatusClass(status: string): string {
        switch (status) {
            case "running":
                return "status-running";
            case "stopped":
                return "status-stopped";
            case "creating":
            case "starting":
            case "stopping":
                return "status-creating";
            case "error":
                return "status-error";
            default:
                return "status-unknown";
        }
    }

    // Distro detection for icon selection
    function getDistro(image: string): string {
        if (!image) return "linux";
        
        // Handle both full image names (rexec/ubuntu:latest) and simple names (ubuntu)
        const lower = image.toLowerCase();
        
        // Extract base name - handle formats like "rexec/ubuntu:latest", "ubuntu-24", "ubuntu"
        const baseName = lower.split('/').pop()?.split(':')[0] || lower;
        // Also get the core name without version suffix (ubuntu-24 -> ubuntu)
        const coreName = baseName.split('-')[0];
        
        // Direct match on core name first
        const directMatches = ['ubuntu', 'debian', 'alpine', 'fedora', 'centos', 'rocky', 'alma', 'arch', 'archlinux', 'kali', 'manjaro', 'mint', 'gentoo', 'void', 'nixos', 'slackware', 'parrot', 'blackarch', 'oracle', 'rhel', 'devuan', 'elementary'];
        if (directMatches.includes(coreName)) {
            // Normalize archlinux to arch
            return coreName === 'archlinux' ? 'arch' : coreName;
        }
        if (directMatches.includes(baseName)) {
            return baseName === 'archlinux' ? 'arch' : baseName;
        }
        
        // Partial matches for special cases
        if (lower.includes("ubuntu")) return "ubuntu";
        if (lower.includes("debian")) return "debian";
        if (lower.includes("alpine")) return "alpine";
        if (lower.includes("fedora")) return "fedora";
        if (lower.includes("centos") || lower.includes("centos-stream")) return "centos";
        if (lower.includes("rocky")) return "rocky";
        if (lower.includes("alma")) return "alma";
        if (lower.includes("archlinux") || lower.includes("arch")) return "arch";
        if (lower.includes("kali")) return "kali";
        if (lower.includes("opensuse") || lower.includes("suse") || lower.includes("tumbleweed")) return "suse";
        if (lower.includes("rhel") || lower.includes("redhat") || lower.includes("ubi")) return "rhel";
        if (lower.includes("manjaro")) return "manjaro";
        if (lower.includes("mint")) return "mint";
        if (lower.includes("gentoo")) return "gentoo";
        if (lower.includes("void")) return "void";
        if (lower.includes("nixos") || lower.includes("nix")) return "nixos";
        if (lower.includes("slackware")) return "slackware";
        if (lower.includes("parrot")) return "parrot";
        if (lower.includes("blackarch")) return "blackarch";
        if (lower.includes("oracle")) return "oracle";
        if (lower.includes("devuan")) return "devuan";
        if (lower.includes("elementary")) return "elementary";
        if (lower.includes("openeuler")) return "linux";
        
        return "linux";
    }

    // Reactive
    $: containerList = $containers.containers;
    $: isLoading = $containers.isLoading;
    $: containerLimit = $containers.limit;
    $: currentlyCreating = $isCreating;
    $: creatingInfo = $creatingContainer;
    // Count creating as part of limit
    $: effectiveCount = containerList.length + (currentlyCreating ? 1 : 0);
    
    // Check subscription status
    $: isPaidUser = $auth.user?.tier === 'pro' || $auth.user?.tier === 'enterprise' || $auth.user?.subscription_active;
</script>

<ConfirmModal
    bind:show={showDeleteConfirm}
    title="Delete Terminal"
    message={containerToDelete
        ? `Are you sure you want to delete "${containerToDelete.name}"? This action cannot be undone and all data will be lost.`
        : ""}
    confirmText="Delete"
    cancelText="Cancel"
    variant="danger"
    on:confirm={confirmDelete}
    on:cancel={cancelDelete}
/>

<TerminalSettingsModal
    bind:show={showSettingsModal}
    container={settingsContainer}
    isPaidUser={isPaidUser}
    on:close={closeSettings}
/>

<div class="dashboard">
    <div class="dashboard-header">
        <div class="dashboard-title">
            <h1>Terminals</h1>
            <span class="count-badge">
                {effectiveCount} / {containerLimit}
            </span>
        </div>
        <div class="dashboard-actions">
            {#if $wsConnected}
                <span class="live-indicator">
                    <span class="live-dot"></span>
                    Live
                </span>
            {/if}
            <button
                class="btn btn-secondary btn-sm"
                onclick={toggleShortcuts}
                title="Keyboard Shortcuts"
            >
                <svg
                    class="icon"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <rect x="3" y="5" width="18" height="14" rx="2" />
                    <line x1="7" y1="15" x2="17" y2="15" />
                    <line x1="7" y1="9" x2="7" y2="9.01" />
                    <line x1="11" y1="9" x2="11" y2="9.01" />
                    <line x1="15" y1="9" x2="15" y2="9.01" />
                </svg>
            </button>
            <button
                class="btn btn-secondary btn-sm"
                onclick={() => containers.fetchContainers()}
            >
                <svg
                    class="icon"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <path
                        d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"
                    />
                </svg>
                Refresh
            </button>
            <button
                class="btn btn-primary"
                onclick={() => dispatch("create")}
                disabled={effectiveCount >= containerLimit || currentlyCreating}
            >
                {#if currentlyCreating}
                    <span class="spinner-sm"></span>
                    Creating...
                {:else}
                    <svg
                        class="icon"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                    >
                        <line x1="12" y1="5" x2="12" y2="19" /><line
                            x1="5"
                            y1="12"
                            x2="19"
                            y2="12"
                        />
                    </svg>
                    New Terminal
                {/if}
            </button>
        </div>
    </div>

    {#if isLoading && containerList.length === 0}
        <div class="loading-state">
            <div class="spinner"></div>
            <p>Loading terminals...</p>
        </div>
    {:else if containerList.length === 0}
        <div class="empty-state">
            <div class="empty-icon">
                <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.5"
                >
                    <rect x="2" y="3" width="20" height="14" rx="2" />
                    <path d="M8 21h8M12 17v4M6 8l4 4-4 4M12 16h4" />
                </svg>
            </div>
            <h2>No Terminals Yet</h2>
            <p>
                Create your first terminal to access a cloud environment, GPU workspace, or connect to remote resources.
            </p>
            <button
                class="btn btn-primary btn-lg"
                onclick={() => dispatch("create")}
            >
                <svg
                    class="icon"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <line x1="12" y1="5" x2="12" y2="19" /><line
                        x1="5"
                        y1="12"
                        x2="19"
                        y2="12"
                    />
                </svg>
                Create Terminal
            </button>
        </div>
    {:else}
        <div class="containers-grid">
            {#if currentlyCreating && creatingInfo}
                <div class="container-card creating-card">
                    <div class="container-header">
                        <span class="container-icon creating-icon">
                            <PlatformIcon platform={getDistro(creatingInfo.image || '')} size={32} />
                        </span>
                        <div class="container-info">
                            <h2 class="container-name">
                                {creatingInfo.name || "New Terminal"}
                            </h2>
                            <span class="container-image"
                                >{creatingInfo.image}</span
                            >
                        </div>
                        <span class="container-status status-creating">
                            <span class="status-dot"></span>
                            creating
                        </span>
                    </div>

                    <div class="creating-progress">
                        <div class="progress-bar">
                            <div
                                class="progress-fill"
                                style="width: {Math.round(
                                    creatingInfo.progress,
                                )}%"
                            ></div>
                        </div>
                        <div class="progress-info">
                            <span class="progress-stage"
                                >{creatingInfo.stage}</span
                            >
                            <span class="progress-percent"
                                >{Math.round(creatingInfo.progress)}%</span
                            >
                        </div>
                        <p class="progress-message">{creatingInfo.message}</p>
                    </div>
                </div>
            {/if}
            {#each containerList as container (container.id)}
                {@const containerConnected = connectedIds.has(container.id)}
                <div
                    class="container-card"
                    class:active={hasActiveSession(container.id)}
                    class:connected={containerConnected}
                    class:loading={isContainerLoading(container.id)}
                    class:deleting={getLoadingState(container.id) === 'deleting'}
                    class:starting={getLoadingState(container.id) === 'starting'}
                    class:stopping={getLoadingState(container.id) === 'stopping'}
                >
                    {#if isContainerLoading(container.id)}
                        <div class="loading-overlay">
                            <div class="loading-content">
                                <div class="spinner"></div>
                                <span>
                                    {#if getLoadingState(container.id) === 'deleting'}
                                        Deleting...
                                    {:else if getLoadingState(container.id) === 'starting'}
                                        Starting...
                                    {:else if getLoadingState(container.id) === 'stopping'}
                                        Stopping...
                                    {/if}
                                </span>
                            </div>
                        </div>
                    {/if}

                    <div class="container-header">
                        <span class="container-icon">
                            <PlatformIcon platform={getDistro(container.image)} size={32} />
                        </span>
                        <div class="container-info">
                            <h2 class="container-name">{container.name}</h2>
                            <span class="container-image"
                                >{container.image}</span
                            >
                        </div>
                        <span
                            class="container-status {getStatusClass(
                                container.status,
                            )}"
                        >
                            <span class="status-dot"></span>
                            {container.status}
                        </span>
                    </div>

                    <div class="container-meta">
                        <div class="meta-item">
                            <span class="meta-label">Created</span>
                            <span class="meta-value"
                                >{formatRelativeTime(
                                    container.created_at,
                                )}</span
                            >
                        </div>
                        {#if container.idle_seconds !== undefined && container.status === "running"}
                            <div class="meta-item">
                                <span class="meta-label">Idle</span>
                                <span class="meta-value"
                                    >{Math.floor(
                                        container.idle_seconds / 60,
                                    )}m</span
                                >
                            </div>
                        {/if}
                    </div>

                    {#if container.resources}
                        <div class="container-resources">
                            <span class="resource-spec">
                                <svg
                                    class="resource-icon"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2"
                                >
                                    <rect
                                        x="4"
                                        y="4"
                                        width="16"
                                        height="16"
                                        rx="2"
                                    />
                                    <rect x="9" y="9" width="6" height="6" />
                                </svg>
                                {formatMemory(container.resources.memory_mb)}
                            </span>
                            <span class="resource-divider">/</span>
                            <span class="resource-spec">
                                <svg
                                    class="resource-icon"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2"
                                >
                                    <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
                                </svg>
                                                                {formatCPU(container.resources.cpu_shares)}
                            </span>
                            <span class="resource-divider">/</span>
                            <span class="resource-spec">
                                <svg
                                    class="resource-icon"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="2"
                                >
                                    <circle cx="12" cy="12" r="10" />
                                    <circle cx="12" cy="12" r="3" />
                                </svg>
                                {formatStorage(container.resources.disk_mb)}
                            </span>
                        </div>
                    {/if}

                    <div class="container-actions">
                        {#if container.status === "running"}
                            <div class="action-row">
                                {#if !containerConnected && !isConnecting(container.id)}
                                    <button
                                        class="btn btn-primary btn-sm flex-1"
                                        onclick={() =>
                                            handleConnect(container)}
                                        disabled={isContainerLoading(container.id)}
                                    >
                                        <svg
                                            class="icon"
                                            viewBox="0 0 24 24"
                                            fill="none"
                                            stroke="currentColor"
                                            stroke-width="2"
                                        >
                                            <rect
                                                x="2"
                                                y="3"
                                                width="20"
                                                height="14"
                                                rx="2"
                                            />
                                            <path d="M6 8l4 4-4 4" />
                                        </svg>
                                        Connect
                                    </button>
                                {:else if isConnecting(container.id)}
                                    <button
                                        class="btn btn-secondary btn-sm flex-1 connecting-btn"
                                        disabled
                                    >
                                        <span class="spinner-sm"></span>
                                        Connecting...
                                    </button>
                                {:else}
                                    <button
                                        class="btn btn-secondary btn-sm flex-1 connected-btn"
                                        disabled
                                    >
                                        <svg
                                            class="icon"
                                            viewBox="0 0 24 24"
                                            fill="none"
                                            stroke="currentColor"
                                            stroke-width="2"
                                        >
                                            <path d="M20 6L9 17l-5-5" />
                                        </svg>
                                        Connected
                                    </button>
                                {/if}
                                <button
                                    class="btn btn-icon btn-sm"
                                    title="Settings"
                                    onclick={() => openSettings(container)}
                                    disabled={isContainerLoading(container.id)}
                                >
                                    <svg
                                        class="icon"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <circle cx="12" cy="12" r="3" />
                                        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
                                    </svg>
                                </button>
                            </div>
                            <div class="action-row">
                                <button
                                    class="btn btn-secondary btn-sm flex-1"
                                    onclick={() => handleStop(container)}
                                    disabled={isContainerLoading(container.id)}
                                >
                                    <svg
                                        class="icon"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <rect
                                            x="6"
                                            y="6"
                                            width="12"
                                            height="12"
                                        />
                                    </svg>
                                    Stop
                                </button>
                                <button
                                    class="btn btn-danger btn-sm flex-1"
                                    onclick={() => handleDelete(container)}
                                    disabled={getLoadingState(container.id) === 'deleting'}
                                >
                                    <svg
                                        class="icon"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <path
                                            d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2"
                                        />
                                    </svg>
                                    Delete
                                </button>
                            </div>
                        {:else if container.status === "stopped"}
                            <div class="action-row">
                                <button
                                    class="btn btn-primary btn-sm flex-1"
                                    onclick={() => handleStart(container)}
                                    disabled={isContainerLoading(container.id)}
                                >
                                    <svg
                                        class="icon"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <polygon points="5,3 19,12 5,21" />
                                    </svg>
                                    Start
                                </button>
                                <button
                                    class="btn btn-danger btn-sm flex-1"
                                    onclick={() => handleDelete(container)}
                                    disabled={getLoadingState(container.id) === 'deleting'}
                                >
                                    <svg
                                        class="icon"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <path
                                            d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2"
                                        />
                                    </svg>
                                    Delete
                                </button>
                            </div>
                        {:else if container.status === "error"}
                            <div class="action-row">
                                <button
                                    class="btn btn-danger btn-sm flex-1"
                                    onclick={() => handleDelete(container)}
                                    disabled={getLoadingState(container.id) === 'deleting'}
                                >
                                    <svg
                                        class="icon"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <path
                                            d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2"
                                        />
                                    </svg>
                                    Delete
                                </button>
                            </div>
                        {:else}
                            <div class="action-row">
                                <button
                                    class="btn btn-secondary btn-sm flex-1"
                                    disabled
                                >
                                    <span class="spinner-sm"></span>
                                    {container.status}...
                                </button>
                            </div>
                        {/if}
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

{#if showShortcutsModal}
    <div class="modal-backdrop" onclick={toggleShortcuts}>
        <div class="modal-content shortcuts-modal" onclick={(e) => e.stopPropagation()}>
            <div class="modal-header">
                <h3>Keyboard Shortcuts</h3>
                <button class="close-btn" onclick={toggleShortcuts}>×</button>
            </div>
            <div class="shortcuts-list">
                <div class="shortcut-group">
                    <h4>General</h4>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Alt</span> + <span class="key">1-9</span></div>
                        <span class="desc">Switch tabs</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Alt</span> + <span class="key">D</span></div>
                        <span class="desc">Toggle Dock/Float</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Alt</span> + <span class="key">F</span></div>
                        <span class="desc">Fullscreen</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Alt</span> + <span class="key">M</span></div>
                        <span class="desc">Minimize</span>
                    </div>
                </div>
                <div class="shortcut-group">
                    <h4>macOS Specific</h4>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Cmd</span> + <span class="key">D</span></div>
                        <span class="desc">Split Vertical</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Cmd</span> + <span class="key">Shift</span> + <span class="key">D</span></div>
                        <span class="desc">Split Horizontal</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Cmd</span> + <span class="key">T</span></div>
                        <span class="desc">New Tab</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Cmd</span> + <span class="key">W</span></div>
                        <span class="desc">Close Pane</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Cmd</span> + <span class="key">Arrows</span></div>
                        <span class="desc">Navigate Panes</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Cmd</span> + <span class="key">C</span> / <span class="key">V</span></div>
                        <span class="desc">Native Copy/Paste</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Cmd</span> + <span class="key">K</span></div>
                        <span class="desc">Clear Screen (Ctrl+L)</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys"><span class="key">Cmd</span> + <span class="key">.</span></div>
                        <span class="desc">Send Ctrl+C</span>
                    </div>
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    .dashboard {
        animation: fadeIn 0.2s ease;
    }

    .dashboard-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 24px;
        padding-bottom: 16px;
        border-bottom: 1px solid var(--border);
    }

    .dashboard-title {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .dashboard-title h1 {
        font-size: 20px;
        text-transform: uppercase;
        letter-spacing: 1px;
        margin: 0;
    }

    .count-badge {
        padding: 2px 10px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        font-size: 12px;
        color: var(--accent);
    }

    .dashboard-actions {
        display: flex;
        gap: 8px;
    }

    .icon {
        width: 14px;
        height: 14px;
        flex-shrink: 0;
    }

    /* Loading State */
    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 60px 20px;
        gap: 16px;
    }

    .loading-state p {
        color: var(--text-muted);
    }

    .spinner {
        width: 24px;
        height: 24px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    /* Empty State */
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 60px 20px;
        text-align: center;
        border: 1px dashed var(--border);
        background: var(--bg-card);
    }

    .empty-icon {
        width: 64px;
        height: 64px;
        margin-bottom: 16px;
        color: var(--text-muted);
    }

    .empty-icon svg {
        width: 100%;
        height: 100%;
    }

    .empty-state h2 {
        font-size: 18px;
        margin-bottom: 8px;
        text-transform: uppercase;
    }

    .empty-state p {
        color: var(--text-secondary);
        max-width: 400px;
        margin-bottom: 24px;
        line-height: 1.5;
    }

    /* Live indicator */
    .live-indicator {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.3);
        border-radius: 4px;
        font-size: 11px;
        font-family: var(--font-mono);
        color: var(--accent);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .live-dot {
        width: 8px;
        height: 8px;
        background: var(--accent);
        border-radius: 50%;
        animation: pulse-live 1.5s ease-in-out infinite;
    }

    @keyframes pulse-live {
        0%, 100% { 
            opacity: 1; 
            box-shadow: 0 0 0 0 rgba(0, 255, 65, 0.4);
        }
        50% { 
            opacity: 0.6; 
            box-shadow: 0 0 0 4px rgba(0, 255, 65, 0);
        }
    }

    /* Containers Grid */
    .containers-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 16px;
    }

    .container-card {
        position: relative;
        background: var(--bg-card);
        border: 1px solid var(--border);
        padding: 16px;
        transition: all 0.2s;
    }

    .container-card:hover {
        border-color: var(--text-muted);
    }

    .container-card.active {
        border-color: var(--accent);
        box-shadow: 0 0 10px var(--accent-dim);
    }

    .container-card.connected {
        border-color: #00d9ff;
        box-shadow: 0 0 8px rgba(0, 217, 255, 0.3);
    }



    /* Loading states */
    .container-card.loading {
        position: relative;
        pointer-events: none;
    }

    .container-card.starting {
        opacity: 0.8;
        border-color: var(--accent);
        background: linear-gradient(
            135deg,
            rgba(0, 255, 65, 0.05) 0%,
            rgba(0, 255, 65, 0.1) 100%
        );
    }

    .container-card.stopping {
        opacity: 0.8;
        border-color: #ffd93d;
        background: linear-gradient(
            135deg,
            rgba(255, 217, 61, 0.05) 0%,
            rgba(255, 217, 61, 0.1) 100%
        );
    }

    .container-card.deleting {
        opacity: 0.6;
        border-color: var(--red, #ff6b6b);
        background: linear-gradient(
            135deg,
            rgba(255, 107, 107, 0.05) 0%,
            rgba(255, 0, 60, 0.1) 100%
        );
        transform: scale(0.98);
        transition: all 0.3s ease;
    }

    .loading-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10;
        border-radius: inherit;
    }

    .loading-content {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .container-card.deleting::after {
        content: "";
        position: absolute;
        top: 50%;
        left: 0;
        right: 0;
        height: 2px;
        background: linear-gradient(90deg, transparent, var(--red, #ff6b6b), transparent);
        animation: strikethrough 0.5s ease forwards;
    }

    @keyframes strikethrough {
        from {
            transform: scaleX(0);
        }
        to {
            transform: scaleX(1);
        }
    }

    .container-header {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        margin-bottom: 12px;
    }

    .container-icon {
        width: 32px;
        height: 32px;
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .creating-icon {
        animation: pulse 1s infinite;
    }

    .container-info {
        flex: 1;
        min-width: 0;
    }

    h2.container-name {
        font-size: 14px;
        font-weight: 600;
        margin: 0 0 4px;
        color: var(--text);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .container-image {
        font-size: 11px;
        color: #999999;
    }

    .container-status {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 11px;
        text-transform: uppercase;
        padding: 2px 8px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
    }

    .status-dot {
        width: 6px;
        height: 6px;
    }

    .status-running {
        border-color: var(--green);
        color: var(--green);
    }

    .status-running .status-dot {
        background: var(--green);
    }

    .status-stopped {
        border-color: var(--text-muted);
        color: var(--text-muted);
    }

    .status-stopped .status-dot {
        background: var(--text-muted);
    }

    .status-creating {
        border-color: var(--yellow);
        color: var(--yellow);
    }

    .status-creating .status-dot {
        background: var(--yellow);
        animation: pulse 1s infinite;
    }

    .status-error {
        border-color: var(--red, #ff6b6b);
        color: var(--red, #ff6b6b);
    }

    .status-error .status-dot {
        background: var(--red, #ff6b6b);
    }

    /* Creating Card */
    .creating-card {
        border-color: var(--yellow);
        background: rgba(255, 200, 0, 0.05);
    }

    .creating-progress {
        padding: 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
    }

    .creating-progress .progress-bar {
        height: 4px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        margin-bottom: 8px;
        overflow: hidden;
    }

    .creating-progress .progress-fill {
        height: 100%;
        background: var(--yellow);
        transition: width 0.3s ease;
    }

    .creating-progress .progress-info {
        display: flex;
        justify-content: space-between;
        margin-bottom: 4px;
    }

    .creating-progress .progress-stage {
        font-size: 11px;
        text-transform: uppercase;
        color: var(--yellow);
    }

    .creating-progress .progress-percent {
        font-size: 11px;
        color: var(--text-muted);
    }

    .creating-progress .progress-message {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    /* Connecting button style */
    .connecting-btn {
        background: rgba(255, 217, 61, 0.1) !important;
        border-color: rgba(255, 217, 61, 0.3) !important;
        color: #ffd93d !important;
        cursor: wait !important;
    }

    .connecting-btn .spinner-sm {
        border-color: rgba(255, 217, 61, 0.3);
        border-top-color: #ffd93d;
    }

    /* Connected button style */
    .connected-btn {
        background: rgba(0, 217, 255, 0.1) !important;
        border-color: rgba(0, 217, 255, 0.3) !important;
        color: #00d9ff !important;
        cursor: default !important;
    }

    .connected-btn .icon {
        color: #00d9ff;
    }

    /* Icon-only button */
    .btn-icon {
        padding: 6px 8px !important;
        min-width: auto !important;
    }

    .container-meta {
        display: flex;
        gap: 16px;
        margin-bottom: 8px;
        padding: 8px 10px;
        background: var(--bg-secondary);
        border: 1px solid var(--border-muted);
    }

    .meta-item {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .meta-label {
        font-size: 10px;
        color: #999999;
        text-transform: uppercase;
    }

    .meta-value {
        font-size: 12px;
        color: var(--text-secondary);
    }

    .meta-value.mono {
        font-family: var(--font-mono);
    }

    /* Compact terminal-style resource display */
    .container-resources {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 10px;
        padding: 6px 10px;
        background: var(--bg-secondary);
        border: 1px solid var(--border-muted);
        font-family: var(--font-mono);
        font-size: 11px;
    }

    .resource-spec {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        color: var(--accent);
    }

    .resource-icon {
        width: 12px;
        height: 12px;
        color: var(--text-muted);
    }

    .resource-divider {
        color: var(--text-muted);
    }

    .container-actions {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .action-row {
        display: flex;
        gap: 8px;
    }

    .flex-1 {
        flex: 1;
    }

    .spinner-sm {
        width: 12px;
        height: 12px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        display: inline-block;
        margin-right: 6px;
    }

    @keyframes fadeIn {
        from {
            opacity: 0;
        }
        to {
            opacity: 1;
        }
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

    @media (max-width: 768px) {
        .dashboard-header {
            flex-direction: column;
            align-items: flex-start;
            gap: 12px;
        }

        .containers-grid {
            grid-template-columns: 1fr;
        }
    }

    /* Modal Styles */
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 2000;
        animation: fadeIn 0.2s ease;
    }

    .modal-content {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        width: 100%;
        max-width: 500px;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
        display: flex;
        flex-direction: column;
        animation: slideUp 0.3s ease;
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
    }

    .modal-header h3 {
        margin: 0;
        font-size: 16px;
        font-weight: 600;
    }

    .close-btn {
        background: none;
        border: none;
        color: var(--text-muted);
        font-size: 24px;
        cursor: pointer;
        padding: 0;
        line-height: 1;
    }

    .close-btn:hover {
        color: var(--text);
    }

    .shortcuts-modal {
        max-width: 600px;
    }

    .shortcuts-list {
        padding: 20px;
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 24px;
    }

    .shortcut-group h4 {
        margin: 0 0 12px 0;
        font-size: 12px;
        text-transform: uppercase;
        color: var(--text-muted);
        border-bottom: 1px solid var(--border);
        padding-bottom: 8px;
    }

    .shortcut-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
        font-size: 13px;
    }

    .keys {
        display: flex;
        align-items: center;
        gap: 4px;
        color: var(--text-muted);
        font-size: 12px;
    }

    .key {
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 4px;
        padding: 2px 6px;
        font-family: var(--font-mono);
        color: var(--text);
        min-width: 24px;
        text-align: center;
    }

    .desc {
        color: var(--text-secondary);
    }

    @keyframes slideUp {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
</style>
