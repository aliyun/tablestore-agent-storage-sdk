import type { JsonRequestConvertible } from './base.js';
import { assignIfPresent, toJsonValue } from './base.js';
import type { Unset } from './unset.js';

/**
 * Storage mode of a memory store. Chosen at CreateMemoryStore time and
 * immutable afterwards - the service exposes no code path that rewrites it.
 *
 * - `ots`      structured memory only (default when omitted).
 * - `file+ots` structured memory plus a read-only projection of files.
 *              Item write APIs return 409 `READ_ONLY_STORE`; item version APIs
 *              return 400.
 * - `filemem`  writable file memory.
 *
 * A legacy `oss+ots` value may still be persisted on very old stores and is
 * normalized to `file+ots` on read; it is rejected on write.
 */
export type MemoryStoreStorageMode = 'ots' | 'file+ots' | 'filemem';

export interface ScopeValue {
  appId?: string;
  tenantId?: string;
  agentId?: string;
  runId?: string;
}

export interface MessagePayloadValue {
  role: string;
  content: string;
  messageId?: string;
  timestamp?: string;
  metadata?: Record<string, string>;
}

export interface MemoryUnit {
  id: string;
  conversation_key: string;
  scope: ScopeValue;
  memcell_id?: string;
  unit_type?: string;
  text?: string;
  search_text?: string;
  entities?: string[];
  time_anchor?: Record<string, string>;
  source_turn_ids?: string[];
  speaker?: string;
  topic?: string;
  type_label?: string;
  date_bucket?: string;
  metadata?: Record<string, string>;
  metadata_json?: string;
  metadata_flat?: string;
  deleted?: boolean;
  created_at?: string;
  salience?: number;
  version?: number;
  superseded_by?: string;
  supersedes_id?: string;
}

/**
 * Session-affinity ranking signals. Present only when the server-side ranking
 * policy is enabled and the request carried a valid `contextScope`.
 */
export interface RankingSignals {
  sameSession: boolean;
  sessionAffinityWeight: number;
}

export interface SearchHit {
  unit: MemoryUnit;
  /**
   * Canonical relevance in `[0, 1]`. When `enableRerank` is on (the default)
   * this is the reranker relevance score; otherwise it is the normalized cosine
   * similarity. Results are ordered by this value and `minSimilarity` filters
   * on this value.
   */
  score: number;
  /**
   * Raw normalized cosine similarity in `[0, 1]`, independent of rerank.
   * Informational only - it is neither the sort key nor what `minSimilarity`
   * compares against. Omitted when not computed.
   */
  similarity?: number;
  /**
   * Only returned when the server-side session-affinity ranking policy is
   * enabled and the request carried a valid `contextScope`.
   */
  rankingScore?: number;
  /** @see rankingScore for the conditions under which this is populated. */
  rankingSignals?: RankingSignals;
  source: string;
}

export interface MemoryStoreInfo {
  otsInstance?: string;
  memoryStoreName: string;
  description?: string;
  /** Custom extraction instructions configured on the store. */
  extractInstructions?: string;
  /**
   * Always populated on read: stores without an explicit mode report `ots`.
   * Immutable after creation.
   */
  storageMode?: MemoryStoreStorageMode | string;
  createdAt?: string;
  updatedAt?: string;
  /**
   * Declared by the service but never populated: it is always the zero value
   * `0001-01-01T00:00:00Z`. Do not surface it to users.
   */
  lastSeenAt?: string;
}

/**
 * One ListMemoryStores entry. The service returns only the store name here -
 * `description`, `storageMode` and `extractInstructions` require a per-store
 * GetMemoryStore call.
 */
export interface MemoryStoreListItem {
  memoryStoreName: string;
}

export interface SessionMessage {
  messageId?: string;
  role: string;
  speaker?: string;
  content: string;
  timestamp?: string;
  metadata?: Record<string, string>;
}

export interface MemoryStoreRequestRecord {
  requestId: string;
  operation: string;
  scope?: ScopeValue;
  targetId?: string;
  responseStatus?: number;
  latencyMs?: number;
  createdAt?: string;
  requestSummary?: string;
  responseSummary?: string;
  ingestStatus?: string;
  lastError?: string;
}

export interface CreateMemoryStoreResponse extends MemoryStoreInfo {}
export interface GetMemoryStoreResponse extends MemoryStoreInfo {}
export interface UpdateMemoryStoreResponse extends MemoryStoreInfo {}

export interface ListMemoryStoresResponse {
  stores: MemoryStoreListItem[];
  nextToken?: string;
}

export interface DeleteMemoryStoreResponse {
  deleted: boolean;
  memoryStoreName: string;
}

export interface AddMemoriesResponse {
  requestId?: string;
  status: string;
  acceptedMessages: number;
  scope?: ScopeValue;
  memoryStoreName: string;
  memcellsCreated?: number;
  unitsCreated?: number;
}

export interface SearchMemoriesResponse {
  results: SearchHit[];
  scope?: ScopeValue;
  memoryStoreName: string;
}

export interface ListMemoriesResponse {
  memories: MemoryUnit[];
  scope?: ScopeValue;
  memoryStoreName: string;
  nextToken?: string;
}

export interface GetMemoryResponse {
  memory: MemoryUnit;
  memoryStoreName: string;
}

