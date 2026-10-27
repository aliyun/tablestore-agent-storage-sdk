import { describe, expect, it, vi } from 'vitest';

import { AgentStorageClient } from '../../src/index.js';
import { RequestContext } from '../../src/core/auth/request-context.js';
import { SignV2 } from '../../src/core/auth/sign-v2.js';
import { md5Base64 } from '../../src/core/http/md5.js';

function successResponse(path: string, payload: Record<string, unknown>) {
  const body = JSON.stringify(payload);
  const headers = {
    'x-ots-contentmd5': md5Base64(new TextEncoder().encode(body)),
    'x-ots-requestid': 'req-1',
    'x-ots-date': '2025-04-10T01:02:03.000Z',
    'x-ots-contenttype': 'application/json',
  };
  const signer = new SignV2();
  const requestContext = new RequestContext({ accessKeyId: 'ak', accessKeySecret: 'sk' });
  const signature = signer.makeResponseSignature(path, headers, signer.getSigningKey(requestContext));

  return new Response(body, {
    status: 200,
    headers: {
      ...headers,
      authorization: `OTS ak:${signature}`,
    },
  });
}

function baseOptions(fetchMock: typeof globalThis.fetch) {
  return {
    endpoint: 'https://example.com',
    instanceName: 'instance',
    accessKeyId: 'ak',
    accessKeySecret: 'sk',
    fetch: fetchMock,
  };
}

