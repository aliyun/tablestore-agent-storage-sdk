package agentstorage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func assertJSON(t *testing.T, value any, want string) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var gotObject, wantObject any
	if err := json.Unmarshal(data, &gotObject); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantObject); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotObject, wantObject) {
		t.Fatalf("JSON = %s, want %s", data, want)
	}
}

func TestCurrentMemoryRequestProtocol(t *testing.T) {
	filter := &ScopeFilter{Op: "and", Conditions: []*ScopeFilter{
		{Op: "eq", Field: "appId", Value: String("app")},
		{Op: "not", Condition: &ScopeFilter{Op: "prefix", Field: "runId", Value: String("excluded-")}},
	}}
	assertJSON(t, &CreateMemoryStoreRequest{MemoryStoreName: "store", InputType: StoreInputMessages, OutputTypes: []string{StoreOutputStructured, StoreOutputFile}, WikiReflectionEnabled: Bool(false)},
		`{"memoryStoreName":"store","inputType":"messages","outputTypes":["structured","file"],"wikiReflectionEnabled":false}`)
	assertJSON(t, &UpdateMemoryStoreRequest{MemoryStoreName: "store", Description: String(""), WikiReflectionEnabled: Bool(false)},
		`{"memoryStoreName":"store","description":"","wikiReflectionEnabled":false}`)
	assertJSON(t, &ListMemoryStoreScopesRequest{MemoryStoreName: "store", Filter: filter},
		`{"memoryStoreName":"store","filter":{"op":"and","conditions":[{"op":"eq","field":"appId","value":"app"},{"op":"not","condition":{"op":"prefix","field":"runId","value":"excluded-"}}]}}`)
	assertJSON(t, &SearchMemoriesRequest{MemoryStoreName: "store", Query: "coffee", Filter: &ScopeFilter{Op: "range", Field: "runId", GTE: String("001"), LT: String("010")}, Thresholds: &SearchThresholds{MinScore: .8, MinSimilarity: .3}, EnableRerank: Bool(false)},
		`{"memoryStoreName":"store","query":"coffee","filter":{"op":"range","field":"runId","gte":"001","lt":"010"},"thresholds":{"minScore":0.8,"minSimilarity":0.3},"enableRerank":false}`)
	assertJSON(t, &AddMemoriesRequest{MemoryStoreName: "store", Text: "feedback", Interaction: &InteractionRef{SessionID: "s", TurnID: "2", ReplyToTurnID: "1"}},
		`{"memoryStoreName":"store","text":"feedback","interaction":{"sessionId":"s","turnId":"2","replyToTurnId":"1"}}`)
	assertJSON(t, &ApplyMemoryDreamActionsRequest{MemoryStoreName: "store", DreamID: "d", Mode: DreamApplyRequestModePlan, Approver: "reviewer"},
		`{"memoryStoreName":"store","dreamId":"d","mode":"plan","approver":"reviewer"}`)
	assertJSON(t, &CreateMemoryDreamTaskRequest{MemoryStoreName: "store", Scopes: []Scope{{AppID: "a"}}, OutputScope: &Scope{AppID: "a", RunID: "out"}, MaxItems: 25, Effort: DreamEffortHigh},
		`{"memoryStoreName":"store","scopes":[{"appId":"a"}],"outputScope":{"appId":"a","runId":"out"},"maxItems":25,"effort":"high"}`)
}

