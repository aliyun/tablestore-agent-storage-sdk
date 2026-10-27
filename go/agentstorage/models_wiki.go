package agentstorage

// AgentScope identifies one Wiki; runId is not part of a Wiki scope.
type AgentScope struct {
	AppID    string `json:"appId"`
	TenantID string `json:"tenantId"`
	AgentID  string `json:"agentId"`
}

// WikiScopedRequest is part of the public Wiki API.
type WikiScopedRequest struct {
	MemoryStoreName string     `json:"memoryStoreName"`
	Scope           AgentScope `json:"scope"`
}

// Wiki is part of the public Wiki API.
type Wiki struct {
	Name              string              `json:"name"`
	Summary           string              `json:"summary,omitempty"`
	Status            string              `json:"status"`
	PageCount         int                 `json:"pageCount"`
	HasPendingChanges bool                `json:"hasPendingChanges"`
	CompileOptions    *WikiCompileOptions `json:"compileOptions,omitempty"`
	LatestBuild       *WikiBuild          `json:"latestBuild,omitempty"`
	CreatedAt         string              `json:"createdAt"`
	UpdatedAt         string              `json:"updatedAt"`
}

// WikiBuild is part of the public Wiki API.
type WikiBuild struct {
	Diagnostic       *WikiDiagnostic      `json:"diagnostic,omitempty"`
	PartialAvailable bool                 `json:"partialAvailable,omitempty"`
	CompletedSources int                  `json:"completedSources,omitempty"`
	TotalSources     int                  `json:"totalSources,omitempty"`
	CompletedPages   int                  `json:"completedPages,omitempty"` // a running OpenWiki build: the knowledge pages of its last published snapshot
	PlannedPages     int                  `json:"plannedPages,omitempty"`   // ...of the pages it plans
	Status           string               `json:"status"`
	Step             string               `json:"step,omitempty"`
	Progress         int                  `json:"progress,omitempty"`
	Changes          WikiBuildChanges     `json:"changes"`
	PageCount        int                  `json:"pageCount,omitempty"`
	TotalBytes       int64                `json:"totalBytes,omitempty"`
	Quality          *WikiBuildQuality    `json:"quality,omitempty"`
	ErrorMessage     string               `json:"errorMessage,omitempty"`
	ErrorCode        string               `json:"errorCode,omitempty"`
	FailureKind      string               `json:"failureKind,omitempty"`
	AttemptFailures  []WikiAttemptFailure `json:"attemptFailures,omitempty"`
	CreatedAt        string               `json:"createdAt"`
	UpdatedAt        string               `json:"updatedAt"`
	FinishedAt       *string              `json:"finishedAt,omitempty"`
}

// WikiAttemptFailure is part of the public Wiki API.
type WikiAttemptFailure struct {
	Attempt   int    `json:"attempt"`
	Kind      string `json:"kind"`
	Retryable bool   `json:"retryable"`
}

// WikiDiagnostic is part of the public Wiki API.
type WikiDiagnostic struct {
	OmittedFailures int                     `json:"omittedFailures"`
	Failures        []WikiDiagnosticFailure `json:"failures"`
}

// WikiDiagnosticFailure is part of the public Wiki API.
type WikiDiagnosticFailure struct {
	Tool             string `json:"tool"`
	Code             string `json:"code"`
	Reason           string `json:"reason"`
	Message          string `json:"message"`
	MessageTruncated bool   `json:"messageTruncated"`
	ObservedAtMs     int64  `json:"observedAtMs"`
}

// WikiBuildChanges is part of the public Wiki API.
type WikiBuildChanges struct {
	Added         int `json:"added"`
	Modified      int `json:"modified"`
	Deleted       int `json:"deleted"`
	AffectedPages int `json:"affectedPages"`
}

// WikiBuildQuality is part of the public Wiki API.
type WikiBuildQuality struct {
	SourceCoverage      float64 `json:"sourceCoverage"`
	GroundedPageRate    float64 `json:"groundedPageRate"`
	BrokenLinkCount     int     `json:"brokenLinkCount"`
	OrphanPageCount     int     `json:"orphanPageCount"`
	UncitedPageCount    int     `json:"uncitedPageCount"`
	DuplicateTitleCount int     `json:"duplicateTitleCount"`
}

// IngestWikiResponse is part of the public Wiki API.
type IngestWikiResponse struct {
	Status  string `json:"status"`
	Reused  bool   `json:"reused"`
	Skipped bool   `json:"skipped"`
}

