import { afterAll, describe, expect, it } from 'vitest';
import { join } from 'node:path';

import { NodeAgentStorageClient } from '../../src/node.js';

function requiredEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function createClient(): NodeAgentStorageClient {
  return new NodeAgentStorageClient({
    endpoint: requiredEnv('OTS_ENDPOINT'),
    instanceName: requiredEnv('OTS_INSTANCE_NAME'),
    accessKeyId: requiredEnv('OTS_ACCESS_KEY_ID'),
    accessKeySecret: requiredEnv('OTS_ACCESS_KEY_SECRET'),
    region: process.env.OTS_REGION,
    signVersion: (process.env.OTS_SIGN_VERSION as 'v2' | 'v4' | undefined) ?? 'v2',
    ossEndpoint: requiredEnv('OSS_ENDPOINT'),
    ossBucketName: requiredEnv('OSS_BUCKET_NAME'),
  });
}

describe('knowledge base e2e', () => {
  const client = createClient();
  const knowledgeBaseName = `ts_kb_${Date.now()}`;
  const filePath = join(process.cwd(), 'tests/fixtures/sample-doc.md');
  const createdDocumentIds: string[] = [];

  afterAll(async () => {
    if (createdDocumentIds.length > 0) {
      await client.deleteDocuments(
        {
          knowledgeBaseName,
          documents: createdDocumentIds.map((docId) => ({ docId })),
        },
        true
      ).catch(() => undefined);
    }
    await client.deleteKnowledgeBase({ knowledgeBaseName }).catch(() => undefined);
  });

  it('creates a knowledge base, uploads, retrieves, updates, and cleans up', async () => {
    const createResponse = await client.createKnowledgeBase({
      knowledgeBaseName,
      subspace: false,
      description: 'TypeScript SDK e2e',
      metadata: [
        {
          name: 'source',
          type: 'string',
        },
      ],
    });
    expect(createResponse).toMatchObject({ code: 'SUCCESS' });

    const uploadResponse = (await client.uploadDocuments({
      knowledgeBaseName,
      documents: [{ filePath }],
    })) as {
      code: string;
      data?: { documentDetails?: Array<{ docId?: string }> };
    };
    expect(uploadResponse.code).toBe('SUCCESS');

    const docId = uploadResponse.data?.documentDetails?.[0]?.docId;
    expect(docId).toBeTruthy();
    createdDocumentIds.push(docId!);

    let documentStatus = '';
    for (let attempt = 0; attempt < 60; attempt += 1) {
      const documentResponse = (await client.getDocument({
        knowledgeBaseName,
        docId,
      })) as {
        code: string;
        data?: Array<{ status?: string }>;
      };
      documentStatus = documentResponse.data?.[0]?.status ?? '';
      if (documentStatus === 'completed') {
        break;
      }
      if (documentStatus === 'failed') {
        throw new Error('document processing failed');
      }
      await sleep(2000);
    }
    expect(documentStatus).toBe('completed');

    const listDocumentsResponse = await client.listDocuments({
      knowledgeBaseName,
      maxResults: 10,
    });
    expect(listDocumentsResponse).toMatchObject({ code: 'SUCCESS' });

    const updateDocumentResponse = await client.updateDocument({
      knowledgeBaseName,
      docId,
      metadata: { source: 'typescript-e2e' },
    });
    expect(updateDocumentResponse).toMatchObject({ code: 'SUCCESS' });

    const retrieveResponse = await client.retrieve({
      knowledgeBaseName,
      retrievalQuery: {
        text: 'sample document',
        type: 'TEXT',
      },
    });
    expect(retrieveResponse).toMatchObject({ code: 'SUCCESS' });

    const chunkResponse = (await client.listChunks({
      knowledgeBaseName,
      docId,
      maxResults: 10,
    })) as {
      code: string;
      data?: { chunkDetails?: Array<{ chunkId?: number; docId?: string }> };
    };
    expect(chunkResponse.code).toBe('SUCCESS');

    const chunk = chunkResponse.data?.chunkDetails?.[0];
    if (chunk?.chunkId && chunk.docId) {
      const updateChunksResponse = await client.updateChunks({
        knowledgeBaseName,
        chunks: [
          {
            chunkId: chunk.chunkId,
            docId: chunk.docId,
            title: 'updated by typescript e2e',
          },
        ],
      });
      expect(updateChunksResponse).toMatchObject({ code: 'SUCCESS' });
    }

    const deleteResponse = await client.deleteDocuments(
      {
        knowledgeBaseName,
        documents: [{ docId }],
      },
      true
    );
    expect(deleteResponse).toMatchObject({ code: 'SUCCESS' });
    createdDocumentIds.length = 0;

    const deleteKnowledgeBaseResponse = await client.deleteKnowledgeBase({ knowledgeBaseName });
    expect(deleteKnowledgeBaseResponse).toMatchObject({ code: 'SUCCESS' });
  }, 180000);
});
