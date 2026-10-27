use serde_json::Value;

use crate::error::ClientError;

/// Marker trait for types that know how to serialise themselves into the
/// concrete on-wire JSON expected by the OTS service.
///
/// All built-in request models implement this trait. User code rarely calls
/// it directly — pass the model into a client method and the client invokes
/// it via [`crate::serialization::to_json_request`].
pub trait JsonRequestConvertible {
    /// Render the request as a [`serde_json::Value`] matching the
    /// service-defined schema.
    fn to_json_request(&self) -> Result<Value, ClientError>;
}

impl JsonRequestConvertible for Value {
    fn to_json_request(&self) -> Result<Value, ClientError> {
        Ok(self.clone())
    }
}

/// Helper used by hand-rolled `to_json_request` impls to omit `None` fields
/// while preserving explicit `null`s that come back from [`super::Patch::Clear`].
pub(crate) fn assign_if_some(
    target: &mut serde_json::Map<String, Value>,
    key: &str,
    value: Option<Value>,
) {
    if let Some(value) = value {
        target.insert(key.to_string(), value);
    }
}

/// Serialize an optional inner model, propagating any error from its
/// `to_json_request` implementation.
pub(crate) fn to_json_value<T: JsonRequestConvertible>(
    value: Option<&T>,
) -> Result<Option<Value>, ClientError> {
    match value {
        Some(value) => Ok(Some(value.to_json_request()?)),
        None => Ok(None),
    }
}

/// Public re-export to keep `models::*` self-contained: wraps any
/// [`serde::Serialize`] value as a JSON [`Value`], returning [`Value::Null`] on
/// `None`. Useful when stitching request bodies by hand.
pub fn json_value_or_null<T: serde::Serialize>(value: Option<&T>) -> Result<Value, ClientError> {
    match value {
        Some(value) => serde_json::to_value(value).map_err(|err| {
            ClientError::InvalidArgument(format!("failed to serialize value: {err}"))
        }),
        None => Ok(Value::Null),
    }
}
