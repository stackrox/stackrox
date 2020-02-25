package dackbox

import (
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
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
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/crud"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/debug"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

var (
	initializedBuckets = []struct {
		bucket   []byte
		reader   crud.Reader
		category v1.SearchCategory
		wrapper  indexer.Wrapper
	}{
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
			bucket:   deploymentDackBox.Bucket,
			reader:   deploymentDackBox.Reader,
			category: v1.SearchCategory_DEPLOYMENTS,
			wrapper:  deploymentIndex.Wrapper{},
		},
	}
)

// RemoveReindexBucket removes all of the keys that are handled, so that all handled buckets will be re-indexed on next
// restart.
func RemoveReindexBucket(db *badger.DB) error {
	return db.Update(func(txn *badger.Txn) error {
		for _, initialized := range initializedBuckets {
			if err := txn.Delete(badgerhelper.GetBucketKey(ReindexIfMissingBucket, initialized.bucket)); err != nil {
				return err
			}
		}
		return nil
	})
}

// Init runs all registered initialization functions.
func Init(dacky *dackbox.DackBox, indexQ queue.WaitableQueue, registry indexer.WrapperRegistry, reindexBucket, dirtyBucket, reindexValue []byte) error {
	synchronized := concurrency.NewSignal()
	for _, initialized := range initializedBuckets {
		// Register the wrapper to index the objects.
		registry.RegisterWrapper(initialized.bucket, initialized.wrapper)

		// Reindex objects that need reindexing.
		needsReindex, err := readNeedsReindex(dacky, reindexBucket, initialized.bucket)
		if err != nil {
			return errors.Wrap(err, "unable to read dackbox backup state")
		}

		if err := queueBucketForIndexing(dacky, indexQ, needsReindex, initialized.category, dirtyBucket, initialized.bucket, initialized.reader); err != nil {
			return errors.Wrap(err, "unable to initialize dackbox, initialization function failed")
		}

		// Wait for the reindexing of the bucket to finish.
		synchronized.Reset()
		indexQ.PushSignal(&synchronized)
		synchronized.Wait()

		if err := setDoesNotNeedReindex(dacky, reindexBucket, initialized.bucket, reindexValue); err != nil {
			return errors.Wrap(err, "unable to set dackbox backup state")
		}
	}
	return nil
}

func readNeedsReindex(dacky *dackbox.DackBox, reindexBucket, bucket []byte) (bool, error) {
	txn := dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	// If the key is missing from the reindexing bucket, it means we need to do a full re-index.
	_, err := txn.BadgerTxn().Get(badgerhelper.GetBucketKey(reindexBucket, bucket))
	if err == badger.ErrKeyNotFound {
		return true, nil
	} else if err != nil {
		return true, err
	}
	return false, nil
}

func queueBucketForIndexing(dacky *dackbox.DackBox, indexQ queue.WaitableQueue, needsReindex bool, category v1.SearchCategory, dirtyBucket, bucket []byte, reader crud.Reader) error {
	defer debug.FreeOSMemory()

	txn := dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	// Read all keys that need re-indexing.
	var keys [][]byte
	var err error
	if needsReindex {
		err = blevesearch.ResetIndex(category, globalindex.GetGlobalIndex())
		if err != nil {
			return err
		}
		keys, err = reader.ReadKeysIn(bucket, txn)
		if err != nil {
			return err
		}
	} else {
		keys, err = reader.ReadKeysIn(badgerhelper.GetBucketKey(dirtyBucket, bucket), txn)
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
		msg, err := reader.ReadIn(key, txn)
		if err != nil {
			return err
		}
		indexQ.Push(key, msg)
	}

	return nil
}

func setDoesNotNeedReindex(dacky *dackbox.DackBox, reindexBucket, bucket, reIndexValue []byte) error {
	txn := dacky.NewTransaction()
	defer txn.Discard()

	return txn.BadgerTxn().Set(badgerhelper.GetBucketKey(reindexBucket, bucket), reIndexValue)
}
