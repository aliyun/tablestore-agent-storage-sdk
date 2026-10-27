package e2e

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/tablestore-agent-storage-sdk/go/agentstorage"
)

// validTimeSources / validTimePrecisions mirror the service enums; an unknown
// value means the SDK constants drifted from the service.
var (
	validTimeSources = map[string]bool{
		agentstorage.TimeSourceExplicitEvent:   true,
		agentstorage.TimeSourceSourceMessage:   true,
		agentstorage.TimeSourceLegacyCreatedAt: true,
		agentstorage.TimeSourceManual:          true,
	}
	validTimePrecisions = map[string]bool{
		agentstorage.TimePrecisionSecond:  true,
		agentstorage.TimePrecisionDay:     true,
		agentstorage.TimePrecisionMonth:   true,
		agentstorage.TimePrecisionYear:    true,
		agentstorage.TimePrecisionUnknown: true,
	}
)

// v2Store always creates an isolated store and verifies cleanup.
func v2Store(t *testing.T, ctx context.Context, client *agentstorage.Client) string {
	t.Helper()
	store := currentStore(t, client, agentstorage.StoreInputMessages, []string{agentstorage.StoreOutputStructured})
	waitForMemoryIDIndexReady(t, ctx, client, store, fullScope("v2probe"))
	return store
}

