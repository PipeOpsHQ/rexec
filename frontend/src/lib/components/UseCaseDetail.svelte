<script lang="ts">
    import { createEventDispatcher, onMount } from "svelte";
    import StatusIcon from "./icons/StatusIcon.svelte";
    
    export let slug: string = "";
    
    const dispatch = createEventDispatcher<{
        back: void;
        tryNow: void;
        navigate: { slug: string };
    }>();

    // Extended use case data with detailed content
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
            tagline: "The future is disposable. Zero drift, zero cleanup.",
            description: "Spin up a fresh, clean environment for every task, PR, or experiment. Ephemeral environments eliminate configuration drift, dependency conflicts, and the dreaded 'works on my machine' syndrome. Each session starts from a known state, ensuring reproducible results every time.",
            heroImage: "/images/use-cases/ephemeral-dev.svg",
            benefits: [
                { title: "Zero Setup Time", description: "Go from zero to coding in milliseconds. No more waiting for dependencies to install or environments to configure.", icon: "bolt" },
                { title: "Immutable Infrastructure", description: "Apply infrastructure-as-code principles to your development environment. Every session is identical and reproducible.", icon: "shield" },
                { title: "Clean State Always", description: "No more debugging environment issues. Every terminal starts fresh with exactly what you need.", icon: "connected" },
                { title: "Safe Experimentation", description: "Test dangerous scripts, experiment with system configs, or try new tools without fear of breaking your local machine.", icon: "terminal" }
            ],
            workflow: [
                { step: 1, title: "Choose Your Base", description: "Select from Ubuntu, Debian, Alpine, or bring your own Docker image." },
                { step: 2, title: "Launch Instantly", description: "Your terminal is ready in under 2 seconds with all tools pre-installed." },
                { step: 3, title: "Code & Experiment", description: "Write code, run tests, break things - it's all disposable." },
                { step: 4, title: "Dispose or Persist", description: "Close when done, or save your work to pick up later." }
            ],
            examples: [
                { title: "Testing a New Framework", description: "Want to try out a new JavaScript framework without polluting your local node_modules? Spin up a terminal, experiment freely, and dispose when done.", code: "npx create-next-app@latest my-app\ncd my-app && npm run dev" },
                { title: "Reproducing a Bug", description: "Create an isolated environment that matches production exactly to debug issues without affecting your main setup.", code: "git clone https://github.com/user/repo.git\ncd repo && docker-compose up" },
                { title: "Learning New Technologies", description: "Explore Kubernetes, Terraform, or any tool in a safe sandbox without risking your local configuration." }
            ],
            testimonial: { quote: "Rexec changed how our team approaches development. We no longer waste hours debugging environment issues.", author: "Sarah Chen", role: "Engineering Lead at TechCorp" },
            relatedUseCases: ["collaborative-intelligence", "technical-interviews", "open-source-review"]
        },
        "collaborative-intelligence": {
            title: "Collaborative Intelligence",
            icon: "ai",
            tagline: "A shared workspace for humans and AI agents.",
            description: "Let LLMs and AI agents execute code in a real, safe environment while you supervise. Rexec provides the perfect sandbox for autonomous agents to work alongside humans, with full visibility and control over their actions.",
            heroImage: "/images/use-cases/collaborative-ai.svg",
            benefits: [
                { title: "Sandboxed Execution", description: "AI agents can execute arbitrary code without risking your infrastructure. Full isolation ensures safety.", icon: "shield" },
                { title: "Human-in-the-Loop", description: "Watch AI agents work in real-time. Intervene, guide, or take over whenever needed.", icon: "ai" },
                { title: "Resumable Sessions", description: "Disconnect and reconnect anytime. Agent tasks continue running with full output history preserved.", icon: "reconnect" },
                { title: "Standardized Toolchain", description: "Consistent environment means consistent AI output. No more 'it worked on my agent's machine'.", icon: "connected" }
            ],
            workflow: [
                { step: 1, title: "Connect Your Agent", description: "Integrate with Claude, GPT, or any LLM via our simple API." },
                { step: 2, title: "Define the Task", description: "Give your agent a goal and the tools it needs." },
                { step: 3, title: "Observe & Guide", description: "Watch the agent work in real-time through the terminal." },
                { step: 4, title: "Review & Deploy", description: "Validate the results and deploy with confidence." }
            ],
            examples: [
                { title: "Code Generation & Testing", description: "Let an AI agent write code, run tests, and iterate until tests pass - all in an isolated environment.", code: "# AI Agent executes:\npython -m pytest tests/ --verbose\n# Sees failures, fixes code, re-runs" },
                { title: "Automated DevOps", description: "AI agents can safely execute infrastructure commands, with you watching every step.", code: "terraform plan\nterraform apply -auto-approve" },
                { title: "Data Pipeline Development", description: "Build and test ETL pipelines with AI assistance, validating each transformation step." }
            ],
            relatedUseCases: ["ephemeral-dev-environments", "resumable-sessions", "technical-interviews"]
        },
        "universal-jump-host": {
            title: "Secure Jump Host & Gateway",
            icon: "shield",
            tagline: "Zero-trust access to private infrastructure.",
            description: "Replace complex VPNs and bastion hosts. Rexec provides a secure, audited gateway to your private infrastructure. Enforce MFA, restrict IP access, and log every command for complete compliance and security.",
            heroImage: "/images/use-cases/jump-host.svg",
            benefits: [
                { title: "Browser-Based Access", description: "SSH into any server from any device. No client installation or key management required.", icon: "terminal" },
                { title: "Zero-Trust Security", description: "Enforce Multi-Factor Authentication (MFA) and IP Allow-listing for all connections.", icon: "shield" },
                { title: "Audit & Compliance", description: "Every session is logged. Review command history, connection times, and origin IPs.", icon: "data" },
                { title: "Private VPC Reach", description: "Connect securely to servers in private subnets without exposing public ports.", icon: "connected" }
            ],
            workflow: [
                { step: 1, title: "Deploy Agent", description: "Install the Rexec agent on your bastion or target server." },
                { step: 2, title: "Configure Security", description: "Enable MFA and set up IP whitelists for your team." },
                { step: 3, title: "Connect Securely", description: "Access the server instantly via the encrypted WebSocket tunnel." },
                { step: 4, title: "Monitor Activity", description: "Track all access via the centralized Audit Log dashboard." }
            ],
            examples: [
                { title: "Production Access", description: "Grant temporary, audited access to engineers for emergency debugging.", code: "# No SSH keys to share\n# Just RBAC-controlled access" },
                { title: "Compliance Auditing", description: "Export session logs to satisfy SOC2 or ISO 27001 requirements." },
                { title: "Third-Party Access", description: "Give contractors secure access to specific servers without VPN credentials." }
            ],
            relatedUseCases: ["rexec-agent", "remote-debugging", "hybrid-infrastructure"]
        },
        "instant-education-onboarding": {
            title: "Instant Education & Onboarding",
            icon: "book",
            tagline: "Onboard new engineers in seconds, not days.",
            description: "Provide pre-configured environments for workshops, tutorials, and new hire onboarding. Zero friction means attendees focus on learning, not configuring their machines.",
            heroImage: "/images/use-cases/education.svg",
            benefits: [
                { title: "Standardized Environments", description: "Every learner gets the exact same setup. No more 'my version is different' issues.", icon: "connected" },
                { title: "Interactive Documentation", description: "Documentation that runs. Embed live terminals in your tutorials and guides.", icon: "book" },
                { title: "Zero Friction", description: "Attendees click a link and start coding. No prerequisites, no installation steps.", icon: "bolt" },
                { title: "Focus on Learning", description: "Remove all barriers between learners and the content they need to absorb.", icon: "ai" }
            ],
            workflow: [
                { step: 1, title: "Design Your Curriculum", description: "Create the perfect environment for your learning objectives." },
                { step: 2, title: "Share Access Links", description: "Distribute unique links to each participant or use a shared classroom." },
                { step: 3, title: "Teach Interactively", description: "Watch students work, provide guidance, and share your terminal." },
                { step: 4, title: "Review Progress", description: "See completion rates and identify students who need extra help." }
            ],
            examples: [
                { title: "Coding Bootcamp", description: "Each student gets their own isolated environment with the course materials pre-loaded.", code: "# Pre-configured with:\ngit clone course-materials\nnpm install\ncode ." },
                { title: "New Hire Onboarding", description: "Day-one productivity with pre-configured access to all internal tools and repos." },
                { title: "Conference Workshop", description: "Run hands-on workshops where attendees code along without setup delays." }
            ],
            testimonial: { quote: "Our onboarding time dropped from 3 days to 30 minutes. New hires are productive from day one.", author: "Mike Rodriguez", role: "VP Engineering at StartupXYZ" },
            relatedUseCases: ["technical-interviews", "collaborative-intelligence", "ephemeral-dev-environments"]
        },
        "technical-interviews": {
            title: "Technical Interviews",
            icon: "terminal",
            tagline: "Real coding assessments in real environments.",
            description: "Conduct real-time coding interviews in a real Linux environment, not a constrained web editor. See how candidates actually work, not just whether they can pass synthetic tests.",
            heroImage: "/images/use-cases/interviews.svg",
            benefits: [
                { title: "Full Shell Access", description: "Candidates work in a real environment. Test their actual engineering skills.", icon: "terminal" },
                { title: "Multiplayer Mode", description: "Pair program with candidates. See their thought process in real-time.", icon: "connected" },
                { title: "Custom Challenges", description: "Pre-install repos, datasets, or custom challenges tailored to your role.", icon: "bolt" },
                { title: "Fair Assessment", description: "Every candidate gets the same environment. Standardized and reproducible.", icon: "shield" }
            ],
            workflow: [
                { step: 1, title: "Prepare the Challenge", description: "Set up the environment with your interview questions and test cases." },
                { step: 2, title: "Invite Candidate", description: "Share a unique link. Candidate joins with one click." },
                { step: 3, title: "Collaborate Live", description: "Watch them work, ask questions, or pair program together." },
                { step: 4, title: "Review & Record", description: "Replay the session later for team review and fair evaluation." }
            ],
            examples: [
                { title: "System Design", description: "Give candidates a real server to design and implement a distributed system.", code: "# Candidate implements:\npython server.py --port 8080\ncurl localhost:8080/health" },
                { title: "Debugging Exercise", description: "Present a broken codebase and watch how candidates diagnose and fix issues." },
                { title: "Take-Home Extension", description: "Let candidates extend their take-home project live, explaining their decisions." }
            ],
            relatedUseCases: ["instant-education-onboarding", "collaborative-intelligence", "open-source-review"]
        },
        "open-source-review": {
            title: "Open Source Review",
            icon: "connected",
            tagline: "Review PRs in isolated, disposable environments.",
            description: "Review Pull Requests by instantly spinning up the branch in a clean container. Test without polluting your local machine or risking your development environment.",
            heroImage: "/images/use-cases/open-source.svg",
            benefits: [
                { title: "One-Click PR Environments", description: "Launch any PR branch in seconds. No local git operations needed.", icon: "bolt" },
                { title: "Safe Testing", description: "Run arbitrary code from contributors without risking your machine.", icon: "shield" },
                { title: "No Dependency Conflicts", description: "Each PR gets its own isolated environment with the right dependencies.", icon: "connected" },
                { title: "Instant Disposal", description: "Close the terminal and it's gone. No cleanup required.", icon: "terminal" }
            ],
            workflow: [
                { step: 1, title: "Link Your Repo", description: "Connect your GitHub repository to Rexec." },
                { step: 2, title: "Click on PR", description: "One click to spin up a terminal with the PR branch checked out." },
                { step: 3, title: "Test Thoroughly", description: "Run tests, try the feature, check for security issues." },
                { step: 4, title: "Approve or Request Changes", description: "Make your review decision with full confidence." }
            ],
            examples: [
                { title: "Security Review", description: "Safely execute untrusted code from external contributors in a sandboxed environment.", code: "git checkout pr-branch\nnpm install && npm test\nnpm audit" },
                { title: "Performance Testing", description: "Benchmark PR changes against main branch in identical environments." },
                { title: "Documentation Verification", description: "Confirm that README instructions actually work as written." }
            ],
            relatedUseCases: ["ephemeral-dev-environments", "technical-interviews", "collaborative-intelligence"]
        },
        "gpu-terminals": {
            title: "GPU Terminals for AI/ML",
            icon: "gpu",
            tagline: "Instant-on GPU power for your AI/ML workflows.",
            description: "Rexec will provide instant-on, powerful GPU-enabled terminals for your team's AI/ML model development, training, and fine-tuning. Manage and share these dedicated GPU resources securely, eliminating the complexities of direct infrastructure access.",
            heroImage: "/images/use-cases/gpu-terminals.svg",
            comingSoon: true,
            benefits: [
                { title: "On-Demand GPU Access", description: "Spin up GPU-accelerated terminals when you need them. Pay only for what you use.", icon: "bolt" },
                { title: "Team Resource Management", description: "Allocate GPU quotas across your team. No more fighting over shared resources.", icon: "connected" },
                { title: "Pre-Configured ML Stack", description: "TensorFlow, PyTorch, CUDA - all pre-installed and ready to go.", icon: "ai" },
                { title: "Secure Collaboration", description: "Share running GPU sessions with collaborators without exposing credentials.", icon: "shield" }
            ],
            workflow: [
                { step: 1, title: "Select GPU Tier", description: "Choose from T4, A100, or H100 based on your workload." },
                { step: 2, title: "Launch Environment", description: "Get a fully configured ML environment in seconds." },
                { step: 3, title: "Train & Experiment", description: "Run training jobs, fine-tune models, or experiment with new architectures." },
                { step: 4, title: "Share & Collaborate", description: "Invite team members to your session or share results instantly." }
            ],
            examples: [
                { title: "Model Fine-Tuning", description: "Fine-tune LLMs on your custom dataset with full GPU acceleration.", code: "python train.py --model llama-7b --dataset custom.json\n# Training on NVIDIA A100..." },
                { title: "Jupyter Notebooks", description: "Run GPU-accelerated notebooks for data science and ML experimentation." },
                { title: "Distributed Training", description: "Coordinate multi-GPU training jobs across your team's allocated resources." }
            ],
            relatedUseCases: ["collaborative-intelligence", "real-time-data-processing", "ephemeral-dev-environments"]
        },
        "edge-device-development": {
            title: "Edge Device Development",
            icon: "wifi",
            tagline: "Develop for IoT and edge in the cloud.",
            description: "Develop and test applications for IoT and edge devices in a simulated or emulated environment. Cross-compile for ARM, RISC-V, and other architectures without physical hardware.",
            heroImage: "/images/use-cases/edge-dev.svg",
            benefits: [
                { title: "Cross-Compilation Ready", description: "Toolchains for ARM, RISC-V, and other architectures pre-installed.", icon: "bolt" },
                { title: "Architecture Emulation", description: "Test on various architectures without physical hardware.", icon: "terminal" },
                { title: "Secure Remote Access", description: "Connect to virtual devices securely from anywhere.", icon: "shield" },
                { title: "Rapid Prototyping", description: "Iterate quickly on embedded systems without hardware constraints.", icon: "connected" }
            ],
            workflow: [
                { step: 1, title: "Choose Architecture", description: "Select your target architecture (ARM64, ARMv7, RISC-V, etc.)." },
                { step: 2, title: "Develop & Compile", description: "Write code and cross-compile for your target platform." },
                { step: 3, title: "Emulate & Test", description: "Run your code in an emulated environment." },
                { step: 4, title: "Deploy to Hardware", description: "Push tested binaries to physical devices with confidence." }
            ],
            examples: [
                { title: "Raspberry Pi Development", description: "Develop and test ARM binaries before deploying to physical Pi devices.", code: "arm-linux-gnueabihf-gcc -o app main.c\nqemu-arm ./app" },
                { title: "RISC-V Exploration", description: "Experiment with RISC-V architecture without specialized hardware." },
                { title: "Firmware Development", description: "Build and test embedded firmware in isolated environments." }
            ],
            relatedUseCases: ["ephemeral-dev-environments", "universal-jump-host", "real-time-data-processing"]
        },
        "real-time-data-processing": {
            title: "Real-time Data Processing",
            icon: "data",
            tagline: "Build streaming pipelines in isolated sandboxes.",
            description: "Build, test, and deploy streaming ETL pipelines and real-time analytics applications. High-performance data ingress/egress with monitoring and debugging tools.",
            heroImage: "/images/use-cases/data-processing.svg",
            benefits: [
                { title: "High-Performance I/O", description: "Optimized for high-throughput data ingress and egress.", icon: "bolt" },
                { title: "Streaming Integration", description: "Connect to Kafka, Flink, Spark, and other streaming platforms.", icon: "data" },
                { title: "Isolated Monitoring", description: "Monitor data flows without affecting production systems.", icon: "connected" },
                { title: "Secure Data Access", description: "Connect to data sources with proper credential management.", icon: "shield" }
            ],
            workflow: [
                { step: 1, title: "Configure Data Sources", description: "Connect to your databases, streams, and APIs." },
                { step: 2, title: "Build Pipeline", description: "Develop your ETL logic with real data samples." },
                { step: 3, title: "Test & Validate", description: "Run your pipeline against test data with full monitoring." },
                { step: 4, title: "Deploy with Confidence", description: "Push validated pipelines to production." }
            ],
            examples: [
                { title: "Kafka Stream Processing", description: "Build and test Kafka consumers and producers in isolation.", code: "kafka-console-consumer --topic events \
  --bootstrap-server kafka:9092" },
                { title: "ETL Development", description: "Develop complex data transformations with immediate feedback." },
                { title: "Analytics Prototyping", description: "Build real-time dashboards and analytics before production deployment." }
            ],
            relatedUseCases: ["collaborative-intelligence", "gpu-terminals", "ephemeral-dev-environments"]
        },
        "resumable-sessions": {
            title: "Resumable Terminal Sessions",
            icon: "reconnect",
            tagline: "Start tasks, disconnect, and come back later.",
            description: "Run long-running processes, close your browser, and reconnect anytime. Your terminal session continues in the background with full scrollback history. Never lose work to network drops or accidental tab closures again.",
            heroImage: "/images/use-cases/resumable-sessions.svg",
            benefits: [
                { title: "50K Line History", description: "50,000 lines of scrollback buffer means you never miss output, even for verbose builds or training runs.", icon: "data" },
                { title: "Survives Disconnects", description: "Network drops, laptop sleep, browser crashes - your session keeps running regardless.", icon: "reconnect" },
                { title: "Multi-Device Access", description: "Start on desktop, continue on laptop. Same session, same state, any device.", icon: "connected" },
                { title: "Background Execution", description: "Long builds, ML training, or deployments continue running even when you're away.", icon: "bolt" }
            ],
            workflow: [
                { step: 1, title: "Start Your Task", description: "Launch a build, training job, or any long-running process." },
                { step: 2, title: "Disconnect Freely", description: "Close your browser, switch networks, or shut down your laptop." },
                { step: 3, title: "Reconnect Anytime", description: "Come back hours or days later - your session is still there." },
                { step: 4, title: "Review Output", description: "Scroll back through all the output that happened while you were away." }
            ],
            examples: [
                { title: "ML Training Overnight", description: "Start a model training run, go home, and check results in the morning.", code: "python train.py --epochs 100 --dataset large\n# Close browser, sleep, reconnect next day\n# See: 'Epoch 100/100... Training complete!'" },
                { title: "Large Builds", description: "Kick off a full project build and come back when it's done.", code: "cargo build --release  # Takes 30+ minutes\n# Disconnect, grab coffee, reconnect\n# See: 'Finished release [optimized] target(s)'" },
                { title: "Deployment Monitoring", description: "Watch deployments progress even if you need to switch contexts.", code: "kubectl rollout status deployment/app --watch\n# Disconnect during slow rollout\n# Reconnect to see final status" }
            ],
            testimonial: { quote: "I used to lose hours of work when my VPN dropped. With Rexec's resumable sessions, I just reconnect and everything is still there.", author: "Marcus Rivera", role: "DevOps Engineer" },
            relatedUseCases: ["collaborative-intelligence", "ephemeral-dev-environments", "gpu-terminals"]
        },
        "rexec-cli": {
            title: "Rexec CLI & TUI",
            icon: "terminal",
            tagline: "Manage your terminals from anywhere using our powerful command-line interface.",
            description: "The Rexec CLI brings the power of the platform to your local terminal. Manage sessions, ssh into containers, and use the interactive TUI dashboard without leaving your keyboard.",
            heroImage: "/images/use-cases/cli.svg",
            benefits: [
                { title: "Full Terminal Management", description: "Create, list, connect, and stop terminals directly from your command line.", icon: "terminal" },
                { title: "Interactive TUI", description: "Visual dashboard in your terminal. Navigate sessions with arrow keys and shortcuts.", icon: "bolt" },
                { title: "Scriptable Automation", description: "Pipe commands, automate setups, and integrate Rexec into your local scripts.", icon: "connected" },
                { title: "Secure SSH Integration", description: "Seamlessly SSH into any Rexec container using native tools or the CLI wrapper.", icon: "shield" }
            ],
            workflow: [
                { step: 1, title: "Install CLI", description: "One-line install script for macOS and Linux." },
                { step: 2, title: "Authenticate", description: "Login securely via browser: `rexec login`" },
                { step: 3, title: "Launch TUI", description: "Start the interactive dashboard: `rexec -i`" },
                { step: 4, title: "Connect", description: "Select a session and jump straight in." }
            ],
            examples: [
                { title: "Quick Connect", description: "Instantly connect to a specific container by name or ID.", code: "rexec connect my-container" },
                { title: "Interactive Mode", description: "Launch the TUI to browse all active sessions and manage them visually.", code: "rexec -i" },
                { title: "Port Forwarding", description: "Forward remote ports to your local machine effortlessly.", code: "rexec forward -L 8080:localhost:80 my-container" }
            ],
            relatedUseCases: ["universal-jump-host", "resumable-sessions", "rexec-agent"]
        },
        "rexec-agent": {
            title: "Hybrid Cloud & Remote Agents",
            icon: "connected",
            tagline: "Unify your infrastructure. One dashboard for everything.",
            description: "Turn any Linux server, IoT device, or cloud instance into a managed Rexec terminal. Install our lightweight binary to instantly connect remote resources to your Rexec dashboard with real-time resource monitoring.",
            heroImage: "/images/use-cases/agent.svg",
            benefits: [
                { title: "Universal Compatibility", description: "Works on AWS, Azure, on-prem, or Raspberry Pi. Static binary, zero dependencies.", icon: "bolt" },
                { title: "Real-Time Monitoring", description: "Live CPU, Memory, and Disk usage stats for all connected agents.", icon: "data" },
                { title: "Persistent Connections", description: "Agents maintain a secure outbound tunnel. No inbound firewall ports needed.", icon: "shield" },
                { title: "Centralized Management", description: "Update, restart, and manage diverse infrastructure from one UI.", icon: "connected" }
            ],
            workflow: [
                { step: 1, title: "Register Agent", description: "Create a new agent profile in your Rexec settings." },
                { step: 2, title: "Install One-Liner", description: "Run the curl command on your target machine." },
                { step: 3, title: "Instant Connection", description: "The agent connects back via WebSocket and appears online." },
                { step: 4, title: "Monitor & Control", description: "View live stats and open terminal sessions immediately." }
            ],
            examples: [
                { title: "Home Lab Gateway", description: "Access your home server from anywhere without exposing it to the internet.", code: "curl -sSL rexec.pipeops.io/install-agent.sh | bash" },
                { title: "Multi-Cloud Ops", description: "Manage instances across AWS, GCP, and Azure without switching consoles." },
                { title: "Edge Fleet", description: "Monitor and control a fleet of remote IoT devices." }
            ],
            relatedUseCases: ["universal-jump-host", "hybrid-infrastructure", "remote-debugging"]
        },
        "hybrid-infrastructure": {
            title: "Hybrid Infrastructure Access",
            icon: "shield",
            tagline: "Mix cloud-managed terminals with your own infrastructure.",
            description: "Access everything through a single, unified interface. Seamlessly switch between Rexec's cloud terminals and your on-premise servers without changing tools or context.",
            heroImage: "/images/use-cases/hybrid.svg",
            benefits: [
                { title: "Unified Interface", description: "Manage cloud ephemeral environments and persistent on-prem servers side-by-side.", icon: "bolt" },
                { title: "Centralized Auditing", description: "One audit log for all your infrastructure access, regardless of location.", icon: "data" },
                { title: "Granular Access Control", description: "Define who can access what, across all your environments using Rexec roles.", icon: "shield" },
                { title: "No VPN Required", description: "Securely access internal resources without the headache of VPN client configuration.", icon: "wifi" }
            ],
            workflow: [
                { step: 1, title: "Connect Agents", description: "Deploy Rexec agents to your private infrastructure." },
                { step: 2, title: "Define Policies", description: "Set up teams and roles to control access permissions." },
                { step: 3, title: "Grant Access", description: "Users can now access approved servers instantly." },
                { step: 4, title: "Audit Usage", description: "Track every session and command across your hybrid estate." }
            ],
            examples: [
                { title: "Multi-Cloud Management", description: "Operate across AWS, Azure, and on-premise datacenters from a single URL." },
                { title: "Burst to Cloud", description: "Develop locally, then seamlessly deploy to cloud instances for testing." },
                { title: "Legacy System Access", description: "Modernize access to legacy mainframes or bare-metal servers." }
            ],
            relatedUseCases: ["rexec-agent", "universal-jump-host", "remote-debugging"]
        },
        "remote-debugging": {
            title: "Remote Debugging & Troubleshooting",
            icon: "bug",
            tagline: "Debug production issues directly from your browser.",
            description: "Connect to any server running the Rexec agent for instant access. Troubleshoot live systems with full terminal capabilities, share sessions with colleagues, and resolve incidents faster.",
            heroImage: "/images/use-cases/debugging.svg",
            benefits: [
                { title: "Instant Access", description: "Jump into a troubleshooting session in seconds when every moment counts.", icon: "bolt" },
                { title: "Collaborative Debugging", description: "Invite other engineers to your session to pair-debug complex issues.", icon: "connected" },
                { title: "Secure & Audited", description: "Grant temporary emergency access with full recording of actions taken.", icon: "shield" },
                { title: "No SSH Key Hassle", description: "Don't waste time finding the right key or asking for access during an outage.", icon: "terminal" }
            ],
            workflow: [
                { step: 1, title: "Receive Alert", description: "Get notified of an issue in your monitoring system." },
                { step: 2, title: "Click to Connect", description: "Launch a Rexec session directly from your alert dashboard." },
                { step: 3, title: "Diagnose Issue", description: "Run diagnostic tools, check logs, and inspect processes." },
                { step: 4, title: "Resolve & Close", description: "Fix the problem and terminate the session. All recorded for post-mortem." }
            ],
            examples: [
                { title: "Production Incident", description: "Investigate a high-CPU alert on a production web server.", code: "top -o %CPU\n# Identify runaway process\nkill -9 <pid>" },
                { title: "Log Analysis", description: "Tail logs in real-time to catch intermittent errors.", code: "tail -f /var/log/nginx/error.log | grep 500" },
                { title: "Performance Tuning", description: "Adjust kernel parameters or configuration on a live system." }
            ],
            relatedUseCases: ["rexec-agent", "collaborative-intelligence", "resumable-sessions"]
        }
    };

    $: useCase = useCasesData[slug];
    $: relatedCases = (useCase?.relatedUseCases || [])
        .map(s => {
            const data = useCasesData[s];
            if (!data) return null;
            return { slug: s, title: data.title, icon: data.icon, tagline: data.tagline };
        })
        .filter((c): c is { slug: string; title: string; icon: string; tagline: string } => c !== null);

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
        window.scrollTo({ top: 0, behavior: 'smooth' });
    });
