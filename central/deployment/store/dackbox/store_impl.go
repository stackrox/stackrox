package dackbox

import (
	"time"

	"github.com/gogo/protobuf/proto"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	deploymentDackBox "github.com/stackrox/rox/central/deployment/dackbox"
	imageDackBox "github.com/stackrox/rox/central/image/dackbox"
	"github.com/stackrox/rox/central/metrics"
	namespaceDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	ops "github.com/stackrox/rox/pkg/metrics"
)

// StoreImpl provides an implementation of the Store interface using dackbox.
type StoreImpl struct {
	dacky    *dackbox.DackBox
	keyFence concurrency.KeyFence
}

// New returns a new instance of a deployment store using dackbox.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence) (*StoreImpl, error) {
	return &StoreImpl{
		dacky:    dacky,
		keyFence: keyFence,
	}, nil
}

// CountDeployments returns the number of deployments in badger.
func (b *StoreImpl) CountDeployments() (int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Count, "Deployment")

	txn := b.dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	count, err := deploymentDackBox.Reader.CountIn(deploymentDackBox.Bucket, txn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetDeploymentIDs returns the keys of all deployments stored in badger.
func (b *StoreImpl) GetDeploymentIDs() ([]string, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "Deployment")

	txn := b.dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	var ids []string
	err := badgerhelper.BucketKeyForEach(txn.BadgerTxn(), deploymentDackBox.Bucket, badgerhelper.ForEachOptions{StripKeyPrefix: true}, func(k []byte) error {
		ids = append(ids, string(k))
		return nil
	})
	return ids, err
}

// ListDeployment returns ListDeployment with given id.
func (b *StoreImpl) ListDeployment(id string) (deployment *storage.ListDeployment, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "ListDeployment")

	txn := b.dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	msg, err := deploymentDackBox.ListReader.ReadIn(deploymentDackBox.ListBucketHandler.GetKey(id), txn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ListDeployment), true, nil
}

// ListDeploymentsWithIDs returns list deployments with the given ids.
func (b *StoreImpl) ListDeploymentsWithIDs(ids ...string) ([]*storage.ListDeployment, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Deployment")

	txn := b.dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	var msgs []proto.Message
	var missing []int
	for _, id := range ids {
		msg, err := deploymentDackBox.ListReader.ReadIn(deploymentDackBox.ListBucketHandler.GetKey(id), txn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			msgs = append(msgs, msg)
		}
	}

	ret := make([]*storage.ListDeployment, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ListDeployment))
	}

	return ret, missing, nil
}

// ListDeployments returns all list deployments regardless of request
func (b *StoreImpl) ListDeployments() ([]*storage.ListDeployment, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "Deployment")

	txn := b.dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	msgs, err := deploymentDackBox.ListReader.ReadAllIn(deploymentDackBox.ListBucket, txn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.ListDeployment, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ListDeployment))
	}

	return ret, nil
}

// GetDeployments returns all deployments regardless of request.
func (b *StoreImpl) GetDeployments() ([]*storage.Deployment, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "Deployment")

	txn := b.dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	msgs, err := deploymentDackBox.Reader.ReadAllIn(deploymentDackBox.Bucket, txn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.Deployment, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.Deployment))
	}

	return ret, nil
}

// GetDeployment returns deployment with given id.
func (b *StoreImpl) GetDeployment(id string) (deployment *storage.Deployment, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "Deployment")

	txn := b.dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	msg, err := deploymentDackBox.Reader.ReadIn(deploymentDackBox.BucketHandler.GetKey(id), txn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.Deployment), true, err
}

// GetDeploymentsWithIDs returns deployments with the given ids.
func (b *StoreImpl) GetDeploymentsWithIDs(ids ...string) ([]*storage.Deployment, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Deployment")

	txn := b.dacky.NewReadOnlyTransaction()
	defer txn.Discard()

	var msgs []proto.Message
	var missing []int
	for _, id := range ids {
		msg, err := deploymentDackBox.Reader.ReadIn(deploymentDackBox.BucketHandler.GetKey(id), txn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			msgs = append(msgs, msg)
		}
	}

	ret := make([]*storage.Deployment, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.Deployment))
	}

	return ret, missing, nil
}

