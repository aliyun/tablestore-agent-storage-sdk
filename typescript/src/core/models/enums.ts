export enum MetadataFieldType {
  STRING = 'string',
  LONG = 'long',
  DOUBLE = 'double',
  BOOLEAN = 'boolean',
  DATE = 'date',
  STRING_LIST = 'string_list',
  LONG_LIST = 'long_list',
  DOUBLE_LIST = 'double_list',
  BOOLEAN_LIST = 'boolean_list',
  DATE_LIST = 'date_list',
}

export enum SearchType {
  DENSE_VECTOR = 'DENSE_VECTOR',
  FULL_TEXT = 'FULL_TEXT',
}

export enum RerankingType {
  RRF = 'RRF',
  MODEL = 'MODEL',
  WEIGHT = 'WEIGHT',
}

export enum RetrievalQueryType {
  TEXT = 'TEXT',
}
