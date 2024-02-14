import { DayOfMonth, DayOfWeek } from 'Components/PatternFly/DayPickerDropdown';
import {
    ComplianceScanConfigurationStatus,
    DailySchedule,
    MonthlySchedule,
    Schedule,
    ScheduleBase,
    UnsetSchedule,
    WeeklySchedule,
} from 'services/ComplianceEnhancedService';
import { getDayOfMonthWithOrdinal, getTimeHoursMinutes } from 'utils/dateUtils';

export type ScanConfigParameters = {
    name: string;
    description: string;
    intervalType: 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'UNSET';
    time: string;
    daysOfWeek: DayOfWeek[];
    daysOfMonth: DayOfMonth[];
};

export type ScanConfigFormValues = {
    id?: string;
    parameters: ScanConfigParameters;
    clusters: string[];
    profiles: string[];
};

export type PageActions = 'create' | 'edit' | 'clone';

export function convertFormikParametersToSchedule(parameters: ScanConfigParameters): Schedule {
    const { intervalType, time, daysOfWeek, daysOfMonth } = parameters;

    // Convert the time to hour and minute
    const [hourString, minuteString] = time.split(/[: ]+/);
    let hour = parseInt(hourString, 10);
    const minute = parseInt(minuteString, 10);

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
                daysOfWeek: {
                    days: daysOfWeek.map((day) => parseInt(day, 10)),
                },
            };
            return weeklySchedule;
        }

        case 'MONTHLY': {
            const monthlySchedule: MonthlySchedule = {
                ...baseSchedule,
                intervalType: 'MONTHLY',
                daysOfMonth: { days: daysOfMonth.map((day) => parseInt(day, 10)) },
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

export function convertScheduleToFormikParameters(
    scanSchedule: Schedule
): Pick<ScanConfigParameters, 'intervalType' | 'time' | 'daysOfWeek' | 'daysOfMonth'> {
    const { hour, minute } = scanSchedule;

    // eslint-disable-next-line no-nested-ternary
    const adjustedHour = hour > 12 ? hour - 12 : hour === 0 ? 12 : hour;
    const suffix = hour > 12 ? 'PM' : 'AM';
    const time = `${adjustedHour}:${minute.toString().padStart(2, '0')} ${suffix}`;

    let intervalType: 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'UNSET' = 'UNSET';
    let daysOfWeek: DayOfWeek[] = [];
    let daysOfMonth: DayOfMonth[] = [];

    switch (scanSchedule.intervalType) {
        case 'WEEKLY':
            intervalType = 'WEEKLY';
            daysOfWeek = scanSchedule.daysOfWeek.days.map(String) as DayOfWeek[];
            break;
        case 'MONTHLY':
            intervalType = 'MONTHLY';
            daysOfMonth = scanSchedule.daysOfMonth.days.map(String) as DayOfMonth[];
            break;
        case 'DAILY':
            intervalType = 'DAILY';
            break;
        case 'UNSET':
        default:
            break;
    }

    return {
        intervalType,
        daysOfWeek,
        daysOfMonth,
        time,
    };
}

export function convertFormikToScanConfig(formikValues: ScanConfigFormValues) {
    const { id, parameters, clusters, profiles } = formikValues;
    const { name, description } = parameters;

    const scanSchedule = convertFormikParametersToSchedule(parameters);

    return {
        id,
        scanName: name,
        scanConfig: {
            description,
            oneTimeScan: false,
            profiles,
            scanSchedule,
        },
        clusters,
    };
}

export function convertScanConfigToFormik(
    existingConfig: ComplianceScanConfigurationStatus
): ScanConfigFormValues {
    const { id, scanName, scanConfig, clusterStatus } = existingConfig;
    const { description = '', profiles, scanSchedule } = scanConfig;

    const { intervalType, time, daysOfWeek, daysOfMonth } =
        convertScheduleToFormikParameters(scanSchedule);

    return {
        id,
        parameters: {
            name: scanName,
            description,
            intervalType,
            time,
            daysOfWeek,
            daysOfMonth,
        },
        clusters: clusterStatus.map((clusterStatus) => clusterStatus.clusterId),
        profiles,
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
