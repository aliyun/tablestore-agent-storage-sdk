import { OTSJSONClient, type SignVersion } from '../http/ots-json-client.js';
import { DEFAULT_USER_AGENT } from '../version.js';
import type { JsonRequestConvertible } from '../models/base.js';
import type {
  AddDocumentsRequest,
  DeleteDocumentsRequest,
  GetDocumentRequest,
  ListDocumentsRequest,
  UpdateDocumentRequest,
  UploadDocumentsRequest,
} from '../models/document.js';
import type {
  CreateKnowledgeBaseRequest,
  DeleteKnowledgeBaseRequest,
  DescribeKnowledgeBaseRequest,
  ListKnowledgeBaseRequest,
  UpdateKnowledgeBaseRequest,
} from '../models/knowledge-base.js';
import type { ListChunksRequest, UpdateChunksRequest } from '../models/chunk.js';
import type { RetrieveRequest } from '../models/retrieve.js';
import {
  ITEM_TYPE_MEMORY_FILE,
  type AddItemRequest,
  type AddItemResponse,
  type DeleteItemRequest,
  type DeleteItemResponse,
  type GetItemRequest,
  type GetItemResponse,
  type GetItemVersionRequest,
  type GetItemVersionResponse,
  type ListItemsRequest,
  type ListItemsResponse,
  type ListItemVersionsRequest,
  type ListItemVersionsResponse,
  type RedactItemVersionRequest,
  type RedactItemVersionResponse,
  type UpdateItemRequest,
  type UpdateItemResponse,
} from '../models/file.js';
import type {
  AddMemoriesRequest,
  AddMemoriesResponse,
  ApplyMemoryDreamActionsRequest,
  ApplyMemoryDreamActionsResponse,
  CancelMemoryDreamTaskRequest,
  CancelMemoryDreamTaskResponse,
  CreateMemoryDreamTaskRequest,
  CreateMemoryDreamTaskResponse,
  CreateMemoryStoreRequest,
  CreateMemoryStoreResponse,
  DeleteMemoryRequest,
  DeleteMemoryResponse,
  DeleteMemoryStoreRequest,
  DeleteMemoryStoreResponse,
  GetMemoryDreamTaskRequest,
  GetMemoryDreamTaskResponse,
  GetMemoryRequest,
  GetMemoryResponse,
  GetMemoryStoreRequest,
  GetMemoryStoreResponse,
  GetMemoryTaskRequest,
  GetMemoryTaskResponse,
  ListMemoriesRequest,
  ListMemoriesResponse,
  ListMemoryDreamActionsRequest,
  ListMemoryDreamActionsResponse,
  ListMemoryDreamTasksRequest,
  ListMemoryDreamTasksResponse,
  ListMemoryStoreMessagesRequest,
  ListMemoryStoreMessagesResponse,
  ListMemoryStoreRequestsRequest,
  ListMemoryStoreRequestsResponse,
  ListMemoryStoreScopesRequest,
  ListMemoryStoreScopesResponse,
  ListMemoryStoresRequest,
  ListMemoryStoresResponse,
  ListMemoryTasksRequest,
  ListMemoryTasksResponse,
  SearchMemoriesRequest,
  SearchMemoriesResponse,
  UpdateMemoryRequest,
  UpdateMemoryResponse,
  UpdateMemoryStoreRequest,
  UpdateMemoryStoreResponse,
} from '../models/memory.js';
import { toJSONRequest } from '../serialization/to-json-request.js';

export interface AgentStorageClientOptions {
  endpoint: string;
  instanceName: string;
  accessKeyId?: string;
  accessKeySecret?: string;
  stsToken?: string;
  apiKey?: string;
  region?: string;
  signVersion?: SignVersion;
  fetch?: typeof globalThis.fetch;
  userAgentSuffix?: string;
  extraHeaders?: Record<string, string>;
}

type RequestInput<T extends JsonRequestConvertible> = T | Record<string, unknown>;

/**
 * Normalizes an item request body, filling `type: 'memoryfile'` when the
 * caller did not provide a non-empty item type.
 */
function toItemRequest(request?: Record<string, unknown> | null): Record<string, unknown> {
  const body = (toJSONRequest(request ?? {}) ?? {}) as Record<string, unknown>;
  if (typeof body.type === 'string' && body.type.length > 0) {
    return body;
  }
  const { type: _omitted, ...rest } = body;
  return { type: ITEM_TYPE_MEMORY_FILE, ...rest };
}

export class AgentStorageClient {
  protected readonly client: OTSJSONClient;

