import { describe, expect, it } from 'vitest';

import { RequestContext } from '../../src/core/auth/request-context.js';
import { SignV2 } from '../../src/core/auth/sign-v2.js';
import { buildDateHeader, buildRequestHeaders } from '../../src/core/http/headers.js';

describe('headers', () => {
  it('formats the OTS date header like the Python SDK', () => {
    expect(buildDateHeader(new Date('2025-04-10T01:02:03.456Z'))).toBe('2025-04-10T01:02:03.000Z');
  });

  it('builds signed request headers with md5', () => {
    const requestContext = new RequestContext({ accessKeyId: 'ak', accessKeySecret: 'sk' });
    const headers = buildRequestHeaders({
      body: new TextEncoder().encode('hello'),
      query: '/CreateKnowledgeBase',
      instanceName: 'instance',
      credentials: { accessKeyId: 'ak', accessKeySecret: 'sk' },
      signer: new SignV2(),
      requestContext,
      userAgent: 'test-agent',
      date: new Date('2025-04-10T01:02:03.456Z'),
    });

    expect(headers['x-ots-apiversion']).toBe('2015-12-31');
    expect(headers['x-ots-instancename']).toBe('instance');
    expect(headers['User-Agent']).toBe('test-agent');
    expect(headers['x-ots-contentmd5']).toBe('XUFAKrxLKna5cZ2REBfFkg==');
    expect(headers['x-ots-signature']).toBeTypeOf('string');
  });
});
