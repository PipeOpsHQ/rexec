<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { fade, scale } from "svelte/transition";
    import { api, formatMemory, formatStorage, formatCPU } from "$utils/api";
    import { toast } from "$stores/toast";
    import { containers, type Container } from "$stores/containers";
    import { terminal } from "$stores/terminal";

    export let show: boolean = false;
    export let container: Container | null = null;
    export let isPaidUser: boolean = false;

    const dispatch = createEventDispatcher<{
        close: void;
        updated: Container;
    }>();

    let name = "";
    let memoryMB = 512;
    let cpuShares = 512;
    let diskMB = 2048;
    let isSaving = false;
    let initialized = false;

    // Trial/free tier limits - generous during 60-day trial period
    $: resourceLimits = {
        minMemory: 256,
        maxMemory: isPaidUser ? 8192 : 4096,  // 4GB for trial, 8GB for paid
        minCPU: 250,
        maxCPU: isPaidUser ? 4000 : 2000,     // 2 vCPU for trial, 4 for paid
        minDisk: 1024,
        maxDisk: isPaidUser ? 51200 : 16384   // 16GB for trial, 50GB for paid
    };

    // Initialize form values when modal opens
    function initializeValues() {
        if (!container) return;
        
        name = container.name || "";
        
        // Get raw values from container resources
        const rawMemory = container.resources?.memory_mb ?? 512;
        const rawCpu = container.resources?.cpu_shares ?? 512;
        const rawDisk = container.resources?.disk_mb ?? 2048;
        
        // Clamp values to be within slider range
        const maxMem = isPaidUser ? 8192 : 4096;
        const maxCpu = isPaidUser ? 4000 : 2000;
        const maxDisk = isPaidUser ? 51200 : 16384;
        
        memoryMB = Math.max(256, Math.min(rawMemory, maxMem));
        cpuShares = Math.max(250, Math.min(rawCpu, maxCpu));
        diskMB = Math.max(1024, Math.min(rawDisk, maxDisk));
        
        console.log('[Settings] Initialized:', { 
            name, memoryMB, cpuShares, diskMB, 
            raw: { rawMemory, rawCpu, rawDisk },
            resources: container.resources 
        });
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
            const response = await api.patch(`/api/containers/${containerId}/settings`, {
                name: name.trim(),
                memory_mb: memoryMB,
                cpu_shares: cpuShares,
                disk_mb: diskMB
            });

            if (response.ok) {
                const responseData = response.data as { container: Container; restarted?: boolean };
                
                // If container was restarted, trigger reconnect after a short delay
                if (responseData.restarted && responseData.container) {
                    toast.success("Terminal settings updated - reconnecting...");
                    
                    const oldContainerId = container.id;
                    const newContainerId = responseData.container.id;

                    setTimeout(() => {
                        if (oldContainerId && newContainerId) {
                            // Update the session to point to the new container ID and reconnect
                            terminal.updateSessionContainerId(oldContainerId, newContainerId);
                        }
                    }, 2000);
                } else {
                    toast.success("Terminal settings updated");
                }
                
                if (responseData.container) {
                    dispatch("updated", responseData.container);
                }
                handleClose();
                // Refresh containers to get latest data
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

<svelte:window on:keydown={handleKeydown} />

{#if show && container}
    <div class="modal-backdrop" transition:fade={{ duration: 150 }}>
        <div
            class="modal-container"
            transition:scale={{ duration: 150, start: 0.95 }}
        >
            <div class="modal-header">
                <div class="modal-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="3" />
                        <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
                    </svg>
                </div>
                <h2 class="modal-title">Terminal Settings</h2>
                <button class="close-btn" on:click={handleClose}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M18 6L6 18M6 6l12 12" />
                    </svg>
                </button>
            </div>

            <div class="modal-body">
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
                        <span class="value-display">{formatMemory(memoryMB)}</span>
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
                        <span>{formatMemory(resourceLimits.minMemory)}</span>
                        <span>{formatMemory(resourceLimits.maxMemory)}</span>
                    </div>
                </div>

                <div class="form-group">
                    <label for="cpu">
                        CPU
                        <span class="value-display">{formatCPU(cpuShares)}</span>
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
                        <span class="value-display">{formatStorage(diskMB)}</span>
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
                    <svg class="note-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10" />
                        <path d="M12 16v-4M12 8h.01" />
                    </svg>
                    Resource changes require terminal restart to take effect.
                </p>
            </div>

            <div class="modal-actions">
                <button class="btn btn-cancel" on:click={handleClose}>
                    Cancel
                </button>
                <button
                    class="btn btn-confirm"
                    on:click={handleSave}
                    disabled={isSaving || !name.trim()}
                >
                    {#if isSaving}
                        <span class="spinner-sm"></span>
                        Saving...
                    {:else}
                        Save Changes
                    {/if}
                </button>
            </div>

            <div class="modal-border-glow"></div>
        </div>
    </div>
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
        width: 420px;
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
        transition: transform 0.15s, box-shadow 0.15s;
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
        to { transform: rotate(360deg); }
    }

    /* Firefox scrollbar */
    .modal-container {
        scrollbar-width: thin;
        scrollbar-color: var(--border, #1a1a1a) transparent;
    }
</style>
