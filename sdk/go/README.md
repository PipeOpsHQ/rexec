# Rexec Go SDK

Official Go SDK for [Rexec](https://github.com/PipeOpsHQ/rexec) - Terminal as a Service.

## Installation

```bash
go get github.com/PipeOpsHQ/rexec-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    rexec "github.com/PipeOpsHQ/rexec-go"
)

func main() {
    // Create client
    client := rexec.NewClient("https://your-rexec-instance.com", "your-api-token")

    ctx := context.Background()

    // Create a container
    container, err := client.Containers.Create(ctx, &rexec.CreateContainerRequest{
        Image: "ubuntu:24.04",
        Name:  "my-sandbox",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created container: %s\n", container.ID)

    // Connect to terminal
    term, err := client.Terminal.Connect(ctx, container.ID)
    if err != nil {
        log.Fatal(err)
    }
    defer term.Close()

    // Send a command
    term.Write([]byte("echo 'Hello from Rexec!'\n"))

    // Read output
    output, _ := term.Read()
    fmt.Printf("Output: %s\n", output)

    // Clean up
    client.Containers.Delete(ctx, container.ID)
}
```

## API Reference

### Client

```go
// Create a new client
client := rexec.NewClient(baseURL, apiToken)

// Use custom HTTP client
client.SetHTTPClient(&http.Client{Timeout: 60 * time.Second})
```

### Containers

```go
// List all containers
containers, err := client.Containers.List(ctx)

// Get a specific container
container, err := client.Containers.Get(ctx, containerID)

// Create a container
container, err := client.Containers.Create(ctx, &rexec.CreateContainerRequest{
    Image: "ubuntu:24.04",
    Name:  "my-container",
    Environment: map[string]string{
        "MY_VAR": "value",
    },
})

// Start a container
err := client.Containers.Start(ctx, containerID)

// Stop a container
err := client.Containers.Stop(ctx, containerID)

// Delete a container
err := client.Containers.Delete(ctx, containerID)
```

### Files

```go
// List files in a directory
files, err := client.Files.List(ctx, containerID, "/home")

// Download a file
data, err := client.Files.Download(ctx, containerID, "/home/file.txt")

// Create a directory
err := client.Files.Mkdir(ctx, containerID, "/home/newdir")
```

### Terminal

```go
// Connect to terminal
term, err := client.Terminal.Connect(ctx, containerID)
defer term.Close()

// Send input
term.Write([]byte("ls -la\n"))

// Read output
output, err := term.Read()

// Resize terminal
term.Resize(120, 40)
```

## Examples

### Run a Script

```go
func runScript(client *rexec.Client, containerID, script string) error {
    ctx := context.Background()
    
    term, err := client.Terminal.Connect(ctx, containerID)
    if err != nil {
        return err
    }
    defer term.Close()

    // Write script
    term.Write([]byte(script + "\n"))
    
    // Read output
    for {
        output, err := term.Read()
        if err != nil {
            break
        }
        fmt.Print(string(output))
    }
    
    return nil
}
```

### Interactive Session

```go
func interactiveSession(client *rexec.Client, containerID string) error {
    ctx := context.Background()
    
    term, err := client.Terminal.Connect(ctx, containerID)
    if err != nil {
        return err
    }
    defer term.Close()

    // Handle terminal resize
    term.Resize(80, 24)

    // Read from stdin and write to terminal
    go func() {
        buf := make([]byte, 1024)
        for {
            n, _ := os.Stdin.Read(buf)
            term.Write(buf[:n])
        }
    }()

    // Read from terminal and write to stdout
    for {
        output, err := term.Read()
        if err != nil {
            break
        }
        os.Stdout.Write(output)
    }
    
    return nil
}
```

## License

MIT License - see [LICENSE](../../LICENSE) for details.
