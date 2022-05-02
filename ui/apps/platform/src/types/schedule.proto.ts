export type Schedule = UnsetSchedule | DailySchedule | WeeklySchedule;

export type ScheduleIntervalType = 'UNSET' | 'DAILY' | 'WEEKLY'; // | 'MONTHLY'

export type UnsetSchedule = {
    intervalType: 'UNSET';
} & BaseSchedule;

export type DailySchedule = {
    intervalType: 'DAILY';
} & BaseSchedule;

export type WeeklySchedule = {
    intervalType: 'WEEKLY';
    // Sunday = 0, Monday = 1, .... Saturday =  6
    weekly: {
        day: number; // int32
    };
} & BaseSchedule;

export type BaseSchedule = {
    intervalType: ScheduleIntervalType;
    hour: number;
    minute: number;
};
