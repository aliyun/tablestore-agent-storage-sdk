# Tablestore Agent Storage SDK for Go

Go SDK for Tablestore Agent Storage, covering the knowledge-base, document,
chunk, retrieval, memory (store / long-term / short-term / task / dream),
file-memory and Wiki APIs. Version **0.0.3** adds current Memory request fields,
file search and history cleanup, and typed Wiki APIs.
See [protocol coverage and validation](PROTOCOL.md) for the exact scope and
live-test limitations.

## Install

```bash
go get github.com/aliyun/tablestore-agent-storage-sdk/go
```

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

func main() {
	client, err := agentstorage.NewClient(agentstorage.Config{
		Endpoint:        "https://your-instance.cn-hangzhou.ots.aliyuncs.com",
		InstanceName:    "your-instance",
		AccessKeyID:     "...",
		AccessKeySecret: "...",
		// or: APIKey: "..." (HTTPS only)
	})
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	// Memory store + memories
	_, _ = client.CreateMemoryStore(ctx, &agentstorage.CreateMemoryStoreRequest{
		MemoryStoreName: "agent_memory",
		Description:     "Agent long-term memory",
		InputType:       agentstorage.StoreInputMessages,
		OutputTypes:     []string{agentstorage.StoreOutputStructured},
	})
	_, _ = client.AddMemories(ctx, &agentstorage.AddMemoriesRequest{
		MemoryStoreName: "agent_memory",
		Scope:           &agentstorage.Scope{AppID: "app-001", TenantID: "user-001"},
		Messages: []agentstorage.MessagePayload{
			{Role: "user", Content: "我喜欢喝咖啡"},
			{Role: "assistant", Content: "好的，我记住了"},
		},
	})
	hits, err := client.SearchMemories(ctx, &agentstorage.SearchMemoriesRequest{
		MemoryStoreName: "agent_memory",
		Scope:           &agentstorage.Scope{AppID: "app-001", TenantID: "user-001", AgentID: "*", RunID: "*"},
		Query:           "用户喜欢什么饮品",
		TopK:            5,
	})
	if err == nil {
		for _, hit := range hits.Results {
			fmt.Println(hit.Unit.Text, hit.Score)
		}
	}
}
```

## Authentication

- **AccessKey signing** (default): `AccessKeyID` + `AccessKeySecret`
  (+ optional `STSToken`), protocol v2 (HMAC-SHA1). Set
  `SignVersion: agentstorage.SignV4` + `Region` for the v4 protocol.
- **API key**: `APIKey`, requires an `https://` endpoint. Does not apply to OSS.

## Knowledge base + document upload

`UploadDocuments` uploads local files to OSS first and then registers them
via `AddDocuments`; configure `OSSEndpoint` + `OSSBucketName` (credentials
default to the Tablestore access keys):

```go
client, _ := agentstorage.NewClient(agentstorage.Config{
	Endpoint:        "https://...",
	InstanceName:    "...",
	AccessKeyID:     "...",
	AccessKeySecret: "...",
	OSSEndpoint:     "https://oss-cn-hangzhou.aliyuncs.com",
	OSSBucketName:   "your-bucket",
})

resp, err := client.UploadDocuments(ctx, &agentstorage.UploadDocumentsRequest{
	KnowledgeBaseName: "test_kb",
	Documents: []agentstorage.UploadDocumentItem{
		{FilePath: "/path/to/file.pdf", Metadata: map[string]any{"author": "aliyun"}},
	},
})

// Delete documents and their OSS files:
_, err = client.DeleteDocumentsWithOptions(ctx, &agentstorage.DeleteDocumentsRequest{
	KnowledgeBaseName: "test_kb",
	Documents:         []agentstorage.DeleteDocumentItem{{DocID: "doc-1"}},
}, agentstorage.DeleteDocumentsOptions{DeleteFile: true})
```

Retrieval with metadata filters:

```go
result, err := client.Retrieve(ctx, &agentstorage.RetrieveRequest{
	KnowledgeBaseName: "test_kb",
	RetrievalQuery:    agentstorage.RetrievalQuery{Text: "阿里云"},
	RetrievalConfiguration: &agentstorage.RetrievalConfiguration{
		SearchType: []agentstorage.SearchType{agentstorage.SearchTypeDenseVector},
		Filter:     agentstorage.Equals("category", "cloud"),
	},
})
```

