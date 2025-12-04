<script lang="ts">
    import { onMount, onDestroy, tick, createEventDispatcher } from "svelte";
    import { get } from "svelte/store";
    import { terminal, type TerminalSession } from "$stores/terminal";
    import { recordings } from "$stores/recordings";
    import { collab } from "$stores/collab";
    import { toast } from "$stores/toast";
    import { token } from "$stores/auth";
    import { formatMemoryBytes } from "$utils/api";
    import SplitTerminalView from "./SplitTerminalView.svelte";

    export let session: TerminalSession;
    
    // Check if current user is a guest in this session
    $: isGuest = session.isCollabSession === true;
    $: isViewOnly = session.collabMode === 'view';
    
    // Check if this session has active sharing (for pulsing indicator)
    $: hasActiveSharing = $collab.activeSession?.containerId === session.containerId && 
                          $collab.participants.length > 0;

    const dispatch = createEventDispatcher();

    // Track connected status for showing brief "connected" indicator
    let showConnectedIndicator = false;
    let previousStatus = session?.status;
    
    // More actions dropdown state
    let showMoreMenu = false;
    let moreButtonEl: HTMLButtonElement;
    let menuPosition = { top: 0, right: 0 };
    
    function toggleMoreMenu() {
        if (!showMoreMenu && moreButtonEl) {
            const rect = moreButtonEl.getBoundingClientRect();
            menuPosition = {
                top: rect.bottom + 4,
                right: window.innerWidth - rect.right
            };
        }
        showMoreMenu = !showMoreMenu;
    }
    
    // Show connected indicator briefly when status changes to connected
    $: if (session?.status === 'connected' && previousStatus === 'connecting') {
        showConnectedIndicator = true;
        setTimeout(() => {
            showConnectedIndicator = false;
        }, 2000);
    }
    $: previousStatus = session?.status;

    // Check if recording this terminal
    $: isRecording = $recordings.activeRecordings.get(session.containerId)?.recording || false;

    // Get usage color based on percentage
    function getUsageColor(percent: number): string {
        if (percent >= 90) return '#ff4444';      // Red - critical
        if (percent >= 75) return '#ff8800';      // Orange - warning  
        if (percent >= 50) return '#ffcc00';      // Yellow - moderate
        return '#44ff44';                          // Green - healthy
    }

    // Calculate memory percentage
    $: memoryPercent = session.stats.memoryLimit > 0 
        ? (session.stats.memory / session.stats.memoryLimit) * 100 
        : 0;
    
    // CPU is already a percentage (0-100+)
    $: cpuColor = getUsageColor(session.stats.cpu);
    $: memColor = getUsageColor(memoryPercent);

    let containerElement: HTMLDivElement;
    let attachedToContainer: HTMLDivElement | null = null;

    // Synchronously check and attach terminal if needed
    async function ensureAttached() {
        if (!containerElement || !session?.terminal) return;

        // Only reattach if the container has changed
        if (attachedToContainer === containerElement) return;

        // Wait for DOM to be ready
        await tick();

        try {
            // Use the store's reattachTerminal method which properly handles
            // ResizeObserver cleanup and setup for the new container
            terminal.reattachTerminal(session.id, containerElement);
            attachedToContainer = containerElement;
        } catch (e) {
            console.error("Failed to attach terminal:", e);
            // Try to recover by recreating attachment
            attachedToContainer = null;
        }
    }

    onMount(async () => {
        if (containerElement && session) {
            if (session.terminal?.element) {
                // Terminal already exists, reattach to new container
                terminal.reattachTerminal(session.id, containerElement);
                attachedToContainer = containerElement;
            } else {
                // First time - let the store create and attach the terminal
                terminal.attachTerminal(session.id, containerElement);
                attachedToContainer = containerElement;
            }

            // Connect WebSocket if not already connected
            if (
                !session.ws ||
                (session.ws.readyState !== WebSocket.OPEN &&
                    session.ws.readyState !== WebSocket.CONNECTING)
            ) {
                terminal.connectWebSocket(session.id);
            }
        }
    });

    onDestroy(() => {
        // Don't dispose terminal - it's managed by the store
        attachedToContainer = null;
    });

    // Use reactive statement to handle container changes (dock/float switch)
    $: if (
        containerElement &&
        session?.terminal &&
        attachedToContainer !== containerElement
    ) {
        // Use async IIFE to handle the async ensureAttached
        (async () => {
            await ensureAttached();
        })();
    }

    // Actions
    function handleReconnect() {
        terminal.reconnectSession(session.id);
    }

    function handleClear() {
        if (session.terminal) {
            session.terminal.clear();
        }
    }

    function handleCopy() {
        if (session.terminal) {
            const selection = session.terminal.getSelection();
            if (selection) {
                navigator.clipboard.writeText(selection);
            }
        }
    }

    function handlePaste() {
        navigator.clipboard.readText().then((text) => {
            if (session.ws && session.ws.readyState === WebSocket.OPEN) {
                session.ws.send(JSON.stringify({ type: "input", data: text }));
            }
        });
    }

    function handleCopyLink() {
        const url = `${window.location.origin}/terminal/${session.containerId}`;
        navigator.clipboard
            .writeText(url)
            .then(() => {
                toast.success("Terminal link copied to clipboard");
            })
            .catch(() => {
                toast.error("Failed to copy link");
            });
    }

    // File upload/download handlers
    let fileInput: HTMLInputElement;
    let isUploading = false;
    let showDownloadModal = false;
    let downloadPath = '/home/user/';

    function handleUploadClick() {
        fileInput?.click();
    }

    async function handleFileUpload(event: Event) {
        const input = event.target as HTMLInputElement;
        const file = input.files?.[0];
        if (!file) return;

        // Check file size (max 100MB)
        if (file.size > 100 * 1024 * 1024) {
            toast.error('File too large (max 100MB)');
            return;
        }

        isUploading = true;
        const formData = new FormData();
        formData.append('file', file);

        try {
            const authToken = get(token);
            if (!authToken) {
                toast.error('Not authenticated');
                isUploading = false;
                return;
            }
            const response = await fetch(`/api/containers/${session.containerId}/files?path=/home/user/`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${authToken}`
                },
                body: formData
            });

            if (response.ok) {
                const result = await response.json();
                toast.success(`Uploaded ${result.filename} to ${result.path}`);
            } else {
                const error = await response.json();
                toast.error(error.error || 'Upload failed');
            }
        } catch (err) {
            toast.error('Upload failed');
        } finally {
            isUploading = false;
            input.value = ''; // Reset input
        }
    }

    function handleDownloadClick() {
        showDownloadModal = true;
    }

    async function handleDownload() {
        if (!downloadPath.trim()) {
            toast.error('Please enter a file path');
            return;
        }

        try {
            const authToken = get(token);
            if (!authToken) {
                toast.error('Not authenticated');
                return;
            }
            const response = await fetch(`/api/containers/${session.containerId}/files?path=${encodeURIComponent(downloadPath)}`, {
                headers: {
                    'Authorization': `Bearer ${authToken}`
                }
            });

            if (response.ok) {
                const blob = await response.blob();
                const filename = downloadPath.split('/').pop() || 'download';
                
                // Create download link
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = filename;
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);
                
                toast.success(`Downloaded ${filename}`);
                showDownloadModal = false;
            } else {
                const error = await response.json();
                toast.error(error.error || 'Download failed');
            }
        } catch (err) {
            toast.error('Download failed');
        }
    }

    // Collab and Recording handlers
    function handleCollab() {
        console.log('[TerminalPanel] Opening collab panel for:', session.containerId);
        dispatch('openCollab', { containerId: session.containerId });
    }

    async function handleRecording() {
        if (isRecording) {
            const result = await recordings.stopRecording(session.containerId);
            if (result) {
                toast.success(`Recording saved (${result.duration})`);
            }
        } else {
            const recordingId = await recordings.startRecording(session.containerId);
            if (recordingId) {
                toast.success('Recording started');
            }
        }
    }

    function handleRecordingsPanel() {
        console.log('[TerminalPanel] Opening recordings panel for:', session.containerId);
        dispatch('openRecordings', { containerId: session.containerId });
    }

    // Split pane handlers
    function handleSplitHorizontal() {
        terminal.splitPane(session.id, "horizontal");
        toast.success("Split terminal horizontally");
    }

    function handleSplitVertical() {
        terminal.splitPane(session.id, "vertical");
        toast.success("Split terminal vertically");
    }

    // Split pane resizing
    let isResizingSplit = false;
    let resizeIndex = 0;
    let resizeStartPos = 0;
    let resizeStartSizes: number[] = [];
    let splitContainerEl: HTMLDivElement;

    function handleSplitResizeStart(event: MouseEvent, index: number) {
        event.preventDefault();
        isResizingSplit = true;
        resizeIndex = index;
        resizeStartSizes = [...(splitLayout?.sizes || [50, 50])];
        resizeStartPos = splitLayout?.direction === 'horizontal' ? event.clientX : event.clientY;
        
        window.addEventListener('mousemove', handleSplitResizeMove);
        window.addEventListener('mouseup', handleSplitResizeEnd);
    }

    function handleSplitResizeMove(event: MouseEvent) {
        if (!isResizingSplit || !splitContainerEl) return;

        const rect = splitContainerEl.getBoundingClientRect();
        const totalSize = splitLayout?.direction === 'horizontal' ? rect.width : rect.height;
        const currentPos = splitLayout?.direction === 'horizontal' ? event.clientX : event.clientY;
        const startOffset = splitLayout?.direction === 'horizontal' ? rect.left : rect.top;
        
        // Calculate the position as a percentage
        const posPercent = ((currentPos - startOffset) / totalSize) * 100;
        
        // Clamp between 20% and 80%
        const clampedPercent = Math.max(20, Math.min(80, posPercent));
        
        // Update sizes
        const newSizes = [clampedPercent, 100 - clampedPercent];
        terminal.setSplitPaneSizes(session.id, newSizes);
    }

    function handleSplitResizeEnd() {
        isResizingSplit = false;
        window.removeEventListener('mousemove', handleSplitResizeMove);
        window.removeEventListener('mouseup', handleSplitResizeEnd);
        
        // Fit terminals after resize
        setTimeout(() => terminal.fitSession(session.id), 50);
    }

    // Focus terminal when clicking on container
    function handleContainerClick() {
        if (session.terminal) {
            session.terminal.focus();
        }
    }

    // Reactive status
    $: status = session?.status || "disconnected";
    $: isConnected = status === "connected";
    $: isConnecting = status === "connecting";
    $: isDisconnected = status === "disconnected" || status === "error";
    $: isSettingUp = session?.isSettingUp || false;
    $: setupMessage = session?.setupMessage || "";
    $: hasSplitPanes = session?.splitPanes?.size > 0;
    $: splitPanes = session?.splitPanes ? Array.from(session.splitPanes.values()) : [];
    $: splitLayout = session?.splitLayout;
</script>

<div class="terminal-panel-wrapper">
    <div class="terminal-toolbar">
        <div class="toolbar-left">
            <span class="terminal-name">{session.name}</span>
            {#if session.isCollabSession && session.collabMode === 'view'}
                <span class="view-only-badge" title="View Only - You can watch but cannot type">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/>
                    </svg>
                    VIEW ONLY
                </span>
            {/if}
            <span
                class="terminal-status"
                class:connected={isConnected}
                class:connecting={isConnecting}
                class:disconnected={isDisconnected}
            >
                <span class="status-indicator"></span>
                {status}
            </span>
            {#if isConnected && (session.stats.cpu > 0 || session.stats.memory > 0)}
                <span class="terminal-stats">
                    <span class="stat-item stat-cpu" title="CPU Usage ({session.stats.cpu.toFixed(1)}%)">
                        <span class="stat-label">CPU</span>
                        <span class="stat-value" style="color: {cpuColor}">{session.stats.cpu.toFixed(1)}%</span>
                    </span>
                    <span class="stat-divider">|</span>
                    <span class="stat-item stat-mem" title="Memory: {formatMemoryBytes(session.stats.memory)} / {formatMemoryBytes(session.stats.memoryLimit)} ({memoryPercent.toFixed(0)}%)">
                        <span class="stat-label">MEM</span>
                        <span class="stat-value" style="color: {memColor}">{formatMemoryBytes(session.stats.memory)}</span>
                        {#if session.stats.memoryLimit > 0}
                            <span class="stat-limit">/ {formatMemoryBytes(session.stats.memoryLimit)}</span>
                        {/if}
                    </span>
                    <span class="stat-divider">|</span>
                    <span class="stat-item stat-disk" title="Disk: {formatMemoryBytes(session.stats.diskWrite)} used{session.stats.diskLimit > 0 ? ' / ' + formatMemoryBytes(session.stats.diskLimit) + ' limit' : ''} (R:{formatMemoryBytes(session.stats.diskRead)} W:{formatMemoryBytes(session.stats.diskWrite)})">
                        <span class="stat-label">DISK</span>
                        <span class="stat-value">{formatMemoryBytes(session.stats.diskWrite)}</span>
                        {#if session.stats.diskLimit > 0}
                            <span class="stat-limit">/ {formatMemoryBytes(session.stats.diskLimit)}</span>
                        {/if}
                    </span>
                    <span class="stat-divider">|</span>
                    <span class="stat-item stat-net" title="Network: RX {formatMemoryBytes(session.stats.netRx)} / TX {formatMemoryBytes(session.stats.netTx)}">
                        <span class="stat-label">NET</span>
                        <span class="stat-io">
                            <span class="stat-io-item stat-rx">RX:{formatMemoryBytes(session.stats.netRx)}</span>
                            <span class="stat-io-item stat-tx">TX:{formatMemoryBytes(session.stats.netTx)}</span>
                        </span>
                    </span>
                </span>
            {/if}
            {#if isSettingUp}
                <span class="setup-indicator">
                    <span class="setup-spinner"></span>
                    Installing...
                </span>
            {/if}
        </div>

        <div class="toolbar-actions">
            {#if isDisconnected}
                <button
                    class="toolbar-btn reconnect-btn"
                    on:click={handleReconnect}
                    title="Reconnect"
                >
                    <svg class="toolbar-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M11.534 7h3.932a.25.25 0 0 1 .192.41l-1.966 2.36a.25.25 0 0 1-.384 0l-1.966-2.36a.25.25 0 0 1 .192-.41zm-11 2h3.932a.25.25 0 0 0 .192-.41L2.692 6.23a.25.25 0 0 0-.384 0L.342 8.59A.25.25 0 0 0 .534 9z"/>
                        <path fill-rule="evenodd" d="M8 3c-1.552 0-2.94.707-3.857 1.818a.5.5 0 1 1-.771-.636A6.002 6.002 0 0 1 13.917 7H12.9A5.002 5.002 0 0 0 8 3zM3.1 9a5.002 5.002 0 0 0 8.757 2.182.5.5 0 1 1 .771.636A6.002 6.002 0 0 1 2.083 9H3.1z"/>
                    </svg>
                </button>
            {/if}
            
            <!-- Primary Actions (Icons Only) - Only for owners -->
            {#if !isGuest}
                <button class="toolbar-btn icon-btn" on:click={handleSplitHorizontal} title="Split Horizontal">
                    <svg class="toolbar-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"/>
                        <line x1="8" y1="1" x2="8" y2="15" stroke="currentColor" stroke-width="1"/>
                    </svg>
                </button>
                <button class="toolbar-btn icon-btn" on:click={handleSplitVertical} title="Split Vertical">
                    <svg class="toolbar-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"/>
                        <line x1="1" y1="8" x2="15" y2="8" stroke="currentColor" stroke-width="1"/>
                    </svg>
                </button>
                
                <span class="toolbar-divider"></span>
                
                <!-- Recording - Owner only -->
                <button
                    class="toolbar-btn icon-btn"
                    class:recording={isRecording}
                    on:click={handleRecording}
                    title={isRecording ? "Stop Recording" : "Start Recording"}
                >
                    <svg class="toolbar-icon" viewBox="0 0 16 16" fill="currentColor">
                        {#if isRecording}
                            <rect x="4" y="4" width="8" height="8" rx="1"/>
                        {:else}
                            <circle cx="8" cy="8" r="5"/>
                        {/if}
                    </svg>
                </button>
                
                <!-- Collaborate - Owner only (with pulsing when active) -->
                <button
                    class="toolbar-btn icon-btn collab-btn"
                    class:has-participants={hasActiveSharing}
                    on:click={handleCollab}
                    title={hasActiveSharing ? `Collaborate (${$collab.participants.length} connected)` : "Collaborate"}
                >
                    <svg class="toolbar-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M7 14s-1 0-1-1 1-4 5-4 5 3 5 4-1 1-1 1H7zm4-6a3 3 0 1 0 0-6 3 3 0 0 0 0 6z"/>
                        <path fill-rule="evenodd" d="M5.216 14A2.238 2.238 0 0 1 5 13c0-1.355.68-2.75 1.936-3.72A6.325 6.325 0 0 0 5 9c-4 0-5 3-5 4s1 1 1 1h4.216z"/>
                        <path d="M4.5 8a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5z"/>
                    </svg>
                    {#if hasActiveSharing}
                        <span class="collab-badge">{$collab.participants.length}</span>
                    {/if}
                </button>
                
                <!-- File Upload - Owner only -->
                <input 
                    type="file" 
                    bind:this={fileInput} 
                    on:change={handleFileUpload}
                    style="display: none;"
                />
                <button
                    class="toolbar-btn icon-btn upload-btn"
                    on:click={handleUploadClick}
                    disabled={isUploading || !isConnected}
                    title="Upload File to Terminal"
                >
                    {#if isUploading}
                        <div class="mini-spinner"></div>
                    {:else}
                        <svg class="toolbar-icon" viewBox="0 0 16 16" fill="currentColor">
                            <path d="M8 0a.5.5 0 0 1 .5.5v11.793l3.146-3.147a.5.5 0 0 1 .708.708l-4 4a.5.5 0 0 1-.708 0l-4-4a.5.5 0 0 1 .708-.708L7.5 12.293V.5A.5.5 0 0 1 8 0z" transform="rotate(180 8 8)"/>
                            <path d="M4.406 1.342A5.53 5.53 0 0 1 8 0c2.69 0 4.923 2 5.166 4.579C14.758 4.804 16 6.137 16 7.773 16 9.569 14.502 11 12.687 11H10a.5.5 0 0 1 0-1h2.688C13.979 10 15 8.988 15 7.773c0-1.216-1.02-2.228-2.313-2.228h-.5v-.5C12.188 2.825 10.328 1 8 1a4.53 4.53 0 0 0-2.941 1.1c-.757.652-1.153 1.438-1.153 2.055v.448l-.445.049C2.064 4.805 1 5.952 1 7.318 1 8.785 2.23 10 3.781 10H6a.5.5 0 0 1 0 1H3.781C1.708 11 0 9.366 0 7.318c0-1.763 1.266-3.223 2.942-3.593.143-.863.698-1.723 1.464-2.383z"/>
                        </svg>
                    {/if}
                </button>
            {/if}
            
            <!-- File Download - Available to all -->
            <button
                class="toolbar-btn icon-btn download-btn"
                on:click={handleDownloadClick}
                disabled={!isConnected}
                title="Download File from Terminal"
            >
                <svg class="toolbar-icon" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M8 15a.5.5 0 0 1-.5-.5V1.707L4.354 4.854a.5.5 0 1 1-.708-.708l4-4a.5.5 0 0 1 .708 0l4 4a.5.5 0 0 1-.708.708L8.5 1.707V14.5a.5.5 0 0 1-.5.5z" transform="rotate(180 8 8)"/>
                    <path d="M4.406 1.342A5.53 5.53 0 0 1 8 0c2.69 0 4.923 2 5.166 4.579C14.758 4.804 16 6.137 16 7.773 16 9.569 14.502 11 12.687 11H10a.5.5 0 0 1 0-1h2.688C13.979 10 15 8.988 15 7.773c0-1.216-1.02-2.228-2.313-2.228h-.5v-.5C12.188 2.825 10.328 1 8 1a4.53 4.53 0 0 0-2.941 1.1c-.757.652-1.153 1.438-1.153 2.055v.448l-.445.049C2.064 4.805 1 5.952 1 7.318 1 8.785 2.23 10 3.781 10H6a.5.5 0 0 1 0 1H3.781C1.708 11 0 9.366 0 7.318c0-1.763 1.266-3.223 2.942-3.593.143-.863.698-1.723 1.464-2.383z"/>
                </svg>
            </button>
            
            <span class="toolbar-divider"></span>
            
            <!-- More Actions Dropdown -->
            <div class="more-dropdown">
                <button bind:this={moreButtonEl} class="toolbar-btn icon-btn more-btn" on:click={toggleMoreMenu} title="More actions">
                    <svg class="toolbar-icon" viewBox="0 0 16 16" fill="currentColor">
                        <path d="M3 9.5a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3zm5 0a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3zm5 0a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3z"/>
                    </svg>
                </button>
                {#if showMoreMenu}
                    <div 
                        class="more-menu" 
                        style="top: {menuPosition.top}px; right: {menuPosition.right}px;"
                        on:mouseleave={() => showMoreMenu = false}
                    >
                        {#if !isGuest}
                            <button class="menu-item" on:click={() => { handleCopyLink(); showMoreMenu = false; }}>
                                <svg class="menu-icon" viewBox="0 0 16 16" fill="currentColor">
                                    <path d="M4.715 6.542 3.343 7.914a3 3 0 1 0 4.243 4.243l1.828-1.829A3 3 0 0 0 8.586 5.5L8 6.086a1.002 1.002 0 0 0-.154.199 2 2 0 0 1 .861 3.337L6.88 11.45a2 2 0 1 1-2.83-2.83l.793-.792a4.018 4.018 0 0 1-.128-1.287z"/>
                                    <path d="M6.586 4.672A3 3 0 0 0 7.414 9.5l.775-.776a2 2 0 0 1-.896-3.346L9.12 3.55a2 2 0 1 1 2.83 2.83l-.793.792c.112.42.155.855.128 1.287l1.372-1.372a3 3 0 1 0-4.243-4.243L6.586 4.672z"/>
                                </svg>
                                Copy Link
                            </button>
                        {/if}
                        <button class="menu-item" on:click={() => { handleCopy(); showMoreMenu = false; }}>
                            <svg class="menu-icon" viewBox="0 0 16 16" fill="currentColor">
                                <path d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1h1a1 1 0 0 1 1 1V14a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V3.5a1 1 0 0 1 1-1h1v-1z"/>
                                <path d="M9.5 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3zm-3-1A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3z"/>
                            </svg>
                            Copy Selection
                        </button>
                        {#if !isViewOnly}
                            <button class="menu-item" on:click={() => { handlePaste(); showMoreMenu = false; }}>
                                <svg class="menu-icon" viewBox="0 0 16 16" fill="currentColor">
                                    <path d="M3.5 2a.5.5 0 0 0-.5.5v12a.5.5 0 0 0 .5.5h9a.5.5 0 0 0 .5-.5v-12a.5.5 0 0 0-.5-.5H12a.5.5 0 0 1 0-1h.5A1.5 1.5 0 0 1 14 2.5v12a1.5 1.5 0 0 1-1.5 1.5h-9A1.5 1.5 0 0 1 2 14.5v-12A1.5 1.5 0 0 1 3.5 1H4a.5.5 0 0 1 0 1h-.5z"/>
                                    <path d="M10 .5a.5.5 0 0 0-.5-.5h-3a.5.5 0 0 0-.5.5.5.5 0 0 1-.5.5.5.5 0 0 0-.5.5V2a.5.5 0 0 0 .5.5h5A.5.5 0 0 0 11 2v-.5a.5.5 0 0 0-.5-.5.5.5 0 0 1-.5-.5z"/>
                                </svg>
                                Paste
                            </button>
                            <button class="menu-item" on:click={() => { handleClear(); showMoreMenu = false; }}>
                                <svg class="menu-icon" viewBox="0 0 16 16" fill="currentColor">
                                    <path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"/>
                                    <path fill-rule="evenodd" d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"/>
                                </svg>
                                Clear Terminal
                            </button>
                        {/if}
                        {#if !isGuest}
                            <div class="menu-divider"></div>
                            <button class="menu-item" on:click={() => { handleRecordingsPanel(); showMoreMenu = false; }}>
                                <svg class="menu-icon" viewBox="0 0 16 16" fill="currentColor">
                                    <path d="M0 1a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H1a1 1 0 0 1-1-1V1zm4 0v6h8V1H4zm8 8H4v6h8V9zM1 1v2h2V1H1zm2 3H1v2h2V4zM1 7v2h2V7H1zm2 3H1v2h2v-2zm-2 3v2h2v-2H1zM15 1h-2v2h2V1zm-2 3v2h2V4h-2zm2 3h-2v2h2V7zm-2 3v2h2v-2h-2zm2 3h-2v2h2v-2z"/>
                                </svg>
                                View Recordings
                            </button>
                        {/if}
                    </div>
                {/if}
            </div>
        </div>
    </div>

    <!-- Terminal Container - single wrapper, layout changes around the main terminal -->
    <div 
        class="terminal-wrapper"
        class:has-splits={hasSplitPanes && splitLayout}
        class:horizontal={splitLayout?.direction === 'horizontal'}
        class:vertical={splitLayout?.direction === 'vertical'}
        class:resizing={isResizingSplit}
        bind:this={splitContainerEl}
    >
        <!-- Main terminal pane - always present, never recreated -->
        <div 
            class="terminal-pane main-pane"
            style:flex={hasSplitPanes && splitLayout ? (splitLayout?.sizes[0] || 50) : 1}
        >
            <div
                class="terminal-container"
                bind:this={containerElement}
                on:click={handleContainerClick}
                on:keydown={() => {}}
                role="textbox"
                tabindex="0"
            ></div>
        </div>
        
        <!-- Additional split panes (only rendered when splits exist) -->
        {#if hasSplitPanes && splitLayout}
            {#each splitPanes as pane, index (pane.id)}
                <div 
                    class="split-resizer"
                    on:mousedown={(e) => handleSplitResizeStart(e, index)}
                    role="separator"
                    tabindex="-1"
                ></div>
                <div 
                    class="terminal-pane"
                    style="flex: {splitLayout.sizes[index + 1] || 50};"
                >
                    <SplitTerminalView {session} {pane} />
                </div>
            {/each}
        {/if}
    </div>

    <!-- Connection status - minimal, non-blocking -->
    {#if isConnecting}
        <div class="connection-status">
            <span class="rexec-logo">⌘</span>
            <span class="connection-text">rexec</span>
            <span class="connection-dots">...</span>
        </div>
    {:else if showConnectedIndicator}
        <div class="connection-status connected">
            <span class="rexec-logo">⌘</span>
            <span class="connection-text">rexec</span>
            <span class="connected-check">✓</span>
        </div>
    {/if}

    {#if isDisconnected}
        <div class="disconnected-overlay">
            <svg
                class="disconnected-icon"
                viewBox="0 0 16 16"
                fill="currentColor"
            >
                <path
                    d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767L8.982 1.566zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5zm.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"
                />
            </svg>
            <span>Disconnected</span>
            <button class="reconnect-btn" on:click={handleReconnect}>
                <svg
                    class="reconnect-icon"
                    viewBox="0 0 16 16"
                    fill="currentColor"
                >
                    <path
                        d="M11.534 7h3.932a.25.25 0 0 1 .192.41l-1.966 2.36a.25.25 0 0 1-.384 0l-1.966-2.36a.25.25 0 0 1 .192-.41zm-11 2h3.932a.25.25 0 0 0 .192-.41L2.692 6.23a.25.25 0 0 0-.384 0L.342 8.59A.25.25 0 0 0 .534 9z"
                    />
                    <path
                        fill-rule="evenodd"
                        d="M8 3c-1.552 0-2.94.707-3.857 1.818a.5.5 0 1 1-.771-.636A6.002 6.002 0 0 1 13.917 7H12.9A5.002 5.002 0 0 0 8 3zM3.1 9a5.002 5.002 0 0 0 8.757 2.182.5.5 0 1 1 .771.636A6.002 6.002 0 0 1 2.083 9H3.1z"
                    />
                </svg>
                Reconnect
            </button>
        </div>
    {/if}

    {#if isSettingUp}
        <div class="setup-overlay">
            <div class="setup-content">
                <div class="setup-spinner-large"></div>
                <span class="setup-title">Installing packages...</span>
                <span class="setup-detail">{setupMessage}</span>
            </div>
        </div>
    {/if}
    
    <!-- Download Modal -->
    {#if showDownloadModal}
        <div class="download-modal-overlay" on:click={() => showDownloadModal = false} on:keydown={(e) => e.key === 'Escape' && (showDownloadModal = false)} role="presentation">
            <div class="download-modal" on:click|stopPropagation role="dialog" aria-modal="true">
                <div class="download-modal-header">
                    <h3>Download File</h3>
                    <button class="close-btn" on:click={() => showDownloadModal = false}>×</button>
                </div>
                <div class="download-modal-body">
                    <label for="download-path">File Path</label>
                    <input 
                        type="text" 
                        id="download-path"
                        bind:value={downloadPath}
                        placeholder="/home/user/filename.txt"
                        on:keydown={(e) => e.key === 'Enter' && handleDownload()}
                    />
                    <p class="download-hint">Enter the full path to the file you want to download</p>
                </div>
                <div class="download-modal-footer">
                    <button class="btn-cancel" on:click={() => showDownloadModal = false}>Cancel</button>
                    <button class="btn-download" on:click={handleDownload}>Download</button>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .terminal-panel-wrapper {
        display: flex;
        flex-direction: column;
        height: 100%;
        width: 100%;
        position: relative;
        background: #0a0a0a;
    }

    /* Toolbar */
    .terminal-toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 6px 12px;
        background: #111;
        border-bottom: 1px solid var(--border);
        flex-shrink: 0;
        gap: 8px;
        overflow-x: auto;
        /* Hide scrollbar but allow scrolling */
        scrollbar-width: none;
        -ms-overflow-style: none;
        position: relative;
        z-index: 100;
    }

    .terminal-toolbar::-webkit-scrollbar {
        display: none;
    }

    .toolbar-left {
        display: flex;
        align-items: center;
        gap: 12px;
        flex-shrink: 0;
    }

    .terminal-name {
        font-size: 12px;
        color: var(--text);
        font-weight: 500;
    }

    .terminal-status {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 10px;
        text-transform: uppercase;
        padding: 2px 8px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
    }

    .status-indicator {
        width: 6px;
        height: 6px;
    }

    .terminal-status.connected {
        border-color: var(--green);
        color: var(--green);
    }

    .terminal-status.connected .status-indicator {
        background: var(--green);
    }

    .terminal-status.connecting {
        border-color: var(--yellow);
        color: var(--yellow);
    }

    .terminal-status.connecting .status-indicator {
        background: var(--yellow);
        animation: pulse 1s infinite;
    }

    .terminal-status.disconnected {
        border-color: var(--red);
        color: var(--red);
    }

    .terminal-status.disconnected .status-indicator {
        background: var(--red);
    }

    .terminal-stats {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 10px;
        text-transform: uppercase;
        padding: 2px 8px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    .stat-item {
        display: flex;
        align-items: center;
        gap: 4px;
        white-space: nowrap;
        transition: color 0.3s ease;
    }

    .stat-label {
        font-size: 9px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-muted);
        opacity: 0.7;
        background: rgba(255, 255, 255, 0.05);
        padding: 1px 4px;
        border-radius: 2px;
    }

    .stat-value {
        font-weight: 600;
        font-size: 11px;
    }

    .stat-limit {
        opacity: 0.5;
        font-size: 10px;
        font-weight: 400;
    }

    .view-only-badge {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 2px 8px;
        background: rgba(255, 170, 0, 0.15);
        border: 1px solid rgba(255, 170, 0, 0.3);
        color: #ffaa00;
        font-size: 9px;
        font-weight: 600;
        letter-spacing: 0.5px;
        text-transform: uppercase;
    }

    .view-only-badge svg {
        opacity: 0.8;
    }

    .stat-io {
        display: flex;
        gap: 6px;
    }

    .stat-io-item {
        font-size: 10px;
        font-weight: 500;
    }

    .stat-read, .stat-rx {
        color: #88ccff;
    }

    .stat-write, .stat-tx {
        color: #ffaa88;
    }

    .stat-icon {
        width: 10px;
        height: 10px;
        opacity: 0.8;
    }

    .stat-cpu .stat-label {
        background: rgba(68, 255, 68, 0.1);
        color: #88ff88;
    }

    .stat-mem .stat-label {
        background: rgba(136, 170, 255, 0.1);
        color: #aaccff;
    }

    .stat-disk .stat-label {
        background: rgba(255, 230, 109, 0.1);
        color: #ffe66d;
    }

    .stat-net .stat-label {
        background: rgba(162, 155, 254, 0.1);
        color: #a29bfe;
    }

    .stat-divider {
        opacity: 0.3;
        color: var(--text-muted);
    }

    .toolbar-actions {
        display: flex;
        gap: 4px;
        flex-shrink: 0;
    }

    .toolbar-btn {
        display: flex;
        align-items: center;
        gap: 4px;
        background: none;
        border: 1px solid transparent;
        color: var(--text-muted);
        font-size: 11px;
        font-family: var(--font-mono);
        padding: 4px 8px;
        cursor: pointer;
        transition: all 0.15s;
    }

    .toolbar-icon {
        width: 12px;
        height: 12px;
        flex-shrink: 0;
    }

    .toolbar-btn:hover:not(:disabled) {
        color: var(--text);
        background: var(--bg-tertiary);
        border-color: var(--border);
    }

    .toolbar-btn:disabled {
        cursor: default;
        opacity: 0.8;
    }

    .toolbar-btn.reconnect-btn {
        color: var(--red);
        border-color: var(--red);
        background: rgba(255, 0, 60, 0.1);
    }

    .toolbar-btn.reconnect-btn:hover {
        background: var(--red);
        color: var(--bg);
    }

    .toolbar-divider {
        width: 1px;
        height: 16px;
        background: var(--border);
        margin: 0 4px;
    }

    /* Recording Button */
    .toolbar-btn.recording {
        color: #ff4444;
        background: rgba(255, 68, 68, 0.1);
        border-color: #ff4444;
        animation: pulse-red 1.5s ease-in-out infinite;
    }

    .toolbar-btn.recording:hover {
        background: rgba(255, 68, 68, 0.2);
    }

    @keyframes pulse-red {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.7; }
    }

    /* Collaborate Button */
    .toolbar-btn.collab-btn:hover {
        color: var(--green);
        border-color: var(--green);
        background: rgba(0, 255, 136, 0.1);
    }
    
    /* Collab button with active participants - pulsing effect */
    .toolbar-btn.collab-btn.has-participants {
        color: var(--green);
        border-color: var(--green);
        background: rgba(0, 255, 136, 0.15);
        animation: collab-pulse 2s ease-in-out infinite;
        position: relative;
    }
    
    .toolbar-btn.collab-btn.has-participants::before {
        content: '';
        position: absolute;
        inset: -2px;
        border-radius: 4px;
        border: 1px solid var(--green);
        animation: collab-ring 2s ease-in-out infinite;
        pointer-events: none;
    }
    
    @keyframes collab-pulse {
        0%, 100% { 
            background: rgba(0, 255, 136, 0.15);
            box-shadow: 0 0 0 0 rgba(0, 255, 136, 0.4);
        }
        50% { 
            background: rgba(0, 255, 136, 0.25);
            box-shadow: 0 0 8px 2px rgba(0, 255, 136, 0.3);
        }
    }
    
    @keyframes collab-ring {
        0%, 100% { 
            opacity: 0.3;
            transform: scale(1);
        }
        50% { 
            opacity: 0.6;
            transform: scale(1.05);
        }
    }
    
    /* Participant count badge */
    .collab-badge {
        position: absolute;
        top: -4px;
        right: -4px;
        background: var(--green);
        color: var(--bg);
        font-size: 9px;
        font-weight: 700;
        min-width: 14px;
        height: 14px;
        border-radius: 7px;
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 0 3px;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
    }

    /* Upload Button */
    .toolbar-btn.upload-btn:hover {
        color: #4fc3f7;
        border-color: #4fc3f7;
        background: rgba(79, 195, 247, 0.1);
    }

    /* Download Button */
    .toolbar-btn.download-btn:hover {
        color: #81c784;
        border-color: #81c784;
        background: rgba(129, 199, 132, 0.1);
    }

    /* Split Button */
    .toolbar-btn.split-btn:hover {
        color: var(--cyan, #00d9ff);
        border-color: var(--cyan, #00d9ff);
        background: rgba(0, 217, 255, 0.1);
    }

    /* Icon-only buttons */
    .toolbar-btn.icon-btn {
        padding: 6px;
        min-width: 28px;
        justify-content: center;
        position: relative;
    }

    .toolbar-btn.icon-btn .toolbar-icon {
        width: 14px;
        height: 14px;
    }

    .toolbar-btn.icon-btn:hover {
        background: var(--bg-tertiary);
        border-color: var(--border);
        color: var(--text);
    }

    /* Share button highlight */
    .toolbar-btn.share-btn:hover {
        color: var(--accent);
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
    }

    /* More dropdown */
    .more-dropdown {
        position: static;
    }

    .more-btn:hover {
        color: var(--text);
    }

    .more-menu {
        position: fixed;
        min-width: 160px;
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: 6px;
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
        z-index: 10000;
        overflow: hidden;
        animation: menuFadeIn 0.15s ease;
    }

    @keyframes menuFadeIn {
        from {
            opacity: 0;
            transform: translateY(-4px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .menu-item {
        display: flex;
        align-items: center;
        gap: 10px;
        width: 100%;
        padding: 10px 14px;
        background: none;
        border: none;
        color: var(--text-secondary);
        font-size: 12px;
        font-family: var(--font-mono);
        cursor: pointer;
        text-align: left;
        transition: all 0.15s;
    }

    .menu-item:hover {
        background: var(--bg-tertiary);
        color: var(--text);
    }

    .menu-icon {
        width: 14px;
        height: 14px;
        flex-shrink: 0;
        opacity: 0.7;
    }

    .menu-item:hover .menu-icon {
        opacity: 1;
    }

    .menu-divider {
        height: 1px;
        background: var(--border);
        margin: 4px 0;
    }

    /* Terminal Wrapper - always present */
    .terminal-wrapper {
        flex: 1;
        display: flex;
        width: 100%;
        height: 0;
        min-height: 100px;
        overflow: hidden;
    }

    .terminal-wrapper.has-splits.horizontal {
        flex-direction: row;
    }

    .terminal-wrapper.has-splits.vertical {
        flex-direction: column;
    }

    .terminal-pane {
        display: flex;
        flex-direction: column;
        min-width: 100px;
        min-height: 50px;
        overflow: hidden;
    }

    .terminal-pane.main-pane {
        flex: 1;
    }

    .terminal-wrapper.has-splits .terminal-pane.main-pane {
        border-right: 1px solid var(--border);
    }

    .terminal-wrapper.has-splits.vertical .terminal-pane.main-pane {
        border-right: none;
        border-bottom: 1px solid var(--border);
    }

    .terminal-pane .terminal-container {
        flex: 1;
        height: 100%;
        min-height: 0;
    }

    .split-resizer {
        flex-shrink: 0;
        background: var(--border);
        transition: background 0.2s;
        cursor: col-resize;
        position: relative;
    }

    .split-resizer::before {
        content: '';
        position: absolute;
        inset: -4px;
    }

    .terminal-wrapper.has-splits.horizontal .split-resizer {
        width: 4px;
        cursor: col-resize;
    }

    .terminal-wrapper.has-splits.vertical .split-resizer {
        height: 4px;
        cursor: row-resize;
    }

    .split-resizer:hover,
    .terminal-wrapper.resizing .split-resizer {
        background: var(--accent);
    }

    .terminal-wrapper.resizing {
        user-select: none;
    }

    .terminal-wrapper.resizing .terminal-container,
    .terminal-wrapper.resizing .split-pane-terminal {
        pointer-events: none;
    }

    /* Terminal Container */
    .terminal-container {
        flex: 1;
        width: 100%;
        height: 0;
        min-height: 100px;
        overflow: hidden;
        padding: 8px;
    }

    .terminal-container:focus {
        outline: none;
    }

    .terminal-container :global(.xterm) {
        height: 100% !important;
        width: 100% !important;
    }

    .terminal-container :global(.xterm-viewport) {
        overflow-y: auto !important;
    }

    .terminal-container :global(.xterm-screen) {
        height: 100% !important;
        width: 100% !important;
    }

    /* Connection Status - minimal, non-blocking */
    .connection-status {
        position: absolute;
        top: 50px;
        left: 16px;
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        background: rgba(0, 0, 0, 0.6);
        border: 1px solid var(--border);
        border-radius: 4px;
        z-index: 5;
        animation: fadeIn 0.2s ease;
    }

    .rexec-logo {
        font-size: 14px;
        color: var(--accent);
    }

    .connection-text {
        font-size: 11px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    .connection-dots {
        font-size: 11px;
        color: var(--accent);
        animation: dots 1.4s steps(4, end) infinite;
    }

    @keyframes dots {
        0%, 20% { content: ''; opacity: 0.3; }
        40% { content: '.'; opacity: 0.6; }
        60% { content: '..'; opacity: 0.8; }
        80%, 100% { content: '...'; opacity: 1; }
    }

    .connection-status.connected {
        border-color: var(--green);
        animation: fadeInOut 2s ease forwards;
    }

    .connection-status.connected .rexec-logo {
        color: var(--green);
    }

    .connected-check {
        font-size: 12px;
        color: var(--green);
    }

    @keyframes fadeInOut {
        0% { opacity: 0; }
        20% { opacity: 1; }
        80% { opacity: 1; }
        100% { opacity: 0; }
    }

    /* Disconnected Overlay */
    .disconnected-overlay {
        position: absolute;
        bottom: 16px;
        left: 50%;
        transform: translateX(-50%);
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 16px;
        background: rgba(255, 0, 60, 0.1);
        border: 1px solid var(--red);
        z-index: 10;
    }

    .disconnected-icon {
        width: 20px;
        height: 20px;
        color: var(--red);
    }

    .reconnect-btn {
        display: flex;
        align-items: center;
        gap: 6px;
    }

    .reconnect-icon {
        width: 14px;
        height: 14px;
    }

    .disconnected-overlay span {
        color: var(--red);
        font-size: 12px;
    }

    .reconnect-btn {
        background: var(--red);
        border: none;
        color: var(--bg);
        font-size: 11px;
        font-family: var(--font-mono);
        padding: 4px 10px;
        cursor: pointer;
        transition: opacity 0.15s;
    }

    .reconnect-btn:hover {
        opacity: 0.9;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
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

    /* Setup Indicator */
    .setup-indicator {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 10px;
        text-transform: uppercase;
        padding: 2px 8px;
        background: rgba(0, 200, 255, 0.1);
        border: 1px solid var(--cyan, #00c8ff);
        color: var(--cyan, #00c8ff);
        animation: fadeIn 0.2s ease;
    }

    .setup-spinner {
        width: 8px;
        height: 8px;
        border: 1.5px solid rgba(0, 200, 255, 0.3);
        border-top-color: var(--cyan, #00c8ff);
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    /* Setup Overlay */
    .setup-overlay {
        position: absolute;
        bottom: 16px;
        right: 16px;
        z-index: 10;
        animation: fadeIn 0.2s ease;
    }

    .setup-content {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 16px;
        background: rgba(0, 200, 255, 0.1);
        border: 1px solid var(--cyan, #00c8ff);
        backdrop-filter: blur(4px);
    }

    .setup-spinner-large {
        width: 16px;
        height: 16px;
        border: 2px solid rgba(0, 200, 255, 0.3);
        border-top-color: var(--cyan, #00c8ff);
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    .setup-title {
        font-size: 12px;
        color: var(--cyan, #00c8ff);
        font-weight: 500;
    }

    .setup-detail {
        font-size: 11px;
        color: var(--text-muted);
        max-width: 200px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    @keyframes fadeIn {
        from {
            opacity: 0;
        }
        to {
            opacity: 1;
        }
    }

    /* Mobile responsive toolbar */
    @media (max-width: 768px) {
        .terminal-toolbar {
            padding: 8px 10px;
            gap: 10px;
        }

        .toolbar-left {
            gap: 8px;
            min-width: 0;
        }

        .terminal-name {
            max-width: 100px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .terminal-stats {
            padding: 2px 6px;
            font-size: 9px;
        }

        .toolbar-btn {
            padding: 6px 10px;
            font-size: 10px;
        }

        .toolbar-btn .btn-text {
            display: none;
        }

        .toolbar-icon {
            width: 14px;
            height: 14px;
        }
    }

    /* Mini spinner for upload button */
    .mini-spinner {
        width: 12px;
        height: 12px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }

    /* Download Modal */
    .download-modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
    }

    .download-modal {
        background: var(--bg-card);
        border: 1px solid var(--border);
        width: 100%;
        max-width: 400px;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
    }

    .download-modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px;
        border-bottom: 1px solid var(--border);
    }

    .download-modal-header h3 {
        margin: 0;
        font-size: 14px;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--text);
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

    .download-modal-body {
        padding: 20px;
    }

    .download-modal-body label {
        display: block;
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-muted);
        margin-bottom: 8px;
    }

    .download-modal-body input {
        width: 100%;
        padding: 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .download-modal-body input:focus {
        outline: none;
        border-color: var(--accent);
    }

    .download-hint {
        margin: 8px 0 0;
        font-size: 11px;
        color: var(--text-muted);
    }

    .download-modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        padding: 16px;
        border-top: 1px solid var(--border);
    }

    .btn-cancel, .btn-download {
        padding: 10px 20px;
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        border: 1px solid var(--border);
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-cancel {
        background: transparent;
        color: var(--text-secondary);
    }

    .btn-cancel:hover {
        background: var(--bg-secondary);
        color: var(--text);
    }

    .btn-download {
        background: var(--accent);
        border-color: var(--accent);
        color: #000;
    }

    .btn-download:hover {
        filter: brightness(1.1);
    }
</style>
