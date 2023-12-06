import { useCallback, useState } from 'react';

import { createReportConfiguration } from 'services/ReportsService';
import { ReportConfiguration } from 'services/ReportsService.types';
import useAnalytics, { VULNERABILITY_REPORT_CREATED } from 'hooks/useAnalytics';
import { ReportFormValues } from '../forms/useReportFormValues';
import { getReportConfigurationFromFormValues } from '../utils';

export type UseCreateReportProps = {
    onCompleted: (response: ReportConfiguration) => void;
};

export type Result = {
    data: ReportConfiguration | null;
    isLoading: boolean;
    error: string | null;
};

export type CreateReportResult = {
    createReport: (formValues: ReportFormValues) => void;
} & Result;

const defaultResult = {
    data: null,
    isLoading: false,
    error: null,
};

function trackReportCreation(
    analyticsTrack: ReturnType<typeof useAnalytics>['analyticsTrack'],
    reportConfiguration: ReportConfiguration
) {
    const { severities, fixability, imageTypes } = reportConfiguration.vulnReportFilters;
    const { notifiers } = reportConfiguration;

    const hasEmailNotifier =
        notifiers.length > 0 && notifiers.some((notifier) => notifier.emailConfig);
    const isTemplateModified =
        notifiers.length > 0 &&
        notifiers.some(
            (notifier) => notifier.emailConfig.customBody || notifier.emailConfig.customSubject
        );
    analyticsTrack({
        event: VULNERABILITY_REPORT_CREATED,
        properties: {
            SEVERITY_CRITICAL: severities.includes('CRITICAL_VULNERABILITY_SEVERITY') ? 1 : 0,
            SEVERITY_IMPORTANT: severities.includes('IMPORTANT_VULNERABILITY_SEVERITY') ? 1 : 0,
            SEVERITY_MODERATE: severities.includes('MODERATE_VULNERABILITY_SEVERITY') ? 1 : 0,
            SEVERITY_LOW: severities.includes('LOW_VULNERABILITY_SEVERITY') ? 1 : 0,
            CVE_STATUS_FIXABLE: fixability.includes('FIXABLE') ? 1 : 0,
            CVE_STATUS_NOT_FIXABLE: fixability.includes('NOT_FIXABLE') ? 1 : 0,
            IMAGE_TYPE_DEPLOYED: imageTypes.includes('DEPLOYED') ? 1 : 0,
            IMAGE_TYPE_WATCHED: imageTypes.includes('WATCHED') ? 1 : 0,
            EMAIL_NOTIFIER: hasEmailNotifier ? 1 : 0,
            TEMPLATE_MODIFIED: isTemplateModified ? 1 : 0,
        },
    });
}

function useCreateReport({ onCompleted }: UseCreateReportProps): CreateReportResult {
    const { analyticsTrack } = useAnalytics();
    const [result, setResult] = useState<Result>(defaultResult);

    const createReport = useCallback((formValues: ReportFormValues) => {
        setResult({
            data: null,
            isLoading: true,
            error: null,
        });

        const reportConfiguration = getReportConfigurationFromFormValues(formValues);

        // send API call
        createReportConfiguration(reportConfiguration)
            .then((response) => {
                setResult({
                    data: response,
                    isLoading: false,
                    error: null,
                });
                onCompleted(response);
                trackReportCreation(analyticsTrack, reportConfiguration);
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
