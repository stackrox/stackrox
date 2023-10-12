import axios from 'services/instance';
import qs from 'qs';

import { SearchFilter, ApiSortOption } from 'types/search';
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

export type ScanSchedule = {
    scanName: string;
    clusters: string[];
    scanConfig: {
        profiles: string[];
        oneTimeScan: boolean;
        scanSchedule: Schedule | null;
    };
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
export function getScanSchedule(policyId: string): Promise<ScanSchedule> {
    return axios
        .get<ScanSchedule>(`${scanScheduleUrl}/${policyId}`)
        .then((response) => response.data);
}

/*
 * Get policies filtered by an optional query string.
 */
export function getScanSchedules(query = ''): Promise<ScanSchedule[]> {
    const params = qs.stringify({ query });
    return axios
        .get<{ scanSchedules: ScanSchedule[] }>(`${scanScheduleUrl}?${params}`)
        .then((response) => response?.data?.scanSchedules ?? []);
}
