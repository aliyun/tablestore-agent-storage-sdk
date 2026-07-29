package agentstorage

import "encoding/json"

// SearchType enumerates retrieval search modes.
type SearchType string

const (
	SearchTypeDenseVector SearchType = "DENSE_VECTOR"
	SearchTypeFullText    SearchType = "FULL_TEXT"
)

// RerankingType enumerates reranking strategies.
type RerankingType string

const (
	RerankingTypeRRF    RerankingType = "RRF"
	RerankingTypeModel  RerankingType = "MODEL"
	RerankingTypeWeight RerankingType = "WEIGHT"
)

// RetrievalQueryType enumerates retrieval query types.
type RetrievalQueryType string

// RetrievalQueryTypeText is the only supported query type today.
const RetrievalQueryTypeText RetrievalQueryType = "TEXT"

// RetrievalQuery is the query part of RetrieveRequest. An empty Type
// serializes as "TEXT".
type RetrievalQuery struct {
	Text string             `json:"text"`
	Type RetrievalQueryType `json:"type,omitempty"`
}

// MarshalJSON fills in the default query type, matching the TS/Python SDKs.
func (q RetrievalQuery) MarshalJSON() ([]byte, error) {
	queryType := q.Type
	if queryType == "" {
		queryType = RetrievalQueryTypeText
	}
	return json.Marshal(struct {
		Text string             `json:"text"`
		Type RetrievalQueryType `json:"type"`
	}{q.Text, queryType})
}

// DenseVectorSearchConfiguration configures the vector search leg.
// A nil NumberOfResults serializes as the service default of 20.
type DenseVectorSearchConfiguration struct {
	NumberOfResults *int `json:"numberOfResults,omitempty"`
}

// MarshalJSON fills in the default result count, matching the TS SDK.
func (c DenseVectorSearchConfiguration) MarshalJSON() ([]byte, error) {
	return marshalNumberOfResults(c.NumberOfResults)
}

// FullTextSearchConfiguration configures the full-text search leg.
// A nil NumberOfResults serializes as the service default of 20.
type FullTextSearchConfiguration struct {
	NumberOfResults *int `json:"numberOfResults,omitempty"`
}

// MarshalJSON fills in the default result count, matching the TS SDK.
func (c FullTextSearchConfiguration) MarshalJSON() ([]byte, error) {
	return marshalNumberOfResults(c.NumberOfResults)
}

func marshalNumberOfResults(n *int) ([]byte, error) {
	count := 20
	if n != nil {
		count = *n
	}
	return json.Marshal(struct {
		NumberOfResults int `json:"numberOfResults"`
	}{count})
}

// RRFConfiguration configures reciprocal-rank-fusion reranking. Nil fields
// serialize as the defaults (weights 1, k 60).
type RRFConfiguration struct {
	DenseVectorSearchWeight *float64 `json:"denseVectorSearchWeight,omitempty"`
	FullTextSearchWeight    *float64 `json:"fullTextSearchWeight,omitempty"`
	K                       *int     `json:"k,omitempty"`
}

// MarshalJSON fills in the RRF defaults, matching the TS SDK.
func (c RRFConfiguration) MarshalJSON() ([]byte, error) {
	dense, text, k := 1.0, 1.0, 60
	if c.DenseVectorSearchWeight != nil {
		dense = *c.DenseVectorSearchWeight
	}
	if c.FullTextSearchWeight != nil {
		text = *c.FullTextSearchWeight
	}
	if c.K != nil {
		k = *c.K
	}
	return json.Marshal(struct {
		DenseVectorSearchWeight float64 `json:"denseVectorSearchWeight"`
		FullTextSearchWeight    float64 `json:"fullTextSearchWeight"`
		K                       int     `json:"k"`
	}{dense, text, k})
}

// ModelConfiguration configures model-based reranking.
type ModelConfiguration struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

// WeightConfiguration configures weight-based reranking.
type WeightConfiguration struct {
	DenseVectorSearchWeight *float64 `json:"denseVectorSearchWeight,omitempty"`
	FullTextSearchWeight    *float64 `json:"fullTextSearchWeight,omitempty"`
}

// RerankingConfiguration configures result reranking. An empty Type
// serializes as "RRF" and a nil NumberOfResults as 20.
type RerankingConfiguration struct {
	Type                RerankingType        `json:"type,omitempty"`
	NumberOfResults     *int                 `json:"numberOfResults,omitempty"`
	RRFConfiguration    *RRFConfiguration    `json:"rrfConfiguration,omitempty"`
	ModelConfiguration  *ModelConfiguration  `json:"modelConfiguration,omitempty"`
	WeightConfiguration *WeightConfiguration `json:"weightConfiguration,omitempty"`
}

// MarshalJSON fills in the reranking defaults, matching the TS SDK.
func (c RerankingConfiguration) MarshalJSON() ([]byte, error) {
	rerankType := c.Type
	if rerankType == "" {
		rerankType = RerankingTypeRRF
	}
	count := 20
	if c.NumberOfResults != nil {
		count = *c.NumberOfResults
	}
	return json.Marshal(struct {
		Type                RerankingType        `json:"type"`
		NumberOfResults     int                  `json:"numberOfResults"`
		RRFConfiguration    *RRFConfiguration    `json:"rrfConfiguration,omitempty"`
		ModelConfiguration  *ModelConfiguration  `json:"modelConfiguration,omitempty"`
		WeightConfiguration *WeightConfiguration `json:"weightConfiguration,omitempty"`
	}{rerankType, count, c.RRFConfiguration, c.ModelConfiguration, c.WeightConfiguration})
}

// RetrievalConfiguration bundles the retrieval tuning options.
type RetrievalConfiguration struct {
	SearchType                     []SearchType                    `json:"searchType,omitempty"`
	DenseVectorSearchConfiguration *DenseVectorSearchConfiguration `json:"denseVectorSearchConfiguration,omitempty"`
	FullTextSearchConfiguration    *FullTextSearchConfiguration    `json:"fullTextSearchConfiguration,omitempty"`
	RerankingConfiguration         *RerankingConfiguration         `json:"rerankingConfiguration,omitempty"`
	Filter                         *MetadataFilter                 `json:"filter,omitempty"`
}

// RetrieveRequest is the request for Retrieve.
type RetrieveRequest struct {
	KnowledgeBaseName      string                  `json:"knowledgeBaseName"`
	RetrievalQuery         RetrievalQuery          `json:"retrievalQuery"`
	Subspace               []string                `json:"subspace,omitempty"`
	RetrievalConfiguration *RetrievalConfiguration `json:"retrievalConfiguration,omitempty"`
}

// RetrievalResult is one hit returned by Retrieve.
type RetrievalResult struct {
	OssKey   string         `json:"ossKey,omitempty"`
	DocID    string         `json:"docId,omitempty"`
	ChunkID  int64          `json:"chunkId,omitempty"`
	Subspace string         `json:"subspace,omitempty"`
	Score    float64        `json:"score,omitempty"`
	Content  string         `json:"content,omitempty"`
	Title    string         `json:"title,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// RetrievalResultsData is the data payload of Retrieve.
type RetrievalResultsData struct {
	RetrievalResults []RetrievalResult `json:"retrievalResults"`
}

// RetrieveResponse is the response of Retrieve.
type RetrieveResponse = APIResponse[RetrievalResultsData]
