<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { fade, scale } from "svelte/transition";

    export let title: string = "Confirm";
    export let message: string = "";
    export let confirmText: string = "Confirm";
    export let cancelText: string = "Cancel";
    export let variant: "danger" | "warning" | "default" = "default";
    export let show: boolean = false;

    const dispatch = createEventDispatcher<{
        confirm: void;
        cancel: void;
    }>();

    function handleConfirm() {
        dispatch("confirm");
        show = false;
    }

    function handleCancel() {
        dispatch("cancel");
        show = false;
    }

    function handleKeydown(e: KeyboardEvent) {
        if (!show) return;
        if (e.key === "Escape") {
            handleCancel();
        } else if (e.key === "Enter") {
            handleConfirm();
        }
    }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if show}
    <div class="modal-backdrop" transition:fade={{ duration: 150 }}>
        <div
            class="modal-container"
            transition:scale={{ duration: 150, start: 0.95 }}
        >
            <div class="modal-header">
                <div class="modal-icon {variant}">
                    {#if variant === "danger"}
                        <svg viewBox="0 0 16 16" fill="currentColor">
                            <path
                                d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"
                            />
                            <path
                                fill-rule="evenodd"
                                d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"
                            />
                        </svg>
                    {:else if variant === "warning"}
                        <svg viewBox="0 0 16 16" fill="currentColor">
                            <path
                                d="M8.982 1.566a1.13 1.13 0 0 0-1.96 0L.165 13.233c-.457.778.091 1.767.98 1.767h13.713c.889 0 1.438-.99.98-1.767L8.982 1.566zM8 5c.535 0 .954.462.9.995l-.35 3.507a.552.552 0 0 1-1.1 0L7.1 5.995A.905.905 0 0 1 8 5zm.002 6a1 1 0 1 1 0 2 1 1 0 0 1 0-2z"
                            />
                        </svg>
                    {:else}
                        <svg viewBox="0 0 16 16" fill="currentColor">
                            <path
                                d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z"
                            />
                            <path
                                d="m8.93 6.588-2.29.287-.082.38.45.083c.294.07.352.176.288.469l-.738 3.468c-.194.897.105 1.319.808 1.319.545 0 1.178-.252 1.465-.598l.088-.416c-.2.176-.492.246-.686.246-.275 0-.375-.193-.304-.533L8.93 6.588zM9 4.5a1 1 0 1 1-2 0 1 1 0 0 1 2 0z"
                            />
                        </svg>
                    {/if}
                </div>
                <h2 class="modal-title">{title}</h2>
            </div>

            <div class="modal-body">
                <p class="modal-message">{message}</p>
            </div>

            <div class="modal-actions">
                <button class="btn btn-cancel" on:click={handleCancel}>
                    {cancelText}
                </button>
                <button class="btn btn-confirm {variant}" on:click={handleConfirm}>
                    {confirmText}
                </button>
            </div>

            <div class="modal-border-glow {variant}"></div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.8);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
        backdrop-filter: blur(4px);
    }

    .modal-container {
        position: relative;
        width: 380px;
        max-width: 90vw;
        background: var(--bg, #0a0a0a);
        border: 1px solid var(--border, #1a1a1a);
        padding: 24px;
        overflow: hidden;
    }

    .modal-border-glow {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 2px;
        background: var(--accent, #00ff41);
    }

    .modal-border-glow.danger {
        background: linear-gradient(90deg, #ff003c, #ff6b6b);
        box-shadow: 0 0 20px rgba(255, 0, 60, 0.4);
    }

    .modal-border-glow.warning {
        background: linear-gradient(90deg, #ffc800, #ffdd57);
        box-shadow: 0 0 20px rgba(255, 200, 0, 0.4);
    }

    .modal-border-glow.default {
        background: linear-gradient(90deg, var(--accent, #00ff41), #00ffaa);
        box-shadow: 0 0 20px rgba(0, 255, 65, 0.4);
    }

    .modal-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 16px;
    }

    .modal-icon {
        width: 32px;
        height: 32px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 4px;
        flex-shrink: 0;
    }

    .modal-icon svg {
        width: 20px;
        height: 20px;
    }

    .modal-icon.danger {
        background: rgba(255, 0, 60, 0.15);
        color: #ff6b6b;
    }

    .modal-icon.warning {
        background: rgba(255, 200, 0, 0.15);
        color: #ffc800;
    }

    .modal-icon.default {
        background: rgba(0, 255, 65, 0.15);
        color: var(--accent, #00ff41);
    }

    .modal-title {
        font-size: 16px;
        font-weight: 600;
        color: var(--text, #e0e0e0);
        margin: 0;
        font-family: var(--font-mono, monospace);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .modal-body {
        margin-bottom: 24px;
    }

    .modal-message {
        font-size: 13px;
        color: var(--text-secondary, #a0a0a0);
        line-height: 1.5;
        margin: 0;
        font-family: var(--font-mono, monospace);
    }

    .modal-actions {
        display: flex;
        gap: 12px;
        justify-content: flex-end;
    }

    .btn {
        padding: 8px 16px;
        font-size: 12px;
        font-family: var(--font-mono, monospace);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        cursor: pointer;
        transition: all 0.15s;
        border: 1px solid;
    }

    .btn-cancel {
        background: transparent;
        border-color: var(--border, #1a1a1a);
        color: var(--text-muted, #666);
    }

    .btn-cancel:hover {
        border-color: var(--text-muted, #666);
        color: var(--text, #e0e0e0);
        background: var(--bg-tertiary, #1a1a1a);
    }

    .btn-confirm {
        background: var(--accent, #00ff41);
        border-color: var(--accent, #00ff41);
        color: var(--bg, #0a0a0a);
        font-weight: 600;
    }

    .btn-confirm:hover {
        box-shadow: 0 0 15px rgba(0, 255, 65, 0.4);
    }

    .btn-confirm.danger {
        background: #ff003c;
        border-color: #ff003c;
        color: #fff;
    }

    .btn-confirm.danger:hover {
        background: #ff3366;
        border-color: #ff3366;
        box-shadow: 0 0 15px rgba(255, 0, 60, 0.5);
    }

    .btn-confirm.warning {
        background: #ffc800;
        border-color: #ffc800;
        color: #000;
    }

    .btn-confirm.warning:hover {
        background: #ffdd57;
        border-color: #ffdd57;
        box-shadow: 0 0 15px rgba(255, 200, 0, 0.5);
    }
</style>
