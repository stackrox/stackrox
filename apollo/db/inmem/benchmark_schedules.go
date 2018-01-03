package inmem

import (
	"fmt"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type benchmarkScheduleStore struct {
	benchmarkSchedules     map[string]*v1.BenchmarkSchedule
	benchmarkScheduleMutex sync.Mutex

	persistent db.BenchmarkScheduleStorage
}

func newBenchmarkScheduleStore(persistent db.BenchmarkScheduleStorage) *benchmarkScheduleStore {
	return &benchmarkScheduleStore{
		benchmarkSchedules: make(map[string]*v1.BenchmarkSchedule),
		persistent:         persistent,
	}
}

func (s *benchmarkScheduleStore) clone(schedule *v1.BenchmarkSchedule) *v1.BenchmarkSchedule {
	return proto.Clone(schedule).(*v1.BenchmarkSchedule)
}

func (s *benchmarkScheduleStore) loadFromPersistent() error {
	s.benchmarkScheduleMutex.Lock()
	defer s.benchmarkScheduleMutex.Unlock()
	benchmarkSchedules, err := s.persistent.GetBenchmarkSchedules(&v1.GetBenchmarkSchedulesRequest{})
	if err != nil {
		return err
	}
	for _, benchmarkSchedule := range benchmarkSchedules {
		s.benchmarkSchedules[benchmarkSchedule.GetName()] = benchmarkSchedule
	}
	return nil
}

// GetBenchmarkSchedule retrieves a benchmark schedule by name
func (s *benchmarkScheduleStore) GetBenchmarkSchedule(name string) (schedule *v1.BenchmarkSchedule, exists bool, err error) {
	s.benchmarkScheduleMutex.Lock()
	defer s.benchmarkScheduleMutex.Unlock()
	schedule, exists = s.benchmarkSchedules[name]
	return s.clone(schedule), exists, nil
}

// GetBenchmarkSchedule retrieves a benchmark schedule by name
func (s *benchmarkScheduleStore) GetBenchmarkSchedules(*v1.GetBenchmarkSchedulesRequest) ([]*v1.BenchmarkSchedule, error) {
	s.benchmarkScheduleMutex.Lock()
	defer s.benchmarkScheduleMutex.Unlock()
	schedules := make([]*v1.BenchmarkSchedule, 0, len(s.benchmarkSchedules))
	for _, schedule := range s.benchmarkSchedules {
		schedules = append(schedules, s.clone(schedule))
	}
	return schedules, nil
}

func (s *benchmarkScheduleStore) AddBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error {
	s.benchmarkScheduleMutex.Lock()
	defer s.benchmarkScheduleMutex.Unlock()
	schedule.LastUpdated = ptypes.TimestampNow()
	if _, ok := s.benchmarkSchedules[schedule.GetName()]; ok {
		return fmt.Errorf("benchmark schedule %v already exists", schedule.GetName())
	}
	if err := s.persistent.AddBenchmarkSchedule(schedule); err != nil {
		return err
	}
	s.benchmarkSchedules[schedule.GetName()] = schedule
	return nil
}

func (s *benchmarkScheduleStore) UpdateBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error {
	s.benchmarkScheduleMutex.Lock()
	defer s.benchmarkScheduleMutex.Unlock()
	schedule.LastUpdated = ptypes.TimestampNow()
	if err := s.persistent.UpdateBenchmarkSchedule(schedule); err != nil {
		return err
	}
	s.benchmarkSchedules[schedule.GetName()] = schedule
	return nil
}

func (s *benchmarkScheduleStore) RemoveBenchmarkSchedule(name string) error {
	s.benchmarkScheduleMutex.Lock()
	defer s.benchmarkScheduleMutex.Unlock()
	if err := s.persistent.RemoveBenchmarkSchedule(name); err != nil {
		return err
	}
	delete(s.benchmarkSchedules, name)
	return nil
}
