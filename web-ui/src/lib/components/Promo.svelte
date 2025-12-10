<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import { auth } from "$stores/auth";
    import { toast } from "$stores/toast";
    import StatusIcon from "./icons/StatusIcon.svelte";

    const dispatch = createEventDispatcher<{
        guest: void;
        navigate: { view: string };
    }>();

    let isOAuthLoading = false;
    let animatedStats = { terminals: 0, uptime: 0, countries: 0 };
    let visibleSections: Set<string> = new Set();
    let heroLoaded = false;
    let terminalTyping = false;
    let typedLines: number[] = [];
    let currentEraIndex = 0;

    // Realistic early-stage stats
    const targetStats = { terminals: 150, uptime: 99.5, countries: 8 };
    
    // Rotating era words
    const eraWords = ["AI", "Cloud", "DevOps", "Remote"];
    const eraColors = ["#00ff88", "#00d4ff", "#ff6b6b", "#ffd93d"];

    function handleGuestClick() {
        dispatch("guest");
    }

    function navigateTo(view: string) {
        dispatch("navigate", { view });
    }

    async function handleOAuthLogin() {
        if (isOAuthLoading) return;
        isOAuthLoading = true;
        try {
            const url = await auth.getOAuthUrl();
            if (url) {
                window.location.href = url;
            } else {
                toast.error("Unable to connect. Please try again later.");
                isOAuthLoading = false;
            }
        } catch (e) {
            toast.error("Failed to connect. Please try again.");
            isOAuthLoading = false;
        }
    }

    // 3D Tilt effect for comparison cards
    function handleCardTilt(event: MouseEvent) {
        const cardEl = event.currentTarget as HTMLElement;
        const rect = cardEl.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;
        const centerX = rect.width / 2;
        const centerY = rect.height / 2;
        
        const rotateX = ((y - centerY) / centerY) * -12;
        const rotateY = ((x - centerX) / centerX) * 12;
        
        cardEl.style.transform = `perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) scale(1.02)`;
    }

    function handleCardReset(event: MouseEvent) {
        const cardEl = event.currentTarget as HTMLElement;
        cardEl.style.transform = 'perspective(1000px) rotateX(0deg) rotateY(0deg) scale(1)';
    }

    // 3D effect for hero terminal - tracks mouse across entire hero section
    let terminalEl: HTMLElement | null = null;
    
    function handleTerminalMouseMove(event: MouseEvent) {
        if (!terminalEl) return;
        const rect = terminalEl.getBoundingClientRect();
        const centerX = rect.left + rect.width / 2;
        const centerY = rect.top + rect.height / 2;
        
        // Calculate distance from center of terminal
        const deltaX = event.clientX - centerX;
        const deltaY = event.clientY - centerY;
        
        // Limit rotation based on distance (max ~15 degrees)
        const maxRotation = 15;
        const maxDistance = 500;
        
        const rotateY = (deltaX / maxDistance) * maxRotation;
        const rotateX = -(deltaY / maxDistance) * maxRotation;
        
        // Clamp values
        const clampedRotateX = Math.max(-maxRotation, Math.min(maxRotation, rotateX));
        const clampedRotateY = Math.max(-maxRotation, Math.min(maxRotation, rotateY));
        
        // Add subtle translation for depth
        const translateX = (deltaX / maxDistance) * 10;
        const translateY = (deltaY / maxDistance) * 10;
        
        terminalEl.style.transform = `
            perspective(1500px) 
            rotateX(${clampedRotateX}deg) 
            rotateY(${clampedRotateY}deg) 
            translateX(${translateX}px) 
            translateY(${translateY}px)
            scale(1.02)
        `;
        
        // Add dynamic shadow based on tilt
        const shadowX = -rotateY * 2;
        const shadowY = rotateX * 2;
        terminalEl.style.boxShadow = `
            ${shadowX}px ${shadowY}px 60px rgba(0, 255, 136, 0.15),
            0 25px 80px rgba(0, 0, 0, 0.5)
        `;
    }
    
    function handleTerminalMouseLeave() {
        if (!terminalEl) return;
        terminalEl.style.transform = 'perspective(1500px) rotateX(0deg) rotateY(0deg) translateX(0) translateY(0) scale(1)';
        terminalEl.style.boxShadow = '0 25px 80px rgba(0, 0, 0, 0.4)';
    }

    onMount(() => {
        // Trigger hero animations after mount
        setTimeout(() => {
            heroLoaded = true;
        }, 100);

        // Start terminal typing animation
        setTimeout(() => {
            terminalTyping = true;
            // Type each line with delay
            const lines = [0, 1, 2];
            lines.forEach((line, i) => {
                setTimeout(() => {
                    typedLines = [...typedLines, line];
                }, i * 800);
            });
        }, 500);

        // Animate stats on mount
        const duration = 2000;
        const steps = 60;
        const interval = duration / steps;
        let step = 0;

        const timer = setInterval(() => {
            step++;
            const progress = step / steps;
            const eased = 1 - Math.pow(1 - progress, 3);
            
            animatedStats = {
                terminals: Math.round(targetStats.terminals * eased),
                uptime: Math.round(targetStats.uptime * eased * 10) / 10,
                countries: Math.round(targetStats.countries * eased)
            };

            if (step >= steps) clearInterval(timer);
        }, interval);

        // Rotate era words
        const eraTimer = setInterval(() => {
            currentEraIndex = (currentEraIndex + 1) % eraWords.length;
        }, 3000);

        // Intersection observer for scroll animations
        const observer = new IntersectionObserver(
            (entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        visibleSections.add(entry.target.id);
                        visibleSections = visibleSections;
                    }
                });
            },
            { threshold: 0.1 }
        );

        document.querySelectorAll('.promo-section').forEach(section => {
            observer.observe(section);
        });

        return () => {
            clearInterval(timer);
            clearInterval(eraTimer);
            observer.disconnect();
        };
    });

    const useCases = [
        {
            icon: "terminal",
            title: "Remote Development",
            description: "Full Linux environment accessible from anywhere. Code, compile, and deploy without local setup.",
            color: "#00ff88",
            slug: "remote-development"
        },
        {
            icon: "connected",
            title: "Team Collaboration",
            description: "Share your terminal session in real-time. Pair program, debug together, teach and learn.",
            color: "#00d4ff",
            slug: "pair-programming"
        },
        {
            icon: "bolt",
            title: "Instant Demos",
            description: "Spin up isolated environments for product demos, tutorials, or customer support in seconds.",
            color: "#ff6b6b",
            slug: "technical-interviews"
        },
        {
            icon: "key",
            title: "Secure Access",
            description: "Connect to your infrastructure through SSH with enterprise-grade security and audit trails.",
            color: "#ffd93d",
            slug: "universal-jump-host"
        },
        {
            icon: "globe",
            title: "Education & Training",
            description: "Perfect for coding bootcamps, workshops, and tutorials. Students get instant environments.",
            color: "#8b5cf6",
            slug: "education"
        },
        {
            icon: "shield",
            title: "Security Research",
            description: "Isolated sandboxes for malware analysis, penetration testing, and security audits.",
            color: "#f97316",
            slug: "security-research"
        },
        {
            icon: "clock",
            title: "Session Recording",
            description: "Record and replay terminal sessions for documentation, audits, or training materials.",
            color: "#ec4899",
            slug: "session-recording"
        },
        {
            icon: "ai",
            title: "AI-Powered Workflows",
            description: "Built-in AI tools for code generation, debugging assistance, and intelligent automation.",
            color: "#06b6d4",
            slug: "ai-workflows"
        }
    ];

    const features = [
        { icon: "bolt", title: "Sub-second Launch", desc: "Terminals ready in under 500ms" },
        { icon: "shield", title: "Isolated Containers", desc: "Every session is completely sandboxed" },
        { icon: "globe", title: "Browser-based", desc: "No downloads, works everywhere" },
        { icon: "clock", title: "Session Recording", desc: "Replay and share your work" },
        { icon: "ai", title: "AI Tools Built-in", desc: "Claude, GPT, and more pre-installed" },
        { icon: "link", title: "SSH Support", desc: "Connect with your favorite client" },
        { icon: "key", title: "Secure by Default", desc: "End-to-end encryption" },
        { icon: "connected", title: "Real-time Collab", desc: "Share sessions instantly" },
        { icon: "terminal", title: "Multiple OS", desc: "Ubuntu, Debian, Alpine & more" },
        { icon: "copy", title: "Port Forwarding", desc: "Expose your apps publicly" },
        { icon: "snippet", title: "Snippets & Macros", desc: "Save and reuse commands" },
        { icon: "chart", title: "Resource Monitoring", desc: "Track CPU, memory, network" }
    ];

    const moreFeatures = [
        { 
            title: "Instant Environment Provisioning",
            desc: "No more waiting for VMs. Spin up fully-configured Linux environments in milliseconds, not minutes.",
            icon: "bolt"
        },
        { 
            title: "Zero Configuration Required",
            desc: "Everything works out of the box. Pre-installed dev tools, languages, and utilities ready to use.",
            icon: "check"
        },
        { 
            title: "Access From Any Device",
            desc: "Work from your laptop, tablet, or phone. Your terminal follows you everywhere with full functionality.",
            icon: "globe"
        },
        { 
            title: "Enterprise-Grade Security",
            desc: "Container isolation, encrypted connections, and audit trails. Your code never touches our servers.",
            icon: "shield"
        },
        {
            title: "Seamless Team Onboarding",
            desc: "New developers productive in minutes. Share standardized environments with your entire team.",
            icon: "connected"
        },
        {
            title: "Cost-Effective Scaling",
            desc: "Pay only for what you use. No idle VMs, no wasted resources. Scale from 1 to 1000 terminals instantly.",
            icon: "chart"
        }
    ];

    const testimonials = [
        {
            quote: "Finally, a terminal I can access from my iPad without any compromises.",
            author: "Sarah Chen",
            role: "DevOps Lead, TechCorp"
        },
        {
            quote: "We use Rexec for all our customer demos. It's transformed our sales process.",
            author: "Marcus Johnson",
            role: "Solutions Architect"
        },
        {
            quote: "The collaboration feature saved us hours of back-and-forth debugging.",
            author: "Elena Popov",
            role: "Senior Developer"
        }
    ];
