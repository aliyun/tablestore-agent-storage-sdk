import { describe, expect, it, vi } from 'vitest';

import { AgentStorageClient } from '../../src/index.js';

function newClient(fetchMock: ReturnType<typeof vi.fn>) {
  return new AgentStorageClient({
    endpoint: 'https://example.com',
    instanceName: 'inst',
    apiKey: 'key123',
    fetch: fetchMock as typeof globalThis.fetch,
  });
}

function jsonResponse(payload: Record<string, unknown>, status = 200) {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

const scope = { appId: 'app', tenantId: 'tenant', agentId: 'agent', runId: 'run' };
const item = {
  type: 'memoryfile',
  itemId: 'm1',
  scope,
  path: '/a.md',
  content: 'hello',
  contentSha256: 'sha-2',
  contentSizeBytes: 5,
  latestSeq: 2,
  latestVersionId: 'mv2',
  createdAt: '2026-07-16T01:00:00Z',
  updatedAt: '2026-07-16T02:00:00Z',
};
const version = {
  type: 'memoryfile',
  versionId: 'mv1',
  itemId: 'm1',
  scope,
  operation: 'modified',
  path: '/b.md',
  content: 'hello',
  contentSha256: 'sha-2',
  contentSizeBytes: 5,
  createdBy: { actorType: 'session', actorId: 's1' },
  createdAt: '2026-07-16T02:00:00Z',
};
const redactedVersion = {
  ...version,
  path: '',
  content: undefined,
  contentSha256: '',
  contentSizeBytes: 0,
  redacted: true,
  redactedAt: '2026-07-18T03:00:00Z',
  redactedBy: { actorType: 'session', actorId: 'redactor-1' },
};

describe('item JSON APIs', () => {
  it('sends all eight actions through the standard JSON request path', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(item))
      .mockResolvedValueOnce(jsonResponse(item))
      .mockResolvedValueOnce(jsonResponse({ ...item, path: '/b.md' }))
      .mockResolvedValueOnce(jsonResponse({ ...item, path: '/b.md' }))
      .mockResolvedValueOnce(jsonResponse({ type: 'memoryfile', itemId: 'm1', deleted: true }))
      .mockResolvedValueOnce(jsonResponse({ type: 'memoryfile', taskId: 'task-1', status: 'pending' }, 200))
      .mockResolvedValueOnce(jsonResponse({ ...item, path: '/b.md' }))
      .mockResolvedValueOnce(jsonResponse({ ...item, path: '/b.md', content: undefined }))
      .mockResolvedValueOnce(jsonResponse({
        type: 'memoryfile',
        items: [
          { ...item, type: 'memoryfile', content: undefined },
          { type: 'memoryfile_prefix', scope, path: '/notes/sub/' },
        ],
        nextToken: 'n2',
        readOnly: true,
      }))
      .mockResolvedValueOnce(jsonResponse({ type: 'memoryfile', versions: [{ ...version, type: undefined, content: undefined }], nextToken: 'v2' }))
      .mockResolvedValueOnce(jsonResponse(version))
      .mockResolvedValueOnce(jsonResponse(redactedVersion));
    const client = newClient(fetchMock);

    const created = await client.addItem({
      memoryStoreName: 'store', scope, path: '/a.md', content: 'hello', sessionId: 's1', view: 'full',
    });
    await client.updateItem({
      memoryStoreName: 'store', scope, itemId: 'm1', content: 'updated',
      precondition: { expectedVersionId: 'mv1' }, sessionId: 's1',
    });
    await client.updateItem({
      memoryStoreName: 'store', scope, itemId: 'm1', path: '/b.md', overwrite: true,
      precondition: { expectedVersionId: 'mv2' }, sessionId: 's1',
    });
    await client.updateItem({
      memoryStoreName: 'store', scope, itemId: 'm1', content: 'composite', path: '/c.md',
    });
    const deleted = await client.deleteItem({
      memoryStoreName: 'store', scope, path: '/b.md',
      precondition: { expectedVersionId: 'mv3' }, sessionId: 's1',
    });
    const ranged = await client.deleteItem({
      memoryStoreName: 'store',
      scope: { appId: 'app', tenantId: 'tenant', agentId: '*', runId: '*' },
      pathPrefix: '/notes/',
      sessionId: 's1',
    });
    const read = await client.getItem({ memoryStoreName: 'store', scope, path: '/b.md', view: 'full' });
    const stat = await client.getItem({ memoryStoreName: 'store', scope, itemId: 'm1' });
    const listed = await client.listItems({
      memoryStoreName: 'store', scope, pathPrefix: '/notes/', depth: 1, nextToken: 'n1', limit: 20,
    });
    const versions = await client.listItemVersions({
      memoryStoreName: 'store', scope, itemId: 'm1', sessionId: 's1', operation: 'modified',
      createdAtGte: '2026-07-01T00:00:00Z', createdAtLte: '2026-07-31T23:59:59Z',
      nextToken: 'v1', limit: 10,
    });
    const selectedVersion = await client.getItemVersion({
      memoryStoreName: 'store', scope, versionId: 'mv1', view: 'full',
    });
    const redacted = await client.redactItemVersion({
      memoryStoreName: 'store', scope, versionId: 'mv1', sessionId: 'redactor-1',
    });

    const expectedCalls = [
      ['AddItem', { type: 'memoryfile', memoryStoreName: 'store', scope, path: '/a.md', content: 'hello', sessionId: 's1', view: 'full' }],
      ['UpdateItem', { type: 'memoryfile', memoryStoreName: 'store', scope, itemId: 'm1', content: 'updated', precondition: { expectedVersionId: 'mv1' }, sessionId: 's1' }],
      ['UpdateItem', { type: 'memoryfile', memoryStoreName: 'store', scope, itemId: 'm1', path: '/b.md', overwrite: true, precondition: { expectedVersionId: 'mv2' }, sessionId: 's1' }],
      ['UpdateItem', { type: 'memoryfile', memoryStoreName: 'store', scope, itemId: 'm1', content: 'composite', path: '/c.md' }],
      ['DeleteItem', { type: 'memoryfile', memoryStoreName: 'store', scope, path: '/b.md', precondition: { expectedVersionId: 'mv3' }, sessionId: 's1' }],
      ['DeleteItem', { type: 'memoryfile', memoryStoreName: 'store', scope: { appId: 'app', tenantId: 'tenant', agentId: '*', runId: '*' }, pathPrefix: '/notes/', sessionId: 's1' }],
      ['GetItem', { type: 'memoryfile', memoryStoreName: 'store', scope, path: '/b.md', view: 'full' }],
      ['GetItem', { type: 'memoryfile', memoryStoreName: 'store', scope, itemId: 'm1' }],
      ['ListItems', { type: 'memoryfile', memoryStoreName: 'store', scope, pathPrefix: '/notes/', depth: 1, nextToken: 'n1', limit: 20 }],
      ['ListItemVersions', { type: 'memoryfile', memoryStoreName: 'store', scope, itemId: 'm1', sessionId: 's1', operation: 'modified', createdAtGte: '2026-07-01T00:00:00Z', createdAtLte: '2026-07-31T23:59:59Z', nextToken: 'v1', limit: 10 }],
      ['GetItemVersion', { type: 'memoryfile', memoryStoreName: 'store', scope, versionId: 'mv1', view: 'full' }],
      ['RedactItemVersion', { type: 'memoryfile', memoryStoreName: 'store', scope, versionId: 'mv1', sessionId: 'redactor-1' }],
    ] as const;

    expectedCalls.forEach(([action, body], index) => {
      expect(fetchMock).toHaveBeenNthCalledWith(
        index + 1,
        `https://example.com/${action}`,
        expect.objectContaining({ method: 'POST', body: JSON.stringify(body) })
      );
      const headers = fetchMock.mock.calls[index][1].headers as Record<string, string>;
      expect(headers['If-None-Match']).toBeUndefined();
    });

    expect(created).toMatchObject({ type: 'memoryfile', itemId: 'm1', content: 'hello', latestVersionId: 'mv2' });
    expect(created.scope).toEqual(scope);
    expect(deleted).toMatchObject({ type: 'memoryfile', itemId: 'm1', deleted: true });
    expect(ranged).toMatchObject({ type: 'memoryfile', taskId: 'task-1', status: 'pending' });
    expect(read.contentSha256).toBe('sha-2');
    expect(stat.content).toBeUndefined();
    expect(listed).toMatchObject({ type: 'memoryfile', nextToken: 'n2', readOnly: true });
    expect(listed.items).toHaveLength(2);
    expect(listed.items[1]).toMatchObject({ type: 'memoryfile_prefix', path: '/notes/sub/' });
    expect(versions.versions[0].operation).toBe('modified');
    expect(versions.versions[0].createdBy).toEqual({ actorType: 'session', actorId: 's1' });
    expect(selectedVersion).toMatchObject({ versionId: 'mv1', itemId: 'm1', content: 'hello' });
    expect(redacted).toMatchObject({
      versionId: 'mv1', redacted: true, redactedAt: '2026-07-18T03:00:00Z',
      redactedBy: { actorType: 'session', actorId: 'redactor-1' },
    });
    expect(redacted.content).toBeUndefined();
  });

  it('keeps an explicitly provided item type', async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(jsonResponse(item));
    const client = newClient(fetchMock);

    await client.getItem({ type: 'customtype', memoryStoreName: 'store', scope, path: '/a.md' });

    expect(fetchMock).toHaveBeenCalledWith(
      'https://example.com/GetItem',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ type: 'customtype', memoryStoreName: 'store', scope, path: '/a.md' }),
      })
    );
  });
});

describe('item request null handling', () => {
  it('does not throw on a null/undefined request and still injects the type', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ type: 'memoryfile', items: [] }));
    const client = newClient(fetchMock);
    await client.listItems(undefined as never);
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(init.body as string)).toEqual({ type: 'memoryfile' });
  });
});
