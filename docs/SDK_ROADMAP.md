# Rexec SDK Roadmap

This document outlines the plan for expanding SDK support across multiple programming languages.

## Current SDKs ✅

| Language | Package | Status |
|----------|---------|--------|
| **Go** | `github.com/PipeOpsHQ/rexec-go` | ✅ Complete |
| **JavaScript/TypeScript** | `@pipeopshq/rexec` | ✅ Complete |
| **Python** | `rexec` (PyPI) | ✅ Complete |
| **Rust** | `rexec` (crates.io) | ✅ Complete |

## Planned SDKs

### Phase 2: Medium Priority (Q2 2026)

| Language | Package | Priority | Notes |
|----------|---------|----------|-------|
| **Ruby** | `rexec` (RubyGems) | 🟡 Medium | Rails ecosystem, DevOps tools |
| **Java** | `com.pipeopshq:rexec` (Maven) | 🟡 Medium | Enterprise adoption |
| **C#/.NET** | `PipeOpsHQ.Rexec` (NuGet) | 🟡 Medium | Windows/Azure ecosystem |

### Phase 3: Community Driven (Q3+ 2026)

| Language | Package | Priority | Notes |
|----------|---------|----------|-------|
| **PHP** | `pipeopshq/rexec` (Packagist) | 🟢 Low | Web hosting platforms |
| **Swift** | `Rexec` (SPM) | 🟢 Low | macOS/iOS automation |
| **Kotlin** | `rexec` (Maven) | 🟢 Low | Android, modern JVM |

---

## SDK Requirements

Every SDK must implement these core features:

### Required Features

1. **Authentication**
   - API token (Bearer) authentication
   - Token refresh handling
   - Secure token storage recommendations

2. **Container Management**
   - `list()` - List all containers
   - `get(id)` - Get container details
   - `create(options)` - Create new container
   - `delete(id)` - Delete container
   - `start(id)` - Start container
   - `stop(id)` - Stop container

3. **File Operations**
   - `list(containerId, path)` - List directory contents
   - `download(containerId, path)` - Download file
   - `upload(containerId, path, data)` - Upload file
   - `mkdir(containerId, path)` - Create directory
   - `delete(containerId, path)` - Delete file/directory

4. **Terminal WebSocket**
   - Connect to container terminal
   - Send input (stdin)
   - Receive output (stdout/stderr)
   - Resize terminal (cols, rows)
   - Handle connection lifecycle

5. **Error Handling**
   - Typed/structured errors
   - HTTP status code mapping
   - Network error handling
   - Timeout handling

### Optional Features

- Retry logic with exponential backoff
- Connection pooling
- Async/await support (where applicable)
- Streaming file uploads
- Event emitters for terminal data

---

## SDK Structure Template

Each SDK should follow this structure:

```
sdk/<language>/
├── README.md           # Quick start, API reference
├── LICENSE             # MIT license
├── <package-config>    # go.mod, package.json, Cargo.toml, etc.
├── src/
│   ├── client.*        # Main client class/struct
│   ├── containers.*    # Container service
│   ├── files.*         # File service
│   ├── terminal.*      # Terminal/WebSocket service
│   ├── types.*         # Type definitions
│   └── errors.*        # Error types
├── examples/
│   ├── basic/          # Basic usage
│   ├── terminal/       # Interactive terminal
│   └── files/          # File operations
└── tests/
    └── ...             # Unit and integration tests
```

---

## Python SDK Design

### Package Structure

```
sdk/python/
├── README.md
├── pyproject.toml
├── rexec/
│   ├── __init__.py
│   ├── client.py
│   ├── containers.py
│   ├── files.py
│   ├── terminal.py
│   ├── types.py
│   └── exceptions.py
├── examples/
│   └── ...
└── tests/
    └── ...
```

### API Design

```python
from rexec import RexecClient

client = RexecClient(
    base_url="https://your-instance.com",
    token="your-api-token"
)

# Create container
container = await client.containers.create(
    image="ubuntu:24.04",
    name="my-sandbox"
)

# Connect to terminal
async with client.terminal.connect(container.id) as term:
    await term.write(b"echo hello\n")
    async for data in term:
        print(data.decode())

# File operations
files = await client.files.list(container.id, "/home")
content = await client.files.download(container.id, "/etc/passwd")
```

### Dependencies

- `httpx` - Async HTTP client
- `websockets` - WebSocket support
- `pydantic` - Type validation (optional)

---

## Rust SDK Design

### Package Structure

```
sdk/rust/
├── README.md
├── Cargo.toml
├── src/
│   ├── lib.rs
│   ├── client.rs
│   ├── containers.rs
│   ├── files.rs
│   ├── terminal.rs
│   ├── types.rs
│   └── error.rs
├── examples/
│   └── ...
└── tests/
    └── ...
```

### API Design

```rust
use rexec::RexecClient;

#[tokio::main]
async fn main() -> Result<(), rexec::Error> {
    let client = RexecClient::new(
        "https://your-instance.com",
        "your-api-token"
    );

    // Create container
    let container = client.containers()
        .create(CreateContainerRequest {
            image: "ubuntu:24.04".into(),
            name: Some("my-sandbox".into()),
            ..Default::default()
        })
        .await?;

    // Connect to terminal
    let mut term = client.terminal()
        .connect(&container.id)
        .await?;

    term.write(b"echo hello\n").await?;
    
    while let Some(data) = term.read().await? {
        print!("{}", String::from_utf8_lossy(&data));
    }

    Ok(())
}
```

### Dependencies

- `reqwest` - HTTP client
- `tokio-tungstenite` - WebSocket support
- `serde` - Serialization
- `thiserror` - Error handling

---

## Contributing SDKs

Community contributions for new language SDKs are welcome! Please:

1. Open an issue to discuss the SDK design
2. Follow the SDK structure template above
3. Implement all required features
4. Include comprehensive documentation
5. Add examples for common use cases
6. Include tests with >80% coverage
7. Submit a PR referencing the issue

---

## Version Strategy

All SDKs follow semantic versioning (SemVer):

- **Major**: Breaking API changes
- **Minor**: New features, backward compatible
- **Patch**: Bug fixes, backward compatible

SDK versions are independent but should aim to stay compatible with the latest Rexec API version.

---

## Release Process

1. Update version in package config
2. Update CHANGELOG
3. Run tests
4. Build distribution
5. Publish to package registry
6. Tag release in Git
7. Update documentation

---

## Metrics & Success Criteria

Track SDK adoption via:

- Package downloads (npm, PyPI, crates.io, etc.)
- GitHub stars on SDK repos
- Community contributions
- Issue/bug reports
- User feedback

Target: 1000+ weekly downloads per SDK within 6 months of release.
