<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import QRCode from 'qrcode';
  import { auth, isGuest } from '$stores/auth';
  import { security, hasPasscode } from '$stores/security';
  import { agents, type Agent } from '$stores/agents';
  import { toast } from '$stores/toast';
  import { theme as themeStore } from '$stores/theme';
  import StatusIcon from './icons/StatusIcon.svelte';

  const dispatch = createEventDispatcher<{
    back: void;
    connectAgent: { agentId: string; agentName: string };
  }>();

  // Get current host for install commands
  const currentHost = typeof window !== 'undefined' ? window.location.host : 'rexec.pipeops.io';
  const protocol = typeof window !== 'undefined' ? window.location.protocol : 'https:';
  const installUrl = `${protocol}//${currentHost}`;

  // Settings state
  let theme: 'dark' | 'light' = 'dark';
  let fontSize = 14;
  let cursorBlink = true;
  let cursorStyle: 'bar' | 'block' | 'underline' = 'bar';
  let scrollback = 5000;
  let copyOnSelect = false;

  // Security state
  let lockTimeout = 5;
  let showPasscodeModal = false;
  let newPasscode = '';
  let confirmPasscode = '';
  let currentPasscode = '';
  let passcodeError = '';
  let isChangingPasscode = false;

  // Sessions state
  let showSessionsModal = false;
  let sessionsLoading = false;
  let sessionsError = '';
  let sessions: any[] = [];

  // MFA state
  let showMFAModal = false;
  let mfaStep: 'intro' | 'setup' | 'verify' | 'disable' = 'intro';
  let mfaSecret = '';
  let mfaQrDataUrl = '';
  let mfaCode = '';
  let mfaError = '';
  let mfaLoading = false;

  // IP Whitelist state
  let showIPModal = false;
  let allowedIPs: string[] = [];
  let newIP = '';
  let ipError = '';
  let ipLoading = false;

  // Audit Logs state
  let showAuditModal = false;
  let auditLogs: any[] = [];
  let auditLoading = false;
  let auditError = '';

  // Agents state
  let showAgentModal = false;
  let newAgentName = '';
  let newAgentDescription = '';
  let createdAgent: { id: string; name: string } | null = null;
  let showInstallScript = false;
  let copiedScript = false;

  // Profile state
  let profileUsername = '';
  let profileFirstName = '';
  let profileLastName = '';
  let profileLoaded = false;

  $: if ($auth.user && !profileLoaded) {
    profileUsername = $auth.user.username;
    profileFirstName = $auth.user.firstName || '';
    profileLastName = $auth.user.lastName || '';
    profileLoaded = true;
  }

  onMount(() => {
    // Refresh security state to ensure we know if passcode is enabled
    security.refreshFromServer();

    if (!$isGuest) {
      // Initial fetch of registered agents
      agents.fetchAgents();
    }
  });

  async function handleCreateAgent() {
    if (!newAgentName.trim()) {
      toast.error('Agent name is required');
      return;
    }
    const agent = await agents.registerAgent(newAgentName.trim(), newAgentDescription.trim());
    if (agent) {
      createdAgent = agent;
      showInstallScript = true;
      toast.success('Agent registered successfully');
    }
  }

  function closeAgentModal() {
    showAgentModal = false;
    newAgentName = '';
    newAgentDescription = '';
    createdAgent = null;
    showInstallScript = false;
    copiedScript = false;
  }

  function handleConnectAgent(agent: Agent) {
    if (agent.status === 'online') {
      dispatch('connectAgent', { agentId: agent.id, agentName: agent.name });
    } else {
      toast.error('Agent is not online');
    }
  }

  async function handleDeleteAgent(agentId: string) {
    if (confirm('Are you sure you want to delete this agent?')) {
      const success = await agents.deleteAgent(agentId);
      if (success) {
        toast.success('Agent deleted');
      } else {
        toast.error('Failed to delete agent');
      }
    }
  }

  function copyInstallScript() {
    if (createdAgent) {
      const script = agents.getInstallScript(createdAgent.id);
      navigator.clipboard.writeText(script);
      copiedScript = true;
      toast.success('Install script copied to clipboard');
      setTimeout(() => copiedScript = false, 2000);
    }
  }

  function getStatusColor(status: string): string {
    switch (status) {
      case 'online': return 'var(--success)';
      case 'offline': return 'var(--text-muted)';
      default: return 'var(--warning)';
    }
  }

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
      // Load security settings
      const secState = security.getState();
      lockTimeout = secState.lockAfterMinutes;
    } catch (e) {
      console.error('Failed to load settings:', e);
    }
  }

  // Save settings to localStorage and Backend
  async function saveSettings() {
    try {
      localStorage.setItem('rexec_settings', JSON.stringify({
        theme,
        fontSize,
        cursorBlink,
        cursorStyle,
        scrollback,
        copyOnSelect,
      }));

      // Save profile if loaded
      if (profileLoaded && $auth.user) {
          const res = await auth.updateProfile({
              username: profileUsername,
              firstName: profileFirstName,
              lastName: profileLastName,
              allowedIPs: $auth.user.allowedIPs
          });
          if (!res.success) {
               toast.error(res.error || 'Failed to update profile');
               return;
          }
      }

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

  // Update lock timeout
  async function updateLockTimeout() {
    const result = await security.updateLockTimeout(lockTimeout);
    if (result.success) {
      toast.success('Lock timeout updated');
    } else {
      toast.error(result.error || 'Failed to update lock timeout');
    }
  }

  // Open passcode modal
  function openPasscodeModal(isChange: boolean) {
    isChangingPasscode = isChange;
    showPasscodeModal = true;
    newPasscode = '';
    confirmPasscode = '';
    currentPasscode = '';
    passcodeError = '';
  }

  // Close passcode modal
  function closePasscodeModal() {
    showPasscodeModal = false;
    newPasscode = '';
    confirmPasscode = '';
    currentPasscode = '';
    passcodeError = '';
  }

  // Set or change passcode
  async function handleSetPasscode() {
    passcodeError = '';

    if (!newPasscode.trim()) {
      passcodeError = 'Please enter a passcode';
      return;
    }

    if (newPasscode.length < 4) {
      passcodeError = 'Passcode must be at least 4 characters';
      return;
    }

    if (newPasscode !== confirmPasscode) {
      passcodeError = 'Passcodes do not match';
      return;
    }

    const result = await security.setPasscode(
      newPasscode,
      isChangingPasscode && $hasPasscode ? currentPasscode : undefined,
      lockTimeout
    );
    if (!result.success) {
      // Handle case where UI thought no passcode existed but server says otherwise
      if (result.error && (result.error.includes('current_passcode is required') || result.error.includes('already exists'))) {
        await security.refreshFromServer();
        isChangingPasscode = true;
        passcodeError = 'A passcode is already set. Please enter your current passcode.';
        return;
      }

      passcodeError = result.error || 'Failed to set passcode';
      return;
    }

    closePasscodeModal();
    toast.success($hasPasscode ? 'Passcode updated' : 'Screen lock passcode set');
  }

  // Remove passcode
  async function handleRemovePasscode() {
    if (!$hasPasscode) return;
    const result = await security.removePasscode(currentPasscode);
    if (!result.success) {
      passcodeError = result.error || 'Passcode is incorrect';
      return;
    }

    closePasscodeModal();
    toast.success('Screen lock disabled');
  }

  // Keep lockTimeout in sync with server setting
  $: if ($hasPasscode && $security.lockAfterMinutes !== lockTimeout) {
    lockTimeout = $security.lockAfterMinutes;
  }

  // MFA Functions
  async function openMFAModal(mode: 'enable' | 'disable') {
    showMFAModal = true;
    mfaError = '';
    mfaCode = '';
    if (mode === 'enable') {
      mfaStep = 'intro';
    } else {
      mfaStep = 'disable';
    }
  }

  async function startMFASetup() {
    mfaLoading = true;
    mfaError = '';
    try {
      const res = await fetch('/api/mfa/setup', {
        headers: { Authorization: `Bearer ${$auth.token}` }
      });
      if (!res.ok) throw new Error('Failed to start MFA setup');
      const data = await res.json();
      mfaSecret = data.secret;
      
      // Generate QR Code
      mfaQrDataUrl = await QRCode.toDataURL(data.otp_url);
      mfaStep = 'setup';
    } catch (e: any) {
      mfaError = e.message || 'Setup failed';
    } finally {
      mfaLoading = false;
    }
  }

  async function verifyMFA() {
    if (!mfaCode) return;
    mfaLoading = true;
    mfaError = '';
    try {
      const res = await fetch('/api/mfa/verify', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${$auth.token}` 
        },
        body: JSON.stringify({ secret: mfaSecret, code: mfaCode })
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || 'Verification failed');
      }
      
      toast.success('MFA Enabled Successfully');
      auth.fetchProfile(); // Refresh profile to update mfaEnabled status
      showMFAModal = false;
    } catch (e: any) {
      mfaError = e.message || 'Verification failed';
    } finally {
      mfaLoading = false;
    }
  }

  async function disableMFA() {
    if (!mfaCode) return;
    mfaLoading = true;
    mfaError = '';
    try {
      const res = await fetch('/api/mfa/disable', {
        method: 'POST',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${$auth.token}` 
        },
        body: JSON.stringify({ code: mfaCode })
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || 'Disable failed');
      }
      
      toast.success('MFA Disabled');
      auth.fetchProfile();
      showMFAModal = false;
    } catch (e: any) {
      mfaError = e.message || 'Disable failed';
    } finally {
      mfaLoading = false;
    }
  }

  // IP Whitelist Functions
  function openIPModal() {
    allowedIPs = [...($auth.user?.allowedIPs || [])];
    showIPModal = true;
    ipError = '';
    newIP = '';
  }

  function addIP() {
    if (!newIP.trim()) return;
    // Basic validation
    if (!/^[\d\.\/]+$/.test(newIP.trim())) {
      ipError = 'Invalid IP format';
      return;
    }
    if (allowedIPs.includes(newIP.trim())) {
        ipError = 'IP already in list';
        return;
    }
    allowedIPs = [...allowedIPs, newIP.trim()];
    newIP = '';
    ipError = '';
  }

  function removeIP(ip: string) {
    allowedIPs = allowedIPs.filter(i => i !== ip);
  }

  async function saveIPs() {
    ipLoading = true;
    ipError = '';
    try {
      // Need to send username as well since PUT /profile expects it
      const res = await fetch('/api/profile', {
        method: 'PUT',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${$auth.token}` 
        },
        body: JSON.stringify({ username: $auth.user?.username, allowed_ips: allowedIPs })
      });
      if (!res.ok) throw new Error('Failed to save IP whitelist');
      
      toast.success('IP Whitelist Updated');
      auth.fetchProfile();
      showIPModal = false;
    } catch (e: any) {
      ipError = e.message || 'Save failed';
    } finally {
      ipLoading = false;
    }
  }

  // Audit Logs Functions
  async function openAuditModal() {
    showAuditModal = true;
    auditLoading = true;
    auditError = '';
    try {
      const res = await fetch('/api/audit-logs?limit=50', {
        headers: { Authorization: `Bearer ${$auth.token}` }
      });
      if (!res.ok) throw new Error('Failed to fetch logs');
      auditLogs = await res.json();
    } catch (e: any) {
      auditError = e.message || 'Failed to load logs';
    } finally {
      auditLoading = false;
    }
  }

  // Sessions Functions
  async function openSessionsModal() {
    showSessionsModal = true;
    sessionsLoading = true;
    sessionsError = '';
    try {
      const res = await fetch('/api/sessions', {
        headers: { Authorization: `Bearer ${$auth.token}` }
      });
      if (!res.ok) throw new Error('Failed to load sessions');
      const data = await res.json();
      sessions = data.sessions || [];
    } catch (e: any) {
      sessionsError = e.message || 'Failed to load sessions';
    } finally {
      sessionsLoading = false;
    }
  }

  function closeSessionsModal() {
    showSessionsModal = false;
    sessionsError = '';
    sessions = [];
  }

  async function revokeSession(sessionId: string, isCurrent: boolean) {
    if (!confirm('Revoke this session?')) return;
    try {
      const res = await fetch(`/api/sessions/${sessionId}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${$auth.token}`
        }
      });
      if (!res.ok) {
        const d = await res.json().catch(() => ({}));
        throw new Error(d.error || 'Failed to revoke session');
      }
      sessions = sessions.map(s => s.id === sessionId ? { ...s, revoked_at: new Date().toISOString(), revoked_reason: 'revoked_by_user' } : s);
      toast.success('Session revoked');
      if (isCurrent) {
        auth.logout();
      }
    } catch (e: any) {
      toast.error(e.message || 'Failed to revoke session');
    }
  }

  async function revokeOtherSessions() {
    if (!confirm('Revoke all other sessions?')) return;
    try {
      const res = await fetch('/api/sessions/revoke-others', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${$auth.token}`
        }
      });
      if (!res.ok) throw new Error('Failed to revoke sessions');
      // Refresh list
      await openSessionsModal();
      toast.success('Other sessions revoked');
    } catch (e: any) {
      toast.error(e.message || 'Failed to revoke sessions');
    }
  }

  // Load on mount
  loadSettings();
