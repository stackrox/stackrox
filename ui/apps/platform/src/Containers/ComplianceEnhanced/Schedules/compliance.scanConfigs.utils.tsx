import type { DayOfMonth, DayOfWeek } from 'Components/PatternFly/DayPickerDropdown';
import { getProductBranding } from 'constants/productBranding';
import type {
    ComplianceScanConfiguration,
    ComplianceScanConfigurationStatus,
} from 'services/ComplianceScanConfigurationService';
import type { NotifierConfiguration } from 'services/ReportsService.types';
import type {
    DailySchedule,
    MonthlySchedule,
    Schedule,
    ScheduleBase,
    UnsetSchedule,
    WeeklySchedule,
} from 'types/schedule.proto';
import { getHourMinuteStringFromScheduleBase } from 'utils/dateUtils';

// Keep in sync with the backend defaults in central/complianceoperator/v2/scanconfigurations/service/convert.go
// and sensor/kubernetes/complianceoperator/types.go (defaultNodeRoles).
export const defaultNodeRoles: string[] = ['master', 'worker'];

// Special role that selects every node; mutually exclusive with any other role.
export const allNodesRole = '@all';

// A concrete node role is 1-39 lowercase alphanumeric-or-hyphen characters, starting and
// ending with an alphanumeric character. The UI is intentionally stricter than the backend
// nodeRoleRegexp (which allows [A-Za-z0-9]): the Compliance Operator uses the role verbatim
// as a case-sensitive "node-role.kubernetes.io/<role>" label key AND as part of an RFC1123
// (lowercase) object name, so an uppercase role is accepted by the API but then hard-fails at
// the operator. Keep in sync with nodeRoleRegexp in
// central/complianceoperator/v2/scanconfigurations/service/service_impl.go.
export const nodeRoleRegex = /^[a-z0-9]([a-z0-9-]{0,37}[a-z0-9])?$/;

// Shared validation-rule message for a single node role. Used by both the input-time feedback
// in ScanConfigOptions and the submit-time yup rule so the wording comes from one source.
export const nodeRoleValidationMessage =
    'Use 1-39 lowercase alphanumeric characters and hyphens, starting and ending with a letter or number, or @all.';

// A single node role is valid when it is either the @all wildcard or matches the regex.
export function isValidNodeRole(role: string): boolean {
    return role === allNodesRole || nodeRoleRegex.test(role);
}

// A node roles array is valid when every entry is a valid role, @all is not combined with any
// other role, and there are no duplicate roles (the backend rejects duplicates with a 400).
export function areNodeRolesValid(roles: string[]): boolean {
    if (!roles.every(isValidNodeRole)) {
        return false;
    }
    if (roles.includes(allNodesRole) && roles.length > 1) {
        return false;
    }
    // Reject duplicates, mirroring the backend which returns 400 for repeated roles.
    return new Set(roles).size === roles.length;
}

export type ScanConfigParameters = {
    name: string;
    description: string;
    intervalType: 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'UNSET';
    time: string;
    daysOfWeek: DayOfWeek[];
    daysOfMonth: DayOfMonth[];
    nodeRoles: string[];
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
    const { name, description, nodeRoles } = parameters;
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
            nodeRoles,
        },
        clusters,
    };
}

// Legacy configs stored before node roles were configurable have empty/absent nodeRoles
// but actually run master+worker on Sensor (and the backend defaults empty to master+worker),
// so fall back for display. Returns a fresh array for the default so callers never share the
// mutable defaultNodeRoles reference.
export function getNodeRolesForDisplay(nodeRoles?: string[]): string[] {
    return nodeRoles && nodeRoles.length > 0 ? nodeRoles : [...defaultNodeRoles];
}

export function convertScanConfigToFormik(
    existingConfig: ComplianceScanConfigurationStatus
): ScanConfigFormValues {
    const { id, scanName, scanConfig, clusterStatus } = existingConfig;
    const { description = '', notifiers, profiles, scanSchedule, nodeRoles } = scanConfig;

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
            // Fall back to master+worker for legacy configs (see getNodeRolesForDisplay).
            nodeRoles: getNodeRolesForDisplay(nodeRoles),
        },
        clusters: clusterStatus.map((clusterStatus) => clusterStatus.clusterId),
        profiles,
        report: {
            notifierConfigurations: notifiers,
        },
    };
}

// report

const { reportName } = getProductBranding();

export function getBodyDefault(profiles: string[]) {
    return `${reportName} has scanned your clusters for compliance with the profiles in your scan configuration. The attached report lists those checks and associated details to help with remediation. Profiles: ${profiles.join(',')}`;
}

export function getSubjectDefault(scanName: string, profiles: string[]) {
    return `${reportName} Compliance Report for ${scanName} with ${profiles.length} Profiles`;
}
