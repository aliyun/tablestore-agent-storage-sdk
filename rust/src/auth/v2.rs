use std::collections::BTreeMap;

use crate::error::ClientError;

use super::context::RequestContext;
use super::signer::{canonical_request_string, hmac_sha1_base64, Signer, SigningKey};

/// HMAC-SHA1 signer matching the historical OTS v2 protocol.
#[derive(Debug, Default, Clone, Copy)]
pub struct SignV2;

impl SignV2 {
    /// Construct a new V2 signer. Stateless; reuse the same instance.
    pub fn new() -> Self {
        Self
    }
}

impl Signer for SignV2 {
    fn signing_key(&self, context: &RequestContext) -> Result<SigningKey, ClientError> {
        Ok(SigningKey::from(
            context.credentials().access_key_secret.as_str(),
        ))
    }

    fn sign_request(
        &self,
        query: &str,
        headers: &mut BTreeMap<String, String>,
        context: &RequestContext,
    ) -> Result<(), ClientError> {
        let signing_key = self.signing_key(context)?;
        let canonical = canonical_request_string(query, headers);
        let signature = hmac_sha1_base64(signing_key.as_bytes(), canonical.as_bytes());
        headers.insert("x-ots-signature".to_string(), signature);
        Ok(())
    }
}
