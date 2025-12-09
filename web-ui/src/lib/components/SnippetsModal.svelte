<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { fade, scale } from "svelte/transition";
    import { api } from "$utils/api";
    import { toast } from "$stores/toast";
    import { terminal } from "$stores/terminal";
    import ConfirmModal from "./ConfirmModal.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    export let show: boolean = false;
    export let containerId: string | null = null;

    const dispatch = createEventDispatcher<{
        close: void;
    }>();

    interface Snippet {
        id: string;
        name: string;
        content: string;
        language: string;
        description?: string;
        icon?: string;
        category?: string;
        install_command?: string;
        requires_install?: boolean;
        usage_count?: number;
        username?: string;
        created_at: string;
    }

    let snippets: Snippet[] = [];
    let marketplaceSnippets: Snippet[] = [];
    let isLoading = true;
    let isLoadingMarketplace = false;
    let activeTab: "list" | "create" | "marketplace" = "list";
    
    // Marketplace filters
    let marketplaceSearch = "";
    let marketplaceCategory = "all";
    
    // Create state
    let newName = "";
    let newContent = "";
    let isCreating = false;

    // Delete state
    let showDeleteConfirm = false;
    let snippetToDelete: { id: string; name: string } | null = null;

    // Load user snippets
    async function loadSnippets() {
        isLoading = true;
        const { data, error } = await api.get<{ snippets: Snippet[] }>("/api/snippets");
        
        if (data) {
            snippets = data.snippets || [];
        } else if (error) {
            toast.error("Failed to load snippets");
        }
        isLoading = false;
    }

    // Load marketplace snippets
    async function loadMarketplaceSnippets() {
        isLoadingMarketplace = true;
        const params = new URLSearchParams();
        if (marketplaceSearch) params.set("search", marketplaceSearch);
        if (marketplaceCategory !== "all") params.set("language", marketplaceCategory);
        params.set("sort", "popular");

        const { data, error } = await api.get<{ snippets: Snippet[] }>(
            `/api/marketplace/snippets?${params.toString()}`
        );
        
        if (data) {
            marketplaceSnippets = data.snippets || [];
        } else if (error) {
            toast.error("Failed to load marketplace");
        }
        isLoadingMarketplace = false;
    }

    // Refresh when opening
    $: if (show) {
        loadSnippets();
        if (activeTab === "marketplace") loadMarketplaceSnippets();
    }
    
    // Load marketplace when switching to tab
    $: if (activeTab === "marketplace" && marketplaceSnippets.length === 0) {
        loadMarketplaceSnippets();
    }

    function handleClose() {
        show = false;
        dispatch("close");
        // Reset state
        activeTab = "list";
        newName = "";
        newContent = "";
    }

    function handleKeydown(e: KeyboardEvent) {
        if (!show) return;
        if (e.key === "Escape") handleClose();
    }

    async function createSnippet() {
        if (!newName.trim() || !newContent.trim()) {
            toast.error("Name and content are required");
            return;
        }

        isCreating = true;
        const { data, error } = await api.post<Snippet>("/api/snippets", {
            name: newName.trim(),
            content: newContent.trim(),
            language: "bash" // Default for now
        });

        if (data) {
            snippets = [data, ...snippets];
            toast.success("Snippet saved");
            activeTab = "list";
            newName = "";
            newContent = "";
        } else {
            toast.error(error || "Failed to save snippet");
        }
        isCreating = false;
    }

    function deleteSnippet(id: string, name: string) {
        snippetToDelete = { id, name };
        showDeleteConfirm = true;
    }

    async function confirmDeleteSnippet() {
        if (!snippetToDelete) return;
        const { id } = snippetToDelete;
        snippetToDelete = null;

        const { error } = await api.delete(`/api/snippets/${id}`);
        if (!error) {
            snippets = snippets.filter(s => s.id !== id);
            toast.success("Snippet deleted");
        } else {
            toast.error(error || "Failed to delete snippet");
        }
    }

    function runSnippet(snippet: Snippet) {
        if (!containerId) {
            toast.error("No active terminal to run snippet");
            return;
        }

        // Find session ID from container ID
        // Note: This relies on terminal store tracking sessions by container ID
        // We might need to look up session ID if we only have container ID
        // But terminal.sendInput takes sessionId.
        
        // Wait, terminal store has sessions map.
        // We need to find the session for this container.
        let sessionId: string | undefined;
        
        // This is a bit of a hack if we don't have direct access to session ID
        // But typically the caller passes the container ID associated with the active view
        // The store doesn't expose a direct lookup by container ID publicly easily?
        // Actually, we can just send it if we have the session object in parent.
        // But let's assume we can find it or the parent handles the run.
        
        // Alternative: Dispatch 'run' event and let parent handle it.
        // This is cleaner as parent has the session context.
        dispatch('run', { snippet });
        toast.success(`Running "${snippet.name}"`);
        handleClose();
    }
