import type { JsonRequestConvertible } from './base.js';
import { assignIfPresent } from './base.js';
import type { Unset } from './unset.js';

export class UpdateChunkItem implements JsonRequestConvertible {
  readonly chunkId: number;
  readonly docId?: string;
  readonly ossKey?: string;
  readonly title?: string | null | Unset;
  readonly content?: string | null | Unset;
  readonly status?: string;

  constructor(options: {
    chunkId: number;
    docId?: string;
    ossKey?: string;
    title?: string | null | Unset;
    content?: string | null | Unset;
    status?: string;
  }) {
    Object.assign(this, options);
    this.chunkId = options.chunkId;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { chunkId: this.chunkId };
    assignIfPresent(result, 'docId', this.docId);
    assignIfPresent(result, 'ossKey', this.ossKey);
    assignIfPresent(result, 'title', this.title);
    assignIfPresent(result, 'content', this.content);
    assignIfPresent(result, 'status', this.status);
    return result;
  }
}

export class UpdateChunksRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly chunks: UpdateChunkItem[];
  readonly subspace?: string;

  constructor(options: { knowledgeBaseName: string; chunks: UpdateChunkItem[]; subspace?: string }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
    this.chunks = options.chunks;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      knowledgeBaseName: this.knowledgeBaseName,
      chunks: this.chunks.map((item) => item.toJSONRequest()),
    };
    assignIfPresent(result, 'subspace', this.subspace);
    return result;
  }
}

export class ListChunksRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly subspace?: string;
  readonly docId?: string;
  readonly ossKey?: string;
  readonly maxResults?: number;
  readonly nextToken?: string;

  constructor(options: {
    knowledgeBaseName: string;
    subspace?: string;
    docId?: string;
    ossKey?: string;
    maxResults?: number;
    nextToken?: string;
  }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { knowledgeBaseName: this.knowledgeBaseName };
    assignIfPresent(result, 'subspace', this.subspace);
    assignIfPresent(result, 'docId', this.docId);
    assignIfPresent(result, 'ossKey', this.ossKey);
    assignIfPresent(result, 'maxResults', this.maxResults);
    assignIfPresent(result, 'nextToken', this.nextToken);
    return result;
  }
}
