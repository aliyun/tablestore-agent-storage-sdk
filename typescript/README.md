# Tablestore Agent Storage TypeScript SDK

This package contains the TypeScript SDK for Tablestore agent storage, including knowledge-base and memory APIs.

## Entrypoints

- `src/index.ts`: cross-runtime core client
- `src/node.ts`: Node-only client with local file upload support

## Usage

ES modules:

```ts
import { AgentStorageClient } from '@tablestore/agent-storage';

const client = new AgentStorageClient({
  endpoint: 'https://your-instance.cn-beijing.ots.aliyuncs.com',
  instanceName: 'your-instance',
  accessKeyId: process.env.OTS_ACCESS_KEY_ID!,
  accessKeySecret: process.env.OTS_ACCESS_KEY_SECRET!,
});
```

```ts
import { NodeAgentStorageClient } from '@tablestore/agent-storage/node';

const client = new NodeAgentStorageClient({
  endpoint: 'https://your-instance.cn-beijing.ots.aliyuncs.com',
  instanceName: 'your-instance',
  accessKeyId: process.env.OTS_ACCESS_KEY_ID!,
  accessKeySecret: process.env.OTS_ACCESS_KEY_SECRET!,
  ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
  ossBucketName: 'tantan-source',
});
```

CommonJS:

```js
const { AgentStorageClient } = require('@tablestore/agent-storage');
const { NodeAgentStorageClient } = require('@tablestore/agent-storage/node');
```

## Signatures

- default: `v2`
- optional: `v4` with `region`

## Input Modes

All APIs accept:

- plain JSON requests
- model instances with `toJSONRequest()`

## API Surface

- `AgentStorageClient`: cross-runtime client for knowledge-base and memory APIs
- `NodeAgentStorageClient`: Node client that adds OSS-backed document upload

Memory APIs include:

- memory store lifecycle: `createMemoryStore`, `getMemoryStore`, `listMemoryStores`, `updateMemoryStore`, `deleteMemoryStore`
- memory CRUD/search: `addMemories`, `searchMemories`, `listMemories`, `getMemory`, `updateMemory`, `deleteMemory`
- audit/message queries: `listMemoryStoreMessages`, `listMemoryStoreRequests`
- ingest task tracking: `getMemoryTask`, `listMemoryTasks`
- scope discovery: `listMemoryStoreScopes`
- memory dream tasks: `createMemoryDreamTask`, `getMemoryDreamTask`, `listMemoryDreamTasks`, `cancelMemoryDreamTask`
- memory dream actions: `listMemoryDreamActions`, `applyMemoryDreamActions`
- items (file memory): `addItem`, `listItems`, `getItem`, `updateItem`, `deleteItem`, `listItemVersions`, `getItemVersion`, `redactItemVersion`

## Storage modes

A memory store's `storageMode` is chosen at `createMemoryStore` time and cannot
be changed afterwards:

- `ots` (default): structured memory only. Item APIs are rejected with `400`.
- `filemem`: writable, versioned memory files, reached through the item APIs.
- `file+ots`: structured memory plus a **read-only** projected file tree. Item
  reads work, item writes return `409 READ_ONLY_STORE`, and the version APIs
  return `400`.

```ts
await client.createMemoryStore({
  memoryStoreName: 'agent_memory',
  storageMode: 'filemem',
  // description is capped at 1024 bytes; extractInstructions at 4096 runes
  extractInstructions: '只抽取与订单相关的事实',
});
```

`updateMemoryStore` is a strict PATCH and accepts only `description` and
`extractInstructions`. Omitting a field leaves it unchanged; sending `''`
clears it.

`listMemoryStores` returns only `memoryStoreName` per entry — call
`getMemoryStore` per store to read `storageMode`, `description` or
`extractInstructions`.

## Items (file memory)

All eight item actions carry `"type": "memoryfile"`, which the SDK fills in
automatically. `addItem`, `getItem`, `updateItem`, exact `deleteItem`,
`getItemVersion`, and `redactItemVersion` require a **fully specified
four-part scope**. Only `listItems`, `listItemVersions`, and ranged
`deleteItem` accept the left-prefix `*` wildcard: `appId` and `tenantId` stay
exact, and once a segment is `*` every segment after it must be `*` too.

