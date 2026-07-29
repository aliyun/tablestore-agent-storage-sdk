package agentstorage

// AddDocumentItem is one document entry of AddDocumentsRequest.
type AddDocumentItem struct {
	OssKey           string         `json:"ossKey"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	InclusionFilters []string       `json:"inclusionFilters,omitempty"`
	ExclusionFilters []string       `json:"exclusionFilters,omitempty"`
}

// AddDocumentsRequest is the request for AddDocuments.
type AddDocumentsRequest struct {
	KnowledgeBaseName string            `json:"knowledgeBaseName"`
	Documents         []AddDocumentItem `json:"documents"`
	Subspace          string            `json:"subspace,omitempty"`
}

// UploadDocumentItem is one local-file entry of UploadDocumentsRequest.
type UploadDocumentItem struct {
	FilePath string         `json:"filePath"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// UploadDocumentsRequest is the request for UploadDocuments: each FilePath is
// uploaded to OSS first, then registered via AddDocuments.
type UploadDocumentsRequest struct {
	KnowledgeBaseName string               `json:"knowledgeBaseName"`
	Documents         []UploadDocumentItem `json:"documents"`
	Subspace          string               `json:"subspace,omitempty"`
}

// DeleteDocumentItem identifies one document of DeleteDocumentsRequest by
// docId or ossKey.
type DeleteDocumentItem struct {
	DocID  string `json:"docId,omitempty"`
	OssKey string `json:"ossKey,omitempty"`
}

// DeleteDocumentsRequest is the request for DeleteDocuments.
type DeleteDocumentsRequest struct {
	KnowledgeBaseName string               `json:"knowledgeBaseName"`
	Documents         []DeleteDocumentItem `json:"documents"`
	Subspace          string               `json:"subspace,omitempty"`
}

// GetDocumentRequest is the request for GetDocument.
type GetDocumentRequest struct {
	KnowledgeBaseName string `json:"knowledgeBaseName"`
	DocID             string `json:"docId,omitempty"`
	OssKey            string `json:"ossKey,omitempty"`
	Subspace          string `json:"subspace,omitempty"`
}

// ListDocumentsRequest is the request for ListDocuments.
type ListDocumentsRequest struct {
	KnowledgeBaseName string   `json:"knowledgeBaseName"`
	Subspace          []string `json:"subspace,omitempty"`
	MaxResults        int      `json:"maxResults,omitempty"`
	NextToken         string   `json:"nextToken,omitempty"`
}

// UpdateDocumentRequest is the request for UpdateDocument. Metadata follows
// tri-state semantics (nil = keep, Null = clear, NullableOf = replace).
type UpdateDocumentRequest struct {
	KnowledgeBaseName string                    `json:"knowledgeBaseName"`
	Subspace          string                    `json:"subspace,omitempty"`
	OssKey            string                    `json:"ossKey,omitempty"`
	DocID             string                    `json:"docId,omitempty"`
	Metadata          *Nullable[map[string]any] `json:"metadata,omitempty"`
}

// DocumentOperationDetail reports the per-document outcome of
// AddDocuments / UploadDocuments / DeleteDocuments.
type DocumentOperationDetail struct {
	DocID   string `json:"docId,omitempty"`
	Status  string `json:"status,omitempty"`
	OssKey  string `json:"ossKey,omitempty"`
	Message string `json:"message,omitempty"`
}

// DocumentDetailsData is the data payload of AddDocuments / DeleteDocuments.
type DocumentDetailsData struct {
	DocumentDetails []DocumentOperationDetail `json:"documentDetails"`
}

// DocumentInfo describes one document as returned by GetDocument / ListDocuments.
type DocumentInfo struct {
	DocID     string         `json:"docId"`
	OssKey    string         `json:"ossKey,omitempty"`
	Subspace  string         `json:"subspace,omitempty"`
	ChunkNum  int64          `json:"chunkNum,omitempty"`
	Status    string         `json:"status,omitempty"`
	ETag      string         `json:"eTag,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt int64          `json:"createdAt,omitempty"`
	UpdatedAt int64          `json:"updatedAt,omitempty"`
}

// DocumentListData is the data payload of ListDocuments.
type DocumentListData struct {
	Documents []DocumentInfo `json:"documents"`
	NextToken string         `json:"nextToken,omitempty"`
}

type (
	AddDocumentsResponse    = APIResponse[DocumentDetailsData]
	DeleteDocumentsResponse = APIResponse[DocumentDetailsData]
	GetDocumentResponse     = APIResponse[[]DocumentInfo]
	ListDocumentsResponse   = APIResponse[DocumentListData]
	UpdateDocumentResponse  = APIResponse[EmptyData]
)
