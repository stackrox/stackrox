package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolthelper.BoltWrapper
}

func getProcessIndicator(tx *bolt.Tx, id string) (indicator *storage.ProcessIndicator, exists bool, err error) {
	b := tx.Bucket(processIndicatorBucket)
	val := b.Get([]byte(id))
	if val == nil {
		return
	}
	indicator = new(storage.ProcessIndicator)
	exists = true
	err = proto.Unmarshal(val, indicator)
	return
}

// GetProcessIndicator returns indicator with given id.
func (b *storeImpl) GetProcessIndicator(id string) (indicator *storage.ProcessIndicator, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ProcessIndicator")
	err = b.View(func(tx *bolt.Tx) error {
		var err error
		indicator, exists, err = getProcessIndicator(tx, id)
		return err
	})
	return
}

func (b *storeImpl) GetProcessIndicators() ([]*storage.ProcessIndicator, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ProcessIndicator")
	var indicators []*storage.ProcessIndicator
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(processIndicatorBucket)
		return b.ForEach(func(k, v []byte) error {
			var indicator storage.ProcessIndicator
			if err := proto.Unmarshal(v, &indicator); err != nil {
				return err
			}
			indicators = append(indicators, &indicator)
			return nil
		})
	})
	return indicators, err
}

func (b *storeImpl) GetProcessInfoToArgs() (map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetGrouped, "ProcessIndicator")
	processNamesToArgs := make(map[processindicator.ProcessWithContainerInfo][]processindicator.IDAndArgs)
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(uniqueProcessesBucket)
		return b.ForEach(func(k, v []byte) error {
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

func (b *storeImpl) addProcessIndicator(tx *bolt.Tx, indicator *storage.ProcessIndicator, data []byte) (string, error) {
	indicatorBucket := tx.Bucket(processIndicatorBucket)
	indicatorIDBytes := []byte(indicator.GetId())
	if indicatorBucket.Get(indicatorIDBytes) != nil {
		return "", fmt.Errorf("indicator with id '%s' already exists", indicator.GetId())
	}
	uniqueBucket := tx.Bucket(uniqueProcessesBucket)
	secondaryKey, err := getSecondaryKey(indicator)
	if err != nil {
		return "", err
	}
	oldID := uniqueBucket.Get([]byte(secondaryKey))
	var oldIDString string
	if oldID != nil {
		oldIDString = string(oldID)
		// Remove the old indicator.
		if err := removeProcessIndicator(tx, oldIDString); err != nil {
			return "", errors.Wrap(err, "Removing old indicator")
		}
	}
	if err := indicatorBucket.Put(indicatorIDBytes, data); err != nil {
		return "", errors.Wrap(err, "inserting into indicator bucket")
	}
	if err := uniqueBucket.Put(secondaryKey, indicatorIDBytes); err != nil {
		return "", errors.Wrap(err, "inserting into unique field bucket")
	}
	return oldIDString, nil
}

// AddProcessIndicator returns the id of the indicator that was deduped or empty if none was deduped
func (b *storeImpl) AddProcessIndicator(indicator *storage.ProcessIndicator) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "ProcessIndicator")
	if indicator.GetId() == "" {
		return "", errors.New("received malformed indicator with no id")
	}
	indicatorBytes, err := proto.Marshal(indicator)
	if err != nil {
		return "", errors.Wrap(err, "process indicator proto marshalling")
	}

	var oldIDString string
	err = b.Update(func(tx *bolt.Tx) error {
		oldIDString, err = b.addProcessIndicator(tx, indicator, indicatorBytes)
		if err != nil {
			return err
		}
		if oldIDString != "" {
			// Purposefully increment the txn count here as it will be a call in the datastore
			if err := b.BoltWrapper.IncTxnCount(tx); err != nil {
				return err
			}
		}
		return nil
	})
	return oldIDString, err
}

func (b *storeImpl) AddProcessIndicators(indicators ...*storage.ProcessIndicator) ([]string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.AddMany, "ProcessIndicator")
	var deletedIndicators []string
	dataBytes := make([][]byte, 0, len(indicators))
	for _, i := range indicators {
		data, err := proto.Marshal(i)
		if err != nil {
			return nil, err
		}
		dataBytes = append(dataBytes, data)
	}
	err := b.Update(func(tx *bolt.Tx) error {
		for i := 0; i < len(indicators); i++ {
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
		if len(deletedIndicators) > 0 {
			// Purposefully increment the txn count here as it will be a call in the datastore
			if err := b.BoltWrapper.IncTxnCount(tx); err != nil {
				return err
			}
		}
		return nil
	})
	return deletedIndicators, err
}

func removeProcessIndicator(tx *bolt.Tx, id string) error {
	indicator, exists, err := getProcessIndicator(tx, id)
	if err != nil {
		return errors.Wrap(err, "retrieving existing indicator")
	}
	// No error if the indicator didn't exist.
	if !exists {
		return nil
	}
	uniqueBucket := tx.Bucket(uniqueProcessesBucket)
	secondaryKey, err := getSecondaryKey(indicator)
	if err != nil {
		return err
	}
	if err := uniqueBucket.Delete(secondaryKey); err != nil {
		return errors.Wrap(err, "deleting from unique bucket")
	}
	bucket := tx.Bucket(processIndicatorBucket)
	if err := bucket.Delete([]byte(id)); err != nil {
		return errors.Wrap(err, "removing indicator")
	}
	return nil
}

func (b *storeImpl) RemoveProcessIndicator(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicator")
	return b.Update(func(tx *bolt.Tx) error {
		return removeProcessIndicator(tx, id)
	})
}

func (b *storeImpl) RemoveProcessIndicators(ids []string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicator")
	return b.Update(func(tx *bolt.Tx) error {
		for _, i := range ids {
			if err := removeProcessIndicator(tx, i); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *storeImpl) GetTxnCount() (txNum uint64, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		txNum = b.BoltWrapper.GetTxnCount(tx)
		return nil
	})
	return
}

func (b *storeImpl) IncTxnCount() error {
	return b.Update(func(tx *bolt.Tx) error {
		// The b.Update increments the txn count automatically
		return nil
	})
}