export interface UpdateMemoryResponse {
  memory: MemoryUnit;
  updated: boolean;
  memoryStoreName: string;
}

export interface DeleteMemoryResponse {
  deleted: boolean;
  memoryId: string;
  memoryStoreName: string;
}

export interface MemoryStoreSession {
  scope: ScopeValue;
  messages: SessionMessage[];
  nextToken?: string;
}

export interface ListMemoryStoreMessagesResponse {
  session: MemoryStoreSession;
  nextToken?: string;
}

export interface ListMemoryStoreRequestsResponse {
  requests: MemoryStoreRequestRecord[];
  scope?: ScopeValue;
  memoryStoreName: string;
  nextToken?: string;
}

export class Scope implements JsonRequestConvertible<ScopeValue> {
  readonly appId?: string;
  readonly tenantId?: string;
  readonly agentId?: string;
  readonly runId?: string;

  constructor(options: ScopeValue) {
    Object.assign(this, options);
  }

  toJSONRequest(): ScopeValue {
    const result: ScopeValue = {};
    if (this.appId !== undefined) {
      result.appId = this.appId;
    }
    if (this.tenantId !== undefined) {
      result.tenantId = this.tenantId;
    }
    if (this.agentId !== undefined) {
      result.agentId = this.agentId;
    }
    if (this.runId !== undefined) {
      result.runId = this.runId;
    }
    return result;
  }
}

export class MessagePayload implements JsonRequestConvertible<MessagePayloadValue> {
  readonly role: string;
  readonly content: string;
  readonly messageId?: string;
  readonly timestamp?: string;
  readonly metadata?: Record<string, string>;

  constructor(options: MessagePayloadValue) {
    Object.assign(this, options);
    this.role = options.role;
    this.content = options.content;
  }

  toJSONRequest(): MessagePayloadValue {
    const result: MessagePayloadValue = {
      role: this.role,
      content: this.content,
    };
    if (this.messageId !== undefined) {
      result.messageId = this.messageId;
    }
    if (this.timestamp !== undefined) {
      result.timestamp = this.timestamp;
    }
    if (this.metadata !== undefined) {
      result.metadata = this.metadata;
    }
    return result;
  }
}

export class CreateMemoryStoreRequest implements JsonRequestConvertible {
  /** Required. At most 32 characters matching `^[a-zA-Z0-9_]+$`. */
  readonly memoryStoreName: string;
  /**
   * Free-form description. Limit is **1024 bytes** (UTF-8 byte length, not
   * characters) - note this differs from `extractInstructions`, which is
   * measured in code points.
   */
  readonly description?: string;
  /**
   * Custom extraction instructions. Limit is **4096 code points (runes)**, not
   * bytes - use `[...text].length` if you pre-validate on the client.
   */
  readonly extractInstructions?: string;
  /**
   * Storage mode; defaults to `ots` when omitted. **Immutable after creation** -
   * UpdateMemoryStore cannot change it.
   */
  readonly storageMode?: MemoryStoreStorageMode | string;

  constructor(options: {
    memoryStoreName: string;
    description?: string;
    extractInstructions?: string;
    storageMode?: MemoryStoreStorageMode | string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { memoryStoreName: this.memoryStoreName };
    assignIfPresent(result, 'description', this.description);
    assignIfPresent(result, 'extractInstructions', this.extractInstructions);
    assignIfPresent(result, 'storageMode', this.storageMode);
    return result;
  }
}

export class GetMemoryStoreRequest implements JsonRequestConvertible {
  constructor(readonly options: { memoryStoreName: string }) {}
  toJSONRequest() {
    return { memoryStoreName: this.options.memoryStoreName };
  }
}

export class ListMemoryStoresRequest implements JsonRequestConvertible {
  constructor(readonly options: { limit?: number; nextToken?: string } = {}) {}
  toJSONRequest() {
    const result: Record<string, unknown> = {};
    assignIfPresent(result, 'limit', this.options.limit);
    assignIfPresent(result, 'nextToken', this.options.nextToken);
    return result;
  }
}

/**
 * UpdateMemoryStore is a strict PATCH and accepts **only** `description` and
 * `extractInstructions`. Everything else (including `storageMode`) is immutable.
 *
 * Per-field semantics:
 *
 * - omitted / `undefined` / {@link UNSET} / `null` - leave the field unchanged
 * - `''` (empty string) - **clear** the stored value
 *
 * Sending neither field is a no-op: the service simply echoes the current
 * store info back.
 */
export class UpdateMemoryStoreRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  /**
   * `''` clears the description; omitting it leaves it unchanged.
   * Limit is 1024 **bytes**.
   */
  readonly description?: string | null | Unset;
  /**
   * `''` clears the instructions; omitting it leaves them unchanged.
   * Limit is 4096 **code points (runes)**.
   */
  readonly extractInstructions?: string | null | Unset;

  constructor(options: {
    memoryStoreName: string;
    description?: string | null | Unset;
    extractInstructions?: string | null | Unset;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { memoryStoreName: this.memoryStoreName };
    assignIfPresent(result, 'description', this.description);
    assignIfPresent(result, 'extractInstructions', this.extractInstructions);
    return result;
  }
}

export class DeleteMemoryStoreRequest implements JsonRequestConvertible {
  constructor(readonly options: { memoryStoreName: string }) {}
  toJSONRequest() {
    return { memoryStoreName: this.options.memoryStoreName };
  }
}

export class AddMemoriesRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly scope?: Scope | ScopeValue;
  readonly messages?: Array<MessagePayload | MessagePayloadValue>;
  readonly text?: string;
  readonly metadata?: Record<string, string>;
  readonly sync?: boolean;

