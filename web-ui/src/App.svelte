<script lang="ts">
  import { onMount } from 'svelte';
  import { auth, isAuthenticated } from '$stores/auth';
  import { containers } from '$stores/containers';
  import { terminal, hasSessions } from '$stores/terminal';
  import { toast } from '$stores/toast';

  // Components
  import Header from '$components/Header.svelte';
  import Landing from '$components/Landing.svelte';
  import Dashboard from '$components/Dashboard.svelte';
  import CreateTerminal from '$components/CreateTerminal.svelte';
  import Settings from '$components/Settings.svelte';
  import SSHKeys from '$components/SSHKeys.svelte';
  import TerminalView from '$components/terminal/TerminalView.svelte';
  import ToastContainer from '$components/ui/ToastContainer.svelte';

  // App state
  let currentView: 'landing' | 'dashboard' | 'create' | 'settings' | 'sshkeys' = 'landing';
  let isLoading = true;

  // Guest email modal state
  let showGuestModal = false;
  let guestEmail = '';
  let isGuestSubmitting = false;

  function openGuestModal() {
    guestEmail = '';
    showGuestModal = true;
  }

  function closeGuestModal() {
    showGuestModal = false;
    guestEmail = '';
  }

  function validateEmail(email: string): boolean {
    const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return re.test(email);
  }

  async function handleGuestSubmit() {
    if (!guestEmail.trim()) {
      toast.error('Please enter your email');
      return;
    }

    if (!validateEmail(guestEmail.trim())) {
      toast.error('Please enter a valid email');
      return;
    }

    isGuestSubmitting = true;
    const result = await auth.guestLogin(guestEmail.trim());
    isGuestSubmitting = false;

    if (result.success) {
      closeGuestModal();
    }
  }

  function handleGuestKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' && !isGuestSubmitting) {
      handleGuestSubmit();
    } else if (event.key === 'Escape') {
      closeGuestModal();
    }
  }

  // Handle OAuth callback
  async function handleOAuthCallback() {
    const params = new URLSearchParams(window.location.search);
    const code = params.get('code');

    if (code) {
      const result = await auth.exchangeOAuthCode(code);
      if (result.success) {
        // Clear URL params
        window.history.replaceState({}, '', window.location.pathname);
        currentView = 'dashboard';
      }
    }
  }

  // Handle URL-based terminal routing
  async function handleTerminalUrl() {
    const path = window.location.pathname;
    const match = path.match(/^\/(?:terminal\/)?([a-f0-9]{64}|[a-f0-9-]{36})$/i);

    if (match && $isAuthenticated) {
      const containerId = match[1];
      // Fetch container info and create session - TerminalPanel handles WebSocket
      const result = await containers.getContainer(containerId);
      if (result.success && result.container) {
        terminal.createSession(containerId, result.container.name);
      }
    }
  }

  // Initialize app
  onMount(async () => {
    // Check for OAuth callback
    await handleOAuthCallback();

    // Validate existing token
    if ($auth.token) {
      const isValid = await auth.validateToken();
      if (isValid) {
        await auth.fetchProfile();
        currentView = 'dashboard';
        await containers.fetchContainers();
        await handleTerminalUrl();
      } else {
        auth.logout();
      }
    }

    isLoading = false;
  });

  // React to auth changes
  $: if ($isAuthenticated && currentView === 'landing') {
    currentView = 'dashboard';
    containers.fetchContainers();
  }

  $: if (!$isAuthenticated && currentView !== 'landing') {
    currentView = 'landing';
    containers.reset();
    terminal.closeAllSessions();
  }

  // Navigation functions
  function goToDashboard() {
    currentView = 'dashboard';
    window.history.pushState({}, '', '/');
  }

  function goToCreate() {
    currentView = 'create';
  }

  function goToSettings() {
    currentView = 'settings';
  }

  function goToSSHKeys() {
    currentView = 'sshkeys';
  }

  function onContainerCreated(event: CustomEvent<{ id: string; name: string }>) {
    const { id, name } = event.detail;
    currentView = 'dashboard';

    // Create session - TerminalPanel will handle WebSocket connection
    terminal.createSession(id, name);
  }

  // Handle browser navigation
  function handlePopState() {
    const path = window.location.pathname;
    if (path === '/' || path === '') {
      currentView = $isAuthenticated ? 'dashboard' : 'landing';
    }
  }
