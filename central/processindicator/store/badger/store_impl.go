package badger

import (
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

type txWrapper interface {
	Set(k, v []byte) error
	Delete(id []byte) error
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

func (b *storeImpl) GetBatchProcessIndicators(ids []string) ([]*storage.ProcessIndicator, []int, error) {
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

func (b *storeImpl) GetProcessInfoToArgs() (map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetGrouped, "ProcessIndicator")
	processNamesToArgs := make(map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs)

	forEachOpts := badgerhelper.ForEachOptions{
		StripKeyPrefix:  true,
		IteratorOptions: badgerhelper.DefaultIteratorOptions(),
	}
	err := b.View(func(tx *badger.Txn) error {
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

func (b *storeImpl) addProcessIndicator(tx *badger.Txn, batch *badger.WriteBatch, indicator *storage.ProcessIndicator, data []byte) (string, error) {
	indicatorID := []byte(indicator.GetId())
	fullyQualifiedIndicatorID := badgerhelper.GetBucketKey(processIndicatorBucket, indicatorID)

	secondaryKey, err := getSecondaryKey(indicator)
	if err != nil {
		return "", err
	}

	fullyQualifiedUniqueKey := badgerhelper.GetBucketKey(uniqueProcessesBucket, secondaryKey)

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
		if err := b.removeProcessIndicator(tx, batch, oldIDString); err != nil {
			return "", errors.Wrap(err, "Removing old indicator")
		}
	}

	if err := b.AddKeysToIndex(batch, indicatorID); err != nil {
		return "", errors.Wrap(err, "error adding keys to index")
	}
	if err := batch.Set(fullyQualifiedIndicatorID, data); err != nil {
		return "", errors.Wrap(err, "inserting into indicator bucket")
	}
	if err := batch.Set(fullyQualifiedUniqueKey, indicatorID); err != nil {
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

	batch := b.DB.NewWriteBatch()
	defer batch.Cancel()

	var oldIDString string
	err = b.DB.View(func(tx *badger.Txn) error {
		oldIDString, err = b.addProcessIndicator(tx, batch, indicator, indicatorBytes)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if err := batch.Flush(); err != nil {
		return "", errors.Wrap(err, "error flushing process indicator")
	}
	return oldIDString, nil
}

func (b *storeImpl) batchAddProcessIndicators(start, end int, indicators []*storage.ProcessIndicator, dataBytes [][]byte) ([]string, error) {
	writeBatch := b.DB.NewWriteBatch()
	defer writeBatch.Cancel()

	var deletedIndicators []string
	err := b.DB.View(func(tx *badger.Txn) error {
		for i := start; i < end; i++ {
			indicator := indicators[i]
			data := dataBytes[i]
			oldID, err := b.addProcessIndicator(tx, writeBatch, indicator, data)
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
		return nil, errors.Wrap(err, "error iterating over process indicators")
	}
	if err := writeBatch.Flush(); err != nil {
		return nil, errors.Wrap(err, "error flushing process indicators")
	}
	return deletedIndicators, nil
}

func (b *storeImpl) AddProcessIndicators(indicators ...*storage.ProcessIndicator) ([]string, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.AddMany, "ProcessIndicator")

	var deletedIndicators []string
	uniqueProcesses := make(map[string]struct{})
	var filteredIndicators []*storage.ProcessIndicator

	// Run in reverse for newest to oldest
	for i := len(indicators) - 1; i > -1; i-- {
		indicator := indicators[i]
		secondaryKey, err := getSecondaryKey(indicator)
		if err != nil {
			return nil, err
		}
		secondaryString := string(secondaryKey)
		if _, ok := uniqueProcesses[secondaryString]; !ok {
			uniqueProcesses[secondaryString] = struct{}{}
			filteredIndicators = append(filteredIndicators, indicator)
		} else {
			deletedIndicators = append(deletedIndicators, indicator.GetId())
		}
	}

	dataBytes := make([][]byte, 0, len(indicators))
	for _, i := range filteredIndicators {
		data, err := proto.Marshal(i)
		if err != nil {
			return nil, err
		}
		dataBytes = append(dataBytes, data)
	}

	batch := batcher.New(len(filteredIndicators), 1000)
	for start, end, valid := batch.Next(); valid; start, end, valid = batch.Next() {
		batchedIndicators, err := b.batchAddProcessIndicators(start, end, filteredIndicators, dataBytes)
		if err != nil {
			return nil, err
		}
		deletedIndicators = append(deletedIndicators, batchedIndicators...)
	}
	return deletedIndicators, nil
}

func (b *storeImpl) removeProcessIndicator(tx *badger.Txn, txWrapper txWrapper, id string) error {
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

	if err := b.crud.AddKeysToIndex(txWrapper, []byte(id)); err != nil {
		return errors.Wrap(err, "error adding key to index")
	}

	fullyQualifiedSecondaryKey := badgerhelper.GetBucketKey(uniqueProcessesBucket, secondaryKey)
	if err := txWrapper.Delete(fullyQualifiedSecondaryKey); err != nil {
		return errors.Wrap(err, "deleting from unique bucket")
	}

	if err := txWrapper.Delete(badgerhelper.GetBucketKey(processIndicatorBucket, []byte(id))); err != nil {
		return errors.Wrap(err, "removing indicator")
	}
	return nil
}

func (b *storeImpl) RemoveProcessIndicators(ids []string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicators")
	batch := b.DB.NewWriteBatch()
	defer batch.Cancel()
	for _, id := range ids {
		if err := badgerhelper.RetryableUpdate(b.DB, func(tx *badger.Txn) error {
			return b.removeProcessIndicator(tx, batch, id)
		}); err != nil {
			return err
		}
	}
	if err := batch.Flush(); err != nil {
		return errors.Wrap(err, "error on flushing removed process indicators")
	}
	return nil
}

func (b *storeImpl) AckKeysIndexed(keys ...string) error {
	return b.crud.AckKeysIndexed(keys...)
}

func (b *storeImpl) GetKeysToIndex() ([]string, error) {
	return b.crud.GetKeysToIndex()
}

func (b *storeImpl) WalkAll(fn func(pi *storage.ProcessIndicator) error) error {
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
