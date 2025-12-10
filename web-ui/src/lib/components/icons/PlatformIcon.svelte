<script lang="ts">
    interface Props {
        platform: string;
        size?: number;
    }
    
    let { platform, size = 20 }: Props = $props();

    // Platform to icon mapping - Professional, mature SVG icons
    const icons: Record<string, string> = {
        // Debian-based - Clean, professional icons
        ubuntu: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="5.5" r="2" /><circle cx="5.5" cy="15" r="2"/><circle cx="18.5" cy="15" r="2"/><path d="M12 7.5v3M7.2 14l2.3-1.3M16.8 14l-2.3-1.3" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        debian: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z" opacity="0.2"/><path d="M13.5 7c1.5 0 3 1.2 3.5 2.8.4 1.2.2 2.5-.3 3.7-.6 1.2-1.5 2-2.7 2.3-1.5.4-3 0-4-1-.8-.8-1.2-2-1-3.2.2-1.5 1.2-2.8 2.5-3.5.6-.3 1.3-.5 2-.5z"/></svg>`,
        alpine: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 4L4 18h16L12 4z" opacity="0.15"/><path d="M12 4L4 18h16L12 4zm0 3l5.5 9h-11L12 7z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>`,
        fedora: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8z"/><path d="M15 8h-3c-1.1 0-2 .9-2 2v6h2v-3h3v-2h-3v-1h3V8z"/></svg>`,
        centos: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="3" y="3" width="8" height="8" rx="1" opacity="0.3"/><rect x="13" y="3" width="8" height="8" rx="1" opacity="0.5"/><rect x="3" y="13" width="8" height="8" rx="1" opacity="0.5"/><rect x="13" y="13" width="8" height="8" rx="1" opacity="0.7"/><circle cx="12" cy="12" r="3" fill="currentColor"/></svg>`,
        archlinux: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 3L3 21h18L12 3z" opacity="0.1"/><path d="M12 3L3 21h6l3-6 3 6h6L12 3z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>`,
        arch: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 3L3 21h18L12 3z" opacity="0.1"/><path d="M12 3L3 21h6l3-6 3 6h6L12 3z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>`,
        kali: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><path d="M7 8l5 4-5 4M12 16h5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`,

        // Red Hat based
        rocky: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><circle cx="12" cy="12" r="8" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="12" r="4" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="12" r="1.5"/></svg>`,
        alma: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L2 7l10 5 10-5-10-5z" opacity="0.3"/><path d="M2 17l10 5 10-5" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M2 12l10 5 10-5" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M2 7l10 5 10-5" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        rhel: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z"/><path d="M10 8h4v8h-2v-6h-2V8z" fill="var(--bg-primary, #0a0a0a)"/></svg>`,
        oracle: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="3" y="8" width="18" height="8" rx="4" opacity="0.2"/><rect x="3" y="8" width="18" height="8" rx="4" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,

        // Security distros
        parrot: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><path d="M8 8c0-2 2-4 4-4s4 2 4 4c0 3-2 4-4 8-2-4-4-5-4-8z" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="10" cy="8" r="1"/><circle cx="14" cy="8" r="1"/></svg>`,
        blackarch: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L2 22h20L12 2z" opacity="0.15"/><path d="M12 2L2 22h20L12 2z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/><path d="M12 8v8M8 14h8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,

        // Arch-based
        manjaro: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="3" y="3" width="5" height="18" rx="1" opacity="0.8"/><rect x="10" y="8" width="5" height="13" rx="1" opacity="0.6"/><rect x="17" y="13" width="5" height="8" rx="1" opacity="0.4"/></svg>`,

        // SUSE
        opensuse: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="12" r="2"/><circle cx="8" cy="10" r="1"/><circle cx="16" cy="10" r="1"/></svg>`,
        suse: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="12" r="2"/><circle cx="8" cy="10" r="1"/><circle cx="16" cy="10" r="1"/></svg>`,
        tumbleweed: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.5" stroke-dasharray="4 2"/><circle cx="12" cy="12" r="4" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,

        // Independent
        gentoo: `<svg viewBox="0 0 24 24" fill="currentColor"><ellipse cx="12" cy="12" rx="8" ry="10" opacity="0.1"/><ellipse cx="12" cy="12" rx="7" ry="9" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M12 6c-2 0-4 2.5-4 6s2 6 4 6" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        void: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="12" r="4" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        nixos: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L2 12l10 10 10-10L12 2z" opacity="0.1"/><path d="M12 2L2 12l10 10 10-10L12 2z" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M12 7v10M7 12h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,
        slackware: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="4" y="4" width="16" height="16" rx="2" opacity="0.1"/><path d="M8 8h3v8H8zm5 0h3v8h-3z" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,

        // Debian derivatives
        mint: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="3" y="3" width="18" height="18" rx="3" opacity="0.1"/><path d="M8 12l4-4 4 4M12 8v8" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
        elementary: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M12 7v5l3 3" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,
        devuan: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><path d="M8 16l8-8M8 8l8 8" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,

        // Minimal
        busybox: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="4" y="4" width="16" height="16" rx="2" opacity="0.1"/><rect x="4" y="4" width="16" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><rect x="8" y="8" width="8" height="8" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,

        // Generic Linux (Tux)
        linux: `<svg viewBox="0 0 24 24" fill="currentColor"><ellipse cx="12" cy="14" rx="6" ry="7" opacity="0.1"/><ellipse cx="12" cy="14" rx="5.5" ry="6.5" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="7" r="4" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="10" cy="6.5" r="0.75"/><circle cx="14" cy="6.5" r="0.75"/><ellipse cx="12" cy="8.5" rx="1" ry="0.5"/></svg>`,

        // Role/Environment icons - Modern, professional
        standard: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="3" y="4" width="18" height="14" rx="2" opacity="0.1"/><rect x="3" y="4" width="18" height="14" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M7 11l3 3 7-7" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
        node: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L3 7v10l9 5 9-5V7l-9-5z" opacity="0.1"/><path d="M12 2L3 7v10l9 5 9-5V7l-9-5z" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/><path d="M12 7v10M7 9.5v5M17 9.5v5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,
        python: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C8 2 8 4 8 4v3h4v1H6s-4-.5-4 4 3 4.5 3 4.5h2v-2s0-2 2-2h4s2 0 2-2V5s0-3-3-3zm-1 1.5a.75.75 0 110 1.5.75.75 0 010-1.5z" opacity="0.6"/><path d="M12 22c4 0 4-2 4-2v-3h-4v-1h6s4 .5 4-4-3-4.5-3-4.5h-2v2s0 2-2 2h-4s-2 0-2 2v4s0 3 3 3zm1-1.5a.75.75 0 110-1.5.75.75 0 010 1.5z" opacity="0.9"/></svg>`,
        go: `<svg viewBox="0 0 24 24" fill="currentColor"><ellipse cx="8" cy="12" rx="5" ry="6" opacity="0.2"/><ellipse cx="16" cy="12" rx="5" ry="6" opacity="0.2"/><ellipse cx="8" cy="12" rx="4" ry="5" fill="none" stroke="currentColor" stroke-width="1.5"/><ellipse cx="16" cy="12" rx="4" ry="5" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="8" cy="10" r="1.5"/><circle cx="16" cy="10" r="1.5"/></svg>`,
        rust: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="9" opacity="0.1"/><circle cx="12" cy="12" r="8" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M12 4v2M12 18v2M4 12h2M18 12h2" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        neovim: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M4 4v16l6-8-6-8z" opacity="0.6"/><path d="M20 4v16l-6-8 6-8z" opacity="0.4"/><path d="M4 4l16 16M4 20l16-16" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        devops: `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="12" r="10" opacity="0.1"/><path d="M12 2c5.5 0 10 4.5 10 10s-4.5 10-10 10S2 17.5 2 12 6.5 2 12 2z" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M8 12c0-2.2 1.8-4 4-4M16 12c0 2.2-1.8 4-4 4" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="8" cy="12" r="1.5"/><circle cx="16" cy="12" r="1.5"/></svg>`,
        overemployed: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="3" y="5" width="7" height="5" rx="1" opacity="0.3"/><rect x="14" y="5" width="7" height="5" rx="1" opacity="0.3"/><rect x="3" y="14" width="7" height="5" rx="1" opacity="0.3"/><rect x="14" y="14" width="7" height="5" rx="1" opacity="0.3"/><circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        
        // Modern role icons
        "vibe-coder": `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="3" y="3" width="18" height="18" rx="3" opacity="0.1"/><path d="M7 8l4 4-4 4M12 16h5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/><circle cx="17" cy="7" r="2" opacity="0.5"/></svg>`,
        "gpu-alchemist": `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="2" y="6" width="20" height="12" rx="2" opacity="0.1"/><rect x="2" y="6" width="20" height="12" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><rect x="5" y="9" width="3" height="6" rx="0.5" opacity="0.5"/><rect x="10" y="9" width="3" height="6" rx="0.5" opacity="0.5"/><rect x="15" y="9" width="4" height="6" rx="0.5" fill="none" stroke="currentColor" stroke-width="1"/></svg>`,
        "cloud-native": `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M6.5 20c-2.5 0-4.5-2-4.5-4.5S4 11 6.5 11c.3-2.8 2.6-5 5.5-5 2.5 0 4.6 1.6 5.3 3.9C19.5 10.2 21 12 21 14.5c0 2.8-2.2 5-5 5.5H6.5z" opacity="0.15"/><path d="M6.5 20c-2.5 0-4.5-2-4.5-4.5S4 11 6.5 11c.3-2.8 2.6-5 5.5-5 2.5 0 4.6 1.6 5.3 3.9C19.5 10.2 21 12 21 14.5c0 2.8-2.2 5-5 5.5H6.5z" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        "remote-access": `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="4" y="8" width="16" height="12" rx="2" opacity="0.1"/><rect x="4" y="8" width="16" height="12" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M12 4v4M8 4h8" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><circle cx="12" cy="14" r="2" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        "pair-programming": `<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="8" cy="8" r="3" opacity="0.2"/><circle cx="16" cy="8" r="3" opacity="0.2"/><circle cx="8" cy="8" r="2.5" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="16" cy="8" r="2.5" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M4 20c0-3 2-5 4-5h8c2 0 4 2 4 5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,
        "data-science": `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="3" y="3" width="18" height="18" rx="2" opacity="0.1"/><rect x="6" y="13" width="3" height="5" rx="0.5" opacity="0.4"/><rect x="10.5" y="9" width="3" height="9" rx="0.5" opacity="0.6"/><rect x="15" y="6" width="3" height="12" rx="0.5" opacity="0.8"/></svg>`,
        minimalist: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="4" y="6" width="16" height="12" rx="2" opacity="0.1"/><rect x="4" y="6" width="16" height="12" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M4 10h16" stroke="currentColor" stroke-width="1.5"/></svg>`,

        // Docker/container
        custom: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="5" y="10" width="3" height="3" rx="0.5" opacity="0.3"/><rect x="9" y="10" width="3" height="3" rx="0.5" opacity="0.4"/><rect x="13" y="10" width="3" height="3" rx="0.5" opacity="0.5"/><rect x="9" y="6" width="3" height="3" rx="0.5" opacity="0.5"/><rect x="13" y="6" width="3" height="3" rx="0.5" opacity="0.6"/><rect x="17" y="10" width="3" height="3" rx="0.5" opacity="0.7"/><path d="M2 14c0-1 1-2 3-2h14c2 0 3 1 3 2v4c0 1-1 2-2 2H4c-1 0-2-1-2-2v-4z" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,
        docker: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="5" y="10" width="3" height="3" rx="0.5" opacity="0.3"/><rect x="9" y="10" width="3" height="3" rx="0.5" opacity="0.4"/><rect x="13" y="10" width="3" height="3" rx="0.5" opacity="0.5"/><rect x="9" y="6" width="3" height="3" rx="0.5" opacity="0.5"/><rect x="13" y="6" width="3" height="3" rx="0.5" opacity="0.6"/><rect x="17" y="10" width="3" height="3" rx="0.5" opacity="0.7"/><path d="M2 14c0-1 1-2 3-2h14c2 0 3 1 3 2v4c0 1-1 2-2 2H4c-1 0-2-1-2-2v-4z" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`,

        // macOS
        macos: `<svg viewBox="0 0 24 24" fill="currentColor"><path d="M18.7 19.5c-.8 1.2-1.7 2.5-3 2.5-1.4 0-1.8-.8-3.3-.8s-2 .8-3.3.8c-1.3 0-2.3-1.3-3.1-2.5C4.3 17 3 12.5 4.7 9.4c.9-1.5 2.4-2.5 4.1-2.5 1.3 0 2.5.9 3.3.9s2.3-1.1 3.8-.9c.7 0 2.5.3 3.6 2-.1 0-2.2 1.3-2.2 3.8 0 3 2.7 4 2.7 4-.1.1-.4 1.5-1.3 2.8zM13 3.5c.7-.8 1.9-1.5 2.9-1.5.1 1.2-.3 2.4-1 3.2-.7.9-1.8 1.5-3 1.4-.1-1.2.4-2.4 1.1-3.1z" opacity="0.85"/></svg>`,
        
        // Terminal/Shell
        terminal: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="2" y="4" width="20" height="16" rx="2" opacity="0.1"/><rect x="2" y="4" width="20" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M6 9l4 3-4 3M12 15h6" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
        shell: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="2" y="4" width="20" height="16" rx="2" opacity="0.1"/><rect x="2" y="4" width="20" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M6 9l4 3-4 3M12 15h6" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
        bash: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="2" y="4" width="20" height="16" rx="2" opacity="0.1"/><rect x="2" y="4" width="20" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M6 9l4 3-4 3" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><circle cx="15" cy="12" r="1"/></svg>`,
        zsh: `<svg viewBox="0 0 24 24" fill="currentColor"><rect x="2" y="4" width="20" height="16" rx="2" opacity="0.1"/><rect x="2" y="4" width="20" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M7 9h4l-4 6h4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
    };

    const svgContent = $derived(icons[platform.toLowerCase()] || icons.linux);
</script>

<span class="platform-icon" style="width: {size}px; height: {size}px;">
    {@html svgContent}
</span>

<style>
    .platform-icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
        color: inherit;
    }

    .platform-icon :global(svg) {
        width: 100%;
        height: 100%;
    }
</style>
