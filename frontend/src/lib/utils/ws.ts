const WS_PROTOCOL_VERSION = "rexec.v1";
const WS_TOKEN_PROTOCOL_PREFIX = "rexec.token.";

export function getRexecWebSocketProtocols(token: string | null): string[] | undefined {
  if (!token) return undefined;
  return [WS_PROTOCOL_VERSION, `${WS_TOKEN_PROTOCOL_PREFIX}${token}`];
}

export function createRexecWebSocket(wsUrl: string, token: string | null): WebSocket {
  const protocols = getRexecWebSocketProtocols(token);
  return protocols ? new WebSocket(wsUrl, protocols) : new WebSocket(wsUrl);
}

