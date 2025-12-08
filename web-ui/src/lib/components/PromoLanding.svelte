<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    import { auth } from "$stores/auth";
    import { toast } from "$stores/toast";

    const dispatch = createEventDispatcher<{
        guest: void;
        navigate: { view: string };
    }>();

    let isOAuthLoading = false;
    let text = "Initialize...";
    let typedText = "";
    let cursorVisible = true;

    onMount(() => {
        let i = 0;
        const typeInterval = setInterval(() => {
            typedText = text.slice(0, i + 1);
            i++;
            if (i > text.length) clearInterval(typeInterval);
        }, 100);

        const cursorInterval = setInterval(() => {
            cursorVisible = !cursorVisible;
        }, 500);

        return () => {
            clearInterval(typeInterval);
            clearInterval(cursorInterval);
        };
    });

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
                toast.error("Connection failed.");
                isOAuthLoading = false;
            }
        } catch (e) {
            toast.error("Connection failed.");
            isOAuthLoading = false;
        }
    }
</script>

<div class="promo-container">
    <div class="grid-overlay"></div>
    <div class="scanline"></div>
    
    <div class="content-wrapper">
        <header class="hero">
            <div class="glitch-wrapper">
                <h1 class="glitch" data-text="REXEC_SYSTEMS">REXEC_SYSTEMS</h1>
            </div>
            <div class="sub-hero">
                <span class="prompt">&gt;</span> 
                <span class="typed">{typedText}</span>
                <span class="cursor" style:opacity={cursorVisible ? 1 : 0}>_</span>
            </div>
            
            <p class="mission-statement">
                INSTANT. SECURE. EPHEMERAL.
                <br>
                THE TERMINAL LAYER FOR THE WEB.
            </p>

            <div class="cta-group">
                <button class="cyber-btn primary" on:click={handleGuestClick}>
                    <span class="btn-content">INIT_GUEST_SESSION</span>
                    <span class="btn-glitch"></span>
                </button>
                <button class="cyber-btn secondary" on:click={handleOAuthLogin} disabled={isOAuthLoading}>
                    <span class="btn-content">{isOAuthLoading ? "CONNECTING..." : "AUTH_PIPEOPS_ID"}</span>
                </button>
            </div>
        </header>

        <section class="features-grid">
            <div class="cyber-card">
                <div class="card-header">
                    <StatusIcon status="bolt" size={16} />
                    <h3>ZERO_LATENCY</h3>
                </div>
                <p>Spin up environments in milliseconds. No cold starts. Pure speed.</p>
            </div>

            <div class="cyber-card">
                <div class="card-header">
                    <StatusIcon status="shield" size={16} />
                    <h3>SECURE_ENCLAVE</h3>
                </div>
                <p>Isolated sandboxes. Ephemeral filesystems. Your data vanishes on exit.</p>
            </div>

            <div class="cyber-card">
                <div class="card-header">
                    <StatusIcon status="connected" size={16} />
                    <h3>GLOBAL_MESH</h3>
                </div>
                <p>Access your workspace from any node on the network. Browser-native SSH.</p>
            </div>
        </section>

        <footer class="system-status">
            <div class="status-line">
                <span>SYSTEM: ONLINE</span>
                <span>VERSION: 2.0.4</span>
                <span>LATENCY: &lt;15ms</span>
            </div>
        </footer>
    </div>
</div>

