package agentstorage

import (
	"encoding/json"
	"reflect"
	"testing"
)

// SearchMemories carries the current session as contextScope while scope stays
// cross-session, so the service can softly prefer same-session memories.
func TestSearchMemoriesContextScopeSerialization(t *testing.T) {
	assertJSONEqual(t, &SearchMemoriesRequest{
		MemoryStoreName: "store",
		Query:           "coffee",
		Scope:           &Scope{AppID: "a", TenantID: "u", AgentID: "x", RunID: "*"},
		ContextScope:    &Scope{AppID: "a", TenantID: "u", AgentID: "x", RunID: "run-7"},
		TopK:            5,
	}, `{
		"memoryStoreName": "store",
		"query": "coffee",
		"scope": {"appId": "a", "tenantId": "u", "agentId": "x", "runId": "*"},
		"contextScope": {"appId": "a", "tenantId": "u", "agentId": "x", "runId": "run-7"},
		"topK": 5
	}`)

	// Omitted contextScope must not appear in the payload at all.
	assertJSONEqual(t, &SearchMemoriesRequest{
		MemoryStoreName: "store",
		Query:           "coffee",
		Scope:           &Scope{AppID: "a", TenantID: "u"},
	}, `{
		"memoryStoreName": "store",
		"query": "coffee",
		"scope": {"appId": "a", "tenantId": "u"}
	}`)
}

// Session-affinity ranking fields are only present on same-session results and
// only once the service runs in "on" mode.
func TestSearchHitRankingFieldsParsing(t *testing.T) {
	raw := `{
		"results": [
			{
				"unit": {"id": "u1", "scope": {"appId": "a"}, "text": "likes coffee"},
				"score": 0.5,
				"rankingScore": 0.55,
				"rankingSignals": {"sameSession": true, "sessionAffinityWeight": 0.1},
				"source": "vector"
			},
			{
				"unit": {"id": "u2", "scope": {"appId": "a"}, "text": "hates tea"},
				"score": 0.6,
				"rankingScore": 0.6,
				"rankingSignals": {"sameSession": false, "sessionAffinityWeight": 0.1},
				"source": "vector"
			}
		],
		"memoryStoreName": "store"
	}`
	var resp SearchMemoriesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("results length = %d, want 2", len(resp.Results))
	}
	first := resp.Results[0]
	if first.RankingScore == nil {
		t.Fatal("rankingScore missing on first result")
	}
	if *first.RankingScore != 0.55 {
		t.Errorf("rankingScore = %v, want 0.55", *first.RankingScore)
	}
	if first.RankingSignals == nil {
		t.Fatal("rankingSignals missing on first result")
	}
	if !first.RankingSignals.SameSession {
		t.Error("sameSession = false, want true")
	}
	if first.RankingSignals.SessionAffinityWeight != 0.1 {
		t.Errorf("sessionAffinityWeight = %v, want 0.1", first.RankingSignals.SessionAffinityWeight)
	}
	if second := resp.Results[1]; second.RankingSignals == nil || second.RankingSignals.SameSession {
		t.Errorf("second result should be marked cross-session: %+v", second.RankingSignals)
	}
}

// Off/shadow mode responses carry no ranking fields; the pointers stay nil so
// callers can tell "not ranked" from "ranked with score 0".
func TestSearchHitWithoutRankingFields(t *testing.T) {
	raw := `{"results": [{"unit": {"id": "u1"}, "score": 0.9, "source": "vector"}]}`
	var resp SearchMemoriesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	hit := resp.Results[0]
	if hit.RankingScore != nil || hit.RankingSignals != nil {
		t.Errorf("ranking fields should be nil, got %+v / %+v", hit.RankingScore, hit.RankingSignals)
	}
}

