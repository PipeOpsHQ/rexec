<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { recordings, type Recording } from '../stores/recordings';
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
      const response = await fetch(`/api/recordings/${recording.id}/stream`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
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
          <button class="back-btn" on:click={() => { stopPlayback(); selectedRecording = null; }}>←</button>
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
              <button class="ctrl-btn" on:click={togglePause} title={isPaused ? 'Resume' : 'Pause'}>
                {isPaused ? '▶' : '⏸'}
              </button>
              <button class="ctrl-btn" on:click={stopPlayback} title="Stop">■</button>
            {:else}
              <button class="ctrl-btn play" on:click={restartPlayback} title="Play">▶</button>
            {/if}
          </div>
          <button class="ctrl-btn download" on:click={() => downloadRecording(selectedRecording)} title="Download">
            ↓
          </button>
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
    width: 340px;
    background: var(--bg-card, #0a0a0f);
    border: 1px solid var(--border, #1a1a2a);
    border-radius: 8px;
    z-index: 100;
    font-size: 12px;
    font-family: var(--font-mono, 'JetBrains Mono', monospace);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.6), 0 0 1px rgba(255, 68, 68, 0.3);
    overflow: hidden;
  }

  .recording-panel.compact {
    width: 300px;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    background: linear-gradient(180deg, rgba(255, 68, 68, 0.08) 0%, transparent 100%);
    border-bottom: 1px solid rgba(255, 68, 68, 0.2);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .rec-icon {
    color: #ff4444;
    font-size: 12px;
    animation: pulse-glow 2s ease-in-out infinite;
  }

  @keyframes pulse-glow {
    0%, 100% { filter: drop-shadow(0 0 2px #ff4444); }
    50% { filter: drop-shadow(0 0 6px #ff4444); }
  }

  .title {
    color: #ff6666;
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
    color: #ff4444;
    border-color: rgba(255, 68, 68, 0.3);
    background: rgba(255, 68, 68, 0.1);
  }

  .tabs {
    display: flex;
    background: rgba(0, 0, 0, 0.3);
    padding: 4px;
    gap: 4px;
  }

  .tab {
    flex: 1;
    padding: 8px 12px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 4px;
    color: #666;
    font-size: 10px;
    font-weight: 500;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 1px;
    transition: all 0.15s;
  }

  .tab:hover {
    color: #888;
    background: rgba(255, 255, 255, 0.03);
  }

  .tab.active {
    color: #ff6666;
    background: rgba(255, 68, 68, 0.1);
    border-color: rgba(255, 68, 68, 0.3);
  }

  .count {
    opacity: 0.6;
    margin-left: 4px;
    font-size: 9px;
  }

  .panel-content {
    padding: 14px;
    max-height: 320px;
    overflow-y: auto;
  }

  .active-recording {
    display: flex;
    flex-direction: column;
    gap: 16px;
    text-align: center;
    padding: 20px 0;
  }

  .rec-status {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    color: #ff4444;
    font-size: 14px;
    font-weight: 500;
  }

  .elapsed {
    color: #888;
    font-family: var(--font-mono, monospace);
    padding: 4px 10px;
    background: rgba(255, 68, 68, 0.1);
    border-radius: 4px;
    font-size: 13px;
  }

  .rec-dot {
    width: 10px;
    height: 10px;
    background: #ff4444;
    border-radius: 50%;
    box-shadow: 0 0 8px #ff4444;
  }

  .rec-dot.blink {
    animation: blink 1s ease-in-out infinite;
  }

  @keyframes blink {
    0%, 100% { opacity: 1; box-shadow: 0 0 12px #ff4444; }
    50% { opacity: 0.4; box-shadow: 0 0 4px #ff4444; }
  }

  .stop-btn {
    padding: 12px 24px;
    background: linear-gradient(135deg, #ff4444, #cc3333);
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 1.5px;
    transition: all 0.2s;
    box-shadow: 0 4px 12px rgba(255, 68, 68, 0.3);
  }

  .stop-btn:hover {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(255, 68, 68, 0.4);
  }

  .start-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .title-input {
    width: 100%;
    padding: 10px 12px;
    background: rgba(0, 0, 0, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: #fff;
    font-size: 12px;
    font-family: inherit;
    transition: all 0.15s;
  }

  .title-input::placeholder {
    color: #555;
  }

  .title-input:focus {
    outline: none;
    border-color: rgba(255, 68, 68, 0.5);
    box-shadow: 0 0 0 2px rgba(255, 68, 68, 0.1);
  }

  .start-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    padding: 12px;
    background: rgba(255, 68, 68, 0.1);
    border: 1px solid rgba(255, 68, 68, 0.3);
    border-radius: 6px;
    color: #ff6666;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    text-transform: uppercase;
    letter-spacing: 1px;
    transition: all 0.2s;
  }

  .start-btn:hover:not(:disabled) {
    background: rgba(255, 68, 68, 0.15);
    border-color: #ff4444;
    box-shadow: 0 0 16px rgba(255, 68, 68, 0.2);
  }

  .start-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .hint {
    color: #555;
    text-align: center;
    padding: 24px;
    margin: 0;
    font-size: 11px;
  }

  .library-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .loading, .empty {
    color: #555;
    text-align: center;
    padding: 24px;
    margin: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font-size: 11px;
  }

  .recording-item {
    display: flex;
    align-items: center;
    padding: 10px 12px;
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 6px;
    transition: all 0.15s;
  }

  .recording-item:hover {
    border-color: rgba(255, 68, 68, 0.3);
    background: rgba(255, 68, 68, 0.05);
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
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-bottom: 2px;
  }

  .rec-meta {
    color: #666;
    font-size: 10px;
  }

  .rec-actions {
    display: flex;
    gap: 4px;
  }

  .icon-btn {
    background: transparent;
    border: 1px solid transparent;
    border-radius: 4px;
    padding: 4px 6px;
    cursor: pointer;
    font-size: 12px;
    opacity: 0.5;
    transition: all 0.15s;
  }

  .icon-btn:hover {
    opacity: 1;
    background: rgba(255, 255, 255, 0.05);
  }

  .icon-btn.active {
    opacity: 1;
  }

  .icon-btn.delete:hover {
    color: #ff4444;
    border-color: rgba(255, 68, 68, 0.3);
    background: rgba(255, 68, 68, 0.1);
  }

  .player-section {
    padding: 0;
  }

  .player-bar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    background: rgba(0, 0, 0, 0.4);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  .back-btn {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 4px;
    color: #888;
    cursor: pointer;
    font-size: 14px;
    padding: 4px 8px;
    transition: all 0.15s;
  }

  .back-btn:hover {
    color: #fff;
    border-color: rgba(255, 255, 255, 0.2);
  }

  .player-title {
    color: #ddd;
    font-size: 12px;
    font-weight: 500;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .player-duration {
    color: #666;
    font-size: 10px;
    padding: 2px 8px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 4px;
  }

  .player-container {
    background: #050508;
    height: 180px;
    overflow: hidden;
    border-top: 1px solid rgba(255, 68, 68, 0.1);
    border-bottom: 1px solid rgba(255, 68, 68, 0.1);
  }

  .player-container :global(.xterm) {
    height: 100%;
    padding: 8px;
  }

  .player-controls {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    background: rgba(0, 0, 0, 0.4);
  }

  .progress-bar {
    flex: 1;
    height: 4px;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 2px;
    overflow: hidden;
    cursor: pointer;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #ff4444, #ff6666);
    border-radius: 2px;
    transition: width 0.1s ease;
    box-shadow: 0 0 8px rgba(255, 68, 68, 0.4);
  }

  .control-buttons {
    display: flex;
    gap: 6px;
  }

  .ctrl-btn {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 4px;
    color: #888;
    width: 32px;
    height: 32px;
    cursor: pointer;
    font-size: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
  }

  .ctrl-btn:hover {
    color: #fff;
    border-color: rgba(255, 68, 68, 0.5);
    background: rgba(255, 68, 68, 0.1);
  }

  .ctrl-btn.play {
    color: #ff6666;
    border-color: rgba(255, 68, 68, 0.3);
  }

  .ctrl-btn.download {
    color: #666;
    font-size: 14px;
  }

  .ctrl-btn.download:hover {
    color: var(--accent, #00ff88);
    border-color: rgba(0, 255, 136, 0.3);
    background: rgba(0, 255, 136, 0.1);
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
  .panel-content::-webkit-scrollbar {
    width: 4px;
  }

  .panel-content::-webkit-scrollbar-track {
    background: transparent;
  }

  .panel-content::-webkit-scrollbar-thumb {
    background: rgba(255, 68, 68, 0.3);
    border-radius: 2px;
  }

  .panel-content::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 68, 68, 0.5);
  }

  /* Firefox scrollbar */
  .panel-content {
    scrollbar-width: thin;
    scrollbar-color: rgba(255, 68, 68, 0.3) transparent;
  }
</style>
