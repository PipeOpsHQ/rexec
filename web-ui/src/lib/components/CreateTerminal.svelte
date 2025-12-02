<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { containers, type ProgressEvent } from "$stores/containers";
    import { toast } from "$stores/toast";
    import { api } from "$utils/api";

    const dispatch = createEventDispatcher<{
        cancel: void;
        created: { id: string; name: string };
    }>();

    // State
    let containerName = "";
    let selectedImage = "";
    let selectedRole = "standard";
    let customImage = "";
    let isCreating = false;
    let progress = 0;
    let progressMessage = "";
    let progressStage = "";

    // Progress steps for visual display
    const progressSteps = [
        { id: "validating", label: "Validating", icon: "✓" },
        { id: "pulling", label: "Pulling Image", icon: "📦" },
        { id: "creating", label: "Creating Container", icon: "🔧" },
        { id: "starting", label: "Starting", icon: "🚀" },
        { id: "configuring", label: "Configuring", icon: "⚙️" },
        { id: "ready", label: "Ready", icon: "✨" },
    ];

    // Get step status
    function getStepStatus(stepId: string): "pending" | "active" | "completed" {
        const stepOrder = progressSteps.map((s) => s.id);
        const currentIndex = stepOrder.indexOf(progressStage);
        const stepIndex = stepOrder.indexOf(stepId);

        if (stepIndex < currentIndex) return "completed";
        if (stepIndex === currentIndex) return "active";
        return "pending";
    }

    // Round progress to integer
    $: displayProgress = Math.round(progress);

    // Available images
    let images: Array<{
        name: string;
        display_name: string;
        description: string;
        category: string;
        popular?: boolean;
    }> = [];

    // Role to preferred OS mapping
    const roleToOS: Record<string, string> = {
        standard: "alpine", // Minimalist loves lightweight
        node: "ubuntu", // Best Node.js support
        python: "ubuntu", // Best Python/data science support
        go: "alpine", // Go's preferred container OS
        neovim: "arch", // Power users love Arch
        devops: "alpine", // Container standard
        overemployed: "alpine", // Fast startup
    };

    // Auto-select OS when role changes
    $: if (selectedRole && roleToOS[selectedRole]) {
        const preferredOS = roleToOS[selectedRole];
        // Only auto-select if images are loaded and the preferred OS exists
        if (
            images.length > 0 &&
            images.some((img) => img.name === preferredOS)
        ) {
            selectedImage = preferredOS;
        }
    }

    // Available roles with detailed descriptions
    const roles = [
        {
            id: "standard",
            name: "The Minimalist",
            icon: "🧘",
            desc: "I use Arch btw. Just give me a shell.",
            tools: ["bash", "git", "curl", "vim"],
            recommendedOS: "Alpine",
            useCase: "Quick tasks, scripting, and basic development",
        },
        {
            id: "node",
            name: "10x JS Ninja",
            icon: "🚀",
            desc: "Ship fast, break things, npm install everything.",
            tools: ["node", "npm", "yarn", "pnpm", "git"],
            recommendedOS: "Ubuntu",
            useCase: "Full-stack JavaScript/TypeScript development",
        },
        {
            id: "python",
            name: "Data Wizard",
            icon: "🧙‍♂️",
            desc: "Import antigravity. I speak in list comprehensions.",
            tools: ["python3", "pip", "jupyter", "pandas", "numpy"],
            recommendedOS: "Ubuntu",
            useCase: "Data science, ML, and Python development",
        },
        {
            id: "go",
            name: "The Gopher",
            icon: "🐹",
            desc: "If err != nil { panic(err) }. Simplicity is key.",
            tools: ["go", "git", "make", "delve"],
            recommendedOS: "Alpine",
            useCase: "Go development, CLI tools, and microservices",
        },
        {
            id: "neovim",
            name: "Neovim God",
            icon: "⌨️",
            desc: "My config is longer than your code. Mouse? What mouse?",
            tools: ["neovim", "tmux", "fzf", "ripgrep", "lazygit"],
            recommendedOS: "Arch",
            useCase: "Terminal-first development with powerful editing",
        },
        {
            id: "devops",
            name: "YAML Herder",
            icon: "☸️",
            desc: "I don't write code, I write config. Prod is my playground.",
            tools: ["kubectl", "docker", "terraform", "helm", "aws-cli"],
            recommendedOS: "Alpine",
            useCase: "Infrastructure, containers, and cloud operations",
        },
        {
            id: "overemployed",
            name: "The Overemployed",
            icon: "💼",
            desc: "Working 4 remote jobs. Need max efficiency.",
            tools: ["tmux", "git", "ssh", "docker", "zsh"],
            recommendedOS: "Alpine",
            useCase: "Maximum productivity with minimal overhead",
        },
    ];

    // Get current selected role details
    $: currentRole = roles.find((r) => r.id === selectedRole);

    // Image icons
    const imageIcons: Record<string, string> = {
        ubuntu: "🟠",
        debian: "🔴",
        alpine: "🔵",
        fedora: "🔵",
        centos: "🟣",
        rocky: "🟢",
        alma: "🟣",
        arch: "🔷",
        kali: "🐉",
        parrot: "🦜",
        mint: "🌿",
        elementary: "🪶",
        devuan: "🔘",
        blackarch: "🖤",
        manjaro: "🟩",
        opensuse: "🦎",
        tumbleweed: "🌀",
        gentoo: "🗿",
        void: "⬛",
        nixos: "❄️",
        slackware: "📦",
        busybox: "📦",
        amazonlinux: "🟧",
        oracle: "🔶",
        rhel: "🎩",
        openeuler: "🔵",
        clearlinux: "💎",
        photon: "☀️",
        raspberrypi: "🍓",
        scientific: "🔬",
        rancheros: "🐄",
        custom: "📦",
    };

    function getIcon(imageName: string): string {
        const lower = imageName.toLowerCase();
        for (const [key, icon] of Object.entries(imageIcons)) {
            if (lower.includes(key)) return icon;
        }
        return "🐧";
    }

    // Load available images
    onMount(async () => {
        const { data, error } = await api.get<{
            images?: typeof images;
            popular?: typeof images;
        }>("/api/images?all=true");

        if (data) {
            images = data.images || data.popular || [];
        } else if (error) {
            toast.error("Failed to load images");
        }
    });

    // Generate random name
    function generateName(): string {
        const adjectives = [
            "swift",
            "bold",
            "calm",
            "dark",
            "eager",
            "fast",
            "grand",
            "happy",
            "keen",
            "light",
            "merry",
            "noble",
            "proud",
            "quick",
            "rare",
            "sharp",
        ];
        const nouns = [
            "ant",
            "bear",
            "cat",
            "fox",
            "hawk",
            "lion",
            "owl",
            "wolf",
            "tiger",
            "eagle",
            "shark",
            "cobra",
            "raven",
            "viper",
            "lynx",
            "orca",
        ];
        const adj = adjectives[Math.floor(Math.random() * adjectives.length)];
        const noun = nouns[Math.floor(Math.random() * nouns.length)];
        const num = Math.floor(Math.random() * 1000);
        return `${adj}-${noun}-${num}`;
    }

    // Handle image selection and creation
    async function selectAndCreate(imageName: string) {
        if (isCreating) return;

        selectedImage = imageName;

        // For custom image, prompt for input
        if (imageName === "custom") {
            const input = prompt(
                "Enter Docker image name (e.g., nginx:latest):",
            );
            if (!input || !input.trim()) {
                selectedImage = "";
                return;
            }
            customImage = input.trim();
        }

        // Start creation - set state first for immediate UI feedback
        isCreating = true;
        progress = 0;
        progressMessage = "Starting...";
        progressStage = "initializing";

        // Defer heavy work to next tick to allow UI to update
        await new Promise((resolve) => setTimeout(resolve, 0));

        const name = containerName.trim() || generateName();
        const image = selectedImage;
        const custom = selectedImage === "custom" ? customImage : undefined;

        containers.createContainerWithProgress(
            name,
            image,
            custom,
            selectedRole,
            // onProgress
            (event: ProgressEvent) => {
                progress = event.progress || 0;
                progressMessage = event.message || "";
                progressStage = event.stage || "";
            },
            // onComplete
            (container) => {
                const containerName = container?.name || name;
                const containerId = container?.id || container?.db_id;

                if (!containerId) {
                    isCreating = false;
                    toast.error(
                        "Container created but ID not found. Please refresh.",
                    );
                    return;
                }

                // Update progress to show completion
                progress = 100;
                progressMessage = "Terminal ready! Opening...";
                progressStage = "ready";

                toast.success(`Terminal "${containerName}" created!`);

                // Delay before dispatching to ensure container is ready
                // Keep isCreating true until we navigate away
                setTimeout(() => {
                    dispatch("created", {
                        id: containerId,
                        name: containerName,
                    });
                    // isCreating will be reset when component is destroyed on navigation
                }, 800);
            },
            // onError
            (error) => {
                isCreating = false;
                let errorMsg = "Failed to create terminal";
                if (typeof error === "string" && error.trim()) {
                    errorMsg = error;
                } else if (error && typeof error === "object") {
                    errorMsg =
                        error.message || error.error || JSON.stringify(error);
                }
                if (!errorMsg || errorMsg === "undefined") {
                    errorMsg = "Failed to create terminal. Please try again.";
                }
                toast.error(errorMsg);
            },
        );
    }

    function handleCancel() {
        if (!isCreating) {
            dispatch("cancel");
        }
    }
