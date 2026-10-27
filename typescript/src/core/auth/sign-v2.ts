import type { RequestContext } from './request-context.js';
import { SignBase, hmacSha1Base64 } from './sign-base.js';

export class SignV2 extends SignBase {
  getSigningKey(requestContext: RequestContext): string {
    return requestContext.getCredentials().accessKeySecret;
  }

  makeRequestSignatureAndAddHeaders(
    query: string,
    headers: Record<string, string>,
    requestContext: RequestContext
  ): void {
    headers['x-ots-signature'] = hmacSha1Base64(
      this.getSigningKey(requestContext),
      this.getRequestSignatureString(query, headers)
    );
  }
}
