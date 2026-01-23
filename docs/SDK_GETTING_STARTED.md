# SDK Getting Started Guide

This guide walks you through setting up and using a Rexec SDK to programmatically interact with sandboxed environments.

## Prerequisites

1. A running Rexec instance (self-hosted or cloud)
2. An API token (see [Getting a Token](#getting-an-api-token))
3. Your preferred programming language installed

## Getting an API Token

1. Log in to your Rexec dashboard
2. Navigate to **Settings** → **API Tokens**
3. Click **Generate Token**
4. Copy and securely store the token

> ⚠️ **Security**: Never commit tokens to version control. Use environment variables.

```bash
export REXEC_URL="https://your-instance.com"
export REXEC_TOKEN="your-api-token"
```

## Choose Your SDK

| Language | Install Command |
|----------|----------------|
| Go | `go get github.com/PipeOpsHQ/rexec-go` |
| JavaScript/TypeScript | `npm install @pipeopshq/rexec` |
| Python | `pip install rexec` |
| Rust | `cargo add rexec` |
| Ruby | `gem install rexec` |
| Java | Add Maven dependency (see below) |
| C#/.NET | `dotnet add package Rexec` |
| PHP | `composer require pipeopshq/rexec` |

## Step 1: Create a Client

### Go
```go
import rexec "github.com/PipeOpsHQ/rexec-go"

client := rexec.NewClient(os.Getenv("REXEC_URL"), os.Getenv("REXEC_TOKEN"))
```

### JavaScript/TypeScript
```typescript
import { RexecClient } from '@pipeopshq/rexec';

const client = new RexecClient({
  baseURL: process.env.REXEC_URL,
  token: process.env.REXEC_TOKEN,
});
```

### Python
```python
from rexec import RexecClient

client = RexecClient(
    base_url=os.environ["REXEC_URL"],
    token=os.environ["REXEC_TOKEN"]
)
```

### Rust
```rust
use rexec::RexecClient;

let client = RexecClient::new(
    std::env::var("REXEC_URL")?,
    std::env::var("REXEC_TOKEN")?
)?;
```

### Ruby
```ruby
require 'rexec'

client = Rexec::Client.new(ENV['REXEC_URL'], ENV['REXEC_TOKEN'])
```

### Java
```java
import io.pipeops.rexec.RexecClient;

RexecClient client = new RexecClient(
    System.getenv("REXEC_URL"),
    System.getenv("REXEC_TOKEN")
);
```

### C#/.NET
```csharp
using Rexec;

var client = new RexecClient(
    Environment.GetEnvironmentVariable("REXEC_URL"),
    Environment.GetEnvironmentVariable("REXEC_TOKEN")
);
```

### PHP
```php
use Rexec\RexecClient;

$client = new RexecClient(getenv('REXEC_URL'), getenv('REXEC_TOKEN'));
```

## Step 2: Create a Container

### Go
```go
container, err := client.Containers.Create(ctx, &rexec.CreateContainerRequest{
    Image: "ubuntu:24.04",
    Name:  "my-sandbox",
})
```

### JavaScript/TypeScript
```typescript
const container = await client.containers.create({
  image: 'ubuntu:24.04',
  name: 'my-sandbox',
});
```

### Python
```python
container = await client.containers.create(
    image="ubuntu:24.04",
    name="my-sandbox"
)
```

### Rust
```rust
let container = client.containers()
    .create("ubuntu:24.04")
    .name("my-sandbox")
    .send()
    .await?;
```

### Ruby
```ruby
container = client.containers.create('ubuntu:24.04', name: 'my-sandbox')
```

### Java
```java
Container container = client.containers().create(
    new CreateContainerRequest("ubuntu:24.04").setName("my-sandbox")
);
```

### C#/.NET
```csharp
var container = await client.Containers.CreateAsync(
    new CreateContainerRequest("ubuntu:24.04") { Name = "my-sandbox" }
);
```

### PHP
```php
$container = $client->containers()->create('ubuntu:24.04', ['name' => 'my-sandbox']);
```

## Step 3: Start the Container

### Go
```go
err := client.Containers.Start(ctx, container.ID)
```

### JavaScript/TypeScript
```typescript
await client.containers.start(container.id);
```

### Python
```python
await client.containers.start(container.id)
```

### Rust
```rust
client.containers().start(&container.id).await?;
```

### Ruby
```ruby
client.containers.start(container.id)
```

### Java
```java
client.containers().start(container.getId());
```

### C#/.NET
```csharp
await client.Containers.StartAsync(container.Id);
```

### PHP
```php
$client->containers()->start($container->id);
```

## Step 4: Execute Commands

### Go
```go
result, err := client.Containers.Exec(ctx, container.ID, []string{"echo", "Hello World"})
fmt.Println(result.Stdout)
```

### JavaScript/TypeScript
```typescript
const result = await client.containers.exec(container.id, 'echo "Hello World"');
console.log(result.stdout);
```

### Python
```python
result = await client.containers.exec(container.id, "echo 'Hello World'")
print(result.stdout)
```

### Rust
```rust
let result = client.containers()
    .exec(&container.id, &["echo", "Hello World"])
    .await?;
println!("{}", result.stdout);
```

### Ruby
```ruby
result = client.containers.exec(container.id, "echo 'Hello World'")
puts result.stdout
```

### Java
```java
ExecResult result = client.containers().exec(container.getId(), "echo 'Hello World'");
System.out.println(result.getStdout());
```

### C#/.NET
```csharp
var result = await client.Containers.ExecAsync(container.Id, "echo 'Hello World'");
Console.WriteLine(result.Stdout);
```

### PHP
```php
$result = $client->containers()->exec($container->id, "echo 'Hello World'");
echo $result->stdout;
```

## Step 5: Work with Files

### List Files

```go
// Go
files, err := client.Files.List(ctx, container.ID, "/home")
```

```typescript
// JavaScript
const files = await client.files.list(container.id, '/home');
```

```python
# Python
files = await client.files.list(container.id, "/home")
```

### Read Files

```go
// Go
content, err := client.Files.Download(ctx, container.ID, "/etc/hostname")
```

```typescript
// JavaScript
const content = await client.files.read(container.id, '/etc/hostname');
```

```python
# Python
content = await client.files.read(container.id, "/etc/hostname")
```

### Write Files

```go
// Go
err := client.Files.Upload(ctx, container.ID, "/app/script.sh", []byte("#!/bin/bash\necho hello"))
```

```typescript
// JavaScript
await client.files.write(container.id, '/app/script.sh', '#!/bin/bash\necho hello');
```

```python
# Python
await client.files.write(container.id, "/app/script.sh", "#!/bin/bash\necho hello")
```

## Step 6: Interactive Terminal

### Go
```go
term, err := client.Terminal.Connect(ctx, container.ID)
defer term.Close()

term.OnData(func(data []byte) {
    fmt.Print(string(data))
})

term.Write([]byte("ls -la\n"))
```

### JavaScript/TypeScript
```typescript
const terminal = await client.terminal.connect(container.id);

terminal.onData((data) => process.stdout.write(data));
terminal.write('ls -la\n');

// Later: terminal.close();
```

### Python
```python
async with client.terminal.connect(container.id) as term:
    await term.write(b"ls -la\n")
    async for data in term:
        print(data.decode(), end="")
```

### Rust
```rust
let mut term = client.terminal().connect(&container.id).await?;

term.write(b"ls -la\n").await?;

while let Some(data) = term.read().await? {
    print!("{}", String::from_utf8_lossy(&data));
}
```

## Step 7: Clean Up

### Go
```go
client.Containers.Stop(ctx, container.ID)
client.Containers.Delete(ctx, container.ID)
```

### JavaScript/TypeScript
```typescript
await client.containers.stop(container.id);
await client.containers.delete(container.id);
```

### Python
```python
await client.containers.stop(container.id)
await client.containers.delete(container.id)
```

### Rust
```rust
client.containers().stop(&container.id).await?;
client.containers().delete(&container.id).await?;
```

## Complete Examples

See the `examples/` directory in each SDK for complete working examples:

- `examples/basic/` - Basic container operations
- `examples/terminal/` - Interactive terminal usage
- `examples/files/` - File upload/download
- `examples/ci/` - CI/CD integration

## Next Steps

- Read the [SDK Reference](SDK.md) for detailed API documentation
- Check out [Use Cases](SDK.md#use-cases) for real-world examples
- See [Error Handling](SDK.md#error-handling) for robust error management
- Join [GitHub Discussions](https://github.com/PipeOpsHQ/rexec/discussions) for community support
