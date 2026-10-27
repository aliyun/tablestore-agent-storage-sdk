# Changelog

All notable changes to `@tablestore/agent-storage` are documented here.

## 0.0.13

Breaking realignment of the eight item (file-memory) APIs with the service
contract (Claude-aligned). The wire format changed; there is no forward
compatibility with 0.0.12 item payloads. Runtime behaviour is unchanged -
requests are still passed through as-is and every 2xx is accepted (ranged
`deleteItem` now answers HTTP 200).

### Changed

- All item requests: `view` (`'basic' | 'full'`) replaces `includeContent`.
  The default is `'basic'`, so `getItem` no longer returns `content` unless
  `view: 'full'` is passed (the old default included content).
- `getItem` and exact `deleteItem`: exactly one of `itemId` or `path` is
  required; providing both is invalid. Both modes require a fully specified
  four-part scope.
- `addItem`: paths must start with `/` (the service no longer adds it); the
  path limit is 1024 bytes (was 800).
- `updateItem`: a full four-part scope is required, and required `itemId` is
  the only source address. At least one of `content` or `path` is required and
  both may be combined atomically; `path` is the target path. `newPath` is
  removed, as is the empty-mutation no-op. Optimistic concurrency now uses
  `precondition?: ItemPrecondition` with `expectedVersionId` instead of
  `expectedSha256`; `overwrite` applies only when a target `path` is present,
  while `sessionId` and `view` remain optional.
- `deleteItem`: wildcard scopes trigger an asynchronous ranged delete,
  answered immediately with **HTTP 200**
  `{ type, taskId, status: 'pending' }`; optional segment-aligned `pathPrefix`
  and `sessionId` narrow/annotate the range, while `itemId`, `path`, and
  `precondition` are forbidden. Exact deletes accept optional
  `precondition?: ItemPrecondition` (`expectedVersionId`) and `sessionId`, and
  return `{ type, itemId, deleted: true }`.
- `listItems`: new `depth` (0/1; depth 1 rolls deeper directories up into
  `memoryfile_prefix` entries) and `view`; `limit` defaults to 20, max 100
  (was 100/500), capped at 20 with `view: 'full'`; `pathPrefix` must start
  and end with `/`.
- `listItemVersions`: scope-level listing — `itemId` is now an optional
  filter, `path` is removed, and `sessionId`/`apiKeyId` (mutually exclusive),
  `createdAtGte`/`createdAtLte` (RFC3339) and `view` filters are added.
- `getItemVersion` / `redactItemVersion`: addressed by the self-contained
  `versionId` only; `itemId` and `versionSeq` are removed. `getItemVersion`
  gains `view`.
- `Item`: adds `scope` (always echoed) and `latestVersionId`.
- `ItemVersion`: removes `versionSeq` and `sessionId`; adds `scope` and typed
  actor `createdBy`; `redactedBy` is now a typed actor object
  (`{ actorType, actorId }`) instead of a string.
- `listItems`, `listItemVersions`, and ranged `deleteItem` may use the
  left-prefix `*` wildcard (`appId`/`tenantId` exact; once a segment is `*`,
  all following segments must be `*`); each scope field is capped at 128
  bytes.

### Added

- `ITEM_TYPE_MEMORY_FILE_PREFIX` constant, `ItemView`, `ItemActor`, and
  `ItemPrecondition` types, `ItemListFileEntry` / `ItemListPrefixEntry` /
  `ItemListEntry` for typed `listItems` entries, and
  `DeleteItemSingleResponse` / `DeleteItemRangedResponse` (with
  `DeleteItemResponse` now their union).

## 0.0.12

Type-declaration and packaging-metadata release: the request/response models
were realigned with the service contract. **No runtime behaviour changed**, no
dependencies were added or removed, and the module layout (`exports`, `types`)
is unchanged.

### Added

Memory store configuration:

- `CreateMemoryStoreRequest`: `storageMode` (`'ots' | 'file+ots' | 'filemem'`,
  defaults to `ots`, immutable after creation) and `extractInstructions`.
- `UpdateMemoryStoreRequest`: `extractInstructions`. Both patchable fields now
  accept `null` / `UNSET` alongside `string`, matching the PATCH semantics
  already used by `UpdateKnowledgeBaseRequest` — omitted means "leave
  unchanged", `''` means "clear".
- `MemoryStoreInfo`: `storageMode` and `extractInstructions`.
- `MemoryStoreStorageMode` type.

Memory search:

