<script lang="ts">
  import { flip } from 'svelte/animate';
  import { fly, fade } from 'svelte/transition';
  import { activeToasts, toast, type Toast } from '$stores/toast';

  function getIcon(type: Toast['type']): string {
    switch (type) {
      case 'success':
        return '✓';
      case 'error':
        return '✕';
      case 'warning':
        return '⚠';
      case 'loading':
        return '◌';
      default:
        return 'ℹ';
    }
  }
</script>

<div class="toast-container">
  {#each $activeToasts as t (t.id)}
    <div
      class="toast toast-{t.type}"
      animate:flip={{ duration: 200 }}
      in:fly={{ x: 100, duration: 200 }}
      out:fade={{ duration: 150 }}
      on:click={() => toast.dismiss(t.id)}
      role="button"
      tabindex="0"
      on:keydown={(e) => e.key === 'Enter' && toast.dismiss(t.id)}
    >
      <span class="toast-icon" class:spinning={t.type === 'loading'}>
        {getIcon(t.type)}
      </span>
      <span class="toast-message">{t.message}</span>
      <button
        class="toast-close"
        aria-label="Dismiss"
      >
        ×
      </button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    top: 20px;
    right: 20px;
    z-index: 900; /* Below modals (1000+) but above most UI */
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    color: var(--text);
    font-size: 13px;
    font-family: var(--font-mono);
    max-width: 400px;
    pointer-events: auto;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
    cursor: pointer;
  }

  .toast-success {
    border-color: var(--green);
  }

  .toast-success .toast-icon {
    color: var(--green);
  }

  .toast-error {
    border-color: var(--danger);
  }

  .toast-error .toast-icon {
    color: var(--danger);
  }

  .toast-warning {
    border-color: var(--warning);
  }

  .toast-warning .toast-icon {
    color: var(--warning);
  }

  .toast-loading {
    border-color: var(--accent);
  }

  .toast-loading .toast-icon {
    color: var(--accent);
  }

  .toast-info {
    border-color: var(--text-muted);
  }

  .toast-icon {
    font-size: 14px;
    flex-shrink: 0;
  }

  .toast-icon.spinning {
    animation: spin 1s linear infinite;
  }

  .toast-message {
    flex: 1;
    word-break: break-word;
  }

  .toast-close {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 18px;
    cursor: pointer;
    padding: 0;
    line-height: 1;
    opacity: 0.6;
    transition: opacity 0.15s, color 0.15s;
  }

  .toast-close:hover {
    opacity: 1;
    color: var(--text);
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
