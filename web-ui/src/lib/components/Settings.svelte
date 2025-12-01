<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { auth, isGuest } from '$stores/auth';
  import { toast } from '$stores/toast';

  const dispatch = createEventDispatcher<{
    back: void;
  }>();

  // Settings state
  let theme = 'dark';
  let fontSize = 14;
  let cursorBlink = true;
  let cursorStyle: 'bar' | 'block' | 'underline' = 'bar';
  let scrollback = 5000;
  let copyOnSelect = false;

  // Load settings from localStorage
  function loadSettings() {
    try {
      const saved = localStorage.getItem('rexec_settings');
      if (saved) {
        const settings = JSON.parse(saved);
        theme = settings.theme || 'dark';
        fontSize = settings.fontSize || 14;
        cursorBlink = settings.cursorBlink ?? true;
        cursorStyle = settings.cursorStyle || 'bar';
        scrollback = settings.scrollback || 5000;
        copyOnSelect = settings.copyOnSelect ?? false;
      }
    } catch (e) {
      console.error('Failed to load settings:', e);
    }
  }

  // Save settings to localStorage
  function saveSettings() {
    try {
      localStorage.setItem('rexec_settings', JSON.stringify({
        theme,
        fontSize,
        cursorBlink,
        cursorStyle,
        scrollback,
        copyOnSelect,
      }));
      toast.success('Settings saved');
    } catch (e) {
      console.error('Failed to save settings:', e);
      toast.error('Failed to save settings');
    }
  }

  // Reset to defaults
  function resetSettings() {
    theme = 'dark';
    fontSize = 14;
    cursorBlink = true;
    cursorStyle = 'bar';
    scrollback = 5000;
    copyOnSelect = false;
    saveSettings();
  }

  // Load on mount
  loadSettings();
</script>

