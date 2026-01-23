//! Container service for managing sandboxed environments.

use reqwest::Method;
use std::sync::Arc;

use crate::client::ClientInner;
use crate::error::Result;
use crate::types::{Container, CreateContainerRequest};

/// Service for managing containers.
pub struct ContainerService {
    client: Arc<ClientInner>,
}

impl ContainerService {
    pub(crate) fn new(client: Arc<ClientInner>) -> Self {
        Self { client }
    }

    /// List all containers for the authenticated user.
    ///
    /// # Example
    ///
    /// ```rust,no_run
    /// # use rexec::RexecClient;
    /// # async fn example() -> Result<(), rexec::Error> {
    /// let client = RexecClient::new("https://example.com", "token");
    /// let containers = client.containers().list().await?;
    /// for c in containers {
    ///     println!("{}: {}", c.name, c.status);
    /// }
    /// # Ok(())
    /// # }
    /// ```
    pub async fn list(&self) -> Result<Vec<Container>> {
        self.client.request(Method::GET, "/api/containers").await
    }

    /// Get a container by ID.
    ///
    /// # Arguments
    ///
    /// * `id` - Container ID
    pub async fn get(&self, id: &str) -> Result<Container> {
        self.client
            .request(Method::GET, &format!("/api/containers/{}", id))
            .await
    }

    /// Create a new container.
    ///
    /// # Example
    ///
    /// ```rust,no_run
    /// # use rexec::{RexecClient, CreateContainerRequest};
    /// # async fn example() -> Result<(), rexec::Error> {
    /// let client = RexecClient::new("https://example.com", "token");
    /// let container = client.containers()
    ///     .create(CreateContainerRequest::new("ubuntu:24.04")
    ///         .name("my-sandbox")
    ///         .env("MY_VAR", "value"))
    ///     .await?;
    /// # Ok(())
    /// # }
    /// ```
    pub async fn create(&self, request: CreateContainerRequest) -> Result<Container> {
        self.client
            .request_with_body(Method::POST, "/api/containers", &request)
            .await
    }

    /// Delete a container.
    ///
    /// # Arguments
    ///
    /// * `id` - Container ID to delete
    pub async fn delete(&self, id: &str) -> Result<()> {
        self.client
            .request_empty(Method::DELETE, &format!("/api/containers/{}", id))
            .await
    }

    /// Start a stopped container.
    ///
    /// # Arguments
    ///
    /// * `id` - Container ID to start
    pub async fn start(&self, id: &str) -> Result<()> {
        self.client
            .request_empty(Method::POST, &format!("/api/containers/{}/start", id))
            .await
    }

    /// Stop a running container.
    ///
    /// # Arguments
    ///
    /// * `id` - Container ID to stop
    pub async fn stop(&self, id: &str) -> Result<()> {
        self.client
            .request_empty(Method::POST, &format!("/api/containers/{}/stop", id))
            .await
    }
}