  constructor(options: {
    memoryStoreName: string;
    scope?: Scope | ScopeValue;
    messages?: Array<MessagePayload | MessagePayloadValue>;
    text?: string;
    metadata?: Record<string, string>;
    sync?: boolean;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { memoryStoreName: this.memoryStoreName };
    assignIfPresent(result, 'scope', toJsonValue(this.scope));
    assignIfPresent(result, 'messages', this.messages?.map((item) => toJsonValue(item)));
    assignIfPresent(result, 'text', this.text);
    assignIfPresent(result, 'metadata', this.metadata);
    assignIfPresent(result, 'sync', this.sync);
    return result;
  }
}

export class SearchMemoriesRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  /** `appId` and `tenantId` are required; `agentId`/`runId` accept `*`. */
  readonly scope?: Scope | ScopeValue;
  readonly query: string;
  /**
   * Exact key/value equality filters, AND-ed together. The memory search API
   * supports **only** equality matching - the `MetadataFilter` tree
   * (AND/OR/NOT/range) applies to knowledge-base `retrieve`, not here.
   */
  readonly metadata?: Record<string, string>;
  /** Defaults to 10; maximum 50. Values above 50 are rejected with 400. */
  readonly topK?: number;
  /** Defaults to `true` when omitted. */
  readonly enableRerank?: boolean;
  /**
   * Optional session context. Must be a **fully specified four-part scope**
   * (no `*` wildcards) and must be contained by `scope`. Enables the
   * session-affinity ranking policy, which populates `rankingScore` /
   * `rankingSignals` on each hit when the policy is enabled server-side.
   */
  readonly contextScope?: Scope | ScopeValue;
  /**
   * Minimum relevance in `[0, 1]`. Defaults to `0`, meaning no filtering;
   * out-of-range values are rejected with 400.
   *
   * It filters on `results[].score` - the canonical relevance, which **is
   * affected by `enableRerank`** (reranker score when on, normalized cosine
   * similarity when off). It does *not* filter on `results[].similarity`,
   * which is always the raw cosine similarity.
   */
  readonly minSimilarity?: number;
  /**
   * Include supporting evidence on each hit. At most 8 evidence entries per
   * hit, each truncated to 1200 characters.
   */
  readonly includeEvidence?: boolean;

  constructor(options: {
    memoryStoreName: string;
    scope?: Scope | ScopeValue;
    query: string;
    metadata?: Record<string, string>;
    topK?: number;
    enableRerank?: boolean;
    contextScope?: Scope | ScopeValue;
    minSimilarity?: number;
    includeEvidence?: boolean;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.query = options.query;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      query: this.query,
    };
    assignIfPresent(result, 'scope', toJsonValue(this.scope));
    assignIfPresent(result, 'contextScope', toJsonValue(this.contextScope));
    assignIfPresent(result, 'metadata', this.metadata);
    assignIfPresent(result, 'topK', this.topK);
    assignIfPresent(result, 'enableRerank', this.enableRerank);
    assignIfPresent(result, 'minSimilarity', this.minSimilarity);
    assignIfPresent(result, 'includeEvidence', this.includeEvidence);
    return result;
  }
}

export class ListMemoriesRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly scope?: Scope | ScopeValue;
  readonly limit?: number;
  readonly nextToken?: string;

  constructor(options: {
    memoryStoreName: string;
    scope?: Scope | ScopeValue;
    limit?: number;
    nextToken?: string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { memoryStoreName: this.memoryStoreName };
    assignIfPresent(result, 'scope', toJsonValue(this.scope));
    assignIfPresent(result, 'limit', this.limit);
    assignIfPresent(result, 'nextToken', this.nextToken);
    return result;
  }
}

export class GetMemoryRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly memoryId: string;
  readonly scope: Scope | ScopeValue;

  constructor(options: {
    memoryStoreName: string;
    memoryId: string;
    scope: Scope | ScopeValue;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.memoryId = options.memoryId;
    this.scope = options.scope;
  }

  toJSONRequest() {
    return {
      memoryStoreName: this.memoryStoreName,
      memoryId: this.memoryId,
      scope: toJsonValue(this.scope),
    };
  }
}

export class UpdateMemoryRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly memoryId: string;
  readonly scope: Scope | ScopeValue;
  readonly text?: string;
  readonly metadata?: Record<string, string>;

  constructor(options: {
    memoryStoreName: string;
    memoryId: string;
    scope: Scope | ScopeValue;
    text?: string;
    metadata?: Record<string, string>;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.memoryId = options.memoryId;
    this.scope = options.scope;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      memoryId: this.memoryId,
      scope: toJsonValue(this.scope),
    };
    assignIfPresent(result, 'text', this.text);
    assignIfPresent(result, 'metadata', this.metadata);
    return result;
  }
}

