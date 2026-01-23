//! Type definitions for Rexec SDK.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Container status.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum ContainerStatus {
    Running,
    Stopped,
    Creating,
    Error,
    #[serde(other)]
    Unknown,
}

/// Represents a Rexec container/sandbox.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Container {
    /// Container ID.
    pub id: String,
    /// Container name.
    pub name: String,
    /// Docker image.
    pub image: String,
    /// Current status.
    pub status: String,
    /// Creation timestamp.
    pub created_at: String,
    /// Start timestamp (if running).
    pub started_at: Option<String>,
    /// Container labels.
    #[serde(default)]
    pub labels: HashMap<String, String>,
    /// Environment variables.
    #[serde(default)]
    pub environment: HashMap<String, String>,
}

/// Request to create a new container.
#[derive(Debug, Clone, Serialize, Default)]
pub struct CreateContainerRequest {
    /// Docker image to use.
    pub image: String,
    /// Optional container name.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    /// Environment variables.
    #[serde(skip_serializing_if = "HashMap::is_empty", default)]
    pub environment: HashMap<String, String>,
    /// Container labels.
    #[serde(skip_serializing_if = "HashMap::is_empty", default)]
    pub labels: HashMap<String, String>,
}

impl CreateContainerRequest {
    /// Create a new request with the given image.
    pub fn new(image: impl Into<String>) -> Self {
        Self {
            image: image.into(),
            ..Default::default()
        }
    }

    /// Set the container name.
    pub fn name(mut self, name: impl Into<String>) -> Self {
        self.name = Some(name.into());
        self
    }

    /// Add an environment variable.
    pub fn env(mut self, key: impl Into<String>, value: impl Into<String>) -> Self {
        self.environment.insert(key.into(), value.into());
        self
    }

    /// Add a label.
    pub fn label(mut self, key: impl Into<String>, value: impl Into<String>) -> Self {
        self.labels.insert(key.into(), value.into());
        self
    }
}

/// File or directory metadata.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileInfo {
    /// File name.
    pub name: String,
    /// Full path.
    pub path: String,
    /// File size in bytes.
    pub size: u64,
    /// File mode (permissions).
    pub mode: String,
    /// Modification time.
    pub mod_time: String,
    /// Whether this is a directory.
    pub is_dir: bool,
}

/// Terminal resize message.
#[derive(Debug, Clone, Serialize)]
pub struct ResizeMessage {
    #[serde(rename = "type")]
    pub msg_type: String,
    pub cols: u16,
    pub rows: u16,
}

impl ResizeMessage {
    pub fn new(cols: u16, rows: u16) -> Self {
        Self {
            msg_type: "resize".into(),
            cols,
            rows,
        }
    }
}
