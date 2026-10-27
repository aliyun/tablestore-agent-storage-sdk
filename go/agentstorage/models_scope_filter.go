package agentstorage

import "encoding/json"

// ScopeFilter is the scope-directory expression accepted by memory and item
// queries, distinct from the knowledge-base MetadataFilter. Field names are
// directory dimensions (including appId, tenantId, agentId, runId, scopeType,
// scopeKey and text). Operands are strings. Current stores support and/or/not
// with eq/in/exists; text supports eq. Historical stores may also support
// prefix/wildcard/range. The server applies the Store's creation-time contract
// and validates maximum depth (8) and leaf count (64).
type ScopeFilter struct {
	Op         string         `json:"op"`
	Conditions []*ScopeFilter `json:"conditions,omitempty"`
	Condition  *ScopeFilter   `json:"condition,omitempty"`
	Field      string         `json:"field,omitempty"`
	Value      *string        `json:"value,omitempty"`
	Values     []string       `json:"values,omitempty"`
	GT         *string        `json:"gt,omitempty"`
	GTE        *string        `json:"gte,omitempty"`
	LT         *string        `json:"lt,omitempty"`
	LTE        *string        `json:"lte,omitempty"`
}

// SearchThresholds applies both bounds, when nonzero, before ranking boosts.
type SearchThresholds struct {
	MinSimilarity float64 `json:"minSimilarity,omitempty"`
	MinScore      float64 `json:"minScore,omitempty"`
}

// InteractionRef links feedback to an earlier Wiki-reading turn. It is not an
// idempotency key; request identity remains owned by the Tablestore gateway.
type InteractionRef struct {
	SessionID     string `json:"sessionId"`
	TurnID        string `json:"turnId"`
	ReplyToTurnID string `json:"replyToTurnId,omitempty"`
}

const (
	StoreInputMessages    = "messages"
	StoreInputFile        = "file"
	StoreOutputStructured = "structured"
	StoreOutputFile       = "file"
)

const (
	ScopeTypeConversation = "conversation"
	ScopeTypeWiki         = "wiki"
)

// MemoryStoreScope is a heterogeneous Scope Directory entry. Scope preserves
// the familiar dimensions; Dimensions retains all flattened dimension values.
// Use ScopeEntries in ListMemoryStoreScopesResponse to distinguish Wiki entries
// from conversations instead of treating a missing runId as a default run.
type MemoryStoreScope struct {
	Scope
	ScopeType  string            `json:"scopeType"`
	ScopeKey   string            `json:"scopeKey"`
	Dimensions map[string]string `json:"-"`
}

func (s *MemoryStoreScope) UnmarshalJSON(data []byte) error {
	var fields map[string]string
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	var scope Scope
	if err := json.Unmarshal(data, &scope); err != nil {
		return err
	}
	*s = MemoryStoreScope{Scope: scope, ScopeType: fields["scopeType"], ScopeKey: fields["scopeKey"]}
	delete(fields, "scopeType")
	delete(fields, "scopeKey")
	s.Dimensions = fields
	return nil
}

func (s MemoryStoreScope) MarshalJSON() ([]byte, error) {
	fields := make(map[string]string, len(s.Dimensions)+6)
	for key, value := range s.Dimensions {
		fields[key] = value
	}
	for key, value := range map[string]string{"appId": s.AppID, "tenantId": s.TenantID, "agentId": s.AgentID, "runId": s.RunID} {
		if value != "" {
			fields[key] = value
		}
	}
	fields["scopeType"], fields["scopeKey"] = s.ScopeType, s.ScopeKey
	return json.Marshal(fields)
}

func (r *ListMemoryStoreScopesResponse) UnmarshalJSON(data []byte) error {
	type plain ListMemoryStoreScopesResponse
	var decoded struct {
		plain
		Entries []MemoryStoreScope `json:"scopes"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = ListMemoryStoreScopesResponse(decoded.plain)
	r.ScopeEntries = decoded.Entries
	if decoded.Entries != nil {
		r.Scopes = make([]Scope, len(decoded.Entries))
		for i, entry := range decoded.Entries {
			r.Scopes[i] = entry.Scope
		}
	}
	return nil
}

func (r ListMemoryStoreScopesResponse) MarshalJSON() ([]byte, error) {
	type plain ListMemoryStoreScopesResponse
	var entries any = r.Scopes
	if r.ScopeEntries != nil {
		entries = r.ScopeEntries
	}
	return json.Marshal(struct {
		plain
		Entries any `json:"scopes"`
	}{plain: plain(r), Entries: entries})
}