// TestMemoryV2SemanticAndTimeE2E covers the semantic description (title /
// keywords) and the effective-time triple, including a manual UpdateMemory
// override. Missing semantic/time capability fails acceptance explicitly.
func TestMemoryV2SemanticAndTimeE2E(t *testing.T) {
	client := e2eClient(t)
	ctx := context.Background()
	store := v2Store(t, ctx, client)
	scope := fullScope("v2sem")

	if _, err := retry(t, 5, 2*time.Second, func() (*agentstorage.AddMemoriesResponse, error) {
		return client.AddMemories(ctx, &agentstorage.AddMemoriesRequest{
			MemoryStoreName: store,
			Scope:           &scope,
			Messages: []agentstorage.MessagePayload{
				{Role: "user", Content: "我们决定 2026 年 8 月 1 日在杭州办婚礼。", MessageID: "v2_" + suffix(6)},
			},
			Sync: true,
		})
	}, func(err error) bool {
		status, _ := serviceStatus(err)
		return status == 500
	}); err != nil {
		t.Fatalf("AddMemories: %v", err)
	}
	listed := waitForListMatch(t, ctx, client, store, scope, "婚礼", "wedding memory not list-visible")

	var target agentstorage.MemoryUnit
	for _, unit := range listed.Memories {
		if strings.Contains(unit.Text, "婚礼") {
			target = unit
			break
		}
	}
	if target.ID == "" {
		t.Fatalf("no wedding memory in %d units", len(listed.Memories))
	}
	t.Logf("unit: id=%s title=%q keywords=%v effective_at=%q updated_at=%q time_source=%q time_precision=%q created_at=%q",
		target.ID, target.Title, target.Keywords, target.EffectiveAt, target.UpdatedAt,
		target.TimeSource, target.TimePrecision, target.CreatedAt)

	v2Active := target.TimeSource != "" || target.Title != "" || len(target.Keywords) > 0
	if !v2Active {
		t.Fatal("service returns no V2 fields; semantic/time acceptance blocked")
	} else {
		if target.Title == "" {
			t.Error("title is empty on a V2 unit")
		}
		if len([]rune(target.Title)) > agentstorage.MaxMemoryTitleRunes {
			t.Errorf("title longer than %d runes: %q", agentstorage.MaxMemoryTitleRunes, target.Title)
		}
		if len(target.Keywords) > agentstorage.MaxMemoryKeywords {
			t.Errorf("keywords = %d, want <= %d", len(target.Keywords), agentstorage.MaxMemoryKeywords)
		}
		if !validTimeSources[target.TimeSource] {
			t.Errorf("unknown time_source %q", target.TimeSource)
		}
		if target.TimePrecision != "" && !validTimePrecisions[target.TimePrecision] {
			t.Errorf("unknown time_precision %q", target.TimePrecision)
		}
		if target.EffectiveAt != "" {
			if _, err := time.Parse(time.RFC3339, target.EffectiveAt); err != nil {
				t.Errorf("effective_at %q is not RFC3339: %v", target.EffectiveAt, err)
			}
		}
	}

	// --- Manual override: pin title, keywords and the semantic event time ---
	updated, err := client.UpdateMemory(ctx, &agentstorage.UpdateMemoryRequest{
		MemoryStoreName: store,
		MemoryID:        target.ID,
		Scope:           scope,
		Title:           agentstorage.String("婚礼定在 2026-08-01"),
		Keywords:        agentstorage.Strings("婚礼", "杭州", "2026-08-01"),
		EffectiveAt:     agentstorage.String("2026-08-01"),
		TimePrecision:   agentstorage.String(agentstorage.TimePrecisionDay),
	})
	if err != nil {
		if status, code := serviceStatus(err); status == 409 {
			t.Fatalf("UpdateMemory with V2 fields rejected with 409 %s: %v", code, err)
			return
		}
		t.Fatalf("UpdateMemory(v2 fields): %v", err)
	}
	t.Logf("after manual update: title=%q keywords=%v effective_at=%q time_source=%q time_precision=%q updated_at=%q",
		updated.Memory.Title, updated.Memory.Keywords, updated.Memory.EffectiveAt,
		updated.Memory.TimeSource, updated.Memory.TimePrecision, updated.Memory.UpdatedAt)
	if updated.Memory.Title != "婚礼定在 2026-08-01" {
		t.Errorf("title = %q after manual update", updated.Memory.Title)
	}
	if len(updated.Memory.Keywords) != 3 {
		t.Errorf("keywords = %v after manual update", updated.Memory.Keywords)
	}
	if updated.Memory.TimeSource != agentstorage.TimeSourceManual {
		t.Errorf("time_source = %q, want %q", updated.Memory.TimeSource, agentstorage.TimeSourceManual)
	}
	if updated.Memory.TimePrecision != agentstorage.TimePrecisionDay {
		t.Errorf("time_precision = %q, want %q", updated.Memory.TimePrecision, agentstorage.TimePrecisionDay)
	}
	pinned := updated.Memory.EffectiveAt
	if !strings.HasPrefix(pinned, "2026-08-01") {
		t.Errorf("effective_at = %q, want 2026-08-01", pinned)
	}

	// --- A text-only revision moves updated_at but keeps effective_at ---
	revised, err := client.UpdateMemory(ctx, &agentstorage.UpdateMemoryRequest{
		MemoryStoreName: store,
		MemoryID:        target.ID,
		Scope:           scope,
		Text:            "我们决定 2026 年 8 月 1 日在杭州办婚礼（已确认场地）。",
	})
	if err != nil {
		t.Fatalf("UpdateMemory(text): %v", err)
	}
	if revised.Memory.EffectiveAt != pinned {
		t.Errorf("effective_at changed by a text revision: %q -> %q", pinned, revised.Memory.EffectiveAt)
	}
	if revised.Memory.UpdatedAt == "" {
		t.Error("updated_at empty after revision")
	}
	t.Logf("after text revision: effective_at=%q updated_at=%q", revised.Memory.EffectiveAt, revised.Memory.UpdatedAt)
}

