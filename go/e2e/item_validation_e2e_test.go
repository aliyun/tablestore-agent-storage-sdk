package e2e

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	sdk "github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

// Reject authentication/internal errors: a failing request alone is not proof
// that the server validated its parameters. Require the public error identity
// and the gateway request ID, including the legacy validation-code mapping.
func expectItemValidation(t *testing.T, err error) {
	t.Helper()
	var serviceErr *sdk.ServiceError
	if !errors.As(err, &serviceErr) {
		t.Fatalf("expected server parameter rejection, got %v", err)
	}
	valid := serviceErr.StatusCode == 400 && (serviceErr.Code == "OTSParameterInvalid" || serviceErr.Code == "INVALID_PARAMETER" || serviceErr.Code == "VALIDATION_ERROR") ||
		serviceErr.StatusCode == 403 && serviceErr.Code == "OTSValidationFail"
	if !valid || serviceErr.RequestID == "" {
		t.Fatalf("wrong parameter error or missing request ID: %v", err)
	}
	t.Logf("rejected: status=%d code=%s requestId=%s", serviceErr.StatusCode, serviceErr.Code, serviceErr.RequestID)
}

type itemInvalidCase struct {
	name, action string
	fields       map[string]any
}

// These raw requests deliberately bypass typed helpers that reject malformed
// payloads locally. They still use the SDK's normal signed gateway transport.
func runItemInvalidCases(t *testing.T, c *sdk.Client, scoped sdk.ItemScopedRequest, defaults map[string]map[string]any, cases []itemInvalidCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.action+"/"+tc.name, func(t *testing.T) {
			body := map[string]any{"type": sdk.ItemTypeMemoryFile, "memoryStoreName": scoped.MemoryStoreName, "scope": scoped.Scope}
			for k, v := range defaults[tc.action] {
				body[k] = v
			}
			for k, v := range tc.fields {
				if v == nil {
					delete(body, k)
				} else {
					body[k] = v
				}
			}
			err := c.Do(context.Background(), tc.action, body, nil)
			// Strict JSON decoding uses BAD_REQUEST (OTSResponseFail at the
			// gateway). SearchItems explicitly uses INVALID_PARAMETER for it.
			unknownField := tc.name == "unknown field" || strings.HasPrefix(tc.name, "unsupported ")
			if unknownField && tc.action != "SearchItems" {
				var serviceErr *sdk.ServiceError
				if !errors.As(err, &serviceErr) || serviceErr.StatusCode != 400 || (serviceErr.Code != "BAD_REQUEST" && serviceErr.Code != "OTSResponseFail") || serviceErr.RequestID == "" {
					t.Fatalf("expected strict JSON rejection, got %v", err)
				}
				t.Logf("strict JSON rejection: code=%s requestId=%s", serviceErr.Code, serviceErr.RequestID)
			} else {
				expectItemValidation(t, err)
			}
		})
	}
}

