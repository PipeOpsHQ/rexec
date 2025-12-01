<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { toast } from '$stores/toast';
  import { api } from '$utils/api';

  const dispatch = createEventDispatcher<{
    back: void;
  }>();

  interface SSHKey {
    id: string;
    name: string;
    fingerprint: string;
    created_at: string;
    last_used_at?: string;
  }

  let keys: SSHKey[] = [];
  let isLoading = true;
  let showAddModal = false;
  let newKeyName = '';
  let newKeyContent = '';
  let isAdding = false;

  // Load SSH keys
  async function loadKeys() {
    isLoading = true;
    const { data, error } = await api.get<{ keys: SSHKey[] }>('/api/ssh-keys');

    if (data) {
      keys = data.keys || [];
    } else if (error) {
      toast.error('Failed to load SSH keys');
    }
    isLoading = false;
  }

  // Add new SSH key
  async function addKey() {
    if (!newKeyName.trim() || !newKeyContent.trim()) {
      toast.error('Please provide a name and key');
      return;
    }

    isAdding = true;
    const { data, error } = await api.post<SSHKey>('/api/ssh-keys', {
      name: newKeyName.trim(),
      public_key: newKeyContent.trim(),
    });

    if (data) {
      keys = [...keys, data];
      toast.success('SSH key added');
      closeModal();
    } else {
      toast.error(error || 'Failed to add SSH key');
    }
    isAdding = false;
  }

  // Delete SSH key
  async function deleteKey(id: string, name: string) {
    if (!confirm(`Delete SSH key "${name}"? This cannot be undone.`)) {
      return;
    }

    const { error } = await api.delete(`/api/ssh-keys/${id}`);

    if (!error) {
      keys = keys.filter((k) => k.id !== id);
      toast.success('SSH key deleted');
    } else {
      toast.error(error || 'Failed to delete SSH key');
    }
  }

  // Modal helpers
  function openModal() {
    newKeyName = '';
    newKeyContent = '';
    showAddModal = true;
  }

  function closeModal() {
    showAddModal = false;
    newKeyName = '';
    newKeyContent = '';
  }

  // Format date
  function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  onMount(loadKeys);
</script>