</script>

<div class="create-container">
    <div class="create-header">
        <button class="back-btn" on:click={handleCancel} disabled={isCreating}>
            ← Back
        </button>
        <h1>Create Terminal</h1>
    </div>

    {#if isCreating}
        <div class="progress-section">
            <div class="progress-header">
                <h2>Creating Terminal</h2>
                <span class="progress-percent">{displayProgress}%</span>
            </div>

            <div class="progress-bar">
                <div
                    class="progress-fill"
                    style="width: {displayProgress}%"
                ></div>
            </div>

            <!-- Step Indicators -->
            <div class="progress-steps">
                {#each progressSteps as step}
                    <div class="progress-step {getStepStatus(step.id)}">
                        <span class="step-icon">{step.icon}</span>
                        <span class="step-label">{step.label}</span>
                    </div>
                {/each}
            </div>

            <div class="progress-info">
                <span class="progress-message">{progressMessage}</span>
            </div>

            <!-- Role-specific info -->
            {#if currentRole && progressStage === "configuring"}
                <div class="installing-tools">
                    <p class="installing-label">
                        Installing tools for {currentRole.name}:
                    </p>
                    <div class="tools-being-installed">
                        {#each currentRole.tools as tool}
                            <span class="tool-installing">{tool}</span>
                        {/each}
                    </div>
                </div>
            {/if}

            <div class="progress-spinner">
                <div class="spinner-large"></div>
            </div>
        </div>
    {:else}
        <div class="create-form">
            <!-- Terminal Name (Optional) -->
            <div class="form-group name-group">
                <label for="container-name">
                    Terminal Name
                    <span class="optional"
                        >(optional - auto-generated if empty)</span
                    >
                </label>
                <input
                    type="text"
                    id="container-name"
                    bind:value={containerName}
                    placeholder="e.g., my-dev-box"
                    maxlength="64"
                />
            </div>

            <!-- Step 1: Environment/Role Selection -->
            <div class="form-section">
                <div class="section-header">
                    <span class="step-number">1</span>
                    <h2>Choose Your Environment</h2>
                </div>
                <p class="section-desc">What kind of developer are you?</p>

                <div class="role-grid">
                    {#each roles as role}
                        <button
                            type="button"
                            class="role-card"
                            class:selected={selectedRole === role.id}
                            on:click={() => (selectedRole = role.id)}
                            title={role.desc}
                        >
                            <span class="role-icon">{role.icon}</span>
                            <span class="role-name">{role.name}</span>
                        </button>
                    {/each}
                </div>

                <!-- Compact Hero Stat Display -->
                {#if currentRole}
                    <div class="role-info-compact">
                        <div class="role-header-row">
                            <span class="role-icon-lg">{currentRole.icon}</span>
                            <div class="role-name-quote">
                                <span class="role-title">{currentRole.name}</span>
                                <span class="role-quote">"{currentRole.desc}"</span>
                            </div>
                            <span class="role-os-badge">📦 {currentRole.recommendedOS}</span>
                        </div>
                        <div class="role-tools">
                            {#each currentRole.tools as tool}
                                <span class="tool-badge">{tool}</span>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>

            <!-- Step 2: OS Selection -->
            <div class="form-section">
                <div class="section-header">
                    <span class="step-number">2</span>
                    <h2>Select Operating System</h2>
                </div>
                <p class="section-desc">Click to create your terminal</p>

                <div class="image-grid">
                    {#each images as image (image.name)}
                        <button
                            type="button"
                            class="image-card"
                            on:click={() => selectAndCreate(image.name)}
                        >
                            <span class="image-icon">{getIcon(image.name)}</span
                            >
                            <span class="image-name"
                                >{image.display_name || image.name}</span
                            >
                            {#if image.popular}
                                <span class="popular-badge">Popular</span>
                            {/if}
                        </button>
                    {/each}

                    <!-- Custom Image Option -->
                    <button
                        type="button"
                        class="image-card custom-card"
                        on:click={() => selectAndCreate("custom")}
                    >
                        <span class="image-icon">📦</span>
                        <span class="image-name">Custom Image</span>
                    </button>
                </div>
            </div>

            <!-- Cancel Action -->
            <div class="form-actions">
                <button
                    type="button"
                    class="btn btn-secondary"
                    on:click={handleCancel}
                >
                    Cancel
                </button>
            </div>
        </div>
    {/if}
</div>

<style>
    .create-container {
        max-width: 900px;
        margin: 0 auto;
        animation: fadeIn 0.2s ease;
    }

    .create-header {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 32px;
        padding-bottom: 16px;
        border-bottom: 1px solid var(--border);
    }

    .back-btn {
        background: none;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        padding: 6px 12px;
        font-family: var(--font-mono);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.2s;
    }

    .back-btn:hover:not(:disabled) {
        border-color: var(--text);
        color: var(--text);
    }

    .back-btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .create-header h1 {
        font-size: 20px;
        text-transform: uppercase;
        letter-spacing: 1px;
        margin: 0;
    }

    /* Form Styles */
    .create-form {
        display: flex;
        flex-direction: column;
        gap: 32px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .form-group label {
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-secondary);
    }

    .optional {
        color: var(--text-muted);
        text-transform: none;
        font-size: 11px;
    }

    .form-group input {
        width: 100%;
        max-width: 400px;
    }

    .name-group {
        padding-bottom: 16px;
        border-bottom: 1px solid var(--border);
    }

    /* Section Styles */
    .form-section {
        display: flex;
        flex-direction: column;
        gap: 16px;
        overflow: visible;
    }

    .section-header {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .step-number {
        width: 28px;
        height: 28px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--accent);
        color: var(--bg);
        font-size: 14px;
        font-weight: bold;
        font-family: var(--font-mono);
    }

    .section-header h2 {
        font-size: 16px;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        margin: 0;
        color: var(--text);
    }

    .section-desc {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0;
        font-family: var(--font-mono);
    }

    /* Role Grid */
    .role-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
        gap: 12px;
    }

    .role-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 16px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        cursor: pointer;
        transition: all 0.2s;
        font-family: var(--font-mono);
    }

    .role-card:hover {
        border-color: var(--accent);
        background: var(--accent-dim);
    }

    .role-card.selected {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
        box-shadow: 0 0 10px rgba(0, 255, 65, 0.2);
    }

    .role-icon {
        font-size: 28px;
    }

    .role-name {
        font-size: 11px;
        color: var(--text);
        text-align: center;
    }

    /* Role Info Compact Display */
    .role-info-compact {
        margin-top: 12px;
        padding: 12px;
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: 8px;
        border-left: 3px solid var(--accent);
    }

    .role-header-row {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 10px;
    }

    .role-icon-lg {
        font-size: 28px;
        filter: drop-shadow(0 0 4px var(--accent));
    }

    .role-name-quote {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 2px;
        min-width: 0;
    }

    .role-title {
        font-size: 13px;
        font-weight: 600;
        color: var(--text);
        font-family: var(--font-mono);
    }

    .role-quote {
        font-size: 10px;
        color: var(--accent);
        font-style: italic;
        font-family: var(--font-mono);
        opacity: 0.9;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .role-os-badge {
        font-size: 11px;
        color: var(--text-muted);
        font-family: var(--font-mono);
        padding: 4px 8px;
        background: var(--bg-tertiary);
        border-radius: 4px;
    }

    .role-tools {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
    }

    .tool-badge {
        padding: 4px 10px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.3);
        border-radius: 4px;
        font-size: 11px;
        color: var(--accent);
        font-family: var(--font-mono);
    }

    .image-grid {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.6);
        z-index: 9998;
    }

    .popover-close {
        position: absolute;
        top: 8px;
        right: 8px;
        background: none;
        border: none;
        color: var(--text-muted);
        font-size: 14px;
        cursor: pointer;
        padding: 4px 8px;
        line-height: 1;
        transition: color 0.15s;
    }

    .popover-close:hover {
        color: var(--text);
    }

    @keyframes popoverSlide {
        from {
            opacity: 0;
            transform: translateY(-8px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    .stat-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;
        padding-bottom: 8px;
        border-bottom: 1px solid var(--border);
    }

    .stat-class {
        font-size: 12px;
        font-weight: 600;
        color: var(--accent);
        font-family: var(--font-mono);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .stat-level {
        font-size: 10px;
        color: var(--warning, #ffd93d);
        font-family: var(--font-mono);
        padding: 2px 6px;
        background: rgba(255, 217, 61, 0.1);
        border-radius: 3px;
    }

    .stat-bars {
        display: flex;
        flex-direction: column;
        gap: 6px;
        margin-bottom: 10px;
    }

    .stat-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .stat-label {
        font-size: 10px;
        color: var(--text-muted);
        font-family: var(--font-mono);
        width: 60px;
        flex-shrink: 0;
    }

    .stat-bar-track {
        flex: 1;
        height: 6px;
        background: var(--bg-tertiary);
        border-radius: 3px;
        overflow: hidden;
    }

    .stat-bar-fill {
        height: 100%;
        background: linear-gradient(90deg, var(--accent) 0%, #00ff88 100%);
        border-radius: 3px;
        transition: width 0.3s ease;
        box-shadow: 0 0 6px var(--accent);
    }

    .stat-bar-fill.defense {
        background: linear-gradient(90deg, #00d9ff 0%, #0099ff 100%);
        box-shadow: 0 0 6px #00d9ff;
    }

    .stat-bar-fill.speed {
        background: linear-gradient(90deg, #ff6b6b 0%, #ffd93d 100%);
        box-shadow: 0 0 6px #ff6b6b;
    }

    .stat-info-grid {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 8px;
        margin-bottom: 10px;
    }

    .stat-info-cell {
        display: flex;
        flex-direction: column;
        gap: 2px;
        padding: 6px 8px;
        background: var(--bg-tertiary);
        border-radius: 4px;
    }

    .stat-info-key {
        font-size: 8px;
        color: var(--text-muted);
        font-family: var(--font-mono);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .stat-info-val {
        font-size: 11px;
        color: var(--text);
        font-family: var(--font-mono);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .stat-abilities {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .abilities-label {
        font-size: 9px;
        color: var(--text-muted);
        font-family: var(--font-mono);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .abilities-list {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
    }

    .ability-tag {
        padding: 3px 7px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid var(--accent);
        border-radius: 3px;
        font-size: 10px;
        color: var(--accent);
        font-family: var(--font-mono);
    }

    /* Image Grid */
    .image-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
        gap: 12px;
    }

    .image-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 16px 12px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        cursor: pointer;
        transition: all 0.2s;
        font-family: var(--font-mono);
        position: relative;
    }

    .image-card:hover {
        border-color: var(--accent);
        background: var(--accent-dim);
        transform: translateY(-2px);
    }

    .image-icon {
        font-size: 28px;
    }

    .image-name {
        font-size: 11px;
        color: var(--text);
        text-align: center;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 100%;
    }

    .popular-badge {
        position: absolute;
        top: 4px;
        right: 4px;
        font-size: 8px;
        padding: 2px 4px;
        background: var(--accent);
        color: var(--bg);
        text-transform: uppercase;
    }

    .custom-card {
        border-style: dashed;
    }

    /* Form Actions */
    .form-actions {
        display: flex;
        justify-content: flex-start;
        gap: 12px;
        padding-top: 16px;
        border-top: 1px solid var(--border);
    }

    /* Progress Section */
    .progress-section {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 60px 20px;
        text-align: center;
    }

    .progress-header {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 24px;
    }

    .progress-header h2 {
        font-size: 18px;
        text-transform: uppercase;
        margin: 0;
    }

    .progress-percent {
        font-size: 14px;
        color: var(--accent);
        font-weight: 600;
    }

    .progress-bar {
        width: 100%;
        max-width: 400px;
        height: 4px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        margin-bottom: 16px;
        overflow: hidden;
    }

    .progress-fill {
        height: 100%;
        background: var(--accent);
        transition: width 0.3s ease;
    }

    .progress-steps {
        display: flex;
        flex-wrap: wrap;
        justify-content: center;
        gap: 8px;
        margin-bottom: 24px;
        max-width: 500px;
    }

    .progress-step {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 6px 10px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 4px;
        font-size: 11px;
        font-family: var(--font-mono);
        transition: all 0.2s;
    }

    .progress-step.pending {
        opacity: 0.4;
        color: var(--text-muted);
    }

    .progress-step.active {
        border-color: var(--accent);
        color: var(--accent);
        background: rgba(0, 255, 65, 0.1);
        animation: pulse 1.5s infinite;
    }

    .progress-step.completed {
        border-color: var(--green);
        color: var(--green);
    }

    .step-icon {
        font-size: 12px;
    }

    .step-label {
        font-size: 10px;
        text-transform: uppercase;
    }

    .progress-info {
        display: flex;
        flex-direction: column;
        gap: 4px;
        margin-bottom: 16px;
    }

    .progress-message {
        font-size: 13px;
        color: var(--text-muted);
    }

    .installing-tools {
        margin-bottom: 24px;
        padding: 16px;
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: 8px;
        max-width: 400px;
    }

    .installing-label {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0 0 12px 0;
        font-family: var(--font-mono);
    }

    .tools-being-installed {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
    }

    .tool-installing {
        padding: 4px 8px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid var(--accent);
        border-radius: 4px;
        font-size: 11px;
        color: var(--accent);
        font-family: var(--font-mono);
        animation: pulse 1.5s infinite;
    }

    @keyframes pulse {
        0%,
        100% {
            opacity: 1;
        }
        50% {
            opacity: 0.6;
        }
    }

    .progress-spinner {
        margin-top: 20px;
    }

    .spinner-large {
        width: 40px;
        height: 40px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes fadeIn {
        from {
            opacity: 0;
        }
        to {
            opacity: 1;
        }
    }

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    @media (max-width: 600px) {
        .role-grid {
            grid-template-columns: repeat(2, 1fr);
        }

        .image-grid {
            grid-template-columns: repeat(2, 1fr);
        }

        .form-actions {
            flex-direction: column;
        }

        .form-actions button {
            width: 100%;
        }
    }
</style>
