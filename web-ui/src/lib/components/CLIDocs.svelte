<script lang="ts">
    import StatusIcon from "./icons/StatusIcon.svelte";
    import PlatformIcon from "./icons/PlatformIcon.svelte";

    export let onback: (() => void) | undefined = undefined;

    let copiedCommand = "";

    function copyToClipboard(text: string, id: string) {
        navigator.clipboard.writeText(text);
        copiedCommand = id;
        setTimeout(() => { copiedCommand = ""; }, 2000);
    }

    function handleBack() {
        if (onback) onback();
    }
</script>

<div class="docs-page">
    <button class="back-btn" onclick={handleBack}>
        <span class="back-icon">←</span>
        <span>Back</span>
    </button>

    <div class="docs-content">
        <header class="docs-header">
            <div class="header-icon">
                <StatusIcon status="terminal" size={48} />
            </div>
            <h1>rexec CLI</h1>
            <p class="subtitle">Command-line interface for managing cloud terminal environments</p>
        </header>

        <section class="docs-section">
            <h2>What is the rexec CLI?</h2>
            <p>
                The Rexec CLI (`rexec`) provides full access to all Rexec features including terminal management, 
                snippets, and agent mode directly from your terminal.
            </p>
            <div class="feature-grid">
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Terminal Management</h4>
                    <p>Create, list, connect, and delete terminals</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Interactive TUI</h4>
                    <p>Full-featured terminal user interface</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Agent Mode</h4>
                    <p>Register your machine as a rexec node</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Snippets & Macros</h4>
                    <p>Run automation scripts on your terminals</p>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Installation</h2>
            
            <div class="install-method">
                <h3>
                    <StatusIcon status="ready" size={20} />
                    Direct Download (Linux/macOS)
                </h3>
                <div class="code-block">
                    <code>curl -fsSL https://rexec.pipeops.io/install-cli.sh | bash</code>
                    <button 
                        class="copy-btn" 
                        onclick={() => copyToClipboard('curl -fsSL https://rexec.pipeops.io/install-cli.sh | bash', 'cli-direct')}
                    >
                        {copiedCommand === 'cli-direct' ? 'Copied!' : 'Copy'}
                    </button>
                </div>
            </div>

            <div class="install-method">
                <h3>
                    <PlatformIcon platform="macos" size={20} />
                    macOS (Homebrew)
                </h3>
                <div class="code-block">
                    <code>brew tap rexec/tap<br/>brew install rexec</code>
                    <button 
                        class="copy-btn" 
                        onclick={() => copyToClipboard('brew tap rexec/tap && brew install rexec', 'brew')}
                    >
                        {copiedCommand === 'brew' ? 'Copied!' : 'Copy'}
                    </button>
                </div>
            </div>

            <div class="install-method">
                <h3>
                    <StatusIcon status="ready" size={20} />
                    Go Install
                </h3>
                <div class="code-block">
                    <code>go install github.com/rexec/rexec/cmd/rexec-cli@latest</code>
                    <button 
                        class="copy-btn" 
                        onclick={() => copyToClipboard('go install github.com/rexec/rexec/cmd/rexec-cli@latest', 'go-install')}
                    >
                        {copiedCommand === 'go-install' ? 'Copied!' : 'Copy'}
                    </button>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Quick Start</h2>
            <div class="cli-commands">
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec login</code>
                        <span class="command-desc">Login to Rexec</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec ls</code>
                        <span class="command-desc">List your terminals</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec create --name mydev</code>
                        <span class="command-desc">Create a new terminal</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec connect abc123</code>
                        <span class="command-desc">Connect to a terminal</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec -i</code>
                        <span class="command-desc">Launch interactive TUI</span>
                    </div>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Commands Reference</h2>
            
            <h3>Authentication</h3>
            <div class="cli-commands">
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec login [--token TOKEN]</code>
                        <span class="command-desc">Login to Rexec. Use --token to skip interactive prompt.</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec logout</code>
                        <span class="command-desc">Clear saved credentials.</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec whoami</code>
                        <span class="command-desc">Show current user information.</span>
                    </div>
                </div>
            </div>

            <h3 style="margin-top: 24px;">Terminal Management</h3>
            <div class="cli-commands">
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec ls / list</code>
                        <span class="command-desc">List all terminals.</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec create [options]</code>
                        <span class="command-desc">Create a new terminal.</span>
                    </div>
                    <div class="command-details" style="margin-top: 8px; font-size: 13px; color: var(--text-muted);">
                        Options: --name, --image, --role, --memory, --cpu
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec connect &lt;terminal-id&gt;</code>
                        <span class="command-desc">Connect to a terminal (interactive shell).</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec start &lt;terminal-id&gt;</code>
                        <span class="command-desc">Start a stopped terminal.</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec stop &lt;terminal-id&gt;</code>
                        <span class="command-desc">Stop a running terminal.</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec rm &lt;terminal-id&gt;</code>
                        <span class="command-desc">Delete a terminal.</span>
                    </div>
                </div>
            </div>

            <h3 style="margin-top: 24px;">Snippets & Macros</h3>
            <div class="cli-commands">
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec snippets</code>
                        <span class="command-desc">List your snippets.</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec run &lt;snippet-name&gt;</code>
                        <span class="command-desc">Run a snippet on a terminal.</span>
                    </div>
                </div>
            </div>
            
            <h3 style="margin-top: 24px;">Agent Mode</h3>
            <div class="cli-commands">
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec agent register</code>
                        <span class="command-desc">Register this machine.</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec agent start</code>
                        <span class="command-desc">Start the agent.</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec agent status</code>
                        <span class="command-desc">Check agent status.</span>
                    </div>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Tips & Tricks</h2>
            <div class="faq-list">
                <div class="faq-item">
                    <h4>Aliases</h4>
                    <p>Add to your shell config for quick access:</p>
                    <div class="code-block">
                        <code>alias dev="rexec connect abc123"<br/>alias r="rexec -i"</code>
                    </div>
                </div>
                <div class="faq-item">
                    <h4>Shell Completion</h4>
                    <p>Generate completion scripts for your shell:</p>
                    <div class="code-block">
                        <code>rexec completion bash > /etc/bash_completion.d/rexec<br/>rexec completion zsh > ~/.zsh/completions/_rexec</code>
                    </div>
                </div>
                <div class="faq-item">
                    <h4>SSH Integration</h4>
                    <p>Use rexec as an SSH replacement in <code>~/.ssh/config</code>:</p>
                    <div class="code-block">
                        <code>Host rexec-*<br/>    ProxyCommand rexec connect %h</code>
                    </div>
                </div>
            </div>
        </section>
    </div>
