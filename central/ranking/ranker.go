package ranking

import (
	"github.com/stackrox/rox/pkg/sync"
)

type rankedObject struct {
	score                  float32
	priorityClassification int32
}

// Ranker ranks an object based on its score
type Ranker struct {
	idToRankedObject map[string]rankedObject
	sr               scoreRanker

	scoreSorterMutex sync.RWMutex
}

// NewRanker initializes an empty Ranker
func NewRanker() *Ranker {
	return &Ranker{
		idToRankedObject: make(map[string]rankedObject),
		sr:               newScoreRanker(),
	}
}

// GetScoreForID returns the score for given id
func (s *Ranker) GetScoreForID(id string) float32 {
	s.scoreSorterMutex.RLock()
	defer s.scoreSorterMutex.RUnlock()

	return s.idToRankedObject[id].score
}

// GetPriorityClassificationForID returns the risk priority classification for the given id.
func (s *Ranker) GetPriorityClassificationForID(id string) int32 {
	s.scoreSorterMutex.RLock()
	defer s.scoreSorterMutex.RUnlock()

	return s.idToRankedObject[id].priorityClassification
}

// GetRankForID returns the current ranking based on the id of the object added with the score.
func (s *Ranker) GetRankForID(id string) int64 {
	s.scoreSorterMutex.RLock()
	defer s.scoreSorterMutex.RUnlock()

	score := s.idToRankedObject[id].score
	return s.sr.getRankForScore(score)
}

// GetRankForScore returns the rank for an input score, assuming it is ranked.
func (s *Ranker) GetRankForScore(score float32) int64 {
	s.scoreSorterMutex.RLock()
	defer s.scoreSorterMutex.RUnlock()

	return s.sr.getRankForScore(score)
}

// GetScoreForRank gets the score for a given rank, assuming a score has that rank.
func (s *Ranker) GetScoreForRank(rank int64) float32 {
	s.scoreSorterMutex.RLock()
	defer s.scoreSorterMutex.RUnlock()

	return s.sr.getScoreForRank(rank)
}

// Add upserts an id and its score and recomputes the rank
func (s *Ranker) Add(id string, score float32) {
	s.AddWithPriorityClassification(id, score, 0)
}

// AddWithPriorityClassification upserts an id with score and a non-ranking priority classification.
func (s *Ranker) AddWithPriorityClassification(id string, score float32, priorityClassification int32) {
	s.scoreSorterMutex.Lock()
	defer s.scoreSorterMutex.Unlock()

	if old, scored := s.idToRankedObject[id]; scored {
		s.sr.remove(old.score)
	}
	s.sr.add(score)
	s.idToRankedObject[id] = rankedObject{
		score:                  score,
		priorityClassification: priorityClassification,
	}
}

// SetPriorityClassificationForID updates the non-ranking priority classification for an existing id.
func (s *Ranker) SetPriorityClassificationForID(id string, priorityClassification int32) {
	s.scoreSorterMutex.Lock()
	defer s.scoreSorterMutex.Unlock()

	old := s.idToRankedObject[id]
	old.priorityClassification = priorityClassification
	s.idToRankedObject[id] = old
}

// Remove removes an object from having a ranking
func (s *Ranker) Remove(id string) {
	s.scoreSorterMutex.Lock()
	defer s.scoreSorterMutex.Unlock()

	old, ok := s.idToRankedObject[id]
	if ok {
		delete(s.idToRankedObject, id)
		s.sr.remove(old.score)
	}
}
