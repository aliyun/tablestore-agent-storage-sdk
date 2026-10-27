package agentstorage

import "encoding/json"

// Memory store modes that expose the item APIs. FileOTS is a read-only
// projection of structured memories; FileMemory stores files as source data.
const (
	StorageModeFileOTS    = "file+ots"
	StorageModeFileMemory = "filemem"
)

// ItemTypeMemoryFile is the only item type accepted by the item APIs. The
// SDK fills it into any item request whose Type field is empty.
const ItemTypeMemoryFile = "memoryfile"

// ItemTypeMemoryFilePrefix is the directory-marker type emitted by ListItems
// when Depth is 1 and a deeper subtree is rolled up. Prefix entries carry
// only Scope and Path.
const ItemTypeMemoryFilePrefix = "memoryfile_prefix"

// Service limits documented for item APIs.
const (
	// MaxItemContentBytes is the default text-content limit (100 KiB). The
	// effective limit is server-configured; binary-enabled deployments may
	// allow 2,000,000 decoded bytes. The SDK does not enforce this constant.
	MaxItemContentBytes = 102400

	// MaxItemPathBytes is the maximum normalized item path length, in bytes.
	MaxItemPathBytes = 1024

	// MaxItemsPerScope is the conventional per-mount-unit (store + exact
	// scope) file-count cap. Whether the service enforces it is deployment
	// configuration; when enforced, exceeding it fails with 409
	// QUOTA_EXCEEDED.
	MaxItemsPerScope = 2000
)

// ItemOperation is the operation recorded by an item version.
type ItemOperation string

const (
	ItemOperationCreated  ItemOperation = "created"
	ItemOperationModified ItemOperation = "modified"
	ItemOperationDeleted  ItemOperation = "deleted"
)

// ItemView selects how much payload a response carries. ItemViewBasic (the
// default when omitted) omits content; ItemViewFull includes it.
type ItemView string

const (
	ItemViewBasic ItemView = "basic"
	ItemViewFull  ItemView = "full"
)

// ItemActor is the actor attribution on version rows, including Dream writes.
type ItemActor struct {
	ActorType string `json:"actorType"`
	ActorID   string `json:"actorId"`
}

// ItemScopedRequest carries the memoryfile type, store and four-part scope.
// File-input stores accept writes; managed files from message-input stores are
// read-only. Exact operations require a concrete scope; list/range operations
// accept hierarchical wildcards with a concrete appId/tenantId. Wiki system
// items have additional server-enforced read-only rules.
type ItemScopedRequest struct {
	// Type discriminates the item type. Required by the service on all
	// item actions, and "memoryfile" is its only accepted value - the SDK
	// fills it in when the caller leaves it empty.
	Type            string `json:"type"`
	MemoryStoreName string `json:"memoryStoreName"`
	Scope           Scope  `json:"scope"`
}

// setDefaultType fills Type with ItemTypeMemoryFile when unset.
func (r *ItemScopedRequest) setDefaultType() {
	if r.Type == "" {
		r.Type = ItemTypeMemoryFile
	}
}

// Path rules, enforced by the service:
//
//   - must start with "/" - the service no longer adds a missing leading "/"
//   - at most MaxItemPathBytes bytes after NFC normalization
//   - "." and ".." segments, empty segments ("//"), a bare "/", a trailing
//     "/", NUL characters and invalid UTF-8 are all rejected with 400
//   - any valid UTF-8 is otherwise allowed, including CJK; matching is
//     byte-exact and case-sensitive

// AddItemRequest creates ordinary Markdown files (.md). Binary creation is
// limited to Wiki raw paths (/.wiki/raw/...) with the server capability enabled.
type AddItemRequest struct {
	// ContentBase64 sends binary content instead of Content. MediaType is required.
	ContentBase64 *string `json:"contentBase64,omitempty"`
	MediaType     string  `json:"mediaType,omitempty"`
	ItemScopedRequest
	Path string `json:"path"`
	// Content is UTF-8 text. The server applies its configured size limit
	// (see MaxItemContentBytes); oversized content fails with 413 and invalid
	// UTF-8 with 400.
	// Required; an empty string creates an empty file.
	Content string `json:"content"`
	// SessionID is recorded as the version's session actor.
	SessionID string `json:"sessionId,omitempty"`
	// View defaults to ItemViewBasic (response without content);
	// ItemViewFull echoes content.
	View ItemView `json:"view,omitempty"`
}

