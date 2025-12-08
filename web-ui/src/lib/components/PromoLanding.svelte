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

        <section class="use-cases">
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
                        </div> <!-- Rotated in CSS -->
                    </button>
                {/each}
            </div>
        </section>
    </main>
</div>

<style>
    :global(body) {
        background-color: #050505;
        margin: 0;
        font-family: 'Inter', system-ui, -apple-system, sans-serif;
    }

    .promo-page {
        min-height: 100vh;
        color: #888;
        position: relative;
        overflow-x: hidden;
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
        transition: transform 0.2s;
    }

    .btn-primary:hover {
        transform: translateY(-2px);
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
        gap: 40px;
        color: #666;
        font-family: 'JetBrains Mono', monospace;
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .ticker-item {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .separator { color: #222; }

    /* Use Cases */
    .section-header {
        margin-bottom: 40px;
    }

    .section-header h2 {
        font-size: 24px;
        font-weight: 400;
        color: #666;
        margin: 0;
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
        padding: 24px;
        border-radius: 12px;
        text-align: left;
        cursor: pointer;
        transition: all 0.3s ease;
        display: flex;
        flex-direction: column;
        gap: 16px;
        animation: slideUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
        opacity: 0;
        transform: translateY(20px);
        animation-delay: var(--delay);
    }

    .feature-card:hover {
        border-color: #444;
        background: #141414;
        transform: translateY(-4px);
    }

    .card-icon {
        color: #fff;
        background: #1a1a1a;
        width: 48px;
        height: 48px;
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-bottom: 8px;
    }

    .feature-card h3 {
        color: #fff;
        font-size: 16px;
        margin: 0 0 4px 0;
        font-weight: 600;
    }

    .feature-card p {
        color: #666;
        font-size: 13px;
        margin: 0;
        line-height: 1.5;
    }

    .card-arrow {
        margin-top: auto;
        color: #333;
        transform: rotate(180deg);
        align-self: flex-start;
        transition: color 0.2s, transform 0.2s;
    }

    .feature-card:hover .card-arrow {
        color: #fff;
        transform: rotate(180deg) translateX(4px);
    }

    @keyframes blink { 50% { opacity: 0; } }
    @keyframes fadeIn { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }
    @keyframes slideUp { to { opacity: 1; transform: translateY(0); } }

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
    }
</style>