import { isUnset } from './unset.js';

export interface JsonRequestConvertible<T = Record<string, unknown>> {
  toJSONRequest(): T;
}

export function assignIfPresent(
  target: Record<string, unknown>,
  key: string,
  value: unknown
): void {
  if (!isUnset(value)) {
    target[key] = value;
  }
}

export function toJsonValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map((item) => toJsonValue(item));
  }
  if (value && typeof value === 'object' && 'toJSONRequest' in value && typeof value.toJSONRequest === 'function') {
    return value.toJSONRequest();
  }
  return value;
}
