<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    const dispatch = createEventDispatcher<{
        navigate: { view: string };
    }>();


    // SEO metadata for docs page
    onMount(() => {
        document.title = "Documentation | Rexec - Terminal as a Service";
        
        // Update meta tags
        const metaDescription = document.querySelector('meta[name="description"]');
        if (metaDescription) {
            metaDescription.setAttribute("content", "Complete documentation for Rexec - learn about instant cloud terminals, BYOS agents, CLI tools, security features, and API integration.");
        }
        
        const ogTitle = document.querySelector('meta[property="og:title"]');
        if (ogTitle) {
            ogTitle.setAttribute("content", "Documentation | Rexec - Terminal as a Service");
        }
        
        const ogDescription = document.querySelector('meta[property="og:description"]');
        if (ogDescription) {
            ogDescription.setAttribute("content", "Complete documentation for Rexec - learn about instant cloud terminals, BYOS agents, CLI tools, security features, and API integration.");
        }
        
        const twitterTitle = document.querySelector('meta[name="twitter:title"]');
        if (twitterTitle) {
            twitterTitle.setAttribute("content", "Documentation | Rexec - Terminal as a Service");
        }
        
        const twitterDescription = document.querySelector('meta[name="twitter:description"]');
        if (twitterDescription) {
            twitterDescription.setAttribute("content", "Complete documentation for Rexec - learn about instant cloud terminals, BYOS agents, CLI tools, security features, and API integration.");
        }
    });

    // Feature sections
    const features = [
        {
            icon: "bolt",
            title: "Instant Terminals",
            description: "Spin up Linux terminals in seconds with no setup required. Choose from Ubuntu, Debian, Arch, or Kali Linux."
        },
        {
            icon: "shield",
            title: "Network Isolation",
            description: "Each terminal runs in its own isolated Docker container with dedicated networking for maximum security."
        },
        {
            icon: "cloud",
            title: "Persistent Storage",
            description: "Your files and configurations persist across sessions. Pick up exactly where you left off."
        },
        {
            icon: "users",
            title: "Collaborative Sessions",
            description: "Share terminals in real-time with team members. Perfect for pair programming and live debugging."
        },
        {
            icon: "agent",
            title: "BYOS Agent",
            description: "Connect your own servers, VMs, or local machines to the platform using the rexec-agent."
        },
        {
            icon: "terminal",
            title: "CLI & TUI",
            description: "Full-featured command line tools for power users. Manage terminals without leaving your shell."
        }
    ];

    const securityFeatures = [
        {
            icon: "lock",
            title: "JWT Authentication",
            description: "Secure token-based authentication with automatic expiration and refresh."
        },
        {
            icon: "shield",
            title: "MFA Support",
            description: "Two-factor authentication using TOTP for additional account security."
        },
        {
            icon: "server",
            title: "Container Isolation",
            description: "Docker security profiles with no-new-privileges flag prevent privilege escalation."
        },
        {
            icon: "key",
            title: "SSH Key Management",
            description: "Store and inject SSH keys securely into your terminals for seamless git operations."
        },
        {
            icon: "data",
            title: "Encrypted Storage",
            description: "Sensitive data like tokens and SSH keys are encrypted at rest using AES-256."
        },
        {
            icon: "clock",
            title: "Session Timeouts",
            description: "Automatic session expiration and idle cleanup protect against abandoned sessions."
        }
    ];

    const guides = [
        {
            title: "CLI Documentation",
            description: "Install and use the rexec command line tool",
            href: "/docs/cli",
            icon: "cli"
        },
        {
            title: "Agent Setup",
            description: "Connect your own machines to the platform",
            href: "/docs/agent",
            icon: "agent"
        },
        {
            title: "Use Cases",
            description: "Explore what you can build with Rexec",
            href: "/use-cases",
            icon: "rocket"
        },
        {
            title: "AI Tools Guide",
            description: "Integrate with AI coding assistants",
            href: "/guides",
            icon: "ai"
        },
        {
            title: "FAQ",
            description: "Frequently asked questions about Rexec",
            href: "#faq",
            icon: "question"
        }
    ];
</script>

