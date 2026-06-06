//go:build sql_integration

package stats

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPGIndexStats(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	tp := pgtest.ForT(t)
	defer tp.Close()

	result := GetPGIndexStats(ctx, tp.DB, 100)
	require.Empty(t, result.Error)
	assert.NotNil(t, result.Indexes)
	for _, idx := range result.Indexes {
		assert.NotEmpty(t, idx.TableName)
		assert.NotEmpty(t, idx.IndexName)
		assert.NotEmpty(t, idx.IndexType)
		assert.True(t, idx.IsValid, "index %s on %s should be valid", idx.IndexName, idx.TableName)
	}
}
