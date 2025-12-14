<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { admin } from "$stores/admin";
    import { formatRelativeTime, formatMemory, formatStorage, formatCPU } from "$utils/api";
    import PlatformIcon from "./icons/PlatformIcon.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    // Tabs
    type Tab = "users" | "subscribers" | "containers" | "terminals" | "agents";
    let activeTab: Tab = "users";

    function setTab(tab: Tab) {
        activeTab = tab;
    }

    async function loadData() {
        await Promise.all([
            admin.fetchUsers(),
            admin.fetchContainers(),
            admin.fetchTerminals(),
            admin.fetchAgents()
        ]);
    }

    // Load initial data on mount and start WebSocket
    onMount(async () => {
        await loadData(); // Initial data fetch
        admin.startAdminEvents(); // Start WebSocket for live updates
    });

    // Clean up WebSocket on destroy
    onDestroy(() => {
        admin.stopAdminEvents();
    });

    $: users = $admin.users;
    $: subscribers = users.filter((u) => u.subscriptionActive === true);
    $: containers = $admin.containers;
    $: terminals = $admin.terminals;
    $: agents = $admin.agents;
    $: isLoading = $admin.isLoading;
    $: wsConnected = $admin.wsConnected;
    $: wsError = $admin.error;

    // Actions
    async function handleDeleteUser(userId: string) {
        if (confirm("Are you sure you want to delete this user? This action cannot be undone.")) {
            await admin.deleteUser(userId);
        }
    }

    async function handleDeleteContainer(containerId: string) {
        if (confirm("Are you sure you want to delete this container? This will stop and remove the container permanently.")) {
            await admin.deleteContainer(containerId);
        }
    }

    function getDistro(image: string): string {
         if (!image) return "linux";
         const lower = image.toLowerCase();
         if (lower.includes("ubuntu")) return "ubuntu";
         if (lower.includes("debian")) return "debian";
         if (lower.includes("alpine")) return "alpine";
         return "linux";
    }

</script>

