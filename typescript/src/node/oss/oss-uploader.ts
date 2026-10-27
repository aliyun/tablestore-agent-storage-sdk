import OSS from 'ali-oss';

export interface OSSUploaderLike {
  putObject(objectKey: string, filePath: string): Promise<void>;
  deleteObjects(objectKeys: string[]): Promise<void>;
}

export class OSSUploader implements OSSUploaderLike {
  private readonly client: OSS;

  constructor(options: {
    region: string;
    accessKeyId: string;
    accessKeySecret: string;
    bucket: string;
    endpoint?: string;
    stsToken?: string;
  }) {
    this.client = new OSS({
      region: options.region,
      accessKeyId: options.accessKeyId,
      accessKeySecret: options.accessKeySecret,
      bucket: options.bucket,
      endpoint: options.endpoint,
      stsToken: options.stsToken,
    });
  }

  async putObject(objectKey: string, filePath: string): Promise<void> {
    await this.client.put(objectKey, filePath);
  }

  async deleteObjects(objectKeys: string[]): Promise<void> {
    if (objectKeys.length === 0) {
      return;
    }
    await this.client.deleteMulti(objectKeys, { quiet: true });
  }
}
