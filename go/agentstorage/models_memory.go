package agentstorage

// Scope identifies the four-level ownership of memory data. Depending on the
// API, some levels are required, and list-style APIs accept the "*" wildcard.
type Scope struct {
	AppID    string `json:"appId,omitempty"`
	TenantID string `json:"tenantId,omitempty"`
	AgentID  string `json:"agentId,omitempty"`
	RunID    string `json:"runId,omitempty"`
}

// MessagePayload is one conversation message of AddMemoriesRequest.
type MessagePayload struct {
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	MessageID string            `json:"messageId,omitempty"`
	Timestamp string            `json:"timestamp,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Timestamp is a request time bound. Pass an RFC3339 string where the API
// documents RFC3339 (ListMemoryStoreMessages / ListMemoryStoreRequests) or a
// Unix-millisecond integer where the API documents milliseconds (task and
// dream APIs).
type Timestamp = any

// StorageModeOTS is the default storage mode of memory stores; see also
// StorageModeFileOTS in models_file.go.
const StorageModeOTS = "ots"

// ---------------------------------------------------------------------------
// Memory store management
// ---------------------------------------------------------------------------

// CreateMemoryStoreRequest is the request for CreateMemoryStore.
type CreateMemoryStoreRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	Description     string `json:"description,omitempty"`
	// ExtractInstructions injects custom memory-extraction guidance for this store.
	ExtractInstructions string `json:"extractInstructions,omitempty"`
	// StorageMode selects the immutable storage mode ("ots" default,
	// "file+ots" for a read-only projected file tree, or "filemem" for a
	// writable file-memory store).
	StorageMode string `json:"storageMode,omitempty"`
}

// GetMemoryStoreRequest is the request for GetMemoryStore.
type GetMemoryStoreRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
}

// ListMemoryStoresRequest is the request for ListMemoryStores.
type ListMemoryStoresRequest struct {
	Limit     int    `json:"limit,omitempty"`
	NextToken string `json:"nextToken,omitempty"`
}

// UpdateMemoryStoreRequest is the request for UpdateMemoryStore. PATCH
// semantics: nil fields stay unchanged, pointers to the empty string clear
// the stored value.
type UpdateMemoryStoreRequest struct {
	MemoryStoreName     string  `json:"memoryStoreName"`
	Description         *string `json:"description,omitempty"`
	ExtractInstructions *string `json:"extractInstructions,omitempty"`
}

// DeleteMemoryStoreRequest is the request for DeleteMemoryStore.
type DeleteMemoryStoreRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
}

// MemoryStoreInfo describes a memory store.
type MemoryStoreInfo struct {
	OTSInstance         string `json:"otsInstance,omitempty"`
	MemoryStoreName     string `json:"memoryStoreName"`
	Description         string `json:"description,omitempty"`
	ExtractInstructions string `json:"extractInstructions,omitempty"`
	StorageMode         string `json:"storageMode,omitempty"`
	CreatedAt           string `json:"createdAt,omitempty"`
	UpdatedAt           string `json:"updatedAt,omitempty"`
	LastSeenAt          string `json:"lastSeenAt,omitempty"`
}

// MemoryStoreListItem is one entry of ListMemoryStores.
type MemoryStoreListItem struct {
	MemoryStoreName string `json:"memoryStoreName"`
}

type (
	CreateMemoryStoreResponse = MemoryStoreInfo
	GetMemoryStoreResponse    = MemoryStoreInfo
	UpdateMemoryStoreResponse = MemoryStoreInfo
)

// ListMemoryStoresResponse is the response of ListMemoryStores.
type ListMemoryStoresResponse struct {
	Stores    []MemoryStoreListItem `json:"stores"`
	NextToken string                `json:"nextToken,omitempty"`
}

// DeleteMemoryStoreResponse is the response of DeleteMemoryStore.
type DeleteMemoryStoreResponse struct {
	Deleted         bool   `json:"deleted"`
	MemoryStoreName string `json:"memoryStoreName"`
}

// ---------------------------------------------------------------------------
// Long-term memories
// ---------------------------------------------------------------------------

// AddMemoriesRequest is the request for AddMemories. Provide either Messages
// or Text. Scope.AppID is required for writes; wildcards are not allowed.
type AddMemoriesRequest struct {
	MemoryStoreName string            `json:"memoryStoreName"`
	Scope           *Scope            `json:"scope,omitempty"`
	Messages        []MessagePayload  `json:"messages,omitempty"`
	Text            string            `json:"text,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Sync            bool              `json:"sync,omitempty"`
	// ReferenceTime is an optional RFC3339 "as-of" anchor: messages without
	// their own timestamp are anchored to it instead of the server clock.
	ReferenceTime string `json:"referenceTime,omitempty"`
}