describe('AgentStorageClient memory methods', () => {
  it('sends AddMemories and SearchMemories using documented JSON structure', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(successResponse('/AddMemories', { status: 'running', acceptedMessages: 1 }))
      .mockResolvedValueOnce(successResponse('/SearchMemories', { results: [] }));
    const client = new AgentStorageClient(baseOptions(fetchMock as typeof globalThis.fetch));

    await client.addMemories({ memoryStoreName: 'store_a', scope: { appId: 'app_a' }, text: 'hello' });
    await client.searchMemories({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a', tenantId: '__default__' },
      query: 'hello',
      enableRerank: false,
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://example.com/AddMemories',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          scope: { appId: 'app_a' },
          text: 'hello',
        }),
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://example.com/SearchMemories',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          scope: { appId: 'app_a', tenantId: '__default__' },
          query: 'hello',
          enableRerank: false,
        }),
      })
    );
  });

  it('sends GetMemory, UpdateMemory, DeleteMemory, ListMemoryStoreMessages, and ListMemoryStoreRequests', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(successResponse('/GetMemory', { memory: { id: 'unit-1' } }))
      .mockResolvedValueOnce(successResponse('/UpdateMemory', { updated: true }))
      .mockResolvedValueOnce(successResponse('/DeleteMemory', { deleted: true }))
      .mockResolvedValueOnce(successResponse('/ListMemoryStoreMessages', { session: { messages: [] } }))
      .mockResolvedValueOnce(successResponse('/ListMemoryStoreRequests', { requests: [] }));
    const client = new AgentStorageClient(baseOptions(fetchMock as typeof globalThis.fetch));
    const fullScope = {
      appId: 'app_a',
      tenantId: 'tenant_a',
      agentId: 'agent_a',
      runId: 'run_a',
    };

    await client.getMemory({ memoryStoreName: 'store_a', memoryId: 'unit-1', scope: fullScope });
    await client.updateMemory({ memoryStoreName: 'store_a', memoryId: 'unit-1', scope: fullScope, text: 'updated' });
    await client.deleteMemory({ memoryStoreName: 'store_a', memoryId: 'unit-1', scope: fullScope });
    await client.listMemoryStoreMessages({ memoryStoreName: 'store_a', scope: fullScope, limit: 10 });
    await client.listMemoryStoreRequests({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a' },
      operation: 'AddMemories',
      minTimestamp: '2024-01-01T00:00:00Z',
      maxTimestamp: '2024-01-02T00:00:00Z',
    });

    expect(fetchMock).toHaveBeenNthCalledWith(1, 'https://example.com/GetMemory', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(2, 'https://example.com/UpdateMemory', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(3, 'https://example.com/DeleteMemory', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(4, 'https://example.com/ListMemoryStoreMessages', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      'https://example.com/ListMemoryStoreRequests',
      expect.objectContaining({
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          scope: { appId: 'app_a' },
          operation: 'AddMemories',
          minTimestamp: '2024-01-01T00:00:00Z',
          maxTimestamp: '2024-01-02T00:00:00Z',
        }),
      })
    );
  });

  it('sends GetMemoryTask, ListMemoryTasks, and ListMemoryStoreScopes', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        successResponse('/GetMemoryTask', {
          task: { requestId: 'req-1', scope: { appId: 'a' }, status: 'completed', createdAt: '', updatedAt: '' },
          memoryStoreName: 'store_a',
        })
      )
      .mockResolvedValueOnce(
        successResponse('/ListMemoryTasks', {
          tasks: [],
          scope: { appId: 'a' },
          memoryStoreName: 'store_a',
        })
      )
      .mockResolvedValueOnce(
        successResponse('/ListMemoryStoreScopes', {
          scopes: [{ appId: 'a' }],
          scope: { appId: 'a' },
          memoryStoreName: 'store_a',
        })
      );
    const client = new AgentStorageClient(baseOptions(fetchMock as typeof globalThis.fetch));

    await client.getMemoryTask({ memoryStoreName: 'store_a', requestId: 'req-1' });
    await client.listMemoryTasks({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a' },
      status: 'running',
      limit: 50,
    });
    await client.listMemoryStoreScopes({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a', tenantId: '*' },
      limit: 100,
    });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://example.com/GetMemoryTask',
      expect.objectContaining({
        body: JSON.stringify({ memoryStoreName: 'store_a', requestId: 'req-1' }),
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://example.com/ListMemoryTasks',
      expect.objectContaining({
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          scope: { appId: 'app_a' },
          status: 'running',
          limit: 50,
        }),
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      'https://example.com/ListMemoryStoreScopes',
      expect.objectContaining({
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          scope: { appId: 'app_a', tenantId: '*' },
          limit: 100,
        }),
      })
    );
  });

  it('sends the full Memory Dream lifecycle', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        successResponse('/CreateMemoryDreamTask', {
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          status: 'queued',
          createdAt: '2026-06-03T00:00:00Z',
        })
      )
      .mockResolvedValueOnce(
        successResponse('/GetMemoryDreamTask', {
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          status: 'queued',
          applyMode: 'proposal',
          scopeOutputMode: 'preserve_scope',
          input: {
            scopes: [{ appId: 'app_a' }],
            minTimestamp: 0,
            maxTimestamp: 0,
            sessionCount: 0,
            messageCount: 0,
            memoryCount: 0,
          },
          actions: { total: 0, proposed: 0, applied: 0, skipped: 0, failed: 0 },
          createdAt: '2026-06-03T00:00:00Z',
          updatedAt: '2026-06-03T00:00:00Z',
        })
      )
      .mockResolvedValueOnce(
        successResponse('/ListMemoryDreamTasks', {
          memoryStoreName: 'store_a',
          scope: { appId: 'app_a' },
          tasks: [],
        })
      )
      .mockResolvedValueOnce(
        successResponse('/ListMemoryDreamActions', {
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          actions: [],
        })
      )
      .mockResolvedValueOnce(
        successResponse('/ApplyMemoryDreamActions', {
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          applied: 1,
          failed: 0,
          results: [{ actionId: 'a-1', status: 'applied', memoryId: 'm-1' }],
        })
      )
      .mockResolvedValueOnce(
        successResponse('/CancelMemoryDreamTask', {
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          status: 'cancelled',
        })
      );
    const client = new AgentStorageClient(baseOptions(fetchMock as typeof globalThis.fetch));

    await client.createMemoryDreamTask({
      memoryStoreName: 'store_a',
      scopes: [{ appId: 'app_a' }],
      maxSessions: 10,
      applyMode: 'proposal',
    });
    await client.getMemoryDreamTask({ memoryStoreName: 'store_a', dreamId: 'dream-1' });
    await client.listMemoryDreamTasks({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a' },
      status: 'queued',
    });
    await client.listMemoryDreamActions({
      memoryStoreName: 'store_a',
      dreamId: 'dream-1',
      status: 'proposed',
      action: 'ADD',
    });
    await client.applyMemoryDreamActions({
      memoryStoreName: 'store_a',
      dreamId: 'dream-1',
      actionIds: ['a-1'],
      applier: 'admin',
    });
    await client.cancelMemoryDreamTask({ memoryStoreName: 'store_a', dreamId: 'dream-1' });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://example.com/CreateMemoryDreamTask',
      expect.objectContaining({
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          scopes: [{ appId: 'app_a' }],
          maxSessions: 10,
          applyMode: 'proposal',
        }),
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://example.com/GetMemoryDreamTask',
      expect.objectContaining({
        body: JSON.stringify({ memoryStoreName: 'store_a', dreamId: 'dream-1' }),
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      3,
      'https://example.com/ListMemoryDreamTasks',
      expect.objectContaining({
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          scope: { appId: 'app_a' },
          status: 'queued',
        }),
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      4,
      'https://example.com/ListMemoryDreamActions',
      expect.objectContaining({
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          status: 'proposed',
          action: 'ADD',
        }),
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      5,
      'https://example.com/ApplyMemoryDreamActions',
      expect.objectContaining({
        body: JSON.stringify({
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          actionIds: ['a-1'],
          applier: 'admin',
        }),
      })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      6,
      'https://example.com/CancelMemoryDreamTask',
      expect.objectContaining({
        body: JSON.stringify({ memoryStoreName: 'store_a', dreamId: 'dream-1' }),
      })
    );
  });
});