func TestBinaryItemAndWikiOptionalFields(t *testing.T) {
	scoped := ItemScopedRequest{Type: ItemTypeMemoryFile, MemoryStoreName: "s", Scope: Scope{AppID: "a"}}
	assertJSON(t, AddItemRequest{ItemScopedRequest: scoped, Path: "/empty"},
		`{"type":"memoryfile","memoryStoreName":"s","scope":{"appId":"a"},"path":"/empty","content":""}`)
	assertJSON(t, AddItemRequest{ItemScopedRequest: scoped, Path: "/image.png", ContentBase64: String("AAEC"), MediaType: "image/png"},
		`{"type":"memoryfile","memoryStoreName":"s","scope":{"appId":"a"},"path":"/image.png","contentBase64":"AAEC","mediaType":"image/png"}`)
	for _, req := range []any{
		AddItemRequest{Content: "text", ContentBase64: String("AAEC")},
		UpdateItemRequest{ItemID: "id", Content: String(""), ContentBase64: String("AAEC")},
		UpdateItemRequest{ItemID: "id", ExpectedSHA256: "hash"},
		DeleteItemRequest{ExpectedSHA256: "hash"},
		ListItemVersionsRequest{ApiKeyID: "removed"},
	} {
		if _, err := json.Marshal(req); err == nil {
			t.Fatalf("expected serialization error for %T", req)
		}
	}
	ws := WikiScopedRequest{MemoryStoreName: "s", Scope: AgentScope{AppID: "a", TenantID: "t", AgentID: "ag"}}
	assertJSON(t, UpdateWikiRequest{WikiScopedRequest: ws, Summary: String("")},
		`{"memoryStoreName":"s","scope":{"appId":"a","tenantId":"t","agentId":"ag"},"summary":""}`)
	assertJSON(t, UpdateWikiRequest{WikiScopedRequest: ws, CompileOptions: Null[WikiCompileOptions]()},
		`{"memoryStoreName":"s","scope":{"appId":"a","tenantId":"t","agentId":"ag"},"compileOptions":null}`)
	assertJSON(t, UpdateWikiRequest{WikiScopedRequest: ws, CompileOptions: NullableOf(WikiCompileOptions{DocSummary: &WikiDocSummaryOptions{Enabled: false}, Incremental: &WikiIncrementalOptions{Enabled: false}})},
		`{"memoryStoreName":"s","scope":{"appId":"a","tenantId":"t","agentId":"ag"},"compileOptions":{"docSummary":{"enabled":false},"incremental":{"enabled":false}}}`)
}

func TestCurrentPublicResponses(t *testing.T) {
	taskJSON := `{"memoryStoreName":"s","dreamId":"d","status":"awaiting_plan_approval","effort":"high","stage":"executing","failureCode":"no_submission","outputScope":{"appId":"a","runId":"out"},"maxItems":25,"input":{"scopes":[],"expandedScopes":[{"appId":"a"}],"itemCount":3,"sessionCount":2},"actions":{"total":3,"applied":1,"skipped":1,"skippedByReason":{"missing_source":1}},"plan":{"summary":"Review","deleteMemoryIds":[],"approvalExpiresAt":"2026-10-26T09:00:00Z"},"diagnostic":{"schemaVersion":1,"taskId":"d","omittedFailures":2,"failures":[{"code":"E","messageTruncated":true,"observedAtMs":1793000000000,"recoveryRemaining":0}]},"finishedAt":null}`
	var got GetMemoryDreamTaskResponse
	if err := json.Unmarshal([]byte(taskJSON), &got); err != nil {
		t.Fatal(err)
	}
	var listed ListMemoryDreamTasksResponse
	if err := json.Unmarshal([]byte(`{"tasks":[`+taskJSON+`],"nextToken":"opaque"}`), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Tasks) != 1 || !reflect.DeepEqual(listed.Tasks[0].Input, got.Input) || !reflect.DeepEqual(listed.Tasks[0].Actions, got.Actions) {
		t.Fatalf("list dropped nested input/actions: %+v", listed)
	}
	if got.Plan == nil || got.Plan.DeleteMemoryIDs == nil || len(got.Plan.DeleteMemoryIDs) != 0 || got.Status != DreamTaskStatusAwaitingPlanApproval || got.Input.ItemCount != 3 || got.Actions.SkippedByReason["missing_source"] != 1 {
		t.Fatalf("task=%+v", got)
	}
	if got.Diagnostic == nil || len(got.Diagnostic.Failures) != 1 || got.Diagnostic.Failures[0].RecoveryRemaining == nil || *got.Diagnostic.Failures[0].RecoveryRemaining != 0 {
		t.Fatalf("diagnostic=%+v", got.Diagnostic)
	}
	var wiki GetWikiResponse
	err := json.Unmarshal([]byte(`{"name":"Agent","status":"building","pageCount":4,"hasPendingChanges":true,"latestBuild":{"status":"running","partialAvailable":true,"completedPages":4,"plannedPages":9,"completedSources":2,"totalSources":3,"failureKind":"transient","attemptFailures":[{"attempt":1,"kind":"timeout","retryable":true}],"quality":{"groundedPageRate":0.9},"diagnostic":{"omittedFailures":0,"failures":[{"code":"SOURCE","observedAtMs":1793000000000}]}}}`), &wiki)
	if err != nil {
		t.Fatal(err)
	}
	if wiki.LatestBuild == nil || !wiki.LatestBuild.PartialAvailable || wiki.LatestBuild.CompletedPages != 4 || wiki.LatestBuild.Quality == nil || wiki.LatestBuild.Quality.GroundedPageRate != .9 || wiki.LatestBuild.Diagnostic == nil {
		t.Fatalf("wiki=%+v", wiki)
	}
	var item Item
	if err := json.Unmarshal([]byte(`{"itemId":"i","contentBase64":"AAEC","mediaType":"image/png","summary":"image","latestVersionId":"v"}`), &item); err != nil {
		t.Fatal(err)
	}
	if item.Content != nil || item.ContentBase64 == nil || *item.ContentBase64 != "AAEC" || item.MediaType != "image/png" || item.Summary != "image" {
		t.Fatalf("item=%+v", item)
	}
	var ingest GetMemoryTaskResponse
	if err := json.Unmarshal([]byte(`{"task":{"status":"failed","primaryError":"source","primaryErrorStage":"parse","primaryErrorClass":"input","lastErrorStage":"extract","lastErrorClass":"permanent","lastErrorInputRejected":true}}`), &ingest); err != nil {
		t.Fatal(err)
	}
	if ingest.Task.PrimaryErrorStage != "parse" || !ingest.Task.LastErrorInputRejected {
		t.Fatalf("task=%+v", ingest.Task)
	}
}

