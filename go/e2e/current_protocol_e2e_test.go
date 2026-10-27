package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	sdk "github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

// currentStore registers cleanup only for this run's unique Store and verifies
// deletion by reading it back. Existing stores and scopes are never mutated.
func currentStore(t *testing.T, c *sdk.Client, input string, outputs []string) string {
	t.Helper()
	name := fmt.Sprintf("gosdk_current_%d_%s", time.Now().Unix(), suffix(6))
	ctx := context.Background()
	description := "Go SDK current protocol test: " + name
	registerStoreCleanup(t, c, name, description)
	created, err := c.CreateMemoryStore(ctx, &sdk.CreateMemoryStoreRequest{MemoryStoreName: name, Description: description, InputType: input, OutputTypes: outputs, WikiReflectionEnabled: sdk.Bool(false)})
	if err != nil {
		t.Fatalf("CreateMemoryStore: %v", err)
	}
	if created.MemoryStoreName != name || created.InputType != input || len(created.OutputTypes) != len(outputs) {
		t.Fatalf("store contract: %+v", created)
	}
	t.Logf("created isolated store %s input=%s output=%v", name, created.InputType, created.OutputTypes)
	return name
}

func registerStoreCleanup(t *testing.T, c *sdk.Client, name string, allowedDescriptions ...string) {
	t.Helper()
	descriptions := map[string]bool{}
	for _, description := range allowedDescriptions {
		descriptions[description] = true
	}
	// Register before Create: an ambiguous create failure can still leave a
	// provisioned Store. Only delete a Store carrying this run's marker.
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			info, err := c.GetMemoryStore(cleanupCtx, &sdk.GetMemoryStoreRequest{MemoryStoreName: name})
			if status, _ := serviceStatus(err); status == 404 {
				t.Logf("cleanup verified: %s", name)
				return
			}
			if err != nil {
				lastErr = err
				continue
			}
			if !descriptions[info.Description] {
				t.Errorf("cleanup ownership marker mismatch for %s", name)
				return
			}
			_, lastErr = c.DeleteMemoryStore(cleanupCtx, &sdk.DeleteMemoryStoreRequest{MemoryStoreName: name})
			// Delete is idempotent and can resume after an upstream timeout.
			// Always read back before retrying the same unique Store.
			_, err = c.GetMemoryStore(cleanupCtx, &sdk.GetMemoryStoreRequest{MemoryStoreName: name})
			if status, _ := serviceStatus(err); status == 404 {
				t.Logf("cleanup verified: %s", name)
				return
			}
			if err != nil {
				lastErr = err
			}
		}
		t.Errorf("cleanup %s not verified: %v", name, lastErr)
	})
}

