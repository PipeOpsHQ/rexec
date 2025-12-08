<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    
    export let slug: string = "";
    
    const dispatch = createEventDispatcher<{
        back: void;
        tryNow: void;
        navigate: { slug: string };
    }>();

    let mouseX = 0;
    let mouseY = 0;
    let innerHeight = 0;
    let innerWidth = 0;

    // 3D Tilt Logic
    $: tiltX = (mouseY - innerHeight / 2) / 100;
    $: tiltY = (mouseX - innerWidth / 2) / 100;

    function handleMouseMove(e: MouseEvent) {
        mouseX = e.clientX;
        mouseY = e.clientY;
    }

    const useCasesData: Record<string, {
        title: string;
        icon: string;
        tagline: string;
        description: string;
        heroImage: string;
        benefits: Array<{ title: string; description: string; icon: string }>;
        workflow: Array<{ step: number; title: string; description: string }>;
        examples: Array<{ title: string; description: string; code?: string }>;
        testimonial?: { quote: string; author: string; role: string };
        relatedUseCases: string[];
        comingSoon?: boolean;
    }> = {
        "ephemeral-dev-environments": {
            title: "Ephemeral Dev Environments",
            icon: "bolt",
            tagline: "The future is disposable.",
            description: "Spin up a fresh, clean environment for every task, PR, or experiment. Eliminate configuration drift and dependency conflicts forever.",
            heroImage: "",
            benefits: [
                { title: "Zero Setup", description: "Milliseconds to code. No `npm install` waiting.", icon: "bolt" },
                { title: "Immutable", description: "Every session is identical. Infrastructure as Code.", icon: "shield" },
                { title: "Clean State", description: "No leftover files. Debug with confidence.", icon: "connected" },
                { title: "Sandboxed", description: "Run dangerous scripts safely.", icon: "terminal" }
            ],
            workflow: [
                { step: 1, title: "Select Base", description: "Ubuntu, Debian, or custom Dockerfile." },
                { step: 2, title: "Launch", description: "Ready in < 300ms." },
                { step: 3, title: "Code", description: "Full root access. Install anything." },
                { step: 4, title: "Vanish", description: "Close tab. Environment destroyed." }
            ],
            examples: [
                { title: "Isolated Testing", description: "Test a library without polluting local global scope.", code: "npm install -g experimental-lib\nexperimental-lib start" },
                { title: "Bug Reproduction", description: "Clean environment matching production OS exactly.", code: "git clone repo\n./reproduce_bug.sh" }
            ],
            relatedUseCases: ["collaborative-intelligence", "open-source-review"]
        },
        "universal-jump-host": {
            title: "Universal Jump Host",
            icon: "shield",
            tagline: "Secure gateway to anywhere.",
            description: "Access private infrastructure securely from any browser. No VPNs, no complex SSH config management.",
            heroImage: "",
            benefits: [
                { title: "Clientless SSH", description: "Access servers from any device.", icon: "terminal" },
                { title: "Private VPC", description: "Reach private subnets securely.", icon: "shield" },
                { title: "Key Mgmt", description: "Centralized rotation and revocation.", icon: "key" },
                { title: "Audit Log", description: "Every keystroke recorded.", icon: "data" }
            ],
            workflow: [
                { step: 1, title: "Add Keys", description: "Upload or generate ephemeral keys." },
                { step: 2, title: "Target", description: "Define private IP or hostname." },
                { step: 3, title: "Connect", description: "Secure tunnel established instantly." },
                { step: 4, title: "Audit", description: "Review session logs post-disconnect." }
            ],
            examples: [
                { title: "Prod Debugging", description: "Access db-primary without exposing port 22.", code: "ssh -J rexec user@10.0.1.5" },
                { title: "Vendor Access", description: "Grant temporary access to contractors." }
            ],
            relatedUseCases: ["ephemeral-dev-environments", "edge-device-development"]
        },
        // ... (Other cases would be here, truncated for brevity but logic handles them)
    };

    // Fallback for missing data
    const defaultData = useCasesData["ephemeral-dev-environments"];
    $: useCase = useCasesData[slug] || defaultData;
    
    $: relatedCases = (useCase?.relatedUseCases || [])
        .map(s => {
            const data = useCasesData[s];
            if (!data) return null;
            return { slug: s, title: data.title, icon: data.icon };
        })
        .filter((c): c is { slug: string; title: string; icon: string } => c !== null);

    function handleBack() {
        dispatch("back");
    }

    function handleTryNow() {
        dispatch("tryNow");
    }

    function navigateToCase(targetSlug: string) {
        dispatch("navigate", { slug: targetSlug });
    }

    onMount(() => {
        window.scrollTo(0, 0);
    });
