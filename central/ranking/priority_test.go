package ranking

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapPriorities(t *testing.T) {
	rnk := NewRanker()
	rnk.Add("1", 3)
	rnk.Add("2", 5)
	rnk.Add("3", 4)
	rnk.Add("4", 1)
	rnk.Add("5", 3)

	ids := []string{"1", "2", "3", "4", "5"}
	priorities := make([]int64, len(ids))
	setPriorities(rnk, len(ids), 5,
		func(index int) string {
			return ids[index]
		}, func(index int, priority int64) {
			priorities[index] = priority
		})
	expected := []int64{8, 6, 7, 10, 9}
	assert.Equal(t, expected, priorities)
}

func TestMapPrioritiesEmptyInput(t *testing.T) {
	assert.NotPanics(t, func() {
		setPriorities(nil, 0, 0, nil, nil)
	})
}
