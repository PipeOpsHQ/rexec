<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { recordings, type Recording } from '../stores/recordings';
  import { token } from '../stores/auth';
  import { slide } from 'svelte/transition';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';

  export let containerId: string = '';
  export let isOpen = false;
  export let compact = false;

  const dispatch = createEventDispatcher();

  let recordingTitle = '';
  let isStarting = false;
  let currentTab: 'record' | 'library' = 'record';
  let selectedRecording: Recording | null = null;
  let playerElement: HTMLDivElement;
  let playerTerminal: Terminal | null = null;
  let fitAddon: FitAddon | null = null;
  let isPlaying = false;
  let isPaused = false;
  let playbackProgress = 0;
  let playbackTimer: ReturnType<typeof setTimeout> | null = null;
  let recordingEvents: Array<[number, string, string]> = [];
  let currentEventIndex = 0;

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

  onDestroy(() => {
    stopPlayback();
    if (playerTerminal) {
      playerTerminal.dispose();
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

  async function downloadRecording(recording: Recording) {
    try {
      const authToken = get(token);
      if (!authToken) {
        console.error('Not authenticated');
        return;
      }
      const response = await fetch(`/api/recordings/${recording.id}/stream`, {
        headers: {
          'Authorization': `Bearer ${authToken}`
        }
      });
      
      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${recording.title || 'recording'}.cast`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        a.remove();
      }
    } catch (err) {
      console.error('Download failed:', err);
    }
  }

  async function playRecording(recording: Recording) {
    selectedRecording = recording;
    isPlaying = false;
    isPaused = false;
    playbackProgress = 0;
    currentEventIndex = 0;
    recordingEvents = [];
    
    // Wait for DOM to update
    await new Promise(r => setTimeout(r, 50));
    
    // Initialize player terminal
    if (playerElement && !playerTerminal) {
      playerTerminal = new Terminal({
        theme: {
          background: '#0a0a14',
          foreground: '#e0e0e0',
          cursor: '#00ff88',
          black: '#1a1a2e',
          red: '#ff6b6b',
          green: '#00ff88',
          yellow: '#ffd93d',
          blue: '#6c5ce7',
          magenta: '#a29bfe',
          cyan: '#00d4ff',
          white: '#e0e0e0',
        },
        fontSize: 10,
        fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
        cursorStyle: 'block',
        cursorBlink: false,
        scrollback: 1000,
      });
      
      fitAddon = new FitAddon();
      playerTerminal.loadAddon(fitAddon);
      playerTerminal.open(playerElement);
      fitAddon.fit();
    }
    
    if (playerTerminal) {
      playerTerminal.clear();
      playerTerminal.reset();
    }
    
    // Fetch recording data
    try {
      const response = await fetch(`/api/recordings/${recording.id}/stream`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (response.ok) {
        const text = await response.text();
        const lines = text.trim().split('\n');
        
        // Parse asciicast format
        for (let i = 1; i < lines.length; i++) {
          try {
            const event = JSON.parse(lines[i]);
            if (Array.isArray(event) && event.length >= 3) {
              recordingEvents.push(event as [number, string, string]);
            }
          } catch (e) {
            // Skip malformed lines
          }
        }
        
        if (recordingEvents.length > 0) {
          startPlayback();
        }
      }
    } catch (e) {
      console.error('Failed to load recording:', e);
    }
  }

  function startPlayback() {
    if (recordingEvents.length === 0) return;
    
    isPlaying = true;
    isPaused = false;
    playNextEvent();
  }

  function playNextEvent() {
    if (isPaused || currentEventIndex >= recordingEvents.length) {
      if (currentEventIndex >= recordingEvents.length) {
        isPlaying = false;
        playbackProgress = 100;
      }
      return;
    }
    
    const event = recordingEvents[currentEventIndex];
    const [time, type, data] = event;
    
    // Write output to terminal
    if (type === 'o' && playerTerminal) {
      playerTerminal.write(data);
    }
    
    // Update progress
    const totalDuration = recordingEvents[recordingEvents.length - 1][0];
    playbackProgress = (time / totalDuration) * 100;
    
    currentEventIndex++;
    
    // Schedule next event
    if (currentEventIndex < recordingEvents.length) {
      const nextTime = recordingEvents[currentEventIndex][0];
      const delay = Math.max(10, (nextTime - time) * 1000);
      playbackTimer = setTimeout(playNextEvent, delay);
    } else {
      isPlaying = false;
      playbackProgress = 100;
    }
  }

  function togglePause() {
    if (isPaused) {
      isPaused = false;
      playNextEvent();
    } else {
      isPaused = true;
      if (playbackTimer) {
        clearTimeout(playbackTimer);
      }
    }
  }

  function stopPlayback() {
    if (playbackTimer) {
      clearTimeout(playbackTimer);
      playbackTimer = null;
    }
    isPlaying = false;
    isPaused = false;
    currentEventIndex = 0;
    playbackProgress = 0;
    if (playerTerminal) {
      playerTerminal.clear();
      playerTerminal.reset();
    }
  }

  function restartPlayback() {
    stopPlayback();
    if (playerTerminal) {
      playerTerminal.clear();
      playerTerminal.reset();
    }
    currentEventIndex = 0;
    playbackProgress = 0;
    startPlayback();
  }

  function close() {
    stopPlayback();
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
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="panel-overlay" onclick={(e) => { if (e.target === e.currentTarget) close(); }}>
    <div class="recording-panel" class:compact transition:slide={{ duration: 200 }}>
    <div class="panel-header">
      <div class="header-left">
        <span class="rec-icon">⏺</span>
        <span class="title">REC</span>
      </div>
      <button class="close-btn" onclick={close}>×</button>
    </div>

    {#if selectedRecording}
      <div class="player-section">
        <div class="player-bar">
          <button class="back-btn" onclick={() => { stopPlayback(); selectedRecording = null; }}>←</button>
          <span class="player-title">{selectedRecording.title || 'Recording'}</span>
          <span class="player-duration">{selectedRecording.duration}</span>
        </div>
        <div class="player-container" bind:this={playerElement}></div>
        <div class="player-controls">
          <div class="progress-bar">
            <div class="progress-fill" style="width: {playbackProgress}%"></div>
          </div>
          <div class="control-buttons">
            {#if isPlaying}
              <button class="ctrl-btn" onclick={togglePause} title={isPaused ? 'Resume' : 'Pause'}>
                {isPaused ? '▶' : '⏸'}
              </button>
              <button class="ctrl-btn" onclick={stopPlayback} title="Stop">■</button>
            {:else}
              <button class="ctrl-btn play" onclick={restartPlayback} title="Play">▶</button>
            {/if}
          </div>
          <button class="ctrl-btn download" onclick={() => downloadRecording(selectedRecording)} title="Download">
            ↓
          </button>
        </div>
      </div>
    {:else}
      <div class="tabs">
        <button class="tab" class:active={currentTab === 'record'} onclick={() => currentTab = 'record'}>
          Record
        </button>
        <button class="tab" class:active={currentTab === 'library'} onclick={() => currentTab = 'library'}>
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
              <button class="stop-btn" onclick={stopRecording}>■ Stop</button>
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
                <button class="start-btn" onclick={startRecording} disabled={isStarting}>
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
                  <div class="rec-info" onclick={() => playRecording(recording)}>
                    <span class="rec-title">{recording.title || 'Untitled'}</span>
                    <span class="rec-meta">{recording.duration} • {recordings.formatSize(recording.sizeBytes)}</span>
                  </div>
                  <div class="rec-actions">
                    <button 
                      class="icon-btn" 
                      class:active={recording.isPublic}
                      onclick={() => togglePublic(recording)}
                      title={recording.isPublic ? 'Public' : 'Private'}
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        {#if recording.isPublic}
                          <circle cx="12" cy="12" r="10"/>
                          <line x1="2" y1="12" x2="22" y2="12"/>
                          <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
                        {:else}
                          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                        {/if}
                      </svg>
                    </button>
                    {#if recording.isPublic}
                      <button class="icon-btn" onclick={() => copyShareLink(recording)} title="Copy link">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                        </svg>
                      </button>
                    {/if}
                    <button class="icon-btn delete" onclick={() => deleteRecording(recording)} title="Delete">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                      </svg>
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
  </div>
{/if}

<style>
  .panel-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10001;
  }

  .recording-panel {
    width: 360px;
    max-width: 95vw;
    max-height: 85vh;
    background: #0c0c10;
    border: 1px solid #1e1e28;
    border-radius: 8px;
    font-size: 11px;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    box-shadow: 0 8px 40px rgba(0, 0, 0, 0.9), 0 0 20px rgba(255, 71, 87, 0.1);
    overflow: hidden;
  }

  .recording-panel.compact {
    width: 260px;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background: #0f0f14;
    border-bottom: 1px solid #1e1e28;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .rec-icon {
    color: #ff4757;
    font-size: 10px;
  }

  @keyframes pulse-glow {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }

  .title {
    color: #ff4757;
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 1.5px;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: #999;
    font-size: 16px;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 3px;
    transition: all 0.15s;
    line-height: 1;
  }

  .close-btn:hover {
    color: #ff4757;
    background: rgba(255, 71, 87, 0.1);
  }

  .tabs {
    display: flex;
    background: #080809;
    padding: 3px;
    gap: 2px;
  }

  .tab {
    flex: 1;
    padding: 6px 10px;
    background: transparent;
    border: none;
    border-radius: 3px;
    color: #999;
    font-size: 9px;
    font-weight: 600;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    transition: all 0.15s;
  }

  .tab:hover {
    color: #888;
    background: rgba(255, 255, 255, 0.02);
  }

  .tab.active {
    color: #ff4757;
    background: rgba(255, 71, 87, 0.1);
  }

  .count {
    opacity: 0.6;
    margin-left: 3px;
    font-size: 8px;
  }

  .panel-content {
    padding: 12px;
    max-height: 280px;
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
    color: #ff4757;
    font-size: 12px;
    font-weight: 500;
  }

  .elapsed {
    color: #888;
    font-family: var(--font-mono, monospace);
    padding: 3px 8px;
    background: rgba(255, 71, 87, 0.08);
    border-radius: 3px;
    font-size: 11px;
  }

  .rec-dot {
    width: 8px;
    height: 8px;
    background: #ff4757;
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
    background: #ff4757;
    border: none;
    border-radius: 4px;
    color: #fff;
    font-size: 10px;
    font-weight: 600;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 1px;
    transition: all 0.2s;
  }

  .stop-btn:hover {
    background: #ff6b7a;
  }

  .start-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .title-input {
    width: 100%;
    padding: 8px 10px;
    background: #0a0a0c;
    border: 1px solid #1e1e28;
    border-radius: 4px;
    color: #fff;
    font-size: 11px;
    font-family: inherit;
    transition: all 0.15s;
  }

  .title-input::placeholder {
    color: #999;
  }

  .title-input:focus {
    outline: none;
    border-color: #ff4757;
  }

  .start-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 10px;
    background: rgba(255, 71, 87, 0.1);
    border: 1px solid rgba(255, 71, 87, 0.25);
    border-radius: 4px;
    color: #ff4757;
    font-size: 10px;
    font-weight: 600;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    transition: all 0.2s;
  }

  .start-btn:hover:not(:disabled) {
    background: rgba(255, 71, 87, 0.15);
    border-color: #ff4757;
  }

  .start-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .hint {
    color: #999;
    text-align: center;
    padding: 20px;
    margin: 0;
    font-size: 10px;
  }

  .library-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .loading, .empty {
    color: #999;
    text-align: center;
    padding: 20px;
    margin: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font-size: 10px;
  }

  .recording-item {
    display: flex;
    align-items: center;
    padding: 8px 10px;
    background: #0a0a0c;
    border: 1px solid #1a1a22;
    border-radius: 4px;
    transition: all 0.15s;
  }

  .recording-item:hover {
    border-color: rgba(255, 71, 87, 0.3);
  }

  .rec-info {
    flex: 1;
    cursor: pointer;
    overflow: hidden;
  }

  .rec-title {
    display: block;
    color: #ccc;
    font-size: 11px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-bottom: 2px;
  }

  .rec-meta {
    color: #999;
    font-size: 9px;
  }

  .rec-actions {
    display: flex;
    gap: 2px;
  }

  .icon-btn {
    background: transparent;
    border: none;
    border-radius: 3px;
    padding: 4px 5px;
    cursor: pointer;
    font-size: 11px;
    color: #888;
    opacity: 0.7;
    transition: all 0.15s;
  }

  .icon-btn:hover {
    opacity: 1;
    color: #fff;
    background: rgba(255, 255, 255, 0.05);
  }

  .icon-btn.active {
    opacity: 1;
    color: #fff;
  }

  .icon-btn.delete:hover {
    color: #ff4757;
    background: rgba(255, 71, 87, 0.1);
  }

  .player-section {
    padding: 0;
  }

  .player-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    background: #080809;
    border-bottom: 1px solid #1a1a22;
  }

  .back-btn {
    background: transparent;
    border: 1px solid #1e1e28;
    border-radius: 3px;
    color: #a0a0a0;
    cursor: pointer;
    font-size: 12px;
    padding: 3px 6px;
    transition: all 0.15s;
  }

  .back-btn:hover {
    color: #fff;
    border-color: #333;
  }

  .player-title {
    color: #ccc;
    font-size: 11px;
    font-weight: 500;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .player-duration {
    color: #999;
    font-size: 9px;
    padding: 2px 6px;
    background: #0a0a0c;
    border-radius: 3px;
  }

  .player-container {
    background: #050506;
    height: 200px;
    min-height: 160px;
    overflow: hidden;
    border: 1px solid #1a1a22;
    margin: 0 12px;
    border-radius: 4px;
  }

  .player-container :global(.xterm) {
    height: 100%;
    padding: 8px;
  }
  
  .player-container :global(.xterm-viewport) {
    overflow-y: auto !important;
  }

  .player-controls {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    background: #0a0a0c;
    border-top: 1px solid #1a1a22;
    margin: 0 12px 12px;
    border-radius: 0 0 4px 4px;
  }

  .progress-bar {
    flex: 1;
    height: 6px;
    background: #1a1a22;
    border-radius: 3px;
    overflow: hidden;
    cursor: pointer;
    position: relative;
  }
  
  .progress-bar:hover {
    height: 8px;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #ff4757, #ff6b7a);
    border-radius: 3px;
    transition: width 0.1s ease;
    box-shadow: 0 0 8px rgba(255, 71, 87, 0.4);
  }

  .control-buttons {
    display: flex;
    gap: 4px;
  }

  .ctrl-btn {
    background: transparent;
    border: 1px solid #1e1e28;
    border-radius: 3px;
    color: #a0a0a0;
    width: 28px;
    height: 28px;
    cursor: pointer;
    font-size: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
  }

  .ctrl-btn:hover {
    color: #fff;
    border-color: #333;
  }

  .ctrl-btn.play {
    color: #ff4757;
    border-color: rgba(255, 71, 87, 0.3);
  }

  .ctrl-btn.download {
    color: #999;
    font-size: 12px;
  }

  .ctrl-btn.download:hover {
    color: var(--accent, #00ff88);
    border-color: rgba(0, 255, 136, 0.3);
  }

  .spinner-sm {
    width: 12px;
    height: 12px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Scrollbar */
  .panel-content::-webkit-scrollbar {
    width: 3px;
  }

  .panel-content::-webkit-scrollbar-track {
    background: transparent;
  }

  .panel-content::-webkit-scrollbar-thumb {
    background: #222;
    border-radius: 2px;
  }

  .panel-content::-webkit-scrollbar-thumb:hover {
    background: #333;
  }

  /* Firefox scrollbar */
  .panel-content {
    scrollbar-width: thin;
    scrollbar-color: #222 transparent;
  }
</style>
