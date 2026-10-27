package e2e

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

// Observe the public gateway header without consuming the body or bypassing
// SDK request signing, Content-MD5, or response signature validation.
type extractionTrace struct{ requestID string }
type extractionTraceKey struct{}
type extractionTransport struct{}

func (extractionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if trace, ok := req.Context().Value(extractionTraceKey{}).(*extractionTrace); ok && resp != nil {
		trace.requestID = resp.Header.Get("x-ots-requestid")
	}
	return resp, err
}

func extractionClient(t *testing.T) *sdk.Client {
	t.Helper()
	return e2eClientWithHTTPClient(t, &http.Client{Transport: extractionTransport{}, Timeout: 60 * time.Second})
}

func extractionContext() (context.Context, *extractionTrace) {
	trace := &extractionTrace{}
	return context.WithValue(context.Background(), extractionTraceKey{}, trace), trace
}

func extractionTasks(t *testing.T, c *sdk.Client, scoped sdk.ItemScopedRequest) []sdk.MemoryTask {
	t.Helper()
	var tasks []sdk.MemoryTask
	token := ""
	seen := map[string]bool{}
	for {
		resp, err := c.ListMemoryTasks(context.Background(), &sdk.ListMemoryTasksRequest{MemoryStoreName: scoped.MemoryStoreName, Scope: scoped.Scope, Limit: 100, NextToken: token})
		if err != nil {
			t.Fatalf("ListMemoryTasks: %v", err)
		}
		for _, task := range resp.Tasks {
			if task.Scope != scoped.Scope || task.MemoryStoreName != scoped.MemoryStoreName || task.EventType != "file-convert" || task.RequestID == "" {
				t.Fatalf("wrong conversion task identity: %+v", task)
			}
			tasks = append(tasks, task)
		}
		token = resp.NextToken
		if token == "" {
			return tasks
		}
		if seen[token] {
			t.Fatal("task pagination did not advance")
		}
		seen[token] = true
	}
}

