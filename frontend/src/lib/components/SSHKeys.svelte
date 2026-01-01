<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { toast } from "$stores/toast";
    import { api } from "$utils/api";
    import { isGuest, auth } from "$stores/auth";
    import ConfirmModal from "./ConfirmModal.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    const dispatch = createEventDispatcher<{
        back: void;
        run: { command: string };
    }>();

    interface SSHKey {
        id: string;
        name: string;
        fingerprint: string;
        created_at: string;
        last_used_at?: string;
    }

    interface RemoteHost {
        id: string;
        name: string;
        hostname: string;
        port: number;
        username: string;
        identity_file?: string;
        ssh_command: string;
        created_at: string;
    }

    let activeTab: "keys" | "hosts" = "hosts";
    let keys: SSHKey[] = [];
    let hosts: RemoteHost[] = [];
    let isLoading = true;
    let showAddModal = false;

    // Add Host state
    let newHostName = "";
    let newHostAddress = "";
    let newHostPort = 22;
    let newHostUser = "root";
    let newHostKeyPath = "";

    let isAdding = false;
    let isOAuthLoading = false;

    // Delete state
    let showDeleteConfirm = false;
    let itemToDelete: {
        id: string;
        name: string;
        type: "key" | "host";
    } | null = null;

    async function handleOAuthLogin() {
        if (isOAuthLoading) return;

        isOAuthLoading = true;
        try {
            const url = await auth.getOAuthUrl();
            if (url) {
                window.location.href = url;
            } else {
                toast.error(
                    "Unable to connect to PipeOps. Please try again later.",
                );
                isOAuthLoading = false;
            }
        } catch (e) {
            toast.error("Failed to connect to PipeOps. Please try again.");
            isOAuthLoading = false;
        }
    }

    // Load data based on tab
    async function loadData() {
        isLoading = true;
        if (activeTab === "keys") {
            const { data, error } = await api.get<{ keys: SSHKey[] }>(
                "/api/ssh/keys",
            );
            if (data) keys = data.keys || [];
            else if (error) toast.error("Failed to load SSH keys");
        } else {
            const { data, error } = await api.get<{ hosts: RemoteHost[] }>(
                "/api/ssh/hosts",
            );
            if (data) hosts = data.hosts || [];
            else if (error) toast.error("Failed to load remote hosts");
        }
        isLoading = false;
    }

    // Watch tab change
    $: if (activeTab) loadData();

    // Add new Remote Host
    async function addHost() {
        if (
            !newHostName.trim() ||
            !newHostAddress.trim() ||
            !newHostUser.trim()
        ) {
            toast.error("Please fill in all required fields");
            return;
        }

        isAdding = true;
        const { data, error } = await api.post<RemoteHost>("/api/ssh/hosts", {
            name: newHostName.trim(),
            hostname: newHostAddress.trim(),
            port: newHostPort,
            username: newHostUser.trim(),
            identity_file: newHostKeyPath.trim(),
        });

        if (data) {
            hosts = [data, ...hosts]; // Prepend
            toast.success("Remote host added");
            closeModal();
        } else {
            toast.error(error || "Failed to add host");
        }
        isAdding = false;
    }

    // Delete SSH key or Remote Host
    function deleteKey(id: string, name: string, type: "key" | "host") {
        itemToDelete = { id, name, type };
        showDeleteConfirm = true;
    }

    async function confirmDeleteKey() {
        if (!itemToDelete) return;

        const { id, type } = itemToDelete;
        itemToDelete = null;

        let error: string | null | undefined = null;
        if (type === "key") {
            const res = await api.delete(`/api/ssh/keys/${id}`);
            error = res.error;
        } else {
            const res = await api.delete(`/api/ssh/hosts/${id}`);
            error = res.error;
        }

        if (!error) {
            if (type === "key") {
                keys = keys.filter((k) => k.id !== id);
                toast.success("SSH key deleted");
            } else {
                hosts = hosts.filter((h) => h.id !== id);
                toast.success("Remote connection deleted");
            }
        } else {
            toast.error(
                error ||
                    `Failed to delete ${type === "key" ? "SSH key" : "host"}`,
            );
        }
    }

    function cancelDeleteKey() {
        itemToDelete = null;
    }

    function runHost(host: RemoteHost) {
        dispatch("run", { command: host.ssh_command });
    }

    // Modal helpers
    function openModal() {
        // Reset all form fields
        newHostName = "";
        newHostAddress = "";
        newHostPort = 22;
        newHostUser = "root";
        newHostKeyPath = "";
        showAddModal = true;
    }

    function closeModal() {
        showAddModal = false;
    }

    function copyToClipboard(text: string) {
        navigator.clipboard.writeText(text);
        toast.success("Command copied to clipboard");
    }

    // Format date
    function formatDate(dateStr: string): string {
        const date = new Date(dateStr);
        return date.toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
        });
    }

    onMount(loadData);
