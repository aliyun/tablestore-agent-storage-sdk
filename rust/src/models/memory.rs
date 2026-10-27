use std::collections::HashMap;

use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};

use crate::error::ClientError;

use super::base::{assign_if_some, JsonRequestConvertible};

fn string_map_to_value(value: &HashMap<String, String>) -> Value {
    let mut map = Map::new();
    for (k, v) in value {
        map.insert(k.clone(), Value::String(v.clone()));
    }
    Value::Object(map)
}

/// Scope identifying which agent / run a memory belongs to.
#[derive(Debug, Default, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct Scope {
    /// Application id.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub app_id: Option<String>,
    /// Tenant id.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tenant_id: Option<String>,
    /// Agent id.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub agent_id: Option<String>,
    /// Run id.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub run_id: Option<String>,
}

impl JsonRequestConvertible for Scope {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        serde_json::to_value(self).map_err(|err| {
            ClientError::InvalidArgument(format!("failed to serialize Scope: {err}"))
        })
    }
}

/// One message in a conversation transcript.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub struct MessagePayload {
    /// Speaker role (`user` / `assistant` / `system`).
    pub role: String,
    /// Message body.
    pub content: String,
    /// Optional caller-supplied id.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message_id: Option<String>,
    /// ISO-8601 timestamp.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub timestamp: Option<String>,
    /// Free-form metadata.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, String>>,
}

impl JsonRequestConvertible for MessagePayload {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        serde_json::to_value(self).map_err(|err| {
            ClientError::InvalidArgument(format!("failed to serialize MessagePayload: {err}"))
        })
    }
}

/// Memory unit returned by Get / List / Search.
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct MemoryUnit {
    /// Unit id.
    pub id: String,
    /// Conversation key this unit belongs to.
    pub conversation_key: String,
    /// Scope the unit belongs to.
    pub scope: Scope,
    /// Memory cell id.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub memcell_id: Option<String>,
    /// Unit type.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub unit_type: Option<String>,
    /// Free-form text.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub text: Option<String>,
    /// Text used for search indexing.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub search_text: Option<String>,
    /// Entities mentioned in the unit.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub entities: Option<Vec<String>>,
    /// Time anchor metadata.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub time_anchor: Option<HashMap<String, String>>,
    /// Source conversation turn ids.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub source_turn_ids: Option<Vec<String>>,
    /// Speaker name.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub speaker: Option<String>,
    /// Topic label.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub topic: Option<String>,
    /// Type label.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub type_label: Option<String>,
    /// Date bucket (YYYY-MM-DD).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub date_bucket: Option<String>,
    /// Free-form metadata.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, String>>,
    /// JSON-encoded metadata.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub metadata_json: Option<String>,
    /// Flattened metadata.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub metadata_flat: Option<String>,
    /// Tombstone marker.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub deleted: Option<bool>,
    /// Creation timestamp.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
    /// Importance score.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub salience: Option<f64>,
    /// Schema version.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub version: Option<i64>,
    /// Id of the unit that superseded this one.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub superseded_by: Option<String>,
    /// Id of the unit this one supersedes.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub supersedes_id: Option<String>,
}

/// One search hit (`unit + score + source`).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchHit {
    /// Memory unit.
    pub unit: MemoryUnit,
    /// Relevance score.
    pub score: f64,
    /// Source channel (e.g. `dense`, `fulltext`).
    pub source: String,
}

/// Memory store descriptor.
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryStoreInfo {
    /// Backing OTS instance.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub ots_instance: Option<String>,
    /// Memory store identifier.
    pub memory_store_name: String,
    /// Human-readable description.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    /// Creation timestamp.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
    /// Last-update timestamp.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub updated_at: Option<String>,
    /// Last-seen-at timestamp.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub last_seen_at: Option<String>,
}

/// Bare item used by [`ListMemoryStoresResponse`].
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryStoreListItem {
    /// Memory store identifier.
    pub memory_store_name: String,
}

/// One session-history message.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SessionMessage {
    /// Optional caller-supplied id.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub message_id: Option<String>,
    /// Speaker role.
    pub role: String,
    /// Speaker name.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub speaker: Option<String>,
    /// Message body.
    pub content: String,
    /// ISO-8601 timestamp.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub timestamp: Option<String>,
    /// Free-form metadata.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, String>>,
}

