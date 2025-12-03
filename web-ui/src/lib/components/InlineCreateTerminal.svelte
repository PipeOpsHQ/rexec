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
    
    // Resource customization
    let showResources = false;
    let memoryMB = 512;
    let cpuShares = 512;
    let diskMB = 2048;
    
    // Trial limits - allow more memory than CPU
    const resourceLimits = {
        minMemory: 256,
        maxMemory: 2048,  // Allow up to 2GB for trial
        minCPU: 256,
        maxCPU: 1024,
        minDisk: 1024,
        maxDisk: 8192
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
        { id: "validating", label: "Validating" },
        { id: "pulling", label: "Pulling Image" },
        { id: "creating", label: "Creating Container" },
        { id: "starting", label: "Starting" },
        { id: "configuring", label: "Configuring" },
        { id: "ready", label: "Ready" },
    ];

    // Matrix characters for animation
    const matrixChars = '█▓▒░╬╫╪┼┿╀╁╂╃╄╅╆╇╈╉╊╋';
    
    // ASCII art frames for animation
    const asciiFrames = [
`╔══════════════════════════════════════════╗
║  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗ ║
║  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝ ║
║  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║      ║
║  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║      ║
║  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗ ║
║  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ║
╠══════════════════════════════════════════╣
║  ▓▓▓ INITIALIZING SECURE CONTAINER ▓▓▓   ║
╚══════════════════════════════════════════╝`,
`╔══════════════════════════════════════════╗
║  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗ ║
║  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝ ║
║  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║      ║
║  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║      ║
║  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗ ║
║  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ║
╠══════════════════════════════════════════╣
║  ░░░ INITIALIZING SECURE CONTAINER ░░░   ║
╚══════════════════════════════════════════╝`,
`╔══════════════════════════════════════════╗
║  ██████╗ ███████╗██╗  ██╗███████╗ ██████╗ ║
║  ██╔══██╗██╔════╝╚██╗██╔╝██╔════╝██╔════╝ ║
║  ██████╔╝█████╗   ╚███╔╝ █████╗  ██║      ║
║  ██╔══██╗██╔══╝   ██╔██╗ ██╔══╝  ██║      ║
║  ██║  ██║███████╗██╔╝ ██╗███████╗╚██████╗ ║
║  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚══════╝ ╚═════╝ ║
╠══════════════════════════════════════════╣
║  ▒▒▒ INITIALIZING SECURE CONTAINER ▒▒▒   ║
╚══════════════════════════════════════════╝`
    ];
    
    let asciiFrameIndex = 0;
    let asciiFrame = asciiFrames[0];
    let animationInterval: ReturnType<typeof setInterval>;
    
    $: if (isCreating && !animationInterval) {
        animationInterval = setInterval(() => {
            asciiFrameIndex = (asciiFrameIndex + 1) % asciiFrames.length;
            asciiFrame = asciiFrames[asciiFrameIndex];
        }, 400);
    }
    
    $: if (!isCreating && animationInterval) {
        clearInterval(animationInterval);
        animationInterval = undefined as any;
    }

    // Hacker-style log messages
    let logMessages: Array<{ text: string; type: 'info' | 'success' | 'cmd' | 'data' }> = [];
    let logContainer: HTMLDivElement;
    
    const stageLogMessages: Record<string, Array<{ text: string; type: 'info' | 'success' | 'cmd' | 'data' }>> = {
        validating: [
            { text: '$ rexec init --validate', type: 'cmd' },
            { text: '[SYS] Authenticating session...', type: 'info' },
            { text: '[AUTH] Token verified ✓', type: 'success' },
            { text: '[QUOTA] Checking resource allocation...', type: 'info' },
        ],
        pulling: [
            { text: '$ docker pull registry.rexec.io/base', type: 'cmd' },
            { text: '[NET] Connecting to registry...', type: 'info' },
            { text: '[PULL] Downloading layers...', type: 'data' },
            { text: '[CACHE] Layer sha256:a3ed... cached', type: 'info' },
            { text: '[PULL] Extracting filesystem...', type: 'data' },
        ],
        creating: [
            { text: '$ rexec container create --secure', type: 'cmd' },
            { text: '[DOCKER] Allocating container ID...', type: 'info' },
            { text: '[NET] Configuring network namespace...', type: 'info' },
            { text: '[FS] Mounting overlay filesystem...', type: 'data' },
            { text: '[SEC] Applying seccomp profile...', type: 'info' },
        ],
        starting: [
            { text: '$ rexec container start', type: 'cmd' },
            { text: '[INIT] Starting container process...', type: 'info' },
            { text: '[PID] Process spawned: 1', type: 'data' },
            { text: '[TTY] Allocating pseudo-terminal...', type: 'info' },
            { text: '[WS] WebSocket channel ready', type: 'success' },
        ],
        configuring: [
            { text: '$ rexec setup --role ${role}', type: 'cmd' },
            { text: '[PKG] Updating package index...', type: 'info' },
            { text: '[INSTALL] Installing development tools...', type: 'data' },
            { text: '[CONFIG] Writing shell configuration...', type: 'info' },
            { text: '[ENV] Setting environment variables...', type: 'data' },
        ],
        ready: [
            { text: '[SYS] Container ready ✓', type: 'success' },
            { text: '[WS] Terminal connection established', type: 'success' },
            { text: '$ echo "Welcome to Rexec"', type: 'cmd' },
        ],
    };
    
    let prevStage = '';
    $: if (progressStage && progressStage !== prevStage) {
        prevStage = progressStage;
        const newLogs = stageLogMessages[progressStage] || [];
        // Add logs with slight delay for effect
        newLogs.forEach((log, i) => {
            setTimeout(() => {
                logMessages = [...logMessages, log];
                // Auto-scroll to bottom
                if (logContainer) {
                    logContainer.scrollTop = logContainer.scrollHeight;
                }
            }, i * 150);
        });
    }

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
            logMessages = [];
            prevStage = '';
        }

        function handleError(error: string) {
            logMessages = [...logMessages, { text: `[ERROR] ${error}`, type: 'info' as const }];
            progressMessage = error || "Failed to create terminal";
            setTimeout(() => {
                isCreating = false;
                progress = 0;
                progressMessage = "";
                progressStage = "";
                logMessages = [];
                prevStage = '';
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
            <!-- Cyber ASCII Art Header -->
            <div class="ascii-container">
                <pre class="ascii-art">{asciiFrame}</pre>
            </div>
            
            <!-- Matrix-style progress bar -->
            <div class="matrix-progress">
                <div class="matrix-bar">
                    {#each Array(40) as _, i}
                        <span 
                            class="matrix-cell" 
                            class:active={i < (displayProgress / 100) * 40}
                            class:pulse={i === Math.floor((displayProgress / 100) * 40) - 1}
                        >{i < (displayProgress / 100) * 40 ? matrixChars[Math.floor(Math.random() * matrixChars.length)] : '░'}</span>
                    {/each}
                </div>
                <div class="progress-label">
                    <span class="hex-progress">0x{displayProgress.toString(16).toUpperCase().padStart(2, '0')}</span>
                    <span class="stage-text">{progressStage.toUpperCase()}</span>
                    <span class="percent">{displayProgress}%</span>
                </div>
            </div>
            
            <!-- Cyberpunk log stream -->
            <div class="cyber-logs" bind:this={logContainer}>
                <div class="scanline"></div>
                {#each logMessages as log, i}
                    <div class="log-entry" class:cmd={log.type === 'cmd'} class:success={log.type === 'success'} class:data={log.type === 'data'}>
                        <span class="log-time">[{String(i).padStart(3, '0')}]</span>
                        <span class="log-icon">{log.type === 'cmd' ? '▶' : log.type === 'success' ? '✓' : '→'}</span>
                        <span class="log-msg">{log.text}</span>
                        {#if i === logMessages.length - 1}
                            <span class="blink-cursor">█</span>
                        {/if}
                    </div>
                {/each}
            </div>
            
            <!-- DNA-style stage helix -->
            <div class="stage-helix">
                {#each progressSteps as step, i}
                    <div class="helix-node" class:active={progressStage === step.id} class:done={getStepStatus(step.id) === 'completed'}>
                        <span class="node-line">{getStepStatus(step.id) === 'completed' ? '═══' : getStepStatus(step.id) === 'active' ? '▓▓▓' : '───'}</span>
                        <span class="node-dot">{getStepStatus(step.id) === 'completed' ? '◆' : getStepStatus(step.id) === 'active' ? '◈' : '◇'}</span>
                        {#if i < progressSteps.length - 1}
                            <span class="node-line">{getStepStatus(step.id) === 'completed' ? '═══' : '───'}</span>
                        {/if}
                    </div>
                {/each}
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
                        {memoryMB}MB / {cpuShares} CPU / {diskMB}MB
                    </span>
                </button>
                
                {#if showResources}
                    <div class="resource-config">
                        <div class="resource-row">
                            <label>
                                <span class="resource-label">Memory</span>
                                <span class="resource-value">{memoryMB} MB</span>
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
                                <span>{resourceLimits.minMemory}MB</span>
                                <span>{resourceLimits.maxMemory}MB</span>
                            </div>
                        </div>
                        
                        <div class="resource-row">
                            <label>
                                <span class="resource-label">CPU</span>
                                <span class="resource-value">{cpuShares} shares</span>
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
                                <span>{resourceLimits.minCPU}</span>
                                <span>{resourceLimits.maxCPU}</span>
                            </div>
                        </div>
                        
                        <div class="resource-row">
                            <label>
                                <span class="resource-label">Disk</span>
                                <span class="resource-value">{diskMB} MB</span>
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
                                <span>{resourceLimits.minDisk}MB</span>
                                <span>{resourceLimits.maxDisk}MB</span>
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

    /* Progress - Cyberpunk Style */
    .create-progress {
        display: flex;
        flex-direction: column;
        gap: 12px;
        padding: 16px;
        background: linear-gradient(135deg, #000 0%, #0a0a0a 50%, #000 100%);
        border: 1px solid #00ff41;
        border-radius: 4px;
        font-family: var(--font-mono);
        box-shadow: 
            0 0 30px rgba(0, 255, 65, 0.15),
            inset 0 0 60px rgba(0, 255, 65, 0.03),
            0 0 1px #00ff41;
        position: relative;
        overflow: hidden;
    }
    
    .create-progress::before {
        content: '';
        position: absolute;
        top: 0;
        left: -100%;
        width: 100%;
        height: 100%;
        background: linear-gradient(90deg, transparent, rgba(0, 255, 65, 0.05), transparent);
        animation: shimmer 2s infinite;
    }
    
    @keyframes shimmer {
        100% { left: 100%; }
    }

    /* ASCII Art Container */
    .ascii-container {
        text-align: center;
        padding: 8px 0;
    }
    
    .ascii-art {
        font-size: 7px;
        line-height: 1.1;
        color: #00ff41;
        text-shadow: 0 0 10px rgba(0, 255, 65, 0.8), 0 0 20px rgba(0, 255, 65, 0.4);
        margin: 0;
        letter-spacing: 0.5px;
        animation: glow 2s ease-in-out infinite alternate;
    }
    
    @keyframes glow {
        from { text-shadow: 0 0 5px rgba(0, 255, 65, 0.5), 0 0 10px rgba(0, 255, 65, 0.3); }
        to { text-shadow: 0 0 15px rgba(0, 255, 65, 0.9), 0 0 30px rgba(0, 255, 65, 0.5); }
    }
    
    /* Matrix Progress Bar */
    .matrix-progress {
        padding: 8px 0;
    }
    
    .matrix-bar {
        display: flex;
        justify-content: center;
        font-size: 12px;
        letter-spacing: -1px;
    }
    
    .matrix-cell {
        color: #1a3a1a;
        transition: all 0.1s ease;
    }
    
    .matrix-cell.active {
        color: #00ff41;
        text-shadow: 0 0 8px rgba(0, 255, 65, 0.9);
        animation: matrix-flicker 0.1s ease;
    }
    
    .matrix-cell.pulse {
        animation: matrix-pulse 0.5s ease infinite;
    }
    
    @keyframes matrix-flicker {
        0% { opacity: 0.3; }
        50% { opacity: 1; }
        100% { opacity: 0.9; }
    }
    
    @keyframes matrix-pulse {
        0%, 100% { color: #00ff41; text-shadow: 0 0 10px rgba(0, 255, 65, 1); }
        50% { color: #00aa28; text-shadow: 0 0 5px rgba(0, 255, 65, 0.5); }
    }
    
    .progress-label {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-top: 8px;
        padding: 0 4px;
        font-size: 10px;
    }
    
    .hex-progress {
        color: #ff00ff;
        font-weight: bold;
        text-shadow: 0 0 5px rgba(255, 0, 255, 0.5);
    }
    
    .stage-text {
        color: #0af;
        letter-spacing: 2px;
        animation: stage-blink 1s ease infinite;
    }
    
    @keyframes stage-blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.6; }
    }
    
    .percent {
        color: #00ff41;
        font-weight: bold;
    }
    
    /* Cyber Logs */
    .cyber-logs {
        height: 140px;
        overflow-y: auto;
        background: rgba(0, 0, 0, 0.8);
        border: 1px solid #00ff4133;
        border-radius: 2px;
        padding: 8px;
        position: relative;
        scrollbar-width: thin;
        scrollbar-color: #00ff41 #111;
    }
    
    .cyber-logs::-webkit-scrollbar { width: 3px; }
    .cyber-logs::-webkit-scrollbar-track { background: #0a0a0a; }
    .cyber-logs::-webkit-scrollbar-thumb { background: #00ff41; }
    
    .scanline {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 2px;
        background: linear-gradient(90deg, transparent, rgba(0, 255, 65, 0.3), transparent);
        animation: scanline 3s linear infinite;
        pointer-events: none;
    }
    
    @keyframes scanline {
        0% { top: 0; }
        100% { top: 100%; }
    }
    
    .log-entry {
        display: flex;
        gap: 8px;
        font-size: 10px;
        line-height: 1.8;
        color: #666;
        animation: log-slide 0.2s ease;
    }
    
    @keyframes log-slide {
        from { opacity: 0; transform: translateX(-10px); }
        to { opacity: 1; transform: translateX(0); }
    }
    
    .log-time {
        color: #444;
        font-size: 9px;
    }
    
    .log-icon {
        color: #555;
    }
    
    .log-entry.cmd { color: #fff; }
    .log-entry.cmd .log-icon { color: #00ff41; }
    .log-entry.success { color: #00ff41; }
    .log-entry.success .log-icon { color: #00ff41; }
    .log-entry.data { color: #0af; }
    
    .blink-cursor {
        animation: blink 0.7s step-end infinite;
        color: #00ff41;
    }
    
    @keyframes blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0; }
    }
    
    /* DNA Helix Stage Indicator */
    .stage-helix {
        display: flex;
        justify-content: center;
        align-items: center;
        padding: 8px 0;
        border-top: 1px solid rgba(0, 255, 65, 0.15);
    }
    
    .helix-node {
        display: flex;
        align-items: center;
        font-size: 10px;
    }
    
    .node-line {
        color: #222;
        letter-spacing: -2px;
    }
    
    .node-dot {
        color: #333;
        font-size: 12px;
        transition: all 0.3s ease;
    }
    
    .helix-node.active .node-dot {
        color: #00ff41;
        text-shadow: 0 0 10px rgba(0, 255, 65, 1);
        animation: node-pulse 1s ease infinite;
    }
    
    .helix-node.active .node-line {
        color: #00aa28;
    }
    
    .helix-node.done .node-dot {
        color: #00ff41;
    }
    
    .helix-node.done .node-line {
        color: #00ff41;
        text-shadow: 0 0 5px rgba(0, 255, 65, 0.5);
    }
    
    @keyframes node-pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.3); }
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
