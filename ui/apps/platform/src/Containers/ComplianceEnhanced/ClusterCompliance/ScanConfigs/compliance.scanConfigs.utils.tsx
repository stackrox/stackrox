import React, { ReactElement } from 'react';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

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

type ClusterStatusObject = {
    icon: ReactElement;
    statusText: string;
};

export type ScanConfigParameters = {
    name: string;
    description: string;
    intervalType: 'DAILY' | 'WEEKLY' | 'MONTHLY' | null;
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

// TODO: is there a saner way to let TS know string from a subset of numbers is string of that same subset?
function getDayOfWeekString(day): DayOfWeek {
    if (day === 0) {
        return '0';
    }
    if (day === 1) {
        return '1';
    }
    if (day === 2) {
        return '2';
    }
    if (day === 3) {
        return '3';
    }
    if (day === 4) {
        return '4';
    }
    if (day === 5) {
        return '5';
    }
    if (day === 6) {
        return '6';
    }
    return '0'; // fallback, default to first day of week
}
// TODO: is there a saner way to let TS know string from a subset of numbers is string of that same subset?
function getDayOfMonthString(day): DayOfMonth {
    if (day === 1) {
        return '1';
    }
    if (day === 15) {
        return '15';
    }
    return '1'; // fallback, default to first day of month
}

export function convertScheduleToFormikParameters(
    scanSchedule: Schedule
): Pick<ScanConfigParameters, 'intervalType' | 'time' | 'daysOfWeek' | 'daysOfMonth'> {
    const { intervalType, hour, minute } = scanSchedule;

    // eslint-disable-next-line no-nested-ternary
    const adjustedHour = hour > 12 ? hour - 12 : hour === 0 ? 12 : hour;

    const suffix = hour > 12 ? 'PM' : 'AM';

    const timeString = `${adjustedHour}:${minute.toString().padStart(2, '0')} ${suffix}`;

    switch (intervalType) {
        case 'WEEKLY': {
            return {
                intervalType: 'WEEKLY',
                time: timeString,
                daysOfWeek: scanSchedule.daysOfWeek.days.map((day) => getDayOfWeekString(day)),
                daysOfMonth: [],
            };
        }

        case 'MONTHLY': {
            return {
                intervalType: 'MONTHLY',
                time: timeString,
                daysOfWeek: [],
                daysOfMonth: scanSchedule.daysOfMonth.days.map((day) => getDayOfMonthString(day)),
            };
        }

        case 'DAILY': {
            return {
                intervalType: 'DAILY',
                time: timeString,
                daysOfWeek: [],
                daysOfMonth: [],
            };
        }

        case 'UNSET':
        case null:
        default: {
            return {
                intervalType: null,
                time: timeString,
                daysOfWeek: [],
                daysOfMonth: [],
            };
        }
    }
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

export function getClusterStatusObject(errors: string[]): ClusterStatusObject {
    return errors && errors.length && errors[0] !== ''
        ? {
              icon: <ExclamationCircleIcon color="var(--pf-global--danger-color--100)" />,
              statusText: 'Unhealthy',
          }
        : {
              icon: <CheckCircleIcon color="var(--pf-global--success-color--100)" />,
              statusText: 'Healthy',
          };
}
