use std::sync::Arc;

use async_trait::async_trait;
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use chrono::Utc;
use hmac::{Hmac, Mac};
use md5::{Digest, Md5};
use reqwest::header::{HeaderMap, HeaderName, HeaderValue, AUTHORIZATION, CONTENT_TYPE, DATE};
use reqwest::Method;
use sha1::Sha1;
use tokio::fs;

use crate::error::ClientError;
use crate::http::strip_trailing_slashes;

/// Async interface implemented by any uploader that can put objects into,
/// and delete objects from, an object-store bucket.
///
/// The built-in [`AliyunOssUploader`] is the default — supply your own to
/// plug into tests, alternate storage backends, or a richer OSS SDK.
#[async_trait]
pub trait OssUploader: Send + Sync {
    /// Upload `file_path` to `object_key`.
    async fn put_object(&self, object_key: &str, file_path: &str) -> Result<(), ClientError>;

    /// Delete every object in `object_keys` from the bucket.
    async fn delete_objects(&self, object_keys: &[String]) -> Result<(), ClientError>;
}

/// Minimal Aliyun OSS uploader implementing PutObject + DeleteMultipleObjects.
///
/// Only the two operations consumed by `upload_documents` / `delete_documents`
/// are supported. If you need full OSS surface, pass a richer uploader via
/// [`crate::oss::AgentStorageClientExt`].
#[derive(Clone)]
pub struct AliyunOssUploader {
    inner: Arc<Inner>,
}

struct Inner {
    endpoint: String,
    bucket: String,
    access_key_id: String,
    access_key_secret: String,
    sts_token: Option<String>,
    http: reqwest::Client,
}

impl AliyunOssUploader {
    /// Build with a default `reqwest::Client`. `endpoint` should look like
    /// `https://oss-cn-beijing.aliyuncs.com`.
    pub fn new(
        endpoint: impl Into<String>,
        bucket: impl Into<String>,
        access_key_id: impl Into<String>,
        access_key_secret: impl Into<String>,
        sts_token: Option<String>,
    ) -> Result<Self, ClientError> {
        let http = reqwest::Client::builder()
            .build()
            .map_err(ClientError::from)?;
        Ok(Self::with_client(
            endpoint,
            bucket,
            access_key_id,
            access_key_secret,
            sts_token,
            http,
        ))
    }

    /// Build with a caller-supplied `reqwest::Client`.
    pub fn with_client(
        endpoint: impl Into<String>,
        bucket: impl Into<String>,
        access_key_id: impl Into<String>,
        access_key_secret: impl Into<String>,
        sts_token: Option<String>,
        http: reqwest::Client,
    ) -> Self {
        Self {
            inner: Arc::new(Inner {
                endpoint: strip_trailing_slashes(endpoint.into()),
                bucket: bucket.into(),
                access_key_id: access_key_id.into(),
                access_key_secret: access_key_secret.into(),
                sts_token,
                http,
            }),
        }
    }

    fn host_for_bucket(&self) -> String {
        let endpoint_no_scheme = self
            .inner
            .endpoint
            .trim_start_matches("https://")
            .trim_start_matches("http://");
        format!("{}.{}", self.inner.bucket, endpoint_no_scheme)
    }

    fn build_url(&self, key: &str, query: Option<&str>) -> String {
        let host = self.host_for_bucket();
        let scheme = if self.inner.endpoint.starts_with("http://") {
            "http"
        } else {
            "https"
        };
        let key = key.trim_start_matches('/');
        match query {
            Some(q) => format!("{scheme}://{host}/{key}?{q}"),
            None => format!("{scheme}://{host}/{key}"),
        }
    }

    fn sign(
        &self,
        method: &Method,
        content_md5: &str,
        content_type: &str,
        date: &str,
        canonical_resource: &str,
        extra_oss_headers: &[(String, String)],
    ) -> String {
        let mut header_pairs: Vec<(String, String)> = Vec::new();
        for (k, v) in extra_oss_headers {
            header_pairs.push((k.to_ascii_lowercase(), v.trim().to_string()));
        }
        if let Some(token) = &self.inner.sts_token {
            header_pairs.push(("x-oss-security-token".to_string(), token.clone()));
        }
        header_pairs.sort_by(|a, b| a.0.cmp(&b.0));
        let canonical_headers: String = header_pairs
            .iter()
            .map(|(k, v)| format!("{k}:{v}\n"))
            .collect();

        let string_to_sign = format!(
            "{verb}\n{md5}\n{ct}\n{date}\n{canon_headers}{canon_resource}",
            verb = method.as_str(),
            md5 = content_md5,
            ct = content_type,
            date = date,
            canon_headers = canonical_headers,
            canon_resource = canonical_resource,
        );

        let mut mac = Hmac::<Sha1>::new_from_slice(self.inner.access_key_secret.as_bytes())
            .expect("HMAC accepts any key length");
        mac.update(string_to_sign.as_bytes());
        let signature = BASE64.encode(mac.finalize().into_bytes());
        format!("OSS {}:{}", self.inner.access_key_id, signature)
    }
}

