import { afterAll, describe, expect, it } from 'vitest';

import { AgentStorageClient, OTSServiceError } from '../../src/index.js';

function requiredEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

function suffix(length = 8): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let index = 0; index < length; index += 1) {
    result += chars[Math.floor(Math.random() * chars.length)];
  }
  return result;
}

function createApiKeyClient(): AgentStorageClient {
  return new AgentStorageClient({
    endpoint: requiredEnv('OTS_ENDPOINT'),
    instanceName: requiredEnv('OTS_INSTANCE_NAME'),
    apiKey: requiredEnv('OTS_API_KEY'),
  });
}

describe('API key authentication e2e', () => {
  const storeName = `ts_apikey_e2e_${suffix()}`;
  const client = createApiKeyClient();

  afterAll(async () => {
    try {
      await client.deleteMemoryStore({ memoryStoreName: storeName });
    } catch {
      // ignore cleanup errors
    }
  });

  it('creates a memory store using API key auth', async () => {
    const response = await client.createMemoryStore({ memoryStoreName: storeName });
    expect(response.memoryStoreName).toBe(storeName);
  });

  it('lists memory stores using API key auth', async () => {
    const response = await client.listMemoryStores({});
    expect(response.stores).toBeDefined();
    expect(Array.isArray(response.stores)).toBe(true);
  });

  it('gets the created memory store', async () => {
    const response = await client.getMemoryStore({ memoryStoreName: storeName });
    expect(response.memoryStoreName).toBe(storeName);
  });

  it('adds memories and searches using API key auth', async () => {
    const scope = { appId: `ts_apikey_${suffix()}`, tenantId: '__default__', agentId: '__default__', runId: `run_${suffix()}` };
    const addResponse = await client.addMemories({
      memoryStoreName: storeName,
      scope,
      text: 'TypeScript API key authentication test memory.',
      sync: true,
    });
    expect(addResponse.status).toBeDefined();
    expect(addResponse.acceptedMessages).toBeGreaterThanOrEqual(0);

    const searchResponse = await client.searchMemories({
      memoryStoreName: storeName,
      scope,
      query: 'API key authentication',
    });
    expect(searchResponse.results).toBeDefined();
  }, 30000);

  it('returns proper error for invalid API key', async () => {
    const badClient = new AgentStorageClient({
      endpoint: requiredEnv('OTS_ENDPOINT'),
      instanceName: requiredEnv('OTS_INSTANCE_NAME'),
      apiKey: 'invalid-key',
    });

    await expect(badClient.listMemoryStores({})).rejects.toThrow();
  });

  it('deletes the memory store', async () => {
    const response = await client.deleteMemoryStore({ memoryStoreName: storeName });
    expect(response.deleted).toBe(true);
  });
});
