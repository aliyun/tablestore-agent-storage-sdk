//! OTS request signing.
//!
//! Two algorithms are supported: V2 (HMAC-SHA1, default) and V4 (HMAC-SHA256
//! with a multi-step signing-key derivation). Both share the canonical-string
//! computation in [`signer::Signer`].

mod context;
mod signer;
mod v2;
mod v4;

pub use context::{Credentials, RequestContext};
pub use signer::{Signer, SigningKey};
pub use v2::SignV2;
pub use v4::SignV4;