## Tri-state update fields

Update requests distinguish "leave unchanged", "clear" and "replace" via
`Nullable`:

```go
_, err := client.UpdateKnowledgeBase(ctx, &agentstorage.UpdateKnowledgeBaseRequest{
	KnowledgeBaseName: "test_kb",
	Description:       agentstorage.NullableOf("new description"), // replace
	Tags:              agentstorage.Null[[]string](),              // clear
	// omitted fields stay unchanged
})
```

## Memory semantics: title/keywords, event time, session affinity

Long-term memory units carry a server-extracted `Title` + `Keywords` and a
time triple: `EffectiveAt` (when the remembered fact happens — the value to
compare when deciding which memory is newer), `UpdatedAt` (when the memory was
last revised) and `TimeSource` / `TimePrecision` qualifying `EffectiveAt`. All
of them are read-only side effects of `AddMemories`; parse the times with
`time.RFC3339` (the fractional part varies) and never string-compare them.
`UpdateMemory` can hand-correct them:

```go
_, err := client.UpdateMemory(ctx, &agentstorage.UpdateMemoryRequest{
	MemoryStoreName: "agent_memory",
	MemoryID:        unit.ID,
	Scope:           scope, // all four fields, no wildcards
	Title:           agentstorage.String("Wedding set for 2026-08-01"),
	Keywords:        agentstorage.Strings("wedding", "hangzhou"), // Strings() clears
	EffectiveAt:     agentstorage.String("2026-08-01"),           // pair required
	TimePrecision:   agentstorage.String(agentstorage.TimePrecisionDay),
})
```

`SearchMemories` softly prefers the current session when `ContextScope` is
passed alongside a wildcard `Scope`:

```go
resp, err := client.SearchMemories(ctx, &agentstorage.SearchMemoriesRequest{
	MemoryStoreName: "agent_memory",
	Query:           "what coffee does the user like",
	Scope:           &agentstorage.Scope{AppID: "app-001", TenantID: "user-001", AgentID: "assistant", RunID: "*"},
	ContextScope:    &agentstorage.Scope{AppID: "app-001", TenantID: "user-001", AgentID: "assistant", RunID: "session-001"},
	TopK:            5,
})
for _, hit := range resp.Results {
	// RankingScore is nil unless the service runs session affinity in "on"
	// mode; results always come back pre-sorted.
	if hit.RankingScore != nil && hit.RankingSignals != nil && hit.RankingSignals.SameSession {
		fmt.Println("same-session boost", hit.Score, "->", *hit.RankingScore)
	}
}
```

`ContextScope` must be exact (four fields, no `*`) and contained in `Scope`,
and `Scope` itself must keep a wildcard — a fully concrete `Scope` is answered
normally but without any affinity ranking. A service that predates the field
rejects the whole request with `400 BAD_REQUEST`, and one with the V2 write
path disabled answers `409 CONFLICT` to `UpdateMemory` calls carrying
`Title` / `Keywords` / `EffectiveAt` / `TimePrecision`.

## Store input and output types

Create stores with `InputType` (`messages` or `file`) and `OutputTypes`
(`structured`, `file`, or both). Send them together and omit the legacy
`StorageMode`. The service validates supported combinations. `UpdateMemoryStore`
can update output types and `WikiReflectionEnabled`; input type is immutable.
Optional booleans use pointers, so `Bool(false)` is sent explicitly.
Message-input stores that include file output require asynchronous
`AddMemories` (`Sync: false`), including dual structured/file output.
Synchronous `MemoryIDs` are available for structured-only stores.

## Scope filters and search thresholds

`ScopeFilter` is the Memory Scope Directory expression, distinct from the
knowledge-base `MetadataFilter`. `ListMemoryStoreScopes`, `ListMemories`,
`SearchMemories`, `ListItems` and `SearchItems` accept it. Current stores support
`and`, `or`, `not`, `eq`, `in` and `exists`; the service applies historical
`prefix`, `wildcard` and `range` support according to the store's creation-time
contract. Pagination tokens must be replayed with the same scope/filter.