// SearchMemoriesRequest is the request for SearchMemories. Scope.AppID and
// Scope.TenantID are required; AgentID/RunID accept the "*" wildcard.
type SearchMemoriesRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	Query           string `json:"query"`
	Scope           *Scope `json:"scope,omitempty"`
	// ContextScope names the current session so its memories get a bounded
	// soft boost while Scope stays cross-session (typically RunID: "*"). All
	// four fields are required, wildcards are rejected, and it must be
	// contained in Scope. When the service runs session affinity in "on" mode
	// the results carry SearchHit.RankingScore / RankingSignals.
	ContextScope    *Scope            `json:"contextScope,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	TopK            int               `json:"topK,omitempty"`
	EnableRerank    *bool             `json:"enableRerank,omitempty"`
	IncludeEvidence bool              `json:"includeEvidence,omitempty"`
	MinSimilarity   float64           `json:"minSimilarity,omitempty"`
}

// ListMemoriesRequest is the request for ListMemories. The scope accepts
// hierarchical "*" wildcards.
type ListMemoriesRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	Scope           *Scope `json:"scope,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	NextToken       string `json:"nextToken,omitempty"`
}

// GetMemoryRequest is the request for GetMemory. Requires the full 4-field
// scope without wildcards.
type GetMemoryRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	MemoryID        string `json:"memoryId"`
	Scope           Scope  `json:"scope"`
}

// UpdateMemoryRequest is the request for UpdateMemory. At least one of Text,
// Metadata, Title, Keywords or the effective-time pair must be provided.
type UpdateMemoryRequest struct {
	MemoryStoreName string            `json:"memoryStoreName"`
	MemoryID        string            `json:"memoryId"`
	Scope           Scope             `json:"scope"`
	Text            string            `json:"text,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	// Title hand-corrects the server-extracted title (at most
	// MaxMemoryTitleRunes characters). nil leaves it unchanged.
	Title *string `json:"title,omitempty"`
	// Keywords replaces the server-extracted keywords (at most
	// MaxMemoryKeywords entries of MaxMemoryKeywordRunes characters). nil
	// leaves them unchanged; a non-nil empty list clears them. Use Strings().
	Keywords *[]string `json:"keywords,omitempty"`
	// EffectiveAt pins the semantic event time and must be sent together with
	// TimePrecision, whose value decides the accepted layout: "second" takes
	// RFC3339, "day" takes 2006-01-02, "month" takes 2006-01, "year" takes
	// 2006. A manual override sets TimeSource to TimeSourceManual.
	EffectiveAt   *string `json:"effectiveAt,omitempty"`
	TimePrecision *string `json:"timePrecision,omitempty"`
}

// DeleteMemoryRequest is the request for DeleteMemory.
type DeleteMemoryRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	MemoryID        string `json:"memoryId"`
	Scope           Scope  `json:"scope"`
}

// SourceTurnRef links a memory unit back to a source message.
type SourceTurnRef struct {
	MessageID   string `json:"messageId"`
	TimestampMs int64  `json:"timestampMs"`
}

// Values of MemoryUnit.TimeSource, telling where EffectiveAt came from.
const (
	// TimeSourceExplicitEvent means the conversation stated the date.
	TimeSourceExplicitEvent = "explicit_event"
	// TimeSourceSourceMessage means no date was stated, so the source message
	// timestamp is used as the fallback — the service never invents one.
	TimeSourceSourceMessage = "source_message"
	// TimeSourceLegacyCreatedAt means the value comes from a pre-V2 row's
	// created_at.
	TimeSourceLegacyCreatedAt = "legacy_created_at"
	// TimeSourceManual means a client pinned it via UpdateMemory.
	TimeSourceManual = "manual"
)

// Values of MemoryUnit.TimePrecision and UpdateMemoryRequest.TimePrecision.
// The precision states how much of EffectiveAt is meaningful; lower precisions
// are stored as the start of their UTC range.
const (
	TimePrecisionSecond  = "second"
	TimePrecisionDay     = "day"
	TimePrecisionMonth   = "month"
	TimePrecisionYear    = "year"
	TimePrecisionUnknown = "unknown"
)

// Limits the service enforces on manual title/keywords overrides.
const (
	MaxMemoryTitleRunes   = 64
	MaxMemoryKeywords     = 8
	MaxMemoryKeywordRunes = 32
)

// MemoryUnit is one long-term memory unit.
type MemoryUnit struct {
	ID              string `json:"id"`
	ConversationKey string `json:"conversation_key,omitempty"`
	Scope           Scope  `json:"scope"`
	MemcellID       string `json:"memcell_id,omitempty"`
	UnitType        string `json:"unit_type,omitempty"`
	// Title is a short server-extracted summary of Text, useful to let an
	// agent grasp the memory without reading it all. Empty on rows written
	// before the semantic-description rollout.
	Title      string `json:"title,omitempty"`
	Text       string `json:"text,omitempty"`
	SearchText string `json:"search_text,omitempty"`
	// Keywords are the server-extracted retrieval keywords; they take part in
	// full-text, vector and rerank matching.
	Keywords       []string          `json:"keywords,omitempty"`
	Entities       []string          `json:"entities,omitempty"`
	TimeAnchor     map[string]string `json:"time_anchor,omitempty"`
	SourceTurnIDs  []string          `json:"source_turn_ids,omitempty"`
	SourceTurnRefs []SourceTurnRef   `json:"source_turn_refs,omitempty"`
	Speaker        string            `json:"speaker,omitempty"`
	Topic          string            `json:"topic,omitempty"`
	TypeLabel      string            `json:"type_label,omitempty"`
	DateBucket     string            `json:"date_bucket,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	MetadataJSON   string            `json:"metadata_json,omitempty"`
	MetadataFlat   string            `json:"metadata_flat,omitempty"`
	Deleted        bool              `json:"deleted,omitempty"`
	CreatedAt      string            `json:"created_at,omitempty"`
	// EffectiveAt is when the remembered fact happens or holds — the time to
	// compare when deciding which of two memories is newer. It is not the
	// ingestion time. RFC3339; empty when the service has no defensible value.
	EffectiveAt string `json:"effective_at,omitempty"`
	// UpdatedAt is when the memory itself was last revised, which is what a
	// revision (UpdateMemory, dream consolidation) moves — EffectiveAt stays.
	UpdatedAt string `json:"updated_at,omitempty"`
	// TimeSource and TimePrecision qualify EffectiveAt; see the TimeSource*
	// and TimePrecision* constants.
	TimeSource            string   `json:"time_source,omitempty"`
	TimePrecision         string   `json:"time_precision,omitempty"`
	Salience              float64  `json:"salience,omitempty"`
	Version               int      `json:"version,omitempty"`
	SupersededBy          string   `json:"superseded_by,omitempty"`
	SupersedesID          string   `json:"supersedes_id,omitempty"`
	MergedFromMemoryIDs   []string `json:"merged_from_memory_ids,omitempty"`
	DreamID               string   `json:"dream_id,omitempty"`
	DreamActionID         string   `json:"dream_action_id,omitempty"`
	DreamReason           string   `json:"dream_reason,omitempty"`
	DreamSourceMessageIDs []string `json:"dream_source_message_ids,omitempty"`
	DreamSourceMemoryIDs  []string `json:"dream_source_memory_ids,omitempty"`
}

