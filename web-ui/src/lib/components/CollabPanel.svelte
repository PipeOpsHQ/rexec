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
    width: 320px;
    background: var(--bg-card, #0a0a0f);
    border: 1px solid var(--border, #1a1a2a);
    border-radius: 8px;
    z-index: 100;
    font-size: 12px;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6), 0 0 1px rgba(0, 255, 136, 0.3);
    overflow: hidden;
  }

  .collab-panel.compact {
    width: 280px;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    background: linear-gradient(180deg, rgba(0, 255, 136, 0.06) 0%, transparent 100%);
    border-bottom: 1px solid rgba(0, 255, 136, 0.15);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .collab-icon {
    font-size: 14px;
    filter: drop-shadow(0 0 4px rgba(0, 255, 136, 0.4));
  }

  .title {
    color: var(--accent, #00ff88);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 2px;
  }

  .close-btn {
    background: transparent;
    border: 1px solid transparent;
    color: #666;
    font-size: 18px;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
    transition: all 0.15s;
  }

  .close-btn:hover {
    color: var(--accent, #00ff88);
    border-color: rgba(0, 255, 136, 0.3);
    background: rgba(0, 255, 136, 0.1);
  }

  .panel-content {
    padding: 14px;
  }

  .option-row {
    margin-bottom: 16px;
  }

  .label {
    display: block;
    color: #666;
    font-size: 10px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin-bottom: 8px;
  }

  .mode-toggle {
    display: flex;
    gap: 6px;
  }

  .mode-btn {
    flex: 1;
    padding: 10px 12px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    color: #888;
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s;
  }

  .mode-btn:hover {
    background: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.15);
  }

  .mode-btn.active {
    background: rgba(0, 255, 136, 0.1);
    border-color: rgba(0, 255, 136, 0.4);
    color: var(--accent, #00ff88);
    box-shadow: 0 0 12px rgba(0, 255, 136, 0.15);
  }

  .slider-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 12px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
  }

  .slider {
    flex: 1;
    height: 4px;
    -webkit-appearance: none;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 2px;
  }

  .slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 14px;
    height: 14px;
    background: linear-gradient(135deg, var(--accent, #00ff88), #00cc6a);
    border-radius: 50%;
    cursor: pointer;
    box-shadow: 0 0 8px rgba(0, 255, 136, 0.4);
    transition: all 0.15s;
  }

  .slider::-webkit-slider-thumb:hover {
    transform: scale(1.1);
    box-shadow: 0 0 12px rgba(0, 255, 136, 0.6);
  }

  .slider::-moz-range-thumb {
    width: 14px;
    height: 14px;
    background: linear-gradient(135deg, var(--accent, #00ff88), #00cc6a);
    border-radius: 50%;
    cursor: pointer;
    border: none;
    box-shadow: 0 0 8px rgba(0, 255, 136, 0.4);
  }

  .slider-value {
    color: var(--accent, #00ff88);
    font-weight: 600;
    min-width: 24px;
    text-align: center;
    font-size: 13px;
  }

  .start-btn {
    width: 100%;
    padding: 12px;
    background: linear-gradient(135deg, var(--accent, #00ff88), #00cc6a);
    border: none;
    border-radius: 6px;
    color: #000;
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 1px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    transition: all 0.2s;
    box-shadow: 0 4px 16px rgba(0, 255, 136, 0.3);
  }

  .start-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 20px rgba(0, 255, 136, 0.4);
  }

  .start-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
  }

  .share-section {
    margin-bottom: 16px;
  }

  .code-box {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px;
    background: rgba(0, 255, 136, 0.08);
    border: 1px solid rgba(0, 255, 136, 0.25);
    border-radius: 8px;
    margin-bottom: 8px;
  }

  .share-code {
    font-size: 22px;
    font-weight: 700;
    letter-spacing: 4px;
    color: var(--accent, #00ff88);
    text-shadow: 0 0 12px rgba(0, 255, 136, 0.5);
  }

  .copy-btn {
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 4px;
    cursor: pointer;
    font-size: 14px;
    padding: 6px 10px;
    transition: all 0.15s;
  }

  .copy-btn:hover {
    background: rgba(0, 255, 136, 0.15);
    border-color: rgba(0, 255, 136, 0.3);
  }

  .share-url {
    width: 100%;
    padding: 8px 10px;
    background: rgba(0, 0, 0, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    color: #666;
    font-size: 10px;
    font-family: inherit;
  }

  .share-url:focus {
    outline: none;
    border-color: rgba(0, 255, 136, 0.3);
  }

  .participants-section {
    margin-bottom: 14px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    color: #666;
    font-size: 10px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin-bottom: 8px;
  }

  .count {
    color: var(--accent, #00ff88);
  }

  .participants-list {
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    max-height: 120px;
    overflow-y: auto;
  }

  .participant {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    transition: background 0.15s;
  }

  .participant:last-child {
    border-bottom: none;
  }

  .participant:hover {
    background: rgba(0, 255, 136, 0.05);
  }

  .avatar {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 700;
    color: #000;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  }

  .name {
    flex: 1;
    color: #ddd;
    font-size: 12px;
    font-weight: 500;
  }

  .role-tag {
    font-size: 9px;
    padding: 3px 8px;
    background: rgba(0, 255, 136, 0.1);
    border: 1px solid rgba(0, 255, 136, 0.2);
    border-radius: 4px;
    color: var(--accent, #00ff88);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .empty {
    color: #555;
    text-align: center;
    padding: 16px;
    margin: 0;
    font-size: 11px;
  }

  .status-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    margin-bottom: 12px;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #666;
    transition: all 0.3s;
  }

  .status-dot.live {
    background: var(--accent, #00ff88);
    box-shadow: 0 0 10px var(--accent, #00ff88);
    animation: pulse-live 1.5s ease-in-out infinite;
  }

  @keyframes pulse-live {
    0%, 100% { 
      opacity: 1; 
      box-shadow: 0 0 10px var(--accent, #00ff88);
    }
    50% { 
      opacity: 0.7; 
      box-shadow: 0 0 20px var(--accent, #00ff88);
    }
  }

  .status-text {
    flex: 1;
    color: #888;
    font-size: 11px;
    font-weight: 500;
  }

  .mode-tag {
    font-size: 9px;
    padding: 3px 8px;
    background: rgba(0, 217, 255, 0.1);
    border: 1px solid rgba(0, 217, 255, 0.2);
    border-radius: 4px;
    color: #00d9ff;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .end-btn {
    width: 100%;
    padding: 10px;
    background: rgba(255, 68, 68, 0.1);
    border: 1px solid rgba(255, 68, 68, 0.3);
    border-radius: 6px;
    color: #ff6666;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 1px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .end-btn:hover {
    background: rgba(255, 68, 68, 0.15);
    border-color: #ff4444;
    box-shadow: 0 0 16px rgba(255, 68, 68, 0.2);
  }

  .spinner-sm {
    width: 14px;
    height: 14px;
    border: 2px solid transparent;
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
    background: rgba(0, 255, 136, 0.3);
    border-radius: 2px;
  }

  .participants-list::-webkit-scrollbar-thumb:hover {
    background: rgba(0, 255, 136, 0.5);
  }

  /* Firefox scrollbar */
  .participants-list {
    scrollbar-width: thin;
    scrollbar-color: rgba(0, 255, 136, 0.3) transparent;
  }
</style>