// ListItemsRequest is the request for ListItems. The scope may use the
// left-prefix wildcard (see ItemScopedRequest).
type ListItemsRequest struct {
	Filter *ScopeFilter `json:"filter,omitempty"`
	ItemScopedRequest
	// PathPrefix is a directory-style prefix. It must be segment-aligned:
	// start and end with "/" (a bare "/" lists the whole tree, same as
	// omitting it). "."/".." segments, NUL and control/format characters
	// are rejected with 400.
	PathPrefix string `json:"pathPrefix,omitempty"`
	// Depth 0 (the default when nil) lists the whole subtree recursively;
	// 1 lists only direct children and rolls deeper directories up into
	// entries typed ItemTypeMemoryFilePrefix. Any other value is rejected
	// with 400.
	Depth *int `json:"depth,omitempty"`
	// View defaults to ItemViewBasic; ItemViewFull includes each file's
	// content.
	View ItemView `json:"view,omitempty"`
	// Limit defaults to 20; maximum 100, and capped at 20 when View is
	// ItemViewFull.
	Limit int `json:"limit,omitempty"`
	// NextToken is an opaque cursor. It must be replayed with the same
	// PathPrefix that produced it; a cursor outside that prefix is
	// rejected with 400.
	NextToken string `json:"nextToken,omitempty"`
}

// GetItemRequest is the request for GetItem. It addresses the item by
// ItemID or Path (exactly one required).
type GetItemRequest struct {
	ItemScopedRequest
	ItemID string `json:"itemId,omitempty"`
	Path   string `json:"path,omitempty"`
	// View defaults to ItemViewBasic - metadata only. ItemViewFull
	// includes content. It replaces the removed includeContent flag; note
	// the default is inverted relative to the old includeContent=true
	// default.
	View ItemView `json:"view,omitempty"`
}

// UpdateItemRequest updates content and/or path atomically. ItemID identifies
// the item; with ItemID set, Path is the new path. Precondition guards the full
// resource version, including renames. At least one change is required.
//
// For older callers explicitly using NewPath or ExpectedSHA256, Path without
// ItemID is resolved with GetItem first. NewPath is a deprecated rename alias.
// ExpectedSHA256 is also deprecated:
// Client.UpdateItem checks the current hash and pins its version before writing.
type UpdateItemRequest struct {
	ContentBase64 *string           `json:"contentBase64,omitempty"`
	MediaType     string            `json:"mediaType,omitempty"`
	Precondition  *ItemPrecondition `json:"precondition,omitempty"`
	ItemScopedRequest
	ItemID  string  `json:"itemId"`
	Path    *string `json:"path,omitempty"`
	Content *string `json:"content,omitempty"`
	// Deprecated: use ItemID plus Path for renames.
	NewPath   string `json:"-"`
	Overwrite bool   `json:"overwrite,omitempty"`
	// Deprecated: use Precondition. Requires a GetItem read before the write.
	ExpectedSHA256 string `json:"-"`
	// SessionID is recorded as the new version's session actor.
	SessionID string `json:"sessionId,omitempty"`
	// View defaults to ItemViewBasic (response without content);
	// ItemViewFull echoes content.
	View ItemView `json:"view,omitempty"`
}

