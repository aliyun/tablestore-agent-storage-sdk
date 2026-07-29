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
	itemJSON := `{"type":"memoryfile","itemId":"m1","path":"/notes/a.md","content":"hello","contentSha256":"sha-new","contentSizeBytes":5,"latestSeq":2,"createdAt":"2026-07-16T01:00:00Z","updatedAt":"2026-07-16T02:00:00Z"}`
	versionJSON := `{"type":"memoryfile","versionId":"v2","itemId":"m1","versionSeq":2,"operation":"modified","path":"/notes/a.md","content":"hello","contentSha256":"sha-new","contentSizeBytes":5,"sessionId":"session-1","createdAt":"2026-07-16T02:00:00Z"}`

	tests := []struct {
		name     string
		action   string
		wantBody map[string]any
		response string
		call     func(*Client) error
	}{
		{
			name:   "add",
			action: "AddItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"path": "/notes/a.md", "content": "hello", "sessionId": "session-1",
			},
			response: itemJSON,
			call: func(client *Client) error {
				req := &AddItemRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md", Content: "hello", SessionID: "session-1",
				}
				resp, err := client.AddItem(context.Background(), req)
				if err == nil && (resp.Type != ItemTypeMemoryFile || resp.ItemID != "m1" || resp.Content == nil || *resp.Content != "hello") {
					return fmt.Errorf("response = %+v", resp)
				}
				if req.Type != "" {
					return fmt.Errorf("caller request mutated: Type = %q", req.Type)
				}
				return err
			},
		},
		{
			name:   "update content",
			action: "UpdateItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"path": "/notes/a.md", "content": "updated", "expectedSha256": "sha-old", "sessionId": "session-1",
			},
			response: itemJSON,
			call: func(client *Client) error {
				_, err := client.UpdateItem(context.Background(), &UpdateItemRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md", Content: String("updated"),
					ExpectedSHA256: "sha-old", SessionID: "session-1",
				})
				return err
			},
		},
		{
			name:   "update rename",
			action: "UpdateItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"path": "/notes/a.md", "newPath": "/notes/b.md", "overwrite": true,
				"expectedSha256": "sha-new", "sessionId": "session-1",
			},
			response: itemJSON,
			call: func(client *Client) error {
				_, err := client.UpdateItem(context.Background(), &UpdateItemRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md", NewPath: "/notes/b.md",
					Overwrite: true, ExpectedSHA256: "sha-new", SessionID: "session-1",
				})
				return err
			},
		},
		{
			name:   "delete",
			action: "DeleteItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"path": "/notes/a.md", "expectedSha256": "sha-new", "sessionId": "session-1",
			},
			response: `{"type":"memoryfile"}`,
			call: func(client *Client) error {
				resp, err := client.DeleteItem(context.Background(), &DeleteItemRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md",
					ExpectedSHA256: "sha-new", SessionID: "session-1",
				})
				if err == nil && resp.Type != ItemTypeMemoryFile {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "get",
			action: "GetItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope, "path": "/notes/a.md",
			},
			response: itemJSON,
			call: func(client *Client) error {
				resp, err := client.GetItem(context.Background(), &GetItemRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md",
				})
				if err == nil && (resp.ContentSHA256 != "sha-new" || resp.Content == nil || *resp.Content != "hello") {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "get without content",
			action: "GetItem",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"path": "/notes/a.md", "includeContent": false,
			},
			response: `{"type":"memoryfile","itemId":"m1","path":"/notes/a.md","contentSha256":"sha-new","contentSizeBytes":5,"latestSeq":2,"createdAt":"2026-07-16T01:00:00Z","updatedAt":"2026-07-16T02:00:00Z"}`,
			call: func(client *Client) error {
				resp, err := client.GetItem(context.Background(), &GetItemRequest{
					ItemScopedRequest: scoped, Path: "/notes/a.md", IncludeContent: Bool(false),
				})
				if err == nil && (resp.ContentSizeBytes != 5 || resp.Content != nil) {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "list",
			action: "ListItems",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"pathPrefix": "/notes/", "nextToken": "next-1", "limit": float64(20),
			},
			response: `{"type":"memoryfile","items":[{"itemId":"m1","path":"/notes/a.md","contentSha256":"sha-new","contentSizeBytes":5,"latestSeq":2,"createdAt":"2026-07-16T01:00:00Z","updatedAt":"2026-07-16T02:00:00Z"}],"nextToken":"next-2","readOnly":true}`,
			call: func(client *Client) error {
				resp, err := client.ListItems(context.Background(), &ListItemsRequest{
					ItemScopedRequest: scoped, PathPrefix: "/notes/", NextToken: "next-1", Limit: 20,
				})
				if err == nil && (resp.Type != ItemTypeMemoryFile || len(resp.Items) != 1 || resp.Items[0].ItemID != "m1" || resp.NextToken != "next-2" || !resp.ReadOnly) {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "list versions",
			action: "ListItemVersions",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"itemId": "m1", "path": "/notes/a.md",
				"nextToken": "versions-1", "limit": float64(10), "operation": "modified",
			},
			response: `{"type":"memoryfile","versions":[{"versionId":"v2","itemId":"m1","versionSeq":2,"operation":"modified","path":"/notes/a.md","contentSha256":"sha-new","contentSizeBytes":5,"sessionId":"session-1","createdAt":"2026-07-16T02:00:00Z"}],"nextToken":"versions-2"}`,
			call: func(client *Client) error {
				resp, err := client.ListItemVersions(context.Background(), &ListItemVersionsRequest{
					ItemScopedRequest: scoped, ItemID: "m1", Path: "/notes/a.md",
					NextToken: "versions-1", Limit: 10, Operation: ItemOperationModified,
				})
				if err == nil && (len(resp.Versions) != 1 || resp.Versions[0].Operation != ItemOperationModified || resp.NextToken != "versions-2") {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
			},
		},
		{
			name:   "get version",
			action: "GetItemVersion",
			wantBody: map[string]any{
				"type": "memoryfile", "memoryStoreName": "store", "scope": wantScope,
				"itemId": "m1", "versionId": "v2", "versionSeq": float64(2),
			},
			response: versionJSON,
			call: func(client *Client) error {
				resp, err := client.GetItemVersion(context.Background(), &GetItemVersionRequest{
					ItemScopedRequest: scoped, ItemID: "m1", VersionID: "v2", VersionSeq: 2,
				})
				if err == nil && (resp.VersionID != "v2" || resp.ItemID != "m1" || resp.Content == nil || *resp.Content != "hello") {
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
				"itemId": "m1", "versionId": "v2", "versionSeq": float64(2), "sessionId": "redactor-1",
			},
			response: `{"type":"memoryfile","versionId":"v2","itemId":"m1","versionSeq":2,"operation":"modified","path":"","contentSha256":"","contentSizeBytes":0,"sessionId":"session-1","createdAt":"2026-07-16T02:00:00Z","redacted":true,"redactedAt":"2026-07-18T03:00:00Z","redactedBy":"redactor-1"}`,
			call: func(client *Client) error {
				resp, err := client.RedactItemVersion(context.Background(), &RedactItemVersionRequest{
					ItemScopedRequest: scoped, ItemID: "m1", VersionID: "v2", VersionSeq: 2, SessionID: "redactor-1",
				})
				if err == nil && (!resp.Redacted || resp.RedactedAt != "2026-07-18T03:00:00Z" || resp.RedactedBy != "redactor-1" || resp.Content != nil || resp.Path != "") {
					return fmt.Errorf("response = %+v", resp)
				}
				return err
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
				if !reflect.DeepEqual(body, test.wantBody) {
					t.Errorf("body = %#v, want %#v", body, test.wantBody)
				}
				w.Header().Set("Content-Type", "application/json")
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
		fmt.Fprint(w, `{"type":"customtype","itemId":"m1","path":"/a","contentSha256":"s","contentSizeBytes":1,"latestSeq":1,"createdAt":"2026-07-16T01:00:00Z","updatedAt":"2026-07-16T01:00:00Z"}`)
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
