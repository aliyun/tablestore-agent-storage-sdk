//! HTTP layer: request header construction and the JSON over POST client.

mod client;
mod headers;
mod md5;

pub use client::OtsJsonClient;
pub use headers::{build_date_header, build_request_headers, BuildHeadersInput, API_VERSION};
pub use md5::md5_base64;

/// Trim every trailing `/` from `value`. Both the OTS endpoint and the OSS
/// endpoint are joined with `/{path}`, so a trailing slash on the configured
/// URL would produce a `//` and a 404.
pub(crate) fn strip_trailing_slashes(mut value: String) -> String {
    while value.ends_with('/') {
        value.pop();
    }
    value
}
