package store

import (
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/generated/api/v1"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolt.DB
}

func getProcessIndicator(tx *bolt.Tx, id string) (indicator *v1.ProcessIndicator, exists bool, err error) {
	b := tx.Bucket([]byte(processIndicatorBucket))
	val := b.Get([]byte(id))
	if val == nil {
		return
	}
	indicator = new(v1.ProcessIndicator)
	exists = true
	err = proto.Unmarshal(val, indicator)
	return
}

// GetProcessIndicator returns indicator with given id.
func (b *storeImpl) GetProcessIndicator(id string) (indicator *v1.ProcessIndicator, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ProcessIndicator")
	err = b.View(func(tx *bolt.Tx) error {
		var err error
		indicator, exists, err = getProcessIndicator(tx, id)
		return err
	})
	return
}

func (b *storeImpl) GetProcessIndicators() ([]*v1.ProcessIndicator, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ProcessIndicator")
	var indicators []*v1.ProcessIndicator
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(processIndicatorBucket))
		return b.ForEach(func(k, v []byte) error {
			var indicator v1.ProcessIndicator
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
		b := tx.Bucket([]byte(uniqueProcessesBucket))
		return b.ForEach(func(k, v []byte) error {
			uniqueKey := new(v1.ProcessIndicatorUniqueKey)
			if err := proto.Unmarshal(k, uniqueKey); err != nil {
				return fmt.Errorf("key unmarshaling: %s", err)
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
func getSecondaryKey(indicator *v1.ProcessIndicator) ([]byte, error) {
	uniqueKey := &v1.ProcessIndicatorUniqueKey{
		PodId:               indicator.GetPodId(),
		ContainerName:       indicator.GetContainerName(),
		ProcessExecFilePath: indicator.GetSignal().GetExecFilePath(),
		ProcessName:         indicator.GetSignal().GetName(),
		ProcessArgs:         indicator.GetSignal().GetArgs(),
	}
	return proto.Marshal(uniqueKey)
}

func (b *storeImpl) addProcessIndicator(tx *bolt.Tx, indicator *v1.ProcessIndicator, data []byte) (string, error) {
	indicatorBucket := tx.Bucket([]byte(processIndicatorBucket))
	indicatorIDBytes := []byte(indicator.GetId())
	if indicatorBucket.Get(indicatorIDBytes) != nil {
		return "", fmt.Errorf("indicator with id '%s' already exists", indicator.GetId())
	}
	uniqueBucket := tx.Bucket([]byte(uniqueProcessesBucket))
	secondaryKey, err := getSecondaryKey(indicator)
	if err != nil {
		return "", err
	}
	oldID := uniqueBucket.Get([]byte(secondaryKey))
	var oldIDString string
	if oldID != nil {
		// Remove the old indicator.
		removeProcessIndicator(tx, oldID)
		oldIDString = string(oldID)
	}
	if err := indicatorBucket.Put(indicatorIDBytes, data); err != nil {
		return "", fmt.Errorf("inserting into indicator bucket: %s", err)
	}
	if err := uniqueBucket.Put(secondaryKey, indicatorIDBytes); err != nil {
		return "", fmt.Errorf("inserting into unique field bucket: %s", err)
	}
	return oldIDString, nil
}

// AddProcessIndicator returns the id of the indicator that was deduped or empty if none was deduped
func (b *storeImpl) AddProcessIndicator(indicator *v1.ProcessIndicator) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "ProcessIndicator")
	if indicator.GetId() == "" {
		return "", errors.New("received malformed indicator with no id")
	}
	indicatorBytes, err := proto.Marshal(indicator)
	if err != nil {
		return "", fmt.Errorf("process indicator proto marshalling: %s", err)
	}

	var oldIDString string
	err = b.Update(func(tx *bolt.Tx) error {
		oldIDString, err = b.addProcessIndicator(tx, indicator, indicatorBytes)
		return err
	})
	return oldIDString, nil
}

func (b *storeImpl) AddProcessIndicators(indicators ...*v1.ProcessIndicator) ([]string, error) {
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
		return nil
	})
	return deletedIndicators, err
}

func removeProcessIndicator(tx *bolt.Tx, id []byte) error {
	bucket := tx.Bucket([]byte(processIndicatorBucket))
	return bucket.Delete(id)
}

func (b *storeImpl) RemoveProcessIndicator(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicator")
	return b.Update(func(tx *bolt.Tx) error {
		indicator, exists, err := getProcessIndicator(tx, id)
		if err != nil {
			return fmt.Errorf("retrieving existing indicator: %s", err)
		}
		// No error if the indicator didn't exist.
		if !exists {
			return nil
		}
		uniqueBucket := tx.Bucket([]byte(uniqueProcessesBucket))
		secondaryKey, err := getSecondaryKey(indicator)
		if err != nil {
			return err
		}
		if err := uniqueBucket.Delete(secondaryKey); err != nil {
			return fmt.Errorf("deleting from unique bucket: %s", err)
		}
		if err := removeProcessIndicator(tx, []byte(id)); err != nil {
			return fmt.Errorf("removing indicator: %s", err)
		}
		return nil
	})
}