<div class="docs-page">
    <div class="docs-header">
        <h1>Documentation</h1>
        <p class="docs-subtitle">Everything you need to know about Rexec - Terminal as a Service</p>
    </div>

    <!-- Quick Links -->
    <section class="section">
        <h2><StatusIcon status="book" size={20} /> Quick Start Guides</h2>
        <div class="guides-grid">
            {#each guides as guide}
                <a href={guide.href} class="guide-card">
                    <div class="guide-icon">
                        <StatusIcon status={guide.icon} size={24} />
                    </div>
                    <div class="guide-content">
                        <h3>{guide.title}</h3>
                        <p>{guide.description}</p>
                    </div>
                    <StatusIcon status="chevron-right" size={16} />
                </a>
            {/each}
        </div>
    </section>

    <!-- Core Features -->
    <section class="section">
        <h2><StatusIcon status="sparkles" size={20} /> Core Features</h2>
        <div class="features-grid">
            {#each features as feature}
                <div class="feature-card">
                    <div class="feature-icon">
                        <StatusIcon status={feature.icon} size={24} />
                    </div>
                    <h3>{feature.title}</h3>
                    <p>{feature.description}</p>
                </div>
            {/each}
        </div>
    </section>

    <!-- Security -->
    <section class="section">
        <h2><StatusIcon status="shield" size={20} /> Security & Compliance</h2>
        <div class="security-intro">
            <p>
                Rexec is built with security at its core. Every terminal runs in an isolated container 
                with its own network namespace, ensuring complete separation between users.
            </p>
        </div>
        <div class="security-grid">
            {#each securityFeatures as feature}
                <div class="security-card">
                    <div class="security-icon">
                        <StatusIcon status={feature.icon} size={18} />
                    </div>
                    <div class="security-content">
                        <h4>{feature.title}</h4>
                        <p>{feature.description}</p>
                    </div>
                </div>
            {/each}
        </div>
    </section>

    <!-- API Reference -->
    <section class="section">
        <h2><StatusIcon status="code" size={20} /> API Reference</h2>
        <div class="api-section">
            <p>
                Rexec provides a RESTful API for programmatic access to all platform features. 
                Generate API tokens from your account settings to authenticate requests.
            </p>
            <div class="api-endpoints">
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <code>/api/containers</code>
                    <span class="desc">List your terminals</span>
                </div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <code>/api/containers</code>
                    <span class="desc">Create a new terminal</span>
                </div>
                <div class="endpoint">
                    <span class="method delete">DELETE</span>
                    <code>/api/containers/:id</code>
                    <span class="desc">Delete a terminal</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <code>/api/snippets</code>
                    <span class="desc">List your snippets</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <code>/api/agents</code>
                    <span class="desc">List your agents</span>
                </div>
                <div class="endpoint">
                    <span class="method websocket">WS</span>
                    <code>/ws/containers/:id/terminal</code>
                    <span class="desc">Terminal WebSocket</span>
                </div>
            </div>
            <div class="api-auth">
                <h4>Authentication</h4>
                <pre><code>curl -H "Authorization: Bearer YOUR_API_TOKEN" \
     https://rexec.pipeops.io/api/containers</code></pre>
            </div>
        </div>
    </section>

    <!-- Architecture -->
    <section class="section">
        <h2><StatusIcon status="workflow" size={20} /> Architecture Overview</h2>
        <div class="architecture">
            <div class="arch-diagram">
                <div class="arch-layer">
                    <span class="layer-label">Frontend</span>
                    <div class="arch-items">
                        <span class="arch-item">Svelte 5</span>
                        <span class="arch-item">xterm.js</span>
                        <span class="arch-item">PWA</span>
                    </div>
                </div>
                <div class="arch-arrow">↓</div>
                <div class="arch-layer">
                    <span class="layer-label">API Layer</span>
                    <div class="arch-items">
                        <span class="arch-item">Go + Gin</span>
                        <span class="arch-item">WebSocket</span>
                        <span class="arch-item">REST</span>
                    </div>
                </div>
                <div class="arch-arrow">↓</div>
                <div class="arch-layer">
                    <span class="layer-label">Container Runtime</span>
                    <div class="arch-items">
                        <span class="arch-item">Docker SDK</span>
                        <span class="arch-item">Isolated Networks</span>
                        <span class="arch-item">Volume Mounts</span>
                    </div>
                </div>
                <div class="arch-arrow">↓</div>
                <div class="arch-layer">
                    <span class="layer-label">Data Layer</span>
                    <div class="arch-items">
                        <span class="arch-item">PostgreSQL</span>
                        <span class="arch-item">S3 Storage</span>
                        <span class="arch-item">Redis (optional)</span>
                    </div>
                </div>
            </div>
        </div>
    </section>

    <!-- FAQ -->
    <section class="section" id="faq">
        <h2><StatusIcon status="question" size={20} /> Frequently Asked Questions</h2>
        <div class="faq-grid">
            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="question" size={16} />
                    <span>Is this related to the old rexec protocol?</span>
                </summary>
                <div class="faq-answer">
                    <p><strong>No, not at all!</strong> We understand the concern — the legacy <code>rexec</code> (Remote Execution) protocol from the 1980s is indeed deprecated and insecure. Our Rexec is a completely different, modern product.</p>
                    <p>We chose the name "Rexec" as shorthand for <strong>"Remote Execution"</strong> — which perfectly describes what we do: execute commands on remote cloud terminals. Think of it as reclaiming a cool name for a secure, modern use case.</p>
                    <p>Our platform uses <strong>TLS encryption</strong>, <strong>JWT authentication</strong>, <strong>container isolation</strong>, and follows modern security best practices. It's as secure as any enterprise cloud platform.</p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="lock" size={16} />
                    <span>How is Rexec secure?</span>
                </summary>
                <div class="faq-answer">
                    <p>Security is built into every layer:</p>
                    <ul>
                        <li><strong>TLS/HTTPS</strong> — All connections are encrypted end-to-end</li>
                        <li><strong>Container Isolation</strong> — Each terminal runs in its own isolated container</li>
                        <li><strong>OAuth + MFA</strong> — Secure authentication with multi-factor support</li>
                        <li><strong>No Shared State</strong> — Your terminal is yours alone</li>
                        <li><strong>Automatic Cleanup</strong> — Sessions are destroyed when you're done</li>
                    </ul>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="bolt" size={16} />
                    <span>What can I use Rexec for?</span>
                </summary>
                <div class="faq-answer">
                    <p>Rexec is perfect for:</p>
                    <ul>
                        <li><strong>Learning Linux</strong> — Safe environment to experiment</li>
                        <li><strong>Development</strong> — Instant dev environments with pre-installed tools</li>
                        <li><strong>DevOps</strong> — Test scripts, debug issues, run CI/CD tasks</li>
                        <li><strong>Teaching</strong> — Provide students with instant terminals</li>
                        <li><strong>AI/ML Workloads</strong> — Access GPU-enabled environments</li>
                        <li><strong>Remote Access</strong> — Connect to your servers from anywhere</li>
                    </ul>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="save" size={16} />
                    <span>Is my data persistent?</span>
                </summary>
                <div class="faq-answer">
                    <p>Terminals persist between sessions — your files and work are saved until you explicitly delete the terminal. However, once you delete a terminal, all data is permanently removed.</p>
                    <p>For long-term storage, we recommend:</p>
                    <ul>
                        <li><strong>Push to Git</strong> — Commit your work to a remote repository</li>
                        <li><strong>Use Rexec Agents</strong> — Connect your own infrastructure for full control</li>
                        <li><strong>Export files</strong> — Download important files before deleting</li>
                    </ul>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="pricing" size={16} />
                    <span>What does it cost?</span>
                </summary>
                <div class="faq-answer">
                    <p>Rexec offers a generous free tier for personal use. For teams and businesses, we offer Pro and Enterprise plans with additional features like:</p>
                    <ul>
                        <li><strong>More concurrent terminals</strong></li>
                        <li><strong>Longer session times</strong></li>
                        <li><strong>Priority support</strong></li>
                        <li><strong>Custom images</strong></li>
                        <li><strong>SSO/SAML integration</strong></li>
                    </ul>
                    <p>Check our <a href="/pricing">pricing page</a> for details.</p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="agent" size={16} />
                    <span>Can I connect my own servers?</span>
                </summary>
                <div class="faq-answer">
                    <p>Yes! With the <strong>Rexec Agent</strong>, you can connect any Linux or macOS machine to the platform. This enables:</p>
                    <ul>
                        <li><strong>BYOS (Bring Your Own Server)</strong> — Use your own hardware</li>
                        <li><strong>On-premises access</strong> — Securely access internal servers</li>
                        <li><strong>Custom environments</strong> — Full control over your setup</li>
                    </ul>
                    <p>See the <a href="/docs/agent">Agent Setup Guide</a> for installation instructions.</p>
                </div>
            </details>
        </div>
    </section>

    <!-- Support -->
    <section class="section support-section">
        <h2><StatusIcon status="chat" size={20} /> Need Help?</h2>
        <div class="support-cards">
            <a href="https://github.com/pipeops-dev/rexec/issues" target="_blank" rel="noopener" class="support-card">
                <StatusIcon status="bug" size={24} />
                <h4>Report an Issue</h4>
                <p>Found a bug? Let us know on GitHub.</p>
            </a>
            <a href="mailto:support@pipeops.io" class="support-card">
                <StatusIcon status="chat" size={24} />
                <h4>Contact Support</h4>
                <p>Email our support team for help.</p>
            </a>
            <a href="/pricing" class="support-card">
                <StatusIcon status="pricing" size={24} />
                <h4>Upgrade Plan</h4>
                <p>Get priority support with Pro or Enterprise.</p>
            </a>
        </div>
    </section>
</div>

<style>
    .docs-page {
        max-width: 1000px;
        margin: 0 auto;
        padding: 40px 20px;
    }

    .docs-header {
        text-align: center;
        margin-bottom: 48px;
    }

    .docs-header h1 {
        font-size: 32px;
        font-weight: 700;
        margin: 0 0 12px;
        color: var(--text);
    }

    .docs-subtitle {
        font-size: 16px;
        color: var(--text-muted);
        margin: 0;
    }

    .section {
        margin-bottom: 48px;
    }

    .section h2 {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 20px;
        font-weight: 600;
        margin: 0 0 24px;
        padding-bottom: 12px;
        border-bottom: 1px solid var(--border);
        color: var(--text);
    }

    /* Quick Start Guides */
    .guides-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
        gap: 16px;
    }

    .guide-card {
        display: flex;
        align-items: center;
        gap: 16px;
        padding: 16px 20px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        text-decoration: none;
        color: var(--text);
        transition: all 0.2s;
    }

    .guide-card:hover {
        border-color: var(--accent);
        background: var(--bg-tertiary);
    }

    .guide-icon {
        flex-shrink: 0;
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--accent-dim);
        border-radius: 8px;
        color: var(--accent);
    }

    .guide-content {
        flex: 1;
    }

    .guide-content h3 {
        font-size: 14px;
        font-weight: 600;
        margin: 0 0 4px;
    }

    .guide-content p {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0;
    }

    /* Features Grid */
    .features-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
        gap: 20px;
    }

    .feature-card {
        padding: 24px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
    }

    .feature-icon {
        width: 44px;
        height: 44px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--accent-dim);
        border-radius: 8px;
        color: var(--accent);
        margin-bottom: 16px;
    }

    .feature-card h3 {
        font-size: 15px;
        font-weight: 600;
        margin: 0 0 8px;
        color: var(--text);
    }

    .feature-card p {
        font-size: 13px;
        color: var(--text-secondary);
        margin: 0;
        line-height: 1.5;
    }

    /* Security */
    .security-intro {
        margin-bottom: 24px;
        padding: 16px;
        background: var(--accent-dim);
        border: 1px solid var(--accent);
        border-radius: 8px;
    }

    .security-intro p {
        margin: 0;
        font-size: 14px;
        color: var(--text-secondary);
        line-height: 1.6;
    }

    .security-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        gap: 16px;
    }

    .security-card {
        display: flex;
        gap: 12px;
        padding: 16px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
    }

    .security-icon {
        flex-shrink: 0;
        width: 36px;
        height: 36px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 255, 65, 0.1);
        border-radius: 6px;
        color: var(--accent);
    }

    .security-content h4 {
        font-size: 13px;
        font-weight: 600;
        margin: 0 0 4px;
        color: var(--text);
    }

    .security-content p {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0;
        line-height: 1.4;
    }

    /* API Section */
    .api-section {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 24px;
    }

    .api-section > p {
        margin: 0 0 20px;
        font-size: 14px;
        color: var(--text-secondary);
        line-height: 1.5;
    }

    .api-endpoints {
        display: flex;
        flex-direction: column;
        gap: 8px;
        margin-bottom: 20px;
    }

    .endpoint {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 14px;
        background: var(--bg-tertiary);
        border-radius: 6px;
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .method {
        padding: 2px 8px;
        border-radius: 4px;
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
    }

    .method.get { background: rgba(0, 150, 255, 0.2); color: #0096ff; }
    .method.post { background: rgba(0, 255, 65, 0.2); color: var(--accent); }
    .method.delete { background: rgba(255, 60, 60, 0.2); color: #ff3c3c; }
    .method.websocket { background: rgba(180, 100, 255, 0.2); color: #b464ff; }

    .endpoint code {
        flex: 1;
        color: var(--text);
    }

    .endpoint .desc {
        color: var(--text-muted);
        font-size: 12px;
    }

    .api-auth {
        border-top: 1px solid var(--border);
        padding-top: 16px;
    }

    .api-auth h4 {
        font-size: 13px;
        font-weight: 600;
        margin: 0 0 12px;
        color: var(--text);
    }

    .api-auth pre {
        margin: 0;
        padding: 14px;
        background: var(--bg-tertiary);
        border-radius: 6px;
        overflow-x: auto;
    }

    .api-auth code {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--accent);
    }

    /* Architecture */
    .architecture {
        padding: 24px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
    }

    .arch-diagram {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 12px;
    }

    .arch-layer {
        width: 100%;
        max-width: 400px;
        padding: 16px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
        text-align: center;
    }

    .layer-label {
        display: block;
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--accent);
        margin-bottom: 10px;
    }

    .arch-items {
        display: flex;
        justify-content: center;
        gap: 8px;
        flex-wrap: wrap;
    }

    .arch-item {
        padding: 4px 10px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 4px;
        font-size: 12px;
        color: var(--text-secondary);
    }

    .arch-arrow {
        font-size: 20px;
        color: var(--text-muted);
    }

    /* Support */
    .support-section {
        text-align: center;
    }

    .support-cards {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 16px;
    }

    .support-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 12px;
        padding: 24px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        text-decoration: none;
        color: var(--text);
        transition: all 0.2s;
    }

    .support-card:hover {
        border-color: var(--accent);
        background: var(--bg-tertiary);
    }

    .support-card h4 {
        font-size: 14px;
        font-weight: 600;
        margin: 0;
    }

    .support-card p {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0;
    }

    @media (max-width: 768px) {
        .docs-page {
            padding: 24px 16px;
        }

        .docs-header h1 {
            font-size: 24px;
        }

        .guides-grid,
        .features-grid,
        .security-grid {
            grid-template-columns: 1fr;
        }

        .endpoint {
            flex-wrap: wrap;
        }

        .endpoint .desc {
            width: 100%;
            margin-top: 4px;
        }
    }

    /* FAQ */
    .faq-grid {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .faq-item {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        overflow: hidden;
    }

    .faq-question {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px;
        cursor: pointer;
        font-weight: 500;
        color: var(--text);
        list-style: none;
        transition: background 0.2s;
    }

    .faq-question::-webkit-details-marker {
        display: none;
    }

    .faq-question:hover {
        background: var(--bg-tertiary);
    }

    .faq-item[open] .faq-question {
        border-bottom: 1px solid var(--border);
    }

    .faq-answer {
        padding: 16px;
        color: var(--text-muted);
        font-size: 14px;
        line-height: 1.6;
    }

    .faq-answer p {
        margin: 0 0 12px;
    }

    .faq-answer p:last-child {
        margin-bottom: 0;
    }

    .faq-answer ul {
        margin: 12px 0;
        padding-left: 20px;
    }

    .faq-answer li {
        margin-bottom: 8px;
    }

    .faq-answer code {
        background: var(--bg-tertiary);
        padding: 2px 6px;
        border-radius: 4px;
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .faq-answer a {
        color: var(--accent);
        text-decoration: none;
    }

    .faq-answer a:hover {
        text-decoration: underline;
    }
</style>
