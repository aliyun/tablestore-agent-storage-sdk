import { afterAll, describe, expect, it } from 'vitest';

import {
  AgentStorageClient,
  AddMemoriesRequest,
  CreateMemoryStoreRequest,
  ListMemoryStoreMessagesRequest,
  ListMemoryStoreRequestsRequest,
  type MemoryUnit,
  MessagePayload,
  OTSServiceError,
  Scope,
  SearchMemoriesRequest,
} from '../../src/index.js';

function requiredEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function suffix(length = 8): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let index = 0; index < length; index += 1) {
    result += chars[Math.floor(Math.random() * chars.length)];
  }
  return result;
}

function appId(label: string): string {
  return `ts_app_${label}_${suffix()}`;
}

function runId(label: string): string {
  return `ts_run_${label}_${suffix()}`;
}

function fullScope(label: string) {
  return {
    appId: appId(label),
    tenantId: `tenant_${label}_${suffix(4)}`,
    agentId: `agent_${label}_${suffix(4)}`,
    runId: runId(label),
  };
}

function memoryText(unit: Partial<MemoryUnit> | undefined): string {
  if (!unit) {
    return '';
  }
  return String(unit.text ?? '');
}

function createClient(): AgentStorageClient {
  return new AgentStorageClient({
    endpoint: requiredEnv('OTS_ENDPOINT'),
    instanceName: requiredEnv('OTS_INSTANCE_NAME'),
    accessKeyId: requiredEnv('OTS_ACCESS_KEY_ID'),
    accessKeySecret: requiredEnv('OTS_ACCESS_KEY_SECRET'),
    region: process.env.OTS_REGION,
    signVersion: (process.env.OTS_SIGN_VERSION as 'v2' | 'v4' | undefined) ?? 'v2',
  });
}

function isMemoryIdIndexBuildingError(error: unknown): error is OTSServiceError {
  return (
    error instanceof OTSServiceError &&
    error.statusCode === 409 &&
    error.code === 'OTSOperationConflict' &&
    error.message.toLowerCase().includes('secondary index is still building')
  );
}

function isRetryableInternalServerError(error: unknown): error is OTSServiceError {
  return error instanceof OTSServiceError && error.statusCode === 500;
}

function isRetryableMemoryNotFound(error: unknown): error is OTSServiceError {
  return (
    error instanceof OTSServiceError &&
    error.statusCode === 404 &&
    error.code === 'OTSObjectNotExist'
  );
}

async function poll<T>(
  action: () => Promise<T>,
  predicate: (value: T) => boolean,
  options: {
    timeoutMs?: number;
    intervalMs?: number;
    description: string;
  }
): Promise<T> {
  const timeoutMs = options.timeoutMs ?? 90_000;
  const intervalMs = options.intervalMs ?? 2_000;
  const deadline = Date.now() + timeoutMs;
  let lastValue: T | undefined;

  while (Date.now() < deadline) {
    lastValue = await action();
    if (predicate(lastValue)) {
      return lastValue;
    }
    await sleep(intervalMs);
  }

  throw new Error(`${options.description}. Last value: ${JSON.stringify(lastValue)}`);
}

async function waitForListMatch(
  client: AgentStorageClient,
  request: Record<string, unknown>,
  matcher: (item: MemoryUnit) => boolean,
  description: string
) {
  return poll(
    async () => client.listMemories(request),
    (result) => result.memories.some((item) => matcher(item)),
    {
      timeoutMs: 90_000,
      intervalMs: 2_000,
      description,
    }
  );
}

async function waitForSearchHit(
  client: AgentStorageClient,
  request: SearchMemoriesRequest | Record<string, unknown>,
  fragment: string,
  description: string
) {
  return poll(
    async () => client.searchMemories(request),
    (result) => result.results.some((hit) => memoryText(hit.unit).includes(fragment)),
    {
      timeoutMs: 120_000,
      intervalMs: 2_000,
      description,
    }
  );
}

async function waitForMemoryIdIndexReady(
  client: AgentStorageClient,
  memoryStoreName: string,
  scope: Record<string, string>
) {
  const deadline = Date.now() + 120_000;
  while (Date.now() < deadline) {
    try {
      await client.getMemory({
        memoryStoreName,
        memoryId: 'warmup-missing-memory-id',
        scope,
      });
      return;
    } catch (error) {
      if (error instanceof OTSServiceError && error.statusCode === 404) {
        return;
      }
      if (isMemoryIdIndexBuildingError(error)) {
        await sleep(2_000);
        continue;
      }
      throw error;
    }
  }

  throw new Error('memory id index did not become ready in time');
}

