use serde_json::{Map, Value};

use crate::error::ClientError;

use super::base::{assign_if_some, to_json_value, JsonRequestConvertible};
use super::enums::{RerankingType, RetrievalQueryType, SearchType};
use super::metadata_filter::MetadataFilter;

/// Query payload for `Retrieve`.
#[derive(Debug, Clone)]
pub struct RetrievalQuery {
    /// Query text.
    pub text: String,
    /// Query kind. Defaults to [`RetrievalQueryType::Text`].
    pub query_type: RetrievalQueryType,
}

impl RetrievalQuery {
    /// Build a TEXT query.
    pub fn text(text: impl Into<String>) -> Self {
        Self {
            text: text.into(),
            query_type: RetrievalQueryType::Text,
        }
    }
}

impl JsonRequestConvertible for RetrievalQuery {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert("text".to_string(), Value::String(self.text.clone()));
        map.insert(
            "type".to_string(),
            serde_json::to_value(self.query_type).unwrap(),
        );
        Ok(Value::Object(map))
    }
}

/// Dense-vector search knobs.
#[derive(Debug, Clone)]
pub struct DenseVectorSearchConfiguration {
    /// Number of vector hits to fetch. Default `20` matches the TS SDK.
    pub number_of_results: u32,
}

impl Default for DenseVectorSearchConfiguration {
    fn default() -> Self {
        Self {
            number_of_results: 20,
        }
    }
}

impl JsonRequestConvertible for DenseVectorSearchConfiguration {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "numberOfResults".to_string(),
            Value::Number(self.number_of_results.into()),
        );
        Ok(Value::Object(map))
    }
}

/// Full-text search knobs.
#[derive(Debug, Clone)]
pub struct FullTextSearchConfiguration {
    /// Number of full-text hits to fetch. Default `20`.
    pub number_of_results: u32,
}

impl Default for FullTextSearchConfiguration {
    fn default() -> Self {
        Self {
            number_of_results: 20,
        }
    }
}

impl JsonRequestConvertible for FullTextSearchConfiguration {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "numberOfResults".to_string(),
            Value::Number(self.number_of_results.into()),
        );
        Ok(Value::Object(map))
    }
}

/// Reciprocal Rank Fusion configuration.
#[derive(Debug, Clone)]
pub struct RRFConfiguration {
    /// Weight applied to dense-vector ranks.
    pub dense_vector_search_weight: f64,
    /// Weight applied to full-text ranks.
    pub full_text_search_weight: f64,
    /// RRF k constant.
    pub k: u32,
}

impl Default for RRFConfiguration {
    fn default() -> Self {
        Self {
            dense_vector_search_weight: 1.0,
            full_text_search_weight: 1.0,
            k: 60,
        }
    }
}

impl JsonRequestConvertible for RRFConfiguration {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "denseVectorSearchWeight".to_string(),
            serde_json::to_value(self.dense_vector_search_weight).unwrap(),
        );
        map.insert(
            "fullTextSearchWeight".to_string(),
            serde_json::to_value(self.full_text_search_weight).unwrap(),
        );
        map.insert("k".to_string(), Value::Number(self.k.into()));
        Ok(Value::Object(map))
    }
}

/// Model-based reranker configuration.
#[derive(Debug, Default, Clone)]
pub struct ModelConfiguration {
    /// Provider name.
    pub provider: Option<String>,
    /// Model name.
    pub model: Option<String>,
}

impl JsonRequestConvertible for ModelConfiguration {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        assign_if_some(
            &mut map,
            "provider",
            self.provider.clone().map(Value::String),
        );
        assign_if_some(&mut map, "model", self.model.clone().map(Value::String));
        Ok(Value::Object(map))
    }
}

/// Static-weight reranker configuration.
#[derive(Debug, Default, Clone)]
pub struct WeightConfiguration {
    /// Dense-vector weight (0..=1).
    pub dense_vector_search_weight: Option<f64>,
    /// Full-text weight (0..=1).
    pub full_text_search_weight: Option<f64>,
}

impl JsonRequestConvertible for WeightConfiguration {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        assign_if_some(
            &mut map,
            "denseVectorSearchWeight",
            self.dense_vector_search_weight
                .map(|v| serde_json::to_value(v).unwrap()),
        );
        assign_if_some(
            &mut map,
            "fullTextSearchWeight",
            self.full_text_search_weight
                .map(|v| serde_json::to_value(v).unwrap()),
        );
        Ok(Value::Object(map))
    }
}

