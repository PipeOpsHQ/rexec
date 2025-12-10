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
        { value: "all", label: "All", icon: "grid" },
        { value: "system", label: "System", icon: "terminal" },
        { value: "nodejs", label: "Node.js", icon: "nodejs" },
        { value: "python", label: "Python", icon: "python" },
        { value: "golang", label: "Go", icon: "golang" },
        { value: "devops", label: "DevOps", icon: "devops" },
        { value: "editor", label: "Editor", icon: "edit" },
        { value: "ai", label: "AI Tools", icon: "ai" },
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

    function getIconForCategory(category: string): string {
        // Return icon name for StatusIcon component
        switch (category) {
            case "system": return "system";
            case "nodejs": return "nodejs";
            case "python": return "python";
            case "golang": return "golang";
            case "devops": return "devops";
            case "editor": return "edit";
            case "ai": return "ai";
            default: return "file";
        }
    }

    function formatUsageCount(count: number): string {
        if (count >= 1000) return `${(count / 1000).toFixed(1)}k`;
        return count.toString();
    }
</script>

<div class="marketplace">
    <div class="marketplace-container">
        <!-- Header -->
        <header class="marketplace-header">
            <button class="back-btn" onclick={() => dispatch("back")}>
                <span class="back-icon">←</span>
                <span>Back</span>
            </button>
            <div class="header-content">
                <div class="header-badge">
                    <span class="badge-dot"></span>
                    <span>Community Snippets</span>
                </div>
                <h1>Snippet <span class="accent">Marketplace</span></h1>
                <p class="header-desc">Discover, share, and use terminal commands from the community</p>
            </div>
        </header>

        <!-- Search & Filters -->
        <div class="controls">
            <div class="search-wrapper">
                <span class="search-icon"><StatusIcon status="search" size={16} /></span>
                <input 
                    type="text" 
                    class="search-input"
                    placeholder="Search snippets..."
                    bind:value={searchQuery}
                    onkeydown={(e) => e.key === 'Enter' && handleSearch()}
                />
                {#if searchQuery}
                    <button class="search-clear" onclick={() => { searchQuery = ''; loadSnippets(); }}>×</button>
                {/if}
            </div>

            <div class="filter-row">
                <div class="category-pills">
                    {#each categories as cat}
                        <button 
                            class="pill" 
                            class:active={selectedCategory === cat.value}
                            onclick={() => { selectedCategory = cat.value; loadSnippets(); }}
                        >
                            {cat.label}
                        </button>
                    {/each}
                </div>
                <div class="sort-dropdown">
                    <select bind:value={sortBy} onchange={loadSnippets}>
                        <option value="popular">Popular</option>
                        <option value="recent">Recent</option>
                        <option value="name">A-Z</option>
                    </select>
                </div>
            </div>
        </div>

        <!-- Content -->
        <div class="content">
            {#if isLoading}
                <div class="state-container">
                    <div class="loader"></div>
                    <p>Loading snippets...</p>
                </div>
            {:else if snippets.length === 0}
                <div class="state-container">
                    <div class="empty-icon">
                        <StatusIcon status="search" size={48} />
                    </div>
                    <h2>No Snippets Found</h2>
                    <p>Try adjusting your search or category filter</p>
                </div>
            {:else}
                <div class="snippet-list">
                    {#each snippets as snippet (snippet.id)}
                        <article class="snippet-card" class:expanded={expandedSnippet === snippet.id}>
                            <div class="card-main" onclick={() => toggleExpand(snippet.id)}>
                                <div class="card-left">
                                    <span class="card-icon">
                                        <StatusIcon status={snippet.icon || getIconForCategory(snippet.category || '')} size={24} />
                                    </span>
                                </div>
                                <div class="card-center">
                                    <h3 class="card-title">{snippet.name}</h3>
                                    <div class="card-meta">
                                        <span class="meta-author">@{snippet.username || "rexec"}</span>
                                        <span class="meta-sep">·</span>
                                        <span class="meta-uses">{formatUsageCount(snippet.usage_count)} uses</span>
                                        {#if snippet.requires_install}
                                            <span class="meta-sep">·</span>
                                            <span class="meta-install">requires install</span>
                                        {/if}
                                    </div>
                                </div>
                                <div class="card-right">
                                    <div class="card-tags">
                                        <span class="tag tag-lang">{snippet.language}</span>
                                    </div>
                                    <div class="card-actions">
                                        {#if snippet.requires_install && snippet.install_command}
                                            <button 
                                                class="action-btn"
                                                onclick={(e) => { e.stopPropagation(); runInstall(snippet); }}
                                                title="Copy install command"
                                            >
                                                <StatusIcon status="download" size={14} />
                                            </button>
                                        {/if}
                                        <button 
                                            class="action-btn"
                                            onclick={(e) => { e.stopPropagation(); copyToClipboard(snippet.content); }}
                                            title="Copy to clipboard"
                                        >
                                            <StatusIcon status="copy" size={14} />
                                        </button>
                                        <button 
                                            class="action-btn primary"
                                            onclick={(e) => { e.stopPropagation(); useSnippet(snippet); }}
                                            title="Use snippet"
                                        >
                                            <StatusIcon status="play" size={14} />
                                        </button>
                                    </div>
                                    <span class="expand-icon" class:rotated={expandedSnippet === snippet.id}>▼</span>
                                </div>
                            </div>
                            
                            {#if expandedSnippet === snippet.id}
                                <div class="card-expanded">
                                    {#if snippet.description}
                                        <p class="card-desc">{snippet.description}</p>
                                    {/if}
                                    
                                    {#if snippet.requires_install && snippet.install_command}
                                        <div class="install-block">
                                            <div class="install-header">
                                                <StatusIcon status="download" size={14} />
                                                <span>Install first</span>
                                            </div>
                                            <div class="install-content">
                                                <code>{snippet.install_command}</code>
                                                <button 
                                                    class="copy-btn"
                                                    onclick={() => copyToClipboard(snippet.install_command || '')}
                                                >
                                                    Copy
                                                </button>
                                            </div>
                                        </div>
                                    {/if}
                                    
                                    <div class="code-block">
                                        <div class="code-header">
                                            <span class="code-dots">
                                                <span class="dot red"></span>
                                                <span class="dot yellow"></span>
                                                <span class="dot green"></span>
                                            </span>
                                            <span class="code-title">{snippet.name}</span>
                                            <button 
                                                class="code-copy"
                                                onclick={() => copyToClipboard(snippet.content)}
                                            >
                                                Copy
                                            </button>
                                        </div>
                                        <pre class="code-content"><code>{snippet.content}</code></pre>
                                    </div>
                                </div>
                            {/if}
                        </article>
                    {/each}
                </div>
            {/if}
        </div>

        <!-- Stats Footer -->
        <footer class="marketplace-footer">
            <div class="stat">
                <span class="stat-value">{snippets.length}</span>
                <span class="stat-label">snippets</span>
            </div>
            <div class="stat">
                <span class="stat-value">{categories.length - 1}</span>
                <span class="stat-label">categories</span>
            </div>
        </footer>
    </div>
</div>

<style>
    .marketplace {
        min-height: 100vh;
        background: var(--bg);
        animation: fadeIn 0.3s ease;
    }

    .marketplace-container {
        max-width: 900px;
        margin: 0 auto;
        padding: 24px 16px;
    }

    /* Header */
    .marketplace-header {
        margin-bottom: 32px;
        position: relative;
    }

    .back-btn {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        background: none;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        padding: 6px 12px;
        font-family: var(--font-mono);
        font-size: 12px;
        cursor: pointer;
        margin-bottom: 20px;
        transition: all 0.2s;
    }

    .back-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    .back-icon {
        font-size: 14px;
    }

    .header-content {
        text-align: center;
    }

    .header-badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 4px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        font-size: 10px;
        color: var(--text-secondary);
        margin-bottom: 16px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .badge-dot {
        width: 6px;
        height: 6px;
        background: var(--accent);
        animation: pulse 2s ease-in-out infinite;
    }

    .header-content h1 {
        font-size: 28px;
        font-weight: 700;
        margin: 0 0 8px 0;
        color: var(--text);
    }

    .accent {
        color: var(--accent);
    }

    .header-desc {
        color: var(--text-muted);
        font-size: 14px;
        margin: 0;
    }

    /* Controls */
    .controls {
        margin-bottom: 24px;
    }

    .search-wrapper {
        position: relative;
        margin-bottom: 16px;
    }

    .search-icon {
        position: absolute;
        left: 12px;
        top: 50%;
        transform: translateY(-50%);
        color: var(--text-muted);
        pointer-events: none;
    }

    .search-input {
        width: 100%;
        padding: 10px 36px 10px 40px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
        transition: border-color 0.2s;
    }

    .search-input:focus {
        outline: none;
        border-color: var(--accent);
    }

    .search-input::placeholder {
        color: var(--text-muted);
    }

    .search-clear {
        position: absolute;
        right: 8px;
        top: 50%;
        transform: translateY(-50%);
        width: 20px;
        height: 20px;
        background: var(--border);
        border: none;
        color: var(--text);
        cursor: pointer;
        font-size: 14px;
        display: flex;
        align-items: center;
        justify-content: center;
    }

    .filter-row {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        flex-wrap: wrap;
    }

    .category-pills {
        display: flex;
        gap: 6px;
        flex-wrap: wrap;
        flex: 1;
    }

    .pill {
        padding: 6px 14px;
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        font-family: var(--font-mono);
        font-size: 11px;
        cursor: pointer;
        transition: all 0.2s;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .pill:hover {
        border-color: var(--text-muted);
        color: var(--text);
    }

    .pill.active {
        background: var(--accent);
        color: #000;
        border-color: var(--accent);
    }

    .sort-dropdown select {
        padding: 6px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 11px;
        cursor: pointer;
    }

    /* Content */
    .content {
        min-height: 400px;
    }

    .state-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 80px 20px;
        color: var(--text-muted);
        text-align: center;
    }

    .loader {
        width: 32px;
        height: 32px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-bottom: 16px;
    }

    .empty-icon {
        margin-bottom: 16px;
        opacity: 0.5;
    }

    .state-container h2 {
        font-size: 16px;
        color: var(--text);
        margin: 0 0 4px 0;
    }

    .state-container p {
        font-size: 13px;
        margin: 0;
    }

    /* Snippet List */
    .snippet-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .snippet-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        transition: all 0.2s;
    }

    .snippet-card:hover {
        border-color: rgba(0, 255, 102, 0.3);
    }

    .snippet-card.expanded {
        border-color: var(--accent);
    }

    .card-main {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 16px;
        cursor: pointer;
    }

    .card-left {
        flex-shrink: 0;
    }

    .card-icon {
        width: 36px;
        height: 36px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 255, 102, 0.1);
        border: 1px solid rgba(0, 255, 102, 0.2);
        color: var(--accent);
    }

    .card-center {
        flex: 1;
        min-width: 0;
    }

    .card-title {
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        margin: 0 0 4px 0;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .card-meta {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 11px;
        color: var(--text-muted);
    }

    .meta-author {
        color: var(--accent);
    }

    .meta-sep {
        opacity: 0.3;
    }

    .meta-install {
        color: #ffc107;
    }

    .card-right {
        display: flex;
        align-items: center;
        gap: 12px;
        flex-shrink: 0;
    }

    .card-tags {
        display: none;
    }

    @media (min-width: 600px) {
        .card-tags {
            display: flex;
            gap: 6px;
        }
    }

    .tag {
        padding: 2px 8px;
        font-size: 9px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .tag-lang {
        background: var(--accent);
        color: #000;
    }

    .card-actions {
        display: flex;
        gap: 4px;
    }

    .action-btn {
        width: 28px;
        height: 28px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        cursor: pointer;
        transition: all 0.2s;
    }

    .action-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    .action-btn.primary {
        background: var(--accent);
        border-color: var(--accent);
        color: #000;
    }

    .action-btn.primary:hover {
        opacity: 0.9;
    }

    .expand-icon {
        font-size: 10px;
        color: var(--text-muted);
        transition: transform 0.2s;
    }

    .expand-icon.rotated {
        transform: rotate(180deg);
    }

    /* Expanded Card */
    .card-expanded {
        padding: 16px;
        border-top: 1px solid var(--border);
        animation: slideDown 0.2s ease;
    }

    .card-desc {
        font-size: 13px;
        color: var(--text-secondary);
        line-height: 1.5;
        margin: 0 0 16px 0;
    }

    .install-block {
        background: rgba(255, 193, 7, 0.08);
        border: 1px solid rgba(255, 193, 7, 0.2);
        padding: 12px;
        margin-bottom: 16px;
    }

    .install-header {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 11px;
        color: #ffc107;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        margin-bottom: 8px;
    }

    .install-content {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .install-content code {
        flex: 1;
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--accent);
        background: var(--bg);
        padding: 8px 12px;
        overflow-x: auto;
    }

    .copy-btn {
        padding: 6px 12px;
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        font-family: var(--font-mono);
        font-size: 11px;
        cursor: pointer;
        transition: all 0.2s;
        flex-shrink: 0;
    }

    .copy-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    /* Code Block */
    .code-block {
        background: var(--bg);
        border: 1px solid var(--border);
        overflow: hidden;
    }

    .code-header {
        display: flex;
        align-items: center;
        padding: 8px 12px;
        background: rgba(255, 255, 255, 0.02);
        border-bottom: 1px solid var(--border);
    }

    .code-dots {
        display: flex;
        gap: 6px;
        margin-right: 12px;
    }

    .code-dots .dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
    }

    .code-dots .red { background: #ff5f56; }
    .code-dots .yellow { background: #ffbd2e; }
    .code-dots .green { background: #27c93f; }

    .code-title {
        flex: 1;
        font-size: 11px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    .code-copy {
        padding: 4px 10px;
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        font-family: var(--font-mono);
        font-size: 10px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .code-copy:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    .code-content {
        padding: 16px;
        margin: 0;
        overflow-x: auto;
        max-height: 200px;
        font-family: var(--font-mono);
        font-size: 12px;
        line-height: 1.5;
        color: var(--text-muted);
    }

    .code-content code {
        font-family: inherit;
    }

    /* Footer */
    .marketplace-footer {
        display: flex;
        justify-content: center;
        gap: 32px;
        padding: 32px 0 16px;
        margin-top: 32px;
        border-top: 1px solid var(--border);
    }

    .stat {
        text-align: center;
    }

    .stat-value {
        display: block;
        font-size: 24px;
        font-weight: 700;
        color: var(--accent);
        font-family: var(--font-mono);
    }

    .stat-label {
        font-size: 11px;
        color: var(--text-muted);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    /* Animations */
    @keyframes spin { to { transform: rotate(360deg); } }
    @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
    @keyframes slideDown { from { opacity: 0; max-height: 0; } to { opacity: 1; max-height: 500px; } }
    @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
</style>
