import axios from 'services/instance';
import qs from 'qs';

import { SearchFilter, ApiSortOption } from 'types/search';
import { SlimUser } from 'types/user.proto';
import { getListQueryParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { mockGetComplianceScanResultsOverview } from 'Containers/ComplianceEnhanced/MockData/complianceResultsServiceMocks';
import { mockListComplianceProfiles } from 'Containers/ComplianceEnhanced/MockData/complianceProfileServiceMocks';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';
import { Empty } from './types';

const scanScheduleUrl = '/v2/compliance/scan/configurations';
const complianceIntegrationServiceUrl = '/v2/compliance/integrations';
const complianceResultsServiceUrl = '/v2/compliance/scan';
const complianceProfileServiceUrl = '/v2/compliance/profiles';

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

// API types for Scan Configs:
// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/compliance_scan_configuration_service
type BaseComplianceScanConfigurationSettings = {
    oneTimeScan: boolean;
    profiles: string[];
    scanSchedule: Schedule;
    description?: string;
};

export type ClusterScanStatus = {
    clusterId: string;
    errors: string[];
    clusterName: string;
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

// API types for Scan Results:
// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/compliance_results_service

export const ComplianceCheckStatus = {
    UNSET_CHECK_STATUS: 'UNSET_CHECK_STATUS',
    PASS: 'PASS',
    FAIL: 'FAIL',
    ERROR: 'ERROR',
    INFO: 'INFO',
    MANUAL: 'MANUAL',
    NOT_APPLICABLE: 'NOT_APPLICABLE',
    INCONSISTENT: 'INCONSISTENT',
} as const;

export type ComplianceCheckStatus =
    (typeof ComplianceCheckStatus)[keyof typeof ComplianceCheckStatus];

type ComplianceScanCluster = {
    clusterId: string;
    clusterName: string;
};

export type ComplianceCheckStatusCount = {
    count: number;
    status: ComplianceCheckStatus;
};

export type ComplianceScanStatsShim = {
    scanName: string;
    checkStats: ComplianceCheckStatusCount[];
    lastScan: string; // ISO 8601 date string
};

export type ComplianceClusterScanStats = {
    scanStats: ComplianceScanStatsShim;
    cluster: ComplianceScanCluster;
};

export interface ComplianceClusterOverallStats {
    cluster: ComplianceScanCluster;
    checkStats: ComplianceCheckStatusCount[];
    clusterErrors: string[];
}

export interface ComplianceScanResultsOverview {
    scanStats: ComplianceScanStatsShim;
    profileName: string[];
    cluster: ComplianceScanCluster[];
}

export interface ListComplianceScanResultsOverviewResponse {
    scanOverviews: ComplianceScanResultsOverview[];
}

export type ClusterCheckStatus = {
    cluster: ComplianceScanCluster;
    status: ComplianceCheckStatus;
    createdTime: string; // ISO 8601 date string
    checkUid: string;
};

export type ComplianceCheckResult = {
    checkId: string;
    checkName: string;
    clusters: ClusterCheckStatus[];
    description: string;
    instructions: string;
    standard: string;
    control: string;
    rationale: string;
    valuesUsed: string[];
    warnings: string[];
};

type ComplianceScanResult = {
    scanName: string;
    profileName: string;
    checkResults: ComplianceCheckResult[];
};

// API types for Compliance Profiles:
// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/compliance_profile_service
interface ComplianceRule {
    name: string;
    rule_version: string;
    rule_type: string;
    severity: string;
    standard: string;
    control: string;
    title: string;
    description: string;
    rationale: string;
    fixes: string;
}

export interface ComplianceProfile {
    id: string;
    name: string;
    profile_version: string;
    product_type: string[];
    standard: string;
    description: string;
    rules: ComplianceRule[];
    product: string;
    title: string;
}

export type ComplianceProfileSummary = {
    name: string;
    productType: string;
    description: string;
    title: string;
    ruleCount: number;
    profileVersion: string;
};

// API types for Compliance Integrations:
// https://github.com/stackrox/stackrox/blob/master/proto/api/v2/compliance_integration_service
export interface ComplianceIntegration {
    id: string;
    version: string;
    clusterId: string;
    clusterName: string;
    namespace: string;
    statusErrors: string[];
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
                    resolve(
                        mockGetComplianceScanResultsOverview() as ComplianceScanResultsOverview[]
                    );
                }, 2000);
            } else {
                reject(new Error('Request was aborted'));
            }
        });
    });
}

