import queryString from 'qs';

import { ReportConfiguration } from 'types/report.proto';
import searchOptionsToQuery, { RestSearchOption } from 'services/searchOptionsToQuery';
import { RestSortOption } from 'services/sortOption';
import axios from './instance';

const reportUrl = '/v1/report';
const reportServiceUrl = `${reportUrl}/run`;
const reportConfigurationsUrl = `${reportUrl}/configurations`;

export function fetchReports(
    options: RestSearchOption[] = [],
    sortOption: RestSortOption,
    page: number,
    pageSize: number
): Promise<ReportConfiguration[]> {
    const offset = page * pageSize;
    const searchOptions: RestSearchOption[] = [...options];
    const query = searchOptionsToQuery(searchOptions);
    const queryObject: Record<string, string | Record<string, number | string | RestSortOption>> = {
        pagination: {
            offset,
            limit: pageSize,
            sortOption,
        },
    };
    if (query) {
        queryObject.query = query;
    }
    const params = queryString.stringify(queryObject, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<{ reportConfigs: ReportConfiguration[] }>(`${reportConfigurationsUrl}?${params}`)
        .then((response) => {
            return response.data.reportConfigs;
        });
}

// TODO: need a way to get total reports count properly
export function fetchReportsCount(options: RestSearchOption[] = []): Promise<number> {
    const searchOptions: RestSearchOption[] = [...options];
    const query = searchOptionsToQuery(searchOptions);
    const queryObject: Record<string, string | Record<string, number | string | RestSortOption>> = {
        pagination: {
            offset: 0,
            limit: 999999,
        },
    };
    if (query) {
        queryObject.query = query;
    }
    const params = queryString.stringify(queryObject, { arrayFormat: 'repeat', allowDots: true });

    return axios
        .get<{ reportConfigs: ReportConfiguration[] }>(`${reportConfigurationsUrl}?${params}`)
        .then((response) => {
            return response.data.reportConfigs.length;
        });
}

export function fetchReportById(reportId: string): Promise<ReportConfiguration> {
    return axios
        .get<{ reportConfig: ReportConfiguration }>(`${reportConfigurationsUrl}/${reportId}`)
        .then((response) => {
            return response?.data?.reportConfig;
        });
}

export function saveReport(report: ReportConfiguration): Promise<ReportConfiguration> {
    const apiPayload = {
        reportConfig: report,
    };

    const promise = report.id
        ? axios.put<ReportConfiguration>(`${reportConfigurationsUrl}/${report.id}`, apiPayload)
        : axios.post<ReportConfiguration>(reportConfigurationsUrl, apiPayload);

    return promise.then((response) => {
        return response.data;
    });
}

export function deleteReport(reportId: string): Promise<Record<string, never>> {
    return axios.delete(`${reportConfigurationsUrl}/${reportId}`);
}

export function runReport(reportId: string): Promise<Record<string, never>> {
    return axios.post(`${reportServiceUrl}/${reportId}`);
}
