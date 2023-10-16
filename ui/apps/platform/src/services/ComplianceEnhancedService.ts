import axios from 'services/instance';
import qs from 'qs';

import { SearchFilter, ApiSortOption } from 'types/search';
import { SlimUser } from 'types/user.proto';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { mockComplianceScanResultsOverview } from 'Containers/ComplianceEnhanced/Status/MockData/complianceScanResultsOverview';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';

const scanScheduleUrl = '/v2/compliance/scan/configurations';

type Schedule = UnsetSchedule | DailySchedule | WeeklySchedule | MonthlySchedule;

type ScheduleIntervalType = 'UNSET' | 'DAILY' | 'WEEKLY' | 'MONTHLY';

type UnsetSchedule = {
    intervalType: 'UNSET';
} & BaseSchedule;

type DailySchedule = {
    intervalType: 'DAILY';
} & BaseSchedule;

type WeeklySchedule = {
    intervalType: 'WEEKLY';
    // Sunday = 0, Monday = 1, .... Saturday =  6
    daysOfWeek: {
        day: number[]; // int32
    };
} & BaseSchedule;

type MonthlySchedule = {
    intervalType: 'WEEKLY';
    // Sunday = 0, Monday = 1, .... Saturday =  6
    daysOfMonth: {
        day: number[]; // int32
    };
} & BaseSchedule;

type BaseSchedule = {
    intervalType: ScheduleIntervalType;
    hour: number;
    minute: number;
};

// API types for Scan Configs:
// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/compliance_scan_configuration_service
export type BaseComplianceScanConfigurationSettings = {
    oneTimeScan: boolean;
    profiles: string[];
    scanSchedule: Schedule | null;
};

export type ClusterScanStatus = {
    clusterId: string;
    errors: string[];
    clusterName: string;
};

export type ScanConfig = {
    id: string;
    scanName: string;
    clusters: string[];
    scanConfig: BaseComplianceScanConfigurationSettings;
    clusterStatus: ClusterScanStatus[];
    createdTime: string; // ISO 8601 date string
    lastUpdatedTime: string; // ISO 8601 date string
    modifiedBy: SlimUser;
};

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
export function getScanConfig(scanConfigId: string): Promise<ScanConfig> {
    return axios
        .get<ScanConfig>(`${scanScheduleUrl}/${scanConfigId}`)
        .then((response) => response.data);
}

/*
 * Get policies filtered by an optional query string.
 */
export function getScanConfigs(query = ''): Promise<ScanConfig[]> {
    const params = qs.stringify({ query });
    return axios
        .get<{ scanSchedules: ScanConfig[] }>(`${scanScheduleUrl}?${params}`)
        .then((response) => response?.data?.scanSchedules ?? []);
}
