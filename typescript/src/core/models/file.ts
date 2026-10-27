import type { ScopeValue } from './memory.js';

/** Read-only projected-file storage mode. */
export const STORAGE_MODE_FILE_OTS = 'file+ots';

/** Writable file-memory storage mode. */
export const STORAGE_MODE_FILE_MEMORY = 'filemem';

/**
 * The item type accepted by the item APIs. The SDK fills it into any item
 * request whose `type` field is missing or empty.
 */
export const ITEM_TYPE_MEMORY_FILE = 'memoryfile';

/**
 * Directory-marker type emitted by ListItems when `depth: 1` rolls up deeper
 * subtrees. Prefix entries carry only `scope` and `path`.
 */
export const ITEM_TYPE_MEMORY_FILE_PREFIX = 'memoryfile_prefix';

/** Maximum item content size accepted by the service, in bytes (100 KiB). */
export const MAX_ITEM_CONTENT_BYTES = 102400;

/** Maximum normalized item path length, in bytes. */
export const MAX_ITEM_PATH_BYTES = 1024;

/**
 * Conventional per-mount-unit (store + exact scope) file-count cap. Whether
 * the service enforces it is deployment configuration; when enforced,
 * exceeding it fails with 409 `QUOTA_EXCEEDED`.
 */
export const MAX_ITEMS_PER_SCOPE = 2000;

export type ItemOperation = 'created' | 'modified' | 'deleted';

/**
 * Payload view selector: `'basic'` (the default when omitted) omits
 * `content`; `'full'` includes it.
 */
export type ItemView = 'basic' | 'full';

/**
 * Typed actor attribution on version rows. `actorType` is `'session'`,
 * `'api_key'` or `'user'`; `actorId` is the corresponding identifier.
 */
export interface ItemActor {
  actorType: string;
  actorId: string;
}

/**
 * Fields shared by all eight item actions: `AddItem`, `ListItems`, `GetItem`,
 * `UpdateItem`, `DeleteItem`, `ListItemVersions`, `GetItemVersion` and
 * `RedactItemVersion`.
 *
 * Item APIs require a writable `filemem` store. On a `file+ots` read-only
 * projection, writes fail with 409 `READ_ONLY_STORE` and the version APIs fail
 * with 400; on a plain `ots` store every item action fails with 400.
 *
 * Scope rules per action family:
 *
 * - `AddItem`, `GetItem`, `UpdateItem` and single `DeleteItem` require all four
 *   segments (`appId`, `tenantId`, `agentId`, `runId`) **exact** - the `*`
 *   wildcard is rejected with 400.
 * - Listing actions (`ListItems`, `ListItemVersions`) and ranged `DeleteItem`
 *   allow the left-prefix wildcard: `appId`/`tenantId` are required and
 *   exact, `agentId`/`runId` may be `'*'`, and once a segment is `'*'` every
 *   segment after it must be `'*'` too (or omitted).
 * - `GetItemVersion` and `RedactItemVersion` require all four segments
 *   **exact**, like the write actions: the server derives item ownership
 *   from the `versionId` and verifies it against the exact scope.
 *
 * Each scope field is capped at 128 bytes and must not contain `/` or NUL.
 */
export interface ItemScopedRequest {
  /**
   * Item type discriminator. Required by the service on all eight actions, and
   * `memoryfile` is its only accepted value - the SDK fills it in when the
   * caller leaves it missing or empty.
   */
  type?: string;
  memoryStoreName: string;
  scope: ScopeValue;
}

/**
 * Path rules, enforced by the service:
 *
 * - must start with `/` - the service no longer adds a missing leading `/`
 * - at most 1024 **bytes** after NFC normalization
 * - `.` and `..` segments, empty segments (`//`), a bare `/`, a trailing `/`,
 *   NUL characters and invalid UTF-8 are all rejected with 400
 * - any valid UTF-8 is otherwise allowed, including CJK; matching is
 *   byte-exact and case-sensitive
 */
export interface AddItemRequest extends ItemScopedRequest {
  path: string;
  /**
   * UTF-8 text, at most {@link MAX_ITEM_CONTENT_BYTES} bytes - oversized
   * content fails with 413 `PAYLOAD_TOO_LARGE` and invalid UTF-8 with 400.
   * Required; an empty string creates an empty file.
   *
   * Adding an item beyond {@link MAX_ITEMS_PER_SCOPE} files in the same scope
   * fails with 409 `QUOTA_EXCEEDED` when the deployment enforces the quota.
   */
  content: string;
  /** Recorded as the version's `session` actor. */
  sessionId?: string;
  /** Defaults to `'basic'` (response without content); `'full'` echoes content. */
  view?: ItemView;
}

/**
 * Request for ListItems. The scope may use the left-prefix wildcard (see
 * {@link ItemScopedRequest}).
 */
