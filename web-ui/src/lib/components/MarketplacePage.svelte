<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { toast } from "$stores/toast";
    import { api } from "$utils/api";
    import StatusIcon from "./icons/StatusIcon.svelte";

    const dispatch = createEventDispatcher<{
        back: void;
        use: { content: string; name: string };
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
    let searchQuery = "";
    let selectedLanguage = "all";
    let sortBy = "popular";

    const languages = [
        { value: "all", label: "All Languages" },
        { value: "bash", label: "Bash" },
        { value: "python", label: "Python" },
        { value: "javascript", label: "JavaScript" },
        { value: "go", label: "Go" },
    ];

    async function loadSnippets() {
        isLoading = true;
        const params = new URLSearchParams();
        if (searchQuery) params.set("search", searchQuery);
        if (selectedLanguage !== "all") params.set("language", selectedLanguage);
        params.set("sort", sortBy);

        const { data, error } = await api.get<{ snippets: Snippet[] }>(
            `/api/marketplace/snippets?${params.toString()}`
        );
        
        if (data) {
            snippets = data.snippets || [];
        } else if (error) {
            toast.error("Failed to load marketplace");
        }
        isLoading = false;
    }

    onMount(() => {
        loadSnippets();
    });

    function handleSearch() {
        loadSnippets();
    }

    async function useSnippet(snippet: Snippet) {
        // Record usage
        await api.post(`/api/snippets/${snippet.id}/use`, {});
        
        // Dispatch to parent to insert into terminal
        dispatch("use", { content: snippet.content, name: snippet.name });
        toast.success(`"${snippet.name}" copied to clipboard`);
        
        // Also copy to clipboard
        try {
            await navigator.clipboard.writeText(snippet.content);
        } catch (e) {
            // Clipboard may not be available
        }
    }

    function copyToClipboard(content: string) {
        navigator.clipboard.writeText(content);
        toast.success("Copied to clipboard");
    }

    function formatDate(dateStr: string): string {
        return new Date(dateStr).toLocaleDateString();
    }
</script>

<div class="marketplace-page">
    <div class="page-header">
        <button class="back-btn" onclick={() => dispatch("back")}>← Back</button>
        <div class="title-group">
            <h1>🏪 Snippet Marketplace</h1>
            <p class="subtitle">Discover and use community-shared snippets</p>
        </div>
    </div>

    <div class="filters">
        <div class="search-box">
            <input 
                type="text" 
                placeholder="Search snippets..."
                bind:value={searchQuery}
                onkeydown={(e) => e.key === 'Enter' && handleSearch()}
            />
            <button class="search-btn" onclick={handleSearch}>Search</button>
        </div>
        <div class="filter-group">
            <select bind:value={selectedLanguage} onchange={loadSnippets}>
                {#each languages as lang}
                    <option value={lang.value}>{lang.label}</option>
                {/each}
            </select>
            <select bind:value={sortBy} onchange={loadSnippets}>
                <option value="popular">Most Popular</option>
                <option value="recent">Most Recent</option>
                <option value="name">Alphabetical</option>
            </select>
        </div>
    </div>

    <div class="content-body">
        {#if isLoading}
            <div class="loading-state">
                <div class="spinner"></div>
                <p>Loading marketplace...</p>
            </div>
        {:else if snippets.length === 0}
            <div class="empty-state">
                <div class="empty-icon">🔍</div>
                <h2>No Snippets Found</h2>
                <p>Try adjusting your search or filters, or be the first to share a snippet!</p>
            </div>
        {:else}
            <div class="snippets-grid">
                {#each snippets as snippet (snippet.id)}
                    <div class="snippet-card">
                        <div class="snippet-header">
                            <div class="snippet-title">{snippet.name}</div>
                            <div class="snippet-author">by {snippet.username || "Anonymous"}</div>
                        </div>
                        {#if snippet.description}
                            <p class="snippet-description">{snippet.description}</p>
                        {/if}
                        <div class="snippet-preview">
                            <code>{snippet.content.slice(0, 150)}{snippet.content.length > 150 ? '...' : ''}</code>
                        </div>
                        <div class="snippet-footer">
                            <div class="snippet-meta">
                                <span class="language-badge">{snippet.language}</span>
                                <span class="usage-count">📊 {snippet.usage_count} uses</span>
                                <span class="date">{formatDate(snippet.created_at)}</span>
                            </div>
                            <div class="snippet-actions">
                                <button 
                                    class="btn btn-secondary btn-sm"
                                    onclick={() => copyToClipboard(snippet.content)}
                                >
                                    📋 Copy
                                </button>
                                <button 
                                    class="btn btn-primary btn-sm"
                                    onclick={() => useSnippet(snippet)}
                                >
                                    ▶ Use
                                </button>
                            </div>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
    .marketplace-page {
        max-width: 1100px;
        margin: 0 auto;
        padding: 20px;
        animation: fadeIn 0.2s ease;
    }

    .page-header {
        display: flex;
        align-items: center;
        gap: 20px;
        margin-bottom: 24px;
    }

    .title-group {
        flex: 1;
    }

    h1 {
        font-size: 24px;
        margin: 0 0 4px 0;
        font-weight: 600;
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

    .filters {
        display: flex;
        gap: 16px;
        margin-bottom: 24px;
        flex-wrap: wrap;
    }

    .search-box {
        display: flex;
        flex: 1;
        min-width: 250px;
    }

    .search-box input {
        flex: 1;
        padding: 10px 14px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-right: none;
        border-radius: 4px 0 0 4px;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .search-btn {
        padding: 10px 20px;
        background: var(--accent);
        border: 1px solid var(--accent);
        border-radius: 0 4px 4px 0;
        color: #000;
        font-weight: 600;
        cursor: pointer;
    }

    .filter-group {
        display: flex;
        gap: 8px;
    }

    .filter-group select {
        padding: 10px 14px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 4px;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
        cursor: pointer;
    }

    .snippets-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 16px;
    }

    .snippet-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 16px;
        display: flex;
        flex-direction: column;
        gap: 12px;
        transition: border-color 0.2s;
    }

    .snippet-card:hover {
        border-color: var(--accent);
    }

    .snippet-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
    }

    .snippet-title {
        font-weight: 600;
        font-size: 15px;
        color: var(--text);
    }

    .snippet-author {
        font-size: 12px;
        color: var(--text-muted);
    }

    .snippet-description {
        font-size: 13px;
        color: var(--text-secondary);
        margin: 0;
        line-height: 1.4;
    }

    .snippet-preview {
        background: var(--bg);
        border-radius: 4px;
        padding: 10px;
        overflow: hidden;
    }

    .snippet-preview code {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--text-muted);
        white-space: pre-wrap;
        word-break: break-all;
    }

    .snippet-footer {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-top: auto;
    }

    .snippet-meta {
        display: flex;
        gap: 10px;
        font-size: 11px;
        color: var(--text-muted);
        align-items: center;
    }

    .language-badge {
        background: var(--accent);
        color: #000;
        padding: 2px 8px;
        border-radius: 4px;
        font-weight: 600;
        text-transform: uppercase;
        font-size: 10px;
    }

    .snippet-actions {
        display: flex;
        gap: 8px;
    }

    .btn-sm {
        padding: 6px 12px;
        font-size: 12px;
    }

    .btn {
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-family: var(--font-mono);
        transition: all 0.2s;
    }

    .btn-primary {
        background: var(--accent);
        color: #000;
        font-weight: 600;
    }

    .btn-secondary {
        background: var(--bg);
        color: var(--text);
        border: 1px solid var(--border);
    }

    .btn-secondary:hover {
        border-color: var(--text);
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
    }

    .empty-state h2 {
        font-size: 18px;
        margin-bottom: 8px;
        color: var(--text);
    }

    @keyframes spin { to { transform: rotate(360deg); } }
    @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
</style>
