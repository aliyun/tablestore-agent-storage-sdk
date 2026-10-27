package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	sdk "github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

// Explicit opt-in: this contract boundary requires 1001 real conversations.
// It verifies scope resolution, not 1000-conversation retrieval throughput.
func TestReleaseSearchScopeBoundaryE2E(t *testing.T) {
	if os.Getenv("MEMORY_E2E_LARGE_SCOPE") != "1" {
		t.Skip("set MEMORY_E2E_LARGE_SCOPE=1 to create 1001 isolated conversations")
	}
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputMessages, []string{sdk.StoreOutputFile})
	base := fullScope("boundary")
	anchor := base
	anchor.RunID = "scope_0000"
	marker := "BoundaryCafe" + suffix(6)
	added, err := c.AddMemories(ctx, &sdk.AddMemoriesRequest{MemoryStoreName: store, Scope: &anchor, Text: "The user visits " + marker + " every Saturday. It is their favorite coffee shop, where they order latte."})
	if err != nil {
		t.Fatal(err)
	}
	task := poll(t, 3*time.Minute, 2*time.Second, "boundary anchor generation", func() (*sdk.GetMemoryTaskResponse, error) {
		return c.GetMemoryTask(ctx, &sdk.GetMemoryTaskRequest{MemoryStoreName: store, RequestID: added.RequestID, Scope: &anchor})
	}, func(r *sdk.GetMemoryTaskResponse) bool {
		return r.Task.Status == sdk.MemoryTaskStatusCompleted || r.Task.Status == sdk.MemoryTaskStatusFailed
	})
	if task.Task.Status != sdk.MemoryTaskStatusCompleted {
		t.Fatalf("boundary anchor failed: %+v", task.Task)
	}
	wild := base
	wild.RunID = "*"
	register := func(n int) {
		t.Helper()
		scope := base
		scope.RunID = fmt.Sprintf("scope_%04d", n)
		// One request per scope; never retry an ambiguous mutation. A small
		// greeting is sufficient to register the conversation directory entry.
		r, err := c.AddMemories(ctx, &sdk.AddMemoriesRequest{MemoryStoreName: store, Scope: &scope, Text: "Hello."})
		if err != nil || r.RequestID == "" || r.AcceptedMessages != 1 {
			t.Fatalf("scope %d admission: %+v %v", n, r, err)
		}
		if n%100 == 0 {
			t.Logf("registered %d isolated conversations", n+1)
		}
		time.Sleep(100 * time.Millisecond)
	}
	countScopes := func() (int, error) {
		seen := map[sdk.Scope]bool{}
		tokens := map[string]bool{}
		token := ""
		for {
			// An unfiltered directory list can read the primary table before
			// the async index used by SearchItems sees the latest conversation.
			// This redundant filter forces the same scope index for readiness.
			r, err := c.ListMemoryStoreScopes(ctx, &sdk.ListMemoryStoreScopesRequest{MemoryStoreName: store, Scope: wild, Filter: &sdk.ScopeFilter{Op: "eq", Field: "appId", Value: sdk.String(base.AppID)}, Limit: 100, NextToken: token})
			if err != nil {
				return 0, err
			}
			for _, scope := range r.Scopes {
				if scope.AppID != base.AppID || scope.TenantID != base.TenantID || scope.AgentID != base.AgentID {
					return 0, fmt.Errorf("scope directory escaped test selection")
				}
				seen[scope] = true
			}
			token = r.NextToken
			if token == "" {
				return len(seen), nil
			}
			if tokens[token] {
				return 0, fmt.Errorf("scope cursor repeated")
			}
			tokens[token] = true
		}
	}
	assertAnchor := func(t *testing.T, selector sdk.Scope, filter *sdk.ScopeFilter) {
		t.Helper()
		poll(t, 90*time.Second, 3*time.Second, "boundary query retrieves original anchor", func() (*sdk.SearchItemsResponse, error) {
			return c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: selector}, Filter: filter, Query: marker + " favorite coffee shop latte", TopK: 50})
		}, func(r *sdk.SearchItemsResponse) bool {
			for _, hit := range r.Items {
				if hit.Scope.AppID != base.AppID || hit.Scope.TenantID != base.TenantID || hit.Scope.AgentID != base.AgentID {
					t.Fatalf("boundary query leaked outside selection: %+v", hit.Scope)
				}
				if hit.Scope != anchor {
					continue
				}
				item, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: anchor}, ItemID: hit.ItemID, View: sdk.ItemViewFull})
				if err != nil {
					t.Fatal(err)
				}
				if item.Content != nil && strings.Contains(*item.Content, marker) {
					return true
				}
				if item.Content != nil {
					t.Logf("anchor candidate has no exact fixture marker: item=%s content=%q", hit.ItemID, *item.Content)
				}
			}
			return false
		})
	}
	// Establish retrieval before expanding the directory: a completed task alone
	// cannot prove the anchor fact was actually published/indexed.
	assertAnchor(t, anchor, nil)
	for i := 1; i < 1000; i++ {
		register(i)
	}
	poll(t, 3*time.Minute, 3*time.Second, "exactly 1000 conversations indexed", countScopes, func(n int) bool { return n == 1000 })
	t.Run("exactly_1000", func(t *testing.T) { assertAnchor(t, wild, nil) })
	register(1000)
	poll(t, 3*time.Minute, 3*time.Second, "exactly 1001 conversations indexed", countScopes, func(n int) bool { return n == 1001 })
	t.Run("over_1000_rejected", func(t *testing.T) {
		var overflowErr error
		poll(t, 90*time.Second, 3*time.Second, "indexed scope overflow is rejected", func() (bool, error) {
			// Match the successful 1000-scope query exactly; only the indexed
			// scope count changed. The gateway can redact the backend message
			// to "invalid parameter", so assert its public error identity.
			_, overflowErr = c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: wild}, Query: marker + " favorite coffee shop latte", TopK: 50})
			return overflowErr != nil, nil
		}, func(rejected bool) bool { return rejected })
		expectItemValidation(t, overflowErr)
	})
	t.Run("filter_narrows_1001", func(t *testing.T) {
		assertAnchor(t, wild, &sdk.ScopeFilter{Op: "eq", Field: "runId", Value: sdk.String(anchor.RunID)})
	})
	if !t.Failed() {
		t.Log("1000-scope retrieval, 1001-scope rejection and filter narrowing verified")
	}
}