<div class="settings">
  <div class="settings-header">
    <button class="back-btn" on:click={() => dispatch('back')}>
      ← Back
    </button>
    <h1>Settings</h1>
  </div>

  <div class="settings-content">
    <!-- Account Section -->
    <section class="settings-section">
      <h2>Account</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label>Email</label>
          <span class="setting-description">Your account email address</span>
        </div>
        <div class="setting-value">
          <span class="value-text">{$auth.user?.email || 'Not set'}</span>
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label>Account Type</label>
          <span class="setting-description">Your current subscription plan</span>
        </div>
        <div class="setting-value">
          <span class="tier-badge tier-{$auth.user?.tier || 'free'}">
            {$auth.user?.tier?.toUpperCase() || 'FREE'}
          </span>
        </div>
      </div>

      {#if $isGuest}
        <div class="setting-item warning">
          <div class="setting-info">
            <label>⚠️ Guest Account</label>
            <span class="setting-description">
              Your session is temporary. Sign in to save your data.
            </span>
          </div>
          <div class="setting-value">
            <button class="btn btn-primary btn-sm" on:click={() => auth.getOAuthUrl().then(url => url && (window.location.href = url))}>
              Sign In
            </button>
          </div>
        </div>
      {/if}
    </section>

    <!-- Terminal Section -->
    <section class="settings-section">
      <h2>Terminal</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label for="font-size">Font Size</label>
          <span class="setting-description">Terminal font size in pixels</span>
        </div>
        <div class="setting-value">
          <input
            type="number"
            id="font-size"
            bind:value={fontSize}
            min="10"
            max="24"
            class="input-sm"
          />
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label for="cursor-style">Cursor Style</label>
          <span class="setting-description">Terminal cursor appearance</span>
        </div>
        <div class="setting-value">
          <select id="cursor-style" bind:value={cursorStyle} class="select-sm">
            <option value="bar">Bar</option>
            <option value="block">Block</option>
            <option value="underline">Underline</option>
          </select>
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label for="cursor-blink">Cursor Blink</label>
          <span class="setting-description">Enable cursor blinking</span>
        </div>
        <div class="setting-value">
          <label class="toggle">
            <input type="checkbox" id="cursor-blink" bind:checked={cursorBlink} />
            <span class="toggle-slider"></span>
          </label>
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label for="scrollback">Scrollback Lines</label>
          <span class="setting-description">Number of lines to keep in history</span>
        </div>
        <div class="setting-value">
          <select id="scrollback" bind:value={scrollback} class="select-sm">
            <option value={1000}>1,000</option>
            <option value={5000}>5,000</option>
            <option value={10000}>10,000</option>
            <option value={50000}>50,000</option>
          </select>
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label for="copy-on-select">Copy on Select</label>
          <span class="setting-description">Automatically copy selected text</span>
        </div>
        <div class="setting-value">
          <label class="toggle">
            <input type="checkbox" id="copy-on-select" bind:checked={copyOnSelect} />
            <span class="toggle-slider"></span>
          </label>
        </div>
      </div>
    </section>

    <!-- Appearance Section -->
    <section class="settings-section">
      <h2>Appearance</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label for="theme">Theme</label>
          <span class="setting-description">Color scheme for the interface</span>
        </div>
        <div class="setting-value">
          <select id="theme" bind:value={theme} class="select-sm">
            <option value="dark">Dark</option>
            <option value="light" disabled>Light (Coming Soon)</option>
          </select>
        </div>
      </div>
    </section>

    <!-- Actions -->
    <div class="settings-actions">
      <button class="btn btn-secondary" on:click={resetSettings}>
        Reset to Defaults
      </button>
      <button class="btn btn-primary" on:click={saveSettings}>
        Save Settings
      </button>
    </div>
  </div>
</div>

<style>
  .settings {
    max-width: 800px;
    margin: 0 auto;
    animation: fadeIn 0.2s ease;
  }

  .settings-header {
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

  .settings-header h1 {
    font-size: 20px;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin: 0;
  }

  .settings-content {
    display: flex;
    flex-direction: column;
    gap: 32px;
  }

  .settings-section {
    background: var(--bg-card);
    border: 1px solid var(--border);
    padding: 20px;
  }

  .settings-section h2 {
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--accent);
    margin: 0 0 16px;
    padding-bottom: 12px;
    border-bottom: 1px solid var(--border);
  }

  .setting-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 0;
    border-bottom: 1px solid var(--border-muted);
  }

  .setting-item:last-child {
    border-bottom: none;
  }

  .setting-item.warning {
    background: rgba(255, 200, 0, 0.1);
    margin: 0 -20px;
    padding: 12px 20px;
    border: none;
  }

  .setting-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .setting-info label {
    font-size: 13px;
    color: var(--text);
    font-weight: 500;
  }

  .setting-description {
    font-size: 11px;
    color: var(--text-muted);
  }

  .setting-value {
    display: flex;
    align-items: center;
  }

  .value-text {
    font-size: 12px;
    color: var(--text-secondary);
    font-family: var(--font-mono);
  }

  .tier-badge {
    font-size: 10px;
    padding: 3px 8px;
    text-transform: uppercase;
    font-weight: 600;
    letter-spacing: 0.5px;
  }

  .tier-guest {
    background: var(--warning);
    color: var(--bg);
  }

  .tier-free {
    background: var(--text-muted);
    color: var(--bg);
  }

  .tier-pro {
    background: var(--accent);
    color: var(--bg);
  }

  .tier-enterprise {
    background: linear-gradient(135deg, var(--accent), #00a0dc);
    color: var(--bg);
  }

  .input-sm {
    width: 80px;
    padding: 6px 10px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 12px;
    text-align: center;
  }

  .input-sm:focus {
    outline: none;
    border-color: var(--accent);
  }

  .select-sm {
    padding: 6px 10px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 12px;
    cursor: pointer;
  }

  .select-sm:focus {
    outline: none;
    border-color: var(--accent);
  }

  /* Toggle Switch */
  .toggle {
    position: relative;
    display: inline-block;
    width: 44px;
    height: 24px;
  }

  .toggle input {
    opacity: 0;
    width: 0;
    height: 0;
  }

  .toggle-slider {
    position: absolute;
    cursor: pointer;
    inset: 0;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    transition: 0.2s;
  }

  .toggle-slider::before {
    position: absolute;
    content: "";
    height: 16px;
    width: 16px;
    left: 3px;
    bottom: 3px;
    background: var(--text-muted);
    transition: 0.2s;
  }

  .toggle input:checked + .toggle-slider {
    background: var(--accent-dim);
    border-color: var(--accent);
  }

  .toggle input:checked + .toggle-slider::before {
    background: var(--accent);
    transform: translateX(20px);
  }

  .settings-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding-top: 16px;
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

  @media (max-width: 600px) {
    .setting-item {
      flex-direction: column;
      align-items: flex-start;
      gap: 12px;
    }

    .settings-actions {
      flex-direction: column;
    }

    .settings-actions button {
      width: 100%;
    }
  }
</style>
