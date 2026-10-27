//! Cross-SDK signature fixtures.
//!
//! Mirrors `typescript/tests/unit/auth.test.ts` byte-for-byte. The expected
//! signatures originated from the Python SDK and are shared by every language
//! binding — if a number here changes, the SDK has diverged from the rest of
//! the family.

use std::collections::BTreeMap;

use tablestore_agent_storage::auth::{Credentials, RequestContext, SignV2, SignV4, Signer};

const AK_ID: &str = "test_id";
const AK_SECRET: &str = "test_key";
const STS_TOKEN: &str = "test_token";
const REGION: &str = "test-region";
const SIGN_DATE: &str = "20250410";
const QUERY: &str = "test_query";

fn fixture_headers() -> BTreeMap<String, String> {
    let mut headers = BTreeMap::new();
    headers.insert("x-ots-test".to_string(), "test".to_string());
    headers
}

fn fixture_credentials() -> Credentials {
    Credentials {
        access_key_id: AK_ID.to_string(),
        access_key_secret: AK_SECRET.to_string(),
        security_token: Some(STS_TOKEN.to_string()),
    }
}

#[test]
fn v2_matches_python_fixtures() {
    let mut headers = fixture_headers();
    let context = RequestContext::with_sign_date(fixture_credentials(), SIGN_DATE);
    let signer = SignV2::new();

    signer
        .sign_request(QUERY, &mut headers, &context)
        .expect("sign succeeds");
    assert_eq!(
        headers.get("x-ots-signature").map(String::as_str),
        Some("QDhzLv7VESBJtYQY4Li0IhSUOdg=")
    );

    let response_headers = fixture_headers();
    let key = signer.signing_key(&context).unwrap();
    let response_sig = signer.response_signature(QUERY, &response_headers, &key);
    assert_eq!(response_sig, "UjJK/SWed0n9o6JYxvApHGaQABo=");
}

#[test]
fn v4_matches_python_fixtures() {
    let mut headers = fixture_headers();
    let context = RequestContext::with_sign_date(fixture_credentials(), SIGN_DATE);
    let signer = SignV4::with_sign_date(REGION, SIGN_DATE).unwrap();

    signer
        .sign_request(QUERY, &mut headers, &context)
        .expect("sign succeeds");

    assert_eq!(
        headers.get("x-ots-signregion").map(String::as_str),
        Some(REGION)
    );
    assert_eq!(
        headers.get("x-ots-signdate").map(String::as_str),
        Some(SIGN_DATE)
    );
    assert_eq!(
        headers.get("x-ots-signaturev4").map(String::as_str),
        Some("yXnOpODWaU1EYAlLP3l25ksj010uGHS7uxIt5Qiwz4o=")
    );

    let key = signer.signing_key(&context).unwrap();
    assert_eq!(
        std::str::from_utf8(key.as_bytes()).unwrap(),
        "nToxlXrxgCm0L5J0nr/qq/GmtgN9GVBhiRLzdLaVUP0="
    );

    let response_headers = fixture_headers();
    let response_sig = signer.response_signature(QUERY, &response_headers, &key);
    assert_eq!(response_sig, "vIhaUGwv/JSg8ctLNyxbNeNv69A=");
}

#[test]
fn v4_requires_region() {
    let err = SignV4::with_sign_date("", SIGN_DATE).expect_err("empty region is rejected");
    let msg = format!("{err}");
    assert!(msg.contains("region"), "unexpected message: {msg}");
}

#[test]
fn credentials_debug_redacts_secret_and_token() {
    let creds = fixture_credentials();
    let rendered = format!("{creds:?}");
    assert!(
        rendered.contains(AK_ID),
        "AK id should remain visible: {rendered}"
    );
    assert!(
        !rendered.contains(AK_SECRET),
        "AK secret leaked: {rendered}"
    );
    assert!(
        !rendered.contains(STS_TOKEN),
        "STS token leaked: {rendered}"
    );
    assert!(
        rendered.contains("[redacted]"),
        "missing redaction marker: {rendered}"
    );
}