export class DeleteMemoryRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly memoryId: string;
  readonly scope: Scope | ScopeValue;

  constructor(options: {
    memoryStoreName: string;
    memoryId: string;
    scope: Scope | ScopeValue;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.memoryId = options.memoryId;
    this.scope = options.scope;
  }

  toJSONRequest() {
    return {
      memoryStoreName: this.memoryStoreName,
      memoryId: this.memoryId,
      scope: toJsonValue(this.scope),
    };
  }
}

/**
 * Unlike the dream APIs, the message and request-audit listings accept **both**
 * Unix-millisecond values (number or numeric string) and RFC3339 timestamps.
 */
export class ListMemoryStoreMessagesRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly scope: Scope | ScopeValue;
  /** Defaults to 200; maximum 500. */
  readonly limit?: number;
  readonly nextToken?: string;
  /** Unix milliseconds (number or numeric string) or an RFC3339 timestamp. */
  readonly minTimestamp?: number | string;
  /** Unix milliseconds (number or numeric string) or an RFC3339 timestamp. */
  readonly maxTimestamp?: number | string;

  constructor(options: {
    memoryStoreName: string;
    scope: Scope | ScopeValue;
    limit?: number;
    nextToken?: string;
    minTimestamp?: number | string;
    maxTimestamp?: number | string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.scope = options.scope;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      scope: toJsonValue(this.scope),
    };
    assignIfPresent(result, 'limit', this.limit);
    assignIfPresent(result, 'nextToken', this.nextToken);
    assignIfPresent(result, 'minTimestamp', this.minTimestamp);
    assignIfPresent(result, 'maxTimestamp', this.maxTimestamp);
    return result;
  }
}

/**
 * Unlike the dream APIs, the message and request-audit listings accept **both**
 * Unix-millisecond values (number or numeric string) and RFC3339 timestamps.
 */
export class ListMemoryStoreRequestsRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly scope?: Scope | ScopeValue;
  /** Defaults to 50; maximum 100. */
  readonly limit?: number;
  readonly nextToken?: string;
  readonly operation?: string;
  /** Unix milliseconds (number or numeric string) or an RFC3339 timestamp. */
  readonly minTimestamp?: number | string;
  /** Unix milliseconds (number or numeric string) or an RFC3339 timestamp. */
  readonly maxTimestamp?: number | string;

  constructor(options: {
    memoryStoreName: string;
    scope?: Scope | ScopeValue;
    limit?: number;
    nextToken?: string;
    operation?: string;
    minTimestamp?: number | string;
    maxTimestamp?: number | string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { memoryStoreName: this.memoryStoreName };
    assignIfPresent(result, 'scope', toJsonValue(this.scope));
    assignIfPresent(result, 'limit', this.limit);
    assignIfPresent(result, 'nextToken', this.nextToken);
    assignIfPresent(result, 'operation', this.operation);
    assignIfPresent(result, 'minTimestamp', this.minTimestamp);
    assignIfPresent(result, 'maxTimestamp', this.maxTimestamp);
    return result;
  }
}

export type MemoryTaskStatus = 'queued' | 'running' | 'completed' | 'failed' | 'needs_reconcile';

export interface MemoryTask {
  requestId: string;
  eventType?: string;
  memoryStoreName?: string;
  conversationKey?: string;
  scope: ScopeValue;
  status: MemoryTaskStatus;
  /** Number of messages accepted by this ingest request. */
  acceptedMessages?: number;
  derivedMemcellId?: string;
  /**
   * IDs of the long-term memory units extracted by this task. There is **no
   * separate "extracted memory count" field** - use `derivedUnitIds.length`.
   */
  derivedUnitIds?: string[];
  lastError?: string;
  createdAt: string;
  updatedAt: string;
  finishedAt?: string;
}

export interface GetMemoryTaskResponse {
  task: MemoryTask;
  memoryStoreName: string;
}

export interface ListMemoryTasksResponse {
  tasks: MemoryTask[];
  scope: ScopeValue;
  memoryStoreName: string;
  nextToken?: string;
}

export interface ListMemoryStoreScopesResponse {
  scopes: ScopeValue[];
  scope: ScopeValue;
  memoryStoreName: string;
  nextToken?: string;
}

export type DreamTaskStatus =
  | 'queued'
  | 'running'
  | 'planning'
  | 'applying'
  | 'completed'
  | 'completed_with_failures'
  | 'failed'
  | 'cancelled';

export type DreamActionStatus = 'proposed' | 'applied' | 'skipped' | 'failed';

export type DreamApplyMode = 'proposal' | 'safe_auto';

export type DreamScopeOutputMode = 'preserve_scope' | 'promote_scope';

/**
 * Kind of dream task. Defaults to `memory` when omitted.
 *
 * `skill` and `profile` tasks emit extracts instead of memory mutations and
 * **reject `applyMode`** with a 400.
 */
export type DreamTaskType = 'memory' | 'skill' | 'profile';

/**
 * All seven action types. `EMIT_SKILL` and `EMIT_PROFILE` are produced by
 * `skill` / `profile` dream tasks and cannot be applied via
 * {@link ApplyMemoryDreamActionsRequest}.
 */
export type DreamActionType =
  | 'ADD'
  | 'UPDATE'
  | 'DELETE'
  | 'MERGE'
  | 'NOOP'
  | 'EMIT_SKILL'
  | 'EMIT_PROFILE';

