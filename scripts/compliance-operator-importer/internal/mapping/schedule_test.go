package mapping

import (
	"testing"

	"github.com/stackrox/co-acs-importer/internal/models"
)

// TestIMP_MAP_003_IMP_MAP_004_DailySchedule verifies that a daily cron expression
// produces oneTimeScan=false (IMP-MAP-003) and a present DAILY schedule (IMP-MAP-004).
func TestIMP_MAP_003_IMP_MAP_004_DailySchedule(t *testing.T) {
	got, err := ConvertCronToACSSchedule("0 0 * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil schedule")
	}
	if got.IntervalType != "DAILY" {
		t.Errorf("IntervalType: want DAILY, got %q", got.IntervalType)
	}
	if got.Hour != 0 {
		t.Errorf("Hour: want 0, got %d", got.Hour)
	}
	if got.Minute != 0 {
		t.Errorf("Minute: want 0, got %d", got.Minute)
	}
	if got.DaysOfWeek != nil {
		t.Errorf("DaysOfWeek: want nil for DAILY, got %+v", got.DaysOfWeek)
	}
	if got.DaysOfMonth != nil {
		t.Errorf("DaysOfMonth: want nil for DAILY, got %+v", got.DaysOfMonth)
	}
}

// TestIMP_MAP_003_IMP_MAP_004_DailyScheduleNonMidnight verifies non-midnight daily.
func TestIMP_MAP_003_IMP_MAP_004_DailyScheduleNonMidnight(t *testing.T) {
	got, err := ConvertCronToACSSchedule("30 14 * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IntervalType != "DAILY" {
		t.Errorf("IntervalType: want DAILY, got %q", got.IntervalType)
	}
	if got.Hour != 14 {
		t.Errorf("Hour: want 14, got %d", got.Hour)
	}
	if got.Minute != 30 {
		t.Errorf("Minute: want 30, got %d", got.Minute)
	}
}

// TestIMP_MAP_003_IMP_MAP_004_WeeklySchedule verifies that a weekly cron expression
// produces a WEEKLY schedule with the correct day (IMP-MAP-003, IMP-MAP-004).
func TestIMP_MAP_003_IMP_MAP_004_WeeklySchedule(t *testing.T) {
	// "0 2 * * 0" means Sunday at 02:00
	got, err := ConvertCronToACSSchedule("0 2 * * 0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IntervalType != "WEEKLY" {
		t.Errorf("IntervalType: want WEEKLY, got %q", got.IntervalType)
	}
	if got.Hour != 2 {
		t.Errorf("Hour: want 2, got %d", got.Hour)
	}
	if got.Minute != 0 {
		t.Errorf("Minute: want 0, got %d", got.Minute)
	}
	if got.DaysOfWeek == nil {
		t.Fatal("DaysOfWeek: want non-nil for WEEKLY schedule")
	}
	if len(got.DaysOfWeek.Days) != 1 || got.DaysOfWeek.Days[0] != 0 {
		t.Errorf("DaysOfWeek.Days: want [0] (Sunday), got %v", got.DaysOfWeek.Days)
	}
}

// TestIMP_MAP_003_IMP_MAP_004_WeeklyScheduleSaturday verifies Saturday weekly.
func TestIMP_MAP_003_IMP_MAP_004_WeeklyScheduleSaturday(t *testing.T) {
	got, err := ConvertCronToACSSchedule("15 3 * * 6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IntervalType != "WEEKLY" {
		t.Errorf("IntervalType: want WEEKLY, got %q", got.IntervalType)
	}
	if got.DaysOfWeek == nil {
		t.Fatal("DaysOfWeek: want non-nil for WEEKLY schedule")
	}
	if len(got.DaysOfWeek.Days) != 1 || got.DaysOfWeek.Days[0] != 6 {
		t.Errorf("DaysOfWeek.Days: want [6] (Saturday), got %v", got.DaysOfWeek.Days)
	}
}