// DeleteItemRequest is the request for DeleteItem, in one of two modes:
//
//   - Exact delete: exact full scope plus exactly one of ItemID and Path,
//     with an optional resource-version Precondition. Synchronous; the
//     response carries Deleted=true.
//   - Ranged delete: wildcard scope (left-prefix rule, appId/tenantId
//     required). Asynchronous - the service answers 200 with a TaskID
//     identifying the sharded delete chain (it used to answer 202; the
//     accept status moved to 200 because the OTS gateway only forwards
//     200 responses). ItemID, Path and Precondition must all be omitted;
//     PathPrefix optionally narrows the range (segment-aligned, see
//     ListItemsRequest.PathPrefix).
type DeleteItemRequest struct {
	// DeleteVersions also removes history during a ranged delete only.
	DeleteVersions bool              `json:"deleteVersions,omitempty"`
	Precondition   *ItemPrecondition `json:"precondition,omitempty"`
	ItemScopedRequest
	ItemID string `json:"itemId,omitempty"`
	Path   string `json:"path,omitempty"`
	// PathPrefix applies to ranged deletes only: optional segment-aligned
	// narrowing of the scope.
	PathPrefix string `json:"pathPrefix,omitempty"`
	// Deprecated: use Precondition. Client.DeleteItem resolves and checks the
	// current hash, then pins the resource version. Ranged deletes reject it.
	ExpectedSHA256 string `json:"-"`
	// SessionID is recorded as the delete's session actor.
	SessionID string `json:"sessionId,omitempty"`
}

// ListItemVersionsRequest is the request for ListItemVersions - a
// scope-level version listing ordered by createdAt descending. All filters
// are optional; the scope may use the left-prefix wildcard (see
// ItemScopedRequest).
type ListItemVersionsRequest struct {
	Path string `json:"path,omitempty"`
	ItemScopedRequest
	// ItemID is an optional filter; when omitted, versions of every item
	// in scope match.
	ItemID string `json:"itemId,omitempty"`
	// SessionID is an actor filter.
	SessionID string `json:"sessionId,omitempty"`
	// Deprecated: no longer supported by the service. Nonempty values fail locally.
	ApiKeyID string `json:"-"`
	// CreatedAtGte is an inclusive RFC3339 lower bound on createdAt.
	CreatedAtGte string `json:"createdAtGte,omitempty"`
	// CreatedAtLte is an inclusive RFC3339 upper bound on createdAt.
	CreatedAtLte string `json:"createdAtLte,omitempty"`
	// Operation is case-sensitive; any other value is rejected with 400.
	Operation ItemOperation `json:"operation,omitempty"`
	// View defaults to ItemViewBasic; ItemViewFull includes each version's
	// content.
	View ItemView `json:"view,omitempty"`
	// Limit defaults to 20; maximum 100, and capped at 20 when View is
	// ItemViewFull.
	Limit     int    `json:"limit,omitempty"`
	NextToken string `json:"nextToken,omitempty"`
}

// GetItemVersionRequest is the request for GetItemVersion. VersionID is
// self-contained: the server derives all addressing from it and verifies
// ownership against the exact four-part scope (wildcards are rejected).
type GetItemVersionRequest struct {
	ItemScopedRequest
	VersionID string `json:"versionId"`
	// View defaults to ItemViewBasic; ItemViewFull includes content (never
	// on redacted versions).
	View ItemView `json:"view,omitempty"`
}

// RedactItemVersionRequest is the request for RedactItemVersion; the scope
// must be the exact four-part scope owning the version. It irreversibly
// redacts a single historical version snapshot: content, path,
// contentSha256 and contentSizeBytes are cleared while operation, createdAt
// and the original createdBy are kept. The current item state and latestSeq
// are untouched and no new version row is created.
//
// Redaction is idempotent and first-wins: repeating it returns 200 with the
// existing redactedAt/redactedBy. Afterwards GetItemVersion still returns
// 200 for the version but never returns its content.
type RedactItemVersionRequest struct {
	ItemScopedRequest
	VersionID string `json:"versionId"`
	// SessionID is recorded as the session actor in redactedBy on the
	// first redaction.
	SessionID string `json:"sessionId,omitempty"`
}