</script>

<div class="promo">
    <!-- Hero Section - Full Width -->
    <section class="hero" class:loaded={heroLoaded} onmousemove={handleTerminalMouseMove} onmouseleave={handleTerminalMouseLeave}>
        <div class="hero-bg">
            <div class="grid-lines"></div>
            <div class="glow glow-1"></div>
            <div class="glow glow-2"></div>
            <div class="glow glow-3"></div>
            <div class="particles">
                {#each Array(20) as _, i}
                    <div class="particle" style="--delay: {i * 0.5}s; --x: {Math.random() * 100}%; --duration: {5 + Math.random() * 10}s"></div>
                {/each}
            </div>
        </div>


        
        <div class="hero-inner">
            <div class="hero-content">
                <div class="hero-badge animate-fade-up" style="--delay: 0.1s">
                    <span class="pulse"></span>
                    <span>Now Available • Free Tier Included</span>
                </div>

                <h1 class="animate-fade-up" style="--delay: 0.2s">
                    The Terminal
                    <span class="gradient-text">Reimagined</span>
                    for the <span class="era-word" style="--era-color: {eraColors[currentEraIndex]}">{eraWords[currentEraIndex]}</span> Era
                </h1>

                <p class="hero-description animate-fade-up" style="--delay: 0.3s">
                    Instant Linux environments in your browser. No setup. No downloads. 
                    Just powerful, isolated terminals ready when you are.
                </p>

                <div class="hero-actions animate-fade-up" style="--delay: 0.4s">
                    <button class="btn-hero btn-primary-hero" onclick={handleGuestClick}>
                        <span class="btn-icon">▶</span>
                        Start Free Terminal
                        <span class="btn-shine"></span>
                    </button>
                    <button class="btn-hero btn-secondary-hero" onclick={handleOAuthLogin} disabled={isOAuthLoading}>
                        {#if isOAuthLoading}
                            <span class="spinner"></span>
                        {:else}
                            Sign in with PipeOps
                        {/if}
                    </button>
                </div>


            </div>

            <div class="hero-terminal animate-fade-up" style="--delay: 0.6s" bind:this={terminalEl}>
                <div class="terminal-window">
                    <div class="terminal-header">
                        <div class="terminal-buttons">
                            <span class="tb tb-red"></span>
                            <span class="tb tb-yellow"></span>
                            <span class="tb tb-green"></span>
                        </div>
                        <span class="terminal-title">rexec — ubuntu-24.04</span>
                        <div class="terminal-actions">
                            <span class="action-icon">⬚</span>
                        </div>
                    </div>
                    <div class="terminal-body">
                        <div class="line" class:typed={typedLines.includes(0)}>
                            <span class="prompt">~/projects$</span>
                            <span class="command">git clone https://github.com/my-app</span>
                        </div>
                        {#if typedLines.includes(0)}
                            <div class="output animate-line">Cloning into 'my-app'...</div>
                            <div class="output dim animate-line" style="--delay: 0.1s">remote: Enumerating objects: 1247, done.</div>
                            <div class="output dim animate-line" style="--delay: 0.2s">remote: Total 1247 (delta 0), reused 0 (delta 0)</div>
                            <div class="output success animate-line" style="--delay: 0.3s">✓ Clone complete</div>
                        {/if}
                        <div class="line" class:typed={typedLines.includes(1)}>
                            <span class="prompt">~/projects$</span>
                            <span class="command">cd my-app && npm install</span>
                        </div>
                        {#if typedLines.includes(1)}
                            <div class="output animate-line">added 847 packages in 12s</div>
                        {/if}
                        <div class="line" class:typed={typedLines.includes(2)}>
                            <span class="prompt">~/projects/my-app$</span>
                            <span class="command">npm run dev</span>
                        </div>
                        {#if typedLines.includes(2)}
                            <div class="output accent animate-line">▶ Server running at http://localhost:3000</div>
                        {/if}
                        <div class="line">
                            <span class="prompt">~/projects/my-app$</span>
                            <span class="cursor">_</span>
                        </div>
                    </div>
                </div>
                <div class="terminal-reflection"></div>
            </div>
        </div>

        <div class="scroll-indicator animate-fade-up" style="--delay: 1s">
            <span class="scroll-text">Scroll to explore</span>
            <div class="scroll-arrow">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 5v14M19 12l-7 7-7-7"/>
                </svg>
            </div>
        </div>
    </section>

    <!-- Problem/Solution Section -->
    <section id="problem" class="promo-section problem-section" class:visible={visibleSections.has('problem')}>
        <div class="section-content">
            <h2 class="section-title">The Old Way vs <span class="accent">The Rexec Way</span></h2>
            
            <div class="comparison">
                <div 
                    class="comparison-card old-way tilt-card"
                    onmousemove={handleCardTilt}
                    onmouseleave={handleCardReset}
                    role="article"
                >
                    <div class="card-glow"></div>
                    <div class="comparison-header">
                        <span class="comparison-icon error-icon">
                            <StatusIcon status="error" size={24} />
                        </span>
                        <h3>Without Rexec</h3>
                    </div>
                    <ul>
                        <li>Hours setting up local environments</li>
                        <li>Works on my machine syndrome</li>
                        <li>Sharing code via screenshots</li>
                        <li>VPN headaches for remote access</li>
                        <li>Configuration drift across team</li>
                    </ul>
                </div>
                
                <div 
                    class="comparison-card new-way tilt-card"
                    onmousemove={handleCardTilt}
                    onmouseleave={handleCardReset}
                    role="article"
                >
                    <div class="card-glow"></div>
                    <div class="comparison-header">
                        <span class="comparison-icon success-icon">
                            <StatusIcon status="check" size={24} />
                        </span>
                        <h3>With Rexec</h3>
                    </div>
                    <ul>
                        <li>Terminal ready in milliseconds</li>
                        <li>Identical environments everywhere</li>
                        <li>Real-time collaboration built-in</li>
                        <li>Secure browser access anywhere</li>
                        <li>Consistent tooling for everyone</li>
                    </ul>
                </div>
            </div>
        </div>
    </section>

    <!-- Use Cases Section -->
    <section id="usecases" class="promo-section usecases-section" class:visible={visibleSections.has('usecases')}>
        <div class="section-content">
            <h2 class="section-title">Built for <span class="accent">Every Workflow</span></h2>
            <p class="section-subtitle">From solo developers to enterprise teams</p>
            
            <div class="usecases-grid">
                {#each useCases as useCase, i}
                    <div class="usecase-card" style="--accent-color: {useCase.color}" style:animation-delay="{i * 100}ms">
                        <div class="usecase-icon" style="background: {useCase.color}15; color: {useCase.color}">
                            <StatusIcon status={useCase.icon} size={32} />
                        </div>
                        <h3>{useCase.title}</h3>
                        <p>{useCase.description}</p>
                        <button class="usecase-link" onclick={() => navigateTo(`use-cases/${useCase.slug}`)}>
                            Learn more →
                        </button>
                    </div>
                {/each}
            </div>
        </div>
    </section>

    <!-- Features Grid -->
    <section id="features" class="promo-section features-section" class:visible={visibleSections.has('features')}>
        <div class="section-content">
            <h2 class="section-title">Everything You Need, <span class="accent">Nothing You Don't</span></h2>
            
            <div class="features-grid">
                {#each features as feature, i}
                    <div 
                        class="feature-card" 
                        style:animation-delay="{i * 50}ms"
                        role="article"
                    >
                        <span class="feature-icon">
                            <StatusIcon status={feature.icon} size={28} />
                        </span>
                        <h4>{feature.title}</h4>
                        <p>{feature.desc}</p>
                    </div>
                {/each}
            </div>
        </div>
    </section>

    <!-- Why Choose Rexec Section -->
    <section id="whychoose" class="promo-section whychoose-section" class:visible={visibleSections.has('whychoose')}>
        <div class="section-content">
            <h2 class="section-title">Why Choose <span class="accent">Rexec</span></h2>
            <p class="section-subtitle">Built by developers, for developers</p>
            
            <div class="whychoose-grid">
                {#each moreFeatures as feature, i}
                    <div class="whychoose-card" style:animation-delay="{i * 100}ms">
                        <span class="whychoose-icon">
                            <StatusIcon status={feature.icon} size={32} />
                        </span>
                        <div class="whychoose-content">
                            <h3>{feature.title}</h3>
                            <p>{feature.desc}</p>
                        </div>
                    </div>
                {/each}
            </div>
        </div>
    </section>

    <!-- Social Proof -->
    <section id="testimonials" class="promo-section testimonials-section" class:visible={visibleSections.has('testimonials')}>
        <div class="section-content">
            <h2 class="section-title">Loved by <span class="accent">Developers</span></h2>
            
            <div class="testimonials-grid">
                {#each testimonials as testimonial, i}
                    <div class="testimonial-card" style:animation-delay="{i * 100}ms">
                        <p class="testimonial-quote">"{testimonial.quote}"</p>
                        <div class="testimonial-author">
                            <div class="author-avatar">{testimonial.author.charAt(0)}</div>
                            <div class="author-info">
                                <span class="author-name">{testimonial.author}</span>
                                <span class="author-role">{testimonial.role}</span>
                            </div>
                        </div>
                    </div>
                {/each}
            </div>
        </div>
    </section>

    <!-- CTA Section -->
    <section class="cta-section">
        <div class="cta-content">
            <h2>Ready to Transform Your Workflow?</h2>
            <p>Start with our free tier. No credit card required.</p>
            
            <div class="cta-actions">
                <button class="btn-hero btn-primary-hero btn-large" onclick={handleGuestClick}>
                    <span class="btn-icon">▶</span>
                    Launch Your First Terminal
                </button>
            </div>

            <div class="cta-links">
                <button class="cta-link" onclick={() => navigateTo('pricing')}>
                    View Pricing
                </button>
                <span class="link-dot"></span>
                <button class="cta-link" onclick={() => navigateTo('guides')}>
                    Read the Docs
                </button>
                <span class="link-dot"></span>
                <button class="cta-link" onclick={() => navigateTo('use-cases')}>
                    Explore Use Cases
                </button>
            </div>
        </div>
    </section>

    <!-- Footer -->
    <footer class="promo-footer">
        <div class="footer-content">
            <div class="footer-brand">
                <svg width="24" height="24" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect width="32" height="32" rx="8" fill="#00ff88"/>
                    <path d="M8 10L14 16L8 22" stroke="#000" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
                    <path d="M16 22H24" stroke="#000" stroke-width="2.5" stroke-linecap="round"/>
                </svg>
                <span>Rexec</span>
            </div>
            <div class="footer-links">
                <a href="https://pipeops.io" target="_blank" rel="noopener">PipeOps</a>
                <a href="https://github.com/pipeops-dev" target="_blank" rel="noopener">GitHub</a>
                <a href="https://twitter.com/pabordeaux" target="_blank" rel="noopener">Twitter</a>
            </div>
            <div class="footer-copyright">
                © {new Date().getFullYear()} PipeOps. All rights reserved.
            </div>
        </div>
    </footer>
</div>

<style>
    .promo {
        min-height: 100vh;
        background: var(--bg);
        color: var(--text);
        overflow-x: hidden;
    }

    /* Hero Section - Full Screen */
    .hero {
        position: relative;
        min-height: 100vh;
        width: 100vw;
        margin-left: calc(-50vw + 50%);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 0;
        overflow: hidden;
    }

    .hero-inner {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        width: 100%;
        max-width: 1400px;
        padding: 60px 24px 80px;
        z-index: 1;
    }

    .hero-bg {
        position: absolute;
        inset: 0;
        pointer-events: none;
        overflow: hidden;
    }

    .grid-lines {
        position: absolute;
        inset: 0;
        background-image: 
            linear-gradient(rgba(0, 255, 136, 0.03) 1px, transparent 1px),
            linear-gradient(90deg, rgba(0, 255, 136, 0.03) 1px, transparent 1px);
        background-size: 60px 60px;
        animation: gridMove 20s linear infinite;
    }

    @keyframes gridMove {
        0% { transform: translate(0, 0); }
        100% { transform: translate(60px, 60px); }
    }

    .glow {
        position: absolute;
        border-radius: 50%;
        filter: blur(120px);
        opacity: 0.4;
        animation: glowPulse 8s ease-in-out infinite;
    }

    .glow-1 {
        width: 800px;
        height: 800px;
        background: var(--accent);
        top: -300px;
        right: -200px;
        animation-delay: 0s;
    }

    .glow-2 {
        width: 600px;
        height: 600px;
        background: #00d4ff;
        bottom: -200px;
        left: -200px;
        animation-delay: 2s;
    }

    .glow-3 {
        width: 400px;
        height: 400px;
        background: #8b5cf6;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        animation-delay: 4s;
    }

    @keyframes glowPulse {
        0%, 100% { opacity: 0.3; transform: scale(1); }
        50% { opacity: 0.5; transform: scale(1.1); }
    }

    .glow-3 {
        animation: glowPulse 8s ease-in-out infinite, glowFloat 15s ease-in-out infinite;
    }

    @keyframes glowFloat {
        0%, 100% { transform: translate(-50%, -50%); }
        50% { transform: translate(-40%, -60%); }
    }

    /* Particles */
    .particles {
        position: absolute;
        inset: 0;
        overflow: hidden;
    }

    .particle {
        position: absolute;
        width: 4px;
        height: 4px;
        background: var(--accent);
        border-radius: 50%;
        left: var(--x);
        bottom: -10px;
        opacity: 0.6;
        animation: particleFloat var(--duration) ease-in-out infinite;
        animation-delay: var(--delay);
    }

    @keyframes particleFloat {
        0% { transform: translateY(0) scale(1); opacity: 0; }
        10% { opacity: 0.6; }
        90% { opacity: 0.6; }
        100% { transform: translateY(-100vh) scale(0.5); opacity: 0; }
    }

    /* Animation Classes */
    .animate-fade-up {
        opacity: 0;
        transform: translateY(30px);
        transition: opacity 0.8s cubic-bezier(0.16, 1, 0.3, 1), 
                    transform 0.8s cubic-bezier(0.16, 1, 0.3, 1);
        transition-delay: var(--delay, 0s);
    }

    .hero.loaded .animate-fade-up {
        opacity: 1;
        transform: translateY(0);
    }

    .animate-line {
        animation: lineAppear 0.3s ease-out forwards;
        animation-delay: var(--delay, 0s);
        opacity: 0;
    }

    @keyframes lineAppear {
        from { opacity: 0; transform: translateX(-10px); }
        to { opacity: 1; transform: translateX(0); }
    }

    .hero-content {
        text-align: center;
        max-width: 900px;
        z-index: 1;
    }

    .hero-badge {
        display: inline-flex;
        align-items: center;
        gap: 10px;
        padding: 8px 20px;
        background: rgba(0, 255, 136, 0.1);
        border: 1px solid rgba(0, 255, 136, 0.3);
        border-radius: 100px;
        font-size: 12px;
        color: var(--accent);
        margin-bottom: 32px;
        text-transform: uppercase;
        letter-spacing: 1px;
        backdrop-filter: blur(10px);
    }

    .pulse {
        width: 8px;
        height: 8px;
        background: var(--accent);
        border-radius: 50%;
        animation: pulse 2s ease-in-out infinite;
        box-shadow: 0 0 10px var(--accent);
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; transform: scale(1); box-shadow: 0 0 10px var(--accent); }
        50% { opacity: 0.5; transform: scale(1.3); box-shadow: 0 0 20px var(--accent); }
    }

    .hero h1 {
        font-size: clamp(36px, 7vw, 72px);
        font-weight: 800;
        line-height: 1.1;
        margin-bottom: 24px;
        letter-spacing: -2px;
    }

    .gradient-text {
        background: linear-gradient(135deg, var(--accent) 0%, #00d4ff 50%, #8b5cf6 100%);
        background-size: 200% 200%;
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
        animation: gradientShift 5s ease infinite;
    }

    .era-word {
        color: var(--era-color, var(--accent));
        display: inline-block;
        animation: eraFade 0.5s ease-in-out;
        text-shadow: 0 0 30px var(--era-color);
    }

    @keyframes eraFade {
        0% { opacity: 0; transform: translateY(-10px); }
        100% { opacity: 1; transform: translateY(0); }
    }

    @keyframes gradientShift {
        0%, 100% { background-position: 0% 50%; }
        50% { background-position: 100% 50%; }
    }

    .hero-description {
        font-size: 20px;
        color: var(--text-secondary);
        max-width: 650px;
        margin: 0 auto 48px;
        line-height: 1.7;
    }

    .hero-actions {
        display: flex;
        gap: 16px;
        justify-content: center;
        flex-wrap: wrap;
        margin-bottom: 56px;
    }

    .btn-hero {
        position: relative;
        display: inline-flex;
        align-items: center;
        gap: 10px;
        padding: 18px 36px;
        font-size: 16px;
        font-weight: 600;
        border: none;
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
        overflow: hidden;
    }

    .btn-primary-hero {
        background: linear-gradient(135deg, var(--accent) 0%, #00e676 100%);
        color: #000;
        box-shadow: 0 4px 20px rgba(0, 255, 136, 0.3);
    }

    .btn-primary-hero:hover {
        transform: translateY(-3px) scale(1.02);
        box-shadow: 0 12px 40px rgba(0, 255, 136, 0.4);
    }

    .btn-shine {
        position: absolute;
        top: 0;
        left: -100%;
        width: 100%;
        height: 100%;
        background: linear-gradient(90deg, transparent, rgba(255,255,255,0.3), transparent);
        animation: shine 3s ease-in-out infinite;
    }

    @keyframes shine {
        0% { left: -100%; }
        50%, 100% { left: 100%; }
    }

    .btn-secondary-hero {
        background: rgba(255, 255, 255, 0.05);
        color: var(--text);
        border: 1px solid rgba(255, 255, 255, 0.1);
        backdrop-filter: blur(10px);
    }

    .btn-secondary-hero:hover {
        border-color: var(--accent);
        background: rgba(0, 255, 136, 0.1);
        transform: translateY(-2px);
    }

    .btn-icon {
        font-size: 12px;
    }

    .btn-large {
        padding: 22px 52px;
        font-size: 18px;
    }

    .spinner {
        width: 18px;
        height: 18px;
        border: 2px solid transparent;
        border-top-color: currentColor;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    .hero-stats {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 40px;
        flex-wrap: wrap;
        padding: 24px 32px;
        background: rgba(0, 0, 0, 0.3);
        border: 1px solid rgba(255, 255, 255, 0.05);
        border-radius: 16px;
        backdrop-filter: blur(10px);
    }

    .stat {
        text-align: center;
    }

    .stat-value {
        display: block;
        font-size: 32px;
        font-weight: 700;
        color: var(--accent);
        font-variant-numeric: tabular-nums;
    }

    .stat-label {
        font-size: 12px;
        color: var(--text-muted);
        text-transform: uppercase;
        letter-spacing: 1px;
    }

    .stat-divider {
        width: 1px;
        height: 48px;
        background: linear-gradient(180deg, transparent, var(--border), transparent);
    }

    /* Terminal Window */
    .hero-terminal {
        margin-top: 64px;
        width: 100%;
        max-width: 800px;
        z-index: 1;
        position: relative;
        transform-style: preserve-3d;
        transition: transform 0.1s ease-out, box-shadow 0.1s ease-out;
        will-change: transform;
        border-radius: 16px;
    }

    .terminal-window {
        background: rgba(10, 10, 10, 0.9);
        border: 1px solid rgba(255, 255, 255, 0.1);
        border-radius: 16px;
        overflow: hidden;
        box-shadow: 
            0 25px 100px rgba(0, 0, 0, 0.5),
            0 0 0 1px rgba(255, 255, 255, 0.05) inset;
        backdrop-filter: blur(10px);
        transform-style: preserve-3d;
    }

    .terminal-reflection {
        position: absolute;
        bottom: -60%;
        left: 5%;
        right: 5%;
        height: 60%;
        background: linear-gradient(180deg, rgba(10, 10, 10, 0.3) 0%, transparent 100%);
        transform: scaleY(-1);
        opacity: 0.3;
        filter: blur(2px);
        pointer-events: none;
    }

    .terminal-header {
        display: flex;
        align-items: center;
        padding: 14px 18px;
        background: rgba(17, 17, 17, 0.8);
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    }

    .terminal-buttons {
        display: flex;
        gap: 8px;
    }

    .tb {
        width: 12px;
        height: 12px;
        border-radius: 50%;
        transition: transform 0.2s;
    }

    .tb:hover {
        transform: scale(1.2);
    }

    .tb-red { background: #ff5f56; }
    .tb-yellow { background: #ffbd2e; }
    .tb-green { background: #27c93f; }

    .terminal-title {
        flex: 1;
        text-align: center;
        font-size: 13px;
        color: var(--text-muted);
        font-family: var(--font-mono);
    }

    .action-icon {
        color: var(--text-muted);
        font-size: 14px;
    }

    .terminal-body {
        padding: 24px;
        font-family: var(--font-mono);
        font-size: 14px;
        line-height: 2;
        min-height: 280px;
    }

    .line {
        display: flex;
        gap: 8px;
        opacity: 0.4;
        transition: opacity 0.3s;
    }

    .line.typed {
        opacity: 1;
    }

    .prompt {
        color: var(--accent);
    }

    .command {
        color: var(--text);
    }

    .output {
        color: var(--text-secondary);
        padding-left: 0;
    }

    .output.dim {
        color: var(--text-muted);
    }

    .output.success {
        color: var(--accent);
    }

    .output.accent {
        color: #00d4ff;
    }

    .cursor {
        display: inline-block;
        background: var(--accent);
        color: #000;
        animation: blink 1s step-end infinite;
        padding: 0 2px;
    }

    @keyframes blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0; }
    }

    /* Scroll Indicator */
    .scroll-indicator {
        position: absolute;
        bottom: 40px;
        left: 50%;
        transform: translateX(-50%);
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        color: var(--text-muted);
    }

    .scroll-text {
        font-size: 12px;
        text-transform: uppercase;
        letter-spacing: 2px;
    }

    .scroll-arrow {
        animation: scrollBounce 2s ease-in-out infinite;
    }

    @keyframes scrollBounce {
        0%, 100% { transform: translateY(0); }
        50% { transform: translateY(8px); }
    }

    /* Sections */
    .promo-section {
        padding: 120px 24px;
        opacity: 0;
        transform: translateY(40px);
        transition: opacity 0.6s, transform 0.6s;
    }

    .promo-section.visible {
        opacity: 1;
        transform: translateY(0);
    }

    .section-content {
        max-width: 1100px;
        margin: 0 auto;
    }

    .section-title {
        font-size: clamp(28px, 4vw, 42px);
        font-weight: 700;
        text-align: center;
        margin-bottom: 16px;
    }

    .section-title .accent {
        color: var(--accent);
    }

    .section-subtitle {
        font-size: 16px;
        color: var(--text-secondary);
        text-align: center;
        margin-bottom: 60px;
    }

    /* Problem Section */
    .problem-section {
        background: linear-gradient(180deg, transparent 0%, rgba(0, 255, 136, 0.02) 100%);
    }

    .comparison {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        gap: 24px;
        margin-top: 48px;
        perspective: 1000px;
    }

    .comparison-card {
        padding: 32px;
        border-radius: 12px;
        border: 1px solid var(--border);
        position: relative;
        overflow: hidden;
    }

    /* 3D Tilt Card Effect */
    .tilt-card {
        transform-style: preserve-3d;
        transition: transform 0.1s ease-out, box-shadow 0.3s ease;
        cursor: pointer;
    }

    .tilt-card:hover {
        box-shadow: 
            0 25px 50px rgba(0, 0, 0, 0.5),
            0 0 30px rgba(0, 255, 136, 0.1);
    }

    .tilt-card .card-glow {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: radial-gradient(
            circle at 50% 50%,
            rgba(255, 255, 255, 0.05) 0%,
            transparent 70%
        );
        opacity: 0;
        transition: opacity 0.3s ease;
        pointer-events: none;
    }

    .tilt-card:hover .card-glow {
        opacity: 1;
    }

    .old-way {
        background: rgba(255, 100, 100, 0.05);
        border-color: rgba(255, 100, 100, 0.2);
    }

    .old-way:hover {
        border-color: rgba(255, 100, 100, 0.5);
        box-shadow: 
            0 25px 50px rgba(0, 0, 0, 0.5),
            0 0 30px rgba(255, 100, 100, 0.15);
    }

    .new-way {
        background: rgba(0, 255, 136, 0.05);
        border-color: rgba(0, 255, 136, 0.2);
    }

    .new-way:hover {
        border-color: rgba(0, 255, 136, 0.5);
    }

    .comparison-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 24px;
    }

    .comparison-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 36px;
        height: 36px;
        border-radius: 50%;
    }

    .comparison-icon.error-icon {
        background: rgba(255, 100, 100, 0.15);
        color: #ff6b6b;
    }

    .comparison-icon.success-icon {
        background: rgba(0, 255, 136, 0.15);
        color: var(--accent);
    }

    .comparison-card h3 {
        font-size: 18px;
        font-weight: 600;
    }

    .comparison-card ul {
        list-style: none;
        padding: 0;
        margin: 0;
    }

    .comparison-card li {
        padding: 12px 0;
        border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        color: var(--text-secondary);
        font-size: 14px;
    }

    .comparison-card li:last-child {
        border-bottom: none;
    }

    /* Use Cases Grid */
    .usecases-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
        gap: 24px;
    }

    .usecase-card {
        padding: 32px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
        transition: all 0.3s;
    }

    .usecase-card:hover {
        transform: translateY(-4px);
        border-color: var(--accent-color, var(--accent));
        box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
    }

    .usecase-icon {
        width: 56px;
        height: 56px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 255, 136, 0.1);
        border-radius: 12px;
        margin-bottom: 20px;
        color: var(--accent);
    }

    .usecase-card h3 {
        font-size: 18px;
        font-weight: 600;
        margin-bottom: 12px;
        color: var(--text);
    }

    .usecase-card p {
        font-size: 14px;
        color: var(--text-secondary);
        line-height: 1.6;
        margin-bottom: 16px;
    }

    .usecase-link {
        background: none;
        border: none;
        color: var(--accent);
        font-size: 13px;
        cursor: pointer;
        padding: 0;
        transition: gap 0.2s;
        display: inline-flex;
        gap: 4px;
    }

    .usecase-link:hover {
        gap: 8px;
    }

    /* Features Grid */
    .features-section {
        background: #050505;
    }

    .features-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
        gap: 16px;
    }

    .feature-card {
        padding: 24px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        text-align: center;
        transition: all 0.3s ease;
        position: relative;
        overflow: hidden;
    }

    .feature-card:hover {
        border-color: var(--accent);
        transform: translateY(-4px);
        box-shadow: 0 10px 30px rgba(0, 255, 136, 0.1);
    }

    .feature-card:hover .feature-icon {
        transform: scale(1.15) rotate(5deg);
        background: rgba(0, 255, 136, 0.2);
        box-shadow: 0 0 20px rgba(0, 255, 136, 0.3);
    }

    .feature-card:hover .feature-icon :global(svg) {
        filter: drop-shadow(0 0 8px rgba(0, 255, 136, 0.6));
    }

    .feature-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 48px;
        height: 48px;
        margin: 0 auto 12px;
        background: rgba(0, 255, 136, 0.1);
        border-radius: 10px;
        color: var(--accent);
        transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    }

    .feature-icon :global(svg) {
        transition: filter 0.3s ease;
    }

    .feature-card h4 {
        font-size: 14px;
        font-weight: 600;
        margin-bottom: 8px;
        color: var(--text);
    }

    .feature-card p {
        font-size: 12px;
        color: var(--text-muted);
    }

    /* Why Choose Section */
    .whychoose-section {
        background: linear-gradient(180deg, #050505 0%, rgba(0, 255, 136, 0.03) 100%);
    }

    .whychoose-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
        gap: 24px;
    }

    .whychoose-card {
        display: flex;
        gap: 20px;
        padding: 28px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
        transition: all 0.3s;
    }

    .whychoose-card:hover {
        border-color: var(--accent);
        transform: translateY(-4px);
        box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
    }

    .whychoose-icon {
        flex-shrink: 0;
        width: 56px;
        height: 56px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: rgba(0, 255, 136, 0.1);
        border-radius: 12px;
        color: var(--accent);
    }

    .whychoose-content h3 {
        font-size: 18px;
        font-weight: 600;
        margin-bottom: 8px;
        color: var(--text);
    }

    .whychoose-content p {
        font-size: 14px;
        color: var(--text-secondary);
        line-height: 1.6;
    }

    /* Testimonials */
    .testimonials-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        gap: 24px;
    }

    .testimonial-card {
        padding: 32px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 12px;
    }

    .testimonial-quote {
        font-size: 16px;
        color: var(--text);
        line-height: 1.7;
        margin-bottom: 24px;
        font-style: italic;
    }

    .testimonial-author {
        display: flex;
        align-items: center;
        gap: 12px;
    }

    .author-avatar {
        width: 40px;
        height: 40px;
        background: var(--accent);
        color: #000;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 600;
    }

    .author-info {
        display: flex;
        flex-direction: column;
    }

    .author-name {
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
    }

    .author-role {
        font-size: 12px;
        color: var(--text-muted);
    }

    /* CTA Section */
    .cta-section {
        padding: 120px 24px;
        text-align: center;
        background: linear-gradient(180deg, transparent 0%, rgba(0, 255, 136, 0.05) 100%);
    }

    .cta-content {
        max-width: 600px;
        margin: 0 auto;
    }

    .cta-section h2 {
        font-size: clamp(24px, 4vw, 36px);
        font-weight: 700;
        margin-bottom: 16px;
    }

    .cta-section p {
        font-size: 16px;
        color: var(--text-secondary);
        margin-bottom: 40px;
    }

    .cta-actions {
        margin-bottom: 32px;
    }

    .cta-links {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 16px;
        flex-wrap: wrap;
    }

    .cta-link {
        background: none;
        border: none;
        color: var(--text-secondary);
        font-size: 14px;
        cursor: pointer;
        transition: color 0.2s;
    }

    .cta-link:hover {
        color: var(--accent);
    }

    .link-dot {
        width: 4px;
        height: 4px;
        background: var(--border);
        border-radius: 50%;
    }

    /* Footer */
    .promo-footer {
        padding: 40px 24px;
        border-top: 1px solid var(--border);
    }

    .footer-content {
        max-width: 1100px;
        margin: 0 auto;
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 24px;
    }

    .footer-brand {
        display: flex;
        align-items: center;
        gap: 10px;
        font-weight: 600;
        color: var(--text);
    }

    .footer-links {
        display: flex;
        gap: 24px;
    }

    .footer-links a {
        color: var(--text-secondary);
        text-decoration: none;
        font-size: 14px;
        transition: color 0.2s;
    }

    .footer-links a:hover {
        color: var(--accent);
    }

    .footer-copyright {
        font-size: 12px;
        color: var(--text-muted);
    }

    /* Responsive */
    @media (max-width: 768px) {
        .hero-inner {
            padding: 40px 16px 60px;
        }

        .hero h1 {
            letter-spacing: -1px;
        }

        .hero-description {
            font-size: 16px;
        }

        .hero-stats {
            gap: 24px;
            padding: 20px;
        }

        .stat-value {
            font-size: 24px;
        }

        .stat-divider {
            display: none;
        }

        .hero-terminal {
            margin-top: 40px;
        }

        .terminal-body {
            padding: 16px;
            font-size: 12px;
            min-height: 200px;
        }

        .scroll-indicator {
            display: none;
        }

        .promo-section {
            padding: 80px 16px;
        }

        .comparison {
            grid-template-columns: 1fr;
        }

        .whychoose-grid {
            grid-template-columns: 1fr;
        }

        .whychoose-card {
            flex-direction: column;
            text-align: center;
        }

        .whychoose-icon {
            margin: 0 auto;
        }

        .footer-content {
            flex-direction: column;
            text-align: center;
        }

        .btn-hero {
            padding: 16px 28px;
            font-size: 14px;
        }

        .particles {
            display: none;
        }
    }

    @media (max-width: 480px) {
        .hero h1 {
            font-size: 28px;
        }

        .hero-badge {
            font-size: 10px;
            padding: 6px 14px;
        }

        .hero-actions {
            flex-direction: column;
            width: 100%;
        }

        .btn-hero {
            width: 100%;
            justify-content: center;
        }
    }
</style>
