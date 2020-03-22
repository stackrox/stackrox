package store

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"go.etcd.io/bbolt"
)

const (
	telemetryConfigKey = "telemetryConfig"
)

var (
	log = logging.LoggerForModule()

	telemetryBucket = []byte("telemetry")
	nextSendKey     = []byte("nextSendTS")
)

type storeImpl struct {
	db *bbolt.DB

	telemetryCRUD protoCrud.MessageCrud
}

func alloc() proto.Message {
	return &storage.TelemetryConfiguration{}
}

func keyFunc(_ proto.Message) []byte {
	return []byte(telemetryConfigKey)
}

// New returns a new Store instance using the provided badger DB instance.
func New(db *bbolt.DB) (Store, error) {
	newCrud, err := protoCrud.NewMessageCrud(db, telemetryBucket, keyFunc, alloc)
	if err != nil {
		return nil, err
	}
	return &storeImpl{
		db:            db,
		telemetryCRUD: newCrud,
	}, nil
}

func (s *storeImpl) GetTelemetryConfig() (*storage.TelemetryConfiguration, error) {
	config, err := s.telemetryCRUD.Read(telemetryConfigKey)
	if config == nil {
		return nil, err
	}
	return config.(*storage.TelemetryConfiguration), err
}

func (s *storeImpl) SetTelemetryConfig(configuration *storage.TelemetryConfiguration) error {
	_, _, err := s.telemetryCRUD.Upsert(configuration)
	return err
}

func (s *storeImpl) GetNextSendTime() (time.Time, error) {
	var nextSendTime time.Time
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(telemetryBucket)
		if bucket == nil {
			return nil
		}

		value := bucket.Get(nextSendKey)
		if value == nil {
			return nil
		}
		return nextSendTime.UnmarshalBinary(value)
	})
	return nextSendTime, err
}

func (s *storeImpl) SetNextSendTime(t time.Time) error {
	marshaled, err := t.MarshalBinary()
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(telemetryBucket)
		if bucket == nil {
			return utils.Should(errors.New("telemetry bucket does not exist"))
		}
		return bucket.Put(nextSendKey, marshaled)
	})
}