func TestItemParameterValidationE2E(t *testing.T) {
	c := e2eClient(t)
	ctx := context.Background()
	store := currentStore(t, c, sdk.StoreInputFile, []string{sdk.StoreOutputFile})
	scope := fullScope("itemparams")
	scoped := sdk.ItemScopedRequest{MemoryStoreName: store, Scope: scope}
	item, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: scoped, Path: "/fixture.md", Content: "must remain unchanged"})
	if err != nil {
		t.Fatal(err)
	}
	wild := scope
	wild.RunID = "*"
	badHierarchy := scope
	badHierarchy.AgentID = "*"
	defaults := map[string]map[string]any{
		"AddItem":            {"path": "/new.md", "content": "new"},
		"GetItem":            {"itemId": item.ItemID},
		"UpdateItem":         {"itemId": item.ItemID, "content": "must not be written"},
		"DeleteItem":         {"itemId": item.ItemID},
		"ListItems":          {},
		"ListItemVersions":   {},
		"GetItemVersion":     {"versionId": item.LatestVersionID},
		"RedactItemVersion":  {"versionId": item.LatestVersionID},
		"DeleteItemVersions": {"scope": wild},
	}
	cases := []itemInvalidCase{
		{"missing content", "AddItem", map[string]any{"content": nil}},
		{"both content fields", "AddItem", map[string]any{"contentBase64": "eA==", "mediaType": "image/png"}},
		{"binary missing media type", "AddItem", map[string]any{"content": nil, "contentBase64": "eA=="}},
		{"invalid base64", "AddItem", map[string]any{"content": nil, "contentBase64": "!not-base64!", "mediaType": "image/png"}},
		{"media type on text", "AddItem", map[string]any{"mediaType": "text/markdown"}},
		{"ordinary non markdown", "AddItem", map[string]any{"path": "/new.txt"}},
		{"ordinary binary path", "AddItem", map[string]any{"path": "/pixel.png", "content": nil, "contentBase64": "iVBORw0KGgo=", "mediaType": "image/png"}},
		{"wildcard write", "AddItem", map[string]any{"scope": wild}},
		{"missing address", "GetItem", map[string]any{"itemId": nil}},
		{"ambiguous address", "GetItem", map[string]any{"path": "/fixture.md"}},
		{"wildcard read", "GetItem", map[string]any{"scope": wild}},
		{"missing item id", "UpdateItem", map[string]any{"itemId": nil, "path": "/fixture.md"}},
		{"missing change", "UpdateItem", map[string]any{"content": nil}},
		{"blank destination", "UpdateItem", map[string]any{"path": " "}},
		{"overwrite without destination", "UpdateItem", map[string]any{"overwrite": true}},
		{"both content fields", "UpdateItem", map[string]any{"contentBase64": "eA==", "mediaType": "image/png"}},
		{"binary missing media type", "UpdateItem", map[string]any{"content": nil, "contentBase64": "eA=="}},
		{"invalid base64", "UpdateItem", map[string]any{"content": nil, "contentBase64": "!", "mediaType": "image/png"}},
		{"media type on text", "UpdateItem", map[string]any{"mediaType": "text/markdown"}},
		{"empty precondition", "UpdateItem", map[string]any{"precondition": map[string]any{}}},
		{"invalid precondition", "UpdateItem", map[string]any{"precondition": map[string]any{"expectedVersionId": "bad"}}},
		{"missing address", "DeleteItem", map[string]any{"itemId": nil}},
		{"ambiguous address", "DeleteItem", map[string]any{"path": "/fixture.md"}},
		{"exact prefix", "DeleteItem", map[string]any{"pathPrefix": "/notes/"}},
		{"exact delete versions", "DeleteItem", map[string]any{"deleteVersions": true}},
		{"empty precondition", "DeleteItem", map[string]any{"precondition": map[string]any{}}},
		{"invalid precondition", "DeleteItem", map[string]any{"precondition": map[string]any{"expectedVersionId": "bad"}}},
		{"ranged item id", "DeleteItem", map[string]any{"scope": wild}},
		{"ranged path", "DeleteItem", map[string]any{"scope": wild, "itemId": nil, "path": "/fixture.md"}},
		{"ranged precondition", "DeleteItem", map[string]any{"scope": wild, "itemId": nil, "precondition": map[string]any{"expectedVersionId": item.LatestVersionID}}},
		{"ranged versions with prefix", "DeleteItem", map[string]any{"scope": wild, "itemId": nil, "deleteVersions": true, "pathPrefix": "/notes/"}},
		{"ranged unaligned prefix", "DeleteItem", map[string]any{"scope": wild, "itemId": nil, "pathPrefix": "/notes"}},
		{"invalid hierarchy", "DeleteItem", map[string]any{"scope": badHierarchy, "itemId": nil}},
		{"negative depth", "ListItems", map[string]any{"depth": -1}},
		{"excess depth", "ListItems", map[string]any{"depth": 2}},
		{"unaligned prefix", "ListItems", map[string]any{"pathPrefix": "/notes"}},
		{"invalid hierarchy", "ListItems", map[string]any{"scope": badHierarchy}},
		{"wiki filter", "ListItems", map[string]any{"pathPrefix": "/.wiki/raw/", "filter": map[string]any{"op": "exists", "field": "appId"}}},
		{"ambiguous address", "ListItemVersions", map[string]any{"itemId": item.ItemID, "path": "/fixture.md"}},
		{"path with wildcard", "ListItemVersions", map[string]any{"scope": wild, "path": "/fixture.md"}},
		{"invalid operation", "ListItemVersions", map[string]any{"operation": "renamed"}},
		{"invalid timestamp", "ListItemVersions", map[string]any{"createdAtGte": "yesterday"}},
		{"reversed time range", "ListItemVersions", map[string]any{"createdAtGte": "2026-02-02T00:00:00Z", "createdAtLte": "2026-02-01T00:00:00Z"}},
		{"missing version", "GetItemVersion", map[string]any{"versionId": nil}},
		{"invalid version", "GetItemVersion", map[string]any{"versionId": "bad"}},
		{"missing version", "RedactItemVersion", map[string]any{"versionId": nil}},
		{"invalid version", "RedactItemVersion", map[string]any{"versionId": "bad"}},
		{"wildcard redact", "RedactItemVersion", map[string]any{"scope": wild}},
		{"exact scope", "DeleteItemVersions", map[string]any{"scope": scope}},
		{"invalid hierarchy", "DeleteItemVersions", map[string]any{"scope": badHierarchy}},
	}
	for _, action := range []string{"AddItem", "GetItem", "UpdateItem", "DeleteItem", "ListItems", "ListItemVersions", "GetItemVersion", "RedactItemVersion", "DeleteItemVersions"} {
		cases = append(cases, itemInvalidCase{"unknown field", action, map[string]any{"unsupportedField": true}}, itemInvalidCase{"invalid type", action, map[string]any{"type": "document"}})
	}
	for _, action := range []string{"ListItems", "ListItemVersions"} {
		for _, limit := range []int{-1, 0, 101} {
			cases = append(cases, itemInvalidCase{fmt.Sprintf("limit %d", limit), action, map[string]any{"limit": limit}})
		}
		cases = append(cases, itemInvalidCase{"full limit 21", action, map[string]any{"view": "full", "limit": 21}})
	}
	for _, action := range []string{"AddItem", "GetItem", "UpdateItem", "ListItems", "ListItemVersions", "GetItemVersion"} {
		cases = append(cases, itemInvalidCase{"invalid view", action, map[string]any{"view": "all"}})
	}
	for _, path := range []string{"no-slash.md", "/../bad.md", "/a//bad.md", "/bad\x00.md", "/" + strings.Repeat("x", 1024) + ".md"} {
		cases = append(cases, itemInvalidCase{fmt.Sprintf("invalid path %d", len(cases)), "AddItem", map[string]any{"path": path}})
	}
	for name, value := range map[string]string{"scope slash": "bad/app", "scope nul": "bad\x00app", "scope bytes": strings.Repeat("界", 43)} {
		s := scope
		s.AppID = value
		cases = append(cases, itemInvalidCase{name, "AddItem", map[string]any{"scope": s}})
	}
	for _, field := range []string{"itemId", "versionId", "path", "pathPrefix", "sessionId", "createdAtGte", "operation", "filter", "precondition"} {
		cases = append(cases, itemInvalidCase{"unsupported " + field, "DeleteItemVersions", map[string]any{field: "not-allowed"}})
	}
	runItemInvalidCases(t, c, scoped, defaults, cases)
	// Rejection must leave both the current item and its immutable history intact.
	current, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, View: sdk.ItemViewFull})
	if err != nil || current.LatestVersionID != item.LatestVersionID || current.Content == nil || *current.Content != "must remain unchanged" {
		t.Fatalf("invalid mutations changed fixture: %+v %v", current, err)
	}
	version, err := c.GetItemVersion(ctx, &sdk.GetItemVersionRequest{ItemScopedRequest: scoped, VersionID: item.LatestVersionID, View: sdk.ItemViewFull})
	if err != nil || version.Redacted || version.Content == nil || *version.Content != "must remain unchanged" {
		t.Fatalf("invalid mutations changed history: %+v %v", version, err)
	}
	t.Run("UpdateItem explicit empty and no-op", func(t *testing.T) {
		cleared, err := c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Content: sdk.String(""), Precondition: &sdk.ItemPrecondition{ExpectedVersionID: item.LatestVersionID}, View: sdk.ItemViewFull})
		if err != nil || cleared.Content == nil || *cleared.Content != "" || cleared.ContentSizeBytes != 0 || cleared.LatestVersionID == item.LatestVersionID {
			t.Fatalf("explicit empty content: %+v %v", cleared, err)
		}
		unchanged, err := c.UpdateItem(ctx, &sdk.UpdateItemRequest{ItemScopedRequest: scoped, ItemID: item.ItemID, Content: sdk.String(""), Precondition: &sdk.ItemPrecondition{ExpectedVersionID: cleared.LatestVersionID}})
		if err != nil || unchanged.LatestVersionID != cleared.LatestVersionID {
			t.Fatalf("no-op unexpectedly appended a version: %+v %v", unchanged, err)
		}
	})
	checkItemListFilters(t, c, scoped, item.ItemID)
	t.Run("SearchItems rejects file input", func(t *testing.T) {
		_, err := c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: scoped, Query: "fixture"})
		expectItemValidation(t, err)
	})
}