// TestMemoryV2ContextScopeE2E covers cross-session search with contextScope:
// the request must be accepted in every affinity mode, and when the service
// runs in "on" mode the same-session hits carry the ranking fields.
func TestMemoryV2ContextScopeE2E(t *testing.T) {
	client := e2eClient(t)
	ctx := context.Background()
	store := v2Store(t, ctx, client)

	base := fullScope("v2ctx")
	current := base
	current.RunID = base.RunID + "_current"
	other := base
	other.RunID = base.RunID + "_other"

	add := func(scope agentstorage.Scope, text string) {
		t.Helper()
		if _, err := retry(t, 5, 2*time.Second, func() (*agentstorage.AddMemoriesResponse, error) {
			return client.AddMemories(ctx, &agentstorage.AddMemoriesRequest{
				MemoryStoreName: store,
				Scope:           &scope,
				Text:            text,
				Sync:            true,
			})
		}, func(err error) bool {
			status, _ := serviceStatus(err)
			return status == 500
		}); err != nil {
			t.Fatalf("AddMemories(%s): %v", scope.RunID, err)
		}
	}
	add(current, "Ian sometimes drinks coffee after lunch.")
	add(other, "Ian's favourite coffee is a double espresso from Blue Bottle.")
	waitForListMatch(t, ctx, client, store, current, "coffee", "current-session memory not list-visible")
	waitForListMatch(t, ctx, client, store, other, "espresso", "other-session memory not list-visible")

	// The cross-session scope must keep a wildcard: the service only runs the
	// session lane (and therefore the affinity ranking) for a wildcard scope.
	crossScope := base
	crossScope.RunID = "*"
	contextScope := current
	t.Logf("scope=%+v contextScope=%+v", crossScope, contextScope)

	const query = "What coffee does Ian like?"
	// The search index lags the write, so poll until both sessions are visible.
	plain := poll(t, 120*time.Second, 3*time.Second, "cross-session search returned no results",
		func() (*agentstorage.SearchMemoriesResponse, error) {
			return client.SearchMemories(ctx, &agentstorage.SearchMemoriesRequest{
				MemoryStoreName: store,
				Query:           query,
				Scope:           &crossScope,
				TopK:            10,
			})
		},
		func(resp *agentstorage.SearchMemoriesResponse) bool {
			sessions := map[string]bool{}
			for _, hit := range resp.Results {
				sessions[hit.Unit.Scope.RunID] = true
			}
			return sessions[current.RunID] && sessions[other.RunID]
		})
	for _, hit := range plain.Results {
		if hit.RankingScore != nil || hit.RankingSignals != nil {
			t.Errorf("ranking fields present without contextScope: %+v", hit)
		}
	}

	scoped, err := client.SearchMemories(ctx, &agentstorage.SearchMemoriesRequest{
		MemoryStoreName: store,
		Query:           query,
		Scope:           &crossScope,
		ContextScope:    &contextScope,
		TopK:            10,
	})
	if err != nil {
		t.Fatalf("SearchMemories(contextScope): %v", err)
	}
	if len(scoped.Results) == 0 {
		t.Fatal("no results for the cross-session query")
	}

	ranked := false
	for _, hit := range scoped.Results {
		if hit.RankingScore != nil {
			ranked = true
		}
		t.Logf("hit runId=%s score=%.4f rankingScore=%v signals=%+v text=%q",
			hit.Unit.Scope.RunID, hit.Score, formatRanking(hit.RankingScore), hit.RankingSignals, hit.Unit.Text)
	}
	if !ranked {
		t.Fatal("service returns no rankingScore; affinity-on acceptance blocked")
		return
	}

	var previous float64
	for i, hit := range scoped.Results {
		if hit.RankingScore == nil || hit.RankingSignals == nil {
			t.Errorf("hit %d has partial ranking fields: %+v", i, hit)
			continue
		}
		sameSession := hit.Unit.Scope == contextScope
		if hit.RankingSignals.SameSession != sameSession {
			t.Errorf("hit %d sameSession = %v, want %v (scope %+v)", i, hit.RankingSignals.SameSession, sameSession, hit.Unit.Scope)
		}
		want := hit.Score
		if sameSession {
			want = hit.Score + (1-hit.Score)*hit.RankingSignals.SessionAffinityWeight
		}
		if math.Abs(*hit.RankingScore-want) > 1e-6 {
			t.Errorf("hit %d rankingScore = %v, want %v (score %v, weight %v)",
				i, *hit.RankingScore, want, hit.Score, hit.RankingSignals.SessionAffinityWeight)
		}
		if i > 0 && *hit.RankingScore > previous+1e-9 {
			t.Errorf("results not ordered by rankingScore: hit %d has %v after %v", i, *hit.RankingScore, previous)
		}
		previous = *hit.RankingScore
	}

	// A contextScope carrying a wildcard is rejected by the service.
	badScope := contextScope
	badScope.RunID = "*"
	if _, err := client.SearchMemories(ctx, &agentstorage.SearchMemoriesRequest{
		MemoryStoreName: store,
		Query:           "coffee",
		Scope:           &crossScope,
		ContextScope:    &badScope,
		TopK:            5,
	}); err == nil {
		t.Error("wildcard contextScope was accepted, want a validation error")
	} else {
		status, code := serviceStatus(err)
		t.Logf("wildcard contextScope rejected: status=%d code=%s err=%v", status, code, err)
		if status != 400 {
			t.Errorf("wildcard contextScope status = %d, want 400", status)
		}
	}
}

func formatRanking(value *float64) string {
	if value == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%.4f", *value)
}
