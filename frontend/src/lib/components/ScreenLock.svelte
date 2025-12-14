<script lang="ts">
    import { onMount, onDestroy } from "svelte";
    import { security, hasPasscode, isLocked } from "$stores/security";
    import { isAuthenticated } from "$stores/auth";
    import { toast } from "$stores/toast";
    import { get } from "svelte/store";

    let passcodeInput = "";
    let newPasscode = "";
    let confirmPasscode = "";
    let isSettingUp = false;
    let isVerifying = false;
    let error = "";
    let showSetupPrompt = false;
    let inactivityTimer: ReturnType<typeof setInterval> | null = null;
    let activityTimeout: ReturnType<typeof setTimeout> | null = null;
    
    // Cache current values to avoid get() in handlers
    let currentIsAuthenticated = false;
    let currentHasPasscode = false;
    
    // Subscriptions - initialized in onMount to avoid effect_orphan error
    let unsubAuth: (() => void) | null = null;
    let unsubPasscode: (() => void) | null = null;

    // Activity tracking
    const ACTIVITY_EVENTS = ["mousedown", "mousemove", "keydown", "scroll", "touchstart", "click"];
    const ACTIVITY_DEBOUNCE = 1000; // Debounce activity updates

    function handleActivity() {
        if (!currentIsAuthenticated) return;
        
        // Debounce activity updates
        if (activityTimeout) {
            clearTimeout(activityTimeout);
        }
        activityTimeout = setTimeout(() => {
            security.updateActivity();
        }, ACTIVITY_DEBOUNCE);
    }

    function checkInactivity() {
        if (!currentIsAuthenticated || !currentHasPasscode) return;
        
        if (security.checkInactivity()) {
            security.lock();
        }
    }

    // Handle visibility change (tab hidden/shown)
    function handleVisibilityChange() {
        if (!currentIsAuthenticated) return;
        
        if (document.hidden) {
            // User switched away
            // Optional: Could lock immediately or start shorter timer
        } else {
            // User came back - check if we should be locked
            if (currentHasPasscode && security.checkInactivity()) {
                security.lock();
            }
        }
    }

    onMount(() => {
        // Subscribe to stores inside onMount to avoid effect_orphan error in Svelte 5
        unsubAuth = isAuthenticated.subscribe(value => {
            currentIsAuthenticated = value;
            if (value) {
                security.refreshFromServer();
            }
        });
        
        unsubPasscode = hasPasscode.subscribe(value => {
            currentHasPasscode = value;
        });
        
        // Set up activity listeners
        ACTIVITY_EVENTS.forEach((event) => {
            document.addEventListener(event, handleActivity, { passive: true });
        });

        // Set up visibility change listener
        document.addEventListener("visibilitychange", handleVisibilityChange);

        // Check inactivity every 30 seconds
        inactivityTimer = setInterval(checkInactivity, 30000);

        // Check if should prompt for passcode setup (after 30 seconds)
        setTimeout(() => {
            if (currentIsAuthenticated && !currentHasPasscode) {
                const state = security.getState();
                if (!state.passcodeSetupPromptDismissed) {
                    showSetupPrompt = true;
                }
            }
        }, 30000);
    });

    onDestroy(() => {
        // Cleanup subscriptions
        if (unsubAuth) unsubAuth();
        if (unsubPasscode) unsubPasscode();
        
        ACTIVITY_EVENTS.forEach((event) => {
            document.removeEventListener(event, handleActivity);
        });
        document.removeEventListener("visibilitychange", handleVisibilityChange);
        
        if (inactivityTimer) {
            clearInterval(inactivityTimer);
        }
        if (activityTimeout) {
            clearTimeout(activityTimeout);
        }
    });

    async function handleUnlock() {
        if (!passcodeInput.trim()) {
            error = "Please enter your passcode";
            return;
        }

        isVerifying = true;
        error = "";

        const result = await security.unlockWithPasscode(passcodeInput);
        if (result.success) {
            passcodeInput = "";
            toast.success("Screen unlocked");
        } else {
            error = result.error || "Incorrect passcode";
            passcodeInput = "";
        }

        isVerifying = false;
    }

    async function handleSetupPasscode() {
        if (!newPasscode.trim()) {
            error = "Please enter a passcode";
            return;
        }

        if (newPasscode.length < 4) {
            error = "Passcode must be at least 4 characters";
            return;
        }

        if (newPasscode !== confirmPasscode) {
            error = "Passcodes don't match";
            return;
        }

        isVerifying = true;
        const result = await security.setPasscode(newPasscode);
        isVerifying = false;
        if (!result.success) {
            error = result.error || "Failed to set passcode";
            return;
        }

        newPasscode = "";
        confirmPasscode = "";
        isSettingUp = false;
        showSetupPrompt = false;
        
        toast.success("Screen lock passcode set! Your session is now protected.");
    }

    function dismissSetupPrompt() {
        showSetupPrompt = false;
        security.dismissSetupPrompt();
    }

    function startSetup() {
        showSetupPrompt = false;
        isSettingUp = true;
        error = "";
    }

    function cancelSetup() {
        isSettingUp = false;
        newPasscode = "";
        confirmPasscode = "";
        error = "";
    }

	    function handleKeydown(event: KeyboardEvent) {
	        if (event.key === "Enter") {
	            if ($isLocked) {
	                handleUnlock();
	            } else if (isSettingUp) {
	                handleSetupPasscode();
	            }
	        } else if (event.key === "Escape") {
	            if (isSettingUp) {
	                cancelSetup();
	            } else if (showSetupPrompt) {
	                dismissSetupPrompt();
	            }
	        }
	    }

	    // Clear sensitive inputs whenever the UI locks.
	    $: if ($isLocked) {
	        passcodeInput = "";
	        error = "";
	    }
	</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Lock Screen -->
{#if $isLocked && $isAuthenticated}
    <div class="lock-overlay">
        <div class="lock-container">
            <div class="lock-icon">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                    <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>
            </div>
            
            <h2 class="lock-title">Session Locked</h2>
            <p class="lock-subtitle">Enter your passcode to resume</p>

	            <div class="passcode-form">
	                <input
	                    type="password"
	                    class="passcode-input"
	                    bind:value={passcodeInput}
	                    placeholder="Enter passcode"
	                    disabled={isVerifying}
	                    autofocus
	                    name="screen-lock-passcode"
	                    autocomplete="new-password"
	                    autocorrect="off"
	                    autocapitalize="off"
	                    spellcheck="false"
	                />
                
                {#if error}
                    <p class="error-message">{error}</p>
                {/if}

                <button
                    class="btn btn-primary unlock-btn"
                    onclick={handleUnlock}
                    disabled={isVerifying || !passcodeInput.trim()}
                >
                    {isVerifying ? "Verifying..." : "Unlock"}
                </button>
            </div>

            <p class="lock-hint">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <path d="M12 16v-4"/>
                    <path d="M12 8h.01"/>
                </svg>
                Your terminal sessions are still running in the background
            </p>
        </div>
    </div>
{/if}

<!-- Passcode Setup Prompt -->
{#if showSetupPrompt && $isAuthenticated && !$isLocked}
    <div class="prompt-overlay" onclick={(e) => e.target === e.currentTarget && dismissSetupPrompt()}>
        <div class="prompt-container">
            <div class="prompt-header">
                <div class="prompt-icon">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                    </svg>
                </div>
                <h3>Protect Your Session</h3>
                <button class="prompt-close" onclick={dismissSetupPrompt}>×</button>
            </div>
            
            <div class="prompt-body">
                <p>Set a screen lock passcode to protect your terminals when you're away. Your active sessions will stay running.</p>
                
                <div class="prompt-features">
                    <div class="feature">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="2">
                            <polyline points="20 6 9 17 4 12"/>
                        </svg>
                        <span>Auto-locks after inactivity</span>
                    </div>
                    <div class="feature">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="2">
                            <polyline points="20 6 9 17 4 12"/>
                        </svg>
                        <span>Terminals keep running</span>
                    </div>
                    <div class="feature">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="2">
                            <polyline points="20 6 9 17 4 12"/>
                        </svg>
                        <span>Quick unlock with passcode</span>
                    </div>
                </div>
            </div>

            <div class="prompt-footer">
                <button class="btn btn-secondary" onclick={dismissSetupPrompt}>
                    Maybe Later
                </button>
                <button class="btn btn-primary" onclick={startSetup}>
                    Set Passcode
                </button>
            </div>
        </div>
    </div>
{/if}

<!-- Passcode Setup Modal -->
{#if isSettingUp && $isAuthenticated && !$isLocked}
    <div class="setup-overlay" onclick={(e) => e.target === e.currentTarget && cancelSetup()}>
        <div class="setup-container">
            <div class="setup-header">
                <h3>Set Screen Lock Passcode</h3>
                <button class="setup-close" onclick={cancelSetup}>×</button>
            </div>
            
            <div class="setup-body">
                <div class="form-group">
                    <label for="new-passcode">New Passcode</label>
                <input type="password" class="input" bind:value={newPasscode} placeholder="New Passcode" autocomplete="new-password" autocorrect="off" autocapitalize="off" spellcheck="false" />
                </div>

                <div class="form-group">
                    <label for="confirm-passcode">Confirm Passcode</label>
                <input type="password" class="input" bind:value={confirmPasscode} placeholder="Confirm Passcode" autocomplete="new-password" autocorrect="off" autocapitalize="off" spellcheck="false" />
                </div>

                {#if error}
                    <p class="error-message">{error}</p>
                {/if}

                <p class="setup-hint">
                    Your session will auto-lock after 5 minutes of inactivity. You can change this in Settings.
                </p>
            </div>

            <div class="setup-footer">
                <button class="btn btn-secondary" onclick={cancelSetup} disabled={isVerifying}>
                    Cancel
                </button>
                <button
                    class="btn btn-primary"
                    onclick={handleSetupPasscode}
                    disabled={isVerifying || !newPasscode.trim() || !confirmPasscode.trim()}
                >
                    {isVerifying ? "Setting up..." : "Set Passcode"}
                </button>
            </div>
        </div>
    </div>
{/if}

<style>
    /* Lock Screen Overlay */
    .lock-overlay {
        position: fixed;
        inset: 0;
        background: var(--overlay-bg);
        backdrop-filter: blur(20px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10000;
        animation: lockFadeIn 0.3s ease;
    }

    @keyframes lockFadeIn {
        from {
            opacity: 0;
            backdrop-filter: blur(0);
        }
        to {
            opacity: 1;
            backdrop-filter: blur(20px);
        }
    }

    .lock-container {
        text-align: center;
        max-width: 360px;
        width: 100%;
        padding: 40px 24px;
    }

    .lock-icon {
        color: var(--accent);
        margin-bottom: 24px;
        animation: lockPulse 2s ease-in-out infinite;
    }

    @keyframes lockPulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.6; }
    }

    .lock-title {
        font-size: 24px;
        font-weight: 600;
        color: var(--text);
        margin: 0 0 8px;
        text-transform: uppercase;
        letter-spacing: 2px;
    }

    .lock-subtitle {
        font-size: 14px;
        color: var(--text-muted);
        margin: 0 0 32px;
    }

    .passcode-form {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .passcode-input {
        width: 100%;
        padding: 16px 20px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 18px;
        text-align: center;
        letter-spacing: 4px;
        transition: border-color 0.2s;
    }

    .passcode-input:focus {
        outline: none;
        border-color: var(--accent);
    }

    .passcode-input:disabled {
        opacity: 0.6;
    }

    .unlock-btn {
        padding: 14px 24px;
        font-size: 14px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .error-message {
        color: var(--error);
        font-size: 13px;
        margin: 0;
        padding: 8px 12px;
        background: rgba(255, 77, 77, 0.1);
        border: 1px solid rgba(255, 77, 77, 0.3);
    }

    .lock-hint {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        font-size: 12px;
        color: var(--text-muted);
        margin-top: 32px;
        padding: 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
    }

    /* Prompt Overlay */
    .prompt-overlay {
        position: fixed;
        inset: 0;
        background: var(--overlay-light);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9000;
        animation: fadeIn 0.2s ease;
        padding: 16px;
    }

    @keyframes fadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
    }

    .prompt-container {
        background: var(--bg-card);
        border: 1px solid var(--border);
        max-width: 420px;
        width: 100%;
        animation: slideUp 0.2s ease;
    }

    @keyframes slideUp {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .prompt-header {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
    }

    .prompt-icon {
        color: var(--accent);
    }

    .prompt-header h3 {
        flex: 1;
        margin: 0;
        font-size: 14px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .prompt-close {
        background: none;
        border: none;
        color: var(--text-muted);
        font-size: 24px;
        cursor: pointer;
        padding: 0;
        line-height: 1;
    }

    .prompt-close:hover {
        color: var(--text);
    }

    .prompt-body {
        padding: 20px;
    }

    .prompt-body p {
        font-size: 13px;
        color: var(--text-secondary);
        line-height: 1.6;
        margin: 0 0 16px;
    }

    .prompt-features {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .feature {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 13px;
        color: var(--text);
    }

    .prompt-footer {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        padding: 16px 20px;
        border-top: 1px solid var(--border);
    }

    /* Setup Modal */
    .setup-overlay {
        position: fixed;
        inset: 0;
        background: var(--overlay-bg);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9500;
        animation: fadeIn 0.2s ease;
        padding: 16px;
    }

    .setup-container {
        background: var(--bg-card);
        border: 1px solid var(--border);
        max-width: 400px;
        width: 100%;
        animation: slideUp 0.2s ease;
    }

    .setup-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 16px 20px;
        border-bottom: 1px solid var(--border);
    }

    .setup-header h3 {
        margin: 0;
        font-size: 14px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .setup-close {
        background: none;
        border: none;
        color: var(--text-muted);
        font-size: 24px;
        cursor: pointer;
        padding: 0;
        line-height: 1;
    }

    .setup-close:hover {
        color: var(--text);
    }

    .setup-body {
        padding: 20px;
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

    .setup-hint {
        font-size: 12px;
        color: var(--text-muted);
        margin: 16px 0 0;
        padding: 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
    }

    .setup-footer {
        display: flex;
        justify-content: flex-end;
        gap: 12px;
        padding: 16px 20px;
        border-top: 1px solid var(--border);
    }

    /* Responsive */
    @media (max-width: 480px) {
        .lock-container {
            padding: 32px 16px;
        }

        .lock-title {
            font-size: 20px;
        }

        .passcode-input {
            font-size: 16px;
            padding: 14px 16px;
        }
    }
</style>
