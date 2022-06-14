package manager

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/compliance"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sync"
	"gopkg.in/robfig/cron.v2"
)

// scheduleInstance is an instantiated schedule, i.e., including the business logic around a schedule specification.
type scheduleInstance struct {
	id string // immutable, not protected by mutex

	mutex sync.RWMutex

	spec *storage.ComplianceRunSchedule

	cronSchedule cron.Schedule

	lastRun         *runInstance
	lastFinishedRun *runInstance

	nextRunTime time.Time
}

func (s *scheduleInstance) ToProto() *v1.ComplianceRunScheduleInfo {
	if s == nil {
		return nil
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return &v1.ComplianceRunScheduleInfo{
		Schedule:         s.spec,
		LastRun:          s.lastRun.ToProto(),
		LastCompletedRun: s.lastFinishedRun.ToProto(),
		NextRunTime:      timeToProto(s.nextRunTime),
	}
}

func (s *scheduleInstance) clusterAndStandard() compliance.ClusterStandardPair {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return compliance.ClusterStandardPair{
		ClusterID:  s.spec.GetClusterId(),
		StandardID: s.spec.GetStandardId(),
	}
}

func (s *scheduleInstance) updateNextTimeNoLock() {
	if s.spec.GetSuspended() || s.cronSchedule == nil {
		s.nextRunTime = time.Time{}
	} else {
		s.nextRunTime = s.cronSchedule.Next(time.Now())
	}
}

func (s *scheduleInstance) checkAndUpdate(now time.Time) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.nextRunTime.IsZero() || s.nextRunTime.After(now) {
		return false
	}

	s.updateNextTimeNoLock()
	return true
}

func (s *scheduleInstance) update(spec *storage.ComplianceRunSchedule) error {
	if s.id != spec.GetId() {
		return errors.New("schedule IDs cannot be changed")
	}

	cronSchedule, err := cron.Parse(spec.GetCrontabSpec())
	if err != nil {
		return errors.Wrap(err, "parsing crontab spec")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.spec, s.cronSchedule = spec, cronSchedule
	s.updateNextTimeNoLock()

	return nil
}

func newScheduleInstance(spec *storage.ComplianceRunSchedule) (*scheduleInstance, error) {
	cronSchedule, err := cron.Parse(spec.GetCrontabSpec())
	if err != nil {
		return nil, errors.Wrap(err, "parsing crontab spec")
	}
	si := &scheduleInstance{
		id:           spec.GetId(),
		spec:         spec,
		cronSchedule: cronSchedule,
	}

	si.updateNextTimeNoLock()
	return si, nil
}
