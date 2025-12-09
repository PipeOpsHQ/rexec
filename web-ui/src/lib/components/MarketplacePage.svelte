<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { toast } from "$stores/toast";
    import { api } from "$utils/api";

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
        icon?: string;
        category?: string;
        install_command?: string;
        requires_install?: boolean;
        usage_count: number;
        username?: string;
        is_owner?: boolean;
        created_at: string;
    }

    let snippets: Snippet[] = [];
    let isLoading = true;
    let searchQuery = "";
    let selectedCategory = "all";
    let sortBy = "popular";
    let expandedSnippet: string | null = null;

    const categories = [
        { value: "all", label: "🌐 All", icon: "🌐" },
        { value: "system", label: "🖥️ System", icon: "🖥️" },
        { value: "nodejs", label: "📦 Node.js", icon: "📦" },
        { value: "python", label: "🐍 Python", icon: "🐍" },
        { value: "golang", label: "🐹 Go", icon: "🐹" },
        { value: "devops", label: "☸️ DevOps", icon: "☸️" },
        { value: "editor", label: "✏️ Editor", icon: "✏️" },
        { value: "ai", label: "🤖 AI Tools", icon: "🤖" },
    ];

    async function loadSnippets() {
        isLoading = true;
        const params = new URLSearchParams();
        if (searchQuery) params.set("search", searchQuery);
        if (selectedCategory !== "all") params.set("language", selectedCategory);
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
        await api.post(`/api/snippets/${snippet.id}/use`, {});
        dispatch("use", { content: snippet.content, name: snippet.name });
        toast.success(`"${snippet.name}" copied to clipboard`);
        try {
            await navigator.clipboard.writeText(snippet.content);
        } catch (e) {
            // Clipboard may not be available
        }
    }

    async function runInstall(snippet: Snippet) {
        if (!snippet.install_command) return;
        await navigator.clipboard.writeText(snippet.install_command);
        toast.success("Install command copied! Paste in terminal to install.");
    }

    function copyToClipboard(content: string) {
        navigator.clipboard.writeText(content);
        toast.success("Copied to clipboard");
    }

    function toggleExpand(id: string) {
        expandedSnippet = expandedSnippet === id ? null : id;
    }

    function getCategoryIcon(category: string): string {
        const cat = categories.find(c => c.value === category);
        return cat?.icon || "📄";
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
            <button class="search-btn" onclick={handleSearch}>🔍</button>
        </div>
        <div class="category-tabs">
            {#each categories as cat}
                <button 
                    class="cat-tab" 
                    class:active={selectedCategory === cat.value}
                    onclick={() => { selectedCategory = cat.value; loadSnippets(); }}
                >
                    {cat.label}
                </button>
            {/each}
        </div>
        <div class="sort-group">
            <select bind:value={sortBy} onchange={loadSnippets}>
                <option value="popular">🔥 Popular</option>
                <option value="recent">🆕 Recent</option>
                <option value="name">🔤 A-Z</option>
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
                <p>Try adjusting your search or filters</p>
            </div>
        {:else}
            <div class="snippets-grid">
                {#each snippets as snippet (snippet.id)}
                    <div class="snippet-card" class:expanded={expandedSnippet === snippet.id}>
                        <div class="card-header" onclick={() => toggleExpand(snippet.id)}>
                            <span class="snippet-icon">{snippet.icon || getCategoryIcon(snippet.category || '')}</span>
                            <div class="snippet-info">
                                <span class="snippet-name">{snippet.name}</span>
                                <span class="snippet-meta">
                                    by {snippet.username || "rexec"} · {snippet.usage_count} uses
                                </span>
                            </div>
                            <div class="card-actions">
                                {#if snippet.requires_install && snippet.install_command}
                                    <button 
                                        class="btn-icon install"
                                        onclick={(e) => { e.stopPropagation(); runInstall(snippet); }}
                                        title="Copy install command"
                                    >
                                        ⬇️
                                    </button>
                                {/if}
                                <button 
                                    class="btn-icon copy"
                                    onclick={(e) => { e.stopPropagation(); copyToClipboard(snippet.content); }}
                                    title="Copy snippet"
                                >
                                    📋
                                </button>
                                <button 
                                    class="btn-icon use"
                                    onclick={(e) => { e.stopPropagation(); useSnippet(snippet); }}
                                    title="Use snippet"
                                >
                                    ▶️
                                </button>
                            </div>
                        </div>
                        
                        {#if expandedSnippet === snippet.id}
                            <div class="card-body">
                                {#if snippet.description}
                                    <p class="description">{snippet.description}</p>
                                {/if}
                                
                                {#if snippet.requires_install && snippet.install_command}
                                    <div class="install-section">
                                        <span class="install-label">📦 Install first:</span>
                                        <code class="install-cmd">{snippet.install_command}</code>
                                        <button 
                                            class="copy-install"
                                            onclick={() => copyToClipboard(snippet.install_command || '')}
                                        >
                                            Copy
                                        </button>
                                    </div>
                                {/if}
                                
                                <div class="code-preview">
                                    <pre><code>{snippet.content}</code></pre>
                                </div>
                                
                                <div class="card-footer">
                                    <span class="lang-badge">{snippet.language}</span>
                                    {#if snippet.category}
                                        <span class="cat-badge">{snippet.category}</span>
                                    {/if}
                                </div>
                            </div>
                        {/if}
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
    .marketplace-page {
        max-width: 1000px;
        margin: 0 auto;
        padding: 16px;
        animation: fadeIn 0.2s ease;
    }

    .page-header {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 20px;
    }

    .title-group h1 {
        font-size: 22px;
        margin: 0 0 2px 0;
    }

    .subtitle {
        color: var(--text-muted);
        font-size: 13px;
        margin: 0;
    }

    .back-btn {
        background: none;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        padding: 6px 12px;
        font-family: var(--font-mono);
        font-size: 12px;
        cursor: pointer;
        border-radius: 4px;
    }

    .back-btn:hover {
        border-color: var(--text);
        color: var(--text);
    }

    .filters {
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin-bottom: 20px;
    }

    .search-box {
        display: flex;
        gap: 0;
    }

    .search-box input {
        flex: 1;
        padding: 8px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-right: none;
        border-radius: 4px 0 0 4px;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .search-btn {
        padding: 8px 16px;
        background: var(--accent);
        border: none;
        border-radius: 0 4px 4px 0;
        cursor: pointer;
        font-size: 14px;
    }

    .category-tabs {
        display: flex;
        gap: 6px;
        flex-wrap: wrap;
    }

    .cat-tab {
        padding: 6px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 16px;
        color: var(--text-secondary);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .cat-tab:hover {
        border-color: var(--accent);
    }

    .cat-tab.active {
        background: var(--accent);
        color: #000;
        border-color: var(--accent);
    }

    .sort-group select {
        padding: 6px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 4px;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 12px;
        cursor: pointer;
    }

    .snippets-grid {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .snippet-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        overflow: hidden;
        transition: border-color 0.2s;
    }

    .snippet-card:hover {
        border-color: var(--accent);
    }

    .card-header {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px;
        cursor: pointer;
    }

    .snippet-icon {
        font-size: 20px;
        width: 32px;
        text-align: center;
    }

    .snippet-info {
        flex: 1;
        min-width: 0;
    }

    .snippet-name {
        display: block;
        font-weight: 600;
        font-size: 14px;
        color: var(--text);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .snippet-meta {
        font-size: 11px;
        color: var(--text-muted);
    }

    .card-actions {
        display: flex;
        gap: 4px;
    }

    .btn-icon {
        width: 32px;
        height: 32px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg);
        border: 1px solid var(--border);
        border-radius: 6px;
        cursor: pointer;
        font-size: 14px;
        transition: all 0.2s;
    }

    .btn-icon:hover {
        border-color: var(--accent);
        background: var(--accent);
    }

    .btn-icon.use {
        background: var(--accent);
        border-color: var(--accent);
    }

    .card-body {
        padding: 0 12px 12px 12px;
        border-top: 1px solid var(--border);
        animation: slideDown 0.2s ease;
    }

    .description {
        font-size: 13px;
        color: var(--text-secondary);
        margin: 12px 0;
        line-height: 1.4;
    }

    .install-section {
        display: flex;
        align-items: center;
        gap: 8px;
        background: rgba(255, 193, 7, 0.1);
        border: 1px solid rgba(255, 193, 7, 0.3);
        border-radius: 6px;
        padding: 8px 12px;
        margin-bottom: 12px;
        flex-wrap: wrap;
    }

    .install-label {
        font-size: 12px;
        color: var(--text-secondary);
        white-space: nowrap;
    }

    .install-cmd {
        flex: 1;
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--accent);
        background: var(--bg);
        padding: 4px 8px;
        border-radius: 4px;
        white-space: nowrap;
        overflow-x: auto;
    }

    .copy-install {
        padding: 4px 8px;
        background: var(--bg);
        border: 1px solid var(--border);
        border-radius: 4px;
        font-size: 11px;
        cursor: pointer;
        color: var(--text);
    }

    .copy-install:hover {
        border-color: var(--accent);
    }

    .code-preview {
        background: var(--bg);
        border-radius: 6px;
        padding: 12px;
        overflow-x: auto;
        max-height: 200px;
    }

    .code-preview pre {
        margin: 0;
    }

    .code-preview code {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--text-muted);
        white-space: pre;
    }

    .card-footer {
        display: flex;
        gap: 8px;
        margin-top: 12px;
    }

    .lang-badge, .cat-badge {
        padding: 2px 8px;
        border-radius: 4px;
        font-size: 10px;
        font-weight: 600;
        text-transform: uppercase;
    }

    .lang-badge {
        background: var(--accent);
        color: #000;
    }

    .cat-badge {
        background: var(--bg);
        color: var(--text-secondary);
        border: 1px solid var(--border);
    }

    .loading-state, .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 48px 0;
        color: var(--text-muted);
        text-align: center;
    }

    .spinner {
        width: 28px;
        height: 28px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-bottom: 12px;
    }

    .empty-icon {
        font-size: 40px;
        margin-bottom: 12px;
    }

    .empty-state h2 {
        font-size: 16px;
        margin-bottom: 4px;
        color: var(--text);
    }

    @keyframes spin { to { transform: rotate(360deg); } }
    @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
    @keyframes slideDown { from { opacity: 0; transform: translateY(-8px); } to { opacity: 1; transform: translateY(0); } }
</style>
