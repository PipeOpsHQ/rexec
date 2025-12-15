<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import { auth } from "$stores/auth";
    import { toast } from "$stores/toast";
    import StatusIcon from "./icons/StatusIcon.svelte";

    const dispatch = createEventDispatcher<{
        guest: void;
        navigate: { view: string };
    }>();

    let isOAuthLoading = false;

    function handleGuestClick() {
        dispatch("guest");
    }

    async function handleOAuthLogin() {
        if (isOAuthLoading) return;
        isOAuthLoading = true;
        try {
            const url = await auth.getOAuthUrl();
            if (url) {
                window.location.href = url;
            } else {
                toast.error("Unable to connect. Please try again later.");
                isOAuthLoading = false;
            }
        } catch (e) {
            toast.error("Failed to connect. Please try again.");
            isOAuthLoading = false;
        }
    }

    const useCases = [
        {
            icon: "robot",
            title: "AI Code Execution",
            description: "Safely run AI-generated scripts in isolated containers before they touch your laptop or production.",
            example: "Copy code from ChatGPT → paste into Rexec → validate with tests → deploy with confidence"
        },
        {
            icon: "shield",
            title: "Secure Jump Boxes",
            description: "Access servers without exposing SSH ports. The agent connects outbound — no inbound firewall rules needed.",
            example: "Debug production issues from anywhere without VPN headaches"
        },
        {
            icon: "bolt",
            title: "Instant Dev Environments",
            description: "Spin up a fully-configured terminal in seconds. No Docker setup, no VM provisioning, no waiting.",
            example: "Onboard new team members in minutes, not hours"
        },
        {
            icon: "users",
            title: "Pair Programming",
            description: "Share terminals in real-time with view or control access. Perfect for debugging sessions and demos.",
            example: "\"Can you look at this error?\" → Share link → Collaborate instantly"
        }
    ];

    const features = [
        {
            icon: "terminal",
            title: "Real Terminal UX",
            description: "Full xterm.js with tmux persistence, large paste support, and sub-100ms latency"
        },
        {
            icon: "lock",
            title: "Zero Trust Security",
            description: "Network-isolated containers, token-based auth, no shared state between users"
        },
        {
            icon: "agent",
            title: "Bring Your Own Server",
            description: "Connect any Linux/macOS machine with a single install command"
        },
        {
            icon: "cloud",
            title: "Works Everywhere",
            description: "Browser-based access from any device. PWA support for mobile."
        }
    ];

    const roles = [
        { name: "Vibe Coder", icon: "sparkles", tools: "AI CLIs, neovim, python, node" },
        { name: "DevOps", icon: "server", tools: "kubectl, terraform, ansible, docker" },
        { name: "Python", icon: "code", tools: "python3, pip, venv, jupyter" },
        { name: "Node.js", icon: "code", tools: "node, npm, yarn, pnpm" },
        { name: "Go", icon: "code", tools: "go, golangci-lint, air" },
        { name: "Minimalist", icon: "terminal", tools: "Just the essentials" }
    ];
</script>

