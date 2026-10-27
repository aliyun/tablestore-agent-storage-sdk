import type { JsonRequestConvertible } from '../models/base.js';

export function toJSONRequest<T extends JsonRequestConvertible | Record<string, unknown>>(value: T) {
  if (value && typeof value === 'object' && 'toJSONRequest' in value && typeof value.toJSONRequest === 'function') {
    return value.toJSONRequest();
  }
  return value;
}
