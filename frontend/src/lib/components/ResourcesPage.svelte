<script lang="ts">
    import { onMount } from "svelte";
    import { auth, isAdmin } from "$stores/auth";
    import StatusIcon from "./icons/StatusIcon.svelte";
    import { toast } from "$stores/toast";

    interface Resource {
        id: string;
        title: string;
        description: string;
        type: "video" | "guide";
        content: string;
        video_url: string;
        thumbnail: string;
        duration: string;
        category: string;
        order: number;
        is_published: boolean;
        created_at: string;
        updated_at: string;
    }

    let resources: Resource[] = [];
    let categories: Record<string, Resource[]> = {};
    let isLoading = true;
    let selectedCategory = "";
    let selectedResource: Resource | null = null;
    let playingId: string | null = null;

    $: filteredResources = selectedCategory
        ? resources.filter((t) => t.category === selectedCategory)
        : resources;

    $: visibleCategories = Object.keys(categories).filter(
        (c) => categories[c].length > 0,
    );

    // Admin state
    let showAdminModal = false;
    let editingResource: Resource | null = null;
    let isUploading = false;
    let formData = {
        title: "",
        description: "",
        type: "video" as "video" | "guide",
        content: "",
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

    async function fetchResources() {
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
                resources = [];
                categories = {};
                return;
            }

            if (!res.ok) throw new Error("Failed to fetch resources");

            const data = await res.json();
            resources = data.tutorials || [];

            // Group by category
            categories = {};
            for (const t of resources) {
                if (!t.id) continue;
                const cat = t.category || "getting-started";
                if (!categories[cat]) categories[cat] = [];
                categories[cat].push(t);
            }
        } catch (err) {
            console.error("Failed to fetch resources:", err);
            fetchError = true;
            resources = [];
            categories = {};
        } finally {
            isLoading = false;
        }
    }

    function getEmbedUrl(url: string, autoplay = false): string {
        const params = autoplay ? "?autoplay=1&vq=hd2160" : "?vq=hd2160";

        // Handle YouTube URLs (supports various formats including shorts, live, etc.)
        const ytMatch = url.match(
            /(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/\s]{11})/,
        );
        if (ytMatch && ytMatch[1]) {
            return `https://www.youtube.com/embed/${ytMatch[1]}${params}`;
        }

        // Convert Vimeo URLs
        if (url.includes("vimeo.com/")) {
            const videoId = url.split("vimeo.com/")[1]?.split("?")[0];
            return `https://player.vimeo.com/video/${videoId}${params}`;
        }

        // Screen Studio URLs
        if (url.includes("screen.studio/share/")) {
            const videoId = url.split("screen.studio/share/")[1]?.split("?")[0];
            return `https://screen.studio/embed/${videoId}${params}`;
        }

        // Loom URLs
        if (url.includes("loom.com/share/")) {
            const videoId = url.split("loom.com/share/")[1]?.split("?")[0];
            return `https://www.loom.com/embed/${videoId}${params}`;
        }
        return url;
    }

    function getThumbnail(resource: Resource): string {
        if (resource.thumbnail) return resource.thumbnail;
        // Generate YouTube thumbnail if possible for videos
        if (resource.type === "video" && resource.video_url) {
            const ytMatch = resource.video_url.match(
                /(?:youtube\.com\/(?:[^\/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?\/\s]{11})/,
            );
            if (ytMatch && ytMatch[1]) {
                return `https://img.youtube.com/vi/${ytMatch[1]}/hqdefault.jpg`;
            }
        }
        return "/og-image.png";
    }

    function linkify(text: string): string {
        if (!text) return "";

        const escapeHtml = (str: string) =>
            str
                .replace(/&/g, "&amp;")
                .replace(/</g, "&lt;")
                .replace(/>/g, "&gt;")
                .replace(/"/g, "&quot;")
                .replace(/'/g, "&#039;");

        const parts = text.split(/(https?:\/\/[^\s]+)/g);

        return parts
            .map((part, index) => {
                if (index % 2 === 1) {
                    const match = part.match(/([.,;:"']+)$/);
                    let url = part;
                    let suffix = "";

                    if (match) {
                        suffix = match[1];
                        url = part.slice(0, -suffix.length);
                    }

                    if (!url) return escapeHtml(suffix);

                    return `<a href="${escapeHtml(url)}" target="_blank" rel="noopener noreferrer">${escapeHtml(url)}</a>${escapeHtml(suffix)}`;
                }
                return escapeHtml(part);
            })
            .join("");
    }

    function renderMarkdown(text: string): string {
        if (!text) return "";
        let html = text
            .replace(/^### (.*$)/gim, "<h3>$1</h3>")
            .replace(/^## (.*$)/gim, "<h2>$1</h2>")
            .replace(/^# (.*$)/gim, "<h1>$1</h1>")
            .replace(/\*\*(.*)\*\*/gim, "<strong>$1</strong>")
            .replace(/\*(.*)\*/gim, "<em>$1</em>")
            .replace(
                /!\[(.*?)\]\((.*?)\)/gim,
                "<img src='$2' alt='$1' class='img-fluid' />",
            )
            .replace(
                /\[(.*?)\]\((.*?)\)/gim,
                "<a href='$2' target='_blank' rel='noopener'>$1</a>",
            )
            .replace(/```([^`]+)```/gim, "<pre><code>$1</code></pre>")
            .replace(/`([^`]+)`/gim, "<code>$1</code>")
            .replace(/^\s*-\s+(.*)/gim, "<ul><li>$1</li></ul>")
            .replace(/<\/ul>\s*<ul>/gim, "")
            .replace(/\n\n/gim, "</p><p>");

        return `<p>${html}</p>`;
    }

    async function handleImageUpload(e: Event) {
        const input = e.target as HTMLInputElement;
        if (!input.files || !input.files[0]) return;

        const file = input.files[0];
        const fd = new FormData();
        fd.append("image", file);

        isUploading = true;
        try {
            const res = await fetch("/api/admin/tutorials/images", {
                method: "POST",
                headers: {
                    Authorization: `Bearer ${$auth.token}`,
                },
                body: fd,
            });

            if (!res.ok) throw new Error("Upload failed");

            const data = await res.json();
            formData.content =
                (formData.content || "") + `\n![${file.name}](${data.url})`;
            toast.success("Image uploaded");
        } catch (err) {
            console.error(err);
            toast.error("Failed to upload image");
        } finally {
            isUploading = false;
            input.value = "";
        }
    }

    function openResource(resource: Resource) {
        selectedResource = resource;
        const url = new URL(window.location.href);
        url.searchParams.set("id", resource.id);
        window.history.pushState({}, "", url);
        document.title = `${resource.title} | Rexec Resources`;
    }

    function closeResource() {
        selectedResource = null;
        playingId = null;
        const url = new URL(window.location.href);
        url.searchParams.delete("id");
        window.history.pushState({}, "", url);
        document.title = "Resources - Rexec | Tutorials & Guides";
    }

    function openAdminModal(resource?: Resource) {
        if (resource) {
            editingResource = resource;
            formData = {
                title: resource.title,
                description: resource.description,
                type: (resource.type as "video" | "guide") || "video",
                content: resource.content || "",
                video_url: resource.video_url || "",
                thumbnail: resource.thumbnail,
                duration: resource.duration,
                category: resource.category,
                order: resource.order,
                is_published: resource.is_published,
            };
        } else {
            editingResource = null;
            formData = {
                title: "",
                description: "",
                type: "video",
                content: "",
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
        editingResource = null;
    }

    async function saveResource() {
        try {
            const url = editingResource
                ? `/api/admin/tutorials/${editingResource.id}`
                : "/api/admin/tutorials";
            const method = editingResource ? "PUT" : "POST";

            const res = await fetch(url, {
                method,
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${$auth.token}`,
                },
                body: JSON.stringify(formData),
            });

            if (!res.ok) throw new Error("Failed to save resource");

            toast.success(
                editingResource ? "Resource updated" : "Resource created",
            );
            closeAdminModal();
            await fetchResources();
        } catch (err) {
            toast.error("Failed to save resource");
        }
    }

    async function deleteResource(id: string) {
        if (!confirm("Are you sure you want to delete this resource?")) return;

        try {
            const res = await fetch(`/api/admin/tutorials/${id}`, {
                method: "DELETE",
                headers: {
                    Authorization: `Bearer ${$auth.token}`,
                },
            });

            if (!res.ok) throw new Error("Failed to delete resource");

            toast.success("Resource deleted");
            await fetchResources();
        } catch (err) {
            toast.error("Failed to delete resource");
        }
    }

    async function togglePublished(resource: Resource) {
        try {
            const res = await fetch(`/api/admin/tutorials/${resource.id}`, {
                method: "PUT",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${$auth.token}`,
                },
                body: JSON.stringify({
                    is_published: !resource.is_published,
                }),
            });

            if (!res.ok) throw new Error("Failed to update resource");

            toast.success(
                resource.is_published
                    ? "Resource unpublished"
                    : "Resource published",
            );
            await fetchResources();
        } catch (err) {
            toast.error("Failed to update resource");
        }
    }

    onMount(async () => {
        await fetchResources();
        const params = new URLSearchParams(window.location.search);
        const id = params.get("id");
        if (id) {
            const resource = resources.find((r) => r.id === id);
            if (resource) openResource(resource);
        }
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
                <StatusIcon status="book" size={14} />
                <span>Learning Resources</span>
            </div>
            <h1>
                Master Rexec with <span class="accent">Guides & Tutorials</span>
            </h1>
            <p class="subtitle">
                Learn how to create terminals, manage agents, and automate your
                workflows with videos and step-by-step guides.
            </p>
        </div>
        {#if $isAdmin}
            <button class="btn btn-primary" onclick={() => openAdminModal()}>
                <StatusIcon status="add" size={16} />
                Add Resource
            </button>
        {/if}
    </div>

    {#if isLoading}
        <div class="loading">
            <div class="spinner"></div>
            <p>Loading resources...</p>
        </div>
    {:else if fetchError}
        <div class="empty-state">
            <StatusIcon status="warning" size={48} />
            <h2>Failed to load resources</h2>
            <p>Something went wrong. Please try again later.</p>
            <button class="btn btn-secondary" onclick={fetchResources}>
                Retry
            </button>
        </div>
    {:else if resources.length === 0}
        <div class="empty-state">
            <StatusIcon status="video" size={48} />
            <h2>No resources yet</h2>
            <p>Check back soon for guides and videos on using Rexec.</p>
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
            {#each filteredResources as resource (resource.id)}
                <div
                    class="tutorial-card"
                    class:unpublished={!resource.is_published}
                >
                    {@const thumb = getThumbnail(resource)}
                    <button
                        class="thumbnail"
                        onclick={() => openResource(resource)}
                    >
                        {#if thumb}
                            <img src={thumb} alt={resource.title} />
                        {:else}
                            <div class="placeholder-thumb">
                                <StatusIcon
                                    status={resource.type === "guide"
                                        ? "book"
                                        : "video"}
                                    size={32}
                                />
                            </div>
                        {/if}

                        {#if resource.type === "video"}
                            <div class="play-overlay">
                                <div class="play-button">
                                    <StatusIcon status="play" size={24} />
                                </div>
                            </div>
                        {:else}
                            <div class="type-badge">
                                <StatusIcon status="book" size={12} /> Guide
                            </div>
                        {/if}

                        {#if resource.duration}
                            <span class="duration">{resource.duration}</span>
                        {/if}
                        {#if !resource.is_published && $isAdmin}
                            <span class="draft-badge">Draft</span>
                        {/if}
                    </button>
                    <div class="tutorial-info">
                        <span class="category-tag">
                            <StatusIcon
                                status={categoryIcons[resource.category] ||
                                    "folder"}
                                size={12}
                            />
                            {categoryLabels[resource.category] ||
                                resource.category}
                        </span>
                        <h3>
                            <a
                                href="?id={resource.id}"
                                class="clickable-title"
                                onclick={(e) => {
                                    e.preventDefault();
                                    openResource(resource);
                                }}
                            >
                                {resource.title}
                            </a>
                        </h3>
                        <p class="description">{resource.description}</p>

                        {#if $isAdmin}
                            <div class="admin-actions">
                                <button
                                    class="btn btn-sm btn-ghost"
                                    onclick={() => openAdminModal(resource)}
                                >
                                    <StatusIcon status="edit" size={14} />
                                    Edit
                                </button>
                                <button
                                    class="btn btn-sm btn-ghost"
                                    onclick={() => togglePublished(resource)}
                                >
                                    <StatusIcon
                                        status={resource.is_published
                                            ? "eye-off"
                                            : "eye"}
                                        size={14}
                                    />
                                    {resource.is_published
                                        ? "Unpublish"
                                        : "Publish"}
                                </button>
                                <button
                                    class="btn btn-sm btn-ghost danger"
                                    onclick={() => deleteResource(resource.id)}
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
{#if selectedResource}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
        class="modal-overlay"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        onclick={closeResource}
        onkeydown={(e) => e.key === "Escape" && closeResource()}
    >
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <div
            class="video-modal"
            class:guide-modal={selectedResource.type === "guide"}
            role="document"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}
        >
            <div class="modal-header">
                <h2>{selectedResource.title}</h2>
                <button class="close-btn" onclick={closeResource}>
                    <StatusIcon status="close" size={20} />
                </button>
            </div>

            {#if selectedResource.type === "video"}
                <div class="video-container">
                    <iframe
                        src={getEmbedUrl(selectedResource.video_url)}
                        title={selectedResource.title}
                        frameborder="0"
                        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                        allowfullscreen
                    ></iframe>
                </div>
                <div class="modal-description">
                    <h3>Description</h3>
                    <p>{@html linkify(selectedResource.description)}</p>
                </div>
            {:else}
                <div class="guide-content">
                    {#if selectedResource.thumbnail && !selectedResource.content.includes(selectedResource.thumbnail)}
                        <img
                            src={selectedResource.thumbnail}
                            alt={selectedResource.title}
                            class="guide-hero"
                        />
                    {/if}
                    <div class="markdown-body">
                        {@html renderMarkdown(selectedResource.content)}
                    </div>
                </div>
            {/if}
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
                    {editingResource ? "Edit Resource" : "Add Resource"}
                </h2>
                <button class="close-btn" onclick={closeAdminModal}>
                    <StatusIcon status="close" size={20} />
                </button>
            </div>
            <form
                onsubmit={(e) => {
                    e.preventDefault();
                    saveResource();
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

                <div class="form-row">
                    <div class="form-group">
                        <label for="type">Type</label>
                        <select id="type" bind:value={formData.type}>
                            <option value="video">Video Tutorial</option>
                            <option value="guide">Written Guide</option>
                        </select>
                    </div>
                </div>

                {#if formData.type === "video"}
                    <div class="form-group">
                        <label for="video_url">Video URL *</label>
                        <input
                            type="url"
                            id="video_url"
                            bind:value={formData.video_url}
                            required={formData.type === "video"}
                            placeholder="https://youtube.com/watch?v=..."
                        />
                        <span class="hint"
                            >YouTube, Vimeo, Screen Studio, or Loom URL</span
                        >
                    </div>
                {:else}
                    <div class="form-group">
                        <div
                            class="toolbar"
                            style="display: flex; justify-content: space-between; align-items: center;"
                        >
                            <label for="content" style="margin: 0;"
                                >Content (Markdown) *</label
                            >
                            <label class="btn btn-sm btn-secondary upload-btn">
                                {#if isUploading}
                                    <div class="spinner small"></div>
                                    Uploading...
                                {:else}
                                    <StatusIcon status="upload" size={14} />
                                    Upload Image
                                {/if}
                                <input
                                    type="file"
                                    accept="image/*"
                                    onchange={handleImageUpload}
                                    style="display: none;"
                                    disabled={isUploading}
                                />
                            </label>
                        </div>
                        <textarea
                            id="content"
                            bind:value={formData.content}
                            rows="15"
                            required={formData.type === "guide"}
                            placeholder="# Introduction&#10;&#10;Write your guide here..."
                            style="font-family: monospace;"
                        ></textarea>
                    </div>
                {/if}

                <div class="form-group">
                    <label for="description">Description</label>
                    <textarea
                        id="description"
                        bind:value={formData.description}
                        rows="3"
                        placeholder="Brief description of what this covers..."
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
                        {editingResource ? "Save Changes" : "Create Resource"}
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
        background: rgba(0, 0, 0, 0.4);
        display: flex;
        align-items: center;
        justify-content: center;
        opacity: 0;
        transition: opacity 0.2s;
    }

    .thumbnail:hover .play-overlay {
        opacity: 1;
    }

    .play-button {
        width: 48px;
        height: 48px;
        background: var(--accent);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #000;
        transform: scale(0.9);
        transition: transform 0.2s;
    }

    .type-badge {
        position: absolute;
        top: 8px;
        left: 8px;
        background: rgba(0, 0, 0, 0.8);
        color: var(--text);
        padding: 4px 8px;
        border-radius: 4px;
        font-size: 11px;
        font-weight: 600;
        display: flex;
        align-items: center;
        gap: 4px;
        border: 1px solid var(--border);
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
        line-height: 1.4;
    }

    .tutorial-info h3 .clickable-title {
        cursor: pointer;
        transition: color 0.2s;
        text-decoration: none;
        color: inherit;
    }

    .tutorial-info h3 .clickable-title:hover {
        color: var(--accent);
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
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
        width: 100%;
        max-width: 900px;
        max-height: 90vh;
        overflow: hidden;
        display: flex;
        flex-direction: column;
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
        box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
        z-index: 10;
    }

    .video-container iframe {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
    }

    .modal-description {
        padding: 24px;
        color: var(--text-secondary);
        font-size: 14px;
        line-height: 1.6;
        white-space: pre-wrap;
        overflow-y: auto;
        background: var(--bg-secondary);
        border-top: 1px solid var(--border);
    }

    .modal-description h3 {
        margin: 0 0 12px 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--text);
    }

    .modal-description :global(a) {
        color: var(--accent);
        text-decoration: none;
    }

    .modal-description :global(a:hover) {
        text-decoration: underline;
    }

    /* Guide Modal */
    .guide-modal {
        overflow-y: auto;
        display: block; /* Override flex */
    }

    .guide-content {
        padding: 24px;
        background: var(--bg-card);
        color: var(--text);
    }

    .guide-hero {
        width: 100%;
        max-height: 400px;
        object-fit: cover;
        border-radius: 8px;
        margin-bottom: 24px;
    }

    .markdown-body :global(h1),
    .markdown-body :global(h2),
    .markdown-body :global(h3) {
        color: var(--text);
        margin-top: 24px;
        margin-bottom: 16px;
    }

    .markdown-body :global(p) {
        margin-bottom: 16px;
        line-height: 1.7;
    }

    .markdown-body :global(img) {
        max-width: 100%;
        border-radius: 8px;
        margin: 16px 0;
    }

    .markdown-body :global(pre) {
        background: var(--bg-tertiary);
        padding: 16px;
        border-radius: 8px;
        overflow-x: auto;
        margin-bottom: 16px;
    }

    .markdown-body :global(code) {
        font-family: "JetBrains Mono", monospace;
        font-size: 13px;
    }

    .toolbar {
        margin-bottom: 8px;
    }

    .upload-btn {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        cursor: pointer;
    }

    .spinner.small {
        width: 14px;
        height: 14px;
        border-width: 2px;
    }

    /* Admin Modal */
    .admin-modal {
        background: var(--bg-card);
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