func TestNewActionsDispatch(t *testing.T) {
	ctx := context.Background()
	ws := WikiScopedRequest{MemoryStoreName: "s", Scope: AgentScope{AppID: "a", TenantID: "t", AgentID: "ag"}}
	is := ItemScopedRequest{MemoryStoreName: "s", Scope: Scope{AppID: "a", TenantID: "t", AgentID: "ag", RunID: "r"}}
	cases := []struct {
		action string
		call   func(*Client) error
	}{
		{"GetWiki", func(c *Client) error { _, e := c.GetWiki(ctx, &GetWikiRequest{WikiScopedRequest: ws}); return e }},
		{"UpdateWiki", func(c *Client) error {
			_, e := c.UpdateWiki(ctx, &UpdateWikiRequest{WikiScopedRequest: ws, Name: String("name")})
			return e
		}},
		{"IngestWiki", func(c *Client) error {
			_, e := c.IngestWiki(ctx, &IngestWikiRequest{WikiScopedRequest: ws, ClientToken: "token"})
			return e
		}},
		{"ListWikiPages", func(c *Client) error {
			_, e := c.ListWikiPages(ctx, &ListWikiPagesRequest{WikiScopedRequest: ws, NextToken: "opaque"})
			return e
		}},
		{"GetWikiPage", func(c *Client) error {
			_, e := c.GetWikiPage(ctx, &GetWikiPageRequest{WikiScopedRequest: ws, Path: "/index.md"})
			return e
		}},
		{"SearchWikiPages", func(c *Client) error {
			_, e := c.SearchWikiPages(ctx, &SearchWikiPagesRequest{WikiScopedRequest: ws, Query: "coffee"})
			return e
		}},
		{"SearchItems", func(c *Client) error {
			_, e := c.SearchItems(ctx, &SearchItemsRequest{ItemScopedRequest: is, Query: "coffee"})
			return e
		}},
		{"DeleteItemVersions", func(c *Client) error {
			_, e := c.DeleteItemVersions(ctx, &DeleteItemVersionsRequest{ItemScopedRequest: is})
			return e
		}},
	}
	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			c := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/"+tc.action || r.Method != "POST" {
					t.Errorf("wrong action %s %s", r.Method, r.URL.Path)
				}
				var b map[string]any
				if e := json.NewDecoder(r.Body).Decode(&b); e != nil {
					t.Error(e)
				}
				if b["memoryStoreName"] != "s" {
					t.Errorf("missing store: %v", b)
				}
				scope, ok := b["scope"].(map[string]any)
				if !ok {
					t.Error("scope missing")
					return
				}
				if tc.action == "SearchItems" || tc.action == "DeleteItemVersions" {
					if b["type"] != "memoryfile" || scope["runId"] != "r" {
						t.Errorf("item scope/type=%v", b)
					}
				} else if _, ok := scope["runId"]; ok {
					t.Error("Wiki scope must not include runId")
				}
				fmt.Fprint(w, `{}`)
			})
			if e := tc.call(c); e != nil {
				t.Fatal(e)
			}
		})
	}
	if is.Type != "" {
		t.Fatal("caller mutated")
	}
}

