<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    import { auth } from "$stores/auth";
    import { toast } from "$stores/toast";

    const dispatch = createEventDispatcher<{
        guest: void;
        navigate: { view: string; slug?: string };
    }>();

    let isOAuthLoading = false;
    let terminalLines = [
        { text: "initiating handshake...", color: "text-muted" },
        { text: "connecting to mesh network...", color: "text-muted" },
        { text: "allocating ephemeral resources [cpu: 4, mem: 8gb]", color: "text-success" },
        { text: "mounting secure filesystem...", color: "text-success" },
        { text: "environment ready.", color: "text-accent" },
        { text: "$ _", color: "text-white", type: "prompt" }
    ];
    let visibleLines: typeof terminalLines = [];

    const useCases = [
        {
            slug: "ephemeral-dev-environments",
            title: "Disposable DevEnvs",
            desc: "Fresh state for every task.",
            icon: "box"
        },
        {
            slug: "universal-jump-host",
            title: "Secure Gateway",
            desc: "Zero-trust infrastructure access.",
            icon: "shield"
        },
        {
            slug: "collaborative-intelligence",
            title: "AI Playground",
            desc: "Safe execution for agents.",
            icon: "ai"
        },
        {
            slug: "technical-interviews",
            title: "Live Coding",
            desc: "Real environments, real time.",
            icon: "code"
        }
    ];

    // Scroll Reveal Action
    function reveal(node: HTMLElement) {
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    node.classList.add('visible');
                    observer.unobserve(node);
                }
            });
        }, { threshold: 0.15 });
        
        observer.observe(node);
        return {
            destroy() {
                observer.disconnect();
            }
        };
    }

    onMount(() => {
        let lineIndex = 0;
        const interval = setInterval(() => {
            if (lineIndex < terminalLines.length) {
                visibleLines = [...visibleLines, terminalLines[lineIndex]];
                lineIndex++;
            } else {
                clearInterval(interval);
            }
        }, 600);

        return () => clearInterval(interval);
    });

    function handleGuestClick() {
        dispatch("guest");
    }

    function navigateToUseCase(slug: string) {
        window.location.href = `/use-cases/${slug}`;
    }

    async function handleOAuthLogin() {
        if (isOAuthLoading) return;

        isOAuthLoading = true;
        try {
            const url = await auth.getOAuthUrl();
            if (url) {
                window.location.href = url;
            } else {
                toast.error("Connection failed.");
                isOAuthLoading = false;
            }
        } catch (e) {
            toast.error("Connection failed.");
            isOAuthLoading = false;
        }
    }
</script>

