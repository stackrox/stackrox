package inmem

import (
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/protoconv"
	"github.com/golang/protobuf/proto"
)

type benchmarkTriggerStore struct {
	triggers     map[string]*v1.BenchmarkTrigger
	triggerMutex sync.Mutex

	persistent db.BenchmarkTriggerStorage
}

func newBenchmarkTriggerStore(persistent db.BenchmarkTriggerStorage) *benchmarkTriggerStore {
	return &benchmarkTriggerStore{
		triggers:   make(map[string]*v1.BenchmarkTrigger),
		persistent: persistent,
	}
}

func (s *benchmarkTriggerStore) loadFromPersistent() error {
	s.triggerMutex.Lock()
	defer s.triggerMutex.Unlock()
	triggers, err := s.persistent.GetBenchmarkTriggers(&v1.GetBenchmarkTriggersRequest{})
	if err != nil {
		return err
	}
	for _, trigger := range triggers {
		s.triggers[trigger.Time.String()] = trigger
	}
	return nil
}

func (s *benchmarkTriggerStore) clone(trigger *v1.BenchmarkTrigger) *v1.BenchmarkTrigger {
	return proto.Clone(trigger).(*v1.BenchmarkTrigger)
}

// GetBenchmarkTriggers returns a slice of triggers based on the request
func (s *benchmarkTriggerStore) GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*v1.BenchmarkTrigger, error) {
	s.triggerMutex.Lock()
	defer s.triggerMutex.Unlock()
	nameSet := stringWrap(request.GetNames()).asSet()
	clusterSet := stringWrap(request.GetClusters()).asSet()
	var triggerSlice []*v1.BenchmarkTrigger
	for _, trigger := range s.triggers {
		if _, ok := nameSet[trigger.GetName()]; len(nameSet) > 0 && !ok {
			continue
		}
		// If request clusters is empty then return all
		// If the trigger has no cluster set, then it applies to all clusters
		if len(clusterSet) != 0 && len(trigger.Clusters) != 0 {
			var clusterMatch bool
			for _, cluster := range trigger.Clusters {
				if _, ok := clusterSet[cluster]; ok {
					clusterMatch = true
					break
				}
			}
			if !clusterMatch {
				continue
			}
		}

		// Check from_time <-> end_time
		// If FromTime is after trigger time then filter out
		if request.FromTime != nil && protoconv.CompareProtoTimestamps(request.FromTime, trigger.Time) == 1 {
			continue
		}
		// If the ToTime is less than the trigger time, then filter out
		if request.ToTime != nil && protoconv.CompareProtoTimestamps(request.ToTime, trigger.Time) == -1 {
			continue
		}
		triggerSlice = append(triggerSlice, s.clone(trigger))
	}
	sort.SliceStable(triggerSlice, func(i, j int) bool {
		return protoconv.CompareProtoTimestamps(triggerSlice[i].Time, triggerSlice[j].Time) == 1
	})
	return triggerSlice, nil
}

// AddBenchmarkTrigger upserts a trigger
func (s *benchmarkTriggerStore) AddBenchmarkTrigger(trigger *v1.BenchmarkTrigger) error {
	s.triggerMutex.Lock()
	defer s.triggerMutex.Unlock()
	if err := s.persistent.AddBenchmarkTrigger(trigger); err != nil {
		return err
	}
	s.triggers[trigger.Time.String()] = trigger
	return nil
}
