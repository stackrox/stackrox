package badger

import (
	"fmt"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	generic "github.com/stackrox/rox/pkg/badgerhelper/crud"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

var (
	processIndicatorBucket = []byte("process_indicators")
	uniqueProcessesBucket  = []byte("process_indicators_unique")
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
	globaldb.RegisterBucket(uniqueProcessesBucket, "ProcessIndicator")

	wrapper, err := badgerhelper.NewTxnHelper(db, processIndicatorBucket)
	utils.Must(err)
	return &storeImpl{
		TxnHelper: wrapper,
		DB:        db,
		crud:      generic.NewCRUD(db, processIndicatorBucket, keyFunc, alloc),
	}
}

type storeImpl struct {
	*badgerhelper.TxnHelper
	*badger.DB
	crud generic.Crud
}

func getProcessIndicator(tx *badger.Txn, id string) (indicator *storage.ProcessIndicator, exists bool, err error) {
	key := badgerhelper.GetBucketKey(processIndicatorBucket, []byte(id))

	item, err := tx.Get(key)
	if err != nil && err != badger.ErrKeyNotFound {
		return
	}
	exists = err != badger.ErrKeyNotFound
	if !exists {
		err = nil
		return
	}

	indicator = new(storage.ProcessIndicator)
	err = item.Value(func(v []byte) error {
		return proto.Unmarshal(v, indicator)
	})
	if err != nil {
		return
	}

	return
}

// GetProcessIndicator returns indicator with given id.
func (b *storeImpl) GetProcessIndicator(id string) (indicator *storage.ProcessIndicator, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "ProcessIndicator")
	msg, exists, err := b.crud.Read(id)
	if err != nil || !exists {
		return
	}
	indicator = msg.(*storage.ProcessIndicator)
	return
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

func (b *storeImpl) GetProcessInfoToArgs() (map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetGrouped, "ProcessIndicator")
	processNamesToArgs := make(map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs)

	forEachOpts := badgerhelper.ForEachOptions{
		StripKeyPrefix: true,
		IteratorOptions: &badger.IteratorOptions{
			PrefetchValues: true,
			PrefetchSize:   1000,
		},
	}
	err := b.View(func(tx *badger.Txn) error {
		// func ForEachWithPrefix(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k, v []byte) error) error {
		return badgerhelper.BucketForEach(tx, uniqueProcessesBucket, forEachOpts, func(k, v []byte) error {
			uniqueKey := new(storage.ProcessIndicatorUniqueKey)
			if err := proto.Unmarshal(k, uniqueKey); err != nil {
				return errors.Wrap(err, "key unmarshaling")
			}
			info := processindicator.ProcessWithContainerInfo{
				ContainerName: uniqueKey.GetContainerName(),
				PodID:         uniqueKey.GetPodId(),
				ProcessName:   uniqueKey.GetProcessName(),
			}
			processNamesToArgs[info] = append(processNamesToArgs[info], processindicator.IDAndArgs{
				ID:   string(v),
				Args: uniqueKey.GetProcessArgs(),
			})
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return processNamesToArgs, nil
}

// get the value of the secondary key
func getSecondaryKey(indicator *storage.ProcessIndicator) ([]byte, error) {
	uniqueKey := &storage.ProcessIndicatorUniqueKey{
		PodId:               indicator.GetPodId(),
		ContainerName:       indicator.GetContainerName(),
		ProcessExecFilePath: indicator.GetSignal().GetExecFilePath(),
		ProcessName:         indicator.GetSignal().GetName(),
		ProcessArgs:         indicator.GetSignal().GetArgs(),
	}
	return proto.Marshal(uniqueKey)
}

func (b *storeImpl) addProcessIndicator(tx *badger.Txn, indicator *storage.ProcessIndicator, data []byte) (string, error) {
	indicatorID := []byte(indicator.GetId())
	fullyQualifiedIndicatorID := badgerhelper.GetBucketKey(processIndicatorBucket, indicatorID)

	_, err := tx.Get(fullyQualifiedIndicatorID)
	if err != nil && err != badger.ErrKeyNotFound {
		return "", err
	}
	if err == nil {
		return "", fmt.Errorf("indicator with id %q already exists", indicator.GetId())
	}

	secondaryKey, err := getSecondaryKey(indicator)
	if err != nil {
		return "", err
	}

	uniqueKey := []byte(secondaryKey)
	fullyQualifiedUniqueKey := badgerhelper.GetBucketKey(uniqueProcessesBucket, uniqueKey)

	uniqueKeyItem, err := tx.Get(fullyQualifiedUniqueKey)
	if err != nil && err != badger.ErrKeyNotFound {
		return "", err
	}

	var oldIDString string
	if uniqueKeyItem != nil {
		oldIDBytes, err := uniqueKeyItem.ValueCopy(nil)
		if err != nil {
			return "", err
		}
		oldIDString = string(oldIDBytes)
		if err := removeProcessIndicator(tx, oldIDString); err != nil {
			return "", errors.Wrap(err, "Removing old indicator")
		}
	}

	if err := tx.Set(fullyQualifiedIndicatorID, data); err != nil {
		return "", errors.Wrap(err, "inserting into indicator bucket")
	}
	if err := tx.Set(fullyQualifiedUniqueKey, indicatorID); err != nil {
		return "", errors.Wrap(err, "inserting into unique field bucket")
	}
	return oldIDString, nil
}

// AddProcessIndicator returns the id of the indicator that was deduped or empty if none was deduped
func (b *storeImpl) AddProcessIndicator(indicator *storage.ProcessIndicator) (string, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Add, "ProcessIndicator")
	if indicator.GetId() == "" {
		return "", errors.New("received malformed indicator with no id")
	}
	indicatorBytes, err := proto.Marshal(indicator)
	if err != nil {
		return "", errors.Wrap(err, "process indicator proto marshalling")
	}

	var oldIDString string
	err = b.DB.Update(func(tx *badger.Txn) error {
		oldIDString, err = b.addProcessIndicator(tx, indicator, indicatorBytes)
		if err != nil {
			return err
		}
		return nil
	})
	if oldIDString != "" {
		if err := b.IncTxnCount(); err != nil {
			return oldIDString, err
		}
	}
	if err != nil {
		return "", err
	}
	return oldIDString, b.IncTxnCount()
}

func (b *storeImpl) AddProcessIndicators(indicators ...*storage.ProcessIndicator) ([]string, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.AddMany, "ProcessIndicator")
	var deletedIndicators []string
	dataBytes := make([][]byte, 0, len(indicators))
	for _, i := range indicators {
		data, err := proto.Marshal(i)
		if err != nil {
			return nil, err
		}
		dataBytes = append(dataBytes, data)
	}

	batch := batcher.New(len(indicators), 1000)

	for start, end, valid := batch.Next(); valid; start, end, valid = batch.Next() {
		err := b.DB.Update(func(tx *badger.Txn) error {

			for i := start; i < end; i++ {
				indicator := indicators[i]
				data := dataBytes[i]
				oldID, err := b.addProcessIndicator(tx, indicator, data)
				if err != nil {
					return err
				}
				if oldID != "" {
					deletedIndicators = append(deletedIndicators, oldID)
				}
			}
			return nil
		})
		if err != nil {
			return deletedIndicators, err
		}
	}
	if len(deletedIndicators) > 0 {
		// Purposefully increment the txn count here as it will be a call in the datastore
		if err := b.IncTxnCount(); err != nil {
			log.Errorf("error incrementing txn count: %v", err)
		}
	}
	return deletedIndicators, b.IncTxnCount()
}

func removeProcessIndicator(tx *badger.Txn, id string) error {
	indicator, exists, err := getProcessIndicator(tx, id)
	if err != nil {
		return errors.Wrap(err, "retrieving existing indicator")
	}
	// No error if the indicator didn't exist.
	if !exists {
		return nil
	}

	secondaryKey, err := getSecondaryKey(indicator)
	if err != nil {
		return err
	}
	fullyQualifiedSecondaryKey := badgerhelper.GetBucketKey(uniqueProcessesBucket, secondaryKey)
	if err := tx.Delete(fullyQualifiedSecondaryKey); err != nil {
		return errors.Wrap(err, "deleting from unique bucket")
	}
	if err := tx.Delete(badgerhelper.GetBucketKey(processIndicatorBucket, []byte(id))); err != nil {
		return errors.Wrap(err, "removing indicator")
	}
	return nil
}

func (b *storeImpl) RemoveProcessIndicator(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicator")
	err := b.DB.Update(func(tx *badger.Txn) error {
		return removeProcessIndicator(tx, id)
	})
	if err != nil {
		return err
	}
	return b.IncTxnCount()
}

func (b *storeImpl) RemoveProcessIndicators(ids []string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicators")
	for _, id := range ids {
		if err := badgerhelper.RetryableUpdate(b.DB, func(tx *badger.Txn) error {
			return removeProcessIndicator(tx, id)
		}); err != nil {
			return err
		}
	}
	return b.IncTxnCount()
}

func (b *storeImpl) GetTxnCount() (uint64, error) {
	return b.crud.GetTxnCount(), nil
}

func (b *storeImpl) IncTxnCount() error {
	return b.crud.IncTxnCount()
}
