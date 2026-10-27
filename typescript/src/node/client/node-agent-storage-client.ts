import { OTSClientError } from '../../core/errors/client-error.js';
import { AgentStorageClient, type AgentStorageClientOptions } from '../../core/client/agent-storage-client.js';
import { buildOssFullPath, buildOssObjectKey, parseOssPath } from '../fs/file-path.js';
import { NodeFileReader, type FileReaderLike } from '../fs/file-reader.js';
import { OSSUploader, type OSSUploaderLike } from '../oss/oss-uploader.js';
import { toJSONRequest } from '../../core/serialization/to-json-request.js';

export interface NodeAgentStorageClientOptions extends AgentStorageClientOptions {
  ossEndpoint: string;
  ossBucketName: string;
  ossAccessKeyId?: string;
  ossAccessKeySecret?: string;
  ossUploader?: OSSUploaderLike;
  fileReader?: FileReaderLike;
}

type UploadDocumentInput = {
  filePath: string;
  metadata?: Record<string, unknown>;
};

type UploadDocumentsRequestInput = {
  knowledgeBaseName: string;
  documents: UploadDocumentInput[];
  subspace?: string;
};

type DeleteDocumentInput = {
  docId?: string;
  ossKey?: string;
};

export class NodeAgentStorageClient extends AgentStorageClient {
  private readonly ossBucketName: string;
  private readonly uploader: OSSUploaderLike;
  private readonly fileReader: FileReaderLike;
  private readonly instanceName: string;

  constructor(private readonly options: NodeAgentStorageClientOptions) {
    super(options);
    this.ossBucketName = options.ossBucketName;
    this.instanceName = options.instanceName;
    const ossKeyId = options.ossAccessKeyId ?? options.accessKeyId;
    const ossKeySecret = options.ossAccessKeySecret ?? options.accessKeySecret;
    if (options.ossUploader) {
      this.uploader = options.ossUploader;
    } else if (ossKeyId && ossKeySecret) {
      this.uploader = new OSSUploader({
        region: options.region ?? 'cn-beijing',
        endpoint: options.ossEndpoint,
        accessKeyId: ossKeyId,
        accessKeySecret: ossKeySecret,
        bucket: options.ossBucketName,
        stsToken: options.stsToken,
      });
    } else {
      this.uploader = {
        putObject() {
          return Promise.reject(
            new OTSClientError('OSS upload requires accessKeyId/accessKeySecret. API key auth is not supported for OSS.')
          );
        },
        deleteObjects() {
          return Promise.reject(
            new OTSClientError('OSS delete requires accessKeyId/accessKeySecret. API key auth is not supported for OSS.')
          );
        },
      };
    }
    this.fileReader = options.fileReader ?? new NodeFileReader();
  }

  override async uploadDocuments(request: UploadDocumentsRequestInput | Record<string, unknown>) {
    const payload = toJSONRequest(request) as UploadDocumentsRequestInput;
    if (!payload.knowledgeBaseName) {
      throw new OTSClientError("Request must contain 'knowledgeBaseName' field");
    }
    if (!('documents' in payload)) {
      throw new OTSClientError("Request must contain 'documents' field");
    }
    if (!Array.isArray(payload.documents)) {
      throw new OTSClientError("'documents' field must be a list");
    }

    const uploadedKeys: string[] = [];
    const documents: Record<string, unknown>[] = [];

    for (const document of payload.documents) {
      if (!document.filePath) {
        throw new OTSClientError("Each document must contain 'filePath' field");
      }
      if (!this.fileReader.exists(document.filePath)) {
        const error = new Error(`File not found: ${document.filePath}`);
        error.name = 'FileNotFoundError';
        throw error;
      }

      const objectKey = buildOssObjectKey({
        instanceName: this.instanceName,
        knowledgeBaseName: payload.knowledgeBaseName,
        subspace: payload.subspace,
        filePath: document.filePath,
      });
      try {
        await this.uploader.putObject(objectKey, document.filePath);
      } catch (error) {
        throw new Error(`Failed to upload file to OSS: ${String(error)}`);
      }
      uploadedKeys.push(objectKey);
      const uploadedDocument = {
        ...document,
        ossKey: buildOssFullPath(this.ossBucketName, objectKey),
      } as Record<string, unknown>;
      delete uploadedDocument.filePath;
      documents.push(uploadedDocument);
    }

    try {
      return await this.addDocuments({
        knowledgeBaseName: payload.knowledgeBaseName,
        subspace: payload.subspace,
        documents,
      });
    } catch (error) {
      if (error instanceof Error && error.name === 'OTSServiceError') {
        await this.uploader.deleteObjects(uploadedKeys);
      }
      throw error;
    }
  }

  async deleteDocuments(
    request: { knowledgeBaseName: string; documents: DeleteDocumentInput[]; subspace?: string } | Record<string, unknown>,
    deleteFile = false
  ) {
    const payload = toJSONRequest(request) as {
      knowledgeBaseName: string;
      documents: DeleteDocumentInput[];
      subspace?: string;
    };

    if (!deleteFile) {
      return super.deleteDocuments(payload);
    }

    const objectKeys: string[] = [];
    for (const document of payload.documents) {
      let ossKey = document.ossKey;
      if (!ossKey && document.docId) {
        const response = (await this.getDocument({
          knowledgeBaseName: payload.knowledgeBaseName,
          ...(payload.subspace ? { subspace: payload.subspace } : {}),
          docId: document.docId,
        })) as { data?: Array<{ ossKey?: string }> };
        ossKey = response.data?.[0]?.ossKey;
      }
      if (!ossKey) {
        continue;
      }
      const parsed = parseOssPath(ossKey, this.ossBucketName);
      if (!parsed || parsed.bucketName !== this.ossBucketName) {
        continue;
      }
      objectKeys.push(parsed.objectKey);
    }

    const result = await super.deleteDocuments(payload);
    await this.uploader.deleteObjects(objectKeys);
    return result;
  }
}