<div class="promo-page">
    <div class="background-mesh"></div>
    
    <nav class="nav-header">
        <div class="logo">REXEC <span class="version">v2.0</span></div>
        <div class="nav-links">
            <button class="nav-item" on:click={() => window.location.href = '/pricing'}>Pricing</button>
            <button class="nav-item" on:click={() => window.location.href = '/guides'}>Docs</button>
            <button class="nav-item highlight" on:click={handleOAuthLogin}>Sign In</button>
        </div>
    </nav>

    <main class="content">
        <div class="hero-section">
            <div class="hero-text">
                <h1>
                    Infrastructure at the <br />
                    <span class="gradient-text">Speed of Thought</span>
                </h1>
                <p class="hero-sub">
                    Instantly provision secure, ephemeral Linux environments in your browser. 
                    No setup. No cleanup. Just code.
                </p>
                
                <div class="cta-row">
                    <button class="btn-primary" on:click={handleGuestClick}>
                        <span>Start Instant Session</span>
                        <div class="glow"></div>
                    </button>
                    <div class="command-copy">
                        <code>curl -sL rexec.dev/install | bash</code>
                        <button class="copy-btn" aria-label="Copy command">
                            <StatusIcon status="copy" size={14} />
                        </button>
                    </div>
                </div>
            </div>

            <div class="hero-visual">
                <div class="terminal-window">
                    <div class="terminal-header">
                        <div class="dots">
                            <span class="dot red"></span>
                            <span class="dot yellow"></span>
                            <span class="dot green"></span>
                        </div>
                        <span class="title">guest@rexec-node-01:~</span>
                    </div>
                    <div class="terminal-body">
                        {#each visibleLines as line}
                            <div class="line {line.color}">
                                {#if line.type === 'prompt'}
                                    <span class="prompt-char">&gt;</span>
                                    <span class="cursor">_</span>
                                {:else}
                                    <span class="prefix">[system]</span> {line.text}
                                {/if}
                            </div>
                        {/each}
                    </div>
                </div>
                <div class="visual-glow"></div>
            </div>
        </div>

        <div class="features-ticker">
            <div class="ticker-content">
                <div class="ticker-item">
                    <StatusIcon status="bolt" size={18} />
                    <span>Sub-second Init</span>
                </div>
                <div class="separator">/</div>
                <div class="ticker-item">
                    <StatusIcon status="shield" size={18} />
                    <span>Isolated Kernels</span>
                </div>
                <div class="separator">/</div>
                <div class="ticker-item">
                    <StatusIcon status="globe" size={18} />
                    <span>Global Mesh</span>
                </div>
                <div class="separator">/</div>
                <div class="ticker-item">
                    <StatusIcon status="connected" size={18} />
                    <span>P2P Networking</span>
                </div>
            </div>
        </div>

        <section class="use-cases" use:reveal>
            <div class="section-header">
                <h2>Engineered for <span class="text-white">Modern Workflows</span></h2>
            </div>
            
            <div class="cards-grid">
                {#each useCases as item, i}
                    <button 
                        class="feature-card" 
                        on:click={() => navigateToUseCase(item.slug)}
                        style="--delay: {i * 0.1}s"
                    >
                        <div class="card-icon">
                            <StatusIcon status={item.icon} size={24} />
                        </div>
                        <div class="card-content">
                            <h3>{item.title}</h3>
                            <p>{item.desc}</p>
                        </div>
                        <div class="card-arrow">
                            <StatusIcon status="arrow-left" size={16} /> 
                        </div>
                    </button>
                {/each}
            </div>
        </section>

        <!-- ENHANCED SECTIONS -->

        <section class="architecture-section" use:reveal>
            <div class="section-header">
                <h2>The Rexec <span class="text-white">Architecture</span></h2>
                <p class="section-sub">Built on the edge of the impossible.</p>
            </div>
            
            <div class="arch-diagram">
                <div class="arch-layer">
                    <div class="layer-label">Orchestration Layer</div>
                    <div class="layer-content">Global Mesh Routing & State Mgmt</div>
                    <div class="layer-glow"></div>
                </div>
                <div class="arch-connector">
                    <div class="flow-line"></div>
                </div>
                <div class="arch-layer highlight">
                    <div class="layer-label">Compute Layer</div>
                    <div class="layer-content">MicroVMs + Container Isolation</div>
                    <div class="layer-glow"></div>
                </div>
                <div class="arch-connector">
                    <div class="flow-line"></div>
                </div>
                <div class="arch-layer">
                    <div class="layer-label">Storage Layer</div>
                    <div class="layer-content">Ephemeral Block Storage</div>
                    <div class="layer-glow"></div>
                </div>
            </div>
        </section>

        <section class="compare-section" use:reveal>
            <div class="compare-grid">
                <div class="compare-card">
                    <h3>Traditional VM</h3>
                    <div class="stat-row">
                        <span class="label">Boot Time</span>
                        <span class="value slow">45s - 2m</span>
                    </div>
                    <div class="stat-row">
                        <span class="label">Cost</span>
                        <span class="value">Per Hour (rounded up)</span>
                    </div>
                </div>
                <div class="compare-card highlight">
                    <div class="badge">REXEC</div>
                    <div class="card-glow-bg"></div>
                    <h3>Ephemeral Container</h3>
                    <div class="stat-row">
                        <span class="label">Boot Time</span>
                        <span class="value fast glitch-text" data-text="< 300ms">&lt; 300ms</span>
                    </div>
                    <div class="stat-row">
                        <span class="label">Cost</span>
                        <span class="value">Per Second</span>
                    </div>
                </div>
            </div>
        </section>

        <section class="global-mesh" use:reveal>
            <div class="section-header">
                <h2>Global Edge <span class="text-white">Network</span></h2>
            </div>
            <div class="map-visual">
                <!-- Connected Mesh Network -->
                <svg class="mesh-lines" viewBox="0 0 800 400" preserveAspectRatio="none">
                    <path class="mesh-path" d="M160 120 L200 140 L400 100 L440 120 L640 160 L560 240 L400 100" />
                    <path class="mesh-path" d="M200 140 L560 240" />
                </svg>
                
                <div class="map-grid">
                    <!-- Points -->
                    <div class="map-point" style="top: 30%; left: 20%;">
                        <div class="ripple"></div>
                    </div>
                    <div class="map-point" style="top: 35%; left: 25%;">
                        <div class="ripple" style="animation-delay: 0.2s"></div>
                    </div>
                    <div class="map-point" style="top: 25%; left: 50%;">
                        <div class="ripple" style="animation-delay: 0.5s"></div>
                    </div>
                    <div class="map-point" style="top: 30%; left: 55%;">
                        <div class="ripple" style="animation-delay: 0.7s"></div>
                    </div>
                    <div class="map-point" style="top: 40%; left: 80%;">
                        <div class="ripple" style="animation-delay: 1.1s"></div>
                    </div>
                    <div class="map-point pulse main-node" style="top: 60%; left: 70%;">
                        <div class="ripple"></div>
                        <div class="ripple" style="animation-delay: 0.5s"></div>
                    </div>
                </div>
                <div class="scan-bar"></div>
            </div>
        </section>

        <section class="final-cta" use:reveal>
            <h2>Ready to launch?</h2>
            <button class="btn-primary large" on:click={handleGuestClick}>
                Start Building Now
                <div class="glow"></div>
            </button>
        </section>

    </main>
    
    <footer class="main-footer">
        <div class="footer-content">
            <div class="footer-col">
                <h4>Rexec</h4>
                <a href="/pricing">Pricing</a>
                <a href="/guides">Guides</a>
            </div>
            <div class="footer-col">
                <h4>Legal</h4>
                <!-- svelte-ignore a11y-invalid-attribute -->
                <a href="#">Terms</a>
                <!-- svelte-ignore a11y-invalid-attribute -->
                <a href="#">Privacy</a>
            </div>
            <div class="footer-col">
                <h4>Social</h4>
                <!-- svelte-ignore a11y-invalid-attribute -->
                <a href="#">GitHub</a>
                <!-- svelte-ignore a11y-invalid-attribute -->
                <a href="#">Twitter</a>
            </div>
        </div>
        <div class="footer-bottom">
            &copy; 2025 Rexec Systems.
        </div>
    </footer>
</div>

<style>
    :global(body) {
        background-color: #050505;
        margin: 0;
        font-family: 'Inter', system-ui, -apple-system, sans-serif;
        -webkit-font-smoothing: antialiased;
        -moz-osx-font-smoothing: grayscale;
    }

    .promo-page {
        min-height: 100vh;
        color: #888;
        position: relative;
        overflow-x: hidden;
    }

    /* Scroll Reveal State */
    :global(.visible) {
        opacity: 1 !important;
        transform: translateY(0) !important;
    }

    section {
        opacity: 0;
        transform: translateY(30px);
        transition: opacity 0.8s cubic-bezier(0.16, 1, 0.3, 1), transform 0.8s cubic-bezier(0.16, 1, 0.3, 1);
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

    /* Nav */
    .nav-header {
        position: relative;
        z-index: 10;
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 24px 40px;
        max-width: 1400px;
        margin: 0 auto;
    }

    .logo {
        font-family: 'JetBrains Mono', monospace;
        font-weight: 700;
        font-size: 20px;
        color: #fff;
        letter-spacing: -0.5px;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .version {
        font-size: 10px;
        background: #1a1a1a;
        padding: 2px 6px;
        border-radius: 4px;
        color: #666;
    }

    .nav-links {
        display: flex;
        gap: 32px;
        align-items: center;
    }

    .nav-item {
        background: none;
        border: none;
        color: #888;
        font-size: 14px;
        cursor: pointer;
        transition: color 0.2s;
    }

    .nav-item:hover {
        color: #fff;
    }

    .nav-item.highlight {
        color: #fff;
        font-weight: 500;
    }

    /* Main Content */
    .content {
        position: relative;
        z-index: 10;
        max-width: 1200px;
        margin: 0 auto;
        padding: 80px 24px;
        display: flex;
        flex-direction: column;
        gap: 120px;
    }

    /* Hero */
    .hero-section {
        display: grid;
        grid-template-columns: 1.2fr 1fr;
        gap: 60px;
        align-items: center;
    }

    .hero-text h1 {
        font-size: 64px;
        line-height: 1.1;
        color: #fff;
        margin: 0 0 24px 0;
        letter-spacing: -2px;
        font-weight: 700;
    }

    .gradient-text {
        background: linear-gradient(135deg, #fff 0%, #888 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .hero-sub {
        font-size: 18px;
        line-height: 1.6;
        max-width: 500px;
        margin: 0 0 40px 0;
    }

    .cta-row {
        display: flex;
        gap: 20px;
        align-items: center;
    }

    .btn-primary {
        position: relative;
        background: #fff;
        color: #000;
        border: none;
        padding: 16px 32px;
        font-size: 15px;
        font-weight: 600;
        border-radius: 8px;
        cursor: pointer;
        overflow: hidden;
        transition: transform 0.2s, box-shadow 0.2s;
    }

    .btn-primary.large {
        padding: 20px 40px;
        font-size: 18px;
    }

    .btn-primary:hover {
        transform: translateY(-2px);
        box-shadow: 0 10px 20px rgba(255, 255, 255, 0.1);
    }

    .glow {
        position: absolute;
        top: -50%;
        left: -50%;
        width: 200%;
        height: 200%;
        background: radial-gradient(circle, rgba(255,255,255,0.8) 0%, transparent 60%);
        opacity: 0;
        transform: scale(0.5);
        transition: opacity 0.3s, transform 0.3s;
    }

    .btn-primary:hover .glow {
        opacity: 0.1;
        transform: scale(1);
    }

    .command-copy {
        background: #111;
        border: 1px solid #333;
        padding: 12px 16px;
        border-radius: 8px;
        font-family: 'JetBrains Mono', monospace;
        font-size: 13px;
        color: #aaa;
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .copy-btn {
        background: none;
        border: none;
        color: #666;
        cursor: pointer;
        padding: 0;
        display: flex;
    }

    .copy-btn:hover {
        color: #fff;
    }

    /* Terminal Visual */
    .hero-visual {
        position: relative;
        perspective: 1000px;
    }

    .terminal-window {
        background: #0a0a0a;
        border: 1px solid #333;
        border-radius: 12px;
        box-shadow: 0 40px 80px rgba(0,0,0,0.5);
        overflow: hidden;
        font-family: 'JetBrains Mono', monospace;
        font-size: 13px;
        transform: rotateY(-10deg) rotateX(5deg);
        transition: transform 0.5s ease;
        position: relative;
        z-index: 2;
    }

    .hero-visual:hover .terminal-window {
        transform: rotateY(-5deg) rotateX(2deg) translateY(-10px);
    }

    .visual-glow {
        position: absolute;
        inset: -20px;
        background: radial-gradient(circle at 50% 50%, rgba(0, 255, 136, 0.15), transparent 70%);
        filter: blur(40px);
        z-index: 1;
        transform: translateZ(-50px);
    }

    .terminal-header {
        background: #111;
        padding: 12px 16px;
        border-bottom: 1px solid #222;
        display: flex;
        align-items: center;
    }

    .dots {
        display: flex;
        gap: 6px;
        margin-right: 16px;
    }

    .dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
    }
    .red { background: #ff5f56; }
    .yellow { background: #ffbd2e; }
    .green { background: #27c93f; }

    .terminal-header .title {
        color: #444;
        font-size: 12px;
        flex: 1;
        text-align: center;
    }

    .terminal-body {
        padding: 24px;
        min-height: 280px;
        color: #ccc;
    }

    .line {
        margin-bottom: 8px;
        opacity: 0;
        animation: fadeIn 0.2s forwards;
    }

    .text-muted { color: #666; }
    .text-success { color: #27c93f; }
    .text-accent { color: #00ffaa; }
    .text-white { color: #fff; }

    .prefix {
        color: #444;
        margin-right: 8px;
    }

    .prompt-char {
        color: #ff00ff;
        margin-right: 8px;
    }

    .cursor {
        animation: blink 1s step-end infinite;
    }

    /* Ticker */
    .features-ticker {
        border-top: 1px solid #222;
        border-bottom: 1px solid #222;
        padding: 24px 0;
        display: flex;
        justify-content: center;
        color: #666;
        font-family: 'JetBrains Mono', monospace;
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .ticker-content {
        display: flex;
        gap: 40px;
        align-items: center;
    }

    .ticker-item {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .separator { color: #222; }

    /* Use Cases */
    .section-header {
        margin-bottom: 60px;
        text-align: center;
    }

    .section-header h2 {
        font-size: 36px;
        font-weight: 500;
        color: #666;
        margin: 0;
        letter-spacing: -0.5px;
    }
    
    .section-sub {
        font-size: 16px;
        color: #666;
        margin-top: 12px;
    }

    .text-white { color: #fff; }

    .cards-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
        gap: 24px;
    }

    .feature-card {
        background: #0f0f0f;
        border: 1px solid #222;
        padding: 32px;
        border-radius: 16px;
        text-align: left;
        cursor: pointer;
        transition: all 0.4s cubic-bezier(0.16, 1, 0.3, 1);
        display: flex;
        flex-direction: column;
        gap: 16px;
        position: relative;
        overflow: hidden;
    }

    .feature-card:hover {
        border-color: #333;
        background: #141414;
        transform: translateY(-4px);
        box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
    }

    .card-icon {
        color: #fff;
        background: #1a1a1a;
        width: 48px;
        height: 48px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-bottom: 8px;
        transition: transform 0.4s ease;
    }

    .feature-card:hover .card-icon {
        transform: scale(1.1);
        background: #222;
    }

    .feature-card h3 {
        color: #fff;
        font-size: 18px;
        margin: 0 0 4px 0;
        font-weight: 600;
    }

    .feature-card p {
        color: #666;
        font-size: 14px;
        margin: 0;
        line-height: 1.5;
    }

    .card-arrow {
        margin-top: auto;
        color: #333;
        transform: rotate(180deg);
        align-self: flex-start;
        transition: color 0.2s, transform 0.4s cubic-bezier(0.16, 1, 0.3, 1);
    }

    .feature-card:hover .card-arrow {
        color: #fff;
        transform: rotate(180deg) translateX(6px);
    }

    /* Architecture Section */
    .architecture-section {
        display: flex;
        flex-direction: column;
        align-items: center;
    }

    .arch-diagram {
        display: flex;
        flex-direction: column;
        gap: 0; /* Connected by lines */
        align-items: center;
        width: 100%;
        max-width: 600px;
    }

    .arch-layer {
        width: 100%;
        background: #0f0f0f;
        border: 1px solid #222;
        padding: 24px;
        border-radius: 12px;
        text-align: center;
        position: relative;
        transition: transform 0.4s ease, border-color 0.4s ease, box-shadow 0.4s ease;
        overflow: hidden;
    }

    .arch-layer:hover {
        transform: scale(1.02);
        border-color: #333;
        z-index: 5;
    }

    .arch-layer.highlight {
        border-color: #00ffaa;
        background: rgba(0, 255, 170, 0.03);
        box-shadow: 0 0 30px rgba(0, 255, 170, 0.05);
    }

    .arch-layer.highlight:hover {
        box-shadow: 0 0 40px rgba(0, 255, 170, 0.1);
    }

    .layer-glow {
        position: absolute;
        inset: 0;
        background: radial-gradient(circle at 50% 0%, rgba(255, 255, 255, 0.03), transparent 70%);
        opacity: 0;
        transition: opacity 0.4s;
    }

    .arch-layer:hover .layer-glow {
        opacity: 1;
    }

    .layer-label {
        font-family: 'JetBrains Mono', monospace;
        font-size: 11px;
        color: #666;
        margin-bottom: 8px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .arch-layer.highlight .layer-label {
        color: #00ffaa;
    }

    .layer-content {
        color: #fff;
        font-weight: 500;
        font-size: 16px;
    }

    .arch-connector {
        height: 40px;
        width: 2px;
        background: #222;
        position: relative;
        overflow: hidden;
    }

    .flow-line {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: linear-gradient(to bottom, transparent, #00ffaa, transparent);
        transform: translateY(-100%);
        animation: flow 2s infinite;
    }

    @keyframes flow {
        0% { transform: translateY(-100%); }
        100% { transform: translateY(100%); }
    }

    /* Compare Section */
    .compare-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 40px;
        max-width: 900px;
        margin: 0 auto;
    }

    .compare-card {
        background: #0f0f0f;
        border: 1px solid #222;
        padding: 40px;
        border-radius: 16px;
        position: relative;
        transition: transform 0.4s ease;
    }

    .compare-card.highlight {
        border-color: #00ffaa;
        box-shadow: 0 0 0 1px rgba(0, 255, 170, 0.1);
        overflow: hidden;
    }
    
    .compare-card.highlight:hover {
        transform: scale(1.02);
        box-shadow: 0 20px 50px rgba(0, 255, 170, 0.1);
    }

    .card-glow-bg {
        position: absolute;
        inset: 0;
        background: radial-gradient(circle at 50% 0%, rgba(0, 255, 170, 0.05), transparent 60%);
        z-index: 0;
    }

    .badge {
        position: absolute;
        top: -12px;
        left: 50%;
        transform: translateX(-50%);
        background: #00ffaa;
        color: #000;
        font-size: 11px;
        font-weight: 700;
        padding: 6px 12px;
        border-radius: 20px;
        letter-spacing: 1px;
        z-index: 2;
        box-shadow: 0 4px 12px rgba(0, 255, 170, 0.3);
    }

    .compare-card h3 {
        color: #fff;
        font-size: 20px;
        margin: 0 0 32px 0;
        text-align: center;
        position: relative;
        z-index: 1;
    }

    .stat-row {
        display: flex;
        justify-content: space-between;
        padding: 16px 0;
        border-bottom: 1px solid #222;
        font-size: 15px;
        position: relative;
        z-index: 1;
    }

    .stat-row:last-child {
        border-bottom: none;
    }

    .stat-row .label { color: #666; }
    .stat-row .value { color: #fff; font-family: 'JetBrains Mono', monospace; font-weight: 500; }
    .stat-row .value.slow { color: #ff5f56; }
    .stat-row .value.fast { 
        color: #00ffaa; 
        text-shadow: 0 0 10px rgba(0, 255, 170, 0.3);
    }

    /* Glitch Text Effect */
    .glitch-text {
        position: relative;
    }
    
    .glitch-text::before,
    .glitch-text::after {
        content: attr(data-text);
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: #0f0f0f;
    }
    
    .glitch-text::before {
        left: 2px;
        text-shadow: -1px 0 #ff00c1;
        clip: rect(44px, 450px, 56px, 0);
        animation: glitch-anim-2 5s infinite linear alternate-reverse;
    }

    .glitch-text::after {
        left: -2px;
        text-shadow: -1px 0 #00fff9;
        clip: rect(44px, 450px, 56px, 0);
        animation: glitch-anim-2 5s infinite linear alternate-reverse;
    }

    /* Global Mesh */
    .global-mesh {
        text-align: center;
    }

    .map-visual {
        height: 450px;
        background: #0f0f0f;
        border: 1px solid #222;
        border-radius: 16px;
        position: relative;
        overflow: hidden;
        background-image: 
            linear-gradient(#1a1a1a 1px, transparent 1px),
            linear-gradient(90deg, #1a1a1a 1px, transparent 1px);
        background-size: 60px 60px;
    }

    .mesh-lines {
        position: absolute;
        inset: 0;
        width: 100%;
        height: 100%;
        z-index: 1;
        opacity: 0.3;
    }

    .mesh-path {
        stroke: #00ffaa;
        stroke-width: 1;
        fill: none;
        stroke-dasharray: 10;
        animation: dash 30s linear infinite;
    }

    @keyframes dash {
        to { stroke-dashoffset: 1000; }
    }

    .map-point {
        position: absolute;
        width: 8px;
        height: 8px;
        background: #fff;
        border-radius: 50%;
        z-index: 2;
        transform: translate(-50%, -50%);
    }

    .map-point.main-node {
        background: #00ffaa;
        width: 12px;
        height: 12px;
    }

    .ripple {
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        width: 100%;
        height: 100%;
        border-radius: 50%;
        border: 1px solid #00ffaa;
        animation: ripple 2s infinite cubic-bezier(0, 0.2, 0.8, 1);
        opacity: 0;
    }

    @keyframes ripple {
        0% { width: 0; height: 0; opacity: 0.8; }
        100% { width: 100px; height: 100px; opacity: 0; }
    }

    .scan-bar {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 4px;
        background: linear-gradient(90deg, transparent, #00ffaa, transparent);
        opacity: 0.3;
        animation: scan 4s linear infinite;
        z-index: 3;
    }

    @keyframes scan {
        0% { top: 0; }
        100% { top: 100%; }
    }

    /* Final CTA */
    .final-cta {
        text-align: center;
        padding: 100px 0;
    }

    .final-cta h2 {
        font-size: 56px;
        color: #fff;
        margin: 0 0 40px 0;
        letter-spacing: -2px;
        font-weight: 700;
    }

    /* Footer */
    .main-footer {
        border-top: 1px solid #222;
        padding: 80px 40px 40px 40px;
        background: #0a0a0a;
    }

    .footer-content {
        max-width: 1200px;
        margin: 0 auto;
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 40px;
    }

    .footer-col h4 {
        color: #fff;
        margin: 0 0 24px 0;
        font-size: 14px;
        font-weight: 600;
    }

    .footer-col a {
        display: block;
        color: #666;
        text-decoration: none;
        font-size: 14px;
        margin-bottom: 12px;
        transition: color 0.2s;
    }

    .footer-col a:hover {
        color: #fff;
    }

    .footer-bottom {
        max-width: 1200px;
        margin: 60px auto 0 auto;
        padding-top: 40px;
        border-top: 1px solid #222;
        color: #444;
        font-size: 12px;
        text-align: center;
    }

    @keyframes blink { 50% { opacity: 0; } }
    @keyframes fadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }
    @keyframes slideUp { to { opacity: 1; transform: translateY(0); } }
    @keyframes glitch-anim-2 {
        0% { clip: rect(12px, 9999px, 86px, 0); }
        20% { clip: rect(94px, 9999px, 2px, 0); }
        40% { clip: rect(24px, 9999px, 16px, 0); }
        60% { clip: rect(65px, 9999px, 120px, 0); }
        80% { clip: rect(3px, 9999px, 55px, 0); }
        100% { clip: rect(48px, 9999px, 92px, 0); }
    }

    @media (max-width: 900px) {
        .hero-section {
            grid-template-columns: 1fr;
            text-align: center;
        }
        
        .hero-text h1 { font-size: 42px; }
        .hero-sub { margin: 0 auto 40px auto; }
        .cta-row { justify-content: center; flex-direction: column; }
        
        .features-ticker {
            flex-wrap: wrap;
            justify-content: center;
            gap: 20px;
        }

        .compare-grid {
            grid-template-columns: 1fr;
        }

        .footer-content {
            grid-template-columns: 1fr 1fr;
        }
        
        .final-cta h2 { font-size: 40px; }
    }
</style>