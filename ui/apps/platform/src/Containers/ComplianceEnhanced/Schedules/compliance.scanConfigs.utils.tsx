import type { DayOfMonth, DayOfWeek } from 'Components/PatternFly/DayPickerDropdown';
import { getProductBranding } from 'constants/productBranding';
import type {
    ComplianceScanConfiguration,
    ComplianceScanConfigurationStatus,
    DailySchedule,
    MonthlySchedule,
    Schedule,
    ScheduleBase,
    UnsetSchedule,
    WeeklySchedule,
} from 'services/ComplianceScanConfigurationService';
import type { NotifierConfiguration } from 'services/ReportsService.types';
import { getDayOfMonthWithOrdinal } from 'utils/dateUtils';

export type ScanConfigParameters = {
    name: string;
    description: string;
    intervalType: 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'UNSET';
    time: string;
    daysOfWeek: DayOfWeek[];
    daysOfMonth: DayOfMonth[];
};

export type ScanReportConfiguration = {
    notifierConfigurations: NotifierConfiguration[];
};

export type ScanConfigFormValues = {
    id?: string;
    parameters: ScanConfigParameters;
    clusters: string[];
    profiles: string[];
    report: ScanReportConfiguration;
};

export type PageActions = 'create' | 'edit' | 'clone';

export function getTimeWithHourMinuteFromISO8601(timeISO8601: string) {
    // Given an ISO 8601 date time string from response,
    // for example, 2024-02-29T17:13:28.710959319Z
    // Return yyyy-mm-dd hh:mm UTC
    return `${timeISO8601.slice(0, 10)} ${timeISO8601.slice(11, 16)} UTC`;
}

function padStart2(timeElement: number) {
    return timeElement.toString().padStart(2, '0');
}

export function getHourMinuteStringFromScheduleBase({ hour, minute }: ScheduleBase) {
    // Return 24-hour hh:mm string for hour and minute.
    return [padStart2(hour), padStart2(minute)].join(':');
}

function getScheduleBaseFromHourMinuteString(time: string): ScheduleBase {
    // Return hour and minute for 24-hour hh:mm string.
    const [hourString, minuteString] = time.split(/[: ]+/);
    const hour = parseInt(hourString, 10);
    const minute = parseInt(minuteString, 10);

    return { hour, minute };
}

export function convertFormikParametersToSchedule(parameters: ScanConfigParameters): Schedule {
    const { intervalType, time, daysOfWeek, daysOfMonth } = parameters;

    const baseSchedule = getScheduleBaseFromHourMinuteString(time);

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
    const time = getHourMinuteStringFromScheduleBase(scanSchedule);

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

export function convertFormikToScanConfig(
    formikValues: ScanConfigFormValues
): ComplianceScanConfiguration {
    const { id, parameters, clusters, profiles, report } = formikValues;
    const { name, description } = parameters;
    const { notifierConfigurations } = report;

    const scanSchedule = convertFormikParametersToSchedule(parameters);

    return {
        id,
        scanName: name,
        scanConfig: {
            description,
            oneTimeScan: false,
            profiles,
            scanSchedule,
            notifiers: notifierConfigurations,
        },
        clusters,
    };
}

export function convertScanConfigToFormik(
    existingConfig: ComplianceScanConfigurationStatus
): ScanConfigFormValues {
    const { id, scanName, scanConfig, clusterStatus } = existingConfig;
    const { description = '', notifiers, profiles, scanSchedule } = scanConfig;

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
        report: {
            notifierConfigurations: notifiers,
        },
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

    const timeString = `${getHourMinuteStringFromScheduleBase(schedule)} UTC`;

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

// report

const { reportName } = getProductBranding();

export function getBodyDefault(profiles: string[]) {
    return `${reportName} has scanned your clusters for compliance with the profiles in your scan configuration. The attached report lists those checks and associated details to help with remediation. Profiles: ${profiles.join(',')}`;
}

export function getSubjectDefault(scanName: string, profiles: string[]) {
    return `${reportName} Compliance Report for ${scanName} with ${profiles.length} Profiles`;
}