  constructor(options: AgentStorageClientOptions) {
    const fetchImpl = options.fetch ?? globalThis.fetch;
    const userAgent = options.userAgentSuffix
      ? `${DEFAULT_USER_AGENT} ${options.userAgentSuffix}`
      : DEFAULT_USER_AGENT;

    this.client = new OTSJSONClient({
      endpoint: options.endpoint,
      instanceName: options.instanceName,
      credentials:
        options.accessKeyId && options.accessKeySecret
          ? {
              accessKeyId: options.accessKeyId,
              accessKeySecret: options.accessKeySecret,
              securityToken: options.stsToken,
            }
          : undefined,
      apiKey: options.apiKey,
      signVersion: options.signVersion ?? 'v2',
      region: options.region,
      fetch: fetchImpl,
      userAgent,
      extraHeaders: options.extraHeaders,
    });
  }

  createKnowledgeBase(request: RequestInput<CreateKnowledgeBaseRequest>) {
    return this.client.request('CreateKnowledgeBase', toJSONRequest(request));
  }

  deleteKnowledgeBase(request: RequestInput<DeleteKnowledgeBaseRequest>) {
    return this.client.request('DeleteKnowledgeBase', toJSONRequest(request));
  }

  describeKnowledgeBase(request: RequestInput<DescribeKnowledgeBaseRequest>) {
    return this.client.request('DescribeKnowledgeBase', toJSONRequest(request));
  }

  listKnowledgeBase(request: RequestInput<ListKnowledgeBaseRequest> = {}) {
    return this.client.request('ListKnowledgeBase', toJSONRequest(request));
  }

  updateKnowledgeBase(request: RequestInput<UpdateKnowledgeBaseRequest>) {
    return this.client.request('UpdateKnowledgeBase', toJSONRequest(request));
  }

  addDocuments(request: RequestInput<AddDocumentsRequest>) {
    return this.client.request('AddDocuments', toJSONRequest(request));
  }

  updateDocument(request: RequestInput<UpdateDocumentRequest>) {
    return this.client.request('UpdateDocument', toJSONRequest(request));
  }

  getDocument(request: RequestInput<GetDocumentRequest>) {
    return this.client.request('GetDocument', toJSONRequest(request));
  }

  listDocuments(request: RequestInput<ListDocumentsRequest>) {
    return this.client.request('ListDocuments', toJSONRequest(request));
  }

  deleteDocuments(request: RequestInput<DeleteDocumentsRequest>) {
    return this.client.request('DeleteDocuments', toJSONRequest(request));
  }

  retrieve(request: RequestInput<RetrieveRequest>) {
    return this.client.request('Retrieve', toJSONRequest(request));
  }

  listChunks(request: RequestInput<ListChunksRequest>) {
    return this.client.request('ListChunks', toJSONRequest(request));
  }

  updateChunks(request: RequestInput<UpdateChunksRequest>) {
    return this.client.request('UpdateChunks', toJSONRequest(request));
  }

  createMemoryStore(request: RequestInput<CreateMemoryStoreRequest>) {
    return this.client.request<CreateMemoryStoreResponse>('CreateMemoryStore', toJSONRequest(request));
  }

  getMemoryStore(request: RequestInput<GetMemoryStoreRequest>) {
    return this.client.request<GetMemoryStoreResponse>('GetMemoryStore', toJSONRequest(request));
  }

  listMemoryStores(request: RequestInput<ListMemoryStoresRequest> = {}) {
    return this.client.request<ListMemoryStoresResponse>('ListMemoryStores', toJSONRequest(request));
  }

  updateMemoryStore(request: RequestInput<UpdateMemoryStoreRequest>) {
    return this.client.request<UpdateMemoryStoreResponse>('UpdateMemoryStore', toJSONRequest(request));
  }

  deleteMemoryStore(request: RequestInput<DeleteMemoryStoreRequest>) {
    return this.client.request<DeleteMemoryStoreResponse>('DeleteMemoryStore', toJSONRequest(request));
  }

  addMemories(request: RequestInput<AddMemoriesRequest>) {
    return this.client.request<AddMemoriesResponse>('AddMemories', toJSONRequest(request));
  }

  searchMemories(request: RequestInput<SearchMemoriesRequest>) {
    return this.client.request<SearchMemoriesResponse>('SearchMemories', toJSONRequest(request));
  }

  listMemories(request: RequestInput<ListMemoriesRequest>) {
    return this.client.request<ListMemoriesResponse>('ListMemories', toJSONRequest(request));
  }

  getMemory(request: RequestInput<GetMemoryRequest>) {
    return this.client.request<GetMemoryResponse>('GetMemory', toJSONRequest(request));
  }

  updateMemory(request: RequestInput<UpdateMemoryRequest>) {
    return this.client.request<UpdateMemoryResponse>('UpdateMemory', toJSONRequest(request));
  }

