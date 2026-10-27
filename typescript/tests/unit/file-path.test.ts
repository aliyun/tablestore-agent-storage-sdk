import path from 'node:path';
import { describe, expect, it } from 'vitest';

import { buildOssFullPath, buildOssObjectKey, parseOssPath } from '../../src/node/fs/file-path.js';

describe('file path helpers', () => {
  it('builds object keys with the python-compatible path layout', () => {
    expect(
      buildOssObjectKey({
        instanceName: 'instance',
        knowledgeBaseName: 'kb',
        filePath: '/tmp/doc.md',
      })
    ).toContain('instance/kb/source_doc/_default/tmp/doc.md');
  });

  it('parses oss urls', () => {
    expect(parseOssPath('oss://bucket/path/doc.md')).toEqual({
      bucketName: 'bucket',
      objectKey: 'path/doc.md',
    });
  });

  it('parses raw object keys using the current bucket', () => {
    expect(parseOssPath('path/doc.md', 'bucket')).toEqual({
      bucketName: 'bucket',
      objectKey: 'path/doc.md',
    });
  });

  it('normalises Windows backslashes to forward slashes on Windows only', () => {
    const key = buildOssObjectKey({
      instanceName: 'instance',
      knowledgeBaseName: 'kb',
      filePath: 'C:\\Users\\john\\docs\\report.pdf',
    });
    if (path.sep === '\\') {
      // Windows: backslashes are path separators, should be converted to /
      expect(key).not.toContain('\\');
      expect(key).toContain('C:/Users/john/docs/report.pdf');
    } else {
      // macOS/Linux: backslashes are valid filename characters, should remain
      expect(key).toContain('C:\\Users\\john\\docs\\report.pdf');
    }
  });

  it('handles relative paths without backslashes', () => {
    const key = buildOssObjectKey({
      instanceName: 'instance',
      knowledgeBaseName: 'kb',
      filePath: 'docs/report.pdf',
    });
    expect(key).not.toContain('\\');
  });

  it('builds full oss paths', () => {
    expect(buildOssFullPath('bucket', 'path/doc.md')).toBe('oss://bucket/path/doc.md');
  });
});
