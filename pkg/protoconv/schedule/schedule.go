package schedule

import (
	"errors"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
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
	case storage.Schedule_WEEKLY:
		weekDay := schedule.GetWeekly().GetDay()
		if weekDay < 0 || weekDay > 6 {
			return "", fmt.Errorf("weekday of %d is invalid. Must be between 0 and 6", weekDay)
		}
		return fmt.Sprintf("%d %d * * %d", minutes, hours, weekDay), nil
	case storage.Schedule_DAILY:
		return fmt.Sprintf("%d %d * * *", minutes, hours), nil
	default:
		return "", errors.New("Interval must be daily or weekly")
	}
}
