use std::collections::BTreeMap;

use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use hmac::{Hmac, Mac};
use sha1::Sha1;

use crate::error::ClientError;

use super::context::RequestContext;

const OTS_HEADER_PREFIX: &str = "x-ots-";

/// Signing key produced by [`Signer::signing_key`].
///
/// V2 uses a UTF-8 secret directly; V4 uses an HMAC-derived byte string. The
/// enum hides that detail from the caller.
#[derive(Debug, Clone)]
pub enum SigningKey {
    /// Raw bytes used as HMAC key material.
    Bytes(Vec<u8>),
}

impl SigningKey {
    /// Borrow the key material for HMAC initialisation.
    pub fn as_bytes(&self) -> &[u8] {
        match self {
            SigningKey::Bytes(bytes) => bytes,
        }
    }
}

impl From<&str> for SigningKey {
    fn from(value: &str) -> Self {
        SigningKey::Bytes(value.as_bytes().to_vec())
    }
}

impl From<Vec<u8>> for SigningKey {
    fn from(value: Vec<u8>) -> Self {
        SigningKey::Bytes(value)
    }
}

/// Trait implemented by every supported signing algorithm.
///
/// Implementations are stateless apart from any configuration (region, sign
/// date) supplied at construction. Mutable per-request state lives on
/// [`RequestContext`].
pub trait Signer: Send + Sync {
    /// Derive the signing key for the current request.
    fn signing_key(&self, context: &RequestContext) -> Result<SigningKey, ClientError>;

    /// Compute and inject the algorithm-specific signature headers into
    /// `headers` in place.
    fn sign_request(
        &self,
        query: &str,
        headers: &mut BTreeMap<String, String>,
        context: &RequestContext,
    ) -> Result<(), ClientError>;

    /// Recompute the expected response signature so the caller can compare it
    /// with the `Authorization` header returned by the server.
    fn response_signature(
        &self,
        query: &str,
        headers: &BTreeMap<String, String>,
        key: &SigningKey,
    ) -> String {
        let (uri, _) = split_query(query);
        let headers_string = make_headers_string(headers);
        let payload = format!("{headers_string}\n{uri}");
        hmac_sha1_base64(key.as_bytes(), payload.as_bytes())
    }
}

/// Canonical request string used by both V2 and V4.
///
/// Format: `{uri}\nPOST\n{sortedQuery}\n{sortedXOtsHeaders}\n`.
pub(super) fn canonical_request_string(query: &str, headers: &BTreeMap<String, String>) -> String {
    let (uri, query_string) = split_query(query);
    let sorted_query = sort_and_encode_query(query_string);
    let headers_string = make_headers_string(headers);
    format!("{uri}\nPOST\n{sorted_query}\n{headers_string}\n")
}

/// Join all `x-ots-*` headers, lower-cased, sorted by key.
fn make_headers_string(headers: &BTreeMap<String, String>) -> String {
    let mut parts: Vec<String> = headers
        .iter()
        .filter(|(key, _)| key.to_ascii_lowercase().starts_with(OTS_HEADER_PREFIX))
        .map(|(key, value)| format!("{}:{}", key.to_ascii_lowercase(), value.trim()))
        .collect();
    parts.sort();
    parts.join("\n")
}

/// Split a path/query string, dropping any fragment.
fn split_query(query: &str) -> (&str, &str) {
    let no_fragment = match query.find('#') {
        Some(idx) => &query[..idx],
        None => query,
    };
    match no_fragment.find('?') {
        Some(idx) => (&no_fragment[..idx], &no_fragment[idx + 1..]),
        None => (no_fragment, ""),
    }
}

/// Sort `key=value` pairs by (key, value) and re-encode using URL form.
fn sort_and_encode_query(query: &str) -> String {
    if query.is_empty() {
        return String::new();
    }
    let mut pairs: Vec<(String, String)> = url::form_urlencoded::parse(query.as_bytes())
        .map(|(k, v)| (k.into_owned(), v.into_owned()))
        .collect();
    pairs.sort();
    let mut serializer = url::form_urlencoded::Serializer::new(String::new());
    for (k, v) in &pairs {
        serializer.append_pair(k, v);
    }
    serializer.finish()
}

pub(super) fn hmac_sha1_base64(key: &[u8], content: &[u8]) -> String {
    let mut mac = Hmac::<Sha1>::new_from_slice(key).expect("HMAC accepts any key length");
    mac.update(content);
    BASE64.encode(mac.finalize().into_bytes())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn splits_query_drops_fragment() {
        assert_eq!(split_query("/Foo?a=1#frag"), ("/Foo", "a=1"));
        assert_eq!(split_query("/Foo"), ("/Foo", ""));
    }

    #[test]
    fn sorts_query_pairs() {
        assert_eq!(sort_and_encode_query("b=2&a=1"), "a=1&b=2");
        assert_eq!(sort_and_encode_query(""), "");
    }
}
