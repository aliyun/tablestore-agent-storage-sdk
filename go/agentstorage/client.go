package agentstorage

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config configures a Client.
//
// Authentication: provide either AccessKeyID/AccessKeySecret (request
// signing, default protocol v2) or APIKey (HTTPS only). When both are
// present the API key is used for Tablestore requests and the access keys
// for OSS.
type Config struct {
	// Endpoint is the Tablestore endpoint, e.g.
	// "https://instance.cn-hangzhou.ots.aliyuncs.com".
	Endpoint     string
	InstanceName string

	AccessKeyID     string
	AccessKeySecret string
	// STSToken is an optional STS session token, sent as x-ots-ststoken.
	STSToken string
	// APIKey authenticates via the x-ots-apikey header instead of request
	// signing. Requires an https:// endpoint. It does not apply to OSS.
	APIKey string
	// AllowInsecureAPIKey permits APIKey auth over a non-HTTPS endpoint.
	// Testing only: the key is sent in cleartext. Never enable in production.
	AllowInsecureAPIKey bool

	// SignVersion selects the signing protocol; default SignV2. SignV4
	// requires Region.
	SignVersion SignVersion
	// Region of the instance, e.g. "cn-hangzhou". Required for SignV4.
	Region string

	// HTTPClient overrides the default HTTP client (30s timeout).
	HTTPClient *http.Client
	// Timeout applies to the default HTTP client; ignored when HTTPClient is set.
	Timeout time.Duration
	// UserAgentSuffix is appended to the default User-Agent.
	UserAgentSuffix string
	// ExtraHeaders are attached to every request; x-ots-* keys take part in
	// the signature.
	ExtraHeaders map[string]string

	// OSSEndpoint and OSSBucketName enable UploadDocuments and the
	// delete-file option of DeleteDocumentsWithOptions.
	OSSEndpoint   string
	OSSBucketName string
	// OSSAccessKeyID/OSSAccessKeySecret override the Tablestore credentials
	// for OSS access.
	OSSAccessKeyID     string
	OSSAccessKeySecret string

	// OSSUploader overrides the OSS upload/delete implementation (tests).
	OSSUploader OSSUploader
	// FileExists overrides local file existence checks (tests).
	FileExists func(path string) bool
}

const defaultTimeout = 30 * time.Second

// Client is the Tablestore Agent Storage client, covering the knowledge-base,
// document, chunk, retrieval, memory, file and Wiki APIs.
type Client struct {
	ots          *otsJSONClient
	instanceName string
	bucketName   string
	uploader     OSSUploader
	fileExists   func(path string) bool
}

// NewClient validates cfg and builds a Client.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, newClientError("endpoint is required.")
	}
	if cfg.InstanceName == "" {
		return nil, newClientError("instanceName is required.")
	}
	hasCredentials := cfg.AccessKeyID != "" && cfg.AccessKeySecret != ""
	if !hasCredentials && cfg.APIKey == "" {
		return nil, newClientError("Either credentials (accessKeyId/accessKeySecret) or apiKey must be provided.")
	}
	if cfg.APIKey != "" && !strings.HasPrefix(cfg.Endpoint, "https://") {
		if !cfg.AllowInsecureAPIKey {
			return nil, newClientError("API key authentication requires HTTPS. Endpoint must start with https://.")
		}
		log.Printf("[agentstorage] WARNING: sending the API key over non-HTTPS endpoint %q in cleartext (AllowInsecureAPIKey is for testing only)", cfg.Endpoint)
	}

	var requestSigner signer
	if cfg.APIKey == "" {
		switch cfg.SignVersion {
		case "", SignV2:
			requestSigner = signerV2{}
		case SignV4:
			if cfg.Region == "" {
				return nil, newClientError("region is required for v4 signing.")
			}
			requestSigner = &signerV4{
				region: cfg.Region,
				now: func() string {
					return time.Now().UTC().Format("20060102")
				},
			}
		default:
			return nil, newClientError("unsupported sign version: %s", cfg.SignVersion)
		}
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = defaultTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	userAgent := DefaultUserAgent
	if cfg.UserAgentSuffix != "" {
		userAgent += " " + cfg.UserAgentSuffix
	}

	client := &Client{
		ots: &otsJSONClient{
			endpoint:        strings.TrimRight(cfg.Endpoint, "/"),
			instanceName:    cfg.InstanceName,
			accessKeyID:     cfg.AccessKeyID,
			accessKeySecret: cfg.AccessKeySecret,
			securityToken:   cfg.STSToken,
			apiKey:          cfg.APIKey,
			signer:          requestSigner,
			httpClient:      httpClient,
			userAgent:       userAgent,
			extraHeaders:    cfg.ExtraHeaders,
		},
		instanceName: cfg.InstanceName,
		bucketName:   cfg.OSSBucketName,
		uploader:     cfg.OSSUploader,
		fileExists:   cfg.FileExists,
	}

	if client.uploader == nil {
		ossKeyID := cfg.OSSAccessKeyID
		ossKeySecret := cfg.OSSAccessKeySecret
		if ossKeyID == "" {
			ossKeyID = cfg.AccessKeyID
		}
		if ossKeySecret == "" {
			ossKeySecret = cfg.AccessKeySecret
		}
		if cfg.OSSEndpoint != "" && cfg.OSSBucketName != "" && ossKeyID != "" && ossKeySecret != "" {
			client.uploader = &aliyunOSSUploader{ossConnection{
				endpoint:        cfg.OSSEndpoint,
				bucket:          cfg.OSSBucketName,
				accessKeyID:     ossKeyID,
				accessKeySecret: ossKeySecret,
				securityToken:   cfg.STSToken,
			}}
		}
	}
	if client.fileExists == nil {
		client.fileExists = func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		}
	}
	return client, nil
}

