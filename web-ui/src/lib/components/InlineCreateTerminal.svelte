<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { containers, type ProgressEvent } from "$stores/containers";
    import { api, formatMemory, formatStorage, formatCPU } from "$utils/api";
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
    
    // Resource customization
    let showResources = false;
    let memoryMB = 512;
    let cpuShares = 512;
    let diskMB = 2048;
    
    // Trial limits - generous during 60-day trial period
    const resourceLimits = {
        minMemory: 256,
        maxMemory: 4096,  // 4GB for trial
        minCPU: 250,
        maxCPU: 2000,     // 2 vCPU for trial
        minDisk: 1024,
        maxDisk: 16384    // 16GB for trial
    };

    // Slider event handlers
    function handleMemoryChange(e: Event) {
        memoryMB = parseInt((e.target as HTMLInputElement).value);
    }

    function handleCpuChange(e: Event) {
        cpuShares = parseInt((e.target as HTMLInputElement).value);
    }

    function handleDiskChange(e: Event) {
        diskMB = parseInt((e.target as HTMLInputElement).value);
    }

    const progressSteps = [
        { id: "validating", label: "Validating", icon: "✓" },
        { id: "pulling", label: "Pulling Image", icon: "📦" },
        { id: "creating", label: "Creating Container", icon: "🔧" },
        { id: "starting", label: "Starting", icon: "🚀" },
        { id: "configuring", label: "Configuring", icon: "⚙️" },
        { id: "ready", label: "Ready", icon: "✨" },
    ];

    // Reactive step statuses - must depend on progressStage to update
    $: stepStatuses = progressSteps.reduce((acc, step) => {
        const stepOrder = progressSteps.map((s) => s.id);
        const currentIndex = stepOrder.indexOf(progressStage);
        const stepIndex = stepOrder.indexOf(step.id);
        
        if (stepIndex < currentIndex) acc[step.id] = "completed";
        else if (stepIndex === currentIndex) acc[step.id] = "active";
        else acc[step.id] = "pending";
        return acc;
    }, {} as Record<string, "pending" | "active" | "completed">);

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

    async function createContainer() {
        if (!selectedImage || isCreating) return;

        isCreating = true;
        progress = 0;
        progressMessage = "Starting...";
        progressStage = "validating";

        // Generate a unique name for the container
        const containerName = `${selectedImage}-${Date.now().toString(36)}`;

        function handleProgress(event: ProgressEvent) {
            progress = event.progress;
            progressMessage = event.message;
            progressStage = event.stage;
        }

        function handleComplete(container: { id: string; name: string }) {
            dispatch("created", { id: container.id, name: container.name });
            isCreating = false;
            progress = 0;
            progressMessage = "";
            progressStage = "";
        }

        function handleError(error: string) {
            progressMessage = error || "Failed to create terminal";
            setTimeout(() => {
                isCreating = false;
                progress = 0;
                progressMessage = "";
                progressStage = "";
            }, 3000);
        }

        // Call with correct parameters: name, image, customImage, role, onProgress, onComplete, onError, resources
        containers.createContainerWithProgress(
            containerName,
            selectedImage,
            undefined,  // customImage
            selectedRole,
            handleProgress,
            handleComplete,
            handleError,
            { memory_mb: memoryMB, cpu_shares: cpuShares, disk_mb: diskMB }
        );
    }
</script>