/// One audit-log request entry.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryStoreRequestRecord {
    /// Server-assigned request id.
    pub request_id: String,
    /// Operation name.
    pub operation: String,
    /// Scope used.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub scope: Option<Scope>,
    /// Target memory id.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub target_id: Option<String>,
    /// HTTP status.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub response_status: Option<u16>,
    /// Latency in milliseconds.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub latency_ms: Option<i64>,
    /// Creation timestamp.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
    /// Summary of the request payload.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub request_summary: Option<String>,
    /// Summary of the response payload.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub response_summary: Option<String>,
    /// Ingest pipeline status.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub ingest_status: Option<String>,
    /// Last error message, if any.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub last_error: Option<String>,
}

/// Session container returned by `ListMemoryStoreMessages`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct MemoryStoreSession {
    /// Scope for the session.
    pub scope: Scope,
    /// Messages in chronological order.
    pub messages: Vec<SessionMessage>,
    /// Continuation token, if more pages remain.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub next_token: Option<String>,
}

/// Response for `CreateMemoryStore`.
pub type CreateMemoryStoreResponse = MemoryStoreInfo;
/// Response for `GetMemoryStore`.
pub type GetMemoryStoreResponse = MemoryStoreInfo;
/// Response for `UpdateMemoryStore`.
pub type UpdateMemoryStoreResponse = MemoryStoreInfo;

/// Response for `ListMemoryStores`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ListMemoryStoresResponse {
    /// Stores returned in this page.
    pub stores: Vec<MemoryStoreListItem>,
    /// Continuation token.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub next_token: Option<String>,
}

/// Response for `DeleteMemoryStore`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DeleteMemoryStoreResponse {
    /// Whether the store was deleted.
    pub deleted: bool,
    /// Store name.
    pub memory_store_name: String,
}

/// Response for `AddMemories`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct AddMemoriesResponse {
    /// Server-assigned request id.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub request_id: Option<String>,
    /// Pipeline status (`OK`, `Pending`, ...).
    pub status: String,
    /// Number of messages accepted into the pipeline.
    pub accepted_messages: u32,
    /// Scope echoed back from the request.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub scope: Option<Scope>,
    /// Store name.
    pub memory_store_name: String,
    /// Number of memcells created.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub memcells_created: Option<u32>,
    /// Number of units created.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub units_created: Option<u32>,
}

/// Response for `SearchMemories`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SearchMemoriesResponse {
    /// Hits ranked by relevance.
    pub results: Vec<SearchHit>,
    /// Scope echoed back.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub scope: Option<Scope>,
    /// Store name.
    pub memory_store_name: String,
}

/// Response for `ListMemories`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ListMemoriesResponse {
    /// Memory units in this page.
    pub memories: Vec<MemoryUnit>,
    /// Scope echoed back.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub scope: Option<Scope>,
    /// Store name.
    pub memory_store_name: String,
    /// Continuation token.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub next_token: Option<String>,
}

/// Response for `GetMemory`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GetMemoryResponse {
    /// The requested memory unit.
    pub memory: MemoryUnit,
    /// Store name.
    pub memory_store_name: String,
}

/// Response for `UpdateMemory`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdateMemoryResponse {
    /// Updated memory unit.
    pub memory: MemoryUnit,
    /// Whether anything changed.
    pub updated: bool,
    /// Store name.
    pub memory_store_name: String,
}

/// Response for `DeleteMemory`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DeleteMemoryResponse {
    /// Whether the unit was deleted.
    pub deleted: bool,
    /// Memory id.
    pub memory_id: String,
    /// Store name.
    pub memory_store_name: String,
}

/// Response for `ListMemoryStoreMessages`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ListMemoryStoreMessagesResponse {
    /// Session container.
    pub session: MemoryStoreSession,
    /// Continuation token (may also live inside `session`).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub next_token: Option<String>,
}

/// Response for `ListMemoryStoreRequests`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ListMemoryStoreRequestsResponse {
    /// Request records returned in this page.
    pub requests: Vec<MemoryStoreRequestRecord>,
    /// Scope echoed back.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub scope: Option<Scope>,
    /// Store name.
    pub memory_store_name: String,
    /// Continuation token.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub next_token: Option<String>,
}

/// Request body for `CreateMemoryStore`.
#[derive(Debug, Default, Clone)]
pub struct CreateMemoryStoreRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Human-readable description.
    pub description: Option<String>,
}

