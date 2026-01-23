# Rexec Rust SDK

Official Rust SDK for [Rexec](https://github.com/PipeOpsHQ/rexec) - Terminal as a Service.

## Installation

Add to your `Cargo.toml`:

```toml
[dependencies]
rexec = "1.0"
tokio = { version = "1", features = ["full"] }
```

## Quick Start

```rust
use rexec::{RexecClient, CreateContainerRequest};

#[tokio::main]
async fn main() -> Result<(), rexec::Error> {
    let client = RexecClient::new(
        "https://your-instance.com",
        "your-api-token"
    );

    // Create a container
    let container = client.containers()
        .create(CreateContainerRequest::new("ubuntu:24.04")
            .name("my-sandbox"))
        .await?;

    println!("Created container: {}", container.id);

    // Connect to terminal
    let mut term = client.terminal().connect(&container.id).await?;
    term.write(b"echo 'Hello from Rexec!'\n").await?;

    // Read output
    if let Some(data) = term.read().await? {
        println!("Output: {}", String::from_utf8_lossy(&data));
    }

    // Clean up
    client.containers().delete(&container.id).await?;

    Ok(())
}
```

## API Reference

### RexecClient

```rust
use rexec::{RexecClient, ClientConfig};

// Simple creation
let client = RexecClient::new("https://your-instance.com", "your-token");

// With custom config
let client = RexecClient::with_config(
    ClientConfig::new("https://your-instance.com", "your-token")
        .timeout(60)
);
```

### Containers

```rust
// List all containers
let containers = client.containers().list().await?;
for c in containers {
    println!("{}: {}", c.name, c.status);
}

// Get a specific container
let container = client.containers().get("container-id").await?;

// Create a container with builder pattern
let container = client.containers()
    .create(CreateContainerRequest::new("ubuntu:24.04")
        .name("my-container")
        .env("MY_VAR", "value")
        .label("project", "demo"))
    .await?;

// Start a container
client.containers().start("container-id").await?;

// Stop a container
client.containers().stop("container-id").await?;

// Delete a container
client.containers().delete("container-id").await?;
```

### Files

```rust
// List files in a directory
let files = client.files().list("container-id", "/home").await?;
for f in files {
    let icon = if f.is_dir { "📁" } else { "📄" };
    println!("{} {} ({} bytes)", icon, f.name, f.size);
}

// Download a file
let content = client.files().download("container-id", "/etc/passwd").await?;
println!("{}", String::from_utf8_lossy(&content));

// Create a directory
client.files().mkdir("container-id", "/home/mydir").await?;

// Delete a file
client.files().delete("container-id", "/home/file.txt").await?;
```

### Terminal

```rust
// Connect to terminal
let mut term = client.terminal().connect("container-id").await?;

// Or with custom size
let mut term = client.terminal()
    .connect_with_size("container-id", 120, 40)
    .await?;

// Send commands
term.write(b"ls -la\n").await?;
term.write_str("echo hello\n").await?;

// Read output
while let Some(data) = term.read().await? {
    print!("{}", String::from_utf8_lossy(&data));
}

// Resize terminal
term.resize(150, 50).await?;

// Close connection
term.close().await?;
```

## Examples

### Run a Script

```rust
async fn run_script(
    client: &RexecClient,
    container_id: &str,
    script: &str,
) -> Result<String, rexec::Error> {
    let mut term = client.terminal().connect(container_id).await?;
    let mut output = String::new();

    term.write_str(&format!("{}\nexit\n", script)).await?;

    while let Some(data) = term.read().await? {
        output.push_str(&String::from_utf8_lossy(&data));
    }

    Ok(output)
}

// Usage
let output = run_script(&client, &container.id, "apt update && apt install -y curl").await?;
println!("{}", output);
```

### Concurrent Operations

```rust
use futures::future::join_all;

async fn create_batch(client: &RexecClient, count: usize) -> Vec<Container> {
    let futures: Vec<_> = (0..count)
        .map(|i| {
            client.containers().create(
                CreateContainerRequest::new("ubuntu:24.04")
                    .name(format!("worker-{}", i))
            )
        })
        .collect();

    join_all(futures)
        .await
        .into_iter()
        .filter_map(Result::ok)
        .collect()
}
```

## Error Handling

```rust
use rexec::{RexecClient, Error};

match client.containers().get("invalid-id").await {
    Ok(container) => println!("Found: {}", container.name),
    Err(Error::Api { status_code, message }) => {
        eprintln!("API Error {}: {}", status_code, message);
    }
    Err(Error::Connection(msg)) => {
        eprintln!("Connection Error: {}", msg);
    }
    Err(e) => eprintln!("Error: {}", e),
}
```

## Features

- `default` - Uses native TLS
- `rustls` - Use rustls instead of native TLS

```toml
[dependencies]
rexec = { version = "1.0", features = ["rustls"] }
```

## License

MIT License - see [LICENSE](../../LICENSE) for details.
