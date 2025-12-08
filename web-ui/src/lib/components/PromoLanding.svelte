<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    import { auth } from "$stores/auth";
    import { toast } from "$stores/toast";

    const dispatch = createEventDispatcher<{
        guest: void;
        navigate: { view: string; slug?: string };
    }>();

    let scrollY = 0;
    let innerHeight = 0;
    let innerWidth = 0;
    let mouseX = 0;
    let mouseY = 0;

    // Terminal State
    let terminalLines = [
        { text: "initiating handshake...", color: "text-muted" },
        { text: "connecting to global mesh...", color: "text-muted" },
        { text: "allocating ephemeral resources...", color: "text-success" },
        { text: "mounting secure filesystem...", color: "text-success" },
        { text: "environment ready.", color: "text-accent" },
        { text: "$ _", color: "text-white", type: "prompt" }
    ];
    let visibleLines: typeof terminalLines = [];

    // Use Cases Data
    const useCases = [
        {
            slug: "ephemeral-dev-environments",
            title: "Disposable DevEnvs",
            desc: "Spin up a fresh, clean state for every task or PR. No drift.",
            icon: "box",
            col: "span-2"
        },
        {
            slug: "universal-jump-host",
            title: "Secure Gateway",
            desc: "Zero-trust access to private VPCs without VPNs.",
            icon: "shield",
            col: "span-1"
        },
        {
            slug: "collaborative-intelligence",
            title: "AI Playground",
            desc: "Safe, sandboxed execution for autonomous agents.",
            icon: "ai",
            col: "span-1"
        },
        {
            slug: "technical-interviews",
            title: "Live Interviews",
            desc: "Real-time multiplayer coding in a real Linux shell.",
            icon: "code",
            col: "span-2"
        }
    ];

    onMount(() => {
        // Terminal Typewriter
        let lineIndex = 0;
        const interval = setInterval(() => {
            if (lineIndex < terminalLines.length) {
                visibleLines = [...visibleLines, terminalLines[lineIndex]];
                lineIndex++;
            } else {
                clearInterval(interval);
            }
        }, 800);

        return () => clearInterval(interval);
    });

    function handleMouseMove(e: MouseEvent) {
        mouseX = e.clientX;
        mouseY = e.clientY;
        
        // Update CSS variables for spotlight effect on cards
        const cards = document.querySelectorAll('.bento-card');
        cards.forEach((card: Element) => {
            const rect = card.getBoundingClientRect();
            const x = e.clientX - rect.left;
            const y = e.clientY - rect.top;
            (card as HTMLElement).style.setProperty('--mouse-x', `${x}px`);
            (card as HTMLElement).style.setProperty('--mouse-y', `${y}px`);
        });
    }

    function handleGuestClick() {
        dispatch("guest");
    }

    async function handleOAuthLogin() {
        try {
            const url = await auth.getOAuthUrl();
            if (url) window.location.href = url;
        } catch (e) {
            toast.error("Connection failed.");
        }
    }

    function navigateToUseCase(slug: string) {
        // Since we are now full screen, we can just use location href to ensure clean state
        // or dispatch if the parent handles it.
        // Let's use dispatch to be SPA-friendly if possible, but the parent logic needs to support it.
        // The parent App.svelte logic for 'navigate' just sets currentView.
        // So we will use window.location.href for simplicity and robustness with the slug routing.
        window.location.href = `/use-cases/${slug}`;
    }

    // Calculated transforms
    $: heroScale = 1 - Math.min(scrollY / innerHeight, 0.2);
    $: heroOpacity = 1 - Math.min(scrollY / (innerHeight * 0.5), 1);
    $: terminalTiltX = (mouseY - innerHeight / 2) / 50;
    $: terminalTiltY = (mouseX - innerWidth / 2) / 50;
</script>

<svelte:window 
    bind:scrollY 
    bind:innerHeight 
    bind:innerWidth 
    on:mousemove={handleMouseMove} 
/>