</script>

<svelte:window on:popstate={handlePopState} />

<div class="app">
  {#if isLoading}
    <div class="loading-screen">
      <div class="spinner-large"></div>
      <p>Loading...</p>
    </div>
  {:else}
    <Header
      on:home={goToDashboard}
      on:create={goToCreate}
      on:settings={goToSettings}
      on:sshkeys={goToSSHKeys}
      on:guest={openGuestModal}
    />

    <main class="main">
      {#if currentView === 'landing'}
        <Landing on:guest={openGuestModal} />
      {:else if currentView === 'dashboard'}
        <Dashboard
          on:create={goToCreate}
          on:connect={(e) => {
            // Only create session - TerminalPanel will handle WebSocket connection
            terminal.createSession(e.detail.id, e.detail.name);
          }}
        />
      {:else if currentView === 'create'}
        <CreateTerminal
          on:cancel={goToDashboard}
          on:created={onContainerCreated}
        />
      {:else if currentView === 'settings'}
        <Settings on:back={goToDashboard} />
      {:else if currentView === 'sshkeys'}
        <SSHKeys on:back={goToDashboard} />
      {/if}
    </main>

    <!-- Terminal overlay (floating or docked) -->
    {#if $hasSessions}
      <TerminalView />
    {/if}

    <!-- Toast notifications -->
    <ToastContainer />

    <!-- Guest Email Modal -->
    {#if showGuestModal}
      <div class="modal-overlay" on:click={closeGuestModal} on:keydown={handleGuestKeydown} role="presentation">
        <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true" aria-labelledby="guest-modal-title">
          <div class="modal-header">
            <h2 id="guest-modal-title">Get Started</h2>
            <button class="modal-close" on:click={closeGuestModal} aria-label="Close">×</button>
          </div>

          <div class="modal-body">
            <p class="modal-description">
              Enter your email to start your free guest session. We'll use this to save your work and send you updates.
            </p>

            <div class="form-group">
              <label for="guest-email">Email Address</label>
              <input
                type="email"
                id="guest-email"
                bind:value={guestEmail}
                on:keydown={handleGuestKeydown}
                placeholder="you@example.com"
                disabled={isGuestSubmitting}
              />
            </div>

            <p class="modal-hint">
              🕐 Guest sessions last 30 minutes. Sign in with PipeOps for unlimited access.
            </p>
          </div>

          <div class="modal-footer">
            <button class="btn btn-secondary" on:click={closeGuestModal} disabled={isGuestSubmitting}>
              Cancel
            </button>
            <button
              class="btn btn-primary"
              on:click={handleGuestSubmit}
              disabled={isGuestSubmitting || !guestEmail.trim()}
            >
              {isGuestSubmitting ? 'Starting...' : 'Start Terminal'}
            </button>
          </div>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .app {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .main {
    flex: 1;
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
    width: 100%;
  }

  .loading-screen {
    position: fixed;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    background: var(--bg);
  }

  .loading-screen p {
    color: var(--text-muted);
    font-size: 14px;
  }

  .spinner-large {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  /* Guest Modal Styles */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
    animation: fadeIn 0.15s ease;
  }

  .modal {
    background: var(--bg-card);
    border: 1px solid var(--border);
    width: 100%;
    max-width: 420px;
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
    letter-spacing: 1px;
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
  }

  .modal-description {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0 0 20px;
    line-height: 1.5;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 16px;
  }

  .form-group label {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
  }

  .form-group input {
    width: 100%;
    padding: 12px 14px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 14px;
  }

  .form-group input:focus {
    outline: none;
    border-color: var(--accent);
  }

  .form-group input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .modal-hint {
    font-size: 11px;
    color: var(--text-muted);
    margin: 0;
    padding: 12px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding: 16px 20px;
    border-top: 1px solid var(--border);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
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

  @media (max-width: 600px) {
    .modal {
      margin: 16px;
    }
  }
</style>
