<script lang="ts">
  import { canInstall, promptInstall, deferredPrompt } from '$stores/pwa';
  import { onMount } from 'svelte';

  let installing = false;

  onMount(() => {
    // Debug: log state changes
    const unsub = canInstall.subscribe(val => {
      console.log('PWA: canInstall changed to:', val);
    });
    return unsub;
  });

  async function handleInstall() {
    console.log('PWA: Install button clicked');
    installing = true;
    const result = await promptInstall();
    console.log('PWA: Install result:', result);
    installing = false;
  }
</script>

{#if $canInstall}
  <button
    class="install-button"
    onclick={handleInstall}
    disabled={installing}
    title="Install Rexec as an app"
  >
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
      <polyline points="7 10 12 15 17 10" />
      <line x1="12" y1="15" x2="12" y2="3" />
    </svg>
    <span>{installing ? 'Installing...' : 'Install App'}</span>
  </button>
{/if}

<style>
  .install-button {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    background: linear-gradient(135deg, rgba(0, 255, 65, 0.15), rgba(0, 255, 65, 0.05));
    border: 1px solid rgba(0, 255, 65, 0.4);
    border-radius: 0.5rem;
    color: #00ff41;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .install-button:hover:not(:disabled) {
    background: linear-gradient(135deg, rgba(0, 255, 65, 0.25), rgba(0, 255, 65, 0.1));
    border-color: rgba(0, 255, 65, 0.6);
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(0, 255, 65, 0.2);
  }

  .install-button:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .install-button svg {
    flex-shrink: 0;
  }
</style>
