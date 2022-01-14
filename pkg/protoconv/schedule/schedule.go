package schedule

import (
	"errors"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

// ConvertToCronTab validates and converts storage.Schedule to crontab format
func ConvertToCronTab(schedule *storage.Schedule) (string, error) {
	if schedule.GetHour() < 0 || schedule.GetHour() > 23 {
		return "", errors.New("Schedule hour must be within 0-23")
	}
	if schedule.GetMinute() < 0 || schedule.GetMinute() > 60 {
		return "", errors.New("Schedule hour must be within 0-59")
	}

	hours := schedule.GetHour()
	minutes := schedule.GetMinute()

	switch schedule.GetIntervalType() {
	case storage.Schedule_MONTHLY:
		daysOfMonth := schedule.GetDaysOfMonth().GetDays()
		return fmt.Sprintf("%d %d %s * *", minutes, hours, stringutils.JoinInt32(",", daysOfMonth...)), nil
	case storage.Schedule_WEEKLY:
		i := schedule.GetInterval()
		var weekDays []int32
		if _, ok := i.(*storage.Schedule_Weekly); ok {
			weekDays = []int32{schedule.GetWeekly().GetDay()}

		} else if _, ok := i.(*storage.Schedule_DaysOfWeek_); ok {
			weekDays = schedule.GetDaysOfWeek().GetDays()
		}

		for _, weekDay := range weekDays {
			if weekDay < 0 || weekDay > 6 {
				return "", fmt.Errorf("weekday of %d is invalid. Must be between 0 and 6", weekDay)
			}
		}
		return fmt.Sprintf("%d %d * * %s", minutes, hours, stringutils.JoinInt32(",", weekDays...)), nil
	case storage.Schedule_DAILY:
		return fmt.Sprintf("%d %d * * *", minutes, hours), nil
	default:
		return "", errors.New("Interval must be daily or weekly")
	}
}
