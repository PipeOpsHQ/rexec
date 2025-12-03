<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { collab } from '../stores/collab';
  import { auth, isAuthenticated } from '../stores/auth';
  import { toast } from '../stores/toast';

  export let code: string = '';

  const dispatch = createEventDispatcher();

  let isLoading = true;
  let error = '';
  let sessionInfo: {
    sessionId: string;
    containerId: string;
    mode: string;
    role: string;
    expiresAt: string;
  } | null = null;

  onMount(async () => {
    if (!code) {
      error = 'Invalid session code';
      isLoading = false;
      return;
    }

    // If not authenticated, prompt for guest login
    if (!$isAuthenticated) {
      error = 'Please sign in or start a guest session to join';
      isLoading = false;
      return;
    }

    // Try to join the session
    try {
      const result = await collab.joinSession(code);
      if (result) {
        sessionInfo = result;
        // Connect to the collab websocket
        collab.connectWebSocket(code);
        toast.success(`Joined session as ${result.role}`);
      } else {
        error = 'Session not found or has expired';
      }
    } catch (e) {
      error = 'Failed to join session';
    }
    
    isLoading = false;
  });

  function joinTerminal() {
    if (sessionInfo) {
      dispatch('joined', {
        containerId: sessionInfo.containerId,
        containerName: `Collab Session (${code})`
      });
    }
  }

  function cancel() {
    dispatch('cancel');
  }
</script>

<div class="join-container">
  <div class="join-card">
    <div class="join-header">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
        <circle cx="9" cy="7" r="4"/>
        <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
        <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
      </svg>
      <h1>Join Session</h1>
      <p class="code-display">{code}</p>
    </div>

    {#if isLoading}
      <div class="loading">
        <div class="spinner"></div>
        <p>Connecting to session...</p>
      </div>
    {:else if error}
      <div class="error-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
        <p>{error}</p>
        <button class="btn btn-secondary" on:click={cancel}>Go Back</button>
      </div>
    {:else if sessionInfo}
      <div class="session-info">
        <div class="info-row">
          <span class="label">Mode</span>
          <span class="value mode-badge" class:control={sessionInfo.mode === 'control'}>
            {sessionInfo.mode === 'control' ? 'Full Control' : 'View Only'}
          </span>
        </div>
        <div class="info-row">
          <span class="label">Your Role</span>
          <span class="value role-badge">{sessionInfo.role}</span>
        </div>
        <div class="info-row">
          <span class="label">Expires</span>
          <span class="value">{new Date(sessionInfo.expiresAt).toLocaleTimeString()}</span>
        </div>
      </div>

      <div class="actions">
        <button class="btn btn-secondary" on:click={cancel}>Cancel</button>
        <button class="btn btn-primary" on:click={joinTerminal}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="4 17 10 11 4 5"/>
            <line x1="12" y1="19" x2="20" y2="19"/>
          </svg>
          Open Terminal
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  .join-container {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 60vh;
    padding: 20px;
  }

  .join-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    width: 100%;
    max-width: 400px;
    padding: 32px;
  }

  .join-header {
    text-align: center;
    margin-bottom: 24px;
  }

  .join-header svg {
    color: var(--accent);
    margin-bottom: 16px;
  }

  .join-header h1 {
    font-size: 20px;
    text-transform: uppercase;
    letter-spacing: 2px;
    margin: 0 0 12px;
  }

  .code-display {
    font-family: var(--font-mono);
    font-size: 28px;
    font-weight: 700;
    letter-spacing: 4px;
    color: var(--accent);
    margin: 0;
  }

  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 32px 0;
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .loading p {
    color: var(--text-muted);
    font-size: 14px;
  }

  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 24px 0;
    text-align: center;
  }

  .error-state svg {
    color: var(--error);
    opacity: 0.7;
  }

  .error-state p {
    color: var(--text-secondary);
    margin: 0;
  }

  .session-info {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 20px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    margin-bottom: 24px;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .label {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
  }

  .value {
    font-size: 14px;
    color: var(--text);
  }

  .mode-badge {
    padding: 4px 10px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .mode-badge.control {
    border-color: var(--accent);
    color: var(--accent);
  }

  .role-badge {
    padding: 4px 10px;
    background: var(--accent);
    color: #000;
    font-size: 12px;
    font-weight: 600;
    text-transform: capitalize;
  }

  .actions {
    display: flex;
    gap: 12px;
  }

  .actions .btn {
    flex: 1;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 12px 20px;
    font-size: 14px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border: 1px solid var(--border);
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary {
    background: transparent;
    color: var(--text-secondary);
  }

  .btn-secondary:hover {
    background: var(--bg-secondary);
    color: var(--text);
  }

  .btn-primary {
    background: var(--accent);
    border-color: var(--accent);
    color: #000;
  }

  .btn-primary:hover {
    filter: brightness(1.1);
  }
</style>