</script>

<div class="settings">
  <div class="settings-header">
    <button class="back-btn" onclick={() => dispatch('back')}>
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
          <label for="username">Username</label>
          <span class="setting-description">Unique identifier for your account</span>
        </div>
        <div class="setting-value">
          <input
            type="text"
            id="username"
            bind:value={profileUsername}
            class="input-full"
            style="max-width: 200px;"
          />
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label for="firstname">Full Name</label>
          <span class="setting-description">Your first and last name</span>
        </div>
        <div class="setting-value profile-names">
          <input
            type="text"
            id="firstname"
            bind:value={profileFirstName}
            placeholder="First Name"
            class="input-full"
          />
          <input
            type="text"
            id="lastname"
            bind:value={profileLastName}
            placeholder="Last Name"
            class="input-full"
          />
        </div>
      </div>

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
            <label><StatusIcon status="warning" size={16} /> Guest Account</label>
            <span class="setting-description">
              Your terminals are temporary. Sign in to save your data.
            </span>
          </div>
          <div class="setting-value">
            <button class="btn btn-primary btn-sm" onclick={() => auth.getOAuthUrl().then(url => url && (window.location.href = url))}>
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
          <select 
            id="theme" 
            value={$themeStore} 
            onchange={(e) => themeStore.setTheme((e.target as HTMLSelectElement).value as 'dark' | 'light')} 
            class="select-sm"
          >
            <option value="dark">Dark</option>
            <option value="light">Light</option>
          </select>
        </div>
      </div>
    </section>

    <!-- Security Section -->
    <section class="settings-section">
      <h2>Security</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label>Screen Lock</label>
          <span class="setting-description">
            {#if $hasPasscode}
              Screen lock is enabled
            {:else}
              Protect your session when away
            {/if}
          </span>
        </div>
        <div class="setting-value">
          {#if $hasPasscode}
            <button class="btn btn-secondary btn-sm" onclick={() => openPasscodeModal(true)}>
              Change Passcode
            </button>
          {:else}
            <button class="btn btn-primary btn-sm" onclick={() => openPasscodeModal(false)}>
              Set Passcode
            </button>
          {/if}
        </div>
      </div>

      {#if $hasPasscode}
        <div class="setting-item">
          <div class="setting-info">
            <label for="lock-timeout">Lock After</label>
            <span class="setting-description">Auto-lock after inactivity</span>
          </div>
          <div class="setting-value">
            <select id="lock-timeout" bind:value={lockTimeout} onchange={updateLockTimeout} class="select-sm">
              <option value={1}>1 minute</option>
              <option value={2}>2 minutes</option>
              <option value={5}>5 minutes</option>
              <option value={10}>10 minutes</option>
              <option value={15}>15 minutes</option>
              <option value={30}>30 minutes</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Disable Screen Lock</label>
            <span class="setting-description">Remove passcode protection</span>
          </div>
          <div class="setting-value">
            <button class="btn btn-danger btn-sm" onclick={() => { isChangingPasscode = false; showPasscodeModal = true; }}>
              Disable
            </button>
          </div>
        </div>
      {/if}

      <div class="setting-item">
        <div class="setting-info">
          <label>Two-Factor Authentication</label>
          <span class="setting-description">
            {#if $auth.user?.mfaEnabled}
              MFA is enabled
            {:else}
              Add an extra layer of security
            {/if}
          </span>
        </div>
        <div class="setting-value">
          {#if $auth.user?.mfaEnabled}
            <button class="btn btn-danger btn-sm" onclick={() => openMFAModal('disable')}>
              Disable MFA
            </button>
          {:else}
            <button class="btn btn-primary btn-sm" onclick={() => openMFAModal('enable')}>
              Enable MFA
            </button>
          {/if}
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label>IP Whitelist</label>
          <span class="setting-description">
            Restrict access to your account and API endpoints from specific IP addresses. 
            Leave empty to allow access from any IP. All changes are logged.
          </span>
        </div>
        <div class="setting-value">
          <button class="btn btn-secondary btn-sm" onclick={openIPModal}>
            Manage IPs
          </button>
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label>Audit Logs</label>
          <span class="setting-description">View security events</span>
        </div>
        <div class="setting-value">
          <button class="btn btn-secondary btn-sm" onclick={openAuditModal}>
            View Logs
          </button>
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label>Active Sessions</label>
          <span class="setting-description">Manage devices logged into your account</span>
        </div>
        <div class="setting-value">
          <button class="btn btn-secondary btn-sm" onclick={openSessionsModal}>
            Manage Sessions
          </button>
        </div>
      </div>
    </section>

    <!-- Agents Section -->
    {#if !$isGuest}
    <section class="settings-section">
      <h2>Agents</h2>
      <p class="section-description">
        Connect your own servers, VMs, or local machines to rexec. Install the agent on any Linux/macOS system to access it from anywhere.
      </p>

      <div class="agents-header">
        <button class="btn btn-primary btn-sm" onclick={() => showAgentModal = true}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          Add Agent
        </button>
      </div>

      {#if $agents.loading}
        <div class="agents-loading">Loading agents...</div>
      {:else if $agents.agents.length === 0}
        <div class="agents-empty-state">
          <div class="empty-icon">
            <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="1.5">
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
              <line x1="8" y1="21" x2="16" y2="21"></line>
              <line x1="12" y1="17" x2="12" y2="21"></line>
            </svg>
          </div>
          <h3>Connect Your Own Machine</h3>
          <p class="empty-desc">Bring your own server, VM, or local machine to rexec. Click "Register Agent" to get a personalized install command with your API token.</p>

          <div class="empty-actions">
            <button class="btn btn-primary" onclick={() => showAgentModal = true}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19"></line>
                <line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
              Register Agent
            </button>
            <a href="/docs/agent" class="btn btn-secondary">View Documentation</a>
          </div>
        </div>
      {:else}
        <div class="agents-list">
          {#each $agents.agents as agent}
            <div class="agent-card" class:agent-online={agent.status === 'online'}>
              <div class="agent-header">
                <div class="agent-status">
                  <span 
                    class="status-dot" 
                    class:status-dot-pulse={agent.status === 'online'}
                    style="background: {getStatusColor(agent.status)}"
                  ></span>
                  <span class="status-text">{agent.status}</span>
                </div>
                <div class="agent-actions">
                  {#if agent.status === 'online'}
                    <button 
                      class="btn btn-sm btn-primary" 
                      title="Connect to terminal"
                      onclick={() => handleConnectAgent(agent)}
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="4 17 10 11 4 5"></polyline>
                        <line x1="12" y1="19" x2="20" y2="19"></line>
                      </svg>
                      Connect
                    </button>
                  {/if}
                  <button class="btn btn-icon btn-sm btn-danger-subtle" title="Delete agent" onclick={() => handleDeleteAgent(agent.id)}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="3 6 5 6 21 6"></polyline>
                      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                    </svg>
                  </button>
                </div>
              </div>
              <div class="agent-body">
                <span class="agent-name">{agent.name}</span>
                {#if agent.description}
                  <span class="agent-desc">{agent.description}</span>
                {/if}
              </div>
              <div class="agent-details">
                <div class="agent-detail-item">
                  <span class="detail-label">Platform</span>
                  <span class="detail-value">{agent.os || 'Unknown'}/{agent.arch || 'Unknown'}</span>
                </div>
                <div class="agent-detail-item">
                  <span class="detail-label">Shell</span>
                  <span class="detail-value">{agent.shell || '/bin/bash'}</span>
                </div>
                {#if agent.connected_at}
                  <div class="agent-detail-item">
                    <span class="detail-label">Connected</span>
                    <span class="detail-value">{new Date(agent.connected_at).toLocaleString()}</span>
                  </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>

        <div class="agents-footer">
          <div class="install-inline">
            <span>Install on another machine:</span>
            <code>curl -sSL {installUrl}/install-agent.sh | sudo bash</code>
            <button
              class="btn btn-sm btn-icon copy-btn"
              title="Copy"
              onclick={() => {
                navigator.clipboard.writeText(`curl -sSL ${installUrl}/install-agent.sh | sudo bash`);
                toast.success('Copied!');
              }}
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
              </svg>
            </button>
          </div>
        </div>
      {/if}
    </section>
    {/if}

    <!-- Actions -->
    <div class="settings-actions">
      <button class="btn btn-secondary" onclick={resetSettings}>
        Reset to Defaults
      </button>
      <button class="btn btn-primary" onclick={saveSettings}>
        Save Settings
      </button>
    </div>
  </div>
</div>

<!-- Passcode Modal -->
{#if showPasscodeModal}
  <div class="modal-overlay" onclick={(e) => e.target === e.currentTarget && closePasscodeModal()}>
    <div class="modal">
      <div class="modal-header">
        <h3>
          {#if $hasPasscode && !isChangingPasscode}
            Disable Screen Lock
          {:else if $hasPasscode}
            Change Passcode
          {:else}
            Set Screen Lock Passcode
          {/if}
        </h3>
        <button class="modal-close" onclick={closePasscodeModal}>×</button>
      </div>

      <div class="modal-body">
        {#if $hasPasscode}
          <div class="form-group">
            <label for="current-passcode">Current Passcode</label>
            <input
              type="password"
              id="current-passcode"
              bind:value={currentPasscode}
              placeholder="Enter current passcode"
              class="input-full"
            />
          </div>
        {/if}

        {#if !$hasPasscode || isChangingPasscode}
          <div class="form-group">
            <label for="new-passcode">New Passcode</label>
            <input
              type="password"
              id="new-passcode"
              bind:value={newPasscode}
              placeholder="Enter new passcode (min 4 characters)"
              class="input-full"
            />
          </div>

          <div class="form-group">
            <label for="confirm-passcode">Confirm Passcode</label>
            <input
              type="password"
              id="confirm-passcode"
              bind:value={confirmPasscode}
              placeholder="Confirm new passcode"
              class="input-full"
            />
          </div>
        {/if}

        {#if passcodeError}
          <p class="error-text">{passcodeError}</p>
        {/if}
      </div>

      <div class="modal-footer">
        <button class="btn btn-secondary" onclick={closePasscodeModal}>
          Cancel
        </button>
        {#if $hasPasscode && !isChangingPasscode}
          <button class="btn btn-danger" onclick={handleRemovePasscode}>
            Disable Lock
          </button>
        {:else}
          <button class="btn btn-primary" onclick={handleSetPasscode}>
            {$hasPasscode ? 'Update Passcode' : 'Set Passcode'}
          </button>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Agent Modal -->
{#if showAgentModal}
  <div class="modal-overlay" onclick={(e) => e.target === e.currentTarget && closeAgentModal()}>
    <div class="modal modal-lg">
      <div class="modal-header">
        <h3>{showInstallScript ? 'Install Agent' : 'Add New Agent'}</h3>
        <button class="modal-close" onclick={closeAgentModal}>×</button>
      </div>

      <div class="modal-body">
        {#if showInstallScript && createdAgent}
          <div class="install-success">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--success)" stroke-width="2">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
              <polyline points="22 4 12 14.01 9 11.01"></polyline>
            </svg>
            <h4>Agent "{createdAgent.name}" Registered!</h4>
            <p>Run this command on your server to install the agent:</p>
          </div>

          <div class="install-script-box">
            <code>{agents.getInstallScript(createdAgent.id)}</code>
            <button class="btn btn-sm copy-btn" onclick={copyInstallScript}>
              {copiedScript ? 'Copied!' : 'Copy'}
            </button>
          </div>
          
          {#if agents.getAgentToken(createdAgent.id)}
            <div class="token-notice token-notice-success">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
              </svg>
              <span>This install command includes a permanent API token that never expires.</span>
            </div>
          {:else}
            <div class="token-notice token-notice-warning">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
                <line x1="12" y1="9" x2="12" y2="13"></line>
                <line x1="12" y1="17" x2="12.01" y2="17"></line>
              </svg>
              <span>Using session token (expires in 24h). For permanent access, <a href="/account/api">generate an API token</a>.</span>
            </div>
          {/if}

          <div class="install-notes">
            <h5>Requirements:</h5>
            <ul>
              <li>Linux or macOS</li>
              <li>curl and bash installed</li>
              <li>Root/sudo access (for systemd service)</li>
            </ul>
          </div>
        {:else}
          <div class="form-group">
            <label for="agent-name">Agent Name</label>
            <input
              type="text"
              id="agent-name"
              bind:value={newAgentName}
              placeholder="e.g., production-server-1"
              class="input-full"
            />
          </div>

          <div class="form-group">
            <label for="agent-desc">Description (optional)</label>
            <input
              type="text"
              id="agent-desc"
              bind:value={newAgentDescription}
              placeholder="e.g., Main production web server"
              class="input-full"
            />
          </div>
        {/if}
      </div>

      <div class="modal-footer">
        {#if showInstallScript}
          <button class="btn btn-primary" onclick={closeAgentModal}>
            Done
          </button>
        {:else}
          <button class="btn btn-secondary" onclick={closeAgentModal}>
            Cancel
          </button>
          <button class="btn btn-primary" onclick={handleCreateAgent}>
            Create Agent
          </button>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- MFA Modal -->
{#if showMFAModal}
  <div class="modal-overlay" onclick={(e) => e.target === e.currentTarget && (showMFAModal = false)}>
    <div class="modal">
      <div class="modal-header">
        <h3>Two-Factor Authentication</h3>
        <button class="modal-close" onclick={() => showMFAModal = false}>×</button>
      </div>
      <div class="modal-body">
        {#if mfaStep === 'intro'}
          <p class="modal-text">Two-factor authentication adds an extra layer of security to your account. You'll need an authenticator app like Google Authenticator or Authy.</p>
        {:else if mfaStep === 'setup'}
          <div class="qr-container">
            {#if mfaQrDataUrl}
              <img src={mfaQrDataUrl} alt="MFA QR Code" class="qr-code" />
            {:else}
              <div class="loading-spinner">Loading QR...</div>
            {/if}
          </div>
          <p class="secret-text">Secret: <code>{mfaSecret}</code></p>
          <div class="form-group">
            <label>Verification Code</label>
            <input type="text" bind:value={mfaCode} placeholder="Enter 6-digit code" class="input-full" maxlength="6" />
          </div>
        {:else if mfaStep === 'disable'}
          <p class="modal-text">Enter a code from your authenticator app to disable MFA.</p>
          <div class="form-group">
            <label>Verification Code</label>
            <input type="text" bind:value={mfaCode} placeholder="Enter 6-digit code" class="input-full" maxlength="6" />
          </div>
        {/if}
        {#if mfaError}
          <p class="error-text">{mfaError}</p>
        {/if}
      </div>
      <div class="modal-footer">
        {#if mfaStep === 'intro'}
          <button class="btn btn-secondary" onclick={() => showMFAModal = false}>Cancel</button>
          <button class="btn btn-primary" onclick={startMFASetup} disabled={mfaLoading}>Start Setup</button>
        {:else if mfaStep === 'setup'}
          <button class="btn btn-secondary" onclick={() => showMFAModal = false}>Cancel</button>
          <button class="btn btn-primary" onclick={verifyMFA} disabled={mfaLoading || !mfaCode}>Verify & Enable</button>
        {:else if mfaStep === 'disable'}
          <button class="btn btn-secondary" onclick={() => showMFAModal = false}>Cancel</button>
          <button class="btn btn-danger" onclick={disableMFA} disabled={mfaLoading || !mfaCode}>Disable MFA</button>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- IP Whitelist Modal -->
{#if showIPModal}
  <div class="modal-overlay" onclick={(e) => e.target === e.currentTarget && (showIPModal = false)}>
    <div class="modal">
      <div class="modal-header">
        <h3>IP Whitelist</h3>
        <button class="modal-close" onclick={() => showIPModal = false}>×</button>
      </div>
      <div class="modal-body">
        <p class="modal-text">Only allow access from these IP addresses. Leave empty to allow all.</p>
        
        <div class="ip-list">
          {#each allowedIPs as ip}
            <div class="ip-item">
              <span>{ip}</span>
              <button class="btn-icon btn-sm" onclick={() => removeIP(ip)}>×</button>
            </div>
          {/each}
          {#if allowedIPs.length === 0}
            <p class="empty-text">No IPs whitelisted (All allowed)</p>
          {/if}
        </div>

        <div class="add-ip-form">
          <input type="text" bind:value={newIP} placeholder="e.g. 192.168.1.1 or 10.0.0.0/24" class="input-full" />
          <button class="btn btn-secondary btn-sm" onclick={addIP}>Add</button>
        </div>

        {#if ipError}
          <p class="error-text">{ipError}</p>
        {/if}
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" onclick={() => showIPModal = false}>Cancel</button>
        <button class="btn btn-primary" onclick={saveIPs} disabled={ipLoading}>Save Changes</button>
      </div>
    </div>
  </div>
{/if}

<!-- Audit Logs Modal -->
{#if showAuditModal}
  <div class="modal-overlay" onclick={(e) => e.target === e.currentTarget && (showAuditModal = false)}>
    <div class="modal modal-lg">
      <div class="modal-header">
        <h3>Audit Logs</h3>
        <button class="modal-close" onclick={() => showAuditModal = false}>×</button>
      </div>
      <div class="modal-body">
        {#if auditLoading}
          <div class="loading-text">Loading logs...</div>
        {:else if auditLogs.length === 0}
          <div class="empty-text">No logs found</div>
        {:else}
          <div class="logs-table-wrapper">
            <table class="logs-table">
              <thead>
                <tr>
                  <th>Action</th>
                  <th>IP Address</th>
                  <th>Date</th>
                </tr>
              </thead>
              <tbody>
                {#each auditLogs as log}
                  <tr>
                    <td>{log.action}</td>
                    <td>{log.ip_address}</td>
                    <td>{new Date(log.created_at).toLocaleString()}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
        {#if auditError}
          <p class="error-text">{auditError}</p>
        {/if}
      </div>
      <div class="modal-footer">
        <button class="btn btn-primary" onclick={() => showAuditModal = false}>Close</button>
      </div>
    </div>
  </div>
{/if}

<!-- Sessions Modal -->
{#if showSessionsModal}
  <div class="modal-overlay" onclick={(e) => e.target === e.currentTarget && closeSessionsModal()}>
    <div class="modal modal-lg">
      <div class="modal-header">
        <h3>Active Sessions</h3>
        <button class="modal-close" onclick={closeSessionsModal}>×</button>
      </div>
      <div class="modal-body">
        {#if sessionsLoading}
          <div class="loading-text">Loading sessions...</div>
        {:else if sessions.length === 0}
          <div class="empty-text">No sessions found</div>
        {:else}
          <div class="sessions-list">
            {#each sessions as s (s.id)}
              <div class="session-row" class:current={s.is_current}>
                <div class="session-meta">
                  <div class="session-ua">{s.user_agent || 'Unknown device'}</div>
                  <div class="session-details">
                    <span>{s.ip_address || 'Unknown IP'}</span>
                    <span>•</span>
                    <span>Started {new Date(s.created_at).toLocaleString()}</span>
                    <span>•</span>
                    <span>Last active {new Date(s.last_seen_at).toLocaleString()}</span>
                  </div>
                  {#if s.revoked_at}
                    <div class="session-revoked">
                      Revoked {new Date(s.revoked_at).toLocaleString()}
                    </div>
                  {/if}
                  {#if s.is_current}
                    <div class="session-current-badge">Current session</div>
                  {/if}
                </div>
                <div class="session-actions">
                  {#if !s.revoked_at}
                    <button
                      class="btn btn-danger btn-sm"
                      onclick={() => revokeSession(s.id, s.is_current)}
                    >
                      Revoke
                    </button>
                  {:else}
                    <button class="btn btn-secondary btn-sm" disabled>
                      Revoked
                    </button>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
        {#if sessionsError}
          <p class="error-text">{sessionsError}</p>
        {/if}
      </div>
      <div class="modal-footer">
        <button
          class="btn btn-secondary"
          onclick={revokeOtherSessions}
          disabled={sessionsLoading || sessions.length === 0}
        >
          Revoke Others
        </button>
        <button class="btn btn-primary" onclick={closeSessionsModal}>
          Close
        </button>
      </div>
    </div>
  </div>
{/if}

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

  /* Modal Styles */
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: var(--overlay-bg);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
    padding: 16px;
  }

  .modal {
    background: var(--bg-card);
    border: 1px solid var(--border);
    max-width: 420px;
    width: 100%;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border);
  }

  .modal-header h3 {
    margin: 0;
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 1px;
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
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding: 16px 20px;
    border-top: 1px solid var(--border);
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-group label {
    display: block;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }

  .input-full {
    width: 100%;
    padding: 12px 14px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 14px;
  }

  .input-full:focus {
    outline: none;
    border-color: var(--accent);
  }

  .error-text {
    color: var(--danger);
    font-size: 13px;
    margin: 0;
    padding: 8px 12px;
    background: rgba(255, 77, 77, 0.1);
    border: 1px solid rgba(255, 77, 77, 0.3);
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

  /* Agents Section Styles */
  .section-description {
    font-size: 12px;
    color: var(--text-muted);
    margin: 0 0 16px;
    line-height: 1.5;
  }

  .agents-header {
    margin-bottom: 16px;
  }

  .agents-loading {
    text-align: center;
    padding: 24px;
    color: var(--text-muted);
    font-size: 13px;
  }

  .agents-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 32px;
    text-align: center;
    background: var(--bg-secondary);
    border: 1px dashed var(--border);
  }

  .agents-empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 40px 24px;
    text-align: center;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
  }

  .agents-empty-state .empty-icon {
    margin-bottom: 16px;
    opacity: 0.8;
  }

  .agents-empty-state h3 {
    margin: 0 0 8px;
    font-size: 18px;
    font-weight: 600;
    color: var(--text);
  }

  .agents-empty-state .empty-desc {
    margin: 0 0 24px;
    font-size: 13px;
    color: var(--text-muted);
    max-width: 400px;
    line-height: 1.5;
  }

  .agents-empty-state .install-box {
    width: 100%;
    max-width: 500px;
    margin-bottom: 24px;
  }

  .agents-empty-state .install-command-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    border-radius: 8px;
  }

  .agents-empty-state .install-cmd {
    flex: 1;
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--accent);
    word-break: break-all;
    text-align: left;
  }

  .agents-empty-state .copy-btn {
    flex-shrink: 0;
    padding: 6px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.2s;
  }

  .agents-empty-state .copy-btn:hover {
    background: var(--accent);
    color: white;
    border-color: var(--accent);
  }

  .agents-empty-state .empty-actions {
    display: flex;
    gap: 12px;
  }

  .agents-footer {
    margin-top: 16px;
    padding: 12px 16px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    border-radius: 8px;
  }

  .agents-footer .install-inline {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .agents-footer .install-inline span {
    font-size: 12px;
    color: var(--text-muted);
  }

  .agents-footer .install-inline code {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--accent);
    background: var(--bg-secondary);
    padding: 4px 8px;
    border-radius: 4px;
  }

  .agents-footer .copy-btn {
    padding: 4px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 4px;
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.2s;
  }

  .agents-footer .copy-btn:hover {
    background: var(--accent);
    color: white;
    border-color: var(--accent);
  }

  .agents-empty p {
    margin: 16px 0 4px;
    font-size: 14px;
    color: var(--text-secondary);
  }

  .agents-empty-hint {
    font-size: 12px;
    color: var(--text-muted);
  }

  .agents-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 20px;
  }

  .agent-card {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    background: var(--bg-secondary);
    border: 1px solid #00ff41;
    border-radius: 8px;
    transition: all 0.3s;
    box-shadow: 0 0 10px rgba(0, 255, 65, 0.1);
    font-family: 'JetBrainsMono Nerd Font', 'FiraCode Nerd Font', 'Hack Nerd Font', monospace;
  }

  .agent-card:hover {
    border-color: #00ff41;
    box-shadow: 0 0 20px rgba(0, 255, 65, 0.3), 0 0 30px rgba(0, 255, 65, 0.1);
    transform: translateY(-2px);
  }

  .agent-card.agent-online {
    border-color: #00ff41;
    box-shadow: 0 0 15px rgba(0, 255, 65, 0.3), 0 0 25px rgba(0, 255, 65, 0.15);
  }

  .agent-card.agent-online:hover {
    border-color: #00ff41;
    box-shadow: 0 0 25px rgba(0, 255, 65, 0.4), 0 0 40px rgba(0, 255, 65, 0.2);
  }

  .agent-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .agent-status {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .status-dot-pulse {
    animation: pulse-green 2s infinite;
  }

  @keyframes pulse-green {
    0% {
      box-shadow: 0 0 0 0 rgba(0, 255, 65, 0.7);
    }
    70% {
      box-shadow: 0 0 0 6px rgba(0, 255, 65, 0);
    }
    100% {
      box-shadow: 0 0 0 0 rgba(0, 255, 65, 0);
    }
  }

  .status-text {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: #00ff41;
    font-weight: 600;
  }

  .agent-body {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .agent-name {
    font-size: 14px;
    font-weight: 600;
    color: #00ff41;
    text-shadow: 0 0 10px rgba(0, 255, 65, 0.3);
  }

  .agent-desc {
    font-size: 12px;
    color: #a8dadc;
  }

  .agent-details {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: 8px;
    padding-top: 12px;
    border-top: 1px solid rgba(0, 255, 65, 0.2);
  }

  .agent-detail-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .detail-label {
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: rgba(0, 255, 65, 0.7);
  }

  .detail-value {
    font-size: 12px;
    color: #00ff41;
    font-family: 'JetBrainsMono Nerd Font', 'FiraCode Nerd Font', 'Hack Nerd Font', monospace;
  }

  .agent-actions {
    display: flex;
    gap: 8px;
  }

  .btn-danger-subtle {
    color: var(--text-muted);
  }

  .btn-danger-subtle:hover {
    color: var(--red, #ff6b6b);
    border-color: var(--red, #ff6b6b);
  }

  .btn-icon {
    padding: 6px;
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-icon:hover {
    border-color: var(--danger);
    color: var(--danger);
    background: rgba(255, 77, 77, 0.1);
  }

  .agents-docs {
    margin-top: 20px;
    padding: 16px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
  }

  .agents-docs h4 {
    margin: 0 0 8px;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--accent);
  }

  .agents-docs p {
    margin: 0 0 8px;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .install-command-wrapper {
    position: relative;
    margin-bottom: 12px;
  }

  .install-command {
    display: block;
    padding: 12px;
    padding-right: 60px;
    background: var(--bg);
    border: 1px solid var(--border);
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--accent);
    word-break: break-all;
    margin-bottom: 0;
  }

  .copy-btn-inline {
    position: absolute;
    top: 50%;
    right: 8px;
    transform: translateY(-50%);
    padding: 4px 8px;
    font-size: 10px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
  }

  .docs-link a {
    color: var(--accent);
    text-decoration: none;
    font-size: 12px;
  }

  .docs-link a:hover {
    text-decoration: underline;
  }

  /* Agent Modal Styles */
  .modal-lg {
    max-width: 560px;
  }

  .install-success {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    margin-bottom: 20px;
  }

  .install-success h4 {
    margin: 12px 0 4px;
    font-size: 16px;
    color: var(--text);
  }

  .install-success p {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .install-script-box {
    position: relative;
    padding: 16px;
    padding-right: 70px;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    margin-bottom: 16px;
  }

  .install-script-box code {
    display: block;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--accent);
    word-break: break-all;
    line-height: 1.5;
  }

  .install-script-box .copy-btn {
    position: absolute;
    top: 50%;
    right: 12px;
    transform: translateY(-50%);
  }
  
  .token-notice {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 12px 14px;
    border-radius: 8px;
    font-size: 13px;
    margin-bottom: 16px;
  }
  
  .token-notice svg {
    flex-shrink: 0;
    margin-top: 1px;
  }
  
  .token-notice-success {
    background: rgba(34, 197, 94, 0.1);
    border: 1px solid rgba(34, 197, 94, 0.3);
    color: #22c55e;
  }
  
  .token-notice-warning {
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.3);
    color: #f59e0b;
  }
  
  .token-notice a {
    color: inherit;
    text-decoration: underline;
  }
  
  .token-notice a:hover {
    opacity: 0.8;
  }

  .install-notes {
    background: var(--bg-tertiary);
    padding: 16px;
    border: 1px solid var(--border);
  }

  .install-notes h5 {
    margin: 0 0 8px;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .install-notes ul {
    margin: 0;
    padding-left: 20px;
  }

  .install-notes li {
    font-size: 12px;
    color: var(--text-muted);
    margin-bottom: 4px;
  }

  /* MFA and Security Styles */
  .modal-text {
    font-size: 13px;
    color: var(--text-secondary);
    margin-bottom: 16px;
    line-height: 1.5;
  }

  /* Ensure no purple focus styles anywhere */
  .modal input:focus,
  .modal button:focus,
  .modal select:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 2px rgba(0, 255, 65, 0.2);
  }

  .modal button:focus {
    outline: none;
  }

  .qr-container {
    display: flex;
    justify-content: center;
    align-items: center;
    background: var(--bg-secondary);
    padding: 20px;
    border: 1px solid var(--border);
    border-radius: 0;
    margin-bottom: 16px;
    min-height: 200px;
  }

  .qr-code {
    max-width: 200px;
    height: auto;
  }

  .loading-spinner {
    color: var(--bg); /* Dark text on white bg */
    font-size: 14px;
  }

  .secret-text {
    font-size: 12px;
    color: var(--text-secondary);
    text-align: center;
    margin-bottom: 20px;
    word-break: break-all;
  }

  .secret-text code {
    background: var(--bg-tertiary);
    padding: 2px 6px;
    border-radius: 4px;
    font-family: var(--font-mono);
    color: var(--accent);
    user-select: all;
  }

  .ip-list {
    max-height: 200px;
    overflow-y: auto;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    border-radius: 6px;
    margin-bottom: 16px;
  }

  .ip-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border-muted);
    font-family: var(--font-mono);
    font-size: 13px;
  }

  .ip-item:last-child {
    border-bottom: none;
  }

  .empty-text {
    padding: 16px;
    text-align: center;
    color: var(--text-muted);
    font-size: 13px;
    font-style: italic;
  }

  .add-ip-form {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
  }

  .logs-table-wrapper {
    max-height: 400px;
    overflow-y: auto;
    border: 1px solid var(--border);
    border-radius: 6px;
  }

  .logs-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  .logs-table th, .logs-table td {
    padding: 10px 12px;
    text-align: left;
    border-bottom: 1px solid var(--border-muted);
  }

  .logs-table th {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    font-weight: 600;
    position: sticky;
    top: 0;
  }

  .logs-table tr:last-child td {
    border-bottom: none;
  }

  .loading-text {
    text-align: center;
    padding: 20px;
    color: var(--text-muted);
  }

  /* Sessions modal */
  .sessions-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .session-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 12px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    gap: 12px;
  }

  .session-row.current {
    border-color: var(--accent);
    box-shadow: var(--accent-glow);
  }

  .session-meta {
    flex: 1;
    min-width: 0;
  }

  .session-ua {
    font-size: 12px;
    color: var(--text);
    word-break: break-word;
  }

  .session-details {
    margin-top: 4px;
    font-size: 12px;
    color: var(--text-muted);
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .session-current-badge {
    margin-top: 6px;
    font-size: 11px;
    color: var(--accent);
  }

  .session-revoked {
    margin-top: 6px;
    font-size: 11px;
    color: var(--text-muted);
  }

  .session-actions {
    flex-shrink: 0;
  }

  .profile-names {
    display: flex;
    gap: 8px;
    width: 100%;
    max-width: 300px;
  }
</style>
