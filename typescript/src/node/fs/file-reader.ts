import { createReadStream, existsSync } from 'node:fs';

export interface FileReaderLike {
  exists(filePath: string): boolean;
  createReadStream(filePath: string): NodeJS.ReadableStream;
}

export class NodeFileReader implements FileReaderLike {
  exists(filePath: string): boolean {
    return existsSync(filePath);
  }

  createReadStream(filePath: string): NodeJS.ReadableStream {
    return createReadStream(filePath);
  }
}
