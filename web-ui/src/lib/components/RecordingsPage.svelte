<script lang="ts">
    import { onMount, onDestroy, tick } from 'svelte';
    import { get } from 'svelte/store';
    import { recordings, type Recording } from '../stores/recordings';
    import { token } from '../stores/auth';
    import type { Terminal } from '@xterm/xterm';
    import type { FitAddon } from '@xterm/addon-fit';
    import { loadXtermCore } from '$utils/xterm';
    import StatusIcon from './icons/StatusIcon.svelte';

    let recordingsList: Recording[] = [];
    let isLoading = true;
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
    let searchQuery = '';

    $: filteredRecordings = recordingsList.filter(r => 
        !searchQuery || 
        (r.title?.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (r.containerName?.toLowerCase().includes(searchQuery.toLowerCase()))
    );

    onMount(async () => {
        await recordings.fetchRecordings();
        recordingsList = $recordings.recordings;
        isLoading = false;
    });

    onDestroy(() => {
        stopPlayback();
        if (playerTerminal) {
            playerTerminal.dispose();
        }
    });

    async function playRecording(recording: Recording) {
        selectedRecording = recording;
        isPlaying = false;
        isPaused = false;
        playbackProgress = 0;
        currentEventIndex = 0;
        recordingEvents = [];
        
        // Wait for Svelte to update DOM and render playerElement
        await tick();
        await new Promise(r => setTimeout(r, 50));
        
        if (!playerElement) {
            console.error('Player element not available');
            return;
        }
        
        if (!playerTerminal) {
            let XtermTerminal: (typeof import("@xterm/xterm"))["Terminal"];
            let XtermFitAddon: (typeof import("@xterm/addon-fit"))["FitAddon"];
            try {
                ({ Terminal: XtermTerminal, FitAddon: XtermFitAddon } = await loadXtermCore());
            } catch (e) {
                console.error("[Recording] Failed to load xterm:", e);
                return;
            }

            playerTerminal = new XtermTerminal({
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
                fontSize: 13,
                fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
                cursorStyle: 'block',
                cursorBlink: false,
                scrollback: 5000,
            });
            
            fitAddon = new XtermFitAddon();
            playerTerminal.loadAddon(fitAddon);
            playerTerminal.open(playerElement);
            fitAddon.fit();
        }
        
        if (playerTerminal && playerTerminal.element) {
            try {
                playerTerminal.clear();
                playerTerminal.reset();
            } catch (e) {
                // Ignore errors
            }
        }
        
        try {
            const authToken = get(token);
            const response = await fetch(`/api/recordings/${recording.id}/stream`, {
                headers: {
                    'Authorization': `Bearer ${authToken || localStorage.getItem('token')}`
                }
            });
            
            if (response.ok) {
                const text = await response.text();
                console.log('[Recording] Fetched stream data, length:', text.length);
                const lines = text.trim().split('\n');
                
                for (let i = 1; i < lines.length; i++) {
                    try {
                        const event = JSON.parse(lines[i]);
                        if (Array.isArray(event) && event.length >= 3) {
                            recordingEvents.push(event as [number, string, string]);
                        }
                    } catch (e) {
                        // Skip unparseable lines
                    }
                }
                
                console.log('[Recording] Parsed events:', recordingEvents.length);
                if (recordingEvents.length > 0) {
                    startPlayback();
                } else {
                    console.error('[Recording] No events parsed from recording');
                }
            } else {
                console.error('[Recording] Failed to fetch stream:', response.status, await response.text());
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
        
        if (type === 'o' && playerTerminal) {
            playerTerminal.write(data);
        }
        
        const totalDuration = recordingEvents[recordingEvents.length - 1][0];
        playbackProgress = (time / totalDuration) * 100;
        
        currentEventIndex++;
        
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
        if (playerTerminal && playerTerminal.element) {
            try {
                playerTerminal.clear();
                playerTerminal.reset();
            } catch (e) {
                // Ignore errors
            }
        }
    }

    function restartPlayback() {
        stopPlayback();
        if (playerTerminal && playerTerminal.element) {
            try {
                playerTerminal.clear();
                playerTerminal.reset();
            } catch (e) {
                // Ignore errors
            }
        }
        currentEventIndex = 0;
        playbackProgress = 0;
        startPlayback();
    }

    function closePlayer() {
        stopPlayback();
        selectedRecording = null;
    }

    async function togglePublic(recording: Recording) {
        await recordings.updateRecording(recording.id, { isPublic: !recording.isPublic });
        recordingsList = $recordings.recordings;
    }

    async function deleteRecording(recording: Recording) {
        if (confirm('Delete this recording permanently?')) {
            await recordings.deleteRecording(recording.id);
            recordingsList = $recordings.recordings;
            if (selectedRecording?.id === recording.id) {
                closePlayer();
            }
        }
    }

    function copyShareLink(recording: Recording) {
        const url = `${window.location.origin}${recording.shareUrl}`;
        navigator.clipboard.writeText(url);
    }

    async function downloadRecording(recording: Recording) {
        try {
            const authToken = get(token);
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

    function formatDate(dateStr: string): string {
        return new Date(dateStr).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    function formatDuration(duration: string): string {
        return duration || '0:00';
    }
</script>

<div class="recordings-page">
    <div class="page-header">
        <div class="header-content">
            <h1>
                <StatusIcon status="recording" size={28} />
                Recordings
            </h1>
            <p class="subtitle">View and manage your terminal session recordings</p>
        </div>
        <div class="search-box">
            <StatusIcon status="search" size={16} />
            <input 
                type="text" 
                placeholder="Search recordings..." 
                bind:value={searchQuery}
            />
        </div>
    </div>

    {#if selectedRecording}
        <div class="player-section">
            <div class="player-header">
                <button class="back-btn" onclick={closePlayer}>
                    <StatusIcon status="back" size={16} />
                    Back to list
                </button>
                <div class="player-info">
                    <h2>{selectedRecording.title || 'Untitled Recording'}</h2>
                    <span class="meta">{formatDate(selectedRecording.createdAt)} · {selectedRecording.duration}</span>
                </div>
                <div class="player-actions">
                    <button class="action-btn" onclick={() => downloadRecording(selectedRecording)} title="Download">
                        <StatusIcon status="download" size={16} />
                    </button>
                    {#if selectedRecording.isPublic}
                        <button class="action-btn" onclick={() => copyShareLink(selectedRecording)} title="Copy share link">
                            <StatusIcon status="link" size={16} />
                        </button>
                    {/if}
                </div>
            </div>
            <div class="player-container" bind:this={playerElement}></div>
            <div class="player-controls">
                <div class="progress-bar">
                    <div class="progress-fill" style="width: {playbackProgress}%"></div>
                </div>
                <div class="control-buttons">
                    {#if isPlaying}
                        <button class="ctrl-btn" onclick={togglePause} title={isPaused ? 'Resume' : 'Pause'}>
                            <StatusIcon status={isPaused ? 'play' : 'validating'} size={18} />
                        </button>
                        <button class="ctrl-btn" onclick={stopPlayback} title="Stop">
                            <StatusIcon status="close" size={18} />
                        </button>
                    {:else}
                        <button class="ctrl-btn primary" onclick={restartPlayback} title="Play">
                            <StatusIcon status="play" size={18} />
                        </button>
                    {/if}
                </div>
            </div>
        </div>
    {:else}
        <div class="recordings-grid">
            {#if isLoading}
                <div class="loading">
                    <div class="spinner"></div>
                    <span>Loading recordings...</span>
                </div>
            {:else if filteredRecordings.length === 0}
                <div class="empty-state">
                    <StatusIcon status="recording" size={48} />
                    <h3>{searchQuery ? 'No matching recordings' : 'No recordings yet'}</h3>
                    <p>{searchQuery ? 'Try a different search term' : 'Start a recording from any terminal session'}</p>
                </div>
            {:else}
                {#each filteredRecordings as recording}
                    <div class="recording-card">
                        <div class="card-header" onclick={() => playRecording(recording)}>
                            <div class="rec-icon">
                                <StatusIcon status="recording" size={20} />
                            </div>
                            <div class="rec-info">
                                <h3>{recording.title || 'Untitled'}</h3>
                                <span class="meta">
                                    {formatDate(recording.createdAt)} · {recording.duration} · {recordings.formatSize(recording.sizeBytes)}
                                </span>
                            </div>
                        </div>
                        <div class="card-actions">
                            <button 
                                class="icon-btn" 
                                class:active={recording.isPublic}
                                onclick={(e) => { e.stopPropagation(); togglePublic(recording); }}
                                title={recording.isPublic ? 'Make private' : 'Make public'}
                            >
                                <StatusIcon status={recording.isPublic ? 'globe' : 'lock'} size={14} />
                            </button>
                            {#if recording.isPublic}
                                <button class="icon-btn" onclick={(e) => { e.stopPropagation(); copyShareLink(recording); }} title="Copy link">
                                    <StatusIcon status="link" size={14} />
                                </button>
                            {/if}
                            <button class="icon-btn" onclick={(e) => { e.stopPropagation(); downloadRecording(recording); }} title="Download">
                                <StatusIcon status="download" size={14} />
                            </button>
                            <button class="icon-btn delete" onclick={(e) => { e.stopPropagation(); deleteRecording(recording); }} title="Delete">
                                <StatusIcon status="trash" size={14} />
                            </button>
                        </div>
                    </div>
                {/each}
            {/if}
        </div>
    {/if}
</div>

<style>
    .recordings-page {
        max-width: 1200px;
        margin: 0 auto;
        padding: 40px 20px;
    }

    .page-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 20px;
        margin-bottom: 32px;
        flex-wrap: wrap;
    }

    .header-content h1 {
        display: flex;
        align-items: center;
        gap: 12px;
        font-size: 28px;
        font-weight: 700;
        margin: 0 0 8px 0;
        color: var(--text);
    }

    .subtitle {
        color: var(--text-muted);
        font-size: 14px;
        margin: 0;
    }

    .search-box {
        display: flex;
        align-items: center;
        gap: 10px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 10px 14px;
        min-width: 280px;
    }

    .search-box input {
        flex: 1;
        background: transparent;
        border: none;
        color: var(--text);
        font-size: 14px;
        outline: none;
    }

    .search-box input::placeholder {
        color: var(--text-muted);
    }

    .recordings-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
        gap: 16px;
    }

    .recording-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 10px;
        padding: 16px;
        transition: all 0.2s;
    }

    .recording-card:hover {
        border-color: var(--accent);
        transform: translateY(-2px);
    }

    .card-header {
        display: flex;
        align-items: flex-start;
        gap: 14px;
        cursor: pointer;
        margin-bottom: 12px;
    }

    .rec-icon {
        padding: 10px;
        background: rgba(255, 71, 87, 0.1);
        border-radius: 8px;
        color: #ff4757;
    }

    .rec-info h3 {
        margin: 0 0 4px 0;
        font-size: 15px;
        font-weight: 600;
        color: var(--text);
    }

    .rec-info .meta {
        font-size: 12px;
        color: var(--text-muted);
    }

    .card-actions {
        display: flex;
        gap: 6px;
        border-top: 1px solid var(--border);
        padding-top: 12px;
    }

    .icon-btn {
        padding: 8px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        cursor: pointer;
        transition: all 0.2s;
    }

    .icon-btn:hover {
        color: var(--text);
        border-color: var(--text-muted);
        background: var(--bg-secondary);
    }

    .icon-btn.active {
        color: var(--accent);
        border-color: var(--accent);
    }

    .icon-btn.delete:hover {
        color: #ff4757;
        border-color: #ff4757;
        background: rgba(255, 71, 87, 0.1);
    }

    .loading, .empty-state {
        grid-column: 1 / -1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 60px 20px;
        text-align: center;
        color: var(--text-muted);
    }

    .loading .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
        margin-bottom: 16px;
    }

    .empty-state h3 {
        margin: 16px 0 8px;
        font-size: 18px;
        color: var(--text);
    }

    .empty-state p {
        margin: 0;
        font-size: 14px;
    }

    /* Player Section */
    .player-section {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
        overflow: hidden;
    }

    .player-header {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
    }

    .back-btn {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 8px 12px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        font-size: 13px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .back-btn:hover {
        color: var(--text);
        border-color: var(--text-muted);
    }

    .player-info {
        flex: 1;
    }

    .player-info h2 {
        margin: 0 0 4px;
        font-size: 18px;
        font-weight: 600;
    }

    .player-info .meta {
        font-size: 12px;
        color: var(--text-muted);
    }

    .player-actions {
        display: flex;
        gap: 8px;
    }

    .action-btn {
        padding: 8px 12px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        cursor: pointer;
        transition: all 0.2s;
    }

    .action-btn:hover {
        color: var(--text);
        border-color: var(--text-muted);
    }

    .player-container {
        background: #0a0a14;
        height: 400px;
        min-height: 300px;
    }

    .player-container :global(.xterm) {
        height: 100%;
        padding: 12px;
    }

    .player-controls {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 16px 20px;
        background: var(--bg-secondary);
        border-top: 1px solid var(--border);
    }

    .progress-bar {
        flex: 1;
        height: 6px;
        background: var(--border);
        border-radius: 3px;
        overflow: hidden;
        cursor: pointer;
    }

    .progress-fill {
        height: 100%;
        background: linear-gradient(90deg, #ff4757, #ff6b7a);
        border-radius: 3px;
        transition: width 0.1s ease;
    }

    .control-buttons {
        display: flex;
        gap: 8px;
    }

    .ctrl-btn {
        padding: 10px 14px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        cursor: pointer;
        transition: all 0.2s;
    }

    .ctrl-btn:hover {
        color: var(--text);
        border-color: var(--text-muted);
    }

    .ctrl-btn.primary {
        background: rgba(255, 71, 87, 0.1);
        border-color: #ff4757;
        color: #ff4757;
    }

    .ctrl-btn.primary:hover {
        background: rgba(255, 71, 87, 0.2);
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    @media (max-width: 600px) {
        .page-header {
            flex-direction: column;
        }

        .search-box {
            width: 100%;
            min-width: unset;
        }

        .recordings-grid {
            grid-template-columns: 1fr;
        }

        .player-header {
            flex-direction: column;
            align-items: flex-start;
        }

        .player-actions {
            width: 100%;
            justify-content: flex-end;
        }
    }
</style>
