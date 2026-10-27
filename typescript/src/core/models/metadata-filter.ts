export type Primitive = string | number | boolean;
export type KeyValueCondition = {
  key: string;
  value: Primitive | Primitive[];
};

type MetadataFilterPayload =
  | { equals: KeyValueCondition }
  | { notEquals: KeyValueCondition }
  | { greaterThan: KeyValueCondition }
  | { greaterThanOrEquals: KeyValueCondition }
  | { lessThan: KeyValueCondition }
  | { lessThanOrEquals: KeyValueCondition }
  | { in: KeyValueCondition }
  | { notIn: KeyValueCondition }
  | { startsWith: KeyValueCondition }
  | { stringContains: KeyValueCondition }
  | { listContains: KeyValueCondition }
  | { andAll: MetadataFilterPayload[] }
  | { orAll: MetadataFilterPayload[] }
  | { notAll: MetadataFilterPayload[] };

export class MetadataFilter {
  private constructor(private readonly payload: MetadataFilterPayload) {}

  static equals(key: string, value: Primitive): MetadataFilter {
    return new MetadataFilter({ equals: { key, value } });
  }

  static notEquals(key: string, value: Primitive): MetadataFilter {
    return new MetadataFilter({ notEquals: { key, value } });
  }

  static greaterThan(key: string, value: number): MetadataFilter {
    return new MetadataFilter({ greaterThan: { key, value } });
  }

  static greaterThanOrEquals(key: string, value: number): MetadataFilter {
    return new MetadataFilter({ greaterThanOrEquals: { key, value } });
  }

  static lessThan(key: string, value: number): MetadataFilter {
    return new MetadataFilter({ lessThan: { key, value } });
  }

  static lessThanOrEquals(key: string, value: number): MetadataFilter {
    return new MetadataFilter({ lessThanOrEquals: { key, value } });
  }

  static inList(key: string, value: string[]): MetadataFilter {
    return new MetadataFilter({ in: { key, value } });
  }

  static notInList(key: string, value: string[]): MetadataFilter {
    return new MetadataFilter({ notIn: { key, value } });
  }

  static startsWith(key: string, value: string): MetadataFilter {
    return new MetadataFilter({ startsWith: { key, value } });
  }

  static stringContains(key: string, value: string): MetadataFilter {
    return new MetadataFilter({ stringContains: { key, value } });
  }

  static listContains(key: string, value: string): MetadataFilter {
    return new MetadataFilter({ listContains: { key, value } });
  }

  static andAll(...filters: MetadataFilter[]): MetadataFilter {
    return new MetadataFilter({ andAll: filters.map((filter) => filter.toJSONRequest()) });
  }

  static orAll(...filters: MetadataFilter[]): MetadataFilter {
    return new MetadataFilter({ orAll: filters.map((filter) => filter.toJSONRequest()) });
  }

  static notAll(...filters: MetadataFilter[]): MetadataFilter {
    return new MetadataFilter({ notAll: filters.map((filter) => filter.toJSONRequest()) });
  }

  static builder(): MetadataFilterBuilder {
    return new MetadataFilterBuilder();
  }

  toJSONRequest(): MetadataFilterPayload {
    return this.payload;
  }
}

export class MetadataFilterBuilder {
  private readonly filters: MetadataFilter[] = [];

  add(filter: MetadataFilter): MetadataFilterBuilder {
    this.filters.push(filter);
    return this;
  }

  equals(key: string, value: Primitive): MetadataFilterBuilder {
    return this.add(MetadataFilter.equals(key, value));
  }

  notEquals(key: string, value: Primitive): MetadataFilterBuilder {
    return this.add(MetadataFilter.notEquals(key, value));
  }

  greaterThan(key: string, value: number): MetadataFilterBuilder {
    return this.add(MetadataFilter.greaterThan(key, value));
  }

  greaterThanOrEquals(key: string, value: number): MetadataFilterBuilder {
    return this.add(MetadataFilter.greaterThanOrEquals(key, value));
  }

  lessThan(key: string, value: number): MetadataFilterBuilder {
    return this.add(MetadataFilter.lessThan(key, value));
  }

  lessThanOrEquals(key: string, value: number): MetadataFilterBuilder {
    return this.add(MetadataFilter.lessThanOrEquals(key, value));
  }

  inList(key: string, value: string[]): MetadataFilterBuilder {
    return this.add(MetadataFilter.inList(key, value));
  }

  notInList(key: string, value: string[]): MetadataFilterBuilder {
    return this.add(MetadataFilter.notInList(key, value));
  }

  startsWith(key: string, value: string): MetadataFilterBuilder {
    return this.add(MetadataFilter.startsWith(key, value));
  }

  stringContains(key: string, value: string): MetadataFilterBuilder {
    return this.add(MetadataFilter.stringContains(key, value));
  }

  listContains(key: string, value: string): MetadataFilterBuilder {
    return this.add(MetadataFilter.listContains(key, value));
  }

  buildAnd(): MetadataFilter {
    return this.build('and');
  }

  buildOr(): MetadataFilter {
    return this.build('or');
  }

  buildNot(): MetadataFilter {
    return this.build('not');
  }

  private build(operator: 'and' | 'or' | 'not'): MetadataFilter {
    if (this.filters.length === 0) {
      throw new Error(`Cannot build ${operator} filter with no conditions`);
    }
    if (this.filters.length === 1) {
      return this.filters[0];
    }
    if (operator === 'and') {
      return MetadataFilter.andAll(...this.filters);
    }
    if (operator === 'or') {
      return MetadataFilter.orAll(...this.filters);
    }
    return MetadataFilter.notAll(...this.filters);
  }
}
