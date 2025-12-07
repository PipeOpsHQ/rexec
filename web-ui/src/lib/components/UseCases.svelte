<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    
    const dispatch = createEventDispatcher<{
        tryNow: void;
    }>();

    function handleTryNow() {
        dispatch("tryNow");
    }

    const useCases = [
        {
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
            title: "Collaborative Intelligence",
            icon: "ai",
            description: "A shared workspace for humans and AI agents. Let LLMs execute code in a real, safe environment while you supervise.",
            points: [
                "Sandboxed execution for autonomous agents",
                "Human-in-the-loop oversight",
                "Persistent context for long-running agent tasks",
                "Standardized toolchain for consistent AI output"
            ]
        },
        {
            title: "Universal Jump Host",
            icon: "shield",
            description: "Access your private infrastructure securely from any browser. No VPNs, no complex SSH config management.",
            points: [
                "Browser-based SSH client",
                "Access private VPCs/subnets securely",
                "Manage keys centrally",
                "Audit logs for all session activity"
            ]
        },
        {
            title: "Instant Education & Onboarding",
            icon: "book",
            description: "Onboard new engineers in seconds, not days. Provide pre-configured environments for workshops and tutorials.",
            points: [
                "Standardized team environments",
                "Interactive documentation that runs",
                "Zero friction for workshop attendees",
                "Focus on learning, not configuring"
            ]
        }
    ];
</script>

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
        {#each useCases as useCase}
            <div class="case-card">
                <div class="case-icon">
                    <StatusIcon status={useCase.icon} size={32} />
                </div>
                <h3>{useCase.title}</h3>
                <p class="case-description">{useCase.description}</p>
                <ul class="case-points">
                    {#each useCase.points as point}
                        <li>
                            <span class="bullet">•</span>
                            {point}
                        </li>
                    {/each}
                </ul>
            </div>
        {/each}
    </div>

    <section class="cta-section">
        <h2>Ready to explore?</h2>
        <p>Start your first session and see what's possible.</p>
        <button class="btn btn-primary btn-lg" on:click={handleTryNow}>
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
        transition: transform 0.2s, border-color 0.2s;
        display: flex;
        flex-direction: column;
    }

    .case-card:hover {
        transform: translateY(-4px);
        border-color: var(--accent);
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
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
</style>
