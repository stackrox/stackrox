package schedule

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/tkuchiki/go-timezone"
)

const (
	timeLayout = "15:04PM"
)

const secondsToHours = 60 * 60

// ConvertToCronTab validates and converts storage.Schedule to crontab format
func ConvertToCronTab(schedule *storage.Schedule) (string, error) {
	timeOfDay, err := time.Parse(timeLayout, schedule.GetTimeOfDay())
	if err != nil {
		return "", err
	}

	offset, err := timezone.GetOffset(schedule.GetTimezone(), false)
	if err != nil {
		return "", err
	}

	// The addition and the mod guarantees that the hours will be positive
	hours := (timeOfDay.Hour() + 24 + offset/secondsToHours) % 24
	minutes := timeOfDay.Minute()

	switch val := schedule.Interval.(type) {
	case *storage.Schedule_Weekly:
		weekDay := val.Weekly.Day
		if weekDay < 0 || weekDay > 6 {
			return "", fmt.Errorf("weekday of %d is invalid. Must be between 0 and 6", weekDay)
		}
		return fmt.Sprintf("%d %d * * %d", minutes, hours, weekDay), nil
	case *storage.Schedule_Daily:
		return fmt.Sprintf("%d %d * * *", minutes, hours), nil
	default:
		// Default to daily
		return fmt.Sprintf("%d %d * * *", minutes, hours), nil
	}
}
