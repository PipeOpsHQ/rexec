<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { fade, scale } from "svelte/transition";
    import { api, formatMemory, formatStorage, formatCPU } from "$utils/api";
    import { toast } from "$stores/toast";
    import { userTier, subscriptionActive } from "$stores/auth";
    import {
        containers,
        type Container,
        type PortForward,
    } from "$stores/containers";
    import { terminal } from "$stores/terminal";
    import ConfirmModal from "./ConfirmModal.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    export let show: boolean = false;
    export let container: Container | null = null;
    export let isPaidUser: boolean = false;

    const dispatch = createEventDispatcher<{
        close: void;
        updated: Container;
    }>();

    let activeTab: "settings" | "port-forwards" = "settings";
    let name = "";
    let memoryMB = 512;
    let cpuShares = 512;
    let diskMB = 2048;
    let isSaving = false;
    let initialized = false;

    // Port Forwarding state
    let portForwards: PortForward[] = [];
    let isLoadingForwards = false;
    let showAddForwardModal = false;
    let newForwardName = "";
    let newContainerPort: number | string = "";
    let isAddingForward = false;

    let showDeleteConfirm = false;
    let forwardToDelete: { id: string; name: string } | null = null;

    // Dynamic resource limits based on plan tier
    $: resourceLimits = (() => {
        if ($subscriptionActive) {
            return {
                minMemory: 256,
                maxMemory: 4096,
                minCPU: 250,
                maxCPU: 4000,
                minDisk: 1024,
                maxDisk: 20480,
            };
        }
        switch ($userTier) {
            case "guest":
                return {
                    minMemory: 256,
                    maxMemory: 512,
                    minCPU: 250,
                    maxCPU: 500,
                    minDisk: 1024,
                    maxDisk: 2048,
                };
            case "free":
                return {
                    minMemory: 256,
                    maxMemory: 2048,
                    minCPU: 250,
                    maxCPU: 2000,
                    minDisk: 1024,
                    maxDisk: 10240,
                };
            case "pro":
                return {
                    minMemory: 256,
                    maxMemory: 4096,
                    minCPU: 250,
                    maxCPU: 4000,
                    minDisk: 1024,
                    maxDisk: 20480,
                };
            case "enterprise":
                return {
                    minMemory: 256,
                    maxMemory: 8192,
                    minCPU: 250,
                    maxCPU: 8000,
                    minDisk: 1024,
                    maxDisk: 51200,
                };
            default: // Free fallback
                return {
                    minMemory: 256,
                    maxMemory: 2048,
                    minCPU: 250,
                    maxCPU: 2000,
                    minDisk: 1024,
                    maxDisk: 10240,
                };
        }
    })();

    // Initialize form values when modal opens
    function initializeValues() {
        if (!container) return;

        name = container.name || "";

        // Get raw values from container resources
        const rawMemory = container.resources?.memory_mb ?? 512;
        const rawCpu = container.resources?.cpu_shares ?? 512;
        const rawDisk = container.resources?.disk_mb ?? 2048;

        // Clamp values to be within current plan limits
        memoryMB = Math.max(
            resourceLimits.minMemory,
            Math.min(rawMemory, resourceLimits.maxMemory),
        );
        cpuShares = Math.max(
            resourceLimits.minCPU,
            Math.min(rawCpu, resourceLimits.maxCPU),
        );
        diskMB = Math.max(
            resourceLimits.minDisk,
            Math.min(rawDisk, resourceLimits.maxDisk),
        );
    }

    // React to modal opening
    $: if (show && container && !initialized) {
        initializeValues();
        initialized = true;
    }

    // Reset initialized flag when modal closes
    $: if (!show) {
        initialized = false;
    }

    // Load port forwards when tab changes or modal opens
    $: if (show && container && activeTab === "port-forwards") {
        loadPortForwards();
    }

    async function loadPortForwards() {
        const containerId = container?.db_id || container?.id;
        if (!containerId) return;
        isLoadingForwards = true;
        const { data, error } = await api.get<{ forwards: PortForward[] }>(
            `/api/containers/${containerId}/port-forwards`,
        );

        if (data) {
            portForwards = data.forwards || [];
        } else if (error) {
            toast.error(error || "Failed to load port forwards");
        }
        isLoadingForwards = false;
    }

    function openAddForwardModal() {
        newForwardName = "";
        newContainerPort = "";
        showAddForwardModal = true;
    }

    function closeAddForwardModal() {
        showAddForwardModal = false;
        newForwardName = "";
        newContainerPort = "";
    }

    async function addPortForward() {
        const containerId = container?.db_id || container?.id;
        if (!containerId) return;
        if (!newContainerPort) {
            toast.error("Please specify the container port");
            return;
        }

        const containerPortNum = parseInt(newContainerPort.toString(), 10);

        if (
            isNaN(containerPortNum) ||
            containerPortNum <= 0 ||
            containerPortNum > 65535
        ) {
            toast.error("Invalid port number. Must be between 1 and 65535.");
            return;
        }

        isAddingForward = true;
        const { data, error } = await api.post<PortForward>(
            `/api/containers/${containerId}/port-forwards`,
            {
                name: newForwardName.trim(),
                container_id: containerId,
                container_port: containerPortNum,
                local_port: containerPortNum, // Auto-set (not used in path-based forwarding)
            },
        );

        if (data) {
            portForwards = [...portForwards, data];
            toast.success("Port forward added");
            closeAddForwardModal();
            // TODO: Start local proxy for this forward
        } else {
            toast.error(error || "Failed to add port forward");
        }
        isAddingForward = false;
    }

    function deletePortForward(id: string, name: string) {
        forwardToDelete = { id, name };
        showDeleteConfirm = true;
    }

    async function confirmDeleteForward() {
        const containerId = container?.db_id || container?.id;
        if (!containerId || !forwardToDelete) return;

        const { id } = forwardToDelete;
        forwardToDelete = null;

        const { error } = await api.delete(
            `/api/containers/${containerId}/port-forwards/${id}`,
        );

        if (!error) {
            portForwards = portForwards.filter((pf) => pf.id !== id);
            toast.success("Port forward stopped");
            // TODO: Stop local proxy for this forward
        } else {
            toast.error(error || "Failed to stop port forward");
        }
    }

    function cancelDeleteForward() {
        forwardToDelete = null;
    }

    function handleClose() {
        dispatch("close");
        show = false;
    }

    async function handleSave() {
        if (!container) return;

        // Use db_id if available, fallback to id (docker_id)
        const containerId = container.db_id || container.id;
        if (!containerId) {
            toast.error("Terminal ID not found");
            return;
        }

        isSaving = true;
        try {
            const response = await api.patch(
                `/api/containers/${containerId}/settings`,
                {
                    name: name.trim(),
                    memory_mb: memoryMB,
                    cpu_shares: cpuShares,
                    disk_mb: diskMB,
                },
            );

            if (response.ok) {
                const responseData = response.data as {
                    container: Container;
                    restarted?: boolean;
                };

                // If container was restarted, trigger immediate reconnect
                if (responseData.restarted && responseData.container) {
                    const oldContainerId = container.id;
                    const newContainerId = responseData.container.id;

                    // Immediate reconnect - container is already running with new ID
                    if (oldContainerId && newContainerId) {
                        toast.success(
                            "Settings updated - reconnecting to new container...",
                        );
                        // Always use updateSessionContainerId - it handles both same and different IDs
                        // and properly finds sessions by container ID (not session ID)
                        terminal.updateSessionContainerId(
                            oldContainerId,
                            newContainerId,
                        );
                    }
                } else {
                    toast.success("Terminal settings updated");
                }

                if (responseData.container) {
                    dispatch("updated", responseData.container);
                }
                handleClose();
                // Clear any stuck creating state and refresh containers
                containers.clearCreating();
                containers.fetchContainers();
            } else {
                toast.error(response.error || "Failed to update settings");
            }
        } catch (err) {
            console.error("Settings update error:", err);
            toast.error("Failed to update settings");
        } finally {
            isSaving = false;
        }
    }

    function handleKeydown(e: KeyboardEvent) {
        if (!show) return;
        if (e.key === "Escape") {
            handleClose();
        }
    }
