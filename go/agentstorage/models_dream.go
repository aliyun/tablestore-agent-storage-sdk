package agentstorage

import "encoding/json"

// DreamEffort selects the requested reasoning depth. High effort must be
// enabled on the server; the service never silently downgrades it.
type DreamEffort string

const (
	DreamEffortLow                      DreamEffort       = "low"
	DreamEffortHigh                     DreamEffort       = "high"
	DreamApplyModeAuto                  DreamApplyMode    = "auto"
	DreamApplyModePlan                  DreamApplyMode    = "plan"
	DreamTaskStatusAwaitingPlanApproval DreamTaskStatus   = "awaiting_plan_approval"
	DreamActionStatusStaged             DreamActionStatus = "staged"
	DreamActionCorrectWiki              DreamActionType   = "CORRECT_WIKI"
	DreamApplyRequestModeActions                          = "actions"
	DreamApplyRequestModePlan                             = "plan"
	DreamTaskStagePreparing                               = "preparing"
	DreamTaskStageExecuting                               = "executing"
)

// DreamPlan includes the exact deletion IDs subject to plan approval.
type DreamPlan struct {
	Summary            string   `json:"summary"`
	DeleteMemoryIDs    []string `json:"deleteMemoryIds"`
	ApprovalExpiresAt  string   `json:"approvalExpiresAt,omitempty"`
	ApprovedAt         string   `json:"approvedAt,omitempty"`
	Approver           string   `json:"approver,omitempty"`
	ExecutionExpiresAt string   `json:"executionExpiresAt,omitempty"`
}

// DreamFrozenSourceRef identifies one frozen source used by an action.
// Scope uses the wire array [appId, tenantId, agentId, runId].
type DreamFrozenSourceRef struct {
	Kind               string    `json:"kind"`
	Scope              [4]string `json:"scope"`
	ConversationKey    string    `json:"conversationKey"`
	SourceID           string    `json:"sourceId"`
	MessageTimestampMs int64     `json:"messageTimestampMs"`
	SourceRefID        string    `json:"sourceRefId"`
	CreatedAtMs        *int64    `json:"createdAtMs"`
	Version            *int64    `json:"version"`
	SnapshotHash       string    `json:"snapshotHash"`
}

// DreamDiagnosticFailure is returned in Dream task/action responses.
type DreamDiagnosticFailure struct {
	Tool              string `json:"tool"`
	Code              string `json:"code"`
	Reason            string `json:"reason"`
	RecoveryKind      string `json:"recoveryKind"`
	CandidateDigest   string `json:"candidateDigest"`
	Message           string `json:"message"`
	MessageTruncated  bool   `json:"messageTruncated"`
	ObservedAtMs      int64  `json:"observedAtMs"`
	RecoveryRemaining *int   `json:"recoveryRemaining,omitempty"`
}

// DreamDiagnostic is returned in Dream task/action responses.
type DreamDiagnostic struct {
	SchemaVersion   int                      `json:"schemaVersion"`
	TaskID          string                   `json:"taskId"`
	SandboxID       string                   `json:"sandboxId,omitempty"`
	Phase           string                   `json:"phase,omitempty"`
	OmittedFailures int                      `json:"omittedFailures"`
	Failures        []DreamDiagnosticFailure `json:"failures"`
}

// DreamItemChange is returned in Dream task/action responses.
type DreamItemChange struct {
	ItemID            string   `json:"itemId,omitempty"`
	Path              *string  `json:"path,omitempty"`
	Content           *string  `json:"content,omitempty"`
	ExpectedVersionID string   `json:"expectedVersionId,omitempty"`
	SourceVersionIDs  []string `json:"sourceVersionIds,omitempty"`
}

// DreamWikiCorrectionSpec is returned in Dream task/action responses.
type DreamWikiCorrectionSpec struct {
	Mode              string `json:"mode,omitempty"`
	WikiID            string `json:"wikiId"`
	PagePath          string `json:"pagePath"`
	BaseContentSHA256 string `json:"baseContentSha256,omitempty"`
	ExpectedOldText   string `json:"expectedOldText,omitempty"`
	ReplacementText   string `json:"replacementText,omitempty"`
	AnchorPrefix      string `json:"anchorPrefix,omitempty"`
	AnchorSuffix      string `json:"anchorSuffix,omitempty"`
	SearchText        string `json:"searchText,omitempty"`
	CorrectionID      string `json:"correctionId,omitempty"`
	SessionID         string `json:"sessionId,omitempty"`
	TurnID            string `json:"turnId,omitempty"`
	ReplyToTurnID     string `json:"replyToTurnId,omitempty"`
}

// UnmarshalJSON accepts both the current nested list response and the older
// flattened response. Legacy count/scope fields remain populated for existing
// callers; Input and Actions provide the canonical current representation.
func (t *DreamTask) UnmarshalJSON(data []byte) error {
	type plain DreamTask
	var wire struct {
		plain
		Input   *DreamTaskInputSummary `json:"input"`
		Actions *DreamTaskActionStats  `json:"actions"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*t = DreamTask(wire.plain)
	if wire.Input != nil {
		t.Input = *wire.Input
		t.InputScopes, t.ExpandedScopes = t.Input.Scopes, t.Input.ExpandedScopes
		t.MinTimestamp, t.MaxTimestamp = t.Input.MinTimestamp, t.Input.MaxTimestamp
		t.SessionCount, t.MessageCount, t.MemoryCount = t.Input.SessionCount, t.Input.MessageCount, t.Input.MemoryCount
		t.Incremental = t.Input.Incremental
	} else {
		t.Input = DreamTaskInputSummary{Scopes: t.InputScopes, ExpandedScopes: t.ExpandedScopes, MinTimestamp: t.MinTimestamp, MaxTimestamp: t.MaxTimestamp, SessionCount: t.SessionCount, MessageCount: t.MessageCount, MemoryCount: t.MemoryCount, Incremental: t.Incremental}
	}
	if wire.Actions != nil {
		t.Actions = *wire.Actions
		t.ActionCount, t.ProposedCount, t.AppliedCount = t.Actions.Total, t.Actions.Proposed, t.Actions.Applied
		t.SkippedCount, t.FailedCount, t.SkippedByReason = t.Actions.Skipped, t.Actions.Failed, t.Actions.SkippedByReason
	} else {
		t.Actions = DreamTaskActionStats{Total: t.ActionCount, Proposed: t.ProposedCount, Applied: t.AppliedCount, Skipped: t.SkippedCount, Failed: t.FailedCount, SkippedByReason: t.SkippedByReason}
	}
	return nil
}
