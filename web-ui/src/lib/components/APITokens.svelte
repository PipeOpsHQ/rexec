<script lang="ts">
  import { onMount } from 'svelte';
  import { tokens, type APIToken } from '$stores/tokens';
  import { toast } from '$stores/toast';

  let newTokenName = '';
  let newTokenExpiry = ''; // '', '30', '60', '90', '365'
  let selectedScopes = ['read', 'write'];
  let showCreateModal = false;
  let createdToken = '';
  let showTokenModal = false;
  let tokenCopied = false;

  onMount(() => {
    tokens.fetchTokens();
  });

  async function handleCreateToken() {
    if (!newTokenName.trim()) {
      toast.error('Token name is required');
      return;
    }

    const expiresIn = newTokenExpiry ? parseInt(newTokenExpiry) : undefined;
    const result = await tokens.createToken(newTokenName.trim(), selectedScopes, expiresIn);
    
    if (result) {
      createdToken = result.token;
      showCreateModal = false;
      showTokenModal = true;
      newTokenName = '';
      newTokenExpiry = '';
      selectedScopes = ['read', 'write'];
      toast.success('API token created successfully');
    } else {
      toast.error('Failed to create token');
    }
  }

  async function handleRevokeToken(tokenId: string) {
    if (confirm('Are you sure you want to revoke this token? This cannot be undone.')) {
      const success = await tokens.revokeToken(tokenId);
      if (success) {
        toast.success('Token revoked');
      } else {
        toast.error('Failed to revoke token');
      }
    }
  }

  function copyToken() {
    navigator.clipboard.writeText(createdToken);
    tokenCopied = true;
    toast.success('Token copied to clipboard');
    setTimeout(() => tokenCopied = false, 2000);
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  function formatRelativeTime(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    if (days === 0) return 'Today';
    if (days === 1) return 'Yesterday';
    if (days < 30) return `${days} days ago`;
    return formatDate(dateStr);
  }

  function toggleScope(scope: string) {
    if (selectedScopes.includes(scope)) {
      selectedScopes = selectedScopes.filter(s => s !== scope);
    } else {
      selectedScopes = [...selectedScopes, scope];
    }
  }
</script>

<div class="api-tokens">
  <div class="tokens-header">
    <div class="header-content">
      <h2>API Tokens</h2>
      <p class="description">Create personal access tokens to authenticate with the CLI or API.</p>
    </div>
    <button class="btn btn-primary" onclick={() => showCreateModal = true}>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 5v14M5 12h14" />
      </svg>
      New Token
    </button>
  </div>

  {#if $tokens.loading && $tokens.tokens.length === 0}
    <div class="loading">Loading tokens...</div>
  {:else if $tokens.tokens.length === 0}
    <div class="empty-state">
      <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
      <h3>No API tokens yet</h3>
      <p>Create a token to authenticate with the rexec CLI or make API requests.</p>
      <button class="btn btn-primary" onclick={() => showCreateModal = true}>
        Create your first token
      </button>
    </div>
  {:else}
    <div class="tokens-list">
      {#each $tokens.tokens as token (token.id)}
        <div class="token-card">
          <div class="token-info">
            <div class="token-name">
              <svg class="token-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
              </svg>
              <span>{token.name}</span>
            </div>
            <div class="token-meta">
              <span class="token-prefix">{token.token_prefix}...</span>
              <span class="token-scopes">
                {#each token.scopes as scope}
                  <span class="scope-badge">{scope}</span>
                {/each}
              </span>
            </div>
          </div>
          <div class="token-details">
            <div class="detail">
              <span class="label">Created</span>
              <span class="value">{formatDate(token.created_at)}</span>
            </div>
            {#if token.last_used_at}
              <div class="detail">
                <span class="label">Last used</span>
                <span class="value">{formatRelativeTime(token.last_used_at)}</span>
              </div>
            {:else}
              <div class="detail">
                <span class="label">Last used</span>
                <span class="value never">Never</span>
              </div>
            {/if}
            {#if token.expires_at}
              <div class="detail">
                <span class="label">Expires</span>
                <span class="value">{formatDate(token.expires_at)}</span>
              </div>
            {/if}
          </div>
          <div class="token-actions">
            <button class="btn btn-danger btn-sm" onclick={() => handleRevokeToken(token.id)}>
              Revoke
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Create Token Modal -->
{#if showCreateModal}
  <div class="modal-overlay" onclick={() => showCreateModal = false}>
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h3>Create API Token</h3>
        <button class="close-btn" onclick={() => showCreateModal = false}>×</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label for="token-name">Token Name</label>
          <input
            type="text"
            id="token-name"
            bind:value={newTokenName}
            placeholder="e.g., CLI Token, CI/CD Pipeline"
          />
        </div>

        <div class="form-group">
          <label>Permissions</label>
          <div class="scope-options">
            <label class="scope-option">
              <input type="checkbox" checked={selectedScopes.includes('read')} onchange={() => toggleScope('read')} />
              <span class="scope-label">
                <strong>Read</strong>
                <small>List containers, view status</small>
              </span>
            </label>
            <label class="scope-option">
              <input type="checkbox" checked={selectedScopes.includes('write')} onchange={() => toggleScope('write')} />
              <span class="scope-label">
                <strong>Write</strong>
                <small>Create, start, stop containers</small>
              </span>
            </label>
            <label class="scope-option">
              <input type="checkbox" checked={selectedScopes.includes('admin')} onchange={() => toggleScope('admin')} />
              <span class="scope-label">
                <strong>Admin</strong>
                <small>Delete containers, manage tokens</small>
              </span>
            </label>
          </div>
        </div>

        <div class="form-group">
          <label for="token-expiry">Expiration</label>
          <select id="token-expiry" bind:value={newTokenExpiry}>
            <option value="">No expiration</option>
            <option value="30">30 days</option>
            <option value="60">60 days</option>
            <option value="90">90 days</option>
            <option value="365">1 year</option>
          </select>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" onclick={() => showCreateModal = false}>Cancel</button>
        <button class="btn btn-primary" onclick={handleCreateToken} disabled={!newTokenName.trim()}>
          Create Token
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Token Created Modal -->
{#if showTokenModal}
  <div class="modal-overlay" onclick={() => showTokenModal = false}>
    <div class="modal token-created-modal" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header success">
        <svg class="success-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <h3>Token Created</h3>
      </div>
      <div class="modal-body">
        <p class="warning">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          Make sure to copy your token now. You won't be able to see it again!
        </p>
        <div class="token-display">
          <code>{createdToken}</code>
          <button class="copy-btn" onclick={copyToken}>
            {#if tokenCopied}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M5 13l4 4L19 7" />
              </svg>
            {:else}
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" />
                <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
              </svg>
            {/if}
          </button>
        </div>
        <div class="usage-example">
          <p>Use this token with the CLI:</p>
          <code>rexec login --token "{createdToken}"</code>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-primary" onclick={() => showTokenModal = false}>Done</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .api-tokens {
    padding: 1.5rem;
    max-width: 900px;
    margin: 0 auto;
  }

  .tokens-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 2rem;
    gap: 1rem;
  }

  .header-content h2 {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text-primary, #e0e0e0);
    margin: 0 0 0.5rem 0;
  }

  .description {
    color: var(--text-secondary, #888);
    font-size: 0.875rem;
    margin: 0;
  }

  .tokens-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .token-card {
    background: var(--bg-secondary, #1a1a1a);
    border: 1px solid var(--border-color, #2a2a2a);
    border-radius: 12px;
    padding: 1.25rem;
    display: grid;
    grid-template-columns: 1fr auto auto;
    gap: 1.5rem;
    align-items: center;
  }

  .token-info {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .token-name {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 500;
    color: var(--text-primary, #e0e0e0);
  }

  .token-icon {
    width: 18px;
    height: 18px;
    color: var(--accent, #8b5cf6);
  }

  .token-meta {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .token-prefix {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.75rem;
    color: var(--text-secondary, #888);
    background: var(--bg-tertiary, #0a0a0a);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
  }

  .token-scopes {
    display: flex;
    gap: 0.25rem;
  }

  .scope-badge {
    font-size: 0.625rem;
    text-transform: uppercase;
    padding: 0.125rem 0.375rem;
    background: var(--accent, #8b5cf6);
    color: white;
    border-radius: 3px;
    font-weight: 500;
  }

  .token-details {
    display: flex;
    gap: 1.5rem;
  }

  .detail {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .detail .label {
    font-size: 0.625rem;
    text-transform: uppercase;
    color: var(--text-secondary, #888);
    letter-spacing: 0.5px;
  }

  .detail .value {
    font-size: 0.875rem;
    color: var(--text-primary, #e0e0e0);
  }

  .detail .value.never {
    color: var(--text-secondary, #888);
    font-style: italic;
  }

  .token-actions {
    display: flex;
    gap: 0.5rem;
  }

  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
    background: var(--bg-secondary, #1a1a1a);
    border: 1px dashed var(--border-color, #2a2a2a);
    border-radius: 12px;
  }

  .empty-icon {
    width: 48px;
    height: 48px;
    color: var(--text-secondary, #888);
    margin-bottom: 1rem;
  }

  .empty-state h3 {
    color: var(--text-primary, #e0e0e0);
    margin: 0 0 0.5rem 0;
  }

  .empty-state p {
    color: var(--text-secondary, #888);
    margin: 0 0 1.5rem 0;
  }

  .loading {
    text-align: center;
    padding: 3rem;
    color: var(--text-secondary, #888);
  }

  /* Modal Styles */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(4px);
  }

  .modal {
    background: var(--bg-secondary, #1a1a1a);
    border: 1px solid var(--border-color, #2a2a2a);
    border-radius: 16px;
    width: 100%;
    max-width: 480px;
    max-height: 90vh;
    overflow: auto;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid var(--border-color, #2a2a2a);
  }

  .modal-header h3 {
    margin: 0;
    color: var(--text-primary, #e0e0e0);
  }

  .modal-header.success {
    flex-direction: column;
    gap: 0.5rem;
    text-align: center;
    padding: 1.5rem;
    background: linear-gradient(135deg, rgba(34, 197, 94, 0.1), transparent);
  }

  .success-icon {
    width: 48px;
    height: 48px;
    color: #22c55e;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 1.5rem;
    color: var(--text-secondary, #888);
    cursor: pointer;
    padding: 0;
    line-height: 1;
  }

  .close-btn:hover {
    color: var(--text-primary, #e0e0e0);
  }

  .modal-body {
    padding: 1.5rem;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    padding: 1rem 1.5rem;
    border-top: 1px solid var(--border-color, #2a2a2a);
  }

  .form-group {
    margin-bottom: 1.25rem;
  }

  .form-group label {
    display: block;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary, #e0e0e0);
    margin-bottom: 0.5rem;
  }

  .form-group input,
  .form-group select {
    width: 100%;
    padding: 0.75rem;
    background: var(--bg-tertiary, #0a0a0a);
    border: 1px solid var(--border-color, #2a2a2a);
    border-radius: 8px;
    color: var(--text-primary, #e0e0e0);
    font-size: 0.875rem;
  }

  .form-group input:focus,
  .form-group select:focus {
    outline: none;
    border-color: var(--accent, #8b5cf6);
  }

  .scope-options {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .scope-option {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    cursor: pointer;
    padding: 0.75rem;
    background: var(--bg-tertiary, #0a0a0a);
    border: 1px solid var(--border-color, #2a2a2a);
    border-radius: 8px;
  }

  .scope-option:hover {
    border-color: var(--accent, #8b5cf6);
  }

  .scope-option input {
    margin-top: 2px;
  }

  .scope-label {
    display: flex;
    flex-direction: column;
  }

  .scope-label strong {
    color: var(--text-primary, #e0e0e0);
    font-size: 0.875rem;
  }

  .scope-label small {
    color: var(--text-secondary, #888);
    font-size: 0.75rem;
  }

  .warning {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    padding: 1rem;
    background: rgba(234, 179, 8, 0.1);
    border: 1px solid rgba(234, 179, 8, 0.3);
    border-radius: 8px;
    color: #eab308;
    font-size: 0.875rem;
    margin-bottom: 1.5rem;
  }

  .warning svg {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }

  .token-display {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background: var(--bg-tertiary, #0a0a0a);
    border: 1px solid var(--border-color, #2a2a2a);
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1.5rem;
  }

  .token-display code {
    flex: 1;
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.75rem;
    color: var(--accent, #8b5cf6);
    word-break: break-all;
  }

  .copy-btn {
    background: none;
    border: none;
    padding: 0.5rem;
    cursor: pointer;
    color: var(--text-secondary, #888);
    border-radius: 6px;
  }

  .copy-btn:hover {
    background: var(--bg-secondary, #1a1a1a);
    color: var(--text-primary, #e0e0e0);
  }

  .copy-btn svg {
    width: 18px;
    height: 18px;
  }

  .usage-example {
    background: var(--bg-tertiary, #0a0a0a);
    border-radius: 8px;
    padding: 1rem;
  }

  .usage-example p {
    font-size: 0.75rem;
    color: var(--text-secondary, #888);
    margin: 0 0 0.5rem 0;
  }

  .usage-example code {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.75rem;
    color: var(--text-primary, #e0e0e0);
    word-break: break-all;
  }

  .token-created-modal {
    max-width: 560px;
  }

  /* Button Styles */
  .btn {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.625rem 1rem;
    border-radius: 8px;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: all 0.2s;
  }

  .btn svg {
    width: 16px;
    height: 16px;
  }

  .btn-primary {
    background: var(--accent, #8b5cf6);
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--accent-hover, #7c3aed);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-secondary {
    background: var(--bg-tertiary, #0a0a0a);
    color: var(--text-primary, #e0e0e0);
    border: 1px solid var(--border-color, #2a2a2a);
  }

  .btn-secondary:hover {
    background: var(--bg-secondary, #1a1a1a);
  }

  .btn-danger {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  .btn-danger:hover {
    background: rgba(239, 68, 68, 0.2);
  }

  .btn-sm {
    padding: 0.375rem 0.75rem;
    font-size: 0.75rem;
  }

  @media (max-width: 640px) {
    .token-card {
      grid-template-columns: 1fr;
      gap: 1rem;
    }

    .token-details {
      flex-wrap: wrap;
    }

    .tokens-header {
      flex-direction: column;
    }
  }
</style>
