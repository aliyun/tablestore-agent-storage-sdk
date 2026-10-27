import { md5Base64 } from './md5.js';
import type { Credentials, RequestContext } from '../auth/request-context.js';
import type { SignBase } from '../auth/sign-base.js';
import { DEFAULT_USER_AGENT } from '../version.js';

const API_VERSION = '2015-12-31';

export function buildDateHeader(date = new Date()): string {
  const iso = date.toISOString();
  return `${iso.slice(0, 19)}.000Z`;
}

export function buildRequestHeaders(options: {
  body: Uint8Array;
  query: string;
  instanceName: string;
  credentials: Credentials;
  signer: SignBase;
  requestContext: RequestContext;
  userAgent?: string;
  date?: Date;
  extraHeaders?: Record<string, string>;
}): Record<string, string> {
  const headers: Record<string, string> = {
    'x-ots-date': buildDateHeader(options.date),
    'x-ots-apiversion': API_VERSION,
    'x-ots-accesskeyid': options.credentials.accessKeyId,
    'x-ots-instancename': options.instanceName,
    'x-ots-contentmd5': md5Base64(options.body),
  };

  if (options.credentials.securityToken) {
    headers['x-ots-ststoken'] = options.credentials.securityToken;
  }

  if (options.extraHeaders) {
    for (const [key, value] of Object.entries(options.extraHeaders)) {
      headers[key] = value;
    }
  }

  options.signer.makeRequestSignatureAndAddHeaders(options.query, headers, options.requestContext);
  headers['User-Agent'] = options.userAgent ?? DEFAULT_USER_AGENT;
  return headers;
}
