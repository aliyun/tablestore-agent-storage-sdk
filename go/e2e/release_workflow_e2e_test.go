package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	sdk "github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

func TestReleaseMultiScopeItemsE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputMessages, []string{sdk.StoreOutputFile})
	base := fullScope("multiscope")
	scopes := []sdk.Scope{base, base, base}
	scopes[1].RunID += "_second"
	scopes[2].TenantID += "_excluded"
	markers := []string{"蓝松七号", "青杉八号", "红枫九号"}
	for i, scope := range scopes {
		added, err := c.AddMemories(ctx, &sdk.AddMemoriesRequest{MemoryStoreName: store, Scope: &scope, Text: "用户最喜欢的咖啡馆叫" + markers[i] + "，每周六都会去那里喝拿铁。"})
		if err != nil {
			t.Fatal(err)
		}
		task := poll(t, 3*time.Minute, 2*time.Second, "managed file generation", func() (*sdk.GetMemoryTaskResponse, error) {
			return c.GetMemoryTask(ctx, &sdk.GetMemoryTaskRequest{MemoryStoreName: store, RequestID: added.RequestID, Scope: &scope})
		}, func(r *sdk.GetMemoryTaskResponse) bool {
			return r.Task.Status == sdk.MemoryTaskStatusCompleted || r.Task.Status == sdk.MemoryTaskStatusFailed
		})
		if task.Task.Status != sdk.MemoryTaskStatusCompleted {
			t.Fatalf("file generation failed: %+v", task.Task)
		}
		listed, err := c.ListItems(ctx, &sdk.ListItemsRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: scope}, View: sdk.ItemViewFull, Limit: 20})
		if err != nil || len(listed.Items) == 0 {
			t.Fatalf("managed list: %+v %v", listed, err)
		}
		for _, item := range listed.Items {
			if !listed.ReadOnly || item.Scope != scope {
				t.Fatalf("managed item not readonly or wrong scope: %+v", item)
			}
		}
	}
	wild := base
	wild.RunID = "*"
	search := func(t *testing.T, scope sdk.Scope, filter *sdk.ScopeFilter, want map[sdk.Scope]string) {
		t.Helper()
		poll(t, 90*time.Second, 2*time.Second, "all selected managed files retrievable", func() (*sdk.SearchItemsResponse, error) {
			return c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: scope}, Filter: filter, Query: "用户最喜欢的咖啡馆和拿铁", TopK: 50})
		}, func(r *sdk.SearchItemsResponse) bool {
			seen := map[sdk.Scope]bool{}
			for _, hit := range r.Items {
				marker, ok := want[hit.Scope]
				if !ok {
					t.Fatalf("cross-tenant/filter leak: %+v", hit.Scope)
				}
				item, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: hit.Scope}, ItemID: hit.ItemID, View: sdk.ItemViewFull})
				if err != nil {
					t.Fatal(err)
				}
				if item.Content != nil && strings.Contains(*item.Content, marker) {
					seen[hit.Scope] = true
				}
			}
			return len(seen) == len(want)
		})
	}
	t.Run("wildcard_returns_both_sessions_and_excludes_other_tenant", func(t *testing.T) {
		search(t, wild, nil, map[sdk.Scope]string{scopes[0]: markers[0], scopes[1]: markers[1]})
	})
	t.Run("filter_intersects_wildcard", func(t *testing.T) {
		search(t, wild, &sdk.ScopeFilter{Op: "eq", Field: "runId", Value: sdk.String(scopes[1].RunID)}, map[sdk.Scope]string{scopes[1]: markers[1]})
	})
}

