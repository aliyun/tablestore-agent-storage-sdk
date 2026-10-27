use serde::{Deserialize, Serialize};

/// Type of a metadata field on a knowledge base schema.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum MetadataFieldType {
    /// UTF-8 string.
    #[serde(rename = "string")]
    String,
    /// 64-bit signed integer.
    #[serde(rename = "long")]
    Long,
    /// 64-bit floating point.
    #[serde(rename = "double")]
    Double,
    /// Boolean.
    #[serde(rename = "boolean")]
    Boolean,
    /// ISO-8601 date string.
    #[serde(rename = "date")]
    Date,
    /// List of strings.
    #[serde(rename = "string_list")]
    StringList,
    /// List of longs.
    #[serde(rename = "long_list")]
    LongList,
    /// List of doubles.
    #[serde(rename = "double_list")]
    DoubleList,
    /// List of booleans.
    #[serde(rename = "boolean_list")]
    BooleanList,
    /// List of dates.
    #[serde(rename = "date_list")]
    DateList,
}

/// Search mode requested as part of a retrieval configuration.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum SearchType {
    /// Dense vector / semantic search.
    #[serde(rename = "DENSE_VECTOR")]
    DenseVector,
    /// Inverted-index full-text search.
    #[serde(rename = "FULL_TEXT")]
    FullText,
}

/// Algorithm used to rerank merged search results.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum RerankingType {
    /// Reciprocal Rank Fusion.
    #[serde(rename = "RRF")]
    Rrf,
    /// Model-based reranker (e.g. a hosted rerank model).
    #[serde(rename = "MODEL")]
    Model,
    /// Static weighted blend of dense + full-text scores.
    #[serde(rename = "WEIGHT")]
    Weight,
}

/// Query payload kind. Currently only `TEXT` is defined.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum RetrievalQueryType {
    /// Free-form text query.
    #[serde(rename = "TEXT")]
    Text,
}
