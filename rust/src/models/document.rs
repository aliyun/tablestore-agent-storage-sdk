use std::collections::HashMap;

use serde_json::{Map, Value};

use crate::error::ClientError;

use super::base::{assign_if_some, JsonRequestConvertible};
use super::patch::Patch;

fn metadata_to_value(metadata: &Option<HashMap<String, Value>>) -> Option<Value> {
    metadata.as_ref().map(|m| {
        let mut map = Map::new();
        for (k, v) in m {
            map.insert(k.clone(), v.clone());
        }
        Value::Object(map)
    })
}

/// One entry of an `AddDocuments` request — references an existing OSS object.
#[derive(Debug, Default, Clone)]
pub struct AddDocumentItem {
    /// OSS object key (e.g. `oss://bucket/path/to/doc.md`).
    pub oss_key: String,
    /// Free-form metadata attached to the document.
    pub metadata: Option<HashMap<String, Value>>,
    /// Optional inclusion filters scoped to this document.
    pub inclusion_filters: Option<Vec<String>>,
    /// Optional exclusion filters scoped to this document.
    pub exclusion_filters: Option<Vec<String>>,
}

impl AddDocumentItem {
    /// Build with only the required field.
    pub fn new(oss_key: impl Into<String>) -> Self {
        Self {
            oss_key: oss_key.into(),
            ..Self::default()
        }
    }
}

impl JsonRequestConvertible for AddDocumentItem {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert("ossKey".to_string(), Value::String(self.oss_key.clone()));
        assign_if_some(&mut map, "metadata", metadata_to_value(&self.metadata));
        assign_if_some(
            &mut map,
            "inclusionFilters",
            self.inclusion_filters
                .as_ref()
                .map(|v| Value::Array(v.iter().cloned().map(Value::String).collect())),
        );
        assign_if_some(
            &mut map,
            "exclusionFilters",
            self.exclusion_filters
                .as_ref()
                .map(|v| Value::Array(v.iter().cloned().map(Value::String).collect())),
        );
        Ok(Value::Object(map))
    }
}

/// One entry of an `UploadDocuments` request — references a local file to be
/// uploaded to OSS first.
#[derive(Debug, Default, Clone)]
pub struct UploadDocumentItem {
    /// Absolute or relative path on the local filesystem.
    pub file_path: String,
    /// Free-form metadata attached to the document.
    pub metadata: Option<HashMap<String, Value>>,
}

impl UploadDocumentItem {
    /// Build with only the required field.
    pub fn new(file_path: impl Into<String>) -> Self {
        Self {
            file_path: file_path.into(),
            metadata: None,
        }
    }
}

impl JsonRequestConvertible for UploadDocumentItem {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "filePath".to_string(),
            Value::String(self.file_path.clone()),
        );
        assign_if_some(&mut map, "metadata", metadata_to_value(&self.metadata));
        Ok(Value::Object(map))
    }
}

/// One entry of a `DeleteDocuments` request.
#[derive(Debug, Default, Clone)]
pub struct DeleteDocumentItem {
    /// Server-assigned document id.
    pub doc_id: Option<String>,
    /// OSS object key.
    pub oss_key: Option<String>,
}

impl JsonRequestConvertible for DeleteDocumentItem {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        assign_if_some(&mut map, "docId", self.doc_id.clone().map(Value::String));
        assign_if_some(&mut map, "ossKey", self.oss_key.clone().map(Value::String));
        Ok(Value::Object(map))
    }
}

/// Request body for `AddDocuments`.
#[derive(Debug, Default, Clone)]
pub struct AddDocumentsRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Documents to add (each referencing an OSS object).
    pub documents: Vec<AddDocumentItem>,
    /// Optional subspace.
    pub subspace: Option<String>,
}

impl JsonRequestConvertible for AddDocumentsRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        let docs: Result<Vec<Value>, ClientError> =
            self.documents.iter().map(|d| d.to_json_request()).collect();
        map.insert("documents".to_string(), Value::Array(docs?));
        assign_if_some(
            &mut map,
            "subspace",
            self.subspace.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `UploadDocuments` — only the OSS-extension client (when
/// the `oss-upload` feature is enabled) understands this.
#[derive(Debug, Default, Clone)]
pub struct UploadDocumentsRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Local files to upload.
    pub documents: Vec<UploadDocumentItem>,
    /// Optional subspace.
    pub subspace: Option<String>,
}

impl JsonRequestConvertible for UploadDocumentsRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        let docs: Result<Vec<Value>, ClientError> =
            self.documents.iter().map(|d| d.to_json_request()).collect();
        map.insert("documents".to_string(), Value::Array(docs?));
        assign_if_some(
            &mut map,
            "subspace",
            self.subspace.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `GetDocument`.
#[derive(Debug, Default, Clone)]
pub struct GetDocumentRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Server-assigned document id.
    pub doc_id: Option<String>,
    /// OSS object key.
    pub oss_key: Option<String>,
    /// Optional subspace.
    pub subspace: Option<String>,
}

impl JsonRequestConvertible for GetDocumentRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        assign_if_some(&mut map, "docId", self.doc_id.clone().map(Value::String));
        assign_if_some(&mut map, "ossKey", self.oss_key.clone().map(Value::String));
        assign_if_some(
            &mut map,
            "subspace",
            self.subspace.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `ListDocuments`.
#[derive(Debug, Default, Clone)]
pub struct ListDocumentsRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Optional list of subspaces to scope the listing to.
    pub subspace: Option<Vec<String>>,
    /// Page size.
    pub max_results: Option<u32>,
    /// Continuation token.
    pub next_token: Option<String>,
}

impl JsonRequestConvertible for ListDocumentsRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        assign_if_some(
            &mut map,
            "subspace",
            self.subspace
                .as_ref()
                .map(|v| Value::Array(v.iter().cloned().map(Value::String).collect())),
        );
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

/// Request body for `DeleteDocuments`.
#[derive(Debug, Default, Clone)]
pub struct DeleteDocumentsRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Documents to delete.
    pub documents: Vec<DeleteDocumentItem>,
    /// Optional subspace.
    pub subspace: Option<String>,
}

impl JsonRequestConvertible for DeleteDocumentsRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        let docs: Result<Vec<Value>, ClientError> =
            self.documents.iter().map(|d| d.to_json_request()).collect();
        map.insert("documents".to_string(), Value::Array(docs?));
        assign_if_some(
            &mut map,
            "subspace",
            self.subspace.clone().map(Value::String),
        );
        Ok(Value::Object(map))
    }
}

/// Request body for `UpdateDocument`.
#[derive(Debug, Default, Clone)]
pub struct UpdateDocumentRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Optional subspace.
    pub subspace: Option<String>,
    /// OSS object key.
    pub oss_key: Option<String>,
    /// Server-assigned document id.
    pub doc_id: Option<String>,
    /// Metadata patch (`Unchanged` / `Set(_)` / `Clear`).
    pub metadata: Patch<HashMap<String, Value>>,
}

impl JsonRequestConvertible for UpdateDocumentRequest {
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
        assign_if_some(&mut map, "ossKey", self.oss_key.clone().map(Value::String));
        assign_if_some(&mut map, "docId", self.doc_id.clone().map(Value::String));
        match &self.metadata {
            Patch::Unchanged => {}
            Patch::Set(value) => {
                let mut inner = Map::new();
                for (k, v) in value {
                    inner.insert(k.clone(), v.clone());
                }
                map.insert("metadata".to_string(), Value::Object(inner));
            }
            Patch::Clear => {
                map.insert("metadata".to_string(), Value::Null);
            }
        }
        Ok(Value::Object(map))
    }
}