func checkItemListFilters(t *testing.T, c *sdk.Client, scoped sdk.ItemScopedRequest, fixtureID string) {
	t.Helper()
	t.Run("ListItems/filter isolation and boundaries", func(t *testing.T) {
		ctx := context.Background()
		other := scoped
		other.Scope.RunID += "_other"
		if _, err := c.AddItem(ctx, &sdk.AddItemRequest{ItemScopedRequest: other, Path: "/other.md", Content: "outside requested run"}); err != nil {
			t.Fatal(err)
		}
		wild := scoped
		wild.Scope.RunID = "*"
		filter := &sdk.ScopeFilter{Op: "eq", Field: "runId", Value: sdk.String(scoped.Scope.RunID)}
		first := poll(t, 90*time.Second, time.Second, "filtered file scope visible", func() (*sdk.ListItemsResponse, error) {
			return c.ListItems(ctx, &sdk.ListItemsRequest{ItemScopedRequest: wild, Filter: filter, Limit: 1, View: sdk.ItemViewFull})
		}, func(r *sdk.ListItemsResponse) bool {
			return len(r.Items) == 1 && r.Items[0].ItemID == fixtureID && r.Items[0].Scope == scoped.Scope && r.Items[0].Content != nil
		})
		// A correct first page alone cannot prove isolation: an ignored filter
		// could put the other run on page two. Exhaust any continuation pages.
		seen := map[string]bool{}
		for next := first.NextToken; next != ""; {
			if seen[next] || len(seen) >= 100 {
				t.Fatal("filtered list pagination did not terminate")
			}
			seen[next] = true
			page, err := c.ListItems(ctx, &sdk.ListItemsRequest{ItemScopedRequest: wild, Filter: filter, Limit: 1, View: sdk.ItemViewFull, NextToken: next})
			if err != nil || len(page.Items) != 0 {
				t.Fatalf("filtered continuation leaked or duplicated items: %+v %v", page, err)
			}
			next = page.NextToken
		}
		// A conflicting filter is ANDed with the exact scope, not allowed to widen it.
		filter.Value = sdk.String(other.Scope.RunID)
		result, err := c.ListItems(ctx, &sdk.ListItemsRequest{ItemScopedRequest: scoped, Filter: filter})
		if err != nil || len(result.Items) != 0 || result.NextToken != "" {
			t.Fatalf("filter escaped exact scope: %+v %v", result, err)
		}
		for _, limit := range []int{1, 20, 100} {
			result, err := c.ListItems(ctx, &sdk.ListItemsRequest{ItemScopedRequest: scoped, Limit: limit})
			if err != nil || len(result.Items) != 1 || result.Items[0].ItemID != fixtureID {
				t.Fatalf("valid limit %d: %+v %v", limit, result, err)
			}
		}
	})
}