```ts
const scoped = {
  memoryStoreName: 'agent_memory',
  scope: { appId: 'a', tenantId: 't', agentId: 'ag', runId: 'r' },
};

await client.addItem({ ...scoped, path: '/notes/a.md', content: 'hello' });

// Get and exact delete require exactly one address: itemId or path.
// view defaults to 'basic' (metadata only); use 'full' for content.
const item = await client.getItem({ ...scoped, path: '/notes/a.md', view: 'full' });

// Update requires itemId plus content, a target path, or both.
const updated = await client.updateItem({
  ...scoped,
  itemId: item.itemId,
  content: 'updated',
  path: '/notes/b.md', // Target path; itemId is the source address.
  precondition: { expectedVersionId: item.latestVersionId },
  view: 'full',
});

await client.deleteItem({
  ...scoped,
  itemId: updated.itemId,
  precondition: { expectedVersionId: updated.latestVersionId },
});

const listed = await client.listItems({ ...scoped, pathPrefix: '/notes/', depth: 1 });
// listed.readOnly is true only for file+ots projections; omitted otherwise.
// depth: 1 rolls deeper directories up into entries typed 'memoryfile_prefix'.

// Ranged delete: wildcard scope and no itemId/path/precondition. The service
// immediately returns HTTP 200 with { type, taskId, status: 'pending' }.
const task = await client.deleteItem({
  memoryStoreName: 'agent_memory',
  scope: { appId: 'a', tenantId: 't', agentId: '*', runId: '*' },
  pathPrefix: '/notes/',
});
```

Service-enforced limits: content ≤ 100 KiB (`413 PAYLOAD_TOO_LARGE`), path
≤ 1024 bytes, `limit` defaults to 20 with a maximum of 100 (capped at 20
when `view: 'full'`). Paths must start with `/` (no longer auto-added), be
valid UTF-8, and reject `.`/`..`/empty segments. Every item/version response
echoes its `scope`, items carry a `latestVersionId` head pointer, and version
attribution uses typed actors (`createdBy` / `redactedBy` objects with
`actorType` and `actorId`).

Versions are addressed by the self-contained `versionId` alone —
`getItemVersion` and `redactItemVersion` take no `itemId`/`versionSeq`.
`listItemVersions` is scope-level: besides `itemId` it supports `sessionId`,
`apiKeyId`, `createdAtGte`/`createdAtLte` (RFC3339) and `operation` filters.

`redactItemVersion` irreversibly clears one historical version's content and
path. It is idempotent and first-wins; afterwards `getItemVersion` still
returns `200` for that version but never its content.

## Dream tasks

`taskType` defaults to `memory`. Use `skill` or `profile` to emit extracts
instead of memory mutations — those task types reject `applyMode` with a `400`,
and their `EMIT_SKILL` / `EMIT_PROFILE` actions cannot be applied.

`listMemoryDreamActions` has two mutually exclusive modes: pass `dreamId` to
list one run's actions, or pass `scope` plus `actionType`
(`EMIT_SKILL` / `EMIT_PROFILE`) to list emitted extracts across runs. Sending
both or neither is a `400`.

Dream `minTimestamp` / `maxTimestamp` accept **Unix milliseconds only**;
`listMemoryStoreMessages` and `listMemoryStoreRequests` additionally accept
RFC3339.

## Tests

Run unit tests:

```bash
npm run test:unit
```

Run e2e tests:

```bash
OTS_ENDPOINT=https://your-instance.cn-beijing.ots.aliyuncs.com \
OTS_ACCESS_KEY_ID=your-ak \
OTS_ACCESS_KEY_SECRET=your-sk \
OTS_INSTANCE_NAME=your-instance \
OTS_REGION=cn-beijing \
OTS_SIGN_VERSION=v2 \
OSS_ENDPOINT=https://oss-cn-beijing.aliyuncs.com \
OSS_BUCKET_NAME=tantan-source \
npm run test:e2e
```
