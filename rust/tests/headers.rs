//! Mirror of `typescript/tests/unit/headers.test.ts`.

use std::collections::BTreeMap;

use chrono::{TimeZone, Utc};

use tablestore_agent_storage::auth::{Credentials, RequestContext, SignV2};
use tablestore_agent_storage::http::{
    build_date_header, build_request_headers, md5_base64, BuildHeadersInput,
};

#[test]
fn formats_date_with_fixed_milliseconds() {
    let date =
        Utc.with_ymd_and_hms(2025, 4, 10, 1, 2, 3).unwrap() + chrono::Duration::milliseconds(456);
    assert_eq!(build_date_header(date), "2025-04-10T01:02:03.000Z");
}

#[test]
fn builds_signed_request_headers() {
    let date = Utc.with_ymd_and_hms(2025, 4, 10, 1, 2, 3).unwrap();
    let credentials = Credentials::new("ak", "sk");
    let context = RequestContext::new(credentials.clone());
    let signer = SignV2::new();

    let headers = build_request_headers(BuildHeadersInput {
        body: b"hello",
        query: "/CreateKnowledgeBase",
        instance_name: "instance",
        credentials: &credentials,
        signer: &signer,
        request_context: &context,
        user_agent: "test-agent",
        date: Some(date),
        extra_headers: None,
    })
    .expect("headers build");

    assert_eq!(
        headers.get("x-ots-apiversion").map(String::as_str),
        Some("2015-12-31")
    );
    assert_eq!(
        headers.get("x-ots-instancename").map(String::as_str),
        Some("instance")
    );
    assert_eq!(
        headers.get("User-Agent").map(String::as_str),
        Some("test-agent")
    );
    assert_eq!(
        headers.get("x-ots-contentmd5").map(String::as_str),
        Some("XUFAKrxLKna5cZ2REBfFkg==")
    );
    assert!(headers.contains_key("x-ots-signature"));
    assert!(!headers.contains_key("x-ots-ststoken"));
}

#[test]
fn sts_token_emits_header() {
    let credentials = Credentials {
        access_key_id: "ak".to_string(),
        access_key_secret: "sk".to_string(),
        security_token: Some("session".to_string()),
    };
    let context = RequestContext::new(credentials.clone());
    let signer = SignV2::new();

    let headers = build_request_headers(BuildHeadersInput {
        body: b"{}",
        query: "/Op",
        instance_name: "instance",
        credentials: &credentials,
        signer: &signer,
        request_context: &context,
        user_agent: "ua",
        date: None,
        extra_headers: None,
    })
    .unwrap();

    assert_eq!(
        headers.get("x-ots-ststoken").map(String::as_str),
        Some("session")
    );
}

#[test]
fn extra_headers_ride_along_and_x_ots_extras_get_signed() {
    let date = Utc.with_ymd_and_hms(2025, 4, 10, 1, 2, 3).unwrap();
    let credentials = Credentials::new("ak", "sk");
    let context = RequestContext::new(credentials.clone());
    let signer = SignV2::new();

    let mut extras = BTreeMap::new();
    extras.insert("x-ots-trace-id".to_string(), "trace-abc".to_string());
    extras.insert("x-app-tag".to_string(), "demo".to_string());

    let headers = build_request_headers(BuildHeadersInput {
        body: b"hello",
        query: "/CreateKnowledgeBase",
        instance_name: "instance",
        credentials: &credentials,
        signer: &signer,
        request_context: &context,
        user_agent: "test-agent",
        date: Some(date),
        extra_headers: Some(&extras),
    })
    .expect("headers build");

    assert_eq!(
        headers.get("x-ots-trace-id").map(String::as_str),
        Some("trace-abc")
    );
    assert_eq!(headers.get("x-app-tag").map(String::as_str), Some("demo"));

    let without_extras = build_request_headers(BuildHeadersInput {
        body: b"hello",
        query: "/CreateKnowledgeBase",
        instance_name: "instance",
        credentials: &credentials,
        signer: &signer,
        request_context: &context,
        user_agent: "test-agent",
        date: Some(date),
        extra_headers: None,
    })
    .expect("headers build (no extras)");

    assert_ne!(
        headers.get("x-ots-signature"),
        without_extras.get("x-ots-signature"),
        "x-ots-* extras must alter the signature"
    );
}

#[test]
fn md5_helper_matches_fixture() {
    assert_eq!(md5_base64(b"hello"), "XUFAKrxLKna5cZ2REBfFkg==");
}
