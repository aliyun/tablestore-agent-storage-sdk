use serde::Serialize;
use serde_json::{json, Value};

use crate::error::ClientError;

use super::base::JsonRequestConvertible;

/// Scalar value accepted by metadata filters.
#[derive(Debug, Clone, Serialize, PartialEq)]
#[serde(untagged)]
pub enum PrimitiveValue {
    /// String literal.
    String(String),
    /// Integer literal.
    Integer(i64),
    /// Floating-point literal.
    Double(f64),
    /// Boolean literal.
    Boolean(bool),
    /// List of strings.
    StringList(Vec<String>),
    /// List of integers.
    IntegerList(Vec<i64>),
    /// List of doubles.
    DoubleList(Vec<f64>),
}

impl From<&str> for PrimitiveValue {
    fn from(value: &str) -> Self {
        PrimitiveValue::String(value.to_string())
    }
}

impl From<String> for PrimitiveValue {
    fn from(value: String) -> Self {
        PrimitiveValue::String(value)
    }
}

impl From<i64> for PrimitiveValue {
    fn from(value: i64) -> Self {
        PrimitiveValue::Integer(value)
    }
}

impl From<i32> for PrimitiveValue {
    fn from(value: i32) -> Self {
        PrimitiveValue::Integer(value as i64)
    }
}

impl From<f64> for PrimitiveValue {
    fn from(value: f64) -> Self {
        PrimitiveValue::Double(value)
    }
}

impl From<bool> for PrimitiveValue {
    fn from(value: bool) -> Self {
        PrimitiveValue::Boolean(value)
    }
}

impl From<Vec<String>> for PrimitiveValue {
    fn from(value: Vec<String>) -> Self {
        PrimitiveValue::StringList(value)
    }
}

/// Composable metadata predicate used by [`crate::models::RetrievalConfiguration`].
///
/// Construct leaf predicates with the inherent `equals`, `greater_than`, …
/// methods, then combine them with [`MetadataFilter::and_all`],
/// [`MetadataFilter::or_all`], [`MetadataFilter::not_all`].
///
/// ```
/// use tablestore_agent_storage::models::{MetadataFilter, PrimitiveValue};
///
/// let filter = MetadataFilter::and_all(vec![
///     MetadataFilter::equals("category", PrimitiveValue::from("news")),
///     MetadataFilter::greater_than("score", PrimitiveValue::from(0.5)),
/// ]);
/// ```
#[derive(Debug, Clone, PartialEq)]
pub struct MetadataFilter {
    payload: Value,
}

impl MetadataFilter {
    fn leaf(op: &str, key: impl Into<String>, value: PrimitiveValue) -> Self {
        let key = key.into();
        let payload = json!({ op: { "key": key, "value": value } });
        Self { payload }
    }

    /// `key == value`.
    pub fn equals(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("equals", key, value)
    }

    /// `key != value`.
    pub fn not_equals(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("notEquals", key, value)
    }

    /// `key > value`.
    pub fn greater_than(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("greaterThan", key, value)
    }

    /// `key >= value`.
    pub fn greater_than_or_equals(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("greaterThanOrEquals", key, value)
    }

    /// `key < value`.
    pub fn less_than(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("lessThan", key, value)
    }

    /// `key <= value`.
    pub fn less_than_or_equals(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("lessThanOrEquals", key, value)
    }

    /// `key IN value`.
    pub fn in_list(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("in", key, value)
    }

    /// `key NOT IN value`.
    pub fn not_in_list(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("notIn", key, value)
    }

    /// `key STARTS WITH value`.
    pub fn starts_with(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("startsWith", key, value)
    }

    /// String `contains` predicate.
    pub fn string_contains(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("stringContains", key, value)
    }

    /// List `contains` predicate.
    pub fn list_contains(key: impl Into<String>, value: PrimitiveValue) -> Self {
        Self::leaf("listContains", key, value)
    }

    /// Logical AND of every sub-filter.
    pub fn and_all(filters: Vec<MetadataFilter>) -> Self {
        let items: Vec<Value> = filters.into_iter().map(|f| f.payload).collect();
        Self {
            payload: json!({ "andAll": items }),
        }
    }

    /// Logical OR of every sub-filter.
    pub fn or_all(filters: Vec<MetadataFilter>) -> Self {
        let items: Vec<Value> = filters.into_iter().map(|f| f.payload).collect();
        Self {
            payload: json!({ "orAll": items }),
        }
    }

    /// Logical NOT-AND of every sub-filter.
    pub fn not_all(filters: Vec<MetadataFilter>) -> Self {
        let items: Vec<Value> = filters.into_iter().map(|f| f.payload).collect();
        Self {
            payload: json!({ "notAll": items }),
        }
    }

    /// Start a fluent builder.
    pub fn builder() -> MetadataFilterBuilder {
        MetadataFilterBuilder::default()
    }
}

impl JsonRequestConvertible for MetadataFilter {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        Ok(self.payload.clone())
    }
}

/// Accumulating builder for combining several [`MetadataFilter`] predicates
/// before deciding on the boolean operator.
#[derive(Debug, Default)]
pub struct MetadataFilterBuilder {
    filters: Vec<MetadataFilter>,
}

impl MetadataFilterBuilder {
    /// Append a pre-built filter.
    pub fn push(mut self, filter: MetadataFilter) -> Self {
        self.filters.push(filter);
        self
    }

    /// Shorthand: `equals` leaf.
    pub fn equals(self, key: impl Into<String>, value: PrimitiveValue) -> Self {
        self.push(MetadataFilter::equals(key, value))
    }

    /// Shorthand: `not_equals` leaf.
    pub fn not_equals(self, key: impl Into<String>, value: PrimitiveValue) -> Self {
        self.push(MetadataFilter::not_equals(key, value))
    }

    /// Shorthand: `greater_than` leaf.
    pub fn greater_than(self, key: impl Into<String>, value: PrimitiveValue) -> Self {
        self.push(MetadataFilter::greater_than(key, value))
    }

    /// Combine via AND. Single condition is returned as-is.
    pub fn build_and(self) -> Result<MetadataFilter, ClientError> {
        self.build("and")
    }

    /// Combine via OR. Single condition is returned as-is.
    pub fn build_or(self) -> Result<MetadataFilter, ClientError> {
        self.build("or")
    }

    /// Combine via NOT-AND. Single condition is returned as-is.
    pub fn build_not(self) -> Result<MetadataFilter, ClientError> {
        self.build("not")
    }

    fn build(self, op: &'static str) -> Result<MetadataFilter, ClientError> {
        if self.filters.is_empty() {
            return Err(ClientError::InvalidArgument(format!(
                "cannot build {op} filter with no conditions"
            )));
        }
        if self.filters.len() == 1 {
            return Ok(self.filters.into_iter().next().expect("len == 1"));
        }
        Ok(match op {
            "and" => MetadataFilter::and_all(self.filters),
            "or" => MetadataFilter::or_all(self.filters),
            "not" => MetadataFilter::not_all(self.filters),
            _ => unreachable!("only and/or/not are passed in"),
        })
    }
}
