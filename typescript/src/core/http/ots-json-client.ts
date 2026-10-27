import { OTSClientError } from '../errors/client-error.js';
import { OTSServiceError } from '../errors/service-error.js';
import type { Credentials, RequestContext } from '../auth/request-context.js';
import { SignV2 } from '../auth/sign-v2.js';
import { SignV4 } from '../auth/sign-v4.js';
import type { SignBase } from '../auth/sign-base.js';
import { buildRequestHeaders } from './headers.js';
import { md5Base64 } from './md5.js';
import { RequestContext as RequestContextImpl } from '../auth/request-context.js';
import { DEFAULT_USER_AGENT } from '../version.js';

export type SignVersion = 'v2' | 'v4';

export interface OTSJSONClientOptions {
  endpoint: string;
  instanceName: string;
  credentials?: Credentials;
  apiKey?: string;
  signVersion?: SignVersion;
  region?: string;
  fetch: typeof globalThis.fetch;
  userAgent?: string;
  extraHeaders?: Record<string, string>;
}

function normalizeHeaders(headers: Headers): Record<string, string> {
  const normalized: Record<string, string> = {};
  headers.forEach((value, key) => {
    normalized[key.toLowerCase()] = value;
  });
  return normalized;
}

export class OTSJSONClient {
  private readonly endpoint: string;
  private readonly instanceName: string;
  private readonly credentials?: Credentials;
  private readonly apiKey?: string;
  private readonly fetchImpl: typeof globalThis.fetch;
  private readonly signer?: SignBase;
  private readonly userAgent: string;
  private readonly extraHeaders?: Record<string, string>;

  constructor(options: OTSJSONClientOptions) {
    if (!options.credentials && !options.apiKey) {
      throw new OTSClientError('Either credentials (accessKeyId/accessKeySecret) or apiKey must be provided.');
    }
    if (options.apiKey && !options.endpoint.startsWith('https://')) {
      throw new OTSClientError('API key authentication requires HTTPS. Endpoint must start with https://.');
    }
    this.endpoint = options.endpoint.replace(/\/+$/, '');
    this.instanceName = options.instanceName;
    this.credentials = options.credentials;
    this.apiKey = options.apiKey;
    this.fetchImpl = options.fetch;
    this.userAgent = options.userAgent ?? DEFAULT_USER_AGENT;
    this.extraHeaders = options.extraHeaders;
    if (options.credentials) {
      this.signer =
        (options.signVersion ?? 'v2') === 'v4'
          ? new SignV4({ region: options.region })
          : new SignV2();
    }
  }

  private buildHeaders(action: string, body: string): { headers: Record<string, string>; requestContext?: RequestContext } {
    let headers: Record<string, string>;
    let requestContext: RequestContext | undefined;

    if (this.apiKey) {
      headers = {
        'x-ots-instancename': this.instanceName,
        'x-ots-apikey': this.apiKey,
        'Content-Type': 'application/json',
        'User-Agent': this.userAgent,
      };
      if (this.extraHeaders) {
        for (const [key, value] of Object.entries(this.extraHeaders)) {
          headers[key] = value;
        }
      }
    } else {
      requestContext = new RequestContextImpl(this.credentials!);
      headers = buildRequestHeaders({
        body: new TextEncoder().encode(body),
        query: `/${action}`,
        instanceName: this.instanceName,
        credentials: this.credentials!,
        signer: this.signer!,
        requestContext,
        userAgent: this.userAgent,
        extraHeaders: this.extraHeaders,
      });
      headers['Content-Type'] = 'application/json';
    }
    return { headers, requestContext };
  }

  async request<T>(action: string, payload: Record<string, unknown>): Promise<T> {
    if (!this.fetchImpl) {
      throw new OTSClientError('fetch implementation is required.');
    }

    const body = JSON.stringify(payload);
    const { headers, requestContext } = this.buildHeaders(action, body);

    let response: Response;
    try {
      response = await this.fetchImpl(`${this.endpoint}/${action}`, {
        method: 'POST',
        headers,
        body,
      });
    } catch (error) {
      throw new OTSClientError(`Failed to send request: ${String(error)}`);
    }

    const responseText = await response.text();
    const responseBytes = new TextEncoder().encode(responseText);
    const responseHeaders = normalizeHeaders(response.headers);

    if (!this.apiKey) {
      this.validateResponseHeaders(responseHeaders, responseBytes, response.status, requestContext!, `/${action}`);
    }

    if (!response.ok) {
      let errorPayload: Record<string, unknown> = {};
      try {
        errorPayload = responseText ? (JSON.parse(responseText) as Record<string, unknown>) : {};
      } catch {
        errorPayload = {};
      }
      throw new OTSServiceError(String(errorPayload.message ?? response.statusText), {
        statusCode: response.status,
        requestId: responseHeaders['x-ots-requestid'],
        code: typeof errorPayload.code === 'string' ? errorPayload.code : undefined,
      });
    }

    try {
      return JSON.parse(responseText) as T;
    } catch (error) {
      throw new OTSClientError(`Failed to parse JSON response: ${String(error)}`);
    }
  }

  private validateResponseHeaders(
    headers: Record<string, string>,
    body: Uint8Array,
    status: number,
    requestContext: RequestContext,
    query: string
  ) {
    if (status >= 200 && status < 300) {
      for (const header of ['x-ots-contentmd5', 'x-ots-requestid', 'x-ots-date', 'x-ots-contenttype']) {
        if (!(header in headers)) {
          throw new OTSClientError(`"${header}" is missing in response header.`);
        }
      }
    }

    if (headers['x-ots-contentmd5'] && md5Base64(body) !== headers['x-ots-contentmd5']) {
      throw new OTSClientError('MD5 mismatch in response.');
    }

    const authorization = headers.authorization;
    if (!authorization) {
      if (status >= 200 && status < 300) {
        throw new OTSClientError('"Authorization" is missing in response header.');
      }
      return;
    }

    if (!authorization.startsWith('OTS ')) {
      throw new OTSClientError('Invalid Authorization in response.');
    }

    const [accessKeyId, signature] = authorization.slice(4).split(':');
    if (accessKeyId !== this.credentials!.accessKeyId) {
      throw new OTSClientError('Invalid access key id in response.');
    }

    if (signature !== this.signer!.makeResponseSignature(query, headers, this.signer!.getSigningKey(requestContext))) {
      throw new OTSClientError('Invalid signature in response.');
    }
  }
}
