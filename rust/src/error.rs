//! Error types returned by the SDK.
//!
//! The split mirrors the TypeScript / Python SDKs:
//!
//! - [`ClientError`] covers everything that fails before the server has a
//!   chance to respond (bad config, network failure, response tampering).
//! - [`ServiceError`] is what the server itself reports via a non-2xx HTTP
//!   status. Its fields are stable enough for an agent to branch on.
//!
//! Both bubble up through the unified [`Error`].

use thiserror::Error;

/// Error returned by client construction and request execution.
#[derive(Debug, Error)]
pub enum Error {
    /// Failure that originated on the client side (config, transport, parsing).
    #[error(transparent)]
    Client(#[from] ClientError),
    /// Failure reported by the server with a non-2xx HTTP response.
    #[error(transparent)]
    Service(#[from] ServiceError),
}

/// Errors raised before a response is delivered or while validating one.
#[derive(Debug, Error)]
pub enum ClientError {
    /// A required builder or request field was not supplied.
    #[error("missing required field: {0}")]
    MissingField(&'static str),

    /// A field had an invalid value (e.g. V4 selected without `region`).
    #[error("invalid argument: {0}")]
    InvalidArgument(String),

    /// `fetch`-equivalent failure: connection refused, DNS, timeout, etc.
    #[error("http transport error: {0}")]
    Http(#[from] reqwest::Error),

    /// Response body could not be parsed as JSON.
    #[error("failed to parse JSON response: {0}")]
    InvalidResponseBody(String),

    /// A header that must be present on 2xx responses was missing.
    #[error("missing response header: {0}")]
    MissingResponseHeader(&'static str),

    /// The server response signature did not validate.
    #[error("invalid signature in response")]
    InvalidResponseSignature,

    /// `x-ots-contentmd5` did not match the body we received.
    #[error("MD5 mismatch in response")]
    Md5Mismatch,

    /// `Authorization` header on a 2xx response was malformed.
    #[error("invalid authorization in response: {0}")]
    InvalidResponseAuthorization(&'static str),

    /// File or filesystem error encountered while preparing an upload.
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),

    /// The referenced local file does not exist.
    #[error("file not found: {0}")]
    FileNotFound(String),

    /// An OSS upload / delete call failed.
    #[error("oss error: {0}")]
    Oss(String),
}

/// Error reported by the Tablestore service with a non-2xx HTTP status.
#[derive(Debug, Error, Clone)]
#[error("service error: status={status_code} code={code:?} message={message}")]
pub struct ServiceError {
    /// HTTP status code returned by the server.
    pub status_code: u16,
    /// `x-ots-requestid` header, useful for support tickets.
    pub request_id: Option<String>,
    /// Service-defined error code (`Forbidden`, `OTSObjectNotExist`, ...).
    pub code: Option<String>,
    /// Human-readable error message.
    pub message: String,
}
