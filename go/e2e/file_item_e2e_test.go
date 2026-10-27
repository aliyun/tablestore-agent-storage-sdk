package e2e

// End-to-end validation of the original eight file-memory item APIs.
// Requires the standard OTS gateway credentials (see memory_e2e_test.go).
// OTS_TENANT optionally selects the synthetic test tenant.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

type itemE2ECtx struct {
	t      *testing.T
	client *agentstorage.Client
	pass   int
	fail   int
}

func (x *itemE2ECtx) check(name string, cond bool, detail string) {
	x.t.Helper()
	if cond {
		x.pass++
		x.t.Logf("PASS %s", name)
	} else {
		x.fail++
		x.t.Errorf("FAIL %s %s", name, detail)
	}
}

// expectServiceError asserts fn fails with a ServiceError whose status is in
// allowed. The OTS gateway maps service validation errors (400) to 403
// OTSValidationFail; direct service access keeps 400.
func (x *itemE2ECtx) expectServiceError(name string, allowed map[int]bool, fn func() error) {
	x.t.Helper()
	err := fn()
	if err == nil {
		x.check(name, false, "expected error, got success")
		return
	}
	status, code := serviceStatus(err)
	var serviceErr *agentstorage.ServiceError
	valid := errors.As(err, &serviceErr) && allowed[status] && code != "" && serviceErr.RequestID != ""
	if status == 403 {
		valid = valid && code == "OTSValidationFail"
	}
	x.check(name, valid, fmt.Sprintf("expected %v got %v", keysOf(allowed), err))
}

func (x *itemE2ECtx) expectClientError(name string, fn func() error) {
	x.t.Helper()
	err := fn()
	var clientErr *agentstorage.ClientError
	x.check(name, errors.As(err, &clientErr), fmt.Sprintf("expected ClientError, got %v", err))
}

