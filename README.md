# Tablestore Agent Storage SDK

SDKs for Alibaba Cloud Tablestore agent storage: knowledge bases, documents,
retrieval and memory APIs.

| Language | Source version | Documentation |
| --- | --- | --- |
| Python | 1.0.11 | [Python SDK](python/README.md) |
| TypeScript | 0.0.13 | [TypeScript SDK](typescript/README.md) |
| Rust | 0.0.1 | [Rust SDK](rust/README.md) |
| Go | 0.0.3 | [Go SDK](go/README.md) |

API coverage differs by language. Consult each SDK's documentation and models
for supported requests. The Go SDK's [protocol coverage](go/PROTOCOL.md) includes
the file, Wiki and Dream interfaces and their service capability requirements.
Source versions do not by themselves indicate a new package or tagged release.

Use your Tablestore instance endpoint and provide credentials through your
application's configuration. Never commit credentials or real tenant data.
Live integration tests create and delete resources; configure a dedicated test
instance and follow the relevant SDK's testing instructions.
