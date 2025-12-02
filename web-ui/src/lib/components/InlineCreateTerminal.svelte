<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { containers, type ProgressEvent } from "$stores/containers";
    import { api } from "$utils/api";
    import PlatformIcon from "./icons/PlatformIcon.svelte";

    export let compact = false;

    const dispatch = createEventDispatcher<{
        created: { id: string; name: string };
        cancel: void;
    }>();

    let selectedImage = "";
    let isCreating = false;
    let selectedRole = "standard";
    let progress = 0;
    let progressMessage = "";
    let progressStage = "";

    const progressSteps = [
        { id: "validating", label: "Validating" },
        { id: "pulling", label: "Pulling Image" },
        { id: "creating", label: "Creating Container" },
        { id: "starting", label: "Starting" },
        { id: "configuring", label: "Configuring" },
        { id: "ready", label: "Ready" },
    ];

    function getStepStatus(stepId: string): "pending" | "active" | "completed" {
        const stepOrder = progressSteps.map((s) => s.id);
        const currentIndex = stepOrder.indexOf(progressStage);
        const stepIndex = stepOrder.indexOf(stepId);

        if (stepIndex < currentIndex) return "completed";
        if (stepIndex === currentIndex) return "active";
        return "pending";
    }

    $: displayProgress = Math.round(progress);

    let images: Array<{
        name: string;
        display_name: string;
        description: string;
        category: string;
        popular?: boolean;
    }> = [];

    const roleToOS: Record<string, string> = {
        standard: "alpine",
        node: "ubuntu",
        python: "ubuntu",
        go: "alpine",
        neovim: "arch",
        devops: "alpine",
        overemployed: "alpine",
    };

    $: if (selectedRole && roleToOS[selectedRole]) {
        const preferredOS = roleToOS[selectedRole];
        if (images.length > 0 && images.some((img) => img.name === preferredOS)) {
            selectedImage = preferredOS;
        }
    }

    const roles = [
        {
            id: "standard",
            name: "The Minimalist",
            desc: "I use Arch btw. Just give me a shell.",
            tools: ["bash", "git", "curl", "vim"],
            recommendedOS: "Alpine",
        },
        {
            id: "node",
            name: "10x JS Ninja",
            desc: "Ship fast, break things, npm install everything.",
            tools: ["node", "npm", "yarn", "pnpm", "git"],
            recommendedOS: "Ubuntu",
        },
        {
            id: "python",
            name: "Data Wizard",
            desc: "Import antigravity. I speak in list comprehensions.",
            tools: ["python3", "pip", "jupyter", "pandas", "numpy"],
            recommendedOS: "Ubuntu",
        },
        {
            id: "go",
            name: "The Gopher",
            desc: "If err != nil { panic(err) }. Simplicity is key.",
            tools: ["go", "git", "make", "delve"],
            recommendedOS: "Alpine",
        },
        {
            id: "neovim",
            name: "Neovim God",
            desc: "My config is longer than your code. Mouse? What mouse?",
            tools: ["neovim", "tmux", "fzf", "ripgrep", "lazygit"],
            recommendedOS: "Arch",
        },
        {
            id: "devops",
            name: "YAML Herder",
            desc: "I don't write code, I write config. Prod is my playground.",
            tools: ["kubectl", "docker", "terraform", "helm", "aws-cli"],
            recommendedOS: "Alpine",
        },
        {
            id: "overemployed",
            name: "The Overemployed",
            desc: "Working 4 remote jobs. Need max efficiency.",
            tools: ["tmux", "git", "ssh", "docker", "zsh"],
            recommendedOS: "Alpine",
        },
    ];

    $: currentRole = roles.find((r) => r.id === selectedRole);

    async function loadImages() {
        if (images.length > 0) return;

        const { data, error } = await api.get<{
            images?: typeof images;
            popular?: typeof images;
        }>("/api/images?all=true");

        if (data) {
            images = data.images || data.popular || [];
        }
    }

    onMount(() => {
        loadImages();
    });

    function selectAndCreate(imageName: string) {
        selectedImage = imageName;
        createContainer();
    }

    function createContainer() {
        if (!selectedImage || isCreating) return;

        isCreating = true;
        progress = 0;
        progressMessage = "Starting...";
        progressStage = "validating";

        function handleProgress(event: ProgressEvent) {
            progress = event.progress;
            progressMessage = event.message;
            progressStage = event.stage;
        }

        function handleComplete(container: any) {
            // Reset UI and dispatch event
            isCreating = false;
            progress = 0;
            progressMessage = "";
            progressStage = "";
            dispatch("created", { id: container.id, name: container.name });
        }

        function handleError(error: string) {
            progressMessage = error || "Failed to create terminal";
            // Keep showing error for a moment, then reset
            setTimeout(() => {
                isCreating = false;
                progress = 0;
                progressMessage = "";
                progressStage = "";
            }, 3000);
        }

        // Generate a unique name
        const terminalName = `terminal-${Date.now().toString(36)}`;
        
        // Note: createContainerWithProgress is fire-and-forget with callbacks
        containers.createContainerWithProgress(
            terminalName,
            selectedImage,
            undefined, // customImage
            selectedRole,
            handleProgress,
            handleComplete,
            handleError
        );
    }
