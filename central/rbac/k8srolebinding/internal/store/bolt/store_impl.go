package bolt

import (
	"time"

	bbolt "github.com/etcd-io/bbolt"
	proto "github.com/gogo/protobuf/proto"
	metrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	storage "github.com/stackrox/rox/generated/storage"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	ops "github.com/stackrox/rox/pkg/metrics"
	storecache "github.com/stackrox/rox/pkg/storecache"
)

var (
	bucketName = []byte("rolebindings")
)

type storeImpl struct {
	crud protoCrud.MessageCrud
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.K8SRoleBinding).GetId())
}

func alloc() proto.Message {
	return new(storage.K8SRoleBinding)
}

// NewBoltStore returns a role binding store based on Bolt
func NewBoltStore(db *bbolt.DB, cache storecache.Cache) (store.Store, error) {
	newCrud, err := protoCrud.NewMessageCrud(db, bucketName, key, alloc)
	if err != nil {
		return nil, err
	}
	newCrud = protoCrud.NewCachedMessageCrud(newCrud, cache, "RoleBinding", metrics.IncrementDBCacheCounter)
	return &storeImpl{crud: newCrud}, nil
}

func (s *storeImpl) Delete(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "RoleBinding")
	_, _, err := s.crud.Delete(id)
	return err
}

func (s *storeImpl) Get(id string) (*storage.K8SRoleBinding, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "RoleBinding")
	msg, err := s.crud.Read(id)
	if err != nil {
		return nil, msg == nil, err
	}
	if msg == nil {
		return nil, false, nil
	}
	rolebinding := msg.(*storage.K8SRoleBinding)
	return rolebinding, true, nil
}

func (s *storeImpl) GetMany(ids []string) ([]*storage.K8SRoleBinding, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "RoleBinding")
	msgs, missingIndices, err := s.crud.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	storedKeys := make([]*storage.K8SRoleBinding, 0, len(msgs))
	for _, msg := range msgs {
		storedKeys = append(storedKeys, msg.(*storage.K8SRoleBinding))
	}
	return storedKeys, missingIndices, nil
}

func (s *storeImpl) Walk(fn func(binding *storage.K8SRoleBinding) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "RoleBinding")
	msgs, err := s.crud.ReadAll()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		if err := fn(msg.(*storage.K8SRoleBinding)); err != nil {
			return err
		}
	}
	return nil
}

func (s *storeImpl) Upsert(rolebinding *storage.K8SRoleBinding) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "RoleBinding")
	_, _, err := s.crud.Upsert(rolebinding)
	return err
}
