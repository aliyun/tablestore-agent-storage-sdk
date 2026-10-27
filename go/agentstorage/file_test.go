package agentstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestItemJSONAPIs(t *testing.T) {
	scope := Scope{AppID: "app", TenantID: "tenant", AgentID: "agent", RunID: "run"}
	scoped := ItemScopedRequest{MemoryStoreName: "store", Scope: scope}
	wantScope := map[string]any{"appId": "app", "tenantId": "tenant", "agentId": "agent", "runId": "run"}
	itemJSON := `{"type":"memoryfile","itemId":"m1","scope":{"appId":"app","tenantId":"tenant","agentId":"agent","runId":"run"},"path":"/notes/a.md","content":"hello","contentSha256":"sha-new","contentSizeBytes":5,"latestSeq":2,"latestVersionId":"v2","createdAt":"2026-07-16T01:00:00Z","updatedAt":"2026-07-16T02:00:00Z"}`
	versionJSON := `{"type":"memoryfile","versionId":"v2","itemId":"m1","scope":{"appId":"app","tenantId":"tenant","agentId":"agent","runId":"run"},"operation":"modified","path":"/notes/a.md","content":"hello","contentSha256":"sha-new","contentSizeBytes":5,"createdBy":{"actorType":"session","actorId":"session-1"},"createdAt":"2026-07-16T02:00:00Z"}`

	tests := []struct {
		name     string
		action   string
		status   int
		wantBody map[string]any
		response string
		call     func(*Client) error
	}{
		{
			name:   "add",
			action: "AddItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"path": "/notes/a.md", "content": "hello", "sessionId": "session-1", "view": "full",
			},
			response: itemJSON,
			call: func(client *Client) error {
				req := &AddItemRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md", Content: "hello",
					SessionID: "session-1", View: ItemViewFull,
				}
				resp, err := client.AddItem(context.Background(), req)
				if err == nil && (resp.Type != ItemTypeMemoryFile || resp.ItemID != "m1" ||
					resp.Scope != scope || resp.LatestVersionID != "v2" ||
					resp.Content == nil || *resp.Content != "hello") {
					return fmt.Errorf("response = %+v", resp)
				}
				if req.Type != "" {
					return fmt.Errorf("caller request mutated: Type = %q", req.Type)
				}
				return err
			},
		},
		{
			name:   "update composite content and rename",
			action: "UpdateItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"itemId": "m1", "content": "updated", "path": "/notes/b.md", "overwrite": true,
				"precondition": map[string]any{"expectedVersionId": "v1"}, "sessionId": "session-1", "view": "full",
			},
			response: itemJSON,
			call: func(client *Client) error {
				_, err := client.UpdateItem(context.Background(), &UpdateItemRequest{
					ItemScopedRequest: scoped, ItemID: "m1", Content: String("updated"),
					Path: String("/notes/b.md"), Overwrite: true, Precondition: &ItemPrecondition{ExpectedVersionID: "v1"},
					SessionID: "session-1", View: ItemViewFull,
				})
				return err
			},
		},
		{
			name:   "update empty content",
			action: "UpdateItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"itemId": "m1", "content": "",
			},
			response: itemJSON,
			call: func(client *Client) error {
				_, err := client.UpdateItem(context.Background(), &UpdateItemRequest{
					ItemScopedRequest: scoped, ItemID: "m1", Content: String(""),
				})
				return err
			},
		},
		{
			name:   "delete single by itemId",
			action: "DeleteItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"itemId": "m1", "precondition": map[string]any{"expectedVersionId": "v2"}, "sessionId": "session-1",
			},
			response: `{"type":"memoryfile","itemId":"m1","deleted":true}`,
			call: func(client *Client) error {
				resp, err := client.DeleteItem(context.Background(), &DeleteItemRequest{
					ItemScopedRequest: scoped, ItemID: "m1",
					Precondition: &ItemPrecondition{ExpectedVersionID: "v2"}, SessionID: "session-1",
				})
				if err == nil && (resp.Type != ItemTypeMemoryFile || !resp.Deleted || resp.IsRanged()) {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "delete ranged",
			action: "DeleteItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store",
				"scope":      map[string]any{"appId": "app", "tenantId": "tenant", "agentId": "*", "runId": "*"},
				"pathPrefix": "/notes/",
			},
			response: `{"type":"memoryfile","taskId":"task-1","status":"pending"}`,
			call: func(client *Client) error {
				ranged := scoped
				ranged.Scope = Scope{AppID: "app", TenantID: "tenant", AgentID: "*", RunID: "*"}
				resp, err := client.DeleteItem(context.Background(), &DeleteItemRequest{
					ItemScopedRequest: ranged, PathPrefix: "/notes/",
				})
				if err == nil && (!resp.IsRanged() || resp.TaskID != "task-1" || resp.Status != "pending" || resp.Deleted) {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "get by itemId with full view",
			action: "GetItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"itemId": "m1", "view": "full",
			},
			response: itemJSON,
			call: func(client *Client) error {
				resp, err := client.GetItem(context.Background(), &GetItemRequest{
					ItemScopedRequest: scoped, ItemID: "m1", View: ItemViewFull,
				})
				if err == nil && (resp.ContentSHA256 != "sha-new" || resp.Content == nil || *resp.Content != "hello") {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "get by path defaults to basic",
			action: "GetItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope, "path": "/notes/a.md",
			},
			response: `{"type":"memoryfile","itemId":"m1","scope":{"appId":"app","tenantId":"tenant","agentId":"agent","runId":"run"},"path":"/notes/a.md","contentSha256":"sha-new","contentSizeBytes":5,"latestSeq":2,"latestVersionId":"v2","createdAt":"2026-07-16T01:00:00Z","updatedAt":"2026-07-16T02:00:00Z"}`,
			call: func(client *Client) error {
				resp, err := client.GetItem(context.Background(), &GetItemRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md",
				})
				if err == nil && (resp.ContentSizeBytes != 5 || resp.Content != nil || resp.LatestVersionID != "v2") {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "list depth one",
			action: "ListItems",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"pathPrefix": "/notes/", "depth": float64(1), "limit": float64(20), "nextToken": "next-1",
			},
			response: `{"type":"memoryfile","items":[{"type":"memoryfile","itemId":"m1","scope":{"appId":"app","tenantId":"tenant","agentId":"agent","runId":"run"},"path":"/notes/a.md","contentSha256":"sha-new","contentSizeBytes":5,"latestSeq":2,"latestVersionId":"v2","createdAt":"2026-07-16T01:00:00Z","updatedAt":"2026-07-16T02:00:00Z"},{"type":"memoryfile_prefix","scope":{"appId":"app","tenantId":"tenant","agentId":"agent","runId":"run"},"path":"/notes/deep/"}],"nextToken":"next-2","readOnly":true}`,
			call: func(client *Client) error {
				resp, err := client.ListItems(context.Background(), &ListItemsRequest{
					ItemScopedRequest: scoped, PathPrefix: "/notes/", Depth: Int(1),
					Limit: 20, NextToken: "next-1",
				})
				if err != nil {
					return err
				}
				if resp.Type != ItemTypeMemoryFile || len(resp.Items) != 2 || resp.NextToken != "next-2" || !resp.ReadOnly {
					return fmt.Errorf("response = %+v", resp)
				}
				file, dir := resp.Items[0], resp.Items[1]
				if file.IsPrefix() || file.ItemID != "m1" || file.LatestVersionID != "v2" {
					return fmt.Errorf("file entry = %+v", file)
				}
				if !dir.IsPrefix() || dir.Path != "/notes/deep/" || dir.ItemID != "" {
					return fmt.Errorf("prefix entry = %+v", dir)
				}
				return nil
			},
		},
		{
			name:   "list versions filters",
			action: "ListItemVersions",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"itemId": "m1", "sessionId": "session-1",
				"createdAtGte": "2026-07-01T00:00:00Z", "createdAtLte": "2026-07-31T00:00:00Z",
				"operation": "modified", "view": "full", "limit": float64(10), "nextToken": "versions-1",
			},
			response: `{"type":"memoryfile","versions":[` + versionJSON + `],"nextToken":"versions-2"}`,
			call: func(client *Client) error {
				resp, err := client.ListItemVersions(context.Background(), &ListItemVersionsRequest{
					ItemScopedRequest: scoped, ItemID: "m1", SessionID: "session-1",
					CreatedAtGte: "2026-07-01T00:00:00Z", CreatedAtLte: "2026-07-31T00:00:00Z",
					Operation: ItemOperationModified, View: ItemViewFull,
					Limit: 10, NextToken: "versions-1",
				})
				if err != nil {
					return err
				}
				if len(resp.Versions) != 1 || resp.NextToken != "versions-2" {
					return fmt.Errorf("response = %+v", resp)
				}
				version := resp.Versions[0]
				if version.Operation != ItemOperationModified || version.Scope != scope {
					return fmt.Errorf("version = %+v", version)
				}
				if version.CreatedBy == nil || version.CreatedBy.ActorType != "session" || version.CreatedBy.ActorID != "session-1" {
					return fmt.Errorf("createdBy = %+v", version.CreatedBy)
				}
				return nil
			},
		},
		{
			name:   "list versions path filter",
			action: "ListItemVersions",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope, "path": "/notes/a.md",
			},
			response: `{"type":"memoryfile","versions":[]}`,
			call: func(client *Client) error {
				_, err := client.ListItemVersions(context.Background(), &ListItemVersionsRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md",
				})
				return err
			},
		},
		{
			name:   "get version by versionId",
			action: "GetItemVersion",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"versionId": "v2", "view": "full",
			},
			response: versionJSON,
			call: func(client *Client) error {
				resp, err := client.GetItemVersion(context.Background(), &GetItemVersionRequest{
					ItemScopedRequest: scoped, VersionID: "v2", View: ItemViewFull,
				})
				if err == nil && (resp.VersionID != "v2" || resp.ItemID != "m1" ||
					resp.Content == nil || *resp.Content != "hello") {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "redact version",
			action: "RedactItemVersion",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"versionId": "v2", "sessionId": "redactor-1",
			},
			response: `{"type":"memoryfile","versionId":"v2","itemId":"m1","scope":{"appId":"app","tenantId":"tenant","agentId":"agent","runId":"run"},"operation":"modified","path":"","contentSha256":"","contentSizeBytes":0,"createdBy":{"actorType":"session","actorId":"session-1"},"createdAt":"2026-07-16T02:00:00Z","redacted":true,"redactedAt":"2026-07-18T03:00:00Z","redactedBy":{"actorType":"session","actorId":"redactor-1"}}`,
			call: func(client *Client) error {
				resp, err := client.RedactItemVersion(context.Background(), &RedactItemVersionRequest{
					ItemScopedRequest: scoped, VersionID: "v2", SessionID: "redactor-1",
				})
				if err != nil {
					return err
				}
				if !resp.Redacted || resp.RedactedAt != "2026-07-18T03:00:00Z" || resp.Content != nil || resp.Path != "" {
					return fmt.Errorf("response = %+v", resp)
				}
				if resp.RedactedBy == nil || resp.RedactedBy.ActorType != "session" || resp.RedactedBy.ActorID != "redactor-1" {
					return fmt.Errorf("redactedBy = %+v", resp.RedactedBy)
				}
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/"+test.action {
					t.Errorf("action = %q, want /%s", r.URL.Path, test.action)
				}
				if got := r.Header.Get("If-None-Match"); got != "" {
					t.Errorf("unexpected special header If-None-Match = %q", got)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				for _, obsolete := range []string{"newPath", "expectedSha256", "includeContent"} {
					if _, ok := body[obsolete]; ok {
						t.Fatalf("obsolete field %q in body: %#v", obsolete, body)
					}
				}
				if !reflect.DeepEqual(body, test.wantBody) {
					t.Errorf("body = %#v, want %#v", body, test.wantBody)
				}
				w.Header().Set("Content-Type", "application/json")
				if test.status != 0 {
					w.WriteHeader(test.status)
				}
				fmt.Fprint(w, test.response)
			})
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestItemRequestKeepsExplicitType verifies the SDK does not override an
// explicitly provided item type.
func TestItemRequestKeepsExplicitType(t *testing.T) {
	client := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["type"] != "customtype" {
			t.Errorf("type = %v, want customtype", body["type"])
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"type":"customtype","itemId":"m1","scope":{"appId":"app","tenantId":"tenant","agentId":"agent","runId":"run"},"path":"/a","contentSha256":"s","contentSizeBytes":1,"latestSeq":1,"latestVersionId":"v1","createdAt":"2026-07-16T01:00:00Z","updatedAt":"2026-07-16T01:00:00Z"}`)
	})
	scoped := ItemScopedRequest{
		Type:            "customtype",
		MemoryStoreName: "store",
		Scope:           Scope{AppID: "app", TenantID: "tenant", AgentID: "agent", RunID: "run"},
	}
	if _, err := client.GetItem(context.Background(), &GetItemRequest{ItemScopedRequest: scoped, Path: "/a"}); err != nil {
		t.Fatal(err)
	}
}

// Nil requests must not panic: they degrade to an empty request (plus the
// default type) that the server rejects with a validation error.
func TestItemNilRequestDoesNotPanic(t *testing.T) {
	client := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["type"] != "memoryfile" {
			t.Errorf("type = %v, want memoryfile", body["type"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":"VALIDATION_ERROR","message":"path is required","requestId":"req-1"}`)
	})
	ctx := context.Background()
	calls := []func() error{
		func() error { _, err := client.AddItem(ctx, nil); return err },
		func() error { _, err := client.ListItems(ctx, nil); return err },
		func() error { _, err := client.GetItem(ctx, nil); return err },
		func() error { _, err := client.UpdateItem(ctx, nil); return err },
		func() error { _, err := client.DeleteItem(ctx, nil); return err },
		func() error { _, err := client.ListItemVersions(ctx, nil); return err },
		func() error { _, err := client.GetItemVersion(ctx, nil); return err },
		func() error { _, err := client.RedactItemVersion(ctx, nil); return err },
	}
	for i, call := range calls {
		err := call()
		serviceErr, ok := err.(*ServiceError)
		if !ok || serviceErr.Code != "VALIDATION_ERROR" {
			t.Errorf("call %d: err = %v, want VALIDATION_ERROR service error", i, err)
		}
	}
}