<div class="inline-create" class:compact>
    {#if isCreating}
        <div class="create-progress">
            <div class="progress-header">
                <h2>Creating Terminal</h2>
                <span class="progress-percent">{displayProgress}%</span>
            </div>
            <div class="progress-bar">
                <div class="progress-fill" style="width: {displayProgress}%"></div>
            </div>
            
            <!-- Step Indicators -->
            <div class="progress-steps">
                {#each progressSteps as step (step.id)}
                    <div class="progress-step {stepStatuses[step.id] || 'pending'}">
                        <span class="step-icon">{step.icon}</span>
                        <span class="step-label">{step.label}</span>
                    </div>
                {/each}
            </div>
            
            <p class="progress-message">{progressMessage}</p>
            
            <!-- Role-specific tools being installed -->
            {#if currentRole && progressStage === "configuring"}
                <div class="installing-tools">
                    <p class="installing-label">Installing tools for {currentRole.name}:</p>
                    <div class="tools-installing">
                        {#each currentRole.tools as tool}
                            <span class="tool-badge-installing">{tool}</span>
                        {/each}
                    </div>
                </div>
            {/if}
            
            <div class="progress-spinner">
                <div class="spinner-large"></div>
            </div>
        </div>
    {:else}
        <div class="create-content">
            <!-- Role Selection -->
            <div class="create-section">
                <h4>Environment</h4>
                <div class="role-grid">
                    {#each roles as role}
                        <button
                            class="role-card"
                            class:selected={selectedRole === role.id}
                            on:click={() => (selectedRole = role.id)}
                            title={role.desc}
                        >
                            <PlatformIcon platform={role.id} size={28} />
                            <span class="role-name">{role.name}</span>
                        </button>
                    {/each}
                </div>
                {#if currentRole}
                    <div class="role-info">
                        <div class="role-header-row">
                            <PlatformIcon platform={currentRole.id} size={18} />
                            <span class="role-name-sm">{currentRole.name}</span>
                            <span class="role-os-badge">
                                <PlatformIcon platform={currentRole.recommendedOS.toLowerCase()} size={14} />
                                {currentRole.recommendedOS}
                            </span>
                        </div>
                        <div class="role-tools">
                            {#each currentRole.tools as tool}
                                <span class="tool-badge">{tool}</span>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>

            <!-- Resource Configuration (Trial users can customize) -->
            <div class="create-section">
                <button 
                    class="resource-toggle"
                    on:click={() => showResources = !showResources}
                >
                    <span class="toggle-icon">{showResources ? '▼' : '▶'}</span>
                    <h4>Resources</h4>
                    <span class="resource-preview">
                        {formatMemory(memoryMB)} / {formatCPU(cpuShares)} / {formatStorage(diskMB)}
                    </span>
                </button>
                
                {#if showResources}
                    <div class="resource-config">
                        <div class="resource-row">
                            <label>
                                <span class="resource-label">Memory</span>
                                <span class="resource-value">{formatMemory(memoryMB)}</span>
                            </label>
                            <input 
                                type="range" 
                                value={memoryMB}
                                on:input={handleMemoryChange}
                                min={resourceLimits.minMemory}
                                max={resourceLimits.maxMemory}
                                step="128"
                            />
                            <div class="resource-range">
                                <span>{formatMemory(resourceLimits.minMemory)}</span>
                                <span>{formatMemory(resourceLimits.maxMemory)}</span>
                            </div>
                        </div>
                        
                        <div class="resource-row">
                            <label>
                                <span class="resource-label">CPU</span>
                                <span class="resource-value">{formatCPU(cpuShares)}</span>
                            </label>
                            <input 
                                type="range" 
                                value={cpuShares}
                                on:input={handleCpuChange}
                                min={resourceLimits.minCPU}
                                max={resourceLimits.maxCPU}
                                step="128"
                            />
                            <div class="resource-range">
                                <span>{formatCPU(resourceLimits.minCPU)}</span>
                                <span>{formatCPU(resourceLimits.maxCPU)}</span>
                            </div>
                        </div>
                        
                        <div class="resource-row">
                            <label>
                                <span class="resource-label">Disk</span>
                                <span class="resource-value">{formatStorage(diskMB)}</span>
                            </label>
                            <input 
                                type="range" 
                                value={diskMB}
                                on:input={handleDiskChange}
                                min={resourceLimits.minDisk}
                                max={resourceLimits.maxDisk}
                                step="256"
                            />
                            <div class="resource-range">
                                <span>{formatStorage(resourceLimits.minDisk)}</span>
                                <span>{formatStorage(resourceLimits.maxDisk)}</span>
                            </div>
                        </div>
                        
                        <p class="resource-hint">
                            Trial users can customize resources within these limits
                        </p>
                    </div>
                {/if}
            </div>

            <!-- OS Selection -->
            <div class="create-section">
                <h4>Operating System</h4>
                <div class="os-grid">
                    {#each images as image (image.name)}
                        <button
                            class="os-card"
                            on:click={() => selectAndCreate(image.name)}
                        >
                            <PlatformIcon platform={image.name} size={28} />
                            <span class="os-name">{image.display_name || image.name}</span>
                            {#if image.popular}
                                <span class="popular-badge">Popular</span>
                            {/if}
                        </button>
                    {/each}
                    <button
                        class="os-card"
                        on:click={() => selectAndCreate("custom")}
                    >
                        <PlatformIcon platform="custom" size={28} />
                        <span class="os-name">Custom</span>
                    </button>
                </div>
            </div>
        </div>
    {/if}
</div>

<style>
    .inline-create {
        padding: 16px;
        height: 100%;
        overflow-y: auto;
        background: #0a0a0a;
    }

    .inline-create.compact {
        padding: 12px;
    }

    /* Progress */
    .create-progress {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 16px;
        padding: 60px 24px;
        text-align: center;
        flex: 1;
        min-height: 300px;
    }

    .progress-header {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 8px;
    }

    .progress-header h2 {
        font-size: 18px;
        text-transform: uppercase;
        margin: 0;
        color: var(--text);
        letter-spacing: 1px;
    }

    .progress-percent {
        font-size: 14px;
        color: var(--accent);
        font-weight: 600;
        font-family: var(--font-mono);
    }

    .progress-bar {
        width: 100%;
        max-width: 400px;
        height: 4px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
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
        flex-wrap: wrap;
        justify-content: center;
        gap: 8px;
        margin: 16px 0;
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

    .progress-message {
        color: var(--text-muted);
        font-size: 14px;
        margin: 0;
    }

    .installing-tools {
        margin-top: 16px;
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

    .tools-installing {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        justify-content: center;
    }

    .tool-badge-installing {
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
        0%, 100% { opacity: 1; }
        50% { opacity: 0.6; }
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

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    /* Content */
    .create-content {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .create-section h4 {
        margin: 0 0 12px 0;
        font-size: 13px;
        font-weight: 600;
        color: var(--accent);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    /* Role Grid */
    .role-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
        gap: 8px;
    }

    .role-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 12px 8px;
        background: #1a1a1a;
        border: 1px solid #333;
        border-radius: 6px;
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .role-card:hover {
        border-color: var(--text-muted);
        background: #222;
    }

    .role-card.selected {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
        box-shadow: 0 0 8px rgba(0, 255, 65, 0.2);
    }

    .role-card :global(.platform-icon) {
        filter: drop-shadow(0 0 4px rgba(0, 255, 65, 0.3));
    }

    .role-name {
        font-size: 11px;
        color: #e0e0e0;
        text-align: center;
        font-weight: 500;
    }

    /* Role Info */
    .role-info {
        margin-top: 12px;
        padding: 10px;
        background: #111;
        border: 1px solid #333;
        border-radius: 4px;
    }

    .role-header-row {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;
    }

    .role-name-sm {
        font-size: 12px;
        font-weight: 600;
        color: var(--text);
    }

    .role-os-badge {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        margin-left: auto;
        padding: 2px 6px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 3px;
        font-size: 10px;
        color: var(--text-muted);
    }

    .role-tools {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
    }

    .tool-badge {
        padding: 2px 6px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 3px;
        font-size: 10px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    /* OS Grid */
    .os-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
        gap: 8px;
    }

    .os-card {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 6px;
        padding: 12px 8px;
        background: #1a1a1a;
        border: 1px solid #333;
        border-radius: 6px;
        cursor: pointer;
        transition: all 0.15s ease;
        position: relative;
    }

    .os-card:hover {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
        transform: translateY(-2px);
    }

    .os-card :global(.platform-icon) {
        filter: drop-shadow(0 0 4px rgba(0, 255, 65, 0.3));
    }

    .os-name {
        font-size: 11px;
        color: #e0e0e0;
        text-align: center;
    }

    .popular-badge {
        position: absolute;
        top: 4px;
        right: 4px;
        padding: 1px 4px;
        background: var(--accent);
        color: #000;
        font-size: 8px;
        font-weight: 600;
        border-radius: 2px;
        text-transform: uppercase;
    }

    /* Resource Configuration */
    .resource-toggle {
        display: flex;
        align-items: center;
        gap: 8px;
        width: 100%;
        padding: 8px 12px;
        background: #111;
        border: 1px solid #333;
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .resource-toggle:hover {
        border-color: var(--text-muted);
        background: #1a1a1a;
    }

    .resource-toggle h4 {
        margin: 0;
        font-size: 12px;
        color: var(--accent);
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .toggle-icon {
        font-size: 10px;
        color: var(--text-muted);
    }

    .resource-preview {
        margin-left: auto;
        font-size: 11px;
        font-family: var(--font-mono);
        color: var(--text-muted);
    }

    .resource-config {
        margin-top: 12px;
        padding: 12px;
        background: #111;
        border: 1px solid #333;
        border-radius: 4px;
        display: flex;
        flex-direction: column;
        gap: 16px;
    }

    .resource-row {
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    .resource-row label {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .resource-label {
        font-size: 12px;
        color: var(--text);
        font-weight: 500;
    }

    .resource-value {
        font-size: 12px;
        font-family: var(--font-mono);
        color: var(--accent);
        font-weight: 600;
    }

    .resource-row input[type="range"] {
        width: 100%;
        height: 6px;
        -webkit-appearance: none;
        appearance: none;
        background: #333;
        border-radius: 3px;
        outline: none;
        margin: 8px 0;
        cursor: pointer;
    }

    .resource-row input[type="range"]::-webkit-slider-runnable-track {
        width: 100%;
        height: 6px;
        background: #333;
        border-radius: 3px;
    }

    .resource-row input[type="range"]::-webkit-slider-thumb {
        -webkit-appearance: none;
        width: 16px;
        height: 16px;
        background: var(--accent);
        border-radius: 50%;
        cursor: pointer;
        box-shadow: 0 0 8px rgba(0, 255, 65, 0.5);
        margin-top: -5px;
        transition: transform 0.15s, box-shadow 0.15s;
    }

    .resource-row input[type="range"]::-webkit-slider-thumb:hover {
        transform: scale(1.1);
        box-shadow: 0 0 12px rgba(0, 255, 65, 0.7);
    }

    .resource-row input[type="range"]::-moz-range-track {
        width: 100%;
        height: 6px;
        background: #333;
        border-radius: 3px;
    }

    .resource-row input[type="range"]::-moz-range-thumb {
        width: 16px;
        height: 16px;
        background: var(--accent);
        border: none;
        border-radius: 50%;
        cursor: pointer;
        box-shadow: 0 0 8px rgba(0, 255, 65, 0.5);
    }

    .resource-row input[type="range"]::-moz-range-thumb:hover {
        box-shadow: 0 0 12px rgba(0, 255, 65, 0.7);
    }

    .resource-range {
        display: flex;
        justify-content: space-between;
        font-size: 9px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    .resource-hint {
        font-size: 10px;
        color: var(--text-muted);
        margin: 4px 0 0 0;
        text-align: center;
        font-style: italic;
    }

    /* Compact mode adjustments */
    .compact .role-grid {
        grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    }

    .compact .os-grid {
        grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
    }

    .compact .role-card,
    .compact .os-card {
        padding: 10px 6px;
    }
</style>
