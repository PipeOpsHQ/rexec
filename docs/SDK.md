# Rexec SDK Documentation

Rexec provides official SDKs for programmatically interacting with your sandboxed environments.

## Available SDKs

| SDK | Package | Documentation |
|-----|---------|---------------|
| **Go** | `github.com/PipeOpsHQ/rexec-go` | [Go SDK README](../sdk/go/README.md) |
| **JavaScript/TypeScript** | `@pipeopshq/rexec` | [JS SDK README](../sdk/js/README.md) |
| **Python** | `rexec` (PyPI) | [Python SDK README](../sdk/python/README.md) |
| **Rust** | `rexec` (crates.io) | [Rust SDK README](../sdk/rust/README.md) |
| **Ruby** | `rexec` (RubyGems) | [Ruby SDK README](../sdk/ruby/README.md) |
| **Java** | `io.pipeops:rexec` (Maven) | [Java SDK README](../sdk/java/README.md) |
| **C#/.NET** | `Rexec` (NuGet) | [.NET SDK README](../sdk/dotnet/README.md) |
| **PHP** | `pipeopshq/rexec` (Packagist) | [PHP SDK README](../sdk/php/README.md) |

## Getting an API Token

Before using the SDK, you need to generate an API token:

1. Log in to your Rexec instance
2. Go to **Settings** → **API Tokens**
3. Click **Generate Token**
4. Copy the token and store it securely

> ⚠️ **Security Note**: Never commit API tokens to version control. Use environment variables or secret management tools.

## Quick Start

### Go

```bash
go get github.com/PipeOpsHQ/rexec-go
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    rexec "github.com/PipeOpsHQ/rexec-go"
)

func main() {
    client := rexec.NewClient(
        os.Getenv("REXEC_URL"),
        os.Getenv("REXEC_TOKEN"),
    )

    ctx := context.Background()

    // Create a sandbox
    container, _ := client.Containers.Create(ctx, &rexec.CreateContainerRequest{
        Image: "ubuntu:24.04",
    })

    // Connect to terminal
    term, _ := client.Terminal.Connect(ctx, container.ID)
    defer term.Close()

    term.Write([]byte("echo 'Hello World'\n"))
}
```

### JavaScript/TypeScript

```bash
npm install @pipeopshq/rexec
```

```typescript
import { RexecClient } from '@pipeopshq/rexec';

const client = new RexecClient({
  baseURL: process.env.REXEC_URL,
  token: process.env.REXEC_TOKEN,
});

// Create a sandbox
const container = await client.containers.create({
  image: 'ubuntu:24.04',
});

// Connect to terminal
const terminal = await client.terminal.connect(container.id);
terminal.write('echo "Hello World"\n');
terminal.onData((data) => console.log(data));
```

### Python

```bash
pip install rexec
```

```python
import asyncio
from rexec import RexecClient

async def main():
    async with RexecClient(
        base_url=os.environ["REXEC_URL"],
        token=os.environ["REXEC_TOKEN"]
    ) as client:
        # Create a sandbox
        container = await client.containers.create("ubuntu:24.04")
        
        # Execute a command
        result = await client.containers.exec(container.id, "echo 'Hello World'")
        print(result.stdout)

asyncio.run(main())
```

### Rust

```bash
cargo add rexec
```

```rust
use rexec::RexecClient;

#[tokio::main]
async fn main() -> Result<(), rexec::Error> {
    let client = RexecClient::new(
        std::env::var("REXEC_URL")?,
        std::env::var("REXEC_TOKEN")?
    )?;

    // Create a sandbox
    let container = client.containers()
        .create("ubuntu:24.04")
        .await?;

    // Connect to terminal
    let mut term = client.terminal()
        .connect(&container.id)
        .await?;

    term.write(b"echo 'Hello World'\n").await?;
    Ok(())
}
```

### Ruby

```bash
gem install rexec
```

```ruby
require 'rexec'

client = Rexec::Client.new(
  ENV['REXEC_URL'],
  ENV['REXEC_TOKEN']
)

# Create a sandbox
container = client.containers.create('ubuntu:24.04')

# Execute a command
result = client.containers.exec(container.id, "echo 'Hello World'")
puts result.stdout
```

### Java

```xml
<dependency>
    <groupId>io.pipeops</groupId>
    <artifactId>rexec</artifactId>
    <version>1.0.0</version>
</dependency>
```

```java
import io.pipeops.rexec.*;

public class Example {
    public static void main(String[] args) throws RexecException {
        RexecClient client = new RexecClient(
            System.getenv("REXEC_URL"),
            System.getenv("REXEC_TOKEN")
        );

        // Create a sandbox
        Container container = client.containers().create("ubuntu:24.04");

        // Execute a command
        ExecResult result = client.containers().exec(
            container.getId(), "echo 'Hello World'"
        );
        System.out.println(result.getStdout());
    }
}
```

### C#/.NET

```bash
dotnet add package Rexec
```

```csharp
using Rexec;

var client = new RexecClient(
    Environment.GetEnvironmentVariable("REXEC_URL"),
    Environment.GetEnvironmentVariable("REXEC_TOKEN")
);

// Create a sandbox
var container = await client.Containers.CreateAsync("ubuntu:24.04");

// Execute a command
var result = await client.Containers.ExecAsync(
    container.Id, "echo 'Hello World'"
);
Console.WriteLine(result.Stdout);
```

### PHP

```bash
composer require pipeopshq/rexec
```