<div class="dashboard">
    <div class="dashboard-header">
        <div class="dashboard-title">
            <h1>Admin Dashboard</h1>
            {#if !wsConnected}
                <span class="status-indicator error">Disconnected</span>
            {:else}
                <span class="status-indicator connected">Live</span>
            {/if}
        </div>
        <div class="dashboard-actions">
            {#if wsError}
                <div class="alert alert-error">
                    {wsError}
                </div>
            {/if}
            <button
                class="btn btn-secondary btn-sm"
                onclick={loadData}
                disabled={isLoading}
            >
                <svg
                    class="icon"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <path
                        d="M23 4v6h-6M1 20v-6h6M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"
                    />
                </svg>
                Refresh
            </button>
        </div>
    </div>

    <!-- Tabs -->
    <div class="tabs">
        <button
            class="tab-btn"
            class:active={activeTab === "users"}
            onclick={() => setTab("users")}
        >
            Users ({users.length})
        </button>
        <button
            class="tab-btn"
            class:active={activeTab === "subscribers"}
            onclick={() => setTab("subscribers")}
        >
            Subscribers ({subscribers.length})
        </button>
        <button
            class="tab-btn"
            class:active={activeTab === "containers"}
            onclick={() => setTab("containers")}
        >
            Containers ({containers.length})
        </button>
        <button
            class="tab-btn"
            class:active={activeTab === "terminals"}
            onclick={() => setTab("terminals")}
        >
            Active Terminals ({terminals.length})
        </button>
        <button
            class="tab-btn"
            class:active={activeTab === "agents"}
            onclick={() => setTab("agents")}
        >
            Agents ({agents.length})
        </button>
    </div>

    {#if isLoading}
        <div class="loading-state">
            <div class="spinner"></div>
            <p>Loading data...</p>
        </div>
    {:else}
        <div class="tab-content">
            {#if activeTab === "users"}
                <div class="data-table-container">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>User</th>
                                <th>Email</th>
                                <th>Role</th>
                                <th>Tier</th>
                                <th>Containers</th>
                                <th>Created</th>
                                <th>Last Login</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each users as user (user.id)}
                                <tr>
                                    <td>
                                        <div class="user-info">
                                            <div class="avatar">{user.username.charAt(0).toUpperCase()}</div>
                                            <span>{user.username}</span>
                                        </div>
                                    </td>
                                    <td>{user.email}</td>
                                    <td>
                                        {#if user.isAdmin}
                                            <span class="badge admin">Admin</span>
                                        {:else}
                                            <span class="badge user">User</span>
                                        {/if}
                                    </td>
                                    <td><span class="badge tier-{user.tier}">{user.tier}</span></td>
                                    <td>{user.containerCount}</td>
                                    <td>{new Date(user.created_at).toLocaleDateString()}</td>
                                    <td>{user.updated_at ? formatRelativeTime(user.updated_at) : '-'}</td>
                                    <td>
                                        <button class="btn-icon danger" onclick={() => handleDeleteUser(user.id)} title="Delete User">
                                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                                            </svg>
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {:else if activeTab === "subscribers"}
                <div class="data-table-container">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>User</th>
                                <th>Email</th>
                                <th>Tier</th>
                                <th>Containers</th>
                                <th>Created</th>
                                <th>Last Login</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each subscribers as user (user.id)}
                                <tr>
                                    <td>
                                        <div class="user-info">
                                            <div class="avatar">{user.username.charAt(0).toUpperCase()}</div>
                                            <span>{user.username}</span>
                                        </div>
                                    </td>
                                    <td>{user.email}</td>
                                    <td><span class="badge tier-{user.tier}">{user.tier}</span></td>
                                    <td>{user.containerCount}</td>
                                    <td>{new Date(user.created_at).toLocaleDateString()}</td>
                                    <td>{user.updated_at ? formatRelativeTime(user.updated_at) : "-"}</td>
                                    <td>
                                        <button
                                            class="btn-icon danger"
                                            onclick={() => handleDeleteUser(user.id)}
                                            title="Delete User"
                                        >
                                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                                            </svg>
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {:else if activeTab === "containers"}
                <div class="data-table-container">
                    <table class="data-table">
                         <thead>
                            <tr>
                                <th>Name</th>
                                <th>User</th>
                                <th>Image</th>
                                <th>Status</th>
                                <th>Resources</th>
                                <th>Created</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each containers as container (container.id)}
                                <tr>
                                    <td>
                                        <div class="container-name-cell">
                                            <PlatformIcon platform={getDistro(container.image)} size={16} />
                                            <span>{container.name}</span>
                                        </div>
                                    </td>
                                    <td>
                                        <div class="user-cell">
                                            <span class="user-name">{container.username}</span>
                                            <span class="user-email">{container.userEmail}</span>
                                        </div>
                                    </td>
                                    <td class="mono">{container.image}</td>
                                    <td>
                                        <span class="status-badge {container.status}">
                                            {container.status}
                                        </span>
                                    </td>
                                    <td>
                                        {#if container.resources}
                                            <div class="resources-cell">
                                                <span>{formatMemory(container.resources.memory_mb)}</span>
                                                <span>/</span>
                                                <span>{formatCPU(container.resources.cpu_shares)}</span>
                                            </div>
                                        {:else}
                                            -
                                        {/if}
                                    </td>
                                    <td>{formatRelativeTime(container.created_at)}</td>
                                    <td>
                                        <button class="btn-icon danger" onclick={() => handleDeleteContainer(container.id)} title="Delete Container">
                                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                                            </svg>
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {:else if activeTab === "terminals"}
                <div class="data-table-container">
                     <table class="data-table">
                        <thead>
                            <tr>
                                <th>Terminal ID</th>
                                <th>User</th>
                                <th>Container</th>
                                <th>Status</th>
                                <th>Connected At</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each terminals as term (term.id)}
                                <tr>
                                    <td class="mono">{term.id}</td>
                                    <td>{term.username}</td>
                                    <td>{term.name}</td>
                                    <td>
                                        <span class="status-badge {term.status}">
                                            {term.status}
                                        </span>
                                    </td>
                                    <td>{formatRelativeTime(term.connected_at)}</td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {:else if activeTab === "agents"}
                <div class="data-table-container">
                     <table class="data-table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>User</th>
                                <th>Status</th>
                                <th>Platform</th>
                                <th>Specs</th>
                                <th>Last Seen</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each agents as agent (agent.id)}
                                <tr>
                                    <td>
                                        <div class="container-name-cell">
                                            <PlatformIcon platform={agent.os} size={16} />
                                            <span>{agent.name}</span>
                                        </div>
                                    </td>
                                    <td>{agent.username || agent.user_id.slice(0, 8)}</td>
                                    <td>
                                        <span class="status-badge {agent.status}">
                                            {agent.status}
                                        </span>
                                    </td>
                                    <td class="mono">{agent.os}/{agent.arch}</td>
                                    <td>
                                        {#if agent.system_info && agent.system_info.memory && agent.system_info.num_cpu}
                                            <div class="resources-cell">
                                                <span>{formatMemory(Math.round((agent.system_info.memory.total || 0) / 1024 / 1024))}</span>
                                                <span>/</span>
                                                <span>{agent.system_info.num_cpu} CPU</span>
                                            </div>
                                        {:else}
                                            -
                                        {/if}
                                    </td>
                                    <td>{agent.last_ping ? formatRelativeTime(agent.last_ping) : '-'}</td>
                                    <td>
                                        <button class="btn-icon danger" onclick={() => handleDeleteUser(agent.id)} title="Delete Agent (TODO)">
                                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2" />
                                            </svg>
                                        </button>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .dashboard {
        animation: fadeIn 0.2s ease;
        padding: 20px;
    }

    .dashboard-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 24px;
        padding-bottom: 16px;
        border-bottom: 1px solid var(--border);
    }

    .dashboard-title {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .dashboard-title h1 {
        font-size: 20px;
        text-transform: uppercase;
        letter-spacing: 1px;
        margin: 0;
    }

    .status-indicator {
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        padding: 4px 8px;
        border-radius: 12px;
    }

    .status-indicator.connected {
        background: rgba(0, 255, 65, 0.1);
        color: var(--green);
        border: 1px solid rgba(0, 255, 65, 0.2);
    }

    .status-indicator.error, .status-indicator.disconnected {
        background: rgba(255, 0, 0, 0.1);
        color: var(--red);
        border: 1px solid rgba(255, 0, 0, 0.2);
    }

    .alert.alert-error {
        background: rgba(255, 0, 0, 0.1);
        color: var(--red);
        padding: 8px 16px;
        border-radius: 4px;
        border: 1px solid rgba(255, 0, 0, 0.2);
        margin-right: 16px;
        font-size: 13px;
    }


    /* Tabs */
    .tabs {
        display: flex;
        gap: 16px;
        margin-bottom: 24px;
        border-bottom: 1px solid var(--border);
    }

    .tab-btn {
        background: none;
        border: none;
        padding: 12px 16px;
        color: var(--text-muted);
        font-size: 14px;
        cursor: pointer;
        position: relative;
    }

    .tab-btn:hover {
        color: var(--text);
    }

    .tab-btn.active {
        color: var(--accent);
    }

    .tab-btn.active::after {
        content: "";
        position: absolute;
        bottom: -1px;
        left: 0;
        width: 100%;
        height: 2px;
        background: var(--accent);
    }

    /* Table */
    .data-table-container {
        overflow-x: auto;
        border: 1px solid var(--border);
        border-radius: 4px;
    }

    .data-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 14px;
    }

    .data-table th, .data-table td {
        padding: 12px 16px;
        text-align: left;
        border-bottom: 1px solid var(--border);
    }

    .data-table th {
        background: var(--bg-secondary);
        color: var(--text-muted);
        text-transform: uppercase;
        font-size: 11px;
        font-weight: 600;
    }

    .data-table tr:last-child td {
        border-bottom: none;
    }

    .data-table tr:hover {
        background: var(--bg-secondary);
    }
    
    .mono {
        font-family: var(--font-mono);
        font-size: 12px;
    }

    /* User Info */
    .user-info {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .avatar {
        width: 24px;
        height: 24px;
        background: var(--accent);
        color: var(--bg);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: bold;
        font-size: 12px;
    }

    .user-cell {
        display: flex;
        flex-direction: column;
    }
    
    .user-email {
        font-size: 11px;
        color: var(--text-muted);
    }

    /* Badges */
    .badge {
        display: inline-block;
        padding: 2px 6px;
        border-radius: 4px;
        font-size: 10px;
        text-transform: uppercase;
        font-weight: bold;
    }

    .badge.admin {
        background: var(--accent);
        color: var(--bg);
    }

    .badge.user {
        background: var(--bg-tertiary);
        color: var(--text-muted);
        border: 1px solid var(--border);
    }
    
    .badge.tier-free {
        background: var(--bg-tertiary);
        color: var(--text);
    }
    
    .badge.tier-pro {
        background: #ffd700;
        color: #000;
    }
    
    .badge.tier-guest {
        background: var(--bg-tertiary);
        color: var(--text-muted);
    }

    .status-badge {
        display: inline-flex;
        align-items: center;
        padding: 2px 8px;
        border-radius: 12px;
        font-size: 11px;
        text-transform: uppercase;
        font-weight: 600;
    }
    
    .status-badge.running, .status-badge.connected {
        background: rgba(0, 255, 65, 0.1);
        color: var(--green);
        border: 1px solid rgba(0, 255, 65, 0.2);
    }
    
    .status-badge.stopped, .status-badge.disconnected {
        background: var(--bg-tertiary);
        color: var(--text-muted);
        border: 1px solid var(--border);
    }

    /* Container Info */
    .container-name-cell {
        display: flex;
        align-items: center;
        gap: 8px;
    }
    
    .resources-cell {
        display: flex;
        gap: 4px;
        font-family: var(--font-mono);
        font-size: 11px;
        color: var(--text-secondary);
    }

    /* Buttons */
    .btn-icon {
        background: none;
        border: none;
        padding: 4px;
        cursor: pointer;
        color: var(--text-muted);
        border-radius: 4px;
    }
    
    .btn-icon:hover {
        background: var(--bg-tertiary);
        color: var(--text);
    }
    
    .btn-icon.danger:hover {
        color: var(--red);
        background: rgba(255, 0, 0, 0.1);
    }
    
    .btn-icon svg {
        width: 16px;
        height: 16px;
    }

    /* Loading */
    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 60px 20px;
        gap: 16px;
        color: var(--text-muted);
    }
    
    .spinner {
        width: 24px;
        height: 24px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    @keyframes fadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }
</style>