async function withRetry<T>(
  action: () => Promise<T>,
  options: {
    attempts?: number;
    intervalMs?: number;
    shouldRetry: (error: unknown) => boolean;
  }
): Promise<T> {
  const attempts = options.attempts ?? 3;
  const intervalMs = options.intervalMs ?? 2_000;
  let lastError: unknown;

  for (let attempt = 0; attempt < attempts; attempt += 1) {
    try {
      return await action();
    } catch (error) {
      lastError = error;
      if (attempt === attempts - 1 || !options.shouldRetry(error)) {
        throw error;
      }
      await sleep(intervalMs);
    }
  }

  throw lastError;
}

async function waitForMemoryDeletion(
  client: AgentStorageClient,
  memoryStoreName: string,
  memoryId: string,
  scope: Record<string, string>
) {
  const deadline = Date.now() + 120_000;
  while (Date.now() < deadline) {
    try {
      const result = await client.getMemory({
        memoryStoreName,
        memoryId,
        scope,
      });
      if (result.memory.deleted) {
        return;
      }
      await sleep(2_000);
    } catch (error) {
      if (isMemoryIdIndexBuildingError(error)) {
        await sleep(2_000);
        continue;
      }
      if (error instanceof OTSServiceError && error.statusCode === 404) {
        return;
      }
      throw error;
    }
  }

  throw new Error('deleted memory was still readable after timeout');
}

async function warmMemoryStore(client: AgentStorageClient, memoryStoreName: string) {
  const sentinelScope = fullScope('warm');

  await sleep(60_000);
  await client.addMemories({
    memoryStoreName,
    scope: sentinelScope,
    text: 'TypeScript SDK warm sentinel memory for store readiness.',
    sync: true,
  });

  await waitForListMatch(
    client,
    { memoryStoreName, scope: sentinelScope, limit: 10 },
    (item) => memoryText(item).toLowerCase().includes('warm sentinel'),
    'warm sentinel memory did not become list-visible'
  );
  await waitForMemoryIdIndexReady(client, memoryStoreName, sentinelScope);
}

async function findMemoryId(
  client: AgentStorageClient,
  memoryStoreName: string,
  scope: Record<string, string>,
  fragment: string
) {
  const listed = await waitForListMatch(
    client,
    { memoryStoreName, scope, limit: 50 },
    (item) => memoryText(item).includes(fragment),
    `memory containing ${fragment} did not appear in list`
  );

  const found = listed.memories.find((item) =>
    memoryText(item).includes(fragment)
  );
  if (!found) {
    throw new Error(`Memory containing ${fragment} was not found`);
  }
  return found.id;
}

