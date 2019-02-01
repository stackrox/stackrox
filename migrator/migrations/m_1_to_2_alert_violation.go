package migrations

import (
	"fmt"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/types"
)

var (
	alertsBucket = []byte("alerts")

	alertViolationMigration = types.Migration{
		StartingSeqNum: 1,
		VersionAfter:   storage.Version{SeqNum: 2},
		Run: func(db *bolt.DB) error {
			alertsBucket := bolthelpers.TopLevelRef(db, alertsBucket)
			modifiedAlertBytes := make(map[string][]byte)
			err := alertsBucket.View(func(b *bolt.Bucket) error {
				return b.ForEach(func(k, v []byte) error {
					alert := new(storage.Alert)
					err := proto.Unmarshal(v, alert)
					if err != nil {
						return fmt.Errorf("proto umarshaling failed: %s", err)
					}
					indexToRemove := -1
					for i, violation := range alert.GetViolations() {
						if len(violation.GetDEPRECATEDProcesses()) > 0 {
							alert.ProcessViolation = &storage.Alert_ProcessViolation{
								Message:   violation.GetMessage(),
								Processes: violation.GetDEPRECATEDProcesses(),
							}
							// Exactly one alert will have processes in the old schema.
							indexToRemove = i
							break
						}
					}
					if indexToRemove != -1 {
						alert.Violations = append(alert.Violations[:indexToRemove], alert.Violations[indexToRemove+1:]...)
						alertBytes, err := proto.Marshal(alert)
						if err != nil {
							return fmt.Errorf("marshaling %+v: %v", alert, err)
						}
						modifiedAlertBytes[alert.GetId()] = alertBytes
					}
					return nil
				})
			})
			if err != nil {
				return fmt.Errorf("failed to read existing alerts into memory: %v", err)
			}
			return alertsBucket.Update(func(b *bolt.Bucket) error {
				for id, alertBytes := range modifiedAlertBytes {
					err := b.Put([]byte(id), alertBytes)
					if err != nil {
						return fmt.Errorf("inserting alert %s: %v", id, err)
					}
				}
				return nil
			})
		},
	}
)

func init() {
	mustRegisterMigration(alertViolationMigration)
}