// RankingSignals explains why a SearchHit was reordered.
type RankingSignals struct {
	// SameSession is true when the unit's scope exactly matches the
	// SearchMemoriesRequest.ContextScope of the query.
	SameSession bool `json:"sameSession"`
	// SessionAffinityWeight is the server-side weight used for the boost.
	SessionAffinityWeight float64 `json:"sessionAffinityWeight"`
}

// SearchHit is one SearchMemories result.
type SearchHit struct {
	Unit MemoryUnit `json:"unit"`
	// Score is the canonical relevance in [0,1]: the rerank relevance score
	// when rerank is enabled, otherwise the normalized cosine similarity.
	Score float64 `json:"score"`
	// Similarity is the raw normalized cosine similarity, informational.
	Similarity float64 `json:"similarity,omitempty"`
	// RankingScore is the score the results are ordered by when the service
	// applies session affinity: Score + (1-Score)*weight for hits that match
	// ContextScope exactly, Score otherwise. nil means no affinity ranking was
	// applied (no ContextScope sent, or the service is not in "on" mode).
	RankingScore *float64 `json:"rankingScore,omitempty"`
	// RankingSignals is present alongside RankingScore.
	RankingSignals *RankingSignals `json:"rankingSignals,omitempty"`
	Source         string          `json:"source,omitempty"`
}

// AddMemoriesResponse is the response of AddMemories.
type AddMemoriesResponse struct {
	RequestID        string `json:"requestId,omitempty"`
	Status           string `json:"status"`
	AcceptedMessages int    `json:"acceptedMessages"`
	Scope            *Scope `json:"scope,omitempty"`
	MemoryStoreName  string `json:"memoryStoreName"`
	// MemcellsCreated/UnitsCreated are only returned for sync writes.
	MemcellsCreated int `json:"memcellsCreated,omitempty"`
	UnitsCreated    int `json:"unitsCreated,omitempty"`
}

// SearchMemoriesResponse is the response of SearchMemories.
type SearchMemoriesResponse struct {
	Results []SearchHit `json:"results"`
	// Evidence carries short-term source evidence when IncludeEvidence is set.
	Evidence        []SearchHit `json:"evidence,omitempty"`
	Scope           *Scope      `json:"scope,omitempty"`
	MemoryStoreName string      `json:"memoryStoreName"`
}