// Read paths expose the semantic description (title/keywords) and the time
// triple (effective_at / updated_at / time_source+time_precision).
func TestMemoryUnitV2FieldsParsing(t *testing.T) {
	raw := `{
		"memory": {
			"id": "u1",
			"scope": {"appId": "a", "tenantId": "u", "agentId": "x", "runId": "r"},
			"unit_type": "atomic_fact",
			"title": "婚礼定在 8 月 1 日",
			"text": "我们 8 月 1 日办婚礼",
			"keywords": ["婚礼", "8月1日"],
			"created_at": "2026-07-16T03:04:05Z",
			"effective_at": "2026-08-01T00:00:00Z",
			"updated_at": "2026-07-16T03:04:05Z",
			"time_source": "explicit_event",
			"time_precision": "day",
			"merged_from_memory_ids": ["m1", "m2"],
			"salience": 0.5,
			"version": 2
		},
		"memoryStoreName": "store"
	}`
	var resp GetMemoryResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	unit := resp.Memory
	if unit.Title != "婚礼定在 8 月 1 日" {
		t.Errorf("title = %q", unit.Title)
	}
	if !reflect.DeepEqual(unit.Keywords, []string{"婚礼", "8月1日"}) {
		t.Errorf("keywords = %v", unit.Keywords)
	}
	if unit.EffectiveAt != "2026-08-01T00:00:00Z" {
		t.Errorf("effective_at = %q", unit.EffectiveAt)
	}
	if unit.UpdatedAt != "2026-07-16T03:04:05Z" {
		t.Errorf("updated_at = %q", unit.UpdatedAt)
	}
	if unit.TimeSource != TimeSourceExplicitEvent {
		t.Errorf("time_source = %q, want %q", unit.TimeSource, TimeSourceExplicitEvent)
	}
	if unit.TimePrecision != TimePrecisionDay {
		t.Errorf("time_precision = %q, want %q", unit.TimePrecision, TimePrecisionDay)
	}
	if !reflect.DeepEqual(unit.MergedFromMemoryIDs, []string{"m1", "m2"}) {
		t.Errorf("merged_from_memory_ids = %v", unit.MergedFromMemoryIDs)
	}
}

// Legacy rows never touched by a V2 write keep the pre-V2 payload: the new
// fields must simply stay empty instead of failing to parse.
func TestMemoryUnitLegacyRowParsing(t *testing.T) {
	raw := `{"memory": {"id": "u1", "text": "old fact", "created_at": "2026-01-01T00:00:00Z"}}`
	var resp GetMemoryResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	unit := resp.Memory
	if unit.Title != "" || unit.EffectiveAt != "" || unit.TimeSource != "" || len(unit.Keywords) != 0 {
		t.Errorf("legacy row should carry no V2 fields: %+v", unit)
	}
}

// UpdateMemory can hand-correct title/keywords and pin the semantic event time.
func TestUpdateMemoryV2FieldsSerialization(t *testing.T) {
	assertJSONEqual(t, &UpdateMemoryRequest{
		MemoryStoreName: "store",
		MemoryID:        "u1",
		Scope:           Scope{AppID: "a", TenantID: "u", AgentID: "x", RunID: "r"},
		Title:           String("婚礼定在 8 月 1 日"),
		Keywords:        Strings("婚礼", "8月1日"),
		EffectiveAt:     String("2026-08-01"),
		TimePrecision:   String(TimePrecisionDay),
	}, `{
		"memoryStoreName": "store",
		"memoryId": "u1",
		"scope": {"appId": "a", "tenantId": "u", "agentId": "x", "runId": "r"},
		"title": "婚礼定在 8 月 1 日",
		"keywords": ["婚礼", "8月1日"],
		"effectiveAt": "2026-08-01",
		"timePrecision": "day"
	}`)

	// An empty (non-nil) keyword list clears the keywords; nil omits the field.
	assertJSONEqual(t, &UpdateMemoryRequest{
		MemoryStoreName: "store",
		MemoryID:        "u1",
		Scope:           Scope{AppID: "a", TenantID: "u", AgentID: "x", RunID: "r"},
		Keywords:        Strings(),
	}, `{
		"memoryStoreName": "store",
		"memoryId": "u1",
		"scope": {"appId": "a", "tenantId": "u", "agentId": "x", "runId": "r"},
		"keywords": []
	}`)

	// Text-only updates keep the pre-V2 payload shape.
	assertJSONEqual(t, &UpdateMemoryRequest{
		MemoryStoreName: "store",
		MemoryID:        "u1",
		Scope:           Scope{AppID: "a", TenantID: "u", AgentID: "x", RunID: "r"},
		Text:            "new text",
	}, `{
		"memoryStoreName": "store",
		"memoryId": "u1",
		"scope": {"appId": "a", "tenantId": "u", "agentId": "x", "runId": "r"},
		"text": "new text"
	}`)
}

// The enum values must match the service exactly; a typo here silently breaks
// caller comparisons.
func TestMemoryTimeEnumValues(t *testing.T) {
	got := []string{
		TimeSourceExplicitEvent, TimeSourceSourceMessage, TimeSourceLegacyCreatedAt, TimeSourceManual,
		TimePrecisionSecond, TimePrecisionDay, TimePrecisionMonth, TimePrecisionYear, TimePrecisionUnknown,
	}
	want := []string{
		"explicit_event", "source_message", "legacy_created_at", "manual",
		"second", "day", "month", "year", "unknown",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("enum values = %v, want %v", got, want)
	}
}
