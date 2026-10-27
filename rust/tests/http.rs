//! End-to-end HTTP flow against a `wiremock` server. Verifies the signed
//! request layout we emit and the response validation we run.

use std::collections::BTreeMap;

use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use hmac::{Hmac, Mac};
use serde_json::{json, Value};
use sha1::Sha1;
use wiremock::matchers::{header, method, path};
use wiremock::{Mock, MockServer, ResponseTemplate};

use tablestore_agent_storage::http::md5_base64;
use tablestore_agent_storage::{AgentStorageClient, Error};

/// Build a 2xx response body that the client will accept as authentic.
fn sign_response(action: &str, body: &[u8], secret: &str, ak: &str) -> ResponseTemplate {
    let mut headers = BTreeMap::new();
    headers.insert("x-ots-contentmd5".to_string(), md5_base64(body));
    headers.insert("x-ots-requestid".to_string(), "req-1".to_string());
    headers.insert(
        "x-ots-date".to_string(),
        "2025-04-10T01:02:03.000Z".to_string(),
    );
    headers.insert(
        "x-ots-contenttype".to_string(),
        "application/json".to_string(),
    );

    let canonical = format!("{}\n/{}", make_headers_string(&headers), action);
    let mut mac = Hmac::<Sha1>::new_from_slice(secret.as_bytes()).unwrap();
    mac.update(canonical.as_bytes());
    let signature = BASE64.encode(mac.finalize().into_bytes());

    let mut tpl = ResponseTemplate::new(200).set_body_bytes(body);
    for (k, v) in &headers {
        tpl = tpl.append_header(k.as_str(), v.as_str());
    }
    tpl = tpl.append_header("authorization", format!("OTS {ak}:{signature}").as_str());
    tpl
}

fn make_headers_string(headers: &BTreeMap<String, String>) -> String {
    let mut parts: Vec<String> = headers
        .iter()
        .filter(|(k, _)| k.to_ascii_lowercase().starts_with("x-ots-"))
        .map(|(k, v)| format!("{}:{}", k.to_ascii_lowercase(), v.trim()))
        .collect();
    parts.sort();
    parts.join("\n")
}

#[tokio::test]
async fn signs_request_and_validates_response() {
    let server = MockServer::start().await;
    let payload = serde_json::to_vec(&json!({ "code": "OK", "data": {} })).unwrap();

    Mock::given(method("POST"))
        .and(path("/ListKnowledgeBase"))
        .and(header("x-ots-apiversion", "2015-12-31"))
        .and(header("x-ots-instancename", "instance"))
        .and(header("x-ots-accesskeyid", "ak"))
        .respond_with(sign_response("ListKnowledgeBase", &payload, "sk", "ak"))
        .expect(1)
        .mount(&server)
        .await;

    let client = AgentStorageClient::builder()
        .endpoint(server.uri())
        .instance_name("instance")
        .access_key_id("ak")
        .access_key_secret("sk")
        .build()
        .unwrap();

    let value: Value = client.list_knowledge_base(json!({})).await.unwrap();
    assert_eq!(value.get("code").and_then(Value::as_str), Some("OK"));
}

#[tokio::test]
async fn maps_server_error_into_service_error() {
    let server = MockServer::start().await;
    let payload =
        serde_json::to_vec(&json!({ "code": "OTSAuthFailed", "message": "bad key" })).unwrap();

    Mock::given(method("POST"))
        .and(path("/CreateKnowledgeBase"))
        .respond_with(
            ResponseTemplate::new(403)
                .set_body_bytes(payload)
                .append_header("x-ots-requestid", "req-err"),
        )
        .expect(1)
        .mount(&server)
        .await;

    let client = AgentStorageClient::builder()
        .endpoint(server.uri())
        .instance_name("instance")
        .access_key_id("ak")
        .access_key_secret("sk")
        .build()
        .unwrap();

    let err = client
        .create_knowledge_base(json!({ "knowledgeBaseName": "kb" }))
        .await
        .expect_err("403 surfaces as ServiceError");

    match err {
        Error::Service(service) => {
            assert_eq!(service.status_code, 403);
            assert_eq!(service.request_id.as_deref(), Some("req-err"));
            assert_eq!(service.code.as_deref(), Some("OTSAuthFailed"));
            assert_eq!(service.message, "bad key");
        }
        other => panic!("expected service error, got {other:?}"),
    }
}

#[tokio::test]
async fn forwards_extra_headers_on_every_request() {
    let server = MockServer::start().await;
    let payload = serde_json::to_vec(&json!({ "code": "OK", "data": {} })).unwrap();

    Mock::given(method("POST"))
        .and(path("/ListKnowledgeBase"))
        .and(header("x-ots-trace-id", "trace-xyz"))
        .and(header("x-app-tag", "rust-sdk"))
        .respond_with(sign_response("ListKnowledgeBase", &payload, "sk", "ak"))
        .expect(1)
        .mount(&server)
        .await;

    let client = AgentStorageClient::builder()
        .endpoint(server.uri())
        .instance_name("instance")
        .access_key_id("ak")
        .access_key_secret("sk")
        .extra_header("x-ots-trace-id", "trace-xyz")
        .extra_header("x-app-tag", "rust-sdk")
        .build()
        .unwrap();

    let _: Value = client.list_knowledge_base(json!({})).await.unwrap();
}

#[tokio::test]
async fn rejects_response_with_bad_signature() {
    let server = MockServer::start().await;
    let body = serde_json::to_vec(&json!({ "code": "OK" })).unwrap();

    Mock::given(method("POST"))
        .and(path("/ListKnowledgeBase"))
        .respond_with(
            ResponseTemplate::new(200)
                .set_body_bytes(body.clone())
                .append_header("x-ots-contentmd5", md5_base64(&body).as_str())
                .append_header("x-ots-requestid", "req-1")
                .append_header("x-ots-date", "2025-04-10T01:02:03.000Z")
                .append_header("x-ots-contenttype", "application/json")
                .append_header("authorization", "OTS ak:not-a-real-signature"),
        )
        .expect(1)
        .mount(&server)
        .await;

    let client = AgentStorageClient::builder()
        .endpoint(server.uri())
        .instance_name("instance")
        .access_key_id("ak")
        .access_key_secret("sk")
        .build()
        .unwrap();

    let err = client
        .list_knowledge_base(json!({}))
        .await
        .expect_err("tampered signature");
    let msg = format!("{err}");
    assert!(
        msg.contains("invalid signature"),
        "expected signature error, got: {msg}"
    );
}