</script>

<svelte:window onkeydown={handleKeydown} />

<ConfirmModal
    bind:show={showDeleteConfirm}
    title="Delete Snippet"
    message={snippetToDelete ? `Are you sure you want to delete "${snippetToDelete.name}"?` : ""}
    confirmText="Delete"
    variant="danger"
    on:confirm={confirmDeleteSnippet}
/>

{#if show}
    <div class="modal-backdrop" transition:fade={{ duration: 150 }}>
        <div class="modal-container" transition:scale={{ duration: 150, start: 0.95 }}>
            <div class="modal-header">
                <div class="modal-title-group">
                    <span class="modal-icon"><StatusIcon status="bolt" size={20} /></span>
                    <h2>Snippets & Macros</h2>
                </div>
                <button class="close-btn" onclick={handleClose}>×</button>
            </div>

            <div class="tabs">
                <button 
                    class="tab-btn" 
                    class:active={activeTab === "list"} 
                    onclick={() => activeTab = "list"}
                >
                    My Snippets
                </button>
                <button 
                    class="tab-btn" 
                    class:active={activeTab === "marketplace"} 
                    onclick={() => activeTab = "marketplace"}
                >
                    Marketplace
                </button>
                <button 
                    class="tab-btn" 
                    class:active={activeTab === "create"} 
                    onclick={() => activeTab = "create"}
                >
                    Create
                </button>
            </div>

            <div class="modal-body">
                {#if activeTab === "list"}
                    {#if isLoading}
                        <div class="loading-state">
                            <div class="spinner"></div>
                            <p>Loading snippets...</p>
                        </div>
                    {:else if snippets.length === 0}
                        <div class="empty-state">
                            <div class="empty-icon"><StatusIcon status="terminal" size={32} /></div>
                            <p>No snippets found.</p>
                            <button class="btn btn-primary" onclick={() => activeTab = "create"}>
                                Create Your First Snippet
                            </button>
                        </div>
                    {:else}
                        <div class="snippets-list">
                            {#each snippets as snippet (snippet.id)}
                                <div class="snippet-card">
                                    <div class="snippet-info">
                                        <div class="snippet-name">{snippet.name}</div>
                                        <div class="snippet-preview">{snippet.content}</div>
                                    </div>
                                    <div class="snippet-actions">
                                        <button 
                                            class="btn btn-primary btn-sm"
                                            onclick={() => runSnippet(snippet)}
                                            title="Run in terminal"
                                        >
                                            Run
                                        </button>
                                        <button 
                                            class="btn btn-icon btn-sm btn-delete"
                                            onclick={() => deleteSnippet(snippet.id, snippet.name)}
                                            title="Delete"
                                        >
                                            <StatusIcon status="trash" size={14} />
                                        </button>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                {:else if activeTab === "marketplace"}
                    <!-- Marketplace Search -->
                    <div class="marketplace-controls">
                        <div class="search-box">
                            <StatusIcon status="search" size={14} />
                            <input 
                                type="text" 
                                placeholder="Search snippets..."
                                bind:value={marketplaceSearch}
                                onkeydown={(e) => e.key === 'Enter' && loadMarketplaceSnippets()}
                            />
                        </div>
                        <select 
                            class="category-select"
                            bind:value={marketplaceCategory}
                            onchange={() => loadMarketplaceSnippets()}
                        >
                            <option value="all">All</option>
                            <option value="system">System</option>
                            <option value="nodejs">Node.js</option>
                            <option value="python">Python</option>
                            <option value="golang">Go</option>
                            <option value="devops">DevOps</option>
                            <option value="ai">AI Tools</option>
                        </select>
                    </div>

                    {#if isLoadingMarketplace}
                        <div class="loading-state">
                            <div class="spinner"></div>
                            <p>Loading marketplace...</p>
                        </div>
                    {:else if marketplaceSnippets.length === 0}
                        <div class="empty-state">
                            <div class="empty-icon"><StatusIcon status="grid" size={32} /></div>
                            <p>No snippets found in marketplace.</p>
                        </div>
                    {:else}
                        <div class="snippets-list marketplace-list">
                            {#each marketplaceSnippets as snippet (snippet.id)}
                                <div class="snippet-card marketplace-card">
                                    <div class="snippet-header">
                                        <div class="snippet-icon">
                                            <StatusIcon status={snippet.category || 'file'} size={16} />
                                        </div>
                                        <div class="snippet-meta">
                                            <div class="snippet-name">{snippet.name}</div>
                                            {#if snippet.username}
                                                <div class="snippet-author">by {snippet.username}</div>
                                            {/if}
                                        </div>
                                        {#if snippet.usage_count}
                                            <div class="snippet-uses">{snippet.usage_count} uses</div>
                                        {/if}
                                    </div>
                                    {#if snippet.description}
                                        <div class="snippet-desc">{snippet.description}</div>
                                    {/if}
                                    <div class="snippet-preview">{snippet.content}</div>
                                    <div class="snippet-actions">
                                        {#if snippet.install_command}
                                            <button 
                                                class="btn btn-secondary btn-sm"
                                                onclick={() => {
                                                    navigator.clipboard.writeText(snippet.install_command || '');
                                                    toast.success("Install command copied!");
                                                }}
                                                title="Copy install command"
                                            >
                                                Install
                                            </button>
                                        {/if}
                                        <button 
                                            class="btn btn-primary btn-sm"
                                            onclick={() => runSnippet(snippet)}
                                            title="Run in terminal"
                                        >
                                            Run
                                        </button>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                {:else}
                    <div class="create-form">
                        <div class="form-group">
                            <label for="snip-name">Name</label>
                            <input 
                                id="snip-name" 
                                type="text" 
                                bind:value={newName} 
                                placeholder="e.g. Install Node.js"
                                class="input"
                            />
                        </div>
                        <div class="form-group">
                            <label for="snip-content">Command / Script</label>
                            <textarea 
                                id="snip-content" 
                                bind:value={newContent} 
                                placeholder="npm install -g ..."
                                class="input textarea"
                                rows="6"
                            ></textarea>
                            <p class="hint">Multi-line scripts will be executed sequentially.</p>
                        </div>
                        <div class="form-actions">
                            <button class="btn btn-secondary" onclick={() => activeTab = "list"}>Cancel</button>
                            <button 
                                class="btn btn-primary" 
                                onclick={createSnippet}
                                disabled={isCreating || !newName.trim() || !newContent.trim()}
                            >
                                {isCreating ? "Saving..." : "Save Snippet"}
                            </button>
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
        backdrop-filter: blur(4px);
    }

    .modal-container {
        width: 500px;
        max-width: 90vw;
        max-height: 85vh;
        background: var(--bg-card);
        border: 1px solid var(--border);
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .modal-header {
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .modal-title-group {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .modal-icon {
        font-size: 20px;
    }

    .modal-header h2 {
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .close-btn {
        background: none;
        border: none;
        font-size: 24px;
        color: var(--text-muted);
        cursor: pointer;
        padding: 0;
        line-height: 1;
    }

    .close-btn:hover {
        color: var(--text);
    }

    .tabs {
        display: flex;
        border-bottom: 1px solid var(--border);
        padding: 0 10px;
        background: var(--bg-secondary);
    }

    .tab-btn {
        padding: 12px 16px;
        background: none;
        border: none;
        border-bottom: 2px solid transparent;
        color: var(--text-muted);
        cursor: pointer;
        font-size: 13px;
        font-weight: 500;
        transition: all 0.2s;
    }

    .tab-btn:hover {
        color: var(--text);
    }

    .tab-btn.active {
        color: var(--accent);
        border-bottom-color: var(--accent);
    }

    .modal-body {
        padding: 20px;
        overflow-y: auto;
        flex: 1;
    }

    /* List Styles */
    .snippets-list {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .snippet-card {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 4px;
        transition: border-color 0.2s;
    }

    .snippet-card:hover {
        border-color: var(--text-muted);
    }

    .snippet-info {
        flex: 1;
        min-width: 0;
        margin-right: 16px;
    }

    .snippet-name {
        font-weight: 600;
        font-size: 14px;
        margin-bottom: 4px;
        color: var(--text);
    }

    .snippet-preview {
        font-family: var(--font-mono);
        font-size: 11px;
        color: var(--text-muted);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .snippet-actions {
        display: flex;
        gap: 8px;
    }

    /* Form Styles */
    .form-group {
        margin-bottom: 16px;
    }

    .form-group label {
        display: block;
        margin-bottom: 8px;
        font-size: 12px;
        color: var(--text-secondary);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .input {
        width: 100%;
        padding: 10px 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .input:focus {
        outline: none;
        border-color: var(--accent);
    }

    .textarea {
        resize: vertical;
        min-height: 100px;
    }

    .hint {
        margin-top: 6px;
        font-size: 11px;
        color: var(--text-muted);
    }

    .form-actions {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        margin-top: 24px;
    }

    /* Buttons */
    .btn {
        padding: 8px 16px;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-primary {
        background: var(--accent);
        color: var(--bg);
        border: 1px solid var(--accent);
    }

    .btn-primary:hover {
        box-shadow: 0 0 10px rgba(0, 255, 65, 0.3);
    }

    .btn-secondary {
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text);
    }

    .btn-secondary:hover {
        border-color: var(--text-muted);
    }

    .btn-icon {
        background: transparent;
        border: 1px solid transparent;
        color: var(--text-muted);
        padding: 6px;
        font-size: 14px;
    }

    .btn-icon:hover {
        background: rgba(255, 255, 255, 0.1);
        color: var(--text);
    }

    .btn-delete:hover {
        background: rgba(255, 100, 100, 0.2);
        color: #ff6b6b;
    }

    .btn-sm {
        padding: 4px 10px;
        font-size: 11px;
    }

    /* Loading/Empty */
    .loading-state, .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 40px 0;
        color: var(--text-muted);
    }

    .spinner {
        width: 24px;
        height: 24px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-bottom: 12px;
    }

    .empty-icon {
        margin-bottom: 12px;
        opacity: 0.5;
        color: var(--accent);
    }

    @keyframes spin { to { transform: rotate(360deg); } }

    /* Marketplace Styles */
    .marketplace-controls {
        display: flex;
        gap: 10px;
        margin-bottom: 16px;
    }

    .search-box {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 4px;
    }

    .search-box input {
        flex: 1;
        background: none;
        border: none;
        color: var(--text);
        font-size: 13px;
        outline: none;
    }

    .search-box input::placeholder {
        color: var(--text-muted);
    }

    .category-select {
        padding: 8px 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 4px;
        color: var(--text);
        font-size: 12px;
        cursor: pointer;
    }

    .category-select:focus {
        outline: none;
        border-color: var(--accent);
    }

    .marketplace-list {
        max-height: 400px;
        overflow-y: auto;
    }

    .marketplace-card {
        flex-direction: column;
        align-items: stretch;
        gap: 8px;
    }

    .snippet-header {
        display: flex;
        align-items: center;
        gap: 10px;
    }

    .snippet-icon {
        color: var(--accent);
        opacity: 0.8;
    }

    .snippet-meta {
        flex: 1;
        min-width: 0;
    }

    .snippet-author {
        font-size: 10px;
        color: var(--text-muted);
    }

    .snippet-uses {
        font-size: 10px;
        color: var(--text-muted);
        background: var(--bg);
        padding: 2px 6px;
        border-radius: 10px;
    }

    .snippet-desc {
        font-size: 12px;
        color: var(--text-secondary);
        line-height: 1.4;
    }

    .marketplace-card .snippet-actions {
        justify-content: flex-end;
        margin-top: 4px;
        padding-top: 8px;
        border-top: 1px solid var(--border);
    }
</style>