<div class="ssh-keys">
  <div class="ssh-keys-header">
    <button class="back-btn" on:click={() => dispatch('back')}>
      ← Back
    </button>
    <h1>SSH Keys</h1>
  </div>

  <div class="ssh-keys-content">
    <div class="section-header">
      <p class="section-description">
        SSH keys allow you to connect to your terminals securely without a password.
      </p>
      <button class="btn btn-primary" on:click={openModal}>
        + Add SSH Key
      </button>
    </div>

    {#if isLoading}
      <div class="loading-state">
        <div class="spinner"></div>
        <p>Loading SSH keys...</p>
      </div>
    {:else if keys.length === 0}
      <div class="empty-state">
        <div class="empty-icon">🔑</div>
        <h2>No SSH Keys</h2>
        <p>Add an SSH key to enable secure passwordless access to your terminals.</p>
        <button class="btn btn-primary" on:click={openModal}>
          + Add Your First Key
        </button>
      </div>
    {:else}
      <div class="keys-list">
        {#each keys as key (key.id)}
          <div class="key-card">
            <div class="key-icon">🔑</div>
            <div class="key-info">
              <div class="key-name">{key.name}</div>
              <div class="key-fingerprint">{key.fingerprint}</div>
              <div class="key-meta">
                <span>Added {formatDate(key.created_at)}</span>
                {#if key.last_used_at}
                  <span>• Last used {formatDate(key.last_used_at)}</span>
                {/if}
              </div>
            </div>
            <button
              class="btn btn-danger btn-sm"
              on:click={() => deleteKey(key.id, key.name)}
            >
              Delete
            </button>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Instructions -->
    <div class="instructions">
      <h3>How to generate an SSH key</h3>
      <div class="instruction-steps">
        <div class="step">
          <span class="step-number">1</span>
          <div class="step-content">
            <p>Open your terminal and run:</p>
            <code>ssh-keygen -t ed25519 -C "your_email@example.com"</code>
          </div>
        </div>
        <div class="step">
          <span class="step-number">2</span>
          <div class="step-content">
            <p>Copy your public key:</p>
            <code>cat ~/.ssh/id_ed25519.pub</code>
          </div>
        </div>
        <div class="step">
          <span class="step-number">3</span>
          <div class="step-content">
            <p>Paste the key above and give it a memorable name.</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- Add Key Modal -->
{#if showAddModal}
  <div class="modal-overlay" on:click={closeModal} role="presentation">
    <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
      <div class="modal-header">
        <h2>Add SSH Key</h2>
        <button class="modal-close" on:click={closeModal}>×</button>
      </div>

      <div class="modal-body">
        <div class="form-group">
          <label for="key-name">Name</label>
          <input
            type="text"
            id="key-name"
            bind:value={newKeyName}
            placeholder="e.g., MacBook Pro, Work Laptop"
            maxlength="64"
          />
        </div>

        <div class="form-group">
          <label for="key-content">Public Key</label>
          <textarea
            id="key-content"
            bind:value={newKeyContent}
            placeholder="ssh-ed25519 AAAA... or ssh-rsa AAAA..."
            rows="4"
          ></textarea>
          <span class="form-hint">
            Paste your public key (usually from ~/.ssh/id_ed25519.pub or ~/.ssh/id_rsa.pub)
          </span>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn btn-secondary" on:click={closeModal} disabled={isAdding}>
          Cancel
        </button>
        <button
          class="btn btn-primary"
          on:click={addKey}
          disabled={isAdding || !newKeyName.trim() || !newKeyContent.trim()}
        >
          {isAdding ? 'Adding...' : 'Add Key'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .ssh-keys {
    max-width: 800px;
    margin: 0 auto;
    animation: fadeIn 0.2s ease;
  }

  .ssh-keys-header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 32px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border);
  }

  .back-btn {
    background: none;
    border: 1px solid var(--border);
    color: var(--text-secondary);
    padding: 6px 12px;
    font-family: var(--font-mono);
    font-size: 12px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .back-btn:hover {
    border-color: var(--text);
    color: var(--text);
  }

  .ssh-keys-header h1 {
    font-size: 20px;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin: 0;
  }

  .ssh-keys-content {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
  }

  .section-description {
    color: var(--text-muted);
    font-size: 13px;
    margin: 0;
  }

  /* Loading State */
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 60px 20px;
    gap: 16px;
  }

  .loading-state p {
    color: var(--text-muted);
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  /* Empty State */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 60px 20px;
    text-align: center;
    border: 1px dashed var(--border);
    background: var(--bg-card);
  }

  .empty-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }

  .empty-state h2 {
    font-size: 18px;
    margin-bottom: 8px;
    text-transform: uppercase;
  }

  .empty-state p {
    color: var(--text-muted);
    max-width: 400px;
    margin-bottom: 24px;
  }

  /* Keys List */
  .keys-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .key-card {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    transition: border-color 0.2s;
  }

  .key-card:hover {
    border-color: var(--text-muted);
  }

  .key-icon {
    font-size: 24px;
  }

  .key-info {
    flex: 1;
    min-width: 0;
  }

  .key-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--text);
    margin-bottom: 4px;
  }

  .key-fingerprint {
    font-size: 11px;
    font-family: var(--font-mono);
    color: var(--text-secondary);
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .key-meta {
    font-size: 11px;
    color: var(--text-muted);
  }

  /* Instructions */
  .instructions {
    background: var(--bg-card);
    border: 1px solid var(--border);
    padding: 20px;
    margin-top: 16px;
  }

  .instructions h3 {
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--accent);
    margin: 0 0 16px;
  }

  .instruction-steps {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .step {
    display: flex;
    gap: 12px;
  }

  .step-number {
    width: 24px;
    height: 24px;
    background: var(--accent);
    color: var(--bg);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: bold;
    flex-shrink: 0;
  }

  .step-content {
    flex: 1;
  }

  .step-content p {
    margin: 0 0 8px;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .step-content code {
    display: block;
    padding: 10px 12px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--accent);
    overflow-x: auto;
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    animation: fadeIn 0.15s ease;
  }

  .modal {
    background: var(--bg-card);
    border: 1px solid var(--border);
    width: 100%;
    max-width: 500px;
    max-height: 90vh;
    overflow-y: auto;
    animation: slideIn 0.2s ease;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border);
  }

  .modal-header h2 {
    font-size: 16px;
    text-transform: uppercase;
    margin: 0;
  }

  .modal-close {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 24px;
    cursor: pointer;
    padding: 0;
    line-height: 1;
  }

  .modal-close:hover {
    color: var(--text);
  }

  .modal-body {
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .form-group label {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
  }

  .form-group input,
  .form-group textarea {
    width: 100%;
    padding: 10px 12px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 13px;
  }

  .form-group input:focus,
  .form-group textarea:focus {
    outline: none;
    border-color: var(--accent);
  }

  .form-group textarea {
    resize: vertical;
    min-height: 100px;
  }

  .form-hint {
    font-size: 11px;
    color: var(--text-muted);
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding: 16px 20px;
    border-top: 1px solid var(--border);
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateY(-20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @media (max-width: 600px) {
    .section-header {
      flex-direction: column;
      align-items: flex-start;
    }

    .key-card {
      flex-wrap: wrap;
    }

    .key-info {
      width: 100%;
      order: 1;
    }

    .key-icon {
      order: 0;
    }

    .key-card .btn {
      order: 2;
      margin-left: auto;
    }
  }
</style>