</script>

<ConfirmModal
    bind:show={showDeleteConfirm}
    title="Delete {itemToDelete?.type === 'key'
        ? 'SSH Key'
        : 'Remote Connection'}"
    message={itemToDelete
        ? `Are you sure you want to delete "${itemToDelete.name}"? This action cannot be undone.`
        : ""}
    confirmText="Delete"
    cancelText="Cancel"
    variant="danger"
    on:confirm={confirmDeleteKey}
    on:cancel={cancelDeleteKey}
/>

<div class="ssh-keys">
    <div class="ssh-keys-header">
        <button class="back-btn" onclick={() => dispatch("back")}>
            ← Back
        </button>
        <h1>SSH Keys</h1>
    </div>

    <div class="ssh-keys-content">
        <!-- Tabs removed as inbound SSH is temporarily disabled -->

        <div class="section-header">
            <p class="section-description">
                Save remote server details to quickly connect FROM your Rexec
                terminals.
            </p>
            <button
                class="btn btn-primary"
                onclick={openModal}
                disabled={$isGuest}
                title={$isGuest ? "Sign in with PipeOps to manage SSH" : ""}
            >
                {#if $isGuest}<StatusIcon status="lock" size={14} />{/if} Add Connection
            </button>
        </div>

        {#if isLoading}
            <div class="loading-state">
                <div class="spinner"></div>
                <p>Loading...</p>
            </div>
        {:else if $isGuest}
            <div class="guest-state">
                <div class="guest-icon">
                    <StatusIcon status="lock" size={48} />
                </div>
                <h2>SSH Features Locked</h2>
                <p>
                    Sign in with PipeOps to manage your SSH configuration and
                    enable secure access features.
                </p>
                <button
                    class="btn btn-primary"
                    onclick={handleOAuthLogin}
                    disabled={isOAuthLoading}
                >
                    {#if isOAuthLoading}
                        <span class="btn-spinner"></span>
                        Connecting...
                    {:else}
                        Sign in with PipeOps
                    {/if}
                </button>
            </div>
        {:else if hosts.length === 0}
            <div class="empty-state">
                <div class="empty-icon">
                    <StatusIcon status="globe" size={48} />
                </div>
                <h2>No Remote Connections</h2>
                <p>
                    Add a remote server to quickly SSH into it from your Rexec
                    terminal.
                </p>
                <button class="btn btn-primary" onclick={openModal}>
                    + Add Your First Connection
                </button>
            </div>
        {:else}
            <div class="keys-list">
                {#each hosts as host (host.id)}
                    <div class="key-card">
                        <div class="key-icon">
                            <StatusIcon status="globe" size={24} />
                        </div>
                        <div class="key-info">
                            <div class="key-name">{host.name}</div>
                            <div class="key-fingerprint">
                                {host.username}@{host.hostname}:{host.port}
                            </div>
                            <div class="key-meta">
                                {#if host.identity_file}
                                    <span
                                        >Identity: {host.identity_file} •
                                    </span>
                                {/if}
                                <span>Added {formatDate(host.created_at)}</span>
                            </div>
                        </div>
                        <div class="actions">
                            <button
                                class="btn btn-secondary btn-sm"
                                onclick={() => runHost(host)}
                                title="Run in Terminal"
                            >
                                <StatusIcon status="terminal" size={14} /> Run
                            </button>
                            <button
                                class="btn btn-secondary btn-sm"
                                onclick={() =>
                                    copyToClipboard(host.ssh_command)}
                                title="Copy SSH Command"
                            >
                                Copy
                            </button>
                            <button
                                class="btn btn-danger btn-sm"
                                onclick={() =>
                                    deleteKey(host.id, host.name, "host")}
                            >
                                Delete
                            </button>
                        </div>
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>

<!-- Add Modal -->
{#if showAddModal}
    <div class="modal-overlay" onclick={closeModal} role="presentation">
        <div
            class="modal"
            onclick={(e) => e.stopPropagation()}
            role="dialog"
            aria-modal="true"
        >
            <div class="modal-header">
                <h2>Add Remote Connection</h2>
                <button class="modal-close" onclick={closeModal}>×</button>
            </div>

            <div class="modal-body">
                <div class="form-group">
                    <label for="host-name">Name</label>
                    <input
                        type="text"
                        id="host-name"
                        bind:value={newHostName}
                        placeholder="e.g., Production DB"
                        maxlength="64"
                    />
                </div>
                <div class="form-row">
                    <div class="form-group" style="flex: 2">
                        <label for="host-address">Hostname / IP</label>
                        <input
                            type="text"
                            id="host-address"
                            bind:value={newHostAddress}
                            placeholder="e.g., 192.168.1.10"
                        />
                    </div>
                    <div class="form-group" style="flex: 1">
                        <label for="host-port">Port</label>
                        <input
                            type="number"
                            id="host-port"
                            bind:value={newHostPort}
                            placeholder="22"
                        />
                    </div>
                </div>
                <div class="form-group">
                    <label for="host-user">Username</label>
                    <input
                        type="text"
                        id="host-user"
                        bind:value={newHostUser}
                        placeholder="root"
                    />
                </div>
                <div class="form-group">
                    <label for="host-key">Identity File (Optional)</label>
                    <input
                        type="text"
                        id="host-key"
                        bind:value={newHostKeyPath}
                        placeholder="e.g., ~/.ssh/id_rsa"
                    />
                    <span class="form-hint"
                        >Path to private key inside the terminal</span
                    >
                </div>
            </div>

            <div class="modal-footer">
                <button
                    class="btn btn-secondary"
                    onclick={closeModal}
                    disabled={isAdding}
                >
                    Cancel
                </button>
                <button
                    class="btn btn-primary"
                    onclick={addHost}
                    disabled={isAdding}
                >
                    {isAdding ? "Adding..." : "Add"}
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    .ssh-keys {
        max-width: 800px;
        margin: 0 auto;
        animation: fadeIn 0.2s ease;
    }

    .ssh-keys-header {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 32px;
        padding-bottom: 16px;
        border-bottom: 1px solid var(--border);
    }

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
        color: var(--text-secondary);
        cursor: pointer;
        font-size: 14px;
        border-bottom: 2px solid transparent;
        transition: all 0.2s;
    }

    .tab-btn:hover {
        color: var(--text);
    }

    .tab-btn.active {
        color: var(--accent);
        border-bottom-color: var(--accent);
    }

    .back-btn {
        background: none;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        padding: 6px 12px;
        font-family: var(--font-mono);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .back-btn:hover {
        border-color: var(--text);
        color: var(--text);
    }

    .ssh-keys-header h1 {
        font-size: 20px;
        text-transform: uppercase;
        letter-spacing: 1px;
        margin: 0;
    }

    .ssh-keys-content {
        display: flex;
        flex-direction: column;
        gap: 24px;
    }

    .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 16px;
    }

    .section-description {
        color: var(--text-muted);
        font-size: 13px;
        margin: 0;
    }

    /* Loading State */
    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 60px 20px;
        gap: 16px;
    }

    .loading-state p {
        color: var(--text-muted);
    }

    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    /* Empty State */
    .empty-state,
    .guest-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 60px 20px;
        text-align: center;
        border: 1px dashed var(--border);
        background: var(--bg-card);
    }

    .empty-icon,
    .guest-icon {
        font-size: 48px;
        margin-bottom: 16px;
    }

    .empty-state h2,
    .guest-state h2 {
        font-size: 18px;
        margin-bottom: 8px;
        text-transform: uppercase;
    }

    .empty-state p,
    .guest-state p {
        color: var(--text-muted);
        max-width: 400px;
        margin-bottom: 24px;
    }

    .guest-state {
        border-color: var(--warning);
        background: rgba(255, 200, 0, 0.05);
    }

    .guest-icon {
        opacity: 0.7;
    }

    /* Keys List */
    .keys-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .key-card {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 16px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        transition: border-color 0.2s;
    }

    .key-card:hover {
        border-color: var(--text-muted);
    }

    .key-icon {
        font-size: 24px;
    }

    .key-info {
        flex: 1;
        min-width: 0;
    }

    .key-name {
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        margin-bottom: 4px;
    }

    .key-fingerprint {
        font-size: 11px;
        font-family: var(--font-mono);
        color: var(--text-secondary);
        margin-bottom: 4px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .key-meta {
        font-size: 11px;
        color: var(--text-muted);
    }

    /* Instructions */
    .instructions {
        background: var(--bg-card);
        border: 1px solid var(--border);
        padding: 20px;
        margin-top: 16px;
    }

    .instructions h3 {
        font-size: 14px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--accent);
        margin: 0 0 16px;
    }

    .instruction-steps {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .step {
        display: flex;
        gap: 12px;
    }

    .step-number {
        width: 24px;
        height: 24px;
        background: var(--accent);
        color: var(--bg);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 12px;
        font-weight: bold;
        flex-shrink: 0;
    }

    .step-content {
        flex: 1;
    }

    .step-content p {
        margin: 0 0 8px;
        font-size: 13px;
        color: var(--text-secondary);
    }

    .step-content code {
        display: block;
        padding: 10px 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--accent);
        overflow-x: auto;
    }

    /* Modal */
    .modal-overlay {
        position: fixed;
        inset: 0;
        background: var(--overlay-bg);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
        animation: fadeIn 0.15s ease;
    }

    .modal {
        background: var(--bg-card);
        border: 1px solid var(--border);
        width: 100%;
        max-width: 500px;
        max-height: 90vh;
        overflow-y: auto;
        animation: slideIn 0.2s ease;
    }

    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
    }

    .modal-header h2 {
        font-size: 16px;
        text-transform: uppercase;
        margin: 0;
    }

    .modal-close {
        background: none;
        border: none;
        color: var(--text-muted);
        font-size: 24px;
        cursor: pointer;
        padding: 0;
        line-height: 1;
    }

    .modal-close:hover {
        color: var(--text);
    }

    .modal-body {
        padding: 20px;
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .form-group label {
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-secondary);
    }

    .form-group input,
    .form-group textarea {
        width: 100%;
        padding: 10px 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .form-group input:focus,
    .form-group textarea:focus {
        outline: none;
        border-color: var(--accent);
    }

    .form-group textarea {
        resize: vertical;
        min-height: 100px;
    }

    .form-row {
        display: flex;
        gap: 12px;
        margin-bottom: 16px;
    }

    .form-hint {
        font-size: 11px;
        color: var(--text-muted);
    }

    .actions {
        display: flex;
        gap: 8px;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        padding: 16px 20px;
        border-top: 1px solid var(--border);
    }

    @keyframes fadeIn {
        from {
            opacity: 0;
        }
        to {
            opacity: 1;
        }
    }

    @keyframes slideIn {
        from {
            opacity: 0;
            transform: translateY(-20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .btn-spinner {
        display: inline-block;
        width: 14px;
        height: 14px;
        border: 2px solid transparent;
        border-top-color: currentColor;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-right: 8px;
        vertical-align: middle;
    }

    .btn:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    @media (max-width: 768px) {
        .ssh-keys {
            padding: 12px;
        }

        .ssh-keys-header {
            margin-bottom: 16px;
        }

        .ssh-keys-header h1 {
            font-size: 18px;
        }

        .back-btn {
            width: 32px;
            height: 32px;
        }

        .section-header {
            flex-direction: column;
            align-items: flex-start;
            gap: 12px;
        }

        .section-description {
            font-size: 12px;
        }

        .key-card {
            flex-wrap: wrap;
            padding: 12px;
            gap: 10px;
        }

        .key-info {
            width: 100%;
            order: 1;
        }

        .key-icon {
            order: 0;
        }

        .key-name {
            font-size: 13px;
        }

        .key-fingerprint {
            font-size: 11px;
        }

        .key-meta {
            font-size: 10px;
        }

        .actions {
            order: 2;
            width: 100%;
            justify-content: flex-end;
        }

        .key-card .btn {
            padding: 6px 10px;
            font-size: 11px;
        }

        .empty-state,
        .guest-state {
            padding: 24px 16px;
        }

        .empty-state h2,
        .guest-state h2 {
            font-size: 16px;
        }

        .empty-state p,
        .guest-state p {
            font-size: 12px;
        }

        .modal {
            margin: 12px;
            max-width: calc(100% - 24px);
        }

        .modal-header {
            padding: 12px 14px;
        }

        .modal-header h2 {
            font-size: 15px;
        }

        .modal-body {
            padding: 14px;
        }

        .form-row {
            flex-direction: column;
            gap: 12px;
        }

        .form-group label {
            font-size: 12px;
        }

        .form-group input {
            font-size: 13px;
            padding: 8px 10px;
        }

        .form-hint {
            font-size: 10px;
        }

        .modal-footer {
            padding: 12px 14px;
            flex-wrap: wrap;
            gap: 8px;
        }

        .modal-footer .btn {
            flex: 1;
            min-width: calc(50% - 4px);
        }
    }

    @media (max-width: 480px) {
        .ssh-keys {
            padding: 8px;
        }

        .ssh-keys-header {
            gap: 8px;
            margin-bottom: 12px;
        }

        .ssh-keys-header h1 {
            font-size: 16px;
        }

        .back-btn {
            width: 28px;
            height: 28px;
        }

        .section-description {
            font-size: 11px;
        }

        .section-header .btn {
            width: 100%;
            justify-content: center;
        }

        .key-card {
            padding: 10px;
            gap: 8px;
        }

        .key-icon {
            width: 36px;
            height: 36px;
        }

        .key-icon :global(svg) {
            width: 18px;
            height: 18px;
        }

        .key-name {
            font-size: 12px;
        }

        .key-fingerprint {
            font-size: 10px;
        }

        .key-meta {
            font-size: 9px;
            flex-wrap: wrap;
        }

        .actions {
            gap: 6px;
        }

        .key-card .btn {
            padding: 5px 8px;
            font-size: 10px;
        }

        .empty-icon :global(svg),
        .guest-icon :global(svg) {
            width: 36px;
            height: 36px;
        }

        .empty-state h2,
        .guest-state h2 {
            font-size: 14px;
        }

        .empty-state p,
        .guest-state p {
            font-size: 11px;
        }

        .modal {
            margin: 8px;
            max-width: calc(100% - 16px);
        }

        .modal-header {
            padding: 10px 12px;
        }

        .modal-header h2 {
            font-size: 14px;
        }

        .modal-body {
            padding: 12px;
        }

        .form-group {
            margin-bottom: 12px;
        }

        .form-group label {
            font-size: 11px;
            margin-bottom: 4px;
        }

        .form-group input {
            font-size: 12px;
            padding: 6px 8px;
        }

        .form-hint {
            font-size: 9px;
        }

        .modal-footer {
            padding: 10px 12px;
            flex-direction: column;
            gap: 6px;
        }

        .modal-footer .btn {
            width: 100%;
            min-width: auto;
        }
    }

    @media (max-width: 360px) {
        .ssh-keys {
            padding: 6px;
        }

        .ssh-keys-header h1 {
            font-size: 14px;
        }

        .back-btn {
            width: 24px;
            height: 24px;
        }

        .section-description {
            font-size: 10px;
        }

        .key-card {
            padding: 8px;
        }

        .key-icon {
            width: 30px;
            height: 30px;
        }

        .key-name {
            font-size: 11px;
        }

        .key-fingerprint {
            font-size: 9px;
        }

        .key-meta {
            font-size: 8px;
        }

        .key-card .btn {
            padding: 4px 6px;
            font-size: 9px;
        }

        .modal {
            margin: 4px;
            max-width: calc(100% - 8px);
        }

        .modal-header {
            padding: 8px 10px;
        }

        .modal-header h2 {
            font-size: 13px;
        }

        .modal-body {
            padding: 10px;
        }

        .form-group input {
            font-size: 11px;
            padding: 5px 6px;
        }

        .modal-footer {
            padding: 8px 10px;
        }

        .modal-footer .btn {
            padding: 8px 10px;
            font-size: 10px;
        }
    }
</style>
