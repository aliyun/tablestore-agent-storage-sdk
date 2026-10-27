use std::sync::Arc;

use async_trait::async_trait;
use serde_json::{Map, Value};

use crate::client::AgentStorageClient;
use crate::error::{ClientError, Error};
use crate::DeleteOptions;

use super::path::{build_oss_full_path, build_oss_object_key, parse_oss_path};
use super::uploader::{AliyunOssUploader, OssUploader};

/// Bucket-side configuration consumed by [`AgentStorageClientExt::upload_documents`].
#[derive(Debug, Clone)]
pub struct OssUploadConfig {
    /// OSS endpoint (e.g. `https://oss-cn-beijing.aliyuncs.com`).
    pub endpoint: String,
    /// Bucket name files are uploaded into.
    pub bucket: String,
}

/// OSS extension methods bolted onto [`AgentStorageClient`] when the
/// `oss-upload` feature is enabled.
///
/// `upload_documents` mirrors `NodeAgentStorageClient.uploadDocuments` in the
/// TS SDK: each local file is uploaded to OSS first, then the resulting
/// `oss://bucket/key` URIs are handed to `AddDocuments`. If the index call
/// fails, the uploaded objects are best-effort deleted.
#[async_trait]
pub trait AgentStorageClientExt {
    /// Upload local files to OSS, then call `AddDocuments`.
    async fn upload_documents(&self, request: Value, oss: &OssUploadConfig)
        -> Result<Value, Error>;

    /// Same as [`AgentStorageClient::delete_documents`], with an option to
    /// also delete the backing OSS objects.
    ///
    /// With `delete_file = true`, the index entry is deleted first and the OSS
    /// objects are deleted second. If the OSS delete fails the error is
    /// surfaced as [`Error::Client`] — the caller knows the index row is gone
    /// but the bucket object may still be present.
    async fn delete_documents_with_oss(
        &self,
        request: Value,
        oss: &OssUploadConfig,
        options: DeleteOptions,
    ) -> Result<Value, Error>;

    /// Override the OSS uploader (e.g. for tests). The default is an
    /// [`AliyunOssUploader`] built from the configured Tablestore credentials.
    async fn upload_documents_with(
        &self,
        request: Value,
        oss: &OssUploadConfig,
        uploader: Arc<dyn OssUploader>,
    ) -> Result<Value, Error>;

    /// Same as [`Self::delete_documents_with_oss`] with a caller-supplied
    /// uploader. Useful for tests, alternate clouds, or custom retry / telemetry
    /// wrappers.
    async fn delete_documents_with_oss_using(
        &self,
        request: Value,
        oss: &OssUploadConfig,
        options: DeleteOptions,
        uploader: Arc<dyn OssUploader>,
    ) -> Result<Value, Error>;
}

#[async_trait]
impl AgentStorageClientExt for AgentStorageClient {
    async fn upload_documents(
        &self,
        request: Value,
        oss: &OssUploadConfig,
    ) -> Result<Value, Error> {
        let uploader = self.default_uploader(oss)?;
        self.upload_documents_with(request, oss, uploader).await
    }

    async fn upload_documents_with(
        &self,
        request: Value,
        oss: &OssUploadConfig,
        uploader: Arc<dyn OssUploader>,
    ) -> Result<Value, Error> {
        let request_obj = request.as_object().ok_or_else(|| {
            ClientError::InvalidArgument(
                "upload_documents request must be a JSON object".to_string(),
            )
        })?;
        let knowledge_base_name = request_obj
            .get("knowledgeBaseName")
            .and_then(Value::as_str)
            .ok_or_else(|| {
                ClientError::InvalidArgument("request must contain 'knowledgeBaseName'".to_string())
            })?
            .to_string();
        let subspace = request_obj
            .get("subspace")
            .and_then(Value::as_str)
            .map(str::to_string);
        let documents = request_obj
            .get("documents")
            .and_then(Value::as_array)
            .ok_or_else(|| {
                ClientError::InvalidArgument("request must contain 'documents' array".to_string())
            })?
            .clone();

        let mut uploaded_keys: Vec<String> = Vec::with_capacity(documents.len());
        let mut prepared: Vec<Value> = Vec::with_capacity(documents.len());

        for document in &documents {
            let doc_obj = document.as_object().ok_or_else(|| {
                ClientError::InvalidArgument("each document must be a JSON object".to_string())
            })?;
            let file_path = doc_obj
                .get("filePath")
                .and_then(Value::as_str)
                .ok_or_else(|| {
                    ClientError::InvalidArgument(
                        "each document must contain 'filePath'".to_string(),
                    )
                })?;
            if !tokio::fs::try_exists(file_path).await.unwrap_or(false) {
                return Err(Error::Client(ClientError::FileNotFound(
                    file_path.to_string(),
                )));
            }
            let object_key = build_oss_object_key(
                self.inner.instance_name.as_str(),
                &knowledge_base_name,
                subspace.as_deref(),
                file_path,
            );
            uploader
                .put_object(&object_key, file_path)
                .await
                .map_err(Error::Client)?;
            uploaded_keys.push(object_key.clone());

            let mut next = doc_obj.clone();
            next.remove("filePath");
            next.insert(
                "ossKey".to_string(),
                Value::String(build_oss_full_path(&oss.bucket, &object_key)),
            );
            prepared.push(Value::Object(next));
        }

        let mut add_request = Map::new();
        add_request.insert(
            "knowledgeBaseName".to_string(),
            Value::String(knowledge_base_name.clone()),
        );
        if let Some(subspace) = &subspace {
            add_request.insert("subspace".to_string(), Value::String(subspace.clone()));
        }
        add_request.insert("documents".to_string(), Value::Array(prepared));

        match self.add_documents(Value::Object(add_request)).await {
            Ok(response) => Ok(response),
            Err(Error::Service(err)) => {
                let _ = uploader.delete_objects(&uploaded_keys).await;
                Err(Error::Service(err))
            }
            Err(other) => Err(other),
        }
    }

