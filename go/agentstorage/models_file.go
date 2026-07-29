package agentstorage

// Memory store modes that expose the item APIs. FileOTS is a read-only
// projection of structured memories; FileMemory stores files as source data.
const (
	StorageModeFileOTS    = "file+ots"
	StorageModeFileMemory = "filemem"
)

// ItemTypeMemoryFile is the only item type currently accepted by the item
// APIs. The SDK fills it into any item request whose Type field is empty.
const ItemTypeMemoryFile = "memoryfile"

// ItemOperation is the operation recorded by an item version.
type ItemOperation string

const (
	ItemOperationCreated  ItemOperation = "created"
	ItemOperationModified ItemOperation = "modified"
	ItemOperationDeleted  ItemOperation = "deleted"
)

// ItemScopedRequest carries the item type, store and exact four-part scope
// shared by all item actions.
type ItemScopedRequest struct {
	// Type discriminates the item type. Leave empty to let the SDK send
	// ItemTypeMemoryFile.
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

// AddItemRequest is the request for AddItem.
type AddItemRequest struct {
	ItemScopedRequest
	Path      string `json:"path"`
	Content   string `json:"content"`
	SessionID string `json:"sessionId,omitempty"`
}

// ListItemsRequest is the request for ListItems.
type ListItemsRequest struct {
	ItemScopedRequest
	PathPrefix string `json:"pathPrefix,omitempty"`
	NextToken  string `json:"nextToken,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// GetItemRequest is the request for GetItem. IncludeContent defaults to true
// when nil; set it to false (agentstorage.Bool(false)) to fetch metadata
// only.
type GetItemRequest struct {
	ItemScopedRequest
	Path           string `json:"path"`
	IncludeContent *bool  `json:"includeContent,omitempty"`
}

// UpdateItemRequest is the request for UpdateItem. Exactly one of Content
// (rewrite the content) and NewPath (rename) must be set. Overwrite applies
// to the rename branch; ExpectedSHA256 guards the source row on both
// branches.
type UpdateItemRequest struct {
	ItemScopedRequest
	Path           string  `json:"path"`
	Content        *string `json:"content,omitempty"`
	NewPath        string  `json:"newPath,omitempty"`
	Overwrite      bool    `json:"overwrite,omitempty"`
	ExpectedSHA256 string  `json:"expectedSha256,omitempty"`
	SessionID      string  `json:"sessionId,omitempty"`
}

// DeleteItemRequest is the request for DeleteItem.
type DeleteItemRequest struct {
	ItemScopedRequest
	Path           string `json:"path"`
	ExpectedSHA256 string `json:"expectedSha256,omitempty"`
	SessionID      string `json:"sessionId,omitempty"`
}

// ListItemVersionsRequest is the request for ListItemVersions.
type ListItemVersionsRequest struct {
	ItemScopedRequest
	ItemID    string        `json:"itemId"`
	Path      string        `json:"path,omitempty"`
	NextToken string        `json:"nextToken,omitempty"`
	Limit     int           `json:"limit,omitempty"`
	Operation ItemOperation `json:"operation,omitempty"`
}

// GetItemVersionRequest is the request for GetItemVersion.
type GetItemVersionRequest struct {
	ItemScopedRequest
	ItemID     string `json:"itemId"`
	VersionID  string `json:"versionId"`
	VersionSeq int64  `json:"versionSeq"`
}

// RedactItemVersionRequest is the request for RedactItemVersion.
type RedactItemVersionRequest struct {
	ItemScopedRequest
	ItemID     string `json:"itemId"`
	VersionID  string `json:"versionId"`
	VersionSeq int64  `json:"versionSeq"`
	SessionID  string `json:"sessionId,omitempty"`
}

// Item is the current state of one item. Content is nil when the response
// omits it (ListItems, or GetItem with IncludeContent=false); an empty file
// has Content pointing at "". Type echoes ItemTypeMemoryFile in single-item
// responses.
type Item struct {
	Type             string  `json:"type,omitempty"`
	ItemID           string  `json:"itemId"`
	Path             string  `json:"path"`
	Content          *string `json:"content,omitempty"`
	ContentSHA256    string  `json:"contentSha256"`
	ContentSizeBytes int64   `json:"contentSizeBytes"`
	LatestSeq        int64   `json:"latestSeq"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

// ItemVersion is one immutable item version. Content is nil when the response
// omits it (ListItemVersions, or redacted versions); an empty version has
// Content pointing at "". The redaction fields appear only after
// RedactItemVersion.
type ItemVersion struct {
	Type             string        `json:"type,omitempty"`
	VersionID        string        `json:"versionId"`
	ItemID           string        `json:"itemId"`
	VersionSeq       int64         `json:"versionSeq"`
	Operation        ItemOperation `json:"operation"`
	Path             string        `json:"path"`
	Content          *string       `json:"content,omitempty"`
	ContentSHA256    string        `json:"contentSha256"`
	ContentSizeBytes int64         `json:"contentSizeBytes"`
	SessionID        string        `json:"sessionId,omitempty"`
	CreatedAt        string        `json:"createdAt"`
	Redacted         bool          `json:"redacted,omitempty"`
	RedactedAt       string        `json:"redactedAt,omitempty"`
	RedactedBy       string        `json:"redactedBy,omitempty"`
}

type (
	AddItemResponse           = Item
	GetItemResponse           = Item
	UpdateItemResponse        = Item
	GetItemVersionResponse    = ItemVersion
	RedactItemVersionResponse = ItemVersion
)

// DeleteItemResponse is the response of DeleteItem; it carries only the
// echoed item type.
type DeleteItemResponse struct {
	Type string `json:"type"`
}

// ListItemsResponse is the response of ListItems. ReadOnly is true for
// file+ots projections and omitted for writable stores.
type ListItemsResponse struct {
	Type      string `json:"type"`
	Items     []Item `json:"items"`
	NextToken string `json:"nextToken,omitempty"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// ListItemVersionsResponse is the response of ListItemVersions.
type ListItemVersionsResponse struct {
	Type      string        `json:"type"`
	Versions  []ItemVersion `json:"versions"`
	NextToken string        `json:"nextToken,omitempty"`
}
