<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { collab } from '../stores/collab';
  import { fade, fly } from 'svelte/transition';

  export let containerId: string;
  export let isOpen = false;

  const dispatch = createEventDispatcher();

  let mode: 'view' | 'control' = 'view';
  let maxUsers = 5;
  let isStarting = false;
  let shareCode = '';
  let shareUrl = '';
  let copied = false;

  $: session = $collab.activeSession;
  $: participants = $collab.participants;
  $: isConnected = $collab.isConnected;

  async function startSession() {
    isStarting = true;
    const result = await collab.startSession(containerId, mode, maxUsers);
    isStarting = false;
    
    if (result) {
      shareCode = result.shareCode;
      shareUrl = `${window.location.origin}/join/${shareCode}`;
      collab.connectWebSocket(shareCode);
    }
  }

  async function endSession() {
    if (session) {
      await collab.endSession(session.id);
    }
    close();
  }

  function copyLink() {
    navigator.clipboard.writeText(shareUrl);
    copied = true;
    setTimeout(() => copied = false, 2000);
  }

  function close() {
    isOpen = false;
    dispatch('close');
  }
</script>

{#if isOpen}
  <div class="collab-overlay" transition:fade={{ duration: 200 }} on:click={close}>
    <div class="collab-panel" transition:fly={{ y: 20, duration: 300 }} on:click|stopPropagation>
      <div class="panel-header">
        <h3>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
            <circle cx="9" cy="7" r="4"/>
            <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
          </svg>
          Collaborate
        </h3>
        <button class="close-btn" on:click={close}>✕</button>
      </div>

      {#if !session}
        <div class="setup-section">
          <p class="description">Share your terminal session with others in real-time.</p>
          
          <div class="option-group">
            <label>Access Mode</label>
            <div class="mode-buttons">
              <button 
                class="mode-btn" 
                class:active={mode === 'view'}
                on:click={() => mode = 'view'}
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                  <circle cx="12" cy="12" r="3"/>
                </svg>
                View Only
              </button>
              <button 
                class="mode-btn" 
                class:active={mode === 'control'}
                on:click={() => mode = 'control'}
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 20h9"/>
                  <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/>
                </svg>
                Full Control
              </button>
            </div>
          </div>

          <div class="option-group">
            <label>Max Participants</label>
            <input type="range" min="2" max="10" bind:value={maxUsers} />
            <span class="value">{maxUsers} users</span>
          </div>

          <button class="start-btn" on:click={startSession} disabled={isStarting}>
            {#if isStarting}
              <span class="spinner"></span>
              Starting...
            {:else}
              Start Session
            {/if}
          </button>
        </div>
      {:else}
        <div class="active-section">
          <div class="share-box">
            <label>Share Code</label>
            <div class="code-display">
              <span class="code">{shareCode}</span>
              <button class="copy-btn" on:click={copyLink}>
                {copied ? '✓ Copied' : 'Copy Link'}
              </button>
            </div>
            <input class="share-url" readonly value={shareUrl} />
          </div>

          <div class="participants-section">
            <h4>Participants ({participants.length})</h4>
            <div class="participants-list">
              {#each participants as participant}
                <div class="participant" style="--color: {participant.color}">
                  <span class="avatar" style="background: {participant.color}">
                    {participant.username.charAt(0).toUpperCase()}
                  </span>
                  <span class="name">{participant.username}</span>
                  <span class="role">{participant.role}</span>
                </div>
              {:else}
                <p class="empty">Waiting for participants...</p>
              {/each}
            </div>
          </div>

          <div class="status-bar">
            <span class="status" class:connected={isConnected}>
              <span class="dot"></span>
              {isConnected ? 'Live' : 'Connecting...'}
            </span>
            <span class="mode-badge">{mode === 'view' ? 'View Only' : 'Full Control'}</span>
          </div>

          <button class="end-btn" on:click={endSession}>
            End Session
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .collab-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
  }

  .collab-panel {
    background: #1a1a2e;
    border: 1px solid #2a2a4e;
    border-radius: 12px;
    width: 400px;
    max-width: 90vw;
    overflow: hidden;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid #2a2a4e;
  }

  .panel-header h3 {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0;
    font-size: 1.1rem;
    color: #fff;
  }

  .close-btn {
    background: none;
    border: none;
    color: #666;
    font-size: 1.25rem;
    cursor: pointer;
    padding: 0.25rem;
  }

  .close-btn:hover {
    color: #fff;
  }

  .setup-section, .active-section {
    padding: 1.25rem;
  }

  .description {
    color: #888;
    margin: 0 0 1.25rem;
    font-size: 0.9rem;
  }

  .option-group {
    margin-bottom: 1.25rem;
  }

  .option-group label {
    display: block;
    color: #aaa;
    font-size: 0.85rem;
    margin-bottom: 0.5rem;
  }

  .mode-buttons {
    display: flex;
    gap: 0.5rem;
  }

  .mode-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.75rem;
    background: #252542;
    border: 1px solid #3a3a5e;
    border-radius: 8px;
    color: #888;
    cursor: pointer;
    transition: all 0.2s;
  }

  .mode-btn:hover {
    background: #2a2a4e;
    color: #fff;
  }

  .mode-btn.active {
    background: #3a3a6e;
    border-color: #00ff88;
    color: #00ff88;
  }

  input[type="range"] {
    width: 100%;
    margin: 0.5rem 0;
  }

  .value {
    color: #00ff88;
    font-size: 0.85rem;
  }

  .start-btn {
    width: 100%;
    padding: 0.875rem;
    background: linear-gradient(135deg, #00ff88, #00cc6a);
    border: none;
    border-radius: 8px;
    color: #000;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    transition: transform 0.2s, box-shadow 0.2s;
  }

  .start-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 4px 20px rgba(0, 255, 136, 0.3);
  }

  .start-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .share-box {
    margin-bottom: 1.25rem;
  }

  .code-display {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }

  .code {
    font-family: 'JetBrains Mono', monospace;
    font-size: 1.5rem;
    font-weight: 700;
    color: #00ff88;
    letter-spacing: 0.1em;
  }

  .copy-btn {
    padding: 0.375rem 0.75rem;
    background: #252542;
    border: 1px solid #3a3a5e;
    border-radius: 6px;
    color: #888;
    font-size: 0.8rem;
    cursor: pointer;
  }

  .copy-btn:hover {
    background: #2a2a4e;
    color: #fff;
  }

  .share-url {
    width: 100%;
    padding: 0.5rem;
    background: #0a0a1a;
    border: 1px solid #2a2a4e;
    border-radius: 6px;
    color: #666;
    font-size: 0.75rem;
    font-family: monospace;
  }

  .participants-section h4 {
    color: #aaa;
    font-size: 0.85rem;
    margin: 0 0 0.75rem;
  }

  .participants-list {
    background: #0a0a1a;
    border-radius: 8px;
    padding: 0.5rem;
    max-height: 150px;
    overflow-y: auto;
  }

  .participant {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem;
    border-radius: 6px;
  }

  .participant:hover {
    background: #1a1a2e;
  }

  .avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.8rem;
    font-weight: 600;
    color: #000;
  }

  .name {
    flex: 1;
    color: #fff;
    font-size: 0.9rem;
  }

  .role {
    font-size: 0.7rem;
    padding: 0.2rem 0.5rem;
    background: #252542;
    border-radius: 4px;
    color: #888;
    text-transform: capitalize;
  }

  .empty {
    color: #666;
    text-align: center;
    padding: 1rem;
    font-size: 0.85rem;
  }

  .status-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem;
    background: #0a0a1a;
    border-radius: 8px;
    margin: 1rem 0;
  }

  .status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: #888;
    font-size: 0.85rem;
  }

  .status.connected {
    color: #00ff88;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: currentColor;
  }

  .status.connected .dot {
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }

  .mode-badge {
    font-size: 0.75rem;
    padding: 0.25rem 0.5rem;
    background: #252542;
    border-radius: 4px;
    color: #888;
  }

  .end-btn {
    width: 100%;
    padding: 0.75rem;
    background: #3a1a1a;
    border: 1px solid #5a2a2a;
    border-radius: 8px;
    color: #ff6b6b;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .end-btn:hover {
    background: #4a1a1a;
    border-color: #ff6b6b;
  }
</style>