export interface ListItemsRequest extends ItemScopedRequest {
  /**
   * Directory-style prefix. Must be segment-aligned: start **and end** with
   * `/` (a bare `/` lists the whole tree, same as omitting it). `.`/`..`
   * segments, NUL and control/format characters are rejected with 400.
   */
  pathPrefix?: string;
  /**
   * `0` (default) lists the whole subtree recursively; `1` lists only direct
   * children and rolls deeper directories up into entries typed
   * `memoryfile_prefix`. Any other value is rejected with 400.
   */
  depth?: 0 | 1;
  /** Defaults to `'basic'`; `'full'` includes each file's content. */
  view?: ItemView;
  /** Defaults to 20; maximum 100, and capped at 20 when `view: 'full'`. */
  limit?: number;
  /**
   * Opaque cursor. It must be replayed with the **same `pathPrefix`** that
   * produced it; a cursor outside that prefix is rejected with 400.
   */
  nextToken?: string;
}

/**
 * Request for GetItem. Exactly one of `itemId` and `path` is required;
 * providing neither or both is rejected by the service.
 */
export interface GetItemRequest extends ItemScopedRequest {
  itemId?: string;
  path?: string;
  /**
   * Defaults to `'basic'` - metadata only. `'full'` includes content.
   * Replaces the removed `includeContent` flag; note the default is inverted
   * relative to the old `includeContent: true` default.
   */
  view?: ItemView;
}

/**
 * Complete resource-version precondition shared by exact Update and Delete.
 * The expected version changes on both content updates and renames.
 */
export interface ItemPrecondition {
  expectedVersionId: string;
}

/**
 * Request for UpdateItem. `itemId` is the only address and is required.
 * At least one of `content` (rewrite) and `path` (new target path) is
 * required; both may be supplied for an atomic composite update.
 */
export interface UpdateItemRequest extends ItemScopedRequest {
  itemId: string;
  /** See {@link AddItemRequest.content} for size and encoding limits. */
  content?: string;
  /** New target path. Same path rules as {@link AddItemRequest}; must not be blank. */
  path?: string;
  /**
   * When the target `path` is occupied, clobber the occupier. Rejected unless
   * `path` is present.
   */
  overwrite?: boolean;
  /**
   * Optional compare-and-swap guard on the complete resource version.
   */
  precondition?: ItemPrecondition;
  /** Recorded as the new version's `session` actor. */
  sessionId?: string;
  /** Defaults to `'basic'` (response without content); `'full'` echoes content. */
  view?: ItemView;
}

/**
 * Request for DeleteItem, in one of two modes:
 *
 * - **Single delete**: exact full scope plus exactly one of `itemId` or
 *   `path`; optional `precondition`. Synchronous; the response carries
 *   `deleted: true`.
 * - **Ranged delete**: wildcard scope (left-prefix rule, `appId`/`tenantId`
 *   required). Asynchronous - the service answers **200** with a `taskId`
 *   identifying the sharded delete chain (it used to answer 202; the accept
 *   status moved to 200 because the OTS gateway only forwards 200
 *   responses). `itemId`, `path` and `precondition` must all be omitted;
 *   `pathPrefix` optionally narrows the range (segment-aligned, see
 *   {@link ListItemsRequest.pathPrefix}).
 */
export interface DeleteItemRequest extends ItemScopedRequest {
  itemId?: string;
  path?: string;
  /** Ranged delete only: optional segment-aligned narrowing of the scope. */
  pathPrefix?: string;
  /** Exact single delete only; rejected on ranged deletes. */
  precondition?: ItemPrecondition;
  /** Recorded as the delete's `session` actor. */
  sessionId?: string;
}

/**
 * Request for ListItemVersions - a scope-level version listing ordered by
 * `createdAt` descending. All filters are optional; the scope may use the
 * left-prefix wildcard (see {@link ItemScopedRequest}).
 */
export interface ListItemVersionsRequest extends ItemScopedRequest {
  /** Optional filter; when omitted, versions of every item in scope match. */
  itemId?: string;
  /** Actor filter; mutually exclusive with `apiKeyId`. */
  sessionId?: string;
  /**
   * Actor filter; mutually exclusive with `sessionId`. Note: the service
   * currently rejects a non-empty `apiKeyId` with 400 (api-key actors
   * cannot be derived until authentication context is wired into item
   * mutations); the field is kept for wire compatibility.
   */
  apiKeyId?: string;
  /** Inclusive RFC3339 lower bound on `createdAt`. */
  createdAtGte?: string;
  /** Inclusive RFC3339 upper bound on `createdAt`. */
  createdAtLte?: string;
  /** Case-sensitive; any other value is rejected with 400. */
  operation?: ItemOperation;
  /** Defaults to `'basic'`; `'full'` includes each version's content. */
  view?: ItemView;
  /** Defaults to 20; maximum 100, and capped at 20 when `view: 'full'`. */
  limit?: number;
  nextToken?: string;
}

