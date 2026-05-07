export type ScheduleBase = {
    hour: number;
    minute: number;
};

export type UnsetSchedule = ScheduleBase & {
    intervalType: 'UNSET';
};

export type DailySchedule = ScheduleBase & {
    intervalType: 'DAILY';
};

export type LegacyWeeklySchedule = ScheduleBase & {
    intervalType: 'WEEKLY';
    weekly: { day: number };
};

export type WeeklySchedule = ScheduleBase & {
    intervalType: 'WEEKLY';
    daysOfWeek: { days: number[] };
};

export type MonthlySchedule = ScheduleBase & {
    intervalType: 'MONTHLY';
    daysOfMonth: { days: number[] };
};

// based on api/v2/common.proto
export type Schedule = UnsetSchedule | DailySchedule | WeeklySchedule | MonthlySchedule;

// based on storage/schedule.proto
export type LegacySchedule =
    | UnsetSchedule
    | DailySchedule
    | WeeklySchedule
    | LegacyWeeklySchedule
    | MonthlySchedule;
