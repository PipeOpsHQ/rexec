<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { auth } from "$stores/auth";
    import StatusIcon from "./icons/StatusIcon.svelte";

    export let section: string; // current section (settings, ssh, billing, snippets)

    const dispatch = createEventDispatcher<{
        navigate: { view: string };
        logout: void;
    }>();

    function navigate(view: string) {
        dispatch("navigate", { view });
    }

    // Navigation items
    const navItems = [
        { id: "overview", label: "Overview", href: "/account", icon: "chart" },
        {
            id: "settings",
            label: "Settings",
            href: "/account/settings",
            icon: "settings",
        },
        { id: "ssh", label: "SSH Keys", href: "/account/ssh", icon: "key" },
        { id: "api", label: "API Tokens", href: "/account/api", icon: "code" },
        {
            id: "recordings",
            label: "Recordings",
            href: "/account/recordings",
            icon: "video",
        },
        {
            id: "billing",
            label: "Billing",
            href: "/account/billing",
            icon: "invoice",
        },
        {
            id: "snippets",
            label: "Snippets",
            href: "/account/snippets",
            icon: "snippet",
        },
    ];

    function isActive(itemId: string): boolean {
        if (!section && itemId === "overview") return true;
        return section === itemId;
    }

    function handleNav(href: string) {
        window.history.pushState({}, "", href);
        window.dispatchEvent(new PopStateEvent("popstate"));
    }
</script>

<div class="account-layout">
    <div class="account-header">
        <div class="header-left">
            <button class="back-btn" onclick={() => navigate("dashboard")}>
                <svg
                    class="icon"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                >
                    <path d="M19 12H5M12 19l-7-7 7-7" />
                </svg>
            </button>
            <div class="header-info">
                <h1>Account</h1>
                <p class="user-info">
                    {$auth.user?.email || "User"} · {$auth.user?.tier?.toUpperCase() ||
                        "FREE"}
                </p>
            </div>
        </div>
    </div>

    <div class="account-content">
        <nav class="account-nav">
            {#each navItems as item}
                <button
                    class="nav-item"
                    class:active={isActive(item.id)}
                    onclick={() => handleNav(item.href)}
                >
                    <StatusIcon status={item.icon} size={18} />
                    <span>{item.label}</span>
                </button>
            {/each}
        </nav>

        <div class="account-main">
            <slot />
        </div>
    </div>
</div>

<style>
    .account-layout {
        max-width: 1200px;
        margin: 0 auto;
        padding: 20px;
    }

    .account-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 32px;
        padding-bottom: 20px;
        border-bottom: 1px solid var(--border);
    }

    .header-left {
        display: flex;
        align-items: center;
        gap: 16px;
    }

    .back-btn {
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        cursor: pointer;
        color: var(--text-secondary);
        transition: all 0.2s;
    }

    .back-btn:hover {
        background: var(--bg-tertiary);
        border-color: var(--accent);
        color: var(--accent);
    }

    .back-btn .icon {
        width: 20px;
        height: 20px;
    }

    .header-info h1 {
        font-size: 24px;
        font-weight: 700;
        margin: 0 0 4px 0;
        color: var(--text);
    }

    .user-info {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0;
    }

    .account-content {
        display: grid;
        grid-template-columns: 220px 1fr;
        gap: 32px;
    }

    .account-nav {
        display: flex;
        flex-direction: column;
        gap: 4px;
        position: sticky;
        top: 80px;
        height: fit-content;
    }

    .nav-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 16px;
        background: transparent;
        border: 1px solid transparent;
        border-radius: 8px;
        color: var(--text-secondary);
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        text-align: left;
        width: 100%;
    }

    .nav-item:hover {
        background: var(--bg-secondary);
        color: var(--text);
    }

    .nav-item.active {
        background: var(--bg-secondary);
        border-color: var(--accent);
        color: var(--accent);
    }

    .account-main {
        min-height: 400px;
    }

    @media (max-width: 768px) {
        .account-layout {
            padding: 12px;
        }

        .account-header {
            margin-bottom: 20px;
            padding-bottom: 16px;
        }

        .account-content {
            grid-template-columns: 1fr;
            gap: 16px;
        }

        .account-nav {
            position: static;
            flex-direction: row;
            overflow-x: auto;
            padding: 0 0 8px 0;
            margin: 0 -12px;
            padding-left: 12px;
            padding-right: 12px;
            gap: 6px;
            -webkit-overflow-scrolling: touch;
            scrollbar-width: none;
            -ms-overflow-style: none;
            /* Fade hints for scroll */
            mask-image: linear-gradient(
                to right,
                transparent 0,
                black 8px,
                black calc(100% - 8px),
                transparent 100%
            );
            -webkit-mask-image: linear-gradient(
                to right,
                transparent 0,
                black 8px,
                black calc(100% - 8px),
                transparent 100%
            );
        }

        .account-nav::-webkit-scrollbar {
            display: none;
        }

        .nav-item {
            white-space: nowrap;
            min-width: fit-content;
            padding: 10px 12px;
            font-size: 13px;
            flex-shrink: 0;
        }

        .nav-item span {
            display: none;
        }

        .header-left {
            gap: 12px;
        }

        .back-btn {
            width: 36px;
            height: 36px;
        }

        .back-btn .icon {
            width: 18px;
            height: 18px;
        }

        .header-info h1 {
            font-size: 18px;
        }

        .user-info {
            font-size: 11px;
        }

        .account-main {
            min-height: 300px;
        }
    }

    @media (max-width: 480px) {
        .account-layout {
            padding: 8px;
        }

        .account-header {
            margin-bottom: 16px;
            padding-bottom: 12px;
        }

        .header-left {
            gap: 10px;
        }

        .back-btn {
            width: 32px;
            height: 32px;
        }

        .back-btn .icon {
            width: 16px;
            height: 16px;
        }

        .header-info h1 {
            font-size: 16px;
        }

        .user-info {
            font-size: 10px;
        }

        .nav-item {
            padding: 8px 10px;
            font-size: 12px;
            border-radius: 6px;
        }

        .account-main {
            min-height: 250px;
        }
    }

    @media (max-width: 360px) {
        .account-layout {
            padding: 6px;
        }

        .account-header {
            margin-bottom: 12px;
            padding-bottom: 10px;
        }

        .header-left {
            gap: 8px;
        }

        .back-btn {
            width: 28px;
            height: 28px;
        }

        .back-btn .icon {
            width: 14px;
            height: 14px;
        }

        .header-info h1 {
            font-size: 14px;
        }

        .user-info {
            font-size: 9px;
        }

        .account-nav {
            gap: 4px;
            margin: 0 -6px;
            padding: 0 6px 6px;
        }

        .nav-item {
            padding: 6px 8px;
            font-size: 11px;
            border-radius: 5px;
        }

        .nav-item :global(svg) {
            width: 14px;
            height: 14px;
        }
    }
</style>
