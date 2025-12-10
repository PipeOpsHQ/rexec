<script lang="ts">
    import { onMount } from "svelte";
    import { isAuthenticated, token } from "../stores/auth";
    import { login as pipeopsLogin } from "../api";

    let callback = "";
    let isLoggingIn = false;
    let error = "";

    onMount(() => {
        callback = localStorage.getItem("cli_callback") || "";
        
        // If already authenticated, redirect
        if ($isAuthenticated && callback && $token) {
            redirectWithToken($token);
        }
    });

    function redirectWithToken(authToken: string) {
        if (callback) {
            localStorage.removeItem("cli_callback");
            window.location.href = `${callback}?token=${encodeURIComponent(authToken)}`;
        }
    }

    async function handlePipeOpsLogin() {
        isLoggingIn = true;
        error = "";
        try {
            await pipeopsLogin();
            // After successful login, check if we need to redirect
            if (callback && $token) {
                redirectWithToken($token);
            }
        } catch (e: any) {
            error = e.message || "Login failed";
        } finally {
            isLoggingIn = false;
        }
    }

    // Watch for auth changes
    $: if ($isAuthenticated && callback && $token) {
        redirectWithToken($token);
    }
</script>

<div class="cli-login-container">
    <div class="cli-login-card">
        <div class="logo">
            <svg width="48" height="48" viewBox="0 0 40 40" fill="none">
                <rect width="40" height="40" rx="8" fill="url(#gradient)" />
                <path
                    d="M12 14L18 20L12 26M20 26H28"
                    stroke="white"
                    stroke-width="2.5"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                />
                <defs>
                    <linearGradient id="gradient" x1="0" y1="0" x2="40" y2="40">
                        <stop offset="0%" stop-color="#6366f1" />
                        <stop offset="100%" stop-color="#8b5cf6" />
                    </linearGradient>
                </defs>
            </svg>
        </div>

        <h1>Rexec CLI Login</h1>
        <p class="subtitle">Authorize the CLI to access your Rexec account</p>

        {#if error}
            <div class="error">{error}</div>
        {/if}

        <button 
            class="login-btn" 
            onclick={handlePipeOpsLogin}
            disabled={isLoggingIn}
        >
            {#if isLoggingIn}
                <span class="spinner"></span>
                Connecting...
            {:else}
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
                    <polyline points="10 17 15 12 10 7" />
                    <line x1="15" y1="12" x2="3" y2="12" />
                </svg>
                Login with PipeOps
            {/if}
        </button>

        <div class="info">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="16" x2="12" y2="12" />
                <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
            <span>After login, you'll be redirected back to your terminal</span>
        </div>

        <div class="footer">
            <a href="/">Cancel and go to dashboard</a>
        </div>
    </div>
</div>

<style>
    .cli-login-container {
        min-height: 100vh;
        display: flex;
        align-items: center;
        justify-content: center;
        background: linear-gradient(135deg, #0a0a0a 0%, #1a1a2e 50%, #0a0a0a 100%);
        padding: 20px;
    }

    .cli-login-card {
        background: rgba(255, 255, 255, 0.03);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 16px;
        padding: 48px;
        max-width: 420px;
        width: 100%;
        text-align: center;
        backdrop-filter: blur(10px);
    }

    .logo {
        margin-bottom: 24px;
    }

    h1 {
        font-size: 24px;
        font-weight: 600;
        color: #fff;
        margin: 0 0 8px;
    }

    .subtitle {
        color: #888;
        margin: 0 0 32px;
        font-size: 14px;
    }

    .error {
        background: rgba(239, 68, 68, 0.1);
        border: 1px solid rgba(239, 68, 68, 0.3);
        color: #ef4444;
        padding: 12px 16px;
        border-radius: 8px;
        margin-bottom: 20px;
        font-size: 14px;
    }

    .login-btn {
        width: 100%;
        padding: 14px 24px;
        background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
        color: white;
        border: none;
        border-radius: 10px;
        font-size: 16px;
        font-weight: 500;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 10px;
        transition: all 0.2s ease;
    }

    .login-btn:hover:not(:disabled) {
        transform: translateY(-2px);
        box-shadow: 0 8px 24px rgba(99, 102, 241, 0.3);
    }

    .login-btn:disabled {
        opacity: 0.7;
        cursor: not-allowed;
    }

    .spinner {
        width: 18px;
        height: 18px;
        border: 2px solid rgba(255, 255, 255, 0.3);
        border-top-color: white;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    .info {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        margin-top: 24px;
        color: #666;
        font-size: 13px;
    }

    .info svg {
        flex-shrink: 0;
    }

    .footer {
        margin-top: 32px;
        padding-top: 24px;
        border-top: 1px solid rgba(255, 255, 255, 0.1);
    }

    .footer a {
        color: #888;
        text-decoration: none;
        font-size: 14px;
        transition: color 0.2s;
    }

    .footer a:hover {
        color: #fff;
    }
</style>
