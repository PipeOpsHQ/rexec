<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { recordings, type Recording } from '../stores/recordings';
  import { fade, fly } from 'svelte/transition';

  export let containerId: string = '';
  export let isOpen = false;

  const dispatch = createEventDispatcher();

  let recordingTitle = '';
  let isStarting = false;
  let currentTab: 'record' | 'library' = 'record';
  let selectedRecording: Recording | null = null;

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
  <div class="recording-overlay" transition:fade={{ duration: 200 }} on:click={close}>
    <div class="recording-panel" transition:fly={{ y: 20, duration: 300 }} on:click|stopPropagation>
      <div class="panel-header">
        <h3>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <circle cx="12" cy="12" r="3" fill="currentColor"/>
          </svg>
          Recordings
        </h3>
        <button class="close-btn" on:click={close}>✕</button>
      </div>

      {#if selectedRecording}
        <div class="player-section">
          <div class="player-header">
            <button class="back-btn" on:click={() => selectedRecording = null}>
              ← Back
            </button>
            <h4>{selectedRecording.title}</h4>
          </div>
          <div class="player-container">
            <div class="asciinema-player">
              <!-- In production, integrate asciinema-player here -->
              <div class="player-placeholder">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polygon points="5 3 19 12 5 21 5 3"/>
                </svg>
                <p>Terminal Recording</p>
                <span class="duration">{selectedRecording.duration}</span>
              </div>
            </div>
          </div>
          <div class="player-controls">
            <a href={`/api/recordings/${selectedRecording.id}/stream`} class="download-btn" download>
              Download .cast
            </a>
          </div>
        </div>
      {:else}
        <div class="tabs">
          <button 
            class="tab" 
            class:active={currentTab === 'record'}
            on:click={() => currentTab = 'record'}
          >
            Record
          </button>
          <button 
            class="tab" 
            class:active={currentTab === 'library'}
            on:click={() => currentTab = 'library'}
          >
            Library ({recordingsList.length})
          </button>
        </div>

        {#if currentTab === 'record'}
          <div class="record-section">
            {#if isRecordingActive}
              <div class="active-recording">
                <div class="recording-indicator">
                  <span class="rec-dot"></span>
                  <span>Recording</span>
                </div>
                <div class="recording-stats">
                  <div class="stat">
                    <span class="label">Duration</span>
                    <span class="value">{Math.floor((activeRecording?.durationMs || 0) / 1000)}s</span>
                  </div>
                  <div class="stat">
                    <span class="label">Events</span>
                    <span class="value">{activeRecording?.eventsCount || 0}</span>
                  </div>
                </div>
                <button class="stop-btn" on:click={stopRecording}>
                  <span class="stop-icon">■</span>
                  Stop Recording
                </button>
              </div>
            {:else}
              <div class="start-recording">
                <p class="description">
                  Record your terminal session to share or replay later.
                </p>
                
                {#if containerId}
                  <div class="input-group">
                    <label>Recording Title (optional)</label>
                    <input 
                      type="text" 
                      bind:value={recordingTitle}
                      placeholder="My awesome session"
                    />
                  </div>

                  <button class="start-btn" on:click={startRecording} disabled={isStarting}>
                    {#if isStarting}
                      <span class="spinner"></span>
                      Starting...
                    {:else}
                      <span class="rec-dot"></span>
                      Start Recording
                    {/if}
                  </button>
                {:else}
                  <p class="no-terminal">
                    Connect to a terminal to start recording.
                  </p>
                {/if}
              </div>
            {/if}
          </div>
        {:else}
          <div class="library-section">
            {#if isLoading}
              <div class="loading">
                <span class="spinner"></span>
                Loading recordings...
              </div>
            {:else if recordingsList.length === 0}
              <div class="empty-library">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="17 8 12 3 7 8"/>
                  <line x1="12" y1="3" x2="12" y2="15"/>
                </svg>
                <p>No recordings yet</p>
                <span>Start recording to capture your terminal sessions</span>
              </div>
            {:else}
              <div class="recordings-list">
                {#each recordingsList as recording}
                  <div class="recording-item">
                    <div class="recording-info" on:click={() => playRecording(recording)}>
                      <span class="title">{recording.title || 'Untitled Recording'}</span>
                      <div class="meta">
                        <span>{recording.duration}</span>
                        <span>•</span>
                        <span>{recordings.formatSize(recording.sizeBytes)}</span>
                        <span>•</span>
                        <span>{formatDate(recording.createdAt)}</span>
                      </div>
                    </div>
                    <div class="recording-actions">
                      <button 
                        class="action-btn" 
                        class:active={recording.isPublic}
                        on:click={() => togglePublic(recording)}
                        title={recording.isPublic ? 'Make Private' : 'Make Public'}
                      >
                        {#if recording.isPublic}
                          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                            <line x1="2" y1="12" x2="22" y2="12"/>
                            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
                          </svg>
                        {:else}
                          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                          </svg>
                        {/if}
                      </button>
                      {#if recording.isPublic}
                        <button 
                          class="action-btn"
                          on:click={() => copyShareLink(recording)}
                          title="Copy Share Link"
                        >
                          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                          </svg>
                        </button>
                      {/if}
                      <button 
                        class="action-btn delete"
                        on:click={() => deleteRecording(recording)}
                        title="Delete"
                      >
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <polyline points="3 6 5 6 21 6"/>
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                        </svg>
                      </button>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      {/if}
    </div>
  </div>
{/if}

<style>
  .recording-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
  }

  .recording-panel {
    background: #1a1a2e;
    border: 1px solid #2a2a4e;
    border-radius: 12px;
    width: 480px;
    max-width: 90vw;
    max-height: 80vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;
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

  .tabs {
    display: flex;
    border-bottom: 1px solid #2a2a4e;
  }

  .tab {
    flex: 1;
    padding: 0.875rem;
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    font-size: 0.9rem;
    transition: all 0.2s;
    position: relative;
  }

  .tab:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.05);
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
    height: 2px;
    background: #00ff88;
  }

  .record-section, .library-section {
    padding: 1.25rem;
    overflow-y: auto;
    flex: 1;
  }

  .description {
    color: #888;
    margin: 0 0 1.25rem;
    font-size: 0.9rem;
  }

  .input-group {
    margin-bottom: 1.25rem;
  }

  .input-group label {
    display: block;
    color: #aaa;
    font-size: 0.85rem;
    margin-bottom: 0.5rem;
  }

  .input-group input {
    width: 100%;
    padding: 0.75rem;
    background: #0a0a1a;
    border: 1px solid #2a2a4e;
    border-radius: 8px;
    color: #fff;
    font-size: 0.9rem;
  }

  .input-group input:focus {
    outline: none;
    border-color: #00ff88;
  }

  .start-btn {
    width: 100%;
    padding: 0.875rem;
    background: #3a1a1a;
    border: 1px solid #5a2a2a;
    border-radius: 8px;
    color: #ff6b6b;
    font-weight: 600;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    transition: all 0.2s;
  }

  .start-btn:hover:not(:disabled) {
    background: #4a1a1a;
    border-color: #ff6b6b;
  }

  .start-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .rec-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: #ff6b6b;
    animation: blink 1s ease-in-out infinite;
  }

  @keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.3; }
  }

  .active-recording {
    text-align: center;
  }

  .recording-indicator {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    color: #ff6b6b;
    font-size: 1.25rem;
    font-weight: 600;
    margin-bottom: 1.5rem;
  }

  .recording-stats {
    display: flex;
    justify-content: center;
    gap: 2rem;
    margin-bottom: 1.5rem;
  }

  .stat {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .stat .label {
    color: #666;
    font-size: 0.75rem;
    text-transform: uppercase;
    margin-bottom: 0.25rem;
  }

  .stat .value {
    color: #fff;
    font-size: 1.5rem;
    font-weight: 600;
    font-family: 'JetBrains Mono', monospace;
  }

  .stop-btn {
    padding: 0.875rem 2rem;
    background: #ff6b6b;
    border: none;
    border-radius: 8px;
    color: #fff;
    font-weight: 600;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    transition: all 0.2s;
  }

  .stop-btn:hover {
    background: #ff5555;
    transform: scale(1.02);
  }

  .stop-icon {
    font-size: 0.75rem;
  }

  .no-terminal {
    color: #666;
    text-align: center;
    padding: 2rem;
  }

  .loading, .empty-library {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem 1rem;
    color: #666;
    text-align: center;
  }

  .empty-library svg {
    margin-bottom: 1rem;
    opacity: 0.5;
  }

  .empty-library p {
    color: #888;
    margin: 0 0 0.25rem;
  }

  .empty-library span {
    font-size: 0.85rem;
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .recordings-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .recording-item {
    display: flex;
    align-items: center;
    padding: 0.75rem;
    background: #0a0a1a;
    border-radius: 8px;
    transition: background 0.2s;
  }

  .recording-item:hover {
    background: #151525;
  }

  .recording-info {
    flex: 1;
    cursor: pointer;
  }

  .recording-info .title {
    display: block;
    color: #fff;
    font-size: 0.9rem;
    margin-bottom: 0.25rem;
  }

  .recording-info .meta {
    display: flex;
    gap: 0.5rem;
    color: #666;
    font-size: 0.75rem;
  }

  .recording-actions {
    display: flex;
    gap: 0.25rem;
  }

  .action-btn {
    padding: 0.5rem;
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    border-radius: 4px;
    transition: all 0.2s;
  }

  .action-btn:hover {
    background: #252542;
    color: #fff;
  }

  .action-btn.active {
    color: #00ff88;
  }

  .action-btn.delete:hover {
    color: #ff6b6b;
  }

  .player-section {
    padding: 1.25rem;
  }

  .player-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .back-btn {
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .back-btn:hover {
    color: #fff;
  }

  .player-header h4 {
    margin: 0;
    color: #fff;
  }

  .player-container {
    background: #0a0a1a;
    border-radius: 8px;
    overflow: hidden;
    margin-bottom: 1rem;
  }

  .player-placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem;
    color: #666;
  }

  .player-placeholder svg {
    margin-bottom: 1rem;
    opacity: 0.5;
  }

  .player-placeholder p {
    margin: 0 0 0.25rem;
    color: #888;
  }

  .player-placeholder .duration {
    font-size: 0.85rem;
  }

  .player-controls {
    display: flex;
    justify-content: center;
  }

  .download-btn {
    padding: 0.625rem 1.25rem;
    background: #252542;
    border: 1px solid #3a3a5e;
    border-radius: 6px;
    color: #888;
    text-decoration: none;
    font-size: 0.85rem;
    transition: all 0.2s;
  }

  .download-btn:hover {
    background: #2a2a4e;
    color: #fff;
  }
</style>
