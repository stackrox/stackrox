package postgres

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stretchr/testify/assert"
)

// AssertSQLQueryString a utility function for test purpose.
func AssertSQLQueryString(t testing.TB, q *v1.Query, schema *walker.Schema, expected string) {
	// No type info in test utility, so arrayFields is nil
	actual, err := standardizeSelectQueryAndPopulatePath(context.Background(), q, schema, SELECT, nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual.AsSQL())
}
