<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import {
        containers,
        isCreating,
        creatingContainer,
        type Container,
    } from "$stores/containers";
    import { terminal } from "$stores/terminal";
    import { toast } from "$stores/toast";
    import { formatRelativeTime } from "$utils/api";

    const dispatch = createEventDispatcher<{
        create: void;
        connect: { id: string; name: string };
    }>();

    // Container actions
    async function handleStart(container: Container) {
        const toastId = toast.loading(`Starting ${container.name}...`);
        const result = await containers.startContainer(container.id);

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
        const toastId = toast.loading(`Stopping ${container.name}...`);
        const result = await containers.stopContainer(container.id);

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

    async function handleDelete(container: Container) {
        if (
            !confirm(
                `Delete terminal "${container.name}"? This cannot be undone.`,
            )
        ) {
            return;
        }

        const toastId = toast.loading(`Deleting ${container.name}...`);
        const result = await containers.deleteContainer(container.id);

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

    function handleConnect(container: Container) {
        dispatch("connect", { id: container.id, name: container.name });
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

    function getImageIcon(image: string): string {
        const lower = image.toLowerCase();
        if (lower.includes("ubuntu")) return "🟠";
        if (lower.includes("debian")) return "🔴";
        if (lower.includes("alpine")) return "🔵";
        if (lower.includes("fedora")) return "🔵";
        if (
            lower.includes("centos") ||
            lower.includes("rocky") ||
            lower.includes("alma")
        )
            return "🟣";
        if (lower.includes("arch")) return "🔷";
        if (lower.includes("kali")) return "🐉";
        return "🐧";
    }

    // Reactive
    $: containerList = $containers.containers;
    $: isLoading = $containers.isLoading;
    $: containerLimit = $containers.limit;
    $: currentlyCreating = $isCreating;
    $: creatingInfo = $creatingContainer;
    // Count creating as part of limit
    $: effectiveCount = containerList.length + (currentlyCreating ? 1 : 0);
</script>

<div class="dashboard">
    <div class="dashboard-header">
        <div class="dashboard-title">
            <h1>Terminals</h1>
            <span class="count-badge">
                {effectiveCount} / {containerLimit}
            </span>
        </div>
        <div class="dashboard-actions">
            <button
                class="btn btn-secondary btn-sm"
                on:click={() => containers.fetchContainers()}
            >
                ↻ Refresh
            </button>
            <button
                class="btn btn-primary"
                on:click={() => dispatch("create")}
                disabled={effectiveCount >= containerLimit || currentlyCreating}
            >
                {#if currentlyCreating}
                    Creating...
                {:else}
                    + New Terminal
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
            <div class="empty-icon">📦</div>
            <h2>No Terminals Yet</h2>
            <p>
                Create your first terminal to get started with a Linux
                environment in seconds.
            </p>
            <button
                class="btn btn-primary btn-lg"
                on:click={() => dispatch("create")}
            >
                + Create Terminal
            </button>
        </div>
    {:else}
        <div class="containers-grid">
            {#if currentlyCreating && creatingInfo}
                <div class="container-card creating-card">
                    <div class="container-header">
                        <span class="container-icon">⏳</span>
                        <div class="container-info">
                            <h3 class="container-name">
                                {creatingInfo.name || "New Terminal"}
                            </h3>
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
                                style="width: {creatingInfo.progress}%"
                            ></div>
                        </div>
                        <div class="progress-info">
                            <span class="progress-stage"
                                >{creatingInfo.stage}</span
                            >
                            <span class="progress-percent"
                                >{creatingInfo.progress}%</span
                            >
                        </div>
                        <p class="progress-message">{creatingInfo.message}</p>
                    </div>
                </div>
            {/if}
            {#each containerList as container (container.id)}
                <div
                    class="container-card"
                    class:active={hasActiveSession(container.id)}
                >
                    <div class="container-header">
                        <span class="container-icon"
                            >{getImageIcon(container.image)}</span
                        >
                        <div class="container-info">
                            <h3 class="container-name">{container.name}</h3>
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
                        {#if container.idle_seconds !== undefined}
                            <div class="meta-item">
                                <span class="meta-label">Idle</span>
                                <span class="meta-value"
                                    >{Math.floor(
                                        container.idle_seconds / 60,
                                    )}m</span
                                >
                            </div>
                        {/if}
                        <div class="meta-item">
                            <span class="meta-label">ID</span>
                            <span class="meta-value mono"
                                >{container.id.slice(0, 12)}</span
                            >
                        </div>
                    </div>

                    <div class="container-actions">
                        {#if container.status === "running"}
                            <div class="action-row">
                                <button
                                    class="btn btn-primary btn-sm flex-1"
                                    on:click={() => handleConnect(container)}
                                >
                                    {hasActiveSession(container.id)
                                        ? "+ New Tab"
                                        : "Connect"}
                                </button>
                                <button
                                    class="btn btn-secondary btn-sm"
                                    title="SSH Info"
                                >
                                    SSH
                                </button>
                            </div>
                            <div class="action-row">
                                <button
                                    class="btn btn-secondary btn-sm flex-1"
                                    on:click={() => handleStop(container)}
                                >
                                    Stop
                                </button>
                                <button
                                    class="btn btn-danger btn-sm flex-1"
                                    on:click={() => handleDelete(container)}
                                >
                                    Delete
                                </button>
                            </div>
                        {:else if container.status === "stopped"}
                            <div class="action-row">
                                <button
                                    class="btn btn-primary btn-sm flex-1"
                                    on:click={() => handleStart(container)}
                                >
                                    Start
                                </button>
                                <button
                                    class="btn btn-danger btn-sm flex-1"
                                    on:click={() => handleDelete(container)}
                                >
                                    Delete
                                </button>
                            </div>
                        {:else if container.status === "error"}
                            <div class="action-row">
                                <button
                                    class="btn btn-danger btn-sm flex-1"
                                    on:click={() => handleDelete(container)}
                                >
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
        font-size: 48px;
        margin-bottom: 16px;
    }

    .empty-state h2 {
        font-size: 18px;
        margin-bottom: 8px;
        text-transform: uppercase;
    }

    .empty-state p {
        color: var(--text-muted);
        max-width: 400px;
        margin-bottom: 24px;
    }

    /* Containers Grid */
    .containers-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 16px;
    }

    .container-card {
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

    .container-header {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        margin-bottom: 16px;
    }

    .container-icon {
        font-size: 24px;
        line-height: 1;
    }

    .container-info {
        flex: 1;
        min-width: 0;
    }

    .container-name {
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
        color: var(--text-muted);
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

    .container-meta {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 8px;
        margin-bottom: 16px;
        padding: 12px;
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
        color: var(--text-muted);
        text-transform: uppercase;
    }

    .meta-value {
        font-size: 12px;
        color: var(--text-secondary);
    }

    .meta-value.mono {
        font-family: var(--font-mono);
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
</style>