#[async_trait]
impl OssUploader for AliyunOssUploader {
    async fn put_object(&self, object_key: &str, file_path: &str) -> Result<(), ClientError> {
        let bytes = fs::read(file_path).await.map_err(|err| match err.kind() {
            std::io::ErrorKind::NotFound => ClientError::FileNotFound(file_path.to_string()),
            _ => ClientError::from(err),
        })?;
        let mut hasher = Md5::new();
        hasher.update(&bytes);
        let content_md5 = BASE64.encode(hasher.finalize());
        let date = format_http_date(Utc::now());
        let content_type = "application/octet-stream";
        let canonical_resource = format!(
            "/{}/{}",
            self.inner.bucket,
            object_key.trim_start_matches('/')
        );

        let authorization = self.sign(
            &Method::PUT,
            &content_md5,
            content_type,
            &date,
            &canonical_resource,
            &[],
        );

        let mut headers = HeaderMap::new();
        headers.insert(AUTHORIZATION, header_value(&authorization)?);
        headers.insert(DATE, header_value(&date)?);
        headers.insert("content-md5", header_value(&content_md5)?);
        headers.insert(
            CONTENT_TYPE,
            HeaderValue::from_static("application/octet-stream"),
        );
        if let Some(token) = &self.inner.sts_token {
            headers.insert(
                HeaderName::from_static("x-oss-security-token"),
                header_value(token)?,
            );
        }

        let url = self.build_url(object_key, None);
        let response = self
            .inner
            .http
            .put(&url)
            .headers(headers)
            .body(bytes)
            .send()
            .await
            .map_err(ClientError::from)?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(ClientError::Oss(format!(
                "PutObject {object_key} failed: status={status} body={body}"
            )));
        }
        Ok(())
    }

    async fn delete_objects(&self, object_keys: &[String]) -> Result<(), ClientError> {
        if object_keys.is_empty() {
            return Ok(());
        }

        let mut xml =
            String::from(r#"<?xml version="1.0" encoding="UTF-8"?><Delete><Quiet>true</Quiet>"#);
        for key in object_keys {
            xml.push_str("<Object><Key>");
            xml.push_str(&xml_escape(key));
            xml.push_str("</Key></Object>");
        }
        xml.push_str("</Delete>");
        let body = xml.into_bytes();

        let mut hasher = Md5::new();
        hasher.update(&body);
        let content_md5 = BASE64.encode(hasher.finalize());
        let date = format_http_date(Utc::now());
        let content_type = "application/xml";
        let canonical_resource = format!("/{}/?delete", self.inner.bucket);

        let authorization = self.sign(
            &Method::POST,
            &content_md5,
            content_type,
            &date,
            &canonical_resource,
            &[],
        );

        let mut headers = HeaderMap::new();
        headers.insert(AUTHORIZATION, header_value(&authorization)?);
        headers.insert(DATE, header_value(&date)?);
        headers.insert("content-md5", header_value(&content_md5)?);
        headers.insert(CONTENT_TYPE, HeaderValue::from_static("application/xml"));
        if let Some(token) = &self.inner.sts_token {
            headers.insert(
                HeaderName::from_static("x-oss-security-token"),
                header_value(token)?,
            );
        }

        let url = self.build_url("", Some("delete"));
        let response = self
            .inner
            .http
            .post(&url)
            .headers(headers)
            .body(body)
            .send()
            .await
            .map_err(ClientError::from)?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(ClientError::Oss(format!(
                "DeleteMultipleObjects failed: status={status} body={body}"
            )));
        }
        Ok(())
    }
}

fn header_value(value: &str) -> Result<HeaderValue, ClientError> {
    HeaderValue::from_str(value)
        .map_err(|err| ClientError::InvalidArgument(format!("invalid header value: {err}")))
}

fn format_http_date(date: chrono::DateTime<Utc>) -> String {
    date.format("%a, %d %b %Y %H:%M:%S GMT").to_string()
}

fn xml_escape(value: &str) -> String {
    let mut out = String::with_capacity(value.len());
    for ch in value.chars() {
        match ch {
            '&' => out.push_str("&amp;"),
            '<' => out.push_str("&lt;"),
            '>' => out.push_str("&gt;"),
            '"' => out.push_str("&quot;"),
            '\'' => out.push_str("&apos;"),
            _ => out.push(ch),
        }
    }
    out
}
