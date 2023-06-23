package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	acDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/activecomponent"
	clusterDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/cluster"
	deploymentDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/deployment"
	imageDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/image"
	namespaceDackBox "github.com/stackrox/rox/migrator/migrations/dackboxhelpers/namespace"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stackrox/rox/pkg/dackbox"
)

// StoreImpl provides an implementation of the Store interface using dackbox.
type StoreImpl struct {
	dacky    *dackbox.DackBox
	keyFence concurrency.KeyFence
}

// New returns a new instance of a deployment store using dackbox.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence) *StoreImpl {
	return &StoreImpl{
		dacky:    dacky,
		keyFence: keyFence,
	}
}

// Count returns the number of deployments in dackbox.
func (b *StoreImpl) Count(_ context.Context) (int, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return 0, err
	}
	defer txn.Discard()

	count, err := deploymentDackBox.Reader.CountIn(deploymentDackBox.Bucket, txn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetIDs returns the keys of all deployments stored in RocksDB.
func (b *StoreImpl) GetIDs(_ context.Context) ([]string, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer txn.Discard()

	var ids []string
	err = txn.BucketKeyForEach(deploymentDackBox.Bucket, true, func(k []byte) error {
		ids = append(ids, string(k))
		return nil
	})
	return ids, err
}

// GetListDeployment returns ListDeployment with given id.
func (b *StoreImpl) GetListDeployment(_ context.Context, id string) (deployment *storage.ListDeployment, exists bool, err error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer txn.Discard()

	msg, err := deploymentDackBox.ListReader.ReadIn(deploymentDackBox.ListBucketHandler.GetKey(id), txn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ListDeployment), true, nil
}

// GetManyListDeployments returns list deployments with the given ids.
func (b *StoreImpl) GetManyListDeployments(_ context.Context, ids ...string) ([]*storage.ListDeployment, []int, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
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

// Get returns deployment with given id.
func (b *StoreImpl) Get(_ context.Context, id string) (deployment *storage.Deployment, exists bool, err error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer txn.Discard()

	msg, err := deploymentDackBox.Reader.ReadIn(deploymentDackBox.BucketHandler.GetKey(id), txn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.Deployment), true, err
}

// GetMany returns deployments with the given ids.
func (b *StoreImpl) GetMany(_ context.Context, ids []string) ([]*storage.Deployment, []int, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
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

// Upsert updates a deployment to dackbox.
func (b *StoreImpl) Upsert(_ context.Context, deployment *storage.Deployment) error {
	var imageKeys [][]byte
	for _, container := range deployment.GetContainers() {
		imageKeys = append(imageKeys, imageDackBox.BucketHandler.GetKey(container.GetImage().GetId()))
	}
	deploymentKey := deploymentDackBox.KeyFunc(deployment)
	namespaceKey := namespaceDackBox.BucketHandler.GetKey(deployment.GetNamespaceId())
	clusterKey := clusterDackBox.BucketHandler.GetKey(deployment.GetClusterId())

	keysToLock := concurrency.DiscreteKeySet(append(imageKeys,
		deploymentKey,
		namespaceKey,
		clusterKey,
	)...)

	return b.keyFence.DoStatusWithLock(keysToLock, func() error {
		txn, err := b.dacky.NewTransaction()
		if err != nil {
			return err
		}
		defer txn.Discard()

		g := txn.Graph()
		acKeys := g.GetRefsFromPrefix(deploymentKey, acDackBox.Bucket)
		// Clear cluster pointing to the namespace before setting the new one.
		// This is to handle situations where a new cluster bundle is generated for an existing cluster, as the cluster
		// ID will change, the the IDs for child objects will remain the same.
		g.DeleteRefsTo(namespaceKey)
		g.AddRefs(clusterKey, namespaceKey)
		g.AddRefs(namespaceKey, deploymentKey)
		// Merge image keys and active component keys
		g.SetRefs(deploymentKey, append(acKeys, imageKeys...))

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

// Delete deletes an deployment and it's list object counter-part.
func (b *StoreImpl) Delete(_ context.Context, id string) error {
	namespaceKey, allKeys := b.collectDeploymentKeys(id)
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(allKeys...), func() error {
		txn, err := b.dacky.NewTransaction()
		if err != nil {
			return err
		}
		defer txn.Discard()

		g := txn.Graph()

		acKeys := g.GetRefsFromPrefix(deploymentDackBox.BucketHandler.GetKey(id), acDackBox.Bucket)
		for _, key := range acKeys {
			if err := acDackBox.Deleter.DeleteIn(key, txn); err != nil {
				return err
			}
		}
		err = deploymentDackBox.Deleter.DeleteIn(deploymentDackBox.BucketHandler.GetKey(id), txn)
		if err != nil {
			return err
		}
		err = deploymentDackBox.ListDeleter.DeleteIn(deploymentDackBox.ListBucketHandler.GetKey(id), txn)
		if err != nil {
			return err
		}

		// If the namespace has no more deployments, remove refs in both directions.
		if namespaceKey != nil && deploymentDackBox.BucketHandler.CountFilteredRefsFrom(g, namespaceKey) == 0 {
			g.DeleteRefsFrom(namespaceKey)
			// This deletes all references from cluster to this namespace.
			g.DeleteRefsTo(namespaceKey)
		}

		return txn.Commit()
	})
}

func (b *StoreImpl) collectDeploymentKeys(id string) ([]byte, [][]byte) {
	graphView := b.dacky.NewGraphView()
	defer graphView.Discard()

	deploymentKey := deploymentDackBox.BucketHandler.GetKey(id)
	deploymentKeys := sortedkeys.SortedKeys{deploymentKey}

	imageKeys := imageDackBox.BucketHandler.GetFilteredRefsFrom(graphView, deploymentKey)

	namespaceKeys := namespaceDackBox.BucketHandler.GetFilteredRefsTo(graphView, deploymentKey)

	// Deployment should have a single namespace link up. If not, early exit.
	if len(namespaceKeys) != 1 {
		return nil, sortedkeys.DisjointPrefixUnion(deploymentKeys, imageKeys, namespaceKeys)
	}
	namespaceKey := namespaceKeys[0]

	clusterKeys := clusterDackBox.BucketHandler.GetFilteredRefsTo(graphView, namespaceKey)

	return namespaceKey, sortedkeys.DisjointPrefixUnion(deploymentKeys, imageKeys, namespaceKeys, clusterKeys)
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

// UpsertMany batches objects into the DB
func (b *StoreImpl) UpsertMany(ctx context.Context, objs []*storage.Deployment) error {
	for _, obj := range objs {
		if err := b.Upsert(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}