```php
<?php
require 'vendor/autoload.php';

use Rexec\RexecClient;

$client = new RexecClient(
    getenv('REXEC_URL'),
    getenv('REXEC_TOKEN')
);

// Create a sandbox
$container = $client->containers()->create('ubuntu:24.04');

// Execute a command
$result = $client->containers()->exec($container->id, "echo 'Hello World'");
echo $result->stdout;
```

## Core Concepts

### Containers

Containers are isolated Linux environments powered by Docker. Each container:

- Has its own filesystem
- Can run any Linux command
- Is isolated from other containers
- Can be started, stopped, and deleted

### Terminal Sessions

Terminal sessions provide real-time WebSocket connections to containers:

- Full PTY support
- Resizable terminals
- Binary data support for tools like vim, nano, etc.

### Files

The file API allows you to:

- List files and directories
- Upload files to containers
- Download files from containers
- Create directories

## API Endpoints

The SDKs wrap these REST API endpoints:

### Containers

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/containers` | List all containers |
| `POST` | `/api/containers` | Create a container |
| `GET` | `/api/containers/:id` | Get container details |
| `DELETE` | `/api/containers/:id` | Delete a container |
| `POST` | `/api/containers/:id/start` | Start a container |
| `POST` | `/api/containers/:id/stop` | Stop a container |

### Files

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/containers/:id/files/list` | List files |
| `GET` | `/api/containers/:id/files` | Download a file |
| `POST` | `/api/containers/:id/files` | Upload a file |
| `POST` | `/api/containers/:id/files/mkdir` | Create directory |
| `DELETE` | `/api/containers/:id/files` | Delete a file |

### WebSocket

| Endpoint | Description |
|----------|-------------|
| `/ws/terminal/:containerId` | Terminal connection |

## Use Cases

### CI/CD Integration

Run tests in isolated environments:

```typescript
const container = await client.containers.create({
  image: 'node:20',
  environment: { CI: 'true' }
});

const terminal = await client.terminal.connect(container.id);
terminal.write('npm install && npm test\n');
```

### Remote Development

Provide cloud development environments:

```go
container, _ := client.Containers.Create(ctx, &rexec.CreateContainerRequest{
    Image: "ubuntu:24.04",
    Environment: map[string]string{
        "EDITOR": "vim",
    },
})
```

### Education Platforms

Provide sandboxed coding environments for students:

```typescript
// Create a container per student
const studentContainer = await client.containers.create({
  image: 'python:3.12',
  name: `student-${studentId}`,
  labels: { course: 'intro-python' }
});
```

### Automated Testing

Run integration tests in clean environments:

```go
func TestMyApp(t *testing.T) {
    container, _ := client.Containers.Create(ctx, &rexec.CreateContainerRequest{
        Image: "golang:1.22",
    })
    defer client.Containers.Delete(ctx, container.ID)

    term, _ := client.Terminal.Connect(ctx, container.ID)
    term.Write([]byte("go test ./...\n"))
}
```

## Rate Limits

API requests are rate-limited to ensure fair usage:

- **Container creation**: 10 per minute
- **API requests**: 100 per minute
- **WebSocket connections**: 5 concurrent per user

## Error Handling

All SDKs provide structured error handling:

### Go

```go
container, err := client.Containers.Get(ctx, "invalid-id")
if err != nil {
    if apiErr, ok := err.(*rexec.APIError); ok {
        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
    }
}
```

### JavaScript/TypeScript

```typescript
try {
  await client.containers.get('invalid-id');
} catch (error) {
  if (error instanceof RexecError) {
    console.error(`API Error ${error.statusCode}: ${error.message}`);
  }
}
```

### Python

```python
from rexec import RexecException

try:
    container = await client.containers.get("invalid-id")
except RexecException as e:
    if e.status_code:
        print(f"API Error {e.status_code}: {e}")
```

### Rust

```rust
match client.containers().get("invalid-id").await {
    Ok(container) => println!("Got: {}", container.id),
    Err(rexec::Error::Api { status, message }) => {
        eprintln!("API Error {}: {}", status, message);
    }
    Err(e) => eprintln!("Error: {}", e),
}
```

### Ruby

```ruby
begin
  container = client.containers.get("invalid-id")
rescue Rexec::Error => e
  puts "Error: #{e.message}"
end
```

### Java

```java
try {
    Container container = client.containers().get("invalid-id");
} catch (RexecException e) {
    if (e.isApiError()) {
        System.out.println("API Error " + e.getStatusCode() + ": " + e.getMessage());
    }
}
```

### C#/.NET

```csharp
try {
    var container = await client.Containers.GetAsync("invalid-id");
} catch (RexecException ex) when (ex.IsApiError) {
    Console.WriteLine($"API Error {ex.StatusCode}: {ex.Message}");
}
```

### PHP

```php
try {
    $container = $client->containers()->get("invalid-id");
} catch (RexecException $e) {
    if ($e->isApiError()) {
        echo "API Error " . $e->getStatusCode() . ": " . $e->getMessage();
    }
}
```

## Best Practices

1. **Reuse clients**: Create one client instance and reuse it
2. **Handle errors**: Always check for and handle errors appropriately
3. **Clean up**: Delete containers when done to free resources
4. **Use environment variables**: Never hardcode tokens
5. **Set timeouts**: Configure appropriate timeouts for your use case

## Support

- [GitHub Issues](https://github.com/PipeOpsHQ/rexec/issues)
- [GitHub Discussions](https://github.com/PipeOpsHQ/rexec/discussions)
