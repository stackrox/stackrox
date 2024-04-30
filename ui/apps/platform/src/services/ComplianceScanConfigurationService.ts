import axios from 'services/instance';
import qs from 'qs';

import { ApiSortOption } from 'types/search';
import { SlimUser } from 'types/user.proto';

import { complianceV2Url } from './ComplianceCommon';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';
import { Empty } from './types';

const complianceScanConfigBaseUrl = `${complianceV2Url}/scan/configurations`;

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

export type WeeklySchedule = ScheduleBase & {
    intervalType: 'WEEKLY';
    daysOfWeek: { days: number[] };
};

export type MonthlySchedule = ScheduleBase & {
    intervalType: 'MONTHLY';
    daysOfMonth: { days: number[] };
};

export type Schedule = UnsetSchedule | DailySchedule | WeeklySchedule | MonthlySchedule;

export type IntervalType = Schedule['intervalType'];

type SuiteStatus = {
    phase: string;
    result: string;
    errorMessage: string;
    lastTransitionTime: string; // ISO 8601 date string
};

export type ClusterScanStatus = {
    clusterId: string;
    errors: string[];
    clusterName: string;
    suiteStatus: SuiteStatus;
};

type BaseComplianceScanConfigurationSettings = {
    oneTimeScan: boolean;
    profiles: string[];
    scanSchedule: Schedule;
    description?: string;
};

export type ComplianceScanConfiguration = {
    id?: string;
    scanName: string;
    scanConfig: BaseComplianceScanConfigurationSettings;
    clusters: string[];
};

export type ComplianceScanConfigurationStatus = {
    id: string;
    scanName: string;
    scanConfig: BaseComplianceScanConfigurationSettings;
    clusterStatus: ClusterScanStatus[];
    createdTime: string; // ISO 8601 date string;
    lastUpdatedTime: string; // ISO 8601 date string;
    modifiedBy: SlimUser;
};

/*
 * Fetches a list of scan configurations.
 */
export function listComplianceScanConfigurations(
    sortOption: ApiSortOption,
    page?: number,
    pageSize?: number
): Promise<ComplianceScanConfigurationStatus[]> {
    let offset: number | undefined;
    if (typeof page === 'number' && typeof pageSize === 'number') {
        offset = page > 0 ? page * pageSize : 0;
    }
    const query = {
        pagination: { offset, limit: pageSize, sortOption },
    };
    const params = qs.stringify(query, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<{
            configurations: ComplianceScanConfigurationStatus[];
        }>(`${complianceScanConfigBaseUrl}?${params}`)
        .then((response) => {
            return response?.data?.configurations ?? [];
        });
}

/*
 * Fetches a scan configuration by ID.
 */
export function getComplianceScanConfiguration(
    scanConfigId: string
): CancellableRequest<ComplianceScanConfigurationStatus> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<ComplianceScanConfigurationStatus>(
                `${complianceScanConfigBaseUrl}/${scanConfigId}`,
                {
                    signal,
                }
            )
            .then((response) => response.data)
    );
}

/*
 * Creates a new scan configuration or updates an existing one.
 */
export function saveScanConfig(
    complianceScanConfiguration: ComplianceScanConfiguration
): Promise<ComplianceScanConfiguration> {
    const promise = complianceScanConfiguration.id
        ? axios.put<ComplianceScanConfiguration>(
              `${complianceScanConfigBaseUrl}/${complianceScanConfiguration.id}`,
              complianceScanConfiguration
          )
        : axios.post<ComplianceScanConfiguration>(
              complianceScanConfigBaseUrl,
              complianceScanConfiguration
          );

    return promise.then((response) => response.data);
}

/*
 * Deletes a scan configuration by ID.
 */
export function deleteComplianceScanConfiguration(scanConfigId: string) {
    return axios
        .delete<Empty>(`${complianceScanConfigBaseUrl}/${scanConfigId}`)
        .then((response) => {
            return response.data;
        });
}

/*
 * Returns the count of scan configurations.
 */
export function getComplianceScanConfigurationsCount(): Promise<number> {
    return axios
        .get<{
            count: number;
        }>(`${complianceV2Url}/scan/count/configurations`)
        .then((response) => {
            return response?.data?.count ?? 0;
        });
}

/*
 * Initiates a compliance scan for a give configuration ID.
 */
export function runComplianceScanConfiguration(scanConfigId: string) {
    return axios
        .post<Empty>(`${complianceScanConfigBaseUrl}/${scanConfigId}/run`)
        .then((response) => {
            return response.data;
        });
}
