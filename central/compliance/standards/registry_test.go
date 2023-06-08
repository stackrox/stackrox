//go:build sql_integration

package standards

import (
	"context"
	"testing"

	pgControl "github.com/stackrox/rox/central/compliance/standards/control"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	pgStandard "github.com/stackrox/rox/central/compliance/standards/standard"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexer(t *testing.T) {
	tp := pgtest.ForT(t)
	defer tp.Close()

	standardStore := pgStandard.New(tp)
	standardIndexer := pgStandard.NewIndexer(tp)

	controlStore := pgControl.New(tp)
	controlIndexer := pgControl.NewIndexer(tp)

	registry, err := NewRegistry(standardStore, standardIndexer, controlStore, controlIndexer, nil, metadata.AllStandards...)
	require.NoError(t, err)

	ctx := sac.WithAllAccess(context.Background())

	results, err := registry.SearchStandards(ctx, search.NewQueryBuilder().AddStrings(search.StandardID, "pci").ProtoQuery())
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}
