import { createHash } from 'node:crypto';

export function md5Base64(content: Uint8Array | string): string {
  const hash = createHash('md5');
  hash.update(content);
  return hash.digest('base64');
}
