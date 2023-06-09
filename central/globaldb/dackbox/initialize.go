package dackbox

import (
	"bytes"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	activeComponentDackBox "github.com/stackrox/rox/central/activecomponent/dackbox"
	clusterCVEEdgeDackBox "github.com/stackrox/rox/central/clustercveedge/dackbox"
	componentCVEEdgeDackBox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	imagecomponentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	imagecomponentEdgeDackBox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imageCVEEdgeDackbox "github.com/stackrox/rox/central/imagecveedge/dackbox"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	nodeComponentEdgeDackBox "github.com/stackrox/rox/central/nodecomponentedge/dackbox"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	rawDackbox "github.com/stackrox/rox/pkg/dackbox/raw"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/env"
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
		},
		{
			bucket:   componentCVEEdgeDackBox.Bucket,
			reader:   componentCVEEdgeDackBox.Reader,
			category: v1.SearchCategory_COMPONENT_VULN_EDGE,
		},
		{
			bucket:   clusterCVEEdgeDackBox.Bucket,
			reader:   clusterCVEEdgeDackBox.Reader,
			category: v1.SearchCategory_CLUSTER_VULN_EDGE,
		},
		{
			bucket:   activeComponentDackBox.Bucket,
			reader:   activeComponentDackBox.Reader,
			category: v1.SearchCategory_ACTIVE_COMPONENT,
		},
	}
)

func init() {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		migratedBuckets := []bucketRef{
			{
				bucket:   imagecomponentEdgeDackBox.Bucket,
				reader:   imagecomponentEdgeDackBox.Reader,
				category: v1.SearchCategory_IMAGE_COMPONENT_EDGE,
			},
			{
				bucket:   imageDackBox.Bucket,
				reader:   imageDackBox.Reader,
				category: v1.SearchCategory_IMAGES,
			},
			{
				bucket:   imagecomponentDackBox.Bucket,
				reader:   imagecomponentDackBox.Reader,
				category: v1.SearchCategory_IMAGE_COMPONENTS,
			},
			{
				bucket:   deploymentDackBox.Bucket,
				reader:   deploymentDackBox.Reader,
				category: v1.SearchCategory_DEPLOYMENTS,
			},
			{
				bucket:   imageCVEEdgeDackbox.Bucket,
				reader:   imageCVEEdgeDackbox.Reader,
				category: v1.SearchCategory_IMAGE_VULN_EDGE,
			},
			{
				bucket:   nodeDackBox.Bucket,
				reader:   nodeDackBox.Reader,
				category: v1.SearchCategory_NODES,
			},
			{
				bucket:   nodeComponentEdgeDackBox.Bucket,
				reader:   nodeComponentEdgeDackBox.Reader,
				category: v1.SearchCategory_NODE_COMPONENT_EDGE,
			},
		}
		initializedBuckets = append(initializedBuckets, migratedBuckets...)
	}
}

// Init runs all registered initialization functions.
func Init(dacky *dackbox.DackBox, indexQ queue.WaitableQueue, dirtyBucket []byte) error {
	synchronized := concurrency.NewSignal()

	globalIndex := globalindex.GetGlobalIndex()
	for _, initialized := range initializedBuckets {
		// Register the wrapper to index the objects.
		rawDackbox.RegisterIndex(initialized.bucket, initialized.wrapper)

		if err := queueBucketForIndexing(dacky, globalIndex, indexQ, initialized.category, dirtyBucket, initialized.bucket, initialized.reader); err != nil {
			return errors.Wrap(err, "unable to initialize dackbox, initialization function failed")
		}

		// Wait for the reindexing of the bucket to finish.
		synchronized.Reset()
		indexQ.PushSignal(&synchronized)
		synchronized.Wait()

		if err := markInitialIndexingComplete(globalIndex, []byte(initialized.category.String())); err != nil {
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

func queueBucketForIndexing(dacky *dackbox.DackBox, index bleve.Index, indexQ queue.WaitableQueue, category v1.SearchCategory, dirtyBucket, bucket []byte, reader crud.Reader) error {
	defer debug.FreeOSMemory()

	txn, err := dacky.NewReadOnlyTransaction()
	if err != nil {
		return err
	}
	defer txn.Discard()

	// Read all keys that need re-indexing.
	var keys [][]byte

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
