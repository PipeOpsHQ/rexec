<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { collab } from '../stores/collab';
  import { auth, isAuthenticated } from '../stores/auth';
  import { toast } from '../stores/toast';

  export let code: string = '';

  const dispatch = createEventDispatcher();

  let isLoading = true;
  let error = '';
  let needsAuth = false;
  let guestEmail = '';
  let isSubmittingGuest = false;
  let sessionInfo: {
    sessionId: string;
    containerId: string;
    containerName: string;
    mode: string;
    role: string;
    expiresAt: string;
  } | null = null;

  async function attemptJoin() {
    if (!code) {
      error = 'Invalid session code';
      isLoading = false;
      return;
    }

    // If not authenticated, show guest login form
    if (!$isAuthenticated) {
      needsAuth = true;
      isLoading = false;
      return;
    }

    // Try to join the session
    try {
      const result = await collab.joinSession(code);
      if (result) {
        sessionInfo = {
          sessionId: result.id,
          containerId: result.containerId,
          containerName: result.containerName,
          mode: result.mode,
          role: result.role,
          expiresAt: result.expiresAt
        };
        // Connect to the collab websocket
        collab.connectWebSocket(code);
        toast.success(`Joined terminal as ${result.role}`);
      } else {
        error = 'Terminal not found or sharing has ended';
      }
    } catch (e) {
      error = 'Failed to join terminal';
    }
    
    isLoading = false;
  }

  onMount(() => {
    attemptJoin();
  });

  // React to auth changes - retry join when user authenticates
  $: if ($isAuthenticated && needsAuth) {
    needsAuth = false;
    isLoading = true;
    attemptJoin();
  }

  async function handleGuestSubmit() {
    if (!guestEmail.trim()) {
      toast.error('Please enter your email');
      return;
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(guestEmail.trim())) {
      toast.error('Please enter a valid email');
      return;
    }

    isSubmittingGuest = true;
    const result = await auth.guestLogin(guestEmail.trim());
    isSubmittingGuest = false;

    if (result.success) {
      // attemptJoin will be triggered by the reactive statement above
      toast.success('Guest session started!');
    } else {
      toast.error('Failed to start guest session');
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !isSubmittingGuest) {
      handleGuestSubmit();
    }
  }

  function joinTerminal() {
    if (sessionInfo) {
      dispatch('joined', {
        containerId: sessionInfo.containerId,
        containerName: `Shared Terminal (${code})`,
        mode: sessionInfo.mode,
        role: sessionInfo.role
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
      <div class="join-icon">
        <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
          <line x1="8" y1="21" x2="16" y2="21"/>
          <line x1="12" y1="17" x2="12" y2="21"/>
        </svg>
      </div>
      <h1>Join Shared Terminal</h1>
      <p class="code-display">{code}</p>
    </div>

    {#if isLoading}
      <div class="loading">
        <div class="spinner"></div>
        <p>Connecting to terminal...</p>
      </div>
    {:else if needsAuth}
      <div class="auth-prompt">
        <p class="auth-description">Login to join this shared terminal session</p>
        
        <!-- PipeOps Login Button -->
        <button class="btn btn-pipeops" onclick={() => auth.login()}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
          </svg>
          Login with PipeOps
        </button>
        
        <div class="auth-divider">
          <span>or continue as guest</span>
        </div>
        
        <div class="form-group">
          <label for="guest-email">Email Address</label>
          <input
            type="email"
            id="guest-email"
            bind:value={guestEmail}
            onkeydown={handleKeydown}
            placeholder="you@example.com"
            disabled={isSubmittingGuest}
          />
        </div>
        <div class="actions">
          <button class="btn btn-secondary" onclick={cancel} disabled={isSubmittingGuest}>Cancel</button>
          <button class="btn btn-primary" onclick={handleGuestSubmit} disabled={isSubmittingGuest || !guestEmail.trim()}>
            {isSubmittingGuest ? 'Connecting...' : 'Join as Guest'}
          </button>
        </div>
      </div>
    {:else if error}
      <div class="error-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
        <p>{error}</p>
        <button class="btn btn-secondary" onclick={cancel}>Go Back</button>
      </div>
    {:else if sessionInfo}
      <div class="session-info">
        <div class="terminal-card">
          <div class="terminal-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5"/>
              <line x1="12" y1="19" x2="20" y2="19"/>
            </svg>
          </div>
          <div class="terminal-details">
            <span class="terminal-name">{sessionInfo.containerName}</span>
            <span class="terminal-shared">
              <span class="shared-badge">LIVE</span>
              Shared Terminal
            </span>
          </div>
        </div>
        
        <div class="info-grid">
          <div class="info-item">
            <span class="label">Access</span>
            <span class="value mode-badge" class:control={sessionInfo.mode === 'control'}>
              {sessionInfo.mode === 'control' ? 'Full Control' : 'View Only'}
            </span>
          </div>
          <div class="info-item">
            <span class="label">Your Role</span>
            <span class="value role-badge">{sessionInfo.role}</span>
          </div>
          <div class="info-item full-width">
            <span class="label">Session Expires</span>
            <span class="value">{new Date(sessionInfo.expiresAt).toLocaleTimeString()}</span>
          </div>
        </div>
      </div>

      <div class="actions">
        <button class="btn btn-secondary" onclick={cancel}>Cancel</button>
        <button class="btn btn-primary" onclick={joinTerminal}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="4 17 10 11 4 5"/>
            <line x1="12" y1="19" x2="20" y2="19"/>
          </svg>
          Connect Now
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
    min-height: 80vh;
    padding: 20px;
    background: linear-gradient(180deg, rgba(0, 255, 136, 0.02) 0%, transparent 50%);
  }

  .join-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    width: 100%;
    max-width: 420px;
    padding: 32px;
    box-shadow: 0 8px 40px rgba(0, 0, 0, 0.5);
  }

  .join-header {
    text-align: center;
    margin-bottom: 28px;
  }

  .join-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 64px;
    height: 64px;
    background: rgba(0, 255, 136, 0.1);
    border: 1px solid rgba(0, 255, 136, 0.2);
    border-radius: 12px;
    margin-bottom: 16px;
  }

  .join-icon svg {
    color: var(--accent);
  }

  .join-header h1 {
    font-size: 18px;
    text-transform: uppercase;
    letter-spacing: 2px;
    margin: 0 0 12px;
    color: var(--text);
  }

  .code-display {
    font-family: var(--font-mono);
    font-size: 24px;
    font-weight: 700;
    letter-spacing: 6px;
    color: var(--accent);
    margin: 0;
    padding: 12px 20px;
    background: rgba(0, 255, 136, 0.05);
    border: 1px dashed rgba(0, 255, 136, 0.3);
    display: inline-block;
  }

  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 40px 0;
  }

  .spinner {
    width: 36px;
    height: 36px;
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
    font-size: 13px;
    margin: 0;
  }

  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 32px 0;
    text-align: center;
  }

  .error-state svg {
    color: var(--error);
    opacity: 0.7;
  }

  .error-state p {
    color: var(--text-secondary);
    margin: 0;
    font-size: 14px;
  }

  .session-info {
    margin-bottom: 24px;
  }

  .terminal-card {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 16px;
    background: rgba(0, 255, 65, 0.05);
    border: 1px solid rgba(0, 255, 65, 0.2);
    margin-bottom: 16px;
  }

  .terminal-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    background: rgba(0, 255, 136, 0.1);
    border-radius: 8px;
  }

  .terminal-icon svg {
    color: var(--accent);
  }

  .terminal-details {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
  }

  .terminal-name {
    font-size: 16px;
    font-weight: 600;
    color: var(--text);
    font-family: var(--font-mono);
  }

  .terminal-shared {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 11px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .shared-badge {
    padding: 2px 6px;
    background: var(--accent);
    color: #000;
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.5px;
    animation: pulse-badge 2s ease-in-out infinite;
  }

  @keyframes pulse-badge {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.7; }
  }

  .info-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    padding: 16px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
  }

  .info-item {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .info-item.full-width {
    grid-column: span 2;
  }

  .label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
  }

  .value {
    font-size: 13px;
    color: var(--text);
  }

  .mode-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    width: fit-content;
  }

  .mode-badge.control {
    border-color: var(--accent);
    color: var(--accent);
    background: rgba(0, 255, 136, 0.05);
  }

  .role-badge {
    display: inline-block;
    padding: 6px 12px;
    background: var(--accent);
    color: #000;
    font-size: 11px;
    font-weight: 600;
    text-transform: capitalize;
    width: fit-content;
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
    padding: 14px 24px;
    font-size: 13px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 1px;
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

  .auth-prompt {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .auth-description {
    color: var(--text-secondary);
    font-size: 14px;
    text-align: center;
    margin: 0;
    line-height: 1.5;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .form-group label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
  }

  .form-group input {
    width: 100%;
    padding: 14px 16px;
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

  .btn-pipeops {
    width: 100%;
    background: linear-gradient(135deg, #6366f1, #8b5cf6);
    border-color: #6366f1;
    color: #fff;
    font-size: 14px;
    padding: 16px 24px;
  }

  .btn-pipeops:hover {
    filter: brightness(1.1);
  }

  .auth-divider {
    display: flex;
    align-items: center;
    gap: 16px;
    color: var(--text-muted);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .auth-divider::before,
  .auth-divider::after {
    content: '';
    flex: 1;
    height: 1px;
    background: var(--border);
  }
</style>