/** Action types that {@link ListMemoryDreamActionsRequest} accepts in scope mode. */
export type DreamExtractActionType = Extract<DreamActionType, 'EMIT_SKILL' | 'EMIT_PROFILE'>;

export type DreamSkippedReason =
  | 'stale_target_memory'
  | 'task_cancelled'
  | 'invalid_action'
  | 'scope_violation'
  | 'missing_source'
  | 'delete_requires_manual_apply'
  | 'promote_requires_manual_apply'
  | 'confidence_below_threshold'
  | 'superseded_by_skill'
  | 'superseded_by_profile'
  /** NOOP confirmed in safe_auto that no change was needed - a success, not a rejection. */
  | 'noop_nothing_to_apply'
  /** The action's write set overlaps an earlier action in the same plan. */
  | 'conflicting_action';

export type DreamActionListOrderBy = 'created_at_asc' | 'created_at_desc' | 'confidence_desc';

/**
 * The only threshold keys the service accepts. Matching is case-insensitive.
 * `delete` is explicitly rejected with 400 - DELETE actions never auto-apply.
 */
export type DreamConfidenceThresholdKey = 'add' | 'update' | 'merge';

/**
 * Per-action confidence thresholds, each in `[0, 1]`. Only `add`, `update` and
 * `merge` are accepted; any other key (notably `delete`) is rejected with 400.
 * Service defaults: ADD 0.80, UPDATE 0.85, MERGE 0.70.
 *
 * The `Record<string, number>` arm is retained so dynamically built maps keep
 * type-checking; prefer the literal-key form for editor completion.
 */
export type DreamConfidenceThresholds =
  | Partial<Record<DreamConfidenceThresholdKey, number>>
  | Record<string, number>;

/** Body of the memory an ADD/UPDATE/MERGE action proposes to write. */
export interface DreamMemorySpec {
  text: string;
  title?: string;
  searchText?: string;
  keywords?: string[];
  unitType: string;
  metadata?: Record<string, string>;
  effectiveAt?: string;
  timePrecision?: string;
}

/** Body of the extract an `EMIT_SKILL` / `EMIT_PROFILE` action produces. */
export interface DreamExtractSpec {
  kind: 'skill' | 'profile' | string;
  name: string;
  description: string;
  instructions: string;
}

export interface DreamActionRecord {
  dreamId: string;
  actionId: string;
  memoryStoreName: string;
  action: DreamActionType;
  status: DreamActionStatus;
  targetScope: ScopeValue;
  sourceScopes?: ScopeValue[];
  targetMemoryId?: string;
  targetMemoryVersion?: number;
  targetMemoryUpdatedAtMs?: number;
  newMemory?: DreamMemorySpec;
  /** Present on `EMIT_SKILL` / `EMIT_PROFILE` actions instead of `newMemory`. */
  newExtract?: DreamExtractSpec;
  sourceMessageIds?: string[];
  sourceMemoryIds?: string[];
  reason?: string;
  confidence?: number;
  resultMemoryId?: string;
  applyStep?: string;
  applyDetailJson?: string;
  skippedReason?: DreamSkippedReason;
  skippedReasonDetail?: string;
  appliedAt?: string;
  applier?: string;
  attemptCount?: number;
  error?: string;
  createdAt: string;
  updatedAt: string;
}

export interface DreamTask {
  dreamId: string;
  memoryStoreName: string;
  status: DreamTaskStatus;
  /** Defaults to `memory` on rows written before task types existed. */
  taskType?: DreamTaskType;
  applyMode: DreamApplyMode;
  scopeOutputMode: DreamScopeOutputMode;
  clientToken?: string;
  triggerType?: string;
  expandedScopeLimit?: number;
  inputScopes?: ScopeValue[];
  expandedScopes?: ScopeValue[];
  targetScopes?: ScopeValue[];
  /** Unix milliseconds. */
  minTimestamp?: number;
  /** Unix milliseconds. */
  maxTimestamp?: number;
  /** Whether this run resumed from the previous run's watermark. */
  incremental?: boolean;
  /**
   * Highest message timestamp (Unix milliseconds) this run actually processed.
   * The next incremental run for the same `(scope, taskType)` starts at
   * `processedThroughMs + 1`. Zero means nothing was processed.
   */
  processedThroughMs?: number;
  maxSessions?: number;
  maxMessages?: number;
  maxMemories?: number;
  /**
   * Effective thresholds for this task. Left as `Record<string, number>` rather
   * than {@link DreamConfidenceThresholds} so existing readers keep compiling;
   * in practice the keys are only `add`, `update` and `merge`.
   */
  confidenceThresholds?: Record<string, number>;
  instructions?: string;
  sessionCount?: number;
  messageCount?: number;
  memoryCount?: number;
  actionCount?: number;
  proposedCount?: number;
  appliedCount?: number;
  skippedCount?: number;
  /** Breakdown of `skippedCount` by reason. Absent when nothing was skipped. */
  skippedByReason?: Partial<Record<DreamSkippedReason, number>>;
  failedCount?: number;
  lastError?: string;
  version?: number;
  createdAt: string;
  updatedAt: string;
  finishedAt?: string;
}

export interface DreamTaskInputSummary {
  scopes: ScopeValue[];
  /** Unix milliseconds. */
  minTimestamp: number;
  /** Unix milliseconds. */
  maxTimestamp: number;
  /** Whether this run resumed from the previous run's watermark. */
  incremental?: boolean;
  sessionCount: number;
  messageCount: number;
  memoryCount: number;
}