<div class="promo-page">
    <div class="background-mesh">
        <div class="grid-overlay"></div>
    </div>
    
    <nav class="nav-bar" class:scrolled={scrollY > 50}>
        <div class="nav-content">
            <div class="brand">REXEC</div>
            <div class="nav-actions">
                <button class="nav-link" on:click={() => window.location.href = '/pricing'}>Pricing</button>
                <button class="nav-link" on:click={() => window.location.href = '/guides'}>Docs</button>
                <button class="btn-signin" on:click={handleOAuthLogin}>Sign In</button>
            </div>
        </div>
    </nav>

    <main class="content">
        <!-- HERO SECTION -->
        <section class="hero">
            <div class="hero-bg-glow"></div>
            
            <div 
                class="hero-content" 
                style="transform: scale({heroScale}); opacity: {heroOpacity};"
            >
                <h1 class="hero-title">
                    Infrastructure <br />
                    <span class="gradient-text">for the Impatient.</span>
                </h1>
                <p class="hero-subtitle">
                    Instant, ephemeral Linux environments. <br />
                    No setup. No cleanup. Just code.
                </p>
                
                <div class="hero-cta">
                    <button class="btn-primary" on:click={handleGuestClick}>
                        <span class="btn-text">Start Instant Session</span>
                        <div class="btn-shine"></div>
                    </button>
                    <div class="copy-command">
                        <span class="prompt">$</span>
                        <code>curl -sL rexec.dev/install | bash</code>
                    </div>
                </div>
            </div>

            <!-- 3D TERMINAL -->
            <div 
                class="terminal-stage"
                style="transform: perspective(1000px) rotateX({-terminalTiltX}deg) rotateY({terminalTiltY}deg)"
            >
                <div class="terminal-window">
                    <div class="terminal-header">
                        <div class="controls">
                            <span class="dot red"></span>
                            <span class="dot yellow"></span>
                            <span class="dot green"></span>
                        </div>
                        <div class="tab">root@rexec-node</div>
                    </div>
                    <div class="terminal-body">
                        {#each visibleLines as line}
                            <div class="line {line.color}">
                                {#if line.type === 'prompt'}
                                    <span class="arrow">➜</span> <span class="path">~</span>
                                    <span class="cursor">_</span>
                                {:else}
                                    <span class="timestamp">[{new Date().toLocaleTimeString('en-US', {hour12: false, hour: '2-digit', minute:'2-digit', second:'2-digit'})}]</span>
                                    {line.text}
                                {/if}
                            </div>
                        {/each}
                    </div>
                    <div class="reflection"></div>
                </div>
            </div>
        </section>

        <!-- BENTO GRID FEATURES -->
        <section class="bento-section">
            <div class="section-label">Capabilities</div>
            <h2 class="section-title">Engineered for <span class="text-white">Speed</span></h2>
            
            <div class="bento-grid">
                <!-- Large Card -->
                <div class="bento-card span-2 highlight-card">
                    <div class="card-bg-anim"></div>
                    <div class="card-content">
                        <div class="icon-box"><StatusIcon status="bolt" size={24}/></div>
                        <h3>Sub-second Init</h3>
                        <p>Cold starts are a thing of the past. Our microVMs boot faster than you can blink.</p>
                        <div class="stat-display">
                            <div class="stat-value">&lt; 300ms</div>
                            <div class="stat-label">Boot Time</div>
                        </div>
                    </div>
                </div>

                <!-- Tall Card -->
                <div class="bento-card span-1 row-span-2 tall-card">
                    <div class="card-content">
                        <div class="icon-box"><StatusIcon status="globe" size={24}/></div>
                        <h3>Global Mesh</h3>
                        <p>Connect from anywhere. Your session follows you across the edge.</p>
                        <div class="mesh-viz">
                            <div class="mesh-point p1"></div>
                            <div class="mesh-point p2"></div>
                            <div class="mesh-point p3"></div>
                            <div class="mesh-line l1"></div>
                            <div class="mesh-line l2"></div>
                        </div>
                    </div>
                </div>

                <!-- Standard Cards -->
                <div class="bento-card span-1">
                    <div class="card-content">
                        <div class="icon-box"><StatusIcon status="shield" size={24}/></div>
                        <h3>Isolated</h3>
                        <p>Ephemeral filesystems that vanish on exit.</p>
                    </div>
                </div>

                <div class="bento-card span-2">
                    <div class="card-content">
                        <div class="icon-box"><StatusIcon status="connected" size={24}/></div>
                        <h3>P2P Networking</h3>
                        <p>Direct peer-to-peer connections for low-latency collaboration.</p>
                    </div>
                </div>
            </div>
        </section>

        <!-- USE CASES CAROUSEL/LIST -->
        <section class="use-cases-section">
            <div class="section-label">Workflows</div>
            <h2 class="section-title">Build Impossible Things</h2>
            
            <div class="use-cases-list">
                {#each useCases as useCase}
                    <!-- svelte-ignore a11y-click-events-have-key-events -->
                    <div class="use-case-row" on:click={() => navigateToUseCase(useCase.slug)} role="button" tabindex="0">
                        <div class="uc-icon">
                            <StatusIcon status={useCase.icon} size={20} />
                        </div>
                        <div class="uc-info">
                            <h4>{useCase.title}</h4>
                            <p>{useCase.desc}</p>
                        </div>
                        <button class="uc-arrow">→</button>
                    </div>
                {/each}
            </div>
        </section>

        <!-- CTA -->
        <section class="final-cta">
            <div class="cta-container">
                <h2>Ready to ship?</h2>
                <p>Join thousands of developers building on the edge.</p>
                <button class="btn-primary large" on:click={handleGuestClick}>
                    <span class="btn-text">Launch Terminal</span>
                    <div class="btn-shine"></div>
                </button>
            </div>
        </section>
    </main>

    <footer>
        <div class="footer-links">
            <span>© 2025 Rexec</span>
            <a href="/pricing">Pricing</a>
            <a href="/guides">Guides</a>
            <a href="https://github.com/rexec/rexec">GitHub</a>
        </div>
    </footer>
</div>

<style>
    :global(body) {
        background-color: #050505;
        margin: 0;
        font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
        color: #fff;
        overflow-x: hidden;
    }

    .promo-page {
        min-height: 100vh;
        background: #050505;
        position: relative;
    }

    /* Background */
    .background-mesh {
        position: fixed;
        inset: 0;
        z-index: 0;
        pointer-events: none;
        overflow: hidden;
    }

    .grid-overlay {
        position: absolute;
        inset: -50%;
        width: 200%;
        height: 200%;
        background-image: 
            linear-gradient(rgba(0, 255, 170, 0.03) 1px, transparent 1px),
            linear-gradient(90deg, rgba(0, 255, 170, 0.03) 1px, transparent 1px);
        background-size: 60px 60px;
        transform: rotateX(60deg) translateY(-100px) translateZ(-200px);
        animation: grid-move 20s linear infinite;
        opacity: 0.3;
    }

    @keyframes grid-move {
        0% { transform: rotateX(60deg) translateY(0) translateZ(-200px); }
        100% { transform: rotateX(60deg) translateY(60px) translateZ(-200px); }
    }

    /* --- Navigation --- */
    .nav-bar {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        z-index: 100;
        padding: 20px 0;
        transition: all 0.3s ease;
    }

    .nav-bar.scrolled {
        background: rgba(0, 0, 0, 0.6);
        backdrop-filter: blur(12px);
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        padding: 16px 0;
    }

    .nav-content {
        max-width: 1200px;
        margin: 0 auto;
        padding: 0 24px;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .brand {
        font-family: 'JetBrains Mono', monospace;
        font-weight: 800;
        font-size: 18px;
        letter-spacing: -0.5px;
    }

    .nav-actions {
        display: flex;
        gap: 24px;
        align-items: center;
    }

    .nav-link {
        background: none;
        border: none;
        color: #888;
        font-size: 14px;
        cursor: pointer;
        transition: color 0.2s;
    }

    .nav-link:hover { color: #fff; }

    .btn-signin {
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.1);
        color: #fff;
        padding: 8px 16px;
        border-radius: 20px;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn-signin:hover { background: rgba(255, 255, 255, 0.15); }

    /* --- Hero --- */
    .hero {
        height: 100vh;
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        text-align: center;
        position: relative;
        padding-top: 60px; /* Offset for nav */
    }

    .hero-bg-glow {
        position: absolute;
        width: 600px;
        height: 600px;
        background: radial-gradient(circle, rgba(0, 255, 170, 0.15) 0%, transparent 70%);
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        filter: blur(80px);
        z-index: 0;
        pointer-events: none;
    }

    .hero-content {
        position: relative;
        z-index: 2;
        margin-bottom: 60px;
        transition: transform 0.1s linear, opacity 0.1s linear;
    }

    .hero-title {
        font-size: 80px;
        line-height: 1;
        font-weight: 700;
        letter-spacing: -2px;
        margin-bottom: 24px;
        background: linear-gradient(180deg, #fff 0%, rgba(255,255,255,0.7) 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .gradient-text {
        background: linear-gradient(90deg, #00FFA3 0%, #00D1FF 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .hero-subtitle {
        font-size: 20px;
        color: #888;
        line-height: 1.5;
        margin-bottom: 40px;
    }

    .hero-cta {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 16px;
    }

    .btn-primary {
        background: #fff;
        color: #000;
        border: none;
        padding: 16px 40px;
        border-radius: 30px;
        font-size: 16px;
        font-weight: 600;
        cursor: pointer;
        position: relative;
        overflow: hidden;
        transition: transform 0.2s, box-shadow 0.2s;
    }

    .btn-primary:hover {
        transform: scale(1.05);
        box-shadow: 0 0 30px rgba(255, 255, 255, 0.3);
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

    @keyframes shine {
        0% { left: -100%; }
        20% { left: 100%; }
        100% { left: 100%; }
    }

    .copy-command {
        font-family: 'JetBrains Mono', monospace;
        font-size: 13px;
        color: #666;
        background: rgba(255, 255, 255, 0.05);
        padding: 8px 16px;
        border-radius: 12px;
        border: 1px solid rgba(255, 255, 255, 0.1);
        display: flex;
        gap: 10px;
    }

    .prompt { color: #00FFA3; }

    /* --- Terminal 3D --- */
    .terminal-stage {
        perspective: 1000px;
        z-index: 2;
        width: 90%;
        max-width: 700px;
    }

    .terminal-window {
        background: rgba(10, 10, 10, 0.8);
        backdrop-filter: blur(20px);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 12px;
        box-shadow: 
            0 20px 50px rgba(0,0,0,0.5),
            0 0 0 1px rgba(255,255,255,0.05);
        overflow: hidden;
        transition: transform 0.1s ease-out; /* Smooth follow */
    }

    .terminal-header {
        background: rgba(255, 255, 255, 0.03);
        padding: 12px 16px;
        display: flex;
        align-items: center;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .controls { display: flex; gap: 8px; }
    .dot { width: 10px; height: 10px; border-radius: 50%; }
    .red { background: #FF5F56; }
    .yellow { background: #FFBD2E; }
    .green { background: #27C93F; }

    .tab {
        flex: 1;
        text-align: center;
        font-family: 'JetBrains Mono', monospace;
        font-size: 12px;
        color: #666;
    }

    .terminal-body {
        padding: 20px;
        font-family: 'JetBrains Mono', monospace;
        font-size: 13px;
        text-align: left;
        min-height: 240px;
    }

    .line { margin-bottom: 6px; }
    .line.text-muted { color: #555; }
    .line.text-success { color: #27C93F; }
    .line.text-accent { color: #00FFA3; }
    .line.text-white { color: #fff; }
    
    .arrow { color: #27C93F; margin-right: 6px; }
    .path { color: #00D1FF; margin-right: 6px; }
    .timestamp { color: #444; margin-right: 8px; }
    .cursor { animation: blink 1s step-end infinite; }

    /* --- Bento Grid --- */
    .bento-section {
        max-width: 1000px;
        margin: 100px auto;
        padding: 0 24px;
        position: relative;
        z-index: 5;
    }

    .section-label {
        color: #00FFA3;
        font-size: 13px;
        text-transform: uppercase;
        letter-spacing: 1px;
        margin-bottom: 12px;
        font-weight: 600;
    }

    .section-title {
        font-size: 42px;
        margin: 0 0 60px 0;
        color: #888;
        font-weight: 600;
        letter-spacing: -1px;
    }

    .text-white { color: #fff; }

    .bento-grid {
        display: grid;
        grid-template-columns: 1fr 1fr 1fr;
        grid-auto-rows: 240px;
        gap: 20px;
    }

    .bento-card {
        background: rgba(10, 10, 10, 0.7);
        backdrop-filter: blur(10px);
        border-radius: 20px;
        border: 1px solid rgba(255, 255, 255, 0.05);
        padding: 30px;
        position: relative;
        overflow: hidden;
        cursor: pointer;
        transition: transform 0.3s ease;
    }

    /* Spotlight Effect Logic */
    .bento-card::before {
        content: "";
        position: absolute;
        top: 0; left: 0; right: 0; bottom: 0;
        background: radial-gradient(800px circle at var(--mouse-x) var(--mouse-y), rgba(255, 255, 255, 0.06), transparent 40%);
        z-index: 1;
        opacity: 0;
        transition: opacity 0.3s;
    }

    .bento-card:hover::before { opacity: 1; }
    .bento-card:hover { transform: translateY(-4px); }

    .bento-card.span-2 { grid-column: span 2; }
    .bento-card.span-1 { grid-column: span 1; }
    .bento-card.row-span-2 { grid-row: span 2; }

    .card-content {
        position: relative;
        z-index: 2;
        height: 100%;
        display: flex;
        flex-direction: column;
    }

    .icon-box {
        width: 48px;
        height: 48px;
        background: rgba(255,255,255,0.05);
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-bottom: 20px;
        color: #fff;
    }

    .bento-card h3 {
        font-size: 20px;
        margin: 0 0 10px 0;
        font-weight: 600;
    }

    .bento-card p {
        font-size: 14px;
        color: #888;
        line-height: 1.5;
        margin: 0;
        max-width: 90%;
    }

    .highlight-card {
        background: linear-gradient(135deg, rgba(255,255,255,0.05) 0%, rgba(255,255,255,0.02) 100%);
    }

    .stat-display {
        margin-top: auto;
        border-top: 1px solid rgba(255,255,255,0.1);
        padding-top: 16px;
    }

    .stat-value {
        font-size: 32px;
        font-weight: 700;
        color: #00FFA3;
        font-family: 'JetBrains Mono', monospace;
    }

    .stat-label {
        font-size: 12px;
        color: #666;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    /* Mesh Viz in Tall Card */
    .mesh-viz {
        position: relative;
        flex-grow: 1;
        margin-top: 20px;
    }
    
    .mesh-point {
        position: absolute;
        width: 8px;
        height: 8px;
        background: #fff;
        border-radius: 50%;
        box-shadow: 0 0 10px rgba(255,255,255,0.5);
    }
    .p1 { top: 20%; left: 20%; animation: float 3s infinite ease-in-out; }
    .p2 { top: 50%; left: 70%; animation: float 4s infinite ease-in-out 1s; }
    .p3 { top: 80%; left: 30%; animation: float 3.5s infinite ease-in-out 0.5s; }

    .mesh-line {
        position: absolute;
        background: rgba(255,255,255,0.2);
        height: 1px;
        transform-origin: left center;
    }
    .l1 { top: 22%; left: 22%; width: 130px; transform: rotate(25deg); }
    .l2 { top: 52%; left: 72%; width: 100px; transform: rotate(140deg); }

    @keyframes float {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(-10px); }
    }

    /* --- Use Cases List --- */
    .use-cases-section {
        max-width: 800px;
        margin: 100px auto;
        padding: 0 24px;
        position: relative;
        z-index: 5;
    }

    .use-cases-list {
        border-top: 1px solid rgba(255,255,255,0.1);
    }

    .use-case-row {
        display: flex;
        align-items: center;
        padding: 24px 0;
        border-bottom: 1px solid rgba(255,255,255,0.1);
        cursor: pointer;
        transition: background 0.2s;
    }

    .use-case-row:hover {
        background: rgba(255,255,255,0.02);
    }

    .use-case-row:hover .uc-arrow {
        transform: translateX(5px);
        color: #fff;
    }

    .uc-icon {
        color: #666;
        margin-right: 24px;
    }

    .uc-info h4 {
        margin: 0 0 4px 0;
        font-size: 16px;
    }

    .uc-info p {
        margin: 0;
        color: #666;
        font-size: 14px;
    }

    .uc-arrow {
        margin-left: auto;
        background: none;
        border: none;
        color: #444;
        font-size: 20px;
        transition: all 0.2s;
    }

    /* --- CTA --- */
    .final-cta {
        padding: 120px 24px;
        text-align: center;
        position: relative;
        z-index: 5;
    }

    .cta-container h2 {
        font-size: 48px;
        margin: 0 0 20px 0;
        background: linear-gradient(180deg, #fff 0%, #666 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .cta-container p {
        color: #888;
        font-size: 18px;
        margin-bottom: 40px;
    }

    .btn-primary.large {
        padding: 20px 48px;
        font-size: 18px;
        background: #fff;
        color: #000;
        border-radius: 40px;
    }

    /* --- Footer --- */
    footer {
        padding: 40px 0;
        border-top: 1px solid rgba(255,255,255,0.05);
        text-align: center;
        position: relative;
        z-index: 5;
    }

    .footer-links {
        display: flex;
        justify-content: center;
        gap: 30px;
        font-size: 13px;
        color: #666;
    }

    .footer-links a {
        color: #666;
        text-decoration: none;
    }

    .footer-links a:hover { color: #fff; }

    /* Mobile */
    @media (max-width: 768px) {
        .hero-title { font-size: 48px; }
        .bento-grid { grid-template-columns: 1fr; grid-auto-rows: auto; }
        .bento-card.span-2 { grid-column: span 1; }
        .bento-card.row-span-2 { grid-row: span 1; }
    }
</style>