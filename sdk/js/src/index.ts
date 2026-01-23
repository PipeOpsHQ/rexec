/**
 * Rexec JavaScript/TypeScript SDK
 * Official SDK for Rexec - Terminal as a Service
 * 
 * @packageDocumentation
 */

export interface RexecConfig {
  /** Base URL of your Rexec instance */
  baseURL: string;
  /** API token for authentication */
  token: string;
  /** Custom fetch implementation (optional) */
  fetch?: typeof fetch;
}

export interface Container {
  id: string;
  name: string;
  image: string;
  status: 'running' | 'stopped' | 'creating' | 'error';
  created_at: string;
  started_at?: string;
  labels?: Record<string, string>;
  environment?: Record<string, string>;
}

export interface CreateContainerRequest {
  /** Container name (optional) */
  name?: string;
  /** Docker image to use */
  image: string;
  /** Environment variables */
  environment?: Record<string, string>;
  /** Labels */
  labels?: Record<string, string>;
}

export interface FileInfo {
  name: string;
  path: string;
  size: number;
  mode: string;
  mod_time: string;
  is_dir: boolean;
}

export interface TerminalOptions {
  /** Terminal columns */
  cols?: number;
  /** Terminal rows */
  rows?: number;
}

export class RexecError extends Error {
  constructor(
    public statusCode: number,
    message: string
  ) {
    super(message);
    this.name = 'RexecError';
  }
}

/**
 * Terminal connection for real-time interaction
 */
export class Terminal {
  private ws: WebSocket;
  private messageHandlers: ((data: string | ArrayBuffer) => void)[] = [];
  private closeHandlers: (() => void)[] = [];
  private errorHandlers: ((error: Error) => void)[] = [];

  constructor(ws: WebSocket) {
    this.ws = ws;
    
    this.ws.onmessage = (event) => {
      this.messageHandlers.forEach(handler => handler(event.data));
    };
    
    this.ws.onclose = () => {
      this.closeHandlers.forEach(handler => handler());
    };
    
    this.ws.onerror = (event) => {
      this.errorHandlers.forEach(handler => handler(new Error('WebSocket error')));
    };
  }

  /**
   * Send data to the terminal
   */
  write(data: string | ArrayBuffer): void {
    if (this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    }
  }

  /**
   * Register a handler for incoming data
   */
  onData(handler: (data: string | ArrayBuffer) => void): void {
    this.messageHandlers.push(handler);
  }

  /**
   * Register a handler for connection close
   */
  onClose(handler: () => void): void {
    this.closeHandlers.push(handler);
  }

  /**
   * Register a handler for errors
   */
  onError(handler: (error: Error) => void): void {
    this.errorHandlers.push(handler);
  }

  /**
   * Resize the terminal
   */
  resize(cols: number, rows: number): void {
    this.write(JSON.stringify({ type: 'resize', cols, rows }));
  }

  /**
   * Close the terminal connection
   */
  close(): void {
    this.ws.close();
  }

  /**
   * Get the WebSocket ready state
   */
  get readyState(): number {
    return this.ws.readyState;
  }
}

/**
 * Container service for managing sandboxed environments
 */
export class ContainerService {
  constructor(private client: RexecClient) {}

  /**
   * List all containers
   */
  async list(): Promise<Container[]> {
    return this.client.request<Container[]>('GET', '/api/containers');
  }

  /**
   * Get a container by ID
   */
  async get(id: string): Promise<Container> {
    return this.client.request<Container>('GET', `/api/containers/${id}`);
  }

  /**
   * Create a new container
   */
  async create(options: CreateContainerRequest): Promise<Container> {
    return this.client.request<Container>('POST', '/api/containers', options);
  }

  /**
   * Delete a container
   */
  async delete(id: string): Promise<void> {
    await this.client.request('DELETE', `/api/containers/${id}`);
  }

  /**
   * Start a container
   */
  async start(id: string): Promise<void> {
    await this.client.request('POST', `/api/containers/${id}/start`);
  }

  /**
   * Stop a container
   */
  async stop(id: string): Promise<void> {
    await this.client.request('POST', `/api/containers/${id}/stop`);
  }
}

