<script lang="ts">
    import { createEventDispatcher, onMount, onDestroy } from "svelte";
    import { toast } from "$stores/toast";
    import { api } from "$utils/api";
    import StatusIcon from "./icons/StatusIcon.svelte";
    import ConfirmModal from "./ConfirmModal.svelte";

    declare const ace: any; // Declare ace to avoid TypeScript errors

    const dispatch = createEventDispatcher<{
        back: void;
    }>();

    interface Snippet {
        id: string;
        name: string;
        content: string;
        language: string;
        is_public: boolean;
        description?: string;
        usage_count: number;
        username?: string;
        is_owner?: boolean;
        created_at: string;
    }

    let snippets: Snippet[] = [];
    let isLoading = true;
    let showCreate = false;
    
    // Create state
    let newName = "";
    let newContent = "";
    let newDescription = "";
    let newIsPublic = false;
    let isCreating = false;

    // Ace Editor instance
    let editor: any;
    let editorElement: HTMLDivElement; // Bind to this div

    // Delete state
    let showDeleteConfirm = false;
    let snippetToDelete: { id: string; name: string } | null = null;

    // Load snippets
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

    onMount(() => {
        loadSnippets();

        // Initialize Ace Editor once the component is mounted
        // This needs to be done dynamically if 'showCreate' is false initially
        // or ensure editorElement is rendered. We can use a reactive statement
        // for when showCreate becomes true.
    });

    // Reactive statement to initialize editor when the form is shown
    $: if (showCreate && editorElement && typeof ace !== 'undefined' && !editor) {
        editor = ace.edit(editorElement);
        editor.setTheme("ace/theme/dracula");
        editor.session.setMode("ace/mode/sh"); // Bash syntax highlighting
        editor.session.setUseWrapMode(true);
        editor.setShowPrintMargin(false);
        editor.setFontSize("13px");

        // Update newContent when editor content changes
        editor.session.on("change", () => {
            newContent = editor.getValue();
        });
        // Set initial content
        editor.setValue(newContent, -1); // -1 to move cursor to start
    }

    // Update editor content when newContent changes programmatically (e.g. after save/clear)
    $: if (editor && newContent !== editor.getValue()) {
        editor.setValue(newContent, -1);
    }

    onDestroy(() => {
        if (editor) {
            editor.destroy();
            editor.container.remove();
        }
    });

    async function createSnippet() {
        if (!newName.trim() || !newContent.trim()) {
            toast.error("Name and content are required");
            return;
        }

        isCreating = true;
        const { data, error } = await api.post<Snippet>("/api/snippets", {
            name: newName.trim(),
            content: newContent.trim(),
            description: newDescription.trim(),
            is_public: newIsPublic,
            language: "bash"
        });

        if (data) {
            snippets = [data, ...snippets];
            toast.success("Snippet saved");
            showCreate = false;
            newName = "";
            newContent = "";
            newDescription = "";
            newIsPublic = false;
        } else {
            toast.error(error || "Failed to save snippet");
        }
        isCreating = false;
    }

    async function togglePublic(snippet: Snippet) {
        const { data, error } = await api.put<Snippet>(`/api/snippets/${snippet.id}`, {
            is_public: !snippet.is_public
        });
        
        if (data) {
            snippets = snippets.map(s => s.id === snippet.id ? { ...s, is_public: data.is_public } : s);
            toast.success(data.is_public ? "Snippet is now public" : "Snippet is now private");
        } else {
            toast.error(error || "Failed to update snippet");
        }
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
    
    function copyToClipboard(text: string) {
        navigator.clipboard.writeText(text);
        toast.success("Copied to clipboard");
    }
</script>

<ConfirmModal
    bind:show={showDeleteConfirm}
    title="Delete Snippet"
    message={snippetToDelete ? `Are you sure you want to delete "${snippetToDelete.name}"?` : ""}
    confirmText="Delete"
    variant="danger"
    on:confirm={confirmDeleteSnippet}
/>

<div class="snippets-page">
    <div class="page-header">
        <button class="back-btn" onclick={() => dispatch("back")}>
            ← Back
        </button>
        <div class="title-group">
            <h1>Snippets & Macros</h1>
            <p class="subtitle">Save frequently used commands and scripts.</p>
        </div>
        <div class="header-actions">
            <a href="/marketplace" class="btn btn-secondary">
                🏪 Marketplace
            </a>
            {#if !showCreate}
                <button class="btn btn-primary" onclick={() => showCreate = true}>
                    + Create New
                </button>
            {/if}
        </div>
    </div>

    {#if showCreate}
        <div class="create-form-card">
            <div class="card-header">
                <h3>New Snippet</h3>
                <button class="close-btn" onclick={() => showCreate = false}>×</button>
            </div>
            <div class="card-body">
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
                    <label for="snip-desc">Description (optional)</label>
                    <input 
                        id="snip-desc" 
                        type="text" 
                        bind:value={newDescription} 
                        placeholder="Brief description of what this snippet does"
                        class="input"
                    />
                </div>
                <div class="form-group">
                    <label for="snip-content">Command / Script</label>
                    <div class="ace-editor-container" bind:this={editorElement}></div>
                </div>
                <div class="form-group checkbox-group">
                    <label class="checkbox-label">
                        <input 
                            type="checkbox" 
                            bind:checked={newIsPublic}
                        />
                        <span class="checkbox-text">
                            <strong>Make public</strong>
                            <small>Share in marketplace for others to use</small>
                        </span>
                    </label>
                </div>
                <div class="form-actions">
                    <button class="btn btn-secondary" onclick={() => {
                        showCreate = false;
                        newContent = "";
                        newName = "";
                        newDescription = "";
                        newIsPublic = false;
                    }}>Cancel</button>
                    <button 
                        class="btn btn-primary" 
                        onclick={createSnippet}
                        disabled={isCreating || !newName.trim() || !newContent.trim()}
                    >
                        {isCreating ? "Saving..." : "Save Snippet"}
                    </button>
                </div>
            </div>
        </div>
    {/if}

    <div class="content-body">
        {#if isLoading}
            <div class="loading-state">
                <div class="spinner"></div>
                <p>Loading...</p>
            </div>
        {:else if snippets.length === 0 && !showCreate}
            <div class="empty-state">
                <div class="empty-icon"><StatusIcon status="script" size={48} /></div>
                <h2>No Snippets</h2>
                <p>Save your favorite commands to run them instantly in any terminal.</p>
                <button class="btn btn-primary" onclick={() => showCreate = true}>
                    Create Your First Snippet
                </button>
            </div>
        {:else}
            <div class="snippets-grid">
                {#each snippets as snippet (snippet.id)}
                    <div class="snippet-card">
                        <div class="snippet-header">
                            <div class="snippet-title">
                                {snippet.name}
                                {#if snippet.is_public}
                                    <span class="public-badge" title="Public - visible in marketplace"><StatusIcon status="globe" size={14} /></span>
                                {/if}
                            </div>
                            <div class="snippet-actions">
                                <button 
                                    class="action-btn" 
                                    title={snippet.is_public ? "Make private" : "Make public"}
                                    onclick={() => togglePublic(snippet)}
                                >
                                    <StatusIcon status={snippet.is_public ? "unlock" : "lock"} size={14} />
                                </button>
                                <button 
                                    class="btn-icon" 
                                    onclick={() => copyToClipboard(snippet.content)}
                                    title="Copy to clipboard"
                                >
                                    <StatusIcon status="copy" size={14} />
                                </button>
                                <button 
                                    class="btn-icon danger" 
                                    onclick={() => deleteSnippet(snippet.id, snippet.name)}
                                    title="Delete"
                                >
                                    <StatusIcon status="close" size={14} />
                                </button>
                            </div>
                        </div>
                        <div class="snippet-content">
                            <code>{snippet.content}</code>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
    .snippets-page {
        max-width: 900px;
        margin: 0 auto;
        padding: 20px;
        animation: fadeIn 0.2s ease;
    }

    .page-header {
        display: flex;
        align-items: center;
        gap: 20px;
        margin-bottom: 30px;
    }

    .title-group {
        flex: 1;
    }

    .header-actions {
        display: flex;
        gap: 10px;
    }

    h1 {
        font-size: 24px;
        margin: 0 0 4px 0;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .subtitle {
        color: var(--text-muted);
        font-size: 14px;
        margin: 0;
    }

    .back-btn {
        background: none;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        padding: 8px 16px;
        font-family: var(--font-mono);
        font-size: 13px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .back-btn:hover {
        border-color: var(--text);
        color: var(--text);
    }

    /* Create Form */
    .create-form-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 6px;
        margin-bottom: 30px;
        overflow: hidden;
        animation: slideDown 0.2s ease;
    }

    .card-header {
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
        display: flex;
        justify-content: space-between;
        align-items: center;
        background: var(--bg-secondary);
    }

    .card-header h3 {
        margin: 0;
        font-size: 14px;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--accent);
    }

    .close-btn {
        background: none;
        border: none;
        color: var(--text-muted);
        font-size: 20px;
        cursor: pointer;
    }

    .card-body {
        padding: 20px;
    }

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

    /* Ace Editor specific styles */
    .ace-editor-container {
        height: 200px; /* Adjust height as needed */
        width: 100%;
        font-family: var(--font-mono);
        font-size: 13px;
        border: 1px solid var(--border);
    }

    .form-actions {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        margin-top: 20px;
    }

    /* List */
    .snippets-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 20px;
    }

    .snippet-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 6px;
        overflow: hidden;
        transition: border-color 0.2s;
    }

    .snippet-card:hover {
        border-color: var(--text-muted);
    }

    .snippet-header {
        padding: 12px 16px;
        border-bottom: 1px solid var(--border);
        display: flex;
        justify-content: space-between;
        align-items: center;
        background: var(--bg-secondary);
    }

    .snippet-title {
        font-weight: 600;
        font-size: 13px;
        color: var(--text);
    }

    .snippet-actions {
        display: flex;
        gap: 8px;
    }

    .snippet-content {
        padding: 12px 16px;
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--text-secondary);
        background: var(--bg-card);
        white-space: pre-wrap;
        max-height: 150px;
        overflow-y: auto;
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
        border: none;
        color: var(--text-muted);
        padding: 4px;
        cursor: pointer;
        transition: color 0.2s;
    }

    .btn-icon:hover {
        color: var(--text);
    }

    .btn-icon.danger:hover {
        color: #ff4444;
    }

    /* Loading/Empty */
    .loading-state, .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 60px 0;
        color: var(--text-muted);
        text-align: center;
    }

    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-bottom: 16px;
    }

    .empty-icon {
        font-size: 48px;
        margin-bottom: 16px;
        opacity: 0.5;
    }

    .empty-state h2 {
        font-size: 18px;
        margin-bottom: 8px;
        color: var(--text);
    }

    .empty-state p {
        max-width: 400px;
        margin-bottom: 24px;
    }

    @keyframes spin { to { transform: rotate(360deg); } }
    @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
    @keyframes slideDown { from { transform: translateY(-10px); opacity: 0; } to { transform: translateY(0); opacity: 1; } }

    /* Checkbox group */
    .checkbox-group {
        margin-top: 8px;
    }

    .checkbox-label {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        cursor: pointer;
    }

    .checkbox-label input[type="checkbox"] {
        margin-top: 3px;
        accent-color: var(--accent);
    }

    .checkbox-text {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .checkbox-text strong {
        font-size: 14px;
        color: var(--text);
    }

    .checkbox-text small {
        font-size: 12px;
        color: var(--text-muted);
    }

    .public-badge {
        font-size: 12px;
        margin-left: 6px;
    }

    .snippet-description {
        font-size: 12px;
        color: var(--text-muted);
        margin-top: 4px;
        font-style: italic;
    }

    .snippet-meta {
        display: flex;
        gap: 12px;
        font-size: 11px;
        color: var(--text-muted);
        margin-top: 8px;
    }

    .usage-count {
        display: flex;
        align-items: center;
        gap: 4px;
    }
</style>
