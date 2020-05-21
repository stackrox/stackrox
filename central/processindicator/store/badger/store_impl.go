package badger

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	generic "github.com/stackrox/rox/pkg/badgerhelper/crud"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()
)

var (
	processIndicatorBucket = []byte("process_indicators2")
)

func alloc() proto.Message {
	return &storage.ProcessIndicator{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.ProcessIndicator).GetId())
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *badger.DB) store.Store {
	globaldb.RegisterBucket(processIndicatorBucket, "ProcessIndicator")

	return &storeImpl{
		crud: generic.NewCRUD(db, processIndicatorBucket, keyFunc, alloc),
		DB:   db,
	}
}

type storeImpl struct {
	*badger.DB
	crud generic.Crud
}

// GetProcessIndicator returns indicator with given id.
func (b *storeImpl) Get(id string) (indicator *storage.ProcessIndicator, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "ProcessIndicator")
	msg, exists, err := b.crud.Read(id)
	if err != nil || !exists {
		return
	}
	indicator = msg.(*storage.ProcessIndicator)
	return
}

func (b *storeImpl) GetMany(ids []string) ([]*storage.ProcessIndicator, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Alert")

	msgs, missingIndices, err := b.crud.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	processes := make([]*storage.ProcessIndicator, 0, len(msgs))
	for _, m := range msgs {
		processes = append(processes, m.(*storage.ProcessIndicator))
	}
	return processes, missingIndices, nil
}

func (b *storeImpl) GetProcessIndicators() ([]*storage.ProcessIndicator, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "ProcessIndicator")

	msgs, err := b.crud.ReadAll()
	if err != nil {
		return nil, err
	}
	indicators := make([]*storage.ProcessIndicator, 0, len(msgs))
	for _, m := range msgs {
		indicators = append(indicators, m.(*storage.ProcessIndicator))
	}
	return indicators, nil
}

func (b *storeImpl) UpsertMany(indicators []*storage.ProcessIndicator) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.AddMany, "ProcessIndicator")

	msgs := make([]proto.Message, 0, len(indicators))
	for _, i := range indicators {
		msgs = append(msgs, i)
	}
	return b.crud.UpsertBatch(msgs)
}

func (b *storeImpl) DeleteMany(ids []string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicators")

	if err := b.crud.DeleteBatch(ids); err != nil {
		return errors.Wrap(err, "removing indicators")
	}
	return nil
}

func (b *storeImpl) AckKeysIndexed(keys ...string) error {
	return b.crud.AckKeysIndexed(keys...)
}

func (b *storeImpl) GetKeysToIndex() ([]string, error) {
	return b.crud.GetKeysToIndex()
}

func (b *storeImpl) Walk(fn func(pi *storage.ProcessIndicator) error) error {
	opts := badgerhelper.ForEachOptions{
		IteratorOptions: badgerhelper.DefaultIteratorOptions(),
	}
	return b.DB.View(func(tx *badger.Txn) error {
		return badgerhelper.BucketForEach(tx, processIndicatorBucket, opts, func(k, v []byte) error {
			var processIndicator storage.ProcessIndicator
			if err := proto.Unmarshal(v, &processIndicator); err != nil {
				return err
			}
			return fn(&processIndicator)
		})
	})
}
