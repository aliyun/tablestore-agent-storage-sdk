import { describe, expect, it, vi } from 'vitest';

import { AgentStorageClient } from '../../src/index.js';

describe('API key authentication', () => {
  it('sends x-ots-apikey header instead of signature when apiKey is configured', async () => {
    const responseBody = JSON.stringify({ stores: [] });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(responseBody, { status: 200, headers: { 'content-type': 'application/json' } })
    );

    const client = new AgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'test-instance',
      apiKey: 'test-key',
      fetch: fetchMock as typeof globalThis.fetch,
    });

    await client.listMemoryStores({});

    expect(fetchMock).toHaveBeenCalledWith(
      'https://example.com/ListMemoryStores',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({}),
      })
    );

    const callArgs = fetchMock.mock.calls[0];
    const headers = callArgs[1].headers as Record<string, string>;
    expect(headers['x-ots-apikey']).toBe('test-key');
    expect(headers['x-ots-instancename']).toBe('test-instance');
    expect(headers['Content-Type']).toBe('application/json');
    expect(headers['x-ots-accesskeyid']).toBeUndefined();
    expect(headers['authorization']).toBeUndefined();
  });

  it('rejects API key with non-HTTPS endpoint', () => {
    expect(
      () =>
        new AgentStorageClient({
          endpoint: 'http://example.com',
          instanceName: 'test',
          apiKey: 'key',
        })
    ).toThrow('API key authentication requires HTTPS');
  });

  it('rejects when neither credentials nor apiKey is provided', () => {
    expect(
      () =>
        new AgentStorageClient({
          endpoint: 'https://example.com',
          instanceName: 'test',
        })
    ).toThrow('Either credentials (accessKeyId/accessKeySecret) or apiKey must be provided');
  });

  it('skips response signature validation in apiKey mode', async () => {
    const responseBody = JSON.stringify({ memoryStoreName: 'store_a' });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(responseBody, { status: 200, headers: { 'content-type': 'application/json' } })
    );

    const client = new AgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'inst',
      apiKey: 'key123',
      fetch: fetchMock as typeof globalThis.fetch,
    });

    const result = await client.createMemoryStore({ memoryStoreName: 'store_a' });
    expect(result).toEqual({ memoryStoreName: 'store_a' });
  });

  it('throws OTSServiceError on non-2xx response in apiKey mode', async () => {
    const responseBody = JSON.stringify({ code: 'VALIDATION_ERROR', message: 'bad request' });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(responseBody, { status: 400, headers: { 'x-ots-requestid': 'req-1' } })
    );

    const client = new AgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'inst',
      apiKey: 'key123',
      fetch: fetchMock as typeof globalThis.fetch,
    });

    await expect(client.createMemoryStore({ memoryStoreName: '' })).rejects.toThrow('bad request');
  });

  it('includes extraHeaders in apiKey mode requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response('{}', { status: 200, headers: { 'content-type': 'application/json' } })
    );

    const client = new AgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'inst',
      apiKey: 'key123',
      extraHeaders: { 'x-custom': 'value' },
      fetch: fetchMock as typeof globalThis.fetch,
    });

    await client.listMemoryStores({});

    const headers = fetchMock.mock.calls[0][1].headers as Record<string, string>;
    expect(headers['x-custom']).toBe('value');
  });
});
