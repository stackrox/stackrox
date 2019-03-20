package ranking

import (
	"sort"

	"github.com/stackrox/rox/pkg/sync"
)

// Ranker ranks an object based on its score
type Ranker struct {
	objMap           map[string]float32
	objScoreMap      map[string]int64
	scoreSorterMutex sync.RWMutex
}

type scoreElement struct {
	id    string
	score float32
}

func (s *Ranker) compute() {
	deployments := make([]*scoreElement, 0, len(s.objMap))
	for k, v := range s.objMap {
		deployments = append(deployments, &scoreElement{id: k, score: v})
	}
	sort.Slice(deployments, func(i, j int) bool { return deployments[i].score > deployments[j].score })
	prevScore := float32(-1)
	var currPriority int64
	for _, d := range deployments {
		if d.score != prevScore {
			currPriority++
			prevScore = d.score
		}
		s.objScoreMap[d.id] = currPriority
	}
}

// NewRanker initializes an empty Ranker
func NewRanker() *Ranker {
	ss := &Ranker{
		objMap:      make(map[string]float32),
		objScoreMap: make(map[string]int64),
	}
	return ss
}

// Get returns the current ranking based on the id
func (s *Ranker) Get(id string) int64 {
	s.scoreSorterMutex.RLock()
	defer s.scoreSorterMutex.RUnlock()
	return s.objScoreMap[id]
}

// Add upserts an id and its score and recomputes the rank
func (s *Ranker) Add(id string, score float32) {
	s.scoreSorterMutex.Lock()
	defer s.scoreSorterMutex.Unlock()
	val, ok := s.objMap[id]
	if ok && val == score {
		return
	}
	s.objMap[id] = score
	s.compute()
}

// Remove removes an object from having a ranking
func (s *Ranker) Remove(id string) {
	s.scoreSorterMutex.Lock()
	defer s.scoreSorterMutex.Unlock()
	delete(s.objMap, id)
	s.compute()
}
