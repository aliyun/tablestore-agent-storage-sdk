# Go SDK 0.0.3 protocol coverage

Version 0.0.3 adds typed requests and responses for the Memory API features
below. Service availability and limits depend on the target deployment.
Knowledge-base, document, chunk and retrieval APIs remain supported.

| Area | Actions | Changes in 0.0.3 |
| --- | --- | --- |
| Stores | CreateMemoryStore, GetMemoryStore, UpdateMemoryStore, DeleteMemoryStore, ListMemoryStores | inputType/outputTypes, storeUid, Wiki reflection setting, explicit false and empty-list serialization |
| Memories | AddMemories, SearchMemories, ListMemories, GetMemory, UpdateMemory, DeleteMemory | interaction, synchronous memoryIds, filters, thresholds, optional scope for ID lookup, empty metadata updates |
| Audit and ingestion | ListMemoryStoreMessages, ListMemoryStoreRequests, GetMemoryTask, ListMemoryTasks, ListMemoryStoreScopes | structured task errors, filter-only scope queries, heterogeneous scope directory entries |
| Dream | CreateMemoryDreamTask, GetMemoryDreamTask, ListMemoryDreamTasks, CancelMemoryDreamTask, ListMemoryDreamActions, ApplyMemoryDreamActions | effort, file input/output, plan approval, diagnostics, nested list responses, file and Wiki correction actions |
| Files | AddItem, GetItem, ListItems, UpdateItem, DeleteItem, ListItemVersions, GetItemVersion, RedactItemVersion, SearchItems, DeleteItemVersions | itemId/path/version protocol, binary content, media types, summaries, filters, ranged history deletion |
| Wiki | GetWiki, UpdateWiki, IngestWiki, ListWikiPages, GetWikiPage, SearchWikiPages | compile options, interaction and reflection, partial publication, quality and failure diagnostics |

## Compatibility

- Requests use the public Tablestore instance endpoint and retain OTS SignV2 /
  SignV4 authentication, response validation and service request IDs.
- Resource version preconditions protect both content and path. Deprecated
  `NewPath` and `ExpectedSHA256` fields are translated by the typed client;
  their removed wire names are not sent. SHA-guarded updates first read the
  current hash and version, then write with a version precondition. A conflict
  is returned without retrying the write. Mutating guard reads use basic view;
  legacy no-op requests retain the requested response view.
- `UpdateItem.ItemID` identifies the source; `Path` is the rename destination.
  Legacy path addressing requires an explicit `NewPath` or `ExpectedSHA256`.
  A missing ItemID on a current-format request fails before reading or writing.
- In `UpdateMemory`, nil Metadata leaves stored metadata unchanged, a non-nil
  empty map clears it, and a populated map replaces it. Before 0.0.3, empty
  maps were omitted. Review conditionally constructed maps when upgrading.
- `ListMemoryStoreScopesResponse.ScopeEntries` preserves heterogeneous
  directory fields; `Scopes` remains the legacy four-field projection.
- New stores support scope filters using and/or/not with eq/in/exists.
  Historical prefix/wildcard/range support depends on the store's contract.
- Private callbacks and administrative routes are not SDK methods.

## Item request rules

| Action | Parameter requirements |
| --- | --- |
| AddItem | Ordinary files use `.md` paths. Text and Base64 are mutually exclusive; binary data requires a media type. Binary creation uses Wiki raw paths and requires service support. |
| GetItem | Exactly one of itemId/path, concrete scope, basic/full view. |
| UpdateItem | itemId identifies the source. At least one content/contentBase64/path field is required; empty content clears the file. overwrite requires a destination path. Identical updates do not add a version. |
| DeleteItem | Exact deletion accepts one address and an optional version guard. Ranged deletion forbids itemId/path/precondition. deleteVersions requires wildcard scope and cannot be combined with pathPrefix. |
| ListItems | depth is 0 or 1. Prefixes are segment-aligned. Explicit limit is 1–100, or 1–20 for full view. Scope and filter intersect. Wiki system paths have additional restrictions. |
| SearchItems | Searches managed files from message-input stores with file output. Requires a nonempty query and concrete appId. topK is 0–50; thresholds are 0–1. Filters select scopes; pagination is not supported. |
| ListItemVersions | itemId/path are exclusive; path requires exact scope. RFC3339 time bounds must be ordered. Operation is created/modified/deleted. |
| GetItemVersion | Requires an opaque version ID and matching scope ownership. Historical content is immutable. |
| RedactItemVersion | Exact scope and version ID; irreversibly clears historical content. Managed files are read-only. |
| DeleteItemVersions | Wildcard scope without additional selectors. Current files must be removed first. Poll the same complete scope until both versions and nextToken are empty. |

Use GetItem for binary content and summaries. ListItems omits binary bodies even
in full view. Effective size limits and supported binary formats are enforced
by the service; the SDK does not treat a documented default as a universal limit.

## Validation and known limitations

Unit tests cover request JSON, optional/empty/null values, action routing,
response decoding, signing, error classification and legacy compatibility.
Without credentials, `go test ./...` skips live integration suites.

The live tests use synthetic data in uniquely named stores and perform
ownership-checked cleanup. They check persisted content, retrieval, versions,
scope isolation and terminal task results. A task status alone does not prove
that the intended output is usable.

- File-input structured extraction and retrieval have dedicated tests for
  retained-source and structured-only modes.
- A known service compatibility issue can reject file-derived structured
  memories as high-effort skill Dream inputs with `invalid_projection` after
  extraction succeeds. The source metadata does not satisfy Dream's input
  validation. The regression remains a failing test on affected deployments;
  the dependent generated-artifact preservation path remains unverified there.
- Ranged deletion and historical-version deletion require deployment support.
  Successful cleanup semantics have not been verified on deployments where
  those operations are unavailable.
- Structured-only public API checks do not prove physical deletion of source,
  version or intermediate-artifact records.
- Binary-format tests verify storage and byte/hash/version readback for ten
  formats. Successful parsing and Wiki publication are exercised with PDF;
  this does not establish parsing support for every Office or image format.
- The 1001-conversation boundary test requires `MEMORY_E2E_LARGE_SCOPE=1` and
  runs separately. It checks a scope limit, not throughput or load capacity.

Run the live suites only against a dedicated test instance with the required
service capabilities. See [README.md](README.md#test) for environment variables
and commands. No credentials are included in the repository.