/**
 * File service for managing files in containers
 */
export class FileService {
  constructor(private client: RexecClient) {}

  /**
   * List files in a directory
   */
  async list(containerId: string, path: string = '/'): Promise<FileInfo[]> {
    const encodedPath = encodeURIComponent(path);
    return this.client.request<FileInfo[]>('GET', `/api/containers/${containerId}/files/list?path=${encodedPath}`);
  }

  /**
   * Download a file
   */
  async download(containerId: string, path: string): Promise<ArrayBuffer> {
    const encodedPath = encodeURIComponent(path);
    const response = await this.client.rawRequest('GET', `/api/containers/${containerId}/files?path=${encodedPath}`);
    return response.arrayBuffer();
  }

  /**
   * Create a directory
   */
  async mkdir(containerId: string, path: string): Promise<void> {
    await this.client.request('POST', `/api/containers/${containerId}/files/mkdir`, { path });
  }
}

/**
 * Terminal service for WebSocket connections
 */
export class TerminalService {
  constructor(private client: RexecClient) {}

  /**
   * Connect to a container's terminal
   */
  connect(containerId: string, options?: TerminalOptions): Promise<Terminal> {
    return new Promise((resolve, reject) => {
      const wsURL = this.client.getWebSocketURL(`/ws/terminal/${containerId}`);
      
      // Use native WebSocket or ws package
      const WebSocketImpl = typeof WebSocket !== 'undefined' 
        ? WebSocket 
        : require('ws');
      
      const ws = new WebSocketImpl(wsURL, {
        headers: {
          'Authorization': `Bearer ${this.client.getToken()}`
        }
      });

      ws.onopen = () => {
        const terminal = new Terminal(ws);
        if (options?.cols && options?.rows) {
          terminal.resize(options.cols, options.rows);
        }
        resolve(terminal);
      };

      ws.onerror = (error: Error) => {
        reject(new Error(`Failed to connect to terminal: ${error.message || 'WebSocket error'}`));
      };
    });
  }
}

/**
 * Main Rexec client
 * 
 * @example
 * ```typescript
 * const client = new RexecClient({
 *   baseURL: 'https://your-rexec-instance.com',
 *   token: 'your-api-token'
 * });
 * 
 * // Create a container
 * const container = await client.containers.create({
 *   image: 'ubuntu:24.04',
 *   name: 'my-sandbox'
 * });
 * 
 * // Connect to terminal
 * const terminal = await client.terminal.connect(container.id);
 * terminal.write('echo hello\n');
 * terminal.onData((data) => console.log(data));
 * ```
 */
export class RexecClient {
  private baseURL: string;
  private token: string;
  private fetchImpl: typeof fetch;

  /** Container management */
  public containers: ContainerService;
  /** File operations */
  public files: FileService;
  /** Terminal connections */
  public terminal: TerminalService;

  constructor(config: RexecConfig) {
    this.baseURL = config.baseURL.replace(/\/$/, '');
    this.token = config.token;
    this.fetchImpl = config.fetch || fetch;

    this.containers = new ContainerService(this);
    this.files = new FileService(this);
    this.terminal = new TerminalService(this);
  }

  /**
   * Make an authenticated API request
   * @internal
   */
  async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const response = await this.rawRequest(method, path, body);
    
    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new RexecError(response.status, error.error || error.message || 'Request failed');
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return response.json();
  }

  /**
   * Make a raw API request
   * @internal
   */
  async rawRequest(method: string, path: string, body?: unknown): Promise<Response> {
    const headers: Record<string, string> = {
      'Authorization': `Bearer ${this.token}`,
      'Accept': 'application/json',
    };

    if (body) {
      headers['Content-Type'] = 'application/json';
    }

    return this.fetchImpl(`${this.baseURL}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  /**
   * Get WebSocket URL
   * @internal
   */
  getWebSocketURL(path: string): string {
    const url = new URL(this.baseURL);
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    url.pathname = path;
    return url.toString();
  }

  /**
   * Get the API token
   * @internal
   */
  getToken(): string {
    return this.token;
  }
}

// Default export
export default RexecClient;