<style>
    :global(body) {
        background-color: #050505;
    }

    .promo-container {
        position: relative;
        min-height: 100vh;
        width: 100%;
        background-color: #030303;
        color: #e0e0e0;
        font-family: "JetBrains Mono", "Fira Code", monospace;
        overflow: hidden;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
    }

    /* Grid Background */
    .grid-overlay {
        position: absolute;
        inset: 0;
        background-image: 
            linear-gradient(rgba(0, 255, 170, 0.03) 1px, transparent 1px),
            linear-gradient(90deg, rgba(0, 255, 170, 0.03) 1px, transparent 1px);
        background-size: 40px 40px;
        z-index: 1;
        pointer-events: none;
        perspective: 500px;
        transform: scale(1.2);
    }

    /* CRT Scanline */
    .scanline {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: linear-gradient(
            to bottom,
            transparent 50%,
            rgba(0, 0, 0, 0.2) 51%
        );
        background-size: 100% 4px;
        pointer-events: none;
        z-index: 2;
        opacity: 0.6;
    }

    .content-wrapper {
        position: relative;
        z-index: 10;
        max-width: 1000px;
        width: 100%;
        padding: 2rem;
        display: flex;
        flex-direction: column;
        gap: 4rem;
        text-align: center;
    }

    /* Hero Typography */
    .hero {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 1.5rem;
    }

    .glitch-wrapper {
        position: relative;
    }

    .glitch {
        font-size: 5rem;
        font-weight: 900;
        text-transform: uppercase;
        position: relative;
        text-shadow: 2px 2px 0px #ff00ff, -2px -2px 0px #00ffff;
        animation: glitch-anim 2s infinite linear alternate-reverse;
        margin: 0;
        letter-spacing: -2px;
    }
    
    .glitch::before,
    .glitch::after {
        content: attr(data-text);
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
    }

    .glitch::before {
        left: 2px;
        text-shadow: -1px 0 #ff00c1;
        clip: rect(44px, 450px, 56px, 0);
        animation: glitch-anim-2 5s infinite linear alternate-reverse;
    }

    .glitch::after {
        left: -2px;
        text-shadow: -1px 0 #00fff9;
        clip: rect(44px, 450px, 56px, 0);
        animation: glitch-anim-2 5s infinite linear alternate-reverse;
    }

    .sub-hero {
        font-size: 1.5rem;
        color: #00ffaa;
        background: rgba(0, 255, 170, 0.05);
        padding: 0.5rem 1rem;
        border: 1px solid rgba(0, 255, 170, 0.2);
        box-shadow: 0 0 15px rgba(0, 255, 170, 0.1);
    }

    .prompt {
        color: #ff00ff;
        margin-right: 0.5rem;
    }

    .mission-statement {
        font-size: 1.1rem;
        line-height: 1.6;
        color: #888;
        letter-spacing: 2px;
        max-width: 600px;
        border-left: 2px solid #333;
        padding-left: 1rem;
    }

    /* Buttons */
    .cta-group {
        display: flex;
        gap: 2rem;
        margin-top: 1rem;
    }

    .cyber-btn {
        position: relative;
        padding: 1rem 2rem;
        background: transparent;
        border: none;
        cursor: pointer;
        font-family: inherit;
        font-size: 1rem;
        text-transform: uppercase;
        letter-spacing: 2px;
        transition: all 0.2s;
        clip-path: polygon(10px 0, 100% 0, 100% calc(100% - 10px), calc(100% - 10px) 100%, 0 100%, 0 10px);
    }

    .cyber-btn.primary {
        background: #00ffaa;
        color: #000;
        font-weight: 700;
    }

    .cyber-btn.primary:hover {
        background: #ccffee;
        box-shadow: 0 0 20px rgba(0, 255, 170, 0.6);
        transform: translateY(-2px);
    }

    .cyber-btn.secondary {
        background: transparent;
        border: 1px solid #00ffaa;
        color: #00ffaa;
    }

    .cyber-btn.secondary:hover {
        background: rgba(0, 255, 170, 0.1);
        box-shadow: 0 0 15px rgba(0, 255, 170, 0.2);
    }

    /* Features Grid */
    .features-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 2rem;
        width: 100%;
    }

    .cyber-card {
        background: rgba(20, 20, 20, 0.8);
        border: 1px solid #333;
        padding: 2rem;
        text-align: left;
        transition: all 0.3s ease;
        position: relative;
        overflow: hidden;
    }

    .cyber-card::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 2px;
        background: linear-gradient(90deg, transparent, #00ffaa, transparent);
        transform: translateX(-100%);
        transition: transform 0.5s;
    }

    .cyber-card:hover {
        border-color: #00ffaa;
        transform: translateY(-5px);
        box-shadow: 0 5px 20px rgba(0, 0, 0, 0.5);
    }

    .cyber-card:hover::before {
        transform: translateX(100%);
    }

    .card-header {
        display: flex;
        align-items: center;
        gap: 1rem;
        margin-bottom: 1rem;
        color: #00ffaa;
    }

    .card-header h3 {
        font-size: 1.1rem;
        margin: 0;
        font-weight: 400;
    }

    .cyber-card p {
        color: #888;
        font-size: 0.9rem;
        line-height: 1.5;
        margin: 0;
    }

    /* Footer Status */
    .system-status {
        margin-top: auto;
        width: 100%;
        border-top: 1px solid #333;
        padding-top: 2rem;
    }

    .status-line {
        display: flex;
        justify-content: space-between;
        color: #555;
        font-size: 0.8rem;
        text-transform: uppercase;
    }

    /* Animations */
    @keyframes glitch-anim {
        0% { transform: skew(0deg); }
        20% { transform: skew(-2deg); }
        40% { transform: skew(2deg); }
        60% { transform: skew(-1deg); }
        80% { transform: skew(3deg); }
        100% { transform: skew(0deg); }
    }

    @keyframes glitch-anim-2 {
        0% { clip: rect(12px, 9999px, 86px, 0); }
        20% { clip: rect(94px, 9999px, 2px, 0); }
        40% { clip: rect(24px, 9999px, 16px, 0); }
        60% { clip: rect(65px, 9999px, 120px, 0); }
        80% { clip: rect(3px, 9999px, 55px, 0); }
        100% { clip: rect(48px, 9999px, 92px, 0); }
    }

    @media (max-width: 768px) {
        .glitch { font-size: 3rem; }
        .cta-group { flex-direction: column; width: 100%; }
        .cyber-btn { width: 100%; }
        .features-grid { grid-template-columns: 1fr; }
    }
</style>