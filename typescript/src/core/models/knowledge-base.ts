import { MetadataFieldType } from './enums.js';
import type { JsonRequestConvertible } from './base.js';
import { assignIfPresent, toJsonValue } from './base.js';
import type { Unset } from './unset.js';

export class MetadataField implements JsonRequestConvertible {
  readonly name: string;
  readonly type: MetadataFieldType | string;

  constructor(options: { name: string; type: MetadataFieldType | string }) {
    this.name = options.name;
    this.type = options.type;
  }

  toJSONRequest() {
    return {
      name: this.name,
      type: this.type,
    };
  }
}

export class EmbeddingConfiguration implements JsonRequestConvertible {
  readonly provider?: string;
  readonly url?: string;
  readonly apiKey?: string;
  readonly model?: string;
  readonly dimension?: number;

  constructor(options: {
    provider?: string;
    url?: string;
    apiKey?: string;
    model?: string;
    dimension?: number;
  }) {
    Object.assign(this, options);
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {};
    assignIfPresent(result, 'provider', this.provider);
    assignIfPresent(result, 'url', this.url);
    assignIfPresent(result, 'apiKey', this.apiKey);
    assignIfPresent(result, 'model', this.model);
    assignIfPresent(result, 'dimension', this.dimension);
    return result;
  }
}

type RetrievalConfigurationLike = JsonRequestConvertible;

export class CreateKnowledgeBaseRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly description?: string;
  readonly subspace?: boolean;
  readonly tags?: string[];
  readonly metadata?: MetadataField[];
  readonly retrievalConfiguration?: RetrievalConfigurationLike;
  readonly embeddingConfiguration?: EmbeddingConfiguration;

  constructor(options: {
    knowledgeBaseName: string;
    description?: string;
    subspace?: boolean;
    tags?: string[];
    metadata?: MetadataField[];
    retrievalConfiguration?: RetrievalConfigurationLike;
    embeddingConfiguration?: EmbeddingConfiguration;
  }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      knowledgeBaseName: this.knowledgeBaseName,
    };
    assignIfPresent(result, 'description', this.description);
    assignIfPresent(result, 'subspace', this.subspace);
    assignIfPresent(result, 'tags', this.tags);
    assignIfPresent(result, 'metadata', this.metadata?.map((item) => item.toJSONRequest()));
    assignIfPresent(result, 'retrievalConfiguration', toJsonValue(this.retrievalConfiguration));
    assignIfPresent(result, 'embeddingConfiguration', toJsonValue(this.embeddingConfiguration));
    return result;
  }
}

export class DeleteKnowledgeBaseRequest implements JsonRequestConvertible {
  constructor(readonly options: { knowledgeBaseName: string }) {}

  toJSONRequest() {
    return { knowledgeBaseName: this.options.knowledgeBaseName };
  }
}

export class DescribeKnowledgeBaseRequest implements JsonRequestConvertible {
  constructor(readonly options: { knowledgeBaseName: string }) {}

  toJSONRequest() {
    return { knowledgeBaseName: this.options.knowledgeBaseName };
  }
}

export class ListKnowledgeBaseRequest implements JsonRequestConvertible {
  constructor(readonly options: { maxResults?: number; nextToken?: string } = {}) {}

  toJSONRequest() {
    const result: Record<string, unknown> = {};
    assignIfPresent(result, 'maxResults', this.options.maxResults);
    assignIfPresent(result, 'nextToken', this.options.nextToken);
    return result;
  }
}

export class UpdateKnowledgeBaseRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly description?: string | null | Unset;
  readonly tags?: string[] | null | Unset;
  readonly retrievalConfiguration?: RetrievalConfigurationLike;

  constructor(options: {
    knowledgeBaseName: string;
    description?: string | null | Unset;
    tags?: string[] | null | Unset;
    retrievalConfiguration?: RetrievalConfigurationLike;
  }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      knowledgeBaseName: this.knowledgeBaseName,
    };
    assignIfPresent(result, 'description', this.description);
    assignIfPresent(result, 'tags', this.tags);
    assignIfPresent(result, 'retrievalConfiguration', toJsonValue(this.retrievalConfiguration));
    return result;
  }
}
