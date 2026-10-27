import { describe, expect, it } from 'vitest';

import {
  MetadataField,
  MetadataFieldType,
  MetadataFilter,
  RetrievalConfiguration,
  RetrievalQuery,
  RetrieveRequest,
  SearchType,
  UNSET,
  UpdateChunkItem,
  UpdateChunksRequest,
  UpdateDocumentRequest,
  UpdateKnowledgeBaseRequest,
  toJSONRequest,
} from '../../src/index.js';

describe('serialization', () => {
  it('omits UNSET fields but preserves null fields', () => {
    const payload = toJSONRequest(
      new UpdateKnowledgeBaseRequest({
        knowledgeBaseName: 'kb',
        description: null,
        tags: UNSET,
      })
    );

    expect(payload).toEqual({
      knowledgeBaseName: 'kb',
      description: null,
    });
  });

  it('serializes filter builders and nested retrieve models', () => {
    const payload = toJSONRequest(
      new RetrieveRequest({
        knowledgeBaseName: 'kb',
        retrievalQuery: new RetrievalQuery({ text: 'hello' }),
        retrievalConfiguration: new RetrievalConfiguration({
          searchType: [SearchType.DENSE_VECTOR, SearchType.FULL_TEXT],
          filter: MetadataFilter.builder().equals('author', 'aliyun').greaterThan('rank', 1).buildAnd(),
        }),
      })
    );

    expect(payload).toEqual({
      knowledgeBaseName: 'kb',
      retrievalQuery: {
        text: 'hello',
        type: 'TEXT',
      },
      retrievalConfiguration: {
        searchType: ['DENSE_VECTOR', 'FULL_TEXT'],
        filter: {
          andAll: [
            { equals: { key: 'author', value: 'aliyun' } },
            { greaterThan: { key: 'rank', value: 1 } },
          ],
        },
      },
    });
  });

  it('serializes document and chunk updates consistently', () => {
    const documentPayload = toJSONRequest(
      new UpdateDocumentRequest({
        knowledgeBaseName: 'kb',
        docId: 'doc-1',
        metadata: null,
      })
    );

    const chunkPayload = toJSONRequest(
      new UpdateChunksRequest({
        knowledgeBaseName: 'kb',
        chunks: [
          new UpdateChunkItem({
            chunkId: 1,
            docId: 'doc-1',
            title: UNSET,
            content: null,
          }),
        ],
      })
    );

    expect(documentPayload).toEqual({
      knowledgeBaseName: 'kb',
      docId: 'doc-1',
      metadata: null,
    });
    expect(chunkPayload).toEqual({
      knowledgeBaseName: 'kb',
      chunks: [
        {
          chunkId: 1,
          docId: 'doc-1',
          content: null,
        },
      ],
    });
  });

  it('serializes knowledge base metadata fields', () => {
    expect(
      toJSONRequest({
        metadata: [new MetadataField({ name: 'author', type: MetadataFieldType.STRING }).toJSONRequest()],
      })
    ).toEqual({
      metadata: [{ name: 'author', type: 'string' }],
    });
  });
});