describe('memory e2e', () => {
  const client = createClient();
  const memoryStoreName = `tsmem_${Date.now()}_${suffix(6)}`;

  afterAll(async () => {
    await client.deleteMemoryStore({ memoryStoreName }).catch(() => undefined);
  });

  it(
    'creates a memory store and exercises memory lifecycle with json and model inputs',
    async () => {
      const createStore = await client.createMemoryStore(
        new CreateMemoryStoreRequest({
          memoryStoreName,
          description: 'TypeScript SDK memory e2e',
        })
      );
      expect(createStore.memoryStoreName).toBe(memoryStoreName);

      const fetchedStore = await client.getMemoryStore({ memoryStoreName });
      expect(fetchedStore.description).toBe('TypeScript SDK memory e2e');

      const listedStores = await client.listMemoryStores({ limit: 100 });
      expect(listedStores.stores.some((store) => store.memoryStoreName === memoryStoreName)).toBe(true);

      const updatedStore = await client.updateMemoryStore({
        memoryStoreName,
        description: 'TypeScript SDK memory e2e updated',
      });
      expect(updatedStore.description).toBe('TypeScript SDK memory e2e updated');

      await warmMemoryStore(client, memoryStoreName);

      const conversationScope = fullScope('messages');
      const addMessagesResult = await withRetry(
        () =>
          client.addMemories(
            new AddMemoriesRequest({
              memoryStoreName,
              scope: new Scope(conversationScope),
              messages: [
                new MessagePayload({
                  role: 'user',
                  content: 'Ian enjoys espresso in the morning.',
                  messageId: `msg_${suffix(6)}_1`,
                }),
                new MessagePayload({
                  role: 'assistant',
                  content: 'I will remember Ian enjoys espresso.',
                  messageId: `msg_${suffix(6)}_2`,
                }),
              ],
              sync: true,
            })
          ),
        {
          attempts: 5,
          intervalMs: 2_000,
          shouldRetry: isRetryableInternalServerError,
        }
      );
      expect(addMessagesResult.memoryStoreName).toBe(memoryStoreName);
      expect(addMessagesResult.acceptedMessages).toBe(2);

      await waitForListMatch(
        client,
        { memoryStoreName, scope: conversationScope, limit: 20 },
        (item) => memoryText(item).toLowerCase().includes('espresso'),
        'conversation memory did not become list-visible'
      );

      const messagesResponse = await client.listMemoryStoreMessages(
        new ListMemoryStoreMessagesRequest({
          memoryStoreName,
          scope: new Scope(conversationScope),
          limit: 20,
        })
      );
      expect(messagesResponse.session.scope.appId).toBe(conversationScope.appId);
      expect(messagesResponse.session.messages.length).toBeGreaterThanOrEqual(2);
      expect(messagesResponse.session.messages.some((message) => message.role === 'user')).toBe(true);

      const requestsResponse = await client.listMemoryStoreRequests(
        new ListMemoryStoreRequestsRequest({
          memoryStoreName,
          scope: new Scope({ appId: conversationScope.appId }),
          operation: 'AddMemories',
          limit: 20,
        })
      );
      expect(requestsResponse.memoryStoreName).toBe(memoryStoreName);
      expect(Array.isArray(requestsResponse.requests)).toBe(true);
      if (requestsResponse.requests.length > 0) {
        expect(requestsResponse.requests.every((request) => request.operation === 'AddMemories')).toBe(true);
      }

      const exactScope = fullScope('crud');
      const addTextResult = await client.addMemories({
        memoryStoreName,
        scope: exactScope,
        text: 'Victor is a pilot based in Kunming.',
        metadata: { city: 'kunming', source: 'ts-e2e' },
        sync: true,
      });
      expect(addTextResult.acceptedMessages).toBe(1);

      const memoryId = await findMemoryId(client, memoryStoreName, exactScope, 'Kunming');

      const getResult = await client.getMemory({
        memoryStoreName,
        memoryId,
        scope: exactScope,
      });
      expect(memoryText(getResult.memory)).toContain('Kunming');

      const updateResult = await withRetry(
        () =>
          client.updateMemory({
            memoryStoreName,
            memoryId,
            scope: exactScope,
            text: 'Victor is a pilot based in Chengdu.',
            metadata: { city: 'chengdu', source: 'ts-e2e-updated' },
          }),
        {
          attempts: 5,
          intervalMs: 2_000,
          shouldRetry: isRetryableMemoryNotFound,
        }
      );
      expect(updateResult.updated).toBe(true);
      expect(memoryText(updateResult.memory)).toContain('Chengdu');

      const searchResult = await waitForSearchHit(
        client,
        new SearchMemoriesRequest({
          memoryStoreName,
          scope: new Scope(exactScope),
          query: 'Victor Chengdu pilot',
          topK: 5,
          enableRerank: false,
        }),
        'Chengdu',
        'updated memory did not become searchable'
      );
      expect(searchResult.memoryStoreName).toBe(memoryStoreName);

      const listedMemories = await client.listMemories({
        memoryStoreName,
        scope: exactScope,
        limit: 20,
      });
      expect(listedMemories.memories.some((memory) => memory.id === memoryId)).toBe(true);

      const deleteResult = await withRetry(
        () =>
          client.deleteMemory({
            memoryStoreName,
            memoryId,
            scope: exactScope,
          }),
        {
          attempts: 10,
          intervalMs: 3_000,
          shouldRetry: isMemoryIdIndexBuildingError,
        }
      );
      expect(deleteResult.deleted).toBe(true);
      expect(deleteResult.memoryId).toBe(memoryId);

      await waitForMemoryDeletion(client, memoryStoreName, memoryId, exactScope);

      const deleteStore = await client.deleteMemoryStore({ memoryStoreName });
      expect(deleteStore.deleted).toBe(true);

      await expect(client.getMemoryStore({ memoryStoreName })).rejects.toMatchObject({
        name: 'OTSServiceError',
        statusCode: 404,
      } satisfies Partial<OTSServiceError>);
    },
    360_000
  );
});
