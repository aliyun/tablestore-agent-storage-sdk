use std::collections::BTreeMap;
use std::sync::Arc;

use reqwest::header::{HeaderMap, HeaderName, HeaderValue, CONTENT_TYPE};
use serde::de::DeserializeOwned;
use serde_json::Value;

use crate::auth::{Credentials, RequestContext, Signer};
use crate::error::{ClientError, Error, ServiceError};

use super::headers::{build_request_headers, BuildHeadersInput};
use super::md5::md5_base64;
use super::strip_trailing_slashes;

const REQUIRED_RESPONSE_HEADERS: &[&str] = &[
    "x-ots-contentmd5",
    "x-ots-requestid",
    "x-ots-date",
    "x-ots-contenttype",
];

/// Low-level POST-JSON client used by [`crate::AgentStorageClient`].
///
/// Owns the signer and the underlying `reqwest::Client`, so cloning is cheap
/// and the connection pool is shared. Most users will not construct this
/// directly — go through [`crate::AgentStorageClientBuilder`] instead.
#[derive(Clone)]
pub struct OtsJsonClient {
    inner: Arc<Inner>,
}

struct Inner {
    endpoint: String,
    instance_name: String,
    credentials: Credentials,
    signer: Box<dyn Signer>,
    http: reqwest::Client,
    user_agent: String,
    extra_headers: BTreeMap<String, String>,
}

impl OtsJsonClient {
    /// Construct a client. `endpoint` may include a trailing slash; it is
    /// stripped before use. `extra_headers` are injected into every signed
    /// request before the signature is computed.
    pub fn new(
        endpoint: impl Into<String>,
        instance_name: impl Into<String>,
        credentials: Credentials,
        signer: Box<dyn Signer>,
        http: reqwest::Client,
        user_agent: impl Into<String>,
        extra_headers: BTreeMap<String, String>,
    ) -> Self {
        let endpoint = strip_trailing_slashes(endpoint.into());
        Self {
            inner: Arc::new(Inner {
                endpoint,
                instance_name: instance_name.into(),
                credentials,
                signer,
                http,
                user_agent: user_agent.into(),
                extra_headers,
            }),
        }
    }

    /// POST `payload` as JSON to `/{action}` and deserialize the response.
    ///
    /// Errors fall into the two arms of [`Error`]: transport / validation
    /// problems land in [`Error::Client`]; server-reported non-2xx responses
    /// land in [`Error::Service`] with as much detail as the body provided.
    pub async fn request<T: DeserializeOwned>(
        &self,
        action: &str,
        payload: Value,
    ) -> Result<T, Error> {
        let body = serde_json::to_vec(&payload).map_err(|err| {
            ClientError::InvalidArgument(format!("failed to serialize request body: {err}"))
        })?;
        let query = format!("/{action}");
        let url = format!("{}/{}", self.inner.endpoint, action);

        let request_context = RequestContext::new(self.inner.credentials.clone());
        let extra_headers = if self.inner.extra_headers.is_empty() {
            None
        } else {
            Some(&self.inner.extra_headers)
        };
        let headers_map = build_request_headers(BuildHeadersInput {
            body: &body,
            query: &query,
            instance_name: &self.inner.instance_name,
            credentials: &self.inner.credentials,
            signer: self.inner.signer.as_ref(),
            request_context: &request_context,
            user_agent: &self.inner.user_agent,
            date: None,
            extra_headers,
        })?;

        let mut header_map = to_header_map(&headers_map)?;
        header_map.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));

        let response = self
            .inner
            .http
            .post(&url)
            .headers(header_map)
            .body(body)
            .send()
            .await
            .map_err(ClientError::from)?;

        let status = response.status();
        let response_headers = collect_response_headers(response.headers());
        let response_bytes = response.bytes().await.map_err(ClientError::from)?;

        self.validate_response_headers(
            &response_headers,
            &response_bytes,
            status.as_u16(),
            &request_context,
            &query,
        )?;

        if !status.is_success() {
            let payload: Value = serde_json::from_slice(&response_bytes).unwrap_or(Value::Null);
            let message = payload
                .get("message")
                .and_then(Value::as_str)
                .map(str::to_string)
                .unwrap_or_else(|| {
                    status
                        .canonical_reason()
                        .unwrap_or("request failed")
                        .to_string()
                });
            let code = payload
                .get("code")
                .and_then(Value::as_str)
                .map(str::to_string);
            return Err(Error::Service(ServiceError {
                status_code: status.as_u16(),
                request_id: response_headers.get("x-ots-requestid").cloned(),
                code,
                message,
            }));
        }

        serde_json::from_slice::<T>(&response_bytes)
            .map_err(|err| Error::Client(ClientError::InvalidResponseBody(err.to_string())))
    }

    fn validate_response_headers(
        &self,
        headers: &BTreeMap<String, String>,
        body: &[u8],
        status: u16,
        request_context: &RequestContext,
        query: &str,
    ) -> Result<(), ClientError> {
        let is_success = (200..300).contains(&status);

        if is_success {
            for required in REQUIRED_RESPONSE_HEADERS {
                if !headers.contains_key(*required) {
                    return Err(ClientError::MissingResponseHeader(required));
                }
            }
        }

        if let Some(expected_md5) = headers.get("x-ots-contentmd5") {
            if &md5_base64(body) != expected_md5 {
                return Err(ClientError::Md5Mismatch);
            }
        }

        let Some(authorization) = headers.get("authorization") else {
            if is_success {
                return Err(ClientError::MissingResponseHeader("authorization"));
            }
            return Ok(());
        };

        let Some(rest) = authorization.strip_prefix("OTS ") else {
            return Err(ClientError::InvalidResponseAuthorization(
                "missing OTS prefix",
            ));
        };

        let Some((access_key_id, signature)) = rest.split_once(':') else {
            return Err(ClientError::InvalidResponseAuthorization(
                "missing ':' separator",
            ));
        };

        if access_key_id != self.inner.credentials.access_key_id {
            return Err(ClientError::InvalidResponseAuthorization(
                "access key id mismatch",
            ));
        }

        let signing_key = self.inner.signer.signing_key(request_context)?;
        let expected = self
            .inner
            .signer
            .response_signature(query, headers, &signing_key);
        if signature != expected {
            return Err(ClientError::InvalidResponseSignature);
        }

        Ok(())
    }
}

fn to_header_map(headers: &BTreeMap<String, String>) -> Result<HeaderMap, ClientError> {
    let mut map = HeaderMap::with_capacity(headers.len() + 1);
    for (key, value) in headers {
        let name = HeaderName::from_bytes(key.as_bytes()).map_err(|err| {
            ClientError::InvalidArgument(format!("invalid header name {key}: {err}"))
        })?;
        let value = HeaderValue::from_str(value).map_err(|err| {
            ClientError::InvalidArgument(format!("invalid header value for {key}: {err}"))
        })?;
        map.insert(name, value);
    }
    Ok(map)
}

fn collect_response_headers(headers: &HeaderMap) -> BTreeMap<String, String> {
    let mut map = BTreeMap::new();
    for (name, value) in headers {
        let Ok(value) = value.to_str() else {
            continue;
        };
        map.insert(name.as_str().to_ascii_lowercase(), value.to_string());
    }
    map
}
