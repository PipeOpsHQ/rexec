<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    
    const dispatch = createEventDispatcher<{
        tryNow: void;
        navigate: { slug: string };
    }>();

    let mouseX = 0;
    let mouseY = 0;

    function handleMouseMove(e: MouseEvent) {
        mouseX = e.clientX;
        mouseY = e.clientY;
        
        const cards = document.querySelectorAll('.use-case-card');
        cards.forEach((card: Element) => {
            const rect = card.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
            (card as HTMLElement).style.setProperty('--mouse-x', `${x}px`);
            (card as HTMLElement).style.setProperty('--mouse-y', `${y}px`);
        });
    }

    function handleTryNow() {
        dispatch("tryNow");
    }

    function navigateToCase(slug: string) {
        dispatch("navigate", { slug });
    }

    const useCases = [
        {
            slug: "ephemeral-dev-environments",
            title: "Disposable DevEnvs",
            icon: "bolt",
            tagline: "Fresh state. Every time.",
            description: "Spin up a fresh, clean environment for every task. Zero drift, zero cleanup."
        },
        {
            slug: "universal-jump-host",
            title: "Secure Gateway",
            icon: "shield",
            tagline: "Zero-trust access.",
            description: "Access private VPCs securely from any browser without VPNs."
        },
        {
            slug: "collaborative-intelligence",
            title: "AI Playground",
            icon: "ai",
            tagline: "Safe agent execution.",
            description: "A sandboxed workspace for humans and AI agents to build together."
        },
        {
            slug: "technical-interviews",
            title: "Live Interviews",
            icon: "terminal",
            tagline: "Real code. Real time.",
            description: "Conduct coding interviews in a real Linux environment, not a web editor."
        },
        {
            slug: "open-source-review",
            title: "PR Review",
            icon: "connected",
            tagline: "One-click testing.",
            description: "Review Pull Requests by instantly spinning up the branch in a clean container."
        },
        {
            slug: "gpu-terminals",
            title: "GPU Terminals",
            icon: "gpu",
            tagline: "AI/ML Ready.",
            description: "Instant access to H100s/A100s for model training and fine-tuning.",
            comingSoon: true
        },
        {
            slug: "edge-device-development",
            title: "Edge Emulation",
            icon: "wifi",
            tagline: "IoT in the cloud.",
            description: "Develop and test for ARM/RISC-V devices without physical hardware."
        },
        {
            slug: "real-time-data-processing",
            title: "Data Pipelines",
            icon: "data",
            tagline: "Stream processing.",
            description: "Build and test streaming ETL pipelines with Kafka and Flink."
        }
    ];
</script>

<svelte:window on:mousemove={handleMouseMove} />

