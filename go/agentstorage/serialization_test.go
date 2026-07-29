package agentstorage

import (
	"encoding/json"
	"reflect"
	"testing"
)

func marshalToMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertJSONEqual(t *testing.T, value any, wantJSON string) {
	t.Helper()
	var want map[string]any
	if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
		t.Fatal(err)
	}
	got := marshalToMap(t, value)
	if !reflect.DeepEqual(got, want) {
		gotData, _ := json.Marshal(got)
		t.Errorf("serialized payload mismatch:\n got: %s\nwant: %s", gotData, wantJSON)
	}
}

func TestUpdateKnowledgeBaseUnsetAndNull(t *testing.T) {
	// nil (unset) fields are omitted, Null fields serialize as JSON null.
	assertJSONEqual(t, &UpdateKnowledgeBaseRequest{
		KnowledgeBaseName: "kb",
		Description:       Null[string](),
	}, `{"knowledgeBaseName":"kb","description":null}`)

	assertJSONEqual(t, &UpdateKnowledgeBaseRequest{
		KnowledgeBaseName: "kb",
		Description:       NullableOf("new"),
		Tags:              NullableOf([]string{"tag"}),
	}, `{"knowledgeBaseName":"kb","description":"new","tags":["tag"]}`)
}

func TestRetrieveRequestSerialization(t *testing.T) {
	filter, err := NewMetadataFilterBuilder().
		Equals("author", "aliyun").
		GreaterThan("rank", 1).
		BuildAnd()
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, &RetrieveRequest{
		KnowledgeBaseName: "kb",
		RetrievalQuery:    RetrievalQuery{Text: "hello"},
		RetrievalConfiguration: &RetrievalConfiguration{
			SearchType: []SearchType{SearchTypeDenseVector, SearchTypeFullText},
			Filter:     filter,
		},
	}, `{
		"knowledgeBaseName": "kb",
		"retrievalQuery": {"text": "hello", "type": "TEXT"},
		"retrievalConfiguration": {
			"searchType": ["DENSE_VECTOR", "FULL_TEXT"],
			"filter": {
				"andAll": [
					{"equals": {"key": "author", "value": "aliyun"}},
					{"greaterThan": {"key": "rank", "value": 1}}
				]
			}
		}
	}`)
}

func TestRetrieveConfigurationDefaults(t *testing.T) {
	assertJSONEqual(t, &RetrievalConfiguration{
		DenseVectorSearchConfiguration: &DenseVectorSearchConfiguration{},
		FullTextSearchConfiguration:    &FullTextSearchConfiguration{NumberOfResults: Int(5)},
		RerankingConfiguration: &RerankingConfiguration{
			RRFConfiguration: &RRFConfiguration{},
		},
	}, `{
		"denseVectorSearchConfiguration": {"numberOfResults": 20},
		"fullTextSearchConfiguration": {"numberOfResults": 5},
		"rerankingConfiguration": {
			"type": "RRF",
			"numberOfResults": 20,
			"rrfConfiguration": {"denseVectorSearchWeight": 1, "fullTextSearchWeight": 1, "k": 60}
		}
	}`)
}

func TestDocumentAndChunkUpdates(t *testing.T) {
	assertJSONEqual(t, &UpdateDocumentRequest{
		KnowledgeBaseName: "kb",
		DocID:             "doc-1",
		Metadata:          Null[map[string]any](),
	}, `{"knowledgeBaseName":"kb","docId":"doc-1","metadata":null}`)

	assertJSONEqual(t, &UpdateChunksRequest{
		KnowledgeBaseName: "kb",
		Chunks: []UpdateChunkItem{
			{ChunkID: 1, DocID: "doc-1", Content: Null[string]()},
		},
	}, `{"knowledgeBaseName":"kb","chunks":[{"chunkId":1,"docId":"doc-1","content":null}]}`)
}

func TestCreateKnowledgeBaseSerialization(t *testing.T) {
	assertJSONEqual(t, &CreateKnowledgeBaseRequest{
		KnowledgeBaseName: "kb",
		Description:       "desc",
		Subspace:          Bool(true),
		Tags:              []string{"tag"},
		Metadata:          []MetadataField{{Name: "author", Type: MetadataFieldTypeString}},
		EmbeddingConfiguration: &EmbeddingConfiguration{
			Provider: "p", Model: "m", Dimension: 128,
		},
	}, `{
		"knowledgeBaseName": "kb",
		"description": "desc",
		"subspace": true,
		"tags": ["tag"],
		"metadata": [{"name": "author", "type": "string"}],
		"embeddingConfiguration": {"provider": "p", "model": "m", "dimension": 128}
	}`)
}

