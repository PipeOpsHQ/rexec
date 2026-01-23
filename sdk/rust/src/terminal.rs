//! Terminal service for WebSocket connections to containers.

use futures_util::{SinkExt, StreamExt};
use std::sync::Arc;
use tokio::sync::Mutex;
use tokio_tungstenite::{connect_async, tungstenite::Message};

use crate::client::ClientInner;
use crate::error::{Error, Result};
use crate::types::ResizeMessage;

type WsStream = tokio_tungstenite::WebSocketStream<
    tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>,
>;

/// WebSocket terminal connection to a container.
pub struct Terminal {
    ws: Arc<Mutex<WsStream>>,
    closed: bool,
}

impl Terminal {
    fn new(ws: WsStream) -> Self {
        Self {
            ws: Arc::new(Mutex::new(ws)),
            closed: false,
        }
    }

    /// Send data to the terminal.
    ///
    /// # Arguments
    ///
    /// * `data` - Data to send
    pub async fn write(&mut self, data: &[u8]) -> Result<()> {
        if self.closed {
            return Err(Error::TerminalClosed);
        }

        let mut ws = self.ws.lock().await;
        ws.send(Message::Binary(data.to_vec())).await?;
        Ok(())
    }

    /// Send a string to the terminal.
    pub async fn write_str(&mut self, data: &str) -> Result<()> {
        self.write(data.as_bytes()).await
    }

    /// Read data from the terminal.
    ///
    /// # Returns
    ///
    /// Bytes received from the terminal, or `None` if connection closed.
    pub async fn read(&mut self) -> Result<Option<Vec<u8>>> {
        if self.closed {
            return Err(Error::TerminalClosed);
        }

        let mut ws = self.ws.lock().await;
        
        match ws.next().await {
            Some(Ok(Message::Binary(data))) => Ok(Some(data)),
            Some(Ok(Message::Text(text))) => Ok(Some(text.into_bytes())),
            Some(Ok(Message::Close(_))) => {
                self.closed = true;
                Ok(None)
            }
            Some(Ok(_)) => self.read_inner(&mut ws).await,
            Some(Err(e)) => {
                self.closed = true;
                Err(e.into())
            }
            None => {
                self.closed = true;
                Ok(None)
            }
        }
    }

    async fn read_inner(&mut self, ws: &mut WsStream) -> Result<Option<Vec<u8>>> {
        // Skip ping/pong messages
        loop {
            match ws.next().await {
                Some(Ok(Message::Binary(data))) => return Ok(Some(data)),
                Some(Ok(Message::Text(text))) => return Ok(Some(text.into_bytes())),
                Some(Ok(Message::Close(_))) => {
                    self.closed = true;
                    return Ok(None);
                }
                Some(Ok(_)) => continue,
                Some(Err(e)) => {
                    self.closed = true;
                    return Err(e.into());
                }
                None => {
                    self.closed = true;
                    return Ok(None);
                }
            }
        }
    }

    /// Resize the terminal.
    ///
    /// # Arguments
    ///
    /// * `cols` - Number of columns
    /// * `rows` - Number of rows
    pub async fn resize(&mut self, cols: u16, rows: u16) -> Result<()> {
        if self.closed {
            return Err(Error::TerminalClosed);
        }

        let msg = ResizeMessage::new(cols, rows);
        let json = serde_json::to_string(&msg)?;

        let mut ws = self.ws.lock().await;
        ws.send(Message::Text(json)).await?;
        Ok(())
    }

    /// Check if the connection is closed.
    pub fn is_closed(&self) -> bool {
        self.closed
    }

    /// Close the terminal connection.
    pub async fn close(&mut self) -> Result<()> {
        if !self.closed {
            self.closed = true;
            let mut ws = self.ws.lock().await;
            ws.close(None).await?;
        }
        Ok(())
    }
}

/// Service for terminal WebSocket connections.
pub struct TerminalService {
    client: Arc<ClientInner>,
}

impl TerminalService {
    pub(crate) fn new(client: Arc<ClientInner>) -> Self {
        Self { client }
    }

    /// Connect to a container's terminal.
    ///
    /// # Arguments
    ///
    /// * `container_id` - Container ID
    ///
    /// # Example
    ///
    /// ```rust,no_run
    /// # use rexec::RexecClient;
    /// # async fn example() -> Result<(), rexec::Error> {
    /// let client = RexecClient::new("https://example.com", "token");
    /// let mut term = client.terminal().connect("container-id").await?;
    ///
    /// term.write(b"ls -la\n").await?;
    ///
    /// while let Some(data) = term.read().await? {
    ///     print!("{}", String::from_utf8_lossy(&data));
    /// }
    /// # Ok(())
    /// # }
    /// ```
    pub async fn connect(&self, container_id: &str) -> Result<Terminal> {
        self.connect_with_size(container_id, 80, 24).await
    }

    /// Connect to a container's terminal with custom size.
    ///
    /// # Arguments
    ///
    /// * `container_id` - Container ID
    /// * `cols` - Terminal width in columns
    /// * `rows` - Terminal height in rows
    pub async fn connect_with_size(
        &self,
        container_id: &str,
        cols: u16,
        rows: u16,
    ) -> Result<Terminal> {
        let ws_url = self.client.ws_url(&format!("/ws/terminal/{}", container_id))?;

        let request = http::Request::builder()
            .uri(&ws_url)
            .header("Authorization", format!("Bearer {}", self.client.token))
            .header("Host", url::Url::parse(&self.client.base_url)?.host_str().unwrap_or(""))
            .header("Connection", "Upgrade")
            .header("Upgrade", "websocket")
            .header("Sec-WebSocket-Version", "13")
            .header("Sec-WebSocket-Key", tokio_tungstenite::tungstenite::handshake::client::generate_key())
            .body(())
            .map_err(|e| Error::connection(e.to_string()))?;

        let (ws, _) = connect_async(request)
            .await
            .map_err(|e| Error::connection(e.to_string()))?;

        let mut terminal = Terminal::new(ws);

        // Set initial size
        terminal.resize(cols, rows).await?;

        Ok(terminal)
    }
}