<div class="usecases-page">
    <div class="background-mesh"></div>
    
    <div class="page-header">
        <div class="header-badge">
            <span class="dot"></span>
            <span>PROTOCOLS</span>
        </div>
        <h1>Deployment <span class="gradient-text">Scenarios</span></h1>
        <p class="subtitle">
            Choose your workflow. Rexec adapts to your needs with specialized environments.
        </p>
    </div>

    <div class="cases-grid">
        {#each useCases as useCase, i}
            <button 
                class="use-case-card" 
                class:coming-soon={useCase.comingSoon}
                style="--delay: {i * 50}ms"
                on:click={() => navigateToCase(useCase.slug)}
            >
                <div class="card-glow"></div>
                <div class="card-content">
                    <div class="card-header">
                        <div class="icon-box">
                            <StatusIcon status={useCase.icon} size={24} />
                        </div>
                        {#if useCase.comingSoon}
                            <span class="badge">SOON</span>
                        {/if}
                    </div>
                    
                    <h3>{useCase.title}</h3>
                    <p class="tagline">{useCase.tagline}</p>
                    <p class="description">{useCase.description}</p>
                    
                    <div class="card-footer">
                        <span class="learn-more">Explore Protocol</span>
                        <span class="arrow">→</span>
                    </div>
                </div>
            </button>
        {/each}
    </div>

    <section class="cta-section">
        <div class="cta-content">
            <h2>Ready to build?</h2>
            <button class="btn-primary" on:click={handleTryNow}>
                <span class="btn-text">Initialize Session</span>
                <div class="btn-shine"></div>
            </button>
        </div>
    </section>
</div>

<style>
    .usecases-page {
        min-height: 100vh;
        background-color: #050505;
        color: #fff;
        padding: 80px 20px;
        position: relative;
        overflow: hidden;
        font-family: 'Inter', system-ui, -apple-system, sans-serif;
    }

    /* Background */
    .background-mesh {
        position: fixed;
        inset: 0;
        background-image: 
            radial-gradient(circle at 15% 50%, rgba(30, 30, 30, 0.4) 0%, transparent 25%),
            radial-gradient(circle at 85% 30%, rgba(20, 40, 30, 0.4) 0%, transparent 25%);
        z-index: 0;
        pointer-events: none;
    }

    .background-mesh::after {
        content: "";
        position: absolute;
        inset: 0;
        background: url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%231a1a1a' fill-opacity='0.4'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E");
        opacity: 0.5;
    }

    .page-header {
        position: relative;
        z-index: 2;
        text-align: center;
        margin-bottom: 80px;
        max-width: 800px;
        margin-left: auto;
        margin-right: auto;
    }

    .header-badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 4px 12px;
        background: rgba(255, 255, 255, 0.05);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 20px;
        font-size: 11px;
        color: #00ffaa;
        margin-bottom: 24px;
        text-transform: uppercase;
        letter-spacing: 1px;
        font-family: 'JetBrains Mono', monospace;
    }

    .header-badge .dot {
        width: 6px;
        height: 6px;
        background: #00ffaa;
        border-radius: 50%;
        box-shadow: 0 0 10px #00ffaa;
        animation: pulse 2s infinite;
    }

    h1 {
        font-size: 48px;
        font-weight: 700;
        margin-bottom: 24px;
        letter-spacing: -1px;
        line-height: 1.1;
    }

    .gradient-text {
        background: linear-gradient(90deg, #fff 0%, #888 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .subtitle {
        font-size: 18px;
        color: #888;
        line-height: 1.6;
    }

    /* Grid */
    .cases-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 24px;
        max-width: 1200px;
        margin: 0 auto 80px auto;
        position: relative;
        z-index: 2;
    }

    .use-case-card {
        background: rgba(10, 10, 10, 0.6);
        border: 1px solid rgba(255, 255, 255, 0.05);
        border-radius: 16px;
        padding: 32px;
        text-align: left;
        cursor: pointer;
        position: relative;
        overflow: hidden;
        transition: transform 0.3s ease;
        display: flex;
        flex-direction: column;
        animation: slideUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) backwards;
        animation-delay: var(--delay);
        backdrop-filter: blur(10px);
    }

    .use-case-card:hover {
        transform: translateY(-4px);
        background: rgba(20, 20, 20, 0.8);
    }

    /* Spotlight */
    .use-case-card::before {
        content: "";
        position: absolute;
        top: 0; left: 0; right: 0; bottom: 0;
        background: radial-gradient(600px circle at var(--mouse-x) var(--mouse-y), rgba(255, 255, 255, 0.04), transparent 40%);
        z-index: 1;
        opacity: 0;
        transition: opacity 0.3s;
        pointer-events: none;
    }

    .use-case-card:hover::before {
        opacity: 1;
    }

    .card-content {
        position: relative;
        z-index: 2;
        height: 100%;
        display: flex;
        flex-direction: column;
    }

    .card-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 24px;
    }

    .icon-box {
        width: 48px;
        height: 48px;
        background: rgba(255, 255, 255, 0.03);
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #fff;
        border: 1px solid rgba(255, 255, 255, 0.05);
        transition: all 0.3s ease;
    }

    .use-case-card:hover .icon-box {
        background: rgba(255, 255, 255, 0.08);
        transform: scale(1.05);
        color: #00ffaa;
        border-color: rgba(0, 255, 170, 0.2);
    }

    .badge {
        font-size: 10px;
        background: rgba(255, 238, 0, 0.1);
        color: #ffee00;
        padding: 4px 8px;
        border-radius: 4px;
        font-family: 'JetBrains Mono', monospace;
        letter-spacing: 1px;
    }

    h3 {
        font-size: 20px;
        font-weight: 600;
        margin: 0 0 8px 0;
        color: #fff;
    }

    .tagline {
        font-size: 13px;
        color: #00ffaa;
        font-family: 'JetBrains Mono', monospace;
        margin: 0 0 12px 0;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .description {
        font-size: 14px;
        color: #888;
        line-height: 1.6;
        margin: 0 0 24px 0;
        flex-grow: 1;
    }

    .card-footer {
        display: flex;
        align-items: center;
        gap: 8px;
        padding-top: 20px;
        border-top: 1px solid rgba(255, 255, 255, 0.05);
        margin-top: auto;
    }

    .learn-more {
        font-size: 13px;
        color: #666;
        font-weight: 500;
        transition: color 0.2s;
    }

    .arrow {
        color: #444;
        transition: transform 0.2s, color 0.2s;
    }

    .use-case-card:hover .learn-more {
        color: #fff;
    }

    .use-case-card:hover .arrow {
        color: #00ffaa;
        transform: translateX(4px);
    }

    .cta-section {
        position: relative;
        z-index: 2;
        text-align: center;
        padding: 60px 0;
    }

    .btn-primary {
        background: #fff;
        color: #000;
        border: none;
        padding: 16px 32px;
        border-radius: 30px;
        font-size: 15px;
        font-weight: 600;
        cursor: pointer;
        position: relative;
        overflow: hidden;
        transition: transform 0.2s;
    }

    .btn-primary:hover {
        transform: scale(1.05);
        box-shadow: 0 0 30px rgba(255, 255, 255, 0.2);
    }

    .btn-shine {
        position: absolute;
        top: 0;
        left: -100%;
        width: 100%;
        height: 100%;
        background: linear-gradient(90deg, transparent, rgba(255,255,255,0.8), transparent);
        animation: shine 3s infinite;
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; transform: scale(1); }
        50% { opacity: 0.5; transform: scale(0.8); }
    }

    @keyframes slideUp {
        from { opacity: 0; transform: translateY(20px); }
        to { opacity: 1; transform: translateY(0); }
    }

    @keyframes shine {
        0% { left: -100%; }
        20% { left: 100%; }
        100% { left: 100%; }
    }

    @media (max-width: 768px) {
        h1 { font-size: 36px; }
        .cases-grid { grid-template-columns: 1fr; }
    }
</style>