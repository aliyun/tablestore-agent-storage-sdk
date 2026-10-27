//! High-level [`AgentStorageClient`] and its builder.

use std::collections::BTreeMap;
use std::sync::Arc;
use std::time::Duration;

use serde::de::DeserializeOwned;
use serde_json::Value;

use crate::auth::{Credentials, SignV2, SignV4, Signer};
use crate::error::{ClientError, Error};
use crate::http::OtsJsonClient;
use crate::models::{
    AddDocumentsRequest, AddMemoriesRequest, AddMemoriesResponse, CreateKnowledgeBaseRequest,
    CreateMemoryStoreRequest, CreateMemoryStoreResponse, DeleteDocumentsRequest,
    DeleteKnowledgeBaseRequest, DeleteMemoryRequest, DeleteMemoryResponse,
    DeleteMemoryStoreRequest, DeleteMemoryStoreResponse, DescribeKnowledgeBaseRequest,
    GetDocumentRequest, GetMemoryRequest, GetMemoryResponse, GetMemoryStoreRequest,
    GetMemoryStoreResponse, JsonRequestConvertible, ListChunksRequest, ListDocumentsRequest,
    ListKnowledgeBaseRequest, ListMemoriesRequest, ListMemoriesResponse,
    ListMemoryStoreMessagesRequest, ListMemoryStoreMessagesResponse,
    ListMemoryStoreRequestsRequest, ListMemoryStoreRequestsResponse, ListMemoryStoresRequest,
    ListMemoryStoresResponse, RetrieveRequest, SearchMemoriesRequest, SearchMemoriesResponse,
    UpdateChunksRequest, UpdateDocumentRequest, UpdateKnowledgeBaseRequest, UpdateMemoryRequest,
    UpdateMemoryResponse, UpdateMemoryStoreRequest, UpdateMemoryStoreResponse,
};
use crate::DEFAULT_USER_AGENT;

/// Signing protocol version requested at client build time.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum SignVersion {
    /// HMAC-SHA1 (default, matches the TS SDK default).
    #[default]
    V2,
    /// HMAC-SHA256 with the OTS v4 derived key. Requires `region`.
    V4,
}

/// Optional flags accepted by [`AgentStorageClient::delete_documents`] (and
/// the OSS-extension `delete_documents_with_oss`).
#[derive(Debug, Default, Clone, Copy)]
pub struct DeleteOptions {
    /// When `true`, also delete the backing OSS objects after the documents
    /// are removed from the index. Only honoured by the OSS extension.
    pub delete_file: bool,
}

/// Either a typed request model or a free-form [`serde_json::Value`].
///
/// Every client method accepts an `impl Into<RequestBody>`; pass whichever
/// shape is most convenient. Typed structs catch field-name typos at compile
/// time; `Value` is the simplest path when wiring the SDK into an agent
/// tool-call.
#[derive(Debug, Clone)]
pub enum RequestBody {
    /// A pre-built JSON object.
    Value(Value),
}

impl RequestBody {
    /// Lower into [`serde_json::Value`].
    pub fn into_value(self) -> Value {
        match self {
            RequestBody::Value(value) => value,
        }
    }
}

impl From<Value> for RequestBody {
    fn from(value: Value) -> Self {
        RequestBody::Value(value)
    }
}

impl From<&Value> for RequestBody {
    fn from(value: &Value) -> Self {
        RequestBody::Value(value.clone())
    }
}

macro_rules! request_body_from_model {
    ($($ty:ty),+ $(,)?) => {
        $(
            impl From<$ty> for RequestBody {
                fn from(value: $ty) -> Self {
                    let json = JsonRequestConvertible::to_json_request(&value)
                        .expect("typed request model is always serializable");
                    RequestBody::Value(json)
                }
            }

            impl From<&$ty> for RequestBody {
                fn from(value: &$ty) -> Self {
                    let json = JsonRequestConvertible::to_json_request(value)
                        .expect("typed request model is always serializable");
                    RequestBody::Value(json)
                }
            }
        )+
    };
}

