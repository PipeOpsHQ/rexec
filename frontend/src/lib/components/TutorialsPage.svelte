<script lang="ts">
    import { onMount } from "svelte";
    import { auth, isAdmin } from "$stores/auth";
    import StatusIcon from "./icons/StatusIcon.svelte";
    import { toast } from "$stores/toast";

    interface Tutorial {
        id: string;
        title: string;
        description: string;
        video_url: string;
        thumbnail: string;
        duration: string;
        category: string;
        order: number;
        is_published: boolean;
        created_at: string;
        updated_at: string;
    }

    let tutorials: Tutorial[] = [];
    let categories: Record<string, Tutorial[]> = {};
    let isLoading = true;
    let selectedCategory = "";
    let selectedTutorial: Tutorial | null = null;

    // Admin state
    let showAdminModal = false;
    let editingTutorial: Tutorial | null = null;
    let formData = {
        title: "",
        description: "",
        video_url: "",
        thumbnail: "",
        duration: "",
        category: "getting-started",
        order: 0,
        is_published: false,
    };

    const categoryLabels: Record<string, string> = {
        "getting-started": "Getting Started",
        agents: "Agents & BYOS",
        containers: "Containers",
        cli: "CLI & TUI",
        advanced: "Advanced",
        tips: "Tips & Tricks",
    };

    const categoryIcons: Record<string, string> = {
        "getting-started": "rocket",
        agents: "agent",
        containers: "container",
        cli: "cli",
        advanced: "settings",
        tips: "bolt",
    };

    let fetchError = false;

    async function fetchTutorials() {
        try {
            fetchError = false;
            const endpoint = $isAdmin
                ? "/api/admin/tutorials"
                : "/api/public/tutorials";
            const headers: Record<string, string> = {};

            if ($isAdmin && $auth.token) {
                headers["Authorization"] = `Bearer ${$auth.token}`;
            }

            const res = await fetch(endpoint, { headers });

            if (res.status === 404) {
                // Endpoint not available yet (not deployed) - show empty state gracefully
                tutorials = [];
                categories = {};
                return;
            }

            if (!res.ok) throw new Error("Failed to fetch tutorials");

            const data = await res.json();
            tutorials = data.tutorials || [];

            // Group by category
            categories = {};
            for (const t of tutorials) {
                const cat = t.category || "getting-started";
                if (!categories[cat]) categories[cat] = [];
                categories[cat].push(t);
            }
        } catch (err) {
            console.error("Failed to fetch tutorials:", err);
            fetchError = true;
            tutorials = [];
            categories = {};
        } finally {
            isLoading = false;
        }
    }

    function getEmbedUrl(url: string): string {
        // Convert YouTube watch URLs to embed URLs
        if (url.includes("youtube.com/watch")) {
            const videoId = new URL(url).searchParams.get("v");
            return `https://www.youtube.com/embed/${videoId}`;
        }
        if (url.includes("youtu.be/")) {
            const videoId = url.split("youtu.be/")[1]?.split("?")[0];
            return `https://www.youtube.com/embed/${videoId}`;
        }
        // Convert Vimeo URLs
        if (url.includes("vimeo.com/")) {
            const videoId = url.split("vimeo.com/")[1]?.split("?")[0];
            return `https://player.vimeo.com/video/${videoId}`;
        }

        // Screen Studio URLs
        if (url.includes("screen.studio/share/")) {
            const videoId = url.split("screen.studio/share/")[1]?.split("?")[0];
            return `https://screen.studio/embed/${videoId}`;
        }

        // Loom URLs
        if (url.includes("loom.com/share/")) {
            const videoId = url.split("loom.com/share/")[1]?.split("?")[0];
            return `https://www.loom.com/embed/${videoId}`;
        }
        return url;
    }

    function getThumbnail(tutorial: Tutorial): string {
        if (tutorial.thumbnail) return tutorial.thumbnail;
        // Generate YouTube thumbnail if possible
        if (
            tutorial.video_url.includes("youtube.com") ||
            tutorial.video_url.includes("youtu.be")
        ) {
            let videoId = "";
            if (tutorial.video_url.includes("youtube.com/watch")) {
                videoId =
                    new URL(tutorial.video_url).searchParams.get("v") || "";
            } else if (tutorial.video_url.includes("youtu.be/")) {
                videoId =
                    tutorial.video_url.split("youtu.be/")[1]?.split("?")[0] ||
                    "";
            }
            if (videoId) {
                return `https://img.youtube.com/vi/${videoId}/maxresdefault.jpg`;
            }
        }
        return "/og-image.png";
    }

    function openTutorial(tutorial: Tutorial) {
        selectedTutorial = tutorial;
    }

    function closeTutorial() {
        selectedTutorial = null;
    }

    // Admin functions
    function openAdminModal(tutorial?: Tutorial) {
        if (tutorial) {
            editingTutorial = tutorial;
            formData = {
                title: tutorial.title,
                description: tutorial.description,
                video_url: tutorial.video_url,
                thumbnail: tutorial.thumbnail,
                duration: tutorial.duration,
                category: tutorial.category,
                order: tutorial.order,
                is_published: tutorial.is_published,
            };
        } else {
            editingTutorial = null;
            formData = {
                title: "",
                description: "",
                video_url: "",
                thumbnail: "",
                duration: "",
                category: "getting-started",
                order: 0,
                is_published: false,
            };
        }
        showAdminModal = true;
    }

    function closeAdminModal() {
        showAdminModal = false;
        editingTutorial = null;
    }

    async function saveTutorial() {
        try {
            const url = editingTutorial
                ? `/api/admin/tutorials/${editingTutorial.id}`
                : "/api/admin/tutorials";
            const method = editingTutorial ? "PUT" : "POST";

            const res = await fetch(url, {
                method,
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${$auth.token}`,
                },
                body: JSON.stringify(formData),
            });

            if (!res.ok) throw new Error("Failed to save tutorial");

            toast.success(
                editingTutorial ? "Tutorial updated" : "Tutorial created",
            );
            closeAdminModal();
            await fetchTutorials();
        } catch (err) {
            toast.error("Failed to save tutorial");
        }
    }

    async function deleteTutorial(id: string) {
        if (!confirm("Are you sure you want to delete this tutorial?")) return;

        try {
            const res = await fetch(`/api/admin/tutorials/${id}`, {
                method: "DELETE",
                headers: {
                    Authorization: `Bearer ${$auth.token}`,
                },
            });

            if (!res.ok) throw new Error("Failed to delete tutorial");

            toast.success("Tutorial deleted");
            await fetchTutorials();
        } catch (err) {
            toast.error("Failed to delete tutorial");
        }
    }

    async function togglePublished(tutorial: Tutorial) {
        try {
            const res = await fetch(`/api/admin/tutorials/${tutorial.id}`, {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${$auth.token}`,
                },
                body: JSON.stringify({
                    is_published: !tutorial.is_published,
                }),
            });

            if (!res.ok) throw new Error("Failed to update tutorial");

            toast.success(
                tutorial.is_published
                    ? "Tutorial unpublished"
                    : "Tutorial published",
            );
            await fetchTutorials();
        } catch (err) {
            toast.error("Failed to update tutorial");
        }
    }

    onMount(() => {
        fetchTutorials();
    });

    $: filteredTutorials = selectedCategory
        ? tutorials.filter((t) => t.category === selectedCategory)
        : tutorials;

    $: visibleCategories = Object.keys(categories).sort((a, b) => {
        const order = [
            "getting-started",
            "containers",
            "agents",
            "cli",
            "advanced",
            "tips",
        ];
        return order.indexOf(a) - order.indexOf(b);
    });
