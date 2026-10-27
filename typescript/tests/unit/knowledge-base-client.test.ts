import { describe, expect, it, vi } from 'vitest';

import { AgentStorageClient } from '../../src/index.js';
import { RequestContext } from '../../src/core/auth/request-context.js';
import { SignV2 } from '../../src/core/auth/sign-v2.js';
import { md5Base64 } from '../../src/core/http/md5.js';

function buildSuccessHeaders(body: string, accessKeyId: string, accessKeySecret: string) {
  const responseHeaders = {
    'x-ots-contentmd5': md5Base64(new TextEncoder().encode(body)),
    'x-ots-requestid': 'req-1',
    'x-ots-date': '2025-04-10T01:02:03.000Z',
    'x-ots-contenttype': 'application/json',
  };
  const signer = new SignV2();
  const requestContext = new RequestContext({ accessKeyId, accessKeySecret });
  const signature = signer.makeResponseSignature('/CreateKnowledgeBase', responseHeaders, signer.getSigningKey(requestContext));
  return {
    ...responseHeaders,
    authorization: `OTS ${accessKeyId}:${signature}`,
  };
}

describe('AgentStorageClient knowledge base methods', () => {
  it('sends CreateKnowledgeBase as a POST JSON request with V2 headers by default', async () => {
    const responseBody = JSON.stringify({ code: 'SUCCESS', data: {} });
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(responseBody, {
        status: 200,
        headers: buildSuccessHeaders(responseBody, 'ak', 'sk'),
      })
    );
    const client = new AgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      fetch: fetchMock,
    });

    await client.createKnowledgeBase({ knowledgeBaseName: 'kb' });

    expect(fetchMock).toHaveBeenCalledWith(
      'https://example.com/CreateKnowledgeBase',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          'x-ots-accesskeyid': 'ak',
          'x-ots-instancename': 'instance',
          'x-ots-signature': expect.any(String),
        }),
        body: JSON.stringify({ knowledgeBaseName: 'kb' }),
      })
    );
  });
});