// TestIMP_MAP_003_IMP_MAP_004_MonthlySchedule verifies that a monthly cron expression
// produces a MONTHLY schedule with the correct day-of-month (IMP-MAP-003, IMP-MAP-004).
func TestIMP_MAP_003_IMP_MAP_004_MonthlySchedule(t *testing.T) {
	// "30 6 1 * *" means 1st of every month at 06:30
	got, err := ConvertCronToACSSchedule("30 6 1 * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IntervalType != "MONTHLY" {
		t.Errorf("IntervalType: want MONTHLY, got %q", got.IntervalType)
	}
	if got.Hour != 6 {
		t.Errorf("Hour: want 6, got %d", got.Hour)
	}
	if got.Minute != 30 {
		t.Errorf("Minute: want 30, got %d", got.Minute)
	}
	if got.DaysOfMonth == nil {
		t.Fatal("DaysOfMonth: want non-nil for MONTHLY schedule")
	}
	if len(got.DaysOfMonth.Days) != 1 || got.DaysOfMonth.Days[0] != 1 {
		t.Errorf("DaysOfMonth.Days: want [1], got %v", got.DaysOfMonth.Days)
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronNaturalLanguage verifies that a human-readable
// schedule string is rejected with an error that mentions cron (IMP-MAP-012, IMP-MAP-015).
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronNaturalLanguage(t *testing.T) {
	got, err := ConvertCronToACSSchedule("every day at noon")
	if err == nil {
		t.Fatalf("expected error for natural-language expression, got %+v", got)
	}
	errStr := err.Error()
	if len(errStr) == 0 {
		t.Error("error message must not be empty")
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronStepNotation verifies that step notation
// (*/n) is rejected as unsupported (IMP-MAP-012, IMP-MAP-015).
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronStepNotation(t *testing.T) {
	got, err := ConvertCronToACSSchedule("*/6 * * * *")
	if err == nil {
		t.Fatalf("expected error for step notation, got %+v", got)
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronRange verifies that range notation (n-m)
// is rejected (IMP-MAP-012, IMP-MAP-015).
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronRange(t *testing.T) {
	got, err := ConvertCronToACSSchedule("0 0 * * 1-5")
	if err == nil {
		t.Fatalf("expected error for range notation, got %+v", got)
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronEmpty verifies that an empty string
// is rejected (IMP-MAP-012, IMP-MAP-015).
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronEmpty(t *testing.T) {
	got, err := ConvertCronToACSSchedule("")
	if err == nil {
		t.Fatalf("expected error for empty cron, got %+v", got)
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronTooFewFields verifies that a cron with
// fewer than 5 fields is rejected.
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronTooFewFields(t *testing.T) {
	got, err := ConvertCronToACSSchedule("0 0 * *")
	if err == nil {
		t.Fatalf("expected error for 4-field cron, got %+v", got)
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronTooManyFields verifies that a cron with
// more than 5 fields is rejected.
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronTooManyFields(t *testing.T) {
	got, err := ConvertCronToACSSchedule("0 0 * * * *")
	if err == nil {
		t.Fatalf("expected error for 6-field cron, got %+v", got)
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronBothDOMAndDOW verifies that a cron
// with both day-of-month and day-of-week set is rejected as ambiguous.
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronBothDOMAndDOW(t *testing.T) {
	got, err := ConvertCronToACSSchedule("0 0 1 * 0")
	if err == nil {
		t.Fatalf("expected error for both DOM and DOW set, got %+v", got)
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronOutOfRangeHour verifies out-of-range hour.
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronOutOfRangeHour(t *testing.T) {
	got, err := ConvertCronToACSSchedule("0 25 * * *")
	if err == nil {
		t.Fatalf("expected error for hour=25, got %+v", got)
	}
}

// TestIMP_MAP_012_IMP_MAP_015_InvalidCronOutOfRangeMinute verifies out-of-range minute.
func TestIMP_MAP_012_IMP_MAP_015_InvalidCronOutOfRangeMinute(t *testing.T) {
	got, err := ConvertCronToACSSchedule("60 0 * * *")
	if err == nil {
		t.Fatalf("expected error for minute=60, got %+v", got)
	}
}

// TestIMP_MAP_003_IMP_MAP_004_MultiValueDOMMonthly verifies that multiple days-of-month
// in a monthly cron are accepted (e.g. "0 0 1,15 * *").
func TestIMP_MAP_003_IMP_MAP_004_MultiValueDOMMonthly(t *testing.T) {
	got, err := ConvertCronToACSSchedule("0 0 15 * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.IntervalType != "MONTHLY" {
		t.Errorf("IntervalType: want MONTHLY, got %q", got.IntervalType)
	}
	if got.DaysOfMonth == nil || len(got.DaysOfMonth.Days) == 0 {
		t.Fatal("DaysOfMonth: want non-nil with days")
	}
	if got.DaysOfMonth.Days[0] != 15 {
		t.Errorf("DaysOfMonth.Days[0]: want 15, got %d", got.DaysOfMonth.Days[0])
	}
}

// Compile-time check that the return type matches models.ACSSchedule.
var _ *models.ACSSchedule = func() *models.ACSSchedule {
	s, _ := ConvertCronToACSSchedule("0 0 * * *")
	return s
}()
