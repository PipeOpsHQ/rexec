<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    
    const dispatch = createEventDispatcher<{
        tryNow: void;
        navigate: { slug: string };
    }>();

    function handleTryNow() {
        dispatch("tryNow");
    }

    function navigateToCase(slug: string) {
        dispatch("navigate", { slug });
    }

    const useCases = [
        {
            slug: "ephemeral-dev-environments",
            title: "Ephemeral Dev Environments",
            icon: "bolt",
            description: "The future is disposable. Spin up a fresh, clean environment for every task, PR, or experiment. No drift, no cleanup.",
            points: [
                "Zero setup time - milliseconds to code",
                "Immutable infrastructure patterns applied to dev",
                "Always clean state - avoid 'works on my machine'",
                "Perfect for testing dangerous scripts"
            ]
        },
        {
            slug: "collaborative-intelligence",
            title: "Collaborative Intelligence",
            icon: "ai",
            description: "A shared workspace for humans and AI agents. Let LLMs execute code in a real, safe environment while you supervise.",
            points: [
                "Sandboxed execution for autonomous agents",
                "Human-in-the-loop oversight",
                "Resumable sessions - disconnect and reconnect anytime",
                "Standardized toolchain for consistent AI output"
            ]
        },
        {
            slug: "universal-jump-host",
            title: "Secure Jump Host & Gateway",
            icon: "shield",
            description: "Zero-trust access to your private infrastructure. Replace VPNs with a secure, audited browser-based gateway.",
            points: [
                "Enforce MFA and IP Whitelisting",
                "Audit logs for every command executed",
                "Access private VPCs securely",
                "No SSH key management required"
            ]
        },
        {
            slug: "rexec-agent",
            title: "Hybrid Cloud & Remote Agents",
            icon: "connected",
            description: "Unify your infrastructure. Connect any Linux server, IoT device, or cloud instance to your Rexec dashboard.",
            points: [
                "Real-time resource monitoring (CPU/RAM)",
                "Secure outbound WebSocket tunnels",
                "Manage on-prem and cloud side-by-side",
                "Works on AWS, Azure, or Raspberry Pi"
            ]
        },
        {
            slug: "instant-education-onboarding",
            title: "Instant Education & Onboarding",
            icon: "book",
            description: "Onboard new engineers in seconds, not days. Provide pre-configured environments for workshops and tutorials.",
            points: [
                "Standardized team environments",
                "Interactive documentation that runs",
                "Zero friction for workshop attendees",
                "Focus on learning, not configuring"
            ]
        },
        {
            slug: "technical-interviews",
            title: "Technical Interviews",
            icon: "terminal",
            description: "Conduct real-time coding interviews in a real Linux environment, not a constrained web editor.",
            points: [
                "Full shell access for realistic assessment",
                "Multiplayer mode for pair programming",
                "Pre-install custom challenges/repos",
                "Review candidate approach in real-time"
            ]
        },
        {
            slug: "open-source-review",
            title: "Open Source Review",
            icon: "connected",
            description: "Review Pull Requests by instantly spinning up the branch in a clean container. Test without polluting your local machine.",
            points: [
                "One-click environment for any PR",
                "Verify build/test scripts safely",
                "No dependency conflicts with local setup",
                "Dispose immediately after review"
            ]
        },
        {
            slug: "gpu-terminals",
            title: "GPU Terminals for AI/ML (Coming Soon)",
            icon: "gpu",
            description: "Rexec will provide instant-on, powerful GPU-enabled terminals for your team's AI/ML model development, training, and fine-tuning. Manage and share these dedicated GPU resources securely, eliminating the complexities of direct infrastructure access and SSH key sharing.",
            points: [
                "On-demand access to GPU-accelerated terminals",
                "Centralized team management of GPU resources",
                "Pre-configured with ML frameworks (TensorFlow, PyTorch)",
                "Isolated for reproducible experiments and data security",
                "Securely share running GPU sessions with collaborators",
                "Flexible scaling and collaborative resource allocation"
            ],
            comingSoon: true
        },
        {
            slug: "edge-device-development",
            title: "Edge Device Development",
            icon: "wifi",
            description: "Develop and test applications for IoT and edge devices in a simulated or emulated environment.",
            points: [
                "Cross-compilation toolchains ready",
                "Test on various architectures (ARM, RISC-V)",
                "Secure remote access to virtual devices",
                "Rapid prototyping for embedded systems"
            ]
        },
        {
            slug: "real-time-data-processing",
            title: "Real-time Data Processing",
            icon: "data",
            description: "Build, test, and deploy streaming ETL pipelines and real-time analytics applications.",
            points: [
                "High-performance data ingress/egress",
                "Integrated with Kafka, Flink, Spark (coming soon)",
                "Monitor data flows in isolation",
                "Secure access to data sources"
            ]
        },
        {
            slug: "resumable-sessions",
            title: "Resumable Terminal Sessions",
            icon: "reconnect",
            description: "Start long-running tasks, close your browser, and come back later. Your terminal session keeps running in the background with full output history.",
            points: [
                "50,000 lines of scrollback history",
                "Sessions persist across disconnects",
                "See all output that happened while away",
                "Perfect for builds, training, and deployments"
            ]
        },
        {
            slug: "rexec-cli",
            title: "Rexec CLI & TUI",
            icon: "terminal",
            description: "Manage your terminals from anywhere using our powerful command-line interface with an interactive TUI mode.",
            points: [
                "Full terminal management from your shell",
                "Interactive TUI dashboard (rexec -i)",
                "Create, connect, and manage terminals",
                "Run snippets and macros directly",
                "Install via: curl -fsSL rexec.pipeops.io/install.sh | bash"
            ]
        },
        {
            slug: "hybrid-infrastructure",
            title: "Hybrid Infrastructure Access",
            icon: "shield",
            description: "Mix cloud-managed terminals with your own infrastructure. Access everything through a single, unified interface.",
            points: [
                "Combine Rexec terminals with self-hosted",
                "Unified access control and audit logging",
                "No VPN or complex networking required",
                "Share access without sharing SSH keys",
                "Perfect for multi-cloud environments"
            ]
        },
        {
            slug: "remote-debugging",
            title: "Remote Debugging & Troubleshooting",
            icon: "bug",
            description: "Debug production issues directly from your browser. Connect to any server running the Rexec agent for instant access.",
            points: [
                "Instant shell access to production servers",
                "No SSH key distribution needed",
                "Browser-based with full terminal capabilities",
                "Share sessions for pair debugging",
                "Complete audit trail of all commands"
            ]
        }
    ];