// ListMemoriesResponse is the response of ListMemories.
type ListMemoriesResponse struct {
	Memories        []MemoryUnit `json:"memories"`
	Scope           *Scope       `json:"scope,omitempty"`
	MemoryStoreName string       `json:"memoryStoreName"`
	NextToken       string       `json:"nextToken,omitempty"`
}

// GetMemoryResponse is the response of GetMemory.
type GetMemoryResponse struct {
	Memory          MemoryUnit `json:"memory"`
	MemoryStoreName string     `json:"memoryStoreName"`
}

// UpdateMemoryResponse is the response of UpdateMemory.
type UpdateMemoryResponse struct {
	Memory          MemoryUnit `json:"memory"`
	Updated         bool       `json:"updated"`
	MemoryStoreName string     `json:"memoryStoreName"`
}

// DeleteMemoryResponse is the response of DeleteMemory.
type DeleteMemoryResponse struct {
	Deleted         bool   `json:"deleted"`
	MemoryID        string `json:"memoryId"`
	MemoryStoreName string `json:"memoryStoreName"`
}

// ---------------------------------------------------------------------------
// Short-term memories and audit
// ---------------------------------------------------------------------------

// ListMemoryStoreMessagesRequest is the request for ListMemoryStoreMessages.
// Requires the full 4-field scope. Timestamps are RFC3339 strings.
type ListMemoryStoreMessagesRequest struct {
	MemoryStoreName string    `json:"memoryStoreName"`
	Scope           Scope     `json:"scope"`
	Limit           int       `json:"limit,omitempty"`
	NextToken       string    `json:"nextToken,omitempty"`
	MinTimestamp    Timestamp `json:"minTimestamp,omitempty"`
	MaxTimestamp    Timestamp `json:"maxTimestamp,omitempty"`
}

