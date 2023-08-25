package timeutil

import (
	"time"

	"github.com/stackrox/rox/pkg/mathutil"
)

const (
	// MonthsInYear is the number of months in a calendar year
	MonthsInYear = 12
	// HoursInDay is the number of hours in a day
	HoursInDay = 24
	// MinutesInHour is the number of minutes in an hour
	MinutesInHour = 60
	// SecondsInMinute is the number of seconds in a minute
	SecondsInMinute = 60

	// DaysInWeek is the number of days in a week
	DaysInWeek = 7
)

var (
	maxDaysInMonth = [...]int{
		31,
		29,
		31,
		30,
		31,
		30,
		31,
		31,
		30,
		31,
		30,
		31,
	}
)

// DaysInMonth returns the number of days in the given month and year.
func DaysInMonth(month int, year int) int {
	if month == int(time.February) && !IsLeapYear(year) {
		return 28
	}
	return maxDaysInMonth[month-1]
}

// MaxDaysInMonth returns the number of days in the given month in *any* year.
func MaxDaysInMonth(month int) int {
	return maxDaysInMonth[month-1]
}

// IsLeapYear returns whether the given year is a leap year.
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// TimeDiffDays returns the duration t1 - t2 in days
func TimeDiffDays(t1 time.Time, t2 time.Time) int {
	hours := int(t1.Sub(t2).Hours())
	return hours / HoursInDay
}

// Date is a representation of a time point (in some time zone) with second granularity.
type Date struct {
	Year, Month, Day, Hour, Minute, Second int
}

func (d *Date) clampDayOfMonth() {
	if d.Day > DaysInMonth(d.Month, d.Year) {
		d.Day = DaysInMonth(d.Month, d.Year)
	}
}

// AddYear adds the given value to the calendar year of this date. The day of month will be clamped to the maximum
// number of days in the resulting month/year combination (regardless of whether num is negative or positive).
func (d *Date) AddYear(num int) {
	d.Year += num
	d.clampDayOfMonth()
}

// AddMonth adds the given value to the calendar month of this date. The day of month will be clamped to the maximum
// number of days in the resulting month/year combination (regardless of whether num is negative or positive).
func (d *Date) AddMonth(num int) {
	newMonth := d.Month + num
	if newMonth < 1 {
		d.Year -= (-newMonth)/MonthsInYear + 1
	} else {
		d.Year += (newMonth - 1) / MonthsInYear
	}
	d.Month = mathutil.Mod(newMonth-1, MonthsInYear) + 1
	d.clampDayOfMonth()
}

// AddDay adds the given value to the calendar day of this date.
func (d *Date) AddDay(num int) {
	newDay := d.Day + num
	for newDay < 1 {
		d.Month--
		if d.Month < 1 {
			d.Month = 12
			d.Year--
		}
		newDay += DaysInMonth(d.Month, d.Year)
	}
	for newDay > DaysInMonth(d.Month, d.Year) {
		newDay -= DaysInMonth(d.Month, d.Year)
		d.Month++
		if d.Month > 12 {
			d.Month = 1
			d.Year++
		}
	}
	d.Day = newDay
}

// AddHour adds the given value to the hour value of this date.
func (d *Date) AddHour(num int) {
	newHour := d.Hour + num
	if newHour < 0 {
		d.AddDay(-((-newHour-1)/HoursInDay + 1))
	} else if newHour >= HoursInDay {
		d.AddDay(newHour / HoursInDay)
	}
	newHour = mathutil.Mod(newHour, HoursInDay)
	d.Hour = newHour
}

// AddMinute adds the given value to the minute value of this date.
func (d *Date) AddMinute(num int) {
	newMinute := d.Minute + num
	if newMinute < 0 {
		d.AddHour(-((-newMinute-1)/MinutesInHour + 1))
	} else if newMinute >= MinutesInHour {
		d.AddHour(newMinute / MinutesInHour)
	}
	newMinute = mathutil.Mod(newMinute, MinutesInHour)
	d.Minute = newMinute
}

// AddSecond adds the given value to the second value of this date.
func (d *Date) AddSecond(num int) {
	newSecond := d.Second + num
	if newSecond < 0 {
		d.AddMinute(-((-newSecond-1)/SecondsInMinute + 1))
	} else if newSecond >= SecondsInMinute {
		d.AddMinute(newSecond / SecondsInMinute)
	}
	newSecond = mathutil.Mod(newSecond, SecondsInMinute)
	d.Second = newSecond
}

