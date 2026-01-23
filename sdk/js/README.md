# Rexec JavaScript/TypeScript SDK

Official JavaScript/TypeScript SDK for [Rexec](https://github.com/PipeOpsHQ/rexec) - Terminal as a Service.

## Installation

```bash
npm install @pipeopshq/rexec
# or
yarn add @pipeopshq/rexec
# or
pnpm add @pipeopshq/rexec
```

For Node.js environments, you'll also need the `ws` package:

```bash
npm install ws
```

## Quick Start

```typescript
import { RexecClient } from '@pipeopshq/rexec';

const client = new RexecClient({
  baseURL: 'https://your-rexec-instance.com',
  token: 'your-api-token'
});

// Create a container
const container = await client.containers.create({
  image: 'ubuntu:24.04',
  name: 'my-sandbox'
});

console.log(`Created container: ${container.id}`);

// Connect to terminal
const terminal = await client.terminal.connect(container.id);

terminal.onData((data) => {
  console.log('Output:', data);
});

terminal.write('echo "Hello from Rexec!"\n');

// Clean up
await client.containers.delete(container.id);
```

## API Reference

### RexecClient

```typescript
import { RexecClient } from '@pipeopshq/rexec';

const client = new RexecClient({
  baseURL: 'https://your-rexec-instance.com',
  token: 'your-api-token',
  fetch: customFetch // optional custom fetch implementation
});
```

### Containers

```typescript
// List all containers
const containers = await client.containers.list();

// Get a specific container
const container = await client.containers.get(containerId);

// Create a container
const container = await client.containers.create({
  image: 'ubuntu:24.04',
  name: 'my-container',
  environment: {
    MY_VAR: 'value'
  }
});

// Start a container
await client.containers.start(containerId);

// Stop a container
await client.containers.stop(containerId);

// Delete a container
await client.containers.delete(containerId);
```

### Files

```typescript
// List files in a directory
const files = await client.files.list(containerId, '/home');

// Download a file
const data = await client.files.download(containerId, '/home/file.txt');

// Create a directory
await client.files.mkdir(containerId, '/home/newdir');
```

### Terminal

```typescript
// Connect to terminal
const terminal = await client.terminal.connect(containerId, {
  cols: 120,
  rows: 40
});

// Send input
terminal.write('ls -la\n');

// Handle output
terminal.onData((data) => {
  console.log(data);
});

// Handle close
terminal.onClose(() => {
  console.log('Terminal closed');
});

// Handle errors
terminal.onError((error) => {
  console.error('Terminal error:', error);
});

// Resize terminal
terminal.resize(150, 50);

// Close connection
terminal.close();
```

## Examples

### Run a Script

```typescript
async function runScript(client: RexecClient, containerId: string, script: string): Promise<string> {
  const terminal = await client.terminal.connect(containerId);
  
  return new Promise((resolve, reject) => {
    let output = '';
    
    terminal.onData((data) => {
      output += data.toString();
    });
    
    terminal.onClose(() => {
      resolve(output);
    });
    
    terminal.onError(reject);
    
    terminal.write(script + '\n');
    terminal.write('exit\n');
  });
}

const output = await runScript(client, container.id, 'apt update && apt install -y nodejs');
console.log(output);
```

### Browser Usage

```html
<script type="module">
import { RexecClient } from 'https://unpkg.com/@pipeopshq/rexec/dist/index.mjs';

const client = new RexecClient({
  baseURL: 'https://your-rexec-instance.com',
  token: 'your-api-token'
});

// List containers
const containers = await client.containers.list();
console.log(containers);
</script>
```

### With xterm.js

```typescript
import { Terminal } from 'xterm';
import { RexecClient } from '@pipeopshq/rexec';

const xterm = new Terminal();
xterm.open(document.getElementById('terminal'));

const client = new RexecClient({
  baseURL: 'https://your-rexec-instance.com',
  token: 'your-api-token'
});

const container = await client.containers.create({ image: 'ubuntu:24.04' });
const rexecTerminal = await client.terminal.connect(container.id, {
  cols: xterm.cols,
  rows: xterm.rows
});

// Connect xterm to rexec terminal
rexecTerminal.onData((data) => {
  xterm.write(data);
});

xterm.onData((data) => {
  rexecTerminal.write(data);
});

xterm.onResize(({ cols, rows }) => {
  rexecTerminal.resize(cols, rows);
});
```

## TypeScript

The SDK is written in TypeScript and includes full type definitions:

```typescript
import { 
  RexecClient, 
  Container, 
  CreateContainerRequest,
  FileInfo,
  Terminal,
  RexecError 
} from '@pipeopshq/rexec';

// All types are available
const request: CreateContainerRequest = {
  image: 'ubuntu:24.04',
  name: 'typed-container'
};
```

## Error Handling

```typescript
import { RexecClient, RexecError } from '@pipeopshq/rexec';

try {
  await client.containers.get('non-existent-id');
} catch (error) {
  if (error instanceof RexecError) {
    console.error(`API Error ${error.statusCode}: ${error.message}`);
  } else {
    console.error('Network error:', error);
  }
}
```

## License

MIT License - see [LICENSE](../../LICENSE) for details.