// Item is the current state of one item. Content is nil when the response
// omits it (the default basic view); an empty file has Content pointing at
// "". Type echoes ItemTypeMemoryFile in single-item responses.
type Item struct {
	Summary          string  `json:"summary,omitempty"`
	ContentBase64    *string `json:"contentBase64,omitempty"`
	MediaType        string  `json:"mediaType,omitempty"`
	Type             string  `json:"type,omitempty"`
	ItemID           string  `json:"itemId"`
	Scope            Scope   `json:"scope"`
	Path             string  `json:"path"`
	Content          *string `json:"content,omitempty"`
	ContentSHA256    string  `json:"contentSha256"`
	ContentSizeBytes int64   `json:"contentSizeBytes"`
	LatestSeq        int64   `json:"latestSeq"`
	// LatestVersionID is the head-version pointer.
	LatestVersionID string `json:"latestVersionId"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// ItemListEntry is one ListItems entry. File entries carry Type
// ItemTypeMemoryFile plus the list fields; binary content and summaries are
// omitted even in full view (use GetItem for those). Directory roll-ups with
// Depth 1 carry Type ItemTypeMemoryFilePrefix and only Scope and Path.
type ItemListEntry struct {
	MediaType        string  `json:"mediaType,omitempty"`
	Type             string  `json:"type"`
	ItemID           string  `json:"itemId,omitempty"`
	Scope            Scope   `json:"scope"`
	Path             string  `json:"path"`
	Content          *string `json:"content,omitempty"`
	ContentSHA256    string  `json:"contentSha256,omitempty"`
	ContentSizeBytes int64   `json:"contentSizeBytes,omitempty"`
	LatestSeq        int64   `json:"latestSeq,omitempty"`
	LatestVersionID  string  `json:"latestVersionId,omitempty"`
	CreatedAt        string  `json:"createdAt,omitempty"`
	UpdatedAt        string  `json:"updatedAt,omitempty"`
}

// IsPrefix reports whether the entry is a directory roll-up marker.
func (e *ItemListEntry) IsPrefix() bool { return e.Type == ItemTypeMemoryFilePrefix }

// ItemVersion is one immutable item version. Content is nil when the
// response omits it (the default basic view, or redacted versions); an
// empty version has Content pointing at "". The redaction fields appear
// only after RedactItemVersion.
type ItemVersion struct {
	ContentBase64    *string       `json:"contentBase64,omitempty"`
	MediaType        string        `json:"mediaType,omitempty"`
	Type             string        `json:"type,omitempty"`
	VersionID        string        `json:"versionId"`
	ItemID           string        `json:"itemId"`
	Scope            Scope         `json:"scope"`
	Operation        ItemOperation `json:"operation"`
	Path             string        `json:"path"`
	Content          *string       `json:"content,omitempty"`
	ContentSHA256    string        `json:"contentSha256"`
	ContentSizeBytes int64         `json:"contentSizeBytes"`
	// CreatedBy is the typed actor that wrote this version.
	CreatedBy *ItemActor `json:"createdBy,omitempty"`
	CreatedAt string     `json:"createdAt"`
	Redacted  bool       `json:"redacted,omitempty"`
	// RedactedAt is the time of the first redaction. Present only once
	// redacted.
	RedactedAt string `json:"redactedAt,omitempty"`
	// RedactedBy is the typed actor of the first redaction. Present only
	// once redacted.
	RedactedBy *ItemActor `json:"redactedBy,omitempty"`
}

type (
	AddItemResponse           = Item
	GetItemResponse           = Item
	UpdateItemResponse        = Item
	GetItemVersionResponse    = ItemVersion
	RedactItemVersionResponse = ItemVersion
)

// DeleteItemResponse covers both DeleteItem outcomes: a synchronous single
// delete carries ItemID and Deleted=true; an asynchronous ranged delete
// carries TaskID and Status="pending" (also HTTP 200).
type DeleteItemResponse struct {
	Type    string `json:"type"`
	ItemID  string `json:"itemId,omitempty"`
	Deleted bool   `json:"deleted,omitempty"`
	TaskID  string `json:"taskId,omitempty"`
	Status  string `json:"status,omitempty"`
}

// IsRanged reports whether the response came from an asynchronous ranged
// delete.
func (r *DeleteItemResponse) IsRanged() bool { return r.TaskID != "" }

// ListItemsResponse is the response of ListItems. ReadOnly is true for
// file+ots projections and omitted for writable stores.
type ListItemsResponse struct {
	Type      string          `json:"type"`
	Items     []ItemListEntry `json:"items"`
	NextToken string          `json:"nextToken,omitempty"`
	ReadOnly  bool            `json:"readOnly,omitempty"`
}

// ListItemVersionsResponse is the response of ListItemVersions.
type ListItemVersionsResponse struct {
	Type      string        `json:"type"`
	Versions  []ItemVersion `json:"versions"`
	NextToken string        `json:"nextToken,omitempty"`
}

// ItemPrecondition guards the complete resource version (content and path).
type ItemPrecondition struct {
	ExpectedVersionID string `json:"expectedVersionId"`
}

// SearchItemsRequest searches managed file summaries. Filter uses ScopeFilter,
// and Thresholds applies to the returned similarity/canonical score.
type SearchItemsRequest struct {
	ItemScopedRequest
	Query      string            `json:"query"`
	Filter     *ScopeFilter      `json:"filter,omitempty"`
	TopK       int               `json:"topK,omitempty"`
	Thresholds *SearchThresholds `json:"thresholds,omitempty"`
}
type ItemSearchHit struct {
	Type            string  `json:"type"`
	ItemID          string  `json:"itemId"`
	Scope           Scope   `json:"scope"`
	Path            string  `json:"path"`
	Summary         string  `json:"summary"`
	LatestVersionID string  `json:"latestVersionId"`
	LatestSeq       int64   `json:"latestSeq"`
	Score           float64 `json:"score"`
	Similarity      float64 `json:"similarity"`
}
type SearchItemsResponse struct {
	Type            string          `json:"type"`
	MemoryStoreName string          `json:"memoryStoreName"`
	Scope           Scope           `json:"scope"`
	Items           []ItemSearchHit `json:"items"`
}

// DeleteItemVersionsRequest asynchronously deletes historical versions in a
// wildcard scope. The service requires that no current items remain there.
type DeleteItemVersionsRequest struct{ ItemScopedRequest }
type DeleteItemVersionsResponse struct {
	Type   string `json:"type"`
	TaskID string `json:"taskId"`
	Status string `json:"status"`
}

// MarshalJSON preserves text creation (including empty files) while omitting
// content entirely for a binary upload.
func (r AddItemRequest) MarshalJSON() ([]byte, error) {
	type plain AddItemRequest
	content := &r.Content
	if r.ContentBase64 != nil {
		if r.Content != "" {
			return nil, newClientError("content and contentBase64 are mutually exclusive")
		}
		content = nil
	}
	return json.Marshal(struct {
		plain
		Content *string `json:"content,omitempty"`
	}{plain: plain(r), Content: content})
}

// MarshalJSON never emits the removed newPath/expectedSha256 wire fields.
// Use Client.UpdateItem to resolve legacy path addressing and SHA guards.
func (r UpdateItemRequest) MarshalJSON() ([]byte, error) {
	if r.ExpectedSHA256 != "" {
		return nil, newClientError("ExpectedSHA256 requires Client.UpdateItem; use Precondition for raw requests")
	}
	if r.ItemID == "" && r.Path != nil {
		return nil, newClientError("path addressing requires Client.UpdateItem; use ItemID for raw requests")
	}
	if r.Content != nil && r.ContentBase64 != nil {
		return nil, newClientError("content and contentBase64 are mutually exclusive")
	}
	type plain UpdateItemRequest
	if r.NewPath != "" {
		if r.Path != nil && *r.Path != r.NewPath {
			return nil, newClientError("Path and NewPath specify different rename targets")
		}
		r.Path = String(r.NewPath)
	}
	return json.Marshal(plain(r))
}

func (r DeleteItemRequest) MarshalJSON() ([]byte, error) {
	if r.ExpectedSHA256 != "" {
		return nil, newClientError("ExpectedSHA256 requires Client.DeleteItem; use Precondition for raw requests")
	}
	type plain DeleteItemRequest
	return json.Marshal(plain(r))
}

func (r ListItemVersionsRequest) MarshalJSON() ([]byte, error) {
	if r.ApiKeyID != "" {
		return nil, newClientError("apiKeyId is no longer supported by ListItemVersions")
	}
	type plain ListItemVersionsRequest
	return json.Marshal(plain(r))
}
