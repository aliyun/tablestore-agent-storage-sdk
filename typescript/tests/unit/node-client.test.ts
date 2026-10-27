import { describe, expect, it, vi } from 'vitest';

import { NodeAgentStorageClient } from '../../src/node.js';
import { RequestContext } from '../../src/core/auth/request-context.js';
import { SignV2 } from '../../src/core/auth/sign-v2.js';
import { md5Base64 } from '../../src/core/http/md5.js';
import { OTSClientError } from '../../src/core/errors/client-error.js';
import { OTSServiceError } from '../../src/core/errors/service-error.js';

function makeSignedJsonResponse(path: string, payload: Record<string, unknown>) {
  const body = JSON.stringify(payload);
  const headers = {
    'x-ots-contentmd5': md5Base64(new TextEncoder().encode(body)),
    'x-ots-requestid': 'req-1',
    'x-ots-date': '2025-04-10T01:02:03.000Z',
    'x-ots-contenttype': 'application/json',
  };
  const signer = new SignV2();
  const requestContext = new RequestContext({ accessKeyId: 'ak', accessKeySecret: 'sk' });
  const signature = signer.makeResponseSignature(path, headers, signer.getSigningKey(requestContext));

  return new Response(body, {
    status: 200,
    headers: {
      ...headers,
      authorization: `OTS ak:${signature}`,
    },
  });
}

