use serde::{Serialize, Serializer};

/// Three-state field used by partial-update requests.
///
/// The TypeScript SDK uses an `UNSET` sentinel to distinguish "leave this
/// field alone" from "explicitly clear it". Rust expresses the same shape as
/// a small enum:
///
/// - [`Patch::Unchanged`] — omit the field from the JSON body.
/// - [`Patch::Set`] — send the given value.
/// - [`Patch::Clear`] — send `null` to clear it on the server.
///
/// When serialised inside an `Option` wrapper, `Unchanged` becomes `None`.
#[derive(Debug, Default, Clone, PartialEq, Eq)]
pub enum Patch<T> {
    /// Leave the existing server-side value untouched.
    #[default]
    Unchanged,
    /// Replace the existing value with `T`.
    Set(T),
    /// Explicitly clear the existing value.
    Clear,
}

impl<T> Patch<T> {
    /// `true` when the patch should be omitted from the request body.
    pub fn is_unchanged(&self) -> bool {
        matches!(self, Patch::Unchanged)
    }

    /// Borrow the inner value if the patch sets one.
    pub fn as_set(&self) -> Option<&T> {
        match self {
            Patch::Set(value) => Some(value),
            _ => None,
        }
    }
}

impl<T> From<T> for Patch<T> {
    fn from(value: T) -> Self {
        Patch::Set(value)
    }
}

impl<T> From<Option<T>> for Patch<T> {
    fn from(value: Option<T>) -> Self {
        match value {
            Some(value) => Patch::Set(value),
            None => Patch::Unchanged,
        }
    }
}

/// `Serialize` cannot distinguish [`Patch::Unchanged`] from [`Patch::Clear`]
/// on its own — both emit `null`, because serde has no way to signal "skip
/// this field" from inside a `Serialize` impl. To get the "omit vs. send
/// null" behaviour the wire protocol needs, use `#[serde(skip_serializing_if =
/// "Patch::is_unchanged")]` on each `Patch<T>` field at the containing
/// struct:
///
/// ```ignore
/// #[derive(Serialize)]
/// struct UpdateThing {
///     #[serde(skip_serializing_if = "Patch::is_unchanged")]
///     description: Patch<String>,
/// }
/// ```
///
/// The hand-written [`JsonRequestConvertible`](crate::models::JsonRequestConvertible)
/// impls in this crate already match on the three variants directly and so do
/// not need that attribute.
impl<T: Serialize> Serialize for Patch<T> {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        match self {
            Patch::Unchanged => serializer.serialize_none(),
            Patch::Set(value) => value.serialize(serializer),
            Patch::Clear => serializer.serialize_none(),
        }
    }
}
