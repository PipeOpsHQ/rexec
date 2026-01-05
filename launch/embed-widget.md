# Introducing the Rexec Embeddable Terminal Widget

## Cloud Shell for Any Website

We're excited to announce the **Rexec Embed Widget** — a JavaScript library that lets you add a fully-featured cloud terminal to any website with just a few lines of code.

Think Google Cloud Shell, but embeddable anywhere.

## The Problem

Adding interactive terminal experiences to documentation, tutorials, or learning platforms has always been painful:

- **Self-hosting complexity**: Running terminal infrastructure means managing containers, WebSockets, security, and scaling.
- **Poor user experience**: Many solutions require users to install software or create accounts.
- **Limited customization**: Most embedded terminals look out of place and can't match your brand.

For AI-powered tools and documentation sites, there's an additional challenge: **giving users (or AI agents) a safe sandbox** to execute code without risking their local environment.

## The Solution

The Rexec Embed Widget is a single script tag that gives you:

```html
<script src="https://rexec.dev/embed/rexec.min.js"></script>
<div id="terminal" style="height: 400px;"></div>
<script>
  Rexec.embed('#terminal', { shareCode: 'ABC123' });
</script>
```

That's it. A real Linux terminal, running in the cloud, embedded in your page.

## Key Features

### 🚀 One-Line Integration
No build step, no dependencies. Include the script and call `Rexec.embed()`.

### 👥 Multiple Connection Modes
- **Share codes**: Let guests join sessions without accounts
- **API tokens**: Programmatic access for authenticated users
- **On-demand containers**: Spin up fresh environments per user

### 🎨 Fully Customizable
- Dark and light themes (or bring your own colors)
- Adjustable fonts, sizes, and cursor styles
- Event callbacks for complete control over behavior

### ⚡ Production-Ready
- WebSocket streaming for low latency
- Auto-reconnection on network issues
- WebGL rendering for smooth performance

## Use Cases

### Interactive Documentation
Let users try your CLI tool directly in the docs. No installation, no friction.

### Learning Platforms
Hands-on coding exercises with real Linux environments. Each student gets their own sandbox.

### Product Demos
Showcase command-line tools without requiring users to install anything.

### AI Agent Sandboxes
Give LLMs and AI assistants a safe execution environment. No more "run this on your machine" risks.

## How to Get Started

1. **Get a share code**: Create a terminal in your Rexec dashboard and click the share button.
2. **Add the script**: Include the embed script in your HTML.
3. **Initialize**: Call `Rexec.embed()` with your share code.

For authenticated access, generate an API token from your account settings.

Full documentation: [rexec.dev/docs/embed](/docs/embed)

## What's Next

We're working on:
- Session recording and playback for embedded terminals
- Pre-built environment templates (Node, Python, Go, etc.)
- Analytics and usage insights for embedded terminals
- White-label options for enterprise

## Try It Today

The embed widget is available now for all Rexec users. Free tier includes shared session access; Pro and Enterprise tiers get API tokens and on-demand container creation.

We'd love to hear how you're using embedded terminals. What would make this indispensable for your docs or platform? Let us know.

[Try the Embed Widget →](/docs/embed)
