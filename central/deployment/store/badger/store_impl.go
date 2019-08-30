package badger

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	generic "github.com/stackrox/rox/pkg/badgerhelper/crud"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	deploymentBucket     = []byte("deployments")
	deploymentListBucket = []byte("deployments_list")
)

var (
	log = logging.LoggerForModule()
)

func alloc() proto.Message {
	return &storage.Deployment{}
}

func listAlloc() proto.Message {
	return &storage.ListDeployment{}
}

func deploymentConverter(msg proto.Message) proto.Message {
	return convertDeploymentToDeploymentList(msg.(*storage.Deployment))
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.Deployment).GetId())
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *badger.DB) (store.Store, error) {
	globaldb.RegisterBucket(deploymentBucket, "Deployment")
	globaldb.RegisterBucket(deploymentListBucket, "Deployment")
	return &storeImpl{
		deploymentCRUD: generic.NewCRUDWithPartial(db, deploymentBucket, keyFunc, alloc, deploymentListBucket, listAlloc, deploymentConverter),
	}, nil
}

type storeImpl struct {
	deploymentCRUD generic.Crud
}

func (b *storeImpl) msgsToListDeployments(msgs []proto.Message) []*storage.ListDeployment {
	deployments := make([]*storage.ListDeployment, 0, len(msgs))
	for _, m := range msgs {
		d := m.(*storage.ListDeployment)
		deployments = append(deployments, d)
	}
	return deployments
}

func (b *storeImpl) msgsToDeployments(msgs []proto.Message) []*storage.Deployment {
	deployments := make([]*storage.Deployment, 0, len(msgs))
	for _, m := range msgs {
		d := m.(*storage.Deployment)
		deployments = append(deployments, d)
	}
	return deployments
}

// GetListDeployment returns a list deployment with given id.
func (b *storeImpl) ListDeployment(id string) (deployment *storage.ListDeployment, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "ListDeployment")

	var msg proto.Message
	msg, exists, err = b.deploymentCRUD.ReadPartial(id)
	if err != nil || !exists {
		return
	}
	deployment = msg.(*storage.ListDeployment)
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *storeImpl) ListDeployments() ([]*storage.ListDeployment, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "ListDeployment")

	msgs, err := b.deploymentCRUD.ReadAllPartial()
	if err != nil {
		return nil, err
	}
	return b.msgsToListDeployments(msgs), nil
}

func (b *storeImpl) ListDeploymentsWithIDs(ids ...string) ([]*storage.ListDeployment, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "ListDeployment")

	msgs, indices, err := b.deploymentCRUD.ReadBatchPartial(ids)
	if err != nil {
		return nil, nil, err
	}
	return b.msgsToListDeployments(msgs), indices, nil
}

func convertDeploymentToDeploymentList(d *storage.Deployment) *storage.ListDeployment {
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

// GetDeployment returns deployment with given id.
func (b *storeImpl) GetDeployment(id string) (deployment *storage.Deployment, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "Deployment")

	var msg proto.Message
	msg, exists, err = b.deploymentCRUD.Read(id)
	if err != nil || !exists {
		return
	}
	deployment = msg.(*storage.Deployment)
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *storeImpl) GetDeployments() ([]*storage.Deployment, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Deployment")

	msgs, err := b.deploymentCRUD.ReadAll()
	if err != nil {
		return nil, err
	}
	return b.msgsToDeployments(msgs), nil
}

func (b *storeImpl) GetDeploymentsWithIDs(ids ...string) ([]*storage.Deployment, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Deployment")

	msgs, indices, err := b.deploymentCRUD.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	return b.msgsToDeployments(msgs), indices, nil
}

// CountDeployments returns the number of deployments.
func (b *storeImpl) CountDeployments() (int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Count, "Deployment")
	return b.deploymentCRUD.Count()
}

// UpsertDeployment adds a deployment to bolt, or updates it if it exists already.
func (b *storeImpl) UpsertDeployment(deployment *storage.Deployment) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Add, "Deployment")

	return b.deploymentCRUD.Upsert(deployment)
}

// UpdateDeployment updates a deployment to bolt
func (b *storeImpl) UpdateDeployment(deployment *storage.Deployment) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Update, "Deployment")

	return b.deploymentCRUD.Update(deployment)
}

// RemoveDeployment removes a deployment
func (b *storeImpl) RemoveDeployment(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "Deployment")

	return b.deploymentCRUD.Delete(id)
}

func (b *storeImpl) GetTxnCount() (txNum uint64, err error) {
	return b.deploymentCRUD.GetTxnCount(), nil
}

func (b *storeImpl) IncTxnCount() error {
	return b.deploymentCRUD.IncTxnCount()
}
