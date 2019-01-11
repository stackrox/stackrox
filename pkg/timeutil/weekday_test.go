package timeutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeekdayOfMonthToDayOfMonth(t *testing.T) {
	t.Parallel()

	// Jan 1, 2019 is a Tuesday

	// 0: first Sunday -> Jan 6
	assert.Equal(t, 6, WeekdayOfMonthToDayOfMonth(0, 1, 2019))
	// 1: first Monday -> Jan 7
	assert.Equal(t, 7, WeekdayOfMonthToDayOfMonth(1, 1, 2019))
	// 2: first Tuesday -> Jan 1
	assert.Equal(t, 1, WeekdayOfMonthToDayOfMonth(2, 1, 2019))
	// 3: first Wednesday -> Jan 2
	assert.Equal(t, 2, WeekdayOfMonthToDayOfMonth(3, 1, 2019))

	// 14: third Sunday -> Jan 20
	assert.Equal(t, 20, WeekdayOfMonthToDayOfMonth(14, 1, 2019))
	// 16: third Tuesday -> Jan 15
	assert.Equal(t, 15, WeekdayOfMonthToDayOfMonth(16, 1, 2019))
	// 17: third Wednesday -> Jan 16
	assert.Equal(t, 16, WeekdayOfMonthToDayOfMonth(17, 1, 2019))

	// 32: fifth Thursday -> Jan 31
	assert.Equal(t, 31, WeekdayOfMonthToDayOfMonth(32, 1, 2019))
	// 33: fifth Friday -> only four Fridays in Jan 2019
	assert.Equal(t, -1, WeekdayOfMonthToDayOfMonth(33, 1, 2019))
}