func TestLegacyItemGuardPreservesAtomicity(t *testing.T) {
	for _, action := range []string{"UpdateItem", "DeleteItem"} {
		for _, scenario := range []string{"success", "hash mismatch", "version missing", "concurrent conflict"} {
			t.Run(action+"/"+scenario, func(t *testing.T) {
				reads, writes := 0, 0
				client := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					var b map[string]any
					if e := json.NewDecoder(r.Body).Decode(&b); e != nil {
						t.Error(e)
					}
					if r.URL.Path == "/GetItem" {
						reads++
						if b["view"] != "basic" {
							t.Errorf("guard read must not fetch content: %v", b)
						}
						if b["path"] != "/old" {
							t.Errorf("get body=%v", b)
						}
						version := "v1"
						if scenario == "version missing" {
							version = ""
						}
						fmt.Fprintf(w, `{"itemId":"i","contentSha256":"hash","latestVersionId":%q}`, version)
						return
					}
					writes++
					if r.URL.Path != "/"+action {
						t.Errorf("action=%s", r.URL.Path)
					}
					if _, ok := b["expectedSha256"]; ok {
						t.Error("obsolete hash on wire")
					}
					if _, ok := b["newPath"]; ok {
						t.Error("obsolete newPath on wire")
					}
					pre, ok := b["precondition"].(map[string]any)
					if !ok || pre["expectedVersionId"] != "v1" || b["itemId"] != "i" {
						t.Errorf("guard lost: %v", b)
					}
					if action == "UpdateItem" && b["path"] != "/new" {
						t.Errorf("rename lost: %v", b)
					}
					if action == "UpdateItem" && b["view"] != "full" {
						t.Errorf("update response view lost: %v", b)
					}
					if action == "DeleteItem" {
						if _, ok := b["path"]; ok {
							t.Error("ambiguous delete address")
						}
					}
					if scenario == "concurrent conflict" {
						w.WriteHeader(409)
						fmt.Fprint(w, `{"code":"PRECONDITION_FAILED","message":"version changed"}`)
						return
					}
					fmt.Fprint(w, `{"itemId":"i","latestVersionId":"v2","deleted":true}`)
				})
				sha := "hash"
				if scenario == "hash mismatch" {
					sha = "wrong"
				}
				scoped := ItemScopedRequest{MemoryStoreName: "s"}
				var err error
				if action == "UpdateItem" {
					req := &UpdateItemRequest{ItemScopedRequest: scoped, Path: String("/old"), NewPath: "/new", Content: String(""), ExpectedSHA256: sha, View: ItemViewFull}
					_, err = client.UpdateItem(context.Background(), req)
					if req.ItemID != "" || req.Path == nil || *req.Path != "/old" || req.Precondition != nil || req.Type != "" || req.ExpectedSHA256 != sha || req.View != ItemViewFull {
						t.Fatal("caller mutated")
					}
				} else {
					_, err = client.DeleteItem(context.Background(), &DeleteItemRequest{ItemScopedRequest: scoped, Path: "/old", ExpectedSHA256: sha})
				}
				if reads != 1 {
					t.Errorf("reads=%d", reads)
				}
				if scenario == "hash mismatch" || scenario == "version missing" {
					var clientErr *ClientError
					if !errors.As(err, &clientErr) || writes != 0 {
						t.Fatalf("must fail before write: err=%v writes=%d", err, writes)
					}
				} else if scenario == "concurrent conflict" {
					var serviceErr *ServiceError
					if !errors.As(err, &serviceErr) || serviceErr.StatusCode != 409 || writes != 1 {
						t.Fatalf("conflict must not retry: %v writes=%d", err, writes)
					}
				} else if err != nil || writes != 1 {
					t.Fatalf("err=%v writes=%d", err, writes)
				}
			})
		}
	}
}

func TestLegacyItemNoOpPreservesView(t *testing.T) {
	for _, view := range []ItemView{ItemViewBasic, ItemViewFull} {
		t.Run(string(view), func(t *testing.T) {
			calls := 0
			client := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if r.URL.Path != "/GetItem" || body["path"] != "/old" || body["view"] != string(view) {
					t.Errorf("no-op must read the requested view: action=%s body=%v", r.URL.Path, body)
				}
				response := `{"itemId":"i","contentSha256":"hash","latestVersionId":"v1"}`
				if view == ItemViewFull {
					response = `{"itemId":"i","contentSha256":"hash","latestVersionId":"v1","content":"original"}`
				}
				fmt.Fprint(w, response)
			})
			item, err := client.UpdateItem(context.Background(), &UpdateItemRequest{
				Path: String("/old"), ExpectedSHA256: "hash", View: view,
			})
			if err != nil || calls != 1 {
				t.Fatalf("no-op must not write: err=%v calls=%d", err, calls)
			}
			if view == ItemViewFull {
				if item.Content == nil || *item.Content != "original" {
					t.Fatalf("full response lost original content: %+v", item)
				}
			} else if item.Content != nil {
				t.Fatalf("basic response unexpectedly included content: %+v", item)
			}
		})
	}
}

