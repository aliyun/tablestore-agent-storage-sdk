//! OSS key/URI helpers — mirrors the path utility tests baked into the TS
//! Node client.

#![cfg(feature = "oss-upload")]

use tablestore_agent_storage::oss::{
    build_oss_full_path, build_oss_object_key, parse_oss_path, DEFAULT_SUBSPACE,
};

#[test]
fn builds_object_key_with_default_subspace() {
    let key = build_oss_object_key("inst", "kb", None, "/tmp/example.md");
    assert_eq!(
        key,
        format!("inst/kb/source_doc/{DEFAULT_SUBSPACE}/tmp/example.md")
    );
}

#[test]
fn builds_object_key_with_custom_subspace() {
    let key = build_oss_object_key("inst", "kb", Some("custom"), "/data/x.md");
    assert_eq!(key, "inst/kb/source_doc/custom/data/x.md");
}

#[test]
fn parses_full_oss_uri() {
    let parsed = parse_oss_path("oss://bucket/key/path.md", None).unwrap();
    assert_eq!(parsed.bucket_name, "bucket");
    assert_eq!(parsed.object_key, "key/path.md");
}

#[test]
fn wraps_bare_key_with_default_bucket() {
    let parsed = parse_oss_path("raw/key.md", Some("bucket")).unwrap();
    assert_eq!(parsed.bucket_name, "bucket");
    assert_eq!(parsed.object_key, "raw/key.md");
}

#[test]
fn full_path_round_trips() {
    let full = build_oss_full_path("bucket", "key/path.md");
    assert_eq!(full, "oss://bucket/key/path.md");
    let parsed = parse_oss_path(&full, None).unwrap();
    assert_eq!(parsed.bucket_name, "bucket");
    assert_eq!(parsed.object_key, "key/path.md");
}

#[test]
fn bare_key_without_default_bucket_is_none() {
    assert!(parse_oss_path("raw/key.md", None).is_none());
}
