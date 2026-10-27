use std::collections::BTreeMap;

use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use chrono::Utc;
use hmac::{Hmac, Mac};
use sha2::Sha256;

use crate::error::ClientError;

use super::context::RequestContext;
use super::signer::{canonical_request_string, Signer, SigningKey};

const V4_SIGNATURE_PRODUCT: &str = "ots";
const V4_SIGNATURE_SEED: &str = "aliyun";
const V4_SIGNATURE_CONSTANT: &str = "aliyun_v4_request";
const V4_SIGNATURE_SALT: &str = "ots";

/// HMAC-SHA256 signer matching the OTS v4 protocol.
///
/// V4 requires a region and a sign date. If `sign_date` is omitted the signer
/// stamps each request with the current UTC date (`YYYYMMDD`), matching the
/// behaviour of the other SDKs.
#[derive(Debug, Clone)]
pub struct SignV4 {
    region: String,
    fixed_sign_date: Option<String>,
    auto_update: bool,
}

impl SignV4 {
    /// Build a V4 signer that auto-stamps the UTC date on every request.
    pub fn new(region: impl Into<String>) -> Result<Self, ClientError> {
        let region = region.into();
        if region.is_empty() {
            return Err(ClientError::InvalidArgument(
                "region is required for V4 signing".to_string(),
            ));
        }
        Ok(Self {
            region,
            fixed_sign_date: None,
            auto_update: true,
        })
    }

    /// Build a V4 signer pinned to a specific sign date. Mainly for fixtures.
    pub fn with_sign_date(
        region: impl Into<String>,
        sign_date: impl Into<String>,
    ) -> Result<Self, ClientError> {
        let mut signer = Self::new(region)?;
        signer.fixed_sign_date = Some(sign_date.into());
        signer.auto_update = false;
        Ok(signer)
    }

    fn resolve_sign_date(&self, context: &RequestContext) -> Result<String, ClientError> {
        let date = if self.auto_update {
            Utc::now().format("%Y%m%d").to_string()
        } else {
            self.fixed_sign_date
                .clone()
                .ok_or(ClientError::InvalidArgument(
                    "v4 sign_date is None".to_string(),
                ))?
        };
        context.set_sign_date(&date);
        Ok(date)
    }
}

impl Signer for SignV4 {
    fn signing_key(&self, context: &RequestContext) -> Result<SigningKey, ClientError> {
        let access_key_secret = &context.credentials().access_key_secret;
        let sign_date = context.sign_date().ok_or(ClientError::InvalidArgument(
            "v4 sign_date is None".to_string(),
        ))?;

        let seeded_key = format!("{V4_SIGNATURE_SEED}_v4{access_key_secret}");
        let first = hmac_sha256(seeded_key.as_bytes(), sign_date.as_bytes());
        let second = hmac_sha256(&first, self.region.as_bytes());
        let third = hmac_sha256(&second, V4_SIGNATURE_PRODUCT.as_bytes());
        let fourth = hmac_sha256(&third, V4_SIGNATURE_CONSTANT.as_bytes());
        // Match the other SDKs: base64-encode the derived bytes, then use the
        // ASCII string as the signing key for the final request signature.
        let base64_key = BASE64.encode(fourth).into_bytes();
        Ok(SigningKey::from(base64_key))
    }

    fn sign_request(
        &self,
        query: &str,
        headers: &mut BTreeMap<String, String>,
        context: &RequestContext,
    ) -> Result<(), ClientError> {
        let sign_date = self.resolve_sign_date(context)?;
        headers.insert("x-ots-signdate".to_string(), sign_date);
        headers.insert("x-ots-signregion".to_string(), self.region.clone());

        let signing_key = self.signing_key(context)?;
        let canonical = canonical_request_string(query, headers);
        let payload = format!("{canonical}{V4_SIGNATURE_SALT}");
        let signature = hmac_sha256_base64(signing_key.as_bytes(), payload.as_bytes());
        headers.insert("x-ots-signaturev4".to_string(), signature);
        Ok(())
    }
}

fn hmac_sha256(key: &[u8], content: &[u8]) -> Vec<u8> {
    let mut mac = Hmac::<Sha256>::new_from_slice(key).expect("HMAC accepts any key length");
    mac.update(content);
    mac.finalize().into_bytes().to_vec()
}

fn hmac_sha256_base64(key: &[u8], content: &[u8]) -> String {
    BASE64.encode(hmac_sha256(key, content))
}
