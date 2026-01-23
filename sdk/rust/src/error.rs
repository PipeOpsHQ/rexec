//! Error types for Rexec SDK.

use thiserror::Error;

/// Result type alias for Rexec operations.
pub type Result<T> = std::result::Result<T, Error>;

/// Error types for Rexec SDK.
#[derive(Error, Debug)]
pub enum Error {
    /// API error with status code and message.
    #[error("API error {status_code}: {message}")]
    Api {
        status_code: u16,
        message: String,
    },

    /// HTTP request error.
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),

    /// WebSocket error.
    #[error("WebSocket error: {0}")]
    WebSocket(#[from] tokio_tungstenite::tungstenite::Error),

    /// JSON serialization/deserialization error.
    #[error("JSON error: {0}")]
    Json(#[from] serde_json::Error),

    /// URL parsing error.
    #[error("URL error: {0}")]
    Url(#[from] url::ParseError),

    /// Connection error.
    #[error("Connection error: {0}")]
    Connection(String),

    /// Terminal closed.
    #[error("Terminal connection closed")]
    TerminalClosed,
}

impl Error {
    /// Create an API error.
    pub fn api(status_code: u16, message: impl Into<String>) -> Self {
        Self::Api {
            status_code,
            message: message.into(),
        }
    }

    /// Create a connection error.
    pub fn connection(message: impl Into<String>) -> Self {
        Self::Connection(message.into())
    }
}