request_body_from_model!(
    CreateKnowledgeBaseRequest,
    DeleteKnowledgeBaseRequest,
    DescribeKnowledgeBaseRequest,
    ListKnowledgeBaseRequest,
    UpdateKnowledgeBaseRequest,
    AddDocumentsRequest,
    UpdateDocumentRequest,
    GetDocumentRequest,
    ListDocumentsRequest,
    DeleteDocumentsRequest,
    RetrieveRequest,
    ListChunksRequest,
    UpdateChunksRequest,
    CreateMemoryStoreRequest,
    GetMemoryStoreRequest,
    ListMemoryStoresRequest,
    UpdateMemoryStoreRequest,
    DeleteMemoryStoreRequest,
    AddMemoriesRequest,
    SearchMemoriesRequest,
    ListMemoriesRequest,
    GetMemoryRequest,
    UpdateMemoryRequest,
    DeleteMemoryRequest,
    ListMemoryStoreMessagesRequest,
    ListMemoryStoreRequestsRequest,
);

/// Cross-platform async client for the Tablestore Agent Storage service.
///
/// Internally wraps an [`OtsJsonClient`] (which holds a `reqwest::Client`) so
/// cloning is cheap and connections are pooled across calls.
#[derive(Clone)]
pub struct AgentStorageClient {
    pub(crate) inner: Arc<AgentStorageInner>,
}

pub(crate) struct AgentStorageInner {
    pub(crate) http: OtsJsonClient,
    #[cfg(feature = "oss-upload")]
    pub(crate) instance_name: String,
    #[cfg(feature = "oss-upload")]
    pub(crate) credentials: Credentials,
}

impl AgentStorageClient {
    /// Start configuring a new client.
    pub fn builder() -> AgentStorageClientBuilder {
        AgentStorageClientBuilder::default()
    }

    async fn request<T: DeserializeOwned>(
        &self,
        action: &str,
        body: RequestBody,
    ) -> Result<T, Error> {
        self.inner.http.request(action, body.into_value()).await
    }

