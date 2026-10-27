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

describe('AgentStorageClient memory store methods', () => {
  it('sends CreateMemoryStore to the memory endpoint with JSON payload', async () => {
    const fetchMock = vi.fn().mockResolvedValue(successResponse('/CreateMemoryStore', { memoryStoreName: 'store_a' }));
    const client = new AgentStorageClient(baseOptions(fetchMock as typeof globalThis.fetch));

    await client.createMemoryStore({ memoryStoreName: 'store_a' });

    expect(fetchMock).toHaveBeenCalledWith(
      'https://example.com/CreateMemoryStore',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ memoryStoreName: 'store_a' }),
      })
    );
  });

  it('sends ListMemoryStores and UpdateMemoryStore', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(successResponse('/ListMemoryStores', { stores: [] }))
      .mockResolvedValueOnce(successResponse('/UpdateMemoryStore', { memoryStoreName: 'store_a', description: 'after' }));
    const client = new AgentStorageClient(baseOptions(fetchMock as typeof globalThis.fetch));

    await client.listMemoryStores({ limit: 10 });
    await client.updateMemoryStore({ memoryStoreName: 'store_a', description: 'after' });

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://example.com/ListMemoryStores',
      expect.objectContaining({ body: JSON.stringify({ limit: 10 }) })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://example.com/UpdateMemoryStore',
      expect.objectContaining({ body: JSON.stringify({ memoryStoreName: 'store_a', description: 'after' }) })
    );
  });
});
