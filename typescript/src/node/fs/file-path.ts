import path, { resolve } from 'node:path';

const DEFAULT_SUBSPACE = '_default';

export function getSubspace(subspace?: string): string {
  return subspace ?? DEFAULT_SUBSPACE;
}

export function buildOssObjectKey(options: {
  instanceName: string;
  knowledgeBaseName: string;
  subspace?: string;
  filePath: string;
}): string {
  const normalizedPath = resolve(options.filePath)
    .split(path.sep)
    .join('/')
    .replace(/^\/+/, '');
  return `${options.instanceName}/${options.knowledgeBaseName}/source_doc/${getSubspace(options.subspace)}/${normalizedPath}`;
}

export function buildOssFullPath(bucketName: string, objectKey: string): string {
  return `oss://${bucketName}/${objectKey}`;
}

export function parseOssPath(
  ossPath: string,
  defaultBucketName?: string
): { bucketName: string; objectKey: string } | null {
  if (!ossPath.startsWith('oss://')) {
    return defaultBucketName
      ? {
          bucketName: defaultBucketName,
          objectKey: ossPath,
        }
      : null;
  }
  const withoutScheme = ossPath.slice('oss://'.length);
  const separatorIndex = withoutScheme.indexOf('/');
  if (separatorIndex < 0) {
    return null;
  }
  return {
    bucketName: withoutScheme.slice(0, separatorIndex),
    objectKey: withoutScheme.slice(separatorIndex + 1),
  };
}
