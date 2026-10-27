//! Request and response models for every supported API.
//!
//! The shape and naming of these types mirrors the TypeScript SDK
//! (`@tablestore/agent-storage`) on purpose, so an agent that already knows the
//! TS surface can reach for the same field names.
//!
//! All request models implement [`base::JsonRequestConvertible`] and can be
//! handed straight to a client method. Anywhere a typed model is accepted, a
//! `serde_json::Value` is accepted too — see [`crate::RequestBody`].

mod base;
mod chunk;
mod document;
mod enums;
mod knowledge_base;
mod memory;
mod metadata_filter;
mod patch;
mod retrieve;

pub use base::{json_value_or_null, JsonRequestConvertible};
pub use chunk::{ListChunksRequest, UpdateChunkItem, UpdateChunksRequest};
pub use document::{
    AddDocumentItem, AddDocumentsRequest, DeleteDocumentItem, DeleteDocumentsRequest,
    GetDocumentRequest, ListDocumentsRequest, UpdateDocumentRequest, UploadDocumentItem,
    UploadDocumentsRequest,
};
pub use enums::{MetadataFieldType, RerankingType, RetrievalQueryType, SearchType};
pub use knowledge_base::{
    CreateKnowledgeBaseRequest, DeleteKnowledgeBaseRequest, DescribeKnowledgeBaseRequest,
    EmbeddingConfiguration, ListKnowledgeBaseRequest, MetadataField, UpdateKnowledgeBaseRequest,
};
pub use memory::{
    AddMemoriesRequest, AddMemoriesResponse, CreateMemoryStoreRequest, CreateMemoryStoreResponse,
    DeleteMemoryRequest, DeleteMemoryResponse, DeleteMemoryStoreRequest, DeleteMemoryStoreResponse,
    GetMemoryRequest, GetMemoryResponse, GetMemoryStoreRequest, GetMemoryStoreResponse,
    ListMemoriesRequest, ListMemoriesResponse, ListMemoryStoreMessagesRequest,
    ListMemoryStoreMessagesResponse, ListMemoryStoreRequestsRequest,
    ListMemoryStoreRequestsResponse, ListMemoryStoresRequest, ListMemoryStoresResponse,
    MemoryStoreInfo, MemoryStoreListItem, MemoryStoreRequestRecord, MemoryStoreSession, MemoryUnit,
    MessagePayload, Scope, SearchHit, SearchMemoriesRequest, SearchMemoriesResponse,
    SessionMessage, UpdateMemoryRequest, UpdateMemoryResponse, UpdateMemoryStoreRequest,
    UpdateMemoryStoreResponse,
};
pub use metadata_filter::{MetadataFilter, MetadataFilterBuilder, PrimitiveValue};
pub use patch::Patch;
pub use retrieve::{
    DenseVectorSearchConfiguration, FullTextSearchConfiguration, ModelConfiguration,
    RRFConfiguration, RerankingConfiguration, RetrievalConfiguration, RetrievalQuery,
    RetrieveRequest, WeightConfiguration,
};