func keysOf(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// rawDo sends action through the SDK transport but decodes the response into
// a generic map, so wire-level presence/absence of keys can be asserted.
func rawDo(ctx context.Context, client *agentstorage.Client, action string, payload any) (map[string]any, error) {
	var result map[string]any
	if err := client.Do(ctx, action, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func TestFileItemE2E(t *testing.T) {
	client := e2eClient(t)
	ctx := context.Background()
	// The gateway may map VALIDATION_ERROR to 403 OTSValidationFail.
	// Select by the returned protocol, never by a hardcoded endpoint address.
	validationReject := map[int]bool{400: true, 403: true}

	tenant := os.Getenv("OTS_TENANT")
	if tenant == "" {
		tenant = "go"
	}
	store := currentStore(t, client, agentstorage.StoreInputFile, []string{agentstorage.StoreOutputFile})
	s1 := agentstorage.Scope{AppID: "sdke2e", TenantID: tenant, AgentID: "ag1", RunID: "r1"}
	s2 := agentstorage.Scope{AppID: "sdke2e", TenantID: tenant, AgentID: "ag1", RunID: "r2"}
	s3 := agentstorage.Scope{AppID: "sdke2e", TenantID: tenant, AgentID: "ag2", RunID: "r1"}
	wild := agentstorage.Scope{AppID: "sdke2e", TenantID: tenant, AgentID: "*", RunID: "*"}
	scoped := func(scope agentstorage.Scope) agentstorage.ItemScopedRequest {
		return agentstorage.ItemScopedRequest{MemoryStoreName: store, Scope: scope}
	}

	x := &itemE2ECtx{t: t, client: client}

	// ---- 1. AddItem ----
	// First add goes through the raw escape hatch so the wire-level absence
	// of "content" in the basic view can be asserted.
	rawAddRequest := &agentstorage.AddItemRequest{
		ItemScopedRequest: scoped(s1), Path: "/notes/a.md", Content: "alpha", SessionID: "s1",
	}
	// rawDo bypasses the typed AddItem wrapper, so fill the default type
	// the wrapper would normally inject.
	rawAddRequest.Type = agentstorage.ItemTypeMemoryFile
	rawAdd, err := rawDo(ctx, client, "AddItem", rawAddRequest)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	x.check("AddItem type echo", rawAdd["type"] == agentstorage.ItemTypeMemoryFile, fmt.Sprint(rawAdd))
	idA, _ := rawAdd["itemId"].(string)
	x.check("AddItem itemId", idA != "", fmt.Sprint(rawAdd))
	scopeEcho, _ := rawAdd["scope"].(map[string]any)
	x.check("AddItem scope echo",
		scopeEcho["appId"] == "sdke2e" && scopeEcho["tenantId"] == tenant && scopeEcho["agentId"] == "ag1" && scopeEcho["runId"] == "r1",
		fmt.Sprint(scopeEcho))
	latestVersionID, _ := rawAdd["latestVersionId"].(string)
	// Version IDs are opaque; publication-backed stores use a newer encoding.
	x.check("AddItem latestVersionId", latestVersionID != "", latestVersionID)
	_, hasContent := rawAdd["content"]
	x.check("AddItem basic no content", !hasContent, fmt.Sprint(rawAdd))
	sizeA, _ := rawAdd["contentSizeBytes"].(float64)
	shaA0, _ := rawAdd["contentSha256"].(string)
	x.check("AddItem sha/size", sizeA == 5 && shaA0 != "", fmt.Sprint(rawAdd))
	seqA0, _ := rawAdd["latestSeq"].(float64)
	x.check("AddItem latestSeq positive int", seqA0 > 0, fmt.Sprint(rawAdd["latestSeq"]))

	addB, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
		ItemScopedRequest: scoped(s1), Path: "/notes/b.md", Content: "beta", View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("AddItem b: %v", err)
	}
	x.check("AddItem view=full echoes content", addB.Content != nil && *addB.Content == "beta", fmt.Sprint(addB))
	idB := addB.ItemID

	addEmpty, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
		ItemScopedRequest: scoped(s1), Path: "/empty.md", Content: "",
	})
	if err != nil {
		t.Fatalf("AddItem empty: %v", err)
	}
	x.check("AddItem empty file size 0", addEmpty.ContentSizeBytes == 0, fmt.Sprint(addEmpty))
	idEmpty := addEmpty.ItemID

	if _, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
		ItemScopedRequest: scoped(s2), Path: "/r2.md", Content: "r2",
	}); err != nil {
		t.Fatalf("AddItem r2: %v", err)
	}
	addC, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
		ItemScopedRequest: scoped(s3), Path: "/x/deep/c.md", Content: "gamma",
	})
	if err != nil {
		t.Fatalf("AddItem c: %v", err)
	}
	x.check("AddItem second agent scope", addC.ItemID != "", addC.ItemID)

	x.expectServiceError("AddItem no leading slash -> 400", validationReject, func() error {
		_, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
			ItemScopedRequest: scoped(s1), Path: "bad.md", Content: "x",
		})
		return err
	})
	x.expectServiceError("AddItem missing content -> 400", validationReject, func() error {
		err := client.Do(ctx, "AddItem", map[string]any{
			"type": agentstorage.ItemTypeMemoryFile, "memoryStoreName": store, "scope": s1, "path": "/nocontent.md",
		}, nil)
		return err
	})
	x.expectServiceError("AddItem duplicate path -> 409", map[int]bool{409: true}, func() error {
		_, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
			ItemScopedRequest: scoped(s1), Path: "/notes/a.md", Content: "dup",
		})
		return err
	})

	// ---- 2. GetItem ----
	getBasic, err := client.GetItem(ctx, &agentstorage.GetItemRequest{ItemScopedRequest: scoped(s1), Path: "/notes/a.md"})
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	x.check("GetItem default basic", getBasic.Content == nil && getBasic.ItemID == idA, fmt.Sprint(getBasic))
	getFull, err := client.GetItem(ctx, &agentstorage.GetItemRequest{
		ItemScopedRequest: scoped(s1), Path: "/notes/a.md", View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("GetItem full: %v", err)
	}
	x.check("GetItem view=full content", getFull.Content != nil && *getFull.Content == "alpha", fmt.Sprint(getFull))
	current, err := client.GetItem(ctx, &agentstorage.GetItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: idA, View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("GetItem by itemId: %v", err)
	}
	x.check("GetItem by itemId", current.Content != nil && *current.Content == "alpha" && current.Path == "/notes/a.md", fmt.Sprint(current))
	x.expectServiceError("GetItem itemId+path -> 400", validationReject, func() error {
		_, err := client.GetItem(ctx, &agentstorage.GetItemRequest{
			ItemScopedRequest: scoped(s1), ItemID: idB, Path: "/notes/a.md",
		})
		return err
	})
	getEmpty, err := client.GetItem(ctx, &agentstorage.GetItemRequest{
		ItemScopedRequest: scoped(s1), Path: "/empty.md", View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("GetItem empty: %v", err)
	}
	x.check("GetItem empty file content ''", getEmpty.Content != nil && *getEmpty.Content == "", fmt.Sprint(getEmpty))
	x.expectServiceError("GetItem missing -> 404", map[int]bool{404: true}, func() error {
		_, err := client.GetItem(ctx, &agentstorage.GetItemRequest{ItemScopedRequest: scoped(s1), Path: "/nope.md"})
		return err
	})
	x.expectServiceError("GetItem no addressing -> 400", validationReject, func() error {
		_, err := client.GetItem(ctx, &agentstorage.GetItemRequest{ItemScopedRequest: scoped(s1)})
		return err
	})

	// ---- 3. UpdateItem ----
	updated, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: idA, Content: agentstorage.String("alpha-2"),
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: current.LatestVersionID},
		SessionID:    "s1", View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("UpdateItem content: %v", err)
	}
	x.check("UpdateItem content branch",
		updated.Content != nil && *updated.Content == "alpha-2" && updated.ItemID == idA, fmt.Sprint(updated))
	x.check("UpdateItem latestSeq increases", updated.LatestSeq > int64(seqA0),
		fmt.Sprintf("%v -> %v", seqA0, updated.LatestSeq))
	x.expectServiceError("UpdateItem CAS mismatch -> 409", map[int]bool{409: true}, func() error {
		_, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
			ItemScopedRequest: scoped(s1), ItemID: idA, Content: agentstorage.String("x"),
			Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: addB.LatestVersionID},
		})
		return err
	})
	renamed, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: idA, Path: agentstorage.String("/notes/a3.md"),
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: updated.LatestVersionID},
		SessionID:    "s1",
	})
	if err != nil {
		t.Fatalf("UpdateItem rename: %v", err)
	}
	x.check("UpdateItem rename branch", renamed.Path == "/notes/a3.md" && renamed.Content == nil, fmt.Sprint(renamed))
	updComposite, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: idA, Content: agentstorage.String("alpha-3"),
		Path:         agentstorage.String("/notes/a4.md"),
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: renamed.LatestVersionID},
		SessionID:    "s1", View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("UpdateItem composite: %v", err)
	}
	x.check("UpdateItem composite content+target path",
		updComposite.Content != nil && *updComposite.Content == "alpha-3" &&
			updComposite.Path == "/notes/a4.md" && updComposite.ItemID == idA, fmt.Sprint(updComposite))
	x.expectServiceError("UpdateItem missing content and target path -> 400", validationReject, func() error {
		_, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
			ItemScopedRequest: scoped(s1), ItemID: idA,
		})
		return err
	})

	if _, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
		ItemScopedRequest: scoped(s1), Path: "/target.md", Content: "old-target",
	}); err != nil {
		t.Fatalf("AddItem target: %v", err)
	}
	target, err := client.GetItem(ctx, &agentstorage.GetItemRequest{ItemScopedRequest: scoped(s1), Path: "/target.md"})
	if err != nil {
		t.Fatalf("GetItem target: %v", err)
	}
	x.expectServiceError("UpdateItem rename occupied overwrite=false -> 409", map[int]bool{409: true}, func() error {
		_, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
			ItemScopedRequest: scoped(s1), ItemID: idA, Path: agentstorage.String("/target.md"),
			Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: updComposite.LatestVersionID},
		})
		return err
	})
	updOverwrite, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: idA, Path: agentstorage.String("/target.md"),
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: updComposite.LatestVersionID},
		Overwrite:    true, SessionID: "s1", View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("UpdateItem overwrite: %v", err)
	}
	x.check("UpdateItem overwrite preserves source itemId", updOverwrite.ItemID == idA, updOverwrite.ItemID)
	x.check("UpdateItem overwrite preserves source content",
		updOverwrite.Content != nil && *updOverwrite.Content == "alpha-3", fmt.Sprint(updOverwrite.Content))
	x.expectServiceError("UpdateItem overwrite displaces occupied target -> 404", map[int]bool{404: true}, func() error {
		_, err := client.GetItem(ctx, &agentstorage.GetItemRequest{ItemScopedRequest: scoped(s1), ItemID: target.ItemID})
		return err
	})

	// ---- 4. ListItems ----
	listAll, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{ItemScopedRequest: scoped(s1), Limit: 100})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	paths := make([]string, 0, len(listAll.Items))
	for _, entry := range listAll.Items {
		paths = append(paths, entry.Path)
	}
	sort.Strings(paths)
	x.check("ListItems exact scope paths",
		fmt.Sprint(paths) == fmt.Sprint([]string{"/empty.md", "/notes/b.md", "/target.md"}), fmt.Sprint(paths))
	allNoContent, allTyped, allVersioned := true, true, true
	for _, entry := range listAll.Items {
		if entry.Content != nil {
			allNoContent = false
		}
		if entry.Type != agentstorage.ItemTypeMemoryFile {
			allTyped = false
		}
		if entry.LatestVersionID == "" {
			allVersioned = false
		}
	}
	x.check("ListItems basic entries no content", allNoContent, "")
	x.check("ListItems entries typed memoryfile", allTyped, "")
	x.check("ListItems entries carry latestVersionId", allVersioned, "")
	rawListRequest := &agentstorage.ListItemsRequest{ItemScopedRequest: scoped(s1), Limit: 100}
	rawListRequest.Type = agentstorage.ItemTypeMemoryFile
	rawList, err := rawDo(ctx, client, "ListItems", rawListRequest)
	if err != nil {
		t.Fatalf("ListItems raw: %v", err)
	}
	_, hasReadOnly := rawList["readOnly"]
	x.check("ListItems filemem no readOnly", !hasReadOnly, fmt.Sprint(rawList["readOnly"]))

	listPrefix, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{
		ItemScopedRequest: scoped(s1), PathPrefix: "/notes/", View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("ListItems prefix: %v", err)
	}
	contents := make([]string, 0, len(listPrefix.Items))
	for _, entry := range listPrefix.Items {
		if entry.Content != nil {
			contents = append(contents, *entry.Content)
		}
	}
	sort.Strings(contents)
	x.check("ListItems prefix+full content", fmt.Sprint(contents) == fmt.Sprint([]string{"beta"}), fmt.Sprint(contents))

	if _, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
		ItemScopedRequest: scoped(s3), Path: "/x/top.md", Content: "top",
	}); err != nil {
		t.Fatalf("AddItem top: %v", err)
	}
	listDepth1, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{
		ItemScopedRequest: scoped(s3), PathPrefix: "/x/", Depth: agentstorage.Int(1),
	})
	if err != nil {
		t.Fatalf("ListItems depth=1: %v", err)
	}
	rollupSeen, topSeen := false, false
	var prefixEntry *agentstorage.ItemListEntry
	for i := range listDepth1.Items {
		entry := &listDepth1.Items[i]
		if entry.IsPrefix() && entry.Path == "/x/deep/" {
			rollupSeen = true
			prefixEntry = entry
		}
		if entry.Type == agentstorage.ItemTypeMemoryFile && entry.Path == "/x/top.md" {
			topSeen = true
		}
	}
	x.check("ListItems depth=1 rolls up subdir", rollupSeen && topSeen, fmt.Sprint(listDepth1.Items))
	x.check("ListItems prefix entry scope+path only",
		prefixEntry != nil && prefixEntry.Scope == s3 && prefixEntry.ItemID == "", fmt.Sprint(prefixEntry))
	listDepth0, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{
		ItemScopedRequest: scoped(s3), PathPrefix: "/x/", Depth: agentstorage.Int(0),
	})
	if err != nil {
		t.Fatalf("ListItems depth=0: %v", err)
	}
	deepPaths := make([]string, 0, len(listDepth0.Items))
	for _, entry := range listDepth0.Items {
		deepPaths = append(deepPaths, entry.Path)
	}
	sort.Strings(deepPaths)
	x.check("ListItems depth=0 recursive",
		fmt.Sprint(deepPaths) == fmt.Sprint([]string{"/x/deep/c.md", "/x/top.md"}), fmt.Sprint(deepPaths))
	x.expectServiceError("ListItems bad depth -> 400", validationReject, func() error {
		_, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{ItemScopedRequest: scoped(s1), Depth: agentstorage.Int(2)})
		return err
	})
	x.expectServiceError("ListItems unaligned prefix -> 400", validationReject, func() error {
		_, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{ItemScopedRequest: scoped(s1), PathPrefix: "/notes"})
		return err
	})

	seen, pages := []string{}, 0
	token := ""
	for {
		pageReq := &agentstorage.ListItemsRequest{ItemScopedRequest: scoped(s1), Limit: 1, NextToken: token}
		page, err := client.ListItems(ctx, pageReq)
		if err != nil {
			t.Fatalf("ListItems pagination: %v", err)
		}
		for _, entry := range page.Items {
			seen = append(seen, entry.Path)
		}
		pages++
		token = page.NextToken
		if token == "" {
			break
		}
		if pages > 10 {
			x.check("ListItems pagination terminates", false, "too many pages")
			break
		}
	}
	sort.Strings(seen)
	x.check("ListItems pagination walks all",
		fmt.Sprint(seen) == fmt.Sprint([]string{"/empty.md", "/notes/b.md", "/target.md"}), fmt.Sprint(seen))

	listWild, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{ItemScopedRequest: scoped(wild), Limit: 100})
	if err != nil {
		t.Fatalf("ListItems wildcard: %v", err)
	}
	scopesSeen := make([]string, 0, len(listWild.Items))
	for _, entry := range listWild.Items {
		scopesSeen = append(scopesSeen, entry.Scope.AgentID+"/"+entry.Scope.RunID)
	}
	sort.Strings(scopesSeen)
	x.check("ListItems wildcard scope spans agents",
		fmt.Sprint(scopesSeen) == fmt.Sprint([]string{"ag1/r1", "ag1/r1", "ag1/r1", "ag1/r2", "ag2/r1", "ag2/r1"}),
		fmt.Sprint(scopesSeen))
	x.expectServiceError("ListItems runId after wildcard agentId -> 400", validationReject, func() error {
		_, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{
			ItemScopedRequest: scoped(agentstorage.Scope{AppID: "sdke2e", TenantID: tenant, AgentID: "*", RunID: "r1"}),
		})
		return err
	})
	listAgentWild, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{
		ItemScopedRequest: scoped(agentstorage.Scope{AppID: "sdke2e", TenantID: tenant, AgentID: "ag1", RunID: "*"}),
		Limit:             100,
	})
	if err != nil {
		t.Fatalf("ListItems agent wildcard: %v", err)
	}
	x.check("ListItems agent exact + run wildcard ok", len(listAgentWild.Items) >= 4, fmt.Sprint(len(listAgentWild.Items)))

	// ---- 5. ListItemVersions (dedicated item with known actor history) ----
	va, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
		ItemScopedRequest: scoped(s1), Path: "/ver.md", Content: "v1", SessionID: "s1",
	})
	if err != nil {
		t.Fatalf("AddItem ver: %v", err)
	}
	verID := va.ItemID
	upd2, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: verID, Content: agentstorage.String("v2"), SessionID: "s1",
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: va.LatestVersionID},
	})
	if err != nil {
		t.Fatalf("UpdateItem v2: %v", err)
	}
	upd3, err := client.UpdateItem(ctx, &agentstorage.UpdateItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: verID, Content: agentstorage.String("v3"), SessionID: "s1",
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: upd2.LatestVersionID},
	})
	if err != nil {
		t.Fatalf("UpdateItem v3: %v", err)
	}
	verHead := upd3.LatestVersionID

	versAll, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{ItemScopedRequest: scoped(s1), ItemID: verID})
	if err != nil {
		t.Fatalf("ListItemVersions: %v", err)
	}
	ops := make([]string, 0, len(versAll.Versions))
	for _, version := range versAll.Versions {
		ops = append(ops, string(version.Operation))
	}
	x.check("ListItemVersions newest first",
		fmt.Sprint(ops) == fmt.Sprint([]string{"modified", "modified", "created"}), fmt.Sprint(ops))
	v0 := versAll.Versions[0]
	x.check("Version payload scope echo", v0.Scope == s1, fmt.Sprint(v0.Scope))
	x.check("Version payload createdBy session actor",
		v0.CreatedBy != nil && v0.CreatedBy.ActorType == "session" && v0.CreatedBy.ActorID == "s1",
		fmt.Sprint(v0.CreatedBy))
	rawVersRequest := &agentstorage.ListItemVersionsRequest{ItemScopedRequest: scoped(s1), ItemID: verID}
	rawVersRequest.Type = agentstorage.ItemTypeMemoryFile
	rawVers, err := rawDo(ctx, client, "ListItemVersions", rawVersRequest)
	if err != nil {
		t.Fatalf("ListItemVersions raw: %v", err)
	}
	noLegacyKeys := true
	if rawVersions, ok := rawVers["versions"].([]any); ok && len(rawVersions) > 0 {
		if first, ok := rawVersions[0].(map[string]any); ok {
			_, hasSeq := first["versionSeq"]
			_, hasSession := first["sessionId"]
			noLegacyKeys = !hasSeq && !hasSession
		}
	}
	x.check("Version payload no versionSeq/sessionId keys", noLegacyKeys, fmt.Sprint(rawVers))
	noVersionContent := true
	for _, version := range versAll.Versions {
		if version.Content != nil {
			noVersionContent = false
		}
	}
	x.check("Version basic no content", noVersionContent, "")

	versCreated, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{
		ItemScopedRequest: scoped(s1), ItemID: verID, Operation: agentstorage.ItemOperationCreated,
	})
	if err != nil {
		t.Fatalf("ListItemVersions operation filter: %v", err)
	}
	x.check("ListItemVersions operation filter",
		len(versCreated.Versions) == 1 && versCreated.Versions[0].Operation == agentstorage.ItemOperationCreated,
		fmt.Sprint(ops))
	firstVer := ""
	if len(versCreated.Versions) == 1 {
		firstVer = versCreated.Versions[0].VersionID
	}
	versSession, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{
		ItemScopedRequest: scoped(s1), ItemID: verID, SessionID: "s1",
	})
	if err != nil {
		t.Fatalf("ListItemVersions sessionId filter: %v", err)
	}
	x.check("ListItemVersions sessionId filter", len(versSession.Versions) == 3, fmt.Sprint(len(versSession.Versions)))
	// The SDK rejects the removed apiKeyId filter before sending a request.
	x.expectClientError("ListItemVersions apiKeyId removed", func() error {
		_, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{
			ItemScopedRequest: scoped(s1), ItemID: verID, ApiKeyID: "k-none",
		})
		return err
	})
	x.expectClientError("ListItemVersions sessionId+apiKeyId removed", func() error {
		_, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{
			ItemScopedRequest: scoped(s1), ItemID: verID, SessionID: "s1", ApiKeyID: "k1",
		})
		return err
	})
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	versWindow, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{
		ItemScopedRequest: scoped(s1), ItemID: verID, CreatedAtLte: future, View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("ListItemVersions time window: %v", err)
	}
	windowContents := make([]string, 0, len(versWindow.Versions))
	for _, version := range versWindow.Versions {
		if version.Content != nil {
			windowContents = append(windowContents, *version.Content)
		}
	}
	sort.Strings(windowContents)
	x.check("ListItemVersions time filter + full content",
		len(versWindow.Versions) == 3 && fmt.Sprint(windowContents) == fmt.Sprint([]string{"v1", "v2", "v3"}),
		fmt.Sprint(windowContents))
	versEmpty, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{
		ItemScopedRequest: scoped(s1), ItemID: verID, CreatedAtGte: future,
	})
	if err != nil {
		t.Fatalf("ListItemVersions empty window: %v", err)
	}
	x.check("ListItemVersions empty window", len(versEmpty.Versions) == 0, fmt.Sprint(versEmpty.Versions))
	versWild, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{ItemScopedRequest: scoped(wild), ItemID: verID})
	if err != nil {
		t.Fatalf("ListItemVersions wildcard: %v", err)
	}
	x.check("ListItemVersions wildcard scope", len(versWild.Versions) == 3, fmt.Sprint(len(versWild.Versions)))
	versPaged, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{ItemScopedRequest: scoped(s1), Limit: 2})
	if err != nil {
		t.Fatalf("ListItemVersions limit: %v", err)
	}
	x.check("ListItemVersions limit+nextToken", len(versPaged.Versions) == 2 && versPaged.NextToken != "",
		fmt.Sprint(len(versPaged.Versions)))

	// ---- 6. GetItemVersion ----
	gvFull, err := client.GetItemVersion(ctx, &agentstorage.GetItemVersionRequest{
		ItemScopedRequest: scoped(s1), VersionID: verHead, View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("GetItemVersion: %v", err)
	}
	x.check("GetItemVersion by versionId only", gvFull.VersionID == verHead, gvFull.VersionID)
	x.check("GetItemVersion full content", gvFull.Content != nil && *gvFull.Content == "v3", fmt.Sprint(gvFull.Content))
	gvBasic, err := client.GetItemVersion(ctx, &agentstorage.GetItemVersionRequest{ItemScopedRequest: scoped(s1), VersionID: verHead})
	if err != nil {
		t.Fatalf("GetItemVersion basic: %v", err)
	}
	x.check("GetItemVersion default basic no content", gvBasic.Content == nil, fmt.Sprint(gvBasic))
	gvFirst, err := client.GetItemVersion(ctx, &agentstorage.GetItemVersionRequest{
		ItemScopedRequest: scoped(s1), VersionID: firstVer, View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("GetItemVersion first: %v", err)
	}
	x.check("GetItemVersion first version content", gvFirst.Content != nil && *gvFirst.Content == "v1", fmt.Sprint(gvFirst.Content))
	x.expectServiceError("GetItemVersion bad versionId -> 400", validationReject, func() error {
		_, err := client.GetItemVersion(ctx, &agentstorage.GetItemVersionRequest{
			ItemScopedRequest: scoped(s1), VersionID: "not-a-version",
		})
		return err
	})
	x.expectServiceError("GetItemVersion foreign scope -> 400", validationReject, func() error {
		_, err := client.GetItemVersion(ctx, &agentstorage.GetItemVersionRequest{
			ItemScopedRequest: scoped(s2), VersionID: firstVer, View: agentstorage.ItemViewFull,
		})
		return err
	})

	// ---- 7. RedactItemVersion ----
	redacted, err := client.RedactItemVersion(ctx, &agentstorage.RedactItemVersionRequest{
		ItemScopedRequest: scoped(s1), VersionID: firstVer, SessionID: "redactor",
	})
	if err != nil {
		t.Fatalf("RedactItemVersion: %v", err)
	}
	x.check("Redact redacted flag", redacted.Redacted, fmt.Sprint(redacted))
	x.check("Redact clears content fields",
		redacted.ContentSHA256 == "" && redacted.ContentSizeBytes == 0 && redacted.Path == "", fmt.Sprint(redacted))
	x.check("Redact redactedBy typed actor",
		redacted.RedactedBy != nil && redacted.RedactedBy.ActorType == "session" && redacted.RedactedBy.ActorID == "redactor",
		fmt.Sprint(redacted.RedactedBy))
	firstAt := redacted.RedactedAt
	redacted2, err := client.RedactItemVersion(ctx, &agentstorage.RedactItemVersionRequest{
		ItemScopedRequest: scoped(s1), VersionID: firstVer, SessionID: "other",
	})
	if err != nil {
		t.Fatalf("RedactItemVersion repeat: %v", err)
	}
	x.check("Redact idempotent first-wins",
		redacted2.RedactedAt == firstAt && redacted2.RedactedBy != nil && redacted2.RedactedBy.ActorID == "redactor",
		redacted2.RedactedAt)
	gvRedacted, err := client.GetItemVersion(ctx, &agentstorage.GetItemVersionRequest{
		ItemScopedRequest: scoped(s1), VersionID: firstVer, View: agentstorage.ItemViewFull,
	})
	if err != nil {
		t.Fatalf("GetItemVersion redacted: %v", err)
	}
	x.check("GetItemVersion redacted keeps 200 without content",
		gvRedacted.Content == nil && gvRedacted.Redacted, fmt.Sprint(gvRedacted))

	// ---- 8. DeleteItem ----
	x.expectServiceError("DeleteItem itemId+path -> 400", validationReject, func() error {
		_, err := client.DeleteItem(ctx, &agentstorage.DeleteItemRequest{
			ItemScopedRequest: scoped(s1), ItemID: idA, Path: "/target.md",
		})
		return err
	})
	delEmpty, err := client.DeleteItem(ctx, &agentstorage.DeleteItemRequest{
		ItemScopedRequest: scoped(s1), Path: "/empty.md",
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: addEmpty.LatestVersionID},
	})
	if err != nil {
		t.Fatalf("DeleteItem single: %v", err)
	}
	x.check("DeleteItem single tombstone", delEmpty.Deleted && delEmpty.ItemID == idEmpty, fmt.Sprint(delEmpty))
	x.expectServiceError("DeleteItem again -> 404", map[int]bool{404: true}, func() error {
		_, err := client.DeleteItem(ctx, &agentstorage.DeleteItemRequest{ItemScopedRequest: scoped(s1), Path: "/empty.md"})
		return err
	})
	delByID, err := client.DeleteItem(ctx, &agentstorage.DeleteItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: idB,
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: addB.LatestVersionID},
	})
	if err != nil {
		t.Fatalf("DeleteItem by itemId: %v", err)
	}
	x.check("DeleteItem by itemId", delByID.Deleted && delByID.ItemID == idB, fmt.Sprint(delByID))
	versDeleted, err := client.ListItemVersions(ctx, &agentstorage.ListItemVersionsRequest{
		ItemScopedRequest: scoped(s1), ItemID: idB, Operation: agentstorage.ItemOperationDeleted,
	})
	if err != nil {
		t.Fatalf("ListItemVersions deleted: %v", err)
	}
	x.check("DeleteItem records deleted version", len(versDeleted.Versions) == 1, fmt.Sprint(len(versDeleted.Versions)))
	delRenamedSource, err := client.DeleteItem(ctx, &agentstorage.DeleteItemRequest{
		ItemScopedRequest: scoped(s1), ItemID: idA,
		Precondition: &agentstorage.ItemPrecondition{ExpectedVersionID: updOverwrite.LatestVersionID},
	})
	if err != nil {
		t.Fatalf("DeleteItem overwrite-renamed source: %v", err)
	}
	x.check("DeleteItem uses overwrite-renamed source version",
		delRenamedSource.Deleted && delRenamedSource.ItemID == idA, fmt.Sprint(delRenamedSource))

	t.Run("ranged delete", func(t *testing.T) {
		x := &itemE2ECtx{t: t, client: client}
		for _, bulk := range []struct{ path, content string }{
			{"/bulk/one.md", "1"}, {"/bulk/two.md", "2"}, {"/keep.md", "keep"},
		} {
			if _, err := client.AddItem(ctx, &agentstorage.AddItemRequest{
				ItemScopedRequest: scoped(s2), Path: bulk.path, Content: bulk.content,
			}); err != nil {
				t.Fatalf("AddItem %s: %v", bulk.path, err)
			}
		}
		rangedScope := agentstorage.Scope{AppID: "sdke2e", TenantID: tenant, AgentID: "ag1", RunID: "*"}
		rangedResp, rangedErr := client.DeleteItem(ctx, &agentstorage.DeleteItemRequest{
			ItemScopedRequest: scoped(rangedScope), PathPrefix: "/bulk/", SessionID: "s1",
		})
		if rangedErr != nil {
			t.Fatalf("DeleteItem ranged: %v", rangedErr)
		}
		// The accept response is HTTP 200 with taskId/status (moved from 202
		// so the OTS gateway can forward it).
		x.check("DeleteItem ranged accept taskId",
			rangedResp.IsRanged() && rangedResp.Status == "pending", fmt.Sprint(rangedResp))
		x.expectServiceError("DeleteItem ranged with itemId -> 400", validationReject, func() error {
			_, err := client.DeleteItem(ctx, &agentstorage.DeleteItemRequest{
				ItemScopedRequest: scoped(wild), ItemID: verID,
			})
			return err
		})
		deadline := time.Now().Add(90 * time.Second)
		gone := false
		for time.Now().Before(deadline) {
			remaining, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{
				ItemScopedRequest: scoped(rangedScope), PathPrefix: "/bulk/", Limit: 100,
			})
			if err != nil {
				t.Fatalf("ListItems during ranged delete poll: %v", err)
			}
			if len(remaining.Items) == 0 && remaining.NextToken == "" {
				gone = true
				break
			}
			time.Sleep(2 * time.Second)
		}
		x.check("Ranged delete removed /bulk/*", gone, "files still present after 90s")
		keep, err := client.ListItems(ctx, &agentstorage.ListItemsRequest{ItemScopedRequest: scoped(s2), Limit: 100})
		if err != nil {
			t.Fatalf("ListItems after ranged delete: %v", err)
		}
		keepPaths := make([]string, 0, len(keep.Items))
		for _, entry := range keep.Items {
			keepPaths = append(keepPaths, entry.Path)
		}
		sort.Strings(keepPaths)
		x.check("Ranged delete kept /keep.md and /r2.md",
			fmt.Sprint(keepPaths) == fmt.Sprint([]string{"/keep.md", "/r2.md"}), fmt.Sprint(keepPaths))

	})

	t.Logf("Core Item checks: %d passed, %d failed (ranged deletion reported separately)", x.pass, x.fail)
	if x.fail > 0 {
		t.Fatalf("%d checks failed", x.fail)
	}
}
