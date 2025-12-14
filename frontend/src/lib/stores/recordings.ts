import { writable, get } from 'svelte/store';
import { auth } from './auth';

// Types
export interface Recording {
  id: string;
  title: string;
  durationMs: number;
  duration: string;
  sizeBytes: number;
  isPublic: boolean;
  shareToken: string;
  shareUrl: string;
  createdAt: string;
}

export interface RecordingStatus {
  recording: boolean;
  recordingId?: string;
  startedAt?: string;
  durationMs?: number;
  eventsCount?: number;
}

// Store state
interface RecordingState {
  recordings: Recording[];
  activeRecordings: Map<string, RecordingStatus>; // containerId -> status
  isLoading: boolean;
  error: string | null;
}

function createRecordingStore() {
  const { subscribe, set, update } = writable<RecordingState>({
    recordings: [],
    activeRecordings: new Map(),
    isLoading: false,
    error: null
  });

  const API_BASE = '';

  async function fetchRecordings(): Promise<void> {
    const token = get(auth).token;
    if (!token) return;

    update(s => ({ ...s, isLoading: true, error: null }));

    try {
      const res = await fetch(`${API_BASE}/api/recordings`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (!res.ok) throw new Error('Failed to fetch recordings');

      const data = await res.json();
      const recordings = (data.recordings || []).map((r: any) => ({
        id: r.id,
        title: r.title,
        durationMs: r.duration_ms,
        duration: r.duration,
        sizeBytes: r.size_bytes,
        isPublic: r.is_public,
        shareToken: r.share_token,
        shareUrl: r.share_url,
        createdAt: r.created_at
      }));

      update(s => ({ ...s, recordings, isLoading: false }));
    } catch (err: any) {
      update(s => ({ ...s, error: err.message, isLoading: false }));
    }
  }

  async function startRecording(containerId: string, title?: string): Promise<string | null> {
    const token = get(auth).token;
    if (!token) {
      console.error('[Recording] No auth token');
      return null;
    }



    try {
      const res = await fetch(`${API_BASE}/api/recordings/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ container_id: containerId, title })
      });

      if (!res.ok) {
        const error = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
        console.error('[Recording] Start failed:', error);
        throw new Error(error.error || 'Failed to start recording');
      }

      const data = await res.json();
      
      update(s => {
        const activeRecordings = new Map(s.activeRecordings);
        activeRecordings.set(containerId, {
          recording: true,
          recordingId: data.recording_id,
          startedAt: data.started_at,
          durationMs: 0,
          eventsCount: 0
        });
        return { ...s, activeRecordings };
      });

      return data.recording_id;
    } catch (err: any) {
      update(s => ({ ...s, error: err.message }));
      return null;
    }
  }

  async function stopRecording(containerId: string): Promise<Recording | null> {
    const token = get(auth).token;
    if (!token) {
      console.error('[Recording] No auth token');
      return null;
    }



    try {
      const res = await fetch(`${API_BASE}/api/recordings/stop/${containerId}`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (!res.ok) {
        const error = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
        console.error('[Recording] Stop failed:', error);
        throw new Error(error.error || 'Failed to stop recording');
      }

      const data = await res.json();
      
      update(s => {
        const activeRecordings = new Map(s.activeRecordings);
        activeRecordings.delete(containerId);
        return { ...s, activeRecordings };
      });

      // Refresh recordings list
      await fetchRecordings();

      return {
        id: data.recording_id,
        title: '',
        durationMs: data.duration_ms,
        duration: data.duration,
        sizeBytes: data.size_bytes,
        isPublic: false,
        shareToken: data.share_token,
        shareUrl: `/r/${data.share_token}`,
        createdAt: new Date().toISOString()
      };
    } catch (err: any) {
      update(s => ({ ...s, error: err.message }));
      return null;
    }
  }

  async function getRecordingStatus(containerId: string): Promise<RecordingStatus> {
    const token = get(auth).token;
    if (!token) return { recording: false };

    try {
      const res = await fetch(`${API_BASE}/api/recordings/status/${containerId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (!res.ok) return { recording: false };

      const data = await res.json();
      
      if (data.recording) {
        update(s => {
          const activeRecordings = new Map(s.activeRecordings);
          activeRecordings.set(containerId, {
            recording: true,
            recordingId: data.recording_id,
            startedAt: data.started_at,
            durationMs: data.duration_ms,
            eventsCount: data.events_count
          });
          return { ...s, activeRecordings };
        });
      }

      return data;
    } catch (err) {
      return { recording: false };
    }
  }

  async function updateRecording(id: string, updates: { isPublic?: boolean; title?: string }): Promise<boolean> {
    const token = get(auth).token;
    if (!token) return false;

    try {
      const res = await fetch(`${API_BASE}/api/recordings/${id}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ is_public: updates.isPublic, title: updates.title })
      });

      if (res.ok) {
        await fetchRecordings();
        return true;
      }
    } catch (err) {
      console.error('[Recording] Failed to update:', err);
    }
    return false;
  }

  async function deleteRecording(id: string): Promise<boolean> {
    const token = get(auth).token;
    if (!token) return false;

    try {
      const res = await fetch(`${API_BASE}/api/recordings/${id}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (res.ok) {
        update(s => ({
          ...s,
          recordings: s.recordings.filter(r => r.id !== id)
        }));
        return true;
      }
    } catch (err) {
      console.error('[Recording] Failed to delete:', err);
    }
    return false;
  }

  function isRecording(containerId: string): boolean {
    const state = get({ subscribe });
    return state.activeRecordings.get(containerId)?.recording || false;
  }

  function getActiveRecording(containerId: string): RecordingStatus | undefined {
    const state = get({ subscribe });
    return state.activeRecordings.get(containerId);
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }

  function reset() {
    set({
      recordings: [],
      activeRecordings: new Map(),
      isLoading: false,
      error: null
    });
  }

  return {
    subscribe,
    fetchRecordings,
    startRecording,
    stopRecording,
    getRecordingStatus,
    updateRecording,
    deleteRecording,
    isRecording,
    getActiveRecording,
    formatSize,
    reset
  };
}

export const recordings = createRecordingStore();
