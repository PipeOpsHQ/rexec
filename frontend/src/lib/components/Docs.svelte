<script lang="ts">
    import { onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    // SEO metadata for docs page
    onMount(() => {
        document.title = "Documentation | Rexec - Terminal as a Service";

        // Update meta tags
        const metaDescription = document.querySelector(
            'meta[name="description"]',
        );
        if (metaDescription) {
            metaDescription.setAttribute(
                "content",
                "Complete documentation for Rexec - learn about instant cloud terminals, BYOS agents, CLI tools, security features, and API integration.",
            );
        }

        const ogTitle = document.querySelector('meta[property="og:title"]');
        if (ogTitle) {
            ogTitle.setAttribute(
                "content",
                "Documentation | Rexec - Terminal as a Service",
            );
        }

        const ogDescription = document.querySelector(
            'meta[property="og:description"]',
        );
        if (ogDescription) {
            ogDescription.setAttribute(
                "content",
                "Complete documentation for Rexec - learn about instant cloud terminals, BYOS agents, CLI tools, security features, and API integration.",
            );
        }

        const twitterTitle = document.querySelector(
            'meta[name="twitter:title"]',
        );
        if (twitterTitle) {
            twitterTitle.setAttribute(
                "content",
                "Documentation | Rexec - Terminal as a Service",
            );
        }

        const twitterDescription = document.querySelector(
            'meta[name="twitter:description"]',
        );
        if (twitterDescription) {
            twitterDescription.setAttribute(
                "content",
                "Complete documentation for Rexec - learn about instant cloud terminals, BYOS agents, CLI tools, security features, and API integration.",
            );
        }

        // Handle hash fragments for anchor links (e.g., #faq)
        if (window.location.hash) {
            // Use setTimeout to ensure the DOM is fully rendered
            setTimeout(() => {
                const targetId = window.location.hash.substring(1);
                const targetElement = document.getElementById(targetId);
                if (targetElement) {
                    targetElement.scrollIntoView({
                        behavior: "smooth",
                        block: "start",
                    });
                }
            }, 100);
        }
    });

    // Feature sections
    const features = [
        {
            icon: "bolt",
            title: "Instant Terminals",
            description:
                "Spin up Linux terminals in seconds with no setup required. Choose from Ubuntu, Debian, Arch, or Kali Linux.",
        },
        {
            icon: "shield",
            title: "Network Isolation",
            description:
                "Each terminal runs in its own isolated Docker container with dedicated networking for maximum security.",
        },
        {
            icon: "cloud",
            title: "Persistent Storage",
            description:
                "Your files and configurations persist across sessions. Pick up exactly where you left off.",
        },
        {
            icon: "users",
            title: "Collaborative Sessions",
            description:
                "Share terminals in real-time with team members. Perfect for pair programming and live debugging.",
        },
        {
            icon: "agent",
            title: "BYOS Agent",
            description:
                "Connect your own servers, VMs, or local machines to the platform using the rexec-agent.",
        },
        {
            icon: "terminal",
            title: "CLI & TUI",
            description:
                "Full-featured command line tools for power users. Manage terminals without leaving your shell.",
        },
    ];

    const securityFeatures = [
        {
            icon: "lock",
            title: "JWT Authentication",
            description:
                "Secure token-based authentication with automatic expiration and refresh.",
        },
        {
            icon: "shield",
            title: "MFA Support",
            description:
                "Two-factor authentication using TOTP. Lock individual terminals with MFA for extra protection.",
        },
        {
            icon: "server",
            title: "Container Isolation",
            description:
                "Docker security profiles with no-new-privileges flag prevent privilege escalation.",
        },
        {
            icon: "key",
            title: "SSH Key Management",
            description:
                "Store and inject SSH keys securely into your terminals for seamless git operations.",
        },
        {
            icon: "data",
            title: "Encrypted Storage",
            description:
                "Sensitive data like tokens and SSH keys are encrypted at rest using AES-256.",
        },
        {
            icon: "clock",
            title: "Session Timeouts",
            description:
                "Automatic session expiration and idle cleanup protect against abandoned sessions.",
        },
    ];

    const guides = [
        {
            title: "CLI Documentation",
            description: "Install and use the rexec command line tool",
            href: "/docs/cli",
            icon: "cli",
        },
        {
            title: "Agent Setup",
            description: "Connect your own machines to the platform",
            href: "/docs/agent",
            icon: "agent",
        },
        {
            title: "Embed Widget",
            description: "Add a cloud terminal to any website",
            href: "/docs/embed",
            icon: "code",
        },
        {
            title: "Use Cases",
            description: "Explore what you can build with Rexec",
            href: "/use-cases",
            icon: "rocket",
        },
        {
            title: "AI Tools Guide",
            description: "Integrate with AI coding assistants",
            href: "/guides",
            icon: "ai",
        },
        {
            title: "FAQ",
            description: "Frequently asked questions about Rexec",
            href: "#faq",
            icon: "question",
        },
    ];
</script>

<div class="docs-page">
    <div class="docs-header">
        <h1>Documentation</h1>
        <p class="docs-subtitle">
            Everything you need to know about Rexec - Terminal as a Service
        </p>
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
                Rexec is built with security at its core. Every terminal runs in
                an isolated container with its own network namespace, ensuring
                complete separation between users.
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
    <section class="section" id="api">
        <h2><StatusIcon status="code" size={20} /> API Reference</h2>
        <div class="api-section">
            <p>
                Rexec provides a RESTful API for programmatic access to all
                platform features. Generate API tokens from your account
                settings to authenticate requests.
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
                <pre><code
                        >curl -H "Authorization: Bearer YOUR_API_TOKEN" \
     https://rexec.sh/api/containers</code
                    ></pre>
            </div>
        </div>
    </section>

    <!-- SDKs -->
    <section class="section" id="sdks">
        <h2><StatusIcon status="code" size={20} /> Official SDKs</h2>
        <div class="sdk-section">
            <p>
                Use our official SDKs to programmatically interact with Rexec in your preferred language.
                All SDKs provide container management, file operations, and interactive terminal support.
            </p>
            <div class="sdk-grid">
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk/go" target="_blank" class="sdk-card">
                    <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                    <div class="sdk-info">
                        <h4>Go</h4>
                        <code>go get github.com/PipeOpsHQ/rexec-go</code>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk/js" target="_blank" class="sdk-card">
                    <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                    <div class="sdk-info">
                        <h4>JavaScript / TypeScript</h4>
                        <code>npm install @pipeopshq/rexec</code>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk/python" target="_blank" class="sdk-card">
                    <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                    <div class="sdk-info">
                        <h4>Python</h4>
                        <code>pip install rexec</code>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk/rust" target="_blank" class="sdk-card">
                    <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                    <div class="sdk-info">
                        <h4>Rust</h4>
                        <code>cargo add rexec</code>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk/ruby" target="_blank" class="sdk-card">
                    <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                    <div class="sdk-info">
                        <h4>Ruby</h4>
                        <code>gem install rexec</code>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk/java" target="_blank" class="sdk-card">
                    <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                    <div class="sdk-info">
                        <h4>Java</h4>
                        <code>io.pipeops:rexec</code>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk/dotnet" target="_blank" class="sdk-card">
                    <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                    <div class="sdk-info">
                        <h4>C# / .NET</h4>
                        <code>dotnet add package Rexec</code>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk/php" target="_blank" class="sdk-card">
                    <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                    <div class="sdk-info">
                        <h4>PHP</h4>
                        <code>composer require pipeopshq/rexec</code>
                    </div>
                </a>
            </div>
            <div class="sdk-example">
                <h4>Quick Example (JavaScript)</h4>
                <pre><code>import {'{'} RexecClient {'}'} from '@pipeopshq/rexec';

const client = new RexecClient({'{'} 
  baseURL: 'https://rexec.sh', 
  token: 'YOUR_API_TOKEN' 
{'}'});

// Create a sandbox
const container = await client.containers.create({'{'} 
  image: 'ubuntu:24.04' 
{'}'});

// Execute commands
const result = await client.containers.exec(
  container.id, 
  'echo "Hello from Rexec!"'
);
console.log(result.stdout);

// Interactive terminal
const terminal = await client.terminal.connect(container.id);
terminal.onData((data) => console.log(data));
terminal.write('ls -la\\n');</code></pre>
            </div>
            <div class="sdk-links">
                <a href="https://github.com/PipeOpsHQ/rexec/blob/main/docs/SDK.md" target="_blank" class="sdk-link">
                    <StatusIcon status="book" size={14} />
                    Full SDK Documentation
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/blob/main/docs/SDK_GETTING_STARTED.md" target="_blank" class="sdk-link">
                    <StatusIcon status="rocket" size={14} />
                    Getting Started Guide
                </a>
            </div>
        </div>
    </section>

    <!-- Architecture -->
    <section class="section">
        <h2>
            <StatusIcon status="workflow" size={20} /> Architecture Overview
        </h2>
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
        <h2>
            <StatusIcon status="question" size={20} /> Frequently Asked Questions
        </h2>
        <div class="faq-grid">
            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="question" size={16} />
                    <span>Is Rexec a Virtual Machine?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        <strong>No!</strong> Rexec is <strong>not</strong> a VM
                        or hypervisor. It's a
                        <strong>Terminal as a Service</strong> platform that gives
                        you instant terminal access to:
                    </p>
                    <ul>
                        <li>
                            <strong>Cloud containers</strong> — Lightweight Linux
                            environments we provision for you
                        </li>
                        <li>
                            <strong>Your own machines</strong> — Any laptop, server,
                            or device running the Rexec Agent
                        </li>
                    </ul>
                    <p>
                        Think of it as a cloud-based terminal multiplexer, not a
                        virtualization platform. There's no hypervisor, no VM
                        overhead — just fast, secure terminal access from
                        anywhere.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="agent" size={16} />
                    <span>Can I access my laptop or server from anywhere?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        <strong>Yes!</strong> This is exactly what the
                        <strong>Rexec Agent</strong> is for. Install a lightweight
                        agent on any machine, and you get instant terminal access
                        from the Rexec dashboard — no SSH port exposure, no VPN, no
                        complex setup.
                    </p>
                    <p><strong>How it works:</strong></p>
                    <ul>
                        <li>
                            <strong>No inbound ports</strong> — The agent connects
                            outbound, so no firewall holes needed
                        </li>
                        <li>
                            <strong>Works from any browser</strong> — Access your
                            machine from desktop, tablet, or phone
                        </li>
                        <li>
                            <strong>AI-native</strong> — Continue coding with AI tools
                            without opening your laptop
                        </li>
                        <li>
                            <strong>Auto-reconnect</strong> — Connection drops are
                            handled automatically
                        </li>
                    </ul>
                    <p>Get started in 30 seconds:</p>
                    <div class="code-block">
                        <code
                            >curl -sSL https://rexec.sh/install-agent.sh | sudo bash</code
                        >
                    </div>
                    <p>
                        See the <a href="/docs/agent">Agent Setup Guide</a> for more
                        details.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="question" size={16} />
                    <span>Is this related to the old rexec protocol?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        <strong>No, not at all!</strong> We understand the
                        concern — the legacy <code>rexec</code> (Remote Execution)
                        protocol from the 1980s is indeed deprecated and insecure.
                        Our Rexec is a completely different, modern product.
                    </p>
                    <p>
                        We chose the name "Rexec" as shorthand for <strong
                            >"Remote Execution"</strong
                        > — which perfectly describes what we do: execute commands
                        on remote cloud terminals. Think of it as reclaiming a cool
                        name for a secure, modern use case.
                    </p>
                    <p>
                        Our platform uses <strong>TLS encryption</strong>,
                        <strong>JWT authentication</strong>,
                        <strong>container isolation</strong>, and follows modern
                        security best practices. It's as secure as any
                        enterprise cloud platform.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="lock" size={16} />
                    <span>How is Rexec secure?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        Security is built into every layer of Rexec. We follow
                        industry best practices and enterprise-grade security
                        standards:
                    </p>

                    <p><strong>Encryption & Transport Security</strong></p>
                    <ul>
                        <li>
                            <strong>TLS 1.3 everywhere</strong> — All connections
                            use modern TLS encryption. No plaintext, ever.
                        </li>
                        <li>
                            <strong>WebSocket over HTTPS</strong> — Terminal sessions
                            use encrypted WebSocket connections
                        </li>
                        <li>
                            <strong>Certificate pinning</strong> — Agent connections
                            verify server certificates
                        </li>
                    </ul>

                    <p><strong>Authentication & Access Control</strong></p>
                    <ul>
                        <li>
                            <strong>OAuth 2.0 + PKCE</strong> — Secure authentication
                            flow preventing token interception
                        </li>
                        <li>
                            <strong>Multi-Factor Authentication (MFA)</strong> — Optional
                            TOTP-based 2FA for your account
                        </li>
                        <li>
                            <strong>Short-lived JWT tokens</strong> — Access tokens
                            expire quickly; refresh tokens are securely stored
                        </li>
                        <li>
                            <strong>API tokens with scopes</strong> — Fine-grained
                            permissions for programmatic access
                        </li>
                    </ul>

                    <p><strong>Container Isolation</strong></p>
                    <ul>
                        <li>
                            <strong>Kernel namespaces</strong> — Each terminal runs
                            in isolated PID, network, and mount namespaces
                        </li>
                        <li>
                            <strong>Resource limits (cgroups)</strong> — CPU, memory,
                            and I/O are strictly limited per container
                        </li>
                        <li>
                            <strong>Read-only root filesystem</strong> — System files
                            cannot be modified
                        </li>
                        <li>
                            <strong>No privileged mode</strong> — Containers cannot
                            access host resources
                        </li>
                        <li>
                            <strong>Seccomp profiles</strong> — Dangerous syscalls
                            are blocked at the kernel level
                        </li>
                    </ul>

                    <p><strong>Data Protection</strong></p>
                    <ul>
                        <li>
                            <strong>No shared state</strong> — Your terminal is completely
                            isolated from other users
                        </li>
                        <li>
                            <strong>Encrypted at rest</strong> — Persistent data is
                            encrypted on disk
                        </li>
                        <li>
                            <strong>Automatic cleanup</strong> — Deleted terminals
                            are securely wiped
                        </li>
                        <li>
                            <strong>No telemetry or logging of commands</strong> —
                            We don't log what you type
                        </li>
                    </ul>

                    <p><strong>Infrastructure Security</strong></p>
                    <ul>
                        <li>
                            <strong>SOC 2 compliant hosting</strong> — Deployed on
                            enterprise-grade cloud infrastructure
                        </li>
                        <li>
                            <strong>Regular security audits</strong> — Penetration
                            testing and vulnerability assessments
                        </li>
                        <li>
                            <strong>DDoS protection</strong> — Built-in rate limiting
                            and traffic filtering
                        </li>
                        <li>
                            <strong>Incident response</strong> — 24/7 monitoring with
                            rapid response procedures
                        </li>
                    </ul>

                    <p>
                        For enterprise deployments, we also offer <strong
                            >self-hosted options</strong
                        >, <strong>SSO/SAML integration</strong>, and
                        <strong>audit logging</strong>.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="bolt" size={16} />
                    <span>What can I use Rexec for?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        Rexec gives you instant access to Linux environments
                        from anywhere. Here's what developers and teams are
                        using it for:
                    </p>

                    <p><strong>Software Development</strong></p>
                    <ul>
                        <li>
                            <strong>Instant dev environments</strong> — Spin up a
                            full Linux environment in seconds with your preferred
                            distro
                        </li>
                        <li>
                            <strong>Cross-platform testing</strong> — Test your code
                            on Ubuntu, Debian, Fedora, Alpine, and more
                        </li>
                        <li>
                            <strong>Dependency isolation</strong> — Keep project dependencies
                            separate without Docker on your machine
                        </li>
                        <li>
                            <strong>Pair programming</strong> — Share your terminal
                            session with teammates in real-time
                        </li>
                    </ul>

                    <p><strong>DevOps & SRE</strong></p>
                    <ul>
                        <li>
                            <strong>Debug production issues</strong> — Access a clean
                            environment to test and reproduce bugs
                        </li>
                        <li>
                            <strong>Script testing</strong> — Safely test deployment
                            scripts before running on production
                        </li>
                        <li>
                            <strong>CI/CD debugging</strong> — Reproduce pipeline
                            failures in an identical environment
                        </li>
                        <li>
                            <strong>Infrastructure management</strong> — Connect to
                            your servers via Rexec Agents for secure remote access
                        </li>
                    </ul>

                    <p><strong>Learning & Teaching</strong></p>
                    <ul>
                        <li>
                            <strong>Learn Linux safely</strong> — Experiment without
                            fear of breaking your machine
                        </li>
                        <li>
                            <strong>Coding bootcamps</strong> — Provide students with
                            identical, pre-configured environments
                        </li>
                        <li>
                            <strong>Workshops & demos</strong> — Live-code in a browser
                            without setup friction
                        </li>
                        <li>
                            <strong>Certifications prep</strong> — Practice for Linux
                            certifications (RHCSA, LFCS, etc.)
                        </li>
                    </ul>

                    <p><strong>AI & Machine Learning</strong></p>
                    <ul>
                        <li>
                            <strong>GPU-enabled environments</strong> — Access CUDA-ready
                            containers for ML workloads
                        </li>
                        <li>
                            <strong>Model training</strong> — Run long-running training
                            jobs in persistent terminals
                        </li>
                        <li>
                            <strong>Jupyter alternative</strong> — Full terminal access
                            for complex workflows
                        </li>
                    </ul>

                    <p><strong>Remote Access & BYOS</strong></p>
                    <ul>
                        <li>
                            <strong>Access your homelab</strong> — Connect to home
                            servers from anywhere via Agent
                        </li>
                        <li>
                            <strong>On-premises servers</strong> — Securely access
                            internal infrastructure through the browser
                        </li>
                        <li>
                            <strong>Multi-job workflows</strong> — Manage terminals
                            across multiple machines from one dashboard
                        </li>
                        <li>
                            <strong>Mobile access</strong> — SSH into your servers
                            from your phone or tablet
                        </li>
                    </ul>

                    <p><strong>Security & Pentesting</strong></p>
                    <ul>
                        <li>
                            <strong>Isolated testing</strong> — Run security tools
                            in a sandboxed environment
                        </li>
                        <li>
                            <strong>CTF competitions</strong> — Spin up Kali or ParrotOS
                            for capture-the-flag challenges
                        </li>
                        <li>
                            <strong>Malware analysis</strong> — Safely analyze suspicious
                            files in a throwaway container
                        </li>
                    </ul>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="save" size={16} />
                    <span>Is my data persistent?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        Terminals persist between sessions — your files and work
                        are saved until you explicitly delete the terminal.
                        However, once you delete a terminal, all data is
                        permanently removed.
                    </p>
                    <p>For long-term storage, we recommend:</p>
                    <ul>
                        <li>
                            <strong>Push to Git</strong> — Commit your work to a remote
                            repository
                        </li>
                        <li>
                            <strong>Use Rexec Agents</strong> — Connect your own infrastructure
                            for full control
                        </li>
                        <li>
                            <strong>Export files</strong> — Download important files
                            before deleting
                        </li>
                    </ul>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="pricing" size={16} />
                    <span>What does it cost?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        Rexec offers a generous free tier for personal use. <strong
                            >Sign in with PipeOps</strong
                        > to unlock:
                    </p>
                    <ul>
                        <li>
                            <strong>Persistent terminals</strong> — Your terminals
                            survive browser restarts
                        </li>
                        <li>
                            <strong>SSH key management</strong> — Store and inject
                            keys into terminals
                        </li>
                        <li>
                            <strong>Snippets library</strong> — Save and reuse commands
                        </li>
                        <li>
                            <strong>Agent connections</strong> — Connect your own
                            servers
                        </li>
                        <li>
                            <strong>Session history</strong> — Access your terminal
                            recordings
                        </li>
                    </ul>
                    <p>
                        For teams and power users, <strong>Pro</strong> and
                        <strong>Enterprise</strong> plans add:
                    </p>
                    <ul>
                        <li>
                            <strong>More concurrent terminals</strong> (10+ vs 3)
                        </li>
                        <li><strong>Unlimited agents</strong></li>
                        <li><strong>Priority support</strong></li>
                        <li><strong>Custom images</strong></li>
                        <li><strong>SSO/SAML integration</strong></li>
                    </ul>
                    <p>
                        Check our <a href="/pricing">pricing page</a> for details.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="agent" size={16} />
                    <span>Can I connect my own servers?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        Yes! With the <strong>Rexec Agent</strong>, you can
                        connect any Linux or macOS machine to the platform. This
                        enables:
                    </p>
                    <ul>
                        <li>
                            <strong>BYOS (Bring Your Own Server)</strong> — Use your
                            own hardware
                        </li>
                        <li>
                            <strong>On-premises access</strong> — Securely access
                            internal servers
                        </li>
                        <li>
                            <strong>Custom environments</strong> — Full control over
                            your setup
                        </li>
                    </ul>
                    <p>
                        See the <a href="/docs/agent">Agent Setup Guide</a> for installation
                        instructions.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="bolt" size={16} />
                    <span>Can I use Rexec with AI coding tools?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        <strong>Absolutely!</strong> Rexec was designed with AI-native
                        workflows in mind. Use cases include:
                    </p>
                    <ul>
                        <li>
                            <strong>Continue building remotely</strong> — Register
                            your dev machine as an agent and keep coding from any
                            AI CLI (Claude, Cursor, Windsurf, etc.) without opening
                            your laptop
                        </li>
                        <li>
                            <strong>Sandboxed execution</strong> — Let AI tools run
                            commands in isolated containers
                        </li>
                        <li>
                            <strong>Persistent environments</strong> — Your AI tools
                            can pick up where they left off
                        </li>
                    </ul>
                    <p>
                        Imagine being at dinner and continuing your build on
                        your home machine through an AI assistant — that's what
                        Rexec enables.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="container" size={16} />
                    <span
                        >What's the difference between containers and agents?</span
                    >
                </summary>
                <div class="faq-answer">
                    <p>
                        Both give you terminal access, but they serve different
                        purposes:
                    </p>
                    <p><strong>Containers (provisioned by Rexec):</strong></p>
                    <ul>
                        <li>
                            One-click setup — choose Ubuntu, Debian, Alpine,
                            Fedora, etc.
                        </li>
                        <li>
                            Isolated and disposable — perfect for experiments
                            and sandboxes
                        </li>
                        <li>Pre-configured with dev tools and SSH</li>
                    </ul>
                    <p><strong>Agents (your own machines):</strong></p>
                    <ul>
                        <li>
                            Full access to your laptop, server, Raspberry Pi, or
                            any machine
                        </li>
                        <li>Uses your machine's resources and network</li>
                        <li>
                            No inbound ports or VPN required — agent connects
                            outbound
                        </li>
                        <li>
                            Perfect for accessing your dev environment from
                            anywhere
                        </li>
                    </ul>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="code" size={16} />
                    <span>What SDKs are available?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        We provide official SDKs for <strong>8 programming languages</strong>:
                    </p>
                    <ul>
                        <li><strong>Go</strong> — <code>go get github.com/PipeOpsHQ/rexec-go</code></li>
                        <li><strong>JavaScript/TypeScript</strong> — <code>npm install @pipeopshq/rexec</code></li>
                        <li><strong>Python</strong> — <code>pip install rexec</code></li>
                        <li><strong>Rust</strong> — <code>cargo add rexec</code></li>
                        <li><strong>Ruby</strong> — <code>gem install rexec</code></li>
                        <li><strong>Java</strong> — Maven: <code>io.pipeops:rexec</code></li>
                        <li><strong>C#/.NET</strong> — <code>dotnet add package Rexec</code></li>
                        <li><strong>PHP</strong> — <code>composer require pipeopshq/rexec</code></li>
                    </ul>
                    <p>
                        All SDKs support container management, file operations, and interactive WebSocket terminals.
                        See the <a href="#sdks">SDK section</a> or <a href="https://github.com/PipeOpsHQ/rexec/blob/main/docs/SDK.md" target="_blank">full documentation</a>.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="code" size={16} />
                    <span>Can I automate container creation via API?</span>
                </summary>
                <div class="faq-answer">
                    <p>
                        <strong>Yes!</strong> All Rexec features are available through our REST API and SDKs:
                    </p>
                    <ul>
                        <li><strong>Create containers</strong> — Spin up sandboxed environments programmatically</li>
                        <li><strong>Execute commands</strong> — Run scripts and capture output</li>
                        <li><strong>File operations</strong> — Upload, download, and manage files</li>
                        <li><strong>Interactive terminals</strong> — Connect via WebSocket for real-time I/O</li>
                    </ul>
                    <p>
                        Common use cases include CI/CD pipelines, code execution services, 
                        education platforms, and automated testing. Generate an API token 
                        from Settings to get started.
                    </p>
                </div>
            </details>

            <details class="faq-item">
                <summary class="faq-question">
                    <StatusIcon status="code" size={16} />
                    <span>How do I get an API token?</span>
                </summary>
                <div class="faq-answer">
                    <p>To generate an API token:</p>
                    <ol>
                        <li>Log in to your Rexec dashboard</li>
                        <li>Go to <strong>Settings</strong> → <strong>API Tokens</strong></li>
                        <li>Click <strong>Generate Token</strong></li>
                        <li>Copy and securely store the token</li>
                    </ol>
                    <p>
                        <strong>Security tip:</strong> Never commit tokens to version control. 
                        Use environment variables or a secrets manager.
                    </p>
                </div>
            </details>
        </div>
    </section>

    <!-- Support -->
    <section class="section support-section">
        <h2><StatusIcon status="chat" size={20} /> Need Help?</h2>
        <div class="support-cards">
            <a
                href="https://github.com/pipeops-dev/rexec/issues"
                target="_blank"
                rel="noopener"
                class="support-card"
            >
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
        scroll-margin-top: 20px;
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

    .method.get {
        background: rgba(0, 150, 255, 0.2);
        color: #0096ff;
    }
    .method.post {
        background: rgba(0, 255, 65, 0.2);
        color: var(--accent);
    }
    .method.delete {
        background: rgba(255, 60, 60, 0.2);
        color: #ff3c3c;
    }
    .method.websocket {
        background: rgba(180, 100, 255, 0.2);
        color: #b464ff;
    }

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

    /* SDK Section */
    .sdk-section {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 24px;
    }

    .sdk-section > p {
        margin: 0 0 20px;
        font-size: 14px;
        color: var(--text-secondary);
        line-height: 1.5;
    }

    .sdk-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
        gap: 12px;
        margin-bottom: 24px;
    }

    .sdk-card {
        display: flex;
        align-items: center;
        gap: 14px;
        padding: 14px 16px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
        text-decoration: none;
        transition: all 0.2s ease;
    }

    .sdk-card:hover {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
        transform: translateY(-2px);
    }

    .sdk-icon {
        font-size: 24px;
        width: 36px;
        text-align: center;
        color: var(--accent);
    }

    .sdk-info h4 {
        margin: 0 0 4px;
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
    }

    .sdk-info code {
        font-family: var(--font-mono);
        font-size: 11px;
        color: var(--text-muted);
    }

    .sdk-example {
        margin-bottom: 20px;
        border-top: 1px solid var(--border);
        padding-top: 20px;
    }

    .sdk-example h4 {
        font-size: 13px;
        font-weight: 600;
        margin: 0 0 12px;
        color: var(--text);
    }

    .sdk-example pre {
        margin: 0;
        padding: 16px;
        background: var(--bg-tertiary);
        border-radius: 6px;
        overflow-x: auto;
    }

    .sdk-example code {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--accent);
        line-height: 1.6;
    }

    .sdk-links {
        display: flex;
        gap: 16px;
        flex-wrap: wrap;
    }

    .sdk-link {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--accent);
        text-decoration: none;
        font-size: 13px;
        font-weight: 500;
        transition: all 0.2s ease;
    }

    .sdk-link:hover {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
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
