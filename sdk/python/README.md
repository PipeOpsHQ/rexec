# Rexec Python SDK

Official Python SDK for [Rexec](https://github.com/PipeOpsHQ/rexec) - Terminal as a Service.

## Installation

```bash
pip install rexec
```

## Quick Start

```python
import asyncio
from rexec import RexecClient

async def main():
    async with RexecClient("https://your-instance.com", "your-token") as client:
        # Create a container
        container = await client.containers.create(
            image="ubuntu:24.04",
            name="my-sandbox"
        )
        print(f"Created container: {container.id}")

        # Connect to terminal
        async with client.terminal.connect(container.id) as term:
            await term.write(b"echo 'Hello from Rexec!'\n")
            
            # Read output
            data = await term.read()
            print(data.decode())

        # Clean up
        await client.containers.delete(container.id)

asyncio.run(main())
```

## API Reference

### RexecClient

```python
from rexec import RexecClient

# Create client (use as context manager for automatic cleanup)
async with RexecClient(
    base_url="https://your-instance.com",
    token="your-api-token",
    timeout=30.0,  # optional
) as client:
    ...

# Or manage manually
client = RexecClient("https://your-instance.com", "your-token")
try:
    ...
finally:
    await client.close()
```

### Containers

```python
# List all containers
containers = await client.containers.list()
for c in containers:
    print(f"{c.name}: {c.status}")

# Get a specific container
container = await client.containers.get(container_id)

# Create a container
container = await client.containers.create(
    image="ubuntu:24.04",
    name="my-container",
    environment={"MY_VAR": "value"},
    labels={"project": "demo"}
)

# Start a container
await client.containers.start(container_id)

# Stop a container
await client.containers.stop(container_id)

# Delete a container
await client.containers.delete(container_id)
```

### Files

```python
# List files in a directory
files = await client.files.list(container_id, "/home")
for f in files:
    prefix = "📁" if f.is_dir else "📄"
    print(f"{prefix} {f.name}")

# Download a file
content = await client.files.download(container_id, "/etc/passwd")
print(content.decode())

# Upload a file
await client.files.upload(
    container_id,
    "/home/script.py",
    b"print('hello world')"
)

# Create a directory
await client.files.mkdir(container_id, "/home/mydir")

# Delete a file
await client.files.delete(container_id, "/home/script.py")
```

### Terminal

```python
# Connect to terminal (recommended: use as context manager)
async with client.terminal.connect(
    container_id,
    cols=120,
    rows=40,
    timeout=30.0
) as term:
    # Send commands
    await term.write(b"ls -la\n")
    
    # Read output
    data = await term.read()
    print(data.decode())
    
    # Resize terminal
    await term.resize(150, 50)

# Or iterate over output
async with client.terminal.connect(container_id) as term:
    await term.write(b"echo hello && sleep 1 && echo world\n")
    async for data in term:
        print(data.decode(), end="")
```

## Examples

### Run a Script and Capture Output

```python
async def run_script(client: RexecClient, container_id: str, script: str) -> str:
    output = []
    
    async with client.terminal.connect(container_id) as term:
        await term.write(f"{script}\nexit\n".encode())
        
        async for data in term:
            output.append(data.decode())
    
    return "".join(output)

# Usage
result = await run_script(client, container.id, "apt update && apt install -y curl")
print(result)
```

### Batch Container Operations

```python
async def create_batch(client: RexecClient, count: int) -> list[Container]:
    import asyncio
    
    tasks = [
        client.containers.create(
            image="ubuntu:24.04",
            name=f"worker-{i}"
        )
        for i in range(count)
    ]
    
    return await asyncio.gather(*tasks)

# Create 5 containers in parallel
containers = await create_batch(client, 5)
```

### File Sync

```python
import os
from pathlib import Path

async def sync_directory(
    client: RexecClient,
    container_id: str,
    local_path: Path,
    remote_path: str
):
    """Sync a local directory to a container."""
    await client.files.mkdir(container_id, remote_path)
    
    for item in local_path.iterdir():
        remote_item = f"{remote_path}/{item.name}"
        
        if item.is_file():
            content = item.read_bytes()
            await client.files.upload(container_id, remote_item, content)
            print(f"Uploaded: {remote_item}")
        elif item.is_dir():
            await sync_directory(client, container_id, item, remote_item)
```

## Error Handling

```python
from rexec import RexecClient, RexecAPIError, RexecConnectionError

try:
    container = await client.containers.get("invalid-id")
except RexecAPIError as e:
    print(f"API Error {e.status_code}: {e.message}")
except RexecConnectionError as e:
    print(f"Connection Error: {e}")
```

## Type Hints

The SDK is fully typed for excellent IDE support:

```python
from rexec import (
    RexecClient,
    Container,
    CreateContainerRequest,
    FileInfo,
    Terminal,
)

container: Container = await client.containers.create(image="ubuntu:24.04")
files: list[FileInfo] = await client.files.list(container.id, "/")
```

## Requirements

- Python 3.9+
- `httpx` for HTTP requests
- `websockets` for terminal connections

## License

MIT License - see [LICENSE](../../LICENSE) for details.
