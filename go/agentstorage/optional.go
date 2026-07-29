package agentstorage

import "encoding/json"

// Nullable distinguishes the three states an updatable field can be in:
//
//   - field is a nil *Nullable[T]  -> omitted from the request (leave unchanged)
//   - Null[T]()                    -> serialized as JSON null (clear the value)
//   - NullableOf(v)                -> serialized as v
//
// It mirrors the UNSET sentinel of the Python/TypeScript SDKs.
type Nullable[T any] struct {
	value *T
}

// NullableOf wraps a concrete value.
func NullableOf[T any](v T) *Nullable[T] {
	return &Nullable[T]{value: &v}
}

// Null returns a Nullable that serializes to JSON null.
func Null[T any]() *Nullable[T] {
	return &Nullable[T]{}
}

// MarshalJSON implements json.Marshaler.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if n.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.value = &v
	return nil
}

// Value returns the wrapped value and whether it is non-null.
func (n *Nullable[T]) Value() (T, bool) {
	if n == nil || n.value == nil {
		var zero T
		return zero, false
	}
	return *n.value, true
}

// Pointer helpers for optional scalar request fields.

// String returns a pointer to v.
func String(v string) *string { return &v }

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }

// Int returns a pointer to v.
func Int(v int) *int { return &v }

// Int64 returns a pointer to v.
func Int64(v int64) *int64 { return &v }

// Float64 returns a pointer to v.
func Float64(v float64) *float64 { return &v }

// Strings returns a pointer to the given values, for optional list fields that
// distinguish "omitted" (nil) from "set to the empty list" (Strings()).
func Strings(v ...string) *[]string {
	if v == nil {
		v = []string{}
	}
	return &v
}
