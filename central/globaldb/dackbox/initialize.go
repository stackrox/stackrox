package dackbox

import (
	"bytes"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	activeComponentDackBox "github.com/stackrox/rox/central/activecomponent/dackbox"
	activeComponentIndex "github.com/stackrox/rox/central/activecomponent/index"
	clusterCVEEdgeDackBox "github.com/stackrox/rox/central/clustercveedge/dackbox"
	clusterCVEEdgeIndex "github.com/stackrox/rox/central/clustercveedge/index"
	componentCVEEdgeDackBox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	imageIndex "github.com/stackrox/rox/central/image/index"
	imagecomponentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	imagecomponentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imagecomponentEdgeDackBox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imagecomponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeDackbox "github.com/stackrox/rox/central/imagecveedge/dackbox"
	imageCVEEdgeIndex "github.com/stackrox/rox/central/imagecveedge/index"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeDackBox "github.com/stackrox/rox/central/nodecomponentedge/dackbox"
	nodeComponentEdgeIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/debug"
)

type bucketRef struct {
	bucket   []byte
	reader   crud.Reader
	category v1.SearchCategory
	wrapper  indexer.Wrapper
}

var (
	initializedBuckets = []bucketRef{
		{
			bucket:   cveDackBox.Bucket,
			reader:   cveDackBox.Reader,
			category: v1.SearchCategory_VULNERABILITIES,
			wrapper:  cveIndex.Wrapper{},
		},
		{
			bucket:   componentCVEEdgeDackBox.Bucket,
			reader:   componentCVEEdgeDackBox.Reader,
			category: v1.SearchCategory_COMPONENT_VULN_EDGE,
			wrapper:  componentCVEEdgeIndex.Wrapper{},
		},
		{
			bucket:   clusterCVEEdgeDackBox.Bucket,
			reader:   clusterCVEEdgeDackBox.Reader,
			category: v1.SearchCategory_CLUSTER_VULN_EDGE,
			wrapper:  clusterCVEEdgeIndex.Wrapper{},
		},
		{
			bucket:   imagecomponentDackBox.Bucket,
			reader:   imagecomponentDackBox.Reader,
			category: v1.SearchCategory_IMAGE_COMPONENTS,
			wrapper:  imagecomponentIndex.Wrapper{},
		},
		{
			bucket:   imagecomponentEdgeDackBox.Bucket,
			reader:   imagecomponentEdgeDackBox.Reader,
			category: v1.SearchCategory_IMAGE_COMPONENT_EDGE,
			wrapper:  imagecomponentEdgeIndex.Wrapper{},
		},
		{
			bucket:   imageDackBox.Bucket,
			reader:   imageDackBox.Reader,
			category: v1.SearchCategory_IMAGES,
			wrapper:  imageIndex.Wrapper{},
		},
		{
			bucket:   activeComponentDackBox.Bucket,
			reader:   activeComponentDackBox.Reader,
			category: v1.SearchCategory_ACTIVE_COMPONENT,
			wrapper:  activeComponentIndex.Wrapper{},
		},
		{
			bucket:   deploymentDackBox.Bucket,
			reader:   deploymentDackBox.Reader,
			category: v1.SearchCategory_DEPLOYMENTS,
			wrapper:  deploymentIndex.Wrapper{},
		},
		{
			bucket:   nodeComponentEdgeDackBox.Bucket,
			reader:   nodeComponentEdgeDackBox.Reader,
			category: v1.SearchCategory_NODE_COMPONENT_EDGE,
			wrapper:  nodeComponentEdgeIndex.Wrapper{},
		},
		{
			bucket:   nodeDackBox.Bucket,
			reader:   nodeDackBox.Reader,
			category: v1.SearchCategory_NODES,
			wrapper:  nodeIndex.Wrapper{},
		},
		{
			bucket:   imageCVEEdgeDackbox.Bucket,
			reader:   imageCVEEdgeDackbox.Reader,
			category: v1.SearchCategory_IMAGE_VULN_EDGE,
			wrapper:  imageCVEEdgeIndex.Wrapper{},
		},
	}
)

// Init runs all registered initialization functions.
func Init(dacky *dackbox.DackBox, indexQ queue.WaitableQueue, registry indexer.WrapperRegistry, reindexBucket, dirtyBucket, reindexValue []byte) error {
	synchronized := concurrency.NewSignal()

	for _, initialized := range initializedBuckets {
		// Register the wrapper to index the objects.
		registry.RegisterWrapper(initialized.bucket, initialized.wrapper)

		if err := queueBucketForIndexing(dacky, indexQ, initialized.category, dirtyBucket, initialized.bucket, initialized.reader); err != nil {
			return errors.Wrap(err, "unable to initialize dackbox, initialization function failed")
		}

		// Wait for the reindexing of the bucket to finish.
		synchronized.Reset()
		indexQ.PushSignal(&synchronized)
		synchronized.Wait()

		if err := markInitialIndexingComplete(globalindex.GetGlobalIndex(), []byte(initialized.category.String())); err != nil {
			return errors.Wrap(err, "setting initial indexing complete")
		}
	}
	return nil
}

func markInitialIndexingComplete(index bleve.Index, bucket []byte) error {
	return index.SetInternal(bucket, []byte("old"))
}

func needsInitialIndexing(index bleve.Index, bucket []byte) (bool, error) {
	data, err := index.GetInternal(bucket)
	if err != nil {
		return false, err
	}
	return !bytes.Equal([]byte("old"), data), nil
}

func queueBucketForIndexing(dacky *dackbox.DackBox, indexQ queue.WaitableQueue, category v1.SearchCategory, dirtyBucket, bucket []byte, reader crud.Reader) error {
	defer debug.FreeOSMemory()

	txn, err := dacky.NewReadOnlyTransaction()
	if err != nil {
		return err
	}
	defer txn.Discard()

	// Read all keys that need re-indexing.
	var keys [][]byte

	index := globalindex.GetGlobalIndex()
	needsReindex, err := needsInitialIndexing(index, []byte(category.String()))
	if err != nil {
		return err
	}

	if needsReindex {
		log.Infof("re-indexing all keys in bucket: %s", bucket)
		keys, err = reader.ReadKeysIn(bucket, txn)
		if err != nil {
			return err
		}
	} else {
		keys, err = reader.ReadKeysIn(dbhelper.GetBucketKey(dirtyBucket, bucket), txn)
		if err != nil {
			return err
		}
	}
	if len(keys) == 0 {
		log.Infof("no keys to reindex in bucket: %s", bucket)
	} else {
		log.Infof("indexing %d keys in bucket: %s", len(keys), bucket)
	}

	// Push them into the indexing queue.
	for _, key := range keys {
		if dbhelper.HasPrefix(dirtyBucket, key) {
			key = dbhelper.StripBucket(dirtyBucket, key)
		}
		msg, err := reader.ReadIn(key, txn)
		if err != nil {
			return err
		}
		indexQ.Push(key, msg)
	}

	return nil
}
