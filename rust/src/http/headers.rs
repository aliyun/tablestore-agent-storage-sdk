use std::collections::BTreeMap;

use chrono::{DateTime, Utc};

use crate::auth::{Credentials, RequestContext, Signer};
use crate::error::ClientError;

use super::md5::md5_base64;

/// OTS protocol version sent in `x-ots-apiversion`.
pub const API_VERSION: &str = "2015-12-31";

/// Format the `x-ots-date` value the same way as the other SDKs:
/// `YYYY-MM-DDTHH:MM:SS.000Z`. Millisecond field is fixed to `.000` to match
/// the Python / TypeScript implementations byte-for-byte.
pub fn build_date_header(date: DateTime<Utc>) -> String {
    date.format("%Y-%m-%dT%H:%M:%S.000Z").to_string()
}

/// Inputs for [`build_request_headers`].
pub struct BuildHeadersInput<'a> {
    /// Raw request body — used only for the MD5 hash.
    pub body: &'a [u8],
    /// Request path including action, e.g. `/CreateKnowledgeBase`.
    pub query: &'a str,
    /// Tablestore instance name; sent as `x-ots-instancename`.
    pub instance_name: &'a str,
    /// Signing credentials.
    pub credentials: &'a Credentials,
    /// Active signer implementation.
    pub signer: &'a dyn Signer,
    /// Mutable request context (V4 stores its sign date here).
    pub request_context: &'a RequestContext,
    /// `User-Agent` to emit.
    pub user_agent: &'a str,
    /// Pinning the timestamp is handy for fixtures; otherwise `Utc::now()`.
    pub date: Option<DateTime<Utc>>,
    /// Caller-supplied headers injected before signing. `x-ots-*` entries take
    /// part in the signature; everything else simply rides along.
    pub extra_headers: Option<&'a BTreeMap<String, String>>,
}

/// Build the full set of headers for an OTS JSON request, including the
/// signature headers produced by `signer`.
pub fn build_request_headers(
    input: BuildHeadersInput<'_>,
) -> Result<BTreeMap<String, String>, ClientError> {
    let date = input.date.unwrap_or_else(Utc::now);

    let mut headers = BTreeMap::new();
    headers.insert("x-ots-date".to_string(), build_date_header(date));
    headers.insert("x-ots-apiversion".to_string(), API_VERSION.to_string());
    headers.insert(
        "x-ots-accesskeyid".to_string(),
        input.credentials.access_key_id.clone(),
    );
    headers.insert(
        "x-ots-instancename".to_string(),
        input.instance_name.to_string(),
    );
    headers.insert("x-ots-contentmd5".to_string(), md5_base64(input.body));

    if let Some(token) = &input.credentials.security_token {
        headers.insert("x-ots-ststoken".to_string(), token.clone());
    }

    if let Some(extras) = input.extra_headers {
        for (key, value) in extras {
            headers.insert(key.clone(), value.clone());
        }
    }

    input
        .signer
        .sign_request(input.query, &mut headers, input.request_context)?;

    headers.insert("User-Agent".to_string(), input.user_agent.to_string());
    Ok(headers)
}
