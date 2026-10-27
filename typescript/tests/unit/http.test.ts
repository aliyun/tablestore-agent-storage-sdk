import { describe, expect, it, vi } from 'vitest';

import { OTSClientError, OTSServiceError } from '../../src/index.js';
import { OTSJSONClient } from '../../src/core/http/ots-json-client.js';

function makeResponse(body: Record<string, unknown>, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: {
      'x-ots-contentmd5': 'x',
      'x-ots-requestid': 'req-1',
      'x-ots-date': '2025-04-10T01:02:03.000Z',
      'x-ots-contenttype': 'application/json',
      authorization: 'OTS ak:signature',
      ...(init?.headers ?? {}),
    },
    ...init,
  });
}

describe('OTSJSONClient', () => {
  it('maps network errors to OTSClientError', async () => {
    const client = new OTSJSONClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      credentials: { accessKeyId: 'ak', accessKeySecret: 'sk' },
      fetch: vi.fn().mockRejectedValue(new Error('network down')),
    });

    await expect(client.request('CreateKnowledgeBase', { knowledgeBaseName: 'kb' })).rejects.toBeInstanceOf(
      OTSClientError
    );
  });

  it('maps non-2xx responses to OTSServiceError when authorization is absent', async () => {
    const client = new OTSJSONClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      credentials: { accessKeyId: 'ak', accessKeySecret: 'sk' },
      fetch: vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 'Forbidden', message: 'denied' }), {
          status: 403,
          headers: {
            'x-ots-requestid': 'req-2',
          },
        })
      ),
    });

    await expect(client.request('CreateKnowledgeBase', { knowledgeBaseName: 'kb' })).rejects.toMatchObject({
      name: OTSServiceError.name,
      statusCode: 403,
    });
  });
});