// Search parameters must be tested on a message-input store that can actually
// search. A wrong store mode would reject every request and mask missing checks.
func checkSearchItemParameters(t *testing.T, c *sdk.Client, scoped sdk.ItemScopedRequest, query, itemID string) {
	t.Helper()
	ctx := context.Background()
	wild := scoped.Scope
	wild.AgentID = "*"
	defaults := map[string]map[string]any{"SearchItems": {"query": query}}
	cases := []itemInvalidCase{
		{"empty query", "SearchItems", map[string]any{"query": " "}},
		{"negative topK", "SearchItems", map[string]any{"topK": -1}},
		{"topK 51", "SearchItems", map[string]any{"topK": 51}},
		{"invalid hierarchy", "SearchItems", map[string]any{"scope": wild}},
		{"missing app", "SearchItems", map[string]any{"scope": sdk.Scope{TenantID: scoped.Scope.TenantID}}},
		{"wildcard app", "SearchItems", map[string]any{"scope": sdk.Scope{AppID: "*", TenantID: "*", AgentID: "*", RunID: "*"}}},
	}
	for _, field := range []string{"minScore", "minSimilarity"} {
		for _, value := range []float64{-0.01, 1.01} {
			cases = append(cases, itemInvalidCase{fmt.Sprintf("%s %g", field, value), "SearchItems", map[string]any{"thresholds": map[string]any{field: value}}})
		}
	}
	for _, field := range []string{"contextScope", "metadata", "enableRerank", "includeEvidence", "minSimilarity", "pathPrefix", "nextToken"} {
		cases = append(cases, itemInvalidCase{"unsupported " + field, "SearchItems", map[string]any{field: true}})
	}
	for name, filter := range map[string]any{
		"unknown operator":  map[string]any{"op": "invalid", "field": "runId", "value": "x"},
		"missing eq value":  map[string]any{"op": "eq", "field": "runId"},
		"empty in":          map[string]any{"op": "in", "field": "runId", "values": []string{}},
		"empty and":         map[string]any{"op": "and", "conditions": []any{}},
		"empty not":         map[string]any{"op": "not"},
		"exists with value": map[string]any{"op": "exists", "field": "runId", "value": "x"},
	} {
		cases = append(cases, itemInvalidCase{name, "SearchItems", map[string]any{"filter": filter}})
	}
	runItemInvalidCases(t, c, scoped, defaults, cases)
	t.Run("valid topK and scope filter", func(t *testing.T) {
		for _, topK := range []int{0, 1, 50} {
			result, err := c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: scoped, Query: query, TopK: topK})
			limit := topK
			if limit == 0 {
				limit = 10
			}
			if err != nil || len(result.Items) == 0 || len(result.Items) > limit {
				t.Fatalf("valid topK %d: %+v %v", topK, result, err)
			}
			for _, hit := range result.Items {
				if hit.Scope != scoped.Scope {
					t.Fatalf("exact search escaped scope: %+v", hit)
				}
			}
		}
		// Filter is ANDed with scope. It must not turn an exact query into
		// a query for a different run, even when the text remains identical.
		result, err := c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: scoped, Query: query, Filter: &sdk.ScopeFilter{Op: "eq", Field: "runId", Value: sdk.String(scoped.Scope.RunID + "_other")}})
		if err != nil || len(result.Items) != 0 {
			t.Fatalf("conflicting scope filter: %+v %v", result, err)
		}
		result, err = c.SearchItems(ctx, &sdk.SearchItemsRequest{ItemScopedRequest: scoped, Query: query, Thresholds: &sdk.SearchThresholds{MinScore: 1, MinSimilarity: 1}})
		if err != nil {
			t.Fatal(err)
		}
		for _, hit := range result.Items {
			if hit.Score < 1 || hit.Similarity < 1 {
				t.Fatalf("threshold ignored: %+v", hit)
			}
		}
	})
	t.Run("managed files reject writes", func(t *testing.T) {
		before, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: scoped, ItemID: itemID, View: sdk.ItemViewFull})
		if err != nil {
			t.Fatal(err)
		}
		for _, action := range []string{"AddItem", "UpdateItem", "DeleteItem", "RedactItemVersion"} {
			body := map[string]any{"type": sdk.ItemTypeMemoryFile, "memoryStoreName": scoped.MemoryStoreName, "scope": scoped.Scope}
			switch action {
			case "AddItem":
				body["path"], body["content"] = "/must-not-create.md", "readonly"
			case "UpdateItem":
				body["itemId"], body["content"] = itemID, "must not overwrite"
			case "DeleteItem":
				body["itemId"] = itemID
			case "RedactItemVersion":
				body["versionId"] = before.LatestVersionID
			}
			err := c.Do(ctx, action, body, nil)
			var serviceErr *sdk.ServiceError
			if !errors.As(err, &serviceErr) || serviceErr.StatusCode != 409 || (serviceErr.Code != "READ_ONLY_STORE" && serviceErr.Code != "OTSOperationConflict") || serviceErr.RequestID == "" {
				t.Errorf("%s read-only rejection: %v", action, err)
			}
		}
		after, err := c.GetItem(ctx, &sdk.GetItemRequest{ItemScopedRequest: scoped, ItemID: itemID, View: sdk.ItemViewFull})
		if err != nil || after.LatestVersionID != before.LatestVersionID || after.Content == nil || before.Content == nil || *after.Content != *before.Content {
			t.Fatalf("read-only fixture changed: %+v %v", after, err)
		}
	})
}
