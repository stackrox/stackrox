package tests

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"strings"
	"testing"

	. "github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/dackbox/graph/testutils"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func bucketHandler(prefix string) *dbhelper.BucketHandler {
	return &dbhelper.BucketHandler{BucketPrefix: []byte(prefix + ":")}
}

var (
	cveHandler        = bucketHandler("cve")
	componentHandler  = bucketHandler("component")
	imageHandler      = bucketHandler("image")
	deploymentHandler = bucketHandler("deployment")
	nsHandler         = bucketHandler("namespace")
	clusterHandler    = bucketHandler("cluster")
	nodeHandler       = bucketHandler("node")
)

func TestCachedSearch(t *testing.T) {
	searchPath := BackwardsBucketPath(
		cveHandler,
		componentHandler,
		imageHandler,
		deploymentHandler,
		nsHandler,
		clusterHandler,
	)
	irrelevantPath := BackwardsBucketPath(
		cveHandler,
		componentHandler,
		nodeHandler,
		clusterHandler,
	)

	targetPred := func(key []byte) (bool, error) {
		if !bytes.HasPrefix(key, clusterHandler.BucketPrefix) {
			panic("target object should always be in the cluster bucket")
		}
		targetID := clusterHandler.GetID(key)
		if strings.HasPrefix(targetID, "unreachable") {
			panic("encountered an unreachable node")
		}
		if strings.HasPrefix(targetID, "irrelevant") {
			panic("encountered a destination node that should be irrelevant")
		}
		if strings.HasPrefix(targetID, "error") {
			return false, errors.New("found an error node")
		}
		return strings.HasPrefix(targetID, "target"), nil
	}

	idPool := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		idPool = append(idPool, uuid.NewV4().String())
	}

	g := graph.NewGraph()

	// Add 1000 paths to the graph, all from the same start node
	for i := 0; i < 1000; i++ {
		var ids []string
		ids = append(ids, "start")
		for range searchPath.Elements[1:] {
			ids = append(ids, idPool[rand.Int()%len(idPool)])
		}
		testutils.AddPathsToGraph(g, searchPath.KeyPath(ids...))
	}
	// Add 1000 paths that aren't reachable from the start node
	for i := 0; i < 1000; i++ {
		var ids []string
		ids = append(ids, "not-start")
		for range searchPath.Elements[1:] {
			ids = append(ids, "unreachable-"+idPool[rand.Int()%len(idPool)])
		}
		testutils.AddPathsToGraph(g, searchPath.KeyPath(ids...))
	}
	// Add 1000 paths from the start node that aren't relevant
	for i := 0; i < 1000; i++ {
		var ids []string
		ids = append(ids, "start")
		for range irrelevantPath.Elements[1:] {
			ids = append(ids, "irrelevant-"+idPool[rand.Int()%len(idPool)])
		}
		testutils.AddPathsToGraph(g, irrelevantPath.KeyPath(ids...))
	}

	t.Run("unsuccessful search", func(t *testing.T) {
		// Graph contains no target yet, so search should not succeed (no error).
		searcher := NewCachedSearcher(g, targetPred, searchPath)
		found, err := searcher.Search(context.Background(), "start")
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("successful search", func(t *testing.T) {
		succG := g.Copy()

		// Add a single successful path
		var ids []string
		ids = append(ids, "start")
		for range searchPath.Elements[1:] {
			ids = append(ids, "target-"+idPool[rand.Int()%len(idPool)])
		}
		testutils.AddPathsToGraph(succG, searchPath.KeyPath(ids...))

		searcher := NewCachedSearcher(succG, targetPred, searchPath)
		found, err := searcher.Search(context.Background(), "start")
		assert.NoError(t, err)
		assert.True(t, found)
	})

	t.Run("error search", func(t *testing.T) {
		errG := g.Copy()

		// Add a single error path
		var ids []string
		ids = append(ids, "start")
		for range searchPath.Elements[1:] {
			ids = append(ids, "error-"+idPool[rand.Int()%len(idPool)])
		}
		testutils.AddPathsToGraph(errG, searchPath.KeyPath(ids...))

		searcher := NewCachedSearcher(errG, targetPred, searchPath)
		_, err := searcher.Search(context.Background(), "start")
		assert.Error(t, err)
	})
}
