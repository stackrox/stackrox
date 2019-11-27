package badger

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/alert/convert"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	generic "github.com/stackrox/rox/pkg/badgerhelper/crud"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	log = logging.LoggerForModule()

	alertBucket     = []byte("alerts")
	alertListBucket = []byte("alerts_list")
)

type storeImpl struct {
	db *badger.DB

	alertCRUD generic.Crud
}

func alloc() proto.Message {
	return &storage.Alert{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.Alert).GetId())
}

func listAlloc() proto.Message {
	return &storage.ListAlert{}
}

func alertConverter(msg proto.Message) proto.Message {
	return convert.AlertToListAlert(msg.(*storage.Alert))
}

// New returns a new Store instance using the provided badger DB instance.
func New(db *badger.DB) store.Store {
	globaldb.RegisterBucket(alertBucket, "Alert")
	globaldb.RegisterBucket(alertListBucket, "Alert")
	return &storeImpl{
		db:        db,
		alertCRUD: generic.NewCRUDWithPartial(db, alertBucket, keyFunc, alloc, alertListBucket, listAlloc, alertConverter),
	}
}

// GetAlert returns an alert with given id.
func (b *storeImpl) ListAlert(id string) (alert *storage.ListAlert, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "ListAlert")

	msg, exists, err := b.alertCRUD.ReadPartial(id)
	if err != nil || !exists {
		return
	}
	alert = msg.(*storage.ListAlert)
	return
}

// ListAlerts returns a minimal form of the Alert struct for faster marshalling
func (b *storeImpl) ListAlerts() ([]*storage.ListAlert, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "ListAlert")

	msgs, err := b.alertCRUD.ReadAllPartial()
	if err != nil {
		return nil, err
	}
	alerts := make([]*storage.ListAlert, 0, len(msgs))
	for _, m := range msgs {
		alerts = append(alerts, m.(*storage.ListAlert))
	}
	return alerts, nil
}

// GetAlert returns an alert with given id.
func (b *storeImpl) GetAlert(id string) (alert *storage.Alert, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "Alert")

	msg, exists, err := b.alertCRUD.Read(id)
	if err != nil || !exists {
		return
	}
	alert = msg.(*storage.Alert)
	return
}

func (b *storeImpl) GetAlertIDs() ([]string, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "AlertIDs")

	var keys []string
	err := b.db.View(func(tx *badger.Txn) error {
		return badgerhelper.BucketKeyForEach(tx, alertListBucket, badgerhelper.ForEachOptions{StripKeyPrefix: true}, func(k []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	return keys, err
}

func (b *storeImpl) GetListAlerts(ids []string) ([]*storage.ListAlert, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "ListAlert")

	msgs, indices, err := b.alertCRUD.ReadBatchPartial(ids)
	if err != nil {
		return nil, nil, err
	}
	alerts := make([]*storage.ListAlert, 0, len(msgs))
	for _, m := range msgs {
		alerts = append(alerts, m.(*storage.ListAlert))
	}
	return alerts, indices, nil
}

func (b *storeImpl) GetAlerts(ids []string) ([]*storage.Alert, []int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Alert")

	msgs, missingIndices, err := b.alertCRUD.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	alerts := make([]*storage.Alert, 0, len(msgs))
	for _, m := range msgs {
		alerts = append(alerts, m.(*storage.Alert))
	}
	return alerts, missingIndices, nil
}

// AddAlert adds an alert into Badger
func (b *storeImpl) UpsertAlert(alert *storage.Alert) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Add, "Alert")
	return b.alertCRUD.Upsert(alert)
}

func (b *storeImpl) GetTxnCount() (txNum uint64, err error) {
	return b.alertCRUD.GetTxnCount(), nil
}

func (b *storeImpl) IncTxnCount() error {
	return b.alertCRUD.IncTxnCount()
}

// DeleteAlert removes an alert
func (b *storeImpl) DeleteAlert(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "Alert")
	return b.alertCRUD.Delete(id)
}

func (b *storeImpl) DeleteAlerts(ids ...string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.RemoveMany, "Alert")
	return b.alertCRUD.DeleteBatch(ids)
}

func (b *storeImpl) WalkAll(fn func(*storage.ListAlert) error) error {
	opts := badgerhelper.ForEachOptions{
		IteratorOptions: badgerhelper.DefaultIteratorOptions(),
	}
	return b.db.View(func(tx *badger.Txn) error {
		return badgerhelper.BucketForEach(tx, alertListBucket, opts, func(k, v []byte) error {
			var alert storage.ListAlert
			if err := proto.Unmarshal(v, &alert); err != nil {
				return err
			}
			return fn(&alert)
		})
	})
}
