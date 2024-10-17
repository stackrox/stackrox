import Raven from 'raven-js';
import axios from 'services/instance';
import qs from 'qs';

import { ApiSortOption, SearchFilter } from 'types/search';
import { SlimUser } from 'types/user.proto';
import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { getQueryString } from 'utils/queryStringUtils';
import { Snapshot } from 'types/reportJob';
import { ComplianceProfileSummary, complianceV2Url } from './ComplianceCommon';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';
import { NotifierConfiguration } from './ReportsService.types';
import { Empty } from './types';

const complianceScanConfigBaseUrl = `${complianceV2Url}/scan/configurations`;
export const complianceReportDownloadURL = '/v2/compliance/scan/configurations/reports/download';

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
    notifiers: NotifierConfiguration[];
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
    createdTime: string; // ISO 8601 date string
    lastUpdatedTime: string; // ISO 8601 date string
    modifiedBy: SlimUser;
    lastExecutedTime: string | null; // either ISO 8601 date string or null when scan is in progress
};

export type ComplianceReportSnapshot = Snapshot & {
    scanConfigId: string;
    reportData: ComplianceScanConfigurationStatus;
};

export type ListComplianceScanConfigurationsResponse = {
    configurations: ComplianceScanConfigurationStatus[];
    totalCount: number; // int32
};

export type ListComplianceScanConfigsProfileResponse = {
    profiles: ComplianceProfileSummary[];
    totalCount: number;
};

export type ListComplianceScanConfigsClusterProfileResponse = {
    profiles: ComplianceProfileSummary[];
    totalCount: number;
    clusterId: string;
    clusterName: string;
};

/*
 * Fetches a list of scan configurations.
 */
export function listComplianceScanConfigurations(
    sortOption?: ApiSortOption,
    page = 1, // one-based page for compatibility with PatternFly Pagination element
    perPage = 0
): Promise<ListComplianceScanConfigurationsResponse> {
    const query = {
        pagination: getPaginationParams({ page, perPage, sortOption }),
    };
    const params = qs.stringify(query, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<ListComplianceScanConfigurationsResponse>(`${complianceScanConfigBaseUrl}?${params}`)
        .then((response) => {
            return response?.data ?? { configurations: [], totalCount: 0 };
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
 * Initiates a compliance scan for a give configuration ID.
 */
export function runComplianceScanConfiguration(scanConfigId: string) {
    return axios
        .post<Empty>(`${complianceScanConfigBaseUrl}/${scanConfigId}/run`)
        .then((response) => {
            return response.data;
        });
}

export type ComplianceReportRunState = 'SUBMITTED' | 'ERROR';

export type ComplianceRunReportResponse = {
    runState: ComplianceReportRunState;
    submittedAt: string; // ISO 8601 date string
    errorMsg: string;
};

/*
 * Run an on demand compliance report for the scan configuration ID.
 */
export function runComplianceReport(
    scanConfigId: string,
    reportNotificationMethod: 'EMAIL' | 'DOWNLOAD'
): Promise<ComplianceRunReportResponse> {
    const body = { scanConfigId, reportNotificationMethod };
    return axios
        .post<ComplianceRunReportResponse>(`${complianceScanConfigBaseUrl}/reports/run`, body)
        .then((response) => {
            return response.data;
        });
}

/**
 * Fetches all profiles that are included in a scan configuration.
 */
export function listComplianceScanConfigProfiles(
    scanConfigSearchFilter: SearchFilter
): Promise<ListComplianceScanConfigsProfileResponse> {
    const query = getRequestQueryStringForSearchFilter(scanConfigSearchFilter);
    const params = qs.stringify({ query }, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<ListComplianceScanConfigsProfileResponse>(
            `${complianceScanConfigBaseUrl}/profiles/collection?${params}`
        )
        .then((response) => response.data);
}

/**
 * Fetches all profiles that are included in a scan configuration on a specific cluster.
 */
export function listComplianceScanConfigClusterProfiles(
    clusterId: string,
    scanConfigSearchFilter: SearchFilter
): Promise<ListComplianceScanConfigsClusterProfileResponse> {
    const query = getRequestQueryStringForSearchFilter(scanConfigSearchFilter);
    const params = qs.stringify({ query: { query } }, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<ListComplianceScanConfigsClusterProfileResponse>(
            `${complianceScanConfigBaseUrl}/clusters/${clusterId}/profiles/collection?${params}`
        )
        .then((response) => response.data);
}

/*
 *  Fetches a list of snapshots (scan executions) from a given scan config
 */

export type FetchComplianceReportHistoryServiceProps = {
    id: string;
    query: string;
    page: number;
    perPage: number;
    sortOption: ApiSortOption;
    showMyHistory: boolean;
};

export type ReportHistoryResponse = {
    complianceReportSnapshots: ComplianceReportSnapshot[];
};

export function fetchComplianceReportHistory({
    id,
    query,
    page,
    perPage,
    sortOption,
    showMyHistory,
}: FetchComplianceReportHistoryServiceProps): Promise<ComplianceReportSnapshot[]> {
    const params = getQueryString(
        {
            reportParamQuery: {
                query,
                pagination: getPaginationParams({ page, perPage, sortOption }),
            },
        },
        { arrayFormat: 'repeat', allowDots: true, addQueryPrefix: false }
    );
    return axios
        .get<ReportHistoryResponse>(
            `/v2/compliance/scan/configurations/${id}/reports/${showMyHistory ? 'my-history' : 'history'}?${params}`
        )
        .then((response) => {
            return response.data.complianceReportSnapshots;
        })
        .catch((error) => {
            Raven.captureException(error);
            return Promise.reject(error);
        });
}

export function deleteDownloadableComplianceReport(reportJobId: string) {
    return axios
        .delete<Empty>(`/v2/compliance/scan/configurations/reports/${reportJobId}`)
        .then((response) => {
            return response.data;
        });
}