// WikiPage is part of the public Wiki API.
type WikiPage struct {
	Path                   string              `json:"path"`
	Title                  string              `json:"title,omitempty"`
	Type                   string              `json:"type,omitempty"`
	Description            string              `json:"description,omitempty"`
	Content                string              `json:"content,omitempty"`
	Sources                []string            `json:"sources,omitempty"`
	OutgoingLinks          []string            `json:"outgoingLinks,omitempty"`
	IncomingLinks          []string            `json:"incomingLinks,omitempty"`
	IncomingLinkCount      int                 `json:"incomingLinkCount,omitempty"`
	IncomingLinksTruncated bool                `json:"incomingLinksTruncated,omitempty"`
	Score                  float64             `json:"score,omitempty"`
	Reflection             *WikiReflectionInfo `json:"reflection,omitempty"`
}

// WikiDreamActionRef is part of the public Wiki API.
type WikiDreamActionRef struct {
	DreamID  string `json:"dreamId"`
	ActionID string `json:"actionId"`
}

// WikiCompileOptions configures Wiki compilation.
type WikiCompileOptions struct {
	Instructions string                  `json:"instructions,omitempty"`
	Exclude      []string                `json:"exclude,omitempty"`
	DocSummary   *WikiDocSummaryOptions  `json:"docSummary,omitempty"`
	Language     string                  `json:"language,omitempty"`
	MaxPageBytes int                     `json:"maxPageBytes,omitempty"`
	Incremental  *WikiIncrementalOptions `json:"incremental,omitempty"`
}

// WikiDocSummaryOptions configures Wiki compilation.
type WikiDocSummaryOptions struct {
	Enabled bool `json:"enabled"`
}

// WikiIncrementalOptions configures Wiki compilation.
type WikiIncrementalOptions struct {
	Enabled bool `json:"enabled"`
}

// WikiReflectionInfo describes corrections applied to the returned page.
type WikiReflectionInfo struct {
	Status                 string   `json:"status"`
	CorrectionCount        int      `json:"correctionCount,omitempty"`
	BaseContentSHA256      string   `json:"baseContentSha256,omitempty"`
	EffectiveContentSHA256 string   `json:"effectiveContentSha256,omitempty"`
	AppliedActionIDs       []string `json:"appliedActionIds,omitempty"`
	MatchedBy              string   `json:"matchedBy,omitempty"`
}

type GetWikiRequest struct{ WikiScopedRequest }
type GetWikiResponse = Wiki
type UpdateWikiResponse = Wiki

// UpdateWikiRequest patches metadata and build options. A nil CompileOptions
// leaves options unchanged; Null[WikiCompileOptions]() or an empty value resets
// them. Use NullableOf(WikiCompileOptions{...}) to replace them.
type UpdateWikiRequest struct {
	WikiScopedRequest
	Name           *string                       `json:"name,omitempty"`
	Summary        *string                       `json:"summary,omitempty"`
	CompileOptions *Nullable[WikiCompileOptions] `json:"compileOptions,omitempty"`
}
type IngestWikiRequest struct {
	WikiScopedRequest
	ClientToken     string               `json:"clientToken,omitempty"`
	DreamActionRefs []WikiDreamActionRef `json:"dreamActionRefs,omitempty"`
}
type ListWikiPagesRequest struct {
	WikiScopedRequest
	BuildID   string `json:"buildId,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	NextToken string `json:"nextToken,omitempty"`
}
type GetWikiPageRequest struct {
	WikiScopedRequest
	BuildID               string          `json:"buildId,omitempty"`
	Path                  string          `json:"path"`
	Interaction           *InteractionRef `json:"interaction,omitempty"`
	IncludeReflectionInfo bool            `json:"includeReflectionInfo,omitempty"`
}
type SearchWikiPagesRequest struct {
	WikiScopedRequest
	Query                 string          `json:"query"`
	Limit                 int             `json:"limit,omitempty"`
	Interaction           *InteractionRef `json:"interaction,omitempty"`
	IncludeReflectionInfo bool            `json:"includeReflectionInfo,omitempty"`
}
type ListWikiPagesResponse struct {
	Pages     []WikiPage `json:"pages"`
	NextToken string     `json:"nextToken,omitempty"`
}
type GetWikiPageResponse = WikiPage
type SearchWikiPagesResponse struct {
	Pages []WikiPage `json:"pages"`
}