func TestCurrentFileProtocolE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputFile})
	scope := fullScope("currentfile")
	scoped := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: scope}
	item, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: scoped, Path: "/notes/source.md", Content: "Go SDK protocol validation: blue coffee cup.", View: sdk.ItemViewFull})
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if item.Content == nil || !strings.Contains(*item.Content, "blue coffee") || item.LatestVersionID == "" {
		t.Fatalf("item readback: %+v", item)
	}
	changed, err := c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Path: sdk.String("/notes/renamed.md"), Content: sdk.String("Updated blue coffee cup."), Precondition: &sdk.ItemPrecondition{ExpectedVersionID: item.LatestVersionID}, View: sdk.ItemViewFull})
	if err != nil {
		t.Fatalf("UpdateItem current protocol: %v", err)
	}
	if changed.Path != "/notes/renamed.md" || changed.LatestVersionID == item.LatestVersionID || changed.Content == nil || *changed.Content != "Updated blue coffee cup." {
		t.Fatalf("atomic update=%+v", changed)
	}
	_, err = c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Content: sdk.String("must not overwrite"), Precondition: &sdk.ItemPrecondition{ExpectedVersionID: item.LatestVersionID}})
	if s, code := serviceStatus(err); s != 409 {
		t.Fatalf("stale version: status=%d code=%s err=%v", s, code, err)
	} else {
		t.Logf("stale version rejected: %d %s", s, code)
	}
	legacy, err := c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, Path: sdk.String(changed.Path), NewPath: "/notes/legacy.md", Content: sdk.String("Legacy adapter blue coffee cup."), ExpectedSHA256: changed.ContentSHA256, View: sdk.ItemViewFull})
	if err != nil || legacy.Path != "/notes/legacy.md" {
		t.Fatalf("legacy adapter: result=%+v err=%v", legacy, err)
	}
	versions, err := c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: scoped, Path: legacy.Path, View: sdk.ItemViewFull, Limit: 20})
	if err != nil || len(versions.Versions) < 3 {
		t.Fatalf("versions: result=%+v err=%v", versions, err)
	}
	original, err := c.GetItemVersion(ctx, &sdk.GetItemVersionRequest{ItemScopedRequest: scoped, VersionID: item.LatestVersionID, View: sdk.ItemViewFull})
	if err != nil || original.Content == nil || *original.Content != *item.Content {
		t.Fatalf("immutable original: result=%+v err=%v", original, err)
	}
	t.Run("wiki binary", func(t *testing.T) {
		// Binary AddItem is supported only in the Wiki raw namespace.
		wikiScope := scoped
		wikiScope.Scope.RunID = ""
		// Encode a valid PNG rather than relying on an opaque fixture checksum.
		var buffer bytes.Buffer
		if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
			t.Fatal(err)
		}
		pngContent := base64.StdEncoding.EncodeToString(buffer.Bytes())
		binary, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: wikiScope, Path: "/.wiki/raw/pixel.png", ContentBase64: sdk.String(pngContent), MediaType: "image/png", View: sdk.ItemViewFull})
		if err != nil {
			t.Fatalf("binary AddItem: %v", err)
		}
		got, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: wikiScope, ItemID: binary.ItemID, View: sdk.ItemViewFull})
		if err != nil || got.ContentBase64 == nil || *got.ContentBase64 != pngContent || got.MediaType != "image/png" || got.Content != nil {
			t.Fatalf("binary roundtrip: result=%+v err=%v", got, err)
		}
		buffer.Reset()
		if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 1))); err != nil {
			t.Fatal(err)
		}
		updatedPNG := base64.StdEncoding.EncodeToString(buffer.Bytes())
		updatedBinary, err := c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: wikiScope, ItemID: binary.ItemID, ContentBase64: sdk.String(updatedPNG), MediaType: "image/png", Precondition: &sdk.ItemPrecondition{ExpectedVersionID: binary.LatestVersionID}, View: sdk.ItemViewFull})
		if err != nil || updatedBinary.LatestVersionID == binary.LatestVersionID || updatedBinary.ContentBase64 == nil || *updatedBinary.ContentBase64 != updatedPNG || updatedBinary.Content != nil {
			t.Fatalf("binary update: result=%+v err=%v", updatedBinary, err)
		}
		original, err := c.GetItemVersion(ctx, &sdk.GetItemVersionRequest{ItemScopedRequest: wikiScope, VersionID: binary.LatestVersionID, View: sdk.ItemViewFull})
		if err != nil || original.ContentBase64 == nil || *original.ContentBase64 != pngContent || original.Content != nil {
			t.Fatalf("immutable binary version: result=%+v err=%v", original, err)
		}
		if _, err := c.DeleteItem(ctx, &sdk.DeleteItemRequest{ItemScopedRequest: wikiScope, ItemID: binary.ItemID, Precondition: &sdk.ItemPrecondition{ExpectedVersionID: updatedBinary.LatestVersionID}}); err != nil {
			t.Fatal(err)
		}
		_, err = c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: wikiScope, ItemID: binary.ItemID})
		if status, _ := serviceStatus(err); status != 404 {
			t.Fatalf("deleted binary remains readable: %v", err)
		}
	})
	if _, err := c.DeleteItem(ctx, &sdk.DeleteItemRequest{ItemScopedRequest: scoped, ItemID: legacy.ItemID, Precondition: &sdk.ItemPrecondition{ExpectedVersionID: legacy.LatestVersionID}}); err != nil {
		t.Fatal(err)
	}
	_, err = c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: scoped, ItemID: legacy.ItemID})
	if s, _ := serviceStatus(err); s != 404 {
		t.Fatalf("deleted item readable: %v", err)
	}
	t.Run("history purge", func(t *testing.T) {
		versionScope := scoped
		versionScope.Scope.RunID = "*"
		remaining, err := c.ListItems(ctx, &sdk.ListItemsRequest{ItemScopedRequest: versionScope})
		if err != nil || len(remaining.Items) != 0 || remaining.NextToken != "" {
			t.Fatalf("history purge requires empty current items: %+v %v", remaining, err)
		}
		purged, err := c.DeleteItemVersions(ctx, &sdk.DeleteItemVersionsRequest{ItemScopedRequest: versionScope})
		if err != nil || purged.TaskID == "" || purged.Status != "pending" {
			t.Fatalf("DeleteItemVersions: result=%+v err=%v", purged, err)
		}
		poll(t, 90*time.Second, time.Second, "version deletion", func() (*sdk.ListItemVersionsResponse, error) {
			return c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: versionScope})
		}, func(r *sdk.ListItemVersionsResponse) bool { return len(r.Versions) == 0 && r.NextToken == "" })
	})
	t.Run("delete with versions", func(t *testing.T) {
		target := scoped
		target.Scope.AgentID += "_purge"
		if _, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: target, Path: "/remove.md", Content: "remove current and history"}); err != nil {
			t.Fatal(err)
		}
		target.Scope.RunID = "*"
		deleted, err := c.DeleteItem(ctx, &sdk.DeleteItemRequest{ItemScopedRequest: target, DeleteVersions: true})
		if err != nil || deleted.TaskID == "" || deleted.Status != "pending" {
			t.Fatalf("DeleteItem with versions: %+v %v", deleted, err)
		}
		poll(t, 90*time.Second, time.Second, "current item removal", func() (*sdk.ListItemsResponse, error) {
			return c.ListItems(ctx, &sdk.ListItemsRequest{ItemScopedRequest: target})
		}, func(r *sdk.ListItemsResponse) bool { return len(r.Items) == 0 && r.NextToken == "" })
		poll(t, 90*time.Second, time.Second, "combined history removal", func() (*sdk.ListItemVersionsResponse, error) {
			return c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: target})
		}, func(r *sdk.ListItemVersionsResponse) bool { return len(r.Versions) == 0 && r.NextToken == "" })
	})
	t.Run("wiki empty scope protocol", func(t *testing.T) {
		ws := sdk.WikiScopedRequest{MemoryStoreName: store, Scope: sdk.AgentScope{AppID: scope.AppID, TenantID: scope.TenantID, AgentID: "empty_wiki"}}
		wiki, err := c.GetWiki(ctx, &sdk.GetWikiRequest{WikiScopedRequest: ws})
		if err != nil {
			t.Fatal(err)
		}
		if wiki.Status == "" || wiki.PageCount != 0 {
			t.Fatalf("empty wiki=%+v", wiki)
		}
		_, err = c.UpdateWiki(ctx, &sdk.UpdateWikiRequest{WikiScopedRequest: ws, Name: sdk.String("SDK contract wiki"), Summary: sdk.String("protocol test"), CompileOptions: sdk.NullableOf(sdk.WikiCompileOptions{Language: "en", DocSummary: &sdk.WikiDocSummaryOptions{Enabled: false}})})
		if err != nil {
			t.Fatal(err)
		}
		wiki, err = c.GetWiki(ctx, &sdk.GetWikiRequest{WikiScopedRequest: ws})
		if err != nil || wiki.Name != "SDK contract wiki" || wiki.CompileOptions == nil || wiki.CompileOptions.Language != "en" || wiki.CompileOptions.DocSummary == nil || wiki.CompileOptions.DocSummary.Enabled {
			t.Fatalf("Wiki options readback: %+v %v", wiki, err)
		}
		_, err = c.UpdateWiki(ctx, &sdk.UpdateWikiRequest{WikiScopedRequest: ws, Summary: sdk.String(""), CompileOptions: sdk.Null[sdk.WikiCompileOptions]()})
		if err != nil {
			t.Fatal(err)
		}
		wiki, err = c.GetWiki(ctx, &sdk.GetWikiRequest{WikiScopedRequest: ws})
		if err != nil || wiki.Summary != "" || wiki.CompileOptions != nil {
			t.Fatalf("Wiki reset: %+v %v", wiki, err)
		}
		// No build has ever been published: page APIs return NotFound, and
		// Ingest rejects a source-free first build (it is not a no-change skip).
		_, err = c.ListWikiPages(ctx, &sdk.ListWikiPagesRequest{WikiScopedRequest: ws})
		if status, _ := serviceStatus(err); status != 404 {
			t.Fatalf("unpublished pages: %v", err)
		}
		_, err = c.SearchWikiPages(ctx, &sdk.SearchWikiPagesRequest{WikiScopedRequest: ws, Query: "coffee", Limit: 3})
		if status, _ := serviceStatus(err); status != 404 {
			t.Fatalf("unpublished search: %v", err)
		}
		_, err = c.GetWikiPage(ctx, &sdk.GetWikiPageRequest{WikiScopedRequest: ws, Path: "/missing.md"})
		if status, _ := serviceStatus(err); status != 404 {
			t.Fatalf("missing page: %v", err)
		}
		_, err = c.IngestWiki(ctx, &sdk.IngestWikiRequest{WikiScopedRequest: ws, ClientToken: "go-sdk-empty-wiki"})
		if status, _ := serviceStatus(err); status != 400 {
			t.Fatalf("empty source ingest must be rejected: %v", err)
		}
		t.Logf("six Wiki actions verified: metadata/options/reset readback, unpublished pages 404, no-source ingest 400")
	})
}

