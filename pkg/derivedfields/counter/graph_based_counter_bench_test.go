package counter

import (
	"context"
	"math"
	"testing"

	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dbhelper"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numFroms = 10000
)

var (
	prefix4    = []byte("pre4")
	prefixPath = [][]byte{prefix1, prefix2, prefix3, prefix4}
)

func TestLinearGraphDerivedFieldCounting(t *testing.T) {
	linkFactor := 1
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
		filtered.WithReadAccess(),
	)
	require.NoError(t, err, "filter creation should have succeeded")

	db, dacky := setupTest(t)
	froms := generateGraph(t, dacky, prefixPath, linkFactor)
	counter := NewGraphBasedDerivedFieldCounter(dacky, dackbox.Path{Path: prefixPath, ForwardTraversal: true}, filter)

	expectedCounts := getExpectedCounts(froms, linkFactor, len(prefixPath))

	actualCounts, err := counter.Count(ctx, froms...)
	assert.NoError(t, err)
	assert.EqualValues(t, expectedCounts, actualCounts)

	rocksdbtest.TearDownRocksDB(db)
}

func BenchmarkLinearGraphDerivedFieldCounting(b *testing.B) {
	linkFactor := 1
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
		filtered.WithReadAccess(),
	)
	require.NoError(b, err, "filter creation should have succeeded")

	db, dacky := setupTest(b)
	froms := generateGraph(b, dacky, prefixPath, linkFactor)
	counter := NewGraphBasedDerivedFieldCounter(dacky, dackbox.Path{Path: prefixPath, ForwardTraversal: true}, filter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := counter.Count(ctx, froms...)
		require.NoError(b, err)
	}
	b.StopTimer()

	rocksdbtest.TearDownRocksDB(db)
}

func BenchmarkBranchedGraphDerivedFieldCounting(b *testing.B) {
	linkFactor := 2
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
		filtered.WithReadAccess(),
	)
	require.NoError(b, err, "filter creation should have succeeded")

	db, dacky := setupTest(b)
	froms := generateGraph(b, dacky, prefixPath, linkFactor)
	counter := NewGraphBasedDerivedFieldCounter(dacky, dackbox.Path{Path: prefixPath, ForwardTraversal: true}, filter)

	expectedCounts := getExpectedCounts(froms, linkFactor, len(prefixPath))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		actualCounts, err := counter.Count(ctx, froms...)
		require.NoError(b, err)
		require.EqualValues(b, expectedCounts, actualCounts)
	}
	b.StopTimer()

	rocksdbtest.TearDownRocksDB(db)
}

func getExpectedCounts(froms []string, linkFactor, graphDepth int) map[string]int32 {
	expectedCounts := make(map[string]int32)
	for _, from := range froms {
		expectedCounts[from] = int32(math.Pow(float64(linkFactor), float64(graphDepth-1)))
	}

	return expectedCounts
}

func setupTest(t require.TestingT) (*rocksdb.RocksDB, *dackbox.DackBox) {
	db, err := rocksdb.NewTemp("reference")
	require.NoErrorf(t, err, "failed to create DB")

	dacky, err := dackbox.NewRocksDBDackBox(db, nil, []byte{}, []byte{}, []byte{})
	require.NoErrorf(t, err, "failed to create counter")

	return db, dacky
}

func generateGraph(t require.TestingT, dacky *dackbox.DackBox, prefixPath [][]byte, edgeFactor int) []string {
	froms := make([]string, 0, numFroms)

	if len(prefixPath) == 0 {
		return froms
	}

	for i := 0; i < numFroms; i++ {
		from := uuid.NewV4().String()
		froms = append(froms, from)
		genSubGraph(t, dacky, dbhelper.GetBucketKey(prefixPath[0], []byte(from)), prefixPath, 0, edgeFactor)
	}
	return froms
}

func genSubGraph(t require.TestingT, dacky *dackbox.DackBox, from []byte, prefixPath [][]byte, level, edgeFactor int) {
	if level+1 >= len(prefixPath) {
		return
	}

	tos := make([][]byte, 0, edgeFactor)
	for edge := 0; edge < edgeFactor; edge++ {
		to := dbhelper.GetBucketKey(prefixPath[level+1], []byte(uuid.NewV4().String()))
		tos = append(tos, to)
		addLink(t, dacky, from, to)
	}

	for _, to := range tos {
		genSubGraph(t, dacky, to, prefixPath, level+1, edgeFactor)
	}
}

func addLink(t require.TestingT, dacky *dackbox.DackBox, from []byte, to []byte) {
	view, err := dacky.NewTransaction()
	assert.NoError(t, err)
	defer view.Discard()

	view.Graph().AddRefs(from, to)
	err = view.Commit()
	require.NoError(t, err, "commit should have succeeded")
}
