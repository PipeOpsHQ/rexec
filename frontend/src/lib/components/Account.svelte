<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { auth, isAuthenticated, isGuest, isAdmin } from "$stores/auth";
    import StatusIcon from "./icons/StatusIcon.svelte";
    
    const dispatch = createEventDispatcher<{
        navigate: { view: string };
        logout: void;
    }>();

    function navigate(view: string) {
        dispatch("navigate", { view });
    }

    function handleLogout() {
        dispatch("logout");
    }
</script>

<div class="account-page">
    <div class="page-header">
        <h1>My Account</h1>
        <p class="subtitle">Manage your profile, settings, and subscription.</p>
    </div>

    <div class="account-grid">
        <!-- Profile Card -->
        <div class="profile-card">
            <div class="profile-header">
                <div class="avatar">
                    {$auth.user?.name?.charAt(0).toUpperCase() || "U"}
                </div>
                <div class="profile-info">
                    <h2>{$auth.user?.name || "User"}</h2>
                    <p class="email">{$auth.user?.email || "No email"}</p>
                    <div class="badges">
                        <span class="badge tier">{$auth.user?.tier?.toUpperCase() || "FREE"} PLAN</span>
                        {#if $isGuest}
                            <span class="badge guest">GUEST</span>
                        {/if}
                        {#if $isAdmin}
                            <span class="badge admin">ADMIN</span>
                        {/if}
                    </div>
                </div>
            </div>
            
            <div class="profile-actions">
                <button class="action-btn danger" onclick={handleLogout}>
                    <StatusIcon status="logout" size={16} />
                    <span>Sign Out</span>
                </button>
            </div>
        </div>

        <!-- Quick Links Grid -->
        <div class="links-grid">
            <button class="link-card" onclick={() => navigate('dashboard')} disabled={$isGuest}>
                <div class="icon-wrapper">
                    <StatusIcon status="chart" size={24} />
                </div>
                <div class="card-content">
                    <h3>Dashboard</h3>
                    <p>Manage your terminals and active sessions.</p>
                </div>
            </button>

            <button class="link-card" onclick={() => navigate('settings')} disabled={$isGuest}>
                <div class="icon-wrapper">
                    <StatusIcon status="settings" size={24} />
                </div>
                <div class="card-content">
                    <h3>Settings</h3>
                    <p>Configure appearance, behavior, and preferences.</p>
                </div>
            </button>

            <button class="link-card" onclick={() => navigate('sshkeys')} disabled={$isGuest}>
                <div class="icon-wrapper">
                    <StatusIcon status="key" size={24} />
                </div>
                <div class="card-content">
                    <h3>SSH Keys</h3>
                    <p>Manage SSH keys for secure terminal access.</p>
                </div>
            </button>

            <button class="link-card" onclick={() => navigate('billing')} disabled={$isGuest}>
                <div class="icon-wrapper">
                    <StatusIcon status="invoice" size={24} />
                </div>
                <div class="card-content">
                    <h3>Billing & Plans</h3>
                    <p>Manage your subscription and payment methods.</p>
                </div>
            </button>
            
            <button class="link-card" onclick={() => navigate('snippets')} disabled={$isGuest}>
                <div class="icon-wrapper">
                    <StatusIcon status="snippet" size={24} />
                </div>
                <div class="card-content">
                    <h3>Snippets</h3>
                    <p>Create and manage your reusable command snippets.</p>
                </div>
            </button>

            <button class="link-card" onclick={() => navigate('recordings')} disabled={$isGuest}>
                <div class="icon-wrapper">
                    <StatusIcon status="recording" size={24} />
                </div>
                <div class="card-content">
                    <h3>Recordings</h3>
                    <p>View and manage your terminal session recordings.</p>
                </div>
            </button>

            <button class="link-card" onclick={() => navigate('pricing')}>
                <div class="icon-wrapper">
                    <StatusIcon status="pricing" size={24} />
                </div>
                <div class="card-content">
                    <h3>Pricing</h3>
                    <p>View available plans and features.</p>
                </div>
            </button>
        </div>
    </div>

    <!-- Developer Tools Section -->
    <div class="section-divider">
        <h2>Developer Tools</h2>
    </div>

    <div class="links-grid">
        <button class="link-card" onclick={() => navigate('docs/cli')} disabled={$isGuest}>
            <div class="icon-wrapper accent">
                <StatusIcon status="cli" size={24} />
            </div>
            <div class="card-content">
                <h3>CLI Reference</h3>
                <p>Command line tools for managing your environment.</p>
            </div>
        </button>

        <button class="link-card" onclick={() => navigate('docs/agent')} disabled={$isGuest}>
            <div class="icon-wrapper accent">
                <StatusIcon status="server" size={24} />
            </div>
            <div class="card-content">
                <h3>Agents</h3>
                <p>Connect and manage your own infrastructure.</p>
            </div>
        </button>

        {#if $isAdmin}
            <button class="link-card" onclick={() => navigate('admin')}>
                <div class="icon-wrapper danger">
                    <StatusIcon status="shield" size={24} />
                </div>
                <div class="card-content">
                    <h3>Admin Portal</h3>
                    <p>System administration and user management.</p>
                </div>
            </button>
        {/if}
    </div>
</div>

<style>
    .account-page {
        max-width: 1000px;
        margin: 0 auto;
        padding: 40px 20px;
    }

    .page-header {
        margin-bottom: 40px;
    }

    h1 {
        font-size: 32px;
        font-weight: 700;
        margin: 0 0 8px 0;
    }

    .subtitle {
        color: var(--text-muted);
        font-size: 16px;
    }

    .account-grid {
        display: grid;
        grid-template-columns: 1fr;
        gap: 32px;
        margin-bottom: 40px;
    }

    /* Profile Card */
    .profile-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 24px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: 20px;
    }

    .profile-header {
        display: flex;
        align-items: center;
        gap: 20px;
    }

    .avatar {
        width: 64px;
        height: 64px;
        border-radius: 50%;
        background: var(--accent);
        color: var(--bg);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 24px;
        font-weight: 700;
    }

    .profile-info h2 {
        font-size: 20px;
        margin: 0 0 4px 0;
        color: var(--text);
    }

    .email {
        color: var(--text-muted);
        font-size: 14px;
        margin: 0 0 12px 0;
    }

    .badges {
        display: flex;
        gap: 8px;
    }

    .badge {
        font-size: 11px;
        font-weight: 600;
        padding: 4px 8px;
        border-radius: 4px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .badge.tier {
        background: rgba(var(--accent-rgb), 0.1);
        color: var(--accent);
        border: 1px solid rgba(var(--accent-rgb), 0.2);
    }

    .badge.guest {
        background: rgba(255, 165, 0, 0.1);
        color: orange;
        border: 1px solid rgba(255, 165, 0, 0.2);
    }

    .badge.admin {
        background: rgba(255, 0, 0, 0.1);
        color: #ff4d4d;
        border: 1px solid rgba(255, 0, 0, 0.2);
    }

    .action-btn {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        border: 1px solid var(--border);
        background: transparent;
        color: var(--text);
    }

    .action-btn:hover {
        background: var(--bg-secondary);
        border-color: var(--text-muted);
    }

    .action-btn.danger {
        color: #ff4d4d;
        border-color: rgba(255, 0, 0, 0.3);
    }

    .action-btn.danger:hover {
        background: rgba(255, 0, 0, 0.1);
    }

    /* Links Grid */
    .links-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 20px;
    }

    .link-card {
        display: flex;
        align-items: flex-start;
        gap: 16px;
        padding: 20px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
        text-align: left;
        cursor: pointer;
        transition: all 0.2s;
        width: 100%;
    }

    .link-card:hover:not(:disabled) {
        border-color: var(--accent);
        transform: translateY(-2px);
    }

    .link-card:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .icon-wrapper {
        padding: 10px;
        background: var(--bg-secondary);
        border-radius: 8px;
        color: var(--text-secondary);
    }

    .icon-wrapper.accent {
        color: var(--accent);
        background: rgba(var(--accent-rgb), 0.1);
    }

    .icon-wrapper.danger {
        color: #ff4d4d;
        background: rgba(255, 0, 0, 0.1);
    }

    .card-content h3 {
        font-size: 16px;
        margin: 0 0 4px 0;
        color: var(--text);
    }

    .card-content p {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0;
        line-height: 1.4;
    }

    .section-divider {
        margin: 40px 0 24px;
        border-bottom: 1px solid var(--border);
        padding-bottom: 12px;
    }

    .section-divider h2 {
        font-size: 18px;
        font-weight: 600;
        color: var(--text-secondary);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    @media (max-width: 600px) {
        .profile-card {
            flex-direction: column;
            align-items: flex-start;
        }
        
        .profile-actions {
            width: 100%;
        }

        .action-btn {
            width: 100%;
            justify-content: center;
        }
    }
</style>
