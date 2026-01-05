<script lang="ts">
    import StatusIcon from "./icons/StatusIcon.svelte";

    export let onback: (() => void) | undefined = undefined;

    let copiedCommand = "";
    let activeExampleTab = "basic";
    let showLivePreview = false;
    let previewShareCode = "";
    let previewToken = "";
    let previewImage = "ubuntu";
    let previewRole = "default";
    let previewMode: "share" | "new" = "share";
    let previewTerminalInstance: any = null;

    function copyToClipboard(text: string, id: string) {
        navigator.clipboard.writeText(text);
        copiedCommand = id;
        setTimeout(() => {
            copiedCommand = "";
        }, 2000);
    }

    function handleBack() {
        if (onback) onback();
    }

    const currentHost =
        typeof window !== "undefined" ? window.location.host : "rexec.dev";
    const protocol =
        typeof window !== "undefined" ? window.location.protocol : "https:";
    const baseUrl = `${protocol}//${currentHost}`;

    // Helper to build code snippets with script tags (avoids Svelte parsing issues)
    function getQuickStartCode() {
        return `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 400px;"></div>

<script>
  const term = Rexec.embed('#terminal', {
    shareCode: 'your-share-code'
  });
<\/script>`;
    }

    // Example code snippets
    const examples = {
        basic: {
            title: "Basic Setup",
            description:
                "The simplest way to embed a terminal with a share code.",
            code: `<!-- 1. Include the Rexec embed script -->
<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<!-- 2. Add a container element -->
<div id="terminal" style="width: 100%; height: 400px;"></div>

<!-- 3. Initialize the terminal -->
<script>
  const terminal = Rexec.embed('#terminal', {
    shareCode: 'ABC123'  // Get this from terminal share button
  });
<\/script>`,
            copyText: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 400px;"></div>

<script>
  const terminal = Rexec.embed('#terminal', {
    shareCode: 'ABC123'
  });
<\/script>`,
        },
        newContainer: {
            title: "Create New Container",
            description:
                "Spin up a fresh container for each user with an API token.",
            code: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 400px;"></div>

<script>
  const terminal = Rexec.embed('#terminal', {
    token: 'your-api-token',
    image: 'ubuntu',           // or 'alpine', 'debian', 'fedora', etc.
    role: 'python',            // or 'node', 'go', 'rust', 'default'

    onReady: (term) => {
      console.log('Container ID:', term.session.containerId);
      // Save for later reconnection
      localStorage.setItem('lastContainer', term.session.containerId);
    }
  });
<\/script>`,
            copyText: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 400px;"></div>

<script>
  const terminal = Rexec.embed('#terminal', {
    token: 'your-api-token',
    image: 'ubuntu',
    role: 'python',
    onReady: (term) => {
      console.log('Container ID:', term.session.containerId);
      localStorage.setItem('lastContainer', term.session.containerId);
    }
  });
<\/script>`,
        },
        reconnect: {
            title: "Reconnect to Container",
            description:
                "Reconnect to a previously created container using its ID.",
            code: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 400px;"></div>

<script>
  // Get saved container ID
  const containerId = localStorage.getItem('lastContainer');

  if (containerId) {
    const terminal = Rexec.embed('#terminal', {
      token: 'your-api-token',
      container: containerId,

      onError: (err) => {
        if (err.code === 'CONTAINER_NOT_FOUND') {
          console.log('Container expired, create a new one');
          localStorage.removeItem('lastContainer');
        }
      }
    });
  }
<\/script>`,
            copyText: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 400px;"></div>

<script>
  const containerId = localStorage.getItem('lastContainer');

  if (containerId) {
    const terminal = Rexec.embed('#terminal', {
      token: 'your-api-token',
      container: containerId,
      onError: (err) => {
        if (err.code === 'CONTAINER_NOT_FOUND') {
          localStorage.removeItem('lastContainer');
        }
      }
    });
  }
<\/script>`,
        },
        advanced: {
            title: "Advanced Configuration",
            description:
                "Full customization with themes, fonts, and event handlers.",
            code: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 500px;"></div>

<script>
  const terminal = Rexec.embed('#terminal', {
    // Authentication
    token: 'your-api-token',

    // Container config
    image: 'alpine',
    role: 'python',

    // Appearance
    theme: 'dark',              // or 'light', or custom object
    fontSize: 14,
    fontFamily: 'JetBrains Mono, monospace',
    cursorStyle: 'block',       // 'block', 'underline', 'bar'
    cursorBlink: true,

    // Behavior
    initialCommand: 'python3 --version && echo "Ready!"',
    autoReconnect: true,
    maxReconnectAttempts: 10,
    scrollback: 5000,

    // Event callbacks
    onReady: (term) => {
      console.log('Connected!', term.session);
    },

    onStateChange: (state) => {
      // 'idle', 'connecting', 'connected', 'reconnecting', 'error'
      document.getElementById('status').textContent = state;
    },

    onData: (data) => {
      // Handle terminal output if needed
    },

    onError: (error) => {
      console.error('Error:', error.code, error.message);
    },

    onDisconnect: () => {
      console.log('Disconnected');
    }
  });
<\/script>`,
            copyText: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 500px;"></div>

<script>
  const terminal = Rexec.embed('#terminal', {
    token: 'your-api-token',
    image: 'alpine',
    role: 'python',
    theme: 'dark',
    fontSize: 14,
    fontFamily: 'JetBrains Mono, monospace',
    cursorStyle: 'block',
    cursorBlink: true,
    initialCommand: 'python3 --version && echo "Ready!"',
    autoReconnect: true,
    maxReconnectAttempts: 10,
    scrollback: 5000,
    onReady: (term) => {
      console.log('Connected!', term.session);
    },
    onStateChange: (state) => {
      document.getElementById('status').textContent = state;
    },
    onError: (error) => {
      console.error('Error:', error.code, error.message);
    }
  });
<\/script>`,
        },
        customTheme: {
            title: "Custom Theme",
            description: "Create your own color scheme for the terminal.",
            code: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 400px;"></div>

<script>
  // Tokyo Night theme example
  const terminal = Rexec.embed('#terminal', {
    shareCode: 'ABC123',

    theme: {
      background: '#1a1b26',
      foreground: '#a9b1d6',
      cursor: '#c0caf5',
      selectionBackground: '#33467c',

      // ANSI colors
      black: '#15161e',
      red: '#f7768e',
      green: '#9ece6a',
      yellow: '#e0af68',
      blue: '#7aa2f7',
      magenta: '#bb9af7',
      cyan: '#7dcfff',
      white: '#a9b1d6',

      // Bright variants
      brightBlack: '#414868',
      brightRed: '#f7768e',
      brightGreen: '#9ece6a',
      brightYellow: '#e0af68',
      brightBlue: '#7aa2f7',
      brightMagenta: '#bb9af7',
      brightCyan: '#7dcfff',
      brightWhite: '#c0caf5'
    }
  });

  // Or use built-in presets:
  // theme: Rexec.DARK_THEME
  // theme: Rexec.LIGHT_THEME
<\/script>`,
            copyText: `<script src="${baseUrl}/embed/rexec.min.js"><\/script>

<div id="terminal" style="width: 100%; height: 400px;"></div>

<script>
  const terminal = Rexec.embed('#terminal', {
    shareCode: 'ABC123',
    theme: {
      background: '#1a1b26',
      foreground: '#a9b1d6',
      cursor: '#c0caf5',
      selectionBackground: '#33467c',
      black: '#15161e',
      red: '#f7768e',
      green: '#9ece6a',
      yellow: '#e0af68',
      blue: '#7aa2f7',
      magenta: '#bb9af7',
      cyan: '#7dcfff',
      white: '#a9b1d6'
    }
  });
<\/script>`,
        },
    };

    function highlightCode(code: string): string {
        // Simple syntax highlighting
        return (
            code
                // Comments first (HTML and JS)
                .replace(
                    /(&lt;!--.*?--&gt;)/g,
                    '<span class="hl-comment">$1</span>',
                )
                .replace(
                    /(\/\/.*?)(?=\n|$)/g,
                    '<span class="hl-comment">$1</span>',
                )
                // Strings (single and double quotes)
                .replace(
                    /('(?:[^'\\]|\\.)*')/g,
                    '<span class="hl-string">$1</span>',
                )
                .replace(
                    /("(?:[^"\\]|\\.)*")/g,
                    '<span class="hl-string">$1</span>',
                )
                // Keywords
                .replace(
                    /\b(const|let|var|if|else|function|return|true|false|null|undefined)\b/g,
                    '<span class="hl-keyword">$1</span>',
                )
                // HTML tags (simplified)
                .replace(
                    /(&lt;\/?)(script|div|style)(&gt;|[\s&])/gi,
                    '$1<span class="hl-tag">$2</span>$3',
                )
                // Properties/keys
                .replace(/(\w+):/g, '<span class="hl-property">$1</span>:')
                // Numbers
                .replace(/\b(\d+)\b/g, '<span class="hl-number">$1</span>')
        );
    }

    function escapeHtml(text: string): string {
        return text
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;");
    }

    function canLaunchPreview(): boolean {
        if (previewMode === "share") {
            return !!previewShareCode.trim();
        } else {
            return !!previewToken.trim();
        }
    }

    function launchPreview() {
        if (!canLaunchPreview()) return;

        // Destroy existing terminal if any
        if (previewTerminalInstance) {
            try {
                previewTerminalInstance.destroy();
            } catch (e) {
                // ignore
            }
            previewTerminalInstance = null;
        }

        showLivePreview = true;

        // Wait for DOM update, then initialize terminal
        setTimeout(() => {
            const container = document.getElementById("live-preview-terminal");
            if (container && (window as any).Rexec) {
                const config: any = {};

                if (previewMode === "share") {
                    config.shareCode = previewShareCode.trim();
                } else {
                    config.token = previewToken.trim();
                    config.image = previewImage;
                    config.role = previewRole;
                }

                config.onReady = (term: any) => {
                    console.log("Preview connected:", term.session);
                };

                config.onError = (err: any) => {
                    console.error("Preview error:", err);
                };

                previewTerminalInstance = (window as any).Rexec.embed(
                    container,
                    config,
                );
            }
        }, 100);
    }

    function closePreview() {
        if (previewTerminalInstance) {
            try {
                previewTerminalInstance.destroy();
            } catch (e) {
                // ignore
            }
            previewTerminalInstance = null;
        }
        showLivePreview = false;
    }
