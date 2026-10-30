package ranking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRankerStoresPriorityClassificationWithScore(t *testing.T) {
	rnk := NewRanker()

	rnk.AddWithPriorityClassification("deployment-1", 10, 3)
	assert.Equal(t, float32(10), rnk.GetScoreForID("deployment-1"))
	assert.Equal(t, int32(3), rnk.GetPriorityClassificationForID("deployment-1"))

	rnk.SetPriorityClassificationForID("deployment-1", 5)
	assert.Equal(t, float32(10), rnk.GetScoreForID("deployment-1"))
	assert.Equal(t, int32(5), rnk.GetPriorityClassificationForID("deployment-1"))

	rnk.Add("deployment-1", 20)
	assert.Equal(t, float32(20), rnk.GetScoreForID("deployment-1"))
	assert.Equal(t, int32(0), rnk.GetPriorityClassificationForID("deployment-1"))
}

func TestRanker(t *testing.T) {
	rnk := NewRanker()

	rnk.Add("1", 1.0)
	assert.Equal(t, int64(1), rnk.GetRankForID("1"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(1))

	rnk.Add("2", 1.0)
	assert.Equal(t, int64(1), rnk.GetRankForID("1"))
	assert.Equal(t, int64(1), rnk.GetRankForID("2"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(1))

	rnk.Remove("2")
	assert.Equal(t, int64(1), rnk.GetRankForID("1"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(1))

	rnk.Add("2", 1.0)
	assert.Equal(t, int64(1), rnk.GetRankForID("1"))
	assert.Equal(t, int64(1), rnk.GetRankForID("2"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(1))

	rnk.Remove("3") // Should do nothing
	rnk.Add("3", 2.0)
	rnk.Add("3", 2.0)
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.GetScoreForRank(1))

	assert.Equal(t, int64(2), rnk.GetRankForID("1"))
	assert.Equal(t, int64(2), rnk.GetRankForID("2"))
	assert.Equal(t, int64(2), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(2))

	rnk.Add("4", 1.5)
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.GetScoreForRank(1))

	assert.Equal(t, int64(2), rnk.GetRankForID("4"))
	assert.Equal(t, int64(2), rnk.GetRankForScore(1.5))
	assert.Equal(t, float32(1.5), rnk.GetScoreForRank(2))

	assert.Equal(t, int64(3), rnk.GetRankForID("1"))
	assert.Equal(t, int64(3), rnk.GetRankForID("2"))
	assert.Equal(t, int64(3), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(3))

	rnk.Add("5", 1.5)
	rnk.Add("5", 1.6)
	rnk.Add("5", 1.5)
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.GetScoreForRank(1))

	assert.Equal(t, int64(2), rnk.GetRankForID("4"))
	assert.Equal(t, int64(2), rnk.GetRankForID("5"))
	assert.Equal(t, int64(2), rnk.GetRankForScore(1.5))
	assert.Equal(t, float32(1.5), rnk.GetScoreForRank(2))

	assert.Equal(t, int64(3), rnk.GetRankForID("1"))
	assert.Equal(t, int64(3), rnk.GetRankForID("2"))
	assert.Equal(t, int64(3), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(3))

	rnk.Remove("4")
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.GetScoreForRank(1))

	assert.Equal(t, int64(2), rnk.GetRankForID("5"))
	assert.Equal(t, int64(2), rnk.GetRankForScore(1.5))
	assert.Equal(t, float32(1.5), rnk.GetScoreForRank(2))

	assert.Equal(t, int64(3), rnk.GetRankForID("1"))
	assert.Equal(t, int64(3), rnk.GetRankForID("2"))
	assert.Equal(t, int64(3), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(3))

	rnk.Remove("5")
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))
	assert.Equal(t, int64(1), rnk.GetRankForScore(2.0))
	assert.Equal(t, float32(2.0), rnk.GetScoreForRank(1))

	assert.Equal(t, int64(2), rnk.GetRankForID("1"))
	assert.Equal(t, int64(2), rnk.GetRankForID("2"))
	assert.Equal(t, int64(2), rnk.GetRankForScore(1.0))
	assert.Equal(t, float32(1.0), rnk.GetScoreForRank(2))
}
