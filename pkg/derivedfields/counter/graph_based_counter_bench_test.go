package counter

import (
	"context"
	"math"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/uuid"
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
	)
	require.NoError(t, err, "filter creation should have succeeded")

	db, dacky, dir := setupTest(t)
	froms := generateGraph(t, dacky, prefixPath, linkFactor)
	counter := NewGraphBasedDerivedFieldCounter(dacky, dackbox.Path{Path: prefixPath, ForwardTraversal: true}, filter)

	expectedCounts := getExpectedCounts(froms, linkFactor, len(prefixPath))

	actualCounts, err := counter.Count(ctx, froms...)
	assert.NoError(t, err)
	assert.EqualValues(t, expectedCounts, actualCounts)

	tearDown(db, dir)
}

func BenchmarkLinearGraphDerivedFieldCounting(b *testing.B) {
	linkFactor := 1
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
	)
	require.NoError(b, err, "filter creation should have succeeded")

	db, dacky, dir := setupTest(b)
	froms := generateGraph(b, dacky, prefixPath, linkFactor)
	counter := NewGraphBasedDerivedFieldCounter(dacky, dackbox.Path{Path: prefixPath, ForwardTraversal: true}, filter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := counter.Count(ctx, froms...)
		require.NoError(b, err)
	}

	tearDown(db, dir)
}

func BenchmarkBranchedGraphDerivedFieldCounting(b *testing.B) {
	linkFactor := 2
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	filter, err := filtered.NewSACFilter(
		filtered.WithResourceHelper(sac.ForResource(globalResource)),
	)
	require.NoError(b, err, "filter creation should have succeeded")

	db, dacky, dir := setupTest(b)
	froms := generateGraph(b, dacky, prefixPath, linkFactor)
	counter := NewGraphBasedDerivedFieldCounter(dacky, dackbox.Path{Path: prefixPath, ForwardTraversal: true}, filter)

	expectedCounts := getExpectedCounts(froms, linkFactor, len(prefixPath))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		actualCounts, err := counter.Count(ctx, froms...)
		require.NoError(b, err)
		require.EqualValues(b, expectedCounts, actualCounts)
	}

	tearDown(db, dir)
}

func getExpectedCounts(froms []string, linkFactor, graphDepth int) map[string]int32 {
	expectedCounts := make(map[string]int32)
	for _, from := range froms {
		expectedCounts[from] = int32(math.Pow(float64(linkFactor), float64(graphDepth-1)))
	}

	return expectedCounts
}

func setupTest(t require.TestingT) (*badger.DB, *dackbox.DackBox, string) {
	db, dir, err := badgerhelper.NewTemp("reference")
	require.NoErrorf(t, err, "failed to create DB")

	dacky, err := dackbox.NewDackBox(db, nil, []byte{}, []byte{}, []byte{})
	require.NoErrorf(t, err, "failed to create counter")

	return db, dacky, dir
}

func generateGraph(t require.TestingT, dacky *dackbox.DackBox, prefixPath [][]byte, edgeFactor int) []string {
	froms := make([]string, 0, numFroms)

	if len(prefixPath) == 0 {
		return froms
	}

	for i := 0; i < numFroms; i++ {
		from := uuid.NewV4().String()
		froms = append(froms, from)
		genSubGraph(t, dacky, badgerhelper.GetBucketKey(prefixPath[0], []byte(from)), prefixPath, 0, edgeFactor)
	}
	return froms
}

func genSubGraph(t require.TestingT, dacky *dackbox.DackBox, from []byte, prefixPath [][]byte, level, edgeFactor int) {
	if level+1 >= len(prefixPath) {
		return
	}

	tos := make([][]byte, 0, edgeFactor)
	for edge := 0; edge < edgeFactor; edge++ {
		to := badgerhelper.GetBucketKey(prefixPath[level+1], []byte(uuid.NewV4().String()))
		tos = append(tos, to)
		addLink(t, dacky, from, to)
	}

	for _, to := range tos {
		genSubGraph(t, dacky, to, prefixPath, level+1, edgeFactor)
	}
}

func addLink(t require.TestingT, dacky *dackbox.DackBox, from []byte, to []byte) {
	view := dacky.NewTransaction()
	defer view.Discard()

	err := view.Graph().AddRefs(from, to)
	require.NoError(t, err, "addRef should have succeeded")
	err = view.Commit()
	require.NoError(t, err, "commit should have succeeded")
}

func tearDown(db *badger.DB, dir string) {
	_ = db.Close()
	_ = os.RemoveAll(dir)
}
