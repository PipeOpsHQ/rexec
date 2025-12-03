<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { recordings, type Recording } from '../stores/recordings';
  import { slide } from 'svelte/transition';

  export let containerId: string = '';
  export let isOpen = false;
  export let compact = false; // For inline terminal use

  const dispatch = createEventDispatcher();

  let recordingTitle = '';
  let isStarting = false;
  let currentTab: 'record' | 'library' = 'record';
  let selectedRecording: Recording | null = null;
  let playerElement: HTMLDivElement;

  $: recordingsList = $recordings.recordings;
  $: activeRecording = $recordings.activeRecordings.get(containerId);
  $: isRecordingActive = activeRecording?.recording || false;
  $: isLoading = $recordings.isLoading;

  onMount(() => {
    recordings.fetchRecordings();
    if (containerId) {
      recordings.getRecordingStatus(containerId);
    }
  });

  async function startRecording() {
    if (!containerId) return;
    isStarting = true;
    await recordings.startRecording(containerId, recordingTitle || undefined);
    isStarting = false;
    recordingTitle = '';
  }

  async function stopRecording() {
    if (!containerId) return;
    const result = await recordings.stopRecording(containerId);
    if (result) {
      currentTab = 'library';
    }
  }

  async function togglePublic(recording: Recording) {
    await recordings.updateRecording(recording.id, { isPublic: !recording.isPublic });
  }

  async function deleteRecording(recording: Recording) {
    if (confirm('Delete this recording?')) {
      await recordings.deleteRecording(recording.id);
    }
  }

  function copyShareLink(recording: Recording) {
    const url = `${window.location.origin}${recording.shareUrl}`;
    navigator.clipboard.writeText(url);
  }

  function playRecording(recording: Recording) {
    selectedRecording = recording;
  }

  function close() {
    isOpen = false;
    selectedRecording = null;
    dispatch('close');
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }
</script>