</script>

<div class="inline-create" class:compact>
    {#if isCreating}
        <div class="create-progress">
            <div class="progress-header">
                <span class="progress-percent">{displayProgress}%</span>
            </div>
            <div class="progress-bar">
                <div class="progress-fill" style="width: {displayProgress}%"></div>
            </div>
            
            <!-- Step Indicators -->
            <div class="progress-steps">
                {#each progressSteps as step}
                    <div class="progress-step {getStepStatus(step.id)}">
                        <span class="step-dot"></span>
                        <span class="step-label">{step.label}</span>
                    </div>
                {/each}
            </div>
            
            <p class="progress-message">{progressMessage}</p>
            {#if currentRole && progressStage === "configuring"}
                <div class="installing-tools">
                    <p class="installing-label">Installing {currentRole.name} tools:</p>
                    <div class="tools-installing">
                        {#each currentRole.tools as tool}
                            <span class="tool-badge-installing">{tool}</span>
                        {/each}
                    </div>
                </div>
            {/if}
            <div class="spinner"></div>
        </div>
    {:else}
        <div class="create-content">
            <!-- Header -->
            <div class="create-hero">
                <div class="hero-icon">
                    <span class="terminal-prompt">$</span>
                </div>
                <h2 class="hero-title">Spin up a new terminal</h2>
                <p class="hero-subtitle">Choose your environment, pick an OS, and you're in.</p>
            </div>

            <!-- Role Selection -->
            <div class="create-section">
                <div class="section-header">
                    <span class="section-number">01</span>
                    <h4>Pick your stack</h4>
                </div>
                <div class="role-grid">
                    {#each roles as role}
                        <button
                            class="role-card"
                            class:selected={selectedRole === role.id}
                            on:click={() => (selectedRole = role.id)}
                        >
                            <div class="role-icon-wrap">
                                <PlatformIcon platform={role.id} size={28} />
                            </div>
                            <div class="role-content">
                                <span class="role-name">{role.name}</span>
                                <span class="role-desc">{role.desc}</span>
                            </div>
                            {#if selectedRole === role.id}
                                <span class="role-check">✓</span>
                            {/if}
                        </button>
                    {/each}
                </div>
                {#if currentRole}
                    <div class="role-preview">
                        <div class="preview-label">Pre-installed tools:</div>
                        <div class="role-tools">
                            {#each currentRole.tools as tool}
                                <span class="tool-badge">{tool}</span>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>

            <!-- OS Selection -->
            <div class="create-section">
                <div class="section-header">
                    <span class="section-number">02</span>
                    <h4>Select OS & Launch</h4>
                </div>
                <div class="os-grid">
                    {#each images as image (image.name)}
                        <button
                            class="os-card"
                            on:click={() => selectAndCreate(image.name)}
                        >
                            <div class="os-icon-wrap">
                                <PlatformIcon platform={image.name} size={32} />
                            </div>
                            <span class="os-name">{image.display_name || image.name}</span>
                            {#if image.popular}
                                <span class="popular-badge">★</span>
                            {/if}
                            <span class="launch-arrow">→</span>
                        </button>
                    {/each}
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .inline-create {
        padding: 20px;
        height: 100%;
        overflow-y: auto;
        background: #0a0a0a;
        display: flex;
        flex-direction: column;
    }

    .inline-create.compact {
        padding: 16px;
    }

    /* Progress */
    .create-progress {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 16px;
        padding: 40px 24px;
        text-align: center;
        flex: 1;
        min-height: 300px;
    }

    .progress-header {
        font-size: 32px;
        font-weight: 600;
        color: var(--accent);
        font-family: var(--font-mono);
    }

    .progress-bar {
        width: 100%;
        max-width: 400px;
        height: 6px;
        background: var(--bg-tertiary);
        border-radius: 3px;
        overflow: hidden;
    }

    .progress-fill {
        height: 100%;
        background: var(--accent);
        transition: width 0.3s ease;
    }

    /* Progress Steps */
    .progress-steps {
        display: flex;
        gap: 8px;
        flex-wrap: wrap;
        justify-content: center;
        max-width: 500px;
        margin: 8px 0;
    }

    .progress-step {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        background: rgba(255, 255, 255, 0.03);
        border-radius: 4px;
        font-size: 11px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    .step-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: var(--border);
        transition: all 0.2s ease;
    }

    .progress-step.pending .step-dot {
        background: var(--border);
    }

    .progress-step.active .step-dot {
        background: var(--accent);
        box-shadow: 0 0 8px var(--accent);
        animation: pulse 1s infinite;
    }

    .progress-step.active {
        color: var(--accent);
    }

    .progress-step.completed .step-dot {
        background: var(--accent);
    }

    .progress-step.completed {
        color: var(--text);
    }

    .step-label {
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .progress-message {
        color: var(--text-muted);
        font-size: 14px;
        margin: 0;
    }

    .installing-tools {
        margin-top: 12px;
    }

    .installing-label {
        font-size: 12px;
        color: var(--text-muted);
        margin-bottom: 8px;
    }

    .tools-installing {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        justify-content: center;
    }

    .tool-badge-installing {
        padding: 4px 8px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.3);
        border-radius: 4px;
        font-size: 11px;
        color: var(--accent);
        font-family: var(--font-mono);
        animation: pulse 1s infinite;
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.5; }
    }

    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    /* Content */
    .create-content {
        display: flex;
        flex-direction: column;
        gap: 32px;
        flex: 1;
        max-width: 100%;
    }

    /* Hero Section */
    .create-hero {
        text-align: center;
        padding: 24px 0 16px;
    }

    .hero-icon {
        width: 64px;
        height: 64px;
        margin: 0 auto 16px;
        background: linear-gradient(135deg, rgba(0, 255, 65, 0.15), rgba(0, 255, 65, 0.05));
        border: 2px solid var(--accent);
        border-radius: 16px;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 0 0 30px rgba(0, 255, 65, 0.2);
    }

    .terminal-prompt {
        font-size: 32px;
        font-weight: bold;
        color: var(--accent);
        font-family: var(--font-mono);
        animation: blink 1s step-end infinite;
    }

    @keyframes blink {
        50% { opacity: 0.5; }
    }

    .hero-title {
        margin: 0 0 8px;
        font-size: 24px;
        font-weight: 700;
        color: var(--text);
    }

    .hero-subtitle {
        margin: 0;
        font-size: 14px;
        color: var(--text-muted);
    }

    /* Section Headers */
    .section-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 16px;
    }

    .section-number {
        font-size: 11px;
        font-weight: 700;
        color: var(--accent);
        font-family: var(--font-mono);
        padding: 4px 8px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.3);
        border-radius: 4px;
    }

    .create-section h4 {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    /* Role Grid */
    .role-grid {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .role-card {
        display: flex;
        align-items: center;
        gap: 14px;
        padding: 14px 16px;
        background: rgba(255, 255, 255, 0.02);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 10px;
        cursor: pointer;
        transition: all 0.2s ease;
        text-align: left;
    }

    .role-card:hover {
        border-color: rgba(0, 255, 65, 0.4);
        background: rgba(0, 255, 65, 0.03);
    }

    .role-card.selected {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.08);
        box-shadow: 0 0 20px rgba(0, 255, 65, 0.15), inset 0 0 20px rgba(0, 255, 65, 0.03);
    }

    .role-icon-wrap {
        width: 44px;
        height: 44px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 0, 0, 0.3);
        border-radius: 10px;
        flex-shrink: 0;
    }

    .role-content {
        flex: 1;
        min-width: 0;
    }

    .role-name {
        display: block;
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        margin-bottom: 2px;
    }

    .role-desc {
        display: block;
        font-size: 12px;
        color: var(--text-muted);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .role-check {
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--accent);
        color: #000;
        border-radius: 50%;
        font-size: 12px;
        font-weight: bold;
        flex-shrink: 0;
    }

    /* Role Preview */
    .role-preview {
        margin-top: 12px;
        padding: 12px 14px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(255, 255, 255, 0.06);
        border-radius: 8px;
    }

    .preview-label {
        font-size: 11px;
        color: var(--text-muted);
        text-transform: uppercase;
        letter-spacing: 0.5px;
        margin-bottom: 8px;
    }

    .role-tools {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
    }

    .tool-badge {
        padding: 4px 10px;
        background: rgba(0, 255, 65, 0.08);
        border: 1px solid rgba(0, 255, 65, 0.2);
        border-radius: 4px;
        font-size: 11px;
        color: var(--accent);
        font-family: var(--font-mono);
    }

    /* OS Grid */
    .os-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
        gap: 10px;
    }

    .os-card {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 14px 16px;
        background: rgba(255, 255, 255, 0.02);
        border: 1px solid rgba(255, 255, 255, 0.08);
        border-radius: 10px;
        cursor: pointer;
        transition: all 0.2s ease;
        position: relative;
    }

    .os-card:hover {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.08);
        transform: translateY(-2px);
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3), 0 0 20px rgba(0, 255, 65, 0.1);
    }

    .os-card:hover .launch-arrow {
        opacity: 1;
        transform: translateX(0);
    }

    .os-icon-wrap {
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 0, 0, 0.3);
        border-radius: 8px;
        flex-shrink: 0;
    }

    .os-name {
        flex: 1;
        font-size: 13px;
        font-weight: 500;
        color: var(--text);
        text-align: left;
    }

    .popular-badge {
        color: #ffd700;
        font-size: 14px;
    }

    .launch-arrow {
        font-size: 16px;
        color: var(--accent);
        opacity: 0;
        transform: translateX(-4px);
        transition: all 0.2s ease;
    }

    /* Compact mode adjustments */
    .compact .create-hero {
        padding: 12px 0 8px;
    }

    .compact .hero-icon {
        width: 48px;
        height: 48px;
    }

    .compact .terminal-prompt {
        font-size: 24px;
    }

    .compact .hero-title {
        font-size: 18px;
    }

    .compact .role-card {
        padding: 10px 12px;
    }

    .compact .role-icon-wrap {
        width: 36px;
        height: 36px;
    }

    .compact .os-grid {
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    }

    .compact .os-card {
        padding: 10px 12px;
    }

    .compact .os-icon-wrap {
        width: 32px;
        height: 32px;
    }
</style>
