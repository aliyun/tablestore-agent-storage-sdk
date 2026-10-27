export const UNSET = Symbol('UNSET');

export type Unset = typeof UNSET;

export function isUnset(value: unknown): value is Unset | undefined {
  return value === UNSET || value === undefined;
}
