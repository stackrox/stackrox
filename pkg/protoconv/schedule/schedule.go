package schedule

import (
	"errors"
	"fmt"

	v2 "github.com/stackrox/rox/generated/api/v2"
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

// ConvertV2ScheduleToProto converts v2.Schedule to storage.Schedule
func ConvertV2ScheduleToProto(schedule *v2.Schedule) *storage.Schedule {
	if schedule == nil {
		return nil
	}
	ret := &storage.Schedule{
		IntervalType: storage.Schedule_IntervalType(schedule.GetIntervalType()),
		Hour:         schedule.GetHour(),
		Minute:       schedule.GetMinute(),
	}
	switch schedule.Interval.(type) {
	case *v2.Schedule_Weekly:
		ret.Interval = &storage.Schedule_Weekly{
			Weekly: &storage.Schedule_WeeklyInterval{Day: schedule.GetWeekly().GetDay()},
		}

	case *v2.Schedule_DaysOfWeek_:
		ret.Interval = &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{Days: schedule.GetDaysOfWeek().GetDays()},
		}

	case *v2.Schedule_DaysOfMonth_:
		ret.Interval = &storage.Schedule_DaysOfMonth_{
			DaysOfMonth: &storage.Schedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
}

// ConvertProtoScheduleToV2 converts storage.Schedule to v2.Schedule
func ConvertProtoScheduleToV2(schedule *storage.Schedule) *v2.Schedule {
	if schedule == nil {
		return nil
	}
	ret := &v2.Schedule{
		IntervalType: v2.Schedule_IntervalType(schedule.GetIntervalType()),
		Hour:         schedule.GetHour(),
		Minute:       schedule.GetMinute(),
	}
	switch schedule.Interval.(type) {
	case *storage.Schedule_Weekly:
		ret.Interval = &v2.Schedule_Weekly{
			Weekly: &v2.Schedule_WeeklyInterval{Day: schedule.GetWeekly().GetDay()},
		}

	case *storage.Schedule_DaysOfWeek_:
		ret.Interval = &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{Days: schedule.GetDaysOfWeek().GetDays()},
		}

	case *storage.Schedule_DaysOfMonth_:
		ret.Interval = &v2.Schedule_DaysOfMonth_{
			DaysOfMonth: &v2.Schedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
}