</script>

<svelte:head>
    <title>Rexec Use Cases - The Future of Development</title>
    <meta name="description" content="Discover how Rexec powers ephemeral development environments, AI agent execution, collaborative coding, and secure cloud access." />
    <meta property="og:title" content="Rexec Use Cases" />
    <meta property="og:description" content="Discover how Rexec powers ephemeral development environments, AI agent execution, collaborative coding, and secure cloud access." />
</svelte:head>

<div class="usecases-page">
    <div class="page-header">
        <div class="header-badge">
            <span class="dot"></span>
            <span>Why Rexec?</span>
        </div>
        <h1>Powerful <span class="accent">Use Cases</span></h1>
        <p class="subtitle">
            Rexec is more than just a terminal. It's an ephemeral computing platform designed for modern workflows.
        </p>
    </div>

    <div class="cases-grid">
        {#each useCases as useCase, i}
            <button 
                class="case-card" 
                class:coming-soon={useCase.comingSoon}
                style="animation-delay: {i * 50}ms"
                onclick={() => navigateToCase(useCase.slug)}
            >
                <div class="case-icon">
                    <StatusIcon status={useCase.icon} size={32} />
                </div>
                <h3>{useCase.title}</h3>
                <p class="case-description">{useCase.description}</p>
                <ul class="case-points">
                    {#each useCase.points.slice(0, 3) as point}
                        <li>
                            <span class="bullet">•</span>
                            {point}
                        </li>
                    {/each}
                </ul>
                <div class="card-footer">
                    <span class="learn-more">Learn more <span class="arrow">→</span></span>
                </div>
            </button>
        {/each}
    </div>

    <section class="cta-section">
        <h2>Ready to explore?</h2>
        <p>Start your first session and see what's possible.</p>
        <button class="btn btn-primary btn-lg" onclick={handleTryNow}>
            <StatusIcon status="rocket" size={16} />
            <span>Launch Terminal</span>
        </button>
    </section>
</div>

<style>
    .usecases-page {
        max-width: 1200px;
        margin: 0 auto;
        padding: 40px 20px;
    }

    .page-header {
        text-align: center;
        margin-bottom: 60px;
    }

    .header-badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 4px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        font-size: 11px;
        color: var(--text-secondary);
        margin-bottom: 20px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .header-badge .dot {
        width: 6px;
        height: 6px;
        background: var(--accent);
        animation: blink 1s step-end infinite;
    }

    h1 {
        font-size: 36px;
        font-weight: 700;
        margin-bottom: 16px;
        letter-spacing: 1px;
    }

    h1 .accent {
        color: var(--accent);
        text-shadow: var(--accent-glow);
    }

    .subtitle {
        font-size: 16px;
        color: var(--text-muted);
        max-width: 600px;
        margin: 0 auto;
        line-height: 1.6;
    }

    .cases-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        gap: 30px;
        margin-bottom: 60px;
    }

    .case-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        padding: 30px;
        border-radius: 12px;
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        display: flex;
        flex-direction: column;
        cursor: pointer;
        text-align: left;
        font-family: var(--font-mono);
        width: 100%;
        animation: fadeInUp 0.5s ease both;
        position: relative;
        overflow: hidden;
    }

    .case-card::before {
        content: "";
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 1px;
        background: linear-gradient(90deg, transparent, var(--accent), transparent);
        opacity: 0;
        transition: opacity 0.3s ease;
    }

    .case-card:hover {
        transform: translateY(-8px) scale(1.02);
        border-color: var(--accent);
        box-shadow: 0 20px 40px rgba(0, 255, 65, 0.1);
        background: linear-gradient(135deg, var(--bg-card) 0%, rgba(0, 255, 65, 0.05) 100%);
    }

    .case-card:hover::before {
        opacity: 0.5;
    }

    .case-card:hover .arrow {
        transform: translateX(4px);
    }

    .case-card:hover .learn-more {
        color: var(--accent);
    }

    .case-card:hover .case-icon {
        background: var(--accent);
        color: #000;
        border-color: var(--accent);
        box-shadow: 0 0 20px var(--accent-glow);
    }

    .case-card:active {
        transform: translateY(-4px) scale(1.01);
    }

    .case-card.coming-soon {
        border-color: var(--border);
        background: rgba(255, 255, 255, 0.02);
        opacity: 0.8;
    }

    .case-card.coming-soon:hover {
        border-color: var(--yellow);
        box-shadow: 0 10px 30px rgba(252, 238, 10, 0.1);
        background: linear-gradient(135deg, var(--bg-card) 0%, rgba(252, 238, 10, 0.05) 100%);
        transform: translateY(-4px);
    }

    .case-card.coming-soon:hover .case-icon {
        background: var(--yellow);
        color: #000;
        border-color: var(--yellow);
        box-shadow: 0 0 20px rgba(252, 238, 10, 0.4);
    }

    .case-card.coming-soon .case-icon {
        background: rgba(252, 238, 10, 0.1);
        border-color: rgba(252, 238, 10, 0.3);
        color: var(--yellow);
    }

    .case-card.coming-soon .bullet {
        color: var(--yellow);
    }

    .case-icon {
        margin-bottom: 20px;
        color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
        width: 60px;
        height: 60px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 12px;
        border: 1px solid rgba(0, 255, 65, 0.2);
        transition: all 0.3s ease;
    }

    h3 {
        font-size: 20px;
        margin: 0 0 10px 0;
        color: var(--text);
    }

    .case-description {
        font-size: 14px;
        color: var(--text-secondary);
        margin-bottom: 20px;
        line-height: 1.5;
        flex-grow: 1;
    }

    .case-points {
        list-style: none;
        padding: 0;
        margin: 0;
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .case-points li {
        font-size: 13px;
        color: var(--text-muted);
        display: flex;
        align-items: flex-start;
        gap: 8px;
    }

    .bullet {
        color: var(--accent);
    }

    .cta-section {
        text-align: center;
        padding: 60px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
    }

    .cta-section h2 {
        font-size: 28px;
        margin-bottom: 12px;
    }

    .cta-section p {
        color: var(--text-muted);
        margin-bottom: 24px;
    }

    @keyframes blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0; }
    }

    @keyframes fadeInUp {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .card-footer {
        margin-top: auto;
        padding-top: 20px;
        border-top: 1px solid var(--border);
    }

    .learn-more {
        font-size: 13px;
        color: var(--text-secondary);
        display: flex;
        align-items: center;
        gap: 8px;
        transition: color 0.2s;
    }

    .arrow {
        transition: transform 0.2s;
        display: inline-block;
    }

    @media (max-width: 768px) {
        .cases-grid {
            grid-template-columns: 1fr;
        }
    }
</style>