func waitExtractionTask(t *testing.T, c *sdk.Client, scoped sdk.ItemScopedRequest, trace *extractionTrace) {
	t.Helper()
	if trace.requestID == "" {
		t.Fatal("mutation response has no gateway request ID")
	}
	deadline := time.Now().Add(3 * time.Minute)
	var last sdk.MemoryTask
	for time.Now().Before(deadline) {
		for _, task := range extractionTasks(t, c, scoped) {
			if task.RequestID != trace.requestID {
				continue
			}
			last = task
			switch task.Status {
			case sdk.MemoryTaskStatusCompleted:
				t.Logf("conversion completed requestId=%s", trace.requestID)
				return
			case sdk.MemoryTaskStatusFailed, sdk.MemoryTaskStatusNeedsReconcile:
				t.Fatalf("conversion terminal failure requestId=%s status=%s error=%s primaryError=%s", trace.requestID, task.Status, task.LastError, task.PrimaryError)
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("conversion not completed requestId=%s last=%+v", trace.requestID, last)
}

func extractionMemories(c *sdk.Client, scoped sdk.ItemScopedRequest) ([]sdk.MemoryUnit, error) {
	var units []sdk.MemoryUnit
	token := ""
	seen := map[string]bool{}
	for {
		resp, err := c.ListMemories(context.Background(), &sdk.ListMemoriesRequest{MemoryStoreName: scoped.MemoryStoreName, Scope: &scoped.Scope, Limit: 100, NextToken: token})
		if err != nil {
			return nil, err
		}
		units = append(units, resp.Memories...)
		token = resp.NextToken
		if token == "" {
			return units, nil
		}
		if seen[token] {
			return nil, fmt.Errorf("memory pagination did not advance")
		}
		seen[token] = true
	}
}

// Structured-only conversion uses the legacy source reader and does not bind
// a public version reference (server bindConvertSourceVersion returns early).
// Always require file ID/path/sequence. Retained published sources additionally
// require a complete public version reference; never relax that assertion.
func matchesExtractionSource(unit sdk.MemoryUnit, item *sdk.Item, retainedSource bool) bool {
	m := unit.Metadata
	if m["source_kind"] != "file" || m["source_file_id"] != item.ItemID || m["source_file_path"] != item.Path || m["source_file_version_seq"] != strconv.FormatInt(item.LatestSeq, 10) {
		return false
	}
	if retainedSource || m["source_file_version_id"] != "" || m["source_file_store_uid"] != "" {
		return m["source_file_version_id"] == item.LatestVersionID && m["source_file_store_uid"] != ""
	}
	return true
}

func waitDerived(t *testing.T, c *sdk.Client, scoped sdk.ItemScopedRequest, item *sdk.Item, fragment string, retainedSource bool) []sdk.MemoryUnit {
	t.Helper()
	units := poll(t, 90*time.Second, 2*time.Second, "current derived memory and provenance", func() ([]sdk.MemoryUnit, error) { return extractionMemories(c, scoped) }, func(units []sdk.MemoryUnit) bool {
		found := false
		for _, u := range units {
			if u.Scope != scoped.Scope || u.Deleted {
				return false
			}
			if u.Metadata["source_file_id"] != item.ItemID {
				continue
			}
			if !matchesExtractionSource(u, item, retainedSource) {
				return false
			}
			if strings.Contains(u.Text, fragment) {
				found = true
			}
		}
		return found
	})
	var derived []sdk.MemoryUnit
	for _, u := range units {
		if u.Metadata["source_file_id"] != item.ItemID {
			continue
		}
		got, err := c.GetMemory(context.Background(), &sdk.GetMemoryRequest{MemoryStoreName: scoped.MemoryStoreName, Scope: scoped.Scope, MemoryID: u.ID})
		if err != nil {
			t.Fatalf("GetMemory derived row: %v", err)
		}
		if got.Memory.ID != u.ID || got.Memory.Text != u.Text || !matchesExtractionSource(got.Memory, item, retainedSource) {
			t.Fatalf("derived persisted row mismatch: %+v", got)
		}
		derived = append(derived, u)
	}
	return derived
}

func searchDerived(t *testing.T, c *sdk.Client, scoped sdk.ItemScopedRequest, item *sdk.Item, fragment string, evidence bool) {
	t.Helper()
	resp := poll(t, 90*time.Second, 2*time.Second, "search derived memory", func() (*sdk.SearchMemoriesResponse, error) {
		return c.SearchMemories(context.Background(), &sdk.SearchMemoriesRequest{MemoryStoreName: scoped.MemoryStoreName, Scope: &scoped.Scope, Query: fragment, Metadata: map[string]string{"source_file_id": item.ItemID}, TopK: 20, EnableRerank: sdk.Bool(false), IncludeEvidence: true})
	}, func(resp *sdk.SearchMemoriesResponse) bool {
		found := false
		for _, h := range resp.Results {
			if h.Unit.Scope != scoped.Scope || !matchesExtractionSource(h.Unit, item, evidence) {
				return false
			}
			if strings.Contains(h.Unit.Text, fragment) {
				found = true
			}
		}
		return found
	})
	foundEvidence := false
	for _, h := range resp.Evidence {
		if h.Unit.Scope != scoped.Scope || !matchesExtractionSource(h.Unit, item, evidence) {
			t.Fatalf("wrong evidence provenance: %+v", h)
		}
		if strings.Contains(h.Unit.Text, fragment) {
			foundEvidence = true
		}
	}
	if evidence && !foundEvidence {
		t.Fatalf("source evidence missing: %+v", resp)
	}
	if !evidence && len(resp.Evidence) != 0 {
		t.Fatalf("purged source must not expose evidence: %+v", resp.Evidence)
	}
}

func addExtractionFile(t *testing.T, c *sdk.Client, scoped sdk.ItemScopedRequest, path, content string) *sdk.Item {
	t.Helper()
	ctx, trace := extractionContext()
	item, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: scoped, Path: path, Content: content, View: sdk.ItemViewFull})
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if item.Content == nil || *item.Content != content || item.LatestVersionID == "" {
		t.Fatalf("source write response: %+v", item)
	}
	t.Logf("source saved itemId=%s versionId=%s requestId=%s", item.ItemID, item.LatestVersionID, trace.requestID)
	waitExtractionTask(t, c, scoped, trace)
	return item
}

func TestFileStructuredExtractionE2E(t *testing.T) {
	c := extractionClient(t)
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputFile, sdk.StoreOutputStructured})
	scoped := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: fullScope("extract")}
	ctx := context.Background()
	oldFact := "蓝鲸项目的交付暗号是松林七号。"
	newFact := "蓝鲸项目的交付暗号是琥珀九号。"
	var item *sdk.Item
	var latestDerived []sdk.MemoryUnit
	if !t.Run("add_extract_retrieve_and_source_evidence", func(t *testing.T) {
		item = addExtractionFile(t, c, scoped, "/notes/project.md", oldFact)
		waitDerived(t, c, scoped, item, "松林七号", true)
		searchDerived(t, c, scoped, item, "松林七号", true)
		got, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, View: sdk.ItemViewFull})
		if err != nil || got.Content == nil || *got.Content != oldFact {
			t.Fatalf("retained source: %+v %v", got, err)
		}
	}) {
		return
	}
	original := *item
	if !t.Run("content_update_replaces_old_derivation", func(t *testing.T) {
		mctx, trace := extractionContext()
		updated, err := c.UpdateItem(mctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Content: &newFact, View: sdk.ItemViewFull})
		if err != nil {
			t.Fatal(err)
		}
		if updated.LatestVersionID == item.LatestVersionID || updated.LatestSeq <= item.LatestSeq {
			t.Fatal("content change did not append a version")
		}
		waitExtractionTask(t, c, scoped, trace)
		item = updated
		waitDerived(t, c, scoped, item, "琥珀九号", true)
		searchDerived(t, c, scoped, item, "琥珀九号", true)
		old, err := c.GetItemVersion(ctx, &sdk.GetItemVersionRequest{ItemScopedRequest: scoped, VersionID: original.LatestVersionID, View: sdk.ItemViewFull})
		if err != nil || old.Content == nil || *old.Content != oldFact {
			t.Fatalf("old source version: %+v %v", old, err)
		}
	}) {
		return
	}
	if !t.Run("rename_refreshes_provenance", func(t *testing.T) {
		mctx, trace := extractionContext()
		renamed, err := c.UpdateItem(mctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Path: sdk.String("/archive/project.md"), View: sdk.ItemViewFull})
		if err != nil {
			t.Fatal(err)
		}
		if renamed.Content == nil || *renamed.Content != newFact || renamed.LatestVersionID == item.LatestVersionID {
			t.Fatalf("rename: %+v", renamed)
		}
		waitExtractionTask(t, c, scoped, trace)
		item = renamed
		latestDerived = waitDerived(t, c, scoped, item, "琥珀九号", true)
		searchDerived(t, c, scoped, item, "琥珀九号", true)
	}) {
		return
	}
	if !t.Run("identical_update_has_no_version_or_task", func(t *testing.T) {
		before := extractionTasks(t, c, scoped)
		versions, err := c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		mctx, trace := extractionContext()
		same, err := c.UpdateItem(mctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Path: &item.Path, Content: &newFact})
		if err != nil {
			t.Fatal(err)
		}
		if same.LatestVersionID != item.LatestVersionID || same.LatestSeq != item.LatestSeq || trace.requestID == "" {
			t.Fatalf("no-op changed version or omitted request ID: %+v", same)
		}
		// Submission is synchronous. Repeat reads also catch delayed task visibility.
		for i := 0; i < 3; i++ {
			tasks := extractionTasks(t, c, scoped)
			if len(tasks) != len(before) {
				t.Fatalf("no-op added task: before=%d after=%d", len(before), len(tasks))
			}
			for _, task := range tasks {
				if task.RequestID == trace.requestID {
					t.Fatal("no-op submitted conversion")
				}
			}
			if i < 2 {
				time.Sleep(2 * time.Second)
			}
		}
		after, err := c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Limit: 100})
		if err != nil || len(after.Versions) != len(versions.Versions) || versions.NextToken != "" || after.NextToken != "" {
			t.Fatalf("no-op changed history: %+v %v", after, err)
		}
	}) {
		return
	}
	t.Run("delete_cleans_derived_memories_and_preserves_other_sources", func(t *testing.T) {
		guard := addExtractionFile(t, c, scoped, "/notes/guard.md", "海豚项目的交付暗号是银杏三号。")
		waitDerived(t, c, scoped, guard, "银杏三号", true)
		otherScope := scoped
		otherScope.Scope.RunID += "_other"
		other := addExtractionFile(t, c, otherScope, "/notes/project.md", "白鹭项目的交付暗号是珊瑚五号。")
		waitDerived(t, c, otherScope, other, "珊瑚五号", true)
		mctx, trace := extractionContext()
		deleted, err := c.DeleteItem(mctx, &sdk.DeleteItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID})
		if err != nil || !deleted.Deleted {
			t.Fatalf("DeleteItem: %+v %v", deleted, err)
		}
		waitExtractionTask(t, c, scoped, trace)
		_, err = c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID})
		if status, _ := serviceStatus(err); status != 404 {
			t.Fatalf("deleted source: %v", err)
		}
		poll(t, 90*time.Second, 2*time.Second, "derived cleanup", func() ([]sdk.MemoryUnit, error) { return extractionMemories(c, scoped) }, func(units []sdk.MemoryUnit) bool {
			guardFound := false
			for _, u := range units {
				if u.Metadata["source_file_id"] == item.ItemID {
					return false
				}
				if u.Metadata["source_file_id"] == guard.ItemID {
					guardFound = true
				}
			}
			return guardFound
		})
		poll(t, 90*time.Second, 2*time.Second, "deleted derivation absent from search", func() (*sdk.SearchMemoriesResponse, error) {
			return c.SearchMemories(ctx, &sdk.SearchMemoriesRequest{MemoryStoreName: store, Scope: &scoped.Scope, Query: "琥珀九号", Metadata: map[string]string{"source_file_id": item.ItemID}, IncludeEvidence: true, EnableRerank: sdk.Bool(false)})
		}, func(r *sdk.SearchMemoriesResponse) bool { return len(r.Results) == 0 && len(r.Evidence) == 0 })
		for _, unit := range latestDerived {
			poll(t, 30*time.Second, time.Second, "deleted derived point read", func() (bool, error) {
				_, err := c.GetMemory(ctx, &sdk.GetMemoryRequest{MemoryStoreName: store, Scope: scoped.Scope, MemoryID: unit.ID})
				if status, _ := serviceStatus(err); status == 404 {
					return true, nil
				}
				return false, err
			}, func(absent bool) bool { return absent })
		}
		searchDerived(t, c, scoped, guard, "银杏三号", true)
		searchDerived(t, c, otherScope, other, "珊瑚五号", true)
		old, err := c.GetItemVersion(ctx, &sdk.GetItemVersionRequest{ItemScopedRequest: scoped, VersionID: original.LatestVersionID, View: sdk.ItemViewFull})
		if err != nil || old.Content == nil || *old.Content != oldFact {
			t.Fatalf("delete must retain historical source: %+v %v", old, err)
		}
	})
}

