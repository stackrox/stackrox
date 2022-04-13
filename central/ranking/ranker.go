package ranking

import (
	"github.com/stackrox/stackrox/pkg/sync"
)

// Ranker ranks an object based on its score
type Ranker struct {
	idToScore map[string]float32
	sr        scoreRanker

	scoreSorterMutex sync.RWMutex
}

// NewRanker initializes an empty Ranker
func NewRanker() *Ranker {
	return &Ranker{
		idToScore: make(map[string]float32),
		sr:        newScoreRanker(),
	}
}

// GetScoreForID returns the score for given id
func (s *Ranker) GetScoreForID(id string) float32 {
	s.scoreSorterMutex.RLock()
	defer s.scoreSorterMutex.RUnlock()

	return s.idToScore[id]
}

// GetRankForID returns the current ranking based on the id of the object added with the score.
func (s *Ranker) GetRankForID(id string) int64 {
	s.scoreSorterMutex.RLock()
	defer s.scoreSorterMutex.RUnlock()

	score := s.idToScore[id]
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
	s.scoreSorterMutex.Lock()
	defer s.scoreSorterMutex.Unlock()

	if oldScore, scored := s.idToScore[id]; scored {
		s.sr.remove(oldScore)
	}
	s.sr.add(score)
	s.idToScore[id] = score
}

// Remove removes an object from having a ranking
func (s *Ranker) Remove(id string) {
	s.scoreSorterMutex.Lock()
	defer s.scoreSorterMutex.Unlock()

	score, ok := s.idToScore[id]
	if ok {
		delete(s.idToScore, id)
		s.sr.remove(score)
	}
}