describe('NodeAgentStorageClient', () => {
  it('uploads local files to OSS and then calls AddDocuments', async () => {
    const putObjectMock = vi.fn().mockResolvedValue(undefined);
    const deleteObjectsMock = vi.fn().mockResolvedValue(undefined);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(makeSignedJsonResponse('/AddDocuments', { code: 'SUCCESS', data: { documentDetails: [] } }));

    const client = new NodeAgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      region: 'cn-beijing',
      ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
      ossBucketName: 'bucket',
      fetch: fetchMock,
      fileReader: {
        exists: () => true,
        createReadStream: () => {
          throw new Error('not used');
        },
      },
      ossUploader: {
        putObject: putObjectMock,
        deleteObjects: deleteObjectsMock,
      },
    });

    await client.uploadDocuments({
      knowledgeBaseName: 'kb',
      documents: [{ filePath: '/tmp/file.md' }],
    });

    expect(putObjectMock).toHaveBeenCalledWith(
      expect.stringContaining('instance/kb/source_doc/_default/'),
      '/tmp/file.md'
    );
    expect(fetchMock).toHaveBeenCalledWith(
      'https://example.com/AddDocuments',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('oss://bucket/instance/kb/source_doc/_default/'),
      })
    );
    expect(deleteObjectsMock).not.toHaveBeenCalled();
  });

  it("uses the python-compatible error message when 'documents' is missing", async () => {
    const client = new NodeAgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      region: 'cn-beijing',
      ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
      ossBucketName: 'bucket',
      fetch: vi.fn(),
      fileReader: {
        exists: () => true,
        createReadStream: () => {
          throw new Error('not used');
        },
      },
      ossUploader: {
        putObject: vi.fn().mockResolvedValue(undefined),
        deleteObjects: vi.fn().mockResolvedValue(undefined),
      },
    });

    await expect(
      client.uploadDocuments({
        knowledgeBaseName: 'kb',
      } as Record<string, unknown>)
    ).rejects.toThrow("Request must contain 'documents' field");
  });

  it('preserves extra document fields when converting filePath to ossKey', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(makeSignedJsonResponse('/AddDocuments', { code: 'SUCCESS', data: { documentDetails: [] } }));

    const client = new NodeAgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      region: 'cn-beijing',
      ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
      ossBucketName: 'bucket',
      fetch: fetchMock,
      fileReader: {
        exists: () => true,
        createReadStream: () => {
          throw new Error('not used');
        },
      },
      ossUploader: {
        putObject: vi.fn().mockResolvedValue(undefined),
        deleteObjects: vi.fn().mockResolvedValue(undefined),
      },
    });

    await client.uploadDocuments({
      knowledgeBaseName: 'kb',
      subspace: 'custom',
      documents: [
        {
          filePath: '/tmp/file.md',
          metadata: { source: 'unit' },
          inclusionFilters: ['a'],
          customField: { nested: true },
        } as Record<string, unknown>,
      ],
    });

    const [, requestInit] = fetchMock.mock.calls[0] as [string, RequestInit];
    const body = JSON.parse(String(requestInit.body)) as {
      documents: Array<Record<string, unknown>>;
    };
    expect(body.documents[0]).toMatchObject({
      metadata: { source: 'unit' },
      inclusionFilters: ['a'],
      customField: { nested: true },
    });
    expect(body.documents[0].filePath).toBeUndefined();
    expect(String(body.documents[0].ossKey)).toContain('oss://bucket/instance/kb/source_doc/custom/');
  });

  it('wraps uploader failures with the same message shape as python sdk', async () => {
    const client = new NodeAgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      region: 'cn-beijing',
      ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
      ossBucketName: 'bucket',
      fetch: vi.fn(),
      fileReader: {
        exists: () => true,
        createReadStream: () => {
          throw new Error('not used');
        },
      },
      ossUploader: {
        putObject: vi.fn().mockRejectedValue(new Error('network failed')),
        deleteObjects: vi.fn().mockResolvedValue(undefined),
      },
    });

    await expect(
      client.uploadDocuments({
        knowledgeBaseName: 'kb',
        documents: [{ filePath: '/tmp/file.md' }],
      })
    ).rejects.toThrow('Failed to upload file to OSS: Error: network failed');
  });

  it('does not delete uploaded OSS files on client-side addDocuments failures', async () => {
    const deleteObjectsMock = vi.fn().mockResolvedValue(undefined);
    const client = new NodeAgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      region: 'cn-beijing',
      ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
      ossBucketName: 'bucket',
      fetch: vi.fn(),
      fileReader: {
        exists: () => true,
        createReadStream: () => {
          throw new Error('not used');
        },
      },
      ossUploader: {
        putObject: vi.fn().mockResolvedValue(undefined),
        deleteObjects: deleteObjectsMock,
      },
    });
    vi.spyOn(client, 'addDocuments').mockRejectedValueOnce(new OTSClientError('client-failure'));

    await expect(
      client.uploadDocuments({
        knowledgeBaseName: 'kb',
        documents: [{ filePath: '/tmp/file.md' }],
      })
    ).rejects.toThrow('client-failure');

    expect(deleteObjectsMock).not.toHaveBeenCalled();
  });

  it('deletes uploaded OSS files on service-side addDocuments failures', async () => {
    const deleteObjectsMock = vi.fn().mockResolvedValue(undefined);
    const client = new NodeAgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      region: 'cn-beijing',
      ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
      ossBucketName: 'bucket',
      fetch: vi.fn(),
      fileReader: {
        exists: () => true,
        createReadStream: () => {
          throw new Error('not used');
        },
      },
      ossUploader: {
        putObject: vi.fn().mockResolvedValue(undefined),
        deleteObjects: deleteObjectsMock,
      },
    });
    vi.spyOn(client, 'addDocuments').mockRejectedValueOnce(
      new OTSServiceError('service-failure', { statusCode: 500 })
    );

    await expect(
      client.uploadDocuments({
        knowledgeBaseName: 'kb',
        documents: [{ filePath: '/tmp/file.md' }],
      })
    ).rejects.toThrow('service-failure');

    expect(deleteObjectsMock).toHaveBeenCalledWith([expect.stringContaining('instance/kb/source_doc/_default/')]);
  });

  it('deletes OSS files when deleteFile is true', async () => {
    const deleteObjectsMock = vi.fn().mockResolvedValue(undefined);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(makeSignedJsonResponse('/GetDocument', { code: 'SUCCESS', data: [{ ossKey: 'oss://bucket/path/doc.md' }] }))
      .mockResolvedValueOnce(makeSignedJsonResponse('/DeleteDocuments', { code: 'SUCCESS', data: {} }));

    const client = new NodeAgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      region: 'cn-beijing',
      ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
      ossBucketName: 'bucket',
      fetch: fetchMock,
      fileReader: {
        exists: () => true,
        createReadStream: () => {
          throw new Error('not used');
        },
      },
      ossUploader: {
        putObject: vi.fn(),
        deleteObjects: deleteObjectsMock,
      },
    });

    await client.deleteDocuments(
      {
        knowledgeBaseName: 'kb',
        documents: [{ docId: 'doc-1' }],
      },
      true
    );

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://example.com/GetDocument',
      expect.objectContaining({ method: 'POST' })
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://example.com/DeleteDocuments',
      expect.objectContaining({ method: 'POST' })
    );
    expect(deleteObjectsMock).toHaveBeenCalledWith(['path/doc.md']);
  });

  it('accepts raw object keys when deleting OSS files', async () => {
    const deleteObjectsMock = vi.fn().mockResolvedValue(undefined);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        makeSignedJsonResponse('/GetDocument', {
          code: 'SUCCESS',
          data: [{ ossKey: 'path/doc.md' }],
        })
      )
      .mockResolvedValueOnce(makeSignedJsonResponse('/DeleteDocuments', { code: 'SUCCESS', data: {} }));

    const client = new NodeAgentStorageClient({
      endpoint: 'https://example.com',
      instanceName: 'instance',
      accessKeyId: 'ak',
      accessKeySecret: 'sk',
      region: 'cn-beijing',
      ossEndpoint: 'https://oss-cn-beijing.aliyuncs.com',
      ossBucketName: 'bucket',
      fetch: fetchMock,
      fileReader: {
        exists: () => true,
        createReadStream: () => {
          throw new Error('not used');
        },
      },
      ossUploader: {
        putObject: vi.fn(),
        deleteObjects: deleteObjectsMock,
      },
    });

    await client.deleteDocuments(
      {
        knowledgeBaseName: 'kb',
        documents: [{ docId: 'doc-1' }],
      },
      true
    );

    expect(deleteObjectsMock).toHaveBeenCalledWith(['path/doc.md']);
  });
});