    /// `CreateKnowledgeBase`.
    pub async fn create_knowledge_base(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<Value, Error> {
        self.request("CreateKnowledgeBase", request.into()).await
    }

    /// `DeleteKnowledgeBase`.
    pub async fn delete_knowledge_base(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<Value, Error> {
        self.request("DeleteKnowledgeBase", request.into()).await
    }

    /// `DescribeKnowledgeBase`.
    pub async fn describe_knowledge_base(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<Value, Error> {
        self.request("DescribeKnowledgeBase", request.into()).await
    }

    /// `ListKnowledgeBase`.
    pub async fn list_knowledge_base(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<Value, Error> {
        self.request("ListKnowledgeBase", request.into()).await
    }

    /// `UpdateKnowledgeBase`.
    pub async fn update_knowledge_base(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<Value, Error> {
        self.request("UpdateKnowledgeBase", request.into()).await
    }

    /// `AddDocuments`.
    pub async fn add_documents(&self, request: impl Into<RequestBody>) -> Result<Value, Error> {
        self.request("AddDocuments", request.into()).await
    }

    /// `UpdateDocument`.
    pub async fn update_document(&self, request: impl Into<RequestBody>) -> Result<Value, Error> {
        self.request("UpdateDocument", request.into()).await
    }

    /// `GetDocument`.
    pub async fn get_document(&self, request: impl Into<RequestBody>) -> Result<Value, Error> {
        self.request("GetDocument", request.into()).await
    }

    /// `ListDocuments`.
    pub async fn list_documents(&self, request: impl Into<RequestBody>) -> Result<Value, Error> {
        self.request("ListDocuments", request.into()).await
    }

    /// `DeleteDocuments`. The [`DeleteOptions`] flag is reserved for the
    /// OSS-extension variant — this method ignores it.
    pub async fn delete_documents(&self, request: impl Into<RequestBody>) -> Result<Value, Error> {
        self.request("DeleteDocuments", request.into()).await
    }

    /// `Retrieve`.
    pub async fn retrieve(&self, request: impl Into<RequestBody>) -> Result<Value, Error> {
        self.request("Retrieve", request.into()).await
    }

    /// `ListChunks`.
    pub async fn list_chunks(&self, request: impl Into<RequestBody>) -> Result<Value, Error> {
        self.request("ListChunks", request.into()).await
    }

    /// `UpdateChunks`.
    pub async fn update_chunks(&self, request: impl Into<RequestBody>) -> Result<Value, Error> {
        self.request("UpdateChunks", request.into()).await
    }

    /// `CreateMemoryStore`.
    pub async fn create_memory_store(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<CreateMemoryStoreResponse, Error> {
        self.request("CreateMemoryStore", request.into()).await
    }

    /// `GetMemoryStore`.
    pub async fn get_memory_store(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<GetMemoryStoreResponse, Error> {
        self.request("GetMemoryStore", request.into()).await
    }

    /// `ListMemoryStores`.
    pub async fn list_memory_stores(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<ListMemoryStoresResponse, Error> {
        self.request("ListMemoryStores", request.into()).await
    }

    /// `UpdateMemoryStore`.
    pub async fn update_memory_store(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<UpdateMemoryStoreResponse, Error> {
        self.request("UpdateMemoryStore", request.into()).await
    }

    /// `DeleteMemoryStore`.
    pub async fn delete_memory_store(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<DeleteMemoryStoreResponse, Error> {
        self.request("DeleteMemoryStore", request.into()).await
    }

    /// `AddMemories`.
    pub async fn add_memories(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<AddMemoriesResponse, Error> {
        self.request("AddMemories", request.into()).await
    }

    /// `SearchMemories`.
    pub async fn search_memories(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<SearchMemoriesResponse, Error> {
        self.request("SearchMemories", request.into()).await
    }

    /// `ListMemories`.
    pub async fn list_memories(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<ListMemoriesResponse, Error> {
        self.request("ListMemories", request.into()).await
    }

    /// `GetMemory`.
    pub async fn get_memory(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<GetMemoryResponse, Error> {
        self.request("GetMemory", request.into()).await
    }

    /// `UpdateMemory`.
    pub async fn update_memory(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<UpdateMemoryResponse, Error> {
        self.request("UpdateMemory", request.into()).await
    }

    /// `DeleteMemory`.
    pub async fn delete_memory(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<DeleteMemoryResponse, Error> {
        self.request("DeleteMemory", request.into()).await
    }

    /// `ListMemoryStoreMessages`.
    pub async fn list_memory_store_messages(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<ListMemoryStoreMessagesResponse, Error> {
        self.request("ListMemoryStoreMessages", request.into())
            .await
    }

    /// `ListMemoryStoreRequests`.
    pub async fn list_memory_store_requests(
        &self,
        request: impl Into<RequestBody>,
    ) -> Result<ListMemoryStoreRequestsResponse, Error> {
        self.request("ListMemoryStoreRequests", request.into())
            .await
    }
}

/// Builder for [`AgentStorageClient`].
///
/// Required fields: `endpoint`, `instance_name`, `access_key_id`,
/// `access_key_secret`. Use [`AgentStorageClientBuilder::sign_version`] +
/// [`AgentStorageClientBuilder::region`] when calling the V4 protocol.
#[derive(Default)]
pub struct AgentStorageClientBuilder {
    endpoint: Option<String>,
    instance_name: Option<String>,
    access_key_id: Option<String>,
    access_key_secret: Option<String>,
    sts_token: Option<String>,
    region: Option<String>,
    sign_version: SignVersion,
    http_client: Option<reqwest::Client>,
    timeout: Option<Duration>,
    user_agent_suffix: Option<String>,
    extra_headers: BTreeMap<String, String>,
}

impl AgentStorageClientBuilder {
    /// Tablestore endpoint, e.g. `https://your-instance.cn-beijing.ots.aliyuncs.com`.
    pub fn endpoint(mut self, value: impl Into<String>) -> Self {
        self.endpoint = Some(value.into());
        self
    }

    /// Tablestore instance name.
    pub fn instance_name(mut self, value: impl Into<String>) -> Self {
        self.instance_name = Some(value.into());
        self
    }

    /// AccessKey id.
    pub fn access_key_id(mut self, value: impl Into<String>) -> Self {
        self.access_key_id = Some(value.into());
        self
    }

    /// AccessKey secret.
    pub fn access_key_secret(mut self, value: impl Into<String>) -> Self {
        self.access_key_secret = Some(value.into());
        self
    }

    /// Optional STS session token (sent as `x-ots-ststoken`).
    pub fn sts_token(mut self, value: impl Into<String>) -> Self {
        self.sts_token = Some(value.into());
        self
    }

    /// Region (required for V4 signing).
    pub fn region(mut self, value: impl Into<String>) -> Self {
        self.region = Some(value.into());
        self
    }

    /// Sign protocol version. Defaults to V2.
    pub fn sign_version(mut self, value: SignVersion) -> Self {
        self.sign_version = value;
        self
    }

    /// Supply a pre-built `reqwest::Client` (useful for mTLS, proxies, custom
    /// connection pools, or sharing a client across multiple SDKs).
    pub fn http_client(mut self, client: reqwest::Client) -> Self {
        self.http_client = Some(client);
        self
    }

    /// Request timeout applied to the default `reqwest::Client`. Ignored when
    /// a custom client is supplied via [`Self::http_client`].
    pub fn timeout(mut self, timeout: Duration) -> Self {
        self.timeout = Some(timeout);
        self
    }

    /// Append a suffix to the default `User-Agent`. Useful for telemetry.
    pub fn user_agent_suffix(mut self, suffix: impl Into<String>) -> Self {
        self.user_agent_suffix = Some(suffix.into());
        self
    }

    /// Attach a single header to every signed request. Repeated calls
    /// accumulate; calling with the same key overrides the previous value.
    /// `x-ots-*` keys take part in the signature.
    pub fn extra_header(mut self, key: impl Into<String>, value: impl Into<String>) -> Self {
        self.extra_headers.insert(key.into(), value.into());
        self
    }

    /// Replace the entire extra-header map. Useful when callers already keep a
    /// map handy; for incremental additions prefer [`Self::extra_header`].
    pub fn extra_headers(mut self, headers: BTreeMap<String, String>) -> Self {
        self.extra_headers = headers;
        self
    }

    /// Build the client. Returns [`ClientError::MissingField`] when a
    /// required field is absent, or [`ClientError::InvalidArgument`] when
    /// V4 is requested without a region.
    pub fn build(self) -> Result<AgentStorageClient, ClientError> {
        let endpoint = self.endpoint.ok_or(ClientError::MissingField("endpoint"))?;
        let instance_name = self
            .instance_name
            .ok_or(ClientError::MissingField("instance_name"))?;
        let access_key_id = self
            .access_key_id
            .ok_or(ClientError::MissingField("access_key_id"))?;
        let access_key_secret = self
            .access_key_secret
            .ok_or(ClientError::MissingField("access_key_secret"))?;

        let credentials = Credentials {
            access_key_id,
            access_key_secret,
            security_token: self.sts_token,
        };

        let signer: Box<dyn Signer> = match self.sign_version {
            SignVersion::V2 => Box::new(SignV2::new()),
            SignVersion::V4 => {
                let region = self.region.clone().ok_or_else(|| {
                    ClientError::InvalidArgument("region is required for V4 signing".to_string())
                })?;
                Box::new(SignV4::new(region)?)
            }
        };

        let http_client = match self.http_client {
            Some(client) => client,
            None => {
                let mut builder = reqwest::Client::builder();
                if let Some(timeout) = self.timeout {
                    builder = builder.timeout(timeout);
                }
                builder.build().map_err(ClientError::from)?
            }
        };

        let user_agent = match self.user_agent_suffix {
            Some(suffix) => format!("{DEFAULT_USER_AGENT} {suffix}"),
            None => DEFAULT_USER_AGENT.to_string(),
        };

        let http = OtsJsonClient::new(
            endpoint,
            instance_name.clone(),
            credentials.clone(),
            signer,
            http_client,
            user_agent,
            self.extra_headers,
        );

        Ok(AgentStorageClient {
            inner: Arc::new(AgentStorageInner {
                http,
                #[cfg(feature = "oss-upload")]
                instance_name,
                #[cfg(feature = "oss-upload")]
                credentials,
            }),
        })
    }
}
