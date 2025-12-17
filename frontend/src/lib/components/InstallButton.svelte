<script lang="ts">
  import { canInstall, promptInstall, dismissInstallPrompt } from '$stores/pwa';

  let installing = false;

  async function handleInstall() {
    installing = true;
    await promptInstall();
    installing = false;
  }

  function handleDismiss(e: Event) {
    e.stopPropagation();
    dismissInstallPrompt();
  }
</script>

{#if $canInstall}
  <div class="install-wrapper">
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
    <button
      class="dismiss-button"
      onclick={handleDismiss}
      title="Dismiss"
      aria-label="Dismiss install prompt"
    >
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="18" y1="6" x2="6" y2="18"></line>
        <line x1="6" y1="6" x2="18" y2="18"></line>
      </svg>
    </button>
  </div>
{/if}

<style>
  .install-wrapper {
    display: inline-flex;
    align-items: center;
    gap: 0;
    position: relative;
  }

  .install-button {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    padding-right: 2rem;
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

  .dismiss-button {
    position: absolute;
    right: 4px;
    top: 50%;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    padding: 0;
    background: transparent;
    border: none;
    border-radius: 50%;
    color: rgba(0, 255, 65, 0.6);
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .dismiss-button:hover {
    color: #00ff41;
    background: rgba(0, 255, 65, 0.2);
  }
</style>