</script>

<svelte:window onkeydown={handleKeydown} />

<ConfirmModal
    bind:show={showDeleteConfirm}
    title="Stop Port Forwarding"
    message={forwardToDelete
        ? `Are you sure you want to stop forwarding port ${forwardToDelete.name}?`
        : ""}
    confirmText="Stop"
    cancelText="Cancel"
    variant="danger"
    on:confirm={confirmDeleteForward}
    on:cancel={cancelDeleteForward}
/>

{#if show && container}
    <div class="modal-backdrop" transition:fade={{ duration: 150 }}>
        <div
            class="modal-container"
            transition:scale={{ duration: 150, start: 0.95 }}
        >
            <div class="modal-header">
                <div class="modal-icon">
                    <svg
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
                </div>
                <h2 class="modal-title">Terminal Settings</h2>
                <button class="close-btn" onclick={handleClose}>
                    <svg
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                    >
                        <path d="M18 6L6 18M6 6l12 12" />
                    </svg>
                </button>
            </div>

            <div class="tabs">
                <button
                    class="tab-btn"
                    class:active={activeTab === "settings"}
                    onclick={() => (activeTab = "settings")}
                >
                    General
                </button>
                <button
                    class="tab-btn"
                    class:active={activeTab === "port-forwards"}
                    onclick={() => (activeTab = "port-forwards")}
                >
                    Port Forwarding
                </button>
            </div>

            <div class="modal-body">
                {#if activeTab === "settings"}
                    <div class="form-group">
                        <label for="terminal-name">Terminal Name</label>
                        <input
                            id="terminal-name"
                            type="text"
                            bind:value={name}
                            placeholder="my-terminal"
                            class="input"
                        />
                    </div>

                    <div class="section-header">
                        <span class="section-title">Resources</span>
                        {#if !isPaidUser}
                            <span class="trial-badge">Trial Limits</span>
                        {/if}
                    </div>

                    <div class="form-group">
                        <label for="memory">
                            Memory
                            <span class="value-display"
                                >{formatMemory(memoryMB)}</span
                            >
                        </label>
                        <input
                            id="memory"
                            type="range"
                            bind:value={memoryMB}
                            min={resourceLimits.minMemory}
                            max={resourceLimits.maxMemory}
                            step="128"
                            class="slider"
                        />
                        <div class="range-labels">
                            <span>{formatMemory(resourceLimits.minMemory)}</span
                            >
                            <span>{formatMemory(resourceLimits.maxMemory)}</span
                            >
                        </div>
                    </div>

                    <div class="form-group">
                        <label for="cpu">
                            CPU
                            <span class="value-display"
                                >{formatCPU(cpuShares)}</span
                            >
                        </label>
                        <input
                            id="cpu"
                            type="range"
                            bind:value={cpuShares}
                            min={resourceLimits.minCPU}
                            max={resourceLimits.maxCPU}
                            step="128"
                            class="slider"
                        />
                        <div class="range-labels">
                            <span>{formatCPU(resourceLimits.minCPU)}</span>
                            <span>{formatCPU(resourceLimits.maxCPU)}</span>
                        </div>
                    </div>

                    <div class="form-group">
                        <label for="disk">
                            Disk
                            <span class="value-display"
                                >{formatStorage(diskMB)}</span
                            >
                        </label>
                        <input
                            id="disk"
                            type="range"
                            bind:value={diskMB}
                            min={resourceLimits.minDisk}
                            max={resourceLimits.maxDisk}
                            step="512"
                            class="slider"
                        />
                        <div class="range-labels">
                            <span>{formatStorage(resourceLimits.minDisk)}</span>
                            <span>{formatStorage(resourceLimits.maxDisk)}</span>
                        </div>
                    </div>

                    <p class="note">
                        <svg
                            class="note-icon"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                        >
                            <circle cx="12" cy="12" r="10" />
                            <path d="M12 16v-4M12 8h.01" />
                        </svg>
                        Resource changes require terminal restart to take effect.
                    </p>

                    {#if !isPaidUser && $userTier === "free"}
                        <p class="upgrade-hint">
                            Upgrade to Pro for more resources (4GB RAM, 4 vCPU,
                            20GB Disk).
                        </p>
                    {/if}
                {:else}
                    <div class="port-forwards-header">
                        <p class="section-description">
                            Access services running in your terminal (like
                            localhost:8080) directly from your browser.
                        </p>
                        <button
                            class="btn btn-primary btn-sm"
                            onclick={openAddForwardModal}
                        >
                            + Add Forward
                        </button>
                    </div>

                    {#if isLoadingForwards}
                        <div class="loading-state">
                            <div class="spinner-sm"></div>
                            <p>Loading...</p>
                        </div>
                    {:else if portForwards.length === 0}
                        <div class="empty-state">
                            <div class="empty-icon">
                                <StatusIcon status="plug" size={24} />
                            </div>
                            <p>No active port forwards.</p>
                        </div>
                    {:else}
                        <div class="forwards-list">
                            {#each portForwards as pf (pf.id)}
                                <div class="forward-card">
                                    <div class="forward-info">
                                        <div class="forward-header">
                                            <span class="forward-name">
                                                {pf.name ||
                                                    `Port ${pf.container_port}`}
                                            </span>
                                            <span class="port-badge"
                                                >:{pf.container_port}</span
                                            >
                                        </div>
                                        <div class="forward-url">
                                            <a
                                                href={pf.proxy_url}
                                                target="_blank"
                                                rel="noopener"
                                                class="proxy-link"
                                                title={pf.proxy_url}
                                            >
                                                {pf.proxy_url}
                                            </a>
                                        </div>
                                    </div>
                                    <div class="forward-actions">
                                        <button
                                            class="btn btn-primary btn-xs"
                                            onclick={() =>
                                                window.open(
                                                    pf.proxy_url,
                                                    "_blank",
                                                )}
                                            title="Open in browser"
                                        >
                                            Open
                                        </button>
                                        <button
                                            class="btn btn-secondary btn-xs"
                                            onclick={() => {
                                                navigator.clipboard.writeText(
                                                    pf.proxy_url,
                                                );
                                                toast.success("URL copied!");
                                            }}
                                            title="Copy URL"
                                        >
                                            Copy
                                        </button>
                                        <button
                                            class="btn btn-danger btn-xs"
                                            onclick={() =>
                                                deletePortForward(
                                                    pf.id,
                                                    pf.name ||
                                                        String(
                                                            pf.container_port,
                                                        ),
                                                )}
                                        >
                                            Stop
                                        </button>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                {/if}
            </div>

            <div class="modal-actions">
                {#if activeTab === "settings"}
                    <button class="btn btn-cancel" onclick={handleClose}>
                        Cancel
                    </button>
                    <button
                        class="btn btn-confirm"
                        onclick={handleSave}
                        disabled={isSaving || !name.trim()}
                    >
                        {#if isSaving}
                            <span class="spinner-sm"></span>
                            Saving...
                        {:else}
                            Save Changes
                        {/if}
                    </button>
                {:else}
                    <button class="btn btn-cancel" onclick={handleClose}>
                        Close
                    </button>
                {/if}
            </div>

            <div class="modal-border-glow"></div>
        </div>
    </div>

    <!-- Add Forward Modal Overlay -->
    {#if showAddForwardModal}
        <div
            class="modal-overlay-nested"
            onclick={(e) => { if (e.target === e.currentTarget) closeAddForwardModal(); }}
            onkeydown={(e) => e.key === "Escape" && closeAddForwardModal()}
            transition:fade={{ duration: 150 }}
            role="dialog"
            aria-modal="true"
            aria-labelledby="add-forward-title"
            tabindex="-1"
        >
            <div
                class="modal-container-nested"
                onclick={(e) => e.stopPropagation()}
                transition:scale={{ duration: 150, start: 0.95 }}
            >
                <div class="modal-header">
                    <h3 id="add-forward-title" class="modal-title">
                        Add Port Forward
                    </h3>
                    <button
                        class="close-btn"
                        onclick={closeAddForwardModal}
                        aria-label="Close"
                    >
                        <svg
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            width="16"
                            height="16"
                        >
                            <path d="M18 6L6 18M6 6l12 12" />
                        </svg>
                    </button>
                </div>
                <div class="modal-body">
                    <div class="form-group">
                        <label for="pf-name">Name (Optional)</label>
                        <input
                            id="pf-name"
                            type="text"
                            bind:value={newForwardName}
                            placeholder="e.g. Web Server"
                            class="input"
                        />
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label for="pf-container">Container Port</label>
                            <input
                                id="pf-container"
                                type="number"
                                bind:value={newContainerPort}
                                placeholder="8080"
                                class="input"
                                min="1"
                                max="65535"
                            />
                            <small class="form-hint">The port your app is listening on inside the container</small>
                        </div>
                    </div>
                </div>
                <div class="modal-actions">
                    <button
                        class="btn btn-cancel"
                        onclick={closeAddForwardModal}>Cancel</button
                    >
                    <button
                        class="btn btn-confirm"
                        onclick={addPortForward}
                        disabled={isAddingForward || !newContainerPort}
                    >
                        {isAddingForward ? "Adding..." : "Add Forward"}
                    </button>
                </div>
            </div>
        </div>
    {/if}
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
        backdrop-filter: blur(4px);
    }

    .modal-container {
        position: relative;
        width: 520px;
        max-width: 90vw;
        max-height: 90vh;
        overflow-y: auto;
        background: var(--bg, #0a0a0a);
        border: 1px solid var(--border, #1a1a1a);
        padding: 24px;
    }

    .modal-border-glow {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 2px;
        background: linear-gradient(90deg, var(--accent, #00ff41), #00ffaa);
        box-shadow: 0 0 20px rgba(0, 255, 65, 0.4);
    }

    .modal-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 20px;
    }

    .modal-icon {
        width: 32px;
        height: 32px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 4px;
        flex-shrink: 0;
        background: rgba(0, 255, 65, 0.15);
        color: var(--accent, #00ff41);
    }

    .modal-icon svg {
        width: 18px;
        height: 18px;
    }

    .modal-title {
        font-size: 16px;
        font-weight: 600;
        color: var(--text, #e0e0e0);
        margin: 0;
        font-family: var(--font-mono, monospace);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        flex: 1;
    }

    .close-btn {
        width: 28px;
        height: 28px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: transparent;
        border: 1px solid var(--border, #1a1a1a);
        color: var(--text-muted, #666);
        cursor: pointer;
        transition: all 0.15s;
    }

    .close-btn:hover {
        border-color: var(--text-muted, #666);
        color: var(--text, #e0e0e0);
    }

    .close-btn svg {
        width: 14px;
        height: 14px;
    }

    /* Tabs */
    .tabs {
        display: flex;
        gap: 16px;
        margin-bottom: 20px;
        border-bottom: 1px solid var(--border, #1a1a1a);
    }

    .tab-btn {
        background: none;
        border: none;
        padding: 10px 12px;
        color: var(--text-muted, #666);
        cursor: pointer;
        font-size: 13px;
        font-family: var(--font-mono, monospace);
        border-bottom: 2px solid transparent;
        transition: all 0.15s;
    }

    .tab-btn:hover {
        color: var(--text, #e0e0e0);
    }

    .tab-btn.active {
        color: var(--accent, #00ff41);
        border-bottom-color: var(--accent, #00ff41);
    }

    /* Port Forwarding */
    .port-forwards-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
    }

    .section-description {
        font-size: 12px;
        color: var(--text-muted, #666);
        margin: 0;
        max-width: 70%;
        line-height: 1.4;
    }

    .forwards-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
        max-height: 300px;
        overflow-y: auto;
    }

    .forward-card {
        display: flex;
        flex-direction: column;
        gap: 10px;
        padding: 14px;
        background: var(--bg-secondary, #111);
        border: 1px solid var(--border, #1a1a1a);
        border-radius: 4px;
    }

    .forward-info {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .forward-header {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .forward-name {
        font-size: 13px;
        font-weight: 600;
        color: var(--text, #e0e0e0);
    }

    .forward-url {
        display: flex;
        align-items: center;
        font-size: 11px;
        font-family: var(--font-mono, monospace);
    }

    .forward-details {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 11px;
        font-family: var(--font-mono, monospace);
        color: var(--text-secondary, #a0a0a0);
    }

    .port-badge {
        background: rgba(255, 255, 255, 0.05);
        padding: 2px 6px;
        border-radius: 2px;
    }

    .arrow {
        color: var(--text-muted, #666);
    }

    .proxy-link {
        color: var(--accent, #00ff65);
        text-decoration: none;
        font-size: 11px;
        word-break: break-all;
        background: rgba(0, 255, 65, 0.05);
        padding: 4px 8px;
        border-radius: 2px;
        border: 1px solid rgba(0, 255, 65, 0.15);
    }

    .proxy-link:hover {
        text-decoration: underline;
        background: rgba(0, 255, 65, 0.1);
    }

    .forward-actions {
        display: flex;
        gap: 6px;
        flex-shrink: 0;
        justify-content: flex-end;
    }

    .btn-sm {
        padding: 6px 12px;
        font-size: 11px;
    }

    .btn-xs {
        padding: 4px 8px;
        font-size: 10px;
    }

    .btn-danger {
        background: transparent;
        border-color: #ff003c;
        color: #ff003c;
    }

    .btn-danger:hover {
        background: rgba(255, 0, 60, 0.1);
    }

    /* Nested Modal */
    .modal-overlay-nested {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10001;
        backdrop-filter: blur(4px);
    }

    .modal-container-nested {
        width: 420px;
        max-width: 90vw;
        background: var(--bg, #0a0a0a);
        border: 1px solid var(--border, #1a1a1a);
        border-radius: 8px;
        padding: 24px;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6);
    }

    .form-row {
        display: flex;
        gap: 12px;
    }

    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 40px 0;
        color: var(--text-muted, #666);
    }

    .empty-icon {
        font-size: 24px;
        margin-bottom: 12px;
        opacity: 0.5;
    }

    .modal-body {
        margin-bottom: 24px;
    }

    .form-group {
        margin-bottom: 16px;
    }

    .form-group label {
        display: flex;
        align-items: center;
        justify-content: space-between;
        font-size: 11px;
        font-weight: 500;
        color: var(--text-muted, #666);
        margin-bottom: 8px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        font-family: var(--font-mono, monospace);
    }

    .form-hint {
        display: block;
        font-size: 11px;
        color: var(--text-muted, #888);
        margin-top: 6px;
        font-family: var(--font-mono, monospace);
    }

    .value-display {
        color: var(--accent, #00ff41);
        font-weight: 600;
    }

    .input {
        width: 100%;
        padding: 10px 12px;
        background: var(--bg-secondary, #111);
        border: 1px solid var(--border, #1a1a1a);
        color: var(--text, #e0e0e0);
        font-family: var(--font-mono, monospace);
        font-size: 13px;
        transition: border-color 0.15s;
    }

    .input:focus {
        outline: none;
        border-color: var(--accent, #00ff41);
    }

    .input::placeholder {
        color: var(--text-muted, #666);
    }

    .section-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin: 20px 0 12px;
        padding-top: 16px;
        border-top: 1px solid var(--border, #1a1a1a);
    }

    .section-title {
        font-size: 11px;
        font-weight: 600;
        color: var(--text-secondary, #a0a0a0);
        text-transform: uppercase;
        letter-spacing: 1px;
        font-family: var(--font-mono, monospace);
    }

    .trial-badge {
        font-size: 9px;
        padding: 2px 6px;
        background: rgba(255, 200, 0, 0.15);
        color: #ffc800;
        border: 1px solid rgba(255, 200, 0, 0.3);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        font-family: var(--font-mono, monospace);
    }

    .slider {
        width: 100%;
        height: 6px;
        -webkit-appearance: none;
        appearance: none;
        background: var(--border, #1a1a1a);
        outline: none;
        cursor: pointer;
        border-radius: 3px;
        margin: 8px 0;
    }

    .slider::-webkit-slider-runnable-track {
        width: 100%;
        height: 6px;
        background: var(--border, #1a1a1a);
        border-radius: 3px;
    }

    .slider::-webkit-slider-thumb {
        -webkit-appearance: none;
        appearance: none;
        width: 18px;
        height: 18px;
        background: var(--accent, #00ff41);
        cursor: pointer;
        border: none;
        border-radius: 50%;
        margin-top: -6px;
        box-shadow: 0 0 8px rgba(0, 255, 65, 0.5);
        transition:
            transform 0.15s,
            box-shadow 0.15s;
    }

    .slider::-webkit-slider-thumb:hover {
        transform: scale(1.1);
        box-shadow: 0 0 12px rgba(0, 255, 65, 0.7);
    }

    .slider::-moz-range-track {
        width: 100%;
        height: 6px;
        background: var(--border, #1a1a1a);
        border-radius: 3px;
    }

    .slider::-moz-range-thumb {
        width: 18px;
        height: 18px;
        background: var(--accent, #00ff41);
        cursor: pointer;
        border: none;
        border-radius: 50%;
        box-shadow: 0 0 8px rgba(0, 255, 65, 0.5);
    }

    .slider::-moz-range-thumb:hover {
        box-shadow: 0 0 12px rgba(0, 255, 65, 0.7);
    }

    .slider:focus {
        outline: none;
    }

    .slider:focus::-webkit-slider-thumb {
        box-shadow: 0 0 12px rgba(0, 255, 65, 0.7);
    }

    .range-labels {
        display: flex;
        justify-content: space-between;
        margin-top: 4px;
        font-size: 10px;
        color: var(--text-muted, #666);
        font-family: var(--font-mono, monospace);
    }

    .note {
        display: flex;
        align-items: flex-start;
        gap: 8px;
        margin-top: 16px;
        padding: 10px 12px;
        background: rgba(255, 200, 0, 0.08);
        border: 1px solid rgba(255, 200, 0, 0.2);
        font-size: 11px;
        color: var(--text-secondary, #a0a0a0);
        font-family: var(--font-mono, monospace);
        line-height: 1.4;
    }

    .note-icon {
        width: 14px;
        height: 14px;
        flex-shrink: 0;
        color: #ffc800;
    }

    .upgrade-hint {
        font-size: 11px;
        color: var(--text-muted);
        margin-top: 8px;
        text-align: center;
        font-style: italic;
    }

    .modal-actions {
        display: flex;
        gap: 12px;
        justify-content: flex-end;
    }

    .btn {
        padding: 10px 18px;
        font-size: 12px;
        font-family: var(--font-mono, monospace);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        cursor: pointer;
        transition: all 0.15s;
        border: 1px solid;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .btn-cancel {
        background: transparent;
        border-color: var(--border, #1a1a1a);
        color: var(--text-muted, #666);
    }

    .btn-cancel:hover:not(:disabled) {
        border-color: var(--text-muted, #666);
        color: var(--text, #e0e0e0);
        background: var(--bg-tertiary, #1a1a1a);
    }

    .btn-confirm {
        background: var(--accent, #00ff41);
        border-color: var(--accent, #00ff41);
        color: var(--bg, #0a0a0a);
        font-weight: 600;
    }

    .btn-confirm:hover:not(:disabled) {
        box-shadow: 0 0 15px rgba(0, 255, 65, 0.4);
    }

    .spinner-sm {
        width: 12px;
        height: 12px;
        border: 2px solid transparent;
        border-top-color: currentColor;
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    /* Firefox scrollbar */
    .modal-container {
        scrollbar-width: thin;
        scrollbar-color: var(--border, #1a1a1a) transparent;
    }
</style>