/**
 * Fetches stats for all clusters
 * Note: this function and getSingleClusterCombinedStats call the same API endpoint
 * due to the absence of a dedicated single-cluster endpoint
 */
export function getAllClustersCombinedStats(
    page?: number,
    pageSize?: number
): Promise<ComplianceClusterOverallStats[]> {
    const searchFilter = {};
    const sortOption = {
        field: 'Cluster',
        reversed: false,
    };
    const params = getListQueryParams(searchFilter, sortOption, page, pageSize);

    return axios
        .get<{
            scanStats: ComplianceClusterOverallStats[];
        }>(`${complianceResultsServiceUrl}/stats/overall/cluster?${params}`)
        .then((response) => {
            return response?.data?.scanStats ?? [];
        });
}

/**
 * Fetches stats for a single cluster
 * Note: this function and getAllClustersCombinedStats call the same API endpoint
 * due to the absence of a dedicated single-cluster endpoint
 */
export function getSingleClusterCombinedStats(
    clusterId: string
): Promise<ComplianceClusterOverallStats | null> {
    const query = getRequestQueryStringForSearchFilter({
        'Cluster ID': clusterId,
    });
    const params = qs.stringify({ query });

    return axios
        .get<{
            scanStats: ComplianceClusterOverallStats[];
        }>(`${complianceResultsServiceUrl}/stats/overall/cluster?${params}`)
        .then((response) => {
            const stats = response?.data?.scanStats;
            return stats && stats.length > 0 ? stats[0] : null;
        });
}

/**
 * Fetches stats for all clusters grouped by scan config
 * Note: this function and getSingleClusterStatsByScanConfig call the same API endpoint
 * due to the absence of a dedicated single-cluster endpoint
 */
export function getAllClustersStatsByScanConfig(
    page?: number,
    pageSize?: number
): Promise<ComplianceClusterScanStats[]> {
    // Note: hard-coding the search filter and sort option for now
    const searchFilter = {};
    const sortOption = {
        field: 'Cluster',
        reversed: false,
    };
    const params = getListQueryParams(searchFilter, sortOption, page, pageSize);

    return axios
        .get<{
            scanStats: ComplianceClusterScanStats[];
        }>(`${complianceResultsServiceUrl}/stats/cluster?${params}`)
        .then((response) => {
            return response?.data?.scanStats ?? [];
        });
}

/**
 * Fetches stats for a single cluster grouped by scan config
 * Note: this function and getAllClustersStatsByScanConfig call the same API endpoint
 * due to the absence of a dedicated single-cluster endpoint
 */
export function getSingleClusterStatsByScanConfig(
    clusterId: string
): Promise<ComplianceClusterScanStats[] | null> {
    const query = getRequestQueryStringForSearchFilter({
        'Cluster ID': clusterId,
    });
    const params = qs.stringify({ query });

    return axios
        .get<{
            scanStats: ComplianceClusterScanStats[];
        }>(`${complianceResultsServiceUrl}/stats/cluster?${params}`)
        .then((response) => {
            return response?.data?.scanStats ?? [];
        });
}

export function getSingleClusterResultsByScanConfig(
    clusterId: string,
    scanName: string,
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page?: number,
    pageSize?: number
): Promise<ComplianceScanResult | null> {
    if (!clusterId || !scanName) {
        return Promise.reject(new Error('clusterId and scanName are required'));
    }
    const searchQuery = getRequestQueryStringForSearchFilter({
        'Cluster ID': clusterId,
        'Compliance Scan Config Name': scanName,
        ...searchFilter,
    });

    let offset: number | undefined;
    if (typeof page === 'number' && typeof pageSize === 'number') {
        offset = page > 0 ? page * pageSize : 0;
    }
    const query = {
        query: searchQuery,
        pagination: { offset, limit: pageSize, sortOption },
    };
    const params = qs.stringify(query, { arrayFormat: 'repeat', allowDots: true });

    return axios
        .get<{
            scanResults: ComplianceScanResult[];
        }>(`${complianceResultsServiceUrl}/results?${params}`)
        .then((response) => {
            const results = response?.data?.scanResults ?? [];
            if (results.length > 1) {
                throw new Error('Expected a single result set, but received multiple');
            }
            return results[0] || null;
        });
}

