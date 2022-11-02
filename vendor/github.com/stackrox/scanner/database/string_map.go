package database

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/stackrox/rox/pkg/set"
)

// NewStringToStringsMap creates a NewStringToStringsMap from a string to string set map.
func NewStringToStringsMap(m map[string]set.StringSet) interface {
	driver.Valuer
	sql.Scanner
} {
	return (*StringToStringsMap)(&m)
}

// StringToStringsMap defines driver.Valuer and sql.Scanner for a map from string to set of string
type StringToStringsMap map[string]set.StringSet

type internalMap map[string][]string

// Value returns the JSON-encoded representation
func (m StringToStringsMap) Value() (driver.Value, error) {
	if m == nil {
		return json.Marshal(nil)
	}
	converted := make(internalMap, len(m))
	for k, v := range m {
		converted[k] = v.AsSortedSlice(func(i, j string) bool {
			return i < j
		})
	}
	return json.Marshal(converted)
}

// Scan Decodes a JSON-encoded value
func (m *StringToStringsMap) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	// Unmarshal from json to map[string][]string
	var raw internalMap
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw == nil {
		*m = nil
		return nil
	}
	scanned := make(StringToStringsMap, len(raw))
	for k, v := range raw {
		scanned[k] = set.NewStringSet(v...)
	}
	*m = scanned
	return nil
}

// Merge merges map b to map a.
// If:
//
//	a contains str_a -> ["a", "b"]
//	b contains str_a -> ["b", "c"]
//
// Then after merging:
//
//	a contains str_a -> {"a", "b", "c"}
func (m *StringToStringsMap) Merge(b StringToStringsMap) {
	if len(b) == 0 {
		return
	}
	if *m == nil {
		*m = make(StringToStringsMap)
	}
	for k, v := range b {
		(*m)[k] = (*m)[k].Union(v)
	}
}