func TestCurrentMemoryProtocolE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputMessages, []string{sdk.StoreOutputStructured, sdk.StoreOutputFile})
	scope := fullScope("currentmemory")
	marker := "CobaltCafe" + suffix(6)
	added, err := c.AddMemories(ctx, &sdk.AddMemoriesRequest{MemoryStoreName: store, Scope: &scope, Text: "The user's favorite coffee shop is " + marker + " in Shanghai.", Sync: false, Interaction: &sdk.InteractionRef{SessionID: scope.RunID, TurnID: "turn-1"}, Metadata: map[string]string{"source": "go-sdk-current"}})
	if err != nil {
		t.Fatalf("AddMemories: %v", err)
	}
	if added.RequestID == "" || added.AcceptedMessages != 1 {
		t.Fatalf("add=%+v", added)
	}
	filter := &sdk.ScopeFilter{Op: "and", Conditions: []*sdk.ScopeFilter{{Op: "eq", Field: "appId", Value: sdk.String(scope.AppID)}, {Op: "eq", Field: "tenantId", Value: sdk.String(scope.TenantID)}}}
	listed := poll(t, 150*time.Second, 2*time.Second, "filtered memory visibility", func() (*sdk.ListMemoriesResponse, error) {
		return c.ListMemories(ctx, &sdk.ListMemoriesRequest{MemoryStoreName: store, Scope: &scope, Filter: filter, Limit: 20})
	}, func(r *sdk.ListMemoriesResponse) bool { return memoryListContains(r.Memories, marker) })
	for _, unit := range listed.Memories {
		if unit.Scope != scope {
			t.Fatalf("scope leak: %+v", unit.Scope)
		}
	}
	t.Logf("verified filtered memory contents (%d rows)", len(listed.Memories))
	scopes := poll(t, 90*time.Second, 2*time.Second, "filter-only scope directory", func() (*sdk.ListMemoryStoreScopesResponse, error) {
		return c.ListMemoryStoreScopes(ctx, &sdk.ListMemoryStoreScopesRequest{MemoryStoreName: store, Filter: filter})
	}, func(r *sdk.ListMemoryStoreScopesResponse) bool {
		for _, s := range r.Scopes {
			if s == scope {
				return true
			}
		}
		return false
	})
	for _, s := range scopes.Scopes {
		if s.AppID != scope.AppID || s.TenantID != scope.TenantID {
			t.Fatalf("scope filter leaked: %+v", s)
		}
	}
	t.Logf("verified filtered scope directory (%d rows)", len(scopes.Scopes))
	search := poll(t, 120*time.Second, 2*time.Second, "filtered search marker", func() (*sdk.SearchMemoriesResponse, error) {
		return c.SearchMemories(ctx, &sdk.SearchMemoriesRequest{MemoryStoreName: store, Scope: &scope, Filter: filter, Query: marker, TopK: 5, EnableRerank: sdk.Bool(false), Thresholds: &sdk.SearchThresholds{MinSimilarity: .01}})
	}, func(r *sdk.SearchMemoriesResponse) bool {
		for _, hit := range r.Results {
			if strings.Contains(hit.Unit.Text, marker) {
				return true
			}
		}
		return false
	})
	if search.Results[0].Unit.Scope != scope {
		t.Fatal("search scope mismatch")
	}
	t.Logf("verified filtered memory search (%d hits)", len(search.Results))
	files := poll(t, 150*time.Second, 2*time.Second, "managed file search", func() (*sdk.SearchItemsResponse, error) {
		return c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: scope}, Filter: filter, Query: marker, TopK: 5, Thresholds: &sdk.SearchThresholds{MinScore: .01}})
	}, func(r *sdk.SearchItemsResponse) bool {
		for _, hit := range r.Items {
			if hit.ItemID != "" && hit.LatestVersionID != "" && hit.Scope == scope {
				return true
			}
		}
		return false
	})
	// Summaries may omit literal facts. Verify the retrieved file body rather
	// than requiring a model-generated summary to repeat the source marker.
	fileContainsMarker := false
	for _, hit := range files.Items {
		if hit.Scope != scope {
			t.Fatalf("file search scope leak: %+v", hit.Scope)
		}
		item, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: scope}, ItemID: hit.ItemID, View: sdk.ItemViewFull})
		if err != nil {
			t.Fatal(err)
		}
		if item.Content != nil && strings.Contains(*item.Content, marker) {
			fileContainsMarker = true
		}
	}
	if !fileContainsMarker {
		t.Fatal("retrieved managed files lost the source marker")
	}
	task := poll(t, 90*time.Second, 2*time.Second, "ingest terminal task", func() (*sdk.GetMemoryTaskResponse, error) {
		return c.GetMemoryTask(ctx, &sdk.GetMemoryTaskRequest{MemoryStoreName: store, RequestID: added.RequestID, Scope: &scope})
	}, func(r *sdk.GetMemoryTaskResponse) bool {
		return r.Task.Status == sdk.MemoryTaskStatusCompleted || r.Task.Status == sdk.MemoryTaskStatusFailed
	})
	if task.Task.Status != sdk.MemoryTaskStatusCompleted {
		t.Fatalf("ingest failed: %+v", task.Task)
	}
	t.Run("item selectors", func(t *testing.T) {
		checkSearchItemParameters(t, c, sdk.ItemScopedRequest{MemoryStoreName: store, Scope: scope}, marker, files.Items[0].ItemID)
	})
	t.Logf("verified structured memory, filtered retrieval, managed file search, and ingestion task %s", added.RequestID)
	t.Run("dream", func(t *testing.T) {
		created, err := c.CreateMemoryDreamTask(ctx, &sdk.CreateMemoryDreamTaskRequest{MemoryStoreName: store, Scopes: []sdk.Scope{scope}, Effort: sdk.DreamEffortLow, ApplyMode: sdk.DreamApplyModeProposal, MaxMemories: 10, MaxMessages: 10, ClientToken: "go-sdk-current-dream"})
		if err != nil {
			t.Fatal(err)
		}
		dream := poll(t, 180*time.Second, 2*time.Second, "Dream terminal state", func() (*sdk.GetMemoryDreamTaskResponse, error) {
			return c.GetMemoryDreamTask(ctx, &sdk.GetMemoryDreamTaskRequest{MemoryStoreName: store, DreamID: created.DreamID})
		}, func(r *sdk.GetMemoryDreamTaskResponse) bool {
			return r.Status == sdk.DreamTaskStatusCompleted || r.Status == sdk.DreamTaskStatusCompletedWithFailures || r.Status == sdk.DreamTaskStatusFailed
		})
		if dream.Status != sdk.DreamTaskStatusCompleted {
			t.Fatalf("Dream did not complete cleanly: status=%s failure=%s error=%s", dream.Status, dream.FailureCode, dream.LastError)
		}
		if dream.Input.MemoryCount == 0 {
			t.Fatal("Dream completed without reading the test memory")
		}
		tasks, err := c.ListMemoryDreamTasks(ctx, &sdk.ListMemoryDreamTasksRequest{MemoryStoreName: store, Scope: scope})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, task := range tasks.Tasks {
			if task.DreamID == created.DreamID {
				found = true
				if task.Effort != sdk.DreamEffortLow || task.Actions.Total != dream.Actions.Total || task.Input.MemoryCount != dream.Input.MemoryCount {
					t.Fatalf("List/Get Dream mismatch: %+v vs %+v", task, dream)
				}
			}
		}
		if !found {
			t.Fatal("Dream missing from list")
		}
		actions, err := c.ListMemoryDreamActions(ctx, &sdk.ListMemoryDreamActionsRequest{MemoryStoreName: store, DreamID: created.DreamID})
		if err != nil {
			t.Fatal(err)
		}
		if len(actions.Actions) != dream.Actions.Total {
			t.Fatalf("action count mismatch: rows=%d summary=%d", len(actions.Actions), dream.Actions.Total)
		}
		t.Logf("Dream %s: input memories=%d actions=%d", created.DreamID, dream.Input.MemoryCount, dream.Actions.Total)
	})
}

