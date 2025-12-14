<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import InlineCreateTerminal from "./InlineCreateTerminal.svelte";

    const dispatch = createEventDispatcher<{
        cancel: void;
        created: { id: string; name: string };
        upgrade: void;
    }>();

    function handleCreated(event: CustomEvent<{ id: string; name: string }>) {
        dispatch("created", event.detail);
    }

    function handleCancel() {
        dispatch("cancel");
    }

    function handleUpgrade() {
        dispatch("upgrade");
    }
</script>

<div class="create-page">
    <button class="back-btn" onclick={handleCancel}>
        <span class="back-icon">←</span>
        <span>Back to Dashboard</span>
    </button>

    <InlineCreateTerminal
        compact={false}
        on:created={handleCreated}
        on:cancel={handleCancel}
        on:upgrade={handleUpgrade}
    />
</div>

<style>
    .create-page {
        height: 100%;
        display: flex;
        flex-direction: column;
        background: var(--bg);
        background-image: none;
        overflow: auto;
        padding: 24px;
    }

    .back-btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 8px 14px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        font-size: 13px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s ease;
        width: fit-content;
        margin-bottom: 20px;
    }

    .back-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    .back-icon {
        font-size: 16px;
    }

    .create-page :global(.inline-create) {
        flex: 1;
        max-width: 1200px;
        width: 100%;
        margin: 0 auto;
        padding: 0;
        background: transparent;
    }
</style>