<div class="launch-page">
    <!-- Hero Section -->
    <section class="hero">
        <div class="hero-badge">
            <StatusIcon status="sparkles" size={14} />
            <span>Now in Public Beta</span>
        </div>

        <h1>
            Instant Cloud Terminals
            <span class="gradient-text">For Modern Developers</span>
        </h1>

        <p class="hero-description">
            Disposable dev environments in seconds. Secure access to your own servers.
            A safe place to run AI-generated code. All in one dashboard.
        </p>

        <div class="hero-actions">
            <button class="btn btn-primary btn-xl" onclick={handleGuestClick}>
                <StatusIcon status="bolt" size={18} />
                Try Free — No Sign Up
            </button>
            <button
                class="btn btn-secondary btn-xl"
                onclick={handleOAuthLogin}
                disabled={isOAuthLoading}
            >
                {#if isOAuthLoading}
                    <span class="spinner"></span>
                {:else}
                    <StatusIcon status="user" size={18} />
                {/if}
                Sign in with PipeOps
            </button>
        </div>

        <p class="hero-subtext">
            <StatusIcon status="check" size={14} />
            Free tier includes 3 concurrent terminals
        </p>
    </section>

    <!-- Product Screenshot -->
    <section class="screenshot-section">
        <div class="screenshot-wrapper">
            <img
                src="/screenshot-desktop.png"
                alt="Rexec Dashboard showing terminal management"
                class="screenshot"
            />
            <div class="screenshot-glow"></div>
        </div>
    </section>

    <!-- Social Proof -->
    <section class="social-proof">
        <p class="proof-text">
            Trusted by developers at startups and enterprises building with AI
        </p>
        <div class="proof-stats">
            <div class="stat">
                <span class="stat-value">10K+</span>
                <span class="stat-label">Terminals Created</span>
            </div>
            <div class="stat">
                <span class="stat-value">&lt;2s</span>
                <span class="stat-label">Average Spin-up</span>
            </div>
            <div class="stat">
                <span class="stat-value">99.9%</span>
                <span class="stat-label">Uptime</span>
            </div>
        </div>
    </section>

    <!-- Use Cases -->
    <section class="use-cases">
        <h2>Built for Real Workflows</h2>
        <p class="section-subtitle">See how teams are using Rexec today</p>

        <div class="use-cases-grid">
            {#each useCases as useCase}
                <div class="use-case-card">
                    <div class="use-case-icon">
                        <StatusIcon status={useCase.icon} size={28} />
                    </div>
                    <h3>{useCase.title}</h3>
                    <p>{useCase.description}</p>
                    <div class="use-case-example">
                        <StatusIcon status="chevron-right" size={14} />
                        <span>{useCase.example}</span>
                    </div>
                </div>
            {/each}
        </div>
    </section>

    <!-- AI Coding Highlight -->
    <section class="ai-highlight">
        <div class="ai-content">
            <div class="ai-badge">
                <StatusIcon status="sparkles" size={14} />
                Perfect for AI Coding
            </div>
            <h2>Run AI-Generated Code Safely</h2>
            <p>
                Copy/pasting code from ChatGPT or Claude directly into your laptop is risky.
                Rexec gives you an isolated sandbox to validate LLM-generated scripts before
                they touch your real environment.
            </p>
            <ul class="ai-benefits">
                <li>
                    <StatusIcon status="check" size={16} />
                    <span>Isolated containers — nothing escapes</span>
                </li>
                <li>
                    <StatusIcon status="check" size={16} />
                    <span>Pre-installed AI CLI tools (tgpt, aichat, mods)</span>
                </li>
                <li>
                    <StatusIcon status="check" size={16} />
                    <span>Run tests before deploying to production</span>
                </li>
                <li>
                    <StatusIcon status="check" size={16} />
                    <span>Reset to clean state in one click</span>
                </li>
            </ul>
        </div>
        <div class="ai-visual">
            <div class="code-flow">
                <div class="flow-step">
                    <div class="flow-icon">🤖</div>
                    <span>AI generates code</span>
                </div>
                <div class="flow-arrow">→</div>
                <div class="flow-step active">
                    <div class="flow-icon"><StatusIcon status="terminal" size={20} /></div>
                    <span>Test in Rexec</span>
                </div>
                <div class="flow-arrow">→</div>
                <div class="flow-step">
                    <div class="flow-icon">✓</div>
                    <span>Deploy safely</span>
                </div>
            </div>
        </div>
    </section>

    <!-- Roles/Environments -->
    <section class="roles-section">
        <h2>Pre-Configured Environments</h2>
        <p class="section-subtitle">Choose a role and get a fully-equipped terminal instantly</p>

        <div class="roles-grid">
            {#each roles as role}
                <div class="role-card">
                    <StatusIcon status={role.icon} size={32} />
                    <h4>{role.name}</h4>
                    <p>{role.tools}</p>
                </div>
            {/each}
        </div>
    </section>

    <!-- Features Grid -->
    <section class="features-section">
        <h2>Everything You Need</h2>
        <div class="features-grid">
            {#each features as feature}
                <div class="feature-card">
                    <StatusIcon status={feature.icon} size={24} />
                    <h4>{feature.title}</h4>
                    <p>{feature.description}</p>
                </div>
            {/each}
        </div>
    </section>

    <!-- BYOS Agent -->
    <section class="agent-section">
        <div class="agent-content">
            <h2>Bring Your Own Server</h2>
            <p>
                Connect any Linux or macOS machine to your Rexec dashboard with a single command.
                No inbound ports, no VPN, no SSH exposure.
            </p>
            <div class="agent-code">
                <code>curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash</code>
            </div>
            <ul class="agent-benefits">
                <li><StatusIcon status="check" size={14} /> Outbound-only connection</li>
                <li><StatusIcon status="check" size={14} /> Persistent API tokens</li>
                <li><StatusIcon status="check" size={14} /> Auto-reconnect on disconnect</li>
            </ul>
        </div>
        <div class="agent-visual">
            <div class="server-diagram">
                <div class="server-box">
                    <StatusIcon status="server" size={24} />
                    <span>Your Server</span>
                </div>
                <div class="connection-line">
                    <span>Outbound WebSocket</span>
                </div>
                <div class="server-box rexec">
                    <StatusIcon status="cloud" size={24} />
                    <span>Rexec</span>
                </div>
            </div>
        </div>
    </section>

    <!-- Pricing Teaser -->
    <section class="pricing-section">
        <h2>Start Free, Scale When Ready</h2>
        <div class="pricing-cards">
            <div class="pricing-card">
                <h3>Free</h3>
                <div class="price">$0<span>/month</span></div>
                <ul>
                    <li><StatusIcon status="check" size={14} /> 3 concurrent terminals</li>
                    <li><StatusIcon status="check" size={14} /> 1 agent connection</li>
                    <li><StatusIcon status="check" size={14} /> All pre-configured roles</li>
                    <li><StatusIcon status="check" size={14} /> Community support</li>
                </ul>
                <button class="btn btn-secondary" onclick={handleGuestClick}>
                    Get Started
                </button>
            </div>
            <div class="pricing-card featured">
                <div class="featured-badge">Most Popular</div>
                <h3>Pro</h3>
                <div class="price">$19<span>/month</span></div>
                <ul>
                    <li><StatusIcon status="check" size={14} /> 10 concurrent terminals</li>
                    <li><StatusIcon status="check" size={14} /> Unlimited agents</li>
                    <li><StatusIcon status="check" size={14} /> Priority support</li>
                    <li><StatusIcon status="check" size={14} /> Custom images</li>
                    <li><StatusIcon status="check" size={14} /> Team collaboration</li>
                </ul>
                <button class="btn btn-primary" onclick={() => dispatch('navigate', { view: 'pricing' })}>
                    Upgrade to Pro
                </button>
            </div>
        </div>
    </section>

    <!-- Final CTA -->
    <section class="final-cta">
        <h2>Ready to Try?</h2>
        <p>Get your first terminal running in under 30 seconds. No credit card required.</p>
        <button class="btn btn-primary btn-xl" onclick={handleGuestClick}>
            <StatusIcon status="bolt" size={18} />
            Launch Your First Terminal
        </button>
    </section>
</div>

<style>
    .launch-page {
        max-width: 1200px;
        margin: 0 auto;
        padding: 0 20px;
    }

    /* Hero */
    .hero {
        text-align: center;
        padding: 80px 0 40px;
    }

    .hero-badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 6px 14px;
        background: var(--accent-muted, rgba(59, 130, 246, 0.1));
        border: 1px solid var(--accent);
        border-radius: 20px;
        font-size: 12px;
        font-weight: 500;
        color: var(--accent);
        margin-bottom: 24px;
    }

    .hero h1 {
        font-size: 48px;
        font-weight: 700;
        line-height: 1.1;
        margin: 0 0 24px;
        color: var(--text);
    }

    .gradient-text {
        display: block;
        background: linear-gradient(135deg, var(--accent) 0%, #8b5cf6 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
    }

    .hero-description {
        font-size: 18px;
        color: var(--text-muted);
        max-width: 600px;
        margin: 0 auto 32px;
        line-height: 1.6;
    }

    .hero-actions {
        display: flex;
        gap: 16px;
        justify-content: center;
        flex-wrap: wrap;
        margin-bottom: 16px;
    }

    .btn-xl {
        padding: 14px 28px;
        font-size: 16px;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .hero-subtext {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 6px;
        font-size: 13px;
        color: var(--text-muted);
    }

    .spinner {
        width: 16px;
        height: 16px;
        border: 2px solid transparent;
        border-top-color: currentColor;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    /* Screenshot */
    .screenshot-section {
        padding: 40px 0;
    }

    .screenshot-wrapper {
        position: relative;
        max-width: 900px;
        margin: 0 auto;
    }

    .screenshot {
        width: 100%;
        border-radius: 12px;
        border: 1px solid var(--border);
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    }

    .screenshot-glow {
        position: absolute;
        inset: 0;
        background: radial-gradient(ellipse at center, var(--accent-muted, rgba(59, 130, 246, 0.1)) 0%, transparent 70%);
        pointer-events: none;
        z-index: -1;
        transform: scale(1.2);
    }

    /* Social Proof */
    .social-proof {
        text-align: center;
        padding: 40px 0;
        border-top: 1px solid var(--border);
        border-bottom: 1px solid var(--border);
    }

    .proof-text {
        font-size: 14px;
        color: var(--text-muted);
        margin-bottom: 24px;
    }

    .proof-stats {
        display: flex;
        justify-content: center;
        gap: 60px;
        flex-wrap: wrap;
    }

    .stat {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .stat-value {
        font-size: 32px;
        font-weight: 700;
        color: var(--text);
    }

    .stat-label {
        font-size: 13px;
        color: var(--text-muted);
    }

    /* Sections common */
    section h2 {
        font-size: 32px;
        font-weight: 700;
        text-align: center;
        margin: 0 0 12px;
        color: var(--text);
    }

    .section-subtitle {
        text-align: center;
        color: var(--text-muted);
        margin-bottom: 40px;
    }

    /* Use Cases */
    .use-cases {
        padding: 80px 0;
    }

    .use-cases-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 24px;
    }

    .use-case-card {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 28px;
        transition: all 0.2s;
    }

    .use-case-card:hover {
        border-color: var(--accent);
        transform: translateY(-2px);
    }

    .use-case-icon {
        width: 48px;
        height: 48px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--accent-muted, rgba(59, 130, 246, 0.1));
        border-radius: 10px;
        margin-bottom: 16px;
        color: var(--accent);
    }

    .use-case-card h3 {
        font-size: 18px;
        font-weight: 600;
        margin: 0 0 8px;
        color: var(--text);
    }

    .use-case-card > p {
        font-size: 14px;
        color: var(--text-muted);
        line-height: 1.5;
        margin: 0 0 16px;
    }

    .use-case-example {
        display: flex;
        align-items: flex-start;
        gap: 8px;
        font-size: 13px;
        color: var(--accent);
        padding-top: 12px;
        border-top: 1px solid var(--border);
    }

    /* AI Highlight */
    .ai-highlight {
        padding: 80px 0;
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 60px;
        align-items: center;
        background: var(--bg-secondary);
        margin: 0 -20px;
        padding-left: 40px;
        padding-right: 40px;
    }

    .ai-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        background: var(--accent-muted, rgba(59, 130, 246, 0.1));
        border-radius: 12px;
        font-size: 12px;
        font-weight: 500;
        color: var(--accent);
        margin-bottom: 16px;
    }

    .ai-content h2 {
        text-align: left;
        font-size: 28px;
        margin-bottom: 16px;
    }

    .ai-content > p {
        color: var(--text-muted);
        line-height: 1.6;
        margin-bottom: 24px;
    }

    .ai-benefits {
        list-style: none;
        padding: 0;
        margin: 0;
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .ai-benefits li {
        display: flex;
        align-items: center;
        gap: 10px;
        color: var(--text);
        font-size: 14px;
    }

    .ai-visual {
        display: flex;
        justify-content: center;
    }

    .code-flow {
        display: flex;
        align-items: center;
        gap: 16px;
    }

    .flow-step {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 20px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 12px;
        font-size: 12px;
        color: var(--text-muted);
    }

    .flow-step.active {
        border-color: var(--accent);
        background: var(--accent-muted, rgba(59, 130, 246, 0.1));
        color: var(--accent);
    }

    .flow-icon {
        font-size: 24px;
    }

    .flow-arrow {
        color: var(--text-muted);
        font-size: 20px;
    }

    /* Roles */
    .roles-section {
        padding: 80px 0;
    }

    .roles-grid {
        display: grid;
        grid-template-columns: repeat(6, 1fr);
        gap: 16px;
    }

    .role-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
        padding: 24px 16px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 12px;
        transition: all 0.2s;
    }

    .role-card:hover {
        border-color: var(--accent);
    }

    .role-card h4 {
        font-size: 14px;
        font-weight: 600;
        margin: 12px 0 4px;
        color: var(--text);
    }

    .role-card p {
        font-size: 11px;
        color: var(--text-muted);
        margin: 0;
    }

    /* Features */
    .features-section {
        padding: 80px 0;
    }

    .features-grid {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 24px;
    }

    .feature-card {
        text-align: center;
        padding: 32px 20px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 12px;
    }

    .feature-card h4 {
        font-size: 16px;
        font-weight: 600;
        margin: 16px 0 8px;
        color: var(--text);
    }

    .feature-card p {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0;
        line-height: 1.5;
    }

    /* Agent Section */
    .agent-section {
        padding: 80px 0;
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 60px;
        align-items: center;
    }

    .agent-content h2 {
        text-align: left;
        font-size: 28px;
    }

    .agent-content > p {
        color: var(--text-muted);
        line-height: 1.6;
        margin-bottom: 24px;
    }

    .agent-code {
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 16px;
        margin-bottom: 20px;
        overflow-x: auto;
    }

    .agent-code code {
        font-family: var(--font-mono), monospace;
        font-size: 13px;
        color: var(--accent);
    }

    .agent-benefits {
        list-style: none;
        padding: 0;
        margin: 0;
        display: flex;
        flex-wrap: wrap;
        gap: 16px;
    }

    .agent-benefits li {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 13px;
        color: var(--text-muted);
    }

    .server-diagram {
        display: flex;
        align-items: center;
        gap: 20px;
    }

    .server-box {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 24px 32px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 12px;
        font-size: 13px;
        color: var(--text);
    }

    .server-box.rexec {
        border-color: var(--accent);
        background: var(--accent-muted, rgba(59, 130, 246, 0.1));
    }

    .connection-line {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        font-size: 11px;
        color: var(--text-muted);
    }

    .connection-line::before {
        content: "→";
        font-size: 24px;
        color: var(--accent);
    }

    /* Pricing */
    .pricing-section {
        padding: 80px 0;
        text-align: center;
    }

    .pricing-cards {
        display: flex;
        justify-content: center;
        gap: 24px;
        margin-top: 40px;
    }

    .pricing-card {
        position: relative;
        width: 280px;
        padding: 32px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 16px;
        text-align: left;
    }

    .pricing-card.featured {
        border-color: var(--accent);
        background: var(--bg-tertiary);
    }

    .featured-badge {
        position: absolute;
        top: -12px;
        left: 50%;
        transform: translateX(-50%);
        padding: 4px 12px;
        background: var(--accent);
        color: white;
        font-size: 11px;
        font-weight: 600;
        border-radius: 12px;
    }

    .pricing-card h3 {
        font-size: 20px;
        font-weight: 600;
        margin: 0 0 8px;
        color: var(--text);
    }

    .price {
        font-size: 36px;
        font-weight: 700;
        color: var(--text);
        margin-bottom: 24px;
    }

    .price span {
        font-size: 14px;
        font-weight: 400;
        color: var(--text-muted);
    }

    .pricing-card ul {
        list-style: none;
        padding: 0;
        margin: 0 0 24px;
        display: flex;
        flex-direction: column;
        gap: 12px;
    }

    .pricing-card li {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 13px;
        color: var(--text-muted);
    }

    .pricing-card .btn {
        width: 100%;
        justify-content: center;
    }

    /* Final CTA */
    .final-cta {
        padding: 80px 0;
        text-align: center;
        border-top: 1px solid var(--border);
    }

    .final-cta p {
        color: var(--text-muted);
        margin-bottom: 32px;
    }

    /* Responsive */
    @media (max-width: 1024px) {
        .roles-grid {
            grid-template-columns: repeat(3, 1fr);
        }

        .features-grid {
            grid-template-columns: repeat(2, 1fr);
        }
    }

    @media (max-width: 768px) {
        .hero h1 {
            font-size: 32px;
        }

        .hero-description {
            font-size: 16px;
        }

        .hero-actions {
            flex-direction: column;
            align-items: center;
        }

        .use-cases-grid {
            grid-template-columns: 1fr;
        }

        .ai-highlight {
            grid-template-columns: 1fr;
            padding-left: 20px;
            padding-right: 20px;
        }

        .ai-content h2 {
            text-align: center;
        }

        .roles-grid {
            grid-template-columns: repeat(2, 1fr);
        }

        .features-grid {
            grid-template-columns: 1fr;
        }

        .agent-section {
            grid-template-columns: 1fr;
        }

        .agent-content h2 {
            text-align: center;
        }

        .server-diagram {
            flex-direction: column;
        }

        .connection-line::before {
            content: "↓";
        }

        .pricing-cards {
            flex-direction: column;
            align-items: center;
        }

        .proof-stats {
            gap: 30px;
        }

        .stat-value {
            font-size: 24px;
        }
    }
</style>