/// Reranker top-level configuration.
#[derive(Debug, Clone)]
pub struct RerankingConfiguration {
    /// Algorithm to use. Defaults to RRF.
    pub reranking_type: RerankingType,
    /// Final number of results to return after reranking. Default `20`.
    pub number_of_results: u32,
    /// RRF parameters, used when `reranking_type` is `RRF`.
    pub rrf_configuration: Option<RRFConfiguration>,
    /// Model parameters, used when `reranking_type` is `MODEL`.
    pub model_configuration: Option<ModelConfiguration>,
    /// Static-weight parameters, used when `reranking_type` is `WEIGHT`.
    pub weight_configuration: Option<WeightConfiguration>,
}

impl Default for RerankingConfiguration {
    fn default() -> Self {
        Self {
            reranking_type: RerankingType::Rrf,
            number_of_results: 20,
            rrf_configuration: None,
            model_configuration: None,
            weight_configuration: None,
        }
    }
}

impl JsonRequestConvertible for RerankingConfiguration {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "type".to_string(),
            serde_json::to_value(self.reranking_type).unwrap(),
        );
        map.insert(
            "numberOfResults".to_string(),
            Value::Number(self.number_of_results.into()),
        );
        assign_if_some(
            &mut map,
            "rrfConfiguration",
            to_json_value(self.rrf_configuration.as_ref())?,
        );
        assign_if_some(
            &mut map,
            "modelConfiguration",
            to_json_value(self.model_configuration.as_ref())?,
        );
        assign_if_some(
            &mut map,
            "weightConfiguration",
            to_json_value(self.weight_configuration.as_ref())?,
        );
        Ok(Value::Object(map))
    }
}

/// Top-level retrieval configuration attached to a knowledge base or a single
/// `Retrieve` request.
#[derive(Debug, Default, Clone)]
pub struct RetrievalConfiguration {
    /// Search modes to combine.
    pub search_type: Option<Vec<SearchType>>,
    /// Dense-vector search knobs.
    pub dense_vector_search_configuration: Option<DenseVectorSearchConfiguration>,
    /// Full-text search knobs.
    pub full_text_search_configuration: Option<FullTextSearchConfiguration>,
    /// Reranker knobs.
    pub reranking_configuration: Option<RerankingConfiguration>,
    /// Optional metadata filter applied before search.
    pub filter: Option<MetadataFilter>,
}

impl JsonRequestConvertible for RetrievalConfiguration {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        assign_if_some(
            &mut map,
            "searchType",
            self.search_type.as_ref().map(|v| {
                Value::Array(v.iter().map(|t| serde_json::to_value(t).unwrap()).collect())
            }),
        );
        assign_if_some(
            &mut map,
            "denseVectorSearchConfiguration",
            to_json_value(self.dense_vector_search_configuration.as_ref())?,
        );
        assign_if_some(
            &mut map,
            "fullTextSearchConfiguration",
            to_json_value(self.full_text_search_configuration.as_ref())?,
        );
        assign_if_some(
            &mut map,
            "rerankingConfiguration",
            to_json_value(self.reranking_configuration.as_ref())?,
        );
        assign_if_some(&mut map, "filter", to_json_value(self.filter.as_ref())?);
        Ok(Value::Object(map))
    }
}

/// Request body for `Retrieve`.
#[derive(Debug, Clone)]
pub struct RetrieveRequest {
    /// Knowledge base identifier.
    pub knowledge_base_name: String,
    /// Query payload.
    pub retrieval_query: RetrievalQuery,
    /// Subspaces to scope the search to.
    pub subspace: Option<Vec<String>>,
    /// Per-request retrieval configuration (overrides the KB default).
    pub retrieval_configuration: Option<RetrievalConfiguration>,
}

impl RetrieveRequest {
    /// Build with only the required fields.
    pub fn new(knowledge_base_name: impl Into<String>, retrieval_query: RetrievalQuery) -> Self {
        Self {
            knowledge_base_name: knowledge_base_name.into(),
            retrieval_query,
            subspace: None,
            retrieval_configuration: None,
        }
    }
}

impl JsonRequestConvertible for RetrieveRequest {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        let mut map = Map::new();
        map.insert(
            "knowledgeBaseName".to_string(),
            Value::String(self.knowledge_base_name.clone()),
        );
        map.insert(
            "retrievalQuery".to_string(),
            self.retrieval_query.to_json_request()?,
        );
        assign_if_some(
            &mut map,
            "subspace",
            self.subspace
                .as_ref()
                .map(|v| Value::Array(v.iter().cloned().map(Value::String).collect())),
        );
        assign_if_some(
            &mut map,
            "retrievalConfiguration",
            to_json_value(self.retrieval_configuration.as_ref())?,
        );
        Ok(Value::Object(map))
    }
}
