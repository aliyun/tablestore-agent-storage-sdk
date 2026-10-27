use std::fmt;

use serde_json::{Map, Value};

use crate::error::ClientError;

use super::base::{assign_if_some, to_json_value, JsonRequestConvertible};
use super::enums::MetadataFieldType;
use super::patch::Patch;
use super::retrieve::RetrievalConfiguration;

/// One field of a knowledge-base metadata schema.
#[derive(Debug, Clone)]
pub struct MetadataField {
    /// Field name.
    pub name: String,
    /// Field type. Use the [`MetadataFieldType`] variants, or a raw string for
    /// types the SDK does not yet recognise.
    pub type_name: String,
}

impl MetadataField {
    /// Build from a known enum variant.
    pub fn new(name: impl Into<String>, ty: MetadataFieldType) -> Self {
        let serialized = serde_json::to_value(ty)
            .ok()
            .and_then(|v| v.as_str().map(str::to_string))
            .unwrap_or_else(|| "string".to_string());
        Self {
            name: name.into(),
            type_name: serialized,
        }
    }

    /// Build with a raw type string — escape hatch for future server-side types.
    pub fn with_raw_type(name: impl Into<String>, type_name: impl Into<String>) -> Self {
        Self {
            name: name.into(),
            type_name: type_name.into(),
        }
    }
}

impl JsonRequestConvertible for MetadataField {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert("name".to_string(), Value::String(self.name.clone()));
        map.insert("type".to_string(), Value::String(self.type_name.clone()));
        Ok(Value::Object(map))
    }
}

/// Embedding configuration (`/CreateKnowledgeBase`).
///
/// `Debug` is implemented manually so [`EmbeddingConfiguration::api_key`] is
/// redacted in logs.
#[derive(Default, Clone)]
pub struct EmbeddingConfiguration {
    /// Provider name (e.g. `dashscope`).
    pub provider: Option<String>,
    /// Endpoint URL.
    pub url: Option<String>,
    /// API key forwarded to the embedding service.
    pub api_key: Option<String>,
    /// Model name.
    pub model: Option<String>,
    /// Embedding dimension.
    pub dimension: Option<u32>,
}

impl fmt::Debug for EmbeddingConfiguration {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("EmbeddingConfiguration")
            .field("provider", &self.provider)
            .field("url", &self.url)
            .field("api_key", &self.api_key.as_ref().map(|_| "[redacted]"))
            .field("model", &self.model)
            .field("dimension", &self.dimension)
            .finish()
    }
}

