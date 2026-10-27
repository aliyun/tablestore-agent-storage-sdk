import { describe, expect, it } from 'vitest';

import { RequestContext } from '../../src/core/auth/request-context.js';
import { SignV2 } from '../../src/core/auth/sign-v2.js';
import { SignV4 } from '../../src/core/auth/sign-v4.js';

describe('auth fixtures', () => {
  const testAkId = 'test_id';
  const testAkSecret = 'test_key';
  const testStsToken = 'test_token';
  const testRegion = 'test-region';
  const testSignDate = '20250410';
  const testQuery = 'test_query';
  const headers: Record<string, string> = { 'x-ots-test': 'test' };

  it('produces the Python V2 request and response signature fixtures', () => {
    const requestContext = new RequestContext(
      { accessKeyId: testAkId, accessKeySecret: testAkSecret, securityToken: testStsToken },
      testSignDate
    );
    const signer = new SignV2();
    const requestHeaders = { ...headers };
    signer.makeRequestSignatureAndAddHeaders(testQuery, requestHeaders, requestContext);
    expect(requestHeaders['x-ots-signature']).toBe('QDhzLv7VESBJtYQY4Li0IhSUOdg=');
    expect(signer.makeResponseSignature(testQuery, headers, signer.getSigningKey(requestContext))).toBe(
      'UjJK/SWed0n9o6JYxvApHGaQABo='
    );
  });

  it('produces the Python V4 request and response signature fixtures', () => {
    const requestContext = new RequestContext(
      { accessKeyId: testAkId, accessKeySecret: testAkSecret, securityToken: testStsToken },
      testSignDate
    );
    const signer = new SignV4({ region: testRegion, signDate: testSignDate });
    const requestHeaders = { ...headers };
    signer.makeRequestSignatureAndAddHeaders(testQuery, requestHeaders, requestContext);
    expect(requestHeaders['x-ots-signregion']).toBe(testRegion);
    expect(requestHeaders['x-ots-signdate']).toBe(testSignDate);
    expect(requestHeaders['x-ots-signaturev4']).toBe('yXnOpODWaU1EYAlLP3l25ksj010uGHS7uxIt5Qiwz4o=');
    expect(Buffer.from(signer.getSigningKey(requestContext)).toString('utf8')).toBe(
      'nToxlXrxgCm0L5J0nr/qq/GmtgN9GVBhiRLzdLaVUP0='
    );
    expect(signer.makeResponseSignature(testQuery, headers, signer.getSigningKey(requestContext))).toBe(
      'vIhaUGwv/JSg8ctLNyxbNeNv69A='
    );
  });
});
