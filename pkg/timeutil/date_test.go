package timeutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDate_AddYear(t *testing.T) {
	t.Parallel()

	cases := []struct {
		base     Date
		numYears int
		expected Date
	}{
		{
			base:     Date{Year: 2019, Month: 1, Day: 4, Hour: 13, Minute: 37, Second: 42},
			numYears: 1,
			expected: Date{Year: 2020, Month: 1, Day: 4, Hour: 13, Minute: 37, Second: 42},
		},
		{
			base:     Date{Year: 2020, Month: 2, Day: 29, Hour: 13, Minute: 37, Second: 42},
			numYears: 3,
			expected: Date{Year: 2023, Month: 2, Day: 28, Hour: 13, Minute: 37, Second: 42},
		},
		{
			base:     Date{Year: 2024, Month: 2, Day: 29, Hour: 13, Minute: 37, Second: 42},
			numYears: -4,
			expected: Date{Year: 2020, Month: 2, Day: 29, Hour: 13, Minute: 37, Second: 42},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			curr := c.base
			curr.AddYear(c.numYears)
			assert.Equal(t, c.expected, curr)
		})
	}
}

func TestDate_AddMonth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		base      Date
		numMonths int
		expected  Date
	}{
		{
			base:      Date{Year: 2019, Month: 1, Day: 4, Hour: 13, Minute: 37, Second: 42},
			numMonths: 1,
			expected:  Date{Year: 2019, Month: 2, Day: 4, Hour: 13, Minute: 37, Second: 42},
		},
		{
			base:      Date{Year: 2020, Month: 2, Day: 29, Hour: 13, Minute: 37, Second: 42},
			numMonths: 11,
			expected:  Date{Year: 2021, Month: 1, Day: 29, Hour: 13, Minute: 37, Second: 42},
		},
		{
			base:      Date{Year: 2020, Month: 2, Day: 29, Hour: 13, Minute: 37, Second: 42},
			numMonths: 15,
			expected:  Date{Year: 2021, Month: 5, Day: 29, Hour: 13, Minute: 37, Second: 42},
		},
		{
			base:      Date{Year: 2024, Month: 2, Day: 29, Hour: 13, Minute: 37, Second: 42},
			numMonths: -4,
			expected:  Date{Year: 2023, Month: 10, Day: 29, Hour: 13, Minute: 37, Second: 42},
		},
		{
			base:      Date{Year: 2024, Month: 2, Day: 29, Hour: 13, Minute: 37, Second: 42},
			numMonths: -14,
			expected:  Date{Year: 2022, Month: 12, Day: 29, Hour: 13, Minute: 37, Second: 42},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			curr := c.base
			curr.AddMonth(c.numMonths)
			assert.Equal(t, c.expected, curr)
		})
	}
}

func TestDate_AddDay(t *testing.T) {
	t.Parallel()

	cases := []struct {
		base     Date
		numDays  int
		expected Date
	}{
		{
			base:     Date{Year: 2019, Month: 1, Day: 4, Hour: 13, Minute: 37, Second: 42},
			numDays:  5,
			expected: Date{Year: 2019, Month: 1, Day: 9, Hour: 13, Minute: 37, Second: 42},
		},
		{
			base:     Date{Year: 2019, Month: 2, Day: 27, Hour: 13, Minute: 37, Second: 42},
			numDays:  3,
			expected: Date{Year: 2019, Month: 3, Day: 2, Hour: 13, Minute: 37, Second: 42},
		},
		{
			base:     Date{Year: 2019, Month: 1, Day: 4, Hour: 13, Minute: 37, Second: 42},
			numDays:  -4,
			expected: Date{Year: 2018, Month: 12, Day: 31, Hour: 13, Minute: 37, Second: 42},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			curr := c.base
			curr.AddDay(c.numDays)
			assert.Equal(t, c.expected, curr)
		})
	}
}
