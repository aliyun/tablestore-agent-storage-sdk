import type { JsonRequestConvertible } from './base.js';
import { assignIfPresent, toJsonValue } from './base.js';
import { RerankingType, RetrievalQueryType, SearchType } from './enums.js';
import type { MetadataFilter } from './metadata-filter.js';

export class RetrievalQuery implements JsonRequestConvertible {
  readonly text: string;
  readonly type: RetrievalQueryType | string;

  constructor(options: { text: string; type?: RetrievalQueryType | string }) {
    this.text = options.text;
    this.type = options.type ?? RetrievalQueryType.TEXT;
  }

  toJSONRequest() {
    return {
      text: this.text,
      type: this.type,
    };
  }
}

export class DenseVectorSearchConfiguration implements JsonRequestConvertible {
  constructor(readonly options: { numberOfResults?: number } = {}) {}

  toJSONRequest() {
    return { numberOfResults: this.options.numberOfResults ?? 20 };
  }
}

export class FullTextSearchConfiguration implements JsonRequestConvertible {
  constructor(readonly options: { numberOfResults?: number } = {}) {}

  toJSONRequest() {
    return { numberOfResults: this.options.numberOfResults ?? 20 };
  }
}

export class RRFConfiguration implements JsonRequestConvertible {
  readonly denseVectorSearchWeight: number;
  readonly fullTextSearchWeight: number;
  readonly k: number;

  constructor(options: { denseVectorSearchWeight?: number; fullTextSearchWeight?: number; k?: number } = {}) {
    this.denseVectorSearchWeight = options.denseVectorSearchWeight ?? 1;
    this.fullTextSearchWeight = options.fullTextSearchWeight ?? 1;
    this.k = options.k ?? 60;
  }

  toJSONRequest() {
    return {
      denseVectorSearchWeight: this.denseVectorSearchWeight,
      fullTextSearchWeight: this.fullTextSearchWeight,
      k: this.k,
    };
  }
}

export class ModelConfiguration implements JsonRequestConvertible {
  readonly provider?: string;
  readonly model?: string;

  constructor(options: { provider?: string; model?: string } = {}) {
    Object.assign(this, options);
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {};
    assignIfPresent(result, 'provider', this.provider);
    assignIfPresent(result, 'model', this.model);
    return result;
  }
}

export class WeightConfiguration implements JsonRequestConvertible {
  readonly denseVectorSearchWeight?: number;
  readonly fullTextSearchWeight?: number;

  constructor(options: { denseVectorSearchWeight?: number; fullTextSearchWeight?: number } = {}) {
    Object.assign(this, options);
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {};
    assignIfPresent(result, 'denseVectorSearchWeight', this.denseVectorSearchWeight);
    assignIfPresent(result, 'fullTextSearchWeight', this.fullTextSearchWeight);
    return result;
  }
}

export class RerankingConfiguration implements JsonRequestConvertible {
  readonly type: RerankingType | string;
  readonly numberOfResults: number;
  readonly rrfConfiguration?: RRFConfiguration;
  readonly modelConfiguration?: ModelConfiguration;
  readonly weightConfiguration?: WeightConfiguration;

  constructor(options: {
    type?: RerankingType | string;
    numberOfResults?: number;
    rrfConfiguration?: RRFConfiguration;
    modelConfiguration?: ModelConfiguration;
    weightConfiguration?: WeightConfiguration;
  } = {}) {
    this.type = options.type ?? RerankingType.RRF;
    this.numberOfResults = options.numberOfResults ?? 20;
    this.rrfConfiguration = options.rrfConfiguration;
    this.modelConfiguration = options.modelConfiguration;
    this.weightConfiguration = options.weightConfiguration;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      type: this.type,
      numberOfResults: this.numberOfResults,
    };
    assignIfPresent(result, 'rrfConfiguration', toJsonValue(this.rrfConfiguration));
    assignIfPresent(result, 'modelConfiguration', toJsonValue(this.modelConfiguration));
    assignIfPresent(result, 'weightConfiguration', toJsonValue(this.weightConfiguration));
    return result;
  }
}

export class RetrievalConfiguration implements JsonRequestConvertible {
  readonly searchType?: Array<SearchType | string>;
  readonly denseVectorSearchConfiguration?: DenseVectorSearchConfiguration;
  readonly fullTextSearchConfiguration?: FullTextSearchConfiguration;
  readonly rerankingConfiguration?: RerankingConfiguration;
  readonly filter?: MetadataFilter;

  constructor(options: {
    searchType?: Array<SearchType | string>;
    denseVectorSearchConfiguration?: DenseVectorSearchConfiguration;
    fullTextSearchConfiguration?: FullTextSearchConfiguration;
    rerankingConfiguration?: RerankingConfiguration;
    filter?: MetadataFilter;
  } = {}) {
    Object.assign(this, options);
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {};
    assignIfPresent(result, 'searchType', this.searchType);
    assignIfPresent(result, 'denseVectorSearchConfiguration', toJsonValue(this.denseVectorSearchConfiguration));
    assignIfPresent(result, 'fullTextSearchConfiguration', toJsonValue(this.fullTextSearchConfiguration));
    assignIfPresent(result, 'rerankingConfiguration', toJsonValue(this.rerankingConfiguration));
    assignIfPresent(result, 'filter', toJsonValue(this.filter));
    return result;
  }
}

export class RetrieveRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly retrievalQuery: RetrievalQuery;
  readonly subspace?: string[];
  readonly retrievalConfiguration?: RetrievalConfiguration;

  constructor(options: {
    knowledgeBaseName: string;
    retrievalQuery: RetrievalQuery;
    subspace?: string[];
    retrievalConfiguration?: RetrievalConfiguration;
  }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
    this.retrievalQuery = options.retrievalQuery;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      knowledgeBaseName: this.knowledgeBaseName,
      retrievalQuery: this.retrievalQuery.toJSONRequest(),
    };
    assignIfPresent(result, 'subspace', this.subspace);
    assignIfPresent(result, 'retrievalConfiguration', toJsonValue(this.retrievalConfiguration));
    return result;
  }
}
