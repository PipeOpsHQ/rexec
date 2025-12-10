<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    
    const dispatch = createEventDispatcher();
    
    function goHome() {
        dispatch('home');
    }
    
    // Get the current path for display
    const currentPath = typeof window !== 'undefined' ? window.location.pathname : '/';
</script>

<svelte:head>
    <title>404 - Page Not Found | Rexec</title>
    <meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="not-found">
    <div class="content">
        <div class="terminal-window">
            <div class="terminal-header">
                <div class="terminal-dots">
                    <span class="dot red"></span>
                    <span class="dot yellow"></span>
                    <span class="dot green"></span>
                </div>
                <span class="terminal-title">rexec@error</span>
            </div>
            <div class="terminal-body">
                <div class="line">
                    <span class="prompt">$</span>
                    <span class="command">curl {currentPath}</span>
                </div>
                <div class="line error-line">
                    <span class="error-code">HTTP/1.1 404 Not Found</span>
                </div>
                <div class="line">
                    <span class="output">{"{"}</span>
                </div>
                <div class="line">
                    <span class="output">&nbsp;&nbsp;"error": "PAGE_NOT_FOUND",</span>
                </div>
                <div class="line">
                    <span class="output">&nbsp;&nbsp;"message": "The requested resource does not exist",</span>
                </div>
                <div class="line">
                    <span class="output">&nbsp;&nbsp;"path": "{currentPath}",</span>
                </div>
                <div class="line">
                    <span class="output">&nbsp;&nbsp;"suggestion": "Try navigating to the homepage"</span>
                </div>
                <div class="line">
                    <span class="output">{"}"}</span>
                </div>
                <div class="line">
                    <span class="prompt">$</span>
                    <span class="cursor">_</span>
                </div>
            </div>
        </div>
        
        <div class="error-info">
            <h1>404</h1>
            <h2>Page Not Found</h2>
            <p>The terminal session you're looking for doesn't exist or has been terminated.</p>
            
            <div class="actions">
                <button class="btn btn-primary" onclick={goHome}>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
                        <polyline points="9 22 9 12 15 12 15 22"/>
                    </svg>
                    Go to Dashboard
                </button>
                <a href="/" class="btn btn-secondary">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M19 12H5M12 19l-7-7 7-7"/>
                    </svg>
                    Back to Home
                </a>
            </div>
        </div>
        
        <div class="suggestions">
            <h3>Maybe you were looking for:</h3>
            <div class="suggestion-links">
                <a href="/ui/dashboard">Dashboard</a>
                <a href="/use-cases">Use Cases</a>
                <a href="/pricing">Pricing</a>
                <a href="/guides">Guides</a>
            </div>
        </div>
    </div>
</div>

<style>
    .not-found {
        min-height: calc(100vh - 60px);
        display: flex;
        align-items: center;
        justify-content: center;
        padding: 40px 20px;
    }
    
    .content {
        max-width: 600px;
        width: 100%;
        text-align: center;
    }
    
    .terminal-window {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        overflow: hidden;
        margin-bottom: 32px;
        text-align: left;
    }
    
    .terminal-header {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 14px;
        background: var(--bg-secondary);
        border-bottom: 1px solid var(--border);
    }
    
    .terminal-dots {
        display: flex;
        gap: 6px;
    }
    
    .dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
    }
    
    .dot.red { background: #ff5f56; }
    .dot.yellow { background: #ffbd2e; }
    .dot.green { background: #27c93f; }
    
    .terminal-title {
        font-size: 12px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }
    
    .terminal-body {
        padding: 16px;
        font-family: var(--font-mono);
        font-size: 13px;
        line-height: 1.6;
    }
    
    .line {
        display: flex;
        gap: 8px;
    }
    
    .prompt {
        color: var(--accent);
    }
    
    .command {
        color: var(--text);
    }
    
    .error-line {
        margin: 8px 0;
    }
    
    .error-code {
        color: #ff6b6b;
        font-weight: 600;
    }
    
    .output {
        color: var(--text-secondary);
    }
    
    .cursor {
        color: var(--accent);
        animation: blink 1s step-end infinite;
    }
    
    @keyframes blink {
        50% { opacity: 0; }
    }
    
    .error-info h1 {
        font-size: 72px;
        font-weight: 700;
        color: var(--accent);
        margin: 0;
        line-height: 1;
        font-family: var(--font-mono);
    }
    
    .error-info h2 {
        font-size: 24px;
        font-weight: 600;
        color: var(--text);
        margin: 8px 0 16px;
    }
    
    .error-info p {
        color: var(--text-secondary);
        font-size: 14px;
        margin: 0 0 24px;
    }
    
    .actions {
        display: flex;
        gap: 12px;
        justify-content: center;
        flex-wrap: wrap;
    }
    
    .btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 12px 20px;
        font-size: 14px;
        font-weight: 500;
        border-radius: 6px;
        border: 1px solid var(--border);
        cursor: pointer;
        text-decoration: none;
        transition: all 0.15s ease;
    }
    
    .btn-primary {
        background: var(--accent);
        color: var(--bg);
        border-color: var(--accent);
    }
    
    .btn-primary:hover {
        background: var(--accent-hover);
        transform: translateY(-1px);
    }
    
    .btn-secondary {
        background: transparent;
        color: var(--text);
    }
    
    .btn-secondary:hover {
        background: var(--bg-secondary);
        border-color: var(--accent);
    }
    
    .suggestions {
        margin-top: 48px;
        padding-top: 32px;
        border-top: 1px solid var(--border);
    }
    
    .suggestions h3 {
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 1px;
        color: var(--text-muted);
        margin: 0 0 16px;
    }
    
    .suggestion-links {
        display: flex;
        gap: 16px;
        justify-content: center;
        flex-wrap: wrap;
    }
    
    .suggestion-links a {
        color: var(--accent);
        text-decoration: none;
        font-size: 14px;
        padding: 6px 12px;
        border: 1px solid var(--border);
        transition: all 0.15s ease;
    }
    
    .suggestion-links a:hover {
        background: var(--bg-secondary);
        border-color: var(--accent);
    }
    
    @media (max-width: 480px) {
        .error-info h1 {
            font-size: 56px;
        }
        
        .terminal-body {
            font-size: 11px;
        }
        
        .actions {
            flex-direction: column;
        }
        
        .btn {
            width: 100%;
            justify-content: center;
        }
    }
</style>
