import type { RequestContext } from './request-context.js';
import { createHmac } from 'node:crypto';

import { OTSClientError } from '../errors/client-error.js';
import { SignBase, hmacSha256Base64 } from './sign-base.js';

const V4_SIGNATURE_PRODUCT = 'ots';
const V4_SIGNATURE_SEED = 'aliyun';
const V4_SIGNATURE_PREFIX = `${V4_SIGNATURE_SEED}_v4`;
const V4_SIGNATURE_CONSTANT = `${V4_SIGNATURE_SEED}_v4_request`;
const V4_SIGNATURE_SALT = 'ots';

function hmacSha256Bytes(signingKey: string | Buffer, content: string): Buffer {
  return createHmac('sha256', signingKey).update(content, 'utf8').digest();
}

export class SignV4 extends SignBase {
  readonly signRegion: string;
  readonly signDate?: string;
  readonly autoUpdateV4Sign: boolean;

  constructor(options: { region?: string; signDate?: string; autoUpdateV4Sign?: boolean }) {
    super();
    if (!options.region) {
      throw new OTSClientError('region is not str or is empty.');
    }
    this.signRegion = options.region;
    this.signDate = options.signDate;
    this.autoUpdateV4Sign = options.autoUpdateV4Sign ?? options.signDate === undefined;
  }

  private getAndSetV4SignDate(requestContext: RequestContext): string {
    const currentDate = this.autoUpdateV4Sign
      ? new Date().toISOString().slice(0, 10).replaceAll('-', '')
      : this.signDate;
    const signDate = this.requireSignDate(currentDate);
    requestContext.setSignDate(signDate);
    return signDate;
  }

  getSigningKey(requestContext: RequestContext): Buffer {
    const accessKeySecret = requestContext.getCredentials().accessKeySecret;
    const signDate = this.requireSignDate(requestContext.getSignDate());
    const first = hmacSha256Bytes(`${V4_SIGNATURE_PREFIX}${accessKeySecret}`, signDate);
    const second = hmacSha256Bytes(first, this.signRegion);
    const third = hmacSha256Bytes(second, V4_SIGNATURE_PRODUCT);
    const fourth = hmacSha256Bytes(third, V4_SIGNATURE_CONSTANT);
    return Buffer.from(fourth.toString('base64'), 'utf8');
  }

  makeRequestSignatureAndAddHeaders(
    query: string,
    headers: Record<string, string>,
    requestContext: RequestContext
  ): void {
    headers['x-ots-signdate'] = this.getAndSetV4SignDate(requestContext);
    headers['x-ots-signregion'] = this.signRegion;
    headers['x-ots-signaturev4'] = hmacSha256Base64(
      this.getSigningKey(requestContext),
      `${this.getRequestSignatureString(query, headers)}${V4_SIGNATURE_SALT}`
    );
  }
}
