package badger

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/pod/store"
	"github.com/stackrox/rox/generated/storage"
	generic "github.com/stackrox/rox/pkg/badgerhelper/crud"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
)

const (
	objType = "Pod"
)

var (
	log = logging.LoggerForModule()

	podBucket = []byte("pods")
)

func alloc() proto.Message {
	return &storage.Pod{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.Pod).GetId())
}

// New returns a new Store instance using the provided badger DB instance.
func New(db *badger.DB) store.Store {
	globaldb.RegisterBucket(podBucket, objType)
	return &storeImpl{
		podCRUD: generic.NewCRUD(db, podBucket, keyFunc, alloc),
	}
}

type storeImpl struct {
	podCRUD generic.Crud
}

func (b *storeImpl) msgsToPods(msgs []proto.Message) []*storage.Pod {
	pods := make([]*storage.Pod, 0, len(msgs))
	for _, m := range msgs {
		p := m.(*storage.Pod)
		pods = append(pods, p)
	}
	return pods
}

// Get returns pod with given id.
func (b *storeImpl) Get(id string) (pod *storage.Pod, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, objType)

	var msg proto.Message
	msg, exists, err = b.podCRUD.Read(id)
	if err != nil || !exists {
		return
	}
	pod = msg.(*storage.Pod)
	return
}

func (b *storeImpl) GetMany(ids []string) ([]*storage.Pod, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, objType)

	msgs, indices, err := b.podCRUD.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	return b.msgsToPods(msgs), indices, nil
}

// Upsert adds a pod to badger, or updates it if it exists already.
func (b *storeImpl) Upsert(pod *storage.Pod) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Upsert, objType)
	return b.podCRUD.Upsert(pod)
}

// Delete removes a pod
func (b *storeImpl) Delete(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, objType)
	return b.podCRUD.Delete(id)
}

// AckKeysIndexed acknowledges the indexed keys.
func (b *storeImpl) AckKeysIndexed(keys ...string) error {
	return b.podCRUD.AckKeysIndexed(keys...)
}

// GetKeysToIndex returns teh keys that need to be indexed.
func (b *storeImpl) GetKeysToIndex() ([]string, error) {
	return b.podCRUD.GetKeysToIndex()
}

// GetPodIDs returns the ID for each Pod in the store.
func (b *storeImpl) GetKeys() ([]string, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, objType+"ID")
	return b.podCRUD.GetKeys()
}
