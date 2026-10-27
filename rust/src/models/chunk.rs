use serde_json::{Map, Value};

use crate::error::ClientError;

use super::base::{assign_if_some, JsonRequestConvertible};
use super::patch::Patch;

/// One entry of an `UpdateChunks` request.
#[derive(Debug, Clone)]
pub struct UpdateChunkItem {
    /// Chunk id (within its parent document).
    pub chunk_id: i64,
    /// Server-assigned document id (one of `doc_id`/`oss_key` is required).
    pub doc_id: Option<String>,
    /// OSS object key (one of `doc_id`/`oss_key` is required).
    pub oss_key: Option<String>,
    /// Title patch.
    pub title: Patch<String>,
    /// Content patch.
    pub content: Patch<String>,
    /// Status string.
    pub status: Option<String>,
}

impl UpdateChunkItem {
    /// Build with only the required field.
    pub fn new(chunk_id: i64) -> Self {
        Self {
            chunk_id,
            doc_id: None,
            oss_key: None,
            title: Patch::Unchanged,
            content: Patch::Unchanged,
            status: None,
        }
    }
}

impl JsonRequestConvertible for UpdateChunkItem {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert("chunkId".to_string(), Value::Number(self.chunk_id.into()));
        assign_if_some(&mut map, "docId", self.doc_id.clone().map(Value::String));
        assign_if_some(&mut map, "ossKey", self.oss_key.clone().map(Value::String));
        write_patch_string(&mut map, "title", &self.title);
        write_patch_string(&mut map, "content", &self.content);
        assign_if_some(&mut map, "status", self.status.clone().map(Value::String));
        Ok(Value::Object(map))
    }
}

/// Request body for `UpdateChunks`.
#[derive(Debug, Default, Clone)]
pub struct UpdateChunksRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Chunks to update.
    pub chunks: Vec<UpdateChunkItem>,
    /// Optional subspace.
    pub subspace: Option<String>,
}

impl JsonRequestConvertible for UpdateChunksRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        let chunks: Result<Vec<Value>, ClientError> =
            self.chunks.iter().map(|c| c.to_json_request()).collect();
        map.insert("chunks".to_string(), Value::Array(chunks?));
        assign_if_some(
            &mut map,
            "subspace",
            self.subspace.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `ListChunks`.
#[derive(Debug, Default, Clone)]
pub struct ListChunksRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Optional subspace.
    pub subspace: Option<String>,
    /// Server-assigned document id.
    pub doc_id: Option<String>,
    /// OSS object key.
    pub oss_key: Option<String>,
    /// Page size.
    pub max_results: Option<u32>,
    /// Continuation token.
    pub next_token: Option<String>,
}

impl JsonRequestConvertible for ListChunksRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        assign_if_some(
            &mut map,
            "subspace",
            self.subspace.clone().map(Value::String),
        );
        assign_if_some(&mut map, "docId", self.doc_id.clone().map(Value::String));
        assign_if_some(&mut map, "ossKey", self.oss_key.clone().map(Value::String));
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

fn write_patch_string(map: &mut Map<String, Value>, key: &str, patch: &Patch<String>) {
    match patch {
        Patch::Unchanged => {}
        Patch::Set(value) => {
            map.insert(key.to_string(), Value::String(value.clone()));
        }
        Patch::Clear => {
            map.insert(key.to_string(), Value::Null);
        }
    }
}