impl JsonRequestConvertible for EmbeddingConfiguration {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        assign_if_some(
            &mut map,
            "provider",
            self.provider.clone().map(Value::String),
        );
        assign_if_some(&mut map, "url", self.url.clone().map(Value::String));
        assign_if_some(&mut map, "apiKey", self.api_key.clone().map(Value::String));
        assign_if_some(&mut map, "model", self.model.clone().map(Value::String));
        assign_if_some(
            &mut map,
            "dimension",
            self.dimension.map(|v| Value::Number(v.into())),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `CreateKnowledgeBase`.
#[derive(Debug, Default, Clone)]
pub struct CreateKnowledgeBaseRequest {
    /// Knowledge base identifier (required).
    pub knowledge_base_name: String,
    /// Human-readable description.
    pub description: Option<String>,
    /// Whether to enable subspaces.
    pub subspace: Option<bool>,
    /// Free-form tags.
    pub tags: Option<Vec<String>>,
    /// Metadata schema for documents in this knowledge base.
    pub metadata: Option<Vec<MetadataField>>,
    /// Default retrieval configuration applied at query time.
    pub retrieval_configuration: Option<RetrievalConfiguration>,
    /// Embedding configuration used for dense vector search.
    pub embedding_configuration: Option<EmbeddingConfiguration>,
}

impl CreateKnowledgeBaseRequest {
    /// Build with only the required field.
    pub fn new(knowledge_base_name: impl Into<String>) -> Self {
        Self {
            knowledge_base_name: knowledge_base_name.into(),
            ..Self::default()
        }
    }
}

impl JsonRequestConvertible for CreateKnowledgeBaseRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        assign_if_some(
            &mut map,
            "description",
            self.description.clone().map(Value::String),
        );
        assign_if_some(&mut map, "subspace", self.subspace.map(Value::Bool));
        assign_if_some(
            &mut map,
            "tags",
            self.tags
                .as_ref()
                .map(|t| Value::Array(t.iter().cloned().map(Value::String).collect())),
        );
        if let Some(metadata) = &self.metadata {
            let items: Result<Vec<Value>, ClientError> =
                metadata.iter().map(|m| m.to_json_request()).collect();
            map.insert("metadata".to_string(), Value::Array(items?));
        }
        assign_if_some(
            &mut map,
            "retrievalConfiguration",
            to_json_value(self.retrieval_configuration.as_ref())?,
        );
        assign_if_some(
            &mut map,
            "embeddingConfiguration",
            to_json_value(self.embedding_configuration.as_ref())?,
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `DeleteKnowledgeBase`.
#[derive(Debug, Clone)]
pub struct DeleteKnowledgeBaseRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
}

impl DeleteKnowledgeBaseRequest {
    /// Convenience constructor.
    pub fn new(knowledge_base_name: impl Into<String>) -> Self {
        Self {
            knowledge_base_name: knowledge_base_name.into(),
        }
    }
}

impl JsonRequestConvertible for DeleteKnowledgeBaseRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `DescribeKnowledgeBase`.
#[derive(Debug, Clone)]
pub struct DescribeKnowledgeBaseRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
}

impl DescribeKnowledgeBaseRequest {
    /// Convenience constructor.
    pub fn new(knowledge_base_name: impl Into<String>) -> Self {
        Self {
            knowledge_base_name: knowledge_base_name.into(),
        }
    }
}

impl JsonRequestConvertible for DescribeKnowledgeBaseRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `ListKnowledgeBase`.
#[derive(Debug, Default, Clone)]
pub struct ListKnowledgeBaseRequest {
    /// Optional page size.
    pub max_results: Option<u32>,
    /// Continuation token from a previous response.
    pub next_token: Option<String>,
}

impl JsonRequestConvertible for ListKnowledgeBaseRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        assign_if_some(
            &mut map,
            "maxResults",
            self.max_results.map(|v| Value::Number(v.into())),
        );
        assign_if_some(
            &mut map,
            "nextToken",
            self.next_token.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `UpdateKnowledgeBase`. Use [`Patch`] to express
/// "leave unchanged" vs. "explicitly clear" for nullable fields.
#[derive(Debug, Clone)]
pub struct UpdateKnowledgeBaseRequest {
    /// Knowledge base identifier (required).
    pub knowledge_base_name: String,
    /// Description patch.
    pub description: Patch<String>,
    /// Tags patch.
    pub tags: Patch<Vec<String>>,
    /// Retrieval configuration patch.
    pub retrieval_configuration: Option<RetrievalConfiguration>,
}

impl UpdateKnowledgeBaseRequest {
    /// Build with only the required field.
    pub fn new(knowledge_base_name: impl Into<String>) -> Self {
        Self {
            knowledge_base_name: knowledge_base_name.into(),
            description: Patch::Unchanged,
            tags: Patch::Unchanged,
            retrieval_configuration: None,
        }
    }
}

impl JsonRequestConvertible for UpdateKnowledgeBaseRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        match &self.description {
            Patch::Unchanged => {}
            Patch::Set(value) => {
                map.insert("description".to_string(), Value::String(value.clone()));
            }
            Patch::Clear => {
                map.insert("description".to_string(), Value::Null);
            }
        }
        match &self.tags {
            Patch::Unchanged => {}
            Patch::Set(value) => {
                map.insert(
                    "tags".to_string(),
                    Value::Array(value.iter().cloned().map(Value::String).collect()),
                );
            }
            Patch::Clear => {
                map.insert("tags".to_string(), Value::Null);
            }
        }
        assign_if_some(
            &mut map,
            "retrievalConfiguration",
            to_json_value(self.retrieval_configuration.as_ref())?,
        );
        Ok(Value::Object(map))
    }
}
