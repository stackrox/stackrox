import { useCallback, useState } from 'react';

import { createReportConfiguration } from 'services/ReportsService';
import {
    Fixability,
    ReportConfiguration,
    VulnerabilityReportFilters,
    VulnerabilityReportFiltersBase,
} from 'services/ReportsService.types';
import { ReportFormValues } from '../forms/useReportFormValues';

type Result = {
    data: ReportConfiguration | null;
    isLoading: boolean;
    error: string | null;
};

type CreateReportResult = {
    createReport: (formValues: ReportFormValues) => void;
} & Result;

const defaultResult = {
    data: null,
    isLoading: false,
    error: null,
};

function useCreateReport(): CreateReportResult {
    const [result, setResult] = useState<Result>(defaultResult);

    const createReport = useCallback((formValues: ReportFormValues) => {
        setResult({
            data: null,
            isLoading: true,
            error: null,
        });

        const { reportParameters, deliveryDestinations } = formValues;

        // transform form values to values to be sent through API
        const fixability: Fixability =
            reportParameters.cveStatus.length > 1 ? 'BOTH' : reportParameters.cveStatus[0];

        const vulnReportFiltersBase: VulnerabilityReportFiltersBase = {
            fixability,
            severities: reportParameters.cveSeverities,
            imageTypes: reportParameters.imageType,
        };
        let vulnReportFilters: VulnerabilityReportFilters;
        if (reportParameters.cvesDiscoveredSince === 'SINCE_LAST_REPORT') {
            vulnReportFilters = {
                ...vulnReportFiltersBase,
                lastSuccessfulReport: true,
            };
        } else if (
            reportParameters.cvesDiscoveredSince === 'START_DATE' &&
            reportParameters.cvesDiscoveredStartDate
        ) {
            vulnReportFilters = {
                ...vulnReportFiltersBase,
                startDate: new Date(reportParameters.cvesDiscoveredStartDate).toISOString(),
            };
        } else {
            vulnReportFilters = {
                ...vulnReportFiltersBase,
                allVuln: true,
            };
        }

        const notifiers = deliveryDestinations.map((deliveryDestination) => {
            return {
                emailConfig: {
                    notifierId: deliveryDestination.notifier?.id || '',
                    mailingLists: deliveryDestination.mailingLists,
                },
                notifierName: '',
            };
        });

        const reportData: ReportConfiguration = {
            id: '',
            name: reportParameters.reportName,
            description: reportParameters.description,
            type: 'VULNERABILITY',
            vulnReportFilters,
            resourceScope: {
                collectionScope: {
                    collectionId: reportParameters.reportScope?.id || '',
                    collectionName: reportParameters.reportScope?.name || '',
                },
            },
            notifiers,
            // @TODO: Replace hardcoded values when we do schedule
            schedule: {
                intervalType: 'WEEKLY',
                hour: 0,
                minute: 0,
                daysOfWeek: {
                    days: [3],
                },
            },
        };

        // send API call
        createReportConfiguration(reportData)
            .then((response) => {
                setResult({
                    data: response,
                    isLoading: false,
                    error: null,
                });
            })
            .catch((err) => {
                setResult({
                    data: null,
                    isLoading: false,
                    error: err.response.data.message,
                });
            });
    }, []);

    return {
        ...result,
        createReport,
    };
}

export default useCreateReport;
