//! # Rexec Rust SDK
//!
//! Official Rust SDK for [Rexec](https://github.com/PipeOpsHQ/rexec) - Terminal as a Service.
//!
//! ## Quick Start
//!
//! ```rust,no_run
//! use rexec::{RexecClient, CreateContainerRequest};
//!
//! #[tokio::main]
//! async fn main() -> Result<(), rexec::Error> {
//!     let client = RexecClient::new(
//!         "https://your-instance.com",
//!         "your-api-token"
//!     );
//!
//!     // Create a container
//!     let container = client.containers()
//!         .create(CreateContainerRequest {
//!             image: "ubuntu:24.04".into(),
//!             name: Some("my-sandbox".into()),
//!             ..Default::default()
//!         })
//!         .await?;
//!
//!     println!("Created container: {}", container.id);
//!
//!     // Connect to terminal
//!     let mut term = client.terminal().connect(&container.id).await?;
//!     term.write(b"echo hello\n").await?;
//!
//!     // Clean up
//!     client.containers().delete(&container.id).await?;
//!
//!     Ok(())
//! }
//! ```

mod client;
mod containers;
mod error;
mod files;
mod terminal;
mod types;

pub use client::RexecClient;
pub use containers::ContainerService;
pub use error::Error;
pub use files::FileService;
pub use terminal::{Terminal, TerminalService};
pub use types::*;