func TestScopeDirectoryRetainsHeterogeneousEntries(t *testing.T) {
	wire := `{"memoryStoreName":"s","scope":{},"scopes":[{"scopeType":"conversation","scopeKey":"c1","appId":"a","tenantId":"t","agentId":"ag","runId":"r"},{"scopeType":"wiki","scopeKey":"w1","appId":"a","tenantId":"t","agentId":"ag","wikiId":"wiki-1"}],"nextToken":"opaque"}`
	var r ListMemoryStoreScopesResponse
	if err := json.Unmarshal([]byte(wire), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Scopes) != 2 || len(r.ScopeEntries) != 2 || r.Scopes[0].RunID != "r" || r.ScopeEntries[1].ScopeType != ScopeTypeWiki || r.ScopeEntries[1].ScopeKey != "w1" || r.ScopeEntries[1].Dimensions["wikiId"] != "wiki-1" {
		t.Fatalf("directory fields lost: %+v", r)
	}
	assertJSON(t, r, wire)
	// Older servers return only the four dimensions; both views remain usable.
	if err := json.Unmarshal([]byte(`{"memoryStoreName":"s","scope":{},"scopes":[{"appId":"a","runId":"r"}]}`), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Scopes) != 1 || r.ScopeEntries[0].ScopeType != "" || r.Scopes[0].AppID != "a" {
		t.Fatalf("legacy scope decode=%+v", r)
	}
}

func TestOptionalScopeAndExplicitEmptyUpdates(t *testing.T) {
	assertJSON(t, GetMemoryRequest{MemoryStoreName: "s", MemoryID: "m"}, `{"memoryStoreName":"s","memoryId":"m"}`)
	assertJSON(t, DeleteMemoryRequest{MemoryStoreName: "s", MemoryID: "m"}, `{"memoryStoreName":"s","memoryId":"m"}`)
	assertJSON(t, UpdateMemoryRequest{MemoryStoreName: "s", MemoryID: "m", Metadata: map[string]string{}}, `{"memoryStoreName":"s","memoryId":"m","metadata":{}}`)
	assertJSON(t, CreateMemoryStoreRequest{MemoryStoreName: "s", OutputTypes: []string{}}, `{"memoryStoreName":"s","outputTypes":[]}`)
	assertJSON(t, UpdateMemoryStoreRequest{MemoryStoreName: "s", OutputTypes: []string{}}, `{"memoryStoreName":"s","outputTypes":[]}`)
	var stores ListMemoryStoresResponse
	if err := json.Unmarshal([]byte(`{"stores":[{"memoryStoreName":"s","inputType":"file","outputTypes":["file"],"storeUid":"uid","wikiReflectionEnabled":true}]}`), &stores); err != nil {
		t.Fatal(err)
	}
	if stores.Stores[0].InputType != StoreInputFile || stores.Stores[0].StoreUID != "uid" || !stores.Stores[0].WikiReflectionEnabled {
		t.Fatalf("store list fields lost: %+v", stores)
	}
	var added AddMemoriesResponse
	if err := json.Unmarshal([]byte(`{"requestId":"r","memoryIds":["m1","m2"],"unitsCreated":2}`), &added); err != nil {
		t.Fatal(err)
	}
	if len(added.MemoryIDs) != 2 {
		t.Fatalf("sync memory IDs lost: %+v", added)
	}
}

func TestCurrentItemUpdateNeedsNoCompatibilityRead(t *testing.T) {
	calls := 0
	c := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/UpdateItem" {
			t.Errorf("unexpected compatibility read %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := map[string]any{"type": "memoryfile", "memoryStoreName": "s", "scope": map[string]any{}, "itemId": "i", "path": "/new", "contentBase64": "AAEC", "mediaType": "image/png", "precondition": map[string]any{"expectedVersionId": "v1"}}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("body=%v, want %v", body, want)
		}
		fmt.Fprint(w, `{"itemId":"i","latestVersionId":"v2","contentBase64":"AAEC","mediaType":"image/png"}`)
	})
	pre := &ItemPrecondition{ExpectedVersionID: "v1"}
	req := &UpdateItemRequest{ItemScopedRequest: ItemScopedRequest{MemoryStoreName: "s"}, ItemID: "i", Path: String("/new"), ContentBase64: String("AAEC"), MediaType: "image/png", Precondition: pre}
	result, err := c.UpdateItem(context.Background(), req)
	if err != nil || calls != 1 || result.LatestVersionID != "v2" {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
	}
	if req.Precondition != pre || req.Type != "" || req.Path == nil || *req.Path != "/new" {
		t.Fatal("caller request mutated")
	}
}