// Do sends a raw request: payload is any JSON-marshalable value (typed
// request struct or map) and the response body is unmarshaled into result
// when result is non-nil. It is the escape hatch for API fields not yet
// covered by the typed models.
func (c *Client) Do(ctx context.Context, action string, payload any, result any) error {
	body, err := c.ots.request(ctx, action, payload)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(body, result); err != nil {
		return newClientErrorWithCause(ClientErrorKindResponseDecode, err, "failed to parse JSON response: %v", err)
	}
	return nil
}

func do[T any](c *Client, ctx context.Context, action string, payload any) (*T, error) {
	var result T
	if err := c.Do(ctx, action, payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// Knowledge base APIs
// ---------------------------------------------------------------------------

// CreateKnowledgeBase creates a knowledge base.
func (c *Client) CreateKnowledgeBase(ctx context.Context, req *CreateKnowledgeBaseRequest) (*CreateKnowledgeBaseResponse, error) {
	return do[CreateKnowledgeBaseResponse](c, ctx, "CreateKnowledgeBase", req)
}

// DeleteKnowledgeBase deletes a knowledge base.
func (c *Client) DeleteKnowledgeBase(ctx context.Context, req *DeleteKnowledgeBaseRequest) (*DeleteKnowledgeBaseResponse, error) {
	return do[DeleteKnowledgeBaseResponse](c, ctx, "DeleteKnowledgeBase", req)
}

// DescribeKnowledgeBase returns the details of a knowledge base.
func (c *Client) DescribeKnowledgeBase(ctx context.Context, req *DescribeKnowledgeBaseRequest) (*DescribeKnowledgeBaseResponse, error) {
	return do[DescribeKnowledgeBaseResponse](c, ctx, "DescribeKnowledgeBase", req)
}

// ListKnowledgeBase lists knowledge bases. A nil request lists from the start.
func (c *Client) ListKnowledgeBase(ctx context.Context, req *ListKnowledgeBaseRequest) (*ListKnowledgeBaseResponse, error) {
	if req == nil {
		req = &ListKnowledgeBaseRequest{}
	}
	return do[ListKnowledgeBaseResponse](c, ctx, "ListKnowledgeBase", req)
}

// UpdateKnowledgeBase updates a knowledge base.
func (c *Client) UpdateKnowledgeBase(ctx context.Context, req *UpdateKnowledgeBaseRequest) (*UpdateKnowledgeBaseResponse, error) {
	return do[UpdateKnowledgeBaseResponse](c, ctx, "UpdateKnowledgeBase", req)
}

// ---------------------------------------------------------------------------
// Document APIs
// ---------------------------------------------------------------------------

// AddDocuments registers OSS objects as documents of a knowledge base.
func (c *Client) AddDocuments(ctx context.Context, req *AddDocumentsRequest) (*AddDocumentsResponse, error) {
	return do[AddDocumentsResponse](c, ctx, "AddDocuments", req)
}

// UpdateDocument updates a document's metadata.
func (c *Client) UpdateDocument(ctx context.Context, req *UpdateDocumentRequest) (*UpdateDocumentResponse, error) {
	return do[UpdateDocumentResponse](c, ctx, "UpdateDocument", req)
}

// GetDocument returns a document by docId or ossKey.
func (c *Client) GetDocument(ctx context.Context, req *GetDocumentRequest) (*GetDocumentResponse, error) {
	return do[GetDocumentResponse](c, ctx, "GetDocument", req)
}

// ListDocuments lists the documents of a knowledge base.
func (c *Client) ListDocuments(ctx context.Context, req *ListDocumentsRequest) (*ListDocumentsResponse, error) {
	return do[ListDocumentsResponse](c, ctx, "ListDocuments", req)
}

// DeleteDocuments removes documents from a knowledge base. The backing OSS
// files are kept; use DeleteDocumentsWithOptions to also delete them.
func (c *Client) DeleteDocuments(ctx context.Context, req *DeleteDocumentsRequest) (*DeleteDocumentsResponse, error) {
	return do[DeleteDocumentsResponse](c, ctx, "DeleteDocuments", req)
}

// DeleteDocumentsOptions carries the optional flags of DeleteDocumentsWithOptions.
type DeleteDocumentsOptions struct {
	// DeleteFile also deletes the backing OSS objects (only those in the
	// configured bucket) after the documents are removed.
	DeleteFile bool
}

// DeleteDocumentsWithOptions removes documents and, when opts.DeleteFile is
// set, best-effort deletes the backing OSS objects. Objects living in a
// different bucket than the configured one are skipped.
func (c *Client) DeleteDocumentsWithOptions(ctx context.Context, req *DeleteDocumentsRequest, opts DeleteDocumentsOptions) (*DeleteDocumentsResponse, error) {
	if !opts.DeleteFile {
		return c.DeleteDocuments(ctx, req)
	}
	if c.uploader == nil {
		return nil, newClientError("OSS client not initialized. Please provide OSS configuration.")
	}

	var objectKeys []string
	for _, document := range req.Documents {
		ossKey := document.OssKey
		if ossKey == "" && document.DocID != "" {
			resp, err := c.GetDocument(ctx, &GetDocumentRequest{
				KnowledgeBaseName: req.KnowledgeBaseName,
				Subspace:          req.Subspace,
				DocID:             document.DocID,
			})
			if err != nil {
				continue
			}
			for _, info := range resp.Data {
				if info.OssKey != "" {
					ossKey = info.OssKey
					break
				}
			}
		}
		if ossKey == "" {
			continue
		}
		bucketName, objectKey, ok := parseOssPath(ossKey, c.bucketName)
		if !ok || bucketName != c.bucketName {
			continue
		}
		objectKeys = append(objectKeys, objectKey)
	}

	resp, err := c.DeleteDocuments(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(objectKeys) > 0 {
		// Best-effort: a failed OSS cleanup must not fail the delete itself.
		_ = c.uploader.DeleteObjects(ctx, objectKeys)
	}
	return resp, nil
}

// UploadDocuments uploads local files to OSS and registers them via
// AddDocuments. If AddDocuments fails with a service error, the uploaded
// objects are removed again.
func (c *Client) UploadDocuments(ctx context.Context, req *UploadDocumentsRequest) (*AddDocumentsResponse, error) {
	if c.uploader == nil {
		return nil, newClientError("OSS client not initialized. Please provide OSS configuration.")
	}
	if req.KnowledgeBaseName == "" {
		return nil, newClientError("Request must contain 'knowledgeBaseName' field")
	}

	uploadedKeys := make([]string, 0, len(req.Documents))
	documents := make([]AddDocumentItem, 0, len(req.Documents))
	for _, document := range req.Documents {
		if document.FilePath == "" {
			return nil, newClientError("Each document must contain 'filePath' field")
		}
		if !c.fileExists(document.FilePath) {
			return nil, newClientError("File not found: %s", document.FilePath)
		}
		objectKey := buildOssObjectKey(c.instanceName, req.KnowledgeBaseName, req.Subspace, document.FilePath)
		if err := c.uploader.PutObjectFromFile(ctx, objectKey, document.FilePath); err != nil {
			return nil, newClientErrorWithCause(classifyTransportError(err), err, "Failed to upload file to OSS: %v", err)
		}
		uploadedKeys = append(uploadedKeys, objectKey)
		documents = append(documents, AddDocumentItem{
			OssKey:   buildOssFullPath(c.bucketName, objectKey),
			Metadata: document.Metadata,
		})
	}

	resp, err := c.AddDocuments(ctx, &AddDocumentsRequest{
		KnowledgeBaseName: req.KnowledgeBaseName,
		Subspace:          req.Subspace,
		Documents:         documents,
	})
	if err != nil {
		var serviceErr *ServiceError
		if errors.As(err, &serviceErr) && len(uploadedKeys) > 0 {
			_ = c.uploader.DeleteObjects(ctx, uploadedKeys)
		}
		return nil, err
	}
	return resp, nil
}

// ---------------------------------------------------------------------------
// Retrieval and chunk APIs
// ---------------------------------------------------------------------------

// Retrieve performs vector / full-text retrieval in a knowledge base.
func (c *Client) Retrieve(ctx context.Context, req *RetrieveRequest) (*RetrieveResponse, error) {
	return do[RetrieveResponse](c, ctx, "Retrieve", req)
}

// ListChunks lists the chunks of a document.
func (c *Client) ListChunks(ctx context.Context, req *ListChunksRequest) (*ListChunksResponse, error) {
	return do[ListChunksResponse](c, ctx, "ListChunks", req)
}

// UpdateChunks updates chunk content or status.
func (c *Client) UpdateChunks(ctx context.Context, req *UpdateChunksRequest) (*UpdateChunksResponse, error) {
	return do[UpdateChunksResponse](c, ctx, "UpdateChunks", req)
}

// ---------------------------------------------------------------------------
// Memory store APIs
// ---------------------------------------------------------------------------

// CreateMemoryStore creates a memory store.
func (c *Client) CreateMemoryStore(ctx context.Context, req *CreateMemoryStoreRequest) (*CreateMemoryStoreResponse, error) {
	return do[CreateMemoryStoreResponse](c, ctx, "CreateMemoryStore", req)
}

// GetMemoryStore returns the details of a memory store.
func (c *Client) GetMemoryStore(ctx context.Context, req *GetMemoryStoreRequest) (*GetMemoryStoreResponse, error) {
	return do[GetMemoryStoreResponse](c, ctx, "GetMemoryStore", req)
}

// ListMemoryStores lists memory stores. A nil request lists from the start.
func (c *Client) ListMemoryStores(ctx context.Context, req *ListMemoryStoresRequest) (*ListMemoryStoresResponse, error) {
	if req == nil {
		req = &ListMemoryStoresRequest{}
	}
	return do[ListMemoryStoresResponse](c, ctx, "ListMemoryStores", req)
}

// UpdateMemoryStore updates a memory store (PATCH semantics).
func (c *Client) UpdateMemoryStore(ctx context.Context, req *UpdateMemoryStoreRequest) (*UpdateMemoryStoreResponse, error) {
	return do[UpdateMemoryStoreResponse](c, ctx, "UpdateMemoryStore", req)
}

// DeleteMemoryStore deletes a memory store including all of its data.
func (c *Client) DeleteMemoryStore(ctx context.Context, req *DeleteMemoryStoreRequest) (*DeleteMemoryStoreResponse, error) {
	return do[DeleteMemoryStoreResponse](c, ctx, "DeleteMemoryStore", req)
}

// ---------------------------------------------------------------------------
// Memory APIs
// ---------------------------------------------------------------------------

// AddMemories writes conversation messages or text and extracts long-term
// memories from them.
func (c *Client) AddMemories(ctx context.Context, req *AddMemoriesRequest) (*AddMemoriesResponse, error) {
	return do[AddMemoriesResponse](c, ctx, "AddMemories", req)
}

// SearchMemories retrieves long-term memories.
func (c *Client) SearchMemories(ctx context.Context, req *SearchMemoriesRequest) (*SearchMemoriesResponse, error) {
	return do[SearchMemoriesResponse](c, ctx, "SearchMemories", req)
}

// ListMemories lists long-term memories.
func (c *Client) ListMemories(ctx context.Context, req *ListMemoriesRequest) (*ListMemoriesResponse, error) {
	return do[ListMemoriesResponse](c, ctx, "ListMemories", req)
}

// GetMemory returns a single long-term memory.
func (c *Client) GetMemory(ctx context.Context, req *GetMemoryRequest) (*GetMemoryResponse, error) {
	return do[GetMemoryResponse](c, ctx, "GetMemory", req)
}

// UpdateMemory updates a single long-term memory.
func (c *Client) UpdateMemory(ctx context.Context, req *UpdateMemoryRequest) (*UpdateMemoryResponse, error) {
	return do[UpdateMemoryResponse](c, ctx, "UpdateMemory", req)
}

// DeleteMemory deletes a single long-term memory.
func (c *Client) DeleteMemory(ctx context.Context, req *DeleteMemoryRequest) (*DeleteMemoryResponse, error) {
	return do[DeleteMemoryResponse](c, ctx, "DeleteMemory", req)
}

// ListMemoryStoreMessages lists raw conversation messages (short-term memory).
func (c *Client) ListMemoryStoreMessages(ctx context.Context, req *ListMemoryStoreMessagesRequest) (*ListMemoryStoreMessagesResponse, error) {
	return do[ListMemoryStoreMessagesResponse](c, ctx, "ListMemoryStoreMessages", req)
}

// ListMemoryStoreRequests lists request audit records.
func (c *Client) ListMemoryStoreRequests(ctx context.Context, req *ListMemoryStoreRequestsRequest) (*ListMemoryStoreRequestsResponse, error) {
	return do[ListMemoryStoreRequestsResponse](c, ctx, "ListMemoryStoreRequests", req)
}

// GetMemoryTask returns the status of an async ingestion task.
func (c *Client) GetMemoryTask(ctx context.Context, req *GetMemoryTaskRequest) (*GetMemoryTaskResponse, error) {
	return do[GetMemoryTaskResponse](c, ctx, "GetMemoryTask", req)
}

// ListMemoryTasks lists async ingestion tasks.
func (c *Client) ListMemoryTasks(ctx context.Context, req *ListMemoryTasksRequest) (*ListMemoryTasksResponse, error) {
	return do[ListMemoryTasksResponse](c, ctx, "ListMemoryTasks", req)
}

// ListMemoryStoreScopes lists the scopes present in a memory store.
func (c *Client) ListMemoryStoreScopes(ctx context.Context, req *ListMemoryStoreScopesRequest) (*ListMemoryStoreScopesResponse, error) {
	return do[ListMemoryStoreScopesResponse](c, ctx, "ListMemoryStoreScopes", req)
}

// ---------------------------------------------------------------------------
// Memory dreaming APIs
// ---------------------------------------------------------------------------

// CreateMemoryDreamTask starts a memory consolidation (dream) task.
func (c *Client) CreateMemoryDreamTask(ctx context.Context, req *CreateMemoryDreamTaskRequest) (*CreateMemoryDreamTaskResponse, error) {
	return do[CreateMemoryDreamTaskResponse](c, ctx, "CreateMemoryDreamTask", req)
}

// GetMemoryDreamTask returns progress and result summary of a dream task.
func (c *Client) GetMemoryDreamTask(ctx context.Context, req *GetMemoryDreamTaskRequest) (*GetMemoryDreamTaskResponse, error) {
	return do[GetMemoryDreamTaskResponse](c, ctx, "GetMemoryDreamTask", req)
}

// ListMemoryDreamTasks lists dream tasks.
func (c *Client) ListMemoryDreamTasks(ctx context.Context, req *ListMemoryDreamTasksRequest) (*ListMemoryDreamTasksResponse, error) {
	return do[ListMemoryDreamTasksResponse](c, ctx, "ListMemoryDreamTasks", req)
}

// CancelMemoryDreamTask cancels an unfinished dream task.
func (c *Client) CancelMemoryDreamTask(ctx context.Context, req *CancelMemoryDreamTaskRequest) (*CancelMemoryDreamTaskResponse, error) {
	return do[CancelMemoryDreamTaskResponse](c, ctx, "CancelMemoryDreamTask", req)
}

// ListMemoryDreamActions lists the actions proposed by a dream task.
func (c *Client) ListMemoryDreamActions(ctx context.Context, req *ListMemoryDreamActionsRequest) (*ListMemoryDreamActionsResponse, error) {
	return do[ListMemoryDreamActionsResponse](c, ctx, "ListMemoryDreamActions", req)
}

// ApplyMemoryDreamActions applies proposed dream actions.
func (c *Client) ApplyMemoryDreamActions(ctx context.Context, req *ApplyMemoryDreamActionsRequest) (*ApplyMemoryDreamActionsResponse, error) {
	return do[ApplyMemoryDreamActionsResponse](c, ctx, "ApplyMemoryDreamActions", req)
}

// ---------------------------------------------------------------------------
// Item APIs (file-memory stores)
// ---------------------------------------------------------------------------

// AddItem creates a new item in a file-memory store. SessionID is recorded
// as the version's session actor; View selects whether the response echoes
// content (ItemViewFull) or not (default ItemViewBasic).
func (c *Client) AddItem(ctx context.Context, req *AddItemRequest) (*AddItemResponse, error) {
	if req == nil {
		req = &AddItemRequest{}
	}
	r := *req
	r.setDefaultType()
	return do[AddItemResponse](c, ctx, "AddItem", &r)
}

// ListItems lists items in a scope; wildcard scopes following the
// left-prefix rule are allowed. Depth 1 rolls deeper directories up into
// entries typed ItemTypeMemoryFilePrefix.
func (c *Client) ListItems(ctx context.Context, req *ListItemsRequest) (*ListItemsResponse, error) {
	if req == nil {
		req = &ListItemsRequest{}
	}
	r := *req
	r.setDefaultType()
	return do[ListItemsResponse](c, ctx, "ListItems", &r)
}

// GetItem returns one item addressed by exactly one of ItemID and Path.
// Content is omitted unless View is ItemViewFull (the default ItemViewBasic
// fetches metadata only).
func (c *Client) GetItem(ctx context.Context, req *GetItemRequest) (*GetItemResponse, error) {
	if req == nil {
		req = &GetItemRequest{}
	}
	r := *req
	r.setDefaultType()
	return do[GetItemResponse](c, ctx, "GetItem", &r)
}

// UpdateItem atomically changes content and/or Path using ItemID and an
// optional version Precondition. Legacy path addressing, NewPath and SHA
// guards are translated to the current protocol without mutating req.
func (c *Client) UpdateItem(ctx context.Context, req *UpdateItemRequest) (*UpdateItemResponse, error) {
	if req == nil {
		req = &UpdateItemRequest{}
	}
	r := *req
	r.setDefaultType()
	// Path is a rename destination in the current protocol. Only explicit
	// deprecated fields opt into interpreting it as the legacy source address.
	legacyPath := r.ItemID == "" && (r.NewPath != "" || r.ExpectedSHA256 != "")
	if r.ItemID == "" && r.Path != nil && !legacyPath {
		return nil, newClientErrorWithCause(ClientErrorKindRequestBuild, nil, "UpdateItem requires ItemID; legacy path addressing requires NewPath or ExpectedSHA256")
	}
	if r.ExpectedSHA256 != "" || legacyPath {
		path := ""
		if r.Path != nil {
			path = *r.Path
		}
		// Guard reads need metadata only. A legacy no-op returns the read
		// itself, so only that case needs the caller's requested response view.
		legacyNoOp := r.ItemID == "" && r.Content == nil && r.ContentBase64 == nil && r.NewPath == ""
		view := ItemViewBasic
		if legacyNoOp {
			view = r.View
		}
		item, err := c.resolveItemGuard(ctx, r.ItemScopedRequest, r.ItemID, path, r.ExpectedSHA256, r.Precondition, view)
		if err != nil {
			return nil, err
		}
		if r.ItemID == "" {
			r.ItemID, r.Path = item.ItemID, nil
			// Preserve legacy path-based no-op requests as a read.
			if legacyNoOp {
				return item, nil
			}
		}
		if r.ExpectedSHA256 != "" {
			r.Precondition = &ItemPrecondition{ExpectedVersionID: item.LatestVersionID}
			r.ExpectedSHA256 = ""
		}
	}
	return do[UpdateItemResponse](c, ctx, "UpdateItem", &r)
}

// DeleteItem deletes one item, or a whole wildcard scope asynchronously.
//
// Single delete: exact full scope plus exactly one of ItemID or Path;
// synchronous, the response carries Deleted=true. Ranged delete:
// wildcard scope without ItemID/Path/ExpectedSHA256, optional segment-
// aligned PathPrefix; asynchronous, the service answers 200 and the
// response carries TaskID with Status="pending".
func (c *Client) DeleteItem(ctx context.Context, req *DeleteItemRequest) (*DeleteItemResponse, error) {
	if req == nil {
		req = &DeleteItemRequest{}
	}
	r := *req
	r.setDefaultType()
	if r.ExpectedSHA256 != "" {
		if r.ItemID != "" && r.Path != "" {
			return nil, newClientErrorWithCause(ClientErrorKindRequestBuild, nil, "DeleteItem requires exactly one of ItemID or Path")
		}
		item, err := c.resolveItemGuard(ctx, r.ItemScopedRequest, r.ItemID, r.Path, r.ExpectedSHA256, r.Precondition, ItemViewBasic)
		if err != nil {
			return nil, err
		}
		r.ItemID, r.Path = item.ItemID, ""
		r.Precondition = &ItemPrecondition{ExpectedVersionID: item.LatestVersionID}
		r.ExpectedSHA256 = ""
	}
	return do[DeleteItemResponse](c, ctx, "DeleteItem", &r)
}

// ListItemVersions lists immutable versions in a scope, newest first. All
// filters (ItemID/Path, SessionID, CreatedAtGte/Lte, Operation) are
// optional; the scope may use the left-prefix wildcard.
func (c *Client) ListItemVersions(ctx context.Context, req *ListItemVersionsRequest) (*ListItemVersionsResponse, error) {
	if req == nil {
		req = &ListItemVersionsRequest{}
	}
	r := *req
	r.setDefaultType()
	return do[ListItemVersionsResponse](c, ctx, "ListItemVersions", &r)
}

// GetItemVersion returns one immutable item version identified by
// VersionID, which is self-contained: the server derives all addressing
// from it and verifies scope ownership. Content is omitted unless View is
// ItemViewFull, and never returned for redacted versions.
func (c *Client) GetItemVersion(ctx context.Context, req *GetItemVersionRequest) (*GetItemVersionResponse, error) {
	if req == nil {
		req = &GetItemVersionRequest{}
	}
	r := *req
	r.setDefaultType()
	return do[GetItemVersionResponse](c, ctx, "GetItemVersion", &r)
}

// RedactItemVersion irreversibly clears the content, sha, size and path of
// one item version identified by VersionID, keeping the actor and
// timestamps. It is idempotent and first-wins.
func (c *Client) RedactItemVersion(ctx context.Context, req *RedactItemVersionRequest) (*RedactItemVersionResponse, error) {
	if req == nil {
		req = &RedactItemVersionRequest{}
	}
	r := *req
	r.setDefaultType()
	return do[RedactItemVersionResponse](c, ctx, "RedactItemVersion", &r)
}

// GetWiki returns Wiki metadata and the latest build state.
func (c *Client) GetWiki(ctx context.Context, req *GetWikiRequest) (*GetWikiResponse, error) {
	return do[GetWikiResponse](c, ctx, "GetWiki", req)
}

// UpdateWiki updates Wiki metadata and compile options.
func (c *Client) UpdateWiki(ctx context.Context, req *UpdateWikiRequest) (*UpdateWikiResponse, error) {
	return do[UpdateWikiResponse](c, ctx, "UpdateWiki", req)
}

// IngestWiki requests compilation of pending Wiki sources.
func (c *Client) IngestWiki(ctx context.Context, req *IngestWikiRequest) (*IngestWikiResponse, error) {
	return do[IngestWikiResponse](c, ctx, "IngestWiki", req)
}

// ListWikiPages lists published Wiki pages.
func (c *Client) ListWikiPages(ctx context.Context, req *ListWikiPagesRequest) (*ListWikiPagesResponse, error) {
	return do[ListWikiPagesResponse](c, ctx, "ListWikiPages", req)
}

// GetWikiPage returns one published Wiki page.
func (c *Client) GetWikiPage(ctx context.Context, req *GetWikiPageRequest) (*GetWikiPageResponse, error) {
	return do[GetWikiPageResponse](c, ctx, "GetWikiPage", req)
}

// SearchWikiPages searches published Wiki pages.
func (c *Client) SearchWikiPages(ctx context.Context, req *SearchWikiPagesRequest) (*SearchWikiPagesResponse, error) {
	return do[SearchWikiPagesResponse](c, ctx, "SearchWikiPages", req)
}
