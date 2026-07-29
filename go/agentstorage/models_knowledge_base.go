package agentstorage

import "encoding/json"

// MetadataFieldType enumerates the supported knowledge-base metadata field types.
type MetadataFieldType string

const (
	MetadataFieldTypeString      MetadataFieldType = "string"
	MetadataFieldTypeLong        MetadataFieldType = "long"
	MetadataFieldTypeDouble      MetadataFieldType = "double"
	MetadataFieldTypeBoolean     MetadataFieldType = "boolean"
	MetadataFieldTypeDate        MetadataFieldType = "date"
	MetadataFieldTypeStringList  MetadataFieldType = "string_list"
	MetadataFieldTypeLongList    MetadataFieldType = "long_list"
	MetadataFieldTypeDoubleList  MetadataFieldType = "double_list"
	MetadataFieldTypeBooleanList MetadataFieldType = "boolean_list"
	MetadataFieldTypeDateList    MetadataFieldType = "date_list"
)

// MetadataField declares one metadata field of a knowledge base.
type MetadataField struct {
	Name string            `json:"name"`
	Type MetadataFieldType `json:"type"`
}

// EmbeddingConfiguration configures the embedding model used by a knowledge base.
type EmbeddingConfiguration struct {
	Provider  string `json:"provider,omitempty"`
	URL       string `json:"url,omitempty"`
	APIKey    string `json:"apiKey,omitempty"`
	Model     string `json:"model,omitempty"`
	Dimension int    `json:"dimension,omitempty"`
}

// CreateKnowledgeBaseRequest is the request for CreateKnowledgeBase.
type CreateKnowledgeBaseRequest struct {
	KnowledgeBaseName      string                  `json:"knowledgeBaseName"`
	Description            string                  `json:"description,omitempty"`
	Subspace               *bool                   `json:"subspace,omitempty"`
	Tags                   []string                `json:"tags,omitempty"`
	Metadata               []MetadataField         `json:"metadata,omitempty"`
	RetrievalConfiguration *RetrievalConfiguration `json:"retrievalConfiguration,omitempty"`
	EmbeddingConfiguration *EmbeddingConfiguration `json:"embeddingConfiguration,omitempty"`
}

// DeleteKnowledgeBaseRequest is the request for DeleteKnowledgeBase.
type DeleteKnowledgeBaseRequest struct {
	KnowledgeBaseName string `json:"knowledgeBaseName"`
}

// DescribeKnowledgeBaseRequest is the request for DescribeKnowledgeBase.
type DescribeKnowledgeBaseRequest struct {
	KnowledgeBaseName string `json:"knowledgeBaseName"`
}

// ListKnowledgeBaseRequest is the request for ListKnowledgeBase.
type ListKnowledgeBaseRequest struct {
	MaxResults int    `json:"maxResults,omitempty"`
	NextToken  string `json:"nextToken,omitempty"`
}

// UpdateKnowledgeBaseRequest is the request for UpdateKnowledgeBase.
//
// Description and Tags follow tri-state semantics: leave them nil to keep the
// current value, use Null[...]() to clear, or NullableOf(...) to replace.
type UpdateKnowledgeBaseRequest struct {
	KnowledgeBaseName      string                  `json:"knowledgeBaseName"`
	Description            *Nullable[string]       `json:"description,omitempty"`
	Tags                   *Nullable[[]string]     `json:"tags,omitempty"`
	RetrievalConfiguration *RetrievalConfiguration `json:"retrievalConfiguration,omitempty"`
}

// APIResponse is the generic {code, data, message, requestId} envelope the
// knowledge-base APIs answer with.
type APIResponse[T any] struct {
	Code      string `json:"code"`
	Data      T      `json:"data"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

// EmptyData is the data payload of responses that carry no content.
type EmptyData = map[string]any

// KnowledgeBaseDetail describes one knowledge base as returned by
// DescribeKnowledgeBase / ListKnowledgeBase.
type KnowledgeBaseDetail struct {
	KnowledgeBaseName      string          `json:"knowledgeBaseName"`
	Description            string          `json:"description,omitempty"`
	Subspace               bool            `json:"subspace,omitempty"`
	Tags                   []string        `json:"tags,omitempty"`
	Metadata               []MetadataField `json:"metadata,omitempty"`
	RetrievalConfiguration json.RawMessage `json:"retrievalConfiguration,omitempty"`
	EmbeddingConfiguration json.RawMessage `json:"embeddingConfiguration,omitempty"`
	CreatedAt              int64           `json:"createdAt,omitempty"`
	UpdatedAt              int64           `json:"updatedAt,omitempty"`
}

// KnowledgeBaseListData is the data payload of ListKnowledgeBase.
type KnowledgeBaseListData struct {
	KnowledgeBases []KnowledgeBaseDetail `json:"knowledgeBases"`
	NextToken      string                `json:"nextToken,omitempty"`
}

type (
	CreateKnowledgeBaseResponse   = APIResponse[EmptyData]
	DeleteKnowledgeBaseResponse   = APIResponse[EmptyData]
	DescribeKnowledgeBaseResponse = APIResponse[KnowledgeBaseDetail]
	ListKnowledgeBaseResponse     = APIResponse[KnowledgeBaseListData]
	UpdateKnowledgeBaseResponse   = APIResponse[EmptyData]
)
