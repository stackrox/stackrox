import { ReportConfiguration, ReportConfigurationMappedValues } from 'types/report.proto';
import axios from './instance';

const reportUrl = '/v1/report';
const reportConfigurationsUrl = `${reportUrl}/configurations`;

function mapApiToReport(apiResponse: ReportConfiguration): ReportConfigurationMappedValues {
    const mappedValues: Record<string, unknown> = {};
    Object.keys(apiResponse).forEach((key) => {
        if (key === 'vulnReportFilters') {
            const fixabilityMappedValues =
                apiResponse[key].fixability === 'BOTH'
                    ? ['FIXABLE', 'NOT_FIXABLE']
                    : [apiResponse[key].fixability];
            const vulnReportFiltersMappedValues = {
                fixabilityMappedValues,
                sinceLastReport: apiResponse[key].sinceLastReport,
                severities: apiResponse[key].severities,
            };
            mappedValues.vulnReportFiltersMappedValues = vulnReportFiltersMappedValues;
        } else {
            mappedValues[key] = apiResponse[key];
        }
    });

    return mappedValues as ReportConfigurationMappedValues;
}

function mapReportToApi(report: ReportConfigurationMappedValues): ReportConfiguration {
    const mappedValues: Record<string, unknown> = {};
    Object.keys(report).forEach((key) => {
        if (key === 'vulnReportFiltersMappedValues') {
            const fixability =
                report[key].fixabilityMappedValues.length === 2
                    ? 'BOTH'
                    : [report[key].fixabilityMappedValues];
            const vulnReportFilters = {
                fixability,
                sinceLastReport: report[key].sinceLastReport,
                severities: report[key].severities,
            };
            mappedValues.vulnReportFilters = vulnReportFilters;
        } else {
            mappedValues[key] = report[key];
        }
    });

    return mappedValues as ReportConfiguration;
}

function mapFetchReportApiValues(
    reports: ReportConfiguration[]
): ReportConfigurationMappedValues[] {
    return reports.map(mapApiToReport);
}

export function fetchReports(): Promise<ReportConfigurationMappedValues[]> {
    return axios
        .get<{ reportConfigs: ReportConfiguration[] }>(reportConfigurationsUrl)
        .then((response) => {
            const mappedReports = mapFetchReportApiValues(response.data.reportConfigs);
            // eslint-disable-next-line @typescript-eslint/no-unsafe-return
            return mappedReports;
        });
}

export function fetchReportById(reportId: string): Promise<ReportConfiguration> {
    return axios
        .get<{ reportConfig: ReportConfiguration }>(`${reportConfigurationsUrl}/${reportId}`)
        .then((response) => {
            return response?.data?.reportConfig;
        });
}

export function saveReport(
    report: ReportConfigurationMappedValues
): Promise<ReportConfigurationMappedValues> {
    const apiPayload = {
        reportConfig: mapReportToApi(report),
    };
    const promise = report.id
        ? axios.put<ReportConfiguration>(`${reportConfigurationsUrl}/${report.id}`, apiPayload)
        : axios.post<ReportConfiguration>(reportConfigurationsUrl, apiPayload);
    return promise.then((response) => {
        return mapApiToReport(response.data);
    });
}