- `SearchMemoriesRequest`: `contextScope`, `minSimilarity` and
  `includeEvidence`, all serialized by `toJSONRequest()`.
- `SearchHit`: `similarity`, `rankingScore` and `rankingSignals`, plus the new
  `RankingSignals` type.

Dream:

- `CreateMemoryDreamTaskRequest`: `taskType` and `incremental`.
- `ListMemoryDreamActionsRequest`: `scope` and `actionType`, enabling the
  scope-based query mode for emitted extracts. `dreamId` is now optional
  because the two modes are mutually exclusive.
- `DreamTaskType`, `DreamExtractActionType`, `DreamExtractSpec`,
  `DreamConfidenceThresholdKey` and `DreamConfidenceThresholds` types.
- `DreamMemorySpec`: `title`, `keywords`, `effectiveAt` and `timePrecision`,
  completing the eight fields the service emits.
- `DreamActionRecord`: `newExtract`.
- `DreamTask`: `taskType`, `incremental`, `processedThroughMs` and
  `skippedByReason`.
- `DreamTaskInputSummary`: `incremental`.
- `DreamTaskActionStats`: `skippedByReason`.
- `DreamTaskSummary`: `taskType`, `processedThroughMs` and `instructions`.
- `CancelMemoryDreamTaskResponse`: `taskType`.

Items:

- `MAX_ITEM_CONTENT_BYTES`, `MAX_ITEM_PATH_BYTES` and `MAX_ITEMS_PER_SCOPE`
  constants.

### Changed

- `DreamActionType` gained `EMIT_SKILL` and `EMIT_PROFILE` (5 → 7 values).
- `DreamSkippedReason` gained `superseded_by_skill`, `superseded_by_profile`,
  `noop_nothing_to_apply` and `conflicting_action` (8 → 12 values).
- `DreamApplyActionResult.status` is now `DreamActionStatus | 'not_found'`.
- `ListMemoryStoreMessagesRequest` and `ListMemoryStoreRequestsRequest` accept
  `number | string` timestamps; these two endpoints take either Unix
  milliseconds or RFC3339, while every dream endpoint takes Unix milliseconds
  only.
- Extensive JSDoc covering field limits, defaults, enum values, mutually
  exclusive parameters and error codes.

### Compatibility

No field was renamed, removed, or changed from optional to required. Three
declarations became strictly wider unions and can surface in an exhaustive
`switch` or in an assignment to the narrower type:

- `DreamActionType` and `DreamSkippedReason` have more members.
- `DreamApplyActionResult.status` includes `'not_found'`.

`DreamConfidenceThresholds` deliberately keeps a `Record<string, number>` arm
so dynamically built threshold maps continue to type-check; only `add`,
`update` and `merge` are accepted by the service.

### Packaging

- Declared `license: "Apache-2.0"`, matching `rust/Cargo.toml` and the
  repository `LICENSE`; earlier releases published with no license field.
- Added `repository`, `homepage` and `bugs` pointing at
  `aliyun/tablestore-agent-storage-sdk`, and `keywords`.
- Added `publishConfig` (`registry: https://registry.npmjs.org`,
  `access: public`) so the package always targets the public registry
  regardless of the local npm mirror configuration.
- `CHANGELOG.md` is now shipped inside the tarball via `files`.

## 0.0.11

**Breaking (introduced in this release, commit `88f0171`):** the file-memory
API surface was renamed from `FileMemory*` to `Item*`, replacing nine
`*FileMemory` client methods with eight item methods. Code written against
0.0.9 or 0.0.10 must be updated:

- `createFileMemory` → `addItem`
- `listFileMemories` → `listItems`
- `readFileMemory` → `getItem`, and `statFileMemory` → `getItem` with
  `includeContent: false`
- `updateFileMemory` → `updateItem`, and `renameFileMemory` → `updateItem`
  with `newPath`
- `deleteFileMemory` → `deleteItem`
- `listFileMemoryVersions` → `listItemVersions`
- `getFileMemoryVersion` → `getItemVersion`
- `redactItemVersion` was added (no 0.0.10 equivalent)

Types were renamed to match (`FileMemory` → `Item`, `FileMemoryVersion` →
`ItemVersion`, `CreateFileMemoryRequest` → `AddItemRequest`, …), and the wire
fields changed too: `memoryId` → `itemId`, `memoryVersionId` → `versionId`,
and the list response key `memories` → `items`. Requests now extend
`ItemScopedRequest` and carry `type: 'memoryfile'`, which the client fills in
when omitted. The old `*FileMemory` routes were removed from the service and
return `404`.
