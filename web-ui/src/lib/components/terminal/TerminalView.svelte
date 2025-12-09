<script lang="ts">
    import { onMount, onDestroy, tick } from "svelte";
    import { terminal, sessionCount, isFloating, isFullscreen } from "$stores/terminal";
    import { toast } from "$stores/toast";
    import TerminalPanel from "./TerminalPanel.svelte";
    import InlineCreateTerminal from "../InlineCreateTerminal.svelte";
    import RecordingPanel from "../RecordingPanel.svelte";
    import CollabPanel from "../CollabPanel.svelte";
    import SnippetsModal from "../SnippetsModal.svelte";

    // Track view mode changes to force terminal re-render
    let viewModeKey = 0;
    
    // Panel states
    let showRecordingsPanel = false;
    let showCollabPanel = false;
    let showSnippetsModal = false;
    let panelContainerId: string | null = null;
    
    function handleOpenRecordings(e: CustomEvent<{containerId: string}>) {
        panelContainerId = e.detail.containerId;
        showRecordingsPanel = true;
        showCollabPanel = false;
        showSnippetsModal = false;
    }
    
    function handleOpenCollab(e: CustomEvent<{containerId: string}>) {
        panelContainerId = e.detail.containerId;
        showCollabPanel = true;
        showRecordingsPanel = false;
        showSnippetsModal = false;
    }

    function handleOpenSnippets(e: CustomEvent<{containerId: string}>) {
        panelContainerId = e.detail.containerId;
        showSnippetsModal = true;
        showRecordingsPanel = false;
        showCollabPanel = false;
    }

    function handleRunSnippet(e: CustomEvent<{snippet: any}>) {
        const snippet = e.detail.snippet;
        if (panelContainerId && snippet) {
            // Find session by container ID
            const session = Array.from($terminal.sessions.values()).find(s => s.containerId === panelContainerId);
            if (session) {
                // Send the snippet content to the terminal
                terminal.sendInput(session.id, snippet.content);
                // Auto-submit if it doesn't end with newline
                if (!snippet.content.endsWith('\n')) {
                    terminal.sendInput(session.id, '\n');
                }
            }
        }
    }
    
    function closePanels() {
        showRecordingsPanel = false;
        showCollabPanel = false;
        showSnippetsModal = false;
        panelContainerId = null;
    }

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

    // Docked resize state
    let isResizingDocked = false;
    let dockedResizeStartY = 0;
    let dockedResizeStartHeight = 0;

    // Tab drag-out state
    let draggingTabId: string | null = null;
    let tabDragStart: { x: number; y: number } | null = null;
    let isDraggingTab = false;

    // Get state from store
    $: isMinimized = $terminal.isMinimized;
    $: floatingPosition = $terminal.floatingPosition;
    $: floatingSize = $terminal.floatingSize;
    $: dockedHeight = $terminal.dockedHeight;
    $: sessions = Array.from($terminal.sessions.entries());
    $: dockedSessions = sessions.filter(([_, s]) => !s.isDetached);
    $: detachedSessions = sessions.filter(([_, s]) => s.isDetached);
    $: activeId = $terminal.activeSessionId;

    // Inline create terminal state
    let showCreatePanel = false;



    function openCreatePanel() {
        showCreatePanel = true;
    }

    function closeCreatePanel() {
        showCreatePanel = false;
    }

    function handleInlineCreated(id: string, name: string) {
        showCreatePanel = false;
        terminal.createSession(id, name);
        toast.success(`Created and connected to ${name}`);
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

        // Handle docked resize
        if (isResizingDocked) {
            const deltaY = dockedResizeStartY - event.clientY;
            const deltaVh = (deltaY / window.innerHeight) * 100;
            const newHeight = dockedResizeStartHeight + deltaVh;
            terminal.setDockedHeight(newHeight);
        }
    }

    function handleMouseUp() {
        if (isDragging || isResizing) {
            isDragging = false;
            isResizing = false;
            // Fit terminals after resize
            setTimeout(() => terminal.fitAll(), 50);
        }
        if (isResizingDocked) {
            isResizingDocked = false;
            setTimeout(() => terminal.fitAll(), 50);
        }
    }

    function handleResizeStart(event: MouseEvent) {
        event.preventDefault();
        event.stopPropagation();
        isResizing = true;
    }

    // Docked resize handlers
    function handleDockedResizeStart(event: MouseEvent) {
        event.preventDefault();
        event.stopPropagation();
        isResizingDocked = true;
        dockedResizeStartY = event.clientY;
        dockedResizeStartHeight = dockedHeight;
    }

    function handleDockedTouchStart(event: TouchEvent) {
        event.preventDefault();
        isResizingDocked = true;
        dockedResizeStartY = event.touches[0].clientY;
        dockedResizeStartHeight = dockedHeight;
    }

    function handleDockedTouchMove(event: TouchEvent) {
        if (!isResizingDocked) return;
        const deltaY = dockedResizeStartY - event.touches[0].clientY;
        const deltaVh = (deltaY / window.innerHeight) * 100;
        const newHeight = dockedResizeStartHeight + deltaVh;
        terminal.setDockedHeight(newHeight);
    }

    function handleDockedTouchEnd() {
        if (isResizingDocked) {
            isResizingDocked = false;
            setTimeout(() => terminal.fitAll(), 50);
        }
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

    // Toggle fullscreen mode
    function toggleFullscreen() {
        terminal.toggleFullscreen();
        viewModeKey++;
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

    // Handle container deletion - close any associated terminal sessions
    function handleContainerDeleted(event: Event) {
        const customEvent = event as CustomEvent<{ containerId: string }>;
        const containerId = customEvent.detail?.containerId;
        if (containerId) {
            // Close all sessions for this container
            for (const [sessionId, session] of $terminal.sessions) {
                if (session.containerId === containerId) {
                    terminal.closeSession(sessionId);
                }
            }
        }
    }

    // Global keyboard shortcuts
    function handleGlobalKeydown(event: KeyboardEvent) {
        // Detect platform
        const isMac = /Mac|iPod|iPhone|iPad/.test(navigator.platform || '') || 
                      /Macintosh/.test(navigator.userAgent);
        
        // Use Cmd on Mac, Ctrl on Windows/Linux for app shortcuts
        const modKey = isMac ? event.metaKey : event.ctrlKey;
        const otherModKey = isMac ? event.ctrlKey : event.metaKey;
        
        // Handle app-level shortcuts (Cmd/Ctrl+Key)
        if (modKey && !otherModKey && !event.altKey) {
            const key = event.key.toLowerCase();
            const isShift = event.shiftKey;

            // Cmd/Ctrl+D / Cmd/Ctrl+Shift+D: Split Pane
            if (key === 'd' && activeId) {
                event.preventDefault();
                event.stopPropagation();
                // Shift = Horizontal (Top/Bottom), No shift = Vertical (Left/Right)
                const direction = isShift ? 'horizontal' : 'vertical';
                terminal.splitPane(activeId, direction);
                return;
            }

            // Cmd/Ctrl+T: New Tab (Inline Create)
            if (key === 't') {
                event.preventDefault();
                event.stopPropagation();
                openCreatePanel();
                return;
            }

            // Cmd/Ctrl+W: Close Pane/Tab
            if (key === 'w' && activeId) {
                event.preventDefault();
                event.stopPropagation();
                const session = $terminal.sessions.get(activeId);
                if (session && session.activePaneId && session.activePaneId !== 'main') {
                    terminal.closeSplitPane(activeId, session.activePaneId);
                } else {
                    closeSession(activeId);
                }
                return;
            }
            
            // Cmd/Ctrl+N: New Window (Pop out) - only on Mac to avoid browser new window
            if (isMac && key === 'n' && activeId) {
                event.preventDefault();
                event.stopPropagation();
                popOutTerminal(activeId, window.innerWidth / 2 - 300, window.innerHeight / 2 - 200);
                return;
            }

            // Cmd+. (Mac only): Send Ctrl+C
            if (isMac && key === '.' && activeId) {
                event.preventDefault();
                event.stopPropagation();
                terminal.sendCtrlC(activeId);
                return;
            }

            // Cmd/Ctrl+Arrows: Navigate Panes
            if (activeId && $terminal.sessions.get(activeId)?.splitPanes?.size > 0 && !isShift) {
                let direction: 'left' | 'right' | 'up' | 'down' | null = null;
                switch (event.key) {
                    case 'ArrowLeft':
                        direction = 'left';
                        break;
                    case 'ArrowRight':
                        direction = 'right';
                        break;
                    case 'ArrowUp':
                        direction = 'up';
                        break;
                    case 'ArrowDown':
                        direction = 'down';
                        break;
                }
                if (direction) {
                    event.preventDefault();
                    event.stopPropagation();
                    terminal.navigateSplitPanes(activeId, direction);
                    return;
                }
            }
        }

        // Only handle Alt key combinations to avoid conflict with browser/terminal
        if (!event.altKey) return;

        // Number keys 1-9 for tab switching
        if (event.key >= '1' && event.key <= '9') {
            const index = parseInt(event.key) - 1;
            if (index >= 0 && index < dockedSessions.length) {
                event.preventDefault();
                setActive(dockedSessions[index][0]);
            }
            return;
        }

        switch (event.key.toLowerCase()) {
            case 'd': // Alt+D: Toggle Dock/Float
                event.preventDefault();
                toggleView();
                break;
            case 'f': // Alt+F: Toggle Fullscreen
                event.preventDefault();
                toggleFullscreen();
                break;
            case 'm': // Alt+M: Minimize
                event.preventDefault();
                if (isMinimized) restore(); else minimize();
                break;
        }
    }

    // Window event listeners
    onMount(() => {
        // Listen for container deletions
        window.addEventListener("container-deleted", handleContainerDeleted);
        window.addEventListener("keydown", handleGlobalKeydown);

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
        window.removeEventListener("keydown", handleGlobalKeydown);
    });
</script>

{#if $sessionCount > 0}
    {#if $isFullscreen}
        <!-- Fullscreen Terminal -->
        <div class="fullscreen-terminal">
            <!-- Header -->
            <div class="fullscreen-header">
                <div class="fullscreen-tabs">
                    {#each dockedSessions as [id, session] (id)}
                        <button
                            class="fullscreen-tab"
                            class:active={id === activeId && !showCreatePanel}
                            onclick={() => {
                                showCreatePanel = false;
                                setActive(id);
                            }}
                        >
                            <span class="status-dot {getStatusClass(session.status)}"></span>
                            <span class="tab-name">{session.name}</span>
                            <span
                                class="tab-close"
                                role="button"
                                tabindex="0"
                                onclick={(e) => { e.stopPropagation(); closeSession(id); }}
                                onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); closeSession(id); } }}
                                title="Close terminal"
                                aria-label="Close {session.name}"
                            >
                                ×
                            </span>
                        </button>
                    {/each}
                    <button
                        class="fullscreen-tab new-tab-btn"
                        class:active={showCreatePanel}
                        onclick={openCreatePanel}
                        title="New Terminal"
                    >
                        +
                    </button>
                </div>

                <div class="fullscreen-actions">
                    <button
                        class="btn btn-secondary btn-sm btn-icon share-btn"
                        onclick={() => {
                            if (activeId) {
                                const session = $terminal.sessions.get(activeId);
                                if (session) {
                                    const url = `${window.location.origin}/terminal/${session.containerId}`;
                                    navigator.clipboard.writeText(url);
                                    toast.success("Terminal link copied!");
                                }
                            }
                        }}
                        title="Share Terminal"
                        disabled={!activeId}
                    >
                        <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                            <path d="M13.5 1a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM11 2.5a2.5 2.5 0 1 1 .603 1.628l-6.718 3.12a2.499 2.499 0 0 1 0 1.504l6.718 3.12a2.5 2.5 0 1 1-.488.876l-6.718-3.12a2.5 2.5 0 1 1 0-3.256l6.718-3.12A2.5 2.5 0 0 1 11 2.5zm-8.5 4a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zm11 5.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z"/>
                        </svg>
                    </button>
                    <button
                        class="btn btn-secondary btn-sm btn-icon"
                        onclick={() => activeId && popOutTerminal(activeId, window.innerWidth / 2 - 300, window.innerHeight / 2 - 200)}
                        title="Float window"
                        disabled={!activeId}
                    >
                        <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                            <path d="M5.5 0a.5.5 0 0 1 .5.5v4A1.5 1.5 0 0 1 4.5 6h-4a.5.5 0 0 1 0-1h4a.5.5 0 0 0 .5-.5v-4a.5.5 0 0 1 .5-.5zm5 0a.5.5 0 0 1 .5.5v4a.5.5 0 0 0 .5.5h4a.5.5 0 0 1 0 1h-4A1.5 1.5 0 0 1 10 4.5v-4a.5.5 0 0 1 .5-.5zM0 10.5a.5.5 0 0 1 .5-.5h4A1.5 1.5 0 0 1 6 11.5v4a.5.5 0 0 1-1 0v-4a.5.5 0 0 0-.5-.5h-4a.5.5 0 0 1-.5-.5zm10 1a1.5 1.5 0 0 1 1.5-1.5h4a.5.5 0 0 1 0 1h-4a.5.5 0 0 0-.5.5v4a.5.5 0 0 1-1 0v-4z"/>
                        </svg>
                    </button>
                    <button
                        class="btn btn-secondary btn-sm btn-icon"
                        onclick={toggleFullscreen}
                        title="Exit Fullscreen"
                    >
                        <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                            <path d="M5.5 0a.5.5 0 0 1 .5.5v4A1.5 1.5 0 0 1 4.5 6h-4a.5.5 0 0 1 0-1h4a.5.5 0 0 0 .5-.5v-4a.5.5 0 0 1 .5-.5zm5 0a.5.5 0 0 1 .5.5v4a.5.5 0 0 0 .5.5h4a.5.5 0 0 1 0 1h-4A1.5 1.5 0 0 1 10 4.5v-4a.5.5 0 0 1 .5-.5zM0 10.5a.5.5 0 0 1 .5-.5h4A1.5 1.5 0 0 1 6 11.5v4a.5.5 0 0 1-1 0v-4a.5.5 0 0 0-.5-.5h-4a.5.5 0 0 1-.5-.5zm10 1a1.5 1.5 0 0 1 1.5-1.5h4a.5.5 0 0 1 0 1h-4a.5.5 0 0 0-.5.5v4a.5.5 0 0 1-1 0v-4z"/>
                        </svg>
                    </button>
                    <button
                        class="btn btn-danger btn-sm btn-icon"
                        onclick={() => activeId && closeSession(activeId)}
                        title="Close Terminal"
                        disabled={!activeId}
                    >
                        <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                            <path d="M2.146 2.854a.5.5 0 1 1 .708-.708L8 7.293l5.146-5.147a.5.5 0 0 1 .708.708L8.707 8l5.147 5.146a.5.5 0 0 1-.708.708L8 8.707l-5.146 5.147a.5.5 0 0 1-.708-.708L7.293 8 2.146 2.854Z"/>
                        </svg>
                    </button>
                </div>
            </div>

            <!-- Body -->
            <div class="fullscreen-body">
                {#if showCreatePanel}
                    <div class="create-panel fullscreen-create">
                        <div class="create-panel-header">
                            <h3>New Terminal</h3>
                            <button class="close-create" onclick={closeCreatePanel}>× Cancel</button>
                        </div>
                        <InlineCreateTerminal
                            compact={false}
                            on:created={(e) => handleInlineCreated(e.detail.id, e.detail.name)}
                            on:cancel={closeCreatePanel}
                        />
                    </div>
                {:else}
                    {#each dockedSessions as [id, session] (`full-${viewModeKey}-${id}`)}
                        <div class="terminal-panel" class:active={id === activeId}>
                            <TerminalPanel {session} on:openRecordings={handleOpenRecordings} on:openCollab={handleOpenCollab} on:openSnippets={handleOpenSnippets} />
                        </div>
                    {/each}
                {/if}
            </div>
        </div>
    {:else if $isFloating}
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
                    onmousedown={handleMouseDown}
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
                                onclick={() => {
                                    showCreatePanel = false;
                                    setActive(id);
                                }}
                                onmousedown={(e) => handleTabDragStart(e, id)}
                                title="Drag out to pop to new window"
                            >
                                <span
                                    class="status-dot {getStatusClass(
                                        session.status,
                                    )}"
                                ></span>
                                <span class="tab-name">{session.name}</span>
                                <span
                                    class="tab-close"
                                    role="button"
                                    tabindex="0"
                                    onclick={(e) => { e.stopPropagation(); closeSession(id); }}
                                    onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); closeSession(id); } }}
                                    title="Close terminal"
                                    aria-label="Close {session.name}"
                                >
                                    ×
                                </span>
                            </button>
                        {/each}
                        <button
                            class="new-tab-btn"
                            class:active={showCreatePanel}
                            onclick={openCreatePanel}
                            title="New Terminal"
                        >
                            +
                        </button>
                    </div>

                    <div class="floating-actions">
                        <button 
                            class="float-action-btn share-btn"
                            onclick={() => {
                                if (activeId) {
                                    const session = $terminal.sessions.get(activeId);
                                    if (session) {
                                        const url = `${window.location.origin}/terminal/${session.containerId}`;
                                        navigator.clipboard.writeText(url);
                                        toast.success("Terminal link copied!");
                                    }
                                }
                            }} 
                            title="Share Terminal"
                            disabled={!activeId}
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
                                <path d="M13.5 1a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM11 2.5a2.5 2.5 0 1 1 .603 1.628l-6.718 3.12a2.499 2.499 0 0 1 0 1.504l6.718 3.12a2.5 2.5 0 1 1-.488.876l-6.718-3.12a2.5 2.5 0 1 1 0-3.256l6.718-3.12A2.5 2.5 0 0 1 11 2.5zm-8.5 4a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zm11 5.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z"/>
                            </svg>
                        </button>
                        <button 
                            class="float-action-btn"
                            onclick={() => activeId && popOutTerminal(activeId, floatingPosition.x + 100, floatingPosition.y + 100)} 
                            title="Pop out"
                            disabled={!activeId}
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
                                <path d="M5.5 0a.5.5 0 0 1 .5.5v4A1.5 1.5 0 0 1 4.5 6h-4a.5.5 0 0 1 0-1h4a.5.5 0 0 0 .5-.5v-4a.5.5 0 0 1 .5-.5zm5 0a.5.5 0 0 1 .5.5v4a.5.5 0 0 0 .5.5h4a.5.5 0 0 1 0 1h-4A1.5 1.5 0 0 1 10 4.5v-4a.5.5 0 0 1 .5-.5zM0 10.5a.5.5 0 0 1 .5-.5h4A1.5 1.5 0 0 1 6 11.5v4a.5.5 0 0 1-1 0v-4a.5.5 0 0 0-.5-.5h-4a.5.5 0 0 1-.5-.5zm10 1a1.5 1.5 0 0 1 1.5-1.5h4a.5.5 0 0 1 0 1h-4a.5.5 0 0 0-.5.5v4a.5.5 0 0 1-1 0v-4z"/>
                            </svg>
                        </button>
                        <button class="float-action-btn" onclick={toggleView} title="Dock window">
                            <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
                                <path d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"/>
                                <path d="M6 11.5a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm-2-3a.5.5 0 0 1 .5-.5h7a.5.5 0 0 1 0 1h-7a.5.5 0 0 1-.5-.5zm-2-3a.5.5 0 0 1 .5-.5h11a.5.5 0 0 1 0 1h-11a.5.5 0 0 1-.5-.5z"/>
                            </svg>
                        </button>
                        <button class="float-action-btn" onclick={minimize} title="Minimize">
                            <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
                                <path d="M2 8a.5.5 0 0 1 .5-.5h11a.5.5 0 0 1 0 1h-11A.5.5 0 0 1 2 8Z"/>
                            </svg>
                        </button>
                        <button class="float-action-btn" onclick={toggleFullscreen} title="Fullscreen">
                            <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
                                <path d="M1.5 1a.5.5 0 0 0-.5.5v4a.5.5 0 0 1-1 0v-4A1.5 1.5 0 0 1 1.5 0h4a.5.5 0 0 1 0 1h-4zM10 .5a.5.5 0 0 1 .5-.5h4A1.5 1.5 0 0 1 16 1.5v4a.5.5 0 0 1-1 0v-4a.5.5 0 0 0-.5-.5h-4a.5.5 0 0 1-.5-.5zM.5 10a.5.5 0 0 1 .5.5v4a.5.5 0 0 0 .5.5h4a.5.5 0 0 1 0 1h-4A1.5 1.5 0 0 1 0 14.5v-4a.5.5 0 0 1 .5-.5zm15 0a.5.5 0 0 1 .5.5v4a1.5 1.5 0 0 1-1.5 1.5h-4a.5.5 0 0 1 0-1h4a.5.5 0 0 0 .5-.5v-4a.5.5 0 0 1 .5-.5z"/>
                            </svg>
                        </button>
                        <button
                            class="float-action-btn close-btn"
                            onclick={() => activeId && closeSession(activeId)}
                            title="Close Terminal"
                            disabled={!activeId}
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
                                <path d="M2.146 2.854a.5.5 0 1 1 .708-.708L8 7.293l5.146-5.147a.5.5 0 0 1 .708.708L8.707 8l5.147 5.146a.5.5 0 0 1-.708.708L8 8.707l-5.146 5.147a.5.5 0 0 1-.708-.708L7.293 8 2.146 2.854Z"/>
                            </svg>
                        </button>
                    </div>
                </div>

                <!-- Body -->
                <div class="floating-body">
                    {#if showCreatePanel}
                        <!-- Inline Create Panel -->
                        <div class="create-panel">
                            <div class="create-panel-header">
                                <h3>New Terminal</h3>
                                <button
                                    class="close-create"
                                    onclick={closeCreatePanel}>×</button
                                >
                            </div>
                            <InlineCreateTerminal
                                compact={true}
                                on:created={(e) => handleInlineCreated(e.detail.id, e.detail.name)}
                                on:cancel={closeCreatePanel}
                            />
                        </div>
                    {:else}
                        {#each dockedSessions as [id, session] (`float-${viewModeKey}-${id}`)}
                            <div
                                class="terminal-panel"
                                class:active={id === activeId}
                            >
                                <TerminalPanel {session} on:openRecordings={handleOpenRecordings} on:openCollab={handleOpenCollab} on:openSnippets={handleOpenSnippets} />
                            </div>
                        {/each}
                    {/if}
                </div>

                <!-- Resize Handle -->
                <div
                    class="resize-handle"
                    onmousedown={handleResizeStart}
                    role="separator"
                    tabindex="-1"
                    onkeydown={() => {}}
                ></div>
            </div>
        </div>

        <!-- Minimized bar -->
        {#if isMinimized}
            <div class="minimized-bar">
                <button class="restore-btn" onclick={restore}>
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
                <button class="restore-btn" onclick={restore}>
                    <span class="restore-icon">↑</span>
                    <span
                        >{$sessionCount} Terminal{$sessionCount > 1
                            ? "s"
                            : ""}</span
                    >
                </button>
            </div>
        {:else}
            <div class="docked-terminal" style="height: {dockedHeight}vh;">
                <!-- Resize Handle at Top -->
                <div 
                    class="docked-resize-handle"
                    onmousedown={handleDockedResizeStart}
                    ontouchstart={handleDockedTouchStart}
                    ontouchmove={handleDockedTouchMove}
                    ontouchend={handleDockedTouchEnd}
                    role="separator"
                    aria-orientation="horizontal"
                    aria-valuenow={dockedHeight}
                    aria-valuemin={20}
                    aria-valuemax={90}
                    aria-label="Resize terminal panel"
                    tabindex="-1"
                    title="Drag to resize"
                >
                    <div class="resize-grip"></div>
                </div>
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
                                onclick={() => {
                                    showCreatePanel = false;
                                    setActive(id);
                                }}
                                onmousedown={(e) => handleTabDragStart(e, id)}
                                title="Drag out to pop to new window"
                            >
                                <span
                                    class="status-dot {getStatusClass(
                                        session.status,
                                    )}"
                                ></span>
                                <span class="tab-name">{session.name}</span>
                                <span
                                    class="tab-close"
                                    role="button"
                                    tabindex="0"
                                    onclick={(e) => { e.stopPropagation(); closeSession(id); }}
                                    onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); closeSession(id); } }}
                                    title="Close terminal"
                                    aria-label="Close {session.name}"
                                >
                                    ×
                                </span>
                            </button>
                        {/each}
                        <button
                            class="docked-tab new-tab-btn"
                            class:active={showCreatePanel}
                            onclick={openCreatePanel}
                            title="New Terminal"
                        >
                            +
                        </button>
                    </div>

                    <div class="docked-spacer"></div>

                    <div class="docked-actions">
                        <button
                            class="btn btn-secondary btn-sm btn-icon share-btn"
                            onclick={() => {
                                if (activeId) {
                                    const session = $terminal.sessions.get(activeId);
                                    if (session) {
                                        const url = `${window.location.origin}/terminal/${session.containerId}`;
                                        navigator.clipboard.writeText(url);
                                        toast.success("Terminal link copied!");
                                    }
                                }
                            }}
                            title="Share Terminal"
                            disabled={!activeId}
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                                <path d="M13.5 1a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zM11 2.5a2.5 2.5 0 1 1 .603 1.628l-6.718 3.12a2.499 2.499 0 0 1 0 1.504l6.718 3.12a2.5 2.5 0 1 1-.488.876l-6.718-3.12a2.5 2.5 0 1 1 0-3.256l6.718-3.12A2.5 2.5 0 0 1 11 2.5zm-8.5 4a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zm11 5.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z"/>
                            </svg>
                        </button>
                        <button
                            class="btn btn-secondary btn-sm btn-icon"
                            onclick={() => activeId && popOutTerminal(activeId, window.innerWidth / 2 - 300, window.innerHeight / 2 - 200)}
                            title="Float window"
                            disabled={!activeId}
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                                <path d="M5.5 0a.5.5 0 0 1 .5.5v4A1.5 1.5 0 0 1 4.5 6h-4a.5.5 0 0 1 0-1h4a.5.5 0 0 0 .5-.5v-4a.5.5 0 0 1 .5-.5zm5 0a.5.5 0 0 1 .5.5v4a.5.5 0 0 0 .5.5h4a.5.5 0 0 1 0 1h-4A1.5 1.5 0 0 1 10 4.5v-4a.5.5 0 0 1 .5-.5zM0 10.5a.5.5 0 0 1 .5-.5h4A1.5 1.5 0 0 1 6 11.5v4a.5.5 0 0 1-1 0v-4a.5.5 0 0 0-.5-.5h-4a.5.5 0 0 1-.5-.5zm10 1a1.5 1.5 0 0 1 1.5-1.5h4a.5.5 0 0 1 0 1h-4a.5.5 0 0 0-.5.5v4a.5.5 0 0 1-1 0v-4z"/>
                            </svg>
                        </button>
                        <button
                            class="btn btn-secondary btn-sm btn-icon"
                            onclick={toggleView}
                            title="Toggle dock/float"
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                                <path d="M14 1a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1h12zM2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H2z"/>
                                <path d="M6 11.5a.5.5 0 0 1 .5-.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1-.5-.5zm-2-3a.5.5 0 0 1 .5-.5h7a.5.5 0 0 1 0 1h-7a.5.5 0 0 1-.5-.5zm-2-3a.5.5 0 0 1 .5-.5h11a.5.5 0 0 1 0 1h-11a.5.5 0 0 1-.5-.5z"/>
                            </svg>
                        </button>
                        <button
                            class="btn btn-secondary btn-sm btn-icon"
                            onclick={minimize}
                            title="Minimize"
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                                <path d="M2 8a.5.5 0 0 1 .5-.5h11a.5.5 0 0 1 0 1h-11A.5.5 0 0 1 2 8Z"/>
                            </svg>
                        </button>
                        <button
                            class="btn btn-secondary btn-sm btn-icon"
                            onclick={toggleFullscreen}
                            title="Fullscreen"
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                                <path d="M1.5 1a.5.5 0 0 0-.5.5v4a.5.5 0 0 1-1 0v-4A1.5 1.5 0 0 1 1.5 0h4a.5.5 0 0 1 0 1h-4zM10 .5a.5.5 0 0 1 .5-.5h4A1.5 1.5 0 0 1 16 1.5v4a.5.5 0 0 1-1 0v-4a.5.5 0 0 0-.5-.5h-4a.5.5 0 0 1-.5-.5zM.5 10a.5.5 0 0 1 .5.5v4a.5.5 0 0 0 .5.5h4a.5.5 0 0 1 0 1h-4A1.5 1.5 0 0 1 0 14.5v-4a.5.5 0 0 1 .5-.5zm15 0a.5.5 0 0 1 .5.5v4a1.5 1.5 0 0 1-1.5 1.5h-4a.5.5 0 0 1 0-1h4a.5.5 0 0 0 .5-.5v-4a.5.5 0 0 1 .5-.5z"/>
                            </svg>
                        </button>
                        <button
                            class="btn btn-danger btn-sm btn-icon"
                            onclick={() => activeId && closeSession(activeId)}
                            title="Close Terminal"
                            disabled={!activeId}
                        >
                            <svg viewBox="0 0 16 16" fill="currentColor" width="14" height="14">
                                <path d="M2.146 2.854a.5.5 0 1 1 .708-.708L8 7.293l5.146-5.147a.5.5 0 0 1 .708.708L8.707 8l5.147 5.146a.5.5 0 0 1-.708.708L8 8.707l-5.146 5.147a.5.5 0 0 1-.708-.708L7.293 8 2.146 2.854Z"/>
                            </svg>
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
                                <button
                                    class="close-create"
                                    onclick={closeCreatePanel}
                                    >× Cancel</button
                                >
                            </div>
                            <InlineCreateTerminal
                                compact={false}
                                on:created={(e) => handleInlineCreated(e.detail.id, e.detail.name)}
                                on:cancel={closeCreatePanel}
                            />
                        </div>
                    {:else}
                        {#each dockedSessions as [id, session] (`dock-${viewModeKey}-${id}`)}
                            <div
                                class="terminal-panel"
                                class:active={id === activeId}
                            >
                                <TerminalPanel {session} on:openRecordings={handleOpenRecordings} on:openCollab={handleOpenCollab} on:openSnippets={handleOpenSnippets} />
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
        onmousedown={() => terminal.bringToFront(id)}
        role="dialog"
        tabindex="-1"
    >
        <div
            class="detached-header"
            onmousedown={(e) => handleDetachedMouseDown(e, id)}
            on:dblclick={() => dockSession(id)}
            role="toolbar"
            tabindex="-1"
        >
            <span class="detached-title">{session.name}</span>
            <span class="detached-status status-{session.status}"></span>
            <div class="detached-actions">
                <button
                    onclick={() => dockSession(id)}
                    title="Dock back to terminal panel"
                >
                    ⬒
                </button>
                <button
                    onclick={() => {
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
            <TerminalPanel {session} on:openRecordings={handleOpenRecordings} on:openCollab={handleOpenCollab} on:openSnippets={handleOpenSnippets} />
        </div>
        <div
            class="detached-resize-handle"
            onmousedown={(e) => handleDetachedResizeStart(e, id)}
            role="button"
            tabindex="-1"
        ></div>
    </div>
{/each}

<!-- Recordings Panel Modal -->
{#if panelContainerId}
    <RecordingPanel containerId={panelContainerId} isOpen={showRecordingsPanel} on:close={closePanels} />
{/if}

<!-- Collab Panel Modal -->
{#if panelContainerId}
    <CollabPanel containerId={panelContainerId} isOpen={showCollabPanel} on:close={closePanels} />
{/if}

<!-- Snippets Modal -->
{#if panelContainerId}
    <SnippetsModal 
        containerId={panelContainerId} 
        show={showSnippetsModal} 
        on:close={closePanels} 
        on:run={handleRunSnippet} 
    />
{/if}

<style>
    /* Fullscreen Terminal */
    .fullscreen-terminal {
        position: fixed;
        inset: 0;
        z-index: 9999;
        display: flex;
        flex-direction: column;
        background: var(--bg);
    }

    .fullscreen-header {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 8px 16px;
        background: #0a0a0a;
        border-bottom: 1px solid var(--border);
        flex-shrink: 0;
    }

    .fullscreen-tabs {
        display: flex;
        gap: 4px;
        overflow-x: auto;
        flex: 1;
        scrollbar-width: none;
        -ms-overflow-style: none;
    }

    .fullscreen-tabs::-webkit-scrollbar {
        display: none;
    }

    .fullscreen-tab {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 16px;
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

    .fullscreen-tab:hover {
        background: var(--bg-secondary);
        color: var(--text);
    }

    .fullscreen-tab.active {
        background: var(--bg);
        border-color: var(--accent);
        border-bottom-color: var(--bg);
        color: var(--text);
    }

    .fullscreen-tab.new-tab-btn {
        background: transparent;
        border: 1px dashed var(--border);
        color: var(--text-muted);
        padding: 6px 12px;
        border-radius: 4px;
        font-size: 14px;
        cursor: pointer;
        transition: all 0.15s ease;
        margin-left: 4px;
    }

    .fullscreen-tab.new-tab-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
    }

    .fullscreen-actions {
        display: flex;
        gap: 4px;
        flex-shrink: 0;
    }

    .fullscreen-actions .btn-icon {
        padding: 6px !important;
    }

    .fullscreen-body {
        flex: 1;
        position: relative;
        overflow: hidden;
        background: #050505;
    }

    .fullscreen-create {
        max-width: 1200px;
        margin: 0 auto;
        width: 100%;
    }

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
        gap: 2px;
        align-items: center;
    }

    .floating-actions button,
    .float-action-btn {
        background: none;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 4px 6px;
        font-size: 12px;
        font-family: var(--font-mono);
        transition: all 0.15s ease;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 3px;
    }

    .floating-actions button:hover,
    .float-action-btn:hover {
        color: var(--text);
        background: rgba(255, 255, 255, 0.1);
    }

    .float-action-btn.share-btn:hover {
        color: var(--accent);
        background: rgba(0, 255, 65, 0.15);
    }

    .float-action-btn.close-btn:hover {
        color: var(--red);
        background: rgba(255, 0, 60, 0.15);
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

    /* Docked Resize Handle */
    .docked-resize-handle {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 8px;
        cursor: ns-resize;
        background: transparent;
        z-index: 10;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: background 0.2s;
    }

    .docked-resize-handle:hover {
        background: rgba(0, 255, 65, 0.1);
    }

    .docked-resize-handle:active {
        background: rgba(0, 255, 65, 0.2);
    }

    .docked-resize-handle .resize-grip {
        width: 40px;
        height: 4px;
        background: var(--border);
        border-radius: 2px;
        transition: background 0.2s;
    }

    .docked-resize-handle:hover .resize-grip {
        background: var(--accent);
    }

    /* Docked Terminal Container */
    .docked-terminal {
        position: fixed;
        bottom: 0;
        left: 0;
        right: 0;
        /* height is now set dynamically via style attribute */
        min-height: 150px;
        max-height: 90vh;
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
            height: 100vh !important;
            top: 0;
            border-top: none;
        }

        .docked-resize-handle {
            display: none;
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
        padding: 8px 12px;
    }

    .docked-spacer {
        flex: 1;
    }

    .docked-tabs {
        display: flex;
        gap: 4px;
        overflow-x: auto;
        padding-right: 8px;
        align-items: center;
        /* Hide scrollbar - Firefox */
        scrollbar-width: none;
        /* Hide scrollbar - WebKit */
        -ms-overflow-style: none;
    }

    .docked-tabs::-webkit-scrollbar {
        display: none;
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
        gap: 4px;
    }

    /* Icon-only buttons for window controls */
    .btn-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 6px !important;
        min-width: 28px;
        height: 28px;
    }

    .btn-icon svg {
        flex-shrink: 0;
    }

    .btn-icon.share-btn:hover {
        border-color: var(--accent) !important;
        color: var(--accent) !important;
        background: rgba(0, 255, 65, 0.1) !important;
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
        padding: 4px 6px;
        font-size: 14px;
        line-height: 1;
        min-width: 24px;
        min-height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 4px;
    }

    .tab-close:hover {
        color: var(--red);
        background: rgba(255, 85, 85, 0.1);
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
        overflow: hidden;
    }

    .create-panel :global(.inline-create) {
        flex: 1;
        overflow-y: auto;
    }

    .create-panel-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
        padding-bottom: 12px;
        border-bottom: 1px solid var(--border);
        flex-shrink: 0;
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
        display: flex;
        flex-direction: column;
    }

    .docked-create .create-panel-header {
        flex-shrink: 0;
    }

    .docked-create :global(.inline-create) {
        flex: 1;
        max-width: 1000px;
        margin: 0 auto;
        width: 100%;
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
    
    /* Panel overlay and modal */
    .panel-overlay {
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
        backdrop-filter: blur(4px);
    }
    
    .panel-modal {
        background: #1a1a1a;
        border: 1px solid #333;
        border-radius: 12px;
        max-width: 600px;
        max-height: 80vh;
        width: 90%;
        overflow: hidden;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
    }
</style>
