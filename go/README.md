# Tablestore Agent Storage SDK for Go

Go SDK for Tablestore Agent Storage, covering the knowledge-base, document,
chunk, retrieval, memory (store / long-term / short-term / task / dream) and
file-memory APIs. Feature-aligned with the Python and TypeScript SDKs in
this repository.

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

## Items (file memory)

`filemem` memory stores keep versioned memory files; `file+ots` stores expose
long-term memories as a read-only projected file tree. Both are accessed
through the item APIs: `AddItem`, `ListItems`, `GetItem` (metadata only with
`IncludeContent: agentstorage.Bool(false)`), `UpdateItem` (set exactly one of
`Content` to rewrite or `NewPath` to rename), `DeleteItem`,
`ListItemVersions`, `GetItemVersion` and `RedactItemVersion`. Every request
carries `"type": "memoryfile"`, filled automatically when `Type` is empty:

```go
scoped := agentstorage.ItemScopedRequest{
	MemoryStoreName: "agent_memory",
	Scope:           agentstorage.Scope{AppID: "a", TenantID: "t", AgentID: "ag", RunID: "r"},
}
item, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
	ItemScopedRequest: scoped, Path: "/notes/a.md", Content: "hello",
})
_, err = client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
	ItemScopedRequest: scoped, Path: "/notes/a.md",
	Content:        agentstorage.String("updated"),
	ExpectedSHA256: item.ContentSHA256, // optional compare-and-set guard
})
_, err = client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
	ItemScopedRequest: scoped, Path: "/notes/a.md", NewPath: "/notes/b.md", // rename
})
listed, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{
	ItemScopedRequest: scoped, PathPrefix: "/notes/",
})
_ = listed // listed.ReadOnly is true for file+ots projections
```

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
cd go && go test ./...
```