</script>

<svelte:head>
    {#if useCase}
        <title>{useCase.title} | Rexec - Cloud Development Environment</title>
        <meta name="description" content={useCase.description} />
        <meta name="keywords" content="rexec, {useCase.title.toLowerCase()}, cloud terminal, development environment, {slug.replace(/-/g, ', ')}" />
        
        <!-- Open Graph -->
        <meta property="og:title" content="{useCase.title} - Rexec" />
        <meta property="og:description" content={useCase.tagline + " " + useCase.description} />
        <meta property="og:type" content="article" />
        <meta property="og:url" content="https://rexec.pipeops.io/use-cases/{slug}" />
        <meta property="og:image" content="https://rexec.pipeops.io/og-image.png" />
        <meta property="og:site_name" content="Rexec" />
        
        <!-- Twitter Card -->
        <meta name="twitter:card" content="summary_large_image" />
        <meta name="twitter:title" content="{useCase.title} - Rexec" />
        <meta name="twitter:description" content={useCase.tagline} />
        <meta name="twitter:image" content="https://rexec.pipeops.io/og-image.png" />
        
        <!-- Canonical -->
        <link rel="canonical" href="https://rexec.pipeops.io/use-cases/{slug}" />
        
        <!-- JSON-LD Structured Data -->
        {@html `<script type="application/ld+json">${JSON.stringify({
            "@context": "https://schema.org",
            "@type": "Article",
            "headline": useCase.title,
            "description": useCase.description,
            "image": "https://rexec.pipeops.io/og-image.png",
            "author": {
                "@type": "Organization",
                "name": "Rexec"
            },
            "publisher": {
                "@type": "Organization",
                "name": "Rexec",
                "logo": {
                    "@type": "ImageObject",
                    "url": "https://rexec.pipeops.io/favicon.svg"
                }
            },
            "mainEntityOfPage": {
                "@type": "WebPage",
                "@id": `https://rexec.pipeops.io/use-cases/${slug}`
            }
        })}</script>`}
    {/if}
</svelte:head>

{#if useCase}
    <div class="use-case-detail" class:coming-soon={useCase.comingSoon}>
        <!-- Hero Section -->
        <section class="hero">
            <button class="back-btn" onclick={handleBack}>
                <StatusIcon status="arrow-left" size={16} />
                <span>All Use Cases</span>
            </button>
            
            <div class="hero-content">
                <div class="hero-text">
                    {#if useCase.comingSoon}
                        <div class="coming-soon-badge">
                            <StatusIcon status="clock" size={14} />
                            <span>Coming Soon</span>
                        </div>
                    {/if}
                    <div class="hero-icon" class:coming-soon-icon={useCase.comingSoon}>
                        <StatusIcon status={useCase.icon} size={48} />
                    </div>
                    <h1>{useCase.title}</h1>
                    <p class="tagline">{useCase.tagline}</p>
                    <p class="description">{useCase.description}</p>
                    
                    {#if !useCase.comingSoon}
                        <button class="btn btn-primary btn-lg cta-btn" onclick={handleTryNow}>
                            <StatusIcon status="rocket" size={16} />
                            <span>Try It Now</span>
                        </button>
                    {:else}
                        <button class="btn btn-secondary btn-lg cta-btn" disabled>
                            <StatusIcon status="clock" size={16} />
                            <span>Notify Me When Available</span>
                        </button>
                    {/if}
                </div>
                <div class="hero-visual">
                    <div class="terminal-mockup">
                        <div class="terminal-header">
                            <span class="dot red"></span>
                            <span class="dot yellow"></span>
                            <span class="dot green"></span>
                            <span class="terminal-title">rexec — {useCase.title.toLowerCase().replace(/\s+/g, '-')}</span>
                        </div>
                        <div class="terminal-body">
                            <div class="terminal-line">
                                <span class="prompt">$</span>
                                <span class="command">rexec launch --use-case {slug}</span>
                            </div>
                            <div class="terminal-output">
                                <span class="success">✓</span> Environment ready in 1.2s
                            </div>
                            <div class="terminal-line">
                                <span class="prompt">$</span>
                                <span class="cursor">_</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <!-- Benefits Section -->
        <section class="benefits">
            <h2>Key Benefits</h2>
            <div class="benefits-grid">
                {#each useCase.benefits as benefit, i}
                    <div class="benefit-card" style="animation-delay: {i * 100}ms">
                        <div class="benefit-icon">
                            <StatusIcon status={benefit.icon} size={24} />
                        </div>
                        <h3>{benefit.title}</h3>
                        <p>{benefit.description}</p>
                    </div>
                {/each}
            </div>
        </section>

        <!-- Workflow Section -->
        <section class="workflow">
            <h2>How It Works</h2>
            <div class="workflow-steps">
                {#each useCase.workflow as step, i}
                    <div class="workflow-step" style="animation-delay: {i * 150}ms">
                        <div class="step-number">{step.step}</div>
                        <div class="step-content">
                            <h3>{step.title}</h3>
                            <p>{step.description}</p>
                        </div>
                        {#if i < useCase.workflow.length - 1}
                            <div class="step-connector"></div>
                        {/if}
                    </div>
                {/each}
            </div>
        </section>

        <!-- Examples Section -->
        <section class="examples">
            <h2>Real-World Examples</h2>
            <div class="examples-grid">
                {#each useCase.examples as example, i}
                    <div class="example-card" style="animation-delay: {i * 100}ms">
                        <h3>{example.title}</h3>
                        <p>{example.description}</p>
                        {#if example.code}
                            <div class="code-block">
                                <pre><code>{example.code}</code></pre>
                                <button 
                                    class="copy-btn" 
                                    onclick={() => {
                                        navigator.clipboard.writeText(example.code || '');
                                        // Simple feedback mechanism without state variable for each item
                                        const btn = document.activeElement;
                                        if (btn) btn.textContent = 'Copied!';
                                        setTimeout(() => { if (btn) btn.textContent = 'Copy'; }, 2000);
                                    }}
                                >
                                    Copy
                                </button>
                            </div>
                        {/if}
                    </div>
                {/each}
            </div>
        </section>

        <!-- Testimonial Section -->
        {#if useCase.testimonial}
            <section class="testimonial">
                <div class="testimonial-card">
                    <div class="quote-mark">"</div>
                    <blockquote>{useCase.testimonial.quote}</blockquote>
                    <div class="testimonial-author">
                        <div class="author-avatar">
                            <StatusIcon status="user" size={24} />
                        </div>
                        <div class="author-info">
                            <strong>{useCase.testimonial.author}</strong>
                            <span>{useCase.testimonial.role}</span>
                        </div>
                    </div>
                </div>
            </section>
        {/if}

        <!-- Related Use Cases -->
        {#if relatedCases.length > 0}
            <section class="related">
                <h2>Related Use Cases</h2>
                <div class="related-grid">
                    {#each relatedCases as related}
                        <button class="related-card" onclick={() => navigateToCase(related.slug)}>
                            <div class="related-icon">
                                <StatusIcon status={related.icon} size={24} />
                            </div>
                            <h3>{related.title}</h3>
                            <span class="arrow">→</span>
                        </button>
                    {/each}
                </div>
            </section>
        {/if}

        <!-- CTA Section -->
        <section class="cta-section">
            <h2>Ready to get started?</h2>
            <p>Launch your first terminal and experience the future of development.</p>
            <button class="btn btn-primary btn-lg" onclick={handleTryNow}>
                <StatusIcon status="rocket" size={16} />
                <span>Launch Terminal</span>
            </button>
        </section>
    </div>
{:else}
    <div class="not-found">
        <h1>Use Case Not Found</h1>
        <p>The requested use case doesn't exist.</p>
        <button class="btn btn-primary" onclick={handleBack}>
            <StatusIcon status="arrow-left" size={16} />
            Back to Use Cases
        </button>
    </div>
{/if}

<style>
    :root {
        --bg-primary: #050505;
        --bg-card: #0f0f0f;
        --bg-tertiary: #1a1a1a;
        --border: #222;
        --text: #eee;
        --text-secondary: #aaa;
        --text-muted: #777;
        --accent: #00ffaa;
        --yellow: #fce00a;
        --font-mono: 'JetBrains Mono', monospace;
        --font-sans: 'Inter', sans-serif;
    }

    .use-case-detail {
        max-width: 1200px;
        margin: 0 auto;
        padding: 20px;
    }

    /* Hero Section */
    .hero {
        margin-bottom: 80px;
    }

    .back-btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        background: none;
        border: 1px solid var(--border);
        color: var(--text-secondary);
        padding: 8px 16px;
        font-family: var(--font-mono);
        font-size: 12px;
        cursor: pointer;
        margin-bottom: 40px;
        transition: all 0.2s;
    }

    .back-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    .hero-content {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 60px;
        align-items: center;
    }

    .hero-icon {
        width: 80px;
        height: 80px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.3);
        border-radius: 16px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--accent);
        margin-bottom: 24px;
        animation: fadeInUp 0.5s ease;
    }

    .coming-soon-icon {
        background: rgba(252, 238, 10, 0.1);
        border-color: rgba(252, 238, 10, 0.3);
        color: var(--yellow);
    }

    .coming-soon-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        background: rgba(252, 238, 10, 0.1);
        border: 1px solid var(--yellow);
        color: var(--yellow);
        padding: 4px 12px;
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 1px;
        margin-bottom: 16px;
        animation: pulse 2s infinite;
    }

    h1 {
        font-size: 42px;
        font-weight: 700;
        margin: 0 0 16px 0;
        letter-spacing: -0.5px;
        animation: fadeInUp 0.5s ease 0.1s both;
    }

    .tagline {
        font-size: 20px;
        color: var(--accent);
        margin: 0 0 20px 0;
        animation: fadeInUp 0.5s ease 0.2s both;
    }

    .use-case-detail.coming-soon .tagline {
        color: var(--yellow);
    }

    .description {
        font-size: 16px;
        color: var(--text-secondary);
        line-height: 1.7;
        margin: 0 0 32px 0;
        animation: fadeInUp 0.5s ease 0.3s both;
    }

    .cta-btn {
        animation: fadeInUp 0.5s ease 0.4s both;
    }

    .hero-visual {
        animation: fadeInRight 0.6s ease 0.3s both;
    }

    .terminal-mockup {
        background: #000;
        border: 1px solid var(--border);
        border-radius: 8px;
        overflow: hidden;
        box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
    }

    .terminal-header {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 12px 16px;
        background: #111;
        border-bottom: 1px solid var(--border);
    }

    .dot {
        width: 12px;
        height: 12px;
        border-radius: 50%;
    }

    .dot.red { background: #ff5f56; }
    .dot.yellow { background: #ffbd2e; }
    .dot.green { background: #27c93f; }

    .terminal-title {
        flex: 1;
        text-align: center;
        font-size: 12px;
        color: var(--text-muted);
    }

    .terminal-body {
        padding: 20px;
        font-family: var(--font-mono);
        font-size: 14px;
    }

    .terminal-line {
        margin-bottom: 8px;
    }

    .prompt {
        color: var(--accent);
        margin-right: 8px;
    }

    .command {
        color: var(--text);
    }

    .terminal-output {
        color: var(--text-muted);
        margin-bottom: 8px;
    }

    .terminal-output .success {
        color: var(--accent);
    }

    .cursor {
        background: var(--accent);
        color: #000;
        animation: blink 1s step-end infinite;
    }

    /* Benefits Section */
    .benefits {
        margin-bottom: 80px;
    }

    h2 {
        font-size: 28px;
        margin-bottom: 40px;
        text-align: center;
    }

    .benefits-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 24px;
    }

    .benefit-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        padding: 24px;
        border-radius: 12px;
        transition: all 0.3s ease;
        animation: fadeInUp 0.5s ease both;
    }

    .benefit-card:hover {
        border-color: var(--accent);
        transform: translateY(-4px);
        box-shadow: 0 10px 40px rgba(0, 255, 65, 0.1);
    }

    .benefit-icon {
        width: 48px;
        height: 48px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.2);
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--accent);
        margin-bottom: 16px;
    }

    .benefit-card h3 {
        font-size: 16px;
        margin: 0 0 8px 0;
    }

    .benefit-card p {
        font-size: 13px;
        color: var(--text-secondary);
        line-height: 1.5;
        margin: 0;
    }

    /* Workflow Section */
    .workflow {
        margin-bottom: 80px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 16px;
        padding: 60px;
    }

    .workflow-steps {
        display: flex;
        flex-direction: column;
        gap: 0;
        max-width: 600px;
        margin: 0 auto;
    }

    .workflow-step {
        display: flex;
        align-items: flex-start;
        gap: 20px;
        position: relative;
        animation: fadeInUp 0.5s ease both;
    }

    .step-number {
        width: 40px;
        height: 40px;
        background: var(--accent);
        color: #000;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 700;
        font-size: 16px;
        flex-shrink: 0;
        z-index: 1;
    }

    .step-content {
        padding-bottom: 40px;
    }

    .step-content h3 {
        font-size: 18px;
        margin: 0 0 8px 0;
    }

    .step-content p {
        font-size: 14px;
        color: var(--text-secondary);
        margin: 0;
        line-height: 1.5;
    }

    .step-connector {
        position: absolute;
        left: 19px;
        top: 40px;
        width: 2px;
        height: calc(100% - 40px);
        background: linear-gradient(to bottom, var(--accent), var(--border));
    }

    /* Examples Section */
    .examples {
        margin-bottom: 80px;
    }

    .examples-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        gap: 24px;
    }

    .example-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        padding: 24px;
        border-radius: 12px;
        animation: fadeInUp 0.5s ease both;
        transition: border-color 0.3s;
    }

    .example-card:hover {
        border-color: var(--accent);
    }

    .example-card h3 {
        font-size: 16px;
        margin: 0 0 12px 0;
        color: var(--accent);
    }

    .example-card p {
        font-size: 14px;
        color: var(--text-secondary);
        line-height: 1.5;
        margin: 0 0 16px 0;
    }

    .code-block {
        background: #000;
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 16px;
        overflow-x: auto;
        position: relative;
    }

    .code-block pre {
        margin: 0;
        padding-right: 60px;
    }

    .code-block code {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--text);
        line-height: 1.6;
    }

    .copy-btn {
        position: absolute;
        top: 12px;
        right: 12px;
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.2);
        color: var(--text-secondary);
        font-size: 10px;
        padding: 4px 8px;
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.2s;
        font-family: var(--font-mono);
    }

    .copy-btn:hover {
        background: rgba(255, 255, 255, 0.2);
        color: var(--text);
    }

    /* Testimonial Section */
    .testimonial {
        margin-bottom: 80px;
    }

    .testimonial-card {
        background: linear-gradient(135deg, rgba(0, 255, 65, 0.05) 0%, transparent 50%);
        border: 1px solid var(--border);
        border-radius: 16px;
        padding: 60px;
        text-align: center;
        position: relative;
    }

    .quote-mark {
        font-size: 120px;
        color: var(--accent);
        opacity: 0.2;
        position: absolute;
        top: 20px;
        left: 40px;
        font-family: Georgia, serif;
        line-height: 1;
    }

    blockquote {
        font-size: 24px;
        font-style: italic;
        color: var(--text);
        margin: 0 0 32px 0;
        line-height: 1.5;
        max-width: 700px;
        margin-left: auto;
        margin-right: auto;
    }

    .testimonial-author {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 16px;
    }

    .author-avatar {
        width: 48px;
        height: 48px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--text-muted);
    }

    .author-info {
        text-align: left;
    }

    .author-info strong {
        display: block;
        font-size: 14px;
    }

    .author-info span {
        font-size: 12px;
        color: var(--text-muted);
    }

    /* Related Section */
    .related {
        margin-bottom: 80px;
    }

    .related-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 20px;
    }

    .related-card {
        display: flex;
        align-items: center;
        gap: 16px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        padding: 20px;
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.3s ease;
        text-align: left;
        font-family: var(--font-mono);
        width: 100%;
        color: var(--text);
    }

    .related-card:hover {
        border-color: var(--accent);
        transform: translateX(8px);
    }

    .related-card:hover .arrow {
        transform: translateX(4px);
        color: var(--accent);
    }

    .related-icon {
        width: 40px;
        height: 40px;
        background: rgba(0, 255, 65, 0.1);
        border: 1px solid rgba(0, 255, 65, 0.2);
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--accent);
        flex-shrink: 0;
    }

    .related-card h3 {
        flex: 1;
        font-size: 14px;
        margin: 0;
        color: var(--text);
    }

    .arrow {
        font-size: 18px;
        color: var(--text-muted);
        transition: all 0.2s;
    }

    /* CTA Section */
    .cta-section {
        text-align: center;
        padding: 80px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 16px;
    }

    .cta-section h2 {
        margin-bottom: 16px;
    }

    .cta-section p {
        color: var(--text-muted);
        margin-bottom: 32px;
    }

    /* Not Found */
    .not-found {
        text-align: center;
        padding: 100px 20px;
    }

    .not-found h1 {
        font-size: 32px;
        margin-bottom: 16px;
    }

    .not-found p {
        color: var(--text-muted);
        margin-bottom: 32px;
    }

    /* Animations */
    @keyframes fadeInUp {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }

    @keyframes fadeInRight {
        from {
            opacity: 0;
            transform: translateX(40px);
        }
        to {
            opacity: 1;
            transform: translateX(0);
        }
    }

    @keyframes blink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0; }
    }

    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.7; }
    }

    /* Responsive */
    @media (max-width: 900px) {
        .hero-content {
            grid-template-columns: 1fr;
            gap: 40px;
        }

        h1 {
            font-size: 32px;
        }

        .workflow {
            padding: 40px 24px;
        }

        .testimonial-card {
            padding: 40px 24px;
        }

        blockquote {
            font-size: 18px;
        }

        .quote-mark {
            font-size: 80px;
            top: 10px;
            left: 20px;
        }
    }

    @media (max-width: 600px) {
        .use-case-detail {
            padding: 16px;
        }

        h1 {
            font-size: 28px;
        }

        .cta-section {
            padding: 40px 24px;
        }
    }
</style>