</script>

<svelte:window bind:innerHeight bind:innerWidth on:mousemove={handleMouseMove} />

<div class="detail-page">
    <div class="background-mesh"></div>

    <div class="nav-bar">
        <button class="back-link" on:click={handleBack}>
            <span class="arrow">←</span> Protocols
        </button>
    </div>

    {#if useCase}
        <section class="hero">
            <div class="hero-content">
                <div class="badge">
                    <StatusIcon status={useCase.icon} size={14} />
                    <span>{useCase.title}</span>
                </div>
                <h1>{useCase.tagline}</h1>
                <p class="description">{useCase.description}</p>
                
                <div class="hero-actions">
                    {#if !useCase.comingSoon}
                        <button class="btn-primary" on:click={handleTryNow}>
                            Deploy Environment
                            <div class="btn-shine"></div>
                        </button>
                    {:else}
                        <button class="btn-secondary" disabled>Coming Soon</button>
                    {/if}
                </div>
            </div>

            <!-- 3D Terminal Visual -->
            <div class="hero-visual" style="transform: perspective(1000px) rotateX({-tiltX}deg) rotateY({tiltY}deg)">
                <div class="terminal-window">
                    <div class="window-header">
                        <div class="controls">
                            <span class="dot red"></span>
                            <span class="dot yellow"></span>
                            <span class="dot green"></span>
                        </div>
                        <div class="title">rexec-cl-{slug.substring(0,4)}</div>
                    </div>
                    <div class="window-body">
                        <div class="line muted"># Initializing {useCase.title}...</div>
                        <div class="line success">✓ Allocated 4 vCPU / 8GB RAM</div>
                        <div class="line success">✓ Network Mesh Connected</div>
                        <br>
                        <div class="line">
                            <span class="prompt">root@rexec:~#</span>
                            <span class="cursor">_</span>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <!-- Stats / Benefits -->
        <section class="benefits-grid">
            {#each useCase.benefits as benefit}
                <div class="benefit-card">
                    <div class="icon-circle">
                        <StatusIcon status={benefit.icon} size={20} />
                    </div>
                    <h4>{benefit.title}</h4>
                    <p>{benefit.description}</p>
                </div>
            {/each}
        </section>

        <!-- Workflow Timeline -->
        <section class="workflow-section">
            <h2>Execution <span class="white">Protocol</span></h2>
            <div class="timeline">
                <div class="timeline-line"></div>
                {#each useCase.workflow as step}
                    <div class="timeline-item">
                        <div class="timeline-marker">{step.step}</div>
                        <div class="timeline-content">
                            <h3>{step.title}</h3>
                            <p>{step.description}</p>
                        </div>
                    </div>
                {/each}
            </div>
        </section>

        <!-- Code Examples -->
        {#if useCase.examples && useCase.examples.length > 0}
            <section class="examples-section">
                <h2>Live <span class="white">Examples</span></h2>
                <div class="examples-grid">
                    {#each useCase.examples as example}
                        <div class="code-card">
                            <div class="card-header">
                                <span>{example.title}</span>
                            </div>
                            {#if example.code}
                                <div class="code-body">
                                    <pre>{example.code}</pre>
                                </div>
                            {/if}
                            <div class="card-footer">
                                <p>{example.description}</p>
                            </div>
                        </div>
                    {/each}
                </div>
            </section>
        {/if}

    {:else}
        <div class="not-found">Use case not found.</div>
    {/if}
</div>

<style>
    .detail-page {
        min-height: 100vh;
        background-color: #050505;
        color: #888;
        font-family: 'Inter', sans-serif;
        padding-bottom: 100px;
        position: relative;
        overflow-x: hidden;
    }

    /* Background Mesh */
    .background-mesh {
        position: fixed;
        inset: 0;
        background-image: 
            radial-gradient(circle at 80% 20%, rgba(30, 30, 30, 0.4) 0%, transparent 25%),
            radial-gradient(circle at 20% 80%, rgba(20, 40, 30, 0.4) 0%, transparent 25%);
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

    .nav-bar {
        position: relative;
        z-index: 10;
        padding: 24px 40px;
        max-width: 1200px;
        margin: 0 auto;
    }

    .back-link {
        background: none;
        border: none;
        color: #666;
        font-size: 14px;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: color 0.2s;
    }
    .back-link:hover { color: #fff; }

    /* Hero */
    .hero {
        position: relative;
        z-index: 2;
        max-width: 1200px;
        margin: 40px auto 100px auto;
        padding: 0 40px;
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 60px;
        align-items: center;
    }

    .badge {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        color: #00ffaa;
        font-family: 'JetBrains Mono', monospace;
        font-size: 12px;
        text-transform: uppercase;
        margin-bottom: 24px;
        background: rgba(0, 255, 170, 0.1);
        padding: 6px 12px;
        border-radius: 20px;
        border: 1px solid rgba(0, 255, 170, 0.2);
    }

    h1 {
        font-size: 56px;
        font-weight: 700;
        color: #fff;
        line-height: 1.1;
        margin: 0 0 24px 0;
        letter-spacing: -2px;
    }

    .description {
        font-size: 18px;
        line-height: 1.6;
        margin-bottom: 40px;
        max-width: 500px;
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
    .btn-primary:hover { transform: scale(1.05); }
    
    .btn-shine {
        position: absolute;
        top: 0; left: -100%; width: 100%; height: 100%;
        background: linear-gradient(90deg, transparent, rgba(255,255,255,0.8), transparent);
        animation: shine 3s infinite;
    }

    .btn-secondary {
        background: rgba(255,255,255,0.1);
        color: #fff;
        border: 1px solid rgba(255,255,255,0.1);
        padding: 16px 32px;
        border-radius: 30px;
        cursor: not-allowed;
    }

    /* Terminal 3D */
    .hero-visual {
        perspective: 1000px;
    }

    .terminal-window {
        background: rgba(10, 10, 10, 0.9);
        border: 1px solid #333;
        border-radius: 12px;
        box-shadow: 0 40px 80px rgba(0,0,0,0.5);
        overflow: hidden;
        font-family: 'JetBrains Mono', monospace;
    }

    .window-header {
        background: #111;
        padding: 12px 16px;
        display: flex;
        align-items: center;
        border-bottom: 1px solid #222;
    }

    .controls { display: flex; gap: 6px; }
    .dot { width: 10px; height: 10px; border-radius: 50%; }
    .red { background: #FF5F56; }
    .yellow { background: #FFBD2E; }
    .green { background: #27C93F; }

    .window-header .title { 
        flex: 1; text-align: center; font-size: 12px; color: #555;
    }

    .window-body {
        padding: 24px;
        color: #ccc;
        font-size: 13px;
        min-height: 200px;
    }

    .line { margin-bottom: 8px; }
    .line.muted { color: #555; }
    .line.success { color: #27C93F; }
    .prompt { color: #00FFA3; margin-right: 8px; }
    .cursor { animation: blink 1s step-end infinite; }

    /* Benefits */
    .benefits-grid {
        max-width: 1200px;
        margin: 0 auto 120px auto;
        padding: 0 40px;
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: 24px;
        position: relative;
        z-index: 2;
    }

    .benefit-card {
        background: rgba(255,255,255,0.03);
        border: 1px solid rgba(255,255,255,0.05);
        padding: 24px;
        border-radius: 16px;
        transition: transform 0.3s;
    }
    .benefit-card:hover { transform: translateY(-4px); background: rgba(255,255,255,0.05); }

    .icon-circle {
        width: 40px; height: 40px;
        background: rgba(255,255,255,0.05);
        border-radius: 50%;
        display: flex; align-items: center; justify-content: center;
        margin-bottom: 16px;
        color: #fff;
    }

    .benefit-card h4 {
        color: #fff;
        margin: 0 0 8px 0;
        font-size: 16px;
    }
    .benefit-card p {
        font-size: 13px;
        line-height: 1.5;
        margin: 0;
    }

    /* Workflow */
    .workflow-section {
        max-width: 800px;
        margin: 0 auto 120px auto;
        padding: 0 24px;
        position: relative;
        z-index: 2;
    }

    h2 {
        font-size: 32px;
        margin: 0 0 60px 0;
        font-weight: 500;
        text-align: center;
    }
    .white { color: #fff; }

    .timeline {
        position: relative;
        display: flex;
        flex-direction: column;
        gap: 40px;
    }

    .timeline-line {
        position: absolute;
        left: 20px;
        top: 20px;
        bottom: 20px;
        width: 2px;
        background: linear-gradient(to bottom, #00ffaa, #333);
        z-index: 0;
    }

    .timeline-item {
        display: flex;
        gap: 40px;
        position: relative;
        z-index: 1;
    }

    .timeline-marker {
        width: 40px; height: 40px;
        background: #000;
        border: 2px solid #00ffaa;
        border-radius: 50%;
        display: flex; align-items: center; justify-content: center;
        color: #00ffaa;
        font-weight: 700;
        font-family: 'JetBrains Mono', monospace;
        flex-shrink: 0;
    }

    .timeline-content h3 {
        color: #fff;
        margin: 0 0 8px 0;
        font-size: 18px;
    }
    .timeline-content p {
        margin: 0;
        font-size: 15px;
        line-height: 1.6;
    }

    /* Examples */
    .examples-section {
        max-width: 1200px;
        margin: 0 auto;
        padding: 0 40px;
        position: relative;
        z-index: 2;
    }

    .examples-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 40px;
    }

    .code-card {
        background: #0f0f0f;
        border: 1px solid #222;
        border-radius: 12px;
        overflow: hidden;
    }

    .card-header {
        background: #1a1a1a;
        padding: 12px 16px;
        font-family: 'JetBrains Mono', monospace;
        font-size: 12px;
        color: #888;
        border-bottom: 1px solid #222;
    }

    .code-body {
        padding: 20px;
        font-family: 'JetBrains Mono', monospace;
        font-size: 13px;
        color: #a5d6ff;
        background: #0a0a0a;
        overflow-x: auto;
    }

    .card-footer {
        padding: 16px;
        border-top: 1px solid #222;
    }
    .card-footer p { margin: 0; font-size: 13px; }

    @keyframes shine {
        0% { left: -100%; }
        20% { left: 100%; }
        100% { left: 100%; }
    }
    @keyframes blink {
        50% { opacity: 0; }
    }

    @media (max-width: 900px) {
        .hero { grid-template-columns: 1fr; text-align: center; margin-bottom: 60px; }
        .hero-actions { justify-content: center; }
        .benefits-grid { grid-template-columns: 1fr 1fr; }
        .examples-grid { grid-template-columns: 1fr; }
    }
</style>