/**
 * Request for GetItemVersion. `versionId` is self-contained: the server
 * derives all addressing from it and verifies ownership against the exact
 * four-part scope (wildcards are rejected).
 */
export interface GetItemVersionRequest extends ItemScopedRequest {
  versionId: string;
  /** Defaults to `'basic'`; `'full'` includes content (never on redacted versions). */
  view?: ItemView;
}

/**
 * Irreversibly redacts a single historical version snapshot; the scope must
 * be the exact four-part scope owning the version (wildcards are rejected).
 * `content`, `path`, `contentSha256` and `contentSizeBytes` are cleared
 * while `operation`, `createdAt` and the original `createdBy` are kept. The
 * current item state and `latestSeq` are untouched and no new version row
 * is created.
 *
 * Redaction is idempotent and first-wins: repeating it returns 200 with the
 * existing `redactedAt` / `redactedBy`. Afterwards `GetItemVersion` still
 * returns 200 for the version but never returns its `content`.
 */
export interface RedactItemVersionRequest extends ItemScopedRequest {
  versionId: string;
  /** Recorded as the `session` actor in `redactedBy` on the first redaction. */
  sessionId?: string;
}

/**
 * Current state of one item. `content` appears only when the request used
 * `view: 'full'`. Single-item responses echo `type: 'memoryfile'` at the top
 * level.
 */
export interface Item {
  type?: string;
  itemId: string;
  /** Always echoed. */
  scope: ScopeValue;
  path: string;
  content?: string;
  contentSha256: string;
  contentSizeBytes: number;
  latestSeq: number;
  /** Head-version pointer. */
  latestVersionId: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * One ListItems entry for a file (`type: 'memoryfile'`). Optional fields may
 * be omitted for empty values; `content` appears only with `view: 'full'`.
 */
export interface ItemListFileEntry {
  type: string;
  itemId: string;
  scope: ScopeValue;
  path: string;
  content?: string;
  contentSha256?: string;
  contentSizeBytes?: number;
  latestSeq?: number;
  latestVersionId?: string;
  createdAt?: string;
  updatedAt?: string;
}

/**
 * One ListItems entry for a rolled-up directory (`type:
 * 'memoryfile_prefix'`), emitted only with `depth: 1`. Carries just `scope`
 * and `path`.
 */
export interface ItemListPrefixEntry {
  type: string;
  scope: ScopeValue;
  path: string;
}

export type ItemListEntry = ItemListFileEntry | ItemListPrefixEntry;

/**
 * One immutable item version. `content` appears only with `view: 'full'`
 * and never on redacted versions. The redaction fields appear only after
 * RedactItemVersion.
 */
export interface ItemVersion {
  type?: string;
  versionId: string;
  itemId: string;
  /** Always echoed. */
  scope: ScopeValue;
  operation: ItemOperation;
  path: string;
  content?: string;
  contentSha256: string;
  contentSizeBytes: number;
  /** Typed actor that wrote this version. */
  createdBy?: ItemActor;
  createdAt: string;
  /** Present only after {@link RedactItemVersionRequest}. */
  redacted?: boolean;
  /** Time of the **first** redaction. Present only once redacted. */
  redactedAt?: string;
  /** Typed actor of the **first** redaction. Present only once redacted. */
  redactedBy?: ItemActor;
}

export type AddItemResponse = Item;
export type GetItemResponse = Item;
export type UpdateItemResponse = Item;
export type GetItemVersionResponse = ItemVersion;
export type RedactItemVersionResponse = ItemVersion;

/** Response of a synchronous single DeleteItem (200). */
export interface DeleteItemSingleResponse {
  type: string;
  itemId: string;
  deleted: true;
}

/**
 * Response of an asynchronous ranged DeleteItem (HTTP 200). `taskId`
 * identifies the sharded delete chain.
 */
export interface DeleteItemRangedResponse {
  type: string;
  taskId: string;
  status: 'pending';
}

export type DeleteItemResponse = DeleteItemSingleResponse | DeleteItemRangedResponse;

export interface ListItemsResponse {
  type: string;
  items: ItemListEntry[];
  nextToken?: string;
  /**
   * Emitted as `true` only for `file+ots` read-only projections; the key is
   * omitted entirely for writable `filemem` stores, so it is never `false`.
   * This is the only place `readOnly` appears - `Item` and `ItemVersion` do not
   * carry it.
   */
  readOnly?: boolean;
}

export interface ListItemVersionsResponse {
  type: string;
  versions: ItemVersion[];
  nextToken?: string;
}