// UpsertDeployment updates a deployment to badger.
func (b *StoreImpl) UpsertDeployment(deployment *storage.Deployment) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Upsert, "Deployment")

	var imageKeys [][]byte
	for _, container := range deployment.GetContainers() {
		imageKeys = append(imageKeys, imageDackBox.BucketHandler.GetKey(container.GetImage().GetId()))
	}
	deploymentKey := deploymentDackBox.KeyFunc(deployment)
	namespaceSACKey := namespaceDackBox.SACBucketHandler.GetKey(deployment.GetNamespace())
	namespaceKey := namespaceDackBox.BucketHandler.GetKey(deployment.GetNamespaceId())
	clusterKey := clusterDackBox.BucketHandler.GetKey(deployment.GetClusterId())

	keysToLock := concurrency.DiscreteKeySet(append(imageKeys,
		deploymentKey,
		namespaceKey,
		namespaceSACKey,
		clusterKey,
	)...)

	return b.keyFence.DoStatusWithLock(keysToLock, func() error {
		txn := b.dacky.NewTransaction()
		defer txn.Discard()

		err := txn.Graph().AddRefs(clusterKey, namespaceKey)
		if err != nil {
			return err
		}
		err = txn.Graph().AddRefs(clusterKey, namespaceSACKey)
		if err != nil {
			return err
		}
		err = txn.Graph().AddRefs(namespaceKey, deploymentKey)
		if err != nil {
			return err
		}
		err = txn.Graph().AddRefs(namespaceSACKey, deploymentKey)
		if err != nil {
			return err
		}
		err = txn.Graph().SetRefs(deploymentKey, imageKeys)
		if err != nil {
			return err
		}

		err = deploymentDackBox.Upserter.UpsertIn(nil, deployment, txn)
		if err != nil {
			return err
		}
		err = deploymentDackBox.ListUpserter.UpsertIn(nil, convertDeploymentToListDeployment(deployment), txn)
		if err != nil {
			return err
		}

		return txn.Commit()
	})
}

// RemoveDeployment deletes an deployment and it's list object counter-part.
func (b *StoreImpl) RemoveDeployment(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "Deployment")

	allKeys := b.collectDeploymentKeys(id)
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(allKeys...), func() error {
		txn := b.dacky.NewTransaction()
		defer txn.Discard()

		err := deploymentDackBox.Deleter.DeleteIn(deploymentDackBox.BucketHandler.GetKey(id), txn)
		if err != nil {
			return err
		}
		err = deploymentDackBox.ListDeleter.DeleteIn(deploymentDackBox.ListBucketHandler.GetKey(id), txn)
		if err != nil {
			return err
		}

		return txn.Commit()
	})
}

func (b *StoreImpl) collectDeploymentKeys(id string) [][]byte {
	graphView := b.dacky.NewGraphView()
	defer graphView.Discard()

	deploymentKey := deploymentDackBox.BucketHandler.GetKey(id)
	allKeys := sortedkeys.SortedKeys{deploymentKey}

	imageKeys := imageDackBox.BucketHandler.FilterKeys(graphView.GetRefsFrom(deploymentKey))
	allKeys = allKeys.Union(imageKeys)

	namespaceKeys := namespaceDackBox.BucketHandler.FilterKeys(graphView.GetRefsTo(deploymentKey))
	allKeys = allKeys.Union(namespaceKeys)

	namespaceSACKeys := namespaceDackBox.SACBucketHandler.FilterKeys(graphView.GetRefsTo(deploymentKey))
	allKeys = allKeys.Union(namespaceSACKeys)

	clusterKeys := sortedkeys.SortedKeys{}
	for _, namespaceKey := range namespaceKeys {
		clusterKeys = clusterKeys.Union(clusterDackBox.BucketHandler.FilterKeys(graphView.GetRefsTo(namespaceKey)))
	}
	allKeys = allKeys.Union(clusterKeys)

	return allKeys
}

func convertDeploymentToListDeployment(d *storage.Deployment) *storage.ListDeployment {
	return &storage.ListDeployment{
		Id:        d.GetId(),
		Hash:      d.GetHash(),
		Name:      d.GetName(),
		Cluster:   d.GetClusterName(),
		ClusterId: d.GetClusterId(),
		Namespace: d.GetNamespace(),
		Created:   d.GetCreated(),
		Priority:  d.GetPriority(),
	}
}

// AckKeysIndexed is a stub for the store interface
func (b *StoreImpl) AckKeysIndexed(keys ...string) error {
	return nil
}

// GetKeysToIndex is a stub for the store interface
func (b *StoreImpl) GetKeysToIndex() ([]string, error) {
	return nil, nil
}
