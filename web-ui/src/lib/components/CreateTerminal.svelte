<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import InlineCreateTerminal from "./InlineCreateTerminal.svelte";

    const dispatch = createEventDispatcher<{
        cancel: void;
        created: { id: string; name: string };
    }>();

    function handleCreated(event: CustomEvent<{ id: string; name: string }>) {
        dispatch("created", event.detail);
    }

    function handleCancel() {
        dispatch("cancel");
    }
</script>

<div class="create-container">
    <div class="create-header">
        <button class="back-btn" on:click={handleCancel}>
            ← Back
        </button>
        <h1>Create Terminal</h1>
    </div>

    <div class="create-body">
        <InlineCreateTerminal
            compact={false}
            on:created={handleCreated}
            on:cancel={handleCancel}
        />
    </div>
</div>

<style>
    .create-container {
        height: 100%;
        display: flex;
        flex-direction: column;
        background: #0a0a0a;
        overflow: hidden;
    }

    .create-header {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 20px 24px;
        border-bottom: 1px solid var(--border);
        flex-shrink: 0;
    }

    .create-header h1 {
        margin: 0;
        font-size: 20px;
        font-weight: 600;
        color: var(--text);
    }

    .back-btn {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 8px 16px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        font-size: 13px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .back-btn:hover {
        border-color: var(--text-muted);
        color: var(--text);
    }

    .create-body {
        flex: 1;
        overflow: hidden;
        display: flex;
        flex-direction: column;
    }

    .create-body :global(.inline-create) {
        flex: 1;
        max-width: 1200px;
        margin: 0 auto;
        width: 100%;
    }
</style>
