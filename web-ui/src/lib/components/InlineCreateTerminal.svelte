<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { containers, type ProgressEvent } from "$stores/containers";
    import { api, formatMemory, formatStorage, formatCPU } from "$utils/api";
    import PlatformIcon from "./icons/PlatformIcon.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    export let compact = false;

    const dispatch = createEventDispatcher<{
        created: { id: string; name: string };
        cancel: void;
    }>();

    let selectedImage = "";
    let isCreating = false;
    let selectedRole = "vibe-coder";
    let progress = 0;
    let progressMessage = "";
    let progressStage = "";
    let errorMessage = "";
    let customName = "";
    
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
        { id: "validating", label: "Validating", icon: "validating" },
        { id: "pulling", label: "Pulling Image", icon: "pulling" },
        { id: "creating", label: "Creating Terminal", icon: "creating" },
        { id: "starting", label: "Starting", icon: "starting" },
        { id: "configuring", label: "Configuring", icon: "configuring" },
        { id: "ready", label: "Ready", icon: "ready" },
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

    // Pre-bundled OS images for instant loading - synced with backend GetImageMetadata()
    let images: Array<{
        name: string;
        display_name: string;
        description: string;
        category: string;
        popular?: boolean;
    }> = [
        // Debian-based
        { name: "ubuntu", display_name: "Ubuntu 24.04 LTS", description: "Popular Linux distribution", category: "debian", popular: true },
        { name: "ubuntu-22", display_name: "Ubuntu 22.04 LTS", description: "Previous LTS release", category: "debian" },
        { name: "debian", display_name: "Debian 12", description: "Stable Linux distribution", category: "debian", popular: true },
        { name: "debian-11", display_name: "Debian 11", description: "Previous stable release", category: "debian" },
        { name: "mint", display_name: "Linux Mint", description: "User-friendly Ubuntu-based", category: "debian" },
        
        // RHEL-based
        { name: "fedora", display_name: "Fedora 41", description: "Cutting-edge Linux", category: "rhel", popular: true },
        { name: "centos", display_name: "CentOS Stream 9", description: "Enterprise Linux", category: "rhel" },
        { name: "rocky", display_name: "Rocky Linux 9", description: "Enterprise Linux", category: "rhel", popular: true },
        { name: "alma", display_name: "AlmaLinux 9", description: "Enterprise Linux", category: "rhel" },
        { name: "oracle", display_name: "Oracle Linux 9", description: "Enterprise Linux", category: "rhel" },
        { name: "mageia", display_name: "Mageia 9", description: "Mandriva fork", category: "rhel" },
        
        // Arch-based
        { name: "archlinux", display_name: "Arch Linux", description: "Rolling release Linux", category: "arch", popular: true },
        { name: "manjaro", display_name: "Manjaro", description: "User-friendly Arch", category: "arch" },
        { name: "artix", display_name: "Artix Linux", description: "Arch without systemd", category: "arch" },
        
        // SUSE-based
        { name: "opensuse", display_name: "openSUSE Leap 15.6", description: "Enterprise Linux", category: "suse" },
        { name: "tumbleweed", display_name: "openSUSE Tumbleweed", description: "Rolling release", category: "suse" },
        
        // Independent
        { name: "gentoo", display_name: "Gentoo Linux", description: "Source-based distro", category: "independent" },
        { name: "void", display_name: "Void Linux", description: "Independent with runit", category: "independent" },
        { name: "nixos", display_name: "NixOS", description: "Declarative configuration", category: "independent" },
        { name: "slackware", display_name: "Slackware 15.0", description: "Classic Unix-like", category: "independent" },
        { name: "crux", display_name: "CRUX", description: "Lightweight, BSD-style", category: "independent" },
        { name: "guix", display_name: "Guix System", description: "Transactional package manager", category: "independent" },
        
        // Minimal / Embedded
        { name: "alpine", display_name: "Alpine 3.21", description: "Lightweight Linux (6MB)", category: "minimal", popular: true },
        { name: "alpine-3.20", display_name: "Alpine 3.20", description: "Previous stable", category: "minimal" },
        { name: "busybox", display_name: "BusyBox 1.37", description: "Ultra-minimal (~2MB)", category: "minimal" },
        { name: "tinycore", display_name: "TinyCore", description: "Micro Linux (~16MB)", category: "minimal" },
        
        // Cloud Provider
        { name: "amazonlinux", display_name: "Amazon Linux 2023", description: "Optimized for AWS", category: "cloud", popular: true },
        { name: "amazonlinux2", display_name: "Amazon Linux 2", description: "Legacy AWS (EOL 2025)", category: "cloud" },
        { name: "azurelinux", display_name: "Azure Linux", description: "Microsoft Cloud Linux", category: "cloud" },
        
        // Specialized
        { name: "clearlinux", display_name: "Clear Linux", description: "Intel-optimized", category: "specialized" },
        { name: "photon", display_name: "VMware Photon OS 5.0", description: "Container-optimized", category: "specialized" },
        { name: "rancheros", display_name: "RancherOS (Alpine)", description: "Container-optimized", category: "specialized" },
        { name: "neurodebian", display_name: "NeuroDebian", description: "Neuroscience Research", category: "specialized" },
        
        // Security
        { name: "kali", display_name: "Kali Linux", description: "Penetration testing", category: "security", popular: true },
        { name: "parrot", display_name: "Parrot OS", description: "Security distribution", category: "security" },
        { name: "blackarch", display_name: "BlackArch", description: "Security distribution", category: "security" },
        
        // Embedded / IoT
        { name: "raspberrypi", display_name: "Raspberry Pi OS", description: "Debian-based for ARM", category: "embedded" },
        { name: "openwrt", display_name: "OpenWrt", description: "Router/Embedded OS", category: "embedded" },
        
        // macOS
        { name: "macos", display_name: "macOS", description: "Apple macOS (VM-based)", category: "macos", popular: true },
    ];

    const roleToOS: Record<string, string> = {
        "vibe-coder": "ubuntu",
        "gpu-alchemist": "ubuntu",
        "cloud-native": "alpine",
        "remote-access": "alpine",
        "pair-programming": "archlinux",
        "data-science": "ubuntu",
        "minimalist": "alpine",
    };

    $: if (selectedRole && roleToOS[selectedRole]) {
        const preferredOS = roleToOS[selectedRole];
        if (images.some((img) => img.name === preferredOS)) {
            selectedImage = preferredOS;
        }
    }

    const roles = [
        {
            id: "vibe-coder",
            name: "Vibe Coder",
            desc: "AI-assisted development. Cursor, Copilot, Claude vibes.",
            tools: ["node", "python3", "git", "curl", "jq"],
            recommendedOS: "Ubuntu",
        },
        {
            id: "gpu-alchemist",
            name: "GPU Alchemist",
            desc: "Training models, running inference. GPU go brrr.",
            tools: ["python3", "pip", "cuda-toolkit", "pytorch", "jupyter"],
            recommendedOS: "Ubuntu",
        },
        {
            id: "cloud-native",
            name: "Cloud Native",
            desc: "Kubernetes, containers, microservices. Scale to infinity.",
            tools: ["kubectl", "docker", "helm", "terraform", "aws-cli"],
            recommendedOS: "Alpine",
        },
        {
            id: "remote-access",
            name: "Remote Access",
            desc: "Secure gateway to private resources. Share with team.",
            tools: ["ssh", "tmux", "rsync", "curl", "htop"],
            recommendedOS: "Alpine",
        },
        {
            id: "pair-programming",
            name: "Pair Programming",
            desc: "Code together in real-time. Mob programming made easy.",
            tools: ["neovim", "tmux", "git", "fzf", "ripgrep"],
            recommendedOS: "Arch",
        },
        {
            id: "data-science",
            name: "Data Science",
            desc: "Jupyter, pandas, the whole data stack. Insights await.",
            tools: ["python3", "jupyter", "pandas", "numpy", "matplotlib"],
            recommendedOS: "Ubuntu",
        },
        {
            id: "minimalist",
            name: "Minimalist",
            desc: "Just a shell. Nothing more, nothing less.",
            tools: ["bash", "git", "curl", "vim"],
            recommendedOS: "Alpine",
        },
    ];

    $: currentRole = roles.find((r) => r.id === selectedRole);

    // Images are pre-bundled, no need to load from API
    onMount(() => {
        // Select default image based on role
        if (selectedRole && roleToOS[selectedRole]) {
            selectedImage = roleToOS[selectedRole];
        }
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

        // Use custom name or generate a unique name
        const containerName = customName.trim() 
            ? customName.trim() 
            : `${selectedImage}-${Date.now().toString(36)}`;

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
            isCreating = false;
            progress = 0;
            progressMessage = "";
            progressStage = "";
            errorMessage = error || "Failed to create terminal";
        }

        // Reset error state when starting
        errorMessage = "";

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
    {#if errorMessage}
        <div class="create-error">
            <div class="error-header">
                <span class="error-icon">✖</span>
                <h2>Terminal Creation Failed</h2>
            </div>
            <div class="error-content">
                <p class="error-message">{errorMessage}</p>
                {#if errorMessage.includes("tcp://") || errorMessage.includes("docker") || errorMessage.includes("connect")}
                    <div class="error-hint">
                        <p>This may indicate an issue with the Docker host. Check that:</p>
                        <ul>
                            <li>The Docker daemon is running on the remote host</li>
                            <li>TLS certificates are properly configured</li>
                            <li>Firewall rules allow the connection</li>
                        </ul>
                    </div>
                {/if}
            </div>
            <button class="retry-btn" on:click={() => { errorMessage = ""; }}>
                <span>← Try Again</span>
            </button>
        </div>
    {:else if isCreating}
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
                        <span class="step-icon"><StatusIcon status={step.icon} size={12} /></span>
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
            <!-- Terminal Name -->
            <div class="create-section">
                <h4>Terminal Name</h4>
                <div class="name-input-container">
                    <input 
                        type="text" 
                        bind:value={customName} 
                        placeholder="my-awesome-terminal (optional)"
                        class="name-input"
                        maxlength="64"
                    />
                </div>
            </div>

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

    /* Error Display */
    .create-error {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 20px;
        padding: 40px 24px;
        text-align: center;
        min-height: 300px;
    }

    .error-header {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .error-header h2 {
        font-size: 18px;
        text-transform: uppercase;
        margin: 0;
        color: #ff4444;
        letter-spacing: 1px;
    }

    .error-icon {
        font-size: 24px;
        color: #ff4444;
    }

    .error-content {
        max-width: 500px;
    }

    .error-message {
        color: var(--text);
        font-size: 14px;
        font-family: var(--font-mono);
        background: var(--bg-card);
        border: 1px solid #ff4444;
        border-radius: 4px;
        padding: 12px 16px;
        margin: 0;
        word-break: break-word;
        text-align: left;
    }

    .error-hint {
        margin-top: 16px;
        padding: 12px;
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: 4px;
        text-align: left;
    }

    .error-hint p {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0 0 8px 0;
    }

    .error-hint ul {
        font-size: 11px;
        color: var(--text-muted);
        margin: 0;
        padding-left: 20px;
    }

    .error-hint li {
        margin: 4px 0;
    }

    .retry-btn {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 10px 20px;
        background: transparent;
        border: 1px solid var(--accent);
        border-radius: 4px;
        color: var(--accent);
        font-size: 13px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .retry-btn:hover {
        background: rgba(0, 255, 65, 0.1);
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

    /* Name Input */
    .name-input-container {
        width: 100%;
    }

    .name-input {
        width: 100%;
        padding: 10px 12px;
        background: #111;
        border: 1px solid #333;
        border-radius: 4px;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
        transition: all 0.15s ease;
    }

    .name-input:focus {
        outline: none;
        border-color: var(--accent);
        box-shadow: 0 0 0 1px var(--accent-dim);
    }

    .name-input::placeholder {
        color: var(--text-muted);
        opacity: 0.7;
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
