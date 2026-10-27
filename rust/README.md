# tablestore-agent-storage

Async Rust SDK for the Aliyun Tablestore agent storage service: knowledge-base
and memory APIs, with an optional OSS upload helper. Mirrors the TypeScript
SDK surface 1:1, so an agent that already knows the TS field names will reach
for the same names here.

- `reqwest` + `tokio`, `rustls` by default — no system OpenSSL needed.
- Builder-style client, dual input mode (typed model **or** `serde_json::Value`).
- HMAC-SHA1 (V2, default) and HMAC-SHA256 (V4) signing.
- Cross-language signature fixtures match the Python and TypeScript SDKs
  byte-for-byte.

## Install

```toml
[dependencies]
tablestore-agent-storage = { version = "0.0.1" }
tokio = { version = "1", features = ["macros", "rt-multi-thread"] }
```

Default features pull in `oss-upload` (Aliyun OSS PUT/DELETE helper) and
`rustls-tls`. Disable defaults and opt into `native-tls` to link against the
system TLS stack instead.

## Quick start

```rust
use tablestore_agent_storage::{AgentStorageClient, models::CreateKnowledgeBaseRequest};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = AgentStorageClient::builder()
        .endpoint("https://your-instance.cn-beijing.ots.aliyuncs.com")
        .instance_name("your-instance")
        .access_key_id(std::env::var("OTS_ACCESS_KEY_ID")?)
        .access_key_secret(std::env::var("OTS_ACCESS_KEY_SECRET")?)
        .build()?;

    let response = client
        .create_knowledge_base(CreateKnowledgeBaseRequest::new("kb-demo"))
        .await?;
    println!("{response:#?}");
    Ok(())
}
```

Run the bundled example:

```bash
OTS_ENDPOINT=https://your-instance.cn-beijing.ots.aliyuncs.com \
OTS_INSTANCE=your-instance \
OTS_ACCESS_KEY_ID=... OTS_ACCESS_KEY_SECRET=... \
cargo run --example quickstart
```

## Two input modes

Every client method accepts `impl Into<RequestBody>`, so you can pass either a
typed request model **or** a `serde_json::Value`. Typed models catch field-name
typos at compile time; the JSON form mirrors the on-wire schema and is the
simplest path when wiring the SDK into an agent / tool-call workflow.

```rust
use serde_json::json;
use tablestore_agent_storage::models::{
    RetrievalConfiguration, RetrievalQuery, RetrieveRequest, SearchType,
};

// Typed model
client.retrieve(RetrieveRequest {
    knowledge_base_name: "kb".into(),
    retrieval_query: RetrievalQuery::text("hello world"),
    retrieval_configuration: Some(RetrievalConfiguration {
        search_type: Some(vec![SearchType::DenseVector, SearchType::FullText]),
        ..Default::default()
    }),
    subspace: None,
}).await?;

// Raw JSON — same wire shape
client.retrieve(json!({
    "knowledgeBaseName": "kb",
    "retrievalQuery": { "text": "hello world", "type": "TEXT" },
    "retrievalConfiguration": {
        "searchType": ["DENSE_VECTOR", "FULL_TEXT"],
    },
})).await?;
```

## Partial updates with `Patch`

Update methods use the [`Patch<T>`] enum to mirror the TypeScript SDK's
three-state field (`UNSET` / value / `null`):

- `Patch::Unchanged` — omit the field; leave the server-side value alone.
- `Patch::Set(value)` — replace.
- `Patch::Clear` — send `null` to clear it.

```rust
use tablestore_agent_storage::models::{Patch, UpdateKnowledgeBaseRequest};

client.update_knowledge_base(UpdateKnowledgeBaseRequest {
    knowledge_base_name: "kb".into(),
    description: Patch::Clear,        // explicitly drop the description
    tags: Patch::Unchanged,           // leave tags alone
    retrieval_configuration: None,
}).await?;
```

## Extra request headers

Attach arbitrary headers to every signed request — handy for trace IDs,
multi-tenant tags, or any custom routing your gateway expects. Use
`extra_header` for incremental additions or `extra_headers` to replace the
whole map. Entries whose name starts with `x-ots-` participate in the
signature, so the server sees a signed value.

```rust
use std::collections::BTreeMap;
use tablestore_agent_storage::AgentStorageClient;

let mut presets = BTreeMap::new();
presets.insert("x-app-tag".to_string(), "demo".to_string());

let client = AgentStorageClient::builder()
    .endpoint("…")
    .instance_name("…")
    .access_key_id("…")
    .access_key_secret("…")
    .extra_headers(presets)
    .extra_header("x-ots-trace-id", "trace-abc")
    .build()?;
```

## V2 vs V4 signing

V2 (HMAC-SHA1) is the default. Switch to V4 (HMAC-SHA256) by setting the sign
version and supplying a region:

```rust
use tablestore_agent_storage::{AgentStorageClient, SignVersion};

let client = AgentStorageClient::builder()
    .endpoint("…")
    .instance_name("…")
    .access_key_id("…")
    .access_key_secret("…")
    .sign_version(SignVersion::V4)
    .region("cn-beijing")
    .build()?;
```

## Upload local files to OSS, then index

Enable the `oss-upload` feature (on by default) and bring the extension trait
into scope:

```rust
use serde_json::json;
use tablestore_agent_storage::oss::{AgentStorageClientExt, OssUploadConfig};

let oss = OssUploadConfig {
    endpoint: "https://oss-cn-beijing.aliyuncs.com".into(),
    bucket: "your-bucket".into(),
};

client.upload_documents(
    json!({
        "knowledgeBaseName": "kb",
        "documents": [{ "filePath": "/path/to/local.md" }],
    }),
    &oss,
).await?;
```

The helper uploads each file to OSS first, swaps `filePath` for the resulting
`oss://bucket/key` URI, then calls `AddDocuments`. If the indexing call fails
on the server side, the uploaded objects are best-effort deleted so they don't
linger.

To swap in a custom uploader (tests, alternate clouds, richer OSS SDK),
implement [`OssUploader`] and pass it via `upload_documents_with`.

## Errors

The SDK separates client-side and service-side failures so callers can branch
cleanly:

| Variant         | When it fires                                            |
| --------------- | -------------------------------------------------------- |
| `Error::Client` | bad config, transport failure, response-signature check, MD5 mismatch, file-not-found |
| `Error::Service` | non-2xx from the service, with `status_code`, `request_id`, `code`, `message` |

## Feature flags

- `oss-upload` *(default)* — enables the `oss` module and `AgentStorageClientExt`.
- `rustls-tls` *(default)* — TLS via `rustls`. No system OpenSSL required.
- `native-tls` — TLS via the system stack. Mutually exclusive with
  `rustls-tls`; turn defaults off if you want only this one.

## Tests

```bash
cd rust
cargo test           # unit + integration tests (cross-SDK signature fixtures)
cargo clippy --all-targets --all-features -- -D warnings
```

The integration tests under `rust/tests/` use [`wiremock`] to stand up a fake
OTS endpoint and a mock OSS uploader, so they run offline.

## Versioning

`0.0.x` — API is allowed to shift while we wait for downstream feedback. Pin
exact versions until 0.1 lands.

[`Patch<T>`]: https://docs.rs/tablestore-agent-storage/latest/tablestore_agent_storage/models/enum.Patch.html
[`OssUploader`]: https://docs.rs/tablestore-agent-storage/latest/tablestore_agent_storage/oss/trait.OssUploader.html
[`wiremock`]: https://crates.io/crates/wiremock
