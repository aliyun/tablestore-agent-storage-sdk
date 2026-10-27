import { createHmac } from 'node:crypto';

import { OTSClientError } from '../errors/client-error.js';
import type { RequestContext } from './request-context.js';

const OTS_HEADER_PREFIX = 'x-ots-';

export function hmacSha1Base64(signingKey: string | Buffer, content: string): string {
  return createHmac('sha1', signingKey).update(content, 'utf8').digest('base64');
}

export function hmacSha256Base64(signingKey: string | Buffer, content: string): string {
  return createHmac('sha256', signingKey).update(content, 'utf8').digest('base64');
}

export function sortAndEncodeQuery(queryString: string): string {
  if (!queryString) {
    return '';
  }

  const pairs = [...new URLSearchParams(queryString).entries()].sort(([leftKey, leftValue], [rightKey, rightValue]) => {
    const keyComparison = leftKey.localeCompare(rightKey);
    if (keyComparison !== 0) {
      return keyComparison;
    }
    return leftValue.localeCompare(rightValue);
  });
  return new URLSearchParams(pairs).toString();
}

function splitQuery(query: string): { uri: string; queryString: string } {
  const hashIndex = query.indexOf('#');
  const hashStripped = hashIndex >= 0 ? query.slice(0, hashIndex) : query;
  const queryIndex = hashStripped.indexOf('?');

  if (queryIndex < 0) {
    return {
      uri: hashStripped,
      queryString: '',
    };
  }

  return {
    uri: hashStripped.slice(0, queryIndex),
    queryString: hashStripped.slice(queryIndex + 1),
  };
}

export abstract class SignBase {
  protected makeHeadersString(headers: Record<string, string>): string {
    const headerItems = Object.entries(headers)
      .filter(([key]) => key.toLowerCase().startsWith(OTS_HEADER_PREFIX))
      .map(([key, value]) => `${key.toLowerCase()}:${value.trim()}`)
      .sort();
    return headerItems.join('\n');
  }

  protected getRequestSignatureString(query: string, headers: Record<string, string>): string {
    const { uri, queryString } = splitQuery(query);
    const sortedQuery = sortAndEncodeQuery(queryString);
    const headersString = this.makeHeadersString(headers);
    return `${uri}\nPOST\n${sortedQuery}\n${headersString}\n`;
  }

  makeResponseSignature(query: string, headers: Record<string, string>, signingKey: string | Buffer): string {
    const { uri } = splitQuery(query);
    const headersString = this.makeHeadersString(headers);
    return hmacSha1Base64(signingKey, `${headersString}\n${uri}`);
  }

  protected requireSignDate(signDate: string | undefined): string {
    if (!signDate) {
      throw new OTSClientError('v4 sign_date is None.');
    }
    return signDate;
  }

  abstract getSigningKey(requestContext: RequestContext): string | Buffer;
  abstract makeRequestSignatureAndAddHeaders(
    query: string,
    headers: Record<string, string>,
    requestContext: RequestContext
  ): void;
}