```go
filter := &agentstorage.ScopeFilter{
    Op: "eq", Field: "appId", Value: agentstorage.String("app-001"),
}
scopes, err := client.ListMemoryStoreScopes(ctx, &agentstorage.ListMemoryStoreScopesRequest{
    MemoryStoreName: "agent_memory", Filter: filter, // zero Scope is omitted
})
if err != nil { return }
for _, entry := range scopes.ScopeEntries {
    // Scopes remains the legacy four-field view; ScopeEntries preserves
    // scopeType, scopeKey and all dynamic dimensions, including Wiki entries.
    fmt.Println(entry.ScopeType, entry.ScopeKey, entry.Dimensions)
}
```

Use `SearchMemoriesRequest.Thresholds` for independent `MinSimilarity` and
`MinScore` bounds; both nonzero bounds must pass before session ranking.
The legacy top-level `MinSimilarity` filters canonical score, and must not be
combined with `Thresholds`. `GetMemory`, `UpdateMemory` and `DeleteMemory` can
omit Scope for ID lookup on current servers. An explicitly empty Metadata map
in `UpdateMemory` clears metadata; nil leaves it unchanged.
Before 0.0.3, empty maps were omitted. When upgrading, use nil for conditionally
built maps if an empty result should leave the stored metadata unchanged.

## Items (file memory)

File-input stores accept file writes. Managed files produced by message-input
stores are read-only. The SDK fills `type: "memoryfile"` when `Type` is empty.
`GetItem` and single `DeleteItem` accept exactly one of `ItemID` or `Path`.
`View: ItemViewFull` includes content; the default `ItemViewBasic` omits it.
`ListItems` and `ListItemVersions` accept at most 100 entries per page, capped
at 20 for the full view. `UpdateItem.Path` remains a `*string`: nil omits the
rename target, while an explicit empty string is sent for server validation.
`Overwrite` is meaningful only with a target path.

```go
scoped := agentstorage.ItemScopedRequest{
    MemoryStoreName: "file_store",
    Scope: agentstorage.Scope{AppID: "a", TenantID: "t", AgentID: "ag", RunID: "r"},
}
item, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
    ItemScopedRequest: scoped, Path: "/notes/a.md", Content: "hello",
})
if err != nil { return }
updated, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
    ItemScopedRequest: scoped,
    ItemID: item.ItemID,
    Path: agentstorage.String("/notes/b.md"), // destination; may be combined with Content atomically
    Content: agentstorage.String("updated"),
    Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: item.LatestVersionID},
})
_ = updated
```

Ordinary `AddItem` paths must end in `.md` and use text `Content`.
Binary creation uses the Wiki raw namespace (`/.wiki/raw/...`) and requires
the server binary capability. For those uploads, set `ContentBase64: String(base64.StdEncoding.EncodeToString(data))`
and `MediaType`, and leave `Content` unset. Binary content is returned as
`ContentBase64` on full item/version reads. Availability and size limits depend
on the service configuration. `ListItems` omits binary content and summaries
even in full view; use `GetItem` for binary content and file summaries.

`SearchItems` requires `inputType=messages` with a `file` output; it does not
search uploaded files or Wiki pages. It searches managed-file summaries with optional `Filter`, `TopK`
and `Thresholds`, returning identity, summary, version and scores.
`ListItemVersions` supports an `ItemID` or `Path` filter. `DeleteItemVersions`
removes historical versions asynchronously in a wildcard scope after all current
items in that scope are removed. `DeleteItem.DeleteVersions` applies only to
ranged deletes; a pending task ID means acceptance, not completed deletion.

Compatibility: `NewPath` remains a deprecated rename alias. An `UpdateItem`
with Path but no ItemID resolves the old path through `GetItem` only when it
explicitly uses `NewPath` or `ExpectedSHA256`. Otherwise a missing ItemID fails
locally, so a malformed current request cannot write to the rename destination.
`ExpectedSHA256` on update/delete checks the current hash and then sends a
resource-version precondition, preserving protection across concurrent writes.
A local hash mismatch returns `ClientError`; a subsequent server conflict is
returned unchanged and is not retried. Raw `Do` callers must use the current
wire fields directly. `ApiKeyID` version filtering was removed by the service
and nonempty values now fail locally.