export function getSingleClusterResultsByScanConfigCount(
    clusterId: string,
    scanName: string,
    searchFilter: SearchFilter
): Promise<number> {
    const searchQuery = getRequestQueryStringForSearchFilter({
        'Cluster ID': clusterId,
        'Compliance Scan Config Name': scanName,
        ...searchFilter,
    });

    const query = {
        query: searchQuery,
    };
    const params = qs.stringify(query, { arrayFormat: 'repeat', allowDots: true });

    return axios
        .get<{
            count: number;
        }>(`${complianceResultsServiceUrl}/count/results?${params}`)
        .then((response) => {
            return response?.data?.count ?? 0;
        });
}

/*
 * Get a Scan Schedule.
 */
export function getScanConfig(
    scanConfigId: string
): CancellableRequest<ComplianceScanConfigurationStatus> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<ComplianceScanConfigurationStatus>(`${scanScheduleUrl}/${scanConfigId}`, {
                signal,
            })
            .then((response) => response.data)
    );
}

/*
 * Create a Scan Schedule.
 */
export function saveScanConfig(
    complianceScanConfiguration: ComplianceScanConfiguration
): Promise<ComplianceScanConfiguration> {
    const promise = complianceScanConfiguration.id
        ? axios.put<ComplianceScanConfiguration>(
              `${scanScheduleUrl}/${complianceScanConfiguration.id}`,
              complianceScanConfiguration
          )
        : axios.post<ComplianceScanConfiguration>(scanScheduleUrl, complianceScanConfiguration);

    return promise.then((response) => response.data);
}

/*
 * Get scan configs filtered by an optional query string.
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
    const params = qs.stringify(query, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<{
            configurations: ComplianceScanConfigurationStatus[];
        }>(`${scanScheduleUrl}?${params}`)
        .then((response) => {
            return response?.data?.configurations ?? [];
        });
}

export function getScanConfigsCount(): Promise<number> {
    return axios
        .get<{
            count: number;
        }>(`${complianceResultsServiceUrl}/count/configurations`)
        .then((response) => {
            return response?.data?.count ?? 0;
        });
}

export function deleteScanConfig(scanConfigId: string) {
    return axios.delete<Empty>(`${scanScheduleUrl}/${scanConfigId}`).then((response) => {
        return response.data;
    });
}

export function runComplianceScanConfiguration(scanConfigId: string) {
    return axios.post<Empty>(`${scanScheduleUrl}/${scanConfigId}/run`).then((response) => {
        return response.data;
    });
}

export function listComplianceProfiles(): Promise<ComplianceProfile[]> {
    // TODO: delete the below code once the actual API is ready
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve(mockListComplianceProfiles() as ComplianceProfile[]);
        }, 1000);
    });

    // TODO: Uncomment the below code once the actual API is ready
    // return axios
    //     .get<{ profiles: ComplianceProfile[] }>(complianceProfileServiceUrl)
    //     .then((response) => {
    //         return response?.data?.profiles ?? [];
    //     });
}

export function listComplianceSummaries(clusterIds): Promise<ComplianceProfileSummary[]> {
    const params = qs.stringify({ cluster_ids: clusterIds }, { arrayFormat: 'repeat' });
    return axios
        .get<{
            profiles: ComplianceProfileSummary[];
        }>(`${complianceProfileServiceUrl}/summary?${params}`)
        .then((response) => {
            return response?.data?.profiles ?? [];
        });
}

export function listComplianceIntegrations(): Promise<ComplianceIntegration[]> {
    return axios
        .get<{ integrations: ComplianceIntegration[] }>(complianceIntegrationServiceUrl)
        .then((response) => {
            return response?.data?.integrations ?? [];
        });
}
