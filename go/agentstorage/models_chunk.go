package agentstorage

// UpdateChunkItem is one chunk entry of UpdateChunksRequest. Title and
// Content follow tri-state semantics (nil = keep, Null = clear, NullableOf =
// replace).
type UpdateChunkItem struct {
	ChunkID int64             `json:"chunkId"`
	DocID   string            `json:"docId,omitempty"`
	OssKey  string            `json:"ossKey,omitempty"`
	Title   *Nullable[string] `json:"title,omitempty"`
	Content *Nullable[string] `json:"content,omitempty"`
	Status  string            `json:"status,omitempty"`
}

// UpdateChunksRequest is the request for UpdateChunks.
type UpdateChunksRequest struct {
	KnowledgeBaseName string            `json:"knowledgeBaseName"`
	Chunks            []UpdateChunkItem `json:"chunks"`
	Subspace          string            `json:"subspace,omitempty"`
}

// ListChunksRequest is the request for ListChunks.
type ListChunksRequest struct {
	KnowledgeBaseName string `json:"knowledgeBaseName"`
	Subspace          string `json:"subspace,omitempty"`
	DocID             string `json:"docId,omitempty"`
	OssKey            string `json:"ossKey,omitempty"`
	MaxResults        int    `json:"maxResults,omitempty"`
	NextToken         string `json:"nextToken,omitempty"`
}

// ChunkDetail describes one chunk as returned by ListChunks.
type ChunkDetail struct {
	DocID     string         `json:"docId,omitempty"`
	OssKey    string         `json:"ossKey,omitempty"`
	ChunkID   int64          `json:"chunkId"`
	Content   string         `json:"content,omitempty"`
	Title     string         `json:"title,omitempty"`
	ChunkType string         `json:"chunkType,omitempty"`
	Status    string         `json:"status,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt int64          `json:"createdAt,omitempty"`
	UpdatedAt int64          `json:"updatedAt,omitempty"`
}

// ChunkListData is the data payload of ListChunks.
type ChunkListData struct {
	ChunkDetails []ChunkDetail `json:"chunkDetails"`
	NextToken    string        `json:"nextToken,omitempty"`
}

// ChunkUpdateDetail reports the per-chunk outcome of UpdateChunks.
type ChunkUpdateDetail struct {
	DocID        string `json:"docId,omitempty"`
	OssKey       string `json:"ossKey,omitempty"`
	ChunkID      int64  `json:"chunkId"`
	UpdateStatus string `json:"updateStatus,omitempty"`
	Message      string `json:"message,omitempty"`
}

// ChunkUpdatesData is the data payload of UpdateChunks.
type ChunkUpdatesData struct {
	UpdateDetails []ChunkUpdateDetail `json:"updateDetails"`
}

type (
	ListChunksResponse   = APIResponse[ChunkListData]
	UpdateChunksResponse = APIResponse[ChunkUpdatesData]
)
