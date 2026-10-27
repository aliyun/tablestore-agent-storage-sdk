import type { JsonRequestConvertible } from './base.js';
import { assignIfPresent } from './base.js';
import type { Unset } from './unset.js';

export class AddDocumentItem implements JsonRequestConvertible {
  readonly ossKey: string;
  readonly metadata?: Record<string, unknown>;
  readonly inclusionFilters?: string[];
  readonly exclusionFilters?: string[];

  constructor(options: {
    ossKey: string;
    metadata?: Record<string, unknown>;
    inclusionFilters?: string[];
    exclusionFilters?: string[];
  }) {
    Object.assign(this, options);
    this.ossKey = options.ossKey;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { ossKey: this.ossKey };
    assignIfPresent(result, 'metadata', this.metadata);
    assignIfPresent(result, 'inclusionFilters', this.inclusionFilters);
    assignIfPresent(result, 'exclusionFilters', this.exclusionFilters);
    return result;
  }
}

export class UploadDocumentItem implements JsonRequestConvertible {
  readonly filePath: string;
  readonly metadata?: Record<string, unknown>;

  constructor(options: { filePath: string; metadata?: Record<string, unknown> }) {
    Object.assign(this, options);
    this.filePath = options.filePath;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { filePath: this.filePath };
    assignIfPresent(result, 'metadata', this.metadata);
    return result;
  }
}

export class DeleteDocumentItem implements JsonRequestConvertible {
  readonly docId?: string;
  readonly ossKey?: string;

  constructor(options: { docId?: string; ossKey?: string }) {
    Object.assign(this, options);
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {};
    assignIfPresent(result, 'docId', this.docId);
    assignIfPresent(result, 'ossKey', this.ossKey);
    return result;
  }
}

export class AddDocumentsRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly documents: AddDocumentItem[];
  readonly subspace?: string;

  constructor(options: { knowledgeBaseName: string; documents: AddDocumentItem[]; subspace?: string }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
    this.documents = options.documents;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      knowledgeBaseName: this.knowledgeBaseName,
      documents: this.documents.map((item) => item.toJSONRequest()),
    };
    assignIfPresent(result, 'subspace', this.subspace);
    return result;
  }
}

export class UploadDocumentsRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly documents: UploadDocumentItem[];
  readonly subspace?: string;

  constructor(options: { knowledgeBaseName: string; documents: UploadDocumentItem[]; subspace?: string }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
    this.documents = options.documents;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      knowledgeBaseName: this.knowledgeBaseName,
      documents: this.documents.map((item) => item.toJSONRequest()),
    };
    assignIfPresent(result, 'subspace', this.subspace);
    return result;
  }
}

export class GetDocumentRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly docId?: string;
  readonly ossKey?: string;
  readonly subspace?: string;

  constructor(options: { knowledgeBaseName: string; docId?: string; ossKey?: string; subspace?: string }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { knowledgeBaseName: this.knowledgeBaseName };
    assignIfPresent(result, 'docId', this.docId);
    assignIfPresent(result, 'ossKey', this.ossKey);
    assignIfPresent(result, 'subspace', this.subspace);
    return result;
  }
}

export class ListDocumentsRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly subspace?: string[];
  readonly maxResults?: number;
  readonly nextToken?: string;

  constructor(options: {
    knowledgeBaseName: string;
    subspace?: string[];
    maxResults?: number;
    nextToken?: string;
  }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { knowledgeBaseName: this.knowledgeBaseName };
    assignIfPresent(result, 'subspace', this.subspace);
    assignIfPresent(result, 'maxResults', this.maxResults);
    assignIfPresent(result, 'nextToken', this.nextToken);
    return result;
  }
}

export class DeleteDocumentsRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly documents: DeleteDocumentItem[];
  readonly subspace?: string;

  constructor(options: { knowledgeBaseName: string; documents: DeleteDocumentItem[]; subspace?: string }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
    this.documents = options.documents;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = {
      knowledgeBaseName: this.knowledgeBaseName,
      documents: this.documents.map((item) => item.toJSONRequest()),
    };
    assignIfPresent(result, 'subspace', this.subspace);
    return result;
  }
}

export class UpdateDocumentRequest implements JsonRequestConvertible {
  readonly knowledgeBaseName: string;
  readonly subspace?: string;
  readonly ossKey?: string;
  readonly docId?: string;
  readonly metadata?: Record<string, unknown> | null | Unset;

  constructor(options: {
    knowledgeBaseName: string;
    subspace?: string;
    ossKey?: string;
    docId?: string;
    metadata?: Record<string, unknown> | null | Unset;
  }) {
    Object.assign(this, options);
    this.knowledgeBaseName = options.knowledgeBaseName;
  }

  toJSONRequest() {
    const result: Record<string, unknown> = { knowledgeBaseName: this.knowledgeBaseName };
    assignIfPresent(result, 'subspace', this.subspace);
    assignIfPresent(result, 'ossKey', this.ossKey);
    assignIfPresent(result, 'docId', this.docId);
    assignIfPresent(result, 'metadata', this.metadata);
    return result;
  }
}
