<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { collab } from '../stores/collab';
  import { slide } from 'svelte/transition';

  export let containerId: string;
  export let isOpen = false;
  export let compact = false;

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

  function copyCode() {
    navigator.clipboard.writeText(shareCode);
    copied = true;
    setTimeout(() => copied = false, 2000);
  }

  function close() {
    isOpen = false;
    dispatch('close');
  }
</script>

{#if isOpen}
  <div class="collab-panel" class:compact transition:slide={{ duration: 200 }}>
    <div class="panel-header">
      <div class="header-left">
        <span class="collab-icon">👥</span>
        <span class="title">SHARE</span>
      </div>
      <button class="close-btn" on:click={close}>×</button>
    </div>

    {#if !session}
      <div class="panel-content">
        <div class="option-row">
          <span class="label">Mode</span>
          <div class="mode-toggle">
            <button class="mode-btn" class:active={mode === 'view'} on:click={() => mode = 'view'}>
              👁 View
            </button>
            <button class="mode-btn" class:active={mode === 'control'} on:click={() => mode = 'control'}>
              ✏️ Control
            </button>
          </div>
        </div>

        <div class="option-row">
          <span class="label">Max users</span>
          <div class="slider-row">
            <input type="range" min="2" max="10" bind:value={maxUsers} class="slider" />
            <span class="slider-value">{maxUsers}</span>
          </div>
        </div>

        <button class="start-btn" on:click={startSession} disabled={isStarting}>
          {#if isStarting}
            <span class="spinner-sm"></span>
          {/if}
          {isStarting ? 'Starting...' : 'Start Session'}
        </button>
      </div>
    {:else}
      <div class="panel-content">
        <div class="share-section">
          <div class="code-box">
            <span class="share-code">{shareCode}</span>
            <button class="copy-btn" on:click={copyCode}>
              {copied ? '✓' : '📋'}
            </button>
          </div>
          <input class="share-url" readonly value={shareUrl} on:click|stopPropagation={(e) => e.currentTarget.select()} />
        </div>

        <div class="participants-section">
          <div class="section-header">
            <span>Participants</span>
            <span class="count">{participants.length}/{maxUsers}</span>
          </div>
          <div class="participants-list">
            {#each participants as p}
              <div class="participant">
                <span class="avatar" style="background: {p.color}">{p.username.charAt(0)}</span>
                <span class="name">{p.username}</span>
                <span class="role-tag">{p.role}</span>
              </div>
            {:else}
              <p class="empty">Waiting for others...</p>
            {/each}
          </div>
        </div>

        <div class="status-row">
          <span class="status-dot" class:live={isConnected}></span>
          <span class="status-text">{isConnected ? 'Live' : 'Connecting'}</span>
          <span class="mode-tag">{mode === 'view' ? 'View' : 'Control'}</span>
        </div>

        <button class="end-btn" on:click={endSession}>End Session</button>
      </div>
    {/if}
  </div>
{/if}

<style>
  .collab-panel {
    position: absolute;
    right: 8px;
    top: 40px;
    width: 300px;
    background: #0d0d1a;
    border: 1px solid #1a1a2e;
    z-index: 100;
    font-size: 12px;
  }

  .collab-panel.compact {
    width: 260px;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background: #1a1a2e;
    border-bottom: 1px solid #252542;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .collab-icon {
    font-size: 12px;
  }

  .title {
    color: #888;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 1px;
  }

  .close-btn {
    background: none;
    border: none;
    color: #666;
    font-size: 16px;
    cursor: pointer;
    padding: 0 4px;
  }

  .close-btn:hover {
    color: #fff;
  }

  .panel-content {
    padding: 12px;
  }

  .option-row {
    margin-bottom: 12px;
  }

  .label {
    display: block;
    color: #666;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 6px;
  }

  .mode-toggle {
    display: flex;
    gap: 4px;
  }

  .mode-btn {
    flex: 1;
    padding: 8px;
    background: #0a0a14;
    border: 1px solid #1a1a2e;
    color: #888;
    font-size: 11px;
    cursor: pointer;
  }

  .mode-btn:hover {
    background: #151525;
  }

  .mode-btn.active {
    background: #1a2a1a;
    border-color: #00ff88;
    color: #00ff88;
  }

  .slider-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .slider {
    flex: 1;
    height: 4px;
    -webkit-appearance: none;
    background: #1a1a2e;
    border-radius: 2px;
  }

  .slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 12px;
    height: 12px;
    background: #00ff88;
    border-radius: 50%;
    cursor: pointer;
  }

  .slider-value {
    color: #00ff88;
    font-family: 'JetBrains Mono', monospace;
    min-width: 20px;
    text-align: center;
  }

  .start-btn {
    width: 100%;
    padding: 10px;
    background: linear-gradient(135deg, #00ff88, #00cc6a);
    border: none;
    color: #000;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
  }

  .start-btn:hover:not(:disabled) {
    filter: brightness(1.1);
  }

  .start-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .share-section {
    margin-bottom: 12px;
  }

  .code-box {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    background: #0a0a14;
    border: 1px solid #1a1a2e;
    margin-bottom: 6px;
  }

  .share-code {
    font-family: 'JetBrains Mono', monospace;
    font-size: 18px;
    font-weight: 700;
    letter-spacing: 3px;
    color: #00ff88;
  }

  .copy-btn {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 14px;
    padding: 4px;
  }

  .share-url {
    width: 100%;
    padding: 6px 8px;
    background: #0a0a14;
    border: 1px solid #1a1a2e;
    color: #666;
    font-size: 10px;
    font-family: monospace;
  }

  .participants-section {
    margin-bottom: 12px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    color: #666;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 6px;
  }

  .count {
    color: #888;
  }

  .participants-list {
    background: #0a0a14;
    border: 1px solid #1a1a2e;
    max-height: 100px;
    overflow-y: auto;
  }

  .participant {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    border-bottom: 1px solid #1a1a2e;
  }

  .participant:last-child {
    border-bottom: none;
  }

  .avatar {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    font-weight: 600;
    color: #000;
  }

  .name {
    flex: 1;
    color: #ddd;
    font-size: 11px;
  }

  .role-tag {
    font-size: 9px;
    padding: 2px 6px;
    background: #1a1a2e;
    color: #888;
    text-transform: uppercase;
  }

  .empty {
    color: #666;
    text-align: center;
    padding: 12px;
    margin: 0;
    font-size: 11px;
  }

  .status-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px;
    background: #0a0a14;
    border: 1px solid #1a1a2e;
    margin-bottom: 10px;
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #666;
  }

  .status-dot.live {
    background: #00ff88;
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }

  .status-text {
    flex: 1;
    color: #888;
    font-size: 11px;
  }

  .mode-tag {
    font-size: 9px;
    padding: 2px 6px;
    background: #1a1a2e;
    color: #888;
    text-transform: uppercase;
  }

  .end-btn {
    width: 100%;
    padding: 8px;
    background: #1a0a0a;
    border: 1px solid #3a1a1a;
    color: #ff6666;
    font-size: 11px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    cursor: pointer;
  }

  .end-btn:hover {
    background: #2a0a0a;
    border-color: #ff4444;
  }

  .spinner-sm {
    width: 12px;
    height: 12px;
    border: 1.5px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Scrollbar */
  .participants-list::-webkit-scrollbar {
    width: 4px;
  }

  .participants-list::-webkit-scrollbar-track {
    background: transparent;
  }

  .participants-list::-webkit-scrollbar-thumb {
    background: #333;
  }
</style>
