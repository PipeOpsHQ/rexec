<script lang="ts">
    import StatusIcon from "./icons/StatusIcon.svelte";

    export let onback: (() => void) | undefined = undefined;

    let copiedCommand = "";
    let activeTab = "javascript";

    function copyToClipboard(text: string, id: string) {
        navigator.clipboard.writeText(text);
        copiedCommand = id;
        setTimeout(() => { copiedCommand = ""; }, 2000);
    }

    function handleBack() {
        if (onback) onback();
    }

    const sdks = [
        { id: "go", name: "Go", install: "go get github.com/PipeOpsHQ/rexec-go", github: "https://github.com/PipeOpsHQ/rexec/tree/main/sdk/go" },
        { id: "javascript", name: "JavaScript / TypeScript", install: "npm install @pipeopshq/rexec", github: "https://github.com/PipeOpsHQ/rexec/tree/main/sdk/js" },
        { id: "python", name: "Python", install: "pip install rexec", github: "https://github.com/PipeOpsHQ/rexec/tree/main/sdk/python" },
        { id: "rust", name: "Rust", install: "cargo add rexec", github: "https://github.com/PipeOpsHQ/rexec/tree/main/sdk/rust" },
        { id: "ruby", name: "Ruby", install: "gem install rexec", github: "https://github.com/PipeOpsHQ/rexec/tree/main/sdk/ruby" },
        { id: "java", name: "Java", install: "<!-- Maven -->\n<dependency>\n  <groupId>io.pipeops</groupId>\n  <artifactId>rexec</artifactId>\n</dependency>", github: "https://github.com/PipeOpsHQ/rexec/tree/main/sdk/java" },
        { id: "dotnet", name: "C# / .NET", install: "dotnet add package Rexec", github: "https://github.com/PipeOpsHQ/rexec/tree/main/sdk/dotnet" },
        { id: "php", name: "PHP", install: "composer require pipeopshq/rexec", github: "https://github.com/PipeOpsHQ/rexec/tree/main/sdk/php" },
    ];

    const codeExamples: Record<string, string> = {
        go: `package main

import (
    "fmt"
    rexec "github.com/PipeOpsHQ/rexec-go"
)

func main() {
    client := rexec.NewClient("https://rexec.sh", "YOUR_API_TOKEN")

    // Create a sandboxed container
    container, _ := client.Containers.Create(rexec.CreateOptions{
        Image: "ubuntu:24.04",
    })

    // Execute a command
    result, _ := client.Containers.Exec(container.ID, "echo Hello from Rexec!")
    fmt.Println(result.Stdout)

    // Interactive terminal
    term, _ := client.Terminal.Connect(container.ID)
    term.OnData(func(data string) { fmt.Print(data) })
    term.Write("ls -la\\n")
}`,
        javascript: `import { RexecClient } from '@pipeopshq/rexec';

const client = new RexecClient({ 
  baseURL: 'https://rexec.sh', 
  token: 'YOUR_API_TOKEN' 
});

// Create a sandboxed container
const container = await client.containers.create({ 
  image: 'ubuntu:24.04' 
});

// Execute a command
const result = await client.containers.exec(
  container.id, 
  'echo "Hello from Rexec!"'
);
console.log(result.stdout);

// Interactive terminal
const terminal = await client.terminal.connect(container.id);
terminal.onData((data) => console.log(data));
terminal.write('ls -la\\n');`,
        python: `from rexec import RexecClient

async with RexecClient("https://rexec.sh", "YOUR_API_TOKEN") as client:
    # Create a sandboxed container
    container = await client.containers.create(image="ubuntu:24.04")

    # Execute a command
    result = await client.containers.exec(container.id, "echo Hello from Rexec!")
    print(result.stdout)

    # Interactive terminal
    async with client.terminal.connect(container.id) as term:
        term.on_data(lambda data: print(data, end=""))
        await term.write("ls -la\\n")`,
        rust: `use rexec::RexecClient;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = RexecClient::new("https://rexec.sh", "YOUR_API_TOKEN");

    // Create a sandboxed container
    let container = client.containers()
        .create("ubuntu:24.04")
        .await?;

    // Execute a command
    let result = client.containers()
        .exec(&container.id, "echo Hello from Rexec!")
        .await?;
    println!("{}", result.stdout);

    // Interactive terminal
    let mut term = client.terminal().connect(&container.id).await?;
    term.on_data(|data| print!("{}", data));
    term.write("ls -la\\n").await?;

    Ok(())
}`,
        ruby: `require 'rexec'

client = Rexec::Client.new(
  base_url: 'https://rexec.sh',
  token: 'YOUR_API_TOKEN'
)

# Create a sandboxed container
container = client.containers.create(image: 'ubuntu:24.04')

# Execute a command
result = client.containers.exec(container.id, 'echo Hello from Rexec!')
puts result.stdout

# Interactive terminal
terminal = client.terminal.connect(container.id)
terminal.on_data { |data| print data }
terminal.write("ls -la\\n")`,
        java: `import io.pipeops.rexec.RexecClient;
import io.pipeops.rexec.Container;
import io.pipeops.rexec.ExecResult;

public class Main {
    public static void main(String[] args) {
        RexecClient client = new RexecClient(
            "https://rexec.sh", 
            "YOUR_API_TOKEN"
        );

        // Create a sandboxed container
        Container container = client.containers()
            .create("ubuntu:24.04");

        // Execute a command
        ExecResult result = client.containers()
            .exec(container.getId(), "echo Hello from Rexec!");
        System.out.println(result.getStdout());

        // Interactive terminal
        Terminal terminal = client.terminal().connect(container.getId());
        terminal.onData(data -> System.out.print(data));
        terminal.write("ls -la\\n");
    }
}`,
        dotnet: `using Rexec;

var client = new RexecClient("https://rexec.sh", "YOUR_API_TOKEN");

// Create a sandboxed container
var container = await client.Containers.CreateAsync(new CreateOptions {
    Image = "ubuntu:24.04"
});

// Execute a command
var result = await client.Containers.ExecAsync(
    container.Id, 
    "echo Hello from Rexec!"
);
Console.WriteLine(result.Stdout);

// Interactive terminal
var terminal = await client.Terminal.ConnectAsync(container.Id);
terminal.OnData += (data) => Console.Write(data);
await terminal.WriteAsync("ls -la\\n");`,
        php: `<?php
use PipeOps\\Rexec\\RexecClient;

$client = new RexecClient('https://rexec.sh', 'YOUR_API_TOKEN');

// Create a sandboxed container
$container = $client->containers()->create(['image' => 'ubuntu:24.04']);

// Execute a command
$result = $client->containers()->exec($container->id, 'echo Hello from Rexec!');
echo $result->stdout;

// Interactive terminal
$terminal = $client->terminal()->connect($container->id);
$terminal->onData(function($data) { echo $data; });
$terminal->write("ls -la\\n");`,
    };