## Wiki

Wiki APIs use a three-part `AgentScope`; do not include runId. The methods are
`GetWiki`, `UpdateWiki`, `IngestWiki`, `ListWikiPages`, `GetWikiPage` and
`SearchWikiPages`.

```go
wikiScope := agentstorage.WikiScopedRequest{
    MemoryStoreName: "file_store",
    Scope: agentstorage.AgentScope{AppID: "a", TenantID: "t", AgentID: "ag"},
}
_, err = client.UpdateWiki(ctx, &agentstorage.UpdateWikiRequest{
    WikiScopedRequest: wikiScope,
    CompileOptions: agentstorage.NullableOf(agentstorage.WikiCompileOptions{
        Language: "zh",
        Incremental: &agentstorage.WikiIncrementalOptions{Enabled: true},
    }),
})
```

Omitted compile options stay unchanged; `Null[WikiCompileOptions]()` or an
empty options object resets them. `LatestBuild` retains progress, partial
publication, completed/planned pages, quality, attempt failures and diagnostics.
`WikiPage.Reflection` and optional `InteractionRef` support feedback-aware reads.
An accepted ingest or partial snapshot does not establish a completed build.

## Dream and ingestion tasks

`CreateMemoryDreamTask` supports `Effort`, file input `OutputScope`/`MaxItems`,
and incremental processing. High-effort plan mode uses `DreamApplyModePlan` and
can return `awaiting_plan_approval`; inspect `Plan.DeleteMemoryIDs` before an
explicit `ApplyMemoryDreamActions` request with `Mode: DreamApplyRequestModePlan`
and `Approver`. Creating or polling a task never approves its plan.

Get and List Dream responses expose nested `Input` and `Actions`, stage,
failure code, file outputs, plan and diagnostics. A `staged` action is a private
candidate, not an applied result. List responses also populate the legacy
flat count/scope fields. Ingestion tasks retain primary/last error
stages and classes. Synchronous `AddMemories` exposes `MemoryIDs` when returned
by the service.

## Raw requests

`Client.Do` sends any action with an arbitrary payload, for API fields not
yet covered by the typed models:

```go
var result map[string]any
err := client.Do(ctx, "ListMemoryStores", map[string]any{"limit": 10}, &result)
```

## Errors

- `*agentstorage.ServiceError` — non-2xx service responses
  (`StatusCode` / `Code` / `Message` / `RequestID`).
- `*agentstorage.ClientError` — client-side failures (configuration,
  serialization, transport, response validation).

## Test

```bash
cd go
go test ./...
go test -race ./...
go vet ./...
```

Live tests read `OTS_ENDPOINT`, `OTS_INSTANCE_NAME`, and
`OTS_ACCESS_KEY_ID`/`OTS_ACCESS_KEY_SECRET` (or `OTS_API_KEY`) from the environment.
Do not put credentials in source files. With those variables set:

```bash
go test ./e2e -run '^TestCurrent(File|Memory|StructuredMemory)ProtocolE2E$' -count=1 -v -timeout=15m
```

The current-protocol tests create uniquely named stores, verify state/content
and delete/read back those stores. A missing environment skips live tests; an
API or semantic failure fails them. Empty-scope Wiki checks cover protocol and
configuration; they do not claim successful model-generated Wiki compilation.

File-input structured extraction has its own live suites:

```bash
go test ./e2e -run '^TestFileStructured(Only)?ExtractionE2E$' -count=1 -v -timeout=15m
```

These verify conversion task identity, persisted/retrieved facts, retained-source
update/rename/delete behavior, and structured-only source restrictions/path
reuse. They do not prove physical source/version/artifact purge in
structured-only mode; see [PROTOCOL.md](PROTOCOL.md) for the acceptance boundary.

The formal release suites additionally cover ten binary formats and size limits,
accepted-response loss, concurrent version guards, delete/redact cascades,
multi-Scope file search, successful Wiki publication, and approved Dream plans.
See [PROTOCOL.md](PROTOCOL.md) for coverage and known service limitations.
The 1001-conversation boundary test
requires `MEMORY_E2E_LARGE_SCOPE=1`; it is intentionally a separate live run.
