//! Helpers that translate typed request models into the on-wire JSON used by
//! [`crate::http::OtsJsonClient`].

use serde_json::Value;

use crate::error::ClientError;
use crate::models::JsonRequestConvertible;

/// Convert any [`JsonRequestConvertible`] into a [`serde_json::Value`].
///
/// Use this when you build a request by hand and want to drive
/// [`crate::http::OtsJsonClient::request`] directly.
pub fn to_json_request<T: JsonRequestConvertible>(value: &T) -> Result<Value, ClientError> {
    value.to_json_request()
}
