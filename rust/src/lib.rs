//! Tablestore Agent Storage SDK.
//!
//! Async client for the Aliyun Tablestore agent storage service (knowledge-base
//! and memory APIs). Built on `reqwest` + `tokio` with `rustls` by default so
//! the same binary runs on Linux, macOS and Windows without a system OpenSSL.
//!
//! # Quick start
//!
//! ```no_run
//! use tablestore_agent_storage::AgentStorageClient;
//!
//! # async fn run() -> Result<(), tablestore_agent_storage::Error> {
//! let client = AgentStorageClient::builder()
//!     .endpoint("https://your-instance.cn-beijing.ots.aliyuncs.com")
//!     .instance_name("your-instance")
//!     .access_key_id(std::env::var("OTS_ACCESS_KEY_ID").unwrap())
//!     .access_key_secret(std::env::var("OTS_ACCESS_KEY_SECRET").unwrap())
//!     .build()?;
//!
//! let response = client
//!     .create_knowledge_base(serde_json::json!({
//!         "knowledgeBaseName": "kb-demo",
//!     }))
//!     .await?;
//! # let _ = response;
//! # Ok(())
//! # }
//! ```
//!
//! # Input modes
//!
//! Every request method accepts both a typed request model (e.g.
//! [`models::CreateKnowledgeBaseRequest`]) and a free-form
//! [`serde_json::Value`]. The JSON form mirrors the on-wire schema and is the
//! simplest path when stitching the SDK into an agent / tool-call workflow.

#![cfg_attr(docsrs, feature(doc_cfg))]
#![warn(missing_docs)]

pub mod auth;
pub mod error;
pub mod http;
pub mod models;
mod serialization;

mod client;

#[cfg(feature = "oss-upload")]
#[cfg_attr(docsrs, doc(cfg(feature = "oss-upload")))]
pub mod oss;

pub use client::{
    AgentStorageClient, AgentStorageClientBuilder, DeleteOptions, RequestBody, SignVersion,
};
pub use error::{ClientError, Error, ServiceError};
pub use serialization::to_json_request;

/// SDK version reported in the default `User-Agent` header.
pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Default `User-Agent` reported on every request unless overridden.
pub const DEFAULT_USER_AGENT: &str =
    concat!("tablestore-agent-storage-rust/", env!("CARGO_PKG_VERSION"));