func waitReleaseWiki(t *testing.T, c *sdk.Client, sc sdk.WikiScopedRequest) *sdk.Wiki {
	t.Helper()
	w := poll(t, 8*time.Minute, 3*time.Second, "Wiki publication", func() (*sdk.Wiki, error) {
		return c.GetWiki(context.Background(), &sdk.GetWikiRequest{WikiScopedRequest: sc})
	}, func(w *sdk.Wiki) bool {
		return w.LatestBuild != nil && (w.LatestBuild.Status == "ready" || w.LatestBuild.Status == "failed" || w.LatestBuild.Status == "conflict")
	})
	if w.LatestBuild.Status != "ready" || w.PageCount == 0 {
		t.Fatalf("Wiki not published: status=%s pages=%d build=%+v", w.Status, w.PageCount, w.LatestBuild)
	}
	return w
}

func TestReleaseWikiPublicationE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputFile})
	scope := fullScope("wiki_publish")
	scope.RunID = ""
	is := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: scope}
	ws := sdk.WikiScopedRequest{MemoryStoreName: store, Scope: sdk.AgentScope{AppID: scope.AppID, TenantID: scope.TenantID, AgentID: scope.AgentID}}
	_, err := c.UpdateWiki(ctx, &sdk.UpdateWikiRequest{WikiScopedRequest: ws, Name: sdk.String("Synthetic Cedar handbook"), CompileOptions: sdk.NullableOf(sdk.WikiCompileOptions{Language: "en", Instructions: "Create concise factual pages from the sources. Preserve the Cedar project code and include source citations."})})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/formats/source.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: is, Path: "/.wiki/raw/cedar.pdf", ContentBase64: sdk.String(base64.StdEncoding.EncodeToString(data)), MediaType: "application/pdf"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: is, Path: "/.wiki/raw/handbook.md", Content: "# Cedar handbook\nThe Cedar project deployment review takes place every Tuesday. Mira owns the review and checks the code before release."})
	if err != nil {
		t.Fatal(err)
	}
	req := &sdk.IngestWikiRequest{WikiScopedRequest: ws, ClientToken: "go-sdk-wiki-" + suffix(10)}
	if _, err = c.IngestWiki(ctx, req); err != nil {
		t.Fatal(err)
	}
	w := waitReleaseWiki(t, c, ws)
	t.Logf("Wiki published: pages=%d completedSources=%d totalSources=%d", w.PageCount, w.LatestBuild.CompletedSources, w.LatestBuild.TotalSources)
	var pages []sdk.WikiPage
	token := ""
	seen := map[string]bool{}
	for {
		r, err := c.ListWikiPages(ctx, &sdk.ListWikiPagesRequest{WikiScopedRequest: ws, Limit: 1, NextToken: token})
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, r.Pages...)
		token = r.NextToken
		if token == "" {
			break
		}
		if seen[token] {
			t.Fatal("Wiki pagination did not advance")
		}
		seen[token] = true
	}
	if len(pages) == 0 {
		t.Fatal("ready Wiki has no pages")
	}
	found := false
	for _, p := range pages {
		got, err := c.GetWikiPage(ctx, &sdk.GetWikiPageRequest{WikiScopedRequest: ws, Path: p.Path, IncludeReflectionInfo: true})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got.Content, "4827") {
			for _, source := range got.Sources {
				if strings.Contains(source, "cedar.pdf") {
					found = true
				}
			}
			if len(got.Sources) == 0 {
				t.Fatalf("factual page lost source provenance: %s", p.Path)
			}
		}
	}
	if !found {
		t.Fatal("published pages lost source fact 4827")
	}
	poll(t, 90*time.Second, 2*time.Second, "Wiki search returns persisted fact", func() (*sdk.SearchWikiPagesResponse, error) {
		return c.SearchWikiPages(ctx, &sdk.SearchWikiPagesRequest{WikiScopedRequest: ws, Query: "Cedar project code 4827", Limit: 10})
	}, func(r *sdk.SearchWikiPagesResponse) bool {
		for _, p := range r.Pages {
			got, err := c.GetWikiPage(ctx, &sdk.GetWikiPageRequest{WikiScopedRequest: ws, Path: p.Path})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(got.Content, "4827") {
				return true
			}
		}
		return false
	})
	other := ws
	other.Scope.TenantID += "_excluded"
	r, err := c.SearchWikiPages(ctx, &sdk.SearchWikiPagesRequest{WikiScopedRequest: other, Query: "Cedar project code 4827", Limit: 10})
	if err == nil && len(r.Pages) != 0 {
		t.Fatal("Wiki search leaked to another tenant")
	}
	if err != nil {
		if status, _ := serviceStatus(err); status != 404 {
			t.Fatal(err)
		}
	}
}

