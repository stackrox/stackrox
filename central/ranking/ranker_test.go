package ranking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRanker(t *testing.T) {
	rnk := NewRanker()

	rnk.Add("1", 1.0)
	assert.Equal(t, int64(1), rnk.GetRankForID("1"))

	rnk.Add("2", 1.0)
	assert.Equal(t, int64(1), rnk.GetRankForID("1"))
	assert.Equal(t, int64(1), rnk.GetRankForID("2"))

	rnk.Remove("2")
	assert.Equal(t, int64(1), rnk.GetRankForID("1"))

	rnk.Add("2", 1.0)
	assert.Equal(t, int64(1), rnk.GetRankForID("1"))
	assert.Equal(t, int64(1), rnk.GetRankForID("2"))

	rnk.Remove("3") // Should do nothing
	rnk.Add("3", 2.0)
	rnk.Add("3", 2.0)
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))

	assert.Equal(t, int64(2), rnk.GetRankForID("1"))
	assert.Equal(t, int64(2), rnk.GetRankForID("2"))

	rnk.Add("4", 1.5)
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))

	assert.Equal(t, int64(2), rnk.GetRankForID("4"))

	assert.Equal(t, int64(3), rnk.GetRankForID("1"))
	assert.Equal(t, int64(3), rnk.GetRankForID("2"))

	rnk.Add("5", 1.5)
	rnk.Add("5", 1.6)
	rnk.Add("5", 1.5)
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))

	assert.Equal(t, int64(2), rnk.GetRankForID("4"))
	assert.Equal(t, int64(2), rnk.GetRankForID("5"))

	assert.Equal(t, int64(3), rnk.GetRankForID("1"))
	assert.Equal(t, int64(3), rnk.GetRankForID("2"))

	rnk.Remove("4")
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))

	assert.Equal(t, int64(2), rnk.GetRankForID("5"))

	assert.Equal(t, int64(3), rnk.GetRankForID("1"))
	assert.Equal(t, int64(3), rnk.GetRankForID("2"))

	rnk.Remove("5")
	assert.Equal(t, int64(1), rnk.GetRankForID("3"))

	assert.Equal(t, int64(2), rnk.GetRankForID("1"))
	assert.Equal(t, int64(2), rnk.GetRankForID("2"))
}
