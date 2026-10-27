import { describe, expect, it } from 'vitest';

import {
  AddMemoriesRequest,
  ApplyMemoryDreamActionsRequest,
  CancelMemoryDreamTaskRequest,
  CreateMemoryDreamTaskRequest,
  CreateMemoryStoreRequest,
  GetMemoryDreamTaskRequest,
  GetMemoryTaskRequest,
  ListMemoryDreamActionsRequest,
  ListMemoryDreamTasksRequest,
  ListMemoryStoreMessagesRequest,
  ListMemoryStoreRequestsRequest,
  ListMemoryStoreScopesRequest,
  ListMemoryTasksRequest,
  MessagePayload,
  Scope,
  SearchMemoriesRequest,
  UNSET,
  UpdateMemoryStoreRequest,
  toJSONRequest,
} from '../../src/index.js';

describe('memory serialization', () => {
  it('serializes AddMemoriesRequest models to documented JSON', () => {
    const payload = toJSONRequest(
      new AddMemoriesRequest({
        memoryStoreName: 'store_a',
        scope: new Scope({ appId: 'app_a' }),
        text: 'Alice works at Google.',
      })
    );

    expect(payload).toEqual({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a' },
      text: 'Alice works at Google.',
    });
  });

  it('serializes messages arrays and sync flags', () => {
    const payload = toJSONRequest(
      new AddMemoriesRequest({
        memoryStoreName: 'store_a',
        scope: new Scope({ appId: 'app_a', tenantId: '__default__' }),
        messages: [
          new MessagePayload({
            role: 'user',
            content: 'Ian enjoys espresso.',
            messageId: 'msg-1',
            metadata: { topic: 'coffee' },
          }),
        ],
        sync: true,
      })
    );

    expect(payload).toEqual({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a', tenantId: '__default__' },
      messages: [
        {
          role: 'user',
          content: 'Ian enjoys espresso.',
          messageId: 'msg-1',
          metadata: { topic: 'coffee' },
        },
      ],
      sync: true,
    });
  });

  it('serializes search and message list requests', () => {
    const searchPayload = toJSONRequest(
      new SearchMemoriesRequest({
        memoryStoreName: 'store_a',
        scope: new Scope({ appId: 'app_a', tenantId: '__default__' }),
        query: 'Ian espresso',
        metadata: { topic: 'coffee' },
        topK: 5,
        enableRerank: false,
      })
    );

    const messagesPayload = toJSONRequest(
      new ListMemoryStoreMessagesRequest({
        memoryStoreName: 'store_a',
        scope: new Scope({
          appId: 'app_a',
          tenantId: 'tenant_a',
          agentId: 'agent_a',
          runId: 'run_a',
        }),
        minTimestamp: '2024-01-01T00:00:00Z',
        maxTimestamp: '2024-01-02T00:00:00Z',
        limit: 10,
      })
    );

    expect(searchPayload).toEqual({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a', tenantId: '__default__' },
      query: 'Ian espresso',
      metadata: { topic: 'coffee' },
      topK: 5,
      enableRerank: false,
    });
    expect(messagesPayload).toEqual({
      memoryStoreName: 'store_a',
      scope: {
        appId: 'app_a',
        tenantId: 'tenant_a',
        agentId: 'agent_a',
        runId: 'run_a',
      },
      minTimestamp: '2024-01-01T00:00:00Z',
      maxTimestamp: '2024-01-02T00:00:00Z',
      limit: 10,
    });
  });

  it('serializes request audit filters with timestamp bounds', () => {
    const payload = toJSONRequest(
      new ListMemoryStoreRequestsRequest({
        memoryStoreName: 'store_a',
        scope: new Scope({ appId: 'app_a', tenantId: '*' }),
        operation: 'AddMemories',
        limit: 10,
        nextToken: 'token-1',
        minTimestamp: '2024-01-01T00:00:00Z',
        maxTimestamp: '2024-01-02T00:00:00Z',
      })
    );

    expect(payload).toEqual({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a', tenantId: '*' },
      operation: 'AddMemories',
      limit: 10,
      nextToken: 'token-1',
      minTimestamp: '2024-01-01T00:00:00Z',
      maxTimestamp: '2024-01-02T00:00:00Z',
    });
  });

  it('serializes memory store requests', () => {
    expect(
      toJSONRequest(new CreateMemoryStoreRequest({ memoryStoreName: 'store_a', description: 'store' }))
    ).toEqual({
      memoryStoreName: 'store_a',
      description: 'store',
    });
  });

  it('serializes GetMemoryTaskRequest with optional scope', () => {
    expect(
      toJSONRequest(
        new GetMemoryTaskRequest({
          memoryStoreName: 'store_a',
          requestId: 'req-123',
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      requestId: 'req-123',
    });

    expect(
      toJSONRequest(
        new GetMemoryTaskRequest({
          memoryStoreName: 'store_a',
          requestId: 'req-123',
          scope: new Scope({ appId: 'app_a' }),
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      requestId: 'req-123',
      scope: { appId: 'app_a' },
    });
  });

  it('serializes ListMemoryTasksRequest with all filters', () => {
    expect(
      toJSONRequest(
        new ListMemoryTasksRequest({
          memoryStoreName: 'store_a',
          scope: new Scope({ appId: 'app_a' }),
          status: 'running',
          limit: 50,
          nextToken: 'tok',
          minTimestamp: 1700000000000,
          maxTimestamp: '1700000999000',
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a' },
      status: 'running',
      limit: 50,
      nextToken: 'tok',
      minTimestamp: 1700000000000,
      maxTimestamp: '1700000999000',
    });
  });

  it('serializes ListMemoryStoreScopesRequest', () => {
    expect(
      toJSONRequest(
        new ListMemoryStoreScopesRequest({
          memoryStoreName: 'store_a',
          scope: new Scope({ appId: 'app_a', tenantId: '*' }),
          limit: 100,
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a', tenantId: '*' },
      limit: 100,
    });
  });

  it('serializes CreateMemoryDreamTaskRequest with full options', () => {
    const payload = toJSONRequest(
      new CreateMemoryDreamTaskRequest({
        memoryStoreName: 'store_a',
        scopes: [
          new Scope({ appId: 'app_a', tenantId: 'tenant_a' }),
          { appId: 'app_a', tenantId: 'tenant_b' },
        ],
        minTimestamp: 1700000000000,
        maxTimestamp: 1700001000000,
        maxSessions: 50,
        maxMessages: 2000,
        maxMemories: 500,
        expandedScopeLimit: 100,
        scopeOutputMode: 'preserve_scope',
        applyMode: 'proposal',
        confidenceThresholds: { add: 0.8, update: 0.85 },
        instructions: 'consolidate atomic facts',
        clientToken: 'client-token-1',
      })
    );

    expect(payload).toEqual({
      memoryStoreName: 'store_a',
      scopes: [
        { appId: 'app_a', tenantId: 'tenant_a' },
        { appId: 'app_a', tenantId: 'tenant_b' },
      ],
      minTimestamp: 1700000000000,
      maxTimestamp: 1700001000000,
      maxSessions: 50,
      maxMessages: 2000,
      maxMemories: 500,
      expandedScopeLimit: 100,
      scopeOutputMode: 'preserve_scope',
      applyMode: 'proposal',
      confidenceThresholds: { add: 0.8, update: 0.85 },
      instructions: 'consolidate atomic facts',
      clientToken: 'client-token-1',
    });
  });

  it('serializes GetMemoryDreamTaskRequest and CancelMemoryDreamTaskRequest', () => {
    expect(
      toJSONRequest(new GetMemoryDreamTaskRequest({ memoryStoreName: 'store_a', dreamId: 'dream-1' }))
    ).toEqual({
      memoryStoreName: 'store_a',
      dreamId: 'dream-1',
    });

    expect(
      toJSONRequest(new CancelMemoryDreamTaskRequest({ memoryStoreName: 'store_a', dreamId: 'dream-1' }))
    ).toEqual({
      memoryStoreName: 'store_a',
      dreamId: 'dream-1',
    });
  });

  it('serializes ListMemoryDreamTasksRequest with status and timestamps', () => {
    expect(
      toJSONRequest(
        new ListMemoryDreamTasksRequest({
          memoryStoreName: 'store_a',
          scope: new Scope({ appId: 'app_a' }),
          status: 'completed',
          minTimestamp: 1700000000000,
          maxTimestamp: 1700001000000,
          limit: 20,
          nextToken: 't',
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a' },
      status: 'completed',
      minTimestamp: 1700000000000,
      maxTimestamp: 1700001000000,
      limit: 20,
      nextToken: 't',
    });
  });

  it('serializes ListMemoryDreamActionsRequest with rich filters', () => {
    expect(
      toJSONRequest(
        new ListMemoryDreamActionsRequest({
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          status: 'proposed',
          action: 'UPDATE',
          targetScope: new Scope({ appId: 'app_a', tenantId: 'tenant_a' }),
          sourceMemoryId: 'mem-1',
          minConfidence: 0.5,
          maxConfidence: 0.95,
          orderBy: 'confidence_desc',
          limit: 50,
          nextToken: 'tok',
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      dreamId: 'dream-1',
      status: 'proposed',
      action: 'UPDATE',
      targetScope: { appId: 'app_a', tenantId: 'tenant_a' },
      sourceMemoryId: 'mem-1',
      minConfidence: 0.5,
      maxConfidence: 0.95,
      orderBy: 'confidence_desc',
      limit: 50,
      nextToken: 'tok',
    });
  });

  it('serializes ApplyMemoryDreamActionsRequest', () => {
    expect(
      toJSONRequest(
        new ApplyMemoryDreamActionsRequest({
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          actionIds: ['a-1', 'a-2'],
          applier: 'admin@example.com',
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      dreamId: 'dream-1',
      actionIds: ['a-1', 'a-2'],
      applier: 'admin@example.com',
    });

    expect(
      toJSONRequest(
        new ApplyMemoryDreamActionsRequest({
          memoryStoreName: 'store_a',
          dreamId: 'dream-1',
          actionIds: ['a-1'],
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      dreamId: 'dream-1',
      actionIds: ['a-1'],
    });
  });
});

describe('memory serialization: 0.0.12 fields', () => {
  it('serializes CreateMemoryStoreRequest storageMode and extractInstructions', () => {
    expect(
      toJSONRequest(
        new CreateMemoryStoreRequest({
          memoryStoreName: 'store_a',
          description: 'demo',
          extractInstructions: '只抽取与订单相关的事实',
          storageMode: 'filemem',
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      description: 'demo',
      extractInstructions: '只抽取与订单相关的事实',
      storageMode: 'filemem',
    });
  });

  it('omits UpdateMemoryStore fields that are absent but keeps empty strings', () => {
    expect(
      toJSONRequest(new UpdateMemoryStoreRequest({ memoryStoreName: 'store_a' }))
    ).toEqual({ memoryStoreName: 'store_a' });

    expect(
      toJSONRequest(
        new UpdateMemoryStoreRequest({ memoryStoreName: 'store_a', extractInstructions: '' })
      )
    ).toEqual({ memoryStoreName: 'store_a', extractInstructions: '' });

    expect(
      toJSONRequest(
        new UpdateMemoryStoreRequest({
          memoryStoreName: 'store_a',
          description: UNSET,
          extractInstructions: 'keep',
        })
      )
    ).toEqual({ memoryStoreName: 'store_a', extractInstructions: 'keep' });
  });

  it('serializes SearchMemoriesRequest contextScope, minSimilarity and includeEvidence', () => {
    expect(
      toJSONRequest(
        new SearchMemoriesRequest({
          memoryStoreName: 'store_a',
          query: 'coffee',
          scope: new Scope({ appId: 'app_a', tenantId: 't1' }),
          contextScope: new Scope({
            appId: 'app_a',
            tenantId: 't1',
            agentId: 'agent_a',
            runId: 'run_a',
          }),
          minSimilarity: 0.35,
          includeEvidence: true,
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      query: 'coffee',
      scope: { appId: 'app_a', tenantId: 't1' },
      contextScope: { appId: 'app_a', tenantId: 't1', agentId: 'agent_a', runId: 'run_a' },
      minSimilarity: 0.35,
      includeEvidence: true,
    });
  });

  it('serializes CreateMemoryDreamTaskRequest taskType and incremental', () => {
    expect(
      toJSONRequest(
        new CreateMemoryDreamTaskRequest({
          memoryStoreName: 'store_a',
          scopes: [new Scope({ appId: 'app_a' })],
          taskType: 'skill',
          incremental: true,
          confidenceThresholds: { add: 0.9, merge: 0.6 },
        })
      )
    ).toEqual({
      memoryStoreName: 'store_a',
      scopes: [{ appId: 'app_a' }],
      taskType: 'skill',
      incremental: true,
      confidenceThresholds: { add: 0.9, merge: 0.6 },
    });
  });

  it('serializes ListMemoryDreamActionsRequest in dreamId mode unchanged', () => {
    expect(
      toJSONRequest(
        new ListMemoryDreamActionsRequest({ memoryStoreName: 'store_a', dreamId: 'dream-1' })
      )
    ).toEqual({ memoryStoreName: 'store_a', dreamId: 'dream-1' });
  });

  it('serializes ListMemoryDreamActionsRequest in scope mode without a dreamId key', () => {
    const payload = toJSONRequest(
      new ListMemoryDreamActionsRequest({
        memoryStoreName: 'store_a',
        scope: new Scope({ appId: 'app_a', tenantId: 't1' }),
        actionType: 'EMIT_SKILL',
        limit: 20,
      })
    ) as Record<string, unknown>;

    expect(payload).toEqual({
      memoryStoreName: 'store_a',
      scope: { appId: 'app_a', tenantId: 't1' },
      actionType: 'EMIT_SKILL',
      limit: 20,
    });
    expect(Object.prototype.hasOwnProperty.call(payload, 'dreamId')).toBe(false);
  });
});
