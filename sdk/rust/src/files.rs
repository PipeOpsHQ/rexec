//! File service for managing files in containers.

use reqwest::Method;
use std::sync::Arc;

use crate::client::ClientInner;
use crate::error::Result;
use crate::types::FileInfo;

/// Service for file operations in containers.
pub struct FileService {
    client: Arc<ClientInner>,
}

impl FileService {
    pub(crate) fn new(client: Arc<ClientInner>) -> Self {
        Self { client }
    }

    /// List files in a container directory.
    ///
    /// # Arguments
    ///
    /// * `container_id` - Container ID
    /// * `path` - Directory path to list
    ///
    /// # Example
    ///
    /// ```rust,no_run
    /// # use rexec::RexecClient;
    /// # async fn example() -> Result<(), rexec::Error> {
    /// let client = RexecClient::new("https://example.com", "token");
    /// let files = client.files().list("container-id", "/home").await?;
    /// for f in files {
    ///     let icon = if f.is_dir { "📁" } else { "📄" };
    ///     println!("{} {}", icon, f.name);
    /// }
    /// # Ok(())
    /// # }
    /// ```
    pub async fn list(&self, container_id: &str, path: &str) -> Result<Vec<FileInfo>> {
        let encoded_path = urlencoding::encode(path);
        self.client
            .request(
                Method::GET,
                &format!("/api/containers/{}/files/list?path={}", container_id, encoded_path),
            )
            .await
    }

    /// Download a file from a container.
    ///
    /// # Arguments
    ///
    /// * `container_id` - Container ID
    /// * `path` - Path to the file
    ///
    /// # Returns
    ///
    /// File contents as bytes.
    pub async fn download(&self, container_id: &str, path: &str) -> Result<Vec<u8>> {
        let encoded_path = urlencoding::encode(path);
        let response = self
            .client
            .raw_request(
                Method::GET,
                &format!("/api/containers/{}/files?path={}", container_id, encoded_path),
                None::<()>,
            )
            .await?;

        Ok(response.bytes().await?.to_vec())
    }

    /// Create a directory in a container.
    ///
    /// # Arguments
    ///
    /// * `container_id` - Container ID
    /// * `path` - Directory path to create
    pub async fn mkdir(&self, container_id: &str, path: &str) -> Result<()> {
        #[derive(serde::Serialize)]
        struct MkdirRequest<'a> {
            path: &'a str,
        }

        self.client
            .request_with_body::<serde_json::Value, _>(
                Method::POST,
                &format!("/api/containers/{}/files/mkdir", container_id),
                &MkdirRequest { path },
            )
            .await?;

        Ok(())
    }

    /// Delete a file or directory from a container.
    ///
    /// # Arguments
    ///
    /// * `container_id` - Container ID
    /// * `path` - Path to delete
    pub async fn delete(&self, container_id: &str, path: &str) -> Result<()> {
        let encoded_path = urlencoding::encode(path);
        self.client
            .request_empty(
                Method::DELETE,
                &format!("/api/containers/{}/files?path={}", container_id, encoded_path),
            )
            .await
    }
}
