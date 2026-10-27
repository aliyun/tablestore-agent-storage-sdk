//! OSS upload extension for [`crate::AgentStorageClient`].
//!
//! Available when the `oss-upload` feature is enabled (on by default). Adds:
//!
//! - [`AliyunOssUploader`] — minimal OSS V1 client for `PutObject` /
//!   `DeleteMultipleObjects`.
//! - [`OssUploader`] — trait so users can plug in any uploader (e.g. for
//!   tests, alternate clouds, or a richer SDK).
//! - [`AgentStorageClientExt`] — extension trait that adds
//!   `upload_documents` / `delete_documents_with_oss` methods on top of
//!   [`crate::AgentStorageClient`].

mod extension;
mod path;
mod uploader;

pub use extension::{AgentStorageClientExt, OssUploadConfig};
pub use path::{
    build_oss_full_path, build_oss_object_key, parse_oss_path, OssLocation, DEFAULT_SUBSPACE,
};
pub use uploader::{AliyunOssUploader, OssUploader};