func TestDreamListLegacyFieldsRemainUsable(t *testing.T) {
	var old, current DreamTask
	if err := json.Unmarshal([]byte(`{"dreamId":"d","sessionCount":2,"actionCount":3,"appliedCount":1,"inputScopes":[{"appId":"a"}]}`), &old); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"dreamId":"d","input":{"sessionCount":2,"scopes":[{"appId":"a"}]},"actions":{"total":3,"applied":1}}`), &current); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(old.Input, current.Input) || !reflect.DeepEqual(old.Actions, current.Actions) || current.ActionCount != 3 || current.SessionCount != 2 || len(current.InputScopes) != 1 {
		t.Fatalf("old=%+v current=%+v", old, current)
	}
}

func TestItemCompatibilityRejectsAmbiguousTargets(t *testing.T) {
	calls := 0
	c := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{}`)
	})
	_, err := c.DeleteItem(context.Background(), &DeleteItemRequest{
		ItemID: "one", Path: "/another.md", ExpectedSHA256: "hash",
	})
	var clientErr *ClientError
	if !errors.As(err, &clientErr) || clientErr.Kind() != ClientErrorKindRequestBuild || calls != 0 {
		t.Fatalf("ambiguous delete must fail before reading or writing: err=%v calls=%d", err, calls)
	}
	_, err = c.UpdateItem(context.Background(), &UpdateItemRequest{
		ItemID: "one", Path: String("/one.md"), NewPath: "/another.md",
	})
	if !errors.As(err, &clientErr) || clientErr.Kind() != ClientErrorKindRequestSerialize || calls != 0 {
		t.Fatalf("ambiguous rename must fail before writing: err=%v calls=%d", err, calls)
	}
}

func TestItemUpdatePreservesExplicitEmptyPath(t *testing.T) {
	// Preserve the current public pointer contract: explicit empty paths must
	// reach server validation instead of becoming an omitted rename.
	assertJSON(t, UpdateItemRequest{ItemID: "i", Path: String(""), Content: String("updated")},
		`{"type":"","memoryStoreName":"","scope":{},"itemId":"i","path":"","content":"updated"}`)
}

func TestItemUpdateMissingIDCannotOverwriteDestination(t *testing.T) {
	calls := 0
	c := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		// The destination already exists. A compatibility read must not turn
		// this malformed current request into a write to that unrelated item.
		fmt.Fprint(w, `{"itemId":"unrelated","latestVersionId":"v1"}`)
	})
	req := &UpdateItemRequest{Path: String("/destination.md"), Content: String("new text")}
	_, err := c.UpdateItem(context.Background(), req)
	var clientErr *ClientError
	if !errors.As(err, &clientErr) || clientErr.Kind() != ClientErrorKindRequestBuild || calls != 0 {
		t.Fatalf("missing ItemID must fail without reading or writing: err=%v calls=%d", err, calls)
	}
	if req.ItemID != "" || *req.Path != "/destination.md" || req.Type != "" {
		t.Fatal("caller request mutated")
	}
}

func TestItemUpdateExplicitLegacyRenameWithoutSHA(t *testing.T) {
	reads, writes := 0, 0
	c := newAPIKeyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatal(err)
		}
		switch r.URL.Path {
		case "/GetItem":
			reads++
			if b["path"] != "/source.md" {
				t.Errorf("wrong legacy address: %v", b)
			}
		case "/UpdateItem":
			writes++
			if b["itemId"] != "source" || b["path"] != "/destination.md" {
				t.Errorf("wrong rename target: %v", b)
			}
		default:
			t.Errorf("unexpected action %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"itemId":"source","latestVersionId":"v1"}`)
	})
	_, err := c.UpdateItem(context.Background(), &UpdateItemRequest{Path: String("/source.md"), NewPath: "/destination.md"})
	if err != nil || reads != 1 || writes != 1 {
		t.Fatalf("explicit legacy rename: err=%v reads=%d writes=%d", err, reads, writes)
	}
}
