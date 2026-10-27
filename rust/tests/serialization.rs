//! Mirror of `typescript/tests/unit/serialization.test.ts`. Verifies that the
//! `Patch` enum, the metadata filter builder and the nested retrieval models
//! all serialize to the JSON shape that the cross-language test suite expects.

use std::collections::HashMap;

use serde_json::{json, Value};

use tablestore_agent_storage::models::{
    MetadataField, MetadataFieldType, MetadataFilter, Patch, PrimitiveValue,
    RetrievalConfiguration, RetrievalQuery, RetrieveRequest, SearchType, UpdateChunkItem,
    UpdateChunksRequest, UpdateDocumentRequest, UpdateKnowledgeBaseRequest,
};
use tablestore_agent_storage::to_json_request;

#[test]
fn patch_unchanged_omits_field_but_clear_emits_null() {
    let payload = to_json_request(&UpdateKnowledgeBaseRequest {
        knowledge_base_name: "kb".to_string(),
        description: Patch::Clear,
        tags: Patch::Unchanged,
        retrieval_configuration: None,
    })
    .unwrap();

    assert_eq!(
        payload,
        json!({
            "knowledgeBaseName": "kb",
            "description": null,
        })
    );
}

#[test]
fn metadata_filter_builder_round_trips_through_retrieve() {
    let filter = MetadataFilter::builder()
        .equals("author", PrimitiveValue::from("aliyun"))
        .greater_than("rank", PrimitiveValue::from(1_i64))
        .build_and()
        .unwrap();

    let payload = to_json_request(&RetrieveRequest {
        knowledge_base_name: "kb".to_string(),
        retrieval_query: RetrievalQuery::text("hello"),
        subspace: None,
        retrieval_configuration: Some(RetrievalConfiguration {
            search_type: Some(vec![SearchType::DenseVector, SearchType::FullText]),
            filter: Some(filter),
            ..Default::default()
        }),
    })
    .unwrap();

    assert_eq!(
        payload,
        json!({
            "knowledgeBaseName": "kb",
            "retrievalQuery": { "text": "hello", "type": "TEXT" },
            "retrievalConfiguration": {
                "searchType": ["DENSE_VECTOR", "FULL_TEXT"],
                "filter": {
                    "andAll": [
                        { "equals": { "key": "author", "value": "aliyun" } },
                        { "greaterThan": { "key": "rank", "value": 1 } },
                    ],
                },
            },
        })
    );
}

#[test]
fn document_and_chunk_patches_match_typescript_shape() {
    let document_payload = to_json_request(&UpdateDocumentRequest {
        knowledge_base_name: "kb".to_string(),
        subspace: None,
        oss_key: None,
        doc_id: Some("doc-1".to_string()),
        metadata: Patch::Clear,
    })
    .unwrap();

    let chunk_payload = to_json_request(&UpdateChunksRequest {
        knowledge_base_name: "kb".to_string(),
        chunks: vec![UpdateChunkItem {
            chunk_id: 1,
            doc_id: Some("doc-1".to_string()),
            oss_key: None,
            title: Patch::Unchanged,
            content: Patch::Clear,
            status: None,
        }],
        subspace: None,
    })
    .unwrap();

    assert_eq!(
        document_payload,
        json!({
            "knowledgeBaseName": "kb",
            "docId": "doc-1",
            "metadata": null,
        })
    );
    assert_eq!(
        chunk_payload,
        json!({
            "knowledgeBaseName": "kb",
            "chunks": [
                {
                    "chunkId": 1,
                    "docId": "doc-1",
                    "content": null,
                }
            ],
        })
    );
}

#[test]
fn metadata_field_renders_string_type() {
    let field = MetadataField::new("author", MetadataFieldType::String);
    let payload = to_json_request(&field).unwrap();
    assert_eq!(payload, json!({ "name": "author", "type": "string" }));
}

#[test]
fn patch_set_value_passes_through() {
    let mut metadata = HashMap::new();
    metadata.insert("source".to_string(), Value::String("unit".to_string()));

    let payload = to_json_request(&UpdateDocumentRequest {
        knowledge_base_name: "kb".to_string(),
        subspace: None,
        oss_key: None,
        doc_id: Some("doc-1".to_string()),
        metadata: Patch::Set(metadata),
    })
    .unwrap();

    assert_eq!(
        payload,
        json!({
            "knowledgeBaseName": "kb",
            "docId": "doc-1",
            "metadata": { "source": "unit" },
        })
    );
}

#[test]
fn metadata_filter_builder_rejects_empty() {
    let err = MetadataFilter::builder().build_and().unwrap_err();
    let msg = format!("{err}");
    assert!(msg.contains("no conditions"), "unexpected message: {msg}");
}

#[test]
fn embedding_configuration_debug_redacts_api_key() {
    use tablestore_agent_storage::models::EmbeddingConfiguration;

    let config = EmbeddingConfiguration {
        provider: Some("dashscope".to_string()),
        url: Some("https://example.com".to_string()),
        api_key: Some("super-secret-key".to_string()),
        model: Some("text-embedding-v2".to_string()),
        dimension: Some(1024),
    };
    let rendered = format!("{config:?}");
    assert!(
        rendered.contains("dashscope"),
        "provider should be visible: {rendered}"
    );
    assert!(
        !rendered.contains("super-secret-key"),
        "api_key leaked: {rendered}"
    );
    assert!(
        rendered.contains("[redacted]"),
        "missing redaction marker: {rendered}"
    );
}