func waitReleaseDream(t *testing.T, c *sdk.Client, store, id string, want sdk.DreamTaskStatus) *sdk.GetMemoryDreamTaskResponse {
	t.Helper()
	r := poll(t, 8*time.Minute, 3*time.Second, "Dream "+string(want), func() (*sdk.GetMemoryDreamTaskResponse, error) {
		return c.GetMemoryDreamTask(context.Background(), &sdk.GetMemoryDreamTaskRequest{MemoryStoreName: store, DreamID: id})
	}, func(r *sdk.GetMemoryDreamTaskResponse) bool {
		return r.Status == want || r.Status == sdk.DreamTaskStatusCompleted || r.Status == sdk.DreamTaskStatusFailed || r.Status == sdk.DreamTaskStatusCompletedWithFailures || r.Status == sdk.DreamTaskStatusCancelled
	})
	if r.Status != want {
		t.Fatalf("Dream failed: store=%s dreamId=%s status=%s code=%s error=%s diagnostic=%+v", store, id, r.Status, r.FailureCode, r.LastError, r.Diagnostic)
	}
	return r
}

func TestReleaseDreamPlanE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputMessages, []string{sdk.StoreOutputStructured})
	scope := fullScope("dream_plan")
	for _, fact := range []string{"Cedar 项目的发布暗号是翠竹六号。", "Cedar 项目每周二发布，发布前必须校验暗号翠竹六号。"} {
		r, err := c.AddMemories(ctx, &sdk.AddMemoriesRequest{MemoryStoreName: store, Scope: &scope, Text: fact, Sync: true})
		if err != nil || len(r.MemoryIDs) == 0 {
			t.Fatalf("Dream fixture extraction: %+v %v", r, err)
		}
	}
	req := &sdk.CreateMemoryDreamTaskRequest{MemoryStoreName: store, Scopes: []sdk.Scope{scope}, Effort: sdk.DreamEffortHigh, ApplyMode: sdk.DreamApplyModePlan, MaxMemories: 20, MaxMessages: 20, ClientToken: "plan-" + suffix(10), Instructions: "把 Cedar 项目的两个事实整理为一条完整的发布流程记忆：每周二发布，发布前校验暗号翠竹六号。需要实际生成 ADD 或 MERGE 动作，不要只给出 NOOP。保留事实，不编造新内容。"}
	created, err := c.CreateMemoryDreamTask(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	again, err := c.CreateMemoryDreamTask(ctx, req)
	if err != nil || again.DreamID != created.DreamID {
		t.Fatalf("Dream create not idempotent: %+v %v", again, err)
	}
	planned := waitReleaseDream(t, c, store, created.DreamID, sdk.DreamTaskStatusAwaitingPlanApproval)
	if planned.Plan == nil || planned.Plan.Summary == "" || planned.Plan.ApprovalExpiresAt == "" {
		t.Fatalf("missing reviewable plan: %+v", planned.Plan)
	}
	apply := &sdk.ApplyMemoryDreamActionsRequest{MemoryStoreName: store, DreamID: created.DreamID, Mode: sdk.DreamApplyRequestModePlan, Approver: "go-sdk-synthetic-acceptance"}
	firstApproval, err := c.ApplyMemoryDreamActions(ctx, apply)
	if err != nil {
		t.Fatal(err)
	}
	// TaskManager can recover the same approver's in-flight approval. It must
	// preserve the original approval/execution windows, never start a new one.
	repeated, err := c.ApplyMemoryDreamActions(ctx, apply)
	if err == nil {
		originalAt, e1 := time.Parse(time.RFC3339Nano, firstApproval.ApprovedAt)
		repeatedAt, e2 := time.Parse(time.RFC3339Nano, repeated.ApprovedAt)
		if e1 != nil || e2 != nil {
			t.Fatalf("invalid approval time: %v %v", e1, e2)
		}
		// The durable store keeps milliseconds; the first in-memory response
		// may include a sub-millisecond fraction that readback truncates.
		if repeated.DreamID != firstApproval.DreamID || repeatedAt.UnixMilli() != originalAt.UnixMilli() || repeated.ExecutionExpiresAt != firstApproval.ExecutionExpiresAt {
			t.Fatalf("repeat approval changed original window: %+v -> %+v", firstApproval, repeated)
		}
	} else if status, _ := serviceStatus(err); status != 409 {
		t.Fatalf("repeat approval: %v", err)
	}
	foreign := *apply
	foreign.Approver = "different-reviewer"
	if _, err = c.ApplyMemoryDreamActions(ctx, &foreign); err == nil {
		t.Fatal("different approver reused approval")
	} else if status, _ := serviceStatus(err); status != 409 {
		t.Fatalf("different approver: %v", err)
	}
	done := waitReleaseDream(t, c, store, created.DreamID, sdk.DreamTaskStatusCompleted)
	if done.Plan == nil || done.Plan.ApprovedAt == "" || done.Plan.Approver != apply.Approver || done.Actions.Applied == 0 {
		t.Fatalf("plan did not apply: %+v", done)
	}
	actions, err := c.ListMemoryDreamActions(ctx, &sdk.ListMemoryDreamActionsRequest{MemoryStoreName: store, DreamID: created.DreamID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range actions.Actions {
		if a.Status != sdk.DreamActionStatusApplied || a.ResultMemoryID == "" {
			continue
		}
		got, err := c.GetMemory(ctx, &sdk.GetMemoryRequest{MemoryStoreName: store, MemoryID: a.ResultMemoryID, Scope: a.TargetScope})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got.Memory.Text, "翠竹六号") && strings.Contains(got.Memory.Text, "周二") {
			found = true
		}
	}
	if !found {
		t.Fatal("applied Dream produced no persisted merged fact")
	}
	t.Logf("Dream plan applied and result read back: dreamId=%s actions=%d", created.DreamID, done.Actions.Applied)
	// Cancellation is tested in a stable approval state, avoiding a race with
	// a task that could finish before the cancellation request arrives.
	req.ClientToken = "cancel-" + suffix(10)
	req.Instructions = "审阅 Cedar 项目的现有记忆并给出整理计划。"
	cancelTask, err := c.CreateMemoryDreamTask(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	waitReleaseDream(t, c, store, cancelTask.DreamID, sdk.DreamTaskStatusAwaitingPlanApproval)
	for i := 0; i < 2; i++ {
		r, err := c.CancelMemoryDreamTask(ctx, &sdk.CancelMemoryDreamTaskRequest{MemoryStoreName: store, DreamID: cancelTask.DreamID})
		if err != nil || r.Status != sdk.DreamTaskStatusCancelled {
			t.Fatalf("cancel %d: %+v %v", i, r, err)
		}
	}
	waitReleaseDream(t, c, store, cancelTask.DreamID, sdk.DreamTaskStatusCancelled)
}

// Dream extract artifacts have their own lifecycle. Deleting or redacting an
// input may remove its direct derivation but must preserve published artifacts.
func TestReleaseFileDreamPreservationE2E(t *testing.T) {
	c := extractionClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputStructured, sdk.StoreOutputFile})
	sc := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: fullScope("dream_preserve")}
	item := addExtractionFile(t, c, sc, "/procedure.md", "# Cedar 发布检查流程\n每周二发布 Cedar。首先确认发布负责人 Mira 在场，然后校验发布暗号翠竹六号，最后记录版本号和审核结果。如果暗号不匹配则停止发布。这是团队重复执行的固定流程。")
	for _, unit := range waitDerived(t, c, sc, item, "翠竹六号", true) {
		t.Logf("Dream input read back: memoryId=%s conversationKey=%q sourceFile=%s", unit.ID, unit.ConversationKey, unit.Metadata["source_file_id"])
	}
	searchDerived(t, c, sc, item, "翠竹六号", true)
	created, err := c.CreateMemoryDreamTask(ctx, &sdk.CreateMemoryDreamTaskRequest{MemoryStoreName: store, Scopes: []sdk.Scope{sc.Scope}, TaskType: sdk.DreamTaskTypeSkill, Effort: sdk.DreamEffortHigh, MaxMemories: 20, ClientToken: "preserve-" + suffix(10), Instructions: "从 Cedar 发布检查流程中提炼一份可复用的操作技能。保留 Mira、每周二、翠竹六号及失败时停止发布。请提交 EMIT_SKILL 产物。"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("file-source skill Dream created: dreamId=%s", created.DreamID)
	waitReleaseDream(t, c, store, created.DreamID, sdk.DreamTaskStatusCompleted)
	actions, err := c.ListMemoryDreamActions(ctx, &sdk.ListMemoryDreamActionsRequest{MemoryStoreName: store, DreamID: created.DreamID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	products := map[string]sdk.DreamActionRecord{}
	for _, action := range actions.Actions {
		if action.Action != sdk.DreamActionEmitSkill || action.Status != sdk.DreamActionStatusApplied || action.NewExtract == nil {
			continue
		}
		if action.DreamID != created.DreamID || action.TargetScope != sc.Scope || !strings.Contains(action.NewExtract.Instructions, "翠竹六号") {
			t.Fatalf("Dream artifact lacks identity or fact: %+v", action)
		}
		products[action.ActionID] = action
	}
	if len(products) == 0 {
		t.Fatal("Dream produced no persisted skill artifact")
	}
	// Scope-mode listing reads the actually published current extract manifest,
	// rather than just the task's historical action/audit rows.
	verifyProducts := func() {
		found := map[string]bool{}
		token := ""
		seen := map[string]bool{}
		for {
			r, err := c.ListMemoryDreamActions(ctx, &sdk.ListMemoryDreamActionsRequest{MemoryStoreName: store, Scope: &sc.Scope, ActionType: sdk.DreamActionEmitSkill, Limit: 1, NextToken: token})
			if err != nil {
				t.Fatal(err)
			}
			for _, got := range r.Actions {
				old, ok := products[got.ActionID]
				if !ok {
					t.Fatal("unrelated artifact in isolated current inventory")
				}
				if got.DreamID != old.DreamID || got.Status != sdk.DreamActionStatusApplied || !reflect.DeepEqual(got.NewExtract, old.NewExtract) {
					t.Fatalf("current Dream product changed: %+v", got)
				}
				found[got.ActionID] = true
			}
			token = r.NextToken
			if token == "" {
				break
			}
			if seen[token] {
				t.Fatal("extract cursor repeated")
			}
			seen[token] = true
		}
		if len(found) != len(products) {
			t.Fatalf("current Dream artifacts missing: got=%d want=%d", len(found), len(products))
		}
	}
	verifyProducts()
	rctx, rtrace := extractionContext()
	if _, err = c.RedactItemVersion(rctx, &sdk.RedactItemVersionRequest{ItemScopedRequest: sc, VersionID: item.LatestVersionID}); err != nil {
		t.Fatal(err)
	}
	waitExtractionTask(t, c, sc, rtrace)
	verifyProducts()
	dctx, dtrace := extractionContext()
	if _, err = c.DeleteItem(dctx, &sdk.DeleteItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID}); err != nil {
		t.Fatal(err)
	}
	waitExtractionTask(t, c, sc, dtrace)
	verifyProducts()
	_, err = c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, ItemID: item.ItemID})
	if status, _ := serviceStatus(err); status != 404 {
		t.Fatalf("source remains after delete: %v", err)
	}
	t.Logf("retained %d independent Dream artifacts after source redact and delete", len(products))
}

