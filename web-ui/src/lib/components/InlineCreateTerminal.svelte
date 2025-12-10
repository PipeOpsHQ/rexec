<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { slide } from "svelte/transition";
    import { containers, type ProgressEvent } from "$stores/containers";
    import { roles } from "$stores/roles";
    import { userTier, subscriptionActive } from "$stores/auth";
    import { api, formatMemory, formatStorage, formatCPU } from "$utils/api";
    import PlatformIcon from "./icons/PlatformIcon.svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";

    export let compact = false;

    const dispatch = createEventDispatcher<{
        created: { id: string; name: string };
        cancel: void;
        upgrade: void;
    }>();

    let selectedImage = "";
    let isCreating = false;
    let selectedRole = "standard";
    let progress = 0;
    let progressMessage = "";
    let progressStage = "";
    let errorMessage = "";
    let customName = "";
    let customImage = "";

    // Resource customization
    let showResources = false;
    let memoryMB = 512;
    let cpuShares = 512;
    let diskMB = 2048;

    // Resource limits based on plan tier
    $: resourceLimits = (() => {
        if ($subscriptionActive) {
            return { minMemory: 256, maxMemory: 4096, minCPU: 250, maxCPU: 4000, minDisk: 1024, maxDisk: 20480 };
        }
        switch ($userTier) {
            case "guest":
                return { minMemory: 256, maxMemory: 512, minCPU: 250, maxCPU: 500, minDisk: 1024, maxDisk: 2048 };
            case "free":
                return { minMemory: 256, maxMemory: 2048, minCPU: 250, maxCPU: 2000, minDisk: 1024, maxDisk: 10240 };
            case "pro":
                return { minMemory: 256, maxMemory: 4096, minCPU: 250, maxCPU: 4000, minDisk: 1024, maxDisk: 20480 };
            case "enterprise":
                return { minMemory: 256, maxMemory: 8192, minCPU: 250, maxCPU: 8000, minDisk: 1024, maxDisk: 51200 };
            default: // Free fallback
                return { minMemory: 256, maxMemory: 2048, minCPU: 250, maxCPU: 2000, minDisk: 1024, maxDisk: 10240 };
        }
    })();

    // Clamp values when limits change
    $: {
        if (memoryMB > resourceLimits.maxMemory) memoryMB = resourceLimits.maxMemory;
        if (cpuShares > resourceLimits.maxCPU) cpuShares = resourceLimits.maxCPU;
        if (diskMB > resourceLimits.maxDisk) diskMB = resourceLimits.maxDisk;
    }

    // Check if user can upgrade to get more resources
    $: canUpgrade = !$subscriptionActive && ($userTier === "guest" || $userTier === "free" || $userTier === "pro");
    $: nextTierName = (() => {
        if ($subscriptionActive && $userTier !== "enterprise") return "Enterprise";
        if ($userTier === "guest") return "Free";
        if ($userTier === "free") return "Pro";
        if ($userTier === "pro") return "Enterprise";
        return null;
    })();
    
    // Check if user is at max resources for their tier
    $: isAtMaxMemory = memoryMB >= resourceLimits.maxMemory;
    $: isAtMaxCPU = cpuShares >= resourceLimits.maxCPU;
    $: isAtMaxDisk = diskMB >= resourceLimits.maxDisk;
    $: showUpgradeHint = canUpgrade && (isAtMaxMemory || isAtMaxCPU || isAtMaxDisk);

    // Get next tier limits for comparison
    $: nextTierLimits = (() => {
        // Users with active subscription but not enterprise can upgrade to enterprise
        if ($subscriptionActive && $userTier !== "enterprise") {
            return { maxMemory: 8192, maxCPU: 8000, maxDisk: 51200 }; // Enterprise tier
        }
        switch ($userTier) {
            case "guest":
                return { maxMemory: 2048, maxCPU: 2000, maxDisk: 10240 }; // Free tier
            case "free":
                return { maxMemory: 4096, maxCPU: 4000, maxDisk: 20480 }; // Pro tier
            case "pro":
                return { maxMemory: 8192, maxCPU: 8000, maxDisk: 51200 }; // Enterprise tier
            default:
                return null;
        }
    })();

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
    $: stepStatuses = progressSteps.reduce(
        (acc, step) => {
            const stepOrder = progressSteps.map((s) => s.id);
            const currentIndex = stepOrder.indexOf(progressStage);
            const stepIndex = stepOrder.indexOf(step.id);

            if (stepIndex < currentIndex) acc[step.id] = "completed";
            else if (stepIndex === currentIndex) acc[step.id] = "active";
            else acc[step.id] = "pending";
            return acc;
        },
        {} as Record<string, "pending" | "active" | "completed">,
    );

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
        {
            name: "ubuntu",
            display_name: "Ubuntu 24.04 LTS",
            description: "Popular Linux distribution",
            category: "debian",
            popular: true,
        },
        {
            name: "ubuntu-22",
            display_name: "Ubuntu 22.04 LTS",
            description: "Previous LTS release",
            category: "debian",
        },
        {
            name: "debian",
            display_name: "Debian 12",
            description: "Stable Linux distribution",
            category: "debian",
            popular: true,
        },
        {
            name: "debian-11",
            display_name: "Debian 11",
            description: "Previous stable release",
            category: "debian",
        },
        {
            name: "mint",
            display_name: "Linux Mint",
            description: "User-friendly Ubuntu-based",
            category: "debian",
        },

        // RHEL-based
        {
            name: "fedora",
            display_name: "Fedora 41",
            description: "Cutting-edge Linux",
            category: "rhel",
            popular: true,
        },
        {
            name: "centos",
            display_name: "CentOS Stream 9",
            description: "Enterprise Linux",
            category: "rhel",
        },
        {
            name: "rocky",
            display_name: "Rocky Linux 9",
            description: "Enterprise Linux",
            category: "rhel",
            popular: true,
        },
        {
            name: "alma",
            display_name: "AlmaLinux 9",
            description: "Enterprise Linux",
            category: "rhel",
        },
        {
            name: "oracle",
            display_name: "Oracle Linux 9",
            description: "Enterprise Linux",
            category: "rhel",
        },
        {
            name: "mageia",
            display_name: "Mageia 9",
            description: "Mandriva fork",
            category: "rhel",
        },

        // Arch-based
        {
            name: "archlinux",
            display_name: "Arch Linux",
            description: "Rolling release Linux",
            category: "arch",
            popular: true,
        },
        {
            name: "manjaro",
            display_name: "Manjaro",
            description: "User-friendly Arch",
            category: "arch",
        },
        {
            name: "artix",
            display_name: "Artix Linux",
            description: "Arch without systemd",
            category: "arch",
        },

        // SUSE-based
        {
            name: "opensuse",
            display_name: "openSUSE Leap 15.6",
            description: "Enterprise Linux",
            category: "suse",
        },
        {
            name: "tumbleweed",
            display_name: "openSUSE Tumbleweed",
            description: "Rolling release",
            category: "suse",
        },

        // Independent
        {
            name: "gentoo",
            display_name: "Gentoo Linux",
            description: "Source-based distro",
            category: "independent",
        },
        {
            name: "void",
            display_name: "Void Linux",
            description: "Independent with runit",
            category: "independent",
        },
        {
            name: "nixos",
            display_name: "NixOS",
            description: "Declarative configuration",
            category: "independent",
        },
        {
            name: "slackware",
            display_name: "Slackware 15.0",
            description: "Classic Unix-like",
            category: "independent",
        },
        {
            name: "crux",
            display_name: "CRUX",
            description: "Lightweight, BSD-style",
            category: "independent",
        },
        {
            name: "guix",
            display_name: "Guix System",
            description: "Transactional package manager",
            category: "independent",
        },

        // Minimal / Embedded
        {
            name: "alpine",
            display_name: "Alpine 3.21",
            description: "Lightweight Linux (6MB)",
            category: "minimal",
            popular: true,
        },
        {
            name: "alpine-3.20",
            display_name: "Alpine 3.20",
            description: "Previous stable",
            category: "minimal",
        },
        {
            name: "busybox",
            display_name: "BusyBox 1.37",
            description: "Ultra-minimal (~2MB)",
            category: "minimal",
        },
        {
            name: "tinycore",
            display_name: "TinyCore",
            description: "Micro Linux (~16MB)",
            category: "minimal",
        },

        // Cloud Provider
        {
            name: "amazonlinux",
            display_name: "Amazon Linux 2023",
            description: "Optimized for AWS",
            category: "cloud",
            popular: true,
        },
        {
            name: "amazonlinux2",
            display_name: "Amazon Linux 2",
            description: "Legacy AWS (EOL 2025)",
            category: "cloud",
        },
        {
            name: "azurelinux",
            display_name: "Azure Linux",
            description: "Microsoft Cloud Linux",
            category: "cloud",
        },

        // Specialized
        {
            name: "clearlinux",
            display_name: "Clear Linux",
            description: "Intel-optimized",
            category: "specialized",
        },
        {
            name: "photon",
            display_name: "VMware Photon OS 5.0",
            description: "Container-optimized",
            category: "specialized",
        },
        {
            name: "rancheros",
            display_name: "RancherOS (Alpine)",
            description: "Container-optimized",
            category: "specialized",
        },
        {
            name: "neurodebian",
            display_name: "NeuroDebian",
            description: "Neuroscience Research",
            category: "specialized",
        },

        // Security
        {
            name: "kali",
            display_name: "Kali Linux",
            description: "Penetration testing",
            category: "security",
            popular: true,
        },
        {
            name: "parrot",
            display_name: "Parrot OS",
            description: "Security distribution",
            category: "security",
        },
        {
            name: "blackarch",
            display_name: "BlackArch",
            description: "Security distribution",
            category: "security",
        },

        // Embedded / IoT
        {
            name: "raspberrypi",
            display_name: "Raspberry Pi OS",
            description: "Debian-based for ARM",
            category: "embedded",
        },
        {
            name: "openwrt",
            display_name: "OpenWrt",
            description: "Router/Embedded OS",
            category: "embedded",
        },

        // macOS
        {
            name: "macos",
            display_name: "macOS",
            description: "Apple macOS (VM-based)",
            category: "macos",
            popular: true,
        },
    ];

    const roleToOS: Record<string, string> = {
        standard: "alpine",
        node: "ubuntu",
        python: "ubuntu",
        go: "alpine",
        neovim: "ubuntu",
        devops: "alpine",
        overemployed: "ubuntu",
    };

    $: if (selectedRole && roleToOS[selectedRole]) {
        const preferredOS = roleToOS[selectedRole];
        if (images.some((img) => img.name === preferredOS)) {
            selectedImage = preferredOS;
        }
    }

    // Helper to get role display info
    function getRoleIcon(roleId: string): string {
        const icons: Record<string, string> = {
            standard: "terminal",
            node: "nodejs",
            python: "python",
            go: "golang",
            neovim: "edit",
            devops: "devops",
            overemployed: "ai",
        };
        return icons[roleId] || "terminal";
    }

    function getRoleRecommendedOS(roleId: string): string {
        const osMap: Record<string, string> = {
            standard: "Alpine",
            node: "Ubuntu",
            python: "Ubuntu",
            go: "Alpine",
            neovim: "Ubuntu",
            devops: "Alpine",
            overemployed: "Ubuntu",
        };
        return osMap[roleId] || "Ubuntu";
    }

    $: currentRole = $roles.roles.find((r) => r.id === selectedRole);

    // Fetch roles from API on mount
    onMount(async () => {
        // Load roles from API (cached)
        roles.load();

        // Select default image based on role
        if (selectedRole && roleToOS[selectedRole]) {
            selectedImage = roleToOS[selectedRole];
        }
    });

    function selectAndCreate(imageName: string) {
        selectedImage = imageName;
        if (imageName !== "custom") {
            createContainer();
        }
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
            selectedImage === "custom" ? customImage : undefined,
            selectedRole,
            handleProgress,
            handleComplete,
            handleError,
            { memory_mb: memoryMB, cpu_shares: cpuShares, disk_mb: diskMB },
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
                        <p>
                            This may indicate an issue with the Docker host.
                            Check that:
                        </p>
                        <ul>
                            <li>
                                The Docker daemon is running on the remote host
                            </li>
                            <li>TLS certificates are properly configured</li>
                            <li>Firewall rules allow the connection</li>
                        </ul>
                    </div>
                {/if}
            </div>
            <button
                class="retry-btn"
                onclick={() => {
                    errorMessage = "";
                }}
            >
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
                <div
                    class="progress-fill"
                    style="width: {displayProgress}%"
                ></div>
            </div>

            <!-- Step Indicators -->
            <div class="progress-steps">
                {#each progressSteps as step (step.id)}
                    <div
                        class="progress-step {stepStatuses[step.id] ||
                            'pending'}"
                    >
                        <span class="step-icon"
                            ><StatusIcon status={step.icon} size={12} /></span
                        >
                        <span class="step-label">{step.label}</span>
                    </div>
                {/each}
            </div>

            <p class="progress-message">{progressMessage}</p>

            <!-- Role-specific tools being installed -->
            {#if currentRole && progressStage === "configuring"}
                <div class="installing-tools">
                    <p class="installing-label">
                        Installing tools for {currentRole.name}:
                    </p>
                    <div class="tools-installing">
                        {#each currentRole.packages || [] as tool}
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
            <h1 class="create-title">Create New Terminal</h1>
            
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
                {#if $roles.loading}
                    <div class="role-loading">Loading environments...</div>
                {:else if $roles.roles.length === 0}
                    <div class="role-loading">No environments available</div>
                {:else}
                    <div class="role-grid">
                        {#each $roles.roles as role}
                            <button
                                class="role-card"
                                class:selected={selectedRole === role.id}
                                onclick={() => (selectedRole = role.id)}
                                title={role.description || ''}
                            >
                                <PlatformIcon platform={role.id} size={28} />
                                <span class="role-name">{role.name}</span>
                            </button>
                        {/each}
                    </div>
                {/if}
                {#if currentRole}
                    <div class="role-info">
                        <div class="role-header-row">
                            <PlatformIcon platform={currentRole.id} size={18} />
                            <span class="role-name-sm">{currentRole.name}</span>
                            <span class="role-os-badge">
                                <PlatformIcon
                                    platform={getRoleRecommendedOS(currentRole.id).toLowerCase()}
                                    size={14}
                                />
                                {getRoleRecommendedOS(currentRole.id)}
                            </span>
                        </div>
                        {#if currentRole.packages && currentRole.packages.length > 0}
                            <div class="role-tools">
                                {#each currentRole.packages as tool}
                                    <span class="tool-badge">{tool}</span>
                                {/each}
                            </div>
                        {/if}
                    </div>
                {/if}
            </div>

            <!-- Resource Configuration (Trial users can customize) -->
            <div class="create-section">
                <button
                    class="resource-toggle"
                    onclick={() => (showResources = !showResources)}
                >
                    <span class="toggle-icon">{showResources ? "▼" : "▶"}</span>
                    <h4>Resources</h4>
                    <span class="resource-preview">
                        {formatMemory(memoryMB)} / {formatCPU(cpuShares)} / {formatStorage(
                            diskMB,
                        )}
                    </span>
                </button>

                {#if showResources}
                    <div class="resource-config">
                        <div class="resource-row">
                            <label>
                                <span class="resource-label">Memory</span>
                                <span class="resource-value"
                                    >{formatMemory(memoryMB)}</span
                                >
                            </label>
                            <input
                                type="range"
                                value={memoryMB}
                                oninput={handleMemoryChange}
                                min={resourceLimits.minMemory}
                                max={resourceLimits.maxMemory}
                                step="128"
                            />
                            <div class="resource-range">
                                <span
                                    >{formatMemory(
                                        resourceLimits.minMemory,
                                    )}</span
                                >
                                <span
                                    >{formatMemory(
                                        resourceLimits.maxMemory,
                                    )}</span
                                >
                            </div>
                        </div>

                        <div class="resource-row">
                            <label>
                                <span class="resource-label">CPU</span>
                                <span class="resource-value"
                                    >{formatCPU(cpuShares)}</span
                                >
                            </label>
                            <input
                                type="range"
                                value={cpuShares}
                                oninput={handleCpuChange}
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
                                <span class="resource-value"
                                    >{formatStorage(diskMB)}</span
                                >
                            </label>
                            <input
                                type="range"
                                value={diskMB}
                                oninput={handleDiskChange}
                                min={resourceLimits.minDisk}
                                max={resourceLimits.maxDisk}
                                step="256"
                            />
                            <div class="resource-range">
                                <span
                                    >{formatStorage(
                                        resourceLimits.minDisk,
                                    )}</span
                                >
                                <span
                                    >{formatStorage(
                                        resourceLimits.maxDisk,
                                    )}</span
                                >
                            </div>
                        </div>

                        {#if showUpgradeHint && nextTierLimits}
                            <div class="upgrade-prompt">
                                <div class="upgrade-icon">
                                    <StatusIcon status="bolt" size={16} />
                                </div>
                                <div class="upgrade-content">
                                    <span class="upgrade-title">Need more resources?</span>
                                    <span class="upgrade-desc">
                                        Upgrade to {nextTierName} for up to {formatMemory(nextTierLimits.maxMemory)} RAM, 
                                        {formatCPU(nextTierLimits.maxCPU)}, and {formatStorage(nextTierLimits.maxDisk)} storage.
                                    </span>
                                </div>
                                <button class="upgrade-btn" onclick={() => dispatch("upgrade")}>
                                    Upgrade
                                </button>
                            </div>
                        {:else}
                            <p class="resource-hint">
                                {#if $userTier === "enterprise"}
                                    Enterprise plan resources
                                {:else}
                                    {$userTier === "guest" ? "Guest" : $userTier === "free" ? "Free" : "Pro"} plan limits — 
                                    <button class="upgrade-link" onclick={() => dispatch("upgrade")}>upgrade for more</button>
                                {/if}
                            </p>
                        {/if}
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
                            onclick={() => selectAndCreate(image.name)}
                        >
                            <PlatformIcon platform={image.name} size={28} />
                            <span class="os-name"
                                >{image.display_name || image.name}</span
                            >
                            {#if image.popular}
                                <span class="popular-badge">Popular</span>
                            {/if}
                        </button>
                    {/each}
                    <button
                        class="os-card"
                        class:selected={selectedImage === "custom"}
                        onclick={() => selectAndCreate("custom")}
                    >
                        <PlatformIcon platform="custom" size={28} />
                        <span class="os-name">Custom</span>
                    </button>
                </div>

                {#if selectedImage === "custom"}
                    <div class="custom-image-input" transition:slide>
                        <label>Docker Image</label>
                        <div class="input-row">
                            <input
                                type="text"
                                bind:value={customImage}
                                placeholder="e.g. ubuntu:20.04"
                                onkeydown={(e) =>
                                    e.key === "Enter" &&
                                    customImage &&
                                    createContainer()}
                            />
                            <button
                                class="btn-create"
                                onclick={createContainer}
                                disabled={!customImage}
                            >
                                Create
                            </button>
                        </div>
                    </div>
                {/if}
            </div>

            <!-- Connect Your Own Machine Section -->
            <div class="create-section connect-own-section">
                <div class="section-divider">
                    <span class="divider-line"></span>
                    <span class="divider-text">OR</span>
                    <span class="divider-line"></span>
                </div>
                
                <div class="connect-own-card">
                    <div class="connect-own-header">
                        <StatusIcon status="connected" size={24} />
                        <div class="connect-own-title">
                            <h4>Connect Your Own Machine</h4>
                            <p>Turn any server, VM, or local machine into a rexec terminal</p>
                        </div>
                    </div>
                    
                    <div class="connect-methods">
                        <div class="connect-method">
                            <div class="method-header">
                                <StatusIcon status="terminal" size={16} />
                                <span class="method-title">Quick Install (One-liner)</span>
                            </div>
                            <div class="code-block">
                                <code>curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash</code>
                                <button 
                                    class="copy-btn" 
                                    onclick={() => {
                                        navigator.clipboard.writeText('curl -fsSL https://rexec.pipeops.io/install-agent.sh | bash');
                                        const btn = document.activeElement;
                                        if (btn) btn.textContent = 'Copied!';
                                        setTimeout(() => { if (btn) btn.textContent = 'Copy'; }, 2000);
                                    }}
                                    title="Copy to clipboard"
                                >
                                    Copy
                                </button>
                            </div>
                        </div>

                        <div class="connect-method">
                            <div class="method-header">
                                <StatusIcon status="wrench" size={16} />
                                <span class="method-title">Using rexec CLI</span>
                            </div>
                            <div class="code-block">
                                <code>rexec agent start --token YOUR_TOKEN</code>
                                <button 
                                    class="copy-btn" 
                                    onclick={() => {
                                        navigator.clipboard.writeText('rexec agent start --token YOUR_TOKEN');
                                        const btn = document.activeElement;
                                        if (btn) btn.textContent = 'Copied!';
                                        setTimeout(() => { if (btn) btn.textContent = 'Copy'; }, 2000);
                                    }}
                                    title="Copy to clipboard"
                                >
                                    Copy
                                </button>
                            </div>
                        </div>

                        <div class="connect-method">
                            <div class="method-header">
                                <StatusIcon status="ai" size={16} />
                                <span class="method-title">Interactive TUI</span>
                            </div>
                            <div class="code-block">
                                <code>rexec -i</code>
                                <button 
                                    class="copy-btn" 
                                    onclick={() => {
                                        navigator.clipboard.writeText('rexec -i');
                                        const btn = document.activeElement;
                                        if (btn) btn.textContent = 'Copied!';
                                        setTimeout(() => { if (btn) btn.textContent = 'Copy'; }, 2000);
                                    }}
                                    title="Copy to clipboard"
                                >
                                    Copy
                                </button>
                            </div>
                        </div>
                    </div>

                    <div class="connect-features">
                        <div class="feature-item">
                            <StatusIcon status="ready" size={14} />
                            <span>Resumable sessions with tmux</span>
                        </div>
                        <div class="feature-item">
                            <StatusIcon status="ready" size={14} />
                            <span>Works on Linux, macOS, Windows (WSL)</span>
                        </div>
                        <div class="feature-item">
                            <StatusIcon status="ready" size={14} />
                            <span>Cloud VMs, Raspberry Pi, local dev machines</span>
                        </div>
                    </div>

                    <a href="/docs/agent" class="learn-more-link">
                        <span>Learn more about the rexec agent</span>
                        <StatusIcon status="connected" size={12} />
                    </a>
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

    @keyframes spin {
        to {
            transform: rotate(360deg);
        }
    }

    /* Content */
    .create-content {
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .create-title {
        font-size: 24px;
        font-weight: 600;
        color: var(--text);
        margin: 0 0 8px 0;
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .compact .create-title {
        font-size: 18px;
        margin: 0 0 4px 0;
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

    .role-loading {
        padding: 20px;
        text-align: center;
        color: var(--text-muted);
        font-size: 12px;
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
        color: var(--accent);
    }

    .role-icon {
        font-size: 24px;
        filter: drop-shadow(0 0 4px rgba(0, 255, 65, 0.3));
    }

    .role-icon-sm {
        font-size: 16px;
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
        gap: 6px;
    }

    .extra-tools {
        margin-top: 10px;
        padding-top: 10px;
        border-top: 1px solid #333;
    }

    .extra-tools-label {
        font-size: 11px;
        color: var(--accent);
        font-weight: 500;
        display: block;
        margin-bottom: 6px;
    }

    .extra-tools-grid {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
    }

    .extra-tool-badge {
        padding: 3px 8px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.3);
        border-radius: 4px;
        font-size: 10px;
        color: var(--accent);
        cursor: help;
        transition: all 0.15s ease;
    }

    .extra-tool-badge:hover {
        background: rgba(0, 255, 65, 0.2);
        border-color: var(--accent);
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
        transition:
            transform 0.15s,
            box-shadow 0.15s;
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

    .upgrade-link {
        background: none;
        border: none;
        color: var(--accent);
        cursor: pointer;
        font-size: inherit;
        font-style: inherit;
        padding: 0;
        text-decoration: underline;
    }

    .upgrade-link:hover {
        opacity: 0.8;
    }

    .upgrade-prompt {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 14px;
        background: linear-gradient(135deg, rgba(0, 212, 255, 0.1) 0%, rgba(139, 92, 246, 0.1) 100%);
        border: 1px solid rgba(0, 212, 255, 0.3);
        border-radius: 8px;
        margin-top: 12px;
    }

    .upgrade-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 32px;
        height: 32px;
        background: rgba(0, 212, 255, 0.2);
        border-radius: 6px;
        color: #00d4ff;
        flex-shrink: 0;
    }

    .upgrade-content {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .upgrade-title {
        font-size: 12px;
        font-weight: 600;
        color: var(--text);
    }

    .upgrade-desc {
        font-size: 11px;
        color: var(--text-secondary);
    }

    .upgrade-btn {
        padding: 8px 16px;
        background: linear-gradient(135deg, #00d4ff 0%, #8b5cf6 100%);
        border: none;
        border-radius: 6px;
        color: #fff;
        font-size: 12px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
        flex-shrink: 0;
    }

    .upgrade-btn:hover {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(0, 212, 255, 0.3);
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

    /* Custom Image Input */
    .custom-image-input {
        margin-top: 12px;
        padding: 12px;
        background: #1a1a1a;
        border: 1px solid #333;
        border-radius: 6px;
    }

    .custom-image-input label {
        display: block;
        margin-bottom: 8px;
        font-size: 12px;
        color: var(--accent);
        font-weight: 500;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .input-row {
        display: flex;
        gap: 8px;
    }

    .input-row input {
        flex: 1;
        padding: 8px 12px;
        background: #111;
        border: 1px solid #333;
        border-radius: 4px;
        color: var(--text);
        font-family: var(--font-mono);
        font-size: 13px;
    }

    .input-row input:focus {
        outline: none;
        border-color: var(--accent);
    }

    .btn-create {
        padding: 0 16px;
        background: var(--accent);
        color: #000;
        border: none;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        cursor: pointer;
        transition: all 0.15s;
    }

    .btn-create:hover:not(:disabled) {
        filter: brightness(1.1);
        transform: translateY(-1px);
    }

    .btn-create:disabled {
        opacity: 0.5;
        cursor: not-allowed;
        background: #333;
        color: #a0a0a0;
    }

    .os-card.selected {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
        box-shadow: 0 0 8px rgba(0, 255, 65, 0.2);
        color: var(--accent);
    }

    /* Connect Your Own Machine Section */
    .connect-own-section {
        margin-top: 24px;
    }

    .section-divider {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 20px;
    }

    .divider-line {
        flex: 1;
        height: 1px;
        background: linear-gradient(90deg, transparent, #333, transparent);
    }

    .divider-text {
        font-size: 12px;
        color: var(--text-muted);
        text-transform: uppercase;
        letter-spacing: 2px;
        font-weight: 500;
    }

    .connect-own-card {
        background: linear-gradient(135deg, #0d1117 0%, #161b22 100%);
        border: 1px solid #30363d;
        border-radius: 12px;
        padding: 24px;
        position: relative;
        overflow: hidden;
    }

    .connect-own-card::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 2px;
        background: linear-gradient(90deg, var(--accent), #00d4ff, var(--accent));
        opacity: 0.6;
    }

    .connect-own-header {
        display: flex;
        align-items: flex-start;
        gap: 16px;
        margin-bottom: 20px;
    }

    .connect-own-header :global(svg) {
        color: var(--accent);
        flex-shrink: 0;
        margin-top: 2px;
    }

    .connect-own-title h4 {
        margin: 0 0 4px 0;
        font-size: 16px;
        color: var(--text);
        font-weight: 600;
    }

    .connect-own-title p {
        margin: 0;
        font-size: 13px;
        color: var(--text-muted);
    }

    .connect-methods {
        display: flex;
        flex-direction: column;
        gap: 12px;
        margin-bottom: 20px;
    }

    .connect-method {
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid #30363d;
        border-radius: 8px;
        padding: 12px;
    }

    .method-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;
    }

    .method-header :global(svg) {
        color: var(--text-muted);
    }

    .method-title {
        font-size: 12px;
        color: var(--text-muted);
        font-weight: 500;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .code-block {
        display: flex;
        align-items: center;
        gap: 8px;
        background: #0d0d0d;
        border: 1px solid #222;
        border-radius: 6px;
        padding: 8px 12px;
        overflow-x: auto;
    }

    .code-block code {
        flex: 1;
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--accent);
        white-space: nowrap;
    }

    .copy-btn {
        flex-shrink: 0;
        padding: 4px 10px;
        background: transparent;
        border: 1px solid #444;
        border-radius: 4px;
        color: var(--text-muted);
        font-size: 11px;
        font-family: var(--font-mono);
        cursor: pointer;
        transition: all 0.15s ease;
    }

    .copy-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
    }

    .connect-features {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        margin-bottom: 16px;
        padding-top: 16px;
        border-top: 1px solid #222;
    }

    .feature-item {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        color: var(--text-muted);
    }

    .feature-item :global(svg) {
        color: var(--accent);
    }

    .learn-more-link {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        color: var(--accent);
        text-decoration: none;
        transition: all 0.15s ease;
    }

    .learn-more-link:hover {
        text-decoration: underline;
        filter: brightness(1.2);
    }

    .learn-more-link :global(svg) {
        transition: transform 0.15s ease;
    }

    .learn-more-link:hover :global(svg) {
        transform: translateX(2px);
    }

    /* Responsive adjustments for connect section */
    @media (max-width: 600px) {
        .connect-own-card {
            padding: 16px;
        }

        .connect-own-header {
            flex-direction: column;
            gap: 12px;
        }

        .connect-features {
            flex-direction: column;
            gap: 8px;
        }

        .code-block {
            flex-direction: column;
            align-items: stretch;
            gap: 8px;
        }

        .code-block code {
            overflow-x: auto;
            padding-bottom: 4px;
        }

        .copy-btn {
            align-self: flex-end;
        }
    }
</style>
