//! Main Rexec client.

use reqwest::{Client, Response};
use std::sync::Arc;
use url::Url;

use crate::containers::ContainerService;
use crate::error::{Error, Result};
use crate::files::FileService;
use crate::terminal::TerminalService;

/// Configuration for the Rexec client.
#[derive(Debug, Clone)]
pub struct ClientConfig {
    /// Base URL of the Rexec instance.
    pub base_url: String,
    /// API token for authentication.
    pub token: String,
    /// Request timeout in seconds.
    pub timeout: u64,
}

impl ClientConfig {
    /// Create a new client configuration.
    pub fn new(base_url: impl Into<String>, token: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into(),
            token: token.into(),
            timeout: 30,
        }
    }

    /// Set the request timeout.
    pub fn timeout(mut self, seconds: u64) -> Self {
        self.timeout = seconds;
        self
    }
}

/// Internal client state shared between services.
pub(crate) struct ClientInner {
    pub(crate) http: Client,
    pub(crate) base_url: String,
    pub(crate) token: String,
}

impl ClientInner {
    /// Make an API request and parse JSON response.
    pub(crate) async fn request<T: serde::de::DeserializeOwned>(
        &self,
        method: reqwest::Method,
        path: &str,
    ) -> Result<T> {
        let response = self.raw_request(method, path, None::<()>).await?;
        self.handle_response(response).await
    }

    /// Make an API request with JSON body.
    pub(crate) async fn request_with_body<T, B>(
        &self,
        method: reqwest::Method,
        path: &str,
        body: &B,
    ) -> Result<T>
    where
        T: serde::de::DeserializeOwned,
        B: serde::Serialize,
    {
        let response = self.raw_request(method, path, Some(body)).await?;
        self.handle_response(response).await
    }

    /// Make an API request without expecting a response body.
    pub(crate) async fn request_empty(
        &self,
        method: reqwest::Method,
        path: &str,
    ) -> Result<()> {
        let response = self.raw_request(method, path, None::<()>).await?;
        if response.status().is_success() {
            Ok(())
        } else {
            self.handle_error(response).await
        }
    }

    /// Make a raw API request.
    pub(crate) async fn raw_request<B: serde::Serialize>(
        &self,
        method: reqwest::Method,
        path: &str,
        body: Option<&B>,
    ) -> Result<Response> {
        let url = format!("{}{}", self.base_url, path);
        
        let mut builder = self.http
            .request(method, &url)
            .header("Authorization", format!("Bearer {}", self.token))
            .header("Accept", "application/json");

        if let Some(b) = body {
            builder = builder.json(b);
        }

        Ok(builder.send().await?)
    }

    /// Handle a successful response.
    async fn handle_response<T: serde::de::DeserializeOwned>(
        &self,
        response: Response,
    ) -> Result<T> {
        if response.status().is_success() {
            Ok(response.json().await?)
        } else {
            self.handle_error(response).await
        }
    }

    /// Handle an error response.
    async fn handle_error<T>(&self, response: Response) -> Result<T> {
        let status = response.status().as_u16();
        let message = response
            .json::<serde_json::Value>()
            .await
            .ok()
            .and_then(|v| v.get("error").and_then(|e| e.as_str()).map(String::from))
            .unwrap_or_else(|| "Unknown error".into());

        Err(Error::api(status, message))
    }

    /// Get WebSocket URL for a path.
    pub(crate) fn ws_url(&self, path: &str) -> Result<String> {
        let url = Url::parse(&self.base_url)?;
        let ws_scheme = if url.scheme() == "https" { "wss" } else { "ws" };
        let host = url.host_str().ok_or_else(|| Error::connection("Invalid host"))?;
        let port = url.port().map(|p| format!(":{}", p)).unwrap_or_default();
        Ok(format!("{}://{}{}{}", ws_scheme, host, port, path))
    }
}

/// Main client for interacting with Rexec API.
///
/// # Example
///
/// ```rust,no_run
/// use rexec::RexecClient;
///
/// #[tokio::main]
/// async fn main() -> Result<(), rexec::Error> {
///     let client = RexecClient::new("https://your-instance.com", "your-token");
///     
///     let containers = client.containers().list().await?;
///     for c in containers {
///         println!("{}: {}", c.name, c.status);
///     }
///     
///     Ok(())
/// }
/// ```
#[derive(Clone)]
pub struct RexecClient {
    inner: Arc<ClientInner>,
}

impl RexecClient {
    /// Create a new Rexec client.
    ///
    /// # Arguments
    ///
    /// * `base_url` - Base URL of your Rexec instance
    /// * `token` - API token for authentication
    pub fn new(base_url: impl Into<String>, token: impl Into<String>) -> Self {
        Self::with_config(ClientConfig::new(base_url, token))
    }

    /// Create a new Rexec client with custom configuration.
    pub fn with_config(config: ClientConfig) -> Self {
        let http = Client::builder()
            .timeout(std::time::Duration::from_secs(config.timeout))
            .build()
            .expect("Failed to create HTTP client");

        let inner = Arc::new(ClientInner {
            http,
            base_url: config.base_url.trim_end_matches('/').to_string(),
            token: config.token,
        });

        Self { inner }
    }

    /// Get the container service.
    pub fn containers(&self) -> ContainerService {
        ContainerService::new(self.inner.clone())
    }

    /// Get the file service.
    pub fn files(&self) -> FileService {
        FileService::new(self.inner.clone())
    }

    /// Get the terminal service.
    pub fn terminal(&self) -> TerminalService {
        TerminalService::new(self.inner.clone())
    }
}