// Structured-only stores support synchronous extraction and return memory IDs.
func TestCurrentStructuredMemoryProtocolE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputMessages, []string{sdk.StoreOutputStructured})
	scope := fullScope("sync")
	marker := "SapphireCafe" + suffix(6)
	added, err := c.AddMemories(ctx, &sdk.AddMemoriesRequest{MemoryStoreName: store, Scope: &scope, Text: "The user's favorite coffee shop is " + marker + " in Shanghai.", Sync: true, Metadata: map[string]string{"source": "go-sdk-sync"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.MemoryIDs) == 0 || added.UnitsCreated == 0 {
		t.Fatalf("sync extraction produced no IDs: %+v", added)
	}
	found := false
	for _, id := range added.MemoryIDs {
		got, err := c.GetMemory(ctx, &sdk.GetMemoryRequest{MemoryStoreName: store, MemoryID: id})
		if err != nil {
			t.Fatal(err)
		}
		if got.Memory.Scope != scope {
			t.Fatalf("ID lookup scope mismatch: %+v", got)
		}
		found = found || strings.Contains(got.Memory.Text, marker)
		_, err = c.UpdateMemory(ctx, &sdk.UpdateMemoryRequest{MemoryStoreName: store, MemoryID: id, Metadata: map[string]string{}})
		if err != nil {
			t.Fatal(err)
		}
		got, err = c.GetMemory(ctx, &sdk.GetMemoryRequest{MemoryStoreName: store, MemoryID: id})
		if err != nil || len(got.Memory.Metadata) != 0 {
			t.Fatalf("metadata clear: %+v err=%v", got, err)
		}
		_, err = c.DeleteMemory(ctx, &sdk.DeleteMemoryRequest{MemoryStoreName: store, MemoryID: id})
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.GetMemory(ctx, &sdk.GetMemoryRequest{MemoryStoreName: store, MemoryID: id})
		if status, _ := serviceStatus(err); status != 404 {
			t.Fatalf("deleted memory still visible: %v", err)
		}
	}
	if !found {
		t.Fatal("synchronous extraction lost the source marker")
	}
	t.Logf("verified synchronous IDs, ID-only Get/Update/Delete, metadata clear, and deletion readback")
}