</script>

<div class="docs-page">
    <button class="back-btn" onclick={handleBack}>
        <span class="back-icon">←</span>
        <span>Back</span>
    </button>

    <div class="docs-content">
        <header class="docs-header">
            <div class="header-icon">
                <StatusIcon status="code" size={48} />
            </div>
            <h1>rexec SDKs</h1>
            <p class="subtitle">Official SDKs for programmatic access to Rexec cloud terminals</p>
        </header>

        <section class="docs-section">
            <h2>What are the rexec SDKs?</h2>
            <p>
                The Rexec SDKs provide programmatic access to create, manage, and interact with 
                sandboxed terminal environments. Build AI agents, automated testing pipelines, 
                educational platforms, and more with full API access.
            </p>
            <div class="feature-grid">
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Container Management</h4>
                    <p>Create, start, stop, and delete sandboxed environments</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Command Execution</h4>
                    <p>Run commands and capture stdout/stderr</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>File Operations</h4>
                    <p>Upload, download, and manage files in containers</p>
                </div>
                <div class="feature-card">
                    <StatusIcon status="ready" size={20} />
                    <h4>Interactive Terminal</h4>
                    <p>WebSocket-based real-time terminal sessions</p>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Available SDKs</h2>
            <div class="sdk-grid">
                {#each sdks as sdk}
                    <a href={sdk.github} target="_blank" class="sdk-card">
                        <div class="sdk-icon"><StatusIcon status="code" size={20} /></div>
                        <div class="sdk-info">
                            <h4>{sdk.name}</h4>
                            <code>{sdk.install.split('\n')[0]}</code>
                        </div>
                    </a>
                {/each}
            </div>
        </section>

        <section class="docs-section">
            <h2>Installation</h2>
            <p>Install the SDK for your language:</p>
            
            <div class="install-tabs">
                {#each sdks as sdk}
                    <button 
                        class="tab" 
                        class:active={activeTab === sdk.id}
                        onclick={() => activeTab = sdk.id}
                    >
                        {sdk.name.split(' ')[0]}
                    </button>
                {/each}
            </div>
            
            <div class="code-block">
                <code>{sdks.find(s => s.id === activeTab)?.install || ''}</code>
                <button 
                    class="copy-btn"
                    onclick={() => copyToClipboard(sdks.find(s => s.id === activeTab)?.install || '', 'install')}
                >
                    {copiedCommand === 'install' ? 'Copied!' : 'Copy'}
                </button>
            </div>
        </section>

        <section class="docs-section">
            <h2>Quick Start</h2>
            <p>Get started with a simple example:</p>
            
            <div class="install-tabs">
                {#each sdks as sdk}
                    <button 
                        class="tab" 
                        class:active={activeTab === sdk.id}
                        onclick={() => activeTab = sdk.id}
                    >
                        {sdk.name.split(' ')[0]}
                    </button>
                {/each}
            </div>
            
            <div class="code-block large">
                <pre><code>{codeExamples[activeTab] || ''}</code></pre>
                <button 
                    class="copy-btn"
                    onclick={() => copyToClipboard(codeExamples[activeTab] || '', 'example')}
                >
                    {copiedCommand === 'example' ? 'Copied!' : 'Copy'}
                </button>
            </div>
        </section>

        <section class="docs-section">
            <h2>Authentication</h2>
            <p>All SDKs use Bearer token authentication. Get your API token from the dashboard:</p>
            
            <div class="auth-steps">
                <div class="step">
                    <span class="step-num">1</span>
                    <div class="step-content">
                        <h4>Go to Account Settings</h4>
                        <p>Navigate to <a href="/account/api-tokens">Account → API Tokens</a></p>
                    </div>
                </div>
                <div class="step">
                    <span class="step-num">2</span>
                    <div class="step-content">
                        <h4>Create New Token</h4>
                        <p>Click "Generate Token" and give it a descriptive name</p>
                    </div>
                </div>
                <div class="step">
                    <span class="step-num">3</span>
                    <div class="step-content">
                        <h4>Use in SDK</h4>
                        <p>Pass the token when initializing the client</p>
                    </div>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Core API Concepts</h2>
            
            <div class="api-concepts">
                <div class="concept">
                    <h4><StatusIcon status="box" size={18} /> Containers</h4>
                    <p>Sandboxed Linux environments with configurable images, resources, and lifecycle.</p>
                    <ul>
                        <li><code>create()</code> - Create a new container</li>
                        <li><code>get(id)</code> - Get container details</li>
                        <li><code>list()</code> - List all containers</li>
                        <li><code>start(id)</code> / <code>stop(id)</code> - Lifecycle control</li>
                        <li><code>exec(id, command)</code> - Execute commands</li>
                        <li><code>delete(id)</code> - Remove container</li>
                    </ul>
                </div>
                
                <div class="concept">
                    <h4><StatusIcon status="file" size={18} /> Files</h4>
                    <p>Upload, download, and manage files within containers.</p>
                    <ul>
                        <li><code>list(containerId, path)</code> - List directory contents</li>
                        <li><code>read(containerId, path)</code> - Read file contents</li>
                        <li><code>write(containerId, path, content)</code> - Write file</li>
                        <li><code>delete(containerId, path)</code> - Delete file</li>
                    </ul>
                </div>
                
                <div class="concept">
                    <h4><StatusIcon status="terminal" size={18} /> Terminal</h4>
                    <p>WebSocket-based interactive terminal sessions.</p>
                    <ul>
                        <li><code>connect(containerId)</code> - Open terminal session</li>
                        <li><code>write(data)</code> - Send input to terminal</li>
                        <li><code>onData(callback)</code> - Receive terminal output</li>
                        <li><code>resize(cols, rows)</code> - Resize terminal</li>
                        <li><code>close()</code> - Close session</li>
                    </ul>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Use Cases</h2>
            <div class="use-cases-grid">
                <div class="use-case-card">
                    <StatusIcon status="ai" size={24} />
                    <h4>AI Code Agents</h4>
                    <p>Give LLMs safe sandbox environments to execute and test code</p>
                </div>
                <div class="use-case-card">
                    <StatusIcon status="workflow" size={24} />
                    <h4>CI/CD Pipelines</h4>
                    <p>Run isolated build and test environments on demand</p>
                </div>
                <div class="use-case-card">
                    <StatusIcon status="book" size={24} />
                    <h4>Education Platforms</h4>
                    <p>Provide students with instant coding environments</p>
                </div>
                <div class="use-case-card">
                    <StatusIcon status="bug" size={24} />
                    <h4>Automated Testing</h4>
                    <p>Spin up clean environments for integration tests</p>
                </div>
            </div>
        </section>

        <section class="docs-section">
            <h2>Resources</h2>
            <div class="resources-grid">
                <a href="https://github.com/PipeOpsHQ/rexec/blob/main/docs/SDK.md" target="_blank" class="resource-link">
                    <StatusIcon status="book" size={20} />
                    <div>
                        <h4>Full Documentation</h4>
                        <p>Complete API reference and guides</p>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/blob/main/docs/SDK_GETTING_STARTED.md" target="_blank" class="resource-link">
                    <StatusIcon status="rocket" size={20} />
                    <div>
                        <h4>Getting Started Guide</h4>
                        <p>Step-by-step tutorial</p>
                    </div>
                </a>
                <a href="https://github.com/PipeOpsHQ/rexec/tree/main/sdk" target="_blank" class="resource-link">
                    <StatusIcon status="code" size={20} />
                    <div>
                        <h4>Source Code</h4>
                        <p>SDK source on GitHub</p>
                    </div>
                </a>
                <a href="/account/api-tokens" class="resource-link">
                    <StatusIcon status="key" size={20} />
                    <div>
                        <h4>API Tokens</h4>
                        <p>Generate authentication tokens</p>
                    </div>
                </a>
            </div>
        </section>
    </div>
</div>

<style>
    .docs-page {
        max-width: 900px;
        margin: 0 auto;
        padding: 24px;
    }

    .back-btn {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 8px 16px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text);
        cursor: pointer;
        font-size: 14px;
        margin-bottom: 24px;
        transition: all 0.2s ease;
    }

    .back-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    .back-icon {
        font-size: 16px;
    }

    .docs-content {
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 12px;
        padding: 32px;
    }

    .docs-header {
        text-align: center;
        margin-bottom: 40px;
        padding-bottom: 32px;
        border-bottom: 1px solid var(--border);
    }

    .header-icon {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 80px;
        height: 80px;
        background: rgba(0, 255, 65, 0.1);
        border-radius: 50%;
        margin-bottom: 16px;
        color: var(--accent);
    }

    .docs-header h1 {
        font-size: 32px;
        font-weight: 700;
        margin-bottom: 8px;
        color: var(--text);
    }

    .subtitle {
        color: var(--text-muted);
        font-size: 16px;
    }

    .docs-section {
        margin-bottom: 40px;
    }

    .docs-section h2 {
        font-size: 20px;
        font-weight: 600;
        margin-bottom: 16px;
        color: var(--text);
    }

    .docs-section p {
        color: var(--text-muted);
        line-height: 1.6;
        margin-bottom: 16px;
    }

    .feature-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 16px;
        margin-top: 20px;
    }

    .feature-card {
        padding: 16px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
    }

    .feature-card h4 {
        font-size: 14px;
        font-weight: 600;
        margin: 12px 0 6px;
        color: var(--text);
    }

    .feature-card p {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0;
    }

    .feature-card :global(.status-icon) {
        color: var(--accent);
    }

    .sdk-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 12px;
        margin-top: 16px;
    }

    .sdk-card {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 14px 16px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
        text-decoration: none;
        transition: all 0.2s ease;
    }

    .sdk-card:hover {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
    }

    .sdk-icon {
        color: var(--accent);
    }

    .sdk-info h4 {
        font-size: 14px;
        font-weight: 600;
        color: var(--text);
        margin: 0 0 4px;
    }

    .sdk-info code {
        font-family: var(--font-mono);
        font-size: 11px;
        color: var(--text-muted);
    }

    .install-tabs {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        margin-bottom: 12px;
    }

    .tab {
        padding: 6px 12px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 6px;
        color: var(--text-muted);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .tab:hover {
        border-color: var(--accent);
        color: var(--text);
    }

    .tab.active {
        background: var(--accent);
        border-color: var(--accent);
        color: #000;
    }

    .code-block {
        position: relative;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 16px;
        font-family: var(--font-mono);
        font-size: 13px;
        overflow-x: auto;
    }

    .code-block.large {
        padding: 20px;
    }

    .code-block pre {
        margin: 0;
        white-space: pre-wrap;
        word-break: break-word;
    }

    .code-block code {
        color: var(--text);
        line-height: 1.5;
    }

    .copy-btn {
        position: absolute;
        top: 8px;
        right: 8px;
        padding: 6px 12px;
        background: var(--bg-secondary);
        border: 1px solid var(--border);
        border-radius: 4px;
        color: var(--text-muted);
        font-size: 12px;
        cursor: pointer;
        transition: all 0.2s ease;
    }

    .copy-btn:hover {
        border-color: var(--accent);
        color: var(--accent);
    }

    .auth-steps {
        display: flex;
        flex-direction: column;
        gap: 16px;
        margin-top: 16px;
    }

    .step {
        display: flex;
        align-items: flex-start;
        gap: 16px;
    }

    .step-num {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 28px;
        height: 28px;
        background: var(--accent);
        color: #000;
        font-weight: 600;
        font-size: 14px;
        border-radius: 50%;
        flex-shrink: 0;
    }

    .step-content h4 {
        font-size: 14px;
        font-weight: 600;
        margin: 0 0 4px;
        color: var(--text);
    }

    .step-content p {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0;
    }

    .step-content a {
        color: var(--accent);
        text-decoration: none;
    }

    .step-content a:hover {
        text-decoration: underline;
    }

    .api-concepts {
        display: flex;
        flex-direction: column;
        gap: 24px;
        margin-top: 16px;
    }

    .concept {
        padding: 20px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
    }

    .concept h4 {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 16px;
        font-weight: 600;
        margin: 0 0 8px;
        color: var(--text);
    }

    .concept h4 :global(.status-icon) {
        color: var(--accent);
    }

    .concept p {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0 0 12px;
    }

    .concept ul {
        margin: 0;
        padding: 0;
        list-style: none;
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 6px;
    }

    .concept li {
        font-family: var(--font-mono);
        font-size: 12px;
        color: var(--text-muted);
    }

    .concept li code {
        color: var(--accent);
    }

    .use-cases-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 16px;
        margin-top: 16px;
    }

    .use-case-card {
        padding: 20px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
        text-align: center;
    }

    .use-case-card :global(.status-icon) {
        color: var(--accent);
    }

    .use-case-card h4 {
        font-size: 14px;
        font-weight: 600;
        margin: 12px 0 8px;
        color: var(--text);
    }

    .use-case-card p {
        font-size: 13px;
        color: var(--text-muted);
        margin: 0;
    }

    .resources-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 12px;
        margin-top: 16px;
    }

    .resource-link {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px;
        background: var(--bg-tertiary);
        border: 1px solid var(--border);
        border-radius: 8px;
        text-decoration: none;
        transition: all 0.2s ease;
    }

    .resource-link:hover {
        border-color: var(--accent);
        background: rgba(0, 255, 65, 0.05);
    }

    .resource-link :global(.status-icon) {
        color: var(--accent);
        flex-shrink: 0;
    }

    .resource-link h4 {
        font-size: 14px;
        font-weight: 600;
        margin: 0 0 2px;
        color: var(--text);
    }

    .resource-link p {
        font-size: 12px;
        color: var(--text-muted);
        margin: 0;
    }

    @media (max-width: 768px) {
        .docs-page {
            padding: 16px;
        }

        .docs-content {
            padding: 20px;
        }

        .docs-header h1 {
            font-size: 24px;
        }

        .install-tabs {
            flex-wrap: nowrap;
            overflow-x: auto;
            padding-bottom: 8px;
        }

        .tab {
            white-space: nowrap;
        }
    }
</style>
