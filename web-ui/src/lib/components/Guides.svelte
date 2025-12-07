<script lang="ts">
    import { createEventDispatcher } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    
    const dispatch = createEventDispatcher<{
        navigate: { view: string };
        tryNow: void;
    }>();

    function handleTryNow() {
        dispatch("tryNow");
    }

    const steps = [
        {
            title: "1. Instant Terminal Access",
            icon: "bolt",
            description: "We believe you shouldn't wait for a progress bar. Your terminal is ready to accept commands immediately upon creation.",
            details: [
                "**Zero Waiting**: The terminal starts in milliseconds.",
                "**Immediate Input**: You can start typing standard Linux commands right away.",
                "**Basic Tools**: Core utilities (ls, cd, grep) are available instantly."
            ]
        },
        {
            title: "2. Silent Background Setup",
            icon: "settings",
            description: "While you start working, Rexec silently provisions your environment in the background without blocking you.",
            details: [
                "**Non-Blocking**: Heavy tools install while you work.",
                "**Smart Prioritization**: Essential shell configuration happens first.",
                "**Visual Feedback**: A subtle indicator shows progress without locking the UI."
            ]
        },
        {
            title: "3. Seamless Transition",
            icon: "check",
            description: "Once the background setup is complete, your terminal automatically upgrades to the full experience.",
            details: [
                "**Auto-Upgrade**: Your shell upgrades to Zsh with Oh-My-Zsh automatically.",
                "**Tool Availability**: Compilers, SDKs, and CLIs become available instantly.",
                "**No Restart Needed**: The transition happens live in your current session."
            ]
        }
    ];
</script>

<svelte:head>
    <title>Rexec Product Guide - Instant Terminal Architecture</title>
    <meta name="description" content="Learn how Rexec delivers instant access to Linux terminals while silently provisioning complex environments in the background." />
    <meta property="og:title" content="Rexec Product Guide" />
    <meta property="og:description" content="Learn how Rexec delivers instant access to Linux terminals while silently provisioning complex environments in the background." />
</svelte:head>

<div class="guides-page">
    <div class="page-header">
        <div class="header-badge">
            <span class="dot"></span>
            <span>Product Philosophy</span>
        </div>
        <h1>Instant Access, <span class="accent">Zero Wait</span></h1>
        <p class="subtitle">
            How Rexec delivers a fully configured environment without making you stare at a loading screen.
        </p>
    </div>

    <div class="steps-container">
        {#each steps as step, i}
            <div class="step-card">
                <div class="step-icon">
                    <StatusIcon status={step.icon} size={32} />
                    <div class="step-line" class:last={i === steps.length - 1}></div>
                </div>
                <div class="step-content">
                    <h2>{step.title}</h2>
                    <p class="description">{step.description}</p>
                    <div class="details-grid">
                        {#each step.details as detail}
                            <div class="detail-item">
                                <span class="check">✓</span>
                                <!-- Render rudimentary markdown bolding -->
                                {@html detail.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')}
                            </div>
                        {/each}
                    </div>
                </div>
            </div>
        {/each}
    </div>

    <section class="cta-section">
        <h2>Ready to feel the speed?</h2>
        <p>Start a terminal and type `ls` before the setup finishes.</p>
        <button class="btn btn-primary btn-lg" on:click={handleTryNow}>
            <StatusIcon status="rocket" size={16} />
            <span>Launch Terminal</span>
        </button>
    </section>
</div>

<style>
    .guides-page {
        max-width: 1200px;
        margin: 0 auto;
        padding: 60px 20px;
    }

    .page-header {
        text-align: center;
        margin-bottom: 80px;
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
        font-size: 42px;
        font-weight: 700;
        margin-bottom: 16px;
        letter-spacing: -1px;
    }

    h1 .accent {
        color: var(--accent);
        text-shadow: var(--accent-glow);
    }

    .subtitle {
        font-size: 18px;
        color: var(--text-muted);
        max-width: 600px;
        margin: 0 auto;
        line-height: 1.6;
    }

    .steps-container {
        display: flex;
        flex-direction: column;
        gap: 0;
        margin-bottom: 60px;
    }

    .step-card {
        display: flex;
        gap: 50px;
        padding-bottom: 60px;
    }

    .step-icon {
        display: flex;
        flex-direction: column;
        align-items: center;
        flex-shrink: 0;
        width: 60px;
    }

    .step-line {
        flex: 1;
        width: 2px;
        background: var(--border);
        margin-top: 20px;
    }

    .step-line.last {
        background: linear-gradient(to bottom, var(--border), transparent);
    }

    .step-content {
        flex: 1;
        background: var(--bg-card);
        border: 1px solid var(--border);
        padding: 40px;
        border-radius: 8px;
        margin-top: -10px; /* Align with icon */
    }

    .step-content h2 {
        font-size: 24px;
        margin: 0 0 12px 0;
        color: var(--text);
    }

    .description {
        font-size: 16px;
        color: var(--text-secondary);
        line-height: 1.6;
        margin-bottom: 30px;
    }

    .details-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 16px;
    }

    .detail-item {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        font-size: 14px;
        color: var(--text-muted);
        line-height: 1.5;
    }

    .check {
        color: var(--accent);
        font-weight: bold;
    }

    /* Use global styles for strong tags inside svelte html */
    .detail-item :global(strong) {
        color: var(--text);
        font-weight: 600;
    }

    .cta-section {
        text-align: center;
        padding: 80px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
    }

    .cta-section h2 {
        font-size: 32px;
        margin-bottom: 12px;
    }

    .cta-section p {
        color: var(--text-muted);
        margin-bottom: 32px;
        font-size: 18px;
    }

    @keyframes blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0; }
    }

    @media (max-width: 768px) {
        .step-card {
            flex-direction: column;
            gap: 20px;
        }
        
        .step-icon {
            flex-direction: row;
            align-items: center;
            gap: 16px;
            width: 100%;
        }

        .step-line {
            display: none;
        }

        .step-content {
            margin-top: 0;
            padding: 24px;
        }
    }
</style>