    async fn delete_documents_with_oss(
        &self,
        request: Value,
        oss: &OssUploadConfig,
        options: DeleteOptions,
    ) -> Result<Value, Error> {
        let uploader = self.default_uploader(oss)?;
        self.delete_documents_with_oss_using(request, oss, options, uploader)
            .await
    }

    async fn delete_documents_with_oss_using(
        &self,
        request: Value,
        oss: &OssUploadConfig,
        options: DeleteOptions,
        uploader: Arc<dyn OssUploader>,
    ) -> Result<Value, Error> {
        if !options.delete_file {
            return self.delete_documents(request).await;
        }

        let request_obj = request
            .as_object()
            .ok_or_else(|| {
                ClientError::InvalidArgument(
                    "delete_documents request must be a JSON object".to_string(),
                )
            })?
            .clone();
        let knowledge_base_name = request_obj
            .get("knowledgeBaseName")
            .and_then(Value::as_str)
            .ok_or_else(|| {
                ClientError::InvalidArgument("request must contain 'knowledgeBaseName'".to_string())
            })?
            .to_string();
        let subspace = request_obj
            .get("subspace")
            .and_then(Value::as_str)
            .map(str::to_string);
        let documents = request_obj
            .get("documents")
            .and_then(Value::as_array)
            .ok_or_else(|| {
                ClientError::InvalidArgument("request must contain 'documents' array".to_string())
            })?
            .clone();

        let mut object_keys: Vec<String> = Vec::new();
        for document in &documents {
            let doc_obj = document.as_object().ok_or_else(|| {
                ClientError::InvalidArgument("each document must be a JSON object".to_string())
            })?;
            let mut oss_key = doc_obj
                .get("ossKey")
                .and_then(Value::as_str)
                .map(str::to_string);

            if oss_key.is_none() {
                if let Some(doc_id) = doc_obj.get("docId").and_then(Value::as_str) {
                    let mut lookup = Map::new();
                    lookup.insert(
                        "knowledgeBaseName".to_string(),
                        Value::String(knowledge_base_name.clone()),
                    );
                    if let Some(subspace) = &subspace {
                        lookup.insert("subspace".to_string(), Value::String(subspace.clone()));
                    }
                    lookup.insert("docId".to_string(), Value::String(doc_id.to_string()));
                    let response = self.get_document(Value::Object(lookup)).await?;
                    oss_key = response
                        .get("data")
                        .and_then(Value::as_array)
                        .and_then(|arr| arr.first())
                        .and_then(|first| first.get("ossKey"))
                        .and_then(Value::as_str)
                        .map(str::to_string);
                }
            }

            let Some(oss_key) = oss_key else {
                continue;
            };
            let Some(parsed) = parse_oss_path(&oss_key, Some(&oss.bucket)) else {
                continue;
            };
            if parsed.bucket_name != oss.bucket {
                continue;
            }
            object_keys.push(parsed.object_key);
        }

        let result = self.delete_documents(Value::Object(request_obj)).await?;
        uploader.delete_objects(&object_keys).await?;
        Ok(result)
    }
}

impl AgentStorageClient {
    fn default_uploader(&self, oss: &OssUploadConfig) -> Result<Arc<dyn OssUploader>, ClientError> {
        let creds = &self.inner.credentials;
        let uploader = AliyunOssUploader::new(
            oss.endpoint.clone(),
            oss.bucket.clone(),
            creds.access_key_id.clone(),
            creds.access_key_secret.clone(),
            creds.security_token.clone(),
        )?;
        Ok(Arc::new(uploader))
    }
}