func TestFileStructuredOnlyExtractionE2E(t *testing.T) {
	c := extractionClient(t)
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputStructured})
	scoped := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: fullScope("extractonly")}
	item := addExtractionFile(t, c, scoped, "/notes/source.md", "北极星项目的交付暗号是紫杉六号。")
	waitDerived(t, c, scoped, item, "紫杉六号", false)
	searchDerived(t, c, scoped, item, "紫杉六号", false)
	t.Log("structured-only task completed and derived memory retrieved; public API gates cannot prove physical source/version/artifact deletion")
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"GetItem", func() error {
			_, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID})
			return err
		}},
		{"ListItems", func() error { _, err := c.ListItems(ctx, &sdk.ListItemsRequest{ItemScopedRequest: scoped}); return err }},
		{"SearchItems", func() error {
			_, err := c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: scoped, Query: "紫杉六号"})
			return err
		}},
		{"UpdateItem", func() error {
			_, err := c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Content: sdk.String("changed")})
			return err
		}},
		{"DeleteItem", func() error {
			_, err := c.DeleteItem(ctx, &sdk.DeleteItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID})
			return err
		}},
		{"ListItemVersions", func() error {
			_, err := c.ListItemVersions(ctx, &sdk.ListItemVersionsRequest{ItemScopedRequest: scoped, ItemID: item.ItemID})
			return err
		}},
		{"GetItemVersion", func() error {
			_, err := c.GetItemVersion(ctx, &sdk.GetItemVersionRequest{ItemScopedRequest: scoped, VersionID: item.LatestVersionID})
			return err
		}},
		{"DeleteItemVersions", func() error {
			ranged := scoped
			ranged.Scope.RunID = "*"
			_, err := c.DeleteItemVersions(ctx, &sdk.DeleteItemVersionsRequest{ItemScopedRequest: ranged})
			return err
		}},
		{"RedactItemVersion", func() error {
			_, err := c.RedactItemVersion(ctx, &sdk.RedactItemVersionRequest{ItemScopedRequest: scoped, VersionID: item.LatestVersionID})
			return err
		}},
	} {
		t.Run("source_api_forbidden/"+tc.name, func(t *testing.T) {
			err := tc.call()
			status, code := serviceStatus(err)
			if status != 400 || code != "OTSParameterInvalid" {
				t.Fatalf("expected store-type rejection, got %v", err)
			}
			var serviceErr *sdk.ServiceError
			if !errors.As(err, &serviceErr) || serviceErr.RequestID == "" {
				t.Fatalf("missing rejection request ID: %v", err)
			}
		})
	}
	// A new ID at the same path demonstrates path-index release, but is not
	// substituted for a direct physical-version/artifact readback.
	t.Run("source_path_reusable_and_memories_retained", func(t *testing.T) {
		second := addExtractionFile(t, c, scoped, item.Path, "南十字星项目的交付暗号是雪松八号。")
		if second.ItemID == item.ItemID {
			t.Fatal("new source reused purged file identity")
		}
		waitDerived(t, c, scoped, second, "雪松八号", false)
		searchDerived(t, c, scoped, second, "雪松八号", false)
		searchDerived(t, c, scoped, item, "紫杉六号", false)
	})
}
