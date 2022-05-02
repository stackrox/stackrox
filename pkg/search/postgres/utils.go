package postgres

import (
	"strings"
)

const (
	// IDSeparator is the separator used in IDs constructed from multiple primary keys.
	IDSeparator = "#"
)

// IDFromPks generates a composite ID string from input primary keys.
func IDFromPks(pks []string) string {
	return strings.Join(pks, IDSeparator)
}

// IDToParts returns the parts (primary keys) that make up a composite ID.
func IDToParts(id string) []string {
	return strings.Split(id, IDSeparator)
}