</script>

<svelte:head>
    <script src="{baseUrl}/embed/rexec.min.js"></script>
</svelte:head>

<div class="docs-page">
    <button class="back-btn" onclick={handleBack}>
        <span class="back-icon">←</span>
        <span>Back</span>
    </button>

    <div class="docs-content">
        <header class="docs-header">
            <div class="header-icon">
                <StatusIcon status="code" size={48} />
            </div>
            <h1>Embeddable Terminal Widget</h1>
            <p class="subtitle">
                Add a cloud terminal to any website with a single script tag
            </p>
        </header>

        <section class="docs-section">
            <h2>What is the Embed Widget?</h2>
            <p>
                The Rexec embed widget lets you add a fully-featured cloud
                terminal to any website. Similar to Google Cloud Shell, you can
                embed interactive terminal sessions in documentation, tutorials,
                learning platforms, or anywhere you need live command execution.
            </p>
            <div class="feature-grid">
                <div class="feature-card">
                    <StatusIcon status="bolt" size={20} />
                    <h4>One-Line Integration</h4>
                    <p>
                        Add a terminal with just a script tag and one line of
                        JavaScript
                    </p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="user" size={20} />
                    <h4>Guest & Auth Modes</h4>
                    <p>
                        Support share codes for guests or API tokens for
                        authenticated access
                    </p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="palette" size={20} />
                    <h4>Fully Customizable</h4>
                    <p>
                        Themes, fonts, sizes, and event callbacks for full
                        control
                    </p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="cloud" size={20} />
                    <h4>Cloud Powered</h4>
                    <p>
                        Runs on Rexec infrastructure - no server setup required
                    </p>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Quick Start</h2>
            <p>Add this to your HTML to embed a terminal:</p>
            <div class="code-block large">
                <code
                    >&lt;!-- Include the embed script --&gt;<br />&lt;script
                    src="{baseUrl}/embed/rexec.min.js"&gt;&lt;/script&gt;<br
                    /><br />&lt;!-- Create a container --&gt;<br />&lt;div
                    id="terminal" style="width: 100%; height:
                    400px;"&gt;&lt;/div&gt;<br /><br />&lt;!-- Initialize --&gt;<br
                    />&lt;script&gt;<br /> const term = Rexec.embed('#terminal',
                    &#123;<br /> shareCode: 'your-share-code'<br /> &#125;);<br
                    />&lt;/script&gt;</code
                >
                <button
                    class="copy-btn"
                    onclick={() =>
                        copyToClipboard(getQuickStartCode(), "quick")}
                >
                    {copiedCommand === "quick" ? "Copied!" : "Copy"}
                </button>
            </div>
            <p class="hint">
                Replace 'your-share-code' with an actual session share code from
                your Rexec dashboard.
            </p>
        </section>

        <!-- Interactive Examples Section -->
        <section class="docs-section examples-section">
            <h2>Interactive Examples</h2>
            <p>
                Click on any example to see the full code. All examples are
                copy-ready and can be pasted directly into your HTML file.
            </p>

            <div class="example-tabs">
                {#each Object.entries(examples) as [key, example]}
                    <button
                        class="example-tab"
                        class:active={activeExampleTab === key}
                        onclick={() => (activeExampleTab = key)}
                    >
                        {example.title}
                    </button>
                {/each}
            </div>

            <div class="example-content">
                {#each Object.entries(examples) as [key, example]}
                    {#if activeExampleTab === key}
                        <div class="example-panel">
                            <div class="example-header">
                                <div class="example-info">
                                    <h3>{example.title}</h3>
                                    <p>{example.description}</p>
                                </div>
                                <button
                                    class="copy-btn large"
                                    onclick={() =>
                                        copyToClipboard(example.copyText, key)}
                                >
                                    {copiedCommand === key
                                        ? "✓ Copied!"
                                        : "Copy Code"}
                                </button>
                            </div>
                            <div class="code-editor">
                                <div class="editor-header">
                                    <div class="editor-dots">
                                        <span class="dot red"></span>
                                        <span class="dot yellow"></span>
                                        <span class="dot green"></span>
                                    </div>
                                    <span class="editor-title">index.html</span>
                                </div>
                                <pre class="editor-content"><code
                                        >{@html highlightCode(
                                            escapeHtml(example.code),
                                        )}</code
                                    ></pre>
                            </div>
                        </div>
                    {/if}
                {/each}
            </div>
        </section>

        <!-- Live Preview Section -->
        <section class="docs-section preview-section">
            <h2>Try It Live</h2>
            <p>
                Test the embed widget directly on this page. Choose to join an
                existing session with a share code, or create a new container
                with your API token.
            </p>

            <div class="preview-mode-tabs">
                <button
                    class="preview-mode-tab"
                    class:active={previewMode === "share"}
                    onclick={() => (previewMode = "share")}
                >
                    <StatusIcon status="user" size={16} />
                    Join with Share Code
                </button>
                <button
                    class="preview-mode-tab"
                    class:active={previewMode === "new"}
                    onclick={() => (previewMode = "new")}
                >
                    <StatusIcon status="plus" size={16} />
                    Create New Container
                </button>
            </div>

            <div class="preview-controls">
                {#if previewMode === "share"}
                    <div class="preview-input-group">
                        <input
                            type="text"
                            bind:value={previewShareCode}
                            placeholder="Enter share code (e.g., ABC123)"
                            class="preview-input"
                            onkeydown={(e) =>
                                e.key === "Enter" && launchPreview()}
                        />
                    </div>
                    <p class="preview-hint">
                        Get a share code by creating a terminal in your <a
                            href="/dashboard">dashboard</a
                        > and clicking the share button.
                    </p>
                {:else}
                    <div class="preview-form">
                        <div class="preview-input-group">
                            <label for="preview-token">API Token</label>
                            <input
                                id="preview-token"
                                type="password"
                                bind:value={previewToken}
                                placeholder="Enter your API token"
                                class="preview-input"
                            />
                        </div>
                        <div class="preview-row">
                            <div class="preview-input-group half">
                                <label for="preview-image">Image</label>
                                <select
                                    id="preview-image"
                                    bind:value={previewImage}
                                    class="preview-select"
                                >
                                    <option value="ubuntu">Ubuntu 24.04</option>
                                    <option value="ubuntu-22"
                                        >Ubuntu 22.04</option
                                    >
                                    <option value="debian">Debian 12</option>
                                    <option value="alpine">Alpine</option>
                                    <option value="fedora">Fedora 41</option>
                                    <option value="archlinux">Arch Linux</option
                                    >
                                    <option value="rocky">Rocky Linux 9</option>
                                    <option value="alma">AlmaLinux 9</option>
                                    <option value="opensuse"
                                        >openSUSE Leap</option
                                    >
                                    <option value="kali">Kali Linux</option>
                                </select>
                            </div>
                            <div class="preview-input-group half">
                                <label for="preview-role">Role</label>
                                <select
                                    id="preview-role"
                                    bind:value={previewRole}
                                    class="preview-select"
                                >
                                    <option value="default">Default</option>
                                    <option value="python">Python</option>
                                    <option value="node">Node.js</option>
                                    <option value="go">Go</option>
                                    <option value="rust">Rust</option>
                                </select>
                            </div>
                        </div>
                    </div>
                    <p class="preview-hint">
                        Get an API token from <a href="/account/api"
                            >Account → API Tokens</a
                        >. Your token is not stored or sent anywhere except to
                        the Rexec API.
                    </p>
                {/if}

                <div class="preview-actions">
                    <button
                        class="preview-btn"
                        onclick={launchPreview}
                        disabled={!canLaunchPreview()}
                    >
                        <StatusIcon status="play" size={16} />
                        {showLivePreview
                            ? "Restart Terminal"
                            : "Launch Preview"}
                    </button>
                    {#if showLivePreview}
                        <button
                            class="preview-btn secondary"
                            onclick={closePreview}
                        >
                            Close Preview
                        </button>
                    {/if}
                </div>
            </div>

            {#if showLivePreview}
                <div class="live-preview-container">
                    <div class="preview-header">
                        <StatusIcon status="terminal" size={16} />
                        <span>Live Terminal Preview</span>
                        <span class="preview-code">
                            {#if previewMode === "share"}
                                Share Code: {previewShareCode}
                            {:else}
                                {previewImage} • {previewRole}
                            {/if}
                        </span>
                    </div>
                    <div
                        id="live-preview-terminal"
                        class="preview-terminal"
                    ></div>
                </div>
            {/if}
        </section>

        <section class="docs-section">
            <h2>Connection Methods</h2>

            <div class="method-card">
                <h3>
                    <StatusIcon status="user" size={20} />
                    Join via Share Code (Guest Access)
                </h3>
                <p>
                    Join an existing shared session. No authentication required
                    - perfect for tutorials and demos.
                </p>
                <div class="code-block">
                    <code
                        >const term = Rexec.embed('#terminal', &#123;<br />
                        shareCode: 'ABC123'<br />&#125;);</code
                    >
                    <button
                        class="copy-btn"
                        onclick={() =>
                            copyToClipboard(
                                `const term = Rexec.embed('#terminal', {\n  shareCode: 'ABC123'\n});`,
                                "share",
                            )}
                    >
                        {copiedCommand === "share" ? "Copied!" : "Copy"}
                    </button>
                </div>
            </div>

            <div class="method-card">
                <h3>
                    <StatusIcon status="key" size={20} />
                    Connect to Existing Container
                </h3>
                <p>Connect to a container you own. Requires an API token.</p>
                <div class="code-block">
                    <code
                        >const term = Rexec.embed('#terminal', &#123;<br />
                        token: 'your-api-token',<br /> container: 'container-id'<br
                        />&#125;);</code
                    >
                    <button
                        class="copy-btn"
                        onclick={() =>
                            copyToClipboard(
                                `const term = Rexec.embed('#terminal', {\n  token: 'your-api-token',\n  container: 'container-id'\n});`,
                                "container",
                            )}
                    >
                        {copiedCommand === "container" ? "Copied!" : "Copy"}
                    </button>
                </div>
            </div>

            <div class="method-card">
                <h3>
                    <StatusIcon status="plus" size={20} />
                    Create New Container On-Demand
                </h3>
                <p>
                    Spin up a fresh container for each user. Great for
                    interactive learning platforms.
                </p>
                <div class="code-block">
                    <code
                        >const term = Rexec.embed('#terminal', &#123;<br />
                        token: 'your-api-token',<br /> image: 'ubuntu',<br />
                        role: 'python' // or 'node', 'go', 'rust', 'default'<br
                        />&#125;);</code
                    >
                    <button
                        class="copy-btn"
                        onclick={() =>
                            copyToClipboard(
                                `const term = Rexec.embed('#terminal', {\n  token: 'your-api-token',\n  image: 'ubuntu',\n  role: 'python'\n});`,
                                "create",
                            )}
                    >
                        {copiedCommand === "create" ? "Copied!" : "Copy"}
                    </button>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Configuration Options</h2>
            <div class="options-table">
                <div class="option-row header">
                    <span class="option-name">Option</span>
                    <span class="option-type">Type</span>
                    <span class="option-default">Default</span>
                    <span class="option-desc">Description</span>
                </div>
                <div class="option-row">
                    <span class="option-name"><code>token</code></span>
                    <span class="option-type">string</span>
                    <span class="option-default">-</span>
                    <span class="option-desc">API token for authentication</span
                    >
                </div>
                <div class="option-row">
                    <span class="option-name"><code>container</code></span>
                    <span class="option-type">string</span>
                    <span class="option-default">-</span>
                    <span class="option-desc">Container ID to connect to</span>
                </div>
                <div class="option-row">
                    <span class="option-name"><code>shareCode</code></span>
                    <span class="option-type">string</span>
                    <span class="option-default">-</span>
                    <span class="option-desc"
                        >Share code for joining sessions</span
                    >
                </div>
                <div class="option-row">
                    <span class="option-name"><code>image</code></span>
                    <span class="option-type">string</span>
                    <span class="option-default">'ubuntu'</span>
                    <span class="option-desc"
                        >Base image for new containers</span
                    >
                </div>
                <div class="option-row">
                    <span class="option-name"><code>role</code></span>
                    <span class="option-type">string</span>
                    <span class="option-default">'default'</span>
                    <span class="option-desc"
                        >Environment type for new containers</span
                    >
                </div>
                <div class="option-row">
                    <span class="option-name"><code>baseUrl</code></span>
                    <span class="option-type">string</span>
                    <span class="option-default">'https://rexec.dev'</span>
                    <span class="option-desc">API base URL</span>
                </div>
                <div class="option-row">
                    <span class="option-name"><code>theme</code></span>
                    <span class="option-type">'dark' | 'light' | object</span>
                    <span class="option-default">'dark'</span>
                    <span class="option-desc">Terminal color theme</span>
                </div>
                <div class="option-row">
                    <span class="option-name"><code>fontSize</code></span>
                    <span class="option-type">number</span>
                    <span class="option-default">14</span>
                    <span class="option-desc">Font size in pixels</span>
                </div>
                <div class="option-row">
                    <span class="option-name"><code>cursorStyle</code></span>
                    <span class="option-type"
                        >'block' | 'underline' | 'bar'</span
                    >
                    <span class="option-default">'block'</span>
                    <span class="option-desc">Cursor appearance</span>
                </div>
                <div class="option-row">
                    <span class="option-name"><code>scrollback</code></span>
                    <span class="option-type">number</span>
                    <span class="option-default">5000</span>
                    <span class="option-desc">Lines in scrollback buffer</span>
                </div>
                <div class="option-row">
                    <span class="option-name"><code>showStatus</code></span>
                    <span class="option-type">boolean</span>
                    <span class="option-default">true</span>
                    <span class="option-desc"
                        >Show connection status overlay</span
                    >
                </div>
                <div class="option-row">
                    <span class="option-name"><code>autoReconnect</code></span>
                    <span class="option-type">boolean</span>
                    <span class="option-default">true</span>
                    <span class="option-desc">Auto-reconnect on disconnect</span
                    >
                </div>
                <div class="option-row">
                    <span class="option-name"><code>initialCommand</code></span>
                    <span class="option-type">string</span>
                    <span class="option-default">-</span>
                    <span class="option-desc">Command to run after connect</span
                    >
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Available Images</h2>
            <p>
                Use these image keys in the <code>image</code> configuration option:
            </p>
            <div class="images-grid">
                <div class="image-group">
                    <h4>Debian-based</h4>
                    <div class="image-list">
                        <div class="image-item">
                            <code>'ubuntu'</code>
                            <span>Ubuntu 24.04 LTS (default)</span>
                        </div>
                        <div class="image-item">
                            <code>'ubuntu-22'</code>
                            <span>Ubuntu 22.04 LTS</span>
                        </div>
                        <div class="image-item">
                            <code>'debian'</code>
                            <span>Debian 12 Bookworm</span>
                        </div>
                        <div class="image-item">
                            <code>'kali'</code>
                            <span>Kali Linux</span>
                        </div>
                    </div>
                </div>
                <div class="image-group">
                    <h4>Red Hat-based</h4>
                    <div class="image-list">
                        <div class="image-item">
                            <code>'fedora'</code>
                            <span>Fedora 41</span>
                        </div>
                        <div class="image-item">
                            <code>'rocky'</code>
                            <span>Rocky Linux 9</span>
                        </div>
                        <div class="image-item">
                            <code>'alma'</code>
                            <span>AlmaLinux 9</span>
                        </div>
                    </div>
                </div>
                <div class="image-group">
                    <h4>Other</h4>
                    <div class="image-list">
                        <div class="image-item">
                            <code>'alpine'</code>
                            <span>Alpine 3.21 (minimal)</span>
                        </div>
                        <div class="image-item">
                            <code>'archlinux'</code>
                            <span>Arch Linux</span>
                        </div>
                        <div class="image-item">
                            <code>'opensuse'</code>
                            <span>openSUSE Leap 15.6</span>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Event Callbacks</h2>
            <p>Listen to terminal events for custom behavior:</p>
            <div class="code-block">
                <code
                    >const term = Rexec.embed('#terminal', &#123;<br />
                    shareCode: 'ABC123',<br /><br /> onReady: (terminal) =&gt;
                    &#123;<br /> console.log('Terminal connected!');<br />
                    console.log('Session:', terminal.session);<br /> &#125;,<br
                    /><br /> onStateChange: (state) =&gt; &#123;<br /> //
                    'idle', 'connecting', 'connected', 'reconnecting', 'error'<br
                    />
                    console.log('State:', state);<br /> &#125;,<br /><br />
                    onData: (data) =&gt; &#123;<br /> console.log('Output:',
                    data);<br /> &#125;,<br /><br /> onError: (error) =&gt;
                    &#123;<br /> console.error('Error:', error.code,
                    error.message);<br /> &#125;<br />&#125;);</code
                >
                <button
                    class="copy-btn"
                    onclick={() =>
                        copyToClipboard(
                            `const term = Rexec.embed('#terminal', {\n  shareCode: 'ABC123',\n\n  onReady: (terminal) => {\n    console.log('Terminal connected!');\n  },\n\n  onStateChange: (state) => {\n    console.log('State:', state);\n  },\n\n  onData: (data) => {\n    console.log('Output:', data);\n  },\n\n  onError: (error) => {\n    console.error('Error:', error.code, error.message);\n  }\n});`,
                            "events",
                        )}
                >
                    {copiedCommand === "events" ? "Copied!" : "Copy"}
                </button>
            </div>
        </section>

        <section class="docs-section">
            <h2>Terminal API</h2>
            <p>Control the terminal programmatically:</p>

            <div class="api-group">
                <h3>Methods</h3>
                <div class="cli-commands">
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command"
                                >term.write('echo "Hello"')</code
                            >
                            <span class="command-desc"
                                >Write to terminal input</span
                            >
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.writeln('ls -la')</code>
                            <span class="command-desc"
                                >Write with newline (executes command)</span
                            >
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.clear()</code>
                            <span class="command-desc"
                                >Clear the terminal screen</span
                            >
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.fit()</code>
                            <span class="command-desc"
                                >Fit terminal to container size</span
                            >
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.focus()</code>
                            <span class="command-desc">Focus the terminal</span>
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.setTheme('light')</code>
                            <span class="command-desc"
                                >Change theme dynamically</span
                            >
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.setFontSize(16)</code>
                            <span class="command-desc">Change font size</span>
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.destroy()</code>
                            <span class="command-desc"
                                >Clean up and disconnect</span
                            >
                        </div>
                    </div>
                </div>
            </div>

            <div class="api-group">
                <h3>Properties</h3>
                <div class="cli-commands">
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.state</code>
                            <span class="command-desc"
                                >Current connection state</span
                            >
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.session</code>
                            <span class="command-desc"
                                >Session info (id, containerId, etc.)</span
                            >
                        </div>
                    </div>
                    <div class="command-item">
                        <div class="command-header">
                            <code class="command">term.stats</code>
                            <span class="command-desc"
                                >Container stats (cpu, memory, disk)</span
                            >
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Use Cases</h2>
            <div class="feature-grid">
                <div class="feature-card">
                    <StatusIcon status="book" size={20} />
                    <h4>Interactive Documentation</h4>
                    <p>Let users try commands directly in your docs</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="graduation" size={20} />
                    <h4>Learning Platforms</h4>
                    <p>Hands-on coding exercises with real environments</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="play" size={20} />
                    <h4>Product Demos</h4>
                    <p>Showcase CLI tools without users installing anything</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="robot" size={20} />
                    <h4>AI Agent Sandboxes</h4>
                    <p>Give AI assistants a safe execution environment</p>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Browser Support</h2>
            <div class="browser-grid">
                <div class="browser-item">
                    <StatusIcon status="check" size={20} />
                    <span>Chrome 80+</span>
                </div>
                <div class="browser-item">
                    <StatusIcon status="check" size={20} />
                    <span>Firefox 75+</span>
                </div>
                <div class="browser-item">
                    <StatusIcon status="check" size={20} />
                    <span>Safari 13+</span>
                </div>
                <div class="browser-item">
                    <StatusIcon status="check" size={20} />
                    <span>Edge 80+</span>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>FAQ</h2>
            <div class="faq-list">
                <div class="faq-item">
                    <h4>How do I get a share code?</h4>
                    <p>
                        Create a terminal in your Rexec dashboard, click the
                        share button, and copy the code. Share codes allow guest
                        access to your terminal session.
                    </p>
                </div>
                <div class="faq-item">
                    <h4>How do I get an API token?</h4>
                    <p>
                        Go to <a href="/account/api">Account → API Tokens</a> to generate
                        tokens for programmatic access.
                    </p>
                </div>
                <div class="faq-item">
                    <h4>Is there a rate limit?</h4>
                    <p>
                        Free tier has limits on concurrent sessions. Upgrade to
                        Pro for higher limits and priority access.
                    </p>
                </div>
                <div class="faq-item">
                    <h4>Can I self-host?</h4>
                    <p>
                        The embed widget connects to Rexec cloud infrastructure.
                        For on-premise needs, contact us about enterprise
                        options.
                    </p>
                </div>
            </div>
        </section>
    </div>
</div>

<style>
    .docs-page {
        min-height: 100vh;
        background: var(--bg);
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
        border-bottom: 1px solid var(--border);
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

    .docs-section h3 {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 16px;
        margin: 0 0 12px 0;
        color: var(--text);
    }

    .docs-section h3 :global(svg) {
        color: var(--accent);
    }

    .docs-section p {
        font-size: 14px;
        color: var(--text-muted);
        line-height: 1.7;
        margin: 0 0 16px 0;
    }

    .docs-section p a {
        color: var(--accent);
        text-decoration: none;
    }

    .docs-section p a:hover {
        text-decoration: underline;
    }

    .feature-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 16px;
        margin-top: 20px;
    }

    .feature-card {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
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

    .method-card {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 20px;
        margin-bottom: 16px;
    }

    .method-card h3 {
        margin-top: 0;
    }

    .method-card p {
        margin-bottom: 12px;
    }

    .code-block {
        display: flex;
        align-items: flex-start;
        gap: 12px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
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
        border: 1px solid var(--border);
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
        background: var(--accent-dim);
    }

    .copy-btn.large {
        padding: 10px 20px;
        font-size: 13px;
        font-weight: 500;
    }

    .hint {
        font-size: 12px !important;
        color: var(--text-muted) !important;
        font-style: italic;
    }

    /* Interactive Examples Section */
    .examples-section {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 24px;
    }

    .example-tabs {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        margin-bottom: 20px;
        padding-bottom: 16px;
        border-bottom: 1px solid var(--border);
    }

    .example-tab {
        padding: 8px 16px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        font-size: 13px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .example-tab:hover {
        border-color: var(--accent);
        color: var(--text);
    }

    .example-tab.active {
        background: var(--accent);
        border-color: var(--accent);
        color: var(--bg);
    }

    .example-panel {
        animation: fadeIn 0.2s ease;
    }

    @keyframes fadeIn {
        from {
            opacity: 0;
            transform: translateY(-5px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .example-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        margin-bottom: 16px;
    }

    .example-info h3 {
        font-size: 18px;
        margin: 0 0 6px 0;
        color: var(--text);
    }

    .example-info p {
        margin: 0;
        font-size: 13px;
    }

    .code-editor {
        background: #0d0d1a;
        border-radius: 8px;
        overflow: hidden;
        border: 1px solid var(--border);
    }

    .editor-header {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 14px;
        background: rgba(255, 255, 255, 0.03);
        border-bottom: 1px solid var(--border);
    }

    .editor-dots {
        display: flex;
        gap: 6px;
    }

    .dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
    }

    .dot.red {
        background: #ff5f56;
    }
    .dot.yellow {
        background: #ffbd2e;
    }
    .dot.green {
        background: #27ca40;
    }

    .editor-title {
        font-size: 12px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    .editor-content {
        margin: 0;
        padding: 16px;
        overflow-x: auto;
        max-height: 500px;
        overflow-y: auto;
    }

    .editor-content code {
        font-family: var(--font-mono);
        font-size: 13px;
        line-height: 1.6;
        color: #e0e0e0;
        white-space: pre;
    }

    /* Syntax highlighting */
    .editor-content :global(.hl-comment) {
        color: #6a9955;
    }

    .editor-content :global(.hl-string) {
        color: #ce9178;
    }

    .editor-content :global(.hl-keyword) {
        color: #569cd6;
    }

    .editor-content :global(.hl-tag) {
        color: #4ec9b0;
    }

    .editor-content :global(.hl-property) {
        color: #9cdcfe;
    }

    .editor-content :global(.hl-number) {
        color: #b5cea8;
    }

    /* Live Preview Section */
    .preview-section {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 24px;
    }

    .preview-mode-tabs {
        display: flex;
        gap: 8px;
        margin-bottom: 20px;
    }

    .preview-mode-tab {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        background: transparent;
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        font-size: 13px;
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .preview-mode-tab:hover {
        border-color: var(--accent);
        color: var(--text);
    }

    .preview-mode-tab.active {
        background: var(--accent);
        border-color: var(--accent);
        color: var(--bg);
    }

    .preview-mode-tab :global(svg) {
        width: 16px;
        height: 16px;
    }

    .preview-controls {
        display: flex;
        flex-direction: column;
        gap: 16px;
        margin-bottom: 20px;
    }

    .preview-form {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .preview-row {
        display: flex;
        gap: 12px;
    }

    .preview-input-group {
        display: flex;
        flex-direction: column;
        gap: 6px;
        flex: 1;
    }

    .preview-input-group.half {
        flex: 1;
    }

    .preview-input-group label {
        font-size: 12px;
        color: var(--text-muted);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .preview-input,
    .preview-select {
        padding: 10px 14px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 14px;
    }

    .preview-select {
        cursor: pointer;
    }

    .preview-input:focus,
    .preview-select:focus {
        outline: none;
        border-color: var(--accent);
    }

    .preview-input::placeholder {
        color: var(--text-muted);
    }

    .preview-hint {
        font-size: 12px !important;
        color: var(--text-muted) !important;
        margin: 0 !important;
    }

    .preview-hint a {
        color: var(--accent);
        text-decoration: none;
    }

    .preview-hint a:hover {
        text-decoration: underline;
    }

    .preview-actions {
        display: flex;
        gap: 12px;
        align-items: center;
    }

    .preview-btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 20px;
        background: var(--accent);
        border: none;
        border-radius: 6px;
        color: var(--bg);
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.15s ease;
        white-space: nowrap;
    }

    .preview-btn:hover:not(:disabled) {
        filter: brightness(1.1);
    }

    .preview-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .preview-btn.secondary {
        background: transparent;
        border: 1px solid var(--border);
        color: var(--text-muted);
    }

    .preview-btn.secondary:hover {
        border-color: var(--error);
        color: var(--error);
    }

    .preview-btn :global(svg) {
        width: 16px;
        height: 16px;
    }

    .live-preview-container {
        background: #0d0d1a;
        border-radius: 8px;
        overflow: hidden;
        border: 1px solid var(--border);
        animation: fadeIn 0.3s ease;
    }

    .preview-header {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 10px 14px;
        background: rgba(255, 255, 255, 0.03);
        border-bottom: 1px solid var(--border);
        font-size: 13px;
        color: var(--text-muted);
    }

    .preview-header :global(svg) {
        color: var(--accent);
    }

    .preview-code {
        margin-left: auto;
        font-family: var(--font-mono);
        color: var(--accent);
        font-size: 12px;
    }

    .preview-terminal {
        height: 400px;
        width: 100%;
    }

    /* Images Grid */
    .images-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 20px;
        margin-top: 16px;
    }

    .image-group {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 16px;
    }

    .image-group h4 {
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-muted);
        margin: 0 0 12px 0;
    }

    .image-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .image-item {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 13px;
    }

    .image-item code {
        font-family: var(--font-mono);
        color: var(--accent);
        background: var(--bg-tertiary);
        padding: 2px 6px;
        border-radius: 4px;
        font-size: 12px;
    }

    .image-item span {
        color: var(--text-muted);
    }

    .options-table {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        overflow: hidden;
    }

    .option-row {
        display: grid;
        grid-template-columns: 140px 150px 130px 1fr;
        padding: 12px 16px;
        border-bottom: 1px solid var(--border);
        font-size: 13px;
    }

    .option-row:last-child {
        border-bottom: none;
    }

    .option-row.header {
        background: var(--bg-tertiary);
        font-weight: 600;
        color: var(--text);
        text-transform: uppercase;
        font-size: 11px;
        letter-spacing: 0.5px;
    }

    .option-name code {
        font-family: var(--font-mono);
        color: var(--accent);
        font-size: 12px;
    }

    .option-type {
        color: var(--text-muted);
        font-family: var(--font-mono);
        font-size: 11px;
    }

    .option-default {
        color: var(--text-muted);
        font-family: var(--font-mono);
        font-size: 11px;
    }

    .option-desc {
        color: var(--text);
    }

    .api-group {
        margin-bottom: 24px;
    }

    .api-group h3 {
        font-size: 14px;
        margin: 0 0 12px 0;
        color: var(--text);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .cli-commands {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .command-item {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 6px;
        padding: 12px 16px;
    }

    .command-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
    }

    .command {
        font-family: var(--font-mono);
        font-size: 13px;
        color: var(--accent);
        background: var(--bg-tertiary);
        padding: 4px 8px;
        border-radius: 4px;
    }

    .command-desc {
        color: var(--text-muted);
        font-size: 13px;
    }

    .browser-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
        gap: 12px;
    }

    .browser-item {
        display: flex;
        align-items: center;
        gap: 10px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 6px;
        padding: 12px 16px;
        font-size: 14px;
        color: var(--text);
    }

    .browser-item :global(svg) {
        color: var(--success);
    }

    .faq-list {
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .faq-item {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 20px;
    }

    .faq-item h4 {
        font-size: 14px;
        margin: 0 0 8px 0;
        color: var(--text);
    }

    .faq-item p {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0;
        line-height: 1.6;
    }

    .faq-item a {
        color: var(--accent);
        text-decoration: none;
    }

    .faq-item a:hover {
        text-decoration: underline;
    }

    @media (max-width: 768px) {
        .docs-page {
            padding: 16px;
        }

        .docs-header h1 {
            font-size: 28px;
        }

        .option-row {
            grid-template-columns: 1fr;
            gap: 4px;
        }

        .option-row.header {
            display: none;
        }

        .option-name::before {
            content: "Option: ";
            color: var(--text-muted);
            font-size: 10px;
        }

        .option-type::before {
            content: "Type: ";
            color: var(--text-muted);
        }

        .option-default::before {
            content: "Default: ";
            color: var(--text-muted);
        }

        .command-header {
            flex-direction: column;
            align-items: flex-start;
            gap: 8px;
        }

        .code-block {
            flex-direction: column;
        }

        .copy-btn {
            align-self: flex-end;
        }

        .example-header {
            flex-direction: column;
        }

        .example-tabs {
            overflow-x: auto;
            flex-wrap: nowrap;
            padding-bottom: 12px;
        }

        .example-tab {
            white-space: nowrap;
        }

        .preview-mode-tabs {
            flex-direction: column;
        }

        .preview-row {
            flex-direction: column;
        }

        .preview-terminal {
            height: 300px;
        }
    }

    @media (max-width: 480px) {
        .feature-grid {
            grid-template-columns: 1fr;
        }

        .browser-grid {
            grid-template-columns: 1fr 1fr;
        }

        .images-grid {
            grid-template-columns: 1fr;
        }
    }
</style>
