import {
    DailySchedule,
    MonthlySchedule,
    Schedule,
    ScheduleBase,
    UnsetSchedule,
    WeeklySchedule,
} from 'services/ComplianceEnhancedService';
import { getDayOfMonthWithOrdinal, getTimeHoursMinutes } from 'utils/dateUtils';

import { ScanConfigFormValues, ScanConfigParameters } from './Wizard/useFormikScanConfig';

export function convertFormikParametersToSchedule(parameters: ScanConfigParameters): Schedule {
    const { intervalType, time, daysOfWeek, daysOfMonth } = parameters;

    // Convert the time to hour and minute
    const [hourString, minuteString] = time.split(/[: ]+/);
    let hour = parseInt(hourString);
    const minute = parseInt(minuteString);

    // Convert 12-hour format to 24-hour format
    if (time.includes('PM') && hour < 12) {
        hour += 12;
    }
    if (time.includes('AM') && hour === 12) {
        hour = 0;
    }

    const baseSchedule: ScheduleBase = {
        hour,
        minute,
    };

    switch (intervalType) {
        case 'WEEKLY': {
            const weeklySchedule: WeeklySchedule = {
                ...baseSchedule,
                intervalType: 'WEEKLY',
                daysOfWeek: { days: daysOfWeek.map((day) => parseInt(day)) },
            };
            return weeklySchedule;
        }

        case 'MONTHLY': {
            const monthlySchedule: MonthlySchedule = {
                ...baseSchedule,
                intervalType: 'MONTHLY',
                daysOfMonth: { days: daysOfMonth.map((day) => parseInt(day)) },
            };
            return monthlySchedule;
        }

        case 'DAILY': {
            const dailySchedule: DailySchedule = {
                ...baseSchedule,
                intervalType: 'DAILY',
            };
            return dailySchedule;
        }

        case null:
        default: {
            const unsetSchedule: UnsetSchedule = {
                ...baseSchedule,
                intervalType: 'UNSET',
            };
            return unsetSchedule;
        }
    }
}

export function convertFormikToScanConfig(formikValues: ScanConfigFormValues) {
    const { parameters, clusters, profiles } = formikValues;
    const { name } = parameters;

    const scanSchedule = convertFormikParametersToSchedule(parameters);

    return {
        scanName: name,
        scanConfig: {
            oneTimeScan: false,
            profiles,
            scanSchedule,
        },
        clusters,
    };
}

export function formatScanSchedule(schedule: Schedule) {
    const daysOfWeekMap = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

    const formatDays = (days: string[]): string => {
        if (days.length === 1) {
            return days[0];
        }
        if (days.length === 2) {
            return days.join(' and ');
        }
        return `${days.slice(0, -1).join(', ')}, and ${days[days.length - 1]}`;
    };

    // arbitrary date, we only care about the time
    const date = new Date(2000, 0, 0, schedule.hour, schedule.minute);
    const timeString = getTimeHoursMinutes(date);

    switch (schedule.intervalType) {
        case 'DAILY':
            return `Daily at ${timeString}`;
        case 'WEEKLY': {
            const daysOfWeek = schedule.daysOfWeek.days.map((day) => daysOfWeekMap[day]);
            return `Every ${formatDays(daysOfWeek)} at ${timeString}`;
        }
        case 'MONTHLY': {
            const formattedDaysOfMonth = schedule.daysOfMonth.days.map(getDayOfMonthWithOrdinal);
            return `Monthly on the ${formatDays(formattedDaysOfMonth)} at ${timeString}`;
        }
        default:
            return 'Invalid Schedule';
    }
}