func TestMemoryRequestSerialization(t *testing.T) {
	assertJSONEqual(t, &AddMemoriesRequest{
		MemoryStoreName: "store",
		Scope:           &Scope{AppID: "app", TenantID: "tenant"},
		Messages: []MessagePayload{
			{Role: "user", Content: "hi"},
		},
		Metadata: map[string]string{"source": "chat"},
		Sync:     true,
	}, `{
		"memoryStoreName": "store",
		"scope": {"appId": "app", "tenantId": "tenant"},
		"messages": [{"role": "user", "content": "hi"}],
		"metadata": {"source": "chat"},
		"sync": true
	}`)

	// EnableRerank=false must be serialized (tri-state via pointer).
	assertJSONEqual(t, &SearchMemoriesRequest{
		MemoryStoreName: "store",
		Query:           "coffee",
		Scope:           &Scope{AppID: "app", TenantID: "tenant", AgentID: "*", RunID: "*"},
		TopK:            5,
		EnableRerank:    Bool(false),
		IncludeEvidence: true,
	}, `{
		"memoryStoreName": "store",
		"query": "coffee",
		"scope": {"appId": "app", "tenantId": "tenant", "agentId": "*", "runId": "*"},
		"topK": 5,
		"enableRerank": false,
		"includeEvidence": true
	}`)

	// PATCH semantics: pointer to empty string clears, nil omits.
	assertJSONEqual(t, &UpdateMemoryStoreRequest{
		MemoryStoreName: "store",
		Description:     String(""),
	}, `{"memoryStoreName":"store","description":""}`)

	// Timestamps accept RFC3339 strings or unix milliseconds.
	assertJSONEqual(t, &ListMemoryStoreMessagesRequest{
		MemoryStoreName: "store",
		Scope:           Scope{AppID: "a", TenantID: "t", AgentID: "ag", RunID: "r"},
		MinTimestamp:    "2026-01-01T00:00:00Z",
	}, `{
		"memoryStoreName": "store",
		"scope": {"appId": "a", "tenantId": "t", "agentId": "ag", "runId": "r"},
		"minTimestamp": "2026-01-01T00:00:00Z"
	}`)

	assertJSONEqual(t, &ListMemoryTasksRequest{
		MemoryStoreName: "store",
		Scope:           Scope{AppID: "a"},
		MinTimestamp:    int64(1750000000000),
	}, `{
		"memoryStoreName": "store",
		"scope": {"appId": "a"},
		"minTimestamp": 1750000000000
	}`)
}

func TestDreamRequestSerialization(t *testing.T) {
	assertJSONEqual(t, &CreateMemoryDreamTaskRequest{
		MemoryStoreName: "store",
		Scopes:          []Scope{{AppID: "app", TenantID: "tenant"}},
		TaskType:        DreamTaskTypeMemory,
		ApplyMode:       DreamApplyModeProposal,
		Incremental:     Bool(true),
	}, `{
		"memoryStoreName": "store",
		"scopes": [{"appId": "app", "tenantId": "tenant"}],
		"taskType": "memory",
		"applyMode": "proposal",
		"incremental": true
	}`)
}

func TestMemoryResponseParsing(t *testing.T) {
	raw := `{
		"results": [{
			"unit": {
				"id": "u1",
				"conversation_key": "ck",
				"scope": {"appId": "app"},
				"unit_type": "atomic_fact",
				"text": "likes coffee",
				"created_at": "2026-06-17T07:19:59.962Z",
				"salience": 0.5,
				"version": 1
			},
			"score": 0.9,
			"similarity": 0.8,
			"source": "vector"
		}],
		"scope": {"appId": "app"},
		"memoryStoreName": "store"
	}`
	var resp SearchMemoriesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results length = %d, want 1", len(resp.Results))
	}
	hit := resp.Results[0]
	if hit.Unit.ID != "u1" || hit.Unit.UnitType != "atomic_fact" || hit.Unit.Text != "likes coffee" {
		t.Errorf("unexpected unit: %+v", hit.Unit)
	}
	if hit.Score != 0.9 || hit.Similarity != 0.8 || hit.Source != "vector" {
		t.Errorf("unexpected hit: %+v", hit)
	}
}

func TestKnowledgeBaseEnvelopeParsing(t *testing.T) {
	raw := `{
		"code": "SUCCESS",
		"data": {
			"documents": [{"docId": "d1", "ossKey": "oss://b/k", "chunkNum": 808, "eTag": "73940"}],
			"nextToken": "token"
		},
		"message": "succeed",
		"requestId": "rid"
	}`
	var resp ListDocumentsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != "SUCCESS" || resp.RequestID != "rid" {
		t.Errorf("unexpected envelope: %+v", resp)
	}
	if len(resp.Data.Documents) != 1 || resp.Data.Documents[0].ChunkNum != 808 || resp.Data.Documents[0].ETag != "73940" {
		t.Errorf("unexpected documents: %+v", resp.Data)
	}
	if resp.Data.NextToken != "token" {
		t.Errorf("nextToken = %q", resp.Data.NextToken)
	}
}

func TestMetadataFilterOperators(t *testing.T) {
	cases := []struct {
		filter *MetadataFilter
		want   string
	}{
		{Equals("k", "v"), `{"equals":{"key":"k","value":"v"}}`},
		{NotEquals("k", true), `{"notEquals":{"key":"k","value":true}}`},
		{LessThanOrEquals("k", 3.5), `{"lessThanOrEquals":{"key":"k","value":3.5}}`},
		{InList("k", []string{"a", "b"}), `{"in":{"key":"k","value":["a","b"]}}`},
		{NotInList("k", []string{"a"}), `{"notIn":{"key":"k","value":["a"]}}`},
		{StartsWith("k", "p"), `{"startsWith":{"key":"k","value":"p"}}`},
		{StringContains("k", "s"), `{"stringContains":{"key":"k","value":"s"}}`},
		{ListContains("k", "s"), `{"listContains":{"key":"k","value":"s"}}`},
		{OrAll(Equals("a", 1), Equals("b", 2)), `{"orAll":[{"equals":{"key":"a","value":1}},{"equals":{"key":"b","value":2}}]}`},
		{NotAll(Equals("a", 1), Equals("b", 2)), `{"notAll":[{"equals":{"key":"a","value":1}},{"equals":{"key":"b","value":2}}]}`},
	}
	for _, c := range cases {
		assertJSONEqual(t, c.filter, c.want)
	}

	// A single-condition builder returns the condition itself.
	single, err := NewMetadataFilterBuilder().Equals("k", "v").BuildOr()
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, single, `{"equals":{"key":"k","value":"v"}}`)

	if _, err := NewMetadataFilterBuilder().BuildAnd(); err == nil {
		t.Error("empty builder should fail")
	}
}
