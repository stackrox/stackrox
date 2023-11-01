import axios from 'services/instance';
import qs from 'qs';

import { SearchFilter, ApiSortOption } from 'types/search';
import { SlimUser } from 'types/user.proto';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { mockComplianceScanResultsOverview } from 'Containers/ComplianceEnhanced/Status/MockData/complianceScanResultsOverview';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';

const scanScheduleUrl = '/v2/compliance/scan/configurations';

type ScheduleBase = {
    hour: number;
    minute: number;
};

type UnsetSchedule = ScheduleBase & {
    intervalType: 'UNSET';
};

type DailySchedule = ScheduleBase & {
    intervalType: 'DAILY';
};

type WeeklySchedule = ScheduleBase & {
    intervalType: 'WEEKLY';
    daysOfWeek: { days: number[] };
};

type MonthlySchedule = ScheduleBase & {
    intervalType: 'MONTHLY';
    daysOfMonth: { days: number[] };
};

export type Schedule = UnsetSchedule | DailySchedule | WeeklySchedule | MonthlySchedule;

// API types for Scan Configs:
// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/compliance_scan_configuration_service
type BaseComplianceScanConfigurationSettings = {
    oneTimeScan: boolean;
    profiles: string[];
    scanSchedule: Schedule;
};

type ClusterScanStatus = {
    clusterId: string;
    errors: string[];
    clusterName: string;
};

export type ComplianceScanConfiguration = {
    id: string;
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

// API types for Scan Configs:
// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/compliance_results_service
interface ComplianceScanStatsShim {
    id: string; // TODO: id should be included in api response/proto
    scanName: string;
    numberOfChecks: number; // int32
    numberOfFailingChecks: number; // int32
    numberOfPassingChecks: number; // int32
    lastScan: string; // ISO 8601 date string
}

export interface ComplianceScanResultsOverview {
    scanStats: ComplianceScanStatsShim;
    profileName: string[];
    clusterId: string[];
}

export interface ListComplianceScanResultsOverviewResponse {
    scanOverviews: ComplianceScanResultsOverview[];
}

export function complianceResultsOverview(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page?: number,
    pageSize?: number
): CancellableRequest<ComplianceScanResultsOverview[]> {
    let offset: number | undefined;
    if (typeof page === 'number' && typeof pageSize === 'number') {
        offset = page > 0 ? page * pageSize : 0;
    }
    const query = {
        query: getRequestQueryStringForSearchFilter(searchFilter),
        pagination: { offset, limit: pageSize, sortOption },
    };
    // TODO: remove disabled linter rule when service updated
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const params = qs.stringify({ query }, { allowDots: true });
    return makeCancellableAxiosRequest((signal) => {
        return new Promise((resolve, reject) => {
            if (!signal.aborted) {
                setTimeout(() => {
                    const mockData = mockComplianceScanResultsOverview();
                    resolve(mockData.scanOverviews);
                }, 2000);
            } else {
                reject(new Error('Request was aborted'));
            }
        });
    });
}

/*
 * Get a Scan Schedule.
 */
export function getScanConfig(scanConfigId: string): Promise<ComplianceScanConfigurationStatus> {
    return axios
        .get<ComplianceScanConfigurationStatus>(`${scanScheduleUrl}/${scanConfigId}`)
        .then((response) => response.data);
}

/*
 * Create a Scan Schedule.
 */
export function createScanConfig(
    complianceScanConfiguration: ComplianceScanConfiguration
): Promise<ComplianceScanConfiguration> {
    return axios
        .post<ComplianceScanConfiguration>(`${scanScheduleUrl}`, complianceScanConfiguration)
        .then((response) => response.data);
}

/*
 * Get policies filtered by an optional query string.
 */
export function getScanConfigs(
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
    const params = qs.stringify({ query });
    return axios
        .get<{ configurations: ComplianceScanConfigurationStatus[] }>(
            `${scanScheduleUrl}?${params}`
        )
        .then((response) => {
            return response?.data?.configurations ?? [];
        });
}
