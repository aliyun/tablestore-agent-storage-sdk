use std::path::Path;

/// Default subspace used when [`build_oss_object_key`] is called without one.
pub const DEFAULT_SUBSPACE: &str = "_default";

/// Parsed OSS location.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OssLocation {
    /// Bucket name.
    pub bucket_name: String,
    /// Object key (no leading slash).
    pub object_key: String,
}

/// Build the canonical `<instance>/<kb>/source_doc/<subspace>/<absolute-path>`
/// object key — matches the TypeScript implementation byte-for-byte.
pub fn build_oss_object_key(
    instance_name: &str,
    knowledge_base_name: &str,
    subspace: Option<&str>,
    file_path: &str,
) -> String {
    let absolute = absolute_path(file_path);
    let stripped = absolute.trim_start_matches('/');
    format!(
        "{instance}/{kb}/source_doc/{subspace}/{path}",
        instance = instance_name,
        kb = knowledge_base_name,
        subspace = subspace.unwrap_or(DEFAULT_SUBSPACE),
        path = stripped
    )
}

/// Render `oss://bucket/key`.
pub fn build_oss_full_path(bucket_name: &str, object_key: &str) -> String {
    format!("oss://{bucket_name}/{object_key}")
}

/// Parse an `oss://bucket/key` URI. Returns `None` for an obviously invalid
/// URI, or wraps a bare key in `default_bucket_name` when supplied.
pub fn parse_oss_path(oss_path: &str, default_bucket_name: Option<&str>) -> Option<OssLocation> {
    if let Some(rest) = oss_path.strip_prefix("oss://") {
        let separator = rest.find('/')?;
        Some(OssLocation {
            bucket_name: rest[..separator].to_string(),
            object_key: rest[separator + 1..].to_string(),
        })
    } else {
        default_bucket_name.map(|bucket| OssLocation {
            bucket_name: bucket.to_string(),
            object_key: oss_path.to_string(),
        })
    }
}

fn absolute_path(path: &str) -> String {
    let candidate = Path::new(path);
    if candidate.is_absolute() {
        return path.to_string();
    }
    match std::env::current_dir() {
        Ok(cwd) => cwd.join(candidate).to_string_lossy().into_owned(),
        Err(_) => path.to_string(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn builds_object_key_with_default_subspace() {
        let key = build_oss_object_key("inst", "kb", None, "/tmp/example.md");
        assert_eq!(key, "inst/kb/source_doc/_default/tmp/example.md");
    }

    #[test]
    fn round_trips_full_path() {
        let full = build_oss_full_path("bucket", "key/path.md");
        let parsed = parse_oss_path(&full, None).unwrap();
        assert_eq!(parsed.bucket_name, "bucket");
        assert_eq!(parsed.object_key, "key/path.md");
    }

    #[test]
    fn wraps_bare_key_with_default_bucket() {
        let parsed = parse_oss_path("raw/key.md", Some("bucket")).unwrap();
        assert_eq!(parsed.bucket_name, "bucket");
        assert_eq!(parsed.object_key, "raw/key.md");
    }
}
