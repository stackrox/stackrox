package ranking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreRanker(t *testing.T) {
	rnk := newScoreRanker()

	rnk.add(1.0)
	assert.Equal(t, int64(1), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(1))

	rnk.add(1.0)
	assert.Equal(t, int64(1), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(1))

	rnk.remove(1.0)
	assert.Equal(t, int64(1), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(1))

	rnk.add(1.0)
	assert.Equal(t, int64(1), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(1))

	rnk.add(2.0)
	assert.Equal(t, int64(1), rnk.getRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.getScoreForRank(1))
	assert.Equal(t, int64(2), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(2))

	rnk.add(1.5)
	assert.Equal(t, int64(1), rnk.getRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.getScoreForRank(1))
	assert.Equal(t, int64(2), rnk.getRankForScore(1.5))
	assert.Equal(t, float32(1.5), rnk.getScoreForRank(2))
	assert.Equal(t, int64(3), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(3))

	rnk.add(1.5)
	assert.Equal(t, int64(1), rnk.getRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.getScoreForRank(1))
	assert.Equal(t, int64(2), rnk.getRankForScore(1.5))
	assert.Equal(t, float32(1.5), rnk.getScoreForRank(2))
	assert.Equal(t, int64(3), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(3))

	rnk.remove(1.5)
	assert.Equal(t, int64(1), rnk.getRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.getScoreForRank(1))
	assert.Equal(t, int64(2), rnk.getRankForScore(1.5))
	assert.Equal(t, float32(1.5), rnk.getScoreForRank(2))
	assert.Equal(t, int64(3), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(3))

	rnk.remove(1.5)
	assert.Equal(t, int64(1), rnk.getRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.getScoreForRank(1))
	assert.Equal(t, int64(2), rnk.getRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.getScoreForRank(2))
}
