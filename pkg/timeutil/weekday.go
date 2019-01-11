package timeutil

import "time"

const (
	// MaxWeekdayOfMonth returns the maximum "weekday of month" value for a month in any given year. Since all months
	// can have more than 28 days (4 full weeks), there can be a maximum of five instances of any given weekday.
	MaxWeekdayOfMonth = 5*DaysInWeek - 1
)

// FirstWeekday returns the weekday (0 - Sunday through 6 - Saturday) of the first day of the month of the given year.
func FirstWeekday(month, year int) int {
	return int(time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Weekday())
}

// WeekdayOfMonthToDayOfMonth converts a "weekday of month" value to the day of month for a given month and year.
// A "weekday of month" value allows identifying the k-th <weekday> of a month. For weekdays 0 through 6, values 0-6
// represent the first respective weekday of the month. The k-th <weekday #i> of a month is identified by
// (k - 1) * 7 + i. For example, 15 represents the third Monday of the month.
func WeekdayOfMonthToDayOfMonth(weekdayOfMonth, month, year int) int {
	if weekdayOfMonth < 0 {
		return -1
	}

	weekday := weekdayOfMonth % DaysInWeek
	weeksIn := weekdayOfMonth / DaysInWeek
	weekOffset := weekday - FirstWeekday(month, year)
	if weekOffset < 0 {
		weekOffset += DaysInWeek
	}
	dayOfMonth := 1 + DaysInWeek*weeksIn + weekOffset
	if dayOfMonth > DaysInMonth(month, year) {
		return -1
	}
	return dayOfMonth
}