func TestReleaseFileDreamE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputFile})
	in := fullScope("file_dream")
	out := in
	out.RunID += "_output"
	sc := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: in}
	var sources []*sdk.Item
	sourceVersions := map[string]bool{}
	for i, body := range []string{"Cedar 项目发布前必须核验暗号翠竹六号。", "Cedar 项目每周二发布，由 Mira 负责审核。"} {
		item, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: sc, Path: fmt.Sprintf("/source-%d.md", i), Content: body, View: sdk.ItemViewFull})
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, item)
		sourceVersions[item.LatestVersionID] = true
	}
	r, err := c.CreateMemoryDreamTask(ctx, &sdk.CreateMemoryDreamTaskRequest{MemoryStoreName: store, Scopes: []sdk.Scope{in}, OutputScope: &out, MaxItems: 10, Effort: sdk.DreamEffortHigh, ApplyMode: sdk.DreamApplyModeAuto, ClientToken: "file-" + suffix(10), Instructions: "请将两个来源文件整理成一份 /cedar-release.md 发布指南，包含每周二、Mira 和暗号翠竹六号，使用原始来源版本引用，保留所有原始来源。"})
	if err != nil {
		t.Fatal(err)
	}
	done := waitReleaseDream(t, c, store, r.DreamID, sdk.DreamTaskStatusCompleted)
	if done.OutputScope == nil || *done.OutputScope != out || done.Input.ItemCount != 2 || done.Actions.Applied == 0 {
		t.Fatalf("file Dream incomplete: %+v", done)
	}
	actions, err := c.ListMemoryDreamActions(ctx, &sdk.ListMemoryDreamActionsRequest{MemoryStoreName: store, DreamID: r.DreamID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	referencedVersions := map[string]bool{}
	for _, a := range actions.Actions {
		if a.Status != sdk.DreamActionStatusApplied || a.ResultItemID == "" {
			continue
		}
		if a.TargetScope != out || a.ResultVersionID == "" || a.ItemChange == nil || len(a.ItemChange.SourceVersionIDs) == 0 {
			t.Fatalf("file Dream result lacks scope/version/source: %+v", a)
		}
		for _, versionID := range a.ItemChange.SourceVersionIDs {
			if !sourceVersions[versionID] {
				t.Fatalf("Dream references a version outside its two input files: %s", versionID)
			}
			referencedVersions[versionID] = true
		}
		got, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sdk.ItemScopedRequest{MemoryStoreName: store, Scope: out}, ItemID: a.ResultItemID, View: sdk.ItemViewFull})
		if err != nil {
			t.Fatal(err)
		}
		if got.LatestVersionID != a.ResultVersionID {
			t.Fatal("Dream result version differs from current file")
		}
		if got.Content != nil && strings.Contains(*got.Content, "翠竹六号") && strings.Contains(*got.Content, "周二") && strings.Contains(*got.Content, "Mira") {
			found = true
		}
	}
	if !found {
		t.Fatal("file Dream output lost source facts")
	}
	if len(referencedVersions) != len(sourceVersions) {
		t.Fatalf("Dream did not retain both source version references: got=%d want=%d", len(referencedVersions), len(sourceVersions))
	}
	for _, source := range sources {
		got, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: sc, ItemID: source.ItemID, View: sdk.ItemViewFull})
		if err != nil || got.LatestVersionID != source.LatestVersionID || got.Content == nil || *got.Content != *source.Content {
			t.Fatalf("Dream changed source file: %+v %v", got, err)
		}
	}
	t.Logf("file Dream output content, versions, source references and source isolation verified: dreamId=%s", r.DreamID)
}