impl JsonRequestConvertible for CreateMemoryStoreRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        assign_if_some(
            &mut map,
            "description",
            self.description.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `GetMemoryStore`.
#[derive(Debug, Clone)]
pub struct GetMemoryStoreRequest {
    /// Store identifier.
    pub memory_store_name: String,
}

impl JsonRequestConvertible for GetMemoryStoreRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `ListMemoryStores`.
#[derive(Debug, Default, Clone)]
pub struct ListMemoryStoresRequest {
    /// Page size.
    pub limit: Option<u32>,
    /// Continuation token.
    pub next_token: Option<String>,
}

impl JsonRequestConvertible for ListMemoryStoresRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        assign_if_some(
            &mut map,
            "limit",
            self.limit.map(|v| Value::Number(v.into())),
        );
        assign_if_some(
            &mut map,
            "nextToken",
            self.next_token.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `UpdateMemoryStore`.
#[derive(Debug, Default, Clone)]
pub struct UpdateMemoryStoreRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// New description.
    pub description: Option<String>,
}

impl JsonRequestConvertible for UpdateMemoryStoreRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        assign_if_some(
            &mut map,
            "description",
            self.description.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `DeleteMemoryStore`.
#[derive(Debug, Clone)]
pub struct DeleteMemoryStoreRequest {
    /// Store identifier.
    pub memory_store_name: String,
}

impl JsonRequestConvertible for DeleteMemoryStoreRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `AddMemories`.
#[derive(Debug, Default, Clone)]
pub struct AddMemoriesRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Scope.
    pub scope: Option<Scope>,
    /// Messages to ingest.
    pub messages: Option<Vec<MessagePayload>>,
    /// Free-form text payload (alternative to `messages`).
    pub text: Option<String>,
    /// Free-form metadata.
    pub metadata: Option<HashMap<String, String>>,
    /// If `true`, wait for ingest to finish before returning.
    pub sync: Option<bool>,
}

impl JsonRequestConvertible for AddMemoriesRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        if let Some(scope) = &self.scope {
            map.insert("scope".to_string(), scope.to_json_request()?);
        }
        if let Some(messages) = &self.messages {
            let items: Result<Vec<Value>, ClientError> =
                messages.iter().map(|m| m.to_json_request()).collect();
            map.insert("messages".to_string(), Value::Array(items?));
        }
        assign_if_some(&mut map, "text", self.text.clone().map(Value::String));
        assign_if_some(
            &mut map,
            "metadata",
            self.metadata.as_ref().map(string_map_to_value),
        );
        assign_if_some(&mut map, "sync", self.sync.map(Value::Bool));
        Ok(Value::Object(map))
    }
}

/// Request body for `SearchMemories`.
#[derive(Debug, Default, Clone)]
pub struct SearchMemoriesRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Scope.
    pub scope: Option<Scope>,
    /// Query text.
    pub query: String,
    /// Free-form metadata filter.
    pub metadata: Option<HashMap<String, String>>,
    /// Page size.
    pub top_k: Option<u32>,
    /// Whether to apply server-side reranking.
    pub enable_rerank: Option<bool>,
}

impl JsonRequestConvertible for SearchMemoriesRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        map.insert("query".to_string(), Value::String(self.query.clone()));
        if let Some(scope) = &self.scope {
            map.insert("scope".to_string(), scope.to_json_request()?);
        }
        assign_if_some(
            &mut map,
            "metadata",
            self.metadata.as_ref().map(string_map_to_value),
        );
        assign_if_some(
            &mut map,
            "topK",
            self.top_k.map(|v| Value::Number(v.into())),
        );
        assign_if_some(
            &mut map,
            "enableRerank",
            self.enable_rerank.map(Value::Bool),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `ListMemories`.
#[derive(Debug, Default, Clone)]
pub struct ListMemoriesRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Scope.
    pub scope: Option<Scope>,
    /// Page size.
    pub limit: Option<u32>,
    /// Continuation token.
    pub next_token: Option<String>,
}

impl JsonRequestConvertible for ListMemoriesRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        if let Some(scope) = &self.scope {
            map.insert("scope".to_string(), scope.to_json_request()?);
        }
        assign_if_some(
            &mut map,
            "limit",
            self.limit.map(|v| Value::Number(v.into())),
        );
        assign_if_some(
            &mut map,
            "nextToken",
            self.next_token.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `GetMemory`.
#[derive(Debug, Clone)]
pub struct GetMemoryRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Memory id.
    pub memory_id: String,
    /// Scope (required for memory addressing).
    pub scope: Scope,
}

