//go:build sql_integration

package postgres

import (
	"testing"

	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
)

func TestSchemaExists(t *testing.T) {
	if pkgSchema.ProcessIndicatorNoSerializedChildSchema == nil {
		t.Fatal("Schema is nil!")
	}
	t.Logf("Schema table: %s", pkgSchema.ProcessIndicatorNoSerializedChildSchema.Table)

	// Check if it's registered
	registered := pkgSchema.GetSchemaForTable("process_indicator_no_serialized_child")
	if registered == nil {
		t.Fatal("Schema not registered!")
	}
	t.Logf("Registered schema: %v", registered)
}
