//! Mirror of `typescript/tests/unit/memory-serialization.test.ts`.

use std::collections::HashMap;

use serde_json::json;

use tablestore_agent_storage::models::{
    AddMemoriesRequest, CreateMemoryStoreRequest, ListMemoryStoreMessagesRequest,
    ListMemoryStoreRequestsRequest, MessagePayload, Scope, SearchMemoriesRequest,
};
use tablestore_agent_storage::to_json_request;

#[test]
fn add_memories_with_text_only() {
    let payload = to_json_request(&AddMemoriesRequest {
        memory_store_name: "store_a".to_string(),
        scope: Some(Scope {
            app_id: Some("app_a".to_string()),
            ..Default::default()
        }),
        text: Some("Alice works at Google.".to_string()),
        ..Default::default()
    })
    .unwrap();

    assert_eq!(
        payload,
        json!({
            "memoryStoreName": "store_a",
            "scope": { "appId": "app_a" },
            "text": "Alice works at Google.",
        })
    );
}

#[test]
fn add_memories_with_messages_and_sync_flag() {
    let mut metadata = HashMap::new();
    metadata.insert("topic".to_string(), "coffee".to_string());

    let payload = to_json_request(&AddMemoriesRequest {
        memory_store_name: "store_a".to_string(),
        scope: Some(Scope {
            app_id: Some("app_a".to_string()),
            tenant_id: Some("__default__".to_string()),
            ..Default::default()
        }),
        messages: Some(vec![MessagePayload {
            role: "user".to_string(),
            content: "Ian enjoys espresso.".to_string(),
            message_id: Some("msg-1".to_string()),
            timestamp: None,
            metadata: Some(metadata),
        }]),
        sync: Some(true),
        ..Default::default()
    })
    .unwrap();

    assert_eq!(
        payload,
        json!({
            "memoryStoreName": "store_a",
            "scope": { "appId": "app_a", "tenantId": "__default__" },
            "messages": [
                {
                    "role": "user",
                    "content": "Ian enjoys espresso.",
                    "messageId": "msg-1",
                    "metadata": { "topic": "coffee" },
                }
            ],
            "sync": true,
        })
    );
}

#[test]
fn search_and_message_list_requests() {
    let mut filter_metadata = HashMap::new();
    filter_metadata.insert("topic".to_string(), "coffee".to_string());

    let search_payload = to_json_request(&SearchMemoriesRequest {
        memory_store_name: "store_a".to_string(),
        scope: Some(Scope {
            app_id: Some("app_a".to_string()),
            tenant_id: Some("__default__".to_string()),
            ..Default::default()
        }),
        query: "Ian espresso".to_string(),
        metadata: Some(filter_metadata),
        top_k: Some(5),
        enable_rerank: Some(false),
    })
    .unwrap();

    let messages_payload = to_json_request(&ListMemoryStoreMessagesRequest {
        memory_store_name: "store_a".to_string(),
        scope: Scope {
            app_id: Some("app_a".to_string()),
            tenant_id: Some("tenant_a".to_string()),
            agent_id: Some("agent_a".to_string()),
            run_id: Some("run_a".to_string()),
        },
        min_timestamp: Some("2024-01-01T00:00:00Z".to_string()),
        max_timestamp: Some("2024-01-02T00:00:00Z".to_string()),
        limit: Some(10),
        next_token: None,
    })
    .unwrap();

    assert_eq!(
        search_payload,
        json!({
            "memoryStoreName": "store_a",
            "scope": { "appId": "app_a", "tenantId": "__default__" },
            "query": "Ian espresso",
            "metadata": { "topic": "coffee" },
            "topK": 5,
            "enableRerank": false,
        })
    );
    assert_eq!(
        messages_payload,
        json!({
            "memoryStoreName": "store_a",
            "scope": {
                "appId": "app_a",
                "tenantId": "tenant_a",
                "agentId": "agent_a",
                "runId": "run_a",
            },
            "minTimestamp": "2024-01-01T00:00:00Z",
            "maxTimestamp": "2024-01-02T00:00:00Z",
            "limit": 10,
        })
    );
}

#[test]
fn list_requests_with_operation_and_timestamps() {
    let payload = to_json_request(&ListMemoryStoreRequestsRequest {
        memory_store_name: "store_a".to_string(),
        scope: Some(Scope {
            app_id: Some("app_a".to_string()),
            tenant_id: Some("*".to_string()),
            ..Default::default()
        }),
        operation: Some("AddMemories".to_string()),
        limit: Some(10),
        next_token: Some("token-1".to_string()),
        min_timestamp: Some("2024-01-01T00:00:00Z".to_string()),
        max_timestamp: Some("2024-01-02T00:00:00Z".to_string()),
    })
    .unwrap();

    assert_eq!(
        payload,
        json!({
            "memoryStoreName": "store_a",
            "scope": { "appId": "app_a", "tenantId": "*" },
            "operation": "AddMemories",
            "limit": 10,
            "nextToken": "token-1",
            "minTimestamp": "2024-01-01T00:00:00Z",
            "maxTimestamp": "2024-01-02T00:00:00Z",
        })
    );
}

#[test]
fn create_memory_store_with_description() {
    let payload = to_json_request(&CreateMemoryStoreRequest {
        memory_store_name: "store_a".to_string(),
        description: Some("store".to_string()),
    })
    .unwrap();

    assert_eq!(
        payload,
        json!({
            "memoryStoreName": "store_a",
            "description": "store",
        })
    );
}
