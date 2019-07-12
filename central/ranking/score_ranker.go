package ranking

// Ranker ranks an object based on its score
type scoreRanker interface {
	getRankForScore(score float32) int64
	getScoreForRank(rank int64) float32
	add(score float32)
	remove(score float32)
}

// NewRanker initializes an empty Ranker
func newScoreRanker() scoreRanker {
	ss := &scoreRankerImpl{
		scoreToRank:  make(map[float32]int64),
		rankToScore:  make(map[int64]float32),
		scoreToCount: make(map[float32]int64),
	}
	return ss
}

type scoreRankerImpl struct {
	scoreToRank  map[float32]int64
	rankToScore  map[int64]float32
	scoreToCount map[float32]int64
}

func (s *scoreRankerImpl) getRankForScore(score float32) int64 {
	return s.scoreToRank[score]
}

func (s *scoreRankerImpl) getScoreForRank(rank int64) float32 {
	return s.rankToScore[rank]
}

func (s *scoreRankerImpl) add(score float32) {
	val, ok := s.scoreToCount[score]
	if ok {
		s.scoreToCount[score] = val + 1
	} else {
		s.scoreToCount[score] = 1
		s.addAndCompute(score)
	}
}

func (s *scoreRankerImpl) remove(score float32) {
	val, ok := s.scoreToCount[score]
	if ok && val == 1 {
		delete(s.scoreToCount, score)
		s.removeAndCompute(score)
	} else if ok {
		s.scoreToCount[score] = val - 1
	}
}

// Helper functions

func (s *scoreRankerImpl) addAndCompute(newScore float32) {
	// Find this items rank.
	var maxRank int64
	for score, rank := range s.scoreToRank {
		if score > newScore && rank > maxRank {
			maxRank = rank
		}
	}

	// Move all scores with a smaller score down a rank.
	for score, rank := range s.scoreToRank {
		if score < newScore {
			s.rankToScore[rank+1] = score
		}
	}

	// Insert the new rank and score. (adding one here makes us 1 indexed instead of 0 indexes)
	s.rankToScore[maxRank+1] = newScore

	// Reset score to rank mapping.
	for rank, score := range s.rankToScore {
		s.scoreToRank[score] = rank
	}
}

func (s *scoreRankerImpl) removeAndCompute(newScore float32) {
	lastRank := int64(len(s.rankToScore))
	// Move all scores less than the removed score up a rank.
	for score, rank := range s.scoreToRank {
		if score < newScore {
			s.rankToScore[rank-1] = score
		}
	}
	delete(s.rankToScore, lastRank)

	// Reset score to rank mapping.
	for rank, score := range s.rankToScore {
		s.scoreToRank[score] = rank
	}
	delete(s.scoreToRank, newScore)
}