{#if isOpen}
  <div class="recording-panel" class:compact transition:slide={{ duration: 200 }}>
    <div class="panel-header">
      <div class="header-left">
        <span class="rec-icon">⏺</span>
        <span class="title">REC</span>
      </div>
      <button class="close-btn" on:click={close}>×</button>
    </div>

    {#if selectedRecording}
      <div class="player-section">
        <div class="player-bar">
          <button class="back-btn" on:click={() => selectedRecording = null}>←</button>
          <span class="player-title">{selectedRecording.title || 'Recording'}</span>
        </div>
        <div class="player-container" bind:this={playerElement}>
          <div class="player-placeholder">
            <span class="play-icon">▶</span>
            <span class="duration">{selectedRecording.duration}</span>
          </div>
        </div>
        <div class="player-actions">
          <a href={`/api/recordings/${selectedRecording.id}/stream`} class="action-link" download>
            ↓ Download .cast
          </a>
        </div>
      </div>
    {:else}
      <div class="tabs">
        <button class="tab" class:active={currentTab === 'record'} on:click={() => currentTab = 'record'}>
          Record
        </button>
        <button class="tab" class:active={currentTab === 'library'} on:click={() => currentTab = 'library'}>
          Library <span class="count">{recordingsList.length}</span>
        </button>
      </div>

      <div class="panel-content">
        {#if currentTab === 'record'}
          {#if isRecordingActive}
            <div class="active-recording">
              <div class="rec-status">
                <span class="rec-dot blink"></span>
                <span>Recording</span>
                <span class="elapsed">{Math.floor((activeRecording?.durationMs || 0) / 1000)}s</span>
              </div>
              <button class="stop-btn" on:click={stopRecording}>■ Stop</button>
            </div>
          {:else}
            <div class="start-section">
              {#if containerId}
                <input 
                  type="text" 
                  bind:value={recordingTitle}
                  placeholder="Recording title (optional)"
                  class="title-input"
                />
                <button class="start-btn" on:click={startRecording} disabled={isStarting}>
                  {#if isStarting}
                    <span class="spinner-sm"></span>
                  {:else}
                    <span class="rec-dot"></span>
                  {/if}
                  {isStarting ? 'Starting...' : 'Start Recording'}
                </button>
              {:else}
                <p class="hint">Connect to a terminal first</p>
              {/if}
            </div>
          {/if}
        {:else}
          <div class="library-list">
            {#if isLoading}
              <div class="loading"><span class="spinner-sm"></span> Loading...</div>
            {:else if recordingsList.length === 0}
              <p class="empty">No recordings yet</p>
            {:else}
              {#each recordingsList as recording}
                <div class="recording-item">
                  <div class="rec-info" on:click={() => playRecording(recording)}>
                    <span class="rec-title">{recording.title || 'Untitled'}</span>
                    <span class="rec-meta">{recording.duration} • {recordings.formatSize(recording.sizeBytes)}</span>
                  </div>
                  <div class="rec-actions">
                    <button 
                      class="icon-btn" 
                      class:active={recording.isPublic}
                      on:click={() => togglePublic(recording)}
                      title={recording.isPublic ? 'Public' : 'Private'}
                    >
                      {recording.isPublic ? '🌐' : '🔒'}
                    </button>
                    {#if recording.isPublic}
                      <button class="icon-btn" on:click={() => copyShareLink(recording)} title="Copy link">
                        🔗
                      </button>
                    {/if}
                    <button class="icon-btn delete" on:click={() => deleteRecording(recording)} title="Delete">
                      🗑
                    </button>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  .recording-panel {
    position: absolute;
    right: 8px;
    top: 40px;
    width: 320px;
    background: #0d0d1a;
    border: 1px solid #1a1a2e;
    z-index: 100;
    font-size: 12px;
  }

  .recording-panel.compact {
    width: 280px;
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

  .rec-icon {
    color: #ff4444;
    font-size: 10px;
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

  .tabs {
    display: flex;
    border-bottom: 1px solid #1a1a2e;
  }

  .tab {
    flex: 1;
    padding: 8px;
    background: none;
    border: none;
    color: #666;
    font-size: 11px;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    position: relative;
  }

  .tab:hover {
    color: #888;
  }

  .tab.active {
    color: #00ff88;
  }

  .tab.active::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 1px;
    background: #00ff88;
  }

  .count {
    opacity: 0.5;
    margin-left: 4px;
  }

  .panel-content {
    padding: 12px;
    max-height: 300px;
    overflow-y: auto;
  }

  .active-recording {
    display: flex;
    flex-direction: column;
    gap: 12px;
    text-align: center;
    padding: 16px 0;
  }

  .rec-status {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    color: #ff4444;
    font-size: 13px;
  }

  .elapsed {
    color: #888;
    font-family: 'JetBrains Mono', monospace;
  }

  .rec-dot {
    width: 8px;
    height: 8px;
    background: #ff4444;
    border-radius: 50%;
  }

  .rec-dot.blink {
    animation: blink 1s ease-in-out infinite;
  }

  @keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.3; }
  }

  .stop-btn {
    padding: 10px 20px;
    background: #ff4444;
    border: none;
    color: #fff;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 1px;
  }

  .stop-btn:hover {
    background: #ff5555;
  }

  .start-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .title-input {
    width: 100%;
    padding: 8px 10px;
    background: #0a0a14;
    border: 1px solid #1a1a2e;
    color: #fff;
    font-size: 12px;
  }

  .title-input:focus {
    outline: none;
    border-color: #00ff88;
  }

  .start-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 10px;
    background: #1a0a0a;
    border: 1px solid #3a1a1a;
    color: #ff6666;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .start-btn:hover:not(:disabled) {
    background: #2a0a0a;
    border-color: #ff4444;
  }

  .start-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .hint {
    color: #666;
    text-align: center;
    padding: 20px;
    margin: 0;
  }

  .library-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .loading, .empty {
    color: #666;
    text-align: center;
    padding: 20px;
    margin: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }

  .recording-item {
    display: flex;
    align-items: center;
    padding: 8px;
    background: #0a0a14;
    border: 1px solid transparent;
  }

  .recording-item:hover {
    border-color: #1a1a2e;
  }

  .rec-info {
    flex: 1;
    cursor: pointer;
    overflow: hidden;
  }

  .rec-title {
    display: block;
    color: #ddd;
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .rec-meta {
    color: #666;
    font-size: 10px;
  }

  .rec-actions {
    display: flex;
    gap: 2px;
  }

  .icon-btn {
    background: none;
    border: none;
    padding: 4px 6px;
    cursor: pointer;
    font-size: 12px;
    opacity: 0.5;
  }

  .icon-btn:hover {
    opacity: 1;
  }

  .icon-btn.active {
    opacity: 1;
  }

  .icon-btn.delete:hover {
    color: #ff4444;
  }

  .player-section {
    padding: 0;
  }

  .player-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    background: #1a1a2e;
    border-bottom: 1px solid #252542;
  }

  .back-btn {
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    font-size: 14px;
    padding: 0;
  }

  .back-btn:hover {
    color: #fff;
  }

  .player-title {
    color: #ddd;
    font-size: 12px;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .player-container {
    background: #000;
    aspect-ratio: 16/9;
    max-height: 180px;
  }

  .player-placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #666;
    gap: 8px;
  }

  .play-icon {
    font-size: 24px;
    opacity: 0.5;
  }

  .duration {
    font-size: 11px;
    font-family: 'JetBrains Mono', monospace;
  }

  .player-actions {
    padding: 10px 12px;
    text-align: center;
    border-top: 1px solid #1a1a2e;
  }

  .action-link {
    color: #888;
    font-size: 11px;
    text-decoration: none;
  }

  .action-link:hover {
    color: #00ff88;
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
  .panel-content::-webkit-scrollbar {
    width: 4px;
  }

  .panel-content::-webkit-scrollbar-track {
    background: transparent;
  }

  .panel-content::-webkit-scrollbar-thumb {
    background: #333;
  }
</style>