</script>

<svelte:head>
    <title>Tutorials | Rexec - Terminal as a Service</title>
    <meta
        name="description"
        content="Learn how to use Rexec with video tutorials covering terminals, agents, CLI tools, and more."
    />
</svelte:head>

<div class="tutorials-page">
    <div class="page-header">
        <div class="header-content">
            <div class="header-badge">
                <StatusIcon status="video" size={14} />
                <span>Video Tutorials</span>
            </div>
            <h1>Learn <span class="accent">Rexec</span></h1>
            <p class="subtitle">
                Master terminal workflows with step-by-step video guides
            </p>
        </div>

        {#if $isAdmin}
            <button class="btn btn-primary" onclick={() => openAdminModal()}>
                <StatusIcon status="add" size={16} />
                Add Tutorial
            </button>
        {/if}
    </div>

    {#if isLoading}
        <div class="loading">
            <div class="spinner"></div>
            <p>Loading tutorials...</p>
        </div>
    {:else if fetchError}
        <div class="empty-state">
            <StatusIcon status="warning" size={48} />
            <h2>Unable to load tutorials</h2>
            <p>Please try again later or check your connection.</p>
            <button
                class="btn btn-secondary"
                onclick={fetchTutorials}
                style="margin-top: 16px;"
            >
                Retry
            </button>
        </div>
    {:else if tutorials.length === 0}
        <div class="empty-state">
            <StatusIcon status="video" size={48} />
            <h2>No tutorials yet</h2>
            <p>Check back soon for video guides on using Rexec.</p>
        </div>
    {:else}
        <!-- Category Filter -->
        <div class="category-filter">
            <button
                class="category-btn"
                class:active={selectedCategory === ""}
                onclick={() => (selectedCategory = "")}
            >
                All
            </button>
            {#each visibleCategories as category}
                <button
                    class="category-btn"
                    class:active={selectedCategory === category}
                    onclick={() => (selectedCategory = category)}
                >
                    <StatusIcon
                        status={categoryIcons[category] || "folder"}
                        size={14}
                    />
                    {categoryLabels[category] || category}
                </button>
            {/each}
        </div>

        <!-- Tutorial Grid -->
        <div class="tutorials-grid">
            {#each filteredTutorials as tutorial (tutorial.id)}
                <div
                    class="tutorial-card"
                    class:unpublished={!tutorial.is_published}
                >
                    <button
                        class="thumbnail"
                        onclick={() => openTutorial(tutorial)}
                    >
                        {#if getThumbnail(tutorial)}
                            <img
                                src={getThumbnail(tutorial)}
                                alt={tutorial.title}
                            />
                        {:else}
                            <div class="placeholder-thumb">
                                <StatusIcon status="video" size={32} />
                            </div>
                        {/if}
                        <div class="play-overlay">
                            <div class="play-button">
                                <StatusIcon status="play" size={24} />
                            </div>
                        </div>
                        {#if tutorial.duration}
                            <span class="duration">{tutorial.duration}</span>
                        {/if}
                        {#if !tutorial.is_published && $isAdmin}
                            <span class="draft-badge">Draft</span>
                        {/if}
                    </button>
                    <div class="tutorial-info">
                        <span class="category-tag">
                            <StatusIcon
                                status={categoryIcons[tutorial.category] ||
                                    "folder"}
                                size={12}
                            />
                            {categoryLabels[tutorial.category] ||
                                tutorial.category}
                        </span>
                        <h3>{tutorial.title}</h3>
                        <p class="description">{tutorial.description}</p>

                        {#if $isAdmin}
                            <div class="admin-actions">
                                <button
                                    class="btn btn-sm btn-ghost"
                                    onclick={() => openAdminModal(tutorial)}
                                >
                                    <StatusIcon status="edit" size={14} />
                                    Edit
                                </button>
                                <button
                                    class="btn btn-sm btn-ghost"
                                    onclick={() => togglePublished(tutorial)}
                                >
                                    <StatusIcon
                                        status={tutorial.is_published
                                            ? "eye-off"
                                            : "eye"}
                                        size={14}
                                    />
                                    {tutorial.is_published
                                        ? "Unpublish"
                                        : "Publish"}
                                </button>
                                <button
                                    class="btn btn-sm btn-ghost danger"
                                    onclick={() => deleteTutorial(tutorial.id)}
                                >
                                    <StatusIcon status="trash" size={14} />
                                </button>
                            </div>
                        {/if}
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<!-- Video Modal -->
{#if selectedTutorial}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
        class="modal-overlay"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        onclick={closeTutorial}
        onkeydown={(e) => e.key === "Escape" && closeTutorial()}
    >
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <div
            class="video-modal"
            role="document"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}
        >
            <div class="modal-header">
                <h2>{selectedTutorial.title}</h2>
                <button class="close-btn" onclick={closeTutorial}>
                    <StatusIcon status="close" size={20} />
                </button>
            </div>
            <div class="video-container">
                <iframe
                    src={getEmbedUrl(selectedTutorial.video_url)}
                    title={selectedTutorial.title}
                    frameborder="0"
                    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                    allowfullscreen
                ></iframe>
            </div>
            <div class="modal-description">
                <p>{selectedTutorial.description}</p>
            </div>
        </div>
    </div>
{/if}

<!-- Admin Modal -->
{#if showAdminModal && $isAdmin}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
        class="modal-overlay"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        onclick={closeAdminModal}
        onkeydown={(e) => e.key === "Escape" && closeAdminModal()}
    >
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <div
            class="admin-modal"
            role="document"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}
        >
            <div class="modal-header">
                <h2>
                    {editingTutorial ? "Edit Tutorial" : "Add Tutorial"}
                </h2>
                <button class="close-btn" onclick={closeAdminModal}>
                    <StatusIcon status="close" size={20} />
                </button>
            </div>
            <form
                onsubmit={(e) => {
                    e.preventDefault();
                    saveTutorial();
                }}
            >
                <div class="form-group">
                    <label for="title">Title *</label>
                    <input
                        type="text"
                        id="title"
                        bind:value={formData.title}
                        required
                        placeholder="Getting Started with Rexec"
                    />
                </div>

                <div class="form-group">
                    <label for="video_url">Video URL *</label>
                    <input
                        type="url"
                        id="video_url"
                        bind:value={formData.video_url}
                        required
                        placeholder="https://youtube.com/watch?v=..."
                    />
                    <span class="hint"
                        >Supports YouTube, Vimeo, and Loom URLs</span
                    >
                </div>

                <div class="form-group">
                    <label for="description">Description</label>
                    <textarea
                        id="description"
                        bind:value={formData.description}
                        rows="3"
                        placeholder="Learn how to create your first terminal..."
                    ></textarea>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label for="category">Category</label>
                        <select id="category" bind:value={formData.category}>
                            <option value="getting-started"
                                >Getting Started</option
                            >
                            <option value="containers">Containers</option>
                            <option value="agents">Agents & BYOS</option>
                            <option value="cli">CLI & TUI</option>
                            <option value="advanced">Advanced</option>
                            <option value="tips">Tips & Tricks</option>
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="duration">Duration</label>
                        <input
                            type="text"
                            id="duration"
                            bind:value={formData.duration}
                            placeholder="5:30"
                        />
                    </div>

                    <div class="form-group">
                        <label for="order">Order</label>
                        <input
                            type="number"
                            id="order"
                            bind:value={formData.order}
                            min="0"
                        />
                    </div>
                </div>

                <div class="form-group">
                    <label for="thumbnail">Thumbnail URL (optional)</label>
                    <input
                        type="url"
                        id="thumbnail"
                        bind:value={formData.thumbnail}
                        placeholder="https://..."
                    />
                    <span class="hint"
                        >Leave empty to auto-generate from YouTube</span
                    >
                </div>

                <div class="form-group checkbox-group">
                    <label>
                        <input
                            type="checkbox"
                            bind:checked={formData.is_published}
                        />
                        <span>Publish immediately</span>
                    </label>
                </div>

                <div class="form-actions">
                    <button
                        type="button"
                        class="btn btn-secondary"
                        onclick={closeAdminModal}
                    >
                        Cancel
                    </button>
                    <button type="submit" class="btn btn-primary">
                        {editingTutorial ? "Save Changes" : "Create Tutorial"}
                    </button>
                </div>
            </form>
        </div>
    </div>
{/if}

<style>
    .tutorials-page {
        max-width: 1200px;
        margin: 0 auto;
        padding: 40px 20px;
    }

    .page-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 40px;
    }

    .header-content {
        flex: 1;
    }

    .header-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 20px;
        font-size: 12px;
        color: var(--text-secondary);
        margin-bottom: 16px;
    }

    h1 {
        font-size: 36px;
        font-weight: 700;
        margin-bottom: 12px;
        letter-spacing: -0.5px;
    }

    h1 .accent {
        color: var(--accent);
    }

    .subtitle {
        font-size: 16px;
        color: var(--text-muted);
    }

    .loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 80px 20px;
        gap: 16px;
    }

    .spinner {
        width: 32px;
        height: 32px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 1s linear infinite;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    .empty-state {
        text-align: center;
        padding: 80px 20px;
        color: var(--text-muted);
    }

    .empty-state h2 {
        margin-top: 16px;
        font-size: 20px;
        color: var(--text);
    }

    .empty-state p {
        margin-top: 8px;
    }

    /* Category Filter */
    .category-filter {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        margin-bottom: 32px;
    }

    .category-btn {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 8px 16px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 20px;
        color: var(--text-secondary);
        font-size: 13px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .category-btn:hover {
        border-color: var(--accent);
        color: var(--text);
    }

    .category-btn.active {
        background: var(--accent);
        border-color: var(--accent);
        color: #000;
    }

    /* Tutorial Grid */
    .tutorials-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 24px;
    }

    .tutorial-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
        overflow: hidden;
        transition: all 0.2s;
    }

    .tutorial-card:hover {
        border-color: var(--accent);
        transform: translateY(-2px);
    }

    .tutorial-card.unpublished {
        opacity: 0.7;
    }

    .thumbnail {
        position: relative;
        width: 100%;
        aspect-ratio: 16/9;
        background: var(--bg-tertiary);
        cursor: pointer;
        border: none;
        padding: 0;
    }

    .thumbnail img {
        width: 100%;
        height: 100%;
        object-fit: cover;
    }

    .placeholder-thumb {
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--text-muted);
    }

    .play-overlay {
        position: absolute;
        inset: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 0, 0, 0.3);
        opacity: 0;
        transition: opacity 0.2s;
    }

    .thumbnail:hover .play-overlay {
        opacity: 1;
    }

    .play-button {
        width: 56px;
        height: 56px;
        background: var(--accent);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #000;
        box-shadow: 0 4px 20px rgba(0, 255, 65, 0.4);
    }

    .duration {
        position: absolute;
        bottom: 8px;
        right: 8px;
        padding: 4px 8px;
        background: rgba(0, 0, 0, 0.8);
        border-radius: 4px;
        font-size: 12px;
        color: #fff;
        font-family: var(--font-mono);
    }

    .draft-badge {
        position: absolute;
        top: 8px;
        left: 8px;
        padding: 4px 8px;
        background: var(--warning);
        border-radius: 4px;
        font-size: 11px;
        color: #000;
        font-weight: 600;
        text-transform: uppercase;
    }

    .tutorial-info {
        padding: 16px;
    }

    .category-tag {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        font-size: 11px;
        color: var(--text-muted);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        margin-bottom: 8px;
    }

    .tutorial-info h3 {
        font-size: 16px;
        font-weight: 600;
        margin-bottom: 8px;
        color: var(--text);
    }

    .tutorial-info .description {
        font-size: 13px;
        color: var(--text-secondary);
        line-height: 1.5;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
    }

    .admin-actions {
        display: flex;
        gap: 8px;
        margin-top: 12px;
        padding-top: 12px;
        border-top: 1px solid var(--border);
    }

    .admin-actions .btn {
        flex: 1;
        font-size: 12px;
    }

    .admin-actions .danger {
        color: var(--error);
    }

    .admin-actions .danger:hover {
        background: rgba(255, 60, 60, 0.1);
    }

    /* Modals */
    .modal-overlay {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        padding: 20px;
    }

    .video-modal {
        background: var(--bg-primary);
        border: 1px solid var(--border);
        border-radius: 12px;
        width: 100%;
        max-width: 900px;
        max-height: 90vh;
        overflow: hidden;
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
    }

    .modal-header h2 {
        font-size: 18px;
        font-weight: 600;
    }

    .close-btn {
        background: none;
        border: none;
        color: var(--text-muted);
        cursor: pointer;
        padding: 4px;
        display: flex;
    }

    .close-btn:hover {
        color: var(--text);
    }

    .video-container {
        position: relative;
        width: 100%;
        padding-top: 56.25%;
        background: #000;
    }

    .video-container iframe {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
    }

    .modal-description {
        padding: 16px 20px;
        color: var(--text-secondary);
        font-size: 14px;
        line-height: 1.6;
        white-space: pre-wrap;
    }

    /* Admin Modal */
    .admin-modal {
        background: var(--bg-primary);
        border: 1px solid var(--border);
        border-radius: 12px;
        width: 100%;
        max-width: 600px;
        max-height: 90vh;
        overflow-y: auto;
    }

    .admin-modal form {
        padding: 20px;
    }

    .form-group {
        margin-bottom: 16px;
    }

    .form-group label {
        display: block;
        font-size: 13px;
        font-weight: 500;
        margin-bottom: 6px;
        color: var(--text);
    }

    .form-group input,
    .form-group textarea,
    .form-group select {
        width: 100%;
        padding: 10px 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        color: var(--text);
        font-size: 14px;
    }

    .form-group input:focus,
    .form-group textarea:focus,
    .form-group select:focus {
        outline: none;
        border-color: var(--accent);
    }

    .form-group .hint {
        font-size: 11px;
        color: var(--text-muted);
        margin-top: 4px;
        display: block;
    }

    .form-row {
        display: grid;
        grid-template-columns: 2fr 1fr 1fr;
        gap: 12px;
    }

    .checkbox-group label {
        display: flex;
        align-items: center;
        gap: 8px;
        cursor: pointer;
    }

    .checkbox-group input[type="checkbox"] {
        width: 18px;
        height: 18px;
        accent-color: var(--accent);
    }

    .form-actions {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        margin-top: 24px;
        padding-top: 16px;
        border-top: 1px solid var(--border);
    }

    .btn {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 10px 16px;
        border-radius: 8px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        border: none;
    }

    .btn-primary {
        background: var(--accent);
        color: #000;
    }

    .btn-primary:hover {
        filter: brightness(1.1);
    }

    .btn-secondary {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        color: var(--text);
    }

    .btn-secondary:hover {
        border-color: var(--text-muted);
    }

    .btn-ghost {
        background: transparent;
        color: var(--text-secondary);
    }

    .btn-ghost:hover {
        background: var(--bg-secondary);
        color: var(--text);
    }

    .btn-sm {
        padding: 6px 10px;
        font-size: 12px;
    }

    @media (max-width: 768px) {
        .page-header {
            flex-direction: column;
            gap: 16px;
        }

        .tutorials-grid {
            grid-template-columns: 1fr;
        }

        .form-row {
            grid-template-columns: 1fr;
        }
    }
</style>
