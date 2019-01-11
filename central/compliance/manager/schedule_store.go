package manager

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
)

// ScheduleStore takes care of storing the compliance run schedule specifications.
type ScheduleStore interface {
	ListSchedules() ([]*storage.ComplianceRunSchedule, error)
	UpsertSchedule(schedule *storage.ComplianceRunSchedule) error
	DeleteSchedule(id string) error
}

var (
	schedulesBucket = []byte("compliance-schedules")
)

type scheduleStoreImpl struct {
	crud protoCrud.MessageCrud
}

func key(msg proto.Message) []byte {
	if scheduleMsg, ok := msg.(*storage.ComplianceRunSchedule); ok {
		return []byte(scheduleMsg.GetId())
	}
	return nil
}

func alloc() proto.Message {
	return &storage.ComplianceRunSchedule{}
}

func newScheduleStore(db *bbolt.DB) (*scheduleStoreImpl, error) {
	err := bolthelper.RegisterBucket(db, schedulesBucket)
	if err != nil {
		return nil, err
	}
	return &scheduleStoreImpl{
		crud: protoCrud.NewMessageCrud(
			db,
			schedulesBucket,
			key,
			alloc),
	}, nil
}

func (s *scheduleStoreImpl) ListSchedules() ([]*storage.ComplianceRunSchedule, error) {
	msgs, err := s.crud.ReadAll()
	if err != nil {
		return nil, err
	}
	schedules := make([]*storage.ComplianceRunSchedule, len(msgs))
	for i, msg := range msgs {
		schedules[i] = msg.(*storage.ComplianceRunSchedule)
	}
	return schedules, nil
}

func (s *scheduleStoreImpl) UpsertSchedule(schedule *storage.ComplianceRunSchedule) error {
	return s.crud.Upsert(schedule)
}

func (s *scheduleStoreImpl) DeleteSchedule(id string) error {
	return s.crud.Delete(id)
}
