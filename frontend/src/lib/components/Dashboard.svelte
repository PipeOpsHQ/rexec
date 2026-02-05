<script lang="ts">
    import { createEventDispatcher, tick } from "svelte";
    import { writable } from "svelte/store";
    import {
        containers,
        isCreating,
        creatingContainer,
        wsConnected,
        type Container,
        refreshContainers,
    } from "$stores/containers";
    import { agents } from "$stores/agents";
    import { auth } from "$stores/auth";
    import { terminal, connectedContainerIds } from "$stores/terminal";
    import { toast } from "$stores/toast";
    import {
        formatRelativeTime,
        formatMemory,
        formatStorage,
        formatCPU,
    } from "$utils/api";
    import ConfirmModal from "./ConfirmModal.svelte";
    import TerminalSettingsModal from "./TerminalSettingsModal.svelte";
    import PlatformIcon from "./icons/PlatformIcon.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    // MFA Lock state
    let showMfaModal = false;
    let mfaContainer: Container | null = null;
    let mfaCode = "";
    let mfaError = "";
    let mfaLoading = false;
    let mfaAction: "verify" | "unlock" = "verify"; // verify = access terminal, unlock = remove lock

    function openMfaVerifyModal(container: Container) {
        mfaContainer = container;
        mfaCode = "";
        mfaError = "";
        mfaAction = "verify";
        showMfaModal = true;
    }

    function closeMfaModal() {
        showMfaModal = false;
        mfaContainer = null;
        mfaCode = "";
        mfaError = "";
        mfaLoading = false;
    }

    async function handleMfaSubmit() {
        if (!mfaContainer || mfaCode.length !== 6) {
            mfaError = "Enter a valid 6-digit code";
            return;
        }

        mfaLoading = true;
        mfaError = "";

        try {
            const endpoint =
                mfaAction === "unlock"
                    ? `/api/security/terminal/${mfaContainer.db_id || mfaContainer.id}/mfa-unlock`
                    : `/api/security/terminal/${mfaContainer.db_id || mfaContainer.id}/mfa-verify`;

            const res = await fetch(endpoint, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${$auth.token}`,
                },
                body: JSON.stringify({ code: mfaCode }),
            });

            const data = await res.json();

            if (!res.ok) {
                mfaError = data.error || "Invalid code";
                mfaLoading = false;
                return;
            }

            if (mfaAction === "unlock") {
                toast.success("Terminal MFA lock removed");
                refreshContainers();
                closeMfaModal();
            } else {
                // Verified - now connect to terminal
                toast.success("MFA verified");
                const containerToConnect = mfaContainer;
                closeMfaModal();
                dispatch("connect", {
                    id: containerToConnect.id,
                    name: containerToConnect.name,
                });
            }
        } catch (err) {
            mfaError = "Failed to verify code";
        } finally {
            mfaLoading = false;
        }
    }

    let copiedCommand = ""; // To show 'Copied!' feedback

    // Get current host for install commands
    const currentHost =
        typeof window !== "undefined"
            ? window.location.host
            : "rexec.sh";
    const protocol =
        typeof window !== "undefined" ? window.location.protocol : "https:";
    const installUrl = `${protocol}//${currentHost}`;

    function copyToClipboard(text: string, id: string) {
        navigator.clipboard.writeText(text);
        copiedCommand = id;
        setTimeout(() => {
            copiedCommand = "";
        }, 2000);
    }

    const dispatch = createEventDispatcher<{
        create: void;
        connect: { id: string; name: string };
        showAgentDocs: void;
    }>();

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

    function handleSettingsUpdated(event: CustomEvent<Container>) {
        // Update the settings container with the new data
        settingsContainer = event.detail;
        // Also refresh the containers list
        refreshContainers();
    }

    // Reactive connected container IDs - direct subscription for proper reactivity
    $: connectedIds = $connectedContainerIds;

    // Track loading states for containers - use a reactive store pattern
    const loadingStatesStore = writable<
        Record<string, "starting" | "stopping" | "deleting" | null>
    >({});
    $: loadingStates = $loadingStatesStore;

    // Track the last known status to detect WebSocket updates
    let lastKnownStatus: Record<string, string> = {};

    function setLoading(
        id: string,
        state: "starting" | "stopping" | "deleting" | null,
    ) {
        loadingStatesStore.update((states) => {
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
    $: getLoadingState = (
        id: string,
    ): "starting" | "stopping" | "deleting" | null => {
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
        const currentIds = new Set($containers.containers.map((c) => c.id));
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
        setLoading(container.id, "starting");
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
        setLoading(container.id, "stopping");
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
        if (loadingId) setLoading(loadingId, "deleting");

        // Use tick to ensure DOM updates before API call
        await tick();

        const toastId = toast.loading(`Deleting ${container.name}...`);

        let result;
        if (container.session_type === "agent") {
            // Strip "agent:" prefix if present
            const agentId = container.id.replace(/^agent:/, "");
            const success = await agents.deleteAgent(agentId);
            result = { success };
            // Also remove from containers store UI immediately
            if (success) {
                containers.update((s) => ({
                    ...s,
                    containers: s.containers.filter(
                        (c) => c.id !== container.id,
                    ),
                }));
            }
        } else {
            result = await containers.deleteContainer(
                container.id,
                container.db_id,
            );
        }

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
        // Check if MFA locked first
        if (container.mfa_locked) {
            openMfaVerifyModal(container);
            return;
        }
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
            case "offline":
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
        const baseName = lower.split("/").pop()?.split(":")[0] || lower;
        // Also get the core name without version suffix (ubuntu-24 -> ubuntu)
        const coreName = baseName.split("-")[0];

        // Direct match on core name first
        const directMatches = [
            "ubuntu",
            "debian",
            "alpine",
            "fedora",
            "centos",
            "rocky",
            "alma",
            "arch",
            "archlinux",
            "kali",
            "manjaro",
            "mint",
            "gentoo",
            "void",
            "nixos",
            "slackware",
            "parrot",
            "blackarch",
            "oracle",
            "rhel",
            "devuan",
            "elementary",
        ];
        if (directMatches.includes(coreName)) {
            // Normalize archlinux to arch
            return coreName === "archlinux" ? "arch" : coreName;
        }
        if (directMatches.includes(baseName)) {
            return baseName === "archlinux" ? "arch" : baseName;
        }

        // Partial matches for special cases
        if (lower.includes("ubuntu")) return "ubuntu";
        if (lower.includes("debian")) return "debian";
        if (lower.includes("alpine")) return "alpine";
        if (lower.includes("fedora")) return "fedora";
        if (lower.includes("centos") || lower.includes("centos-stream"))
            return "centos";
        if (lower.includes("rocky")) return "rocky";
        if (lower.includes("alma")) return "alma";
        if (lower.includes("archlinux") || lower.includes("arch"))
            return "arch";
        if (lower.includes("kali")) return "kali";
        if (
            lower.includes("opensuse") ||
            lower.includes("suse") ||
            lower.includes("tumbleweed")
        )
            return "suse";
        if (
            lower.includes("rhel") ||
            lower.includes("redhat") ||
            lower.includes("ubi")
        )
            return "rhel";
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
    $: isPaidUser =
        $auth.user?.tier === "pro" ||
        $auth.user?.tier === "enterprise" ||
        $auth.user?.subscriptionActive;
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

<!-- MFA Verification Modal -->
{#if showMfaModal && mfaContainer}
    <div
        class="modal-overlay"
        onclick={(e) => e.target === e.currentTarget && closeMfaModal()}
    >
        <div class="mfa-modal">
            <div class="mfa-modal-header">
                <div class="mfa-modal-icon">
                    <svg
                        width="24"
                        height="24"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                    >
                        <rect
                            x="3"
                            y="11"
                            width="18"
                            height="11"
                            rx="2"
                            ry="2"
                        />
                        <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                    </svg>
                </div>
                <h3>
                    {mfaAction === "unlock"
                        ? "Remove MFA Lock"
                        : "MFA Protected Terminal"}
                </h3>
                <button class="mfa-close-btn" onclick={closeMfaModal}>
                    <svg
                        width="20"
                        height="20"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                    >
                        <path d="M18 6L6 18M6 6l12 12" />
                    </svg>
                </button>
            </div>
            <div class="mfa-modal-body">
                <p class="mfa-terminal-name">
                    Terminal: <strong>{mfaContainer.name}</strong>
                </p>
                <p class="mfa-description">
                    {#if mfaAction === "unlock"}
                        Enter your authenticator code to remove MFA protection
                        from this terminal.
                    {:else}
                        This terminal is protected with MFA. Enter your
                        authenticator code to access it.
                    {/if}
                </p>
                <div class="mfa-input-group">
                    <input
                        type="text"
                        class="mfa-input"
                        bind:value={mfaCode}
                        placeholder="000000"
                        maxlength="6"
                        autocomplete="one-time-code"
                        inputmode="numeric"
                        pattern="[0-9]*"
                        oninput={(e) => {
                            const target = e.target as HTMLInputElement;
                            target.value = target.value.replace(/\D/g, "");
                            mfaCode = target.value;
                            if (target.value.length === 6) {
                                handleMfaSubmit();
                            }
                        }}
                        onkeydown={(e) => {
                            if (e.key === "Enter" && mfaCode.length === 6) {
                                handleMfaSubmit();
                            }
                        }}
                        disabled={mfaLoading}
                    />
                </div>
                {#if mfaError}
                    <p class="mfa-error">{mfaError}</p>
                {/if}
            </div>
            <div class="mfa-modal-footer">
                <button
                    class="btn btn-secondary"
                    onclick={closeMfaModal}
                    disabled={mfaLoading}
                >
                    Cancel
                </button>
                <button
                    class="btn btn-primary"
                    onclick={handleMfaSubmit}
                    disabled={mfaLoading || mfaCode.length !== 6}
                >
                    {#if mfaLoading}
                        <span class="spinner-sm"></span>
                        Verifying...
                    {:else}
                        {mfaAction === "unlock" ? "Remove Lock" : "Verify"}
                    {/if}
                </button>
            </div>
        </div>
    </div>
{/if}

<TerminalSettingsModal
    bind:show={showSettingsModal}
    container={settingsContainer}
    {isPaidUser}
    on:close={closeSettings}
    on:updated={handleSettingsUpdated}
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
                class="btn btn-secondary btn-sm"
                onclick={() => dispatch("showAgentDocs")}
                title="Connect your own machine"
            >
                <svg
                    class="icon"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <rect x="2" y="3" width="20" height="14" rx="2" />
                    <path d="M8 21h8M12 17v4" />
                    <circle cx="12" cy="10" r="3" />
                </svg>
                <span class="btn-text-desktop">Connect Machine</span>
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
                Create your first terminal to access a cloud environment, GPU
                workspace, or connect to remote resources.
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

            <div class="connect-own-tip">
                <div class="tip-header">
                    <StatusIcon status="connected" size={16} />
                    <span>Or connect your own machine</span>
                </div>
                <p class="tip-text">
                    Use the rexec agent to connect your own server, VM, or local
                    machine.
                </p>
                <div class="tip-code-block">
                    <code class="tip-code"
                        >curl -fsSL {installUrl}/install-agent.sh | sudo bash</code
                    >
                    <button
                        class="copy-btn"
                        onclick={() =>
                            copyToClipboard(
                                `curl -fsSL ${installUrl}/install-agent.sh | sudo bash`,
                                "agent-install",
                            )}
                    >
                        {copiedCommand === "agent-install" ? "Copied!" : "Copy"}
                    </button>
                </div>
            </div>
        </div>
    {:else}
        <div class="containers-grid">
            {#if currentlyCreating && creatingInfo}
                <div class="container-card creating-card">
                    <div class="container-header">
                        <span class="container-icon creating-icon">
                            <PlatformIcon
                                platform={getDistro(creatingInfo.image || "")}
                                size={32}
                            />
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
                {@const isAgent = container.session_type === "agent"}
                <div
                    class="container-card"
                    class:agent-card={isAgent}
                    class:active={hasActiveSession(container.id)}
                    class:connected={containerConnected}
                    class:loading={isContainerLoading(container.id)}
                    class:deleting={getLoadingState(container.id) ===
                        "deleting"}
                    class:starting={getLoadingState(container.id) ===
                        "starting"}
                    class:stopping={getLoadingState(container.id) ===
                        "stopping"}
                >
                    {#if isContainerLoading(container.id)}
                        <div class="loading-overlay">
                            <div class="loading-content">
                                <div class="spinner"></div>
                                <span>
                                    {#if getLoadingState(container.id) === "deleting"}
                                        Deleting...
                                    {:else if getLoadingState(container.id) === "starting"}
                                        Starting...
                                    {:else if getLoadingState(container.id) === "stopping"}
                                        Stopping...
                                    {/if}
                                </span>
                            </div>
                        </div>
                    {/if}

                    <div class="container-header">
                        <span class="container-icon">
                            {#if isAgent}
                                <PlatformIcon
                                    platform={container.distro ||
                                        (container.os === "darwin"
                                            ? "macos"
                                            : container.os === "windows"
                                              ? "windows"
                                              : "linux")}
                                    size={32}
                                />
                            {:else}
                                <PlatformIcon
                                    platform={getDistro(container.image)}
                                    size={32}
                                />
                            {/if}
                        </span>
                        <div class="container-info">
                            <h2 class="container-name">{container.name}</h2>
                            <div class="container-meta-row">
                                <span class="container-image"
                                    >{container.image}</span
                                >
                                {#if isAgent}
                                    <span
                                        class="environment-badge agent-env"
                                        title="Connected via Agent"
                                    >
                                        <svg
                                            class="badge-icon"
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
                                            <path d="M8 21h8M12 17v4" />
                                        </svg>
                                        <span class="badge-text">Agent</span>
                                    </span>
                                {:else if container.role}
                                    <span
                                        class="environment-badge"
                                        title="Environment: {container.role}"
                                    >
                                        <PlatformIcon
                                            platform={container.role}
                                            size={14}
                                        />
                                        <span class="badge-text"
                                            >{container.role}</span
                                        >
                                    </span>
                                {/if}
                            </div>
                        </div>
                        <div class="container-badges">
                            <span
                                class="container-status {getStatusClass(
                                    container.status,
                                )}"
                            >
                                <span
                                    class="status-dot"
                                    class:status-dot-pulse={isAgent &&
                                        container.status === "running"}
                                ></span>
                                {container.status}
                            </span>
                            {#if container.mfa_locked}
                                <span
                                    class="mfa-lock-badge"
                                    title="MFA Protected - requires authenticator code to access"
                                >
                                    <svg
                                        width="14"
                                        height="14"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <rect
                                            x="3"
                                            y="11"
                                            width="18"
                                            height="11"
                                            rx="2"
                                            ry="2"
                                        />
                                        <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                                    </svg>
                                    <span>MFA</span>
                                </span>
                            {/if}
                        </div>
                    </div>

                    <div class="container-meta">
                        <div class="meta-item">
                            <span class="meta-label"
                                >{isAgent ? "Connected" : "Created"}</span
                            >
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
                        {#if isAgent && container.region}
                            <div class="meta-item meta-item-right">
                                <span class="meta-label">Region</span>
                                <span class="meta-value"
                                    >{container.region}</span
                                >
                            </div>
                        {/if}
                    </div>

                    {#if container.resources}
                        <div class="container-resources">
                            <span class="resource-spec" title="Memory">
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
                            <span class="resource-spec" title="CPU">
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
                            <span class="resource-spec" title="Storage">
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

                    {#if container.status === "running" && container.idle_seconds !== undefined}
                        <div class="activity-indicator">
                            <span class="activity-dot" class:active={container.idle_seconds < 60}></span>
                            <span class="activity-text">
                                {container.idle_seconds < 60
                                    ? "Active now"
                                    : container.idle_seconds < 300
                                      ? "Active recently"
                                      : `Idle ${Math.floor(container.idle_seconds / 60)}m`}
                            </span>
                        </div>
                    {/if}

                    <div class="container-actions">
                        {#if isAgent}
                            <div class="action-row">
                                {#if container.status === "running"}
                                    {#if !containerConnected && !isConnecting(container.id)}
                                        <button
                                            class="btn btn-primary btn-sm flex-1 agent-connect-btn"
                                            onclick={() =>
                                                handleConnect(container)}
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
                                {:else}
                                    <button
                                        class="btn btn-secondary btn-sm flex-1"
                                        disabled
                                    >
                                        Offline
                                    </button>
                                {/if}
                                <button
                                    class="btn btn-icon btn-sm"
                                    title="Settings"
                                    onclick={() => openSettings(container)}
                                >
                                    <svg
                                        class="icon"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <circle cx="12" cy="12" r="3" />
                                        <path
                                            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
                                        />
                                    </svg>
                                </button>
                                <button
                                    class="btn btn-icon btn-sm"
                                    title="Disconnect Agent"
                                    onclick={() => handleDelete(container)}
                                    disabled={getLoadingState(container.id) ===
                                        "deleting"}
                                    style="color: var(--red, #ff6b6b); border-color: rgba(255, 107, 107, 0.3);"
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
                                </button>
                                <button
                                    class="btn btn-icon btn-sm"
                                    title="Agent Details"
                                    onclick={() => dispatch("showAgentDocs")}
                                >
                                    <svg
                                        class="icon"
                                        viewBox="0 0 24 24"
                                        fill="none"
                                        stroke="currentColor"
                                        stroke-width="2"
                                    >
                                        <circle cx="12" cy="12" r="10" />
                                        <line x1="12" y1="16" x2="12" y2="12" />
                                        <line
                                            x1="12"
                                            y1="8"
                                            x2="12.01"
                                            y2="8"
                                        />
                                    </svg>
                                </button>
                            </div>
                        {:else if container.status === "running"}
                            <div class="action-row">
                                {#if !containerConnected && !isConnecting(container.id)}
                                    <button
                                        class="btn btn-primary btn-sm flex-1"
                                        onclick={() => handleConnect(container)}
                                        disabled={isContainerLoading(
                                            container.id,
                                        )}
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
                                        <path
                                            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"
                                        />
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
                                    disabled={getLoadingState(container.id) ===
                                        "deleting"}
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
                                    disabled={getLoadingState(container.id) ===
                                        "deleting"}
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
                                    disabled={getLoadingState(container.id) ===
                                        "deleting"}
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
        <div
            class="modal-content shortcuts-modal"
            onclick={(e) => e.stopPropagation()}
        >
            <div class="modal-header">
                <h3>Keyboard Shortcuts</h3>
                <button class="close-btn" onclick={toggleShortcuts}>×</button>
            </div>
            <div class="shortcuts-list">
                <div class="shortcut-group">
                    <h4>General</h4>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Alt</span> +
                            <span class="key">1-9</span>
                        </div>
                        <span class="desc">Switch tabs</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Alt</span> +
                            <span class="key">D</span>
                        </div>
                        <span class="desc">Toggle Dock/Float</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Alt</span> +
                            <span class="key">F</span>
                        </div>
                        <span class="desc">Fullscreen</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Alt</span> +
                            <span class="key">M</span>
                        </div>
                        <span class="desc">Minimize</span>
                    </div>
                </div>
                <div class="shortcut-group">
                    <h4>macOS Specific</h4>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Cmd</span> +
                            <span class="key">D</span>
                        </div>
                        <span class="desc">Split Vertical</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Cmd</span> +
                            <span class="key">Shift</span>
                            + <span class="key">D</span>
                        </div>
                        <span class="desc">Split Horizontal</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Cmd</span> +
                            <span class="key">T</span>
                        </div>
                        <span class="desc">New Tab</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Cmd</span> +
                            <span class="key">W</span>
                        </div>
                        <span class="desc">Close Pane</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Cmd</span> +
                            <span class="key">Arrows</span>
                        </div>
                        <span class="desc">Navigate Panes</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Cmd</span> +
                            <span class="key">C</span>
                            / <span class="key">V</span>
                        </div>
                        <span class="desc">Native Copy/Paste</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Cmd</span> +
                            <span class="key">K</span>
                        </div>
                        <span class="desc">Clear Screen (Ctrl+L)</span>
                    </div>
                    <div class="shortcut-item">
                        <div class="keys">
                            <span class="key">Cmd</span> +
                            <span class="key">.</span>
                        </div>
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
        position: relative;
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
        align-items: center;
        flex-wrap: wrap;
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
        border-radius: 12px;
        animation: fadeInUp 0.4s ease;
    }

    .empty-icon {
        width: 64px;
        height: 64px;
        margin-bottom: 16px;
        color: var(--text-muted);
        animation: float 3s ease-in-out infinite;
    }

    .empty-icon svg {
        width: 100%;
        height: 100%;
    }

    .empty-state h2 {
        font-size: 18px;
        margin-bottom: 8px;
        text-transform: uppercase;
        animation: fadeIn 0.6s ease 0.2s both;
    }

    .empty-state p {
        color: var(--text-secondary);
        max-width: 400px;
        margin-bottom: 24px;
        line-height: 1.5;
        animation: fadeIn 0.6s ease 0.3s both;
    }

    @keyframes float {
        0%, 100% {
            transform: translateY(0px);
        }
        50% {
            transform: translateY(-10px);
        }
    }

    @keyframes fadeInUp {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    /* Connect Own Tip in Empty State */
    .connect-own-tip {
        margin-top: 32px;
        padding: 18px 20px;
        background: var(--bg-card-secondary);
        border: 1px solid var(--border);
        border-left: 3px solid var(--accent);
        border-radius: 10px;
        max-width: 520px;
        width: 100%;
        text-align: left;
        box-shadow: var(--shadow-soft);
    }

    .tip-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;
    }

    .tip-header :global(svg) {
        color: var(--accent);
    }

    .tip-header span {
        font-size: 13px;
        font-weight: 600;
        color: var(--text);
    }

    .tip-text {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0 0 12px 0;
        line-height: 1.5;
    }

    .tip-code-block {
        display: flex;
        align-items: center;
        gap: 10px;
        background: var(--code-bg);
        border: 1px dashed var(--border);
        border-radius: 6px;
        padding: 10px 12px;
    }

    .tip-code {
        flex: 1;
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--accent);
        overflow-x: auto;
        white-space: nowrap;
    }

    .copy-btn {
        flex-shrink: 0;
        padding: 6px 10px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-secondary);
        font-size: 10px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .copy-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
        background: var(--accent-dim);
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
        0%,
        100% {
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
        grid-template-columns: repeat(auto-fill, minmax(min(320px, 100%), 1fr));
        gap: 16px;
    }

    .container-card {
        position: relative;
        background: var(--bg-card);
        border: 1px solid var(--border);
        padding: 16px;
        transition: all 0.2s ease;
        transform: translateY(0);
    }

    .container-card:hover {
        border-color: var(--text-muted);
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }

    .container-card.active {
        border-color: var(--accent);
        box-shadow: 0 0 10px var(--accent-dim);
    }

    .container-card.connected {
        border-color: #00d9ff;
        box-shadow: 0 0 8px rgba(0, 217, 255, 0.3);
    }

    /* Agent card styles - Match regular cards with subtle accent */
    .container-card.agent-card {
        border-color: var(--border);
        background: var(--bg-card);
        box-shadow: none;
    }

    .container-card.agent-card:hover {
        border-color: var(--text-muted);
        box-shadow: none;
        transform: none;
    }

    .container-card.agent-card.connected {
        border-color: #00d9ff;
        box-shadow: 0 0 8px rgba(0, 217, 255, 0.3);
    }

    .agent-env {
        background: rgba(34, 197, 94, 0.12);
        border-color: rgba(34, 197, 94, 0.25);
        color: #4ade80;
        text-shadow: none;
    }

    /* Agent connect button uses accent color */
    .agent-connect-btn {
        background: rgba(var(--accent-rgb), 0.1);
        border-color: rgba(var(--accent-rgb), 0.5);
        color: var(--accent);
    }

    .agent-connect-btn:hover:not(:disabled) {
        background: rgba(var(--accent-rgb), 0.9);
        border-color: var(--accent);
        color: var(--bg);
        box-shadow: 0 0 10px rgba(var(--accent-rgb), 0.3);
    }

    .container-card.agent-card .connecting-btn,
    .container-card.agent-card .connected-btn {
        border-color: rgba(var(--accent-rgb), 0.35);
        color: var(--accent);
    }

    .badge-icon {
        width: 12px;
        height: 12px;
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
        animation: glow-start 2s ease-in-out infinite;
    }

    @keyframes glow-start {
        0%, 100% {
            box-shadow: 0 0 0 0 rgba(0, 255, 65, 0.4);
        }
        50% {
            box-shadow: 0 0 12px 4px rgba(0, 255, 65, 0.2);
        }
    }

    .container-card.stopping {
        opacity: 0.8;
        border-color: #ffd93d;
        background: linear-gradient(
            135deg,
            rgba(255, 217, 61, 0.05) 0%,
            rgba(255, 217, 61, 0.1) 100%
        );
        animation: glow-stop 2s ease-in-out infinite;
    }

    @keyframes glow-stop {
        0%, 100% {
            box-shadow: 0 0 0 0 rgba(255, 217, 61, 0.4);
        }
        50% {
            box-shadow: 0 0 12px 4px rgba(255, 217, 61, 0.2);
        }
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
        animation: shake 0.5s ease;
    }

    @keyframes shake {
        0%, 100% {
            transform: translateX(0) scale(0.98);
        }
        25% {
            transform: translateX(-4px) scale(0.98);
        }
        75% {
            transform: translateX(4px) scale(0.98);
        }
    }

    .loading-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: var(--overlay-light);
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
        background: linear-gradient(
            90deg,
            transparent,
            var(--red, #ff6b6b),
            transparent
        );
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

    .container-badges {
        display: flex;
        flex-direction: column;
        align-items: flex-end;
        gap: 6px;
        flex-shrink: 0;
        margin-left: auto;
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

    @keyframes pulse {
        0%, 100% {
            opacity: 1;
            transform: scale(1);
        }
        50% {
            opacity: 0.7;
            transform: scale(1.05);
        }
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

    .container-meta-row {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .environment-badge {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 2px 8px;
        background: linear-gradient(
            135deg,
            rgba(0, 212, 255, 0.1),
            rgba(0, 212, 255, 0.05)
        );
        border: 1px solid rgba(0, 212, 255, 0.3);
        border-radius: 4px;
        font-size: 10px;
        font-weight: 500;
        color: var(--accent);
        text-transform: capitalize;
        letter-spacing: 0.3px;
    }

    .environment-badge .badge-text {
        max-width: 80px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
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
        border-radius: 0;
        white-space: nowrap;
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

    .status-dot-pulse {
        animation: pulse-green 2s infinite;
    }

    @keyframes pulse-green {
        0% {
            box-shadow: 0 0 0 0 rgba(0, 255, 65, 0.7);
        }
        70% {
            box-shadow: 0 0 0 6px rgba(0, 255, 65, 0);
        }
        100% {
            box-shadow: 0 0 0 0 rgba(0, 255, 65, 0);
        }
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
        animation: pulse-dot 1s infinite;
    }

    @keyframes pulse-dot {
        0%, 100% {
            opacity: 1;
            box-shadow: 0 0 0 0 rgba(255, 200, 0, 0.7);
        }
        50% {
            opacity: 0.8;
            box-shadow: 0 0 0 4px rgba(255, 200, 0, 0);
        }
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
        animation: slideIn 0.3s ease;
    }

    @keyframes slideIn {
        from {
            opacity: 0;
            transform: translateY(-10px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
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
        background: linear-gradient(90deg, var(--yellow), #ffd93d);
        transition: width 0.3s ease;
        position: relative;
        overflow: hidden;
    }

    .creating-progress .progress-fill::after {
        content: "";
        position: absolute;
        top: 0;
        left: 0;
        bottom: 0;
        right: 0;
        background: linear-gradient(
            90deg,
            transparent,
            rgba(255, 255, 255, 0.3),
            transparent
        );
        animation: shimmer 2s infinite;
    }

    @keyframes shimmer {
        0% {
            transform: translateX(-100%);
        }
        100% {
            transform: translateX(100%);
        }
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

    .meta-item-right {
        margin-left: auto;
        text-align: right;
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

    .resource-item {
        display: flex;
        flex-direction: column;
        gap: 2px;
        flex: 1;
    }

    .resource-label {
        font-size: 9px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-muted);
    }

    .resource-value {
        color: var(--accent);
        font-weight: 500;
    }

    .resource-usage {
        color: var(--text-muted);
        font-weight: 400;
        font-size: 10px;
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

    /* Activity Indicator */
    .activity-indicator {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-top: 8px;
        padding: 4px 8px;
        background: var(--bg-secondary);
        border: 1px solid var(--border-muted);
        border-radius: 4px;
        font-size: 11px;
        color: var(--text-muted);
    }

    .activity-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background: var(--text-muted);
        transition: all 0.3s ease;
    }

    .activity-dot.active {
        background: var(--green);
        box-shadow: 0 0 6px rgba(0, 255, 65, 0.5);
        animation: pulse-active 2s ease-in-out infinite;
    }

    @keyframes pulse-active {
        0%, 100% {
            opacity: 1;
            transform: scale(1);
        }
        50% {
            opacity: 0.7;
            transform: scale(1.2);
        }
    }

    .activity-text {
        font-family: var(--font-mono);
        font-size: 10px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
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
        .dashboard {
            padding: 0 8px;
        }

        .dashboard-header {
            flex-direction: column;
            align-items: stretch;
            gap: 12px;
            padding-bottom: 12px;
            margin-bottom: 16px;
        }

        .dashboard-title {
            justify-content: space-between;
            width: 100%;
        }

        .dashboard-title h1 {
            font-size: 16px;
        }

        .dashboard-actions {
            flex-wrap: wrap;
            gap: 6px;
            width: 100%;
        }

        .dashboard-actions .btn {
            flex: 1;
            min-width: calc(50% - 6px);
            justify-content: center;
            padding: 8px 10px;
            font-size: 12px;
        }

        .dashboard-actions .btn-primary {
            flex: 1 1 100%;
            order: -1;
        }

        .dashboard-actions .live-indicator {
            position: absolute;
            top: -20px;
            right: 0;
            font-size: 10px;
        }

        .containers-grid {
            grid-template-columns: 1fr;
            gap: 12px;
        }

        .btn-text-desktop {
            display: none;
        }

        .container-card {
            padding: 12px;
        }

        .container-header {
            flex-wrap: wrap;
            gap: 8px;
        }

        .container-icon {
            width: 36px;
            height: 36px;
            font-size: 18px;
        }

        .container-info {
            flex: 1;
            min-width: 0;
        }

        h2.container-name {
            font-size: 13px;
            word-break: break-word;
        }

        .container-image {
            font-size: 10px;
        }

        .container-meta-row {
            flex-wrap: wrap;
            gap: 4px;
        }

        .environment-badge {
            padding: 2px 6px;
            font-size: 9px;
        }

        .container-status {
            font-size: 10px;
            padding: 3px 8px;
        }

        .container-badges {
            flex-direction: row;
            gap: 4px;
        }

        .mfa-lock-badge {
            font-size: 9px;
            padding: 3px 6px;
        }

        .mfa-lock-badge svg {
            width: 10px;
            height: 10px;
        }

        .container-meta {
            flex-direction: column;
            gap: 4px;
            padding: 8px 0;
        }

        .meta-item {
            flex-direction: row;
            justify-content: space-between;
        }

        .meta-item-right {
            margin-left: 0;
        }

        .container-resources {
            flex-wrap: wrap;
            gap: 6px;
            padding: 8px;
            font-size: 10px;
        }

        .resource-spec {
            font-size: 10px;
        }

        .resource-spec svg {
            width: 12px;
            height: 12px;
        }

        .container-actions {
            gap: 6px;
        }

        .action-row {
            flex-wrap: wrap;
            gap: 6px;
        }

        .action-row .btn {
            flex: 1;
            min-width: calc(50% - 6px);
            padding: 8px 10px;
            font-size: 11px;
            justify-content: center;
        }

        .action-row .btn-primary,
        .action-row .btn.flex-1 {
            flex: 1 1 100%;
        }

        .empty-state {
            padding: 40px 16px;
        }

        .empty-state h2 {
            font-size: 16px;
        }

        .empty-state p {
            font-size: 13px;
        }

        .connect-own-tip {
            padding: 12px;
        }

        .tip-code-block {
            flex-direction: column;
            gap: 8px;
        }

        .tip-code {
            font-size: 11px;
            word-break: break-all;
        }

        .creating-progress {
            padding: 10px 0;
        }

        .progress-info {
            font-size: 11px;
        }

        .progress-message {
            font-size: 11px;
        }
    }

    @media (max-width: 480px) {
        .dashboard {
            padding: 0 4px;
        }

        .dashboard-header {
            gap: 10px;
            margin-bottom: 12px;
        }

        .dashboard-title h1 {
            font-size: 14px;
        }

        .count-badge {
            font-size: 10px;
            padding: 2px 6px;
        }

        .dashboard-actions .btn {
            padding: 6px 8px;
            font-size: 11px;
        }

        .dashboard-actions .icon {
            width: 12px;
            height: 12px;
        }

        .container-card {
            padding: 10px;
        }

        .container-icon {
            width: 32px;
            height: 32px;
            font-size: 16px;
        }

        h2.container-name {
            font-size: 12px;
        }

        .container-status {
            font-size: 9px;
            padding: 2px 6px;
        }

        .action-row .btn {
            padding: 6px 8px;
            font-size: 10px;
        }

        .spinner-sm {
            width: 12px;
            height: 12px;
        }
    }

    @media (max-width: 360px) {
        .dashboard {
            padding: 0 2px;
        }

        .dashboard-header {
            gap: 8px;
            margin-bottom: 10px;
        }

        .dashboard-title h1 {
            font-size: 13px;
        }

        .count-badge {
            font-size: 9px;
            padding: 1px 4px;
        }

        .dashboard-actions {
            gap: 4px;
        }

        .dashboard-actions .btn {
            padding: 5px 6px;
            font-size: 10px;
            min-width: calc(50% - 4px);
        }

        .dashboard-actions .icon {
            width: 10px;
            height: 10px;
        }

        .container-card {
            padding: 8px;
        }

        .container-header {
            gap: 6px;
        }

        .container-icon {
            width: 28px;
            height: 28px;
            font-size: 14px;
        }

        h2.container-name {
            font-size: 11px;
        }

        .container-image {
            font-size: 9px;
        }

        .container-status {
            font-size: 8px;
            padding: 2px 4px;
        }

        .container-meta {
            padding: 6px 0;
            gap: 3px;
        }

        .meta-label,
        .meta-value {
            font-size: 9px;
        }

        .container-resources {
            padding: 6px;
            font-size: 9px;
        }

        .resource-spec svg {
            width: 10px;
            height: 10px;
        }

        .action-row {
            gap: 4px;
        }

        .action-row .btn {
            padding: 5px 6px;
            font-size: 9px;
            min-width: calc(50% - 4px);
        }

        .empty-state {
            padding: 24px 10px;
        }

        .empty-icon svg {
            width: 36px;
            height: 36px;
        }

        .empty-state h2 {
            font-size: 14px;
        }

        .empty-state p {
            font-size: 11px;
        }

        .connect-own-tip {
            padding: 10px;
        }

        .tip-header span {
            font-size: 11px;
        }

        .tip-text {
            font-size: 10px;
        }

        .tip-code {
            font-size: 9px;
        }
    }

    /* Modal Styles */
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: var(--overlay-light);
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
        box-shadow: var(--shadow-soft);
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
        max-height: 90vh;
        overflow-y: auto;
    }

    .shortcuts-list {
        padding: 20px;
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 24px;
    }

    @media (max-width: 600px) {
        .modal-content {
            margin: 10px;
            max-width: calc(100% - 20px);
        }

        .shortcuts-modal {
            max-height: 80vh;
        }

        .shortcuts-list {
            grid-template-columns: 1fr;
            padding: 12px;
            gap: 16px;
        }

        .shortcut-group h4 {
            font-size: 11px;
            margin-bottom: 8px;
        }

        .shortcut-item {
            padding: 6px 0;
        }

        .keys {
            gap: 3px;
        }

        .key {
            font-size: 10px;
            padding: 2px 5px;
        }

        .desc {
            font-size: 11px;
        }

        .modal-header {
            padding: 12px 14px;
        }

        .modal-header h3 {
            font-size: 14px;
        }
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

    /* MFA Lock Badge - Match status badge styling */
    .mfa-lock-badge {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 2px 8px;
        background: rgba(251, 191, 36, 0.15);
        border: 1px solid rgba(251, 191, 36, 0.3);
        border-radius: 0;
        font-size: 11px;
        font-weight: 500;
        color: #fbbf24;
        text-transform: uppercase;
    }

    .mfa-lock-badge svg {
        width: 12px;
        height: 12px;
    }

    /* MFA Modal */
    .modal-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        animation: fadeIn 0.15s ease;
    }

    .mfa-modal {
        background: var(--bg-secondary, #111);
        border: 1px solid var(--border, #333);
        border-radius: 12px;
        width: 100%;
        max-width: 400px;
        margin: 16px;
        animation: slideUp 0.2s ease;
    }

    .mfa-modal-header {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 20px;
        border-bottom: 1px solid var(--border, #333);
    }

    .mfa-modal-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 40px;
        height: 40px;
        background: rgba(251, 191, 36, 0.15);
        border-radius: 8px;
        color: #fbbf24;
    }

    .mfa-modal-header h3 {
        flex: 1;
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--text, #e5e5e5);
    }

    .mfa-close-btn {
        background: none;
        border: none;
        padding: 4px;
        cursor: pointer;
        color: var(--text-muted, #888);
        border-radius: 4px;
        transition: all 0.15s ease;
    }

    .mfa-close-btn:hover {
        background: var(--bg-tertiary, #222);
        color: var(--text, #e5e5e5);
    }

    .mfa-modal-body {
        padding: 20px;
    }

    .mfa-terminal-name {
        margin: 0 0 12px 0;
        font-size: 13px;
        color: var(--text-secondary, #aaa);
    }

    .mfa-terminal-name strong {
        color: var(--text, #e5e5e5);
    }

    .mfa-description {
        margin: 0 0 20px 0;
        font-size: 14px;
        color: var(--text-secondary, #aaa);
        line-height: 1.5;
    }

    .mfa-input-group {
        margin-bottom: 12px;
    }

    .mfa-input {
        width: 100%;
        padding: 14px 16px;
        font-size: 24px;
        font-family: "JetBrains Mono", monospace;
        text-align: center;
        letter-spacing: 8px;
        background: var(--bg-tertiary, #1a1a1a);
        border: 1px solid var(--border, #333);
        border-radius: 8px;
        color: var(--text, #e5e5e5);
        outline: none;
        transition: border-color 0.15s ease;
    }

    .mfa-input:focus {
        border-color: #fbbf24;
    }

    .mfa-input::placeholder {
        letter-spacing: normal;
        font-size: 16px;
        color: var(--text-muted, #666);
    }

    .mfa-error {
        margin: 0;
        padding: 8px 12px;
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.3);
        border-radius: 6px;
        font-size: 13px;
        color: #ef4444;
    }

    .mfa-modal-footer {
        display: flex;
        gap: 12px;
        padding: 16px 20px;
        border-top: 1px solid var(--border, #333);
        justify-content: flex-end;
    }

    .mfa-modal-footer .btn {
        min-width: 100px;
    }

    @media (max-width: 480px) {
        .mfa-modal {
            margin: 10px;
            width: calc(100% - 20px);
        }

        .mfa-modal-header h3 {
            font-size: 16px;
        }

        .mfa-modal-icon {
            width: 48px;
            height: 48px;
        }

        .mfa-modal-icon svg {
            width: 24px;
            height: 24px;
        }

        .mfa-modal-body {
            padding: 16px;
        }

        .mfa-input {
            padding: 12px;
            font-size: 20px;
            letter-spacing: 6px;
        }

        .mfa-modal-footer {
            padding: 12px 16px;
            flex-direction: column;
            gap: 8px;
        }

        .mfa-modal-footer .btn {
            width: 100%;
            min-width: auto;
        }
    }

    @media (max-width: 360px) {
        .mfa-modal {
            margin: 6px;
            width: calc(100% - 12px);
        }

        .mfa-modal-header {
            padding: 12px;
        }

        .mfa-modal-header h3 {
            font-size: 14px;
        }

        .mfa-modal-icon {
            width: 40px;
            height: 40px;
        }

        .mfa-modal-icon svg {
            width: 20px;
            height: 20px;
        }

        .mfa-close-btn {
            width: 28px;
            height: 28px;
        }

        .mfa-modal-body {
            padding: 12px;
        }

        .mfa-terminal-name {
            font-size: 11px;
        }

        .mfa-description {
            font-size: 12px;
            margin-bottom: 14px;
        }

        .mfa-input {
            padding: 10px;
            font-size: 18px;
            letter-spacing: 4px;
        }

        .mfa-input::placeholder {
            font-size: 12px;
        }

        .mfa-error {
            font-size: 11px;
            padding: 6px 10px;
        }

        .mfa-modal-footer {
            padding: 10px 12px;
        }

        .mfa-modal-footer .btn {
            padding: 8px 12px;
            font-size: 12px;
        }
    }
</style>
