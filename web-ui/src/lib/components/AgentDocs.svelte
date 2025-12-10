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
                <StatusIcon status="connected" size={48} />
            </div>
            <h1>rexec Agent</h1>
            <p class="subtitle">Connect your own servers, VMs, and machines to rexec</p>
        </header>

        <section class="docs-section">
            <h2>What is the rexec Agent?</h2>
            <p>
                The rexec agent allows you to connect any machine to rexec and access it from your dashboard. 
                This includes cloud VMs (AWS, GCP, Azure), bare metal servers, Raspberry Pi, 
                or even your local development machine.
            </p>
            <div class="feature-grid">
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Resumable Sessions</h4>
                    <p>Sessions persist across disconnects using tmux</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Secure Connection</h4>
                    <p>Encrypted WebSocket tunnel to your machine</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Cross-Platform</h4>
                    <p>Linux, macOS, and Windows (via WSL)</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Full Access</h4>
                    <p>Access your machine's full environment and tools</p>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Quick Install</h2>
            <p>Run this one-liner on your machine to install and start the agent:</p>
            <div class="code-block large">
                <code>curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash</code>
                <button 
                    class="copy-btn" 
                    onclick={() => copyToClipboard('curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash', 'quick')}
                >
                    {copiedCommand === 'quick' ? 'Copied!' : 'Copy'}
                </button>
            </div>
            <p class="hint">This downloads the rexec CLI, installs it, and starts the agent registration flow.</p>
        </section>

        <section class="docs-section">
            <h2>Manual Installation</h2>
            
            <div class="install-method">
                <h3>
                    <PlatformIcon platform="ubuntu" size={20} />
                    Debian / Ubuntu
                </h3>
                <div class="code-block">
                    <code>sudo apt update && sudo apt install -y curl<br/>curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash</code>
                    <button 
                        class="copy-btn" 
                        onclick={() => copyToClipboard('sudo apt update && sudo apt install -y curl && curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash', 'debian')}
                    >
                        {copiedCommand === 'debian' ? 'Copied!' : 'Copy'}
                    </button>
                </div>
            </div>

            <div class="install-method">
                <h3>
                    <PlatformIcon platform="fedora" size={20} />
                    Fedora / RHEL / CentOS
                </h3>
                <div class="code-block">
                    <code>sudo dnf install -y curl<br/>curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash</code>
                    <button 
                        class="copy-btn" 
                        onclick={() => copyToClipboard('sudo dnf install -y curl && curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash', 'rhel')}
                    >
                        {copiedCommand === 'rhel' ? 'Copied!' : 'Copy'}
                    </button>
                </div>
            </div>

            <div class="install-method">
                <h3>
                    <PlatformIcon platform="archlinux" size={20} />
                    Arch Linux
                </h3>
                <div class="code-block">
                    <code>sudo pacman -S curl<br/>curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash</code>
                    <button 
                        class="copy-btn" 
                        onclick={() => copyToClipboard('sudo pacman -S curl && curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash', 'arch')}
                    >
                        {copiedCommand === 'arch' ? 'Copied!' : 'Copy'}
                    </button>
                </div>
            </div>

            <div class="install-method">
                <h3>
                    <PlatformIcon platform="alpine" size={20} />
                    Alpine Linux
                </h3>
                <div class="code-block">
                    <code>apk add curl bash<br/>curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash</code>
                    <button 
                        class="copy-btn" 
                        onclick={() => copyToClipboard('apk add curl bash && curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash', 'alpine')}
                    >
                        {copiedCommand === 'alpine' ? 'Copied!' : 'Copy'}
                    </button>
                </div>
            </div>

            <div class="install-method">
                <h3>
                    <PlatformIcon platform="macos" size={20} />
                    macOS
                </h3>
                <div class="code-block">
                    <code>curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash</code>
                    <button 
                        class="copy-btn" 
                        onclick={() => copyToClipboard('curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash', 'macos')}
                    >
                        {copiedCommand === 'macos' ? 'Copied!' : 'Copy'}
                    </button>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Using the rexec CLI</h2>
            <p>After installation, you can use the rexec CLI to manage your agent:</p>
            
            <div class="cli-commands">
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec agent start</code>
                        <span class="command-desc">Start the agent and connect to rexec</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec agent stop</code>
                        <span class="command-desc">Stop the running agent</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec agent status</code>
                        <span class="command-desc">Check if agent is running and connected</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec -i</code>
                        <span class="command-desc">Launch interactive TUI mode</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec login</code>
                        <span class="command-desc">Authenticate with your rexec account</span>
                    </div>
                </div>
                <div class="command-item">
                    <div class="command-header">
                        <code class="command">rexec terminals</code>
                        <span class="command-desc">List your terminals</span>
                    </div>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Running as a Service</h2>
            <p>For production servers, run the agent as a systemd service:</p>
            
            <div class="code-block">
                <code># Create systemd service<br/>sudo tee /etc/systemd/system/rexec-agent.service &lt;&lt;EOF<br/>[Unit]<br/>Description=rexec Agent<br/>After=network.target<br/><br/>[Service]<br/>Type=simple<br/>ExecStart=/usr/local/bin/rexec agent start --daemon<br/>Restart=always<br/>RestartSec=10<br/><br/>[Install]<br/>WantedBy=multi-user.target<br/>EOF<br/><br/># Enable and start<br/>sudo systemctl daemon-reload<br/>sudo systemctl enable rexec-agent<br/>sudo systemctl start rexec-agent</code>
            </div>
        </section>

        <section class="docs-section">
            <h2>Supported Platforms</h2>
            <div class="platform-grid">
                <div class="platform-item">
                    <PlatformIcon platform="ubuntu" size={24} />
                    <span>Ubuntu</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="debian" size={24} />
                    <span>Debian</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="fedora" size={24} />
                    <span>Fedora</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="centos" size={24} />
                    <span>CentOS</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="rocky" size={24} />
                    <span>Rocky Linux</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="archlinux" size={24} />
                    <span>Arch Linux</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="alpine" size={24} />
                    <span>Alpine</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="amazonlinux" size={24} />
                    <span>Amazon Linux</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="macos" size={24} />
                    <span>macOS</span>
                </div>
                <div class="platform-item">
                    <PlatformIcon platform="raspberrypi" size={24} />
                    <span>Raspberry Pi</span>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Troubleshooting</h2>
            <div class="faq-list">
                <div class="faq-item">
                    <h4>Agent won't connect</h4>
                    <p>Make sure you're authenticated with <code>rexec login</code> and your network allows outbound WebSocket connections.</p>
                </div>
                <div class="faq-item">
                    <h4>Permission denied</h4>
                    <p>Ensure you have sudo access or run the install script with appropriate permissions.</p>
                </div>
                <div class="faq-item">
                    <h4>Terminal not appearing in dashboard</h4>
                    <p>Check that the agent is running with <code>rexec agent status</code> and that you're logged in to the same account.</p>
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

    .code-block.large {
        padding: 20px;
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

    .hint {
        font-size: 12px !important;
        color: var(--text-muted) !important;
        font-style: italic;
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

    .platform-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        gap: 12px;
        margin-top: 16px;
    }

    .platform-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 16px;
        background: #111;
        border: 1px solid #222;
        border-radius: 8px;
        font-size: 12px;
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
        margin: 0;
    }

    .faq-item code {
        background: #0a0a0a;
        padding: 2px 6px;
        border-radius: 4px;
        color: var(--accent);
        font-size: 12px;
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