</div>

<style>
    .docs-page {
        min-height: 100vh;
        background: #0a0a0a;
        padding: 24px;
        overflow-y: auto;
    }

    .back-btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 8px 14px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        font-size: 13px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s ease;
        margin-bottom: 24px;
    }

    .back-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    .back-icon {
        font-size: 16px;
    }

    .docs-content {
        max-width: 900px;
        margin: 0 auto;
    }

    .docs-header {
        text-align: center;
        margin-bottom: 48px;
        padding-bottom: 32px;
        border-bottom: 1px solid #222;
    }

    .header-icon {
        margin-bottom: 16px;
    }

    .header-icon :global(svg) {
        color: var(--accent);
    }

    .docs-header h1 {
        font-size: 36px;
        margin: 0 0 12px 0;
        background: linear-gradient(135deg, var(--accent), #00d4ff);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .subtitle {
        font-size: 16px;
        color: var(--text-muted);
        margin: 0;
    }

    .docs-section {
        margin-bottom: 48px;
    }

    .docs-section h2 {
        font-size: 20px;
        margin: 0 0 16px 0;
        color: var(--text);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .docs-section p {
        font-size: 14px;
        color: var(--text-muted);
        line-height: 1.7;
        margin: 0 0 16px 0;
    }

    .docs-section h3 {
        font-size: 16px;
        color: var(--text);
        margin: 0 0 12px 0;
    }

    .feature-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 16px;
        margin-top: 20px;
    }

    .feature-card {
        background: #111;
        border: 1px solid #222;
        border-radius: 8px;
        padding: 20px;
    }

    .feature-card :global(svg) {
        color: var(--accent);
        margin-bottom: 12px;
    }

    .feature-card h4 {
        font-size: 14px;
        margin: 0 0 8px 0;
        color: var(--text);
    }

    .feature-card p {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0;
        line-height: 1.5;
    }

    .code-block {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        background: #0d0d0d;
        border: 1px solid #222;
        border-radius: 8px;
        padding: 16px;
        margin: 12px 0;
    }

    .code-block code {
        flex: 1;
        font-family: var(--font-mono);
        font-size: 13px;
        color: var(--accent);
        line-height: 1.6;
        white-space: pre-wrap;
        word-break: break-all;
    }

    .copy-btn {
        flex-shrink: 0;
        padding: 6px 12px;
        background: transparent;
        border: 1px solid #333;
        border-radius: 4px;
        color: var(--text-muted);
        font-size: 11px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .copy-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
    }

    .install-method {
        margin-bottom: 20px;
    }

    .install-method h3 {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 14px;
        color: var(--text);
        margin: 0 0 8px 0;
    }

    .cli-commands {
        display: flex;
        flex-direction: column;
        gap: 8px;
        margin-top: 16px;
    }

    .command-item {
        background: #111;
        border: 1px solid #222;
        border-radius: 6px;
        padding: 12px 16px;
    }

    .command-header {
        display: flex;
        align-items: center;
        gap: 16px;
        flex-wrap: wrap;
    }

    .command {
        font-family: var(--font-mono);
        font-size: 13px;
        color: var(--accent);
        background: #0a0a0a;
        padding: 4px 8px;
        border-radius: 4px;
    }

    .command-desc {
        font-size: 13px;
        color: var(--text-muted);
    }

    .faq-list {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .faq-item {
        background: #111;
        border: 1px solid #222;
        border-radius: 8px;
        padding: 16px;
    }

    .faq-item h4 {
        font-size: 14px;
        color: var(--text);
        margin: 0 0 8px 0;
    }

    .faq-item p {
        font-size: 13px;
        margin: 0 0 8px 0;
    }

    @media (max-width: 600px) {
        .docs-page {
            padding: 16px;
        }

        .docs-header h1 {
            font-size: 28px;
        }

        .code-block {
            flex-direction: column;
        }

        .copy-btn {
            align-self: flex-end;
        }

        .command-header {
            flex-direction: column;
            align-items: flex-start;
            gap: 8px;
        }
    }
</style>