// TimeMonth returns the calendar month as a `time.Month` value.
func (d Date) TimeMonth() time.Month {
	return time.Month(d.Month)
}

// ToTime returns the `time.Time` representation of this date in the given timezone.
func (d Date) ToTime(loc *time.Location) time.Time {
	return time.Date(d.Year, d.TimeMonth(), d.Day, d.Hour, d.Minute, d.Second, 0, loc)
}

// WeekdayOfMonth returns the "weekday of month" of the day represented by this date.
func (d Date) WeekdayOfMonth() int {
	t := d.ToTime(time.UTC)
	return int(t.Weekday()) + ((d.Day-1)/DaysInWeek)*DaysInWeek
}

// DateFromTime converts a `time.Time` to a `Date` representation.
func DateFromTime(t time.Time) Date {
	return Date{
		Year:   t.Year(),
		Month:  int(t.Month()),
		Day:    t.Day(),
		Hour:   t.Hour(),
		Minute: t.Minute(),
		Second: t.Second(),
	}
}

// AdvanceMonth advances this date to the next calendar date where month has the given value (wrapping the year around,
// if necessary).
func (d *Date) AdvanceMonth(newMonth int) {
	if newMonth < d.Month {
		d.Year++
	}
	if newMonth != d.Month {
		d.Day = 1
		d.Hour = 0
		d.Minute = 0
		d.Second = 0
	}
	d.Month = newMonth
}

// AdvanceDay advances this date to the next calendar date where day has the given value (wrapping the month around,
// if necessary).
func (d *Date) AdvanceDay(newDay int) {
	if newDay < d.Day {
		d.AddMonth(1)
		for newDay > DaysInMonth(d.Month, d.Year) {
			d.AddMonth(1)
		}
	}
	if newDay != d.Day {
		d.Hour = 0
		d.Minute = 0
		d.Second = 0
	}
	d.Day = newDay
}

// AdvanceHour advances this date to the next calendar date where hour has the given value (wrapping the day around,
// if necessary).
func (d *Date) AdvanceHour(newHour int) {
	if newHour < d.Hour {
		d.AddDay(1)
	}
	if newHour != d.Hour {
		d.Minute = 0
		d.Second = 0
	}
	d.Hour = newHour
}

// AdvanceMinute advances this date to the next calendar date where minute has the given value (wrapping the hour around,
// if necessary).
func (d *Date) AdvanceMinute(newMinute int) {
	if newMinute < d.Minute {
		d.AddHour(1)
	}
	if newMinute != d.Minute {
		d.Second = 0
	}
	d.Minute = newMinute
}

// AdvanceSecond advances this date to the next calendar date where second has the given value (wrapping the minute around,
// if necessary).
func (d *Date) AdvanceSecond(newSecond int) {
	if newSecond < d.Second {
		d.AddMinute(1)
	}
	d.Second = newSecond
}

// AdvanceYearBy advances the calendar year by the given number. Negative numbers are not allowed and will leave this
// date unchanged.
func (d *Date) AdvanceYearBy(num int) {
	if num <= 0 {
		return
	}
	d.Year += num
	d.Month = 1
	d.Day = 1
	d.Hour = 0
	d.Minute = 0
	d.Second = 0
}

// AdvanceMonthBy advances the calendar month by the given number. Negative numbers are not allowed and will leave this
// date unchanged.
func (d *Date) AdvanceMonthBy(num int) {
	if num <= 0 {
		return
	}
	d.AddMonth(num)
	d.Day = 1
	d.Hour = 0
	d.Minute = 0
	d.Second = 0
}

// AdvanceDayBy advances the calendar day by the given number. Negative numbers are not allowed and will leave this
// date unchanged.
func (d *Date) AdvanceDayBy(num int) {
	if num <= 0 {
		return
	}
	d.AddDay(num)
	d.Hour = 0
	d.Minute = 0
	d.Second = 0
}

// AdvanceHourBy advances the hour of day by the given number. Negative numbers are not allowed and will leave this
// date unchanged.
func (d *Date) AdvanceHourBy(num int) {
	if num <= 0 {
		return
	}
	d.AddHour(num)
	d.Minute = 0
	d.Second = 0
}

// AdvanceMinuteBy advances the minute of hour by the given number. Negative numbers are not allowed and will leave this
// date unchanged.
func (d *Date) AdvanceMinuteBy(num int) {
	if num <= 0 {
		return
	}
	d.AddMinute(num)
	d.Second = 0
}