  deleteMemory(request: RequestInput<DeleteMemoryRequest>) {
    return this.client.request<DeleteMemoryResponse>('DeleteMemory', toJSONRequest(request));
  }

  listMemoryStoreMessages(request: RequestInput<ListMemoryStoreMessagesRequest>) {
    return this.client.request<ListMemoryStoreMessagesResponse>('ListMemoryStoreMessages', toJSONRequest(request));
  }

  listMemoryStoreRequests(request: RequestInput<ListMemoryStoreRequestsRequest>) {
    return this.client.request<ListMemoryStoreRequestsResponse>('ListMemoryStoreRequests', toJSONRequest(request));
  }

  getMemoryTask(request: RequestInput<GetMemoryTaskRequest>) {
    return this.client.request<GetMemoryTaskResponse>('GetMemoryTask', toJSONRequest(request));
  }

  listMemoryTasks(request: RequestInput<ListMemoryTasksRequest>) {
    return this.client.request<ListMemoryTasksResponse>('ListMemoryTasks', toJSONRequest(request));
  }

  listMemoryStoreScopes(request: RequestInput<ListMemoryStoreScopesRequest>) {
    return this.client.request<ListMemoryStoreScopesResponse>('ListMemoryStoreScopes', toJSONRequest(request));
  }

  createMemoryDreamTask(request: RequestInput<CreateMemoryDreamTaskRequest>) {
    return this.client.request<CreateMemoryDreamTaskResponse>('CreateMemoryDreamTask', toJSONRequest(request));
  }

  getMemoryDreamTask(request: RequestInput<GetMemoryDreamTaskRequest>) {
    return this.client.request<GetMemoryDreamTaskResponse>('GetMemoryDreamTask', toJSONRequest(request));
  }

  listMemoryDreamTasks(request: RequestInput<ListMemoryDreamTasksRequest>) {
    return this.client.request<ListMemoryDreamTasksResponse>('ListMemoryDreamTasks', toJSONRequest(request));
  }

  cancelMemoryDreamTask(request: RequestInput<CancelMemoryDreamTaskRequest>) {
    return this.client.request<CancelMemoryDreamTaskResponse>('CancelMemoryDreamTask', toJSONRequest(request));
  }

  listMemoryDreamActions(request: RequestInput<ListMemoryDreamActionsRequest>) {
    return this.client.request<ListMemoryDreamActionsResponse>('ListMemoryDreamActions', toJSONRequest(request));
  }

  applyMemoryDreamActions(request: RequestInput<ApplyMemoryDreamActionsRequest>) {
    return this.client.request<ApplyMemoryDreamActionsResponse>('ApplyMemoryDreamActions', toJSONRequest(request));
  }

  addItem(request: AddItemRequest | Record<string, unknown>) {
    return this.client.request<AddItemResponse>(
      'AddItem',
      toItemRequest(request as Record<string, unknown>)
    );
  }

  listItems(request: ListItemsRequest | Record<string, unknown>) {
    return this.client.request<ListItemsResponse>(
      'ListItems',
      toItemRequest(request as Record<string, unknown>)
    );
  }

  getItem(request: GetItemRequest | Record<string, unknown>) {
    return this.client.request<GetItemResponse>(
      'GetItem',
      toItemRequest(request as Record<string, unknown>)
    );
  }

  updateItem(request: UpdateItemRequest | Record<string, unknown>) {
    return this.client.request<UpdateItemResponse>(
      'UpdateItem',
      toItemRequest(request as Record<string, unknown>)
    );
  }

  deleteItem(request: DeleteItemRequest | Record<string, unknown>) {
    return this.client.request<DeleteItemResponse>(
      'DeleteItem',
      toItemRequest(request as Record<string, unknown>)
    );
  }

  listItemVersions(request: ListItemVersionsRequest | Record<string, unknown>) {
    return this.client.request<ListItemVersionsResponse>(
      'ListItemVersions',
      toItemRequest(request as Record<string, unknown>)
    );
  }

  getItemVersion(request: GetItemVersionRequest | Record<string, unknown>) {
    return this.client.request<GetItemVersionResponse>(
      'GetItemVersion',
      toItemRequest(request as Record<string, unknown>)
    );
  }

  redactItemVersion(request: RedactItemVersionRequest | Record<string, unknown>) {
    return this.client.request<RedactItemVersionResponse>(
      'RedactItemVersion',
      toItemRequest(request as Record<string, unknown>)
    );
  }

  uploadDocuments(_request: RequestInput<UploadDocumentsRequest>) {
    throw new Error('uploadDocuments is only available in the NodeAgentStorageClient.');
  }
}
