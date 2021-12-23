import { ReportConfiguration } from 'types/report.proto';
import axios from './instance';

const reportUrl = '/v1/report';
const reportConfigurationsUrl = `${reportUrl}/configurations`;

export function fetchReports(): Promise<ReportConfiguration[]> {
    return axios
        .get<{ reportConfigs: ReportConfiguration[] }>(reportConfigurationsUrl)
        .then((response) => {
            return response.data.reportConfigs;
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