export interface DreamTaskActionStats {
  total: number;
  proposed: number;
  applied: number;
  skipped: number;
  failed: number;
  /** Breakdown of `skipped` by reason. Absent when nothing was skipped. */
  skippedByReason?: Partial<Record<DreamSkippedReason, number>>;
}

export interface DreamTaskSummary {
  memoryStoreName: string;
  dreamId: string;
  status: DreamTaskStatus;
  /** Always populated; legacy rows are reported as `memory`. */
  taskType?: DreamTaskType;
  applyMode: DreamApplyMode;
  scopeOutputMode: DreamScopeOutputMode;
  input: DreamTaskInputSummary;
  /** Highest message timestamp (Unix milliseconds) processed by this run. */
  processedThroughMs?: number;
  actions: DreamTaskActionStats;
  /** Present only when the task carried non-empty instructions. */
  instructions?: string;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
  finishedAt?: string | null;
}

export interface CreateMemoryDreamTaskResponse {
  memoryStoreName: string;
  dreamId: string;
  status: DreamTaskStatus;
  createdAt: string;
}

export interface GetMemoryDreamTaskResponse extends DreamTaskSummary {}

export interface ListMemoryDreamTasksResponse {
  memoryStoreName: string;
  scope: ScopeValue;
  tasks: DreamTask[];
  nextToken?: string;
}

export interface CancelMemoryDreamTaskResponse {
  memoryStoreName: string;
  dreamId: string;
  status: DreamTaskStatus;
  /** Always populated; legacy rows are reported as `memory`. */
  taskType?: DreamTaskType;
}

export interface ListMemoryDreamActionsResponse {
  memoryStoreName: string;
  /**
   * Echoed back in `dreamId` mode only. In `scope` + `actionType` mode the
   * service **omits this key entirely**, so it is `undefined` at runtime even
   * though it is typed as required for backwards compatibility.
   */
  dreamId: string;
  /** Always an array; never `null`. */
  actions: DreamActionRecord[];
  nextToken?: string;
}

export interface DreamApplyActionResult {
  actionId: string;
  /**
   * In addition to the four {@link DreamActionStatus} values, the service
   * reports `not_found` for action IDs that do not exist in the dream.
   */
  status: DreamActionStatus | 'not_found';
  memoryId?: string;
  error?: string;
}

export interface ApplyMemoryDreamActionsResponse {
  memoryStoreName: string;
  dreamId: string;
  applied: number;
  failed: number;
  notFound?: number;
  results: DreamApplyActionResult[];
}

export class GetMemoryTaskRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly requestId: string;
  /**
   * Optional. When omitted no ownership check is performed. When supplied it is
   * compared against the task's own scope and a mismatch is rejected with 400.
   * Hierarchical `*` wildcards are allowed.
   */
  readonly scope?: Scope | ScopeValue;

  constructor(options: { memoryStoreName: string; requestId: string; scope?: Scope | ScopeValue }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.requestId = options.requestId;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      requestId: this.requestId,
    };
    assignIfPresent(result, 'scope', toJsonValue(this.scope));
    return result;
  }
}

export class ListMemoryTasksRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  /**
   * **Careful:** this scope is a value, not an optional filter. Any segment you
   * leave empty is filled in with the literal `__default__`, so an empty scope
   * means "the `__default__/__default__/__default__/__default__` scope", **not**
   * "all scopes". Use `*` explicitly to match a whole level.
   */
  readonly scope: Scope | ScopeValue;
  readonly status?: MemoryTaskStatus;
  /** Defaults to 50; maximum 100. */
  readonly limit?: number;
  readonly nextToken?: string;
  /** Unix **milliseconds** only (number or numeric string). RFC3339 is rejected. */
  readonly minTimestamp?: number | string;
  /** Unix **milliseconds** only (number or numeric string). RFC3339 is rejected. */
  readonly maxTimestamp?: number | string;

  constructor(options: {
    memoryStoreName: string;
    scope: Scope | ScopeValue;
    status?: MemoryTaskStatus;
    limit?: number;
    nextToken?: string;
    minTimestamp?: number | string;
    maxTimestamp?: number | string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.scope = options.scope;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      scope: toJsonValue(this.scope),
    };
    assignIfPresent(result, 'status', this.status);
    assignIfPresent(result, 'limit', this.limit);
    assignIfPresent(result, 'nextToken', this.nextToken);
    assignIfPresent(result, 'minTimestamp', this.minTimestamp);
    assignIfPresent(result, 'maxTimestamp', this.maxTimestamp);
    return result;
  }
}

export class ListMemoryStoreScopesRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly scope: Scope | ScopeValue;
  readonly limit?: number;
  readonly nextToken?: string;

  constructor(options: {
    memoryStoreName: string;
    scope: Scope | ScopeValue;
    limit?: number;
    nextToken?: string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.scope = options.scope;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      scope: toJsonValue(this.scope),
    };
    assignIfPresent(result, 'limit', this.limit);
    assignIfPresent(result, 'nextToken', this.nextToken);
    return result;
  }
}

export class CreateMemoryDreamTaskRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  /** Required, 1..20 entries. Hierarchical `*` wildcards are allowed. */
  readonly scopes: Array<Scope | ScopeValue>;
  /**
   * Unix **milliseconds** only (number or numeric string). The dream APIs do
   * **not** accept RFC3339 - unlike `ListMemoryStoreMessages` /
   * `ListMemoryStoreRequests`. Negative values and `min > max` are rejected
   * with 400.
   */
  readonly minTimestamp?: number | string;
  /** Unix **milliseconds** only. See {@link minTimestamp}. */
  readonly maxTimestamp?: number | string;
  /** `0..100`; defaults to 100. */
  readonly maxSessions?: number;
  /** `0..20000`; defaults to 5000. */
  readonly maxMessages?: number;
  /** `0..5000`; defaults to 1000. */
  readonly maxMemories?: number;
  /**
   * Cap on how many concrete scopes a wildcard in `scopes` may expand to.
   * `0..1000`; defaults to 200.
   */
  readonly expandedScopeLimit?: number;
  /** Defaults to `preserve_scope`. */
  readonly scopeOutputMode?: DreamScopeOutputMode;
  /**
   * Defaults to `proposal`. **Only valid when `taskType` is `memory`** -
   * sending it with `skill` or `profile` is rejected with 400.
   */
  readonly applyMode?: DreamApplyMode;
  /** Defaults to `memory`. See {@link DreamTaskType}. */
  readonly taskType?: DreamTaskType;
  /** See {@link DreamConfidenceThresholds}; `delete` is rejected with 400. */
  readonly confidenceThresholds?: DreamConfidenceThresholds;
  /** At most **4000 bytes** (UTF-8 byte length, not characters). */
  readonly instructions?: string;
  /**
   * Idempotency key with a **7 day** window: within that window the same token
   * resolves to the same `dreamId` instead of starting a new run.
   */
  readonly clientToken?: string;
  /**
   * Resume from the previous successful run's `processedThroughMs` watermark
   * for the same `(scope, taskType)`. Defaults to `false` (full reprocess) for
   * manually triggered runs. An explicit `minTimestamp` always wins.
   */
  readonly incremental?: boolean;

  constructor(options: {
    memoryStoreName: string;
    scopes: Array<Scope | ScopeValue>;
    minTimestamp?: number | string;
    maxTimestamp?: number | string;
    maxSessions?: number;
    maxMessages?: number;
    maxMemories?: number;
    expandedScopeLimit?: number;
    scopeOutputMode?: DreamScopeOutputMode;
    applyMode?: DreamApplyMode;
    taskType?: DreamTaskType;
    confidenceThresholds?: DreamConfidenceThresholds;
    instructions?: string;
    clientToken?: string;
    incremental?: boolean;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.scopes = options.scopes;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      scopes: this.scopes.map((item) => toJsonValue(item)),
    };
    assignIfPresent(result, 'minTimestamp', this.minTimestamp);
    assignIfPresent(result, 'maxTimestamp', this.maxTimestamp);
    assignIfPresent(result, 'maxSessions', this.maxSessions);
    assignIfPresent(result, 'maxMessages', this.maxMessages);
    assignIfPresent(result, 'maxMemories', this.maxMemories);
    assignIfPresent(result, 'expandedScopeLimit', this.expandedScopeLimit);
    assignIfPresent(result, 'scopeOutputMode', this.scopeOutputMode);
    assignIfPresent(result, 'applyMode', this.applyMode);
    assignIfPresent(result, 'taskType', this.taskType);
    assignIfPresent(result, 'confidenceThresholds', this.confidenceThresholds);
    assignIfPresent(result, 'instructions', this.instructions);
    assignIfPresent(result, 'clientToken', this.clientToken);
    assignIfPresent(result, 'incremental', this.incremental);
    return result;
  }
}

export class GetMemoryDreamTaskRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly dreamId: string;

  constructor(options: { memoryStoreName: string; dreamId: string }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.dreamId = options.dreamId;
  }

  toJSONRequest() {
    return {
      memoryStoreName: this.memoryStoreName,
      dreamId: this.dreamId,
    };
  }
}

export class ListMemoryDreamTasksRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly scope: Scope | ScopeValue;
  readonly status?: DreamTaskStatus;
  /** Unix **milliseconds** only (number or numeric string). RFC3339 is rejected. */
  readonly minTimestamp?: number | string;
  /** Unix **milliseconds** only (number or numeric string). RFC3339 is rejected. */
  readonly maxTimestamp?: number | string;
  /** Defaults to 50; maximum 100. */
  readonly limit?: number;
  readonly nextToken?: string;

  constructor(options: {
    memoryStoreName: string;
    scope: Scope | ScopeValue;
    status?: DreamTaskStatus;
    minTimestamp?: number | string;
    maxTimestamp?: number | string;
    limit?: number;
    nextToken?: string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.scope = options.scope;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      scope: toJsonValue(this.scope),
    };
    assignIfPresent(result, 'status', this.status);
    assignIfPresent(result, 'minTimestamp', this.minTimestamp);
    assignIfPresent(result, 'maxTimestamp', this.maxTimestamp);
    assignIfPresent(result, 'limit', this.limit);
    assignIfPresent(result, 'nextToken', this.nextToken);
    return result;
  }
}

export class CancelMemoryDreamTaskRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly dreamId: string;

  constructor(options: { memoryStoreName: string; dreamId: string }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.dreamId = options.dreamId;
  }

  toJSONRequest() {
    return {
      memoryStoreName: this.memoryStoreName,
      dreamId: this.dreamId,
    };
  }
}

/**
 * ListMemoryDreamActions has **two mutually exclusive query modes**. Exactly
 * one of `dreamId` and `scope` must be supplied; sending both, or neither, is
 * rejected with 400.
 *
 * **1. `dreamId` mode** - list the actions of one dream run. All filters
 * (`status`, `action`, `targetScope`, `sourceMemoryId`, `minConfidence`,
 * `maxConfidence`, `orderBy`) apply, and the response echoes `dreamId`.
 *
 * **2. `scope` + `actionType` mode** - list emitted extracts across dream runs.
 * `actionType` is required here and may only be `EMIT_SKILL` or `EMIT_PROFILE`.
 * Caveats:
 *
 * - `action`, `targetScope`, `sourceMemoryId`, `minConfidence`, `maxConfidence`
 *   and `orderBy` are **validated but silently ignored**; ordering is fixed.
 * - `status`, `limit` and `nextToken` still apply.
 * - The response **omits the `dreamId` key**.
 * - `scope` matches each action's `targetScope` and accepts `*` wildcards.
 *
 * The fields stay individually optional (rather than a discriminated union) so
 * that existing `dreamId`-mode callers keep compiling unchanged; the mode is
 * enforced by the service, not by the type system.
 */
export class ListMemoryDreamActionsRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  /** Required in `dreamId` mode; must be omitted in `scope` mode. */
  readonly dreamId?: string;
  /** Required in `scope` mode; must be omitted in `dreamId` mode. */
  readonly scope?: Scope | ScopeValue;
  /**
   * Required in `scope` mode and only valid there. Only `EMIT_SKILL` and
   * `EMIT_PROFILE` are supported; anything else is rejected with 400.
   */
  readonly actionType?: DreamExtractActionType;
  readonly status?: DreamActionStatus;
  /** Accepts all seven action types. Ignored in `scope` mode. */
  readonly action?: DreamActionType;
  /** Ignored in `scope` mode. */
  readonly targetScope?: Scope | ScopeValue;
  /** Ignored in `scope` mode. */
  readonly sourceMemoryId?: string;
  /** `0..1` and `<= maxConfidence`. Ignored in `scope` mode. */
  readonly minConfidence?: number;
  /** `0..1` and `>= minConfidence`. Ignored in `scope` mode. */
  readonly maxConfidence?: number;
  /** Defaults to `created_at_asc`. Ignored in `scope` mode. */
  readonly orderBy?: DreamActionListOrderBy;
  /** Defaults to 100; maximum 100. */
  readonly limit?: number;
  readonly nextToken?: string;

  constructor(options: {
    memoryStoreName: string;
    dreamId?: string;
    scope?: Scope | ScopeValue;
    actionType?: DreamExtractActionType;
    status?: DreamActionStatus;
    action?: DreamActionType;
    targetScope?: Scope | ScopeValue;
    sourceMemoryId?: string;
    minConfidence?: number;
    maxConfidence?: number;
    orderBy?: DreamActionListOrderBy;
    limit?: number;
    nextToken?: string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
    };
    assignIfPresent(result, 'dreamId', this.dreamId);
    assignIfPresent(result, 'scope', toJsonValue(this.scope));
    assignIfPresent(result, 'actionType', this.actionType);
    assignIfPresent(result, 'status', this.status);
    assignIfPresent(result, 'action', this.action);
    assignIfPresent(result, 'targetScope', toJsonValue(this.targetScope));
    assignIfPresent(result, 'sourceMemoryId', this.sourceMemoryId);
    assignIfPresent(result, 'minConfidence', this.minConfidence);
    assignIfPresent(result, 'maxConfidence', this.maxConfidence);
    assignIfPresent(result, 'orderBy', this.orderBy);
    assignIfPresent(result, 'limit', this.limit);
    assignIfPresent(result, 'nextToken', this.nextToken);
    return result;
  }
}

/**
 * Applies proposed dream actions. `EMIT_SKILL` and `EMIT_PROFILE` actions
 * cannot be applied: including even one of their IDs rejects the **whole**
 * request with 400 before any action is applied, so there is no partial write.
 */
export class ApplyMemoryDreamActionsRequest implements JsonRequestConvertible {
  readonly memoryStoreName: string;
  readonly dreamId: string;
  /** Required and non-empty; at most **100** IDs per call. */
  readonly actionIds: string[];
  /**
   * Free-form audit label recorded on each action actually applied by this
   * call. The service neither validates nor generates it, and it is not
   * rewritten on actions that were already applied.
   */
  readonly applier?: string;

  constructor(options: {
    memoryStoreName: string;
    dreamId: string;
    actionIds: string[];
    applier?: string;
  }) {
    Object.assign(this, options);
    this.memoryStoreName = options.memoryStoreName;
    this.dreamId = options.dreamId;
    this.actionIds = options.actionIds;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      memoryStoreName: this.memoryStoreName,
      dreamId: this.dreamId,
      actionIds: this.actionIds,
    };
    assignIfPresent(result, 'applier', this.applier);
    return result;
  }
}
