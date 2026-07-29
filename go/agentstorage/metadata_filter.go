package agentstorage

import (
	"encoding/json"
	"fmt"
)

// KeyValueCondition is the {key, value} operand of a comparison filter.
type KeyValueCondition struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// MetadataFilter expresses a metadata filter for knowledge-base retrieval.
// A filter holds exactly one operator; combine filters with AndAll / OrAll /
// NotAll. Construct instances with the factory functions (Equals, In, ...) or
// with NewMetadataFilterBuilder.
type MetadataFilter struct {
	payload map[string]any
}

// MarshalJSON implements json.Marshaler.
func (f MetadataFilter) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.payload)
}

func newConditionFilter(operator, key string, value any) *MetadataFilter {
	return &MetadataFilter{payload: map[string]any{operator: KeyValueCondition{Key: key, Value: value}}}
}

func newLogicalFilter(operator string, filters []*MetadataFilter) *MetadataFilter {
	payloads := make([]map[string]any, 0, len(filters))
	for _, filter := range filters {
		payloads = append(payloads, filter.payload)
	}
	return &MetadataFilter{payload: map[string]any{operator: payloads}}
}

// Equals matches records whose metadata value equals value.
func Equals(key string, value any) *MetadataFilter { return newConditionFilter("equals", key, value) }

// NotEquals matches records whose metadata value differs from value.
func NotEquals(key string, value any) *MetadataFilter {
	return newConditionFilter("notEquals", key, value)
}

// GreaterThan matches records whose metadata value is greater than value.
func GreaterThan(key string, value any) *MetadataFilter {
	return newConditionFilter("greaterThan", key, value)
}

// GreaterThanOrEquals matches records whose metadata value is >= value.
func GreaterThanOrEquals(key string, value any) *MetadataFilter {
	return newConditionFilter("greaterThanOrEquals", key, value)
}

// LessThan matches records whose metadata value is less than value.
func LessThan(key string, value any) *MetadataFilter {
	return newConditionFilter("lessThan", key, value)
}

// LessThanOrEquals matches records whose metadata value is <= value.
func LessThanOrEquals(key string, value any) *MetadataFilter {
	return newConditionFilter("lessThanOrEquals", key, value)
}

// InList matches records whose metadata value is one of values.
func InList(key string, values []string) *MetadataFilter {
	return newConditionFilter("in", key, values)
}

// NotInList matches records whose metadata value is none of values.
func NotInList(key string, values []string) *MetadataFilter {
	return newConditionFilter("notIn", key, values)
}

// StartsWith matches string metadata values with the given prefix.
func StartsWith(key, value string) *MetadataFilter {
	return newConditionFilter("startsWith", key, value)
}

// StringContains matches string metadata values containing value.
func StringContains(key, value string) *MetadataFilter {
	return newConditionFilter("stringContains", key, value)
}

// ListContains matches list metadata values containing value.
func ListContains(key, value string) *MetadataFilter {
	return newConditionFilter("listContains", key, value)
}

// AndAll combines filters with logical AND.
func AndAll(filters ...*MetadataFilter) *MetadataFilter { return newLogicalFilter("andAll", filters) }

// OrAll combines filters with logical OR.
func OrAll(filters ...*MetadataFilter) *MetadataFilter { return newLogicalFilter("orAll", filters) }

// NotAll negates the combined filters.
func NotAll(filters ...*MetadataFilter) *MetadataFilter { return newLogicalFilter("notAll", filters) }

// MetadataFilterBuilder accumulates conditions and combines them with
// BuildAnd / BuildOr / BuildNot.
type MetadataFilterBuilder struct {
	filters []*MetadataFilter
}

// NewMetadataFilterBuilder returns an empty builder.
func NewMetadataFilterBuilder() *MetadataFilterBuilder {
	return &MetadataFilterBuilder{}
}

// Add appends a prebuilt filter.
func (b *MetadataFilterBuilder) Add(filter *MetadataFilter) *MetadataFilterBuilder {
	b.filters = append(b.filters, filter)
	return b
}

// Equals appends an equals condition.
func (b *MetadataFilterBuilder) Equals(key string, value any) *MetadataFilterBuilder {
	return b.Add(Equals(key, value))
}

// NotEquals appends a notEquals condition.
func (b *MetadataFilterBuilder) NotEquals(key string, value any) *MetadataFilterBuilder {
	return b.Add(NotEquals(key, value))
}

// GreaterThan appends a greaterThan condition.
func (b *MetadataFilterBuilder) GreaterThan(key string, value any) *MetadataFilterBuilder {
	return b.Add(GreaterThan(key, value))
}

// GreaterThanOrEquals appends a greaterThanOrEquals condition.
func (b *MetadataFilterBuilder) GreaterThanOrEquals(key string, value any) *MetadataFilterBuilder {
	return b.Add(GreaterThanOrEquals(key, value))
}

// LessThan appends a lessThan condition.
func (b *MetadataFilterBuilder) LessThan(key string, value any) *MetadataFilterBuilder {
	return b.Add(LessThan(key, value))
}

// LessThanOrEquals appends a lessThanOrEquals condition.
func (b *MetadataFilterBuilder) LessThanOrEquals(key string, value any) *MetadataFilterBuilder {
	return b.Add(LessThanOrEquals(key, value))
}

// InList appends an in condition.
func (b *MetadataFilterBuilder) InList(key string, values []string) *MetadataFilterBuilder {
	return b.Add(InList(key, values))
}

// NotInList appends a notIn condition.
func (b *MetadataFilterBuilder) NotInList(key string, values []string) *MetadataFilterBuilder {
	return b.Add(NotInList(key, values))
}

// StartsWith appends a startsWith condition.
func (b *MetadataFilterBuilder) StartsWith(key, value string) *MetadataFilterBuilder {
	return b.Add(StartsWith(key, value))
}

// StringContains appends a stringContains condition.
func (b *MetadataFilterBuilder) StringContains(key, value string) *MetadataFilterBuilder {
	return b.Add(StringContains(key, value))
}

// ListContains appends a listContains condition.
func (b *MetadataFilterBuilder) ListContains(key, value string) *MetadataFilterBuilder {
	return b.Add(ListContains(key, value))
}

func (b *MetadataFilterBuilder) build(operator string, combine func(...*MetadataFilter) *MetadataFilter) (*MetadataFilter, error) {
	if len(b.filters) == 0 {
		return nil, fmt.Errorf("cannot build %s filter with no conditions", operator)
	}
	if len(b.filters) == 1 {
		return b.filters[0], nil
	}
	return combine(b.filters...), nil
}

// BuildAnd combines the accumulated conditions with AND. A single condition
// is returned as-is.
func (b *MetadataFilterBuilder) BuildAnd() (*MetadataFilter, error) {
	return b.build("andAll", AndAll)
}

// BuildOr combines the accumulated conditions with OR. A single condition is
// returned as-is.
func (b *MetadataFilterBuilder) BuildOr() (*MetadataFilter, error) {
	return b.build("orAll", OrAll)
}

// BuildNot negates the accumulated conditions. A single condition is
// returned as-is.
func (b *MetadataFilterBuilder) BuildNot() (*MetadataFilter, error) {
	return b.build("notAll", NotAll)
}