impl JsonRequestConvertible for GetMemoryRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        map.insert(
            "memoryId".to_string(),
            Value::String(self.memory_id.clone()),
        );
        map.insert("scope".to_string(), self.scope.to_json_request()?);
        Ok(Value::Object(map))
    }
}

/// Request body for `UpdateMemory`.
#[derive(Debug, Clone)]
pub struct UpdateMemoryRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Memory id.
    pub memory_id: String,
    /// Scope (required).
    pub scope: Scope,
    /// Replacement text.
    pub text: Option<String>,
    /// Replacement metadata.
    pub metadata: Option<HashMap<String, String>>,
}

impl JsonRequestConvertible for UpdateMemoryRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        map.insert(
            "memoryId".to_string(),
            Value::String(self.memory_id.clone()),
        );
        map.insert("scope".to_string(), self.scope.to_json_request()?);
        assign_if_some(&mut map, "text", self.text.clone().map(Value::String));
        assign_if_some(
            &mut map,
            "metadata",
            self.metadata.as_ref().map(string_map_to_value),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `DeleteMemory`.
#[derive(Debug, Clone)]
pub struct DeleteMemoryRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Memory id.
    pub memory_id: String,
    /// Scope (required).
    pub scope: Scope,
}

impl JsonRequestConvertible for DeleteMemoryRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        map.insert(
            "memoryId".to_string(),
            Value::String(self.memory_id.clone()),
        );
        map.insert("scope".to_string(), self.scope.to_json_request()?);
        Ok(Value::Object(map))
    }
}

/// Request body for `ListMemoryStoreMessages`.
#[derive(Debug, Clone)]
pub struct ListMemoryStoreMessagesRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Scope (required).
    pub scope: Scope,
    /// Page size.
    pub limit: Option<u32>,
    /// Continuation token.
    pub next_token: Option<String>,
    /// Inclusive lower bound on `timestamp`.
    pub min_timestamp: Option<String>,
    /// Inclusive upper bound on `timestamp`.
    pub max_timestamp: Option<String>,
}

impl JsonRequestConvertible for ListMemoryStoreMessagesRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        map.insert("scope".to_string(), self.scope.to_json_request()?);
        assign_if_some(
            &mut map,
            "limit",
            self.limit.map(|v| Value::Number(v.into())),
        );
        assign_if_some(
            &mut map,
            "nextToken",
            self.next_token.clone().map(Value::String),
        );
        assign_if_some(
            &mut map,
            "minTimestamp",
            self.min_timestamp.clone().map(Value::String),
        );
        assign_if_some(
            &mut map,
            "maxTimestamp",
            self.max_timestamp.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `ListMemoryStoreRequests`.
#[derive(Debug, Default, Clone)]
pub struct ListMemoryStoreRequestsRequest {
    /// Store identifier.
    pub memory_store_name: String,
    /// Scope.
    pub scope: Option<Scope>,
    /// Page size.
    pub limit: Option<u32>,
    /// Continuation token.
    pub next_token: Option<String>,
    /// Filter to one operation name.
    pub operation: Option<String>,
    /// Inclusive lower bound on `createdAt`.
    pub min_timestamp: Option<String>,
    /// Inclusive upper bound on `createdAt`.
    pub max_timestamp: Option<String>,
}

impl JsonRequestConvertible for ListMemoryStoreRequestsRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "memoryStoreName".to_string(),
            Value::String(self.memory_store_name.clone()),
        );
        if let Some(scope) = &self.scope {
            map.insert("scope".to_string(), scope.to_json_request()?);
        }
        assign_if_some(
            &mut map,
            "limit",
            self.limit.map(|v| Value::Number(v.into())),
        );
        assign_if_some(
            &mut map,
            "nextToken",
            self.next_token.clone().map(Value::String),
        );
        assign_if_some(
            &mut map,
            "operation",
            self.operation.clone().map(Value::String),
        );
        assign_if_some(
            &mut map,
            "minTimestamp",
            self.min_timestamp.clone().map(Value::String),
        );
        assign_if_some(
            &mut map,
            "maxTimestamp",
            self.max_timestamp.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}