// SessionMessage is one raw conversation message.
type SessionMessage struct {
	MessageID string            `json:"messageId,omitempty"`
	Role      string            `json:"role"`
	Speaker   string            `json:"speaker,omitempty"`
	Content   string            `json:"content"`
	Timestamp string            `json:"timestamp,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// MemoryStoreSession bundles the messages of one scope.
type MemoryStoreSession struct {
	Scope     Scope            `json:"scope"`
	Messages  []SessionMessage `json:"messages"`
	NextToken string           `json:"nextToken,omitempty"`
}

// ListMemoryStoreMessagesResponse is the response of ListMemoryStoreMessages.
type ListMemoryStoreMessagesResponse struct {
	Session   MemoryStoreSession `json:"session"`
	NextToken string             `json:"nextToken,omitempty"`
}

// ListMemoryStoreRequestsRequest is the request for ListMemoryStoreRequests.
// Timestamps are RFC3339 strings.
type ListMemoryStoreRequestsRequest struct {
	MemoryStoreName string    `json:"memoryStoreName"`
	Scope           *Scope    `json:"scope,omitempty"`
	Operation       string    `json:"operation,omitempty"`
	Limit           int       `json:"limit,omitempty"`
	NextToken       string    `json:"nextToken,omitempty"`
	MinTimestamp    Timestamp `json:"minTimestamp,omitempty"`
	MaxTimestamp    Timestamp `json:"maxTimestamp,omitempty"`
}

// MemoryStoreRequestRecord is one audit record.
type MemoryStoreRequestRecord struct {
	RequestID       string `json:"requestId"`
	Operation       string `json:"operation"`
	Scope           *Scope `json:"scope,omitempty"`
	TargetID        string `json:"targetId,omitempty"`
	ResponseStatus  int    `json:"responseStatus,omitempty"`
	LatencyMs       int64  `json:"latencyMs,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty"`
	RequestSummary  string `json:"requestSummary,omitempty"`
	ResponseSummary string `json:"responseSummary,omitempty"`
	IngestStatus    string `json:"ingestStatus,omitempty"`
	LastError       string `json:"lastError,omitempty"`
}

// ListMemoryStoreRequestsResponse is the response of ListMemoryStoreRequests.
type ListMemoryStoreRequestsResponse struct {
	Requests        []MemoryStoreRequestRecord `json:"requests"`
	Scope           *Scope                     `json:"scope,omitempty"`
	MemoryStoreName string                     `json:"memoryStoreName"`
	NextToken       string                     `json:"nextToken,omitempty"`
}

// ---------------------------------------------------------------------------
// Async ingestion tasks and scopes
// ---------------------------------------------------------------------------

// MemoryTaskStatus enumerates async ingestion task states.
type MemoryTaskStatus string

const (
	MemoryTaskStatusQueued         MemoryTaskStatus = "queued"
	MemoryTaskStatusRunning        MemoryTaskStatus = "running"
	MemoryTaskStatusCompleted      MemoryTaskStatus = "completed"
	MemoryTaskStatusFailed         MemoryTaskStatus = "failed"
	MemoryTaskStatusNeedsReconcile MemoryTaskStatus = "needs_reconcile"
)

// GetMemoryTaskRequest is the request for GetMemoryTask.
type GetMemoryTaskRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	RequestID       string `json:"requestId"`
	Scope           *Scope `json:"scope,omitempty"`
}

// ListMemoryTasksRequest is the request for ListMemoryTasks. Timestamps are
// Unix milliseconds.
type ListMemoryTasksRequest struct {
	MemoryStoreName string           `json:"memoryStoreName"`
	Scope           Scope            `json:"scope"`
	Status          MemoryTaskStatus `json:"status,omitempty"`
	Limit           int              `json:"limit,omitempty"`
	NextToken       string           `json:"nextToken,omitempty"`
	MinTimestamp    Timestamp        `json:"minTimestamp,omitempty"`
	MaxTimestamp    Timestamp        `json:"maxTimestamp,omitempty"`
}

// ListMemoryStoreScopesRequest is the request for ListMemoryStoreScopes.
type ListMemoryStoreScopesRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	Scope           Scope  `json:"scope"`
	Limit           int    `json:"limit,omitempty"`
	NextToken       string `json:"nextToken,omitempty"`
}

// MemoryTask is the status record of one async ingestion request.
type MemoryTask struct {
	RequestID        string           `json:"requestId"`
	EventType        string           `json:"eventType,omitempty"`
	MemoryStoreName  string           `json:"memoryStoreName,omitempty"`
	ConversationKey  string           `json:"conversationKey,omitempty"`
	Scope            Scope            `json:"scope"`
	Status           MemoryTaskStatus `json:"status"`
	AcceptedMessages int              `json:"acceptedMessages,omitempty"`
	DerivedMemcellID string           `json:"derivedMemcellId,omitempty"`
	DerivedUnitIDs   []string         `json:"derivedUnitIds,omitempty"`
	LastError        string           `json:"lastError,omitempty"`
	CreatedAt        string           `json:"createdAt,omitempty"`
	UpdatedAt        string           `json:"updatedAt,omitempty"`
	FinishedAt       string           `json:"finishedAt,omitempty"`
}

// GetMemoryTaskResponse is the response of GetMemoryTask.
type GetMemoryTaskResponse struct {
	Task            MemoryTask `json:"task"`
	MemoryStoreName string     `json:"memoryStoreName"`
}

// ListMemoryTasksResponse is the response of ListMemoryTasks.
type ListMemoryTasksResponse struct {
	Tasks           []MemoryTask `json:"tasks"`
	Scope           Scope        `json:"scope"`
	MemoryStoreName string       `json:"memoryStoreName"`
	NextToken       string       `json:"nextToken,omitempty"`
}

// ListMemoryStoreScopesResponse is the response of ListMemoryStoreScopes.
type ListMemoryStoreScopesResponse struct {
	Scopes          []Scope `json:"scopes"`
	Scope           Scope   `json:"scope"`
	MemoryStoreName string  `json:"memoryStoreName"`
	NextToken       string  `json:"nextToken,omitempty"`
}

// ---------------------------------------------------------------------------
// Memory dreaming (consolidation)
// ---------------------------------------------------------------------------

// DreamTaskStatus enumerates dream task states.
type DreamTaskStatus string

const (
	DreamTaskStatusQueued                DreamTaskStatus = "queued"
	DreamTaskStatusRunning               DreamTaskStatus = "running"
	DreamTaskStatusPlanning              DreamTaskStatus = "planning"
	DreamTaskStatusApplying              DreamTaskStatus = "applying"
	DreamTaskStatusCompleted             DreamTaskStatus = "completed"
	DreamTaskStatusCompletedWithFailures DreamTaskStatus = "completed_with_failures"
	DreamTaskStatusFailed                DreamTaskStatus = "failed"
	DreamTaskStatusCancelled             DreamTaskStatus = "cancelled"
)

// DreamActionStatus enumerates dream action states.
type DreamActionStatus string

const (
	DreamActionStatusProposed DreamActionStatus = "proposed"
	DreamActionStatusApplied  DreamActionStatus = "applied"
	DreamActionStatusSkipped  DreamActionStatus = "skipped"
	DreamActionStatusFailed   DreamActionStatus = "failed"
)

// DreamApplyMode enumerates dream apply modes.
type DreamApplyMode string

const (
	DreamApplyModeProposal DreamApplyMode = "proposal"
	DreamApplyModeSafeAuto DreamApplyMode = "safe_auto"
)

// DreamScopeOutputMode enumerates where consolidated memories are written.
type DreamScopeOutputMode string

const (
	DreamScopeOutputPreserve DreamScopeOutputMode = "preserve_scope"
	DreamScopeOutputPromote  DreamScopeOutputMode = "promote_scope"
)

// DreamActionType enumerates dream action kinds.
type DreamActionType string

const (
	DreamActionAdd         DreamActionType = "ADD"
	DreamActionUpdate      DreamActionType = "UPDATE"
	DreamActionDelete      DreamActionType = "DELETE"
	DreamActionMerge       DreamActionType = "MERGE"
	DreamActionNoop        DreamActionType = "NOOP"
	DreamActionEmitSkill   DreamActionType = "EMIT_SKILL"
	DreamActionEmitProfile DreamActionType = "EMIT_PROFILE"
)

// DreamTaskType enumerates dream task kinds.
type DreamTaskType string

const (
	DreamTaskTypeMemory  DreamTaskType = "memory"
	DreamTaskTypeSkill   DreamTaskType = "skill"
	DreamTaskTypeProfile DreamTaskType = "profile"
)

// DreamActionListOrderBy enumerates ListMemoryDreamActions orderings.
type DreamActionListOrderBy string

const (
	DreamActionOrderCreatedAtAsc   DreamActionListOrderBy = "created_at_asc"
	DreamActionOrderCreatedAtDesc  DreamActionListOrderBy = "created_at_desc"
	DreamActionOrderConfidenceDesc DreamActionListOrderBy = "confidence_desc"
)

// CreateMemoryDreamTaskRequest is the request for CreateMemoryDreamTask.
// Timestamps are Unix milliseconds.
type CreateMemoryDreamTaskRequest struct {
	MemoryStoreName      string               `json:"memoryStoreName"`
	Scopes               []Scope              `json:"scopes"`
	TaskType             DreamTaskType        `json:"taskType,omitempty"`
	MinTimestamp         Timestamp            `json:"minTimestamp,omitempty"`
	MaxTimestamp         Timestamp            `json:"maxTimestamp,omitempty"`
	MaxSessions          int                  `json:"maxSessions,omitempty"`
	MaxMessages          int                  `json:"maxMessages,omitempty"`
	MaxMemories          int                  `json:"maxMemories,omitempty"`
	ExpandedScopeLimit   int                  `json:"expandedScopeLimit,omitempty"`
	ScopeOutputMode      DreamScopeOutputMode `json:"scopeOutputMode,omitempty"`
	ApplyMode            DreamApplyMode       `json:"applyMode,omitempty"`
	ConfidenceThresholds map[string]float64   `json:"confidenceThresholds,omitempty"`
	Instructions         string               `json:"instructions,omitempty"`
	// Incremental resumes from the previous successful run's watermark when
	// true; nil/false reprocesses the full window.
	Incremental *bool  `json:"incremental,omitempty"`
	ClientToken string `json:"clientToken,omitempty"`
}

// GetMemoryDreamTaskRequest is the request for GetMemoryDreamTask.
type GetMemoryDreamTaskRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	DreamID         string `json:"dreamId"`
}

// ListMemoryDreamTasksRequest is the request for ListMemoryDreamTasks.
// Timestamps are Unix milliseconds.
type ListMemoryDreamTasksRequest struct {
	MemoryStoreName string          `json:"memoryStoreName"`
	Scope           Scope           `json:"scope"`
	Status          DreamTaskStatus `json:"status,omitempty"`
	MinTimestamp    Timestamp       `json:"minTimestamp,omitempty"`
	MaxTimestamp    Timestamp       `json:"maxTimestamp,omitempty"`
	Limit           int             `json:"limit,omitempty"`
	NextToken       string          `json:"nextToken,omitempty"`
}

// CancelMemoryDreamTaskRequest is the request for CancelMemoryDreamTask.
type CancelMemoryDreamTaskRequest struct {
	MemoryStoreName string `json:"memoryStoreName"`
	DreamID         string `json:"dreamId"`
}

// ListMemoryDreamActionsRequest is the request for ListMemoryDreamActions.
// Query either by DreamID, or by Scope + ActionType (EMIT_SKILL/EMIT_PROFILE
// only); the two modes are mutually exclusive.
type ListMemoryDreamActionsRequest struct {
	MemoryStoreName string                 `json:"memoryStoreName"`
	DreamID         string                 `json:"dreamId,omitempty"`
	Scope           *Scope                 `json:"scope,omitempty"`
	ActionType      DreamActionType        `json:"actionType,omitempty"`
	Status          DreamActionStatus      `json:"status,omitempty"`
	Action          DreamActionType        `json:"action,omitempty"`
	TargetScope     *Scope                 `json:"targetScope,omitempty"`
	SourceMemoryID  string                 `json:"sourceMemoryId,omitempty"`
	MinConfidence   *float64               `json:"minConfidence,omitempty"`
	MaxConfidence   *float64               `json:"maxConfidence,omitempty"`
	OrderBy         DreamActionListOrderBy `json:"orderBy,omitempty"`
	Limit           int                    `json:"limit,omitempty"`
	NextToken       string                 `json:"nextToken,omitempty"`
}

// ApplyMemoryDreamActionsRequest is the request for ApplyMemoryDreamActions.
type ApplyMemoryDreamActionsRequest struct {
	MemoryStoreName string   `json:"memoryStoreName"`
	DreamID         string   `json:"dreamId"`
	ActionIDs       []string `json:"actionIds"`
	Applier         string   `json:"applier,omitempty"`
}

// DreamMemorySpec is the proposed memory content of an ADD/UPDATE/MERGE action.
type DreamMemorySpec struct {
	Text       string            `json:"text"`
	SearchText string            `json:"searchText,omitempty"`
	UnitType   string            `json:"unitType,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// DreamExtractSpec is the body of an extract produced by an EMIT_SKILL or
// EMIT_PROFILE action; Kind is "skill" or "profile".
type DreamExtractSpec struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

// DreamActionRecord is one action produced by a dream task.
type DreamActionRecord struct {
	DreamID               string            `json:"dreamId"`
	ActionID              string            `json:"actionId"`
	MemoryStoreName       string            `json:"memoryStoreName,omitempty"`
	Action                DreamActionType   `json:"action"`
	Status                DreamActionStatus `json:"status"`
	TargetScope           Scope             `json:"targetScope"`
	SourceScopes          []Scope           `json:"sourceScopes,omitempty"`
	TargetMemoryID        string            `json:"targetMemoryId,omitempty"`
	TargetMemoryVersion   int               `json:"targetMemoryVersion,omitempty"`
	TargetMemoryUpdatedAt int64             `json:"targetMemoryUpdatedAtMs,omitempty"`
	NewMemory             *DreamMemorySpec  `json:"newMemory,omitempty"`
	NewExtract            *DreamExtractSpec `json:"newExtract,omitempty"`
	SourceMessageIDs      []string          `json:"sourceMessageIds,omitempty"`
	SourceMemoryIDs       []string          `json:"sourceMemoryIds,omitempty"`
	Reason                string            `json:"reason,omitempty"`
	Confidence            float64           `json:"confidence,omitempty"`
	ResultMemoryID        string            `json:"resultMemoryId,omitempty"`
	ApplyStep             string            `json:"applyStep,omitempty"`
	ApplyDetailJSON       string            `json:"applyDetailJson,omitempty"`
	SkippedReason         string            `json:"skippedReason,omitempty"`
	SkippedReasonDetail   string            `json:"skippedReasonDetail,omitempty"`
	AppliedAt             string            `json:"appliedAt,omitempty"`
	Applier               string            `json:"applier,omitempty"`
	AttemptCount          int               `json:"attemptCount,omitempty"`
	Error                 string            `json:"error,omitempty"`
	CreatedAt             string            `json:"createdAt,omitempty"`
	UpdatedAt             string            `json:"updatedAt,omitempty"`
}

// DreamTask is one dream task as returned by ListMemoryDreamTasks.
type DreamTask struct {
	DreamID              string               `json:"dreamId"`
	MemoryStoreName      string               `json:"memoryStoreName,omitempty"`
	Status               DreamTaskStatus      `json:"status"`
	TaskType             DreamTaskType        `json:"taskType,omitempty"`
	ApplyMode            DreamApplyMode       `json:"applyMode,omitempty"`
	ScopeOutputMode      DreamScopeOutputMode `json:"scopeOutputMode,omitempty"`
	ClientToken          string               `json:"clientToken,omitempty"`
	TriggerType          string               `json:"triggerType,omitempty"`
	ExpandedScopeLimit   int                  `json:"expandedScopeLimit,omitempty"`
	InputScopes          []Scope              `json:"inputScopes,omitempty"`
	ExpandedScopes       []Scope              `json:"expandedScopes,omitempty"`
	TargetScopes         []Scope              `json:"targetScopes,omitempty"`
	MinTimestamp         int64                `json:"minTimestamp,omitempty"`
	MaxTimestamp         int64                `json:"maxTimestamp,omitempty"`
	MaxSessions          int                  `json:"maxSessions,omitempty"`
	MaxMessages          int                  `json:"maxMessages,omitempty"`
	MaxMemories          int                  `json:"maxMemories,omitempty"`
	ConfidenceThresholds map[string]float64   `json:"confidenceThresholds,omitempty"`
	Instructions         string               `json:"instructions,omitempty"`
	Incremental          bool                 `json:"incremental,omitempty"`
	// ProcessedThroughMs is the watermark incremental runs resume from.
	ProcessedThroughMs int64          `json:"processedThroughMs,omitempty"`
	SkippedByReason    map[string]int `json:"skippedByReason,omitempty"`
	SessionCount       int            `json:"sessionCount,omitempty"`
	MessageCount       int            `json:"messageCount,omitempty"`
	MemoryCount        int            `json:"memoryCount,omitempty"`
	ActionCount        int            `json:"actionCount,omitempty"`
	ProposedCount      int            `json:"proposedCount,omitempty"`
	AppliedCount       int            `json:"appliedCount,omitempty"`
	SkippedCount       int            `json:"skippedCount,omitempty"`
	FailedCount        int            `json:"failedCount,omitempty"`
	LastError          string         `json:"lastError,omitempty"`
	Version            int            `json:"version,omitempty"`
	CreatedAt          string         `json:"createdAt,omitempty"`
	UpdatedAt          string         `json:"updatedAt,omitempty"`
	FinishedAt         string         `json:"finishedAt,omitempty"`
}

// DreamTaskInputSummary summarizes the input window of a dream task.
type DreamTaskInputSummary struct {
	Scopes       []Scope `json:"scopes,omitempty"`
	MinTimestamp int64   `json:"minTimestamp,omitempty"`
	MaxTimestamp int64   `json:"maxTimestamp,omitempty"`
	SessionCount int     `json:"sessionCount,omitempty"`
	MessageCount int     `json:"messageCount,omitempty"`
	MemoryCount  int     `json:"memoryCount,omitempty"`
	Incremental  bool    `json:"incremental,omitempty"`
}

// DreamTaskActionStats aggregates action counts of a dream task.
type DreamTaskActionStats struct {
	Total    int `json:"total"`
	Proposed int `json:"proposed"`
	Applied  int `json:"applied"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
	// SkippedByReason breaks the skipped count down by skip reason.
	SkippedByReason map[string]int `json:"skippedByReason,omitempty"`
}

// CreateMemoryDreamTaskResponse is the response of CreateMemoryDreamTask.
type CreateMemoryDreamTaskResponse struct {
	MemoryStoreName string          `json:"memoryStoreName"`
	DreamID         string          `json:"dreamId"`
	Status          DreamTaskStatus `json:"status"`
	CreatedAt       string          `json:"createdAt,omitempty"`
}

// GetMemoryDreamTaskResponse is the response of GetMemoryDreamTask.
type GetMemoryDreamTaskResponse struct {
	MemoryStoreName string                `json:"memoryStoreName"`
	DreamID         string                `json:"dreamId"`
	TaskType        DreamTaskType         `json:"taskType,omitempty"`
	Status          DreamTaskStatus       `json:"status"`
	ApplyMode       DreamApplyMode        `json:"applyMode,omitempty"`
	ScopeOutputMode DreamScopeOutputMode  `json:"scopeOutputMode,omitempty"`
	Input           DreamTaskInputSummary `json:"input"`
	Actions         DreamTaskActionStats  `json:"actions"`
	Instructions    string                `json:"instructions,omitempty"`
	// ProcessedThroughMs is the watermark incremental runs resume from.
	ProcessedThroughMs int64  `json:"processedThroughMs,omitempty"`
	LastError          string `json:"lastError,omitempty"`
	CreatedAt          string `json:"createdAt,omitempty"`
	UpdatedAt          string `json:"updatedAt,omitempty"`
	FinishedAt         string `json:"finishedAt,omitempty"`
}

// ListMemoryDreamTasksResponse is the response of ListMemoryDreamTasks.
type ListMemoryDreamTasksResponse struct {
	MemoryStoreName string      `json:"memoryStoreName"`
	Scope           Scope       `json:"scope"`
	Tasks           []DreamTask `json:"tasks"`
	NextToken       string      `json:"nextToken,omitempty"`
}

// CancelMemoryDreamTaskResponse is the response of CancelMemoryDreamTask.
type CancelMemoryDreamTaskResponse struct {
	MemoryStoreName string          `json:"memoryStoreName"`
	DreamID         string          `json:"dreamId"`
	TaskType        DreamTaskType   `json:"taskType,omitempty"`
	Status          DreamTaskStatus `json:"status"`
}

// ListMemoryDreamActionsResponse is the response of ListMemoryDreamActions.
type ListMemoryDreamActionsResponse struct {
	MemoryStoreName string              `json:"memoryStoreName"`
	DreamID         string              `json:"dreamId,omitempty"`
	Actions         []DreamActionRecord `json:"actions"`
	NextToken       string              `json:"nextToken,omitempty"`
}

// DreamApplyActionResult is the per-action outcome of ApplyMemoryDreamActions.
type DreamApplyActionResult struct {
	ActionID string            `json:"actionId"`
	Status   DreamActionStatus `json:"status"`
	MemoryID string            `json:"memoryId,omitempty"`
	Error    string            `json:"error,omitempty"`
}

// ApplyMemoryDreamActionsResponse is the response of ApplyMemoryDreamActions.
type ApplyMemoryDreamActionsResponse struct {
	MemoryStoreName string                   `json:"memoryStoreName"`
	DreamID         string                   `json:"dreamId"`
	Applied         int                      `json:"applied"`
	Failed          int                      `json:"failed"`
	NotFound        int                      `json:"notFound,omitempty"`
	Results         []DreamApplyActionResult `json:"results"`
